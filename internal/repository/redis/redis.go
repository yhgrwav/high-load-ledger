package redees

import (
	"context"
	"errors"
	"fmt"
	"high-load-ledger/internal/domain/repository"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type cacheRepo struct {
	rdb    *redis.Client
	logger *slog.Logger
}

func NewCacheRepository(rdb *redis.Client, logger *slog.Logger) repository.CacheRepository {
	return &cacheRepo{
		rdb:    rdb,
		logger: logger,
	}
}

func (r *cacheRepo) SetBalance(ctx context.Context, accountID uuid.UUID, amount int64, ttl time.Duration) error {
	key := fmt.Sprintf("balance:%s", accountID.String())

	err := r.rdb.Set(ctx, key, amount, ttl).Err()
	if err != nil {
		r.logger.ErrorContext(ctx, "redis set balance error", "err", err, "account_id", accountID)
		return err
	}
	return nil
}

func (r *cacheRepo) GetBalance(ctx context.Context, accountID uuid.UUID) (int64, error) {
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

func (r *cacheRepo) DeleteBalance(ctx context.Context, accountID uuid.UUID) error {
	key := fmt.Sprintf("balance:%s", accountID.String())

	err := r.rdb.Del(ctx, key).Err()
	if err != nil {
		r.logger.ErrorContext(ctx, "redis delete balance error", err)
		return err
	}
	return nil
}

func (r *cacheRepo) SetIdempotencyKey(ctx context.Context, key uuid.UUID, response []byte, ttl time.Duration) error {
	cacheKey := fmt.Sprintf("idempotencyKey:%s", key.String())

	err := r.rdb.Set(ctx, cacheKey, response, ttl).Err()
	if err != nil {
		r.logger.ErrorContext(ctx, "redis: failed to set idempotency key",
			"err", err,
			"key", key.String(),
		)
		return fmt.Errorf("redis set idempotency: %w", err)
	}
	return nil
}

func (r *cacheRepo) GetIdempotencyKey(ctx context.Context, key uuid.UUID) ([]byte, error) {
	cacheKey := fmt.Sprintf("idempotencyKey:%s", key.String())

	result, err := r.rdb.Get(ctx, cacheKey).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, nil
		}
		r.logger.ErrorContext(ctx, "redis: failed to get idempotency key",
			"err", err,
			"key", key.String(),
		)
		return nil, fmt.Errorf("redis get idempotency: %w", err)
	}
	return result, nil
}
