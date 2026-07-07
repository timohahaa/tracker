package middleware

import "net/http"

// LoadWorkspace загружает workspace из path-параметра {workspaceID}
// и кладёт его в контекст. Заглушка.
func (m *Middleware) LoadWorkspace(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
	})
}
