package redees

import (
	"context"
	"errors"
	"fmt"
	"high-load-ledger/internal/domain/entity"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

func idempotencyCacheKey(key uuid.UUID) string {
	return fmt.Sprintf("idempotencyKey:%s", key.String())
}

func (r *CacheRepo) TryGetIdempotencyKey(ctx context.Context, key uuid.UUID, ttl time.Duration) (isUsed bool, err error) {
	got, err := r.rdb.SetNX(ctx, idempotencyCacheKey(key), entity.IdempotencyInProgressMarker, ttl).Result()
	if err != nil {
		r.logger.ErrorContext(ctx, "redis: failed to get idempotency key",
			"err", err,
			"key", key.String(),
		)
		return false, fmt.Errorf("redis get idempotency: %w", err)
	}
	return got, err
}

func (r *CacheRepo) SetIdempotencyKey(ctx context.Context, key uuid.UUID, value []byte, ttl time.Duration) error {
	err := r.rdb.Set(ctx, idempotencyCacheKey(key), value, ttl).Err()
	if err != nil {
		r.logger.ErrorContext(ctx, "redis: failed to set idempotency key",
			"err", err,
			"key", key.String(),
		)
		return fmt.Errorf("redis set idempotency: %w", err)
	}
	return nil
}

func (r *CacheRepo) GetIdempotencyKey(ctx context.Context, key uuid.UUID) (entity.IdempotencyEntry, error) {
	raw, err := r.rdb.Get(ctx, idempotencyCacheKey(key)).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return entity.IdempotencyEntry{State: entity.IdempotencyUnused}, nil
		}
		r.logger.ErrorContext(ctx, "redis: failed to get idempotency key",
			"err", err,
			"key", key.String(),
		)
		return entity.IdempotencyEntry{}, fmt.Errorf("redis get idempotency: %w", err)
	}
	return entity.ParseIdempotencyValue(raw), nil
}

func (r *CacheRepo) DeleteIdempotencyKey(ctx context.Context, key uuid.UUID) error {
	err := r.rdb.Del(ctx, idempotencyCacheKey(key)).Err()
	if err != nil {
		r.logger.ErrorContext(ctx, "redis: failed to delete idempotency key",
			"err", err,
			"key", key.String(),
		)
		return fmt.Errorf("redis delete idempotency: %w", err)
	}
	return nil
}
