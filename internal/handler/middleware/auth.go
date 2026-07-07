package middleware

import (
	"net/http"

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

// SessionAuth аутентифицирует по session cookie (web). Заглушка.
func (m *Middleware) SessionAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
	})
}

// TokenAuth аутентифицирует по Authorization: Bearer <token> (API/MCP). Заглушка.
func (m *Middleware) TokenAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
	})
}
