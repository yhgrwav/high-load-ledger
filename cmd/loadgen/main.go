package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	loadgenconfig "high-load-ledger/loadgen/config"
	"high-load-ledger/loadgen/service"
)

func main() {
	envPaths := []string{".env", "../../.env"}
	for _, path := range envPaths {
		if err := godotenv.Load(path); err == nil {
			log.Printf("loaded env from %s", path)
			break
		}
	}

	cfg, err := loadgenconfig.Load()
	if err != nil {
		log.Fatalf("loadgen config: %v", err)
	}

	if !cfg.Working {
		log.Println("loadgen: LOAD_GEN_WORKING=false, exit")
		return
	}

	metrics := service.NewMetrics(cfg.MetricsPort)
	metrics.Start()
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		metrics.Stop(shutdownCtx)
	}()

	conn, err := grpc.NewClient(
		cfg.GRPCAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatalf("grpc dial: %v", err)
	}
	defer conn.Close()

	acc := service.NewAccountService(conn, cfg.BootstrapWorkers)
	tx := service.NewTxManager(conn)
	core := service.NewCoreService(cfg, tx, acc, metrics)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
		<-sigCh
		cancel()
	}()

	core.LoadGenWorker(ctx)
}
