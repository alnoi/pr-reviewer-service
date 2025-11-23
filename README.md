# PR Reviewer Service

PR Reviewer Service — сервис на Go, который управляет командами, ревьюерами и pull request’ами.  
Он предоставляет простой REST API, содержит юнит‑тесты, интеграционные тесты, нагрузочное тестирование и полноценную observability (трейсы, метрики, профилирование).

##  Архитектура

Сервис построен по принципам Clean Architecture:

- `internal/usecase` — бизнес‑логика  
- `internal/repository` — Postgres‑репозитории  
- `internal/http/v1` — HTTP‑хендлеры  
- `db` — миграции, транзакции  
- `config` — конфигурация  
- `metrics` — Prometheus метрики  

Все детали API перечислены в **openapi.yml** — именно он является контрактом сервиса.

## Тестирование

### Unit Tests
- Покрывают всю бизнес‑логику.
- Используются gomock + testify.
- Table‑driven тесты.

Запуск:

```bash
go test ./... -cover
```

---

### Integration / E2E Tests
- Поднимается реальный Postgres через testcontainers.
- Прогоняются миграции.
- Запускается HTTP‑сервер через httptest.Server.
- Тесты вызывают реальный API.

Запуск:

```bash
go test ./... -tags=integration
```

---

### Load Testing (k6)

Сценарий в `loadtest/pr_load_test.js`.

Запуск:

```bash
k6 run loadtest/pr_load_test.js
```

Результаты нагрузки (10 → 30 VU, 1 минута):

- Всего запросов: **1197**
- Средний RPS: **19.7**
- Ошибок: **0**
- p95 latency: **18.96 ms**
- AVG: **8.17 ms**
- Max: **80.98 ms**

Сервис стабилен под нагрузкой и демонстрирует низкие задержки.

---

## Observability

###  Prometheus Metrics
Метрики доступны по `/metrics`.

Основные бизнес‑метрики:

- `pr_created_total`
- `team_created_total`
- `team_deactivated_total`
- `pr_reassigned_total`


Активируется:

```
JAEGER_COLLECTOR_URL=http://localhost:14268/api/traces
```

### Pyroscope Profiling
Включается флагом:

```
PYROSCOPE_ENABLED=true
```

---

## Конфигурация

```text
HTTP_PORT
METRICS_PORT
JAEGER_COLLECTOR_URL
PYROSCOPE_ENABLED
PYROSCOPE_SERVER_ADDRESS
DB_* (host, port, user, pass, name)
```

---

##  API

Полный контракт: **openapi.yml**  
Содержит:
- все ручки,
- схемы запросов и ответов,
- коды ошибок.

---

## Запуск

```bash
make up
```
