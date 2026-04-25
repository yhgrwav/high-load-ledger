package repository

import (
	"context"
	"high-load-ledger/internal/domain/entity"

	"github.com/google/uuid"
)

type AccountRepository interface {
	CreateAccount(ctx context.Context, acc *entity.Account) error
	GetForUpdate(ctx context.Context, id uuid.UUID) (*entity.Account, error)
	GetByID(ctx context.Context, id uuid.UUID) (*entity.Account, error)
	UpdateBalance(ctx context.Context, id uuid.UUID, newAmount int64) error
}
