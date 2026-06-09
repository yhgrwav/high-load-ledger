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

	server := grpc.NewServer(
		grpc.UnaryInterceptor(interceptors.UnaryMetricsInterceptor(tel.Metrics)),
		grpc.MaxConcurrentStreams(1000),
	)

	initCtx, initCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer initCancel()

	poolCfg, err := pgxpool.ParseConfig(pgCfg.ConnectionString())
	if err != nil {
		lgr.Error("parse postgres pool config", "error", err)
		os.Exit(1)
	}

	poolCfg.MaxConns = 32
	poolCfg.MinConns = 8
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
		Addr:     redisCfg.Addr(),
		Password: redisCfg.Password,
		DB:       redisCfg.DB,
	})
	defer rdb.Close()

	if err := rdb.Ping(initCtx).Err(); err != nil {
		lgr.Error("redis ping failed", "error", err)
		os.Exit(1)
	}

	repo := postgres.NewConnectionPool(pool, lgr)
	cacheRepo := redisRepo.NewCacheRepository(rdb, lgr)

	kafkaWriter := &kafka.Writer{
		Addr:         kafka.TCP(kafkaCfg.BrokerList()...),
		Topic:        kafkaCfg.Topic,
		Balancer:     &kafka.LeastBytes{},
		RequiredAcks: kafka.RequireOne,
		Async:        false,
	}
	defer kafkaWriter.Close()

	producer := kafkainfra.NewProducer(kafkaWriter, lgr)

	transferUC := usecase.NewTransferUseCase(repo, cacheRepo, lgr, redisCfg.TransactionTTL, tel.Metrics, producer)
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
