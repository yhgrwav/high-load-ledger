package repository

import (
	"context"
	"time"

	"github.com/google/uuid"

	"high-load-ledger/internal/domain/entity"
)

type CacheRepository interface {
	SetIdempotencyKey(ctx context.Context, key uuid.UUID, response []byte, ttl time.Duration) error
	GetIdempotencyKey(ctx context.Context, key uuid.UUID) ([]byte, error)

	SetBalance(ctx context.Context, accountID uuid.UUID, amount int64, ttl time.Duration) error
	GetBalance(ctx context.Context, accountID uuid.UUID) (int64, error)
	DeleteBalance(ctx context.Context, accountID uuid.UUID) error

	SetTransaction(ctx context.Context, tx *entity.Transaction, ttl time.Duration) error
	GetTransaction(ctx context.Context, id uuid.UUID) (*entity.Transaction, error)
}
