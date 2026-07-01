# Data Model

## Decisions

### Primary Keys
- UUID v7 everywhere (сортируемый по времени, хорош для cursor pagination).
- Некоторые сущности имеют public-facing ID (issue identifier `TEAM-123`, workspace slug, team key).

### Timestamps
- Все таблицы: `created_at`, `updated_at`, `deleted_at` (uniform soft delete).
- Junction-таблицы тоже имеют `deleted_at` — разрыв связи = soft delete.

### Soft Delete
- Hard delete не используется нигде — все операции удаления через `UPDATE SET deleted_at = now()`.
- Все запросы на чтение фильтруют по `deleted_at IS NULL` (включая JOIN через junction-таблицы).
- UNIQUE constraints → partial unique indexes (`WHERE deleted_at IS NULL`), чтобы soft-deleted записи не блокировали создание новых.
- Каскадный soft delete (удаление родителя → удаление детей) — логика в service layer.
- FK constraints не ломаются: soft-deleted строка остаётся в таблице, FK удовлетворён.

#### Partial Unique Indexes

| Таблица | Поля | Описание |
|---------|------|----------|
| users | email | Перерегистрация после удаления |
| workspaces | slug | Пересоздание workspace с тем же slug |
| workspace_members | (workspace_id, username) | Переиспользование @handle после удаления участника |
| teams | (workspace_id, key) | Удалил команду ENG, создал новую |
| labels | (workspace_id, name) | Удалил label, создал с тем же именем |
| integrations | (workspace_id, provider) | Переподключение интеграции |
| attachments | (url, issue_id) | Пересоздание привязки |
| api_tokens | token_hash | Уникальность среди активных токенов |
| invites | token | Уникальность среди активных инвайтов |

### Multi-workspace
- User может быть в нескольких workspaces.
- Workspace context в API URL: `/api/{workspaceID}/issues` — UUID в path.
- API tokens (для MCP) скоупнуты на workspace.

---

## Tables

### users

| Column | Type | Notes |
|--------|------|-------|
| id | uuid v7 | PK |
| email | text | UNIQUE, NOT NULL |
| password_hash | text | NOT NULL |
| display_name | text | NOT NULL, глобальное имя |
| created_at | timestamptz | NOT NULL |
| updated_at | timestamptz | NOT NULL |
| deleted_at | timestamptz | nullable |

### workspaces

| Column | Type | Notes |
|--------|------|-------|
| id | uuid v7 | PK |
| name | text | NOT NULL |
| slug | text | UNIQUE, NOT NULL, URL-safe |
| created_at | timestamptz | NOT NULL |
| updated_at | timestamptz | NOT NULL |
| deleted_at | timestamptz | nullable |

### workspace_members

| Column | Type | Notes |
|--------|------|-------|
| workspace_id | uuid | FK → workspaces, PK part |
| user_id | uuid | FK → users, PK part |
| role | text | owner / admin / member |
| username | text | UNIQUE per workspace (@handle), partial index (WHERE deleted_at IS NULL) |
| title | text | nullable, job title |
| created_at | timestamptz | NOT NULL |
| updated_at | timestamptz | NOT NULL |
| deleted_at | timestamptz | nullable |

### teams

| Column | Type | Notes |
|--------|------|-------|
| id | uuid v7 | PK |
| workspace_id | uuid | FK → workspaces, NOT NULL |
| name | text | NOT NULL |
| key | text | UNIQUE per workspace (e.g. "ENG"), используется в identifier |
| next_issue_number | int | NOT NULL, DEFAULT 1, атомарный инкремент |
| created_at | timestamptz | NOT NULL |
| updated_at | timestamptz | NOT NULL |
| deleted_at | timestamptz | nullable |

### team_members

| Column | Type | Notes |
|--------|------|-------|
| team_id | uuid | FK → teams, PK part |
| user_id | uuid | FK → users, PK part |
| role | text | owner / member (owners может быть несколько) |
| created_at | timestamptz | NOT NULL |
| updated_at | timestamptz | NOT NULL |
| deleted_at | timestamptz | nullable |

