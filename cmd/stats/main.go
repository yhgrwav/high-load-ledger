package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/segmentio/kafka-go"

	"high-load-ledger/internal/config"
	kafkainfra "high-load-ledger/internal/infra/kafka"
	"high-load-ledger/internal/infra/logger"
	redisRepo "high-load-ledger/internal/repository/redis"
	"high-load-ledger/internal/usecase"
)

func main() {
	config.LoadDotEnv()

	logCfg, err := config.LoadLog()
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	redisCfg, err := config.LoadRedis()
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	kafkaCfg, err := config.LoadKafkaConsumer()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	lgr := logger.New(logCfg.Environment, logCfg.Level, logCfg.AddSource, logCfg.IsJSON)
	lgr.Info("stats worker starting",
		"brokers", kafkaCfg.Brokers,
		"topic", kafkaCfg.Topic,
		"group", kafkaCfg.GroupID,
	)

	initCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	rdb := redis.NewClient(&redis.Options{
		Addr:     redisCfg.Addr(),
		Password: redisCfg.Password,
		DB:       redisCfg.DB,
	})
	defer rdb.Close()

	if err := rdb.Ping(initCtx).Err(); err != nil {
		lgr.Error("redis ping failed", "err", err)
		os.Exit(1)
	}

	cacheRepo := redisRepo.NewCacheRepository(rdb, lgr)
	projector := usecase.NewTransactionCacheProjector(cacheRepo, redisCfg.TransactionTTL, lgr)

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: kafkaCfg.BrokerList(),
		Topic:   kafkaCfg.Topic,
		GroupID: kafkaCfg.GroupID,
	})
	defer reader.Close()

	consumer := kafkainfra.NewConsumer(reader, lgr)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := consumer.Run(ctx, projector.HandleKafkaMessage); err != nil && ctx.Err() == nil {
		lgr.Error("stats worker stopped", "err", err)
		os.Exit(1)
	}

	lgr.Info("stats worker stopped")
}
