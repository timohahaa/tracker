package middleware

import (
	"context"

	"github.com/timohahaa/tracker/internal/model"
)

type ctxKey string

const (
	ctxKeyUser      ctxKey = "user"
	ctxKeyWorkspace ctxKey = "workspace"
)

// WithUser кладёт пользователя в контекст.
func WithUser(ctx context.Context, u model.User) context.Context {
	return context.WithValue(ctx, ctxKeyUser, u)
}

// UserFromContext достаёт текущего пользователя из контекста.
func UserFromContext(ctx context.Context) model.User {
	u, _ := ctx.Value(ctxKeyUser).(model.User)
	return u
}

// WithWorkspace кладёт workspace в контекст.
func WithWorkspace(ctx context.Context, w model.Workspace) context.Context {
	return context.WithValue(ctx, ctxKeyWorkspace, w)
}

// WorkspaceFromContext достаёт текущий workspace из контекста.
func WorkspaceFromContext(ctx context.Context) model.Workspace {
	w, _ := ctx.Value(ctxKeyWorkspace).(model.Workspace)
	return w
}