### statuses

| Column | Type | Notes |
|--------|------|-------|
| id | uuid v7 | PK |
| team_id | uuid | FK → teams, NOT NULL |
| name | text | NOT NULL |
| group | text | NOT NULL, CHECK (backlog / planned / in_progress / completed / canceled) |
| position | int | NOT NULL, порядок внутри группы |
| is_default | bool | NOT NULL, DEFAULT false, дефолтный статус группы |
| created_at | timestamptz | NOT NULL |
| updated_at | timestamptz | NOT NULL |
| deleted_at | timestamptz | nullable |

Default statuses: при создании команды — hardcode в service, INSERT 5 статусов в транзакции.
Constraint ">=1 статус на группу" — service layer: проверка count перед soft delete, `errs.Validation` если последний в группе.

### issues

| Column | Type | Notes |
|--------|------|-------|
| id | uuid v7 | PK |
| team_id | uuid | FK → teams, NOT NULL |
| project_id | uuid | FK → projects, nullable |
| status_id | uuid | FK → statuses, NOT NULL |
| assignee_id | uuid | FK → users, nullable |
| title | text | NOT NULL |
| description | text | nullable, markdown |
| priority | int | NOT NULL, DEFAULT 0. 0=none, 1=urgent, 2=high, 3=medium, 4=low |
| issue_number | int | NOT NULL, per-team sequential |
| created_at | timestamptz | NOT NULL |
| updated_at | timestamptz | NOT NULL |
| deleted_at | timestamptz | nullable |

Identifier (`ENG-123`): НЕ хранится как колонка. Вычисляется из `teams.key + '-' + issues.issue_number`.
При смене team key — identifier меняется автоматически (без UPDATE issues).
Lookup: `JOIN teams ON ... WHERE teams.key = $1 AND issues.issue_number = $2`.

### projects

| Column | Type | Notes |
|--------|------|-------|
| id | uuid v7 | PK |
| workspace_id | uuid | FK → workspaces, NOT NULL |
| name | text | NOT NULL |
| description | text | nullable |
| status | text | planned / in_progress / completed / canceled |
| created_at | timestamptz | NOT NULL |
| updated_at | timestamptz | NOT NULL |
| deleted_at | timestamptz | nullable |

Project ↔ Team relationship: junction table `project_teams` (см. ниже). Все команды равноправны, ролей нет (post-MVP при необходимости).

### labels

| Column | Type | Notes |
|--------|------|-------|
| id | uuid v7 | PK |
| workspace_id | uuid | FK → workspaces, NOT NULL |
| team_id | uuid | FK → teams, nullable (NULL = workspace-level) |
| name | text | UNIQUE per workspace (независимо от team scope) |
| color | text | NOT NULL, hex string (e.g. '#3B82F6'), валидация формата в service |
| created_at | timestamptz | NOT NULL |
| updated_at | timestamptz | NOT NULL |
| deleted_at | timestamptz | nullable |

Promotion: если workspace-level label с таким именем уже есть как team-level, предложить "promote" (убрать team_id). Наоборот (workspace → team) нельзя.

### issue_labels

| Column | Type | Notes |
|--------|------|-------|
| issue_id | uuid | FK → issues, PK part |
| label_id | uuid | FK → labels, PK part |
| created_at | timestamptz | NOT NULL |
| deleted_at | timestamptz | nullable |

### sessions

| Column | Type | Notes |
|--------|------|-------|
| id | text | PK, session token |
| user_id | uuid | FK → users, NOT NULL |
| expires_at | timestamptz | NOT NULL |
| created_at | timestamptz | NOT NULL |

### api_tokens

| Column | Type | Notes |
|--------|------|-------|
| id | uuid v7 | PK |
| user_id | uuid | FK → users, NOT NULL |
| workspace_id | uuid | FK → workspaces, NOT NULL |
| token_hash | text | UNIQUE, NOT NULL |
| name | text | NOT NULL, user-given name |
| last_used_at | timestamptz | nullable |
| created_at | timestamptz | NOT NULL |

### invites

