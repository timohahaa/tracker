package repository

import "github.com/jmoiron/sqlx"

// LabelRepo — доступ к tracker.labels.
type LabelRepo struct {
	db *sqlx.DB
}
