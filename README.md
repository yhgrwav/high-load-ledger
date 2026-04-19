# 💸 High-Load Distributed Ledger Core

[![Go Report Card](https://goreportcard.com/badge/github.com/yhgrwav/high-load-ledger)](https://goreportcard.com/report/github.com/yhgrwav/high-load-ledger)
![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)
![Kafka](https://img.shields.io/badge/Message_Broker-Kafka-black?logo=apachekafka)
![Postgres](https://img.shields.io/badge/DB-PostgreSQL-blue?logo=postgresql)
![Redis](https://img.shields.io/badge/Cache-Redis-red?logo=redis)

Высокопроизводительное ядро платежной системы с асинхронной моделью обработки транзакций. Проект спроектирован для работы в условиях высокой конкурентности и гарантирует целостность данных (ACID) при обработке финансовых потоков.

## 🏗 Архитектурные решения (Design Decisions)

Система разделена на два независимых слоя для обеспечения максимальной пропускной способности:



### 1. Write-Optimized API Gateway (The Producer)
- **Low Latency:** Принимает gRPC-запрос, выполняет быструю валидацию и сразу возвращает `Accepted`.
- **Idempotency Layer:** Использует **Redis** для атомарной проверки `idempotency_key`. Это исключает риск двойных списаний при повторных запросах (Retry Policy).
- **Backpressure:** Вместо прямой записи в БД, данные сбрасываются в **Kafka**, что позволяет системе выдерживать резкие пики трафика (Spike Loads).

### 2. Transaction Processor (The Consumer)
- **Batch Processing:** Воркер читает транзакции из Kafka пачками. Это позволяет использовать **Batch Inserts/Updates** в PostgreSQL, на порядок снижая количество IOPS.
- **Data Integrity:** Обработка каждой транзакции происходит в строгом соответствии с балансом пользователя.
- **Graceful Shutdown:** Реализована корректная остановка: воркер дочитывает текущий батч из Kafka перед завершением процесса, исключая "повисшие" транзакции.

## 🛠 Технологический стек

- **Core:** Golang (Clean Architecture)
- **Transport:** gRPC + Protocol Buffers
- **Infrastructure:** Kafka (Message Broker), Redis (Idempotency Store), PostgreSQL (Primary DB)
- **Deployment:** Docker Compose, Makefile

## 📈 План развития (Roadmap)
- [ ] Внедрение Prometheus метрик (RPS, Latency, Kafka Lag).
- [ ] Настройка Structured Logging для распределенной трассировки.

## 🚀 Быстрый запуск

```bash
# Поднять всю инфраструктуру и приложение
make run

# Прогнать тесты
make test

## Точки оптимизации
1. использование bytes вместо строки