# Frontend Implementation Plan

Пошаговый план разработки фронтенда (SPA) поверх готового REST API. Документ —
источник для генерации промптов: каждая фаза/слайс самодостаточны, содержат scope,
deliverables и Definition of Done. Аналог бэкового `PLAN.md`.

## Принцип: гибрид (фундамент → вертикальные слайсы)

- **Не** «сначала все компоненты, потом все страницы»: UI в вакууме, ничего не работает
  до самого конца, нет ранней проверки проводки с API.
- **Не** чистая вертикаль с ходу («всё про issue»): `issue` тянет `team`, `status`,
  `project`, `label`, `members` — пришлось бы мокать пол-домена.
- **Гибрид**: сначала tooling и cross-cutting фундамент (дизайн-система, api-клиент,
  auth-shell, роутинг/лейаут) — один заход, затем каждую сущность делаем **насквозь**
  (data-хуки → компоненты → страница → состояния → проводка в роутер) в порядке
  зависимостей. Для работы с AI это даёт локальный контекст (одна сущность за раз),
  раннюю проверку интеграции с бэком и естественные точки остановки.

## Стек (из ADR-001)

- **Сборка/язык**: Vite + React 18 + TypeScript, SPA.
- **UI**: shadcn/ui + Tailwind CSS.
- **Данные**: `orval` (генерация TS-клиента + React Query хуков из OpenAPI) + TanStack Query.
- **Состояние**: TanStack Query (server) + React state (UI).
- **Роутинг**: React Router.
- **Auth (web)**: server-side sessions через `httpOnly` cookie (клиент токен не хранит).
- **Dev**: Vite dev-server проксирует `/api` на Go-бэкенд (`:8080`).
- **Prod**: Go-бинарник раздаёт `frontend/dist/` как static.

Источник типов/клиента — OpenAPI-спека бэка (`make swag-init` → `docs/swagger`, в debug
доступна на `/docs` + `/docs/openapi.json`). Клиент регенерируется через `orval`.

## Предпосылки бэка (готовы)

Фаза 1 опирается на два эндпоинта (реализованы, контракты — в `docs/tech/api.md`):

- `GET /api/auth/me` (за `SessionAuth`) → объект пользователя напрямую (без обёртки `{ user }`),
  с полем `settings.current_workspace_id`; `401` если сессии нет/истекла. Используется для
  восстановления сессии при reload (cookie — `httpOnly`, клиент её не читает).
- `GET /api/workspaces` (за `SessionAuth`, без `{workspaceID}` в path) → `{ data: [{ id, name,
  slug, role, ... }] }`. Источник для дефолтного/выбранного workspace и клиентского резолва
  `slug → workspaceID`.
- `PATCH /api/auth/me` (`{ current_workspace_id }`) — серверный персист текущего workspace
  (синхронизация «resume where you left off» между устройствами).

Онбординг в MVP — только через `register` (создаёт workspace + owner); invite-флоу
(`accept-invite`, `invites`) пока `501` и на фронте не делается.

## Обзор фаз

| Фаза | Название | Суть | Блокирует |
|------|----------|------|-----------|
| **0** | Bootstrap & tooling | scaffold Vite/TS, Tailwind+shadcn, ESLint/Prettier, orval, dev-proxy | всё |
| **1** | Фундамент (без домена) | дизайн-токены/тёмная тема, app-shell, api-клиент, auth-bootstrap, роутинг+guards | слайсы |
| **2.1** | Слайс: auth-страницы | login / register | 2.2+ |
| **2.2** | Слайс: workspace-shell | заход в workspace, участники, настройки | 2.3+ |
| **2.3** | Слайс: teams | список, создание, статусы, участники | 2.4 |
| **2.4** | Слайс: issues | list → detail/edit → assignees/labels (3 захода) | 2.5, 2.6 |
| **2.5** | Слайс: projects | список, деталь, issues проекта | — |
| **2.6** | Слайс: labels | ws/team-level CRUD, цвета | — |
| **2.7** | Слайс: settings | GitLab-интеграция, API-токены | — |
| **3** | Polish (опц.) | command palette (cmdk), хоткеи, optimistic UX, toasts | — |

Порядок внутри Фазы 2 — строго по зависимостям (issue тянет team/status/label/project/members).

## Структура `frontend/`

