# Error Handling

Двухуровневая система: доменные ошибки в service layer + раздельный маппинг в каждом транспорте.

## Domain Errors (`internal/errs/`)

```go
type Code string

const (
    CodeNotFound      Code = "not_found"
    CodeForbidden     Code = "forbidden"
    CodeUnauthorized  Code = "unauthorized"
    CodeValidation    Code = "validation"
    CodeAlreadyExists Code = "already_exists"
    CodeInternal      Code = "internal"
)

type AppError struct {
    Code    Code
    Message string
    Detail  string
    Fields  []FieldError
}

type FieldError struct {
    Field   string // "title", "team_id"
    Reason  string // "required", "invalid", "exists"
    Message string
}
```

Service layer возвращает `*AppError`. Не знает про HTTP, MCP, или любой другой транспорт.

## Конструкторы

```go
func NotFound(msg string) *AppError
func Forbidden(msg string) *AppError
func Validation(msg string, fields ...FieldError) *AppError
func AlreadyExists(msg string) *AppError
```

## DB Error Mapping (`internal/errs/db.go`)

```go
func DB(err error) error {
    if errors.Is(err, pgx.ErrNoRows) {
        return NotFound("resource not found")
    }
    if pgErr, ok := err.(*pgconn.PgError); ok && pgErr.Code == "23505" {
        return AlreadyExists("resource already exists")
    }
    return err
}
```

Repository вызывает `errs.DB(err)` при ошибках БД — превращает pg-ошибки в доменные.

## HTTP Error Response

| HTTP Status | Error Code | When |
|-------------|-----------|------|
| 400 | `validation` | Невалидный input |
| 401 | `unauthorized` | Нет сессии/токена |
| 403 | `forbidden` | Нет прав |
| 404 | `not_found` | Ресурс не найден |
| 409 | `already_exists` | Дубликат (slug, email) |
| 500 | `internal` | Внутренняя ошибка |

JSON-формат ответа:

```json
{
  "error": {
    "code": "validation",
    "message": "request parameters didn't validate",
    "fields": [
      { "field": "title", "reason": "required", "message": "title is required" }
    ]
  }
}
```

## Transport Mapping

### REST API (`internal/handler/respond.go`)

```go
func respondError(w http.ResponseWriter, err error) {
    var appErr *errs.AppError
    if !errors.As(err, &appErr) {
        writeJSON(w, 500, errorResponse{Code: "internal", Message: "internal error"})
        return
    }
    writeJSON(w, mapHTTPStatus(appErr.Code), errorResponse{
        Code:    string(appErr.Code),
        Message: appErr.Message,
        Detail:  appErr.Detail,
        Fields:  appErr.Fields,
    })
}

func mapHTTPStatus(code errs.Code) int {
    switch code {
    case errs.CodeNotFound:      return 404
    case errs.CodeValidation:    return 400
    case errs.CodeAlreadyExists: return 409
    case errs.CodeForbidden:     return 403
    case errs.CodeUnauthorized:  return 401
    default:                     return 500
    }
}
```

### MCP (`internal/mcp/errors.go`)

```go
func respondMCPError(err error) *mcp.CallToolResult {
    var appErr *errs.AppError
    if !errors.As(err, &appErr) {
        appErr = &errs.AppError{Code: errs.CodeInternal, Message: "internal error"}
    }
    return &mcp.CallToolResult{
        IsError: true,
        Content: []mcp.Content{{Type: "text", Text: formatMCPError(appErr)}},
    }
}
```

## Flow

```
Repository: SQL error → errs.DB(err) → *AppError
Service:    business rule violation → errs.Forbidden("...") → *AppError
Handler:    receives *AppError → respondError(w, err) → HTTP JSON
MCP:        receives *AppError → respondMCPError(err) → MCP error content
```
