package integration

import (
	"time"

	"github.com/timohahaa/tracker/internal/service"
)

// setupGitLabRequest — тело POST /integrations/gitlab. base_url — валидный URL;
// access_token/webhook_secret обязательны. Секреты в ответ не возвращаются.
type setupGitLabRequest struct {
	BaseURL       string `json:"base_url" validate:"required,url"`
	AccessToken   string `json:"access_token" validate:"required"`
	WebhookSecret string `json:"webhook_secret" validate:"required"`
}

// gitlabSettingsResponse — non-secret настройки интеграции в ответе.
type gitlabSettingsResponse struct {
	BaseURL string `json:"base_url"`
}

// gitlabResponse — публичное представление интеграции GitLab. credentials
// (access_token/webhook_secret) намеренно отсутствуют. Когда интеграция не
// подключена, connected=false, а settings/таймстемпы пусты.
type gitlabResponse struct {
	Provider  string                 `json:"provider"`
	Connected bool                   `json:"connected"`
	Settings  gitlabSettingsResponse `json:"settings"`
	CreatedAt *time.Time             `json:"created_at,omitempty"`
	UpdatedAt *time.Time             `json:"updated_at,omitempty"`
}

func (r setupGitLabRequest) toParams() service.SetupGitLabParams {
	return service.SetupGitLabParams{
		BaseURL:       r.BaseURL,
		AccessToken:   r.AccessToken,
		WebhookSecret: r.WebhookSecret,
	}
}

func toGitLabResponse(gi service.GitLabIntegration) gitlabResponse {
	resp := gitlabResponse{
		Provider:  "gitlab",
		Connected: gi.Connected,
		Settings:  gitlabSettingsResponse{BaseURL: gi.Settings.BaseURL},
	}
	if gi.Connected {
		created := gi.Integration.CreatedAt
		updated := gi.Integration.UpdatedAt
		resp.CreatedAt = &created
		resp.UpdatedAt = &updated
	}
	return resp
}
