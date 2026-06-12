package interceptors

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	ledger "high-load-ledger/gen/go"
	"high-load-ledger/internal/domain/entity"
	"high-load-ledger/internal/domain/repository"
)

func UnaryIdempotencyInterceptor(idem repository.IdempotencyRepository) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if info.FullMethod != ledger.TransactionService_Transfer_FullMethodName {
			return handler(ctx, req)
		}

		key, err := uuid.FromBytes(req.(*ledger.TransferRequest).GetIdempotencyKey())
		if err != nil || key == uuid.Nil {
			return handler(ctx, req)
		}

		claimed, err := idem.SetAndCheck(ctx, key, entity.IDEMPOTENCY_IN_PROCESS)
		if err != nil {
			return nil, status.Errorf(codes.Unavailable, "idempotency: %v", err)
		}
		if !claimed {
			return cachedTransfer(ctx, idem, key)
		}

		resp, err := handler(ctx, req)
		if err != nil {
			_ = idem.DeleteIdempotency(ctx, key)
			return nil, err
		}

		out := resp.(*ledger.TransferResponse)
		txID, err := uuid.FromBytes(out.GetTransactionId())
		if err != nil {
			_ = idem.DeleteIdempotency(ctx, key)
			return nil, status.Errorf(codes.Internal, "invalid transaction_id: %v", err)
		}

		if err := idem.UpdateIdempotencyStatus(ctx, key, completedStatus(txID)); err != nil {
			return nil, status.Errorf(codes.Unavailable, "idempotency: %v", err)
		}

		return out, nil
	}
}

func cachedTransfer(ctx context.Context, idem repository.IdempotencyRepository, key uuid.UUID) (*ledger.TransferResponse, error) {
	st, err := idem.GetIdempotencyStatus(ctx, key)
	if err != nil {
		return nil, status.Errorf(codes.Unavailable, "idempotency: %v", err)
	}

	if txID, ok := parseCompleted(st); ok {
		return &ledger.TransferResponse{TransactionId: txID[:]}, nil
	}

	return nil, status.Error(codes.Aborted, entity.ErrIdempotencyInProgress.Error())
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
