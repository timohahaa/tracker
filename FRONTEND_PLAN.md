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
- **Данные**: `openapi-typescript` (генерация TS-типов из OpenAPI) + `openapi-fetch` +
  `openapi-react-query` (типобезопасные React Query хуки, на `fetch`) + TanStack Query.
- **Состояние**: TanStack Query (server) + React state (UI).
- **Роутинг**: React Router.
- **Auth (web)**: server-side sessions через `httpOnly` cookie (клиент токен не хранит).
- **Пакетный менеджер**: `pnpm`. Прямые зависимости запинены точно (без `^`/`~`),
  `pnpm-lock.yaml` в git, установка `pnpm install --frozen-lockfile`, cooldown на свежие
  версии (`minimumReleaseAge`), build-скрипты зависимостей по allowlist (`allowBuilds`).
- **Dev**: Vite dev-server проксирует `/api` на Go-бэкенд (`:8080`).
- **Prod**: Go-бинарник раздаёт `frontend/dist/` как static.

Источник типов — OpenAPI-спека бэка (`make swag-init` → `docs/swagger/swagger.json`, в debug
доступна на `/docs` + `/docs/openapi.json`). Типы регенерируются через `pnpm run api:gen`
(openapi-typescript → `src/api/generated/schema.ts`); типобезопасные хуки строятся из этих
типов через `openapi-react-query` (см. `src/api/client.ts`, экспорт `$api`).

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
| **0** | Bootstrap & tooling | scaffold Vite/TS, Tailwind+shadcn, ESLint/Prettier, openapi codegen, dev-proxy | всё |
| **1.1** | Дизайн-фундамент | токены, тёмная тема по умолчанию, accent, ThemeProvider, базовый shadcn, preview-страница | 1.4, слайсы |
| **1.2** | Роутинг + примитивы | каркас роутов, guards, error-boundary, рендер `AppError`, empty/skeleton | 1.3, слайсы |
| **1.3** | Auth-bootstrap | current-user/-workspace, session resume, 401→login, slug↔id | 1.4, слайсы |
| **1.4** | App-shell | sidebar + topbar, навигация, тумблер темы | слайсы |
| **2.1** | Слайс: auth-страницы | login / register | 2.2+ |
| **2.2** | Слайс: workspace-shell | заход в workspace, участники, настройки | 2.3+ |
| **2.3** | Слайс: teams | список, создание, статусы, участники | 2.4 |
| **2.4** | Слайс: issues | list → detail/edit → assignees/labels (3 захода) | 2.5, 2.6 |
| **2.5** | Слайс: projects | список, деталь, issues проекта | — |
| **2.6** | Слайс: labels | ws/team-level CRUD, цвета | — |
| **2.7** | Слайс: settings | GitLab-интеграция, API-токены | — |
| **3** | Polish (опц.) | command palette (cmdk), хоткеи, optimistic UX, toasts | — |

Порядок внутри Фазы 2 — строго по зависимостям (issue тянет team/status/label/project/members).
Порядок внутри Фазы 1 — тоже по зависимостям: сперва дизайн (1.1), затем роутинг/примитивы
(1.2), auth-bootstrap (1.3) и только потом app-shell (1.4).

## Структура `frontend/`

```
frontend/
├── src/
│   ├── api/
│   │   ├── generated/      — сгенерированные TS-типы OpenAPI (schema.ts, openapi-typescript)
│   │   └── client.ts       — openapi-fetch клиент ($api, credentials:'include'), нормализация ошибок
│   ├── app/                — провайдеры (Query, Router, Theme), root
│   ├── components/         — shared UI (shadcn + композиты)
│   ├── features/           — вертикальные слайсы (auth, workspace, team, issue, ...)
│   ├── lib/                — утилиты (pagination, errors, theme tokens)
│   ├── pages/              — страницы/лейауты, привязка к роутам
│   └── main.tsx
├── pnpm-workspace.yaml     — pnpm-настройки (cooldown, allowBuilds, savePrefix)
├── vite.config.ts
└── package.json
```

---

# Фаза 0 — Bootstrap & tooling

Только скелет и инструменты, без доменного UI.

**Deliverables:**
1. Scaffold `frontend/` через `pnpm create vite --template react-ts`.
2. Tailwind + `shadcn init`; алиас `@/` в `tsconfig` + `vite.config`.
3. ESLint + Prettier (+ import-order), скрипты `lint`, `typecheck`, `build`, `dev`.
4. `vite.config.ts`: dev-proxy `/api`, `/mcp`, `/docs` → `http://localhost:8080`.
5. TanStack Query provider + Router provider в `main.tsx`.
6. Кодоген типов: скрипт `api:gen` = `openapi-typescript ../docs/swagger/swagger.json
   -o src/api/generated/schema.ts` (генерятся только TS-типы, без рантайма).
7. `api/client.ts`: `openapi-fetch` клиент с `credentials: 'include'` (cookie-сессия) +
   `openapi-react-query` (экспорт `$api`); middleware нормализует ошибки в форму `AppError`
   (`{ code, message, fields[] }`, см. `docs/tech/error-handling.md`). Отмена запросов —
   через AbortSignal React Query.
8. README-раздел: `make swag-init` → `pnpm run api:gen` → `pnpm dev`.

**DoD Фазы 0:**
- `pnpm build`, `pnpm lint`, `pnpm typecheck` — чисто.
- Dev-сервер поднимается, proxy на бэк работает.
- Сгенерированные типы компилятся, `$api` типобезопасен; smoke-запрос `GET /api/auth/me`
  без cookie отдаёт `401` (health-эндпоинта нет — этого достаточно как проверки живости
  proxy + роутера).

---

