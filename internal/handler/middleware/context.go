package middleware

import (
	"context"

	"github.com/timohahaa/tracker/internal/model"
)

type ctxKey string

const (
	ctxKeyUser      ctxKey = "user"
	ctxKeyWorkspace ctxKey = "workspace"
	ctxKeyMember    ctxKey = "member"
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

// WithMember кладёт membership текущего пользователя в контекст (для авторизации
// по роли).
func WithMember(ctx context.Context, m model.WorkspaceMember) context.Context {
	return context.WithValue(ctx, ctxKeyMember, m)
}

// MemberFromContext достаёт membership текущего пользователя из контекста.
func MemberFromContext(ctx context.Context) model.WorkspaceMember {
	m, _ := ctx.Value(ctxKeyMember).(model.WorkspaceMember)
	return m
}
