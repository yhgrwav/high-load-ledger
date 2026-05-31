# Load Generator

Генератор нагрузки для ledger. Поднимается **вместе** с основным стеком:

```bash
cp .env.example .env
# LOAD_GEN_WORKING=true
docker compose up -d --build --scale gateway=2
```

Если `LOAD_GEN_WORKING=false` — контейнер loadgen сразу завершается (exit 0).  
Если `true` — bootstrap аккаунтов, затем load-фаза.

## Что такое пуассоновская нагрузка

**Пуассоновский процесс** — модель, где события (transfer) приходят **случайно**, но в среднем **λ раз в секунду** (ваш RPS).

- интервалы между запросами **не равные** (не «ровно каждые 10 ms»);
- короткие паузы и редкие всплески — как в реальном трафике;
- матожидание интервала = **1 / λ** (при λ=100 RPS средняя пауза ~10 ms).

В коде это `poissonDelay(rps)`: случайная пауза с экспоненциальным распределением.

**Важно:** RPS в `.env` — это **target dispatch rate** (сколько job'ов loadgen **отправляет** в секунду).  
Сколько transfer **успешно обработал сервер** — отдельная метрика на дашборде ledger.

## Модель нагрузки

| Поток | ENV | Назначение |
|-------|-----|------------|
| Valid | `VALID_RPS` | успешные transfer |
| Invalid balance | `INVALID_RPS` | insufficient funds |
| Invalid currency | `INVALID_CURRENCY_RPS` | currency mismatch |

Суммарный target dispatch ≈ `VALID_RPS + INVALID_RPS + INVALID_CURRENCY_RPS`.

## Гарантия achieved dispatch

Loadgen гарантирует **отправку job'ов** (dispatch), а не успех на сервере:

1. **Poisson-scheduler** ведёт расписание `nextAt` — средняя частота = target RPS.
2. Job считается dispatched **в момент постановки в очередь** (не когда worker освободился).
3. Prometheus: `loadgen_target_rps` vs `rate(loadgen_requests_dispatched_total[1m])`.
4. В конце прогона — проверка ±5% по каждому потоку (лог `loadgen WARN` или `within target tolerance`).

Если workers/сервер не успевают, очередь растёт (`loadgen_queue_depth`), dispatch rate может упасть — это сигнал, что **target выше capacity стенда**.

`LOAD_DURATION` относится **только к load-фазе** (bootstrap не входит в таймер).

## Конфигурация

| Переменная | Описание | Default |
|------------|----------|---------|
| `LOAD_GEN_WORKING` | включить генератор | `false` |
| `USERS_AMOUNT` | аккаунтов (делится между валютами из proto) | `1000` |
| `VALID_RPS` | target dispatch valid | `100` |
| `INVALID_RPS` | target dispatch invalid balance | `10` |
| `INVALID_CURRENCY_RPS` | target dispatch invalid currency | `5` |
| `LOAD_DURATION` | длительность load-фазы (`0` = до SIGTERM) | `0` |
| `LOAD_BOOTSTRAP_WORKERS` | параллелизм CreateAccount | `50` |
| `LOAD_TX_WORKERS` | workers transfer | `100` |
| `LOADGEN_GRPC_ADDR` | gRPC ledger (`127.0.0.1:8085` локально, `nginx:80` в docker) | `127.0.0.1:8085` |
| `LOADGEN_METRICS_PORT` | Prometheus `/metrics` | `9092` |

## Grafana

- **Load Generator** — target vs achieved dispatch, queue depth.
- **Ledger Business Metrics** — ответ сервера (gRPC codes, transfer status).

## Структура

```
loadgen/
  config/
  service/
    accounts.go    bootstrap
    transfer.go    gRPC client
    pool.go        аккаунты + валюты из proto
    builder.go     valid / invalid jobs
    scheduler.go   poisson scheduling
    metrics.go     Prometheus
    stats.go       лог + финальная проверка ±5%
    core.go        orchestrator
cmd/loadgen/
docker/loadgen/    Dockerfile + entrypoint
```

## Архитектура

```
Poisson(valid)     ──┐
Poisson(invalid)   ──┼──► jobs ──► workers ──► gRPC ──► ledger
Poisson(currency)  ──┘
        │
        └── metrics :9092/metrics ──► Prometheus ──► Grafana "Load Generator"
```