# Фаза 1 — Фундамент (cross-cutting, без домена)

Разбита на 4 под-фазы по зависимостям. Каждая — самодостаточный чанк с точкой остановки.
Ключевая идея нарезки: **сначала фиксируем дизайн-решения (1.1), потом строим на них**
логику и визуал — не бутстрапим auth/shell до того, как определились с внешним видом.

## 1.1 — Дизайн-фундамент (токены + тема)

Только дизайн-система, без домена. Точка остановки для ревью внешнего вида.

**Deliverables:**
- Палитра: подобрать и зафиксировать hex для `primary`/`accent` в light и dark (+контраст).
- CSS-переменные на `:root`/`.dark` (конвенция shadcn): `background, foreground, card,
  popover, primary, secondary, muted, accent, destructive, border, input, ring`.
- **Тема по умолчанию — dark** (Linear-like). `ThemeProvider`: класс на `<html>`, persist
  в localStorage, respect `prefers-color-scheme`, тумблер light/dark.
- Доменные токены (используются во всём UI):
  - **Status-группы (5)**: Backlog, Planned, In Progress, Completed, Canceled.
  - **Priority (5)**: Urgent, High, Medium, Low, No priority (иконки-«шкала» + цвет).
  - **Labels**: цвет приходит с бэка (`label.color` hex) → точка+текст с контрастом.
- Базовый набор shadcn: button, input, textarea, select, dialog, dropdown-menu, popover,
  command, sonner (toast), avatar, badge, tooltip, tabs, checkbox, scroll-area, skeleton.
- Временная **preview-страница** (kitchen sink) на отдельном роуте: пощупать все токены,
  варианты компонентов, бейджи статусов/приоритетов в обеих темах. Прячется/удаляется позже.

**DoD 1.1:**
- Тема по умолчанию — dark; тумблер light/dark работает и переживает reload.
- На preview-странице видны все дизайн-токены и варианты базовых компонентов в обеих темах.
- `pnpm build`, `pnpm lint`, `pnpm typecheck` — чисто.

## 1.2 — Роутинг + error/loading примитивы

Каркас навигации и cross-cutting UI-примитивы. Логика, слабо зависит от финального дизайна.

**Deliverables:**
- Роутер: публичные `/login`, `/register` (пока заглушки); защищённые `/:workspaceSlug/...`;
  `ProtectedRoute`, `<Suspense>`-фолбэки, error-boundary.
- Общие примитивы: рендер `AppError` (включая `fields[]` под инпутами), пустые состояния,
  skeleton-лоадеры. (Обёртку над cursor-пагинацией НЕ здесь — она в 2.4, где появляется
  первый потребитель.)

**DoD 1.2:**
- Роуты резолвятся; guard редиректит на заглушку `/login`; error-boundary ловит ошибки.
- `pnpm build`, `pnpm lint`, `pnpm typecheck` — чисто.

## 1.3 — Auth-bootstrap (session resume)

Провайдер current-user + current-workspace; 401-обработка → redirect `/login`. Зависит от 1.2.
Флоу старта:
- `GET /api/auth/me` → нет сессии (`401`) → `/login`.
- есть сессия → `GET /api/workspaces`; строим клиентский маппинг `slug ↔ id` (кешируем).
- дефолтный workspace: `me.settings.current_workspace_id`, если валиден и присутствует в
  списке; иначе — первый из списка (или экран выбора, если список пуст — по сути только
  после register такого не бывает).
- смена workspace в UI → `PATCH /api/auth/me { current_workspace_id }` (персист на сервере).
- неизвестный `slug` в URL → 404/редирект на дефолтный workspace.
- `LoadWorkspace` на бэке принимает только UUID `{workspaceID}` — резолв `slug → id`
  целиком клиентский (по списку из `/api/workspaces`).

**DoD 1.3:**
- Незалогиненный → редирект на `/login`.
- Залогиненный с валидной cookie при перезагрузке восстанавливается и попадает в свой
  workspace (URL по slug).

## 1.4 — App-shell

Sidebar (навигация по workspace/teams/projects) + topbar, responsive, `<Outlet/>`.
Зависит от 1.1 (дизайн) и 1.3 (контекст workspace).

**Deliverables:**
- Sidebar + topbar с рабочей навигацией; тумблер темы в topbar; `<Outlet/>` для страниц.

**DoD 1.4:**
- App-shell рендерится с рабочей навигацией.
- `pnpm build`, `pnpm lint`, `pnpm typecheck` — чисто.

---

# Фаза 2 — Вертикальные слайсы (по зависимостям)

Каждый слайс = data-хуки (через `$api`, openapi-react-query) + компоненты + страница +
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

1. Data-хуки через `$api` (openapi-react-query) — query/mutation, корректная инвалидция кэша.
2. Экран(ы) с состояниями: loading (skeleton) / empty / error (`AppError` + `fields[]`).
3. Optimistic update там, где это улучшает UX (статус/приоритет/assignees).
4. Роль-гейтинг UI, где применимо (owner/admin/member).
5. Роуты подключены; навигация из shell работает.
6. `pnpm build`, `pnpm lint`, `pnpm typecheck` — чисто; фича работает против запущенного бэка
   (happy-path + ключевые ошибки 400/401/403/404/409).

---

## Правило нарезки для AI

- Один слайс сущности (хуки + компоненты + страница + состояния) — обычно правильный
  размер чанка: крупный для скорости, влезает в контекст.
- Крупные сущности дробить по операциям (см. `issues`: list → detail → assignees/labels).
- В промпт слайса включать: соответствующий раздел `docs/tech/api.md`, форму DTO из
  сгенерированных типов (`schema.ts`), дизайн-токены из Фазы 1.1 и DoD слайса.
