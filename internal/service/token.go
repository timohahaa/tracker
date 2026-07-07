package service

import "github.com/timohahaa/tracker/internal/repository"

// TokenService — управление API-токенами.
type TokenService struct {
	repos *repository.Repositories
}
