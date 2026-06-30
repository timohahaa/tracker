# API Specification

## Conventions

- Base path: `/api`
- Format: JSON (request & response)
- Auth: session cookie (web) или `Authorization: Bearer <api_token>` (API/MCP)
- Pagination: cursor-based (`?cursor=xxx&limit=50`)
- Errors: единый формат `{ "error": { "code": "not_found", "message": "..." } }`
- Soft delete: DELETE возвращает 204, ресурс помечается deleted_at
- Timestamps: ISO 8601 (UTC)

## Error Format

```json
{
  "error": {
    "code": "validation",
    "message": "title is required"
  }
}
```

| HTTP Status | Error Code | When |
|-------------|-----------|------|
| 400 | `validation` | Невалидный input |
| 401 | `unauthorized` | Нет сессии/токена |
| 403 | `forbidden` | Нет прав |
| 404 | `not_found` | Ресурс не найден |
| 409 | `already_exists` | Дубликат (slug, email) |
| 500 | `internal` | Внутренняя ошибка |

## Pagination

Cursor-based для стабильной пагинации при изменении данных.

**Request:**
```
GET /api/issues?limit=50&cursor=eyJpZCI6Inh4eCJ9
```

**Response:**
```json
{
  "data": [...],
  "pagination": {
    "next_cursor": "eyJpZCI6Inl5eSJ9",
    "has_more": true
  }
}
```

---

## Auth

### POST /api/auth/register

Регистрация + создание workspace.

**Request:**
```json
{
  "email": "user@example.com",
  "password": "securepass123",
  "name": "John Doe",
  "workspace_name": "Acme Inc",
  "workspace_slug": "acme"
}
```

**Response (201):**
```json
{
  "user": { "id": "uuid", "email": "...", "name": "..." },
  "workspace": { "id": "uuid", "name": "...", "slug": "acme" }
}
```

Set-Cookie: session_id (httpOnly, secure).

### POST /api/auth/login

**Request:**
```json
{
  "email": "user@example.com",
  "password": "securepass123"
}
```

**Response (200):**
```json
{
  "user": { "id": "uuid", "email": "...", "name": "..." }
}
```

### POST /api/auth/logout

**Response (204):** No content. Cookie cleared.

### POST /api/auth/invite

Приглашение пользователя в workspace (Owner/Admin only).

**Request:**
```json
{
  "email": "newuser@example.com",
  "role": "member"
}
```

**Response (201):**
```json
{
  "invite": { "id": "uuid", "email": "...", "role": "member", "expires_at": "..." }
}
```

---

## Workspace

### GET /api/workspace

Текущий workspace (определяется из сессии).

**Response (200):**
```json
{
  "id": "uuid",
  "name": "Acme Inc",
  "slug": "acme",
  "created_at": "2026-01-01T00:00:00Z"
}
```

### PATCH /api/workspace

**Request:**
```json
{
  "name": "New Name"
}
```

### GET /api/workspace/members

**Response (200):**
```json
{
  "data": [
    { "id": "uuid", "email": "...", "name": "...", "role": "owner" }
  ]
}
```

### DELETE /api/workspace/members/:userID

Удалить участника из workspace (Owner/Admin).

---

## Teams

### GET /api/teams

**Response (200):**
```json
{
  "data": [
    { "id": "uuid", "name": "Engineering", "key": "ENG", "member_count": 5 }
  ]
}
```

### POST /api/teams

**Request:**
```json
{
  "name": "Engineering",
  "key": "ENG"
}
```

**Response (201):** Team object. Дефолтные статусы (по одному на группу) создаются автоматически.

### GET /api/teams/:teamID

### PATCH /api/teams/:teamID

### GET /api/teams/:teamID/members

### POST /api/teams/:teamID/members

**Request:**
```json
{
  "user_id": "uuid"
}
```

### DELETE /api/teams/:teamID/members/:userID

### GET /api/teams/:teamID/statuses

**Response (200):**
```json
{
  "data": [
    { "id": "uuid", "name": "Backlog", "group": "backlog", "position": 0 },
    { "id": "uuid", "name": "In Progress", "group": "in_progress", "position": 0 },
    { "id": "uuid", "name": "Code Review", "group": "in_progress", "position": 1 }
  ]
}
```

### POST /api/teams/:teamID/statuses

**Request:**
```json
{
  "name": "Code Review",
  "group": "in_progress",
  "position": 1
}
```

### PATCH /api/teams/:teamID/statuses/:statusID

### DELETE /api/teams/:teamID/statuses/:statusID

Нельзя удалить дефолтный статус группы.

---

## Issues

### GET /api/issues

