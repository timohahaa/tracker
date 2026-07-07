package repository

import "github.com/jmoiron/sqlx"

// ProjectRepo — доступ к tracker.projects и project_teams.
type ProjectRepo struct {
	db *sqlx.DB
}