```
frontend/
├── src/
│   ├── api/
│   │   ├── generated/      — сгенерированный orval клиент + React Query хуки
│   │   └── client.ts       — axios-инстанс (withCredentials), нормализация ошибок
│   ├── app/                — провайдеры (Query, Router, Theme), root
│   ├── components/         — shared UI (shadcn + композиты)
│   ├── features/           — вертикальные слайсы (auth, workspace, team, issue, ...)
│   ├── lib/                — утилиты (pagination, errors, theme tokens)
│   ├── pages/              — страницы/лейауты, привязка к роутам
│   └── main.tsx
├── orval.config.ts
├── vite.config.ts
└── package.json
```

---

# Фаза 0 — Bootstrap & tooling

Только скелет и инструменты, без доменного UI.

**Deliverables:**
1. Scaffold `frontend/` через `npm create vite@latest -- --template react-ts`.
2. Tailwind + `shadcn init`; алиас `@/` в `tsconfig` + `vite.config`.
3. ESLint + Prettier (+ import-order), скрипты `lint`, `typecheck`, `build`, `dev`.
4. `vite.config.ts`: dev-proxy `/api`, `/mcp`, `/docs` → `http://localhost:8080`.
5. TanStack Query provider + Router provider в `main.tsx`.
6. `orval.config.ts`: input = OpenAPI-спека бэка (`../docs/swagger/...` или `/docs/openapi.json`),
   output = `src/api/generated/` (React Query mutator на общем axios-инстансе).
7. `api/client.ts`: axios-инстанс с `withCredentials: true` (cookie-сессия), нормализация
   ошибок в форму `AppError` (`{ code, message, fields[] }`, см. `docs/tech/error-handling.md`).
8. README-раздел: `make swag-init` → `npx orval` → `npm run dev`.

**DoD Фазы 0:**
- `npm run build`, `lint`, `typecheck` — чисто.
- Dev-сервер поднимается, proxy на бэк работает.
- Сгенерированный клиент компилится; smoke-запрос `GET /api/auth/me` без cookie отдаёт `401`
  (health-эндпоинта нет — этого достаточно как проверки живости proxy + роутера).

---

# Фаза 1 — Фундамент (cross-cutting, без домена)

Только то, что не зависит от конкретных сущностей и переиспользуется слайсами.

**Deliverables:**
1. **Дизайн-система / темы**:
   - Tailwind-конфиг + CSS-переменные на `:root`/`.dark` (конвенция shadcn):
     `background, foreground, card, popover, primary, secondary, muted, accent,
     destructive, border, input, ring`.
   - **Тема по умолчанию — dark** (Linear-like). `ThemeProvider`: класс на `<html>`,
     persist в localStorage, respect `prefers-color-scheme`, тумблер light/dark.
   - **Выбор акцентного цвета** — задача этой фазы (подобрать и зафиксировать hex для
     `primary`/`accent` в обеих темах).
   - Доменные токены (используются во всём UI):
     - **Status-группы (5)**: Backlog, Planned, In Progress, Completed, Canceled.
     - **Priority (5)**: Urgent, High, Medium, Low, No priority (иконки-«шкала» + цвет).
     - **Labels**: цвет приходит с бэка (`label.color` hex) → точка+текст с контрастом.
   - Базовый набор shadcn: button, input, textarea, select, dialog, dropdown-menu,
     popover, command, sonner (toast), avatar, badge, tooltip, tabs, checkbox,
     scroll-area, skeleton.
2. **App-shell**: sidebar (навигация по workspace/teams/projects) + topbar, responsive,
   `<Outlet/>`.
3. **Auth-bootstrap**: провайдер current-user + current-workspace; 401-interceptor →
   redirect `/login`. Флоу старта:
   - `GET /api/auth/me` → нет сессии (`401`) → `/login`.
   - есть сессия → `GET /api/workspaces`; строим клиентский маппинг `slug ↔ id` (кешируем).
   - дефолтный workspace: `me.settings.current_workspace_id`, если валиден и присутствует в
     списке; иначе — первый из списка (или экран выбора, если список пуст — по сути только
     после register такого не бывает).
   - смена workspace в UI → `PATCH /api/auth/me { current_workspace_id }` (персист на сервере).
   - неизвестный `slug` в URL → 404/редирект на дефолтный workspace.
   - `LoadWorkspace` на бэке принимает только UUID `{workspaceID}` — резолв `slug → id`
     целиком клиентский (по списку из `/api/workspaces`).
