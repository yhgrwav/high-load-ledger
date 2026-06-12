package redis

import (
	"context"
	"fmt"
	"high-load-ledger/internal/domain/entity"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const idempotencyKeyTTL = 10 * time.Minute

// key - uuid
// value - status

// UUID -> formatted string
func castKey(idempotencyKey uuid.UUID) string {
	return fmt.Sprintf("idempotency:%s", idempotencyKey)
}

func (r *CacheRepo) SetAndCheck(ctx context.Context, idempotencyKey uuid.UUID, status entity.IdempotencyStatus) (bool, error) {
	exists, err := r.rdb.SetNX(ctx, castKey(idempotencyKey), status, idempotencyKeyTTL).Result()
	if err != nil {
		r.logger.ErrorContext(ctx, "redis: SetNX internal error", err)
		return false, err
	}
	return exists, err
}

func (r *CacheRepo) GetIdempotencyStatus(ctx context.Context, idempotencyKey uuid.UUID) (entity.IdempotencyStatus, error) {
	val, err := r.rdb.Get(ctx, castKey(idempotencyKey)).Result()
	if err == redis.Nil {
		return entity.IDEMPOTENCY_MISS, nil
	}
	if err != nil {
		return entity.IDEMPOTENCY_STATUS_UNSPECIFIED, err
	}
	return entity.IdempotencyStatus(val), nil
}

func (r *CacheRepo) UpdateIdempotencyStatus(ctx context.Context, idempotencyKey uuid.UUID, status entity.IdempotencyStatus) error {
	return r.rdb.Set(ctx, castKey(idempotencyKey), status, idempotencyKeyTTL).Err()
}

func (r *CacheRepo) DeleteIdempotency(ctx context.Context, idempotencyKey uuid.UUID) error {
	return r.rdb.Del(ctx, castKey(idempotencyKey)).Err()
}
