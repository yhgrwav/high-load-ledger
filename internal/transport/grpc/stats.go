package grpc

import (
	"context"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	ledger "high-load-ledger/gen/go"
)

func (h *Handler) GetTransaction(ctx context.Context, req *ledger.GetTransactionRequest) (*ledger.GetTransactionResponse, error) {
	txID, err := uuid.FromBytes(req.TransactionId)
	if err != nil {
		h.logger.ErrorContext(ctx, "invalid transaction_id", "error", err)
		return nil, status.Error(codes.InvalidArgument, "invalid transaction_id")
	}

	h.logger.InfoContext(ctx, "GetTransaction stub called", "tx_id", txID)
	return nil, status.Error(codes.Unimplemented, "not implemented yet")
}
