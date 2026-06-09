package service

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/google/uuid"

	gen "high-load-ledger/gen/go"
	loadgenconfig "high-load-ledger/loadgen/config"
)

type CoreService struct {
	cfg         *loadgenconfig.Config
	tx          *TxManager
	acc         *AccountService
	stats       *LoadStats
	metrics     *Metrics
	recentTxIDs []uuid.UUID
	txIDsMu     sync.RWMutex
}

func NewCoreService(cfg *loadgenconfig.Config, tx *TxManager, acc *AccountService, metrics *Metrics) *CoreService {
	return &CoreService{
		cfg:         cfg,
		tx:          tx,
		acc:         acc,
		stats:       &LoadStats{},
		metrics:     metrics,
		recentTxIDs: make([]uuid.UUID, 0, 10000),
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
	c.metrics.SetTarget(StreamStats, c.cfg.StatsRPS)

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

	log.Println("loadgen: waiting for upstream to be ready...")
	client := gen.NewAccountServiceClient(c.acc.conn)
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		_, err := client.CreateAccount(ctx, &gen.CreateAccountRequest{Currency: currencies[0]})
		if err == nil {
			break
		}
		time.Sleep(2 * time.Second)
	}
	log.Println("loadgen: upstream is ready, starting bootstrap")

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
				if job.Kind == StreamStats {
					err := c.tx.GetTransaction(ctx, job.TxID)
					c.stats.RecordCompleted(err)
					c.metrics.RecordCompleted(job.Kind, err)
				} else {
					txID, err := c.tx.CreateTx(ctx, job.Currency, job.From, job.To, job.Amount)
					c.stats.RecordCompleted(err)
					c.metrics.RecordCompleted(job.Kind, err)

					if job.Kind == StreamValid && err == nil && txID != uuid.Nil {
						c.txIDsMu.Lock()
						c.recentTxIDs = append(c.recentTxIDs, txID)
						if len(c.recentTxIDs) > 10000 {
							c.recentTxIDs = c.recentTxIDs[1000:]
						}
						c.txIDsMu.Unlock()
					}
				}
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
		{StreamStats, c.cfg.StatsRPS, c.buildStats},
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
		"loadgen: load phase running valid=%.0f/s invalid_balance=%.0f/s invalid_currency=%.0f/s stats=%.0f/s workers=%d",
		c.cfg.ValidRPS, c.cfg.InvalidRPS, c.cfg.InvalidCurrencyRPS, c.cfg.StatsRPS, c.cfg.TxWorkers,
	)

	<-ctx.Done()

	streamsWg.Wait()
	close(jobs)
	workersWg.Wait()
}

func (c *CoreService) buildStats() (TransferJob, bool) {
	c.txIDsMu.RLock()
	defer c.txIDsMu.RUnlock()
	if len(c.recentTxIDs) == 0 {
		return TransferJob{}, false
	}
	idx := rand.Intn(len(c.recentTxIDs))
	return TransferJob{
		Kind: StreamStats,
		TxID: c.recentTxIDs[idx],
	}, true
}
