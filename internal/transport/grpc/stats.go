package grpc

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	ledger "high-load-ledger/gen/go"
	"high-load-ledger/internal/domain/entity"
)

func (h *Handler) GetTransaction(ctx context.Context, req *ledger.GetTransactionRequest) (*ledger.GetTransactionResponse, error) {
	txID, err := uuid.FromBytes(req.TransactionId)
	if err != nil {
		h.logger.ErrorContext(ctx, "invalid transaction_id", "error", err)
		return nil, status.Error(codes.InvalidArgument, "invalid transaction_id")
	}

	tx, err := h.statsUC.GetTransaction(ctx, txID)
	if err != nil {
		if errors.Is(err, entity.ErrTransactionNotFound) {
			return nil, status.Error(codes.NotFound, "transaction not found")
		}
		h.logger.ErrorContext(ctx, "get transaction failed", "error", err, "transaction_id", txID)
		return nil, status.Error(codes.Internal, "get transaction failed")
	}

	return &ledger.GetTransactionResponse{
		Id:       tx.ID[:],
		UserFrom: tx.FromAccountID[:],
		UserTo:   tx.ToAccountID[:],
		Amount:   tx.Amount,
		Currency: ledger.Currency(tx.Currency),
	}, nil
}
