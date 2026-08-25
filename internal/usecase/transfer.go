package usecase

import (
	"bytes"
	"context"
	"errors"
	"high-load-ledger/internal/domain/entity"
	"high-load-ledger/internal/domain/repository"
	"high-load-ledger/internal/infra/telemetry"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
)

// kafkaPublishTimeout bounds the detached (request-independent) publish that runs after commit.
const kafkaPublishTimeout = 3 * time.Second

type transferRepo interface {
	repository.TransactionRepository
	repository.AccountRepository
	repository.PostingRepository
}

type transactionPublisher interface {
	PublishTransaction(ctx context.Context, tx entity.Transaction) error
}

type TransferUseCase struct {
	repo      transferRepo
	cache     repository.CacheRepository
	publisher transactionPublisher
	logger    *slog.Logger
	metrics   *telemetry.PrometheusMetrics
}

func NewTransferUseCase(repo transferRepo, cache repository.CacheRepository, logger *slog.Logger, metrics *telemetry.PrometheusMetrics, publisher transactionPublisher) *TransferUseCase {
	return &TransferUseCase{
		repo:      repo,
		cache:     cache,
		publisher: publisher,
		logger:    logger,
		metrics:   metrics,
	}
}

func (t *TransferUseCase) Transaction(ctx context.Context, req entity.TransactionRequest) (id uuid.UUID, err error) {
	// если не произошло никаких ошибок - метрики оставляем пустыми
	defer func() {
		if t.metrics == nil {
			return
		}

		status := "success"
		switch {
		case err == nil:
			status = "success"
		case errors.Is(err, entity.ErrInvalidAmount),
			errors.Is(err, entity.ErrSameAccountTransfer),
			errors.Is(err, entity.ErrEmptyIdempotencyKey),
			errors.Is(err, entity.ErrInvalidCurrency):
			status = "validation_error"
		case errors.Is(err, entity.ErrCurrencyMismatch):
			status = "currency_mismatch"
		case errors.Is(err, entity.ErrInsufficientFunds):
			status = "insufficient_funds"
		default:
			status = "system_error"
		}

		// если получили ошибку - отдаём статус в метрики
		t.metrics.RecordTransfer(status)
	}()

	// 1. валидируем запрос с помощью вспомогательной функции
	if err := t.validateRequest(req); err != nil {
		return uuid.Nil, err
	}

	// 2. валидируем тип валют (это часть бизнес логики)
	if err = t.validateTransferCurrencies(ctx, req.FromAccountID, req.ToAccountID, req.Currency); err != nil {
		return uuid.Nil, err
	}

	// 3. если валидация прошла успешно - начинаем транзакцию
	tx, err := t.repo.BeginTx(ctx)
	if err != nil {
		return uuid.Nil, err
	}

	// если во время транзакции что-то идёт не так - откатываем
	defer func() {
		if err != nil {
			_ = t.repo.RollbackTx(ctx, tx)
		}
	}()

	// создаём uuidV7 (более оптимизированную для индексации версию) для сущности транзакции
	trxID, err := uuid.NewV7()
	if err != nil {
		return uuid.Nil, err
	}

	// 4. создаём сущность транзакции
	trx := entity.Transaction{
		ID:             trxID,
		IdempotencyKey: req.IdempotencyKey,
		FromAccountID:  req.FromAccountID,
		ToAccountID:    req.ToAccountID,
		Amount:         req.Amount,
		Currency:       req.Currency,
		CreatedAt:      time.Now(),
	}

	// 5. с созданной сущностью идём в базу, чтобы сохранить запись
	if err = t.repo.CreateTransaction(ctx, tx, &trx); err != nil {
		return uuid.Nil, err
	}

	// 6. создаём записи для журнала, по которому наш фоновый воркер будет сверять балансы
	postings := []entity.Posting{
		{TransactionID: trx.ID, AccountID: req.FromAccountID, Amount: -req.Amount},
		{TransactionID: trx.ID, AccountID: req.ToAccountID, Amount: req.Amount},
	}

	// 7. отправляем записи в базу и получаем id постингов, чтобы сразу проставить их в accounts
	postingIDs, err := t.repo.CreatePostings(ctx, tx, postings)
	if err != nil {
		return uuid.Nil, err
	}

	// 8. списание/зачисление без отдельного SELECT ... FOR UPDATE: недостаток средств ловит
	// сам DebitBalance атомарным условным UPDATE (WHERE amount >= $1), а latest_posting_id
	// проставляется тем же UPDATE через postingIDs, так что на 2 строки accounts всего 2
	// UPDATE-а вместо трёх.
	// Порядок операций фиксируем по возрастанию UUID аккаунта (а не по from/to), чтобы у двух
	// конкурентных встречных переводов (A->B и B->A) был единый порядок блокировки строк и
	// не возникал deadlock.
	type balanceOp struct {
		accountID uuid.UUID
		isDebit   bool
	}
	ops := [2]balanceOp{
		{req.FromAccountID, true},
		{req.ToAccountID, false},
	}
	if bytes.Compare(ops[1].accountID[:], ops[0].accountID[:]) < 0 {
		ops[0], ops[1] = ops[1], ops[0]
	}

	for _, op := range ops {
		postingID := postingIDs[op.accountID]
		if op.isDebit {
			err = t.repo.DebitBalance(ctx, tx, op.accountID, req.Amount, postingID)
		} else {
			err = t.repo.CreditBalance(ctx, tx, op.accountID, req.Amount, postingID)
		}
		if err != nil {
			return uuid.Nil, err
		}
	}

	// 9. если всё прошло успешно - коммитим транзакцию
	if err = t.repo.CommitTx(ctx, tx); err != nil {
		return uuid.Nil, err
	}

	// когда всё ок - паблишер отправляет сообщение с закоммиченной транзакцией. Публикация идёт
	// в отдельной горутине с собственным контекстом (не производным от ctx запроса), потому что
	// grpc-go отменяет ctx хендлера сразу после возврата из Transaction — использование ctx тут
	// приводило к "context canceled" под нагрузкой, хотя транзакция уже была закоммичена.
	if t.publisher != nil {
		go func(trx entity.Transaction) {
			pubCtx, cancel := context.WithTimeout(context.Background(), kafkaPublishTimeout)
			defer cancel()
			if err := t.publisher.PublishTransaction(pubCtx, trx); err != nil {
				t.logger.ErrorContext(pubCtx, "transfer: kafka publish failed", "err", err, "transaction_id", trx.ID)
			}
		}(trx)
	}

	return trx.ID, nil
}

