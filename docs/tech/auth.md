# Authentication & Authorization

Гибридная модель: server-side sessions для браузера, API tokens для автоматизации (MCP, API).

## Web (Browser) — Sessions

```
Browser → POST /api/auth/login {email, password}
       ← Set-Cookie: session_id=xxx (httpOnly, secure, sameSite)

Browser → GET /api/issues (Cookie: session_id=xxx)
       → Middleware: lookup session in PostgreSQL → set user in context
```

- Сессии хранятся в PostgreSQL (`sessions` таблица).
- Cookie: httpOnly, secure, SameSite=Lax.
- Сессия имеет expires_at; при каждом запросе можно продлевать (sliding window) — TBD.
- Logout = удаление записи из sessions.

## API / MCP (AI Agents) — Tokens

```
Agent → POST /mcp (Header: Authorization: Bearer <api_token>)
     → Middleware: lookup token hash in PostgreSQL → set user in context
```

- API tokens — длинные случайные строки, привязаны к user + workspace.
- Хранится только hash токена (`token_hash`); raw-значение показывается один раз при создании.
- Создаются в UI (Settings → API Tokens).
- `last_used_at` обновляется при использовании.

## Middleware (`internal/handler/middleware/`)

Два middleware для auth:

**SessionAuth** — для `/api/*` (web):
1. Читает `session_id` из cookie.
2. Ищет сессию в БД, проверяет expires_at.
3. Кладёт user + workspace в context.
4. Если нет/expired → 401.

**TokenAuth** — для `/mcp`, опциональный fallback для `/api/*`:
1. Читает `Authorization: Bearer <token>` из header.
2. Хэширует, ищет в `api_tokens`.
3. Кладёт user + workspace в context.
4. Обновляет `last_used_at`.

`/api/*` пробует сначала session, потом token (позволяет использовать API и из браузера, и из скриптов).

## Authorization

Проверка прав — в service layer (не в middleware):
- **Workspace membership**: user должен быть member workspace.
- **Team membership**: для операций с issues/statuses — user должен быть member команды.
- **Workspace roles**: owner/admin/member — для управления workspace (invite, settings).

```go
// service/issue.go
func (s *IssueService) Create(ctx context.Context, issue model.Issue) (model.Issue, error) {
    user := auth.UserFromContext(ctx)
    if !s.teamRepo.IsMember(ctx, issue.TeamID, user.ID) {
        return model.Issue{}, errs.Forbidden("not a team member")
    }
    // ...
}
```

## Context Helpers (`internal/handler/middleware/context.go`)

```go
func UserFromContext(ctx context.Context) model.User
func WorkspaceFromContext(ctx context.Context) model.Workspace
```
