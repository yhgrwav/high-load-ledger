package postgres

import (
	"context"
	"errors"
	"fmt"
	"high-load-ledger/internal/domain/entity"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (db *Repository) CreateAccount(ctx context.Context, tx entity.CustomTx, acc *entity.Account) error {
	query := `INSERT INTO ledger.accounts(user_id, amount, currency, latest_posting_id) VALUES($1, $2, $3, $4)`

	t, err := db.castTx(ctx, tx)
	if err != nil {
		return err
	}

	_, err = t.Exec(ctx, query, acc.ID, acc.Balance, acc.Currency, acc.LatestPostingID)
	if err != nil {
		db.logger.ErrorContext(ctx, "db: failed to create account", "err", err, "user_id", acc.ID)
		return fmt.Errorf("db: failed to create account: %w", err)
	}
	return nil
}

func (db *Repository) GetByID(ctx context.Context, id uuid.UUID) (*entity.Account, error) {
	query := `select user_id, amount, currency, latest_posting_id, updated_at 
		  FROM ledger.accounts
		  WHERE user_id = $1`
	var account entity.Account
	err := db.pool.QueryRow(ctx, query, id).Scan(&account.ID, &account.Balance, &account.Currency, &account.LatestPostingID, &account.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, entity.ErrAccountNotFound
		}
		db.logger.ErrorContext(ctx, "db: failed to get account", "err", err, "id", id)
		return nil, fmt.Errorf("db: failed to get account: %w", err)
	}
	return &account, nil
}

func (db *Repository) GetCurrencies(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]entity.Currency, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	query := `SELECT user_id, currency FROM ledger.accounts WHERE user_id = ANY($1)`
	rows, err := db.pool.Query(ctx, query, ids)
	if err != nil {
		db.logger.ErrorContext(ctx, "db: failed to get currencies", "err", err)
		return nil, fmt.Errorf("db: failed to get currencies: %w", err)
	}
	defer rows.Close()

	result := make(map[uuid.UUID]entity.Currency, len(ids))
	for rows.Next() {
		var id uuid.UUID
		var currency entity.Currency
		if err := rows.Scan(&id, &currency); err != nil {
			db.logger.ErrorContext(ctx, "db: scan currency failed", "err", err)
			return nil, fmt.Errorf("db: scan currency failed: %w", err)
		}
		result[id] = currency
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("db: rows scan currency failed: %w", err)
	}

	return result, nil
}

func (db *Repository) DebitBalance(ctx context.Context, tx entity.CustomTx, id uuid.UUID, amount int64) error {
	t, err := db.castTx(ctx, tx)
	if err != nil {
		return err
	}

	query := `UPDATE ledger.accounts
              SET amount = amount - $1, updated_at = CURRENT_TIMESTAMP
              WHERE user_id = $2 AND amount >= $1`

	tag, err := t.Exec(ctx, query, amount, id)
	if err != nil {
		db.logger.ErrorContext(ctx, "db: failed to debit balance", "err", err, "id", id)
		return fmt.Errorf("db: failed to debit balance: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return entity.ErrInsufficientFunds
	}
	return nil
}

func (db *Repository) CreditBalance(ctx context.Context, tx entity.CustomTx, id uuid.UUID, amount int64) error {
	t, err := db.castTx(ctx, tx)
	if err != nil {
		return err
	}

	query := `UPDATE ledger.accounts
              SET amount = amount + $1, updated_at = CURRENT_TIMESTAMP
              WHERE user_id = $2`

	tag, err := t.Exec(ctx, query, amount, id)
	if err != nil {
		db.logger.ErrorContext(ctx, "db: failed to credit balance", "err", err, "id", id)
		return fmt.Errorf("db: failed to credit balance: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return entity.ErrAccountNotFound
	}
	return nil
}

func (db *Repository) GetTwoForUpdate(ctx context.Context, tx entity.CustomTx, fromAccountID, toAccountID uuid.UUID) (*entity.Account, *entity.Account, error) {
	t, err := db.castTx(ctx, tx)
	if err != nil {
		return nil, nil, err
	}

	query := `SELECT user_id, amount, currency, latest_posting_id, updated_at
             FROM ledger.accounts
             WHERE user_id IN ($1, $2)
             ORDER BY user_id
             FOR UPDATE`

	rows, err := t.Query(ctx, query, fromAccountID, toAccountID)
	if err != nil {
		db.logger.ErrorContext(ctx, "db: failed to get account", "err", err)
		return nil, nil, fmt.Errorf("failed to get account: %w", err)
	}
	defer rows.Close()

	var acc1, acc2 entity.Account
	var found1, found2 bool

	for rows.Next() {
		var acc entity.Account
		if err := rows.Scan(&acc.ID, &acc.Balance, &acc.Currency, &acc.LatestPostingID, &acc.UpdatedAt); err != nil {
			return nil, nil, err
		}
		if acc.ID == fromAccountID {
			acc1 = acc
			found1 = true
		} else if acc.ID == toAccountID {
			acc2 = acc
			found2 = true
		}
	}

	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	if !found1 || !found2 {
		return nil, nil, entity.ErrAccountNotFound
	}

	return &acc1, &acc2, nil
}
