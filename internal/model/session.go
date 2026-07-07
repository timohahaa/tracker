package model

import "time"

// Session — web-сессия (schema core.sessions). Hard delete при logout.
type Session struct {
	ID        string    `db:"id"`
	UserID    string    `db:"user_id"`
	ExpiresAt time.Time `db:"expires_at"`
	CreatedAt time.Time `db:"created_at"`
}
