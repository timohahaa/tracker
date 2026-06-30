# Issues

## MVP

### Fields

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| Identifier | auto | — | Формат `TEAM-123`, генерируется автоматически |
| Title | text | yes | |
| Description | markdown | no | |
| Status | enum (grouped) | yes | Из набора статусов команды, дефолт — Backlog |
| Assignee | user | no | Один участник команды |
| Priority | enum | no | Urgent, High, Medium, Low, No priority |
| Project | relation | no | |
| Labels | tags | no | Workspace-level + team-level |

### Status Model

Пять групп статусов. В каждой группе — дефолтный статус + кастомные (настраиваются per team).

| Group | Default Status | Semantics |
|-------|---------------|-----------|
| Backlog | Backlog | Задача зафиксирована, но не запланирована |
| Planned | Planned | Задача запланирована к выполнению |
| In Progress | In Progress | Задача в работе |
| Completed | Completed | Задача завершена успешно |
| Canceled | Canceled | Задача отменена |

Команда может добавлять статусы внутри группы. Например, In Progress -> Dev, Code Review, QA.

### Create Issue

- Обязательно: title, team.
- Остальные поля опциональны, заполняются при создании или позже.
- При создании присваивается identifier (`TEAM-N`).

### Issue List / Views

- Просмотр задач списком (основной вид, не доска).
- Базовая фильтрация: по статусу, assignee, priority, project, labels.
- Поиск по title и description.
- Сортировка: по дате создания, обновления, приоритету.

### Edit Issue

- Все поля редактируемы.
- Изменение статуса — через выбор из доступных статусов команды.

## Post-MVP

- Sub-issues (подзадачи).
- Estimates (оценки трудозатрат).
- Due dates.
- Relations между задачами (blocks, related to, duplicate).
- Activity log (история изменений).
- Comments / discussion.
- Attachments.
- Kanban board view (опционально, как альтернативный вид).
- Bulk operations (массовое обновление).
- Templates для задач.
