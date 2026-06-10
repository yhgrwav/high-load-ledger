package redis

import (
	"context"
	"errors"
	"fmt"
	"high-load-ledger/internal/domain/entity"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

func (r *CacheRepo) SetAccountCurrency(ctx context.Context, accountID uuid.UUID, currency entity.Currency, ttl time.Duration) error {
	key := fmt.Sprintf("currency:%s", accountID.String())
	err := r.rdb.Set(ctx, key, strconv.Itoa(int(currency)), ttl).Err()
	if err != nil {
		r.logger.ErrorContext(ctx, "redis set currency error", "err", err, "account_id", accountID)
		return err
	}
	return nil
}

func (r *CacheRepo) GetAccountCurrency(ctx context.Context, accountID uuid.UUID) (entity.Currency, error) {
	key := fmt.Sprintf("currency:%s", accountID.String())
	result, err := r.rdb.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return entity.CURRENCY_UNSPECIFIED, nil
		}
		r.logger.ErrorContext(ctx, "redis get currency error", "err", err, "account_id", accountID)
		return entity.CURRENCY_UNSPECIFIED, err
	}
	res, err := strconv.Atoi(result)
	if err != nil {
		// Старый/битый формат в кэше — считаем cache miss и идём в БД.
		return entity.CURRENCY_UNSPECIFIED, nil
	}
	return entity.Currency(res), nil
}
