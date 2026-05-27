# Handoff: loadgen + Kafka/K8s

Контекст для продолжения разработки. Unit-тесты usecase написаны с помощью **Composer (Cursor AI)**.

## Рекомендуемый порядок работ

1. **Unit-тесты usecase** — готово (`internal/usecase/*_test.go`)
2. **Loadgen** (`cmd/loadgen/`) — следующий шаг
3. **Kafka/K8s** — после стабильного baseline

---

## Пункт 1: Loadgen

**Цель:** отдельный бинарник для high-load, метрики в Grafana/Prometheus.

**Структура:** `cmd/loadgen/main.go` — отдельный процесс, не часть gateway.

**Минимальный функционал:**
- флаги: `-target` (nginx `localhost:8085`), `-rps`, `-duration`, `-workers`
- сценарий: CreateAccount → pool аккаунтов → цикл Transfer + GetBalance
- **уникальный `idempotency_key` на каждый transfer** (UUID v4/v7)
- отчёт: success/error rate, p50/p99 latency

**Куда бить:** nginx (`8085`) — `least_conn` на N реплик `gateway` (`docker compose --scale gateway=N`).

**Prometheus:** DNS SD по имени сервиса `gateway:6767` (все реплики).

**Не усложнять:** `golang.org/x/time/rate`, goroutine pool, gRPC-клиент из `gen/go`.

---

## Пункт 2: Kafka + K8s

### Критично: sync gRPC ≠ Kafka в hot path напрямую

`Transfer` синхронный — клиент ждёт `transaction_id`. Паттерны:

| Паттерн | Описание |
|---------|----------|
| **A. Async API** | `Transfer` → 202 + polling статуса |
| **B. Request-reply** | correlation id + reply topic / Redis (хрупко) |
| **C. Side-effects only** | gRPC sync как сейчас, Kafka для `TransferCompleted` events |

**Рекомендация:** начать с **масштабирования gateway за LB/K8s**, Kafka — для side-effects (фаза 2), не ломая sync API.

### Edge-cases проекта

1. **Posting Worker на N репликах** — сейчас каждый gateway запускает worker с одним `POSTING_WORKER_NAME` → гонка за `worker_cursors`. **Решение:** отдельный Deployment `replicas: 1` для worker.

2. **Идемпотентность** — Redis + `UNIQUE(idempotency_key)` в DB. Kafka consumer: проверка idempotency **до** side effects, commit offset **после** commit tx.

3. **Partition key** — `user_from_id` (порядок операций на одном счёте).

4. **Hot account** — loadgen должен размазывать переводы по многим аккаунтам.

5. **K8s** — readiness = Postgres + Redis ping; `terminationGracePeriodSeconds` ≥ graceful shutdown gateway + worker cursor flush.

6. **Outbox** — publish в Kafka в той же tx, что transfer (transactional outbox).

### План фаз

```
Фаза 0: nginx → 2 gateway, loadgen, тесты, fix prometheus
Фаза 1: K8s Deployment gateway replicas=3, worker replicas=1
Фаза 2: Kafka side-effects (TransferCompleted → analytics)
Фаза 3: Kafka hot path (async Transfer API) — опционально
```

---

## Текущая архитектура (кратко)

- **Hot path:** gRPC → TransferUseCase → Postgres (FOR UPDATE) + Redis idempotency
- **Worker:** PostingWorker батчами по `postings.id`, adaptive `latest_posting_id`
- **Observability:** Prometheus `:6767/metrics`, Grafana `:3000`
- **Multi-instance:** nginx grpc_pass на 2 gateway
