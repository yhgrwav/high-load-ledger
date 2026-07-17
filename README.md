# High-Load Distributed Ledger Core

Ядро платёжной системы на Go: gRPC API, идемпотентные переводы (Redis + PostgreSQL), фоновая сверка балансов.

**Стек:** Go · gRPC · Protobuf · PostgreSQL 18 · Redis · Kafka · Docker · Kubernetes · Prometheus · Grafana

---

## Ветка `ai-experements`: охота за RPS

Эта ветка — не новая фича, а полигон для эксперимента: взять существующую реализацию (`main`) и с помощью Claude и Grok выжать из неё максимум RPS, ничего не сломав по пути. Она умышленно живёт отдельно от `main`, чтобы не портить историю основной ветки экспериментальными правками и промежуточными состояниями.

**Что изменилось в коде (`internal/`) относительно `main`:**

- Убран лишний `SELECT ... FOR UPDATE` перед списанием — атомарный `UPDATE ... WHERE amount >= $1` в `DebitBalance` сам защищает от ухода в минус, без отдельной блокировки. Порядок `Debit`/`Credit` теперь сортируется по UUID аккаунта (не по from/to), чтобы не ловить deadlock на встречных переводах.
- `latest_posting_id` больше не апдейтится отдельным запросом внутри `CreatePostings` — слит в тот же `UPDATE`, что и баланс, в `Debit`/`Credit`.
- Публикация в Kafka после коммита переведена на detached-контекст в отдельной горутине — раньше использовала контекст самого gRPC-запроса, который grpc-go отменяет сразу после возврата хендлера (под нагрузкой это давало шторм `context canceled`, хотя перевод уже был закоммичен).
- Проверка валют аккаунтов (`validateTransferCurrencies`) сделана параллельной вместо двух последовательных походов в Redis.
- `synchronous_commit=off` на Postgres.

**Что добавлено — `k8s/` и `MakeFile`:** полный манифест-сет для локального `kind`-кластера (`make -f MakeFile ledgerup` / `ledgerdown`) — Postgres, Kafka (KRaft), 2×Redis, gateway с HPA, Prometheus/Grafana с дашбордами, `metrics-server`. В `main`/`dev` этого нет и никогда не было закоммичено (см. [docs/THOUGHTS.md](docs/THOUGHTS.md) — история о том, как эта работа однажды уже терялась при `git reset`).

**Реальный прирост, измеренный по факту:**

| Что чинили | Где измерено | RPS до | RPS после |
|---|---|---|---|
| Убран лишний `SELECT FOR UPDATE`, `DB_MAX_CONNS`/`PG_MAX_CONNECTIONS` тюнинг | Docker Compose, тот же хост | ~5300 | ~5650 |
| gRPC connection-pinning в k8s (headless Service + client-side round-robin вместо намертво закреплённых за 3 подами соединений) | Kubernetes (kind) | ~1750 | без изменений сам по себе — вскрыл следующее узкое место ↓ |
| O(n)-копирование всего пула аккаунтов на каждый dispatch в `loadgen` (`AccountPool.snapshotCurrency`, копия 20-25k структур на **каждый** вызов из единственной горутины-диспетчера) | Kubernetes (kind) | ~1800 | **~4300** (×2.4) |

Дальше упёрлись в честный потолок конкретной машины (Ryzen 9 5900X, 24 потока) — под нагрузкой узел `kind` уходил за 21 из 24 ядер, Postgres — под 9 из 16-ядерного лимита, containerd под таким давлением начинал сбоить (рестарты Postgres/Redis не из-за багов приложения, а из-за нехватки ресурсов хоста). Это тот же потолок ~5.5-6k RPS, что был задокументирован при более ранних прогонах через `docker compose` на этом же железе — дальше нужно либо железо помощнее/несколько узлов, либо снижать CPU-стоимость одного перевода ещё сильнее.

