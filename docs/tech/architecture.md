# System Architecture

## Overview

Монолитное приложение (один Go-бинарник), обслуживающее:
- REST API для фронтенда (SPA)
- MCP endpoint для ИИ-агентов
- Webhook endpoint для GitLab
- Static files (собранный фронтенд в production)

```
┌─────────────────────────────────────────────────────┐
│                    Clients                           │
├──────────────┬──────────────┬───────────────────────┤
│  Browser     │  AI Agent    │  GitLab               │
│  (React SPA) │  (MCP)       │  (Webhooks)           │
└──────┬───────┴──────┬───────┴───────────┬───────────┘
       │              │                   │
       ▼              ▼                   ▼
┌─────────────────────────────────────────────────────┐
│              Go HTTP Server (chi)                    │
│                                                     │
│  /api/*          /mcp           /webhooks/*          │
│  REST API        MCP Streamable  Webhook handlers   │
│                  HTTP                               │
├─────────────────────────────────────────────────────┤
│                 Middleware / Auth                    │
│  sessions (web) │ API tokens (MCP/API) │ signature  │
├─────────────────────────────────────────────────────┤
│                Handler Layer                        │
│  (chi handlers + swaggo annotations)                │
├─────────────────────────────────────────────────────┤
│                Service Layer                        │
│  (business logic, domain rules)                     │
├─────────────────────────────────────────────────────┤
│               Repository Layer                      │
│  (sqlx, raw SQL)                                    │
├─────────────────────────────────────────────────────┤
│                PostgreSQL                           │
└─────────────────────────────────────────────────────┘
```

### URL Namespaces

| Path | Purpose | Auth | Code |
|------|---------|------|------|
| `/api/*` | REST API (frontend, users) | Session cookie / API token | `internal/handler/` |
| `/mcp` | MCP endpoint (AI agents) | API token (header) | `internal/mcp/` |
| `/webhooks/*` | Incoming events (GitLab) | Signature verification | `internal/webhook/` |

В production через reverse proxy:
- `app.myapp.com` → `/api/*`
- `mcp.myapp.com` → `/mcp`
- Webhooks — прямой URL, выдаётся при настройке интеграции.

## Monorepo Structure

```
/
├── cmd/app/                — main.go, точка входа
├── config/
│   └── config.go           — парсинг конфигурации (env vars)
├── internal/
│   ├── app/                — bootstrap: DI, инициализация, запуск сервера
│   ├── handler/            — REST API handlers (/api/*)
│   │   ├── auth/           — login, register, logout, invite
│   │   ├── workspace/      — workspace settings, members
│   │   ├── team/           — teams, team members, statuses
│   │   ├── issue/          — CRUD, list, filter
│   │   ├── project/        — CRUD
│   │   ├── label/          — CRUD
│   │   ├── integration/    — GitLab setup
│   │   ├── token/          — API tokens
│   │   ├── middleware/     — auth (session/token), entity context loaders
│   │   ├── router.go       — сборка chi.Mux, все routes
│   │   └── respond.go      — respondJSON, respondError, mapHTTPStatus
│   ├── webhook/            — incoming webhooks (/webhooks/*)
│   │   ├── handler.go      — chi routes для /webhooks
│   │   └── gitlab.go       — обработка GitLab events
│   ├── mcp/                — MCP server (/mcp)
│   │   ├── server.go       — регистрация tools, endpoint
│   │   └── tools.go        — определение tools (обёртки над service)
│   ├── service/            — бизнес-логика
│   │   ├── issue.go
│   │   ├── team.go
│   │   ├── auth.go
│   │   └── gitlab.go
│   ├── repository/         — SQL-запросы (sqlx)
│   │   ├── issue.go
│   │   ├── team.go
│   │   ├── user.go
│   │   └── session.go
│   ├── model/              — доменные структуры
│   └── errs/               — доменные ошибки, коды, маппинг
├── migrations/             — SQL-миграции (golang-migrate)
├── docs/
│   ├── product/
│   ├── tech/
│   └── swagger/            — сгенерированная OpenAPI spec (swag init)
├── frontend/               — Vite + React + shadcn/ui
│   ├── src/
│   │   ├── api/            — сгенерированный клиент (orval)
│   │   ├── components/
│   │   ├── pages/
│   │   └── App.tsx
│   ├── orval.config.ts
│   ├── vite.config.ts
│   └── package.json
├── docker-compose.yml      — PostgreSQL для разработки
├── Dockerfile
├── Makefile
└── go.mod
```

## Components

