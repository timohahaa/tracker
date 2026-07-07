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