4. **Роутинг + guards**: публичные `/login`, `/register`; защищённые `/:workspaceSlug/...`;
   `ProtectedRoute`, `<Suspense>`-фолбэки, error-boundary.
5. **Общие примитивы**: обёртка над cursor-пагинацией (`useInfiniteQuery` над
   `{ data, pagination.next_cursor, has_more }`), рендер `AppError` (включая `fields[]`
   под инпутами), пустые состояния, skeleton-лоадеры.

**DoD Фазы 1:**
- Незалогиненный → редирект на `/login`.
- Залогиненный с валидной cookie при перезагрузке восстанавливается и попадает в свой
  workspace (URL по slug).
- Переключение light/dark работает и переживает reload; тема по умолчанию — dark.
- App-shell рендерится с рабочей навигацией.

---

# Фаза 2 — Вертикальные слайсы (по зависимостям)

Каждый слайс = data-хуки (из сгенерированного клиента) + компоненты + страница +
состояния (loading skeleton / empty / error) + инвалидция кэша/optimistic + проводка
в роутер.

## 2.1 — auth-страницы
- `/login`, `/register` (register создаёт workspace + owner).
- Формы: react-hook-form + zod; маппинг серверных `fields[]` на инпуты.
- После успеха — редирект в workspace (по slug).
- Состояния: submitting, ошибки 400/401/409.

## 2.2 — workspace-shell
- Заход в workspace (по slug), контекст текущего workspace/роли.
- Список участников; настройки (name/slug — для owner/admin).
- Роль-гейтинг UI (owner/admin/member).

## 2.3 — teams
- Список команд, создание команды (дефолтные статусы создаёт бэк).
- Настройки команды; управление статусами (группировка, запрет удаления дефолтного).
- Участники команды (подмножество участников workspace).

## 2.4 — issues (дробится на 3 захода)
1. **List** — основной вид: список с группировкой по статусу; фильтры из `api.md`
   (team / status / status_group / assignee / responsible / priority / project / label /
   due_date / q); сортировка; infinite scroll (cursor); поиск.
2. **Detail / Create / Edit** — панель/страница задачи: title, markdown-описание, статус,
   priority, project, due date; identifier `TEAM-123`; optimistic update.
3. **Assignees & Labels** — участники + один ответственный (⊆ участников);
   replace-семантика `assignees` / `label_ids` + точечные sub-resource операции; цветные метки.

## 2.5 — projects
- Список, деталь (описание, lead, команды), issues проекта, привязка/отвязка issue к проекту.

## 2.6 — labels
- Workspace-level + team-level CRUD; выбор цвета; использование в issue.

## 2.7 — settings
- GitLab-интеграция: setup / status / delete (credentials в UI не показываем).
- API-токены: список, создание (raw-токен показывается один раз → копировать), отзыв.

---

# Фаза 3 — Polish (опционально)

- Command palette (cmdk) — Linear-подобная навигация/действия.
- Клавиатурные шорткаты для частых операций.
- Optimistic UX на смене статуса/приоритета/assignees.
- Toasts, аккуратные empty-states, микро-анимации.

---

## Definition of Done одного слайса (Фаза 2)

1. Data-хуки на сгенерированном клиенте (query/mutation), корректная инвалидция кэша.
2. Экран(ы) с состояниями: loading (skeleton) / empty / error (`AppError` + `fields[]`).
3. Optimistic update там, где это улучшает UX (статус/приоритет/assignees).
4. Роль-гейтинг UI, где применимо (owner/admin/member).
5. Роуты подключены; навигация из shell работает.
6. `npm run build`, `lint`, `typecheck` — чисто; фича работает против запущенного бэка
   (happy-path + ключевые ошибки 400/401/403/404/409).

---

## Правило нарезки для AI

- Один слайс сущности (хуки + компоненты + страница + состояния) — обычно правильный
  размер чанка: крупный для скорости, влезает в контекст.
- Крупные сущности дробить по операциям (см. `issues`: list → detail → assignees/labels).
- В промпт слайса включать: соответствующий раздел `docs/tech/api.md`, форму DTO из
  сгенерированного клиента, дизайн-токены из Фазы 1 и DoD слайса.
