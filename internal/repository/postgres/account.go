package postgres

import (
	"context"
	"errors"
	"fmt"
	"high-load-ledger/internal/domain/entity"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (db *Repository) CreateAccount(ctx context.Context, acc *entity.Account) error {
	query := `INSERT INTO ledger.accounts(user_id, amount, currency) VALUES($1, $2, $3)`

	_, err := db.pool.Exec(ctx, query, acc.ID, acc.Balance, acc.Currency)
	if err != nil {
		db.logger.ErrorContext(ctx, "db: failed to create account", "err", err, "user_id", acc.ID)
		return fmt.Errorf("db: failed to create account: %w", err)
	}
	return nil
}

func (db *Repository) GetForUpdate(ctx context.Context, tx entity.CustomTx, id uuid.UUID) (*entity.Account, error) {
	t, err := db.castTx(ctx, tx)
	if err != nil {
		return nil, err
	}

	query := `SELECT user_id, amount, currency, updated_at 
			  FROM ledger.accounts
			  WHERE user_id = $1
			  FOR UPDATE`
	var account entity.Account

	err = t.QueryRow(ctx, query, id).Scan(&account.ID, &account.Balance, &account.Currency, &account.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, entity.ErrAccountNotFound
		}
		db.logger.ErrorContext(ctx, "db: failed to get account", "err", err, "id", id)
		return nil, fmt.Errorf("db: failed to get account: %w", err)
	}
	return &account, nil
}

func (db *Repository) GetByID(ctx context.Context, id uuid.UUID) (*entity.Account, error) {
	query := `select user_id, amount, currency, updated_at 
		  FROM ledger.accounts
		  WHERE user_id = $1`
	var account entity.Account
	err := db.pool.QueryRow(ctx, query, id).Scan(&account.ID, &account.Balance, &account.Currency, &account.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, entity.ErrAccountNotFound
		}
		db.logger.ErrorContext(ctx, "db: failed to get account", "err", err, "id", id)
		return nil, fmt.Errorf("db: failed to get account: %w", err)
	}
	return &account, nil
}

func (db *Repository) UpdateBalance(ctx context.Context, tx entity.CustomTx, id uuid.UUID, amount int64) error {

	t, err := db.castTx(ctx, tx)
	if err != nil {
		return err
	}

	query := `UPDATE ledger.accounts
              SET amount = $1, updated_at = CURRENT_TIMESTAMP
              WHERE user_id = $2`
	_, err = t.Exec(ctx, query, amount, id)
	if err != nil {
		db.logger.ErrorContext(ctx, "db: failed to update balance", "err", err, "id", id)
		return fmt.Errorf("db: failed to update balance: %w", err)
	}
	return nil
}

func (db *Repository) BeginAccountTx(ctx context.Context) (entity.CustomTx, error) {
	tx, err := db.pool.Begin(ctx)
	if err != nil {
		db.logger.ErrorContext(ctx, "db: failed to begin account transaction", "err", err)
		return nil, fmt.Errorf("db: failed to begin account transaction: %w", err)
	}
	return tx, err
}

func (db *Repository) CommitAccountTx(ctx context.Context, tx entity.CustomTx) error {
	t, err := db.castTx(ctx, tx)
	if err != nil {
		return err
	}
	return t.Commit(ctx)
}

func (db *Repository) RollbackAccountTx(ctx context.Context, tx entity.CustomTx) error {
	t, err := db.castTx(ctx, tx)
	if err != nil {
		return err
	}
	return t.Rollback(ctx)
}

// castTx - локальная вспомогательный метод, который будет типизировать тип any в pgx.Tx (конкретно в этой реализации)
// и будет возвращать сущность, с которой будет удобно работать внутри других методов, а также этот метод поможет избежать
// дублирования кода, что способствует реализации принципа DRY
func (db *Repository) castTx(ctx context.Context, tx entity.CustomTx) (pgx.Tx, error) {
	if tx == nil {
		db.logger.ErrorContext(ctx, "db: failed to cast tx")
		return nil, entity.ErrInvalidTxType
	}
	pgTx, ok := tx.(pgx.Tx)
	if !ok {
		return nil, entity.ErrInvalidTxType
	}
	return pgTx, nil
}
