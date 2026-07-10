package repository

import (
	"context"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"

	"github.com/timohahaa/tracker/internal/errs"
	"github.com/timohahaa/tracker/internal/model"
)

// LabelRepo — доступ к tracker.label и tracker.issue_label.
type LabelRepo struct {
	db Ext
}

// labelSelectColumns — общий список колонок метки.
const labelSelectColumns = `
	id
	, workspace_id
	, team_id
	, name
	, color
	, created_at
	, updated_at
	, deleted_at`

// Create вставляет метку (id/created_at/updated_at генерит БД) и возвращает
// созданную строку. Нарушение партиал-уникальности (ws-level или team-level имя)
// → errs.AlreadyExists (через errs.DB).
func (r *LabelRepo) Create(ctx context.Context, l model.Label) (model.Label, error) {
	const q = `
		INSERT INTO tracker.label (
			workspace_id
			, team_id
			, name
			, color
		)
		VALUES ($1, $2, $3, $4)
		RETURNING` + labelSelectColumns

	var out model.Label
	if err := sqlx.GetContext(ctx, r.db, &out, q,
		l.WorkspaceID, l.TeamID, l.Name, l.Color,
	); err != nil {
		return model.Label{}, errs.DB(err)
	}
	return out, nil
}

// GetByID возвращает активную метку по (workspace_id, id). Нет строки →
// errs.NotFound (через errs.DB).
func (r *LabelRepo) GetByID(ctx context.Context, workspaceID, labelID string) (model.Label, error) {
	const q = `
		SELECT` + labelSelectColumns + `
		FROM tracker.label
		WHERE id = $1 AND workspace_id = $2 AND deleted_at IS NULL`

	var out model.Label
	if err := sqlx.GetContext(ctx, r.db, &out, q, labelID, workspaceID); err != nil {
		return model.Label{}, errs.DB(err)
	}
	return out, nil
}

// List возвращает все активные метки workspace (и workspace-level, и team-level)
// плоским списком. Сортировка стабильна (created_at, id).
func (r *LabelRepo) List(ctx context.Context, workspaceID string) ([]model.Label, error) {
	const q = `
		SELECT` + labelSelectColumns + `
		FROM tracker.label
		WHERE workspace_id = $1 AND deleted_at IS NULL
		ORDER BY created_at, id`

	var out []model.Label
	if err := sqlx.SelectContext(ctx, r.db, &out, q, workspaceID); err != nil {
		return nil, errs.DB(err)
	}
	return out, nil
}

// Update обновляет name/color метки и возвращает обновлённую строку. Нарушение
// партиал-уникальности имени → errs.AlreadyExists (через errs.DB).
func (r *LabelRepo) Update(ctx context.Context, l model.Label) (model.Label, error) {
	const q = `
		UPDATE tracker.label
		SET
			name = $2
			, color = $3
			, updated_at = now()
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING` + labelSelectColumns

	var out model.Label
	if err := sqlx.GetContext(ctx, r.db, &out, q, l.ID, l.Name, l.Color); err != nil {
		return model.Label{}, errs.DB(err)
	}
	return out, nil
}

// Delete мягко удаляет метку (deleted_at = now()). Повторное удаление
// (строка уже удалена/отсутствует) не считается ошибкой.
func (r *LabelRepo) Delete(ctx context.Context, labelID string) error {
	const q = `
		UPDATE tracker.label
		SET deleted_at = now()
		WHERE id = $1 AND deleted_at IS NULL`

	if _, err := r.db.ExecContext(ctx, q, labelID); err != nil {
		return errs.DB(err)
	}
	return nil
}

// GetWorkspaceLabelByName возвращает активную workspace-level метку
// (team_id IS NULL) workspace по имени. Нет строки → errs.NotFound (через
// errs.DB). Используется правилом «нельзя создать team-label поверх ws-label».
func (r *LabelRepo) GetWorkspaceLabelByName(ctx context.Context, workspaceID, name string) (model.Label, error) {
	const q = `
		SELECT` + labelSelectColumns + `
		FROM tracker.label
		WHERE workspace_id = $1 AND team_id IS NULL AND name = $2 AND deleted_at IS NULL`

	var out model.Label
	if err := sqlx.GetContext(ctx, r.db, &out, q, workspaceID, name); err != nil {
		return model.Label{}, errs.DB(err)
	}
	return out, nil
}

// ListTeamLabelsByName возвращает активные team-level метки (team_id IS NOT NULL)
// workspace по имени (across teams). Используется правилом merge при создании
// ws-level метки поверх одноимённых team-меток.
func (r *LabelRepo) ListTeamLabelsByName(ctx context.Context, workspaceID, name string) ([]model.Label, error) {
	const q = `
		SELECT` + labelSelectColumns + `
		FROM tracker.label
		WHERE workspace_id = $1 AND team_id IS NOT NULL AND name = $2 AND deleted_at IS NULL
		ORDER BY created_at, id`

	var out []model.Label
	if err := sqlx.SelectContext(ctx, r.db, &out, q, workspaceID, name); err != nil {
		return nil, errs.DB(err)
	}
	return out, nil
}

// RepointIssueLabels перепривязывает связи issue_label со старых меток
// (fromLabelIDs) на целевую (toLabelID): сначала upsert связей на целевую метку
// (ON CONFLICT восстанавливает soft-deleted), затем soft-delete связей со старых.
// Пустой fromLabelIDs — no-op. Вызывается в рамках merge-транзакции сервиса.
func (r *LabelRepo) RepointIssueLabels(ctx context.Context, fromLabelIDs []string, toLabelID string) error {
	if len(fromLabelIDs) == 0 {
		return nil
	}

	const insertQ = `
		INSERT INTO tracker.issue_label (issue_id, label_id)
		SELECT DISTINCT issue_id, $2::uuid
		FROM tracker.issue_label
		WHERE label_id = ANY($1::uuid[]) AND deleted_at IS NULL
		ON CONFLICT (issue_id, label_id) DO UPDATE SET
			deleted_at = NULL`
	if _, err := r.db.ExecContext(ctx, insertQ, pq.Array(fromLabelIDs), toLabelID); err != nil {
		return errs.DB(err)
	}

	const deleteQ = `
		UPDATE tracker.issue_label
		SET deleted_at = now()
		WHERE label_id = ANY($1::uuid[]) AND deleted_at IS NULL`
	if _, err := r.db.ExecContext(ctx, deleteQ, pq.Array(fromLabelIDs)); err != nil {
		return errs.DB(err)
	}
	return nil
}
