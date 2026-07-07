package repository

import "github.com/jmoiron/sqlx"

// WorkspaceRepo — доступ к org.workspaces и org.workspace_members.
type WorkspaceRepo struct {
	db *sqlx.DB
}
