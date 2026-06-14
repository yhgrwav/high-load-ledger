package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/segmentio/kafka-go"
	"google.golang.org/grpc"

	ledger "high-load-ledger/gen/go"
	"high-load-ledger/internal/config"
	kafkainfra "high-load-ledger/internal/infra/kafka"
	"high-load-ledger/internal/infra/logger"
	"high-load-ledger/internal/infra/telemetry"
	"high-load-ledger/internal/repository/postgres"
	redisRepo "high-load-ledger/internal/repository/redis"
	transport "high-load-ledger/internal/transport/grpc"
	"high-load-ledger/internal/transport/grpc/interceptors"
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
	redisCfg, err := config.LoadRedis()
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	kafkaCfg, err := config.LoadKafkaProducer()
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	grpcCfg, err := config.LoadGRPC()
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	telCfg, err := config.LoadTelemetry()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	lgr := logger.New(logCfg.Environment, logCfg.Level, logCfg.AddSource, logCfg.IsJSON)
	lgr.Info("gateway starting", "db_host", pgCfg.Host, "grpc_port", grpcCfg.Port)

	tel := telemetry.New(telCfg, *lgr)

	initCtx, initCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer initCancel()

	poolCfg, err := pgxpool.ParseConfig(pgCfg.ConnectionString())
	if err != nil {
		lgr.Error("parse postgres pool config", "error", err)
		os.Exit(1)
	}

	poolCfg.MaxConns = pgCfg.MaxConns
	poolCfg.MinConns = pgCfg.MinConns
	poolCfg.MaxConnLifetime = 30 * time.Minute
	poolCfg.MaxConnIdleTime = 5 * time.Minute
	poolCfg.HealthCheckPeriod = 30 * time.Second

	pool, err := pgxpool.NewWithConfig(initCtx, poolCfg)
	if err != nil {
		lgr.Error("create postgres pool", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := pool.Ping(initCtx); err != nil {
		lgr.Error("postgres ping failed", "error", err)
		os.Exit(1)
	}

	rdb := redis.NewClient(&redis.Options{
		Addr:            redisCfg.Addr(),
		Password:        redisCfg.Password,
		DB:              redisCfg.DB,
		PoolSize:        redisCfg.PoolSize,
		MinIdleConns:    redisCfg.MinIdleConns,
		PoolTimeout:     redisCfg.PoolTimeout,
		ConnMaxLifetime: 30 * time.Minute,
		ConnMaxIdleTime: 5 * time.Minute,
		MaxRetries:      2,
	})
	defer func() {
		_ = rdb.Close()
	}()

	icfg, err := config.LoadIdempotency()
	if err != nil {
		lgr.Error("loading idempotency config", "error", err)
		os.Exit(1)

	}

	irdb := redis.NewClient(&redis.Options{
		Addr:            icfg.Addr(),
		Password:        icfg.Password,
		DB:              icfg.DB,
		PoolSize:        icfg.PoolSize,
		MinIdleConns:    icfg.MinIdleConns,
		PoolTimeout:     icfg.PoolTimeout,
		ConnMaxLifetime: 30 * time.Minute,
		ConnMaxIdleTime: 5 * time.Minute,
		MaxRetries:      2,
	})
	defer func() {
		_ = irdb.Close()
	}()

	idempotencyRepo := redisRepo.NewIdempotencyRepository(irdb, lgr)

	if err := irdb.Ping(initCtx).Err(); err != nil {
		lgr.Error("idempotency redis ping failed", "error", err)
		os.Exit(1)
	}
	warmRedisPool(initCtx, irdb, icfg.MinIdleConns, lgr)

	if err := rdb.Ping(initCtx).Err(); err != nil {
		lgr.Error("redis ping failed", "error", err)
		os.Exit(1)
	}
	warmRedisPool(initCtx, rdb, redisCfg.MinIdleConns, lgr)

	repo := postgres.NewConnectionPool(pool, lgr)
	cacheRepo := redisRepo.NewCacheRepository(rdb, lgr)
	chain := []grpc.UnaryServerInterceptor{
		interceptors.UnaryMetricsInterceptor(tel.Metrics),
		interceptors.UnaryIdempotencyInterceptor(idempotencyRepo, lgr),
	}

	server := grpc.NewServer(
		grpc.ChainUnaryInterceptor(chain...),
		grpc.MaxConcurrentStreams(1000),
	)

	kafkaWriter := &kafka.Writer{
		Addr:                   kafka.TCP(kafkaCfg.BrokerList()...),
		Topic:                  kafkaCfg.Topic,
		Balancer:               &kafka.LeastBytes{},
		RequiredAcks:           kafka.RequireOne,
		Async:                  true,
		AllowAutoTopicCreation: true,
		BatchSize:              100,
		BatchTimeout:           5 * time.Millisecond,
	}
	defer func() {
		_ = kafkaWriter.Close()
	}()

	producer := kafkainfra.NewProducer(kafkaWriter, lgr)

	transferUC := usecase.NewTransferUseCase(repo, cacheRepo, lgr, tel.Metrics, producer)
	accountUC := usecase.NewAccountUseCase(repo, cacheRepo, lgr)
	statsUC := usecase.NewStatsUseCase(repo, cacheRepo, lgr)

	handler := transport.NewHandler(transferUC, accountUC, statsUC, lgr)

	ledger.RegisterTransactionServiceServer(server, handler)
	ledger.RegisterAccountServiceServer(server, handler)
	ledger.RegisterStatsServiceServer(server, handler)

	lis, err := net.Listen("tcp", ":"+grpcCfg.Port)
	if err != nil {
		lgr.Error("listen failed", "error", err)
		os.Exit(1)
	}

	errCh := make(chan error, 1)
	tel.Start(errCh)

	go func() {
		lgr.Info("gRPC server running", "port", grpcCfg.Port)
		if err := server.Serve(lis); err != nil {
			errCh <- fmt.Errorf("gRPC server: %w", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-errCh:
		lgr.Error("shutdown after error", "error", err)
	case sig := <-quit:
		lgr.Info("shutdown signal", "signal", sig.String())
	}

	lgr.Info("shutting down gateway")
	server.GracefulStop()

	if err := tel.Stop(context.Background()); err != nil {
		lgr.Error("telemetry shutdown failed", "error", err)
	}

	lgr.Info("gateway stopped")
}

func warmRedisPool(ctx context.Context, rdb *redis.Client, n int, lgr *slog.Logger) {
	if n <= 1 {
		return
	}

	var wg sync.WaitGroup
	wg.Add(n)
	for range n {
		go func() {
			defer wg.Done()
			if err := rdb.Ping(ctx).Err(); err != nil {
				lgr.Warn("redis pool warmup ping failed", "error", err)
			}
		}()
	}
	wg.Wait()
}
