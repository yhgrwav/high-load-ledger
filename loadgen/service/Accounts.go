package service

import (
	"context"
	"fmt"
	"log"
	"sync"

	gen "high-load-ledger/gen/go"

	"github.com/google/uuid"
	"google.golang.org/grpc"
)

type AccountService struct {
	conn    *grpc.ClientConn
	workers int
}

func NewAccountService(conn *grpc.ClientConn, workers int) *AccountService {
	return &AccountService{
		conn:    conn,
		workers: workers,
	}
}

func (a *AccountService) CreateAccounts(ctx context.Context, curr gen.Currency, amount int, maxErrorPct int) ([]ExistingAccount, error) {
	if amount <= 0 {
		return nil, nil
	}

	client := gen.NewAccountServiceClient(a.conn)
	req := &gen.CreateAccountRequest{Currency: curr}

	jobs := make(chan struct{}, amount)
	for i := 0; i < amount; i++ {
		jobs <- struct{}{}
	}
	close(jobs)

	results := make(chan ExistingAccount, amount)
	errorsCh := make(chan struct{}, amount)

	var wg sync.WaitGroup
	workerCount := a.workers
	if workerCount > amount {
		workerCount = amount
	}

	for w := 0; w < workerCount; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range jobs {
				resp, err := client.CreateAccount(ctx, req)
				if err != nil {
					errorsCh <- struct{}{}
					log.Printf("account gen: create account currency=%v: %v", curr, err)
					continue
				}

				id, err := uuid.FromBytes(resp.AccountId)
				if err != nil {
					errorsCh <- struct{}{}
					log.Printf("account gen: parse account id currency=%v: %v", curr, err)
					continue
				}

				balanceResp, err := client.GetBalance(ctx, &gen.GetBalanceRequest{
					AccountId:   id[:],
					RequesterId: id[:],
				})
				if err != nil {
					errorsCh <- struct{}{}
					log.Printf("account gen: get balance currency=%v account=%v: %v", curr, id, err)
					continue
				}

				results <- ExistingAccount{
					ID:       id,
					Currency: curr,
					Balance:  balanceResp.Balance,
				}
			}
		}()
	}

	wg.Wait()
	close(results)
	close(errorsCh)

	created := make([]ExistingAccount, 0, amount)
	for account := range results {
		created = append(created, account)
	}

	errorsCount := len(errorsCh)
	maxErrors := amount * maxErrorPct / 100
	if maxErrors < 1 && amount > 0 {
		maxErrors = 1
	}
	if errorsCount > maxErrors {
		return created, fmt.Errorf("too many account creation errors: %d of %d (max %d%%)", errorsCount, amount, maxErrorPct)
	}

	return created, nil
}
