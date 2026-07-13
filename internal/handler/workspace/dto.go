package workspace

import (
	"time"

	"github.com/timohahaa/tracker/internal/model"
	"github.com/timohahaa/tracker/internal/service"
)

// updateWorkspaceRequest — тело PATCH /api/workspaces/{workspaceID}. Все поля
// опциональны (partial update): nil означает «не менять».
type updateWorkspaceRequest struct {
	Name *string `json:"name" validate:"omitempty,min=1"`
	Slug *string `json:"slug" validate:"omitempty,min=1"`
}

// updateMemberRequest — тело PATCH /api/workspaces/{workspaceID}/members/{userID}.
// Все поля опциональны. Роль owner нельзя назначить (сервис вернёт 403).
type updateMemberRequest struct {
	Role     *string `json:"role" validate:"omitempty,oneof=owner admin member"`
	Username *string `json:"username" validate:"omitempty,min=1"`
	Title    *string `json:"title"`
}

// workspaceResponse — публичное представление workspace.
type workspaceResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// workspaceMineResponse — элемент списка «мои workspace»: workspace + роль
// текущего пользователя.
type workspaceMineResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// workspacesResponse — ответ GET /api/workspaces.
type workspacesResponse struct {
	Data []workspaceMineResponse `json:"data"`
}

// memberResponse — публичное представление участника workspace.
type memberResponse struct {
	UserID      string    `json:"user_id"`
	Email       string    `json:"email,omitempty"`
	DisplayName string    `json:"display_name,omitempty"`
	Role        string    `json:"role"`
	Username    string    `json:"username"`
	Title       *string   `json:"title,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// membersResponse — ответ списка участников.
type membersResponse struct {
	Data []memberResponse `json:"data"`
}

func (r updateWorkspaceRequest) toParams() service.UpdateSettingsParams {
	return service.UpdateSettingsParams{
		Name: r.Name,
		Slug: r.Slug,
	}
}

func (r updateMemberRequest) toParams() service.UpdateMemberParams {
	p := service.UpdateMemberParams{
		Username: r.Username,
		Title:    r.Title,
	}
	if r.Role != nil {
		role := model.WorkspaceRole(*r.Role)
		p.Role = &role
	}
	return p
}

func toWorkspaceResponse(w model.Workspace) workspaceResponse {
	return workspaceResponse{
		ID:        w.ID,
		Name:      w.Name,
		Slug:      w.Slug,
		CreatedAt: w.CreatedAt,
		UpdatedAt: w.UpdatedAt,
	}
}

func toMemberResponse(m model.WorkspaceMember, email, displayName string) memberResponse {
	return memberResponse{
		UserID:      m.UserID,
		Email:       email,
		DisplayName: displayName,
		Role:        string(m.Role),
		Username:    m.Username,
		Title:       m.Title,
		CreatedAt:   m.CreatedAt,
	}
}

func toWorkspacesResponse(list []model.WorkspaceWithRole) workspacesResponse {
	data := make([]workspaceMineResponse, 0, len(list))
	for _, w := range list {
		data = append(data, workspaceMineResponse{
			ID:        w.ID,
			Name:      w.Name,
			Slug:      w.Slug,
			Role:      string(w.Role),
			CreatedAt: w.CreatedAt,
			UpdatedAt: w.UpdatedAt,
		})
	}
	return workspacesResponse{Data: data}
}

func toMembersResponse(members []model.WorkspaceMemberDetails) membersResponse {
	data := make([]memberResponse, 0, len(members))
	for _, m := range members {
		data = append(data, toMemberResponse(m.WorkspaceMember, m.Email, m.DisplayName))
	}
	return membersResponse{Data: data}
}
