package config

import (
	"fmt"
	"time"

	"github.com/caarlos0/env/v11"
)

type Idempotency struct {
	Host         string        `env:"IDEMPOTENCY_REDIS_HOST" envDefault:"127.0.0.1"`
	Port         string        `env:"IDEMPOTENCY_REDIS_PORT" envDefault:"6377"`
	Password     string        `env:"IDEMPOTENCY_REDIS_PASSWORD" envDefault:"password"`
	DB           int           `env:"IDEMPOTENCY_REDIS_DB" envDefault:"0"`
	PoolSize     int           `env:"IDEMPOTENCY_REDIS_POOL_SIZE" envDefault:"1500"`
	MinIdleConns int           `env:"IDEMPOTENCY_REDIS_MIN_IDLE_CONNS" envDefault:"200"`
	PoolTimeout  time.Duration `env:"IDEMPOTENCY_REDIS_POOL_TIMEOUT" envDefault:"4s"`
}

func LoadIdempotency() (*Idempotency, error) {
	var cfg Idempotency

	if err := env.Parse(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (i Idempotency) Addr() string {
	return fmt.Sprintf("%s:%s", i.Host, i.Port)
}
