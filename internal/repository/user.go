package repository

import (
	"context"

	"github.com/jmoiron/sqlx"

	"github.com/timohahaa/tracker/internal/errs"
	"github.com/timohahaa/tracker/internal/model"
)

// UserRepo — доступ к core.user.
type UserRepo struct {
	db Ext
}

// Create вставляет пользователя (id/created_at/updated_at генерит БД) и возвращает
// созданную строку. Нарушение уникальности email → errs.AlreadyExists (через errs.DB).
func (r *UserRepo) Create(ctx context.Context, u model.User) (model.User, error) {
	const q = `
		INSERT INTO core.user (
			email
			, password_hash
			, display_name
		)
		VALUES ($1, $2, $3)
		RETURNING
			id
			, email
			, password_hash
			, display_name
			, settings
			, created_at
			, updated_at
			, deleted_at`

	var out model.User
	if err := sqlx.GetContext(ctx, r.db, &out, q, u.Email, u.PasswordHash, u.DisplayName); err != nil {
		return model.User{}, errs.DB(err)
	}
	return out, nil
}

// GetByEmail ищет активного пользователя по email (регистронезависимо).
// Нет строки → errs.NotFound (через errs.DB).
func (r *UserRepo) GetByEmail(ctx context.Context, email string) (model.User, error) {
	const q = `
		SELECT
			id
			, email
			, password_hash
			, display_name
			, settings
			, created_at
			, updated_at
			, deleted_at
		FROM core.user
		WHERE LOWER(email) = LOWER($1) AND deleted_at IS NULL`

	var out model.User
	if err := sqlx.GetContext(ctx, r.db, &out, q, email); err != nil {
		return model.User{}, errs.DB(err)
	}
	return out, nil
}

// GetByID возвращает активного пользователя по id. Нет строки → errs.NotFound.
func (r *UserRepo) GetByID(ctx context.Context, id string) (model.User, error) {
	const q = `
		SELECT
			id
			, email
			, password_hash
			, display_name
			, settings
			, created_at
			, updated_at
			, deleted_at
		FROM core.user
		WHERE id = $1 AND deleted_at IS NULL`

	var out model.User
	if err := sqlx.GetContext(ctx, r.db, &out, q, id); err != nil {
		return model.User{}, errs.DB(err)
	}
	return out, nil
}

// UpdateSettings перезаписывает jsonb-настройки пользователя. Нет строки —
// не ошибка (idempotent), т.к. вызывается для аутентифицированного пользователя.
func (r *UserRepo) UpdateSettings(ctx context.Context, userID string, settings model.UserSettings) error {
	const q = `
		UPDATE core.user
		SET
			settings = $2
			, updated_at = now()
		WHERE id = $1 AND deleted_at IS NULL`

	if _, err := r.db.ExecContext(ctx, q, userID, settings); err != nil {
		return errs.DB(err)
	}
	return nil
}
