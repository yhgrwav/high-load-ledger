package interceptors

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	ledger "high-load-ledger/gen/go"
	"high-load-ledger/internal/domain/entity"
)

type idempotencyStorage interface {
	CheckOrLock(ctx context.Context, key uuid.UUID) (bool, entity.IdempotencyStatus, error)
	UpdateIdempotencyStatus(ctx context.Context, key uuid.UUID, status entity.IdempotencyStatus) error
	DeleteIdempotency(ctx context.Context, key uuid.UUID) error
}

func UnaryIdempotencyInterceptor(idem idempotencyStorage, logger *slog.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if info.FullMethod != ledger.TransactionService_Transfer_FullMethodName {
			return handler(ctx, req)
		}

		key, err := uuid.FromBytes(req.(*ledger.TransferRequest).GetIdempotencyKey())
		if err != nil || key == uuid.Nil {
			return handler(ctx, req)
		}

		claimed, currentStatus, err := idem.CheckOrLock(ctx, key)
		if err != nil {
			logger.WarnContext(ctx, "idempotency: redis unavailable, fail-open", "err", err, "key", key)
			return handler(ctx, req)
		}

		if !claimed {
			return handleDuplicate(currentStatus)
		}

		defer func() {
			if r := recover(); r != nil {
				_ = idem.DeleteIdempotency(ctx, key)
				logger.ErrorContext(ctx, "idempotency: internal server error", "err", err, "key", key)

				err = status.Errorf(codes.Internal, "internal server error")
			}
		}()

		resp, err := handler(ctx, req)
		if err != nil {
			_ = idem.DeleteIdempotency(ctx, key)
			return nil, err
		}

		out := resp.(*ledger.TransferResponse)
		txID, err := uuid.FromBytes(out.GetTransactionId())
		if err != nil {
			_ = idem.DeleteIdempotency(ctx, key)
			return nil, status.Errorf(codes.Internal, "invalid transaction_id from handler: %v", err)
		}

		go func(k uuid.UUID, targetStatus entity.IdempotencyStatus) {
			asyncCtx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
			defer cancel()

			if err := idem.UpdateIdempotencyStatus(asyncCtx, k, targetStatus); err != nil {
				logger.WarnContext(asyncCtx, "idempotency: async update failed (fail-open)",
					"err", err,
					"key", k,
					"status", targetStatus,
				)
			}
		}(key, completedStatus(txID))

		return out, nil
	}
}

func handleDuplicate(st entity.IdempotencyStatus) (*ledger.TransferResponse, error) {
	if st == entity.IDEMPOTENCY_IN_PROCESS {
		return nil, status.Error(codes.Aborted, entity.ErrIdempotencyInProgress.Error())
	}

	if txID, ok := parseCompleted(st); ok {
		return &ledger.TransferResponse{TransactionId: txID[:]}, nil
	}

	return nil, status.Error(codes.Internal, "unknown idempotency status in cache")
}

func completedStatus(txID uuid.UUID) entity.IdempotencyStatus {
	return entity.IdempotencyStatus("completed:" + txID.String())
}

func parseCompleted(st entity.IdempotencyStatus) (uuid.UUID, bool) {
	raw := string(st)
	if !strings.HasPrefix(raw, "completed:") {
		return uuid.Nil, false
	}
	txID, err := uuid.Parse(strings.TrimPrefix(raw, "completed:"))
	return txID, err == nil
}
