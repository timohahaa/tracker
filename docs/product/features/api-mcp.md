# API / MCP

## MVP

### Overview

API-first подход: любой ИИ-агент может управлять системой через API/MCP. UI и ИИ — равноправные клиенты.

### REST API

CRUD для основных сущностей:

| Endpoint | Methods | Description |
|----------|---------|-------------|
| `/api/issues` | GET, POST | Список / создание задач |
| `/api/issues/:id` | GET, PATCH, DELETE | Чтение / обновление / удаление задачи |
| `/api/projects` | GET, POST | Список / создание проектов |
| `/api/projects/:id` | GET, PATCH, DELETE | Чтение / обновление / удаление проекта |
| `/api/teams` | GET | Список команд |
| `/api/teams/:id` | GET | Информация о команде |

- Аутентификация: API key (per user или per workspace).
- Фильтрация задач: по status, assignee, priority, project, team.
- Пагинация: cursor-based.
- Формат: JSON.

### MCP Server

MCP-сервер, предоставляющий ИИ-агентам tools для работы с системой:

| Tool | Description |
|------|-------------|
| `create_issue` | Создать задачу |
| `get_issue` | Получить задачу по ID |
| `update_issue` | Обновить задачу |
| `delete_issue` | Удалить задачу |
| `search_issues` | Поиск задач с фильтрами |
| `list_projects` | Список проектов |
| `list_teams` | Список команд |

MCP-сервер — обертка над REST API.

## Post-MVP

- GraphQL API (как у Linear).
- Webhooks (система шлет события внешним сервисам).
- OAuth для сторонних приложений.
- Rate limiting.
- Расширенные MCP tools: работа с проектами, командами, статусами, labels.
- MCP resources: контекст workspace, документация по API.
