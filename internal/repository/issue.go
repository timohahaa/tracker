package repository

import (
	"context"

	"github.com/jmoiron/sqlx"

	"github.com/timohahaa/tracker/internal/errs"
	"github.com/timohahaa/tracker/internal/model"
)

// IssueRepo — доступ к tracker.issue, issue_assignee, issue_label.
// Read-операции (List с JOIN, фильтры) при росте выносятся в issue_read.go.
type IssueRepo struct {
	db Ext
}

// issueSelectColumns — общий список колонок задачи с денормализованными
// team_key/identifier (JOIN org.team). Identifier не хранится, вычисляется как
// team.key || '-' || issue_number.
const issueSelectColumns = `
	i.id
	, i.team_id
	, i.project_id
	, i.status_id
	, i.creator_id
	, i.title
	, i.description
	, i.priority
	, i.due_date
	, i.issue_number
	, i.created_at
	, i.updated_at
	, i.deleted_at
	, t.key AS team_key
	, t.key || '-' || i.issue_number AS identifier`

// Create вставляет задачу (id/created_at/updated_at генерит БД) и возвращает
// созданную строку. issue_number должен быть выдан заранее через
// IncrementIssueNumber в той же транзакции.
func (r *IssueRepo) Create(ctx context.Context, i model.Issue) (model.Issue, error) {
	const q = `
		INSERT INTO tracker.issue (
			team_id
			, project_id
			, status_id
			, creator_id
			, title
			, description
			, priority
			, due_date
			, issue_number
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING
			id
			, team_id
			, project_id
			, status_id
			, creator_id
			, title
			, description
			, priority
			, due_date
			, issue_number
			, created_at
			, updated_at
			, deleted_at`

	var out model.Issue
	if err := sqlx.GetContext(ctx, r.db, &out, q,
		i.TeamID, i.ProjectID, i.StatusID, i.CreatorID, i.Title,
		i.Description, i.Priority, i.DueDate, i.IssueNumber,
	); err != nil {
		return model.Issue{}, errs.DB(err)
	}
	return out, nil
}

// IncrementIssueNumber атомарно инкрементирует next_issue_number команды
// (row-level lock) и возвращает новое значение. next_issue_number = «последний
// выданный номер» (DEFAULT 0), поэтому первый issue получает номер 1. Нет
// команды → errs.NotFound (через errs.DB).
func (r *IssueRepo) IncrementIssueNumber(ctx context.Context, teamID string) (int, error) {
	const q = `
		UPDATE org.team
		SET next_issue_number = next_issue_number + 1
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING next_issue_number`

	var n int
	if err := sqlx.GetContext(ctx, r.db, &n, q, teamID); err != nil {
		return 0, errs.DB(err)
	}
	return n, nil
}

// GetByID возвращает активную задачу workspace по UUID вместе с identifier/
// team_key. Assignees/Labels в этом заходе не заполняются. Нет строки →
// errs.NotFound (через errs.DB).
func (r *IssueRepo) GetByID(ctx context.Context, workspaceID, issueID string) (model.IssueDetails, error) {
	const q = `
		SELECT` + issueSelectColumns + `
		FROM tracker.issue i
		JOIN org.team t ON t.id = i.team_id
		WHERE i.id = $1
			AND t.workspace_id = $2
			AND i.deleted_at IS NULL
			AND t.deleted_at IS NULL`

	var out model.IssueDetails
	if err := sqlx.GetContext(ctx, r.db, &out, q, issueID, workspaceID); err != nil {
		return model.IssueDetails{}, errs.DB(err)
	}
	return out, nil
}

// GetByIdentifier возвращает активную задачу workspace по (team key, issue
// number) вместе с identifier/team_key. Нет строки → errs.NotFound.
func (r *IssueRepo) GetByIdentifier(ctx context.Context, workspaceID, teamKey string, issueNumber int) (model.IssueDetails, error) {
	const q = `
		SELECT` + issueSelectColumns + `
		FROM tracker.issue i
		JOIN org.team t ON t.id = i.team_id
		WHERE t.key = $1
			AND i.issue_number = $2
			AND t.workspace_id = $3
			AND i.deleted_at IS NULL
			AND t.deleted_at IS NULL`

	var out model.IssueDetails
	if err := sqlx.GetContext(ctx, r.db, &out, q, teamKey, issueNumber, workspaceID); err != nil {
		return model.IssueDetails{}, errs.DB(err)
	}
	return out, nil
}

// Update частично обновляет задачу (title/description/status_id/priority/
// due_date/project_id) и возвращает обновлённую строку. Нет строки →
// errs.NotFound (через errs.DB).
func (r *IssueRepo) Update(ctx context.Context, i model.Issue) (model.Issue, error) {
	const q = `
		UPDATE tracker.issue
		SET
			title = $2
			, description = $3
			, status_id = $4
			, priority = $5
			, due_date = $6
			, project_id = $7
			, updated_at = now()
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING
			id
			, team_id
			, project_id
			, status_id
			, creator_id
			, title
			, description
			, priority
			, due_date
			, issue_number
			, created_at
			, updated_at
			, deleted_at`

	var out model.Issue
	if err := sqlx.GetContext(ctx, r.db, &out, q,
		i.ID, i.Title, i.Description, i.StatusID, i.Priority, i.DueDate, i.ProjectID,
	); err != nil {
		return model.Issue{}, errs.DB(err)
	}
	return out, nil
}

// Delete мягко удаляет задачу (deleted_at = now()). Повторное удаление
// (строка уже удалена/отсутствует) не считается ошибкой.
func (r *IssueRepo) Delete(ctx context.Context, issueID string) error {
	const q = `
		UPDATE tracker.issue
		SET deleted_at = now()
		WHERE id = $1 AND deleted_at IS NULL`

	if _, err := r.db.ExecContext(ctx, q, issueID); err != nil {
		return errs.DB(err)
	}
	return nil
}

// ProjectInWorkspace сообщает, существует ли активный проект с данным id в
// указанном workspace. Используется для валидации project_id при работе с
// задачей.
func (r *IssueRepo) ProjectInWorkspace(ctx context.Context, workspaceID, projectID string) (bool, error) {
	const q = `
		SELECT EXISTS (
			SELECT 1
			FROM tracker.project
			WHERE id = $1 AND workspace_id = $2 AND deleted_at IS NULL
		)`

	var ok bool
	if err := sqlx.GetContext(ctx, r.db, &ok, q, projectID, workspaceID); err != nil {
		return false, errs.DB(err)
	}
	return ok, nil
}
