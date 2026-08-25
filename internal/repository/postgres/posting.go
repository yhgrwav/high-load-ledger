package postgres

import (
	"context"
	"fmt"
	"high-load-ledger/internal/domain/entity"

	"github.com/google/uuid"
)

// CreatePostings inserts the given postings and returns the generated posting id per account,
// so the caller can fold latest_posting_id into the same UPDATE that adjusts the balance
// (see DebitBalance/CreditBalance) instead of running a separate UPDATE over the same rows.
func (db *Repository) CreatePostings(ctx context.Context, tx entity.CustomTx, postings []entity.Posting) (map[uuid.UUID]int64, error) {
	tr, err := db.castTx(ctx, tx)
	if err != nil {
		return nil, err
	}

	var query string
	var args []any

	switch len(postings) {
	case 1:
		query = `INSERT INTO ledger.postings (transaction_id, account_id, amount)
			VALUES ($1, $2, $3)
			RETURNING id, account_id`
		args = []any{postings[0].TransactionID, postings[0].AccountID, postings[0].Amount}
	case 2:
		query = `INSERT INTO ledger.postings (transaction_id, account_id, amount)
			VALUES ($1, $2, $3), ($4, $5, $6)
			RETURNING id, account_id`
		args = []any{
			postings[0].TransactionID, postings[0].AccountID, postings[0].Amount,
			postings[1].TransactionID, postings[1].AccountID, postings[1].Amount,
		}
	default:
		return nil, nil
	}

	rows, err := tr.Query(ctx, query, args...)
	if err != nil {
		db.logger.ErrorContext(ctx, "db: create postings failed", "err", err)
		return nil, fmt.Errorf("db: create postings failed: %w", err)
	}
	defer rows.Close()

	ids := make(map[uuid.UUID]int64, len(postings))
	for rows.Next() {
		var id int64
		var accountID uuid.UUID
		if err := rows.Scan(&id, &accountID); err != nil {
			db.logger.ErrorContext(ctx, "db: scan created posting failed", "err", err)
			return nil, fmt.Errorf("db: scan created posting failed: %w", err)
		}
		ids[accountID] = id
	}
	if err := rows.Err(); err != nil {
		db.logger.ErrorContext(ctx, "db: rows created postings failed", "err", err)
		return nil, fmt.Errorf("db: rows created postings failed: %w", err)
	}

	return ids, nil
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
