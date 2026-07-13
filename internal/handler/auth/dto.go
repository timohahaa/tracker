package auth

import (
	"time"

	"github.com/timohahaa/tracker/internal/model"
	"github.com/timohahaa/tracker/internal/service"
)

// registerRequest — тело POST /api/auth/register. slug workspace и username
// владельца генерируются на сервере, клиент их не передаёт.
type registerRequest struct {
	Email         string `json:"email" validate:"required,email"`
	Password      string `json:"password" validate:"required,min=8"`
	DisplayName   string `json:"display_name" validate:"required"`
	WorkspaceName string `json:"workspace_name" validate:"required"`
}

// loginRequest — тело POST /api/auth/login.
type loginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

// userResponse — публичное представление пользователя (без password_hash).
type userResponse struct {
	ID          string           `json:"id"`
	Email       string           `json:"email"`
	DisplayName string           `json:"display_name"`
	Settings    settingsResponse `json:"settings"`
	CreatedAt   time.Time        `json:"created_at"`
}

// settingsResponse — глобальные настройки пользователя.
type settingsResponse struct {
	CurrentWorkspaceID *string `json:"current_workspace_id,omitempty"`
}

// updateMeRequest — тело PATCH /api/auth/me. Поля опциональны (partial update):
// отсутствие поля означает «не менять».
type updateMeRequest struct {
	CurrentWorkspaceID *string `json:"current_workspace_id" validate:"omitempty,uuid"`
}

// workspaceResponse — публичное представление workspace.
type workspaceResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	CreatedAt time.Time `json:"created_at"`
}

// registerResponse — ответ регистрации: пользователь и созданный workspace.
type registerResponse struct {
	User      userResponse      `json:"user"`
	Workspace workspaceResponse `json:"workspace"`
}

// loginResponse — ответ логина: текущий пользователь.
type loginResponse struct {
	User userResponse `json:"user"`
}

func (r registerRequest) toParams() service.RegisterParams {
	return service.RegisterParams{
		Email:         r.Email,
		Password:      r.Password,
		DisplayName:   r.DisplayName,
		WorkspaceName: r.WorkspaceName,
	}
}

func toUserResponse(u model.User) userResponse {
	return userResponse{
		ID:          u.ID,
		Email:       u.Email,
		DisplayName: u.DisplayName,
		Settings:    settingsResponse{CurrentWorkspaceID: u.Settings.CurrentWorkspaceID},
		CreatedAt:   u.CreatedAt,
	}
}

func toWorkspaceResponse(w model.Workspace) workspaceResponse {
	return workspaceResponse{
		ID:        w.ID,
		Name:      w.Name,
		Slug:      w.Slug,
		CreatedAt: w.CreatedAt,
	}
}
