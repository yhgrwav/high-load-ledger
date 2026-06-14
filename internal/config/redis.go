package config

import (
	"fmt"
	"time"

	"github.com/caarlos0/env/v11"
)

type Redis struct {
	Host                string        `env:"REDIS_HOST"`
	Port                string        `env:"REDIS_PORT" envDefault:"6379"`
	Password            string        `env:"REDIS_PASSWORD"`
	DB                  int           `env:"REDIS_DB" envDefault:"0"`
	PoolSize            int           `env:"REDIS_POOL_SIZE" envDefault:"500"`
	MinIdleConns        int           `env:"REDIS_MIN_IDLE_CONNS" envDefault:"50"`
	PoolTimeout         time.Duration `env:"REDIS_POOL_TIMEOUT" envDefault:"4s"`
	IdempotencyPoolSize int           `env:"REDIS_IDEMPOTENCY_POOL_SIZE" envDefault:"1024"`
	IdempotencyMinIdle  int           `env:"REDIS_IDEMPOTENCY_MIN_IDLE" envDefault:"64"`
	TransactionTTL      time.Duration `env:"REDIS_TRANSACTION_TTL" envDefault:"67m"`
}

func LoadRedis() (Redis, error) {
	var cfg Redis
	if err := env.Parse(&cfg); err != nil {
		return Redis{}, fmt.Errorf("redis config: %w", err)
	}
	return cfg, nil
}

func (r Redis) Addr() string {
	return fmt.Sprintf("%s:%s", r.Host, r.Port)
}
