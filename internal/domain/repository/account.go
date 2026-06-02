package repository

import (
	"context"
	"high-load-ledger/internal/domain/entity"

	"github.com/google/uuid"
)

type AccountRepository interface {
	CreateAccount(ctx context.Context, tx entity.CustomTx, acc *entity.Account) error
	GetByID(ctx context.Context, id uuid.UUID) (*entity.Account, error)
	GetCurrencies(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]entity.Currency, error)
	DebitBalance(ctx context.Context, tx entity.CustomTx, id uuid.UUID, amount int64) error
	CreditBalance(ctx context.Context, tx entity.CustomTx, id uuid.UUID, amount int64) error
}
