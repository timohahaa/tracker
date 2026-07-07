package service

import (
	"github.com/timohahaa/tracker/config"
	"github.com/timohahaa/tracker/internal/repository"
)

// AuthService — регистрация, логин, сессии, инвайты.
type AuthService struct {
	repos *repository.Repositories
	cfg   config.Config
}
