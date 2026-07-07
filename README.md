# Tracker

## About

<!-- Краткое описание проекта -->

## Documentation

### Product

- [Product Overview](docs/product/overview.md) — видение, аудитория, проблема
- [MVP Scope](docs/product/mvp-scope.md) — границы MVP, критерии успеха
- [Features](docs/product/features/) — детальная проработка фич

### Technical

- [Architecture](docs/tech/architecture.md) — компоненты системы
- [Data Model](docs/tech/data-model.md) — схема данных
- [API](docs/tech/api.md) — контракты API
- [Infrastructure](docs/tech/infrastructure.md) — деплой, CI/CD
- [Non-Functional Requirements](docs/tech/non-functional.md) — перф, безопасность
- [ADRs](docs/tech/adr/) — архитектурные решения

## Getting Started

<!-- Появится когда будет код -->

## Тесты

- Юнит-тесты (БД не нужна):

```bash
make test          # go test ./...
```

- Integration-тесты (реальный PostgreSQL, файлы с build-тегом `integration`):

```bash
make compose-up    # поднять postgres:18 (БД tracker)
make run           # применяет миграции при старте (migrations.Up); можно прервать после "[migrate]: success"
make test-integration
```

`test-integration` прокидывает `TEST_POSTGRES_DSN` (по умолчанию берётся из `POSTGRES_DSN`).
Можно задать его вручную:

```bash
TEST_POSTGRES_DSN='postgres://postgres:password@localhost:5432/tracker?sslmode=disable' \
  go test -tags=integration ./test/...
```

Миграции применяются снаружи — тесты подключаются к уже мигрированной БД.
Если `TEST_POSTGRES_DSN` не задан, integration-тесты корректно пропускаются (`t.Skip`).
