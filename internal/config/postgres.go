package config

import (
	"fmt"

	"github.com/caarlos0/env/v11"
)

type Postgres struct {
	SuperUser     string `env:"DB_SUPER_USER"`
	SuperUserPass string `env:"DB_SUPER_PASSWORD"`
	User          string `env:"DB_USER"`
	Password      string `env:"DB_PASSWORD"`
	Name          string `env:"DB_NAME"`
	Host          string `env:"DB_HOST"`
	Port          string `env:"DB_PORT"`
	SSLMode       string `env:"DB_SSL_MODE"`
	MaxConns      int32  `env:"DB_MAX_CONNS" envDefault:"400"`
	MinConns      int32  `env:"DB_MIN_CONNS" envDefault:"50"`
	DSN           string `env:"DSN"`
}

func LoadPostgres() (Postgres, error) {
	var cfg Postgres
	if err := env.Parse(&cfg); err != nil {
		return Postgres{}, fmt.Errorf("postgres config: %w", err)
	}
	return cfg, nil
}

func (p Postgres) ConnectionString() string {
	if p.DSN != "" {
		return p.DSN
	}
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		p.SuperUser, p.SuperUserPass, p.Host, p.Port, p.Name, p.SSLMode)
}
