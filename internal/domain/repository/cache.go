package repository

import (
	"context"
	"high-load-ledger/internal/domain/entity"
	"time"

	"github.com/google/uuid"
)

type CacheRepository interface {
	TryGetIdempotencyKey(ctx context.Context, key uuid.UUID, ttl time.Duration) (bool, error)
	SetIdempotencyKey(ctx context.Context, key uuid.UUID, value []byte, ttl time.Duration) error
	GetIdempotencyKey(ctx context.Context, key uuid.UUID) (entity.IdempotencyEntry, error)
	DeleteIdempotencyKey(ctx context.Context, key uuid.UUID) error

	SetBalance(ctx context.Context, accountID uuid.UUID, amount int64, ttl time.Duration) error
	GetBalance(ctx context.Context, accountID uuid.UUID) (int64, error)
	DeleteBalance(ctx context.Context, accountID uuid.UUID) error

	SetTransaction(ctx context.Context, tx *entity.Transaction, ttl time.Duration) error
	GetTransaction(ctx context.Context, id uuid.UUID) (*entity.Transaction, error)

	SetAccountCurrency(ctx context.Context, accountID uuid.UUID, currency entity.Currency, ttl time.Duration) error
	GetAccountCurrency(ctx context.Context, accountID uuid.UUID) (entity.Currency, error)
}
