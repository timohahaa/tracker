# Workspace Hierarchy

Иерархия сущностей по аналогии с Linear.

## Model

```
Workspace (Organization)
├── Members (users with roles: Owner, Admin, Member)
├── Labels (workspace-level)
├── Teams
│   ├── Members (subset of workspace members)
│   ├── Labels (team-level)
│   ├── Statuses (custom per team, grouped)
│   ├── Issues (always belong to one team)
│   └── Projects (can be shared across teams)
└── Projects
    └── Issues
```

## MVP

### Workspace

- Один workspace = одна организация/компания.
- Уникальный URL-slug (например `acme`).
- Создается при регистрации первого пользователя (Owner).
- Настройки: название, slug, workspace-level labels.

### Teams

- Функциональные группы внутри workspace (Engineering, Design, Product и т.д.).
- У каждой команды свой набор статусов (на основе групп: Backlog, Planned, In Progress, Completed, Canceled).
- У каждой команды свои team-level labels.
- Участники команды — подмножество участников workspace.
- Пользователь может быть в нескольких командах.

### Projects

- Набор задач к конкретному deliverable (фича, релиз).
- Может быть привязан к одной или нескольким командам.
- Задачи могут существовать без проекта.
- Поля: название, описание, статус проекта (Planned, In Progress, Completed, Canceled).

### Issues

- Атомарная единица работы.
- Всегда принадлежит **одной команде**.
- Опционально привязан к проекту.
- Identifier: формат `TEAM-123` (аббревиатура команды + порядковый номер).

## Post-MVP

- Sub-teams (иерархия команд).
- Initiatives (стратегические цели, объединяющие проекты).
- Milestones внутри проектов.
- Cycles (спринты) на уровне команды.
