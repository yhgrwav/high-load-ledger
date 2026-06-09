package config

import (
	"fmt"
	"time"

	"github.com/caarlos0/env/v11"
)

type PostingWorker struct {
	Name      string        `env:"POSTING_WORKER_NAME" envDefault:"balance_verifier"`
	BatchSize int           `env:"POSTING_WORKER_BATCH_SIZE" envDefault:"100"`
	Backoff   time.Duration `env:"POSTING_WORKER_BACKOFF" envDefault:"5s"`
}

func LoadPostingWorker() (PostingWorker, error) {
	var cfg PostingWorker
	if err := env.Parse(&cfg); err != nil {
		return PostingWorker{}, fmt.Errorf("posting worker config: %w", err)
	}
	return cfg, nil
}
