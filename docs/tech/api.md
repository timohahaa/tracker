# API Map

## Conventions

- Base path: `/api/workspaces/{workspaceID}` — workspace UUID в path (кроме `/api/auth/*`)
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
GET /api/workspaces/{workspaceID}/issues?limit=50&sort=updated_at&order=desc&cursor=eyJ2IjoiMjAyNi0wMS0xNVQxMDowMDowMFoiLCJpZCI6ImFiYy0xMjMifQ==
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

Без workspace prefix — пользователь ещё не выбрал workspace.

| Method | Path | Description | Access |
|--------|------|-------------|--------|
| POST | `/api/auth/register` | Регистрация + создание workspace | Public |
| POST | `/api/auth/login` | Логин, получение session cookie | Public |
| POST | `/api/auth/logout` | Logout, удаление сессии | Authenticated |
| POST | `/api/auth/accept-invite` | Принять инвайт по токену | Public |

---

## Workspace

| Method | Path | Description | Access |
|--------|------|-------------|--------|
| GET | `/api/workspaces/{workspaceID}` | Текущий workspace | Member |
| PATCH | `/api/workspaces/{workspaceID}` | Обновление настроек | Owner/Admin |
| GET | `/api/workspaces/{workspaceID}/members` | Список участников | Member |
| PATCH | `/api/workspaces/{workspaceID}/members/:userID` | Обновить участника (role, username, title) | Owner/Admin |
| DELETE | `/api/workspaces/{workspaceID}/members/:userID` | Удалить участника | Owner/Admin |
| POST | `/api/workspaces/{workspaceID}/invites` | Приглашение в workspace | Owner/Admin |

---

## Teams

| Method | Path | Description | Access |
|--------|------|-------------|--------|
| GET | `/api/workspaces/{workspaceID}/teams` | Список команд workspace | Member |
| POST | `/api/workspaces/{workspaceID}/teams` | Создать команду (+ дефолтные статусы) | Member |
| GET | `/api/workspaces/{workspaceID}/teams/:teamID` | Детали команды | Member |
| PATCH | `/api/workspaces/{workspaceID}/teams/:teamID` | Обновить команду | Team member |
| DELETE | `/api/workspaces/{workspaceID}/teams/:teamID` | Удалить команду | Owner/Admin |
| GET | `/api/workspaces/{workspaceID}/teams/:teamID/members` | Участники команды | Member |
| POST | `/api/workspaces/{workspaceID}/teams/:teamID/members` | Добавить участника | Team member |
| DELETE | `/api/workspaces/{workspaceID}/teams/:teamID/members/:userID` | Удалить участника | Team member |
| GET | `/api/workspaces/{workspaceID}/teams/:teamID/statuses` | Статусы команды | Member |
| POST | `/api/workspaces/{workspaceID}/teams/:teamID/statuses` | Создать статус | Team member |
| PATCH | `/api/workspaces/{workspaceID}/teams/:teamID/statuses/:statusID` | Обновить статус | Team member |
| DELETE | `/api/workspaces/{workspaceID}/teams/:teamID/statuses/:statusID` | Удалить статус (не дефолтный) | Team member |

---

## Issues

| Method | Path | Description | Access |
|--------|------|-------------|--------|
| GET | `/api/workspaces/{workspaceID}/issues` | Список задач (фильтры, поиск, пагинация) | Member |
| POST | `/api/workspaces/{workspaceID}/issues` | Создать задачу | Team member |
| GET | `/api/workspaces/{workspaceID}/issues/:issueID` | Детали задачи (UUID или identifier) | Member |
| PATCH | `/api/workspaces/{workspaceID}/issues/:issueID` | Обновить задачу (partial) | Team member |
| DELETE | `/api/workspaces/{workspaceID}/issues/:issueID` | Удалить задачу | Team member |
| POST | `/api/workspaces/{workspaceID}/issues/:issueID/assignees` | Добавить участника (`{user_id}`) | Team member |
| DELETE | `/api/workspaces/{workspaceID}/issues/:issueID/assignees/:userID` | Убрать участника | Team member |
| PUT | `/api/workspaces/{workspaceID}/issues/:issueID/assignees/:userID/primary` | Назначить ответственным (только из участников) | Team member |

**Assignees:**

- В теле PATCH `/issues/:issueID` можно передать массив `assignees` (replace-семантика всего набора):

```json
{
  "assignees": [
    { "user_id": "...", "is_primary": true },
    { "user_id": "...", "is_primary": false }
  ]
}
```

