package model

import "time"

// Label — метка (schema tracker.label). team_id == nil → workspace-level.
type Label struct {
	ID          string     `db:"id"`
	WorkspaceID string     `db:"workspace_id"`
	TeamID      *string    `db:"team_id"`
	Name        string     `db:"name"`
	Color       string     `db:"color"`
	CreatedAt   time.Time  `db:"created_at"`
	UpdatedAt   time.Time  `db:"updated_at"`
	DeletedAt   *time.Time `db:"deleted_at"`
}

// IssueLabel — связь задача ↔ метка (schema tracker.issue_label).
type IssueLabel struct {
	IssueID   string     `db:"issue_id"`
	LabelID   string     `db:"label_id"`
	CreatedAt time.Time  `db:"created_at"`
	DeletedAt *time.Time `db:"deleted_at"`
}
