package model

import "time"

// TeamRole — роль участника команды.
type TeamRole string

const (
	TeamRoleOwner  TeamRole = "owner"
	TeamRoleMember TeamRole = "member"
)

// Team — команда (schema org.teams).
type Team struct {
	ID              string     `db:"id"`
	WorkspaceID     string     `db:"workspace_id"`
	Name            string     `db:"name"`
	Description     *string    `db:"description"`
	Key             string     `db:"key"`
	NextIssueNumber int        `db:"next_issue_number"`
	CreatedAt       time.Time  `db:"created_at"`
	UpdatedAt       time.Time  `db:"updated_at"`
	DeletedAt       *time.Time `db:"deleted_at"`
}

// TeamMember — участник команды (schema org.team_members).
type TeamMember struct {
	TeamID    string     `db:"team_id"`
	UserID    string     `db:"user_id"`
	Role      TeamRole   `db:"role"`
	CreatedAt time.Time  `db:"created_at"`
	UpdatedAt time.Time  `db:"updated_at"`
	DeletedAt *time.Time `db:"deleted_at"`
}
