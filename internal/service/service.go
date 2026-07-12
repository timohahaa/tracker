package service

import (
	"github.com/timohahaa/tracker/config"
	"github.com/timohahaa/tracker/internal/repository"
)

// Services — агрегатор бизнес-логики. Собирается один раз в app.
type Services struct {
	Auth      *AuthService
	Workspace *WorkspaceService
	Team      *TeamService
	Issue     *IssueService
	Project   *ProjectService
	Label     *LabelService
	Token     *TokenService
	GitLab    *GitLabService
}

// New собирает Services поверх репозиториев и конфига.
func New(repos repository.Store, cfg config.Config) *Services {
	return &Services{
		Auth:      &AuthService{repos: repos, cfg: cfg},
		Workspace: &WorkspaceService{repos: repos},
		Team:      &TeamService{repos: repos},
		Issue:     &IssueService{repos: repos},
		Project:   &ProjectService{repos: repos},
		Label:     &LabelService{repos: repos},
		Token:     &TokenService{repos: repos},
		GitLab:    &GitLabService{repos: repos, gitlab: newGitLabClient()},
	}
}
