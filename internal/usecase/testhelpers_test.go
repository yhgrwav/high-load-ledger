// Test helpers and mocks for usecase unit tests.
// Generated with assistance of Composer (Cursor AI).
package usecase

import (
	"context"
	"io"
	"log/slog"
	"time"

	"high-load-ledger/internal/domain/entity"

	"github.com/google/uuid"
)

type mockTx struct{}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

type mockCache struct {
	getFn func(ctx context.Context, key uuid.UUID) ([]byte, error)
	setFn func(ctx context.Context, key uuid.UUID, response []byte, ttl time.Duration) error
}

func (m *mockCache) GetIdempotencyKey(ctx context.Context, key uuid.UUID) ([]byte, error) {
	if m.getFn != nil {
		return m.getFn(ctx, key)
	}
	return nil, entity.ErrTransactionNotFound
}

func (m *mockCache) SetIdempotencyKey(ctx context.Context, key uuid.UUID, response []byte, ttl time.Duration) error {
	if m.setFn != nil {
		return m.setFn(ctx, key, response, ttl)
	}
	return nil
}

func (m *mockCache) SetBalance(context.Context, uuid.UUID, int64, time.Duration) error { return nil }
func (m *mockCache) GetBalance(context.Context, uuid.UUID) (int64, error)              { return 0, nil }
func (m *mockCache) DeleteBalance(context.Context, uuid.UUID) error                    { return nil }
func (m *mockCache) SetAccountCurrency(context.Context, uuid.UUID, entity.Currency, time.Duration) error {
	return nil
}
func (m *mockCache) GetAccountCurrency(context.Context, uuid.UUID) (entity.Currency, error) {
	return entity.CURRENCY_UNSPECIFIED, nil
}

type mockTransferRepo struct {
	beginTxFn            func(ctx context.Context) (entity.CustomTx, error)
	commitTxFn           func(ctx context.Context, tx entity.CustomTx) error
	rollbackTxFn         func(ctx context.Context, tx entity.CustomTx) error
	getByIDFn            func(ctx context.Context, id uuid.UUID) (*entity.Account, error)
	getCurrenciesFn      func(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]entity.Currency, error)
	debitBalanceFn       func(ctx context.Context, tx entity.CustomTx, id uuid.UUID, amount int64) error
	creditBalanceFn      func(ctx context.Context, tx entity.CustomTx, id uuid.UUID, amount int64) error
	createTransactionFn  func(ctx context.Context, tx entity.CustomTx, tr *entity.Transaction) error
	createPostingsFn     func(ctx context.Context, tx entity.CustomTx, postings []entity.Posting) error
	checkIdempotencyFn   func(ctx context.Context, key uuid.UUID) (uuid.UUID, error)
	getTransactionByIDFn func(ctx context.Context, id uuid.UUID) (*entity.Transaction, error)
}

func (m *mockTransferRepo) BeginTx(ctx context.Context) (entity.CustomTx, error) {
	if m.beginTxFn != nil {
		return m.beginTxFn(ctx)
	}
	return mockTx{}, nil
}

func (m *mockTransferRepo) CommitTx(ctx context.Context, tx entity.CustomTx) error {
	if m.commitTxFn != nil {
		return m.commitTxFn(ctx, tx)
	}
	return nil
}

func (m *mockTransferRepo) RollbackTx(ctx context.Context, tx entity.CustomTx) error {
	if m.rollbackTxFn != nil {
		return m.rollbackTxFn(ctx, tx)
	}
	return nil
}

func (m *mockTransferRepo) GetByID(ctx context.Context, id uuid.UUID) (*entity.Account, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, entity.ErrAccountNotFound
}

func (m *mockTransferRepo) GetCurrencies(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]entity.Currency, error) {
	if m.getCurrenciesFn != nil {
		return m.getCurrenciesFn(ctx, ids)
	}
	return nil, nil
}

func (m *mockTransferRepo) DebitBalance(ctx context.Context, tx entity.CustomTx, id uuid.UUID, amount int64) error {
	if m.debitBalanceFn != nil {
		return m.debitBalanceFn(ctx, tx, id, amount)
	}
	return nil
}

func (m *mockTransferRepo) CreditBalance(ctx context.Context, tx entity.CustomTx, id uuid.UUID, amount int64) error {
	if m.creditBalanceFn != nil {
		return m.creditBalanceFn(ctx, tx, id, amount)
	}
	return nil
}

