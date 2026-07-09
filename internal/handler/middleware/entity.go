package middleware

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/timohahaa/tracker/internal/errs"
	"github.com/timohahaa/tracker/internal/handler/httpx"
)

// LoadWorkspace загружает workspace из path-параметра {workspaceID}, проверяет,
// что текущий пользователь — активный участник, и кладёт workspace и его
// membership в контекст. Нет workspace или пользователь не участник → 404
// (существование workspace не раскрываем не-участникам).
func (m *Middleware) LoadWorkspace(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		workspaceID := chi.URLParam(r, "workspaceID")
		if _, err := uuid.Parse(workspaceID); err != nil {
			httpx.RespondError(w, errs.Validation("invalid path parameter", errs.FieldError{
				Field:   "workspaceID",
				Reason:  "uuid",
				Message: "must be a valid UUID",
			}))
			return
		}
		user := UserFromContext(r.Context())

		ws, member, err := m.services.Workspace.Load(r.Context(), workspaceID, user.ID)
		if err != nil {
			httpx.RespondError(w, err)
			return
		}

		ctx := WithWorkspace(r.Context(), ws)
		ctx = WithMember(ctx, member)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
