package model

import "time"

// WorkspaceRole — роль участника в workspace.
type WorkspaceRole string

const (
	WorkspaceRoleOwner  WorkspaceRole = "owner"
	WorkspaceRoleAdmin  WorkspaceRole = "admin"
	WorkspaceRoleMember WorkspaceRole = "member"
)

// Workspace — организация (schema org.workspaces).
type Workspace struct {
	ID        string     `db:"id"`
	Name      string     `db:"name"`
	Slug      string     `db:"slug"`
	CreatedAt time.Time  `db:"created_at"`
	UpdatedAt time.Time  `db:"updated_at"`
	DeletedAt *time.Time `db:"deleted_at"`
}

// WorkspaceMember — участник workspace (schema org.workspace_members).
type WorkspaceMember struct {
	WorkspaceID string        `db:"workspace_id"`
	UserID      string        `db:"user_id"`
	Role        WorkspaceRole `db:"role"`
	Username    string        `db:"username"`
	Title       *string       `db:"title"`
	CreatedAt   time.Time     `db:"created_at"`
	UpdatedAt   time.Time     `db:"updated_at"`
	DeletedAt   *time.Time    `db:"deleted_at"`
}

// WorkspaceMemberDetails — участник + данные пользователя (JOIN core.user).
// Read-модель для списка участников: встраивает WorkspaceMember и добавляет
// email/display_name из core.user.
type WorkspaceMemberDetails struct {
	WorkspaceMember
	Email       string `db:"email"`
	DisplayName string `db:"display_name"`
}

// Invite — приглашение в workspace (schema org.invites).
type Invite struct {
	ID          string        `db:"id"`
	WorkspaceID string        `db:"workspace_id"`
	Email       string        `db:"email"`
	Role        WorkspaceRole `db:"role"`
	InvitedBy   string        `db:"invited_by"`
	Token       string        `db:"token"`
	ExpiresAt   time.Time     `db:"expires_at"`
	AcceptedAt  *time.Time    `db:"accepted_at"`
	CreatedAt   time.Time     `db:"created_at"`
	DeletedAt   *time.Time    `db:"deleted_at"`
}
