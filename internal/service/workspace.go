package service

import (
	"context"

	"github.com/timohahaa/tracker/internal/errs"
	"github.com/timohahaa/tracker/internal/model"
	"github.com/timohahaa/tracker/internal/repository"
)

// WorkspaceService — настройки workspace, участники, роли.
type WorkspaceService struct {
	repos *repository.Repositories
}

// UpdateSettingsParams — частичное обновление настроек workspace (граница
// service). Nil-поле означает «не менять».
type UpdateSettingsParams struct {
	Name *string
	Slug *string
}

// UpdateMemberParams — частичное обновление участника (граница service).
// Nil-поле означает «не менять».
type UpdateMemberParams struct {
	Role     *model.WorkspaceRole
	Username *string
	Title    *string
}

// Load загружает workspace и membership текущего пользователя. Используется
// middleware LoadWorkspace. Нет workspace или пользователь не участник →
// errs.NotFound (существование workspace не раскрываем не-участникам).
func (s *WorkspaceService) Load(ctx context.Context, workspaceID, userID string) (model.Workspace, model.WorkspaceMember, error) {
	ws, err := s.repos.Workspace.GetByID(ctx, workspaceID)
	if err != nil {
		return model.Workspace{}, model.WorkspaceMember{}, err
	}

	member, err := s.repos.Workspace.GetMember(ctx, workspaceID, userID)
	if err != nil {
		if isNotFound(err) {
			return model.Workspace{}, model.WorkspaceMember{}, errs.NotFound("workspace not found")
		}
		return model.Workspace{}, model.WorkspaceMember{}, err
	}

	return ws, member, nil
}

// UpdateSettings обновляет настройки workspace (name/slug). Доступно только
// owner/admin. slug нормализуется через slugify.
func (s *WorkspaceService) UpdateSettings(ctx context.Context, actor model.WorkspaceMember, params UpdateSettingsParams) (model.Workspace, error) {
	if !canManageWorkspace(actor.Role) {
		return model.Workspace{}, errs.Forbidden("insufficient permissions")
	}

	ws, err := s.repos.Workspace.GetByID(ctx, actor.WorkspaceID)
	if err != nil {
		return model.Workspace{}, err
	}

	if params.Name != nil {
		ws.Name = *params.Name
	}
	if params.Slug != nil {
		slug := slugify(*params.Slug)
		if slug == "" {
			return model.Workspace{}, errs.Validation("invalid slug", errs.FieldError{
				Field:   "slug",
				Reason:  "invalid",
				Message: "must contain url-safe characters",
			})
		}
		ws.Slug = slug
	}

	return s.repos.Workspace.Update(ctx, ws)
}

// ListMembers возвращает активных участников workspace. Membership проверен в
// middleware LoadWorkspace.
func (s *WorkspaceService) ListMembers(ctx context.Context, workspaceID string) ([]model.WorkspaceMemberDetails, error) {
	return s.repos.Workspace.ListMembers(ctx, workspaceID)
}

// UpdateMember обновляет role/username/title участника. Доступно только
// owner/admin. Роль owner нельзя назначить через этот эндпоинт (owner — только
// создатель). Последнего owner нельзя понизить. username уникален per workspace
// → errs.AlreadyExists (через errs.DB).
func (s *WorkspaceService) UpdateMember(ctx context.Context, actor model.WorkspaceMember, targetUserID string, params UpdateMemberParams) (model.WorkspaceMember, error) {
	if !canManageWorkspace(actor.Role) {
		return model.WorkspaceMember{}, errs.Forbidden("insufficient permissions")
	}

	target, err := s.repos.Workspace.GetMember(ctx, actor.WorkspaceID, targetUserID)
	if err != nil {
		return model.WorkspaceMember{}, err
	}

	if params.Role != nil {
		if *params.Role == model.WorkspaceRoleOwner {
			return model.WorkspaceMember{}, errs.Forbidden("cannot assign owner role")
		}
		// Понижение последнего owner недопустимо.
		if target.Role == model.WorkspaceRoleOwner && *params.Role != model.WorkspaceRoleOwner {
			owners, err := s.repos.Workspace.CountOwners(ctx, actor.WorkspaceID)
			if err != nil {
				return model.WorkspaceMember{}, err
			}
			if owners <= 1 {
				return model.WorkspaceMember{}, errs.Forbidden("cannot demote the last owner")
			}
		}
		target.Role = *params.Role
	}
	if params.Username != nil {
		target.Username = *params.Username
	}
	if params.Title != nil {
		target.Title = params.Title
	}

	return s.repos.Workspace.UpdateMember(ctx, target)
}

// RemoveMember мягко удаляет участника. Доступно только owner/admin. Последнего
// owner удалить нельзя.
func (s *WorkspaceService) RemoveMember(ctx context.Context, actor model.WorkspaceMember, targetUserID string) error {
	if !canManageWorkspace(actor.Role) {
		return errs.Forbidden("insufficient permissions")
	}

	target, err := s.repos.Workspace.GetMember(ctx, actor.WorkspaceID, targetUserID)
	if err != nil {
		return err
	}

	if target.Role == model.WorkspaceRoleOwner {
		owners, err := s.repos.Workspace.CountOwners(ctx, actor.WorkspaceID)
		if err != nil {
			return err
		}
		if owners <= 1 {
			return errs.Forbidden("cannot remove the last owner")
		}
	}

	return s.repos.Workspace.DeleteMember(ctx, actor.WorkspaceID, targetUserID)
}

// canManageWorkspace сообщает, может ли роль управлять workspace/участниками
// (owner/admin). member — нет.
func canManageWorkspace(role model.WorkspaceRole) bool {
	return role == model.WorkspaceRoleOwner || role == model.WorkspaceRoleAdmin
}
