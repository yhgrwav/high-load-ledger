package service

import (
	"math/rand"

	gen "high-load-ledger/gen/go"
)

type TransferJob struct {
	Kind     string
	Currency gen.Currency
	From     []byte
	To       []byte
	Amount   int64
}

type TransferBuilder struct {
	pool AccountPool
	rng  *rand.Rand
}

func NewTransferBuilder(pool AccountPool, rng *rand.Rand) *TransferBuilder {
	return &TransferBuilder{pool: pool, rng: rng}
}

func (b *TransferBuilder) BuildValid() (TransferJob, bool) {
	from, to, ok := b.pickFundedPair()
	if !ok {
		return TransferJob{}, false
	}

	amount := int64(1)
	if from.Balance > 1 && b.rng.Int63n(4) == 0 {
		amount = b.rng.Int63n(from.Balance-1) + 1
	}

	return TransferJob{
		Kind:     "valid",
		Currency: from.Currency,
		From:     from.ID[:],
		To:       to.ID[:],
		Amount:   amount,
	}, true
}

func (b *TransferBuilder) BuildInvalidBalance() (TransferJob, bool) {
	from, to, ok := b.pickSameCurrencyPair()
	if !ok {
		return TransferJob{}, false
	}

	amount := from.Balance + 1
	if amount < 1 {
		amount = 1_000_000_000_000
	}

	return TransferJob{
		Kind:     "invalid_balance",
		Currency: from.Currency,
		From:     from.ID[:],
		To:       to.ID[:],
		Amount:   amount,
	}, true
}

func (b *TransferBuilder) BuildInvalidCurrency() (TransferJob, bool) {
	from, to, ok := b.pickDifferentCurrencyPair()
	if !ok {
		return TransferJob{}, false
	}

	return TransferJob{
		Kind:     "invalid_currency",
		Currency: from.Currency,
		From:     from.ID[:],
		To:       to.ID[:],
		Amount:   1,
	}, true
}

func (b *TransferBuilder) pickFundedPair() (from, to ExistingAccount, ok bool) {
	candidates := b.pool.CurrenciesWithMinAccounts(2)
	if len(candidates) == 0 {
		return ExistingAccount{}, ExistingAccount{}, false
	}

	for attempt := 0; attempt < len(candidates)*2; attempt++ {
		curr := candidates[b.rng.Intn(len(candidates))]
		accounts := b.pool[curr]
		fromIdx, toIdx := randomDistinctIndexes(b.rng, len(accounts))
		from = accounts[fromIdx]
		to = accounts[toIdx]
		if from.Balance > 0 && from.ID != to.ID {
			return from, to, true
		}
	}

	return ExistingAccount{}, ExistingAccount{}, false
}

func (b *TransferBuilder) pickSameCurrencyPair() (from, to ExistingAccount, ok bool) {
	candidates := b.pool.CurrenciesWithMinAccounts(2)
	if len(candidates) == 0 {
		return ExistingAccount{}, ExistingAccount{}, false
	}

	curr := candidates[b.rng.Intn(len(candidates))]
	accounts := b.pool[curr]
	fromIdx, toIdx := randomDistinctIndexes(b.rng, len(accounts))
	return accounts[fromIdx], accounts[toIdx], true
}

func (b *TransferBuilder) pickDifferentCurrencyPair() (from, to ExistingAccount, ok bool) {
	withAccounts := b.pool.CurrenciesWithAccounts()
	if len(withAccounts) < 2 {
		return ExistingAccount{}, ExistingAccount{}, false
	}

	fromCurr := withAccounts[b.rng.Intn(len(withAccounts))]
	var toCurr gen.Currency
	for attempt := 0; attempt < 8; attempt++ {
		candidate := withAccounts[b.rng.Intn(len(withAccounts))]
		if candidate != fromCurr {
			toCurr = candidate
			break
		}
	}
	if toCurr == 0 {
		return ExistingAccount{}, ExistingAccount{}, false
	}

	from = b.pool[fromCurr][b.rng.Intn(len(b.pool[fromCurr]))]
	to = b.pool[toCurr][b.rng.Intn(len(b.pool[toCurr]))]
	return from, to, true
}

func randomDistinctIndexes(rng *rand.Rand, size int) (int, int) {
	i := rng.Intn(size)
	j := rng.Intn(size - 1)
	if j >= i {
		j++
	}
	return i, j
}