| Column | Type | Notes |
|--------|------|-------|
| id | uuid v7 | PK |
| workspace_id | uuid | FK → workspaces, NOT NULL |
| email | text | NOT NULL |
| role | text | NOT NULL, роль при принятии |
| invited_by | uuid | FK → users, NOT NULL |
| token | text | UNIQUE, NOT NULL, для invite link |
| expires_at | timestamptz | NOT NULL |
| accepted_at | timestamptz | nullable |
| created_at | timestamptz | NOT NULL |

### project_teams

| Column | Type | Notes |
|--------|------|-------|
| project_id | uuid | FK → projects, PK part |
| team_id | uuid | FK → teams, PK part |
| created_at | timestamptz | NOT NULL |
| deleted_at | timestamptz | nullable |

### integrations

Generic-таблица для внешних интеграций (по аналогии с Linear). Не привязана к конкретному провайдеру — расширяется добавлением нового `provider` значения и кода обработки.

| Column | Type | Notes |
|--------|------|-------|
| id | uuid v7 | PK |
| workspace_id | uuid | FK → workspaces, NOT NULL |
| provider | text | NOT NULL, CHECK ('gitlab', ...) |
| settings | jsonb | NOT NULL, DEFAULT '{}'. Non-secret config (base_url, webhook_url и т.д.), отдаётся в API |
| credentials | jsonb | NOT NULL, DEFAULT '{}'. Секреты (access_token, webhook_secret). Никогда не отдаётся в API. Post-MVP: AES-шифрование |
| created_at | timestamptz | NOT NULL |
| updated_at | timestamptz | NOT NULL |
| deleted_at | timestamptz | nullable |

UNIQUE(workspace_id, provider) — одна интеграция per provider per workspace.

Примеры settings/credentials для GitLab:
- settings: `{ "base_url": "https://gitlab.company.com" }`
- credentials: `{ "access_token": "glpat-...", "webhook_secret": "..." }`

### attachments

Generic-привязка внешних ресурсов к issues (по аналогии с Linear Attachment). URL — идемпотентный ключ: upsert по `(url, issue_id)`.

| Column | Type | Notes |
|--------|------|-------|
| id | uuid v7 | PK |
| issue_id | uuid | FK → issues, NOT NULL |
| integration_id | uuid | FK → integrations, nullable (NULL = создан без интеграции) |
| source_type | text | NOT NULL, CHECK ('gitlab_mr', 'gitlab_commit', ...) |
| url | text | NOT NULL, canonical external URL |
| title | text | NOT NULL, display name |
| metadata | jsonb | NOT NULL, DEFAULT '{}'. Provider-specific данные (branch, author, MR status и т.д.) |
| created_at | timestamptz | NOT NULL |
| updated_at | timestamptz | NOT NULL |
| deleted_at | timestamptz | nullable |

UNIQUE(url, issue_id) WHERE deleted_at IS NULL — один ресурс на issue среди активных, upsert при повторном webhook.

---

## Issue Identifier Generation

1. Service начинает транзакцию.
2. `UPDATE teams SET next_issue_number = next_issue_number + 1 WHERE id = $1 RETURNING next_issue_number` — атомарный инкремент, row-level lock.
3. INSERT issue с `issue_number = returned value`.
4. Commit.

Identifier вычисляется при чтении: `team.key || '-' || issue.issue_number`.

---

## Full-Text Search

MVP: `ILIKE '%query%'` по title + description. Индекс: pg_trgm GIN при необходимости.
Post-MVP: tsvector + GIN index (русский + английский словари).

---

## Webhook Flow (GitLab)

1. GitLab POST → verify webhook_secret из `integrations.credentials` → find integration row
2. Parse event → extract issue identifiers (`TEAM-123`) из branch/title/description
3. Upsert attachment: `INSERT ... ON CONFLICT (url, issue_id) DO UPDATE SET title=$x, metadata=$y, updated_at=now()`
4. Linkback: access_token из `integrations.credentials` → комментарий в GitLab MR

При удалении интеграции (soft delete) attachments остаются как исторические данные.
