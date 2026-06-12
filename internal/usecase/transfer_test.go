// Transfer use case tests — generated with assistance of Composer (Cursor AI).

package usecase

import (
	"context"

	"errors"

	"testing"

	"high-load-ledger/internal/domain/entity"

	"github.com/google/uuid"
)

func newTransferUC(repo *mockTransferRepo, cache *mockCache) *TransferUseCase {
	return NewTransferUseCase(repo, cache, testLogger(), nil, nil)
}

func TestTransferUseCase_validateRequest(t *testing.T) {

	uc := newTransferUC(&mockTransferRepo{}, &mockCache{})

	fromID := uuid.MustParse("00000000-0000-0000-0000-000000000001")

	toID := uuid.MustParse("00000000-0000-0000-0000-000000000002")

	key := uuid.MustParse("00000000-0000-0000-0000-000000000099")

	tests := []struct {
		name string

		req entity.TransactionRequest

		wantErr error
	}{

		{

			name: "invalid amount",

			req: entity.TransactionRequest{

				IdempotencyKey: key,

				FromAccountID: fromID,

				ToAccountID: toID,

				Currency: entity.CURRENCY_USD,

				Amount: 0,
			},

			wantErr: entity.ErrInvalidAmount,
		},

		{

			name: "same account",

			req: entity.TransactionRequest{

				IdempotencyKey: key,

				FromAccountID: fromID,

				ToAccountID: fromID,

				Currency: entity.CURRENCY_USD,

				Amount: 100,
			},

			wantErr: entity.ErrSameAccountTransfer,
		},

		{

			name: "empty idempotency key",

			req: entity.TransactionRequest{

				FromAccountID: fromID,

				ToAccountID: toID,

				Currency: entity.CURRENCY_USD,

				Amount: 100,
			},

			wantErr: entity.ErrEmptyIdempotencyKey,
		},

		{

			name: "invalid currency",

			req: entity.TransactionRequest{

				IdempotencyKey: key,

				FromAccountID: fromID,

				ToAccountID: toID,

				Currency: entity.CURRENCY_UNSPECIFIED,

				Amount: 100,
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

	uc := newTransferUC(repo, &mockCache{})

	_, err := uc.Transaction(context.Background(), entity.TransactionRequest{

		IdempotencyKey: key,

		FromAccountID: fromID,

		ToAccountID: toID,

		Currency: entity.CURRENCY_USD,

		Amount: 100,
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

	uc := newTransferUC(repo, &mockCache{})

	_, err := uc.Transaction(context.Background(), entity.TransactionRequest{

		IdempotencyKey: key,

		FromAccountID: fromID,

		ToAccountID: toID,

		Currency: entity.CURRENCY_USD,

		Amount: 100,
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
		committed bool

		debitAmount int64

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

	uc := newTransferUC(repo, &mockCache{})

	got, err := uc.Transaction(context.Background(), entity.TransactionRequest{

		IdempotencyKey: key,

		FromAccountID: fromID,

		ToAccountID: toID,

		Currency: entity.CURRENCY_USD,

		Amount: 100,
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

	if debitAmount != 100 {

		t.Fatalf("debit amount = %d, want 100", debitAmount)

	}

	if creditAmount != 100 {

		t.Fatalf("credit amount = %d, want 100", creditAmount)

	}

}

func TestTransferUseCase_validateTransferCurrencies(t *testing.T) {

	fromID := uuid.MustParse("00000000-0000-0000-0000-000000000001")

	toID := uuid.MustParse("00000000-0000-0000-0000-000000000002")

	uc := newTransferUC(transferRepoWithAccounts(

		&entity.Account{ID: fromID, Currency: entity.CURRENCY_USD},

		&entity.Account{ID: toID, Currency: entity.CURRENCY_EUR},
	), &mockCache{})

	err := uc.validateTransferCurrencies(context.Background(), fromID, toID, entity.CURRENCY_USD)

	if !errors.Is(err, entity.ErrCurrencyMismatch) {

		t.Fatalf("validateTransferCurrencies() error = %v, want %v", err, entity.ErrCurrencyMismatch)

	}

}

func transferRepoWithAccounts(accounts ...*entity.Account) *mockTransferRepo {

	lookup := make(map[uuid.UUID]*entity.Account, len(accounts))

	for _, acc := range accounts {

		lookup[acc.ID] = acc

	}

	return &mockTransferRepo{

		getByIDFn: func(_ context.Context, id uuid.UUID) (*entity.Account, error) {

			if acc, ok := lookup[id]; ok {

				return acc, nil

			}

			return nil, entity.ErrAccountNotFound

		},

		getCurrenciesFn: func(_ context.Context, ids []uuid.UUID) (map[uuid.UUID]entity.Currency, error) {

			res := make(map[uuid.UUID]entity.Currency)

			for _, id := range ids {

				if acc, ok := lookup[id]; ok {

					res[id] = acc.Currency

				}

			}

			return res, nil

		},
	}

}
