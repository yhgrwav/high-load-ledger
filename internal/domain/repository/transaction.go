package repository

import (
	"context"
	"high-load-ledger/internal/domain/entity"

	"github.com/google/uuid"
)

type TransactionRepository interface {
	CreateTransaction(ctx context.Context, tx entity.CustomTx, tr *entity.Transaction) error
	GetTransactionByID(ctx context.Context, id uuid.UUID) (*entity.Transaction, error)
	CheckIdempotencyKey(ctx context.Context, key uuid.UUID) (*entity.Transaction, error)
	UpdateStatus(ctx context.Context, tx entity.CustomTx, id uuid.UUID, status entity.TransactionStatus) error
}
