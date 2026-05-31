# High-Load Distributed Ledger Core

Ядро платёжной системы на Go: gRPC API, идемпотентные переводы (Redis + PostgreSQL), фоновая сверка балансов.

**Стек:** Go · gRPC · Protobuf · PostgreSQL · Redis · Docker · Prometheus · Grafana

---

## Быстрый старт

**Требования:** Docker, Docker Compose, [grpcurl](https://github.com/fullstorydev/grpcurl)

### Шаг 1. Клонирование и конфигурация

```bash
git clone https://github.com/yhgrwav/high-load-ledger.git
cd high-load-ledger

cp .env.example .env
```

Откройте `.env` и при необходимости измените значения.  
Минимально достаточно оставить defaults из `.env.example` — главное, чтобы файл существовал.

### Шаг 2. Сборка и запуск

```bash
docker compose up -d --build --scale gateway=2
```

Число реплик gateway должно совпадать с `GATEWAY_REPLICAS` в `.env` (по умолчанию `2`).  
Сервис `posting-worker` поднимается **один раз** — не масштабировать (`--scale posting-worker=1` по умолчанию).

- миграции БД накатываются автоматически (`ledger-app-migrate`)
- gRPC-балансировка — через nginx (`least_conn` + DNS resolve)

Проверка:

```bash
docker compose ps
grpcurl -plaintext localhost:8085 list
```

Остановка:

```bash
docker compose down
```

---

## Сервисы

| Сервис | Адрес | Назначение |
|--------|-------|------------|
| gRPC (load balancer) | `localhost:8085` | **Основная точка входа** (nginx → N × gateway) |
| Prometheus UI | `http://localhost:19090` | |
| Grafana | `http://localhost:3000` | login: `admin` / `admin` |
| PostgreSQL | `localhost:5433` | |
| Redis | `localhost:6379` | |

Метрики gateway (`/metrics`) доступны внутри Docker-сети; Prometheus собирает их со всех реплик через DNS service discovery.

Grafana (папка **High Load Ledger**):

| Дашборд | Содержание |
|---------|------------|
| **Ledger** | Transfer: business `result`, gRPC `code`, p99, system errors |
| **Load Generator** | dispatch target/achieved, queue, gRPC errors на valid-потоке |
| **Go Runtime** | goroutines / heap / GC по `job` |

Ключевые series: `ledger_transfer_total`, `ledger_grpc_requests_total{rpc,code}`, `loadgen_dispatched_total`, `loadgen_completed_total`.

Масштабирование gateway без пересборки:

```bash
docker compose up -d --scale gateway=3
```

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
| Posting Worker | `POSTING_WORKER_NAME`, `POSTING_WORKER_BATCH_SIZE`, `POSTING_WORKER_BACKOFF` — только для `cmd/worker` / сервиса `posting-worker` |
| Docker Compose | `GATEWAY_REPLICAS` — число реплик gateway (`--scale gateway=N`) |

Posting Worker — фоновая сверка `accounts.amount` с суммой проводок в `postings`.  
Запускается **отдельным процессом** (`cmd/worker`), не внутри gateway — иначе при `--scale gateway=N` несколько воркеров гоняются за одним курсором в БД.

---

## Разработка

```bash
go test ./internal/usecase/... -v   # unit-тесты
go run ./cmd/gateway               # gRPC API
go run ./cmd/worker                # posting worker (отдельный терминал)
make gen                            # protobuf (нужен protoc)
```

Структура:

```
cmd/gateway/       — gRPC API
cmd/worker/        — PostingWorker (один инстанс)
cmd/loadgen/       — нагрузочный генератор
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
- [x] Prometheus + Grafana, nginx + scale gateway
- [x] Unit-тесты usecase
- [ ] Load generator (`loadgen/`)
- [ ] Integration tests
- [ ] Kafka, Kubernetes
- [ ] OpenTelemetry, шардирование PostgreSQL

---

## Ещё

- [TASK.md](TASK.md) — техническое задание
- [THOUGHTS.md](THOUGHTS.md) — инженерный дневник разработки
