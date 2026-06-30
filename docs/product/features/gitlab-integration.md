# GitLab Integration

## MVP

### Overview

Двусторонняя связь между задачами и GitLab merge requests. Автоматическое обновление статусов задач по событиям в GitLab. По аналогии с Linear.

### Setup

- Admin/Owner подключает GitLab в настройках workspace.
- Указывает GitLab URL (поддержка self-hosted) и API access token.
- Система генерирует webhook URL для настройки в GitLab.
- В GitLab настраивается webhook с событиями: Push events, Merge request events.

### Linking MR to Issue

Привязка MR к задаче по identifier задачи (`TEAM-123`):

1. **В имени ветки** — `feature/TEAM-123-add-login` или `TEAM-123/add-login`.
2. **В title MR** — `TEAM-123: Add login page`.
3. **В описании MR** — magic words: `Fixes TEAM-123`, `Closes TEAM-123`, `Resolves TEAM-123`.

Можно привязать несколько задач к одному MR.

### Auto-status Updates

Автоматическое обновление статуса задачи по событиям MR:

| GitLab Event | Issue Status Transition |
|-------------|------------------------|
| MR opened / draft | -> In Progress (первый статус группы In Progress) |
| MR merged | -> Completed (первый статус группы Completed) |

Автостатусы настраиваемы per team (какой именно статус из группы ставить).

### Linkback

При привязке MR к задаче, система оставляет комментарий в GitLab MR:
- Название задачи, статус, ссылка на задачу в системе.
- Обновляется при изменении статуса задачи.

### Display in Issue

На странице задачи отображаются связанные MR:
- Название MR, статус (open/merged/closed), ссылка на GitLab.

## Post-MVP

- Привязка коммитов (не только MR).
- Pipeline status в задаче (CI passed/failed).
- Reviewers MR видны в задаче.
- Auto-assign задачи при создании MR.
- Интеграция с GitHub, GitFlic.