Самый ценный вывод сессии: **самое дорогое узкое место оказалось не в ledger, а в собственном нагрузочном генераторе** — сервер (`gateway`+`Postgres`) держал 24%-88% загрузки с огромным запасом, пока клиент физически не мог сгенерировать больше ~1800 RPS из-за O(n)-копирования на каждый диспетч.

---

## Быстрый старт (Docker Compose)

**Требования:** Docker, Docker Compose, [grpcurl](https://github.com/fullstorydev/grpcurl)

### Шаг 1. Клонирование и переключение на ветку

```bash
git clone https://github.com/yhgrwav/high-load-ledger.git
cd high-load-ledger
git checkout ai-experements

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

## Быстрый старт (Kubernetes / kind)

Полноценный k8s-стек для локального нагрузочного тестирования с автоскейлингом gateway (HPA) и Grafana/Prometheus — живёт целиком в `k8s/`, только в этой ветке.

**Требования:** Docker, [kind](https://kind.sigs.k8s.io/), `kubectl`, GNU Make.

```bash
git clone https://github.com/yhgrwav/high-load-ledger.git
cd high-load-ledger
git checkout ai-experements

make -f MakeFile ledgerup
```

Одна команда: создаёт (или переиспользует) кластер `kind`, собирает 4 образа, грузит их в кластер, ставит `metrics-server`, применяет `kubectl apply -k k8s/`, ждёт миграции/Kafka-топик/gateway, поднимает port-forward (gRPC/Prometheus/Grafana) и запускает `loadgen`.

После запуска:

| Что | Адрес |
|---|---|
| gRPC | `localhost:50051` |
| Prometheus | `http://localhost:19090` |
| Grafana | `http://localhost:3000` (`admin` / `admin`) |

Перезапустить нагрузочный прогон без пересборки:

```bash
kubectl delete job/loadgen -n ledger --ignore-not-found
kubectl apply -f k8s/loadgen.yaml -n ledger
```

Остановка (удаляет namespace `ledger` и гасит port-forward, сам кластер `kind` остаётся):

```bash
make -f MakeFile ledgerdown
```

**Важно:** `PostingWorker` (`posting-worker` в `k8s/worker.yaml`) должен оставаться в **одном** экземпляре — несколько реплик гоняются за одним и тем же курсором в БД. HPA на него не вешать.

---

## Архитектура (Docker Compose)

```
╔═══════════════════════════════════════════════════╗
║  HIGH-LOAD DISTRIBUTED LEDGER ARCHITECTURE        ║
╚═══════════════════════════════════════════════════╝

              ┌──────────────────┐
              │   LOAD GENERATOR │
              │  (Poisson RPS)   │
              └────────┬─────────┘
                       │
                       ▼
        ┌────────────────────────────────────────┐
        │  CLIENT / EXTERNAL SERVICE             │
        │  (Payment Requests)                    │
        └─────────────────┬──────────────────────┘
                          │ gRPC
                          ▼
    ┌───────────────────────────────────────────────┐
    │  NGINX LOAD BALANCER (least_conn)             │
    │  Port 8085                                    │
    └──────────┬──────────────────┬─────────────────┘
               │                  │
          ┌────▼───┐         ┌────▼───┐
          │Gateway1│         │Gateway2│
          │Replica │         │Replica │
          └────┬───┘         └────┬───┘
               │                  │
               └────────┬─────────┘
                        ▼
    ┌───────────────────────────────────────────────┐
    │  POSTGRESQL 18 (Transactional DB)             │
    │  • accounts • postings • idempotency_keys     │
    └────────────────┬──────────────────────────────┘
                     │
       ┌─────────────┴────────────────────┐
       │                                  │
       ▼                                  ▼
   ┌─────────────┐           ┌──────────────────┐
   │  REDIS 7    │           │  KAFKA CLUSTER   │
   │ (Cache &    │           │ (Event Stream)   │
   │Idempotency) │           │ completed_txns   │
   └─────────────┘           └────────┬─────────┘
                                      │
                                      ▼
                              ┌──────────────────┐
                              │  Stats Worker    │
                              │(Kafka Consumer)  │
                              └──────────────────┘

╔═══════════════════════════════════════════════════╗
║  TECH: Go · gRPC · PostgreSQL · Redis · Kafka     ║
║  METRICS: Prometheus · Grafana                    ║
║  DEPLOYMENT: Docker · Docker Compose · Kubernetes ║
╚═══════════════════════════════════════════════════╝
```

В `k8s/` та же топология, но nginx заменён на `Service` + `HorizontalPodAutoscaler` (gateway скейлится по CPU от 3 до 16 реплик), а gRPC-клиент (`loadgen`) балансирует запросы сам через headless `Service` + round-robin резолвер (`loadgen/service/k8sresolver.go`) — иначе kube-proxy закрепляет каждое соединение за одним подом навсегда.

**Компоненты:**

- **API Gateway** (N реплик за nginx/HPA) — главная точка входа, масштабируется горизонтально
- **PostgreSQL 18** — все persisted данные: счёта, проводки, ключи идемпотентности
- **Redis** — кэш балансов, кэш транзакций и fast-path идемпотентности (отдельный инстанс `idempotency-redis`, чтобы не делить connection pool с общим кэшем)
- **Kafka** — асинхронный stream завершённых транзакций для side-effects
- **Stats Worker** — консьюмер Kafka, пополняет Redis-кэш для `GetTransaction`
- **Posting Worker** — фоновая сверка и коррекция балансов (отдельный процесс, **один инстанс**)

---

## Сервисы

### Docker Compose

| Сервис | Адрес | Назначение |
|--------|-------|------------|
| gRPC (load balancer) | `localhost:8085` | **Основная точка входа** (nginx → N × gateway) |
| Prometheus UI | `http://localhost:19090` | |
| Grafana | `http://localhost:3000` | login: `admin` / `admin` |
| PostgreSQL | `localhost:5433` | PostgreSQL 18 |
| Redis | `localhost:6379` | идемпотентность, кэш транзакций |
| Kafka | `localhost:9092` | события `completed_transactions` |

Метрики gateway (`/metrics`) доступны внутри Docker-сети; Prometheus собирает их со всех реплик через DNS service discovery.

### Kubernetes

| Сервис | Адрес (после `make -f MakeFile ledgerup`) |
|--------|---------------------------------------------|
| gRPC | `localhost:50051` (port-forward → `svc/gateway`) |
| Prometheus | `http://localhost:19090` |
| Grafana | `http://localhost:3000` |

Внутри кластера — `postgres:5432`, `redis:6379`, `idempotency-redis:6379`, `kafka:29092`, `gateway:50051` / `gateway-headless:50051` (последний — для клиентского round-robin, не для одиночных запросов).

### Grafana (папка **High Load Ledger**)

| Дашборд | Содержание |
|---------|------------|
| **Ledger** | Transfer: business `result`, gRPC `code`, p99, system errors |
| **Load Generator** | dispatch target/achieved, queue, gRPC errors на valid-потоке |
| **Go Runtime** | goroutines / heap / GC по `job` |
| **Инфраструктура** | Postgres, Redis (экспортеры). Nginx-панели актуальны только для Docker Compose — в k8s nginx нет, его роль играют Service + HPA, эти 3 панели там ожидаемо пустые |

Ключевые series: `ledger_transfer_total`, `ledger_grpc_requests_total{rpc,code}`, `loadgen_dispatched_total`, `loadgen_completed_total`.

Масштабирование gateway без пересборки (Docker Compose):

```bash
docker compose up -d --scale gateway=3
```

В k8s масштабирование делает `HorizontalPodAutoscaler` (`gateway-hpa`, 3-16 реплик по CPU) автоматически.

---

## gRPC API

Доступные сервисы:

```bash
grpcurl -plaintext localhost:8085 list api.ledger.AccountService
grpcurl -plaintext localhost:8085 list api.ledger.TransactionService
grpcurl -plaintext localhost:8085 list api.ledger.StatsService
```

(для k8s — тот же `grpcurl`, но на `localhost:50051`)

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

Переменные окружения — в [`.env.example`](.env.example). Там же, в конце файла, справочный блок для `k8s/configmap.yaml` (переменные `K8S_*`) с расчётом пулов соединений под HPA.

| Группа | Ключевые переменные |
|--------|---------------------|
| PostgreSQL | `DB_HOST`, `DB_PORT`, `DB_NAME`, `DB_SUPER_USER`, `DB_SUPER_PASSWORD`, `PG_SYNCHRONOUS_COMMIT` |
| Redis | `REDIS_HOST`, `REDIS_PORT`, `REDIS_TRANSACTION_TTL` |
| Приложение | `GRPC_PORT`, `METRICS_PORT`, `SERVICE_NAME` |
| Posting Worker | `POSTING_WORKER_NAME`, `POSTING_WORKER_BATCH_SIZE`, `POSTING_WORKER_BACKOFF` — только для `cmd/worker` / сервиса `posting-worker` |
| Docker Compose | `GATEWAY_REPLICAS` — число реплик gateway (`--scale gateway=N`) |
| Loadgen | `LOADGEN_GRPC_CONNECTIONS`, `LOAD_TX_WORKERS`, `VALID_RPS` — см. [loadgen/README.md](loadgen/README.md) |

Posting Worker — фоновая сверка `accounts.amount` с суммой проводок в `postings`.
Запускается **отдельным процессом** (`cmd/worker`), не внутри gateway — иначе при `--scale gateway=N` (или в k8s при скейле HPA) несколько воркеров гоняются за одним курсором в БД.

---

## Разработка

```bash
go test ./internal/usecase/... -v   # unit-тесты
go run ./cmd/gateway                # gRPC API
go run ./cmd/worker                 # posting worker (отдельный терминал)
make -f MakeFile gen                # protobuf (нужен protoc)
```

Структура:

```
cmd/gateway/       — gRPC API (+ Kafka producer)
cmd/worker/        — PostingWorker (один инстанс)
cmd/stats/         — Kafka consumer → Redis-кэш для GetTransaction
cmd/loadgen/       — нагрузочный генератор
api/ledger/        — protobuf
internal/          — domain, usecase, repository, transport
migrations/        — SQL
k8s/                — манифесты Kubernetes (только в этой ветке)
docker/            — prometheus, grafana, worker, stats, loadgen
```

---

## Документация

| Документ | Описание |
|----------|----------|
| [docs/TASK.md](docs/TASK.md) | Техническое задание и чеклист этапов |
| [docs/HANDOFF.md](docs/HANDOFF.md) | Контекст для продолжения: loadgen, Kafka, K8s |
| [docs/DATABASE_README.md](docs/DATABASE_README.md) | PostgreSQL: миграции, пользователи, ограничения |
| [docs/kafka.md](docs/kafka.md) | Заметки по интеграции Kafka |
| [docs/AI.md](docs/AI.md) | Подход к использованию ИИ в проекте |
| [docs/THOUGHTS.md](docs/THOUGHTS.md) | Инженерный дневник разработки |
| [loadgen/README.md](loadgen/README.md) | Нагрузочный генератор: poisson RPS, метрики, `.env` |

### Прочее

| Ресурс | Описание |
|--------|----------|
| [`.env.example`](.env.example) | Все переменные окружения (БД, Redis, Kafka, loadgen, k8s-справочник) |
| [`api/ledger/ledger.proto`](api/ledger/ledger.proto) | gRPC-контракт |
| [`docker-compose.yaml`](docker-compose.yaml) | Полный стек: gateway, workers, observability |
| [`k8s/`](k8s/) | Манифесты Kubernetes + `kustomization.yaml` |
| [`MakeFile`](MakeFile) | `ledgerup`/`ledgerdown` — поднять/снести k8s-стек одной командой |
| [`nginx.conf`](nginx.conf) | gRPC load balancing (`least_conn`), используется только в Docker Compose |
| [LICENSE](LICENSE) | Лицензия |

---
