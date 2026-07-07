package repository

import "github.com/jmoiron/sqlx"

// AttachmentRepo — доступ к tracker.attachments.
type AttachmentRepo struct {
	db *sqlx.DB
}
