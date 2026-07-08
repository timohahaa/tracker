package service

import (
	"context"
	"strings"

	"github.com/timohahaa/tracker/internal/errs"
	"github.com/timohahaa/tracker/internal/model"
	"github.com/timohahaa/tracker/internal/repository"
)

// TeamService — команды, участники, статусы.
type TeamService struct {
	repos repository.Store
}

// defaultStatus — дефолтный статус, создаваемый вместе с командой (по одному на
// группу). Первый по position в группе считается дефолтным при автопереходах.
type defaultStatus struct {
	name  string
	group model.StatusGroup
}

// defaultStatuses — набор дефолтных статусов (по одному на каждую группу),
// создаётся в транзакции при создании команды.
var defaultStatuses = []defaultStatus{
	{name: "Backlog", group: model.StatusGroupBacklog},
	{name: "Todo", group: model.StatusGroupPlanned},
	{name: "In Progress", group: model.StatusGroupInProgress},
	{name: "Done", group: model.StatusGroupCompleted},
	{name: "Canceled", group: model.StatusGroupCanceled},
}

// CreateTeamParams — входные данные создания команды (граница service).
type CreateTeamParams struct {
	Name        string
	Key         string
	Description *string
}

// UpdateTeamParams — частичное обновление команды. Nil-поле означает «не менять».
type UpdateTeamParams struct {
	Name        *string
	Key         *string
	Description *string
}

// AddTeamMemberParams — входные данные добавления участника в команду.
type AddTeamMemberParams struct {
	UserID string
	Role   model.TeamRole
}

// CreateStatusParams — входные данные создания статуса.
type CreateStatusParams struct {
	Name     string
	Group    model.StatusGroup
	Position int
}

// UpdateStatusParams — частичное обновление статуса (name/position). group не
// меняется. Nil-поле означает «не менять».
type UpdateStatusParams struct {
	Name     *string
	Position *int
}

// CreateTeam создаёт команду вместе с 5 дефолтными статусами (по одному на
// группу) и добавляет создателя участником-owner — всё в одной транзакции.
// Доступно любому участнику workspace (проверено в middleware LoadWorkspace).
// key нормализуется (upper/trim); дубль key per workspace → errs.AlreadyExists.
func (s *TeamService) CreateTeam(ctx context.Context, actor model.WorkspaceMember, params CreateTeamParams) (model.Team, error) {
	key := normalizeTeamKey(params.Key)
	if key == "" {
		return model.Team{}, errs.Validation("invalid team key", errs.FieldError{
			Field:   "key",
			Reason:  "invalid",
			Message: "must contain alphanumeric characters",
		})
	}

	var team model.Team
	err := s.repos.WithTx(ctx, func(tx repository.Store) error {
		var err error
		team, err = tx.Team().Create(ctx, model.Team{
			WorkspaceID: actor.WorkspaceID,
			Name:        params.Name,
			Description: params.Description,
			Key:         key,
		})
		if err != nil {
			return err
		}

		for _, ds := range defaultStatuses {
			if _, err := tx.Status().Create(ctx, model.Status{
				TeamID:   team.ID,
				Name:     ds.name,
				Group:    ds.group,
				Position: 0,
			}); err != nil {
				return err
			}
		}

		_, err = tx.Team().AddMember(ctx, model.TeamMember{
			TeamID: team.ID,
			UserID: actor.UserID,
			Role:   model.TeamRoleOwner,
		})
		return err
	})
	if err != nil {
		return model.Team{}, err
	}
	return team, nil
}

// ListTeams возвращает команды workspace. Membership проверен в middleware.
func (s *TeamService) ListTeams(ctx context.Context, workspaceID string) ([]model.Team, error) {
	return s.repos.Team().List(ctx, workspaceID)
}

// GetTeam возвращает команду workspace по id. Нет команды → errs.NotFound.
func (s *TeamService) GetTeam(ctx context.Context, workspaceID, teamID string) (model.Team, error) {
	return s.repos.Team().Get(ctx, workspaceID, teamID)
}

// UpdateTeam обновляет name/description/key команды. Доступно участнику команды.
// key нормализуется; дубль key per workspace → errs.AlreadyExists.
func (s *TeamService) UpdateTeam(ctx context.Context, actor model.WorkspaceMember, teamID string, params UpdateTeamParams) (model.Team, error) {
	team, err := s.repos.Team().Get(ctx, actor.WorkspaceID, teamID)
	if err != nil {
		return model.Team{}, err
	}
	if err := s.requireTeamMember(ctx, teamID, actor.UserID); err != nil {
		return model.Team{}, err
	}

	if params.Name != nil {
		team.Name = *params.Name
	}
	if params.Key != nil {
		key := normalizeTeamKey(*params.Key)
		if key == "" {
			return model.Team{}, errs.Validation("invalid team key", errs.FieldError{
				Field:   "key",
				Reason:  "invalid",
				Message: "must contain alphanumeric characters",
			})
		}
		team.Key = key
	}
	if params.Description != nil {
		team.Description = params.Description
	}

	return s.repos.Team().Update(ctx, team)
}

// DeleteTeam мягко удаляет команду. Доступно только owner/admin workspace.
func (s *TeamService) DeleteTeam(ctx context.Context, actor model.WorkspaceMember, teamID string) error {
	if !canManageWorkspace(actor.Role) {
		return errs.Forbidden("insufficient permissions")
	}
	team, err := s.repos.Team().Get(ctx, actor.WorkspaceID, teamID)
	if err != nil {
		return err
	}
	return s.repos.Team().Delete(ctx, team.ID)
}

