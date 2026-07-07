package repository

import (
	"context"

	"github.com/jmoiron/sqlx"

	"github.com/timohahaa/tracker/internal/errs"
	"github.com/timohahaa/tracker/internal/model"
)

// WorkspaceRepo — доступ к org.workspace и org.workspace_member.
type WorkspaceRepo struct {
	db Ext
}

// Create вставляет workspace (id/created_at/updated_at генерит БД) и возвращает
// созданную строку. Нарушение уникальности slug → errs.AlreadyExists (через errs.DB).
func (r *WorkspaceRepo) Create(ctx context.Context, ws model.Workspace) (model.Workspace, error) {
	const q = `
		INSERT INTO org.workspace (
			name
			, slug
		)
		VALUES ($1, $2)
		RETURNING
			id
			, name
			, slug
			, created_at
			, updated_at
			, deleted_at`

	var out model.Workspace
	if err := sqlx.GetContext(ctx, r.db, &out, q, ws.Name, ws.Slug); err != nil {
		return model.Workspace{}, errs.DB(err)
	}
	return out, nil
}

// SlugTaken сообщает, занят ли slug активным workspace (регистронезависимо).
func (r *WorkspaceRepo) SlugTaken(ctx context.Context, slug string) (bool, error) {
	const q = `
		SELECT EXISTS (
			SELECT 1 FROM org.workspace
			WHERE LOWER(slug) = LOWER($1) AND deleted_at IS NULL
		)`

	var exists bool
	if err := sqlx.GetContext(ctx, r.db, &exists, q, slug); err != nil {
		return false, errs.DB(err)
	}
	return exists, nil
}

// CreateMember добавляет участника workspace и возвращает созданную строку.
// Нарушение уникальности (username/owner) → errs.AlreadyExists (через errs.DB).
func (r *WorkspaceRepo) CreateMember(ctx context.Context, m model.WorkspaceMember) (model.WorkspaceMember, error) {
	const q = `
		INSERT INTO org.workspace_member (
			workspace_id
			, user_id
			, role
			, username
			, title
		)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING
			workspace_id
			, user_id
			, role
			, username
			, title
			, created_at
			, updated_at
			, deleted_at`

	var out model.WorkspaceMember
	if err := sqlx.GetContext(ctx, r.db, &out, q, m.WorkspaceID, m.UserID, m.Role, m.Username, m.Title); err != nil {
		return model.WorkspaceMember{}, errs.DB(err)
	}
	return out, nil
}
