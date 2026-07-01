# API / MCP

## MVP

### Overview

API-first подход: любой ИИ-агент может управлять системой через API/MCP. UI и ИИ — равноправные клиенты.

### REST API

CRUD для основных сущностей. Все endpoint'ы в контексте workspace (UUID в path):

| Endpoint | Methods | Description |
|----------|---------|-------------|
| `/api/workspaces/:workspace_id/issues` | GET, POST | Список / создание задач |
| `/api/workspaces/:workspace_id/issues/:id` | GET, PATCH, DELETE | Чтение / обновление / удаление задачи |
| `/api/workspaces/:workspace_id/projects` | GET, POST | Список / создание проектов |
| `/api/workspaces/:workspace_id/projects/:id` | GET, PATCH, DELETE | Чтение / обновление / удаление проекта |
| `/api/workspaces/:workspace_id/teams` | GET | Список команд |
| `/api/workspaces/:workspace_id/teams/:id` | GET | Информация о команде |

- Аутентификация: session token (cookie) для UI, API key для внешних клиентов.
- Фильтрация задач: по status, assignee, priority, project, team.
- Пагинация: cursor-based.
- Формат: JSON.

### MCP Server

MCP-сервер — отдельный сервис с прямым доступом к БД (MCP как транспортный протокол, не обёртка над REST). Предоставляет ИИ-агентам tools для работы с системой:

| Tool | Description |
|------|-------------|
| `create_issue` | Создать задачу |
| `get_issue` | Получить задачу по ID |
| `update_issue` | Обновить задачу |
| `delete_issue` | Удалить задачу |
| `search_issues` | Поиск задач с фильтрами |
| `list_projects` | Список проектов |
| `list_teams` | Список команд |

Аутентификация: API key (per user, скоупнут на workspace).

## Post-MVP

- GraphQL API (как у Linear).
- Webhooks (система шлет события внешним сервисам).
- OAuth для сторонних приложений.
- Rate limiting.
- Расширенные MCP tools: работа с проектами, командами, статусами, labels.
- MCP resources: контекст workspace, документация по API.
