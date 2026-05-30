package service

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"

	loadgenconfig "high-load-ledger/loadgen/config"
)

type CoreService struct {
	cfg     *loadgenconfig.Config
	tx      *TxManager
	acc     *AccountService
	stats   *LoadStats
	metrics *Metrics
}

func NewCoreService(cfg *loadgenconfig.Config, tx *TxManager, acc *AccountService, metrics *Metrics) *CoreService {
	return &CoreService{
		cfg:     cfg,
		tx:      tx,
		acc:     acc,
		stats:   &LoadStats{},
		metrics: metrics,
	}
}

func (c *CoreService) LoadGenWorker(ctx context.Context) {
	if !c.cfg.Working {
		log.Println("loadgen: disabled (LOAD_GEN_WORKING=false)")
		return
	}

	c.metrics.SetTarget(StreamValid, c.cfg.ValidRPS)
	c.metrics.SetTarget(StreamInvalidBalance, c.cfg.InvalidRPS)
	c.metrics.SetTarget(StreamInvalidCurrency, c.cfg.InvalidCurrencyRPS)

	log.Println("loadgen: started")
	defer log.Println("loadgen: finished")

	pool, err := c.bootstrapAccounts(ctx)
	if err != nil {
		log.Printf("loadgen: bootstrap failed: %v", err)
		return
	}
	if pool.Total() < 2 {
		log.Println("loadgen: not enough accounts for transfer load")
		return
	}

	log.Printf("loadgen: bootstrap done, accounts=%d", pool.Total())

	loadCtx := ctx
	if c.cfg.LoadDuration > 0 {
		var cancel context.CancelFunc
		loadCtx, cancel = context.WithTimeout(ctx, c.cfg.LoadDuration)
		defer cancel()
		log.Printf("loadgen: load phase duration=%s", c.cfg.LoadDuration)
	}

	c.runLoad(loadCtx, pool)
}

func (c *CoreService) bootstrapAccounts(ctx context.Context) (AccountPool, error) {
	pool := make(AccountPool)
	currencies := GetValidCurrencies()
	if len(currencies) == 0 {
		return nil, fmt.Errorf("no currencies in proto enum")
	}

	perCurrency := c.cfg.UsersAmount / len(currencies)
	if perCurrency == 0 {
		perCurrency = 1
	}

	for _, curr := range currencies {
		accounts, err := c.acc.CreateAccounts(ctx, curr, perCurrency, c.cfg.BootstrapMaxError)
		if err != nil {
			log.Printf("loadgen: create accounts currency=%v: %v", curr, err)
		}
		if len(accounts) > 0 {
			pool[curr] = append(pool[curr], accounts...)
		}
	}

	if pool.Total() == 0 {
		return pool, fmt.Errorf("no accounts created")
	}

	return pool, nil
}

func (c *CoreService) runLoad(ctx context.Context, pool AccountPool) {
	builder := NewTransferBuilder(pool, rand.New(rand.NewSource(time.Now().UnixNano())))
	jobs := make(chan TransferJob, 4096)

	go c.stats.RunReporter(ctx, c.cfg)

	var workersWg sync.WaitGroup
	for i := 0; i < c.cfg.TxWorkers; i++ {
		workersWg.Add(1)
		go func() {
			defer workersWg.Done()
			for job := range jobs {
				_, err := c.tx.CreateTx(ctx, job.Currency, job.From, job.To, job.Amount)
				c.stats.RecordCompleted(err)
				c.metrics.RecordCompleted(job.Kind, err)
			}
		}()
	}

	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				c.metrics.SetQueueDepth(len(jobs))
			}
		}
	}()

	streams := []struct {
		stream string
		rps    float64
		build  func() (TransferJob, bool)
	}{
		{StreamValid, c.cfg.ValidRPS, builder.BuildValid},
		{StreamInvalidBalance, c.cfg.InvalidRPS, builder.BuildInvalidBalance},
		{StreamInvalidCurrency, c.cfg.InvalidCurrencyRPS, builder.BuildInvalidCurrency},
	}

	var streamsWg sync.WaitGroup
	for _, s := range streams {
		streamsWg.Add(1)
		go func(stream string, rps float64, build func() (TransferJob, bool)) {
			defer streamsWg.Done()
			runPoissonStream(ctx, stream, rps, jobs, build, c.metrics, c.stats)
		}(s.stream, s.rps, s.build)
	}

	log.Printf(
		"loadgen: load phase running valid=%.0f/s invalid_balance=%.0f/s invalid_currency=%.0f/s workers=%d",
		c.cfg.ValidRPS, c.cfg.InvalidRPS, c.cfg.InvalidCurrencyRPS, c.cfg.TxWorkers,
	)

	<-ctx.Done()

	streamsWg.Wait()
	close(jobs)
	workersWg.Wait()
}
