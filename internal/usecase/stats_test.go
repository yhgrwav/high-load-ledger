// Stats use case tests — generated with assistance of Composer (Cursor AI).
package usecase

import (
	"context"
	"errors"
	"testing"

	"high-load-ledger/internal/domain/entity"

	"github.com/google/uuid"
)

func TestStatsUseCase_GetTransaction_success(t *testing.T) {
	txID := uuid.MustParse("00000000-0000-0000-0000-00000000cc01")
	expected := &entity.Transaction{ID: txID, Amount: 100, Currency: entity.CURRENCY_USD}

	repo := &mockStatsRepo{
		getTransactionByIDFn: func(_ context.Context, id uuid.UUID) (*entity.Transaction, error) {
			if id != txID {
				t.Fatalf("unexpected transaction id: %v", id)
			}
			return expected, nil
		},
	}

	uc := NewStatsUseCase(repo, testLogger())

	got, err := uc.GetTransaction(context.Background(), txID)
	if err != nil {
		t.Fatalf("GetTransaction() error = %v", err)
	}
	if got != expected {
		t.Fatalf("GetTransaction() = %+v, want %+v", got, expected)
	}
}

func TestStatsUseCase_GetTransaction_notFound(t *testing.T) {
	uc := NewStatsUseCase(&mockStatsRepo{}, testLogger())

	_, err := uc.GetTransaction(context.Background(), uuid.New())
	if !errors.Is(err, entity.ErrTransactionNotFound) {
		t.Fatalf("GetTransaction() error = %v, want %v", err, entity.ErrTransactionNotFound)
	}
}
