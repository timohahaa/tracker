package repository

import (
	"context"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"

	"github.com/timohahaa/tracker/internal/errs"
	"github.com/timohahaa/tracker/internal/model"
)

// assigneeSelectColumns — общий список колонок участника задачи
// (tracker.issue_assignee).
const assigneeSelectColumns = `
	issue_id
	, user_id
	, is_primary
	, created_at
	, updated_at
	, deleted_at`

// ListAssignees возвращает активных участников задачи (deleted_at IS NULL).
// Сортировка стабильна (created_at, user_id). is_primary == ответственный.
func (r *IssueRepo) ListAssignees(ctx context.Context, issueID string) ([]model.IssueAssignee, error) {
	const q = `
		SELECT` + assigneeSelectColumns + `
		FROM tracker.issue_assignee
		WHERE issue_id = $1 AND deleted_at IS NULL
		ORDER BY created_at, user_id`

	var out []model.IssueAssignee
	if err := sqlx.SelectContext(ctx, r.db, &out, q, issueID); err != nil {
		return nil, errs.DB(err)
	}
	return out, nil
}

// ListAssigneesByIssues батчем загружает активных участников для набора задач
// (без N+1) и раскладывает их по issue_id. Пустой вход → пустая map.
func (r *IssueRepo) ListAssigneesByIssues(ctx context.Context, issueIDs []string) (map[string][]model.IssueAssignee, error) {
	out := make(map[string][]model.IssueAssignee, len(issueIDs))
	if len(issueIDs) == 0 {
		return out, nil
	}

	const q = `
		SELECT` + assigneeSelectColumns + `
		FROM tracker.issue_assignee
		WHERE issue_id = ANY($1::uuid[]) AND deleted_at IS NULL
		ORDER BY created_at, user_id`

	var rows []model.IssueAssignee
	if err := sqlx.SelectContext(ctx, r.db, &rows, q, pq.Array(issueIDs)); err != nil {
		return nil, errs.DB(err)
	}
	for _, a := range rows {
		out[a.IssueID] = append(out[a.IssueID], a)
	}
	return out, nil
}

// GetAssignee возвращает активного участника задачи по (issue_id, user_id).
// Нет строки → errs.NotFound (через errs.DB).
func (r *IssueRepo) GetAssignee(ctx context.Context, issueID, userID string) (model.IssueAssignee, error) {
	const q = `
		SELECT` + assigneeSelectColumns + `
		FROM tracker.issue_assignee
		WHERE issue_id = $1 AND user_id = $2 AND deleted_at IS NULL`

	var out model.IssueAssignee
	if err := sqlx.GetContext(ctx, r.db, &out, q, issueID, userID); err != nil {
		return model.IssueAssignee{}, errs.DB(err)
	}
	return out, nil
}

// AddAssignee добавляет участника задачи (upsert по composite PK). Управлением
// «ответственного» метод НЕ занимается — это делает сервис отдельным шагом
// (ClearPrimary + SetPrimary в транзакции). Поэтому is_primary при апсерте
// трактуется так:
//   - активная строка (deleted_at IS NULL) — is_primary НЕ меняется (повторное
//     добавление участника идемпотентно и не снимает с него ответственность);
//   - восстановление soft-deleted строки — is_primary берётся из входящего
//     значения (сервис всегда передаёт false), чтобы не «воскресить» устаревший
//     is_primary и не нарушить partial unique idx_tracker_issue_assignee_uniq_primary.
//
// Возвращает актуальную строку.
func (r *IssueRepo) AddAssignee(ctx context.Context, a model.IssueAssignee) (model.IssueAssignee, error) {
	const q = `
		INSERT INTO tracker.issue_assignee (
			issue_id
			, user_id
			, is_primary
		)
		VALUES ($1, $2, $3)
		ON CONFLICT (issue_id, user_id) DO UPDATE SET
			is_primary = CASE
				WHEN tracker.issue_assignee.deleted_at IS NULL THEN tracker.issue_assignee.is_primary
				ELSE EXCLUDED.is_primary
			END
			, deleted_at = NULL
			, updated_at = now()
		RETURNING` + assigneeSelectColumns

	var out model.IssueAssignee
	if err := sqlx.GetContext(ctx, r.db, &out, q, a.IssueID, a.UserID, a.IsPrimary); err != nil {
		return model.IssueAssignee{}, errs.DB(err)
	}
	return out, nil
}

// RemoveAssignee мягко удаляет участника задачи (deleted_at = now()). Повторное
// удаление не считается ошибкой (идемпотентно). Снятие участника-ответственного
// оставляет задачу без ответственного (partial unique учитывает только активные
// строки).
func (r *IssueRepo) RemoveAssignee(ctx context.Context, issueID, userID string) error {
	const q = `
		UPDATE tracker.issue_assignee
		SET deleted_at = now()
		WHERE issue_id = $1 AND user_id = $2 AND deleted_at IS NULL`

	if _, err := r.db.ExecContext(ctx, q, issueID, userID); err != nil {
		return errs.DB(err)
	}
	return nil
}

// ClearPrimary снимает флаг is_primary с текущего ответственного задачи (если
// есть). Идемпотентно. Вызывается перед SetPrimary в одной транзакции, т.к.
// partial unique index нельзя сделать DEFERRABLE.
func (r *IssueRepo) ClearPrimary(ctx context.Context, issueID string) error {
	const q = `
		UPDATE tracker.issue_assignee
		SET is_primary = false, updated_at = now()
		WHERE issue_id = $1 AND is_primary AND deleted_at IS NULL`

	if _, err := r.db.ExecContext(ctx, q, issueID); err != nil {
		return errs.DB(err)
	}
	return nil
}

// SetPrimary выставляет is_primary активному участнику задачи. Вызывается после
// ClearPrimary в одной транзакции. Нарушение partial unique (два primary без
// предварительного ClearPrimary) → errs.AlreadyExists (через errs.DB).
func (r *IssueRepo) SetPrimary(ctx context.Context, issueID, userID string) error {
	const q = `
		UPDATE tracker.issue_assignee
		SET is_primary = true, updated_at = now()
		WHERE issue_id = $1 AND user_id = $2 AND deleted_at IS NULL`

	if _, err := r.db.ExecContext(ctx, q, issueID, userID); err != nil {
		return errs.DB(err)
	}
	return nil
}
