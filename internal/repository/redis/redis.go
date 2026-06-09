package redees

import (
	"log/slog"

	"github.com/redis/go-redis/v9"
)

type CacheRepo struct {
	rdb    *redis.Client
	logger *slog.Logger
}

func NewCacheRepository(rdb *redis.Client, logger *slog.Logger) *CacheRepo {
	return &CacheRepo{
		rdb:    rdb,
		logger: logger,
	}
}
