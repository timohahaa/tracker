# Data Model

## Decisions

### Primary Keys
- UUID v7 everywhere (сортируемый по времени, хорош для cursor pagination). Генерируется нативным `uuidv7()` в `DEFAULT` (требует PostgreSQL 18+).
- Некоторые сущности имеют public-facing ID (issue identifier `TEAM-123`, workspace slug, team key).

### Timestamps
- Все таблицы: `created_at`, `updated_at`, `deleted_at` (uniform soft delete).
- Исключения: `sessions` (нет `updated_at`/`deleted_at` — сессии истекают по `expires_at`, logout = hard delete).
- Junction-таблицы тоже имеют `deleted_at` — разрыв связи = soft delete.
- Junction-таблицы используют composite PK + upsert-паттерн: удаление = `UPDATE SET deleted_at = now()`, восстановление = `UPDATE SET deleted_at = NULL` (+ обновление полей вроде `role`). INSERT только при первом добавлении, далее — upsert. История удалений/восстановлений не сохраняется (при необходимости — через activity log).

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
| workspace_members | (workspace_id) WHERE role = 'owner' | Один owner на workspace |
| teams | (workspace_id, key) | Удалил команду ENG, создал новую |
| labels | (workspace_id, name) WHERE team_id IS NULL | Workspace-level labels уникальны по имени |
| labels | (team_id, name) WHERE team_id IS NOT NULL | Team-level labels уникальны внутри команды |
| integrations | (workspace_id, provider) | Переподключение интеграции |
| statuses | (team_id, group, position) | Уникальный порядок внутри группы |
| issues | (team_id, issue_number) | Уникальность identifier внутри команды |
| issue_assignees | (issue_id) WHERE is_primary | Один ответственный на issue |
| invites | (workspace_id, email) WHERE accepted_at IS NULL | Один активный инвайт на email в workspace |
| attachments | (url, issue_id) | Пересоздание привязки |

### Multi-workspace
- User может быть в нескольких workspaces.
- Workspace context в API URL: `/api/workspaces/{workspaceID}/issues` — UUID в path.
- API tokens (для MCP) скоупнуты на workspace.

---

## Schemas

Таблицы разложены по Postgres-схемам. Критерий разбиения:

- **`core`** — глобальная идентичность и auth-артефакты: `users`, `sessions`, `api_tokens`.
- **`org`** — оргструктура и доступ, **группировки людей**: `workspaces`, `workspace_members`, `invites`, `teams`, `team_members`.
- **`tracker`** — трекинг работы и планирование, **группировки задач**: `projects`, `project_teams`, `statuses`, `issues`, `issue_assignees`, `labels`, `issue_labels`, `attachments`.
- **`integration`** — внешние интеграции: `integrations` (таблица `connection`).

Правило одной строкой: **`team` группирует людей → `org`; `project` группирует задачи → `tracker`.** Отсюда `projects`/`project_teams` живут в `tracker` рядом с `issues`, а не в `org`. Post-MVP планировочные сущности (`initiatives`, `milestones`, `cycles`) по этому же критерию лягут в `tracker` (или в отдельную `planning`).

Направление FK строго «вниз» по слоям: `tracker → org → core`, `integration → org`, `tracker → integration` (`attachments.integration_id → integrations`). Пересечения схем допустимы (например `tracker.issues → org.teams`), но обратных ссылок (вверх) нет.

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
| description | text | nullable |
| key | text | UNIQUE per workspace (e.g. "ENG"), используется в identifier |
| next_issue_number | int | NOT NULL, DEFAULT 0, последний выданный номер (атомарный инкремент, см. Issue Identifier Generation) |
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
| created_at | timestamptz | NOT NULL |
| updated_at | timestamptz | NOT NULL |
| deleted_at | timestamptz | nullable |

Default statuses: при создании команды — hardcode в service, INSERT 5 статусов (по одному на группу) в транзакции.
"Дефолтный статус группы" = первый по position в группе. Используется при автоматических переходах (GitLab events, создание issue без указания статуса → первый в группе backlog).
Constraint ">=1 статус на группу" — service layer: проверка count перед soft delete, `errs.Validation` если последний в группе.

### issues

| Column | Type | Notes |
|--------|------|-------|
| id | uuid v7 | PK |
| team_id | uuid | FK → teams, NOT NULL |
| project_id | uuid | FK → projects, nullable |
| status_id | uuid | FK → statuses, NOT NULL |
| creator_id | uuid | FK → users, nullable. NULL если создано интеграцией/системой. Post-MVP: bot_actor для non-human creators |
| title | text | NOT NULL |
| description | text | nullable, markdown |
| priority | text | NOT NULL, DEFAULT 'none', CHECK (none / urgent / high / medium / low). Сортировка: CASE WHEN в SQL |
| due_date | date | nullable |
| issue_number | int | NOT NULL, per-team sequential |
| created_at | timestamptz | NOT NULL |
| updated_at | timestamptz | NOT NULL |
| deleted_at | timestamptz | nullable |

Identifier (`ENG-123`): НЕ хранится как колонка. Вычисляется из `teams.key + '-' + issues.issue_number`.
При смене team key — identifier меняется автоматически (без UPDATE issues).
Lookup: `JOIN teams ON ... WHERE teams.key = $1 AND issues.issue_number = $2`.

Assignees: у issue нет колонки `assignee_id`. Участники (assignees) вынесены в junction `issue_assignees` (несколько участников + один ответственный `is_primary`), см. ниже.

