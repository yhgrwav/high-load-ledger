## Техническое Задание для High-Load Distributed Ledger Core

### Цель проекта
- Достичь 1000-2000 RPS на один инстанс
- Реализовать покрытие современными инструментами для production-ready сервиса

### Технологический стек
- Язык: Go 1.25+
- Взаимодействие: gRPC (Protobuf)
- Брокер сообщений: Apache Kafka
- Кэширование и идемпотентность: Redis
- Хранилище данных: PostgreSQL
- observability: grafana, prometheus, slog, jaeger
- Инфраструктура: Docker, Docker Compose, Makefile

### План реализации
- [x] Protobuf контракт для AccountService, TransactionService, StatsService
- [x] Миграции для PostgreSQL (ledger schema, accounts/transactions/postings tables)
- [x] Чистая архитектура (domain, repository, usecase, transport слои)
- [x] Инициализация конфига из .env
- [x] Логирование с slog (structured logging)
- [x] Healthcheck для Postgres в docker-compose
- [ ] Интеграция с Kafka (async обработка транзакций)
- [ ] Unit и интеграционные тесты
- [ ] Метрики Prometheus (RPS, latency, Kafka lag)
- [ ] Трейсы OpenTelemetry
- [ ] Шардирование PostgreSQL (по user_id)
- [ ] Kubernetes манифесты для деплоя
- [ ] Микросервис для генерации нагрузки (load testing)
- [ ] Интеграция Nginx
