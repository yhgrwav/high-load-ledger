package repository

import (
	"context"
	"high-load-ledger/internal/domain/entity"
	"time"

	"github.com/google/uuid"
)

type CacheRepository interface {
	SetIdempotencyKey(ctx context.Context, key uuid.UUID, response []byte, ttl time.Duration) error
	GetIdempotencyKey(ctx context.Context, key uuid.UUID) ([]byte, error)

	SetBalance(ctx context.Context, accountID uuid.UUID, amount int64, ttl time.Duration) error
	GetBalance(ctx context.Context, accountID uuid.UUID) (int64, error)
	DeleteBalance(ctx context.Context, accountID uuid.UUID) error

	SetAccountCurrency(ctx context.Context, accountID uuid.UUID, currency entity.Currency, ttl time.Duration) error
	GetAccountCurrency(ctx context.Context, accountID uuid.UUID) (entity.Currency, error)
}
