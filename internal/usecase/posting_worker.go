package usecase

import (
	"context"
	"errors"
	"fmt"
	"high-load-ledger/internal/infra/telemetry"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

const (
	defaultBatchSize       = 100
	defaultBackoff         = 5 * time.Second
	maxBalanceVersionSteps = 8
)

type postingWorkerRepo interface {
	GetPostingsSum(ctx context.Context, accountID uuid.UUID, limitID int64) (int64, error)
	GetAccountBalanceSnapshot(ctx context.Context, accountID uuid.UUID) (balance int64, latestPostingID int64, err error)
	ApplyBalanceCorrection(ctx context.Context, accountID uuid.UUID, newAmount int64) error
}

type postingWorkerPool interface {
	ReadWrite(ctx context.Context, fn func(ctx context.Context) error) error
}

type postingCursorStore interface {
	GetCursorPosition(ctx context.Context, workerName string, batchSize int) (cursorPosition, upperLimit int64, err error)
	GetActiveAccounts(ctx context.Context, lastCheckedID, maxID int64) ([]uuid.UUID, error)
	UpdateCursorPosition(ctx context.Context, workerName string, position int64) error
}

type PostingWorker struct {
	repo        postingWorkerRepo
	pool        postingWorkerPool
	cursor      postingCursorStore
	logger      *slog.Logger
	metrics     *telemetry.PrometheusMetrics
	workerName  string
	batchSize   int
	backoffTime time.Duration
}

func NewPostingWorker(
	repo postingWorkerRepo,
	pool postingWorkerPool,
	cursor postingCursorStore,
	logger *slog.Logger,
	metrics *telemetry.PrometheusMetrics,
	workerName string,
	batchSize int,
	backoff time.Duration,
) (*PostingWorker, error) {
	if workerName == "" {
		return nil, fmt.Errorf("posting worker: worker name is required")
	}
	if batchSize <= 0 {
		batchSize = defaultBatchSize
	}
	if backoff <= 0 {
		backoff = defaultBackoff
	}

	return &PostingWorker{
		repo:        repo,
		pool:        pool,
		cursor:      cursor,
		logger:      logger,
		metrics:     metrics,
		workerName:  workerName,
		batchSize:   batchSize,
		backoffTime: backoff,
	}, nil
}

// суть воркера - пачками рассчитывать и валидировать балансы юзеров
// это будет происходить следующим образом:
// у нас есть postings, у которых id - bigint (автоинкремент), соответственно мы можем перебирать все значения снизу вверх
// мы запрашиваем какой-то батч на условно 100 записей. воркер получает батч, где условие следующее - если id > 100 { игнорировать }
// соответственно новые записи на момент проверки никак не повлияют на проверку
// мы получаем список записей postings, затем мы хотим оттуда вытянуть уникальных юзеров, т.е. в долгосрочной перспективе
// на нашу проверку никак не повлияют неактивные юзеры, что сделает нашу проверку лёгкой для базы.
// мы получили условно из 100 postings 20 уникальных юзеров.
// для каждого юзера мы делаем один общий запрос из двух подзапросов:
// 1. просим собрать сумму всех записей пользователя, но с условием - where id <= 100
// это условие означает следующее: если у юзера в момент проверки появились записи с id 125, 260 и 320 - нам плевать, ведь мы проверяем отдельный отрезок.
// 2. мы берём текущий баланс из аккаунта юзера, т.е. то, что наша база записала как "истинное" значение
// затем мы выполняем проверку, например:
// сумма postings равна 150 usd
// текущий баланс равен условно 200: ALERT!!!
// что произошло? мы получили баланс юзера, которому пришло 50 usd во время проверки, что поломало полностью нашу логику
// единственное верное решение - сделать версионирование баланса
// как это будет работать?
// у нас всё ещё в памяти лежит сумма postings 150 usd, но теперь мы уже при обращении к балансу делаем проверку:
// т.к. мы проверяем айдишники в каком-то диапазоне (в нашем случае 0-100), то и смотреть на состояние счёта мы будем через призму этого отрезка.
// если у аккаунта в базе последний привязанный posting_id > 100 (например 125) - это значит, что баланс в моменте уже убежал вперёд.
// тогда мы просто подвигаем нашу верхнюю границу для расчёта суммы конкретно этого юзера с 100 до 125.
// запрашиваем сумму postings юзера уже с условием where id <= 125 и сравниваем результат с его текущим балансом.
// получаем новую развилку:
// если сумма postings == текущий баланс { всё хорошо, выходим из проверки юзера, а глобальный курсор воркера всё так же сдвигается на 100 }
// если невалидно { алёрт, бизнес-логика (в моём случае я просто подправлю разницу), лог }
func (w *PostingWorker) Run(ctx context.Context) {
	if w.cursor == nil {
		w.logger.WarnContext(ctx, "posting worker: cursor store is nil, Run is disabled")
		return
	}

	w.logger.InfoContext(ctx, "posting worker started", "worker", w.workerName, "batch_size", w.batchSize)
	defer w.logger.InfoContext(ctx, "posting worker finished", "worker", w.workerName)

	// Переиспользуемый таймер вместо time.After на каждой итерации - иначе утечка timer-объектов в бесконечном цикле
	// 0 указывается чтобы таймер считался сработавшим и не ждём ненужный бэкофф на старте
	backoffTimer := time.NewTimer(0)
	if !backoffTimer.Stop() {
		<-backoffTimer.C
	}
	defer backoffTimer.Stop()

	var lastCommittedUpper int64

	for {
		select {
		case <-ctx.Done():
			// при shutdown сохраняем последний успешно обработанный upperLimit, чтобы не терять прогресс
			if lastCommittedUpper > 0 {
				if err := w.cursor.UpdateCursorPosition(ctx, w.workerName, lastCommittedUpper); err != nil {
					w.logger.WarnContext(ctx, "posting worker: failed to persist cursor on shutdown", "err", err)
				}
			}
			return
		default:
			// читаем текущую позицию курсора и верхнюю границу батча postings (cursor + batchSize, но не выше MAX(id))
			cursorPosition, upperLimit, err := w.cursor.GetCursorPosition(ctx, w.workerName, w.batchSize)
			if err != nil {
				w.logger.ErrorContext(ctx, "posting worker: get cursor position failed", "err", err)
				if !w.waitBackoff(ctx, backoffTimer) {
					return
				}
				continue
			}

			// новых postings нет — ждём backoff и начинаем итерацию заново.
			if cursorPosition >= upperLimit {
				if !w.waitBackoff(ctx, backoffTimer) {
					return
				}
				continue
			}

			// собираем уникальные account_id, у которых есть postings в диапазоне (cursorPosition, upperLimit]
			users, err := w.cursor.GetActiveAccounts(ctx, cursorPosition, upperLimit)
			if err != nil {
				w.logger.ErrorContext(ctx, "posting worker: get active accounts failed", "err", err)
				if !w.waitBackoff(ctx, backoffTimer) {
					return
				}
				continue
			}

			for _, id := range users {
				// проверяем баланс каждого активного аккаунта относительно upperLimit текущего батча
				if err := w.ValidateBalance(ctx, id, upperLimit); err != nil {
					if errors.Is(err, context.Canceled) {
						return
					}
					w.logger.ErrorContext(ctx, "posting worker: validate balance failed", "err", err, "account_id", id)
				}
			}

			// батч обработан - фиксируем курсор на upperLimit и переходим к следующему отрезку
			if err := w.cursor.UpdateCursorPosition(ctx, w.workerName, upperLimit); err != nil {
				w.logger.ErrorContext(ctx, "posting worker: update cursor position failed", "err", err)
				if !w.waitBackoff(ctx, backoffTimer) {
					return
				}
				continue
			}
			lastCommittedUpper = upperLimit
		}
	}
}

func (w *PostingWorker) ValidateBalance(ctx context.Context, id uuid.UUID, batchMaxID int64) error {
	effectiveMax := batchMaxID

	for step := 0; step < maxBalanceVersionSteps; step++ {
		// прерываем проверку аккаунта, если приложение получило сигнал shutdown
		if err := ctx.Err(); err != nil {
			return err
		}

		// снимок баланса и latest_posting_id - версия, до которой amount гарантированно актуален
		balance, latestPostingID, err := w.repo.GetAccountBalanceSnapshot(ctx, id)
		if err != nil {
			return err
		}

		// если hot path успел записать проводки за пределами батча - расширяем верхнюю границу только для этого аккаунта.
		if latestPostingID > effectiveMax {
			effectiveMax = latestPostingID
		}

		// считаем сумму проводок строго до effectiveMax: WHERE account_id = $1 AND id <= $2
		postingsSum, err := w.repo.GetPostingsSum(ctx, id, effectiveMax)
		if err != nil {
			return err
		}

		if postingsSum == balance {
			return nil
		}

		// между чтениями могли появиться новые postings - перечитываем версию и пробуем ещё раз
		_, refreshedLatest, err := w.repo.GetAccountBalanceSnapshot(ctx, id)
		if err != nil {
			return err
		}
		if refreshedLatest > effectiveMax {
			effectiveMax = refreshedLatest
			continue
		}

		return w.correctBalance(ctx, id)
	}

	w.logger.WarnContext(ctx, "posting worker: version steps exhausted, forcing correction",
		"account_id", id, "batch_max_id", batchMaxID)
	return w.correctBalance(ctx, id)
}

func (w *PostingWorker) correctBalance(ctx context.Context, id uuid.UUID) error {
	// tx прокидывается через context, usecase не знает про pgx.Tx
	return w.pool.ReadWrite(ctx, func(ctx context.Context) error {
		// limitID = 0 — полная сумма всех проводок аккаунта для коррекции.
		fullSum, err := w.repo.GetPostingsSum(ctx, id, 0)
		if err != nil {
			return err
		}

		if err := w.repo.ApplyBalanceCorrection(ctx, id, fullSum); err != nil {
			return err
		}

		if w.metrics != nil {
			w.metrics.RecordBalanceCorrection()
		}
		w.logger.WarnContext(ctx, "posting worker: balance corrected", "account_id", id, "balance", fullSum)
		return nil
	})
}

// waitBackoff блокирует текущую итерацию Run на backoffTime
// возвращает false, если за время ожидания пришла отмена контекста (shutdown) — тогда Run должен завершиться
// возвращает true, если таймер сработал — Run продолжает следующую итерацию цикла
func (w *PostingWorker) waitBackoff(ctx context.Context, timer *time.Timer) bool {
	timer.Reset(w.backoffTime)
	select {
	case <-ctx.Done():
		if !timer.Stop() {
			<-timer.C
		}
		return false
	case <-timer.C:
		return true
	}
}
