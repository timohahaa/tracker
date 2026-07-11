package repository

import (
	"context"

	"github.com/jmoiron/sqlx"

	"github.com/timohahaa/tracker/internal/errs"
	"github.com/timohahaa/tracker/internal/model"
)

// TokenRepo — доступ к core.api_token.
type TokenRepo struct {
	db Ext
}

// tokenSelectColumns — общий список колонок токена.
const tokenSelectColumns = `
	id
	, user_id
	, workspace_id
	, name
	, last_used_at
	, created_at
	, deleted_at`

// Create вставляет API-токен (id генерит БД через uuidv7() — он же Bearer-токен)
// и возвращает созданную строку.
func (r *TokenRepo) Create(ctx context.Context, t model.APIToken) (model.APIToken, error) {
	const q = `
		INSERT INTO core.api_token (
			user_id
			, workspace_id
			, name
		)
		VALUES ($1, $2, $3)
		RETURNING` + tokenSelectColumns

	var out model.APIToken
	if err := sqlx.GetContext(ctx, r.db, &out, q, t.UserID, t.WorkspaceID, t.Name); err != nil {
		return model.APIToken{}, errs.DB(err)
	}
	return out, nil
}

// ListByUser возвращает активные токены пользователя в workspace, сортировка
// стабильна (created_at DESC, id DESC).
func (r *TokenRepo) ListByUser(ctx context.Context, workspaceID, userID string) ([]model.APIToken, error) {
	const q = `
		SELECT` + tokenSelectColumns + `
		FROM core.api_token
		WHERE user_id = $1 AND workspace_id = $2 AND deleted_at IS NULL
		ORDER BY created_at DESC, id DESC`

	var out []model.APIToken
	if err := sqlx.SelectContext(ctx, r.db, &out, q, userID, workspaceID); err != nil {
		return nil, errs.DB(err)
	}
	return out, nil
}

// Delete мягко отзывает токен (deleted_at = now()). Повторный отзыв (строка уже
// удалена/отсутствует) не считается ошибкой.
func (r *TokenRepo) Delete(ctx context.Context, id string) error {
	const q = `UPDATE core.api_token SET deleted_at = now() WHERE id = $1 AND deleted_at IS NULL`
	if _, err := r.db.ExecContext(ctx, q, id); err != nil {
		return errs.DB(err)
	}
	return nil
}

// GetByID возвращает активный API-токен по id (он же Bearer-токен).
// Нет строки → errs.NotFound (через errs.DB).
func (r *TokenRepo) GetByID(ctx context.Context, id string) (model.APIToken, error) {
	const q = `
		SELECT
			id
			, user_id
			, workspace_id
			, name
			, last_used_at
			, created_at
			, deleted_at
		FROM core.api_token
		WHERE id = $1 AND deleted_at IS NULL`

	var out model.APIToken
	if err := sqlx.GetContext(ctx, r.db, &out, q, id); err != nil {
		return model.APIToken{}, errs.DB(err)
	}
	return out, nil
}

// UpdateLastUsed проставляет last_used_at = now() при использовании токена.
func (r *TokenRepo) UpdateLastUsed(ctx context.Context, id string) error {
	const q = `UPDATE core.api_token SET last_used_at = now() WHERE id = $1`
	if _, err := r.db.ExecContext(ctx, q, id); err != nil {
		return errs.DB(err)
	}
	return nil
}
