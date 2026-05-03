package usecase

import (
	"context"
	"high-load-ledger/internal/domain/entity"
	"high-load-ledger/internal/domain/repository"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

type AccountUseCase struct {
	repo   repository.AccountRepository
	logger *slog.Logger
}

func NewAccountUseCase(repo repository.AccountRepository, logger *slog.Logger) *AccountUseCase {
	return &AccountUseCase{
		repo:   repo,
		logger: logger,
	}
}

func (a *AccountUseCase) CreateAccount(ctx context.Context, id uuid.UUID, currency entity.Currency) (uuid.UUID, error) {
	account := &entity.Account{
		ID:        id,
		Balance:   0,
		Currency:  currency,
		UpdatedAt: time.Now(),
	}

	err := a.repo.CreateAccount(ctx, account)
	if err != nil {
		return uuid.Nil, err
	}
	return id, nil
}

func (a *AccountUseCase) GetBalance(ctx context.Context, id uuid.UUID) (*entity.Account, error) {
	return a.repo.GetByID(ctx, id)
}
