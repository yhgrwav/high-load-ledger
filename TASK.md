## Техническое Задание для High-Load Distributed Ledger Core

### Цель проекта
- Разработать сервис с современным стеком технологий, выдерживающий высокий RPS
- Получить практический опыт с полноценной чистой архитектурой и идиоматичными подходами на разных этапах разработки

### Технологический стек
- Язык: Go 1.25+
- Взаимодействие: gRPC (Protobuf)
- Брокер сообщений: Apache Kafka
- Кэширование и идемпотентность: Redis
- Хранилище данных: PostgreSQL
- observability: grafana, prometheus, slog
- Инфраструктура: Docker, Docker Compose, Makefile

### План реализации
- [x] Protobuf контракт для AccountService, TransactionService, StatsService
- [x] Миграции для PostgreSQL (ledger schema, accounts/transactions/postings tables)
- [x] Чистая архитектура (domain, repository, usecase, transport слои)
- [x] Инициализация конфига из .env
- [x] Логирование с slog (structured logging)
- [x] Healthcheck для Postgres в docker-compose
- [ ] Интеграция с Kafka (async обработка транзакций)
- [x] Метрики Prometheus
- [ ] Шардирование PostgreSQL (по user_id)
- [ ] Kubernetes манифесты для деплоя
- [ ] Микросервис для генерации нагрузки (load testing)
- [ ] Интеграция Nginx
- [ ] Unit и интеграционные тесты
