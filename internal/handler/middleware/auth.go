package middleware

import (
	"net/http"
	"strings"

	"github.com/timohahaa/tracker/internal/handler/httpx"
	"github.com/timohahaa/tracker/internal/service"
)

// Middleware — набор middleware, зависящих от service layer.
type Middleware struct {
	services *service.Services
}

// New создаёт набор middleware.
func New(services *service.Services) *Middleware {
	return &Middleware{services: services}
}

// SessionAuth аутентифицирует по session cookie (web): читает session_id,
// резолвит сессию в пользователя (проверка срока + продление) и кладёт его в
// контекст. Нет/протухла → 401.
func (m *Middleware) SessionAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var sessionID string
		if c, err := r.Cookie(m.services.Auth.SessionCookieName()); err == nil {
			sessionID = c.Value
		}

		user, err := m.services.Auth.AuthBySession(r.Context(), sessionID)
		if err != nil {
			httpx.RespondError(w, err)
			return
		}

		next.ServeHTTP(w, r.WithContext(WithUser(r.Context(), user)))
	})
}

// TokenAuth аутентифицирует по Authorization: Bearer <token> (API/MCP): резолвит
// токен в пользователя, его workspace (скоуп токена) и membership, обновляет
// last_used_at. Кладёт всё в контекст теми же хелперами, что и web-цепочка
// (SessionAuth + LoadWorkspace), чтобы downstream (MCP tools) читали актора
// одинаково. Нет/невалиден → 401.
func (m *Middleware) TokenAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)

		user, ws, member, err := m.services.Auth.AuthByTokenWithWorkspace(r.Context(), token)
		if err != nil {
			httpx.RespondError(w, err)
			return
		}

		ctx := WithUser(r.Context(), user)
		ctx = WithWorkspace(ctx, ws)
		ctx = WithMember(ctx, member)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// bearerToken извлекает токен из заголовка Authorization: Bearer <token>.
func bearerToken(r *http.Request) string {
	const prefix = "Bearer "
	h := r.Header.Get("Authorization")
	if len(h) < len(prefix) || !strings.EqualFold(h[:len(prefix)], prefix) {
		return ""
	}
	return strings.TrimSpace(h[len(prefix):])
}
