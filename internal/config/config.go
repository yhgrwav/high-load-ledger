package config

import (
	"fmt"
	"time"

	"github.com/caarlos0/env/v11"
)

type Config struct {
	SuperUser              string        `env:"DB_SUPER_USER"`
	SuperUserPass          string        `env:"DB_SUPER_PASSWORD"`
	DBUser                 string        `env:"DB_USER"`
	DBPass                 string        `env:"DB_PASSWORD"`
	DBName                 string        `env:"DB_NAME"`
	DBHost                 string        `env:"DB_HOST"`
	DBPort                 string        `env:"DB_PORT"`
	DBSSLMode              string        `env:"DB_SSL_MODE"`
	DSN                    string        `env:"DSN"`
	RedisHost              string        `env:"REDIS_HOST"`
	RedisPort              string        `env:"REDIS_PORT" envDefault:"6379"`
	RedisPassword          string        `env:"REDIS_PASSWORD"`
	RedisDB                int           `env:"REDIS_DB" envDefault:"0"`
	RedisTransactionTTL    time.Duration `env:"REDIS_TRANSACTION_TTL" envDefault:"67m"`
	GRPCPort               string        `env:"GRPC_PORT" envDefault:"50051"`
	LogLevel               string        `env:"LOG_LEVEL" envDefault:"info"`
	AddSource              bool          `env:"ADD_SOURCE" envDefault:"true"`
	IsJSON                 bool          `env:"IS_JSON" envDefault:"true"`
	ServiceName            string        `env:"SERVICE_NAME" envDefault:"ledger"`
	MetricsPort            string        `env:"METRICS_PORT" envDefault:"6767"`
	PostingWorkerEnabled   bool          `env:"POSTING_WORKER_ENABLED" envDefault:"true"`
	PostingWorkerName      string        `env:"POSTING_WORKER_NAME,required,notEmpty"`
	PostingWorkerBatchSize int           `env:"POSTING_WORKER_BATCH_SIZE" envDefault:"100"`
	PostingWorkerBackoff   time.Duration `env:"POSTING_WORKER_BACKOFF" envDefault:"5s"`
}

func NewConfig() (*Config, error) {
	var cfg Config

	if err := env.Parse(&cfg); err != nil {
		return nil, fmt.Errorf("error parsing config: %w", err)
	}
	return &cfg, nil
}
