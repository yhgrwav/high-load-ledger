package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"

	"high-load-ledger/internal/config"
	"high-load-ledger/internal/infra/logger"
	"high-load-ledger/internal/infra/telemetry"
	"high-load-ledger/internal/repository/postgres"
	"high-load-ledger/internal/usecase"
)

func main() {
	envPaths := []string{
		".env",
		"../../.env",
	}
	loaded := false
	for _, path := range envPaths {
		if err := godotenv.Load(path); err == nil {
			log.Printf("Loaded .env from %s", path)
			loaded = true
			break
		}
	}
	if !loaded {
		log.Print("No .env file found in any of the expected locations")
	}

	cfg, err := config.NewConfig()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}
	if cfg.PostingWorkerName == "" {
		log.Fatal("POSTING_WORKER_NAME is required")
	}

	lgr := logger.New(cfg.LogLevel, cfg.AddSource, cfg.IsJSON)
	lgr.Info("Posting worker starting...", "name", cfg.PostingWorkerName, "host", cfg.DBHost)

	tel := telemetry.New(cfg, *lgr)

	initCtx, initCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer initCancel()

	dsn := cfg.DSN
	if dsn == "" {
		dsn = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
			cfg.SuperUser, cfg.SuperUserPass, cfg.DBHost, cfg.DBPort, cfg.DBName, cfg.DBSSLMode)
	}

	pool, err := pgxpool.New(initCtx, dsn)
	if err != nil {
		lgr.Error("Unable to create connection pool", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := pool.Ping(initCtx); err != nil {
		lgr.Error("Database is unreachable", "error", err, "dsn", dsn)
		os.Exit(1)
	}

	repo := postgres.NewConnectionPool(pool, lgr)

	postingWorker, err := usecase.NewPostingWorker(
		repo,
		repo,
		repo,
		lgr,
		tel.Metrics,
		cfg.PostingWorkerName,
		cfg.PostingWorkerBatchSize,
		cfg.PostingWorkerBackoff,
	)
	if err != nil {
		lgr.Error("failed to create posting worker", "error", err)
		os.Exit(1)
	}

	errCh := make(chan error, 1)
	tel.Start(errCh)

	appCtx, appCancel := context.WithCancel(context.Background())
	defer appCancel()

	go postingWorker.Run(appCtx)
	lgr.Info("posting worker is running", "name", cfg.PostingWorkerName)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-errCh:
		lgr.Error("Critical error occurred, shutting down", "error", err)
	case sig := <-quit:
		lgr.Info("Received shutdown signal", "signal", sig.String())
	}

	lgr.Info("Shutting down posting worker gracefully...")
	appCancel()

	if err := tel.Stop(context.Background()); err != nil {
		lgr.Error("failed to shutdown telemetry", "error", err)
	}

	lgr.Info("Posting worker stopped completely")
}