- Точечные операции — через sub-resource endpoints выше.
- Правила: `is_primary` максимум у одного участника; ответственный обязан быть среди участников; снятие ответственного оставляет задачу без ответственного.

Форма поля `assignees` в ответе issue совпадает с примером выше.

**Фильтры GET /api/workspaces/{workspaceID}/issues:**

| Param | Type | Notes |
|-------|------|-------|
| team | string | team ID (UUID). Поддержка key (ENG) — post-MVP |
| status | string | status ID |
| status_group | string | backlog, planned, in_progress, completed, canceled |
| assignee | string | user ID (любой участник задачи) |
| responsible | string | user ID (только ответственный) |
| priority | string | urgent, high, medium, low, none |
| project | string | project ID |
| label | string | label ID (repeatable) |
| due_date_before | string | задачи с дедлайном до даты (ISO 8601 date) |
| due_date_after | string | задачи с дедлайном после даты (ISO 8601 date) |
| q | string | full-text search (title + description) |
| sort | string | created_at, updated_at, priority (default: updated_at) |
| order | string | asc, desc (default: desc) |

---

## Projects

| Method | Path | Description | Access |
|--------|------|-------------|--------|
| GET | `/api/workspaces/{workspaceID}/projects` | Список проектов | Member |
| POST | `/api/workspaces/{workspaceID}/projects` | Создать проект | Member |
| GET | `/api/workspaces/{workspaceID}/projects/:projectID` | Детали проекта | Member |
| PATCH | `/api/workspaces/{workspaceID}/projects/:projectID` | Обновить проект | Member |
| DELETE | `/api/workspaces/{workspaceID}/projects/:projectID` | Удалить проект | Owner/Admin |

---

## Labels

| Method | Path | Description | Access |
|--------|------|-------------|--------|
| GET | `/api/workspaces/{workspaceID}/labels` | Все labels (workspace + team level) | Member |
| POST | `/api/workspaces/{workspaceID}/labels` | Создать label (team_id=null → workspace) | Member |
| PATCH | `/api/workspaces/{workspaceID}/labels/:labelID` | Обновить label (name, color) | Member |
| DELETE | `/api/workspaces/{workspaceID}/labels/:labelID` | Удалить label | Member |

---

## Integrations

| Method | Path | Description | Access |
|--------|------|-------------|--------|
| GET | `/api/workspaces/{workspaceID}/integrations/gitlab` | Статус GitLab-интеграции | Member |
| POST | `/api/workspaces/{workspaceID}/integrations/gitlab` | Настроить GitLab (URL + token) | Owner/Admin |
| DELETE | `/api/workspaces/{workspaceID}/integrations/gitlab` | Удалить интеграцию | Owner/Admin |

---

## API Tokens

| Method | Path | Description | Access |
|--------|------|-------------|--------|
| GET | `/api/workspaces/{workspaceID}/tokens` | Список токенов текущего пользователя | Authenticated |
| POST | `/api/workspaces/{workspaceID}/tokens` | Создать токен (raw показывается один раз) | Authenticated |
| DELETE | `/api/workspaces/{workspaceID}/tokens/:tokenID` | Отозвать токен | Authenticated |

---

## MCP Tools

Endpoint: `/mcp` (HTTP Streamable). Auth: `Authorization: Bearer <api_token>`.

| Tool | Maps to | Description |
|------|---------|-------------|
| `create_issue` | POST /api/workspaces/{workspaceID}/issues | Создать задачу (принимает `assignees` с `is_primary`) |
| `get_issue` | GET /api/workspaces/{workspaceID}/issues/:id | Получить задачу |
| `update_issue` | PATCH /api/workspaces/{workspaceID}/issues/:id | Обновить задачу (`assignees` — replace-семантика) |
| `delete_issue` | DELETE /api/workspaces/{workspaceID}/issues/:id | Удалить задачу |
| `search_issues` | GET /api/workspaces/{workspaceID}/issues | Поиск с фильтрами |
| `list_projects` | GET /api/workspaces/{workspaceID}/projects | Список проектов |
| `list_teams` | GET /api/workspaces/{workspaceID}/teams | Список команд |

---

## Webhooks

| Method | Path | Description | Auth |
|--------|------|-------------|------|
| POST | `/webhooks/gitlab/{integrationID}` | Входящие события GitLab | X-Gitlab-Token |

Обрабатываемые события:
- `merge_request` (open, merge, close) — автостатус задачи
- `push` — привязка коммитов (future)

Issue matching: парсинг identifier (`TEAM-123`) из branch name, MR title, MR description.
