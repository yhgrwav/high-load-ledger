// Posting worker tests — generated with assistance of Composer (Cursor AI).
package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestNewPostingWorker_requiresName(t *testing.T) {
	_, err := NewPostingWorker(
		&mockPostingWorkerRepo{},
		&mockPostingWorkerPool{},
		&mockPostingCursor{},
		testLogger(),
		nil,
		"",
		100,
		0,
	)
	if err == nil {
		t.Fatal("NewPostingWorker() expected error for empty name")
	}
}

func TestNewPostingWorker_defaults(t *testing.T) {
	w, err := NewPostingWorker(
		&mockPostingWorkerRepo{},
		&mockPostingWorkerPool{},
		&mockPostingCursor{},
		testLogger(),
		nil,
		"balance_verifier",
		0,
		0,
	)
	if err != nil {
		t.Fatalf("NewPostingWorker() error = %v", err)
	}
	if w.batchSize != defaultBatchSize {
		t.Fatalf("batchSize = %d, want %d", w.batchSize, defaultBatchSize)
	}
	if w.backoffTime != defaultBackoff {
		t.Fatalf("backoffTime = %v, want %v", w.backoffTime, defaultBackoff)
	}
}

func TestPostingWorker_ValidateBalance_matchWithinBatch(t *testing.T) {
	accountID := uuid.MustParse("00000000-0000-0000-0000-000000000001")

	repo := &mockPostingWorkerRepo{
		getAccountBalanceSnapshotFn: func(context.Context, uuid.UUID) (int64, int64, error) {
			return 500, 80, nil
		},
		getPostingsSumFn: func(_ context.Context, _ uuid.UUID, limitID int64) (int64, error) {
			if limitID != 100 {
				t.Fatalf("expected limitID 100, got %d", limitID)
			}
			return 500, nil
		},
	}

	w, err := NewPostingWorker(repo, &mockPostingWorkerPool{}, &mockPostingCursor{}, testLogger(), nil, "test", 100, 0)
	if err != nil {
		t.Fatalf("NewPostingWorker() error = %v", err)
	}

	if err := w.ValidateBalance(context.Background(), accountID, 100); err != nil {
		t.Fatalf("ValidateBalance() error = %v", err)
	}
}

func TestPostingWorker_ValidateBalance_adaptiveBoundary(t *testing.T) {
	accountID := uuid.MustParse("00000000-0000-0000-0000-000000000001")

	repo := &mockPostingWorkerRepo{
		getAccountBalanceSnapshotFn: func(context.Context, uuid.UUID) (int64, int64, error) {
			return 750, 125, nil
		},
		getPostingsSumFn: func(_ context.Context, _ uuid.UUID, limitID int64) (int64, error) {
			if limitID != 125 {
				t.Fatalf("expected adaptive limitID 125, got %d", limitID)
			}
			return 750, nil
		},
	}

	w, err := NewPostingWorker(repo, &mockPostingWorkerPool{}, &mockPostingCursor{}, testLogger(), nil, "test", 100, 0)
	if err != nil {
		t.Fatalf("NewPostingWorker() error = %v", err)
	}

	if err := w.ValidateBalance(context.Background(), accountID, 100); err != nil {
		t.Fatalf("ValidateBalance() error = %v", err)
	}
}

func TestPostingWorker_ValidateBalance_triggersCorrection(t *testing.T) {
	accountID := uuid.MustParse("00000000-0000-0000-0000-000000000001")

	var corrected bool

	repo := &mockPostingWorkerRepo{
		getAccountBalanceSnapshotFn: func(context.Context, uuid.UUID) (int64, int64, error) {
			return 999, 50, nil
		},
		getPostingsSumFn: func(_ context.Context, _ uuid.UUID, limitID int64) (int64, error) {
			if limitID == 0 {
				return 500, nil
			}
			return 500, nil
		},
		applyBalanceCorrectionFn: func(_ context.Context, id uuid.UUID, amount int64) error {
			corrected = true
			if id != accountID {
				t.Fatalf("unexpected account id: %v", id)
			}
			if amount != 500 {
				t.Fatalf("correction amount = %d, want 500", amount)
			}
			return nil
		},
	}

	pool := &mockPostingWorkerPool{
		readWriteFn: func(_ context.Context, fn func(context.Context) error) error {
			return fn(context.Background())
		},
	}

	w, err := NewPostingWorker(repo, pool, &mockPostingCursor{}, testLogger(), nil, "test", 100, 0)
	if err != nil {
		t.Fatalf("NewPostingWorker() error = %v", err)
	}

	if err := w.ValidateBalance(context.Background(), accountID, 100); err != nil {
		t.Fatalf("ValidateBalance() error = %v", err)
	}
	if !corrected {
		t.Fatal("expected balance correction")
	}
}

func TestPostingWorker_ValidateBalance_contextCanceled(t *testing.T) {
	accountID := uuid.MustParse("00000000-0000-0000-0000-000000000001")

	w, err := NewPostingWorker(&mockPostingWorkerRepo{}, &mockPostingWorkerPool{}, &mockPostingCursor{}, testLogger(), nil, "test", 100, 0)
	if err != nil {
		t.Fatalf("NewPostingWorker() error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err = w.ValidateBalance(ctx, accountID, 100)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("ValidateBalance() error = %v, want %v", err, context.Canceled)
	}
}

func TestPostingWorker_waitBackoff(t *testing.T) {
	w, err := NewPostingWorker(&mockPostingWorkerRepo{}, &mockPostingWorkerPool{}, &mockPostingCursor{}, testLogger(), nil, "test", 100, time.Millisecond)
	if err != nil {
		t.Fatalf("NewPostingWorker() error = %v", err)
	}

	timer := time.NewTimer(0)
	if !timer.Stop() {
		<-timer.C
	}
	defer timer.Stop()

	if !w.waitBackoff(context.Background(), timer) {
		t.Fatal("waitBackoff() expected true on timer fire")
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if w.waitBackoff(ctx, timer) {
		t.Fatal("waitBackoff() expected false on canceled context")
	}
}
