# MVP Scope

## Goal

Дать командам рабочий таск-трекер с Linear-like UX, моделью статусов, GitLab-интеграцией и API/MCP для ИИ-агентов. Доказать, что подход "система + ИИ" работает лучше, чем "доска с карточками".

## Hierarchy

```
Workspace (1 per organization)
├── Members (Owner, Admin, Member)
├── Labels (workspace-level)
├── Teams
│   ├── Members (subset of workspace members)
│   ├── Labels (team-level)
│   ├── Statuses (grouped, custom per team)
│   └── Issues (always belong to one team)
└── Projects (can span multiple teams)
    └── Issues
```

Issue всегда принадлежит одной команде. Опционально — проекту. Identifier: `TEAM-123`.

## Included Features

| Feature | Details | Spec |
|---------|---------|------|
| Auth | Email + пароль, инвайты, роли (Owner/Admin/Member) | [auth.md](features/auth.md) |
| Workspace | Создание workspace, настройки, workspace-level labels | [workspace-hierarchy.md](features/workspace-hierarchy.md) |
| Teams | Команды, свои статусы, team-level labels | [workspace-hierarchy.md](features/workspace-hierarchy.md) |
| Projects | Проекты (кросс-команд), задачи без проекта | [workspace-hierarchy.md](features/workspace-hierarchy.md) |
| Issues | Задачи с полным набором полей (см. ниже) | [issues.md](features/issues.md) |
| Status model | Группы статусов + кастомные статусы per team | [issues.md](features/issues.md) |
| Issue list | Список задач, фильтрация, поиск, сортировка | [issues.md](features/issues.md) |
| GitLab integration | Привязка MR, автостатусы, linkback | [gitlab-integration.md](features/gitlab-integration.md) |
| API/MCP | CRUD задач для ИИ-агентов | [api-mcp.md](features/api-mcp.md) |

### Issue Fields (MVP)

| Field | Type | Notes |
|-------|------|-------|
| Title | text | обязательное |
| Description | markdown | |
| Status | enum (grouped) | группы: Backlog, Planned, In Progress, Completed, Canceled |
| Assignee | user | |
| Priority | enum | Urgent, High, Medium, Low, No priority |
| Project | relation | |
| Labels | tags | |
| Due date | date | дедлайн задачи |

### Status Model

Пять групп статусов, в каждой — дефолтный одноименный статус. Пользователь может добавлять свои статусы внутри группы.

| Group | Default Status | Example Custom Statuses |
|-------|---------------|------------------------|
| Backlog | Backlog | Triage |
| Planned | Planned | Ready for Dev |
| In Progress | In Progress | Dev, QA, Code Review |
| Completed | Completed | Deployed |
| Canceled | Canceled | Duplicate, Won't Fix |

### GitLab Integration (MVP)

- Привязка MR к задаче по ID задачи в имени ветки или описании MR.
- Автоматическое обновление статуса задачи по событиям MR (MR opened -> In Progress, MR merged -> Completed).
- Linkback-комментарий в GitLab MR со ссылкой на задачу.
- Webhook-based: GitLab шлет события, система обрабатывает.

### API/MCP (MVP)

Минимальный CRUD для ИИ-агентов:
- Создание задачи
- Чтение задачи / списка задач
- Обновление задачи
- Удаление задачи
- Поиск задач

## Excluded Features (and Why)

| Feature | Reason | When |
|---------|--------|------|
| Notifications (in-app, email, messengers) | Сложная инфраструктура, не блокирует core value | Post-MVP |
| ИИ-проработка задач (чат, декомпозиция) | Требует зрелой модели данных | Post-MVP |
| Делегирование задач ИИ-агентам | Зависит от MCP + интеграций | Post-MVP |
| Estimates | Не критично для первой версии | Post-MVP |
| Sub-issues | Усложняет модель данных | Post-MVP |
| Cycles / sprints | Противоречит идее "не доска" | Post-MVP, если нужно |
| Мессенджер-интеграции (Telegram, Пачка) | Нужна система уведомлений | Post-MVP |
| Другие git-хостинги (GitHub, GitFlic) | GitLab — первый, остальные по аналогии | Post-MVP |
| OAuth / SSO | Email + пароль достаточно для старта | Post-MVP |
| Billing / payments | Free tier хватит для MVP | Post-MVP |
| AI: daily digest | Автосаммари за день (что сделано, что в работе, блокеры) — per user, per team | Post-MVP |
| AI: project digest | Автоотчет по проекту (прогресс, риски, что осталось) | Post-MVP |
| AI: workspace digest | Общая сводка по всей команде / всем командам / всем проектам | Post-MVP |
| AI: triage | Автоклассификация входящих задач (приоритет, команда) | Post-MVP |
| AI: sprint planning | AI предлагает состав спринта на основе приоритетов и velocity | Post-MVP |
| AI: проработка задач | Чат с AI для уточнения, декомпозиции задачи | Post-MVP |

## Known Limitations

- Только email + пароль, без OAuth.
- Только GitLab, без других git-хостингов.
- Без уведомлений — пользователь сам проверяет обновления.
- Без биллинга — все бесплатно на этапе MVP.
- MCP — только базовый CRUD, без сложных операций.
