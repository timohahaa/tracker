# API Map

## Conventions

- Base path: `/api`
- Format: JSON (request & response)
- Auth: session cookie (web) или `Authorization: Bearer <api_token>` (API/MCP). См. [auth.md](./auth.md).
- Errors: единый формат. См. [error-handling.md](./error-handling.md).
- Pagination: cursor-based (`?cursor=xxx&limit=50`)
- Soft delete: DELETE возвращает 204, ресурс помечается deleted_at
- Timestamps: ISO 8601 (UTC)

## Pagination

Cursor-based (keyset pagination) для стабильной пагинации при live-данных (нет дубликатов/пропусков при обновлениях, стабильная производительность на любой глубине).

**Принципы:**
- Одно поле сортировки + `id` как tiebreaker (аналогично Linear)
- Cursor = base64-encoded JSON с sort value + id
- Смена sort или фильтров → cursor сбрасывается (фронт запрашивает первую страницу)
- Jump to page N не поддерживается (infinite scroll / "Load more")

**Request:**
```
GET /api/issues?limit=50&sort=updated_at&order=desc&cursor=eyJ2IjoiMjAyNi0wMS0xNVQxMDowMDowMFoiLCJpZCI6ImFiYy0xMjMifQ==
```

**Response:**
```json
{
  "data": [...],
  "pagination": {
    "next_cursor": "eyJ2IjoiMjAyNi0wMS0xMFQwODowMDowMFoiLCJpZCI6ImRlZi00NTYifQ==",
    "has_more": true
  }
}
```

**SQL (внутри repository):**
```sql
SELECT * FROM issues
WHERE team_id = $1                              -- фильтры
  AND (updated_at, id) < ($cursor_val, $cursor_id)  -- cursor
ORDER BY updated_at DESC, id DESC
LIMIT 50
```

---

## Auth

| Method | Path | Description | Access |
|--------|------|-------------|--------|
| POST | `/api/auth/register` | Регистрация + создание workspace | Public |
| POST | `/api/auth/login` | Логин, получение session cookie | Public |
| POST | `/api/auth/logout` | Logout, удаление сессии | Authenticated |
| POST | `/api/auth/invite` | Приглашение в workspace | Owner/Admin |

---

## Workspace

| Method | Path | Description | Access |
|--------|------|-------------|--------|
| GET | `/api/workspace` | Текущий workspace | Member |
| PATCH | `/api/workspace` | Обновление настроек | Owner/Admin |
| GET | `/api/workspace/members` | Список участников | Member |
| DELETE | `/api/workspace/members/:userID` | Удалить участника | Owner/Admin |

---

## Teams

| Method | Path | Description | Access |
|--------|------|-------------|--------|
| GET | `/api/teams` | Список команд workspace | Member |
| POST | `/api/teams` | Создать команду (+ дефолтные статусы) | Member |
| GET | `/api/teams/:teamID` | Детали команды | Member |
| PATCH | `/api/teams/:teamID` | Обновить команду | Team member |
| GET | `/api/teams/:teamID/members` | Участники команды | Member |
| POST | `/api/teams/:teamID/members` | Добавить участника | Team member |
| DELETE | `/api/teams/:teamID/members/:userID` | Удалить участника | Team member |
| GET | `/api/teams/:teamID/statuses` | Статусы команды | Member |
| POST | `/api/teams/:teamID/statuses` | Создать статус | Team member |
| PATCH | `/api/teams/:teamID/statuses/:statusID` | Обновить статус | Team member |
| DELETE | `/api/teams/:teamID/statuses/:statusID` | Удалить статус (не дефолтный) | Team member |

---

## Issues

| Method | Path | Description | Access |
|--------|------|-------------|--------|
| GET | `/api/issues` | Список задач (фильтры, поиск, пагинация) | Member |
| POST | `/api/issues` | Создать задачу | Team member |
| GET | `/api/issues/:issueID` | Детали задачи (UUID или identifier) | Member |
| PATCH | `/api/issues/:issueID` | Обновить задачу (partial) | Team member |
| DELETE | `/api/issues/:issueID` | Удалить задачу | Team member |

**Фильтры GET /api/issues:**

| Param | Type | Notes |
|-------|------|-------|
| team | string | team ID or key (ENG) |
| status | string | status ID |
| status_group | string | backlog, planned, in_progress, completed, canceled |
| assignee | string | user ID |
| priority | string | urgent, high, medium, low, none |
| project | string | project ID |
| label | string | label ID (repeatable) |
| q | string | full-text search (title + description) |
| sort | string | created_at, updated_at, priority (default: updated_at) |
| order | string | asc, desc (default: desc) |

---

## Projects

| Method | Path | Description | Access |
|--------|------|-------------|--------|
| GET | `/api/projects` | Список проектов | Member |
| POST | `/api/projects` | Создать проект | Member |
| GET | `/api/projects/:projectID` | Детали проекта | Member |
| PATCH | `/api/projects/:projectID` | Обновить проект | Member |
| DELETE | `/api/projects/:projectID` | Удалить проект | Owner/Admin |

---

## Labels

| Method | Path | Description | Access |
|--------|------|-------------|--------|
| GET | `/api/labels` | Все labels (workspace + team level) | Member |
| POST | `/api/labels` | Создать label (team_id=null → workspace) | Member |
| DELETE | `/api/labels/:labelID` | Удалить label | Member |

---

## Integrations

| Method | Path | Description | Access |
|--------|------|-------------|--------|
| GET | `/api/integrations/gitlab` | Статус GitLab-интеграции | Member |
| POST | `/api/integrations/gitlab` | Настроить GitLab (URL + token) | Owner/Admin |
| DELETE | `/api/integrations/gitlab` | Удалить интеграцию | Owner/Admin |

---

## API Tokens

| Method | Path | Description | Access |
|--------|------|-------------|--------|
| GET | `/api/tokens` | Список токенов текущего пользователя | Authenticated |
| POST | `/api/tokens` | Создать токен (raw показывается один раз) | Authenticated |
| DELETE | `/api/tokens/:tokenID` | Отозвать токен | Authenticated |

---

## MCP Tools

Endpoint: `/mcp` (HTTP Streamable). Auth: `Authorization: Bearer <api_token>`.

| Tool | Maps to | Description |
|------|---------|-------------|
| `create_issue` | POST /api/issues | Создать задачу |
| `get_issue` | GET /api/issues/:id | Получить задачу |
| `update_issue` | PATCH /api/issues/:id | Обновить задачу |
| `delete_issue` | DELETE /api/issues/:id | Удалить задачу |
| `search_issues` | GET /api/issues | Поиск с фильтрами |
| `list_projects` | GET /api/projects | Список проектов |
| `list_teams` | GET /api/teams | Список команд |

---

## Webhooks

| Method | Path | Description | Auth |
|--------|------|-------------|------|
| POST | `/webhooks/gitlab` | Входящие события GitLab | X-Gitlab-Token |

Обрабатываемые события:
- `merge_request` (open, merge, close) — автостатус задачи
- `push` — привязка коммитов (future)

Issue matching: парсинг identifier (`TEAM-123`) из branch name, MR title, MR description.