func (t *TransferUseCase) validateRequest(req entity.TransactionRequest) error {
	if req.Amount <= 0 {
		return entity.ErrInvalidAmount
	}
	if req.FromAccountID == req.ToAccountID {
		return entity.ErrSameAccountTransfer
	}
	if req.IdempotencyKey == uuid.Nil {
		return entity.ErrEmptyIdempotencyKey
	}
	if !req.Currency.IsValid() {
		return entity.ErrInvalidCurrency
	}
	return nil
}

func (t *TransferUseCase) validateTransferCurrencies(ctx context.Context, fromID, toID uuid.UUID, currency entity.Currency) error {
	var fromCurrency, toCurrency entity.Currency

	// оба запроса в кэш независимы, гоняем их параллельно вместо двух последовательных round-trip'ов
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		if c, err := t.cache.GetAccountCurrency(ctx, fromID); err == nil {
			fromCurrency = c
		}
	}()
	go func() {
		defer wg.Done()
		if c, err := t.cache.GetAccountCurrency(ctx, toID); err == nil {
			toCurrency = c
		}
	}()
	wg.Wait()

	// если нет в кэше - идём в базу
	missingIDs := make([]uuid.UUID, 0, 2)
	if fromCurrency == entity.CURRENCY_UNSPECIFIED {
		missingIDs = append(missingIDs, fromID)
	}
	if toCurrency == entity.CURRENCY_UNSPECIFIED {
		missingIDs = append(missingIDs, toID)
	}

	if len(missingIDs) > 0 {
		currencies, err := t.repo.GetCurrencies(ctx, missingIDs)
		if err != nil {
			return err
		}

		if fromCurrency == entity.CURRENCY_UNSPECIFIED {
			if c, ok := currencies[fromID]; ok {
				fromCurrency = c
				_ = t.cache.SetAccountCurrency(ctx, fromID, c, 24*time.Hour)
			} else {
				return entity.ErrAccountNotFound
			}
		}

		if toCurrency == entity.CURRENCY_UNSPECIFIED {
			if c, ok := currencies[toID]; ok {
				toCurrency = c
				_ = t.cache.SetAccountCurrency(ctx, toID, c, 24*time.Hour)
			} else {
				return entity.ErrAccountNotFound
			}
		}
	}

	if fromCurrency != currency || toCurrency != currency {
		return entity.ErrCurrencyMismatch
	}

	return nil
}
