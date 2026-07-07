package model

import "time"

// SourceType — тип внешнего ресурса, привязанного к задаче.
type SourceType string

const (
	SourceTypeGitLabMR     SourceType = "gitlab_mr"
	SourceTypeGitLabCommit SourceType = "gitlab_commit"
)

// Attachment — привязка внешнего ресурса к задаче (schema tracker.attachments).
type Attachment struct {
	ID            string     `db:"id"`
	IssueID       string     `db:"issue_id"`
	IntegrationID *string    `db:"integration_id"`
	SourceType    SourceType `db:"source_type"`
	URL           string     `db:"url"`
	Title         string     `db:"title"`
	Metadata      []byte     `db:"metadata"` // jsonb
	CreatedAt     time.Time  `db:"created_at"`
	UpdatedAt     time.Time  `db:"updated_at"`
	DeletedAt     *time.Time `db:"deleted_at"`
}
