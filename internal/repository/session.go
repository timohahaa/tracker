package repository

import (
	"context"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/timohahaa/tracker/internal/errs"
	"github.com/timohahaa/tracker/internal/model"
)

// SessionRepo — доступ к core.session.
type SessionRepo struct {
	db Ext
}

// Create заводит сессию для пользователя (id генерит БД через uuidv7()) и
// возвращает созданную строку с id и created_at.
func (r *SessionRepo) Create(ctx context.Context, userID string, expiresAt time.Time) (model.Session, error) {
	const q = `
		INSERT INTO core.session (
			user_id
			, expires_at
		)
		VALUES ($1, $2)
		RETURNING
			id
			, user_id
			, expires_at
			, created_at`

	var out model.Session
	if err := sqlx.GetContext(ctx, r.db, &out, q, userID, expiresAt); err != nil {
		return model.Session{}, errs.DB(err)
	}
	return out, nil
}

// GetByID возвращает сессию по id (включая expires_at для проверки в сервисе).
// Нет строки → errs.NotFound (через errs.DB).
func (r *SessionRepo) GetByID(ctx context.Context, id string) (model.Session, error) {
	const q = `
		SELECT
			id
			, user_id
			, expires_at
			, created_at
		FROM core.session
		WHERE id = $1`

	var out model.Session
	if err := sqlx.GetContext(ctx, r.db, &out, q, id); err != nil {
		return model.Session{}, errs.DB(err)
	}
	return out, nil
}

// Delete удаляет сессию по id (logout). Отсутствие строки не считается ошибкой.
func (r *SessionRepo) Delete(ctx context.Context, id string) error {
	const q = `DELETE FROM core.session WHERE id = $1`
	if _, err := r.db.ExecContext(ctx, q, id); err != nil {
		return errs.DB(err)
	}
	return nil
}

// Extend сдвигает expires_at сессии (sliding window). Отсутствие строки — не ошибка.
func (r *SessionRepo) Extend(ctx context.Context, id string, expiresAt time.Time) error {
	const q = `UPDATE core.session SET expires_at = $2 WHERE id = $1`
	if _, err := r.db.ExecContext(ctx, q, id, expiresAt); err != nil {
		return errs.DB(err)
	}
	return nil
}