func (m *mockTransferRepo) CreateTransaction(ctx context.Context, tx entity.CustomTx, tr *entity.Transaction) error {
	if m.createTransactionFn != nil {
		return m.createTransactionFn(ctx, tx, tr)
	}
	return nil
}

func (m *mockTransferRepo) CreatePostings(ctx context.Context, tx entity.CustomTx, postings []entity.Posting) error {
	if m.createPostingsFn != nil {
		return m.createPostingsFn(ctx, tx, postings)
	}
	return nil
}

func (m *mockTransferRepo) CheckIdempotencyKey(ctx context.Context, key uuid.UUID) (uuid.UUID, error) {
	if m.checkIdempotencyFn != nil {
		return m.checkIdempotencyFn(ctx, key)
	}
	return uuid.Nil, entity.ErrTransactionNotFound
}

func (m *mockTransferRepo) GetTransactionByID(ctx context.Context, id uuid.UUID) (*entity.Transaction, error) {
	if m.getTransactionByIDFn != nil {
		return m.getTransactionByIDFn(ctx, id)
	}
	return nil, entity.ErrTransactionNotFound
}

func (m *mockTransferRepo) CreateAccount(context.Context, entity.CustomTx, *entity.Account) error {
	return nil
}

func (m *mockTransferRepo) ListPostingsByAccountID(context.Context, uuid.UUID, int, int) ([]entity.Posting, error) {
	return nil, nil
}

func (m *mockTransferRepo) GetBalanceFromPostings(context.Context, uuid.UUID) (int64, error) {
	return 0, nil
}

type mockAccountRepo struct {
	beginTxFn           func(ctx context.Context) (entity.CustomTx, error)
	commitTxFn          func(ctx context.Context, tx entity.CustomTx) error
	rollbackTxFn        func(ctx context.Context, tx entity.CustomTx) error
	createAccountFn     func(ctx context.Context, tx entity.CustomTx, acc *entity.Account) error
	createTransactionFn func(ctx context.Context, tx entity.CustomTx, tr *entity.Transaction) error
	createPostingsFn    func(ctx context.Context, tx entity.CustomTx, postings []entity.Posting) error
	getByIDFn           func(ctx context.Context, id uuid.UUID) (*entity.Account, error)
}

func (m *mockAccountRepo) BeginTx(ctx context.Context) (entity.CustomTx, error) {
	if m.beginTxFn != nil {
		return m.beginTxFn(ctx)
	}
	return mockTx{}, nil
}

func (m *mockAccountRepo) CommitTx(ctx context.Context, tx entity.CustomTx) error {
	if m.commitTxFn != nil {
		return m.commitTxFn(ctx, tx)
	}
	return nil
}

func (m *mockAccountRepo) RollbackTx(ctx context.Context, tx entity.CustomTx) error {
	if m.rollbackTxFn != nil {
		return m.rollbackTxFn(ctx, tx)
	}
	return nil
}

func (m *mockAccountRepo) CreateAccount(ctx context.Context, tx entity.CustomTx, acc *entity.Account) error {
	if m.createAccountFn != nil {
		return m.createAccountFn(ctx, tx, acc)
	}
	return nil
}

func (m *mockAccountRepo) CreateTransaction(ctx context.Context, tx entity.CustomTx, tr *entity.Transaction) error {
	if m.createTransactionFn != nil {
		return m.createTransactionFn(ctx, tx, tr)
	}
	return nil
}

func (m *mockAccountRepo) CreatePostings(ctx context.Context, tx entity.CustomTx, postings []entity.Posting) error {
	if m.createPostingsFn != nil {
		return m.createPostingsFn(ctx, tx, postings)
	}
	return nil
}

func (m *mockAccountRepo) GetByID(ctx context.Context, id uuid.UUID) (*entity.Account, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, entity.ErrAccountNotFound
}

func (m *mockAccountRepo) GetCurrencies(context.Context, []uuid.UUID) (map[uuid.UUID]entity.Currency, error) {
	return nil, nil
}

func (m *mockAccountRepo) DebitBalance(context.Context, entity.CustomTx, uuid.UUID, int64) error {
	return nil
}

func (m *mockAccountRepo) CreditBalance(context.Context, entity.CustomTx, uuid.UUID, int64) error {
	return nil
}

func (m *mockAccountRepo) CheckIdempotencyKey(context.Context, uuid.UUID) (uuid.UUID, error) {
	return uuid.Nil, entity.ErrTransactionNotFound
}