// ListMembers возвращает участников команды. Доступно участнику workspace.
func (s *TeamService) ListMembers(ctx context.Context, workspaceID, teamID string) ([]model.TeamMember, error) {
	if _, err := s.repos.Team().Get(ctx, workspaceID, teamID); err != nil {
		return nil, err
	}
	return s.repos.Team().ListMembers(ctx, teamID)
}

// AddMember добавляет участника в команду. Доступно участнику команды. Кандидат
// обязан быть активным участником workspace (team members ⊆ workspace members),
// иначе errs.Validation. Пустая роль трактуется как member.
func (s *TeamService) AddMember(ctx context.Context, actor model.WorkspaceMember, teamID string, params AddTeamMemberParams) (model.TeamMember, error) {
	team, err := s.repos.Team().Get(ctx, actor.WorkspaceID, teamID)
	if err != nil {
		return model.TeamMember{}, err
	}
	if err := s.requireTeamMember(ctx, teamID, actor.UserID); err != nil {
		return model.TeamMember{}, err
	}

	if _, err := s.repos.Workspace().GetMember(ctx, team.WorkspaceID, params.UserID); err != nil {
		if isNotFound(err) {
			return model.TeamMember{}, errs.Validation("user is not a workspace member", errs.FieldError{
				Field:   "user_id",
				Reason:  "invalid",
				Message: "must be an active workspace member",
			})
		}
		return model.TeamMember{}, err
	}

	role := params.Role
	if role == "" {
		role = model.TeamRoleMember
	}
	return s.repos.Team().AddMember(ctx, model.TeamMember{
		TeamID: teamID,
		UserID: params.UserID,
		Role:   role,
	})
}

// RemoveMember мягко снимает участника с команды. Доступно участнику команды.
func (s *TeamService) RemoveMember(ctx context.Context, actor model.WorkspaceMember, teamID, userID string) error {
	if _, err := s.repos.Team().Get(ctx, actor.WorkspaceID, teamID); err != nil {
		return err
	}
	if err := s.requireTeamMember(ctx, teamID, actor.UserID); err != nil {
		return err
	}
	return s.repos.Team().RemoveMember(ctx, teamID, userID)
}

// ListStatuses возвращает статусы команды. Доступно участнику workspace.
func (s *TeamService) ListStatuses(ctx context.Context, workspaceID, teamID string) ([]model.Status, error) {
	if _, err := s.repos.Team().Get(ctx, workspaceID, teamID); err != nil {
		return nil, err
	}
	return s.repos.Status().List(ctx, teamID)
}

// CreateStatus создаёт статус в команде. Доступно участнику команды. Дубль
// (group, position) → errs.AlreadyExists.
func (s *TeamService) CreateStatus(ctx context.Context, actor model.WorkspaceMember, teamID string, params CreateStatusParams) (model.Status, error) {
	if _, err := s.repos.Team().Get(ctx, actor.WorkspaceID, teamID); err != nil {
		return model.Status{}, err
	}
	if err := s.requireTeamMember(ctx, teamID, actor.UserID); err != nil {
		return model.Status{}, err
	}
	return s.repos.Status().Create(ctx, model.Status{
		TeamID:   teamID,
		Name:     params.Name,
		Group:    params.Group,
		Position: params.Position,
	})
}

// UpdateStatus обновляет name/position статуса. Доступно участнику команды.
func (s *TeamService) UpdateStatus(ctx context.Context, actor model.WorkspaceMember, teamID, statusID string, params UpdateStatusParams) (model.Status, error) {
	if _, err := s.repos.Team().Get(ctx, actor.WorkspaceID, teamID); err != nil {
		return model.Status{}, err
	}
	if err := s.requireTeamMember(ctx, teamID, actor.UserID); err != nil {
		return model.Status{}, err
	}

	status, err := s.repos.Status().Get(ctx, teamID, statusID)
	if err != nil {
		return model.Status{}, err
	}
	if params.Name != nil {
		status.Name = *params.Name
	}
	if params.Position != nil {
		status.Position = *params.Position
	}
	return s.repos.Status().Update(ctx, status)
}

// DeleteStatus мягко удаляет статус. Доступно участнику команды. Нельзя удалить
// последний статус в группе (правило «≥1 статус на группу») → errs.Validation.
func (s *TeamService) DeleteStatus(ctx context.Context, actor model.WorkspaceMember, teamID, statusID string) error {
	if _, err := s.repos.Team().Get(ctx, actor.WorkspaceID, teamID); err != nil {
		return err
	}
	if err := s.requireTeamMember(ctx, teamID, actor.UserID); err != nil {
		return err
	}

	status, err := s.repos.Status().Get(ctx, teamID, statusID)
	if err != nil {
		return err
	}

	count, err := s.repos.Status().CountByGroup(ctx, teamID, status.Group)
	if err != nil {
		return err
	}
	if count <= 1 {
		return errs.Validation("cannot delete the last status in its group", errs.FieldError{
			Field:   "group",
			Reason:  "invariant",
			Message: "each status group must keep at least one status",
		})
	}

	return s.repos.Status().Delete(ctx, statusID)
}

// requireTeamMember проверяет, что пользователь — активный участник команды.
// Нет membership → errs.Forbidden.
func (s *TeamService) requireTeamMember(ctx context.Context, teamID, userID string) error {
	if _, err := s.repos.Team().GetMember(ctx, teamID, userID); err != nil {
		if isNotFound(err) {
			return errs.Forbidden("not a team member")
		}
		return err
	}
	return nil
}

// normalizeTeamKey приводит team key к каноничному виду (upper-case, без
// окружающих пробелов). Используется как идентификатор в issue (ENG-123).
func normalizeTeamKey(key string) string {
	return strings.ToUpper(strings.TrimSpace(key))
}
