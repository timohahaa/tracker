package model

import "time"

// Provider — тип внешней интеграции.
type Provider string

const (
	ProviderGitLab Provider = "gitlab"
)

// Integration — внешняя интеграция (schema integration.integrations).
// credentials никогда не отдаётся в API.
type Integration struct {
	ID          string     `db:"id"`
	WorkspaceID string     `db:"workspace_id"`
	Provider    Provider   `db:"provider"`
	Settings    []byte     `db:"settings"`    // jsonb, non-secret
	Credentials []byte     `db:"credentials"` // jsonb, secrets
	CreatedAt   time.Time  `db:"created_at"`
	UpdatedAt   time.Time  `db:"updated_at"`
	DeletedAt   *time.Time `db:"deleted_at"`
}
