package model

import "time"

// APIToken — токен для API/MCP (schema core.api_tokens). ID = Bearer token.
type APIToken struct {
	ID          string     `db:"id"`
	UserID      string     `db:"user_id"`
	WorkspaceID string     `db:"workspace_id"`
	Name        string     `db:"name"`
	LastUsedAt  *time.Time `db:"last_used_at"`
	CreatedAt   time.Time  `db:"created_at"`
	DeletedAt   *time.Time `db:"deleted_at"`
}
