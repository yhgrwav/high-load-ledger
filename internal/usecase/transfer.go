package usecase

import (
	"bytes"
	"context"
	"errors"
	"high-load-ledger/internal/domain/entity"
	"high-load-ledger/internal/domain/repository"
	"high-load-ledger/internal/infra/telemetry"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

type txAccRepo interface {
	repository.TransactionRepository
	repository.AccountRepository
}

type TransferUseCase struct {
	repo              txAccRepo
	cache             repository.CacheRepository
	logger            *slog.Logger
	idempotencyKeyTTL time.Duration
	metrics           *telemetry.PrometheusMetrics
}

func NewTransferUseCase(repo txAccRepo, cache repository.CacheRepository, logger *slog.Logger, ttl time.Duration, metrics *telemetry.PrometheusMetrics) *TransferUseCase {
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
		// обработка кейса пустого поля метрик для тестов которые я никогда не напишу
		if t.metrics == nil {
			return
		}

		// label, который будет передаваться в метрику, по дефолту success, если ошибка - ошибка в свитч-кейсе определяется по типу
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

		// результат кидается в счётчик по статусу
		t.metrics.TransactionResultCounter.WithLabelValues(status).Inc()
	}()

	if err := t.validateRequest(req); err != nil {
		return uuid.Nil, err
	}

	txID, err := t.checkIdempotency(ctx, req.IdempotencyKey)
	if err == nil {
		return txID, nil
	}

	tr, err := t.repo.BeginTx(ctx)
	if err != nil {
		return uuid.Nil, err
	}

	defer func() {
		if err != nil {
			if err := t.repo.RollbackTx(ctx, tr); err != nil {
				t.logger.ErrorContext(ctx, "failed to rollback transaction", "err", err)
			}
		}
	}()

	fromAcc, toAcc, err := t.loadTransferAccounts(ctx, tr, req.FromAccountID, req.ToAccountID)
	if err != nil {
		return uuid.Nil, err
	}

	if fromAcc.Currency != req.Currency || toAcc.Currency != req.Currency {
		return uuid.Nil, entity.ErrCurrencyMismatch
	}
	if fromAcc.Balance < req.Amount {
		return uuid.Nil, entity.ErrInsufficientFunds
	}

	if err = t.repo.UpdateBalance(ctx, tr, req.FromAccountID, fromAcc.Balance-req.Amount); err != nil {
		return uuid.Nil, err
	}
	if err = t.repo.UpdateBalance(ctx, tr, req.ToAccountID, toAcc.Balance+req.Amount); err != nil {
		return uuid.Nil, err
	}

	newTx := entity.Transaction{
		ID:             uuid.New(),
		IdempotencyKey: req.IdempotencyKey,
		FromAccountID:  req.FromAccountID,
		ToAccountID:    req.ToAccountID,
		Amount:         req.Amount,
		Currency:       req.Currency,
		Status:         entity.STATUS_COMPLETED,
		CreatedAt:      time.Now(),
	}

	if err = t.repo.CreateTransaction(ctx, tr, &newTx); err != nil {
		existing, err := t.repo.CheckIdempotencyKey(ctx, req.IdempotencyKey)
		if err == nil {
			_ = t.cache.SetIdempotencyKey(ctx, req.IdempotencyKey, existing.ID[:], t.idempotencyKeyTTL)
			return existing.ID, nil
		}
		return uuid.Nil, err
	}

	err = t.repo.CommitTx(ctx, tr)
	if err != nil {
		return uuid.Nil, err
	}

	_ = t.cache.SetIdempotencyKey(ctx, req.IdempotencyKey, newTx.ID[:], t.idempotencyKeyTTL)

	return newTx.ID, nil
}

func (t *TransferUseCase) checkIdempotency(ctx context.Context, key uuid.UUID) (uuid.UUID, error) {
	val, err := t.cache.GetIdempotencyKey(ctx, key)
	if err == nil && len(val) == 16 {
		if id, uerr := uuid.FromBytes(val); uerr == nil {
			return id, nil
		}
	}

	tr, err := t.repo.CheckIdempotencyKey(ctx, key)
	if err != nil {
		return uuid.Nil, err
	}

	err = t.cache.SetIdempotencyKey(ctx, key, tr.ID[:], t.idempotencyKeyTTL)
	if err != nil {
		t.logger.WarnContext(ctx, "failed to update cache", "err", err)
	}

	return tr.ID, nil
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

func (t *TransferUseCase) loadTransferAccounts(ctx context.Context, tr entity.CustomTx, fromID, toID uuid.UUID) (*entity.Account, *entity.Account, error) {
	lockFirstID, lockSecondID := fromID, toID
	if bytes.Compare(lockFirstID[:], lockSecondID[:]) > 0 {
		lockFirstID, lockSecondID = lockSecondID, lockFirstID
	}

	lockFirstAcc, err := t.repo.GetForUpdate(ctx, tr, lockFirstID)
	if err != nil {
		return nil, nil, err
	}
	lockSecondAcc, err := t.repo.GetForUpdate(ctx, tr, lockSecondID)
	if err != nil {
		return nil, nil, err
	}

	if lockFirstID == fromID {
		return lockFirstAcc, lockSecondAcc, nil
	}
	return lockSecondAcc, lockFirstAcc, nil
}
