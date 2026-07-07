package repository

import "github.com/jmoiron/sqlx"

// TokenRepo — доступ к core.api_tokens.
type TokenRepo struct {
	db *sqlx.DB
}
