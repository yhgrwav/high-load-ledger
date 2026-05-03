package postgres

import (
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
}

func NewConnectionPool(pool *pgxpool.Pool, logger *slog.Logger) *Repository {
	return &Repository{
		pool:   pool,
		logger: logger,
	}
}
