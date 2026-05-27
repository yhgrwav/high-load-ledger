# High-Load Distributed Ledger Core

Ядро платёжной системы на Go: gRPC API, идемпотентные переводы (Redis + PostgreSQL), фоновая сверка балансов.

**Стек:** Go · gRPC · Protobuf · PostgreSQL · Redis · Docker · Prometheus · Grafana

---

## Быстрый старт

**Требования:** Docker, Docker Compose, [grpcurl](https://github.com/fullstorydev/grpcurl)

```bash
git clone https://github.com/yhgrwav/high-load-ledger.git
cd high-load-ledger

cp .env.example .env
docker compose up -d --build
```

Миграции БД накатываются автоматически (`ledger-app-migrate`).

Проверка, что сервисы поднялись:

```bash
docker compose ps
grpcurl -plaintext localhost:8085 list
```

---

## Сервисы

| Сервис | Адрес | Назначение |
|--------|-------|------------|
| gRPC (load balancer) | `localhost:8085` | **Основная точка входа** (nginx) |
| gRPC gateway-1 | `localhost:50051` | Прямой доступ к инстансу |
| gRPC gateway-2 | `localhost:50052` | Прямой доступ к инстансу |
| Prometheus metrics | `http://localhost:6767/metrics` | Метрики gateway-1 |
| Prometheus UI | `http://localhost:19090` | |
| Grafana | `http://localhost:3000` | login: `admin` / `admin` |
| PostgreSQL | `localhost:5433` | |
| Redis | `localhost:6379` | |

---

## gRPC API

Доступные сервисы:

```bash
grpcurl -plaintext localhost:8085 list api.ledger.AccountService
grpcurl -plaintext localhost:8085 list api.ledger.TransactionService
grpcurl -plaintext localhost:8085 list api.ledger.StatsService
```

### 1. Создать счёт

Сервер сам генерирует `account_id` (UUID v7). В запросе — только валюта.

```bash
grpcurl -plaintext -d '{"currency":"CURRENCY_USD"}' \
  localhost:8085 api.ledger.AccountService/CreateAccount
```

Пример ответа:

```json
{
  "accountId": "VQ6EAAKbQdSnFkRmVUQAAA=="
}
```

Создайте второй счёт тем же способом — он понадобится для перевода.

### 2. Проверить баланс

Подставьте `accountId` из ответа CreateAccount (поле в base64):

```bash
grpcurl -plaintext -d '{
  "account_id": "VQ6EAAKbQdSnFkRmVUQAAA==",
  "requester_id": "VQ6EAAKbQdSnFkRmVUQAAA=="
}' localhost:8085 api.ledger.AccountService/GetBalance
```

### 3. Перевод

Для каждого нового перевода нужен **уникальный** `idempotency_key` (16 байт в base64).  
Повтор того же ключа вернёт тот же `transaction_id` без двойного списания.

```bash
grpcurl -plaintext -d '{
  "idempotency_key": "0pDx7mxUSwGQ5tcBdI8IUQ==",
  "user_from_id": "VQ6EAAKbQdSnFkRmVUQAAA==",
  "user_to_id": "mx3rTbt9S62b3SsNez3LbR==",
  "amount": 500,
  "currency": "CURRENCY_USD"
}' localhost:8085 api.ledger.TransactionService/Transfer
```

### 4. Получить транзакцию

```bash
grpcurl -plaintext -d '{
  "transaction_id": "AAAAAAAAAAAAAAAAAAAAAA=="
}' localhost:8085 api.ledger.StatsService/GetTransaction
```

---

## Конфигурация

Переменные окружения — в [`.env.example`](.env.example).

| Группа | Ключевые переменные |
|--------|---------------------|
| PostgreSQL | `DB_HOST`, `DB_PORT`, `DB_NAME`, `DB_SUPER_USER`, `DB_SUPER_PASSWORD` |
| Redis | `REDIS_HOST`, `REDIS_PORT`, `REDIS_TRANSACTION_TTL` |
| Приложение | `GRPC_PORT`, `METRICS_PORT`, `SERVICE_NAME` |
| Posting Worker | `POSTING_WORKER_ENABLED`, `POSTING_WORKER_NAME`, `POSTING_WORKER_BATCH_SIZE`, `POSTING_WORKER_BACKOFF` |

Posting Worker — фоновая сверка `accounts.amount` с суммой проводок в `postings`.  
`POSTING_WORKER_NAME` обязателен, если воркер включён.

---

## Разработка

```bash
go test ./internal/usecase/... -v   # unit-тесты
make gen                            # protobuf (нужен protoc)
```

Структура:

```
cmd/gateway/       — точка входа
api/ledger/        — protobuf
internal/          — domain, usecase, repository, transport
migrations/        — SQL
docker/            — prometheus, grafana
```

---

## Roadmap

- [x] gRPC API, Clean Architecture, миграции
- [x] Переводы с идемпотентностью
- [x] PostingWorker (верификация балансов)
- [x] Prometheus + Grafana, nginx (2 gateway)
- [x] Unit-тесты usecase
- [ ] Load generator (`loadgen/`)
- [ ] Integration tests
- [ ] Kafka, Kubernetes
- [ ] OpenTelemetry, шардирование PostgreSQL

---

## Ещё

- [TASK.md](TASK.md) — техническое задание
- [THOUGHTS.md](THOUGHTS.md) — инженерный дневник разработки
