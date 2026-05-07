package usecase

import (
	"context"
	"high-load-ledger/internal/domain/entity"
	"high-load-ledger/internal/domain/repository"
	"log/slog"
	"math/rand"
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

func (a *AccountUseCase) CreateAccount(ctx context.Context, currency entity.Currency) (uuid.UUID, error) {
	if currency == entity.CURRENCY_UNSPECIFIED {
		return uuid.Nil, entity.ErrInvalidCurrency
	}

	id, err := uuid.NewV7()
	if err != nil {
		a.logger.ErrorContext(ctx, "service: error generating uuid for account: ", err)
		return uuid.Nil, err
	}

	balance := rand.Int()

	account := &entity.Account{
		ID:        id,
		Balance:   int64(balance),
		Currency:  currency,
		UpdatedAt: time.Now().UTC(),
	}

	err = a.repo.CreateAccount(ctx, account)
	if err != nil {
		return uuid.Nil, err
	}
	return id, nil
}

func (a *AccountUseCase) GetBalance(ctx context.Context, id uuid.UUID) (*entity.Account, error) {
	return a.repo.GetByID(ctx, id)
}
