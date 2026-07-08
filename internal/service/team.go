package service

import "github.com/timohahaa/tracker/internal/repository"

// TeamService — команды, участники, статусы.
type TeamService struct {
	repos repository.Store
}
