package usecase

import (
	"context"
	"errors"
	"high-load-ledger/internal/domain/entity"
	"high-load-ledger/internal/domain/repository"
	"high-load-ledger/internal/infra/telemetry"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

type transferRepo interface {
	repository.TransactionRepository
	repository.AccountRepository
	repository.PostingRepository
}

type TransferUseCase struct {
	repo              transferRepo
	cache             repository.CacheRepository
	logger            *slog.Logger
	idempotencyKeyTTL time.Duration
	metrics           *telemetry.PrometheusMetrics
}

func NewTransferUseCase(repo transferRepo, cache repository.CacheRepository, logger *slog.Logger, ttl time.Duration, metrics *telemetry.PrometheusMetrics) *TransferUseCase {
	return &TransferUseCase{
		repo:              repo,
		cache:             cache,
		logger:            logger,
		idempotencyKeyTTL: ttl,
		metrics:           metrics,
	}
}

func (t *TransferUseCase) Transaction(ctx context.Context, req entity.TransactionRequest) (id uuid.UUID, err error) {
	defer func() {
		if t.metrics == nil {
			return
		}

		status := "success"
		switch {
		case err == nil:
			status = "success"
		case errors.Is(err, entity.ErrInvalidAmount),
			errors.Is(err, entity.ErrSameAccountTransfer),
			errors.Is(err, entity.ErrEmptyIdempotencyKey),
			errors.Is(err, entity.ErrInvalidCurrency):
			status = "validation_error"
		case errors.Is(err, entity.ErrCurrencyMismatch):
			status = "currency_mismatch"
		case errors.Is(err, entity.ErrInsufficientFunds):
			status = "insufficient_funds"
		default:
			status = "system_error"
		}

		t.metrics.RecordTransfer(status)
	}()

	if err := t.validateRequest(req); err != nil {
		return uuid.Nil, err
	}

	txID, err := t.checkIdempotency(ctx, req.IdempotencyKey)
	if err == nil {
		return txID, nil
	}
	if !errors.Is(err, entity.ErrTransactionNotFound) {
		return uuid.Nil, err
	}

	if err = t.validateTransferCurrencies(ctx, req.FromAccountID, req.ToAccountID, req.Currency); err != nil {
		return uuid.Nil, err
	}

	tx, err := t.repo.BeginTx(ctx)
	if err != nil {
		return uuid.Nil, err
	}

	defer func() {
		if err != nil {
			_ = t.repo.RollbackTx(ctx, tx)
		}
	}()

	if _, err = t.repo.GetForUpdate(ctx, tx, req.FromAccountID); err != nil {
		return uuid.Nil, err
	}

	if _, err = t.repo.GetInTx(ctx, tx, req.ToAccountID); err != nil {
		return uuid.Nil, err
	}

	trxID, err := uuid.NewV7()
	if err != nil {
		return uuid.Nil, err
	}

	trx := entity.Transaction{
		ID:             trxID,
		IdempotencyKey: req.IdempotencyKey,
		FromAccountID:  req.FromAccountID,
		ToAccountID:    req.ToAccountID,
		Amount:         req.Amount,
		Currency:       req.Currency,
		CreatedAt:      time.Now(),
	}

	if err = t.repo.CreateTransaction(ctx, tx, &trx); err != nil {
		existing, err := t.repo.CheckIdempotencyKey(ctx, req.IdempotencyKey)
		if err == nil {
			_ = t.repo.RollbackTx(ctx, tx)
			_ = t.cache.SetIdempotencyKey(ctx, req.IdempotencyKey, existing.ID[:], t.idempotencyKeyTTL)
			return existing.ID, nil
		}
		return uuid.Nil, err
	}

	postings := []entity.Posting{
		{TransactionID: trx.ID, AccountID: req.FromAccountID, Amount: -req.Amount},
		{TransactionID: trx.ID, AccountID: req.ToAccountID, Amount: req.Amount},
	}
	if err = t.repo.CreatePostings(ctx, tx, postings); err != nil {
		return uuid.Nil, err
	}

	if err = t.repo.DebitBalance(ctx, tx, req.FromAccountID, req.Amount); err != nil {
		return uuid.Nil, err
	}
	if err = t.repo.CreditBalance(ctx, tx, req.ToAccountID, req.Amount); err != nil {
		return uuid.Nil, err
	}

	if err = t.repo.CommitTx(ctx, tx); err != nil {
		return uuid.Nil, err
	}

	_ = t.cache.SetIdempotencyKey(ctx, req.IdempotencyKey, trx.ID[:], t.idempotencyKeyTTL)

	return trx.ID, nil
}

func (t *TransferUseCase) checkIdempotency(ctx context.Context, key uuid.UUID) (uuid.UUID, error) {
	val, err := t.cache.GetIdempotencyKey(ctx, key)
	if err == nil && len(val) == 16 {
		if id, err := uuid.FromBytes(val); err == nil {
			return id, nil
		}
	}

	trx, err := t.repo.CheckIdempotencyKey(ctx, key)
	if err != nil {
		return uuid.Nil, err
	}

	_ = t.cache.SetIdempotencyKey(ctx, key, trx.ID[:], t.idempotencyKeyTTL)

	return trx.ID, nil
}

func (t *TransferUseCase) validateRequest(req entity.TransactionRequest) error {
	if req.Amount <= 0 {
		return entity.ErrInvalidAmount
	}
	if req.FromAccountID == req.ToAccountID {
		return entity.ErrSameAccountTransfer
	}
	if req.IdempotencyKey == uuid.Nil {
		return entity.ErrEmptyIdempotencyKey
	}
	if !req.Currency.IsValid() {
		return entity.ErrInvalidCurrency
	}
	return nil
}

func (t *TransferUseCase) validateTransferCurrencies(ctx context.Context, fromID, toID uuid.UUID, currency entity.Currency) error {
	fromAcc, err := t.repo.GetByID(ctx, fromID)
	if err != nil {
		return err
	}
	if fromAcc.Currency != currency {
		return entity.ErrCurrencyMismatch
	}

	toAcc, err := t.repo.GetByID(ctx, toID)
	if err != nil {
		return err
	}
	if toAcc.Currency != currency {
		return entity.ErrCurrencyMismatch
	}

	return nil
}
