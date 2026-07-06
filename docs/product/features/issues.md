# Issues

## MVP

### Fields

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| Identifier | auto | — | Формат `TEAM-123`, генерируется автоматически |
| Title | text | yes | |
| Description | markdown | no | |
| Status | enum (grouped) | yes | Из набора статусов команды, дефолт — Backlog |
| Assignees | user[] | no | Участники задачи (несколько), члены команды |
| Responsible | user | no | Ответственный, выбирается из участников (как в Kaiten) |
| Priority | enum | no | Urgent, High, Medium, Low, No priority |
| Project | relation | no | |
| Labels | tags | no | Workspace-level + team-level |
| Due date | date | no | Дедлайн задачи |

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

### Assignees & Responsible

- У задачи может быть несколько участников (assignees) и один ответственный (responsible).
- Ответственный выбирается из участников: нельзя назначить ответственным того, кто не участник.
- Ответственный может отсутствовать (участники есть, ответственного нет) — и наоборот, участников может не быть вовсе.
- Смена ответственного — атомарная: прежний перестаёт быть ответственным.
- Снятие участника, который был ответственным → задача остаётся без ответственного (роль никому не передаётся автоматически).
- Участник должен быть членом команды задачи.

### Issue List / Views

- Просмотр задач списком (основной вид, не доска).
- Базовая фильтрация: по статусу, участнику (assignee), ответственному (responsible), priority, project, labels.
- Поиск по title и description.
- Сортировка: по дате создания, обновления, приоритету.

### Edit Issue

- Все поля редактируемы.
- Изменение статуса — через выбор из доступных статусов команды.
- Управление участниками: добавление/удаление участников, назначение ответственного (только из числа участников).

## Post-MVP

- Sub-issues (подзадачи).
- Estimates (оценки трудозатрат).
- Relations между задачами (blocks, related to, duplicate).
- Activity log (история изменений).
- Comments / discussion.
- Attachments.
- Kanban board view + manual issue ordering (ручная сортировка задач внутри вида).
- Favourites (избранные задачи).
- Bulk operations (массовое обновление).
- Templates для задач.
