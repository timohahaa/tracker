package repository

import "github.com/jmoiron/sqlx"

// TeamRepo — доступ к org.teams и org.team_members.
type TeamRepo struct {
	db *sqlx.DB
}
