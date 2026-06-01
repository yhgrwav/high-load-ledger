// Transfer use case tests — generated with assistance of Composer (Cursor AI).
package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"high-load-ledger/internal/domain/entity"

	"github.com/google/uuid"
)

func TestTransferUseCase_validateRequest(t *testing.T) {
	uc := NewTransferUseCase(&mockTransferRepo{}, &mockCache{}, testLogger(), 0, nil)

	fromID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	toID := uuid.MustParse("00000000-0000-0000-0000-000000000002")
	key := uuid.MustParse("00000000-0000-0000-0000-000000000099")

	tests := []struct {
		name    string
		req     entity.TransactionRequest
		wantErr error
	}{
		{
			name: "invalid amount",
			req: entity.TransactionRequest{
				IdempotencyKey: key,
				FromAccountID:  fromID,
				ToAccountID:    toID,
				Currency:       entity.CURRENCY_USD,
				Amount:         0,
			},
			wantErr: entity.ErrInvalidAmount,
		},
		{
			name: "same account",
			req: entity.TransactionRequest{
				IdempotencyKey: key,
				FromAccountID:  fromID,
				ToAccountID:    fromID,
				Currency:       entity.CURRENCY_USD,
				Amount:         100,
			},
			wantErr: entity.ErrSameAccountTransfer,
		},
		{
			name: "empty idempotency key",
			req: entity.TransactionRequest{
				FromAccountID: fromID,
				ToAccountID:   toID,
				Currency:      entity.CURRENCY_USD,
				Amount:        100,
			},
			wantErr: entity.ErrEmptyIdempotencyKey,
		},
		{
			name: "invalid currency",
			req: entity.TransactionRequest{
				IdempotencyKey: key,
				FromAccountID:  fromID,
				ToAccountID:    toID,
				Currency:       entity.CURRENCY_UNSPECIFIED,
				Amount:         100,
			},
			wantErr: entity.ErrInvalidCurrency,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := uc.validateRequest(tt.req)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("validateRequest() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestTransferUseCase_Transaction_idempotencyCacheHit(t *testing.T) {
	cachedID := uuid.MustParse("00000000-0000-0000-0000-00000000aa01")
	key := uuid.MustParse("00000000-0000-0000-0000-000000000099")

	cache := &mockCache{
		getFn: func(_ context.Context, k uuid.UUID) ([]byte, error) {
			if k != key {
				t.Fatalf("unexpected idempotency key: %v", k)
			}
			return cachedID[:], nil
		},
	}

	repo := &mockTransferRepo{
		beginTxFn: func(context.Context) (entity.CustomTx, error) {
			t.Fatal("BeginTx must not be called on cache hit")
			return nil, nil
		},
	}

	uc := NewTransferUseCase(repo, cache, testLogger(), 0, nil)

	fromID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	toID := uuid.MustParse("00000000-0000-0000-0000-000000000002")

	got, err := uc.Transaction(context.Background(), entity.TransactionRequest{
		IdempotencyKey: key,
		FromAccountID:  fromID,
		ToAccountID:    toID,
		Currency:       entity.CURRENCY_USD,
		Amount:         100,
	})
	if err != nil {
		t.Fatalf("Transaction() error = %v", err)
	}
	if got != cachedID {
		t.Fatalf("Transaction() id = %v, want %v", got, cachedID)
	}
}

func TestTransferUseCase_Transaction_insufficientFunds(t *testing.T) {
	key := uuid.MustParse("00000000-0000-0000-0000-000000000099")
	fromID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	toID := uuid.MustParse("00000000-0000-0000-0000-000000000002")

	repo := transferRepoWithAccounts(
		&entity.Account{ID: fromID, Balance: 50, Currency: entity.CURRENCY_USD},
		&entity.Account{ID: toID, Balance: 0, Currency: entity.CURRENCY_USD},
	)
	repo.debitBalanceFn = func(context.Context, entity.CustomTx, uuid.UUID, int64) error {
		return entity.ErrInsufficientFunds
	}

	uc := NewTransferUseCase(repo, &mockCache{}, testLogger(), 0, nil)

	_, err := uc.Transaction(context.Background(), entity.TransactionRequest{
		IdempotencyKey: key,
		FromAccountID:  fromID,
		ToAccountID:    toID,
		Currency:       entity.CURRENCY_USD,
		Amount:         100,
	})
	if !errors.Is(err, entity.ErrInsufficientFunds) {
		t.Fatalf("Transaction() error = %v, want %v", err, entity.ErrInsufficientFunds)
	}
}

func TestTransferUseCase_Transaction_currencyMismatch(t *testing.T) {
	key := uuid.MustParse("00000000-0000-0000-0000-000000000099")
	fromID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	toID := uuid.MustParse("00000000-0000-0000-0000-000000000002")

	repo := transferRepoWithAccounts(
		&entity.Account{ID: fromID, Balance: 1000, Currency: entity.CURRENCY_USD},
		&entity.Account{ID: toID, Balance: 0, Currency: entity.CURRENCY_EUR},
	)

	uc := NewTransferUseCase(repo, &mockCache{}, testLogger(), 0, nil)

	_, err := uc.Transaction(context.Background(), entity.TransactionRequest{
		IdempotencyKey: key,
		FromAccountID:  fromID,
		ToAccountID:    toID,
		Currency:       entity.CURRENCY_USD,
		Amount:         100,
	})
	if !errors.Is(err, entity.ErrCurrencyMismatch) {
		t.Fatalf("Transaction() error = %v, want %v", err, entity.ErrCurrencyMismatch)
	}
}

func TestTransferUseCase_Transaction_success(t *testing.T) {
	key := uuid.MustParse("00000000-0000-0000-0000-000000000099")
	fromID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	toID := uuid.MustParse("00000000-0000-0000-0000-000000000002")

	var (
		committed    bool
		cacheStored  bool
		debitAmount  int64
		creditAmount int64
	)

	repo := transferRepoWithAccounts(
		&entity.Account{ID: fromID, Balance: 1000, Currency: entity.CURRENCY_USD},
		&entity.Account{ID: toID, Balance: 200, Currency: entity.CURRENCY_USD},
	)
	repo.debitBalanceFn = func(_ context.Context, _ entity.CustomTx, _ uuid.UUID, amount int64) error {
		debitAmount = amount
		return nil
	}
	repo.creditBalanceFn = func(_ context.Context, _ entity.CustomTx, _ uuid.UUID, amount int64) error {
		creditAmount = amount
		return nil
	}
	repo.commitTxFn = func(context.Context, entity.CustomTx) error {
		committed = true
		return nil
	}

	cache := &mockCache{
		setFn: func(_ context.Context, k uuid.UUID, _ []byte, _ time.Duration) error {
			if k == key {
				cacheStored = true
			}
			return nil
		},
	}

	uc := NewTransferUseCase(repo, cache, testLogger(), time.Minute, nil)

	got, err := uc.Transaction(context.Background(), entity.TransactionRequest{
		IdempotencyKey: key,
		FromAccountID:  fromID,
		ToAccountID:    toID,
		Currency:       entity.CURRENCY_USD,
		Amount:         100,
	})
	if err != nil {
		t.Fatalf("Transaction() error = %v", err)
	}
	if got == uuid.Nil {
		t.Fatal("Transaction() returned nil id")
	}
	if !committed {
		t.Fatal("expected transaction commit")
	}
	if !cacheStored {
		t.Fatal("expected idempotency key stored in cache")
	}
	if debitAmount != 100 {
		t.Fatalf("debit amount = %d, want 100", debitAmount)
	}
	if creditAmount != 100 {
		t.Fatalf("credit amount = %d, want 100", creditAmount)
	}
}

func TestTransferUseCase_Transaction_duplicateCreateReturnsExisting(t *testing.T) {
	key := uuid.MustParse("00000000-0000-0000-0000-000000000099")
	fromID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	toID := uuid.MustParse("00000000-0000-0000-0000-000000000002")
	existingID := uuid.MustParse("00000000-0000-0000-0000-00000000bb01")

	repo := transferRepoWithAccounts(
		&entity.Account{ID: fromID, Balance: 1000, Currency: entity.CURRENCY_USD},
		&entity.Account{ID: toID, Balance: 0, Currency: entity.CURRENCY_USD},
	)
	repo.createTransactionFn = func(context.Context, entity.CustomTx, *entity.Transaction) error {
		return errors.New("duplicate idempotency key")
	}
	repo.checkIdempotencyFn = func(context.Context, uuid.UUID) (*entity.Transaction, error) {
		return &entity.Transaction{ID: existingID}, nil
	}

	uc := NewTransferUseCase(repo, &mockCache{}, testLogger(), time.Minute, nil)

	got, err := uc.Transaction(context.Background(), entity.TransactionRequest{
		IdempotencyKey: key,
		FromAccountID:  fromID,
		ToAccountID:    toID,
		Currency:       entity.CURRENCY_USD,
		Amount:         100,
	})
	if err != nil {
		t.Fatalf("Transaction() error = %v", err)
	}
	if got != existingID {
		t.Fatalf("Transaction() id = %v, want %v", got, existingID)
	}
}

func TestTransferUseCase_validateTransferCurrencies(t *testing.T) {
	fromID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	toID := uuid.MustParse("00000000-0000-0000-0000-000000000002")

	uc := NewTransferUseCase(transferRepoWithAccounts(
		&entity.Account{ID: fromID, Currency: entity.CURRENCY_USD},
		&entity.Account{ID: toID, Currency: entity.CURRENCY_EUR},
	), &mockCache{}, testLogger(), 0, nil)

	err := uc.validateTransferCurrencies(context.Background(), fromID, toID, entity.CURRENCY_USD)
	if !errors.Is(err, entity.ErrCurrencyMismatch) {
		t.Fatalf("validateTransferCurrencies() error = %v, want %v", err, entity.ErrCurrencyMismatch)
	}
}

func transferRepoWithAccounts(from, to *entity.Account) *mockTransferRepo {
	lookup := func(id uuid.UUID) (*entity.Account, error) {
		switch id {
		case from.ID:
			return from, nil
		case to.ID:
			return to, nil
		default:
			return nil, entity.ErrAccountNotFound
		}
	}

	return &mockTransferRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*entity.Account, error) {
			return lookup(id)
		},
		getForUpdateFn: func(_ context.Context, _ entity.CustomTx, id uuid.UUID) (*entity.Account, error) {
			if id == from.ID {
				return from, nil
			}
			return nil, entity.ErrAccountNotFound
		},
		getInTxFn: func(_ context.Context, _ entity.CustomTx, id uuid.UUID) (*entity.Account, error) {
			return lookup(id)
		},
	}
}
