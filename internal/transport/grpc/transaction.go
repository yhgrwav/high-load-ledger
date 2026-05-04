package grpc

import (
	"context"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	ledger "high-load-ledger/gen/go"
	"high-load-ledger/internal/domain/entity"
)

func (h *Handler) Transfer(ctx context.Context, req *ledger.TransferRequest) (*ledger.TransferResponse, error) {
	ik, err := uuid.FromBytes(req.IdempotencyKey)
	if err != nil {
		h.logger.ErrorContext(ctx, "invalid idempotency key", "error", err)
		return nil, status.Error(codes.InvalidArgument, "invalid idempotency_key")
	}

	fromID, err := uuid.FromBytes(req.UserFromId)
	if err != nil {
		h.logger.ErrorContext(ctx, "invalid user_from_id", "error", err)
		return nil, status.Error(codes.InvalidArgument, "invalid user_from_id")
	}

	toID, err := uuid.FromBytes(req.UserToId)
	if err != nil {
		h.logger.ErrorContext(ctx, "invalid user_to_id", "error", err)
		return nil, status.Error(codes.InvalidArgument, "invalid user_to_id")
	}

	domainReq := entity.TransactionRequest{
		IdempotencyKey: ik,
		FromAccountID:  fromID,
		ToAccountID:    toID,
		Amount:         req.Amount,
		Currency:       entity.Currency(req.Currency),
	}

	txID, err := h.transferUC.Transaction(ctx, domainReq)
	if err != nil {
		h.logger.ErrorContext(ctx, "transfer failed", "error", err)
		return nil, status.Errorf(codes.Internal, "transfer failed: %v", err)
	}

	return &ledger.TransferResponse{
		TransactionId: txID[:],
		Status:        ledger.TransactionStatus_STATUS_PENDING,
	}, nil
}
