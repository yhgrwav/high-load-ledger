package repository

import (
	"context"
	"high-load-ledger/internal/domain/entity"

	"github.com/google/uuid"
)

type IdempotencyRepository interface {
	SetAndCheck(ctx context.Context, idempotencyKey uuid.UUID, status entity.IdempotencyStatus) error
	GetIdempotencyStatus(ctx context.Context, idempotencyKey uuid.UUID) (entity.IdempotencyStatus, error)
	UpdateIdempotencyStatus(ctx context.Context, idempotencyKey uuid.UUID, status entity.IdempotencyStatus) error
	DeleteIdempotency(ctx context.Context, idempotencyKey uuid.UUID) error
}
