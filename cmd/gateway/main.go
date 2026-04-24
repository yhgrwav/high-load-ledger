package main

import (
	"context"
	"high-load-ledger/internal/infra/logger"
	"log"
)

func main() {
	log.Println("Ledger started...")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := config.NewConfig()

	l := logger.New(cfg.logLevel, cfg.addSource, cfg.IsJSON)

}