func (m *mockAccountRepo) GetTransactionByID(context.Context, uuid.UUID) (*entity.Transaction, error) {
	return nil, entity.ErrTransactionNotFound
}

func (m *mockAccountRepo) ListPostingsByAccountID(context.Context, uuid.UUID, int, int) ([]entity.Posting, error) {
	return nil, nil
}

func (m *mockAccountRepo) GetBalanceFromPostings(context.Context, uuid.UUID) (int64, error) {
	return 0, nil
}

type mockPostingWorkerRepo struct {
	getPostingsSumFn            func(ctx context.Context, accountID uuid.UUID, limitID int64) (int64, error)
	getAccountBalanceSnapshotFn func(ctx context.Context, accountID uuid.UUID) (int64, int64, error)
	applyBalanceCorrectionFn    func(ctx context.Context, accountID uuid.UUID, newAmount int64) error
}

func (m *mockPostingWorkerRepo) GetPostingsSum(ctx context.Context, accountID uuid.UUID, limitID int64) (int64, error) {
	if m.getPostingsSumFn != nil {
		return m.getPostingsSumFn(ctx, accountID, limitID)
	}
	return 0, nil
}

func (m *mockPostingWorkerRepo) GetAccountBalanceSnapshot(ctx context.Context, accountID uuid.UUID) (int64, int64, error) {
	if m.getAccountBalanceSnapshotFn != nil {
		return m.getAccountBalanceSnapshotFn(ctx, accountID)
	}
	return 0, 0, nil
}

func (m *mockPostingWorkerRepo) ApplyBalanceCorrection(ctx context.Context, accountID uuid.UUID, newAmount int64) error {
	if m.applyBalanceCorrectionFn != nil {
		return m.applyBalanceCorrectionFn(ctx, accountID, newAmount)
	}
	return nil
}

type mockPostingWorkerPool struct {
	readWriteFn func(ctx context.Context, fn func(ctx context.Context) error) error
}

func (m *mockPostingWorkerPool) ReadWrite(ctx context.Context, fn func(ctx context.Context) error) error {
	if m.readWriteFn != nil {
		return m.readWriteFn(ctx, fn)
	}
	return fn(ctx)
}

type mockPostingCursor struct {
	getCursorPositionFn    func(ctx context.Context, workerName string, batchSize int) (int64, int64, error)
	getActiveAccountsFn    func(ctx context.Context, lastCheckedID, maxID int64) ([]uuid.UUID, error)
	updateCursorPositionFn func(ctx context.Context, workerName string, position int64) error
}

func (m *mockPostingCursor) GetCursorPosition(ctx context.Context, workerName string, batchSize int) (int64, int64, error) {
	if m.getCursorPositionFn != nil {
		return m.getCursorPositionFn(ctx, workerName, batchSize)
	}
	return 0, 0, nil
}

func (m *mockPostingCursor) GetActiveAccounts(ctx context.Context, lastCheckedID, maxID int64) ([]uuid.UUID, error) {
	if m.getActiveAccountsFn != nil {
		return m.getActiveAccountsFn(ctx, lastCheckedID, maxID)
	}
	return nil, nil
}

func (m *mockPostingCursor) UpdateCursorPosition(ctx context.Context, workerName string, position int64) error {
	if m.updateCursorPositionFn != nil {
		return m.updateCursorPositionFn(ctx, workerName, position)
	}
	return nil
}

type mockStatsRepo struct {
	getTransactionByIDFn func(ctx context.Context, id uuid.UUID) (*entity.Transaction, error)
}

func (m *mockStatsRepo) GetTransactionByID(ctx context.Context, id uuid.UUID) (*entity.Transaction, error) {
	if m.getTransactionByIDFn != nil {
		return m.getTransactionByIDFn(ctx, id)
	}
	return nil, entity.ErrTransactionNotFound
}

func (m *mockStatsRepo) BeginTx(context.Context) (entity.CustomTx, error)  { return mockTx{}, nil }
func (m *mockStatsRepo) CommitTx(context.Context, entity.CustomTx) error   { return nil }
func (m *mockStatsRepo) RollbackTx(context.Context, entity.CustomTx) error { return nil }
func (m *mockStatsRepo) CreateTransaction(context.Context, entity.CustomTx, *entity.Transaction) error {
	return nil
}
func (m *mockStatsRepo) CheckIdempotencyKey(context.Context, uuid.UUID) (uuid.UUID, error) {
	return uuid.Nil, entity.ErrTransactionNotFound
}
