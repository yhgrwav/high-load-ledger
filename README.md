# High-Load Distributed Ledger Core

Высокопроизводительное ядро платежной системы с обработкой транзакций через gRPC и гарантией идемпотентности через Redis.

## Быстрый старт

```bash
# Поднять инфраструктуру (PostgreSQL, Redis) и запустить миграции
docker compose up -d

# Запустить приложение
go run ./cmd/gateway
```

## Структура проекта

```
high-load-ledger/
├── api/                    # Protobuf контракты
│   └── ledger/
│       └── ledger.proto
├── cmd/                    # Точка входа
│   └── gateway/
│       └── main.go
├── internal/
│   ├── config/             # Чтение .env
│   ├── domain/             # Domain entities и repository interfaces
│   │   ├── entity/
│   │   └── repository/
│   ├── infra/              # Инфраструктура (logger)
│   ├── repository/         # Реализации repository (PostgreSQL, Redis)
│   ├── transport/          # gRPC handlers
│   └── usecase/            # Бизнес-логика
├── migrations/             # SQL миграции
├── scripts/                # Инициализационные скрипты для БД
├── docker-compose.yaml
├── go.mod
└── go.sum
```

## Примеры gRPC-запросов

### Создание аккаунта
```json
{
  "user_id": "VQ6EAAKbQdSnFkRmVUQAAA==",
  "currency": "CURRENCY_USD"
}
```

### Перевод средств
```json
{
  "idempotency_key": "0pDx7mxUSwGQ5tcBdI8IUQ==",
  "user_from_id": "VQ6EAAKbQdSnFkRmVUQAAA==",
  "user_to_id": "mx3rTbt9S62b3SsNez3LbR==",
  "amount": 500,
  "currency": "CURRENCY_USD"
}
```

### Проверка баланса
```json
{
  "account_id": "VQ6EAAKbQdSnFkRmVUQAAA==",
  "requester_id": "VQ6EAAKbQdSnFkRmVUQAAA=="
}
```

## План развития (Roadmap)
- [x] Protobuf контракты и миграции
- [x] Чистая архитектура
- [x] AccountService и TransactionService
- [ ] StatsService (получение транзакций, обновление статуса)
- [ ] Unit и интеграционные тесты
- [ ] Интеграция с Apache Kafka для асинхронной обработки
- [ ] Prometheus метрики
- [ ] OpenTelemetry трассировка
- [ ] Шардирование PostgreSQL
- [ ] Kubernetes манифесты
