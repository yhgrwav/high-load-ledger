package config

import (
	"fmt"

	"github.com/caarlos0/env/v11"
)

type Log struct {
	Environment string `env:"ENVIRONMENT" envDefault:"development"`
	Level       string `env:"LOG_LEVEL"`
	AddSource   *bool  `env:"ADD_SOURCE"`
	IsJSON      *bool  `env:"IS_JSON"`
}

func LoadLog() (Log, error) {
	var cfg Log
	if err := env.Parse(&cfg); err != nil {
		return Log{}, fmt.Errorf("log config: %w", err)
	}
	return cfg, nil
}
