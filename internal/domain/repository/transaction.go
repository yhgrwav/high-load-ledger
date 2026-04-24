package repository

import (
	"context"
	"high-load-ledger/internal/domain/entity"

	"github.com/google/uuid"
)

type TransactionRepository interface {
	Create(ctx context.Context, tx *entity.Transaction) error
	GetByIdempotencyKey(ctx context.Context, key uuid.UUID) (*entity.Transaction, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status entity.TransactionStatus) error
}
