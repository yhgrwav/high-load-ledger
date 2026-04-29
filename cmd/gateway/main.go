package main

import (
	"context"
	"high-load-ledger/internal/config"
	"high-load-ledger/internal/infra/logger"
	"high-load-ledger/internal/repository/postgres"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	log.Println("Ledger started...")

	cfg, err := config.NewConfig()
	if err != nil {
		log.Fatal(err) // os.Exit(1) w no defer
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	lgr := logger.New(cfg.LogLevel, cfg.AddSource, cfg.IsJSON)

	pool, err := pgxpool.New(ctx, cfg.DSN)
	if err != nil {
		lgr.Error("Unable to create connection pool", "error", err)
		os.Exit(1)
	}

	if err := pool.Ping(ctx); err != nil {
		lgr.Error("Database is unreachable", "error", err)
		os.Exit(1)
	}

	defer pool.Close()

	repo := postgres.NewConnectionPool(pool, lgr)

	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", cfg.RedisHost, cfg.RedisPort),
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})

	if err := rdb.Ping(ctx).Err(); err != nil {
		lgr.Error("Redis is unreachable", "error", err)
		os.Exit(1)
	}

	redees.NewCacheRepository(rdb, lgr)

}
