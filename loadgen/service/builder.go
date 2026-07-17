package service

import (
	"math"
	"math/rand"

	"github.com/google/uuid"

	gen "high-load-ledger/gen/go"
)

// Always-invalid amount for the insufficient-funds stream (independent of local balance).
const invalidBalanceAmount int64 = math.MaxInt64 / 4

type TransferJob struct {
	Kind     string
	Currency gen.Currency
	From     []byte
	To       []byte
	Amount   int64
	TxID     uuid.UUID
	// Reserved marks that AccountPool.ReserveValid already applied this job locally.
	Reserved bool
}

type TransferBuilder struct {
	pool *AccountPool
	rng  *rand.Rand
}

func NewTransferBuilder(pool *AccountPool, rng *rand.Rand) *TransferBuilder {
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

	if !b.pool.ReserveValid(from.ID, to.ID, amount) {
		return TransferJob{}, false
	}

	return TransferJob{
		Kind:     StreamValid,
		Currency: from.Currency,
		From:     from.ID[:],
		To:       to.ID[:],
		Amount:   amount,
		Reserved: true,
	}, true
}

func (b *TransferBuilder) BuildInvalidBalance() (TransferJob, bool) {
	from, to, ok := b.pickSameCurrencyPair()
	if !ok {
		return TransferJob{}, false
	}

	return TransferJob{
		Kind:     StreamInvalidBalance,
		Currency: from.Currency,
		From:     from.ID[:],
		To:       to.ID[:],
		Amount:   invalidBalanceAmount,
	}, true
}

func (b *TransferBuilder) BuildInvalidCurrency() (TransferJob, bool) {
	from, to, ok := b.pickDifferentCurrencyPair()
	if !ok {
		return TransferJob{}, false
	}

	return TransferJob{
		Kind:     StreamInvalidCurrency,
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

	for attempt := 0; attempt < len(candidates)*4; attempt++ {
		curr := candidates[b.rng.Intn(len(candidates))]
		accounts := b.pool.snapshotCurrency(curr)
		if len(accounts) < 2 {
			continue
		}
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
	accounts := b.pool.snapshotCurrency(curr)
	if len(accounts) < 2 {
		return ExistingAccount{}, ExistingAccount{}, false
	}
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

	fromAccounts := b.pool.snapshotCurrency(fromCurr)
	toAccounts := b.pool.snapshotCurrency(toCurr)
	if len(fromAccounts) == 0 || len(toAccounts) == 0 {
		return ExistingAccount{}, ExistingAccount{}, false
	}
	from = fromAccounts[b.rng.Intn(len(fromAccounts))]
	to = toAccounts[b.rng.Intn(len(toAccounts))]
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
