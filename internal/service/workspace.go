package service

import "github.com/timohahaa/tracker/internal/repository"

// WorkspaceService — настройки workspace, участники, роли.
type WorkspaceService struct {
	repos *repository.Repositories
}
