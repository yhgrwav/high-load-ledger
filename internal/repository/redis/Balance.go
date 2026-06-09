package redees

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

func (r *CacheRepo) SetBalance(ctx context.Context, accountID uuid.UUID, amount int64, ttl time.Duration) error {
	key := fmt.Sprintf("balance:%s", accountID.String())

	err := r.rdb.Set(ctx, key, amount, ttl).Err()
	if err != nil {
		r.logger.ErrorContext(ctx, "redis set balance error", "err", err, "account_id", accountID)
		return err
	}
	return nil
}

func (r *CacheRepo) GetBalance(ctx context.Context, accountID uuid.UUID) (int64, error) {
	key := fmt.Sprintf("balance:%s", accountID.String())

	result, err := r.rdb.Get(ctx, key).Int64()
	if err != nil {
		if err == redis.Nil {
			return 0, nil
		}
		r.logger.ErrorContext(ctx, "redis get balance error", "err", err, "account_id", accountID)
		return 0, err
	}
	return result, nil
}

func (r *CacheRepo) DeleteBalance(ctx context.Context, accountID uuid.UUID) error {
	key := fmt.Sprintf("balance:%s", accountID.String())

	err := r.rdb.Del(ctx, key).Err()
	if err != nil {
		r.logger.ErrorContext(ctx, "redis delete balance error", err)
		return err
	}
	return nil
}
