package postgres

import (
	"context"
	"fmt"
	"high-load-ledger/internal/domain/entity"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (db *Repository) CreatePostings(ctx context.Context, tx entity.CustomTx, postings []entity.Posting) error {
	t, err := db.castTx(ctx, tx)
	if err != nil {
		return err
	}

	query := `INSERT INTO postings (id, transaction_id, account_id, amount)
			  VALUES($1, $2, $3, $4)`

	batch := &pgx.Batch{}

	for _, posting := range postings {
		batch.Queue(query, posting.ID, posting.TransactionID, posting.AccountID, posting.Amount)
	}

	result := t.SendBatch(ctx, batch)
	if err := result.Close(); err != nil {
		db.logger.ErrorContext(ctx, "db: batch insert posting failed", "err", err)
		return fmt.Errorf("db: batch insert posting failed: %w", err)
	}

	return nil
}

func (db *Repository) ListPostingsByAccountID(ctx context.Context, accountID uuid.UUID, limit, offset int) ([]entity.Posting, error) {
	query := `SELECT id, transaction_id, account_id, amount
              FROM postings
              WHERE account_id = $1
              ORDER BY created_at DESC
              LIMIT $2 OFFSET $3`

	rows, err := db.pool.Query(ctx, query, accountID, limit, offset)
	if err != nil {
		db.logger.ErrorContext(ctx, "db: query postings failed", "err", err)
		return nil, fmt.Errorf("db: query postings failed: %w", err)
	}
	defer rows.Close()

	var postings []entity.Posting

	for rows.Next() {
		var posting entity.Posting
		err := rows.Scan(&posting.ID, &posting.TransactionID, &posting.AccountID, &posting.Amount)
		if err != nil {
			db.logger.ErrorContext(ctx, "db: scan postings failed", "err", err)
			return nil, fmt.Errorf("db: scan postings failed: %w", err)
		}
		postings = append(postings, posting)
	}
	if err := rows.Err(); err != nil {
		db.logger.ErrorContext(ctx, "db: rows scan postings failed", "err", err)
		return nil, fmt.Errorf("db: rows scan postings failed: %w", err)
	}
	return postings, nil
}

func (db *Repository) GetBalanceFromPostings(ctx context.Context, accountID uuid.UUID) (int64, error) {
	query := `SELECT COALESCE(SUM(amount), 0) FROM postings WHERE account_id = $1`

	var result int64

	err := db.pool.QueryRow(ctx, query, accountID).Scan(&result)
	if err != nil {
		db.logger.ErrorContext(ctx, "db: get balance from postings failed", "err", err, "acc_id", accountID)
		return 0, fmt.Errorf("db: get balance from postings failed: %w", err)
	}

	return result, nil
}
