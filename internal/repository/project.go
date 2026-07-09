package repository

import (
	"context"
	"strconv"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"

	"github.com/timohahaa/tracker/internal/errs"
	"github.com/timohahaa/tracker/internal/model"
)

// ProjectRepo — доступ к tracker.project и project_team.
type ProjectRepo struct {
	db Ext
}

// projectSelectColumns — общий список колонок проекта.
const projectSelectColumns = `
	id
	, workspace_id
	, lead_id
	, name
	, description
	, status
	, created_at
	, updated_at
	, deleted_at`

// Create вставляет проект (id/created_at/updated_at генерит БД) и возвращает
// созданную строку.
func (r *ProjectRepo) Create(ctx context.Context, p model.Project) (model.Project, error) {
	const q = `
		INSERT INTO tracker.project (
			workspace_id
			, lead_id
			, name
			, description
			, status
		)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING` + projectSelectColumns

	var out model.Project
	if err := sqlx.GetContext(ctx, r.db, &out, q,
		p.WorkspaceID, p.LeadID, p.Name, p.Description, p.Status,
	); err != nil {
		return model.Project{}, errs.DB(err)
	}
	return out, nil
}

// GetByID возвращает активный проект по (workspace_id, id). Нет строки →
// errs.NotFound (через errs.DB).
func (r *ProjectRepo) GetByID(ctx context.Context, workspaceID, projectID string) (model.Project, error) {
	const q = `
		SELECT` + projectSelectColumns + `
		FROM tracker.project
		WHERE id = $1 AND workspace_id = $2 AND deleted_at IS NULL`

	var out model.Project
	if err := sqlx.GetContext(ctx, r.db, &out, q, projectID, workspaceID); err != nil {
		return model.Project{}, errs.DB(err)
	}
	return out, nil
}

// Update обновляет lead_id/name/description/status проекта и возвращает
// обновлённую строку. Нет строки → errs.NotFound (через errs.DB).
func (r *ProjectRepo) Update(ctx context.Context, p model.Project) (model.Project, error) {
	const q = `
		UPDATE tracker.project
		SET
			lead_id = $2
			, name = $3
			, description = $4
			, status = $5
			, updated_at = now()
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING` + projectSelectColumns

	var out model.Project
	if err := sqlx.GetContext(ctx, r.db, &out, q,
		p.ID, p.LeadID, p.Name, p.Description, p.Status,
	); err != nil {
		return model.Project{}, errs.DB(err)
	}
	return out, nil
}

// Delete мягко удаляет проект (deleted_at = now()). Повторное удаление
// (строка уже удалена/отсутствует) не считается ошибкой.
func (r *ProjectRepo) Delete(ctx context.Context, projectID string) error {
	const q = `
		UPDATE tracker.project
		SET deleted_at = now()
		WHERE id = $1 AND deleted_at IS NULL`

	if _, err := r.db.ExecContext(ctx, q, projectID); err != nil {
		return errs.DB(err)
	}
	return nil
}

// List возвращает активные проекты workspace с keyset-пагинацией по (sort, id).
// Выбирается page.Limit+1 строк — сервис по «лишней» строке вычисляет has_more.
// page.Sort/Order считаются уже нормализованными сервисом.
func (r *ProjectRepo) List(ctx context.Context, workspaceID string, page model.Page) ([]model.Project, error) {
	args := []any{workspaceID}
	where := "workspace_id = $1 AND deleted_at IS NULL"

	sortExpr := "updated_at"
	if page.Sort == "created_at" {
		sortExpr = "created_at"
	}

	dir, op := "DESC", "<"
	if page.Order == model.SortAsc {
		dir, op = "ASC", ">"
	}

	if page.Cursor != nil {
		args = append(args, page.Cursor.Value, page.Cursor.ID)
		where += " AND (" + sortExpr + ", id) " + op + " ($2::timestamptz, $3::uuid)"
	}

	args = append(args, page.Limit+1)
	q := `
		SELECT` + projectSelectColumns + `
		FROM tracker.project
		WHERE ` + where + `
		ORDER BY ` + sortExpr + ` ` + dir + `, id ` + dir + `
		LIMIT $` + strconv.Itoa(len(args))

	var out []model.Project
	if err := sqlx.SelectContext(ctx, r.db, &out, q, args...); err != nil {
		return nil, errs.DB(err)
	}
	return out, nil
}

// ListTeams возвращает активные команды проекта (junction tracker.project_team).
// Сортировка стабильна (created_at, team_id).
func (r *ProjectRepo) ListTeams(ctx context.Context, projectID string) ([]model.ProjectTeam, error) {
	const q = `
		SELECT
			project_id
			, team_id
			, created_at
			, deleted_at
		FROM tracker.project_team
		WHERE project_id = $1 AND deleted_at IS NULL
		ORDER BY created_at, team_id`

	var out []model.ProjectTeam
	if err := sqlx.SelectContext(ctx, r.db, &out, q, projectID); err != nil {
		return nil, errs.DB(err)
	}
	return out, nil
}

// ListTeamsByProjects батчем загружает активные команды для набора проектов
// (без N+1) и раскладывает их по project_id. Пустой вход → пустая map.
func (r *ProjectRepo) ListTeamsByProjects(ctx context.Context, projectIDs []string) (map[string][]model.ProjectTeam, error) {
	out := make(map[string][]model.ProjectTeam, len(projectIDs))
	if len(projectIDs) == 0 {
		return out, nil
	}

	const q = `
		SELECT
			project_id
			, team_id
			, created_at
			, deleted_at
		FROM tracker.project_team
		WHERE project_id = ANY($1::uuid[]) AND deleted_at IS NULL
		ORDER BY created_at, team_id`

	var rows []model.ProjectTeam
	if err := sqlx.SelectContext(ctx, r.db, &rows, q, pq.Array(projectIDs)); err != nil {
		return nil, errs.DB(err)
	}
	for _, pt := range rows {
		out[pt.ProjectID] = append(out[pt.ProjectID], pt)
	}
	return out, nil
}

// AddTeam привязывает команду к проекту (upsert по composite PK): при повторной
// привязке удалённой связи она восстанавливается (deleted_at = NULL), дубля PK
// не возникает. Возвращает актуальную строку.
func (r *ProjectRepo) AddTeam(ctx context.Context, projectID, teamID string) (model.ProjectTeam, error) {
	const q = `
		INSERT INTO tracker.project_team (
			project_id
			, team_id
		)
		VALUES ($1, $2)
		ON CONFLICT (project_id, team_id) DO UPDATE SET
			deleted_at = NULL
		RETURNING
			project_id
			, team_id
			, created_at
			, deleted_at`

	var out model.ProjectTeam
	if err := sqlx.GetContext(ctx, r.db, &out, q, projectID, teamID); err != nil {
		return model.ProjectTeam{}, errs.DB(err)
	}
	return out, nil
}

// RemoveTeam мягко отвязывает команду от проекта (deleted_at = now()). Повторное
// удаление не считается ошибкой.
func (r *ProjectRepo) RemoveTeam(ctx context.Context, projectID, teamID string) error {
	const q = `
		UPDATE tracker.project_team
		SET deleted_at = now()
		WHERE project_id = $1 AND team_id = $2 AND deleted_at IS NULL`

	if _, err := r.db.ExecContext(ctx, q, projectID, teamID); err != nil {
		return errs.DB(err)
	}
	return nil
}
