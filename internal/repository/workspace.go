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

// GetByID возвращает активный workspace по id. Нет строки → errs.NotFound.
func (r *WorkspaceRepo) GetByID(ctx context.Context, id string) (model.Workspace, error) {
	const q = `
		SELECT
			id
			, name
			, slug
			, created_at
			, updated_at
			, deleted_at
		FROM org.workspace
		WHERE id = $1 AND deleted_at IS NULL`

	var out model.Workspace
	if err := sqlx.GetContext(ctx, r.db, &out, q, id); err != nil {
		return model.Workspace{}, errs.DB(err)
	}
	return out, nil
}

// Update обновляет name/slug workspace и возвращает обновлённую строку.
// Нарушение уникальности slug → errs.AlreadyExists (через errs.DB).
func (r *WorkspaceRepo) Update(ctx context.Context, ws model.Workspace) (model.Workspace, error) {
	const q = `
		UPDATE org.workspace
		SET
			name = $2
			, slug = $3
			, updated_at = now()
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING
			id
			, name
			, slug
			, created_at
			, updated_at
			, deleted_at`

	var out model.Workspace
	if err := sqlx.GetContext(ctx, r.db, &out, q, ws.ID, ws.Name, ws.Slug); err != nil {
		return model.Workspace{}, errs.DB(err)
	}
	return out, nil
}

// ListMembers возвращает активных участников workspace вместе с email/display_name
// из core.user. Сортировка стабильна (created_at, user_id).
func (r *WorkspaceRepo) ListMembers(ctx context.Context, workspaceID string) ([]model.WorkspaceMemberDetails, error) {
	const q = `
		SELECT
			m.workspace_id
			, m.user_id
			, m.role
			, m.username
			, m.title
			, m.created_at
			, m.updated_at
			, m.deleted_at
			, u.email
			, u.display_name
		FROM org.workspace_member m
		JOIN core.user u ON u.id = m.user_id
		WHERE m.workspace_id = $1 AND m.deleted_at IS NULL
		ORDER BY m.created_at, m.user_id`

	var out []model.WorkspaceMemberDetails
	if err := sqlx.SelectContext(ctx, r.db, &out, q, workspaceID); err != nil {
		return nil, errs.DB(err)
	}
	return out, nil
}

// GetMember возвращает активного участника workspace по (workspace_id, user_id).
// Нет строки → errs.NotFound (через errs.DB).
func (r *WorkspaceRepo) GetMember(ctx context.Context, workspaceID, userID string) (model.WorkspaceMember, error) {
	const q = `
		SELECT
			workspace_id
			, user_id
			, role
			, username
			, title
			, created_at
			, updated_at
			, deleted_at
		FROM org.workspace_member
		WHERE workspace_id = $1 AND user_id = $2 AND deleted_at IS NULL`

	var out model.WorkspaceMember
	if err := sqlx.GetContext(ctx, r.db, &out, q, workspaceID, userID); err != nil {
		return model.WorkspaceMember{}, errs.DB(err)
	}
	return out, nil
}

// UpdateMember обновляет role/username/title участника и возвращает обновлённую
// строку. Нарушение уникальности username/owner → errs.AlreadyExists (через errs.DB).
func (r *WorkspaceRepo) UpdateMember(ctx context.Context, m model.WorkspaceMember) (model.WorkspaceMember, error) {
	const q = `
		UPDATE org.workspace_member
		SET
			role = $3
			, username = $4
			, title = $5
			, updated_at = now()
		WHERE workspace_id = $1 AND user_id = $2 AND deleted_at IS NULL
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

// DeleteMember мягко удаляет участника (deleted_at = now()). Повторное удаление
// (строка уже удалена/отсутствует) не считается ошибкой.
func (r *WorkspaceRepo) DeleteMember(ctx context.Context, workspaceID, userID string) error {
	const q = `
		UPDATE org.workspace_member
		SET deleted_at = now()
		WHERE workspace_id = $1 AND user_id = $2 AND deleted_at IS NULL`

	if _, err := r.db.ExecContext(ctx, q, workspaceID, userID); err != nil {
		return errs.DB(err)
	}
	return nil
}

// CountOwners возвращает число активных владельцев workspace (для защиты
// последнего owner).
func (r *WorkspaceRepo) CountOwners(ctx context.Context, workspaceID string) (int, error) {
	const q = `
		SELECT COUNT(*)
		FROM org.workspace_member
		WHERE workspace_id = $1 AND role = 'owner' AND deleted_at IS NULL`

	var n int
	if err := sqlx.GetContext(ctx, r.db, &n, q, workspaceID); err != nil {
		return 0, errs.DB(err)
	}
	return n, nil
}
