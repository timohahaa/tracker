package service

import (
	"context"

	"github.com/timohahaa/tracker/internal/errs"
	"github.com/timohahaa/tracker/internal/model"
	"github.com/timohahaa/tracker/internal/repository"
)

// ProjectService — CRUD проектов, связь с командами.
type ProjectService struct {
	repos repository.Store
}

// Дефолты и кэп пагинации списка проектов.
const (
	DefaultProjectLimit = 50
	MaxProjectLimit     = 100
)

// validProjectStatuses — допустимые статусы проекта (CHECK-констрейнт в БД).
var validProjectStatuses = map[model.ProjectStatus]bool{
	model.ProjectStatusPlanned:    true,
	model.ProjectStatusInProgress: true,
	model.ProjectStatusCompleted:  true,
	model.ProjectStatusCanceled:   true,
}

// allowedProjectSorts — поля сортировки списка проектов.
var allowedProjectSorts = map[string]bool{
	"created_at": true,
	"updated_at": true,
}

// CreateProjectParams — входные данные создания проекта (граница service).
// Status по умолчанию — planned. TeamIDs привязывает команды сразу при создании
// (nil = без команд).
type CreateProjectParams struct {
	Name        string
	Description *string
	LeadID      *string
	Status      model.ProjectStatus
	TeamIDs     *[]string
}

// UpdateProjectParams — частичное обновление проекта. Nil-поле означает
// «не менять». TeamIDs — replace-семантика всего набора: nil = «не трогать
// команды», непустой/пустой слайс = заменить набор целиком (пустой = отвязать
// все).
type UpdateProjectParams struct {
	Name        *string
	Description *string
	LeadID      *string
	Status      *model.ProjectStatus
	TeamIDs     *[]string
}

// ProjectListResult — страница списка проектов с признаком продолжения и
// курсором на следующую страницу. NextCursor != nil только когда HasMore.
type ProjectListResult struct {
	Projects   []model.ProjectDetails
	HasMore    bool
	NextCursor *model.Cursor
}

// CreateProject создаёт проект и (опционально) привязывает команды в одной
// транзакции. Доступно любому участнику workspace (membership гарантирован
// middleware LoadWorkspace). Валидирует status/lead_id/team_ids.
func (s *ProjectService) CreateProject(ctx context.Context, actor model.WorkspaceMember, params CreateProjectParams) (model.ProjectDetails, error) {
	status := params.Status
	if status == "" {
		status = model.ProjectStatusPlanned
	}
	if err := validateProjectStatus(status); err != nil {
		return model.ProjectDetails{}, err
	}
	if err := s.validateLead(ctx, actor.WorkspaceID, params.LeadID); err != nil {
		return model.ProjectDetails{}, err
	}
	if params.TeamIDs != nil {
		if err := s.validateTeams(ctx, actor.WorkspaceID, *params.TeamIDs); err != nil {
			return model.ProjectDetails{}, err
		}
	}

	var project model.Project
	err := s.repos.WithTx(ctx, func(tx repository.Store) error {
		var cerr error
		project, cerr = tx.Project().Create(ctx, model.Project{
			WorkspaceID: actor.WorkspaceID,
			LeadID:      params.LeadID,
			Name:        params.Name,
			Description: params.Description,
			Status:      status,
		})
		if cerr != nil {
			return cerr
		}
		if params.TeamIDs != nil {
			return replaceProjectTeams(ctx, tx, project.ID, *params.TeamIDs)
		}
		return nil
	})
	if err != nil {
		return model.ProjectDetails{}, err
	}

	return s.enrichTeams(ctx, project)
}

// GetProject возвращает проект workspace по id с привязанными командами.
// Доступно участнику workspace. Нет проекта → errs.NotFound.
func (s *ProjectService) GetProject(ctx context.Context, workspaceID, projectID string) (model.ProjectDetails, error) {
	project, err := s.repos.Project().GetByID(ctx, workspaceID, projectID)
	if err != nil {
		return model.ProjectDetails{}, err
	}
	return s.enrichTeams(ctx, project)
}

