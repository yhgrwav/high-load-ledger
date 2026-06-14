package redis

import (
	"context"
	"errors"
	"fmt"
	"high-load-ledger/internal/domain/entity"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const idempotencyKeyTTL = 10 * time.Minute

// т.к. идемпотентность у нас используется в hot path - нужно выделить отдельный инстанс для разгрузки
// пула соединений в кэше, т.к. без этого у меня вместо 3-4к рпс выдаётся 2к ошибок в секунду
type IdempotencyRepository struct {
	conn   *redis.Client
	logger *slog.Logger
}

func NewIdempotencyRepository(conn *redis.Client, logger *slog.Logger) *IdempotencyRepository {
	return &IdempotencyRepository{
		conn:   conn,
		logger: logger,
	}
}

func castKey(idempotencyKey uuid.UUID) string {
	return fmt.Sprintf("idempotency:%s", idempotencyKey.String())
}

func (r *IdempotencyRepository) CheckOrLock(ctx context.Context, key uuid.UUID) (bool, entity.IdempotencyStatus, error) {
	redisKey := castKey(key)

	result, err := r.conn.SetArgs(ctx, redisKey, string(entity.IDEMPOTENCY_IN_PROCESS), redis.SetArgs{
		Mode: "NX",
		Get:  true,
		TTL:  idempotencyKeyTTL,
	}).Result()

	if err != nil {
		if errors.Is(err, redis.Nil) {
			return true, entity.IDEMPOTENCY_IN_PROCESS, nil
		}

		r.logger.ErrorContext(ctx, "redis: internal idempotency error", "err", err, "key", key)
		return false, entity.IDEMPOTENCY_STATUS_UNSPECIFIED, err
	}

	currentStatus := entity.IdempotencyStatus(result)

	return false, currentStatus, nil
}

func (r *IdempotencyRepository) UpdateIdempotencyStatus(ctx context.Context, key uuid.UUID, status entity.IdempotencyStatus) error {
	return r.conn.Set(ctx, castKey(key), string(status), idempotencyKeyTTL).Err()
}

func (r *IdempotencyRepository) DeleteIdempotency(ctx context.Context, key uuid.UUID) error {
	return r.conn.Del(ctx, castKey(key)).Err()
}
