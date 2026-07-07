package service

import "github.com/timohahaa/tracker/internal/repository"

// ProjectService — CRUD проектов, связь с командами.
type ProjectService struct {
	repos *repository.Repositories
}
