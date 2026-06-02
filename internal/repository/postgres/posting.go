package postgres

import (
	"context"
	"fmt"
	"high-load-ledger/internal/domain/entity"

	"github.com/google/uuid"
)

func (db *Repository) CreatePostings(ctx context.Context, tx entity.CustomTx, postings []entity.Posting) error {
	tr, err := db.castTx(ctx, tx)
	if err != nil {
		return err
	}

	var query string
	var args []any

	// нашел идеальный вариант реализации этого метода, в общем логика завязывается на длине postings,
	// т.е. если нам прилетает всего лишь один posting - делаем единичную вставку, если длина 2 - делаем
	// двойную вставку. Мне к сожалению не хватило мозгов так красиво расписать эту идею в коде, но в целом
	// 2 промпта всё исправили и я получил буквально то, о чём мечтал в реализации этого метода.
	// по итогу этот метод является хорошей точкой оптимизации основного метода транзакции, т.к. мы теперь
	// записываем 2 постинга за один сетевой вызов, а как мне сказал мой опытный знакомый - главная проблема
	// сложных систем - сетевые задержки, соответственно нужно избегать лишних запросов, соединений и тд.
	// вообще по сути своей если бы мой проект запускался по канону на разных серверах, в разных уголках планеты
	// с разным пингом - у меня бы и 200 рпс на один инстанс врядли вышел бы, но пока я буду с нулевой задержкой
	// пытаться высосать из этого кода хотя бы 600 рпс на локалке, что звучит вполне возможно.

	switch len(postings) {
	case 1:
		query = `
			WITH inserted AS (
				INSERT INTO ledger.postings (transaction_id, account_id, amount)
				VALUES ($1, $2, $3)
				RETURNING id, account_id
			)
			UPDATE ledger.accounts a
			SET latest_posting_id = GREATEST(a.latest_posting_id, i.id),
			    updated_at = CURRENT_TIMESTAMP
			FROM inserted i
			WHERE a.user_id = i.account_id`
		args = []any{postings[0].TransactionID, postings[0].AccountID, postings[0].Amount}
	case 2:
		query = `
			WITH inserted AS (
				INSERT INTO ledger.postings (transaction_id, account_id, amount)
				VALUES ($1, $2, $3), ($4, $5, $6)
				RETURNING id, account_id
			)
			UPDATE ledger.accounts a
			SET latest_posting_id = GREATEST(a.latest_posting_id, i.id),
			    updated_at = CURRENT_TIMESTAMP
			FROM inserted i
			WHERE a.user_id = i.account_id`
		args = []any{
			postings[0].TransactionID, postings[0].AccountID, postings[0].Amount,
			postings[1].TransactionID, postings[1].AccountID, postings[1].Amount,
		}
	default:
		return nil
	}

	if _, err := tr.Exec(ctx, query, args...); err != nil {
		db.logger.ErrorContext(ctx, "db: create postings failed", "err", err)
		return fmt.Errorf("db: create postings failed: %w", err)
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
