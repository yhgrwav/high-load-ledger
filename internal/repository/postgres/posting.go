package postgres

import (
	"context"
	"fmt"
	"high-load-ledger/internal/domain/entity"

	"github.com/google/uuid"
)

func (db *Repository) CreatePostings(ctx context.Context, tx entity.CustomTx, postings []entity.Posting) error {
	t, err := db.castTx(ctx, tx)
	if err != nil {
		return err
	}

	query := `INSERT INTO ledger.postings (transaction_id, account_id, amount)
			  VALUES($1, $2, $3)
			  RETURNING id`

	accountLatest := make(map[uuid.UUID]int64, len(postings))

	for _, posting := range postings {
		var id int64
		if err := t.QueryRow(ctx, query, posting.TransactionID, posting.AccountID, posting.Amount).Scan(&id); err != nil {
			db.logger.ErrorContext(ctx, "db: insert posting failed", "err", err)
			return fmt.Errorf("db: insert posting failed: %w", err)
		}
		if id > accountLatest[posting.AccountID] {
			accountLatest[posting.AccountID] = id
		}
	}

	updateQuery := `UPDATE ledger.accounts
	                SET latest_posting_id = GREATEST(latest_posting_id, $1),
	                    updated_at = CURRENT_TIMESTAMP
	                WHERE user_id = $2`

	for accountID, latestID := range accountLatest {
		if _, err := t.Exec(ctx, updateQuery, latestID, accountID); err != nil {
			db.logger.ErrorContext(ctx, "db: update latest posting id failed", "err", err, "account_id", accountID)
			return fmt.Errorf("db: update latest posting id failed: %w", err)
		}
	}

	return nil
}

func (db *Repository) ListPostingsByAccountID(ctx context.Context, accountID uuid.UUID, limit, offset int) ([]entity.Posting, error) {
	query := `SELECT id, transaction_id, account_id, amount
              FROM ledger.postings
              WHERE account_id = $1
              ORDER BY id DESC
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
	return db.GetPostingsSum(ctx, accountID, 0)
}
