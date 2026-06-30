# Data Model

## Conventions

- Primary keys: `id UUID DEFAULT gen_random_uuid()`.
- Soft delete: `deleted_at TIMESTAMPTZ` на всех таблицах (если не указано иначе).
- Timestamps: `created_at TIMESTAMPTZ NOT NULL DEFAULT now()`, `updated_at TIMESTAMPTZ NOT NULL DEFAULT now()`.
- Аудит-лог — post-MVP.

## Core Entities

### users

| Column | Type | Notes |
|--------|------|-------|
| id | uuid PK | |
| email | text UNIQUE NOT NULL | |
| password_hash | text NOT NULL | bcrypt/argon2 |
| name | text NOT NULL | display name |
| created_at | timestamptz | |
| updated_at | timestamptz | |
| deleted_at | timestamptz | |

### workspaces

| Column | Type | Notes |
|--------|------|-------|
| id | uuid PK | |
| name | text NOT NULL | |
| slug | text UNIQUE NOT NULL | URL-slug (например `acme`) |
| created_at | timestamptz | |
| updated_at | timestamptz | |
| deleted_at | timestamptz | |

### workspace_members

| Column | Type | Notes |
|--------|------|-------|
| id | uuid PK | |
| workspace_id | uuid FK → workspaces | |
| user_id | uuid FK → users | |
| role | text NOT NULL | `owner`, `admin`, `member` |
| created_at | timestamptz | |
| UNIQUE(workspace_id, user_id) | | |

### teams

| Column | Type | Notes |
|--------|------|-------|
| id | uuid PK | |
| workspace_id | uuid FK → workspaces | |
| name | text NOT NULL | |
| key | text NOT NULL | Аббревиатура для identifiers (ENG, DES) |
| issue_counter | int NOT NULL DEFAULT 0 | Счётчик для TEAM-N |
| created_at | timestamptz | |
| updated_at | timestamptz | |
| deleted_at | timestamptz | |
| UNIQUE(workspace_id, key) | | |

### team_members

| Column | Type | Notes |
|--------|------|-------|
| id | uuid PK | |
| team_id | uuid FK → teams | |
| user_id | uuid FK → users | |
| created_at | timestamptz | |
| UNIQUE(team_id, user_id) | | |

### projects

| Column | Type | Notes |
|--------|------|-------|
| id | uuid PK | |
| workspace_id | uuid FK → workspaces | |
| name | text NOT NULL | |
| description | text | |
| status | text NOT NULL DEFAULT 'planned' | planned, in_progress, completed, canceled |
| created_at | timestamptz | |
| updated_at | timestamptz | |
| deleted_at | timestamptz | |

### issues

| Column | Type | Notes |
|--------|------|-------|
| id | uuid PK | |
| team_id | uuid FK → teams | |
| project_id | uuid FK → projects | nullable |
| identifier | text NOT NULL | `TEAM-123`, generated |
| title | text NOT NULL | |
| description | text | markdown |
| status_id | uuid FK → statuses | |
| priority | text NOT NULL DEFAULT 'none' | urgent, high, medium, low, none |
| assignee_id | uuid FK → users | nullable |
| created_at | timestamptz | |
| updated_at | timestamptz | |
| deleted_at | timestamptz | |
| UNIQUE(team_id, identifier) | | |

### statuses

| Column | Type | Notes |
|--------|------|-------|
| id | uuid PK | |
| team_id | uuid FK → teams | |
| name | text NOT NULL | |
| group | text NOT NULL | backlog, planned, in_progress, completed, canceled |
| position | int NOT NULL DEFAULT 0 | порядок внутри группы |
| created_at | timestamptz | |
| UNIQUE(team_id, name) | | |

### sessions

| Column | Type | Notes |
|--------|------|-------|
| id | uuid PK | session token |
| user_id | uuid FK → users | |
| expires_at | timestamptz NOT NULL | |
| created_at | timestamptz | |

### api_tokens

| Column | Type | Notes |
|--------|------|-------|
| id | uuid PK | |
| user_id | uuid FK → users | |
| workspace_id | uuid FK → workspaces | |
| token_hash | text NOT NULL | hashed token |
| name | text NOT NULL | user-facing label |
| last_used_at | timestamptz | |
| created_at | timestamptz | |
| deleted_at | timestamptz | |

## Open Questions (TBD)

- [ ] **Labels**: одна таблица `labels` с nullable `team_id` (null = workspace-level) или две отдельные таблицы?
- [ ] **Issue ↔ Labels**: many-to-many через `issue_labels` join table.
- [ ] **GitLab integration storage**: отдельная таблица `gitlab_installations` только для GitLab, или generic `integrations` (type + JSON credentials) с прицелом на будущее?
- [ ] **GitLab MR tracking**: таблица `merge_requests` (issue_id, gitlab_mr_id, state, url) для отображения связанных MR в задаче.
- [ ] **Дефолтные статусы**: создаются автоматически при создании команды (по одному на каждую группу).

## Relationships Diagram

```
workspaces 1──* workspace_members *──1 users
workspaces 1──* teams
workspaces 1──* projects
teams      1──* team_members *──1 users
teams      1──* statuses
teams      1──* issues
projects   1──* issues (optional)
issues     *──1 statuses
issues     *──1 users (assignee, optional)
users      1──* sessions
users      1──* api_tokens
```