### Frontend (Vite + React + shadcn/ui)

- SPA с client-side routing (React Router).
- UI-компоненты на shadcn/ui + Tailwind CSS.
- Серверное состояние через TanStack Query (хуки сгенерированы orval из OpenAPI).
- При разработке: Vite dev server проксирует `/api/*` на Go-бэкенд.
- В production: Go-бинарник раздаёт `frontend/dist/` как static files.

### Backend (Go + chi)

Трёхслойная архитектура:

**Handler Layer** — HTTP-обработчики:
- Принимает запрос, валидирует input, вызывает service, формирует response.
- Swaggo-аннотации для генерации OpenAPI.
- Отдельные группы routes: `/api/v1/*`, `/mcp`, `/webhooks/*`.

**Service Layer** — бизнес-логика:
- Генерация issue identifiers (TEAM-123).
- Проверка прав доступа (роли, принадлежность к команде).
- Оркестрация (например: webhook → найти issue → обновить статус → linkback в GitLab).
- Не знает про HTTP, работает с доменными моделями.

**Repository Layer** — доступ к данным:
- Raw SQL через sqlx.
- Один файл = одна сущность (issues, teams, users, sessions...).
- Возвращает доменные модели.

### MCP Server

- HTTP Streamable MCP, endpoint `/mcp`.
- Встроен в тот же Go-бинарник.
- Аутентификация через API token (header).
- Tools = обёртка над service layer (те же операции, что и REST API).
- Позволяет подключить Cursor/другой AI-агент к локальному серверу для тестирования.

### Webhook Handler (`internal/webhook/`)

- Endpoint `/webhooks/gitlab`.
- Верификация подписи (X-Gitlab-Token).
- Синхронная обработка в handler (для MVP).
- Парсинг событий: merge_request (open, merge, close), push.
- Вызывает service layer для обновления статусов задач.
- Отдельный пакет от `handler/` — другой auth, другой namespace.

### Database (PostgreSQL)

- Единственный data store.
- Хранит: пользователей, workspace, команды, задачи, сессии, API tokens, GitLab-интеграцию.
- Миграции через golang-migrate, встроены в бинарник, применяются при старте.

## Authentication & Authorization

### Web (Browser)

```
Browser → POST /api/auth/login {email, password}
       ← Set-Cookie: session_id=xxx (httpOnly, secure, sameSite)

Browser → GET /api/issues (Cookie: session_id=xxx)
       → Middleware: lookup session in PostgreSQL → set user in context
```

Сессии хранятся в PostgreSQL. Cookie — httpOnly, secure, SameSite=Lax.

### API / MCP (AI Agents)

```
Agent → POST /mcp (Header: Authorization: Bearer <api_token>)
     → Middleware: lookup token in PostgreSQL → set user in context
```

API tokens — длинные случайные строки, привязаны к пользователю. Создаются в UI.

## Request Flow (Example: Create Issue)

```
1. Frontend: useCreateIssue({ title: "Fix bug", teamId: "..." })
2. HTTP: POST /api/issues { title, team_id, ... }
3. Middleware: session auth → user in context
4. Handler: parse body → validate → call service
5. Service: check user is team member → generate identifier → call repo
6. Repository: INSERT INTO issues (...) RETURNING * → return model
7. Handler: map model → response DTO → 201 JSON
8. Frontend: TanStack Query invalidates issue list cache
```

## External Integrations

### GitLab (MVP)

- **Inbound**: GitLab → webhook → наш сервер (merge_request events).
- **Outbound**: наш сервер → GitLab API (linkback-комментарии).
- Настройка: admin добавляет GitLab URL + access token в workspace settings.
- Один webhook URL на workspace.

## Development Workflow

```bash
# Запуск БД
docker-compose up -d postgres

# Запуск бэкенда (применяет миграции автоматически)
go run ./cmd/app

# Запуск фронтенда (proxy на бэкенд)
cd frontend && npm run dev

# Генерация OpenAPI spec
swag init -g cmd/app/main.go -o docs/swagger

# Генерация TS-клиента
cd frontend && npx orval
```

## Future Considerations

- **Real-time (WebSocket/SSE)**: chi поддерживает, добавить pub/sub слой (in-process для начала, Redis pub/sub при масштабировании).
- **Выделение MCP в отдельный сервис**: если нагрузка от агентов вырастет.
- **Background jobs**: для email-уведомлений, тяжёлых операций — PostgreSQL-based queue (pgq) или отдельный worker.
- **CDN для фронтенда**: при масштабировании вынести static files из Go-бинарника.
