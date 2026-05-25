package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"high-load-ledger/internal/domain/entity"
)

func (db *Repository) GetPostingsSum(ctx context.Context, accountID uuid.UUID, limitID int64) (int64, error) {
	q := querierFromContext(ctx, db)

	var (
		query string
		args  []any
	)

	if limitID > 0 {
		query = `SELECT COALESCE(SUM(amount), 0)
		          FROM ledger.postings
		          WHERE account_id = $1 AND id <= $2`
		args = []any{accountID, limitID}
	} else {
		query = `SELECT COALESCE(SUM(amount), 0)
		          FROM ledger.postings
		          WHERE account_id = $1`
		args = []any{accountID}
	}

	var sum int64
	if err := q.QueryRow(ctx, query, args...).Scan(&sum); err != nil {
		db.logger.ErrorContext(ctx, "db: get postings sum failed", "err", err, "account_id", accountID, "limit_id", limitID)
		return 0, fmt.Errorf("db: get postings sum failed: %w", err)
	}
	return sum, nil
}

func (db *Repository) GetAccountBalanceSnapshot(ctx context.Context, accountID uuid.UUID) (balance int64, latestPostingID int64, err error) {
	q := querierFromContext(ctx, db)

	query := `SELECT amount, latest_posting_id
	          FROM ledger.accounts
	          WHERE user_id = $1`

	err = q.QueryRow(ctx, query, accountID).Scan(&balance, &latestPostingID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, 0, entity.ErrAccountNotFound
		}
		db.logger.ErrorContext(ctx, "db: get account balance snapshot failed", "err", err, "account_id", accountID)
		return 0, 0, fmt.Errorf("db: get account balance snapshot failed: %w", err)
	}
	return balance, latestPostingID, nil
}

func (db *Repository) ApplyBalanceCorrection(ctx context.Context, accountID uuid.UUID, newAmount int64) error {
	return db.UpdateBalanceFromContext(ctx, accountID, newAmount)
}

func (db *Repository) UpdateBalanceFromContext(ctx context.Context, accountID uuid.UUID, newAmount int64) error {
	q := querierFromContext(ctx, db)

	query := `UPDATE ledger.accounts
	          SET amount = $1,
	              latest_posting_id = (
	                  SELECT COALESCE(MAX(id), 0)
	                  FROM ledger.postings
	                  WHERE account_id = $2
	              ),
	              updated_at = CURRENT_TIMESTAMP
	          WHERE user_id = $2`

	if _, err := q.Exec(ctx, query, newAmount, accountID); err != nil {
		db.logger.ErrorContext(ctx, "db: update balance from context failed", "err", err, "account_id", accountID)
		return fmt.Errorf("db: update balance from context failed: %w", err)
	}
	return nil
}

func (db *Repository) GetCursorPosition(ctx context.Context, workerName string, batchSize int) (cursorPosition, upperLimit int64, err error) {
	query := `SELECT COALESCE(
	            (SELECT position FROM ledger.worker_cursors WHERE worker_name = $1),
	            0
	          )`

	if err = db.pool.QueryRow(ctx, query, workerName).Scan(&cursorPosition); err != nil {
		db.logger.ErrorContext(ctx, "db: get worker cursor position failed", "err", err, "worker", workerName)
		return 0, 0, fmt.Errorf("db: get worker cursor position failed: %w", err)
	}

	var maxPostingID int64
	if err = db.pool.QueryRow(ctx, `SELECT COALESCE(MAX(id), 0) FROM ledger.postings`).Scan(&maxPostingID); err != nil {
		db.logger.ErrorContext(ctx, "db: get max posting id failed", "err", err)
		return 0, 0, fmt.Errorf("db: get max posting id failed: %w", err)
	}

	upperLimit = cursorPosition + int64(batchSize)
	if upperLimit > maxPostingID {
		upperLimit = maxPostingID
	}

	return cursorPosition, upperLimit, nil
}

func (db *Repository) GetActiveAccounts(ctx context.Context, lastCheckedID, maxID int64) ([]uuid.UUID, error) {
	if lastCheckedID >= maxID {
		return nil, nil
	}

	query := `SELECT DISTINCT account_id
	          FROM ledger.postings
	          WHERE id > $1 AND id <= $2
	          ORDER BY account_id`

	rows, err := db.pool.Query(ctx, query, lastCheckedID, maxID)
	if err != nil {
		db.logger.ErrorContext(ctx, "db: get active accounts failed", "err", err)
		return nil, fmt.Errorf("db: get active accounts failed: %w", err)
	}
	defer rows.Close()

	var accounts []uuid.UUID
	for rows.Next() {
		var accountID uuid.UUID
		if err := rows.Scan(&accountID); err != nil {
			db.logger.ErrorContext(ctx, "db: scan active account failed", "err", err)
			return nil, fmt.Errorf("db: scan active account failed: %w", err)
		}
		accounts = append(accounts, accountID)
	}
	if err := rows.Err(); err != nil {
		db.logger.ErrorContext(ctx, "db: iterate active accounts failed", "err", err)
		return nil, fmt.Errorf("db: iterate active accounts failed: %w", err)
	}
	return accounts, nil
}

func (db *Repository) UpdateCursorPosition(ctx context.Context, workerName string, position int64) error {
	query := `INSERT INTO ledger.worker_cursors (worker_name, position)
	          VALUES ($1, $2)
	          ON CONFLICT (worker_name)
	          DO UPDATE SET position = EXCLUDED.position`

	if _, err := db.pool.Exec(ctx, query, workerName, position); err != nil {
		db.logger.ErrorContext(ctx, "db: update worker cursor position failed", "err", err, "worker", workerName)
		return fmt.Errorf("db: update worker cursor position failed: %w", err)
	}
	return nil
}
