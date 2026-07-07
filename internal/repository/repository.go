package repository

import (
	"context"

	"github.com/jmoiron/sqlx"

	"github.com/timohahaa/tracker/internal/errs"
)

// Ext — общий интерфейс для выполнения запросов. Ему удовлетворяют и *sqlx.DB
// (пул соединений), и *sqlx.Tx (транзакция), поэтому репозитории работают в обоих
// режимах без изменений. Для Get/Select используются пакетные sqlx.GetContext/
// sqlx.SelectContext поверх этого интерфейса.
type Ext = sqlx.ExtContext

// Repositories — агрегатор всех репозиториев. Собирается один раз в app.
type Repositories struct {
	db *sqlx.DB // корневой пул — нужен для открытия транзакций в WithTx

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
	r := newRepos(db)
	r.db = db
	return r
}

// newRepos собирает набор репозиториев поверх произвольного исполнителя запросов
// (пул или транзакция).
func newRepos(ext Ext) *Repositories {
	return &Repositories{
		User:        &UserRepo{db: ext},
		Session:     &SessionRepo{db: ext},
		Token:       &TokenRepo{db: ext},
		Workspace:   &WorkspaceRepo{db: ext},
		Team:        &TeamRepo{db: ext},
		Status:      &StatusRepo{db: ext},
		Issue:       &IssueRepo{db: ext},
		Project:     &ProjectRepo{db: ext},
		Label:       &LabelRepo{db: ext},
		Integration: &IntegrationRepo{db: ext},
		Attachment:  &AttachmentRepo{db: ext},
		Invite:      &InviteRepo{db: ext},
	}
}

// WithTx выполняет fn в рамках одной транзакции. Переданный *Repositories
// привязан к транзакции: все запросы через него участвуют в одном BEGIN/COMMIT.
// При ошибке fn (или commit) транзакция откатывается. Ошибки БД маппятся errs.DB.
func (r *Repositories) WithTx(ctx context.Context, fn func(*Repositories) error) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return errs.DB(err)
	}

	txRepos := newRepos(tx)
	txRepos.db = r.db

	if err := fn(txRepos); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := tx.Commit(); err != nil {
		return errs.DB(err)
	}
	return nil
}
