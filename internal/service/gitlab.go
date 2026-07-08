package service

import "github.com/timohahaa/tracker/internal/repository"

// GitLabService — настройка интеграции, обработка webhook-событий, linkback.
type GitLabService struct {
	repos repository.Store
}
