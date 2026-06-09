# Техническое задание

## High-Load Distributed Ledger Core

---

### Цель проекта

- Разработать сервис с современным стеком технологий, выдерживающий высокий RPS
- Получить практический опыт с полноценной чистой архитектурой и идиоматичными подходами на разных этапах разработки

---

### Технологический стек

| Категория | Технологии |
|-----------|------------|
| Язык | Go 1.25+ |
| Взаимодействие | gRPC (Protobuf) |
| Брокер сообщений | Apache Kafka |
| Кэш / идемпотентность | Redis |
| Хранилище | PostgreSQL |
| Observability | Grafana, Prometheus, slog |
| Инфраструктура | Docker, Docker Compose, Makefile |

---

### План реализации

- [x] Protobuf контракт для AccountService, TransactionService, StatsService
- [x] Миграции для PostgreSQL (ledger schema, accounts/transactions/postings tables)
- [x] Чистая архитектура (domain, repository, usecase, transport слои)
- [x] Инициализация конфига из .env
- [x] Логирование с slog (structured logging)
- [x] Healthcheck для Postgres в docker-compose
- [x] Интеграция с Kafka (async обработка транзакций)
- [x] Метрики Prometheus
- [x] PostingWorker (фоновая верификация балансов)
- [ ] Шардирование PostgreSQL (по user_id)
- [ ] Kubernetes манифесты для деплоя
- [x] Микросервис для генерации нагрузки (load testing)
- [x] Интеграция Nginx
- [ ] Unit и интеграционные тесты
- [x] Unit-тесты usecase (Transfer, Account, PostingWorker, Stats)
