package config

import (
	"time"

	"github.com/caarlos0/env/v11"
)

type Config struct {
	Working            bool          `env:"LOAD_GEN_WORKING" envDefault:"false"`
	UsersAmount        int           `env:"USERS_AMOUNT" envDefault:"1000"`
	ValidRPS           float64       `env:"VALID_RPS" envDefault:"100"`
	InvalidRPS         float64       `env:"INVALID_RPS" envDefault:"10"`
	InvalidCurrencyRPS float64       `env:"INVALID_CURRENCY_RPS" envDefault:"5"`
	LoadDuration       time.Duration `env:"LOAD_DURATION" envDefault:"0"`
	BootstrapWorkers   int           `env:"LOAD_BOOTSTRAP_WORKERS" envDefault:"50"`
	TxWorkers          int           `env:"LOAD_TX_WORKERS" envDefault:"100"`
	BootstrapMaxError  int           `env:"BOOTSTRAP_MAX_ERROR_PCT" envDefault:"33"`
	GRPCAddr           string        `env:"LOADGEN_GRPC_ADDR" envDefault:"127.0.0.1:8085"`
	MetricsPort        string        `env:"LOADGEN_METRICS_PORT" envDefault:"9092"`
}

func Load() (*Config, error) {
	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
