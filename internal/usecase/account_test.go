// Account use case tests — generated with assistance of Composer (Cursor AI).
package usecase

import (
	"context"
	"errors"
	"testing"

	"high-load-ledger/internal/domain/entity"

	"github.com/google/uuid"
)

func TestAccountUseCase_CreateAccount_invalidCurrency(t *testing.T) {
	uc := NewAccountUseCase(&mockAccountRepo{}, &mockCache{}, testLogger())

	_, err := uc.CreateAccount(context.Background(), entity.CURRENCY_UNSPECIFIED)
	if !errors.Is(err, entity.ErrInvalidCurrency) {
		t.Fatalf("CreateAccount() error = %v, want %v", err, entity.ErrInvalidCurrency)
	}
}

func TestAccountUseCase_CreateAccount_beginTxError(t *testing.T) {
	repo := &mockAccountRepo{
		beginTxFn: func(context.Context) (entity.CustomTx, error) {
			return nil, errors.New("db unavailable")
		},
	}

	uc := NewAccountUseCase(repo, &mockCache{}, testLogger())

	_, err := uc.CreateAccount(context.Background(), entity.CURRENCY_USD)
	if err == nil {
		t.Fatal("CreateAccount() expected error")
	}
}

func TestAccountUseCase_CreateAccount_success(t *testing.T) {
	var (
		accountCreated bool
		committed      bool
	)

	repo := &mockAccountRepo{
		createAccountFn: func(_ context.Context, _ entity.CustomTx, acc *entity.Account) error {
			accountCreated = true
			if acc.Currency != entity.CURRENCY_USD {
				t.Fatalf("unexpected currency: %v", acc.Currency)
			}
			return nil
		},
		commitTxFn: func(context.Context, entity.CustomTx) error {
			committed = true
			return nil
		},
	}

	uc := NewAccountUseCase(repo, &mockCache{}, testLogger())

	id, err := uc.CreateAccount(context.Background(), entity.CURRENCY_USD)
	if err != nil {
		t.Fatalf("CreateAccount() error = %v", err)
	}
	if id == uuid.Nil {
		t.Fatal("CreateAccount() returned nil id")
	}
	if !accountCreated {
		t.Fatal("expected CreateAccount repo call")
	}
	if !committed {
		t.Fatal("expected transaction commit")
	}
}

func TestAccountUseCase_GetBalance(t *testing.T) {
	accountID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	expected := &entity.Account{ID: accountID, Balance: 500, Currency: entity.CURRENCY_USD}

	repo := &mockAccountRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*entity.Account, error) {
			if id != accountID {
				t.Fatalf("unexpected account id: %v", id)
			}
			return expected, nil
		},
	}

	uc := NewAccountUseCase(repo, &mockCache{}, testLogger())

	got, err := uc.GetBalance(context.Background(), accountID)
	if err != nil {
		t.Fatalf("GetBalance() error = %v", err)
	}
	if got != expected {
		t.Fatalf("GetBalance() = %+v, want %+v", got, expected)
	}
}
