package repository

import (
	"context"

	"github.com/jmoiron/sqlx"

	"github.com/timohahaa/tracker/internal/errs"
	"github.com/timohahaa/tracker/internal/model"
)

// TeamRepo — доступ к org.team и org.team_member.
type TeamRepo struct {
	db Ext
}

// Create вставляет команду (id/created_at/updated_at генерит БД) и возвращает
// созданную строку. Нарушение уникальности key per workspace → errs.AlreadyExists
// (через errs.DB).
func (r *TeamRepo) Create(ctx context.Context, t model.Team) (model.Team, error) {
	const q = `
		INSERT INTO org.team (
			workspace_id
			, name
			, description
			, key
		)
		VALUES ($1, $2, $3, $4)
		RETURNING
			id
			, workspace_id
			, name
			, description
			, key
			, next_issue_number
			, created_at
			, updated_at
			, deleted_at`

	var out model.Team
	if err := sqlx.GetContext(ctx, r.db, &out, q, t.WorkspaceID, t.Name, t.Description, t.Key); err != nil {
		return model.Team{}, errs.DB(err)
	}
	return out, nil
}

// Get возвращает активную команду по (workspace_id, id). Нет строки → errs.NotFound.
func (r *TeamRepo) Get(ctx context.Context, workspaceID, teamID string) (model.Team, error) {
	const q = `
		SELECT
			id
			, workspace_id
			, name
			, description
			, key
			, next_issue_number
			, created_at
			, updated_at
			, deleted_at
		FROM org.team
		WHERE id = $1 AND workspace_id = $2 AND deleted_at IS NULL`

	var out model.Team
	if err := sqlx.GetContext(ctx, r.db, &out, q, teamID, workspaceID); err != nil {
		return model.Team{}, errs.DB(err)
	}
	return out, nil
}

// List возвращает активные команды workspace. Сортировка стабильна (created_at, id).
func (r *TeamRepo) List(ctx context.Context, workspaceID string) ([]model.Team, error) {
	const q = `
		SELECT
			id
			, workspace_id
			, name
			, description
			, key
			, next_issue_number
			, created_at
			, updated_at
			, deleted_at
		FROM org.team
		WHERE workspace_id = $1 AND deleted_at IS NULL
		ORDER BY created_at, id`

	var out []model.Team
	if err := sqlx.SelectContext(ctx, r.db, &out, q, workspaceID); err != nil {
		return nil, errs.DB(err)
	}
	return out, nil
}

// Update обновляет name/description/key команды и возвращает обновлённую строку.
// Нарушение уникальности key per workspace → errs.AlreadyExists (через errs.DB).
func (r *TeamRepo) Update(ctx context.Context, t model.Team) (model.Team, error) {
	const q = `
		UPDATE org.team
		SET
			name = $2
			, description = $3
			, key = $4
			, updated_at = now()
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING
			id
			, workspace_id
			, name
			, description
			, key
			, next_issue_number
			, created_at
			, updated_at
			, deleted_at`

	var out model.Team
	if err := sqlx.GetContext(ctx, r.db, &out, q, t.ID, t.Name, t.Description, t.Key); err != nil {
		return model.Team{}, errs.DB(err)
	}
	return out, nil
}

// Delete мягко удаляет команду (deleted_at = now()). Повторное удаление
// (строка уже удалена/отсутствует) не считается ошибкой.
func (r *TeamRepo) Delete(ctx context.Context, teamID string) error {
	const q = `
		UPDATE org.team
		SET deleted_at = now()
		WHERE id = $1 AND deleted_at IS NULL`

	if _, err := r.db.ExecContext(ctx, q, teamID); err != nil {
		return errs.DB(err)
	}
	return nil
}

// AddMember добавляет участника команды (upsert по composite PK): при повторном
// добавлении удалённого участника связь восстанавливается (deleted_at = NULL) и
// обновляется роль. Возвращает актуальную строку.
func (r *TeamRepo) AddMember(ctx context.Context, m model.TeamMember) (model.TeamMember, error) {
	const q = `
		INSERT INTO org.team_member (
			team_id
			, user_id
			, role
		)
		VALUES ($1, $2, $3)
		ON CONFLICT (team_id, user_id) DO UPDATE SET
			role = EXCLUDED.role
			, deleted_at = NULL
			, updated_at = now()
		RETURNING
			team_id
			, user_id
			, role
			, created_at
			, updated_at
			, deleted_at`

	var out model.TeamMember
	if err := sqlx.GetContext(ctx, r.db, &out, q, m.TeamID, m.UserID, m.Role); err != nil {
		return model.TeamMember{}, errs.DB(err)
	}
	return out, nil
}

// ListMembers возвращает активных участников команды. Сортировка стабильна
// (created_at, user_id).
func (r *TeamRepo) ListMembers(ctx context.Context, teamID string) ([]model.TeamMember, error) {
	const q = `
		SELECT
			team_id
			, user_id
			, role
			, created_at
			, updated_at
			, deleted_at
		FROM org.team_member
		WHERE team_id = $1 AND deleted_at IS NULL
		ORDER BY created_at, user_id`

	var out []model.TeamMember
	if err := sqlx.SelectContext(ctx, r.db, &out, q, teamID); err != nil {
		return nil, errs.DB(err)
	}
	return out, nil
}

// GetMember возвращает активного участника команды по (team_id, user_id).
// Нет строки → errs.NotFound (через errs.DB).
func (r *TeamRepo) GetMember(ctx context.Context, teamID, userID string) (model.TeamMember, error) {
	const q = `
		SELECT
			team_id
			, user_id
			, role
			, created_at
			, updated_at
			, deleted_at
		FROM org.team_member
		WHERE team_id = $1 AND user_id = $2 AND deleted_at IS NULL`

	var out model.TeamMember
	if err := sqlx.GetContext(ctx, r.db, &out, q, teamID, userID); err != nil {
		return model.TeamMember{}, errs.DB(err)
	}
	return out, nil
}

// RemoveMember мягко удаляет участника команды (deleted_at = now()). Повторное
// удаление не считается ошибкой.
func (r *TeamRepo) RemoveMember(ctx context.Context, teamID, userID string) error {
	const q = `
		UPDATE org.team_member
		SET deleted_at = now()
		WHERE team_id = $1 AND user_id = $2 AND deleted_at IS NULL`

	if _, err := r.db.ExecContext(ctx, q, teamID, userID); err != nil {
		return errs.DB(err)
	}
	return nil
}
