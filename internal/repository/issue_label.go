package repository

import (
	"context"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"

	"github.com/timohahaa/tracker/internal/errs"
	"github.com/timohahaa/tracker/internal/model"
)

// issueLabelSelectColumns — колонки метки для JOIN issue_label → label. Префикс
// l.* соответствует псевдониму таблицы label в запросах ниже.
const issueLabelSelectColumns = `
	l.id
	, l.workspace_id
	, l.team_id
	, l.name
	, l.color
	, l.created_at
	, l.updated_at
	, l.deleted_at`

// ListLabels возвращает активные метки задачи (JOIN issue_label → label, обе
// стороны deleted_at IS NULL). Сортировка стабильна (l.created_at, l.id).
func (r *IssueRepo) ListLabels(ctx context.Context, issueID string) ([]model.Label, error) {
	const q = `
		SELECT` + issueLabelSelectColumns + `
		FROM tracker.issue_label il
		JOIN tracker.label l ON l.id = il.label_id
		WHERE il.issue_id = $1
			AND il.deleted_at IS NULL
			AND l.deleted_at IS NULL
		ORDER BY l.created_at, l.id`

	var out []model.Label
	if err := sqlx.SelectContext(ctx, r.db, &out, q, issueID); err != nil {
		return nil, errs.DB(err)
	}
	return out, nil
}

// ListLabelsByIssues батчем загружает активные метки для набора задач (без N+1)
// и раскладывает их по issue_id. Пустой вход → пустая map. issue_id несёт связь
// (в model.Label его нет), поэтому строки сканируются в локальную структуру.
func (r *IssueRepo) ListLabelsByIssues(ctx context.Context, issueIDs []string) (map[string][]model.Label, error) {
	out := make(map[string][]model.Label, len(issueIDs))
	if len(issueIDs) == 0 {
		return out, nil
	}

	const q = `
		SELECT
			il.issue_id AS issue_id
			,` + issueLabelSelectColumns + `
		FROM tracker.issue_label il
		JOIN tracker.label l ON l.id = il.label_id
		WHERE il.issue_id = ANY($1::uuid[])
			AND il.deleted_at IS NULL
			AND l.deleted_at IS NULL
		ORDER BY l.created_at, l.id`

	type row struct {
		IssueID string `db:"issue_id"`
		model.Label
	}
	var rows []row
	if err := sqlx.SelectContext(ctx, r.db, &rows, q, pq.Array(issueIDs)); err != nil {
		return nil, errs.DB(err)
	}
	for _, rw := range rows {
		out[rw.IssueID] = append(out[rw.IssueID], rw.Label)
	}
	return out, nil
}

// AddIssueLabel привязывает метку к задаче (upsert по composite PK). Повторная
// привязка идемпотентна; восстановление soft-deleted связи через
// ON CONFLICT DO UPDATE SET deleted_at = NULL.
func (r *IssueRepo) AddIssueLabel(ctx context.Context, issueID, labelID string) error {
	const q = `
		INSERT INTO tracker.issue_label (issue_id, label_id)
		VALUES ($1, $2)
		ON CONFLICT (issue_id, label_id) DO UPDATE SET
			deleted_at = NULL`

	if _, err := r.db.ExecContext(ctx, q, issueID, labelID); err != nil {
		return errs.DB(err)
	}
	return nil
}

// RemoveIssueLabel мягко удаляет связь метки с задачей (deleted_at = now()).
// Повторное удаление не считается ошибкой (идемпотентно).
func (r *IssueRepo) RemoveIssueLabel(ctx context.Context, issueID, labelID string) error {
	const q = `
		UPDATE tracker.issue_label
		SET deleted_at = now()
		WHERE issue_id = $1 AND label_id = $2 AND deleted_at IS NULL`

	if _, err := r.db.ExecContext(ctx, q, issueID, labelID); err != nil {
		return errs.DB(err)
	}
	return nil
}
