package service

import (
	"sync"

	gen "high-load-ledger/gen/go"

	"github.com/google/uuid"
)

type ExistingAccount struct {
	ID       uuid.UUID
	Currency gen.Currency
	Balance  int64
}

// AccountPool is shared by schedulers (read/build) and workers (balance updates).
type AccountPool struct {
	mu       sync.RWMutex
	accounts map[gen.Currency][]ExistingAccount
	index    map[uuid.UUID]accountRef
}

type accountRef struct {
	currency gen.Currency
	idx      int
}

func NewAccountPool() *AccountPool {
	return &AccountPool{
		accounts: make(map[gen.Currency][]ExistingAccount),
		index:    make(map[uuid.UUID]accountRef),
	}
}

func GetValidCurrencies() []gen.Currency {
	result := make([]gen.Currency, 0, len(gen.Currency_value))
	for _, value := range gen.Currency_value {
		if value == int32(gen.Currency_CURRENCY_UNSPECIFIED) {
			continue
		}
		result = append(result, gen.Currency(value))
	}
	return result
}

func (p *AccountPool) Add(acc ExistingAccount) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.accounts[acc.Currency] = append(p.accounts[acc.Currency], acc)
	p.index[acc.ID] = accountRef{currency: acc.Currency, idx: len(p.accounts[acc.Currency]) - 1}
}

func (p *AccountPool) Total() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	total := 0
	for _, accounts := range p.accounts {
		total += len(accounts)
	}
	return total
}

func (p *AccountPool) CurrenciesWithMinAccounts(min int) []gen.Currency {
	p.mu.RLock()
	defer p.mu.RUnlock()
	var out []gen.Currency
	for currency, accounts := range p.accounts {
		if len(accounts) >= min {
			out = append(out, currency)
		}
	}
	return out
}

func (p *AccountPool) CurrenciesWithAccounts() []gen.Currency {
	p.mu.RLock()
	defer p.mu.RUnlock()
	var out []gen.Currency
	for currency, accounts := range p.accounts {
		if len(accounts) > 0 {
			out = append(out, currency)
		}
	}
	return out
}

func (p *AccountPool) snapshotCurrency(currency gen.Currency) []ExistingAccount {
	p.mu.RLock()
	defer p.mu.RUnlock()
	src := p.accounts[currency]
	out := make([]ExistingAccount, len(src))
	copy(out, src)
	return out
}

// ReserveValid deducts amount from fromID and credits toID in the local model.
// Call before dispatch so subsequent builders see updated balances.
func (p *AccountPool) ReserveValid(fromID, toID uuid.UUID, amount int64) bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	fromRef, okFrom := p.index[fromID]
	toRef, okTo := p.index[toID]
	if !okFrom || !okTo {
		return false
	}
	from := &p.accounts[fromRef.currency][fromRef.idx]
	to := &p.accounts[toRef.currency][toRef.idx]
	if from.Balance < amount {
		return false
	}
	from.Balance -= amount
	to.Balance += amount
	return true
}

// RollbackValid undoes ReserveValid after a failed/rejected transfer.
func (p *AccountPool) RollbackValid(fromID, toID uuid.UUID, amount int64) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if fromRef, ok := p.index[fromID]; ok {
		p.accounts[fromRef.currency][fromRef.idx].Balance += amount
	}
	if toRef, ok := p.index[toID]; ok {
		p.accounts[toRef.currency][toRef.idx].Balance -= amount
	}
}

func (p *AccountPool) BalanceOf(id uuid.UUID) (int64, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	ref, ok := p.index[id]
	if !ok {
		return 0, false
	}
	return p.accounts[ref.currency][ref.idx].Balance, true
}
