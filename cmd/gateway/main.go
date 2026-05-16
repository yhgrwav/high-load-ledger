package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"

	ledger "high-load-ledger/gen/go"
	"high-load-ledger/internal/config"
	"high-load-ledger/internal/infra/logger"
	"high-load-ledger/internal/infra/telemetry"
	"high-load-ledger/internal/repository/postgres"
	redisRepo "high-load-ledger/internal/repository/redis"
	transport "high-load-ledger/internal/transport/grpc"
	"high-load-ledger/internal/transport/grpc/interceptors"
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

	lgr := logger.New(cfg.LogLevel, cfg.AddSource, cfg.IsJSON)
	lgr.Info("Ledger starting...", "user", cfg.DBUser, "host", cfg.DBHost)

	tel := telemetry.New(cfg, *lgr)

	server := grpc.NewServer(
		grpc.UnaryInterceptor(interceptors.UnaryMetricsInterceptor(*tel.Metrics)),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dsn := cfg.DSN
	if dsn == "" {
		dsn = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
			cfg.SuperUser, cfg.SuperUserPass, cfg.DBHost, cfg.DBPort, cfg.DBName, cfg.DBSSLMode)
	}

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		lgr.Error("Unable to create connection pool", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		lgr.Error("Database is unreachable", "error", err, "dsn", dsn)
		os.Exit(1)
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", cfg.RedisHost, cfg.RedisPort),
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})
	defer rdb.Close()

	if err := rdb.Ping(ctx).Err(); err != nil {
		lgr.Error("Redis is unreachable", "error", err)
		os.Exit(1)
	}

	repo := postgres.NewConnectionPool(pool, lgr)

	cacheRepo := redisRepo.NewCacheRepository(rdb, lgr)

	transferUC := usecase.NewTransferUseCase(repo, cacheRepo, lgr, cfg.RedisTransactionTTL)
	accountUC := usecase.NewAccountUseCase(repo, lgr)

	handler := transport.NewHandler(transferUC, accountUC, lgr)

	ledger.RegisterTransactionServiceServer(server, handler)
	ledger.RegisterAccountServiceServer(server, handler)
	ledger.RegisterStatsServiceServer(server, handler)

	lis, err := net.Listen("tcp", ":"+cfg.GRPCPort)
	if err != nil {
		lgr.Error("failed to listen", "error", err)
		os.Exit(1)
	}

	errCh := make(chan error, 1)

	tel.Start(errCh)

	go func() {
		lgr.Info("gRPC server is running", "port", cfg.GRPCPort)
		if err := server.Serve(lis); err != nil {
			errCh <- fmt.Errorf("gRPC server failed: %w", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-errCh:
		lgr.Error("Critical error occurred, shutting down", "error", err)
	case sig := <-quit:
		lgr.Info("Received shutdown signal", "signal", sig.String())
	}

	lgr.Info("Shutting down servers gracefully...")

	server.GracefulStop()
	lgr.Info("gRPC server stopped")

	if err := tel.Stop(context.Background()); err != nil {
		lgr.Error("failed to shutdown telemetry", "error", err)
	}

	lgr.Info("Ledger stopped completely")
}
