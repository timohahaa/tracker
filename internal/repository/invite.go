package repository

import "github.com/jmoiron/sqlx"

// InviteRepo — доступ к org.invites.
type InviteRepo struct {
	db *sqlx.DB
}
