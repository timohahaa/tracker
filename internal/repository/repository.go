package repository

import "github.com/jmoiron/sqlx"

// Repositories — агрегатор всех репозиториев. Собирается один раз в app.
type Repositories struct {
	User        *UserRepo
	Session     *SessionRepo
	Token       *TokenRepo
	Workspace   *WorkspaceRepo
	Team        *TeamRepo
	Status      *StatusRepo
	Issue       *IssueRepo
	Project     *ProjectRepo
	Label       *LabelRepo
	Integration *IntegrationRepo
	Attachment  *AttachmentRepo
	Invite      *InviteRepo
}

// New собирает Repositories поверх одного пула соединений.
func New(db *sqlx.DB) *Repositories {
	return &Repositories{
		User:        &UserRepo{db: db},
		Session:     &SessionRepo{db: db},
		Token:       &TokenRepo{db: db},
		Workspace:   &WorkspaceRepo{db: db},
		Team:        &TeamRepo{db: db},
		Status:      &StatusRepo{db: db},
		Issue:       &IssueRepo{db: db},
		Project:     &ProjectRepo{db: db},
		Label:       &LabelRepo{db: db},
		Integration: &IntegrationRepo{db: db},
		Attachment:  &AttachmentRepo{db: db},
		Invite:      &InviteRepo{db: db},
	}
}
