package repository

import "github.com/jmoiron/sqlx"

// SessionRepo — доступ к core.sessions.
type SessionRepo struct {
	db *sqlx.DB
}
