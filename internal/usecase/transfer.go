package usecase

import (
	"context"
	"high-load-ledger/internal/domain/entity"
	"high-load-ledger/internal/domain/repository"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

type TransferUseCase struct {
	repo              repository.TransactionRepository
	cache             repository.CacheRepository
	logger            *slog.Logger
	idempotencyKeyTTL time.Duration
}

func NewTransferUseCase(repo repository.TransactionRepository, cache repository.CacheRepository, logger *slog.Logger, ttl time.Duration) *TransferUseCase {
	return &TransferUseCase{
		repo:              repo,
		cache:             cache,
		logger:            logger,
		idempotencyKeyTTL: ttl,
	}
}

func (t *TransferUseCase) Transaction(ctx context.Context, req entity.TransactionRequest) (uuid.UUID, error) {
	if err := t.validateRequest(ctx, req); err != nil {
		return uuid.Nil, err
	}

	txID, err := t.checkIdempotency(ctx, req.IdempotencyKey)
	if err == nil {
		return txID, nil
	}

	newTx := entity.Transaction{
		ID:             uuid.New(),
		IdempotencyKey: req.IdempotencyKey,
		FromAccountID:  req.FromAccountID,
		ToAccountID:    req.ToAccountID,
		Amount:         req.Amount,
		Currency:       req.Currency,
		Status:         entity.STATUS_PENDING,
		CreatedAt:      time.Now(),
	}

	tr, err := t.repo.BeginTx(ctx)
	if err != nil {
		return uuid.Nil, err
	}

	defer func() {
		if err != nil {
			if rollbackErr := t.repo.RollbackTx(ctx, tr); rollbackErr != nil {
				t.logger.ErrorContext(ctx, "failed to rollback transaction", "err", rollbackErr)
			}
		}
	}()

	err = t.repo.CreateTransaction(ctx, tr, &newTx)
	if err != nil {
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
	if err == nil {
		return uuid.FromBytes(val)
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

func (t *TransferUseCase) validateRequest(ctx context.Context, req entity.TransactionRequest) error {
	if req.Amount <= 0 {
		return entity.ErrInvalidAmount
	}
	if req.FromAccountID == req.ToAccountID {
		return entity.ErrSameAccountTransfer
	}
	if req.IdempotencyKey == uuid.Nil {
		return entity.ErrEmptyIdempotencyKey
	}
	return nil
}
