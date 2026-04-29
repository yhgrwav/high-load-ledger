package config

import (
	"fmt"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

type Config struct {
	SuperUser     string `env:"DB_SUPER_USER, required"`
	SuperUserPass string `env:"DB_SUPER_PASSWORD, required"`

	DBUser string `env:"DB_USER, required"`
	DBPass string `env:"DB_PASSWORD, required"`

	DBName    string `env:"DB_NAME, required"`
	DBHost    string `env:"DB_HOST, required"`
	DBPort    string `env:"DB_PORT, required"`
	DBSSLMode string `env:"DB_SSL_MODE, required"`
	DSN       string `env:"DSN, required"`

	RedisHost     string `env:"REDIS_HOST, required"`
	RedisPort     string `env:"REDIS_PORT, required"`
	RedisPassword string `env:"REDIS_PASSWORD, required"`
	RedisDB       int    `env:"REDIS_DB" envDefault:"0"`

	GRPCPort string `env:"GRPC_PORT" envDefault:"50051"`

	LogLevel  string `env:"LOG_LEVEL" envDefault:"info"`
	AddSource bool   `env:"ADD_SOURCE" envDefault:"true"`
	IsJSON    bool   `env:"IS_JSON" envDefault:"true"`
}

func NewConfig() (*Config, error) {
	err := godotenv.Load(".env")
	if err != nil {
		fmt.Println("Error loading .env file")
	}

	var cfg Config

	if err := env.Parse(&cfg); err != nil {
		return nil, fmt.Errorf("error parsing config: %w", err)
	}

	return &Config{}, nil
}
