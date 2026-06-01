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

type accountRepo interface {
	repository.AccountRepository
	repository.TransactionRepository
	repository.PostingRepository
}

type AccountUseCase struct {
	repo   accountRepo
	logger *slog.Logger
}

func NewAccountUseCase(repo accountRepo, logger *slog.Logger) *AccountUseCase {
	return &AccountUseCase{
		repo:   repo,
		logger: logger,
	}
}

func (a *AccountUseCase) CreateAccount(ctx context.Context, currency entity.Currency) (id uuid.UUID, err error) {
	if currency == entity.CURRENCY_UNSPECIFIED {
		return uuid.Nil, entity.ErrInvalidCurrency
	}

	id, err = uuid.NewV7()
	if err != nil {
		a.logger.ErrorContext(ctx, "service: error generating uuid for account", "err", err)
		return uuid.Nil, err
	}

	balance := int64(rand.Int())

	account := &entity.Account{
		ID:        id,
		Balance:   balance,
		Currency:  currency,
		UpdatedAt: time.Now().UTC(),
	}

	tx, err := a.repo.BeginTx(ctx)
	if err != nil {
		return uuid.Nil, err
	}

	defer func() {
		if err != nil {
			_ = a.repo.RollbackTx(ctx, tx)
		}
	}()

	if err = a.repo.CreateAccount(ctx, tx, account); err != nil {
		return uuid.Nil, err
	}

	if balance > 0 {
		var openingKey uuid.UUID
		openingKey, err = uuid.NewV7()
		if err != nil {
			return uuid.Nil, err
		}

		var trxID uuid.UUID
		trxID, err = uuid.NewV7()
		if err != nil {
			return uuid.Nil, err
		}

		trx := entity.Transaction{
			ID:             trxID,
			IdempotencyKey: openingKey,
			FromAccountID:  id,
			ToAccountID:    id,
			Amount:         balance,
			Currency:       currency,
			CreatedAt:      time.Now().UTC(),
		}

		if err = a.repo.CreateTransaction(ctx, tx, &trx); err != nil {
			return uuid.Nil, err
		}

		postings := []entity.Posting{
			{TransactionID: trx.ID, AccountID: id, Amount: balance},
		}
		if err = a.repo.CreatePostings(ctx, tx, postings); err != nil {
			return uuid.Nil, err
		}
	}

	if err = a.repo.CommitTx(ctx, tx); err != nil {
		return uuid.Nil, err
	}

	return id, nil
}

func (a *AccountUseCase) GetBalance(ctx context.Context, id uuid.UUID) (*entity.Account, error) {
	return a.repo.GetByID(ctx, id)
}