### projects

| Column | Type | Notes |
|--------|------|-------|
| id | uuid v7 | PK |
| workspace_id | uuid | FK → workspaces, NOT NULL |
| lead_id | uuid | FK → users, nullable |
| name | text | NOT NULL |
| description | text | nullable |
| status | text | planned / in_progress / completed / canceled |
| created_at | timestamptz | NOT NULL |
| updated_at | timestamptz | NOT NULL |
| deleted_at | timestamptz | nullable |

Project ↔ Team relationship: junction table `project_teams` (см. ниже). Все команды равноправны, ролей нет (post-MVP при необходимости).

Валидация: при привязке issue к проекту team issue должна быть в `project_teams`. Если нет — service layer автоматически добавляет team в `project_teams` (аналог поведения Linear "add team to project").

### labels

| Column | Type | Notes |
|--------|------|-------|
| id | uuid v7 | PK |
| workspace_id | uuid | FK → workspaces, NOT NULL |
| team_id | uuid | FK → teams, nullable (NULL = workspace-level) |
| name | text | NOT NULL |
| color | text | NOT NULL, hex string (e.g. '#3B82F6'), валидация формата в service |
| created_at | timestamptz | NOT NULL |
| updated_at | timestamptz | NOT NULL |
| deleted_at | timestamptz | nullable |

Uniqueness:
- Workspace-level: `UNIQUE(workspace_id, name) WHERE team_id IS NULL AND deleted_at IS NULL`
- Team-level: `UNIQUE(team_id, name) WHERE team_id IS NOT NULL AND deleted_at IS NULL`
- Разные команды могут иметь label с одинаковым именем.
- Пересечение workspace ↔ team: нельзя создать team label с именем, которое уже есть на уровне workspace. При создании workspace-level label с именем, которое уже существует как team label — предлагается merge (все team labels с таким именем заменяются одним workspace label). Проверка в service layer.

### issue_labels

| Column | Type | Notes |
|--------|------|-------|
| issue_id | uuid | FK → issues, PK part |
| label_id | uuid | FK → labels, PK part |
| created_at | timestamptz | NOT NULL |
| deleted_at | timestamptz | nullable |

### issue_assignees

Участники задачи (несколько) + один ответственный (аналог Kaiten: «Участники» + «Ответственный»).

| Column | Type | Notes |
|--------|------|-------|
| issue_id | uuid | FK → issues, PK part |
| user_id | uuid | FK → users, PK part |
| is_primary | bool | NOT NULL, DEFAULT false. Ответственный за задачу |
| created_at | timestamptz | NOT NULL |
| updated_at | timestamptz | NOT NULL |
| deleted_at | timestamptz | nullable |

Бизнес-правила (service layer, кроме отмеченного):
- **≤ 1 ответственный** на issue — гарантируется partial unique index `(issue_id) WHERE is_primary AND deleted_at IS NULL` (единственное правило на уровне БД).
- **Ответственный ⊆ участников**: нельзя пометить `is_primary` пользователя, которого нет среди активных assignees.
- **Смена ответственного**: снять `is_primary` у старого → выставить у нового в одной транзакции (partial unique index нельзя `DEFERRABLE`).
- **Снятие участника-ответственного** → issue остаётся без ответственного (без авто-перевода на другого).
- Assignee должен быть **активным** участником команды issue — проверка в коде (FK на `users` этого не гарантирует, см. Soft Delete).
- При удалении участника из команды — каскадно снять его с задач (soft delete).
- 0 участников допустимо (unassigned).

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
| id | uuid v7 | PK, он же Bearer token (передаётся в `Authorization` header) |
| user_id | uuid | FK → users, NOT NULL |
| workspace_id | uuid | FK → workspaces, NOT NULL |
| name | text | NOT NULL, user-given name |
| last_used_at | timestamptz | nullable |
| created_at | timestamptz | NOT NULL |
| deleted_at | timestamptz | nullable, отзыв токена |

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
| deleted_at | timestamptz | nullable, отмена инвайта |

Повторный инвайт: если существует активная запись (accepted_at IS NULL, deleted_at IS NULL) — upsert: обновить token, expires_at, role, invited_by. Новая запись не создаётся.

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
2. `UPDATE teams SET next_issue_number = next_issue_number + 1 WHERE id = $1 RETURNING next_issue_number` — атомарный инкремент, row-level lock. Возвращается инкрементированное значение: `next_issue_number` = «последний выданный номер» (`DEFAULT 0`), поэтому первый issue получает номер 1.
3. INSERT issue с `issue_number = returned value`.
4. Commit.

Identifier вычисляется при чтении: `team.key || '-' || issue.issue_number`.

---

## Full-Text Search

MVP: `ILIKE '%query%'` по title + description. Индекс: pg_trgm GIN при необходимости.
Post-MVP: tsvector + GIN index (русский + английский словари).

---

## Webhook Flow (GitLab)

1. GitLab POST `/webhooks/gitlab/{integrationID}` → find integration by ID → verify `X-Gitlab-Token` against `integrations.credentials.webhook_secret`
2. Parse event → extract issue identifiers (`TEAM-123`) из branch/title/description
3. Upsert attachment: `INSERT ... ON CONFLICT (url, issue_id) DO UPDATE SET title=$x, metadata=$y, updated_at=now()`
4. Linkback: access_token из `integrations.credentials` → комментарий в GitLab MR

При удалении интеграции (soft delete) attachments остаются как исторические данные.
