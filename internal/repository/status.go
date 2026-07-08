package repository

import (
	"context"

	"github.com/jmoiron/sqlx"

	"github.com/timohahaa/tracker/internal/errs"
	"github.com/timohahaa/tracker/internal/model"
)

// StatusRepo — доступ к tracker.status.
type StatusRepo struct {
	db Ext
}

// Create вставляет статус (id/created_at/updated_at генерит БД) и возвращает
// созданную строку. Нарушение уникальности (team_id, group, position) →
// errs.AlreadyExists (через errs.DB).
func (r *StatusRepo) Create(ctx context.Context, s model.Status) (model.Status, error) {
	const q = `
		INSERT INTO tracker.status (
			team_id
			, name
			, status_group
			, position
		)
		VALUES ($1, $2, $3, $4)
		RETURNING
			id
			, team_id
			, name
			, status_group
			, position
			, created_at
			, updated_at
			, deleted_at`

	var out model.Status
	if err := sqlx.GetContext(ctx, r.db, &out, q, s.TeamID, s.Name, s.Group, s.Position); err != nil {
		return model.Status{}, errs.DB(err)
	}
	return out, nil
}

// Get возвращает активный статус по (team_id, id). Нет строки → errs.NotFound.
func (r *StatusRepo) Get(ctx context.Context, teamID, statusID string) (model.Status, error) {
	const q = `
		SELECT
			id
			, team_id
			, name
			, status_group
			, position
			, created_at
			, updated_at
			, deleted_at
		FROM tracker.status
		WHERE id = $1 AND team_id = $2 AND deleted_at IS NULL`

	var out model.Status
	if err := sqlx.GetContext(ctx, r.db, &out, q, statusID, teamID); err != nil {
		return model.Status{}, errs.DB(err)
	}
	return out, nil
}

// List возвращает активные статусы команды, упорядоченные по группе и позиции.
func (r *StatusRepo) List(ctx context.Context, teamID string) ([]model.Status, error) {
	const q = `
		SELECT
			id
			, team_id
			, name
			, status_group
			, position
			, created_at
			, updated_at
			, deleted_at
		FROM tracker.status
		WHERE team_id = $1 AND deleted_at IS NULL
		ORDER BY status_group, position, id`

	var out []model.Status
	if err := sqlx.SelectContext(ctx, r.db, &out, q, teamID); err != nil {
		return nil, errs.DB(err)
	}
	return out, nil
}

// Update обновляет name/position статуса и возвращает обновлённую строку.
// group не меняется. Нарушение уникальности (team_id, group, position) →
// errs.AlreadyExists (через errs.DB).
func (r *StatusRepo) Update(ctx context.Context, s model.Status) (model.Status, error) {
	const q = `
		UPDATE tracker.status
		SET
			name = $2
			, position = $3
			, updated_at = now()
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING
			id
			, team_id
			, name
			, status_group
			, position
			, created_at
			, updated_at
			, deleted_at`

	var out model.Status
	if err := sqlx.GetContext(ctx, r.db, &out, q, s.ID, s.Name, s.Position); err != nil {
		return model.Status{}, errs.DB(err)
	}
	return out, nil
}

// Delete мягко удаляет статус (deleted_at = now()). Повторное удаление не
// считается ошибкой.
func (r *StatusRepo) Delete(ctx context.Context, statusID string) error {
	const q = `
		UPDATE tracker.status
		SET deleted_at = now()
		WHERE id = $1 AND deleted_at IS NULL`

	if _, err := r.db.ExecContext(ctx, q, statusID); err != nil {
		return errs.DB(err)
	}
	return nil
}

// CountByGroup возвращает число активных статусов команды в указанной группе.
// Используется для правила «≥1 статус на группу» перед soft delete.
func (r *StatusRepo) CountByGroup(ctx context.Context, teamID string, group model.StatusGroup) (int, error) {
	const q = `
		SELECT COUNT(*)
		FROM tracker.status
		WHERE team_id = $1 AND status_group = $2 AND deleted_at IS NULL`

	var n int
	if err := sqlx.GetContext(ctx, r.db, &n, q, teamID, group); err != nil {
		return 0, errs.DB(err)
	}
	return n, nil
}
