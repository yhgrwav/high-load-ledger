package config

import (
	"fmt"

	"github.com/caarlos0/env/v11"
)

type GRPC struct {
	Port string `env:"GRPC_PORT" envDefault:"50051"`
}

func LoadGRPC() (GRPC, error) {
	var cfg GRPC
	if err := env.Parse(&cfg); err != nil {
		return GRPC{}, fmt.Errorf("grpc config: %w", err)
	}
	return cfg, nil
}
