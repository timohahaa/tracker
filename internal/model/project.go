package model

import "time"

// ProjectStatus — статус проекта.
type ProjectStatus string

const (
	ProjectStatusPlanned    ProjectStatus = "planned"
	ProjectStatusInProgress ProjectStatus = "in_progress"
	ProjectStatusCompleted  ProjectStatus = "completed"
	ProjectStatusCanceled   ProjectStatus = "canceled"
)

// Project — проект (schema tracker.projects). Может охватывать несколько команд.
type Project struct {
	ID          string        `db:"id"`
	WorkspaceID string        `db:"workspace_id"`
	LeadID      *string       `db:"lead_id"`
	Name        string        `db:"name"`
	Description *string       `db:"description"`
	Status      ProjectStatus `db:"status"`
	CreatedAt   time.Time     `db:"created_at"`
	UpdatedAt   time.Time     `db:"updated_at"`
	DeletedAt   *time.Time    `db:"deleted_at"`
}

// ProjectTeam — связь проект ↔ команда (schema tracker.project_teams).
type ProjectTeam struct {
	ProjectID string     `db:"project_id"`
	TeamID    string     `db:"team_id"`
	CreatedAt time.Time  `db:"created_at"`
	DeletedAt *time.Time `db:"deleted_at"`
}
