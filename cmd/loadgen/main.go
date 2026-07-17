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

	connCount := cfg.GRPCConnections
	if connCount < 1 {
		connCount = 1
	}

	// k8sdns:/// (loadgen/service/k8sresolver.go) + round_robin: a plain "host:port" target
	// resolves to a single address and pins that one backend for the connection's lifetime —
	// behind a k8s ClusterIP, that's one pod, permanently, no matter how many gRPC connections
	// we open (kube-proxy picks the backend once per TCP connection, not per RPC). We use a
	// custom resolver instead of grpc-go's built-in "dns" scheme because that one only
	// re-resolves on an explicit ResolveNow() (triggered internally on connection errors) — on
	// the happy path it resolves once and blocks forever, so it never notices pods an HPA adds
	// later. k8sdns polls on a fixed timer instead, so round_robin keeps rebalancing as the
	// gateway HPA scales gateway-headless up or down.
	const roundRobinServiceConfig = `{"loadBalancingConfig": [{"round_robin":{}}]}`

	conns := make([]*grpc.ClientConn, 0, connCount)
	for i := 0; i < connCount; i++ {
		conn, err := grpc.NewClient(
			service.PeriodicDNSScheme+":///"+cfg.GRPCAddr,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithDefaultCallOptions(grpc.WaitForReady(true)),
			grpc.WithDefaultServiceConfig(roundRobinServiceConfig),
		)
		if err != nil {
			log.Fatalf("grpc dial %d/%d: %v", i+1, connCount, err)
		}
		conns = append(conns, conn)
	}
	defer func() {
		for _, conn := range conns {
			_ = conn.Close()
		}
	}()
	log.Printf("loadgen: grpc connections=%d addr=%s", connCount, cfg.GRPCAddr)

	acc := service.NewAccountService(conns[0], cfg.BootstrapWorkers)
	tx := service.NewTxManager(conns...)
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
