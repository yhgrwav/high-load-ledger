package redees

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"high-load-ledger/internal/domain/entity"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

func (r *CacheRepo) SetTransaction(ctx context.Context, tx *entity.Transaction, ttl time.Duration) error {
	if tx == nil {
		return fmt.Errorf("redis set transaction: nil transaction")
	}

	payload, err := json.Marshal(tx)
	if err != nil {
		return fmt.Errorf("redis set transaction: marshal: %w", err)
	}

	key := transactionCacheKey(tx.ID)
	if err := r.rdb.Set(ctx, key, payload, ttl).Err(); err != nil {
		r.logger.ErrorContext(ctx, "redis set transaction error", "err", err, "transaction_id", tx.ID)
		return fmt.Errorf("redis set transaction: %w", err)
	}
	return nil
}

func (r *CacheRepo) GetTransaction(ctx context.Context, id uuid.UUID) (*entity.Transaction, error) {
	data, err := r.rdb.Get(ctx, transactionCacheKey(id)).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, nil
		}
		r.logger.ErrorContext(ctx, "redis get transaction error", "err", err, "transaction_id", id)
		return nil, fmt.Errorf("redis get transaction: %w", err)
	}

	var tx entity.Transaction
	if err := json.Unmarshal(data, &tx); err != nil {
		return nil, fmt.Errorf("redis get transaction: unmarshal: %w", err)
	}
	return &tx, nil
}

func transactionCacheKey(id uuid.UUID) string {
	return fmt.Sprintf("transaction:%s", id.String())
}
