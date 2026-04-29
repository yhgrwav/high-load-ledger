package postgres

import (
	"context"
	"fmt"
	"high-load-ledger/internal/domain/entity"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (db *Repository) CreateTransaction(ctx context.Context, tx entity.CustomTx, tr *entity.Transaction) error {
	query := `INSERT INTO ledger.Transactions (id, user_from, user_to, currency, amount, idempotency_key, status, created_at)
				  VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	t, err := db.castTx(ctx, tx)
	if err != nil {
		return err
	}
	_, err = t.Exec(ctx, query, tr.ID, tr.FromAccountID, tr.ToAccountID, tr.Currency, tr.Amount, tr.IdempotencyKey, tr.Status, tr.CreatedAt)
	if err != nil {
		db.logger.ErrorContext(ctx, "db: create transaction error", "err", err)
		return err
	}
	return nil
}

func (db *Repository) GetTransactionByID(ctx context.Context, id uuid.UUID) (*entity.Transaction, error) {
	query := `SELECT id, user_from, user_to, currency, amount, idempotency_key, status, created_at
              FROM ledger.Transactions 
              WHERE id = $1`

	var tr entity.Transaction

	err := db.pool.QueryRow(ctx, query, id).Scan(&tr.ID, &tr.FromAccountID, &tr.ToAccountID, &tr.Currency, &tr.Amount, &tr.IdempotencyKey, &tr.Status, &tr.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			db.logger.ErrorContext(ctx, "db: transaction_id not found", "err", err)
			return nil, entity.ErrTransactionNotFound
		}
		db.logger.ErrorContext(ctx, "db: get transaction error", "err", err)
		return nil, fmt.Errorf("db: get transaction error: %w", err)
	}
	return &tr, nil
}

func (db *Repository) CheckIdempotencyKey(ctx context.Context, key uuid.UUID) (*entity.Transaction, error) {
	query := `SELECT id, user_from, user_to, currency, amount, idempotency_key, status, created_at
              FROM ledger.Transactions
              WHERE idempotency_key = $1`

	var tr entity.Transaction

	err := db.pool.QueryRow(ctx, query, key).Scan(&tr.ID, &tr.FromAccountID, &tr.ToAccountID, &tr.Currency, &tr.Amount, &tr.IdempotencyKey, &tr.Status, &tr.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, entity.ErrTransactionNotFound
		}
		db.logger.ErrorContext(ctx, "db: get transaction error", "err", err)
		return nil, fmt.Errorf("db: check idempotency key error: %w", err)
	}
	return &tr, nil
}

func (db *Repository) UpdateStatus(ctx context.Context, tx entity.CustomTx, id uuid.UUID, status entity.TransactionStatus) error {
	t, err := db.castTx(ctx, tx)
	if err != nil {
		return err
	}

	query := `UPDATE ledger.Transactions SET status = $1 WHERE id = $2`
	_, err = t.Exec(ctx, query, status, id)
	if err != nil {
		db.logger.ErrorContext(ctx, "db: update transaction error", "err", err)
		return fmt.Errorf("db: update transaction error: %w", err)
	}
	return nil
}
