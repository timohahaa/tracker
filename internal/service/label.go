package service

import "github.com/timohahaa/tracker/internal/repository"

// LabelService — CRUD меток (workspace- и team-level).
type LabelService struct {
	repos *repository.Repositories
}
