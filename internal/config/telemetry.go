package config

import (
	"fmt"

	"github.com/caarlos0/env/v11"
)

type Telemetry struct {
	ServiceName string `env:"SERVICE_NAME" envDefault:"ledger"`
	MetricsPort string `env:"METRICS_PORT" envDefault:"6767"`
}

func LoadTelemetry() (Telemetry, error) {
	var cfg Telemetry
	if err := env.Parse(&cfg); err != nil {
		return Telemetry{}, fmt.Errorf("telemetry config: %w", err)
	}
	return cfg, nil
}