**Query params:**
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
| cursor | string | pagination cursor |
| limit | int | 1-100, default 50 |

**Response (200):**
```json
{
  "data": [
    {
      "id": "uuid",
      "identifier": "ENG-123",
      "title": "Fix login bug",
      "description": "...",
      "status": { "id": "uuid", "name": "In Progress", "group": "in_progress" },
      "priority": "high",
      "assignee": { "id": "uuid", "name": "John", "email": "..." },
      "project": { "id": "uuid", "name": "Auth Rewrite" },
      "labels": [{ "id": "uuid", "name": "bug", "color": "#ff0000" }],
      "created_at": "...",
      "updated_at": "..."
    }
  ],
  "pagination": { "next_cursor": "...", "has_more": true }
}
```

### POST /api/issues

**Request:**
```json
{
  "title": "Fix login bug",
  "description": "Users can't login with email",
  "team_id": "uuid",
  "status_id": "uuid",
  "priority": "high",
  "assignee_id": "uuid",
  "project_id": "uuid",
  "label_ids": ["uuid", "uuid"]
}
```

Required: `title`, `team_id`. Остальное опционально (status defaults to first backlog status).

**Response (201):** Full issue object.

### GET /api/issues/:issueID

`:issueID` — UUID или identifier (`ENG-123`).

### PATCH /api/issues/:issueID

Partial update — только переданные поля обновляются.

**Request:**
```json
{
  "status_id": "uuid",
  "priority": "medium"
}
```

### DELETE /api/issues/:issueID

**Response (204).**

---

## Projects

### GET /api/projects

**Query params:** `status`, `q`, `cursor`, `limit`.

### POST /api/projects

**Request:**
```json
{
  "name": "Auth Rewrite",
  "description": "...",
  "status": "planned"
}
```

### GET /api/projects/:projectID

### PATCH /api/projects/:projectID

### DELETE /api/projects/:projectID

---

## Labels

### GET /api/labels

Возвращает workspace-level + team-level labels.

**Query params:** `team` (filter by team; без фильтра — все).

**Response (200):**
```json
{
  "data": [
    { "id": "uuid", "name": "bug", "color": "#ff0000", "team_id": null },
    { "id": "uuid", "name": "frontend", "color": "#00ff00", "team_id": "uuid" }
  ]
}
```

### POST /api/labels

**Request:**
```json
{
  "name": "bug",
  "color": "#ff0000",
  "team_id": null
}
```

`team_id: null` → workspace-level. `team_id: "uuid"` → team-level.

### DELETE /api/labels/:labelID

---

## Integrations

### GET /api/integrations/gitlab

**Response (200):**
```json
{
  "configured": true,
  "gitlab_url": "https://gitlab.example.com",
  "webhook_url": "https://webhooks.myapp.com/webhooks/gitlab/uuid-secret"
}
```

### POST /api/integrations/gitlab

**Request:**
```json
{
  "gitlab_url": "https://gitlab.example.com",
  "access_token": "glpat-xxx"
}
```

Генерирует webhook URL и secret. Response включает webhook_url для настройки в GitLab.

### DELETE /api/integrations/gitlab

---

## API Tokens

### GET /api/tokens

**Response (200):**
```json
{
  "data": [
    { "id": "uuid", "name": "Cursor Agent", "last_used_at": "...", "created_at": "..." }
  ]
}
```

### POST /api/tokens

**Request:**
```json
{
  "name": "Cursor Agent"
}
```

**Response (201):**
```json
{
  "id": "uuid",
  "name": "Cursor Agent",
  "token": "tk_live_xxxxxxxxxxxxx"
}
```

`token` показывается **только один раз** при создании.

### DELETE /api/tokens/:tokenID

---

## MCP Tools

MCP endpoint: `/mcp` (HTTP Streamable). Auth: `Authorization: Bearer <api_token>`.

Tools — обёртки над REST API:

| Tool | Maps to |
|------|---------|
| `create_issue` | POST /api/issues |
| `get_issue` | GET /api/issues/:id |
| `update_issue` | PATCH /api/issues/:id |
| `delete_issue` | DELETE /api/issues/:id |
| `search_issues` | GET /api/issues (with filters) |
| `list_projects` | GET /api/projects |
| `list_teams` | GET /api/teams |

---

## Webhooks

### POST /webhooks/gitlab

Входящие события от GitLab. Auth: `X-Gitlab-Token` header с secret.

Обрабатываемые события:
- `merge_request` (open, merge, close) — автостатус задачи
- `push` — привязка коммитов (future)

Issue matching: парсинг identifier (`TEAM-123`) из branch name, MR title, MR description.