// ListProjects возвращает страницу проектов workspace с keyset-пагинацией.
// Нормализует page (дефолты sort=updated_at/order=desc/limit + кэп), валидирует
// sort/order/cursor, выбирает limit+1 строк и по «лишней» вычисляет has_more/
// next_cursor. Команды догружаются батчем (без N+1). Доступно участнику
// workspace.
func (s *ProjectService) ListProjects(ctx context.Context, workspaceID string, page model.Page) (ProjectListResult, error) {
	page, err := normalizeProjectPage(page)
	if err != nil {
		return ProjectListResult{}, err
	}
	// Курсор проектов всегда timestamptz (created_at/updated_at) — валидируем под
	// дату (тот же контракт, что и у issue-курсора).
	if err := validateCursor(page.Cursor, page.Sort); err != nil {
		return ProjectListResult{}, err
	}

	rows, err := s.repos.Project().List(ctx, workspaceID, page)
	if err != nil {
		return ProjectListResult{}, err
	}

	res := ProjectListResult{}
	if len(rows) > page.Limit {
		last := rows[page.Limit-1]
		res.HasMore = true
		res.NextCursor = &model.Cursor{Value: projectSortValue(last, page.Sort), ID: last.ID}
		rows = rows[:page.Limit]
	}

	details, err := s.enrichListTeams(ctx, rows)
	if err != nil {
		return ProjectListResult{}, err
	}
	res.Projects = details
	return res, nil
}

// UpdateProject частично обновляет проект. Доступно участнику workspace.
// Валидирует status/lead_id/team_ids; при заданном team_ids заменяет набор
// команд (replace-семантика) в одной транзакции с обновлением полей.
func (s *ProjectService) UpdateProject(ctx context.Context, actor model.WorkspaceMember, projectID string, params UpdateProjectParams) (model.ProjectDetails, error) {
	project, err := s.repos.Project().GetByID(ctx, actor.WorkspaceID, projectID)
	if err != nil {
		return model.ProjectDetails{}, err
	}

	if params.Name != nil {
		project.Name = *params.Name
	}
	if params.Description != nil {
		project.Description = params.Description
	}
	if params.LeadID != nil {
		if err := s.validateLead(ctx, actor.WorkspaceID, params.LeadID); err != nil {
			return model.ProjectDetails{}, err
		}
		project.LeadID = params.LeadID
	}
	if params.Status != nil {
		if err := validateProjectStatus(*params.Status); err != nil {
			return model.ProjectDetails{}, err
		}
		project.Status = *params.Status
	}
	if params.TeamIDs != nil {
		if err := s.validateTeams(ctx, actor.WorkspaceID, *params.TeamIDs); err != nil {
			return model.ProjectDetails{}, err
		}
	}

	// Команды не трогаем — простой путь (одиночный UPDATE без транзакции).
	if params.TeamIDs == nil {
		updated, err := s.repos.Project().Update(ctx, project)
		if err != nil {
			return model.ProjectDetails{}, err
		}
		return s.enrichTeams(ctx, updated)
	}

	// Replace набора команд + обновление полей — в одной транзакции.
	var updated model.Project
	err = s.repos.WithTx(ctx, func(tx repository.Store) error {
		var uerr error
		updated, uerr = tx.Project().Update(ctx, project)
		if uerr != nil {
			return uerr
		}
		return replaceProjectTeams(ctx, tx, project.ID, *params.TeamIDs)
	})
	if err != nil {
		return model.ProjectDetails{}, err
	}
	return s.enrichTeams(ctx, updated)
}

// DeleteProject мягко удаляет проект. Доступно только owner/admin workspace.
func (s *ProjectService) DeleteProject(ctx context.Context, actor model.WorkspaceMember, projectID string) error {
	if !canManageWorkspace(actor.Role) {
		return errs.Forbidden("insufficient permissions")
	}
	project, err := s.repos.Project().GetByID(ctx, actor.WorkspaceID, projectID)
	if err != nil {
		return err
	}
	return s.repos.Project().Delete(ctx, project.ID)
}

// enrichTeams догружает привязанные команды проекта и собирает read-модель.
func (s *ProjectService) enrichTeams(ctx context.Context, project model.Project) (model.ProjectDetails, error) {
	teams, err := s.repos.Project().ListTeams(ctx, project.ID)
	if err != nil {
		return model.ProjectDetails{}, err
	}
	ids := make([]string, 0, len(teams))
	for _, t := range teams {
		ids = append(ids, t.TeamID)
	}
	return model.ProjectDetails{Project: project, TeamIDs: ids}, nil
}

// enrichListTeams догружает команды для набора проектов одним запросом (без N+1)
// и раскладывает их по строкам, сохраняя порядок.
func (s *ProjectService) enrichListTeams(ctx context.Context, projects []model.Project) ([]model.ProjectDetails, error) {
	out := make([]model.ProjectDetails, 0, len(projects))
	if len(projects) == 0 {
		return out, nil
	}
	ids := make([]string, len(projects))
	for i := range projects {
		ids[i] = projects[i].ID
	}
	byProject, err := s.repos.Project().ListTeamsByProjects(ctx, ids)
	if err != nil {
		return nil, err
	}
	for _, p := range projects {
		teams := byProject[p.ID]
		teamIDs := make([]string, 0, len(teams))
		for _, t := range teams {
			teamIDs = append(teamIDs, t.TeamID)
		}
		out = append(out, model.ProjectDetails{Project: p, TeamIDs: teamIDs})
	}
	return out, nil
}

