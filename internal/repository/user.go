package repository

import "github.com/jmoiron/sqlx"

// UserRepo — доступ к core.users.
type UserRepo struct {
	db *sqlx.DB
}
