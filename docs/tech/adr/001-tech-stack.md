# ADR-001: Tech Stack

## Status

Accepted

## Context

Нужно выбрать технологический стек для MVP AI-native таск-трекера. Ключевые требования:

- **API-first**: REST API для UI + MCP endpoint для ИИ-агентов — оба first-class.
- **Real-time (будущее)**: архитектура должна допускать добавление WebSocket/SSE позже.
- **GitLab webhook-обработка**: приём и обработка HTTP-вебхуков.
- **Linear-like UX**: быстрый SPA с клавиатурной навигацией.
- **Команда**: один разработчик с опытом в Go (MCP-сервер Jarvis, GitLab-интеграции, PostgreSQL).
- **MVP scope**: скорость разработки важнее масштабируемости.

## Considered Options

### Backend

1. **Go + chi + sqlx** — знакомый стек, производительность, простота деплоя (один бинарник).
2. **TypeScript + Node (Fastify/Express)** — единый язык с фронтендом, большая экосистема.
3. **Rust (Axum)** — максимальная производительность, но долгая разработка.

### Database

1. **PostgreSQL** — реляционная модель идеально ложится на домен (workspaces → teams → issues), JSONB для гибкости, full-text search.
2. **SQLite** — проще для старта, но ограничения по конкурентности.

### Frontend

1. **Vite + React + shadcn/ui** — быстрый старт, готовые компоненты, Tailwind.
2. **Next.js** — SSR/SSG не нужны для SPA-трекера, лишняя сложность.
3. **Vue/Svelte** — меньше экосистема UI-компонентов для task tracker UX.

### Auth

1. **Sessions (web) + API tokens (MCP/API)** — гибрид: надёжные отзываемые сессии для браузера, stateless токены для автоматизации.
2. **JWT everywhere** — проще, но нельзя отозвать до истечения.

## Decision

| Layer | Choice | Notes |
|-------|--------|-------|
| Language (backend) | **Go 1.22+** | Знакомый, быстрый, один бинарник |
| HTTP router | **chi** | Лёгкий, идиоматичный, middleware-friendly |
| DB access | **sqlx** + raw SQL | Полный контроль над запросами, без ORM-магии |
| Database | **PostgreSQL 18** | Реляционная модель, JSONB, FTS, надёжность, нативный `uuidv7()` |
| Migrations | **golang-migrate** | Миграции встроены в бинарник, запускаются при старте |
| Frontend | **Vite + React 18 + TypeScript** | SPA, быстрый HMR |
| UI kit | **shadcn/ui + Tailwind CSS** | Быстрый старт, кастомизируемо, доработка позже |
| Auth (web) | **Server-side sessions** (PostgreSQL-backed) | Отзываемые, secure cookies |
| Auth (API/MCP) | **API tokens** (per user) | Stateless, для ИИ-агентов и автоматизации |
| MCP transport | **HTTP Streamable** (`/mcp` endpoint) | Встроен в основной сервер, подключается к Cursor/агентам локально |
| API spec | **swaggo/swag** | Генерация OpenAPI из аннотаций в Go-коде |
| Frontend API client | **orval** + **TanStack Query** | Генерация TS-клиента + React Query хуков из OpenAPI spec |
| Frontend state | **TanStack Query** (server) + React state (UI) | Кеширование, рефетч, optimistic updates |
| Logging | **logrus** | Structured logging |
| Repo structure | **Monorepo** | `frontend/` + Go backend в корне |
| Deploy (MVP) | **Local / Docker** | Без production-инфраструктуры на этапе MVP |

## Consequences

### Positive

- Один бинарник (backend + MCP + миграции) — простой запуск и тестирование.
- PostgreSQL 18 — встроенная функция `uuidv7()`, UUID генерируются на стороне БД без расширений.
- Монорепо — атомарные изменения фронт+бэк, один PR.
- sqlx + raw SQL — предсказуемые запросы, легко оптимизировать.
- shadcn/ui — быстрый старт с приличным UI, можно заменить компоненты позже.
- Гибридная auth — безопасность для web, удобство для API/MCP.

### Negative

- Go — нет единого языка с фронтендом (нельзя шарить типы напрямую).
- Raw SQL — больше boilerplate, чем ORM; нет автогенерации типов.
- Monorepo — фронтенд и бэкенд деплоятся вместе (не проблема для MVP).
- logrus — maintenance mode, но достаточен для MVP.

### Risks

- **Real-time позже**: chi + stdlib поддерживают WebSocket/SSE, но потребуется отдельный слой (hub, pub/sub). Заложить интерфейс.
- **Масштабирование**: один бинарник ограничивает горизонтальное масштабирование. При необходимости — выделить MCP/webhooks в отдельный сервис.
- **Frontend переписывание**: shadcn/ui хорош для старта, но Linear-like UX может потребовать custom design system. Изолировать UI-компоненты для лёгкой замены.
