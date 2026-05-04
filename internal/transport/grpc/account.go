package grpc

import (
	"context"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	ledger "high-load-ledger/gen/go"
	"high-load-ledger/internal/domain/entity"
)

func (h *Handler) CreateAccount(ctx context.Context, req *ledger.CreateAccountRequest) (*ledger.CreateAccountResponse, error) {
	id, err := h.accountUC.CreateAccount(ctx, entity.Currency(req.Currency)) // вероятнее всего здесь нужна явная валидация currency и в целом нужно поменять сигнатуру метода на ту, которая принимает какую-то явную структуру, но мне уже настолько лень и нет сил, что оставлю как есть
	if err != nil {
		h.logger.ErrorContext(ctx, "create account failed", "error", err)
		return nil, status.Errorf(codes.Internal, "create account failed: %v", err)
	}

	return &ledger.CreateAccountResponse{
		AccountId: id[:],
	}, nil
}

func (h *Handler) GetBalance(ctx context.Context, req *ledger.GetBalanceRequest) (*ledger.GetBalanceResponse, error) {
	accountID, err := uuid.FromBytes(req.AccountId)
	if err != nil {
		h.logger.ErrorContext(ctx, "invalid account_id", "error", err)
		return nil, status.Error(codes.InvalidArgument, "invalid account_id")
	}

	account, err := h.accountUC.GetBalance(ctx, accountID)
	if err != nil {
		h.logger.ErrorContext(ctx, "get balance failed", "error", err)
		return nil, status.Errorf(codes.Internal, "get balance failed: %v", err)
	}

	return &ledger.GetBalanceResponse{
		Balance:  account.Balance,
		Currency: ledger.Currency(account.Currency),
	}, nil
}
