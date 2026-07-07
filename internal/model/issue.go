package model

import "time"

// Priority — приоритет задачи.
type Priority string

const (
	PriorityNone   Priority = "none"
	PriorityUrgent Priority = "urgent"
	PriorityHigh   Priority = "high"
	PriorityMedium Priority = "medium"
	PriorityLow    Priority = "low"
)

// Issue — задача (schema tracker.issues). Write-модель, 1:1 с таблицей.
// Identifier (ENG-123) не хранится — вычисляется из team.key + issue_number.
type Issue struct {
	ID          string     `db:"id"`
	TeamID      string     `db:"team_id"`
	ProjectID   *string    `db:"project_id"`
	StatusID    string     `db:"status_id"`
	CreatorID   *string    `db:"creator_id"`
	Title       string     `db:"title"`
	Description *string    `db:"description"`
	Priority    Priority   `db:"priority"`
	DueDate     *time.Time `db:"due_date"`
	IssueNumber int        `db:"issue_number"`
	CreatedAt   time.Time  `db:"created_at"`
	UpdatedAt   time.Time  `db:"updated_at"`
	DeletedAt   *time.Time `db:"deleted_at"`
}

// IssueAssignee — участник задачи (schema tracker.issue_assignees).
// is_primary == ответственный (максимум один).
type IssueAssignee struct {
	IssueID   string     `db:"issue_id"`
	UserID    string     `db:"user_id"`
	IsPrimary bool       `db:"is_primary"`
	CreatedAt time.Time  `db:"created_at"`
	UpdatedAt time.Time  `db:"updated_at"`
	DeletedAt *time.Time `db:"deleted_at"`
}

// IssueDetails — read-модель задачи с денормализованными полями (результат JOIN).
type IssueDetails struct {
	Issue
	Identifier string          `db:"identifier"`
	TeamKey    string          `db:"team_key"`
	Assignees  []IssueAssignee `db:"-"`
	Labels     []Label         `db:"-"`
}