// validateLead проверяет, что lead — активный участник workspace. nil-lead
// валиден (проект без ведущего). Иначе errs.Validation.
func (s *ProjectService) validateLead(ctx context.Context, workspaceID string, leadID *string) error {
	if leadID == nil {
		return nil
	}
	if _, err := s.repos.Workspace().GetMember(ctx, workspaceID, *leadID); err != nil {
		if isNotFound(err) {
			return errs.Validation("invalid lead", errs.FieldError{
				Field:   "lead_id",
				Reason:  "invalid",
				Message: "must be an active workspace member",
			})
		}
		return err
	}
	return nil
}

// validateTeams проверяет, что каждая команда принадлежит workspace. Иначе
// errs.Validation.
func (s *ProjectService) validateTeams(ctx context.Context, workspaceID string, teamIDs []string) error {
	for _, teamID := range teamIDs {
		if _, err := s.repos.Team().Get(ctx, workspaceID, teamID); err != nil {
			if isNotFound(err) {
				return errs.Validation("invalid team", errs.FieldError{
					Field:   "team_ids",
					Reason:  "invalid",
					Message: "must be teams of the workspace",
				})
			}
			return err
		}
	}
	return nil
}

// validateProjectStatus проверяет допустимость статуса проекта. Иначе
// errs.Validation.
func validateProjectStatus(status model.ProjectStatus) error {
	if !validProjectStatuses[status] {
		return errs.Validation("invalid status", errs.FieldError{
			Field:   "status",
			Reason:  "invalid",
			Message: "must be one of planned, in_progress, completed, canceled",
		})
	}
	return nil
}

// replaceProjectTeams приводит набор активных команд проекта к целевому (set) в
// рамках переданной транзакции: soft-delete отсутствующих, upsert (восстановление
// soft-deleted) целевых. Набор считается уже провалидированным (validateTeams).
func replaceProjectTeams(ctx context.Context, tx repository.Store, projectID string, set []string) error {
	current, err := tx.Project().ListTeams(ctx, projectID)
	if err != nil {
		return err
	}

	want := make(map[string]bool, len(set))
	for _, id := range set {
		want[id] = true
	}

	// Отвязать те команды, которых нет в новом наборе.
	for _, cur := range current {
		if !want[cur.TeamID] {
			if rerr := tx.Project().RemoveTeam(ctx, projectID, cur.TeamID); rerr != nil {
				return rerr
			}
		}
	}

	// Привязать новый набор (восстановление soft-deleted через upsert).
	for _, teamID := range set {
		if _, aerr := tx.Project().AddTeam(ctx, projectID, teamID); aerr != nil {
			return aerr
		}
	}
	return nil
}

// normalizeProjectPage применяет дефолты и кэп к параметрам пагинации и
// валидирует sort/order. Невалидные sort/order → errs.Validation.
func normalizeProjectPage(page model.Page) (model.Page, error) {
	if page.Limit <= 0 {
		page.Limit = DefaultProjectLimit
	}
	if page.Limit > MaxProjectLimit {
		page.Limit = MaxProjectLimit
	}
	if page.Sort == "" {
		page.Sort = "updated_at"
	}
	if !allowedProjectSorts[page.Sort] {
		return model.Page{}, errs.Validation("invalid sort", errs.FieldError{
			Field:   "sort",
			Reason:  "invalid",
			Message: "must be one of created_at, updated_at",
		})
	}
	if page.Order == "" {
		page.Order = model.SortDesc
	}
	if page.Order != model.SortAsc && page.Order != model.SortDesc {
		return model.Page{}, errs.Validation("invalid order", errs.FieldError{
			Field:   "order",
			Reason:  "invalid",
			Message: "must be one of asc, desc",
		})
	}
	return page, nil
}

// projectSortValue возвращает значение поля сортировки последнего элемента для
// курсора (RFC3339Nano-таймстамп created_at/updated_at).
func projectSortValue(p model.Project, sort string) string {
	if sort == "created_at" {
		return p.CreatedAt.Format(cursorTimeLayout)
	}
	return p.UpdatedAt.Format(cursorTimeLayout)
}
