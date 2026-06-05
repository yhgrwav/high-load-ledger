package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"high-load-ledger/internal/config"
	"high-load-ledger/internal/infra/logger"
	"high-load-ledger/internal/infra/telemetry"
	"high-load-ledger/internal/repository/postgres"
	"high-load-ledger/internal/usecase"
)

func main() {
	config.LoadDotEnv()

	logCfg, err := config.LoadLog()
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	pgCfg, err := config.LoadPostgres()
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	workerCfg, err := config.LoadPostingWorker()
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	telCfg, err := config.LoadTelemetry()
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	if workerCfg.Name == "" {
		log.Fatal("POSTING_WORKER_NAME is required")
	}

	lgr := logger.New(logCfg.Environment, logCfg.Level, logCfg.AddSource, logCfg.IsJSON)
	lgr.Info("posting worker starting", "name", workerCfg.Name, "db_host", pgCfg.Host)

	tel := telemetry.New(telCfg, *lgr)

	initCtx, initCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer initCancel()

	pool, err := pgxpool.New(initCtx, pgCfg.ConnectionString())
	if err != nil {
		lgr.Error("create postgres pool", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := pool.Ping(initCtx); err != nil {
		lgr.Error("postgres ping failed", "error", err, "dsn", pgCfg.ConnectionString())
		os.Exit(1)
	}

	repo := postgres.NewConnectionPool(pool, lgr)

	postingWorker, err := usecase.NewPostingWorker(
		repo,
		repo,
		repo,
		lgr,
		tel.Metrics,
		workerCfg.Name,
		workerCfg.BatchSize,
		workerCfg.Backoff,
	)
	if err != nil {
		lgr.Error("create posting worker", "error", err)
		os.Exit(1)
	}

	errCh := make(chan error, 1)
	tel.Start(errCh)

	appCtx, appCancel := context.WithCancel(context.Background())
	defer appCancel()

	go postingWorker.Run(appCtx)
	lgr.Info("posting worker running", "name", workerCfg.Name)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-errCh:
		lgr.Error("shutdown after error", "error", err)
	case sig := <-quit:
		lgr.Info("shutdown signal", "signal", sig.String())
	}

	lgr.Info("shutting down posting worker")
	appCancel()

	if err := tel.Stop(context.Background()); err != nil {
		lgr.Error("telemetry shutdown failed", "error", err)
	}

	lgr.Info("posting worker stopped")
}
