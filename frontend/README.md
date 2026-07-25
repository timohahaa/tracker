# Frontend

SPA на Vite + React 18 + TypeScript поверх Go REST API. UI — shadcn/ui + Tailwind.
Данные — TanStack Query через типобезопасные хуки, сгенерированные из OpenAPI-спеки бэка.

План разработки: [`../FRONTEND_PLAN.md`](../FRONTEND_PLAN.md).

## Требования

- **Node** — версия из [`.nvmrc`](./.nvmrc) (`nvm use`).
- **pnpm 11+** — пакетный менеджер проекта (задан в `package.json` → `packageManager`).

## Установка

```bash
pnpm install --frozen-lockfile
```

`--frozen-lockfile` ставит строго по `pnpm-lock.yaml` и падает при расхождении с
`package.json` (аналог `npm ci`).

## Разработка

```bash
# 1. Поднять бэкенд на :8080 (из корня репозитория), см. корневой Makefile.
# 2. Dev-сервер фронта (Vite проксирует /api, /mcp, /docs на :8080):
pnpm dev
```

## Кодоген API-типов

Типы и хуки строятся из OpenAPI-спеки бэка. Спека генерируется на стороне бэка
(`make swag-init` → `docs/swagger/swagger.json`), после чего:

```bash
pnpm run api:gen   # openapi-typescript -> src/api/generated/schema.ts (только типы)
```

Рантайм-хуки собираются из этих типов через `openapi-react-query`. Использование:

```ts
import { $api } from '@/api/client'

const { data, error } = $api.useQuery('get', '/api/workspaces')
const mutation = $api.useMutation('post', '/api/auth/login')
```

Ошибки нормализуются middleware в `src/api/client.ts` в `AppError`
(`{ code, message, fields[], status }`) — в рантайме сужай через `error instanceof AppError`.

## Скрипты

| Скрипт | Назначение |
|--------|-----------|
| `pnpm dev` | Vite dev-сервер |
| `pnpm build` | `tsc -b` + production-сборка в `dist/` |
| `pnpm preview` | Локальный предпросмотр прод-сборки |
| `pnpm typecheck` | Проверка типов без эмита |
| `pnpm lint` | ESLint |
| `pnpm format` | Prettier |
| `pnpm run api:gen` | Генерация TS-типов из OpenAPI |

## Политика зависимостей

Настройки в [`pnpm-workspace.yaml`](./pnpm-workspace.yaml) / [`.npmrc`](./.npmrc):

- Прямые зависимости запинены **точно** (без `^`/`~`); реальный пин всего дерева — в
  `pnpm-lock.yaml` (коммитится).
- **Cooldown**: `minimumReleaseAge` — не устанавливать версии моложе 14 дней (защита от
  supply-chain атак на свежие релизы).
- **Build-скрипты зависимостей** заблокированы по умолчанию; разрешённые перечислены в
  `allowBuilds`. Новые пакеты со скриптами pnpm допишет туда с плейсхолдером — значение
  (`true`/`false`) выставляется осознанно.
- После добавления пакетов прогоняй `pnpm audit`.
