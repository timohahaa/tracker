package repository

import "github.com/jmoiron/sqlx"

// IntegrationRepo — доступ к integration.integrations.
type IntegrationRepo struct {
	db *sqlx.DB
}
