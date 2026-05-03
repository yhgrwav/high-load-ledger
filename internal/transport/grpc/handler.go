package grpc

import (
	"context"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	ledger "high-load-ledger/gen/go"
	"high-load-ledger/internal/domain/entity"
	"high-load-ledger/internal/usecase"
)

type Handler struct {
	ledger.UnimplementedTransactionServiceServer
	ledger.UnimplementedAccountServiceServer
	transferUC *usecase.TransferUseCase
	accountUC  *usecase.AccountUseCase
}

func NewHandler(transferUC *usecase.TransferUseCase, accountUC *usecase.AccountUseCase) *Handler {
	return &Handler{
		transferUC: transferUC,
		accountUC:  accountUC,
	}
}

func (h *Handler) Transfer(ctx context.Context, req *ledger.TransferRequest) (*ledger.TransferResponse, error) {
	ik, err := uuid.FromBytes(req.IdempotencyKey)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid idempotency_key")
	}

	fromID, err := uuid.FromBytes(req.UserFromId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_from_id")
	}

	toID, err := uuid.FromBytes(req.UserToId)
	if err != nil {
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
		return nil, status.Errorf(codes.Internal, "transfer failed: %v", err)
	}

	return &ledger.TransferResponse{
		TransactionId: txID[:],
		Status:        ledger.TransactionStatus_STATUS_PENDING,
	}, nil
}

func (h *Handler) CreateAccount(ctx context.Context, req *ledger.CreateAccountRequest) (*ledger.CreateAccountResponse, error) {
	userID, err := uuid.FromBytes(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	id, err := h.accountUC.CreateAccount(ctx, userID, entity.Currency(req.Currency))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "create account failed: %v", err)
	}

	return &ledger.CreateAccountResponse{
		AccountId: id[:],
	}, nil
}

func (h *Handler) GetBalance(ctx context.Context, req *ledger.GetBalanceRequest) (*ledger.GetBalanceResponse, error) {
	accountID, err := uuid.FromBytes(req.AccountId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid account_id")
	}

	account, err := h.accountUC.GetBalance(ctx, accountID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "get balance failed: %v", err)
	}

	return &ledger.GetBalanceResponse{
		Balance:  account.Balance,
		Currency: ledger.Currency(account.Currency),
	}, nil
}
