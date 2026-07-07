package repository

import "github.com/jmoiron/sqlx"

// StatusRepo — доступ к tracker.statuses.
type StatusRepo struct {
	db *sqlx.DB
}
