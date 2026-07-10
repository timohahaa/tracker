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

// Repositories — агрегатор всех репозиториев и конкретная реализация Store.
// Собирается один раз в app. Поля приватны, доступ — через методы-аксессоры,
// чтобы весь набор ходил за интерфейсом Store (мокается в unit-тестах).
type Repositories struct {
	db *sqlx.DB // корневой пул — нужен для открытия транзакций в WithTx

	user        UserRepository
	session     SessionRepository
	token       TokenRepository
	workspace   WorkspaceRepository
	team        TeamRepository
	status      StatusRepository
	issue       IssueRepository
	project     ProjectRepository
	label       LabelRepository
	integration *IntegrationRepo
	attachment  *AttachmentRepo
	invite      *InviteRepo
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
		user:        &UserRepo{db: ext},
		session:     &SessionRepo{db: ext},
		token:       &TokenRepo{db: ext},
		workspace:   &WorkspaceRepo{db: ext},
		team:        &TeamRepo{db: ext},
		status:      &StatusRepo{db: ext},
		issue:       &IssueRepo{db: ext},
		project:     &ProjectRepo{db: ext},
		label:       &LabelRepo{db: ext},
		integration: &IntegrationRepo{db: ext},
		attachment:  &AttachmentRepo{db: ext},
		invite:      &InviteRepo{db: ext},
	}
}

// Аксессоры репозиториев — реализация Store. Для репозиториев с методами
// возвращают интерфейс (мокается), для пока пустых заглушек — конкретный тип
// (заменяется на интерфейс, когда у репозитория появятся методы).
func (r *Repositories) User() UserRepository           { return r.user }
func (r *Repositories) Session() SessionRepository     { return r.session }
func (r *Repositories) Token() TokenRepository         { return r.token }
func (r *Repositories) Workspace() WorkspaceRepository { return r.workspace }
func (r *Repositories) Team() TeamRepository           { return r.team }
func (r *Repositories) Status() StatusRepository       { return r.status }
func (r *Repositories) Issue() IssueRepository         { return r.issue }
func (r *Repositories) Project() ProjectRepository     { return r.project }
func (r *Repositories) Label() LabelRepository         { return r.label }
func (r *Repositories) Integration() *IntegrationRepo  { return r.integration }
func (r *Repositories) Attachment() *AttachmentRepo    { return r.attachment }
func (r *Repositories) Invite() *InviteRepo            { return r.invite }

// WithTx выполняет fn в рамках одной транзакции. Переданный Store привязан к
// транзакции: все запросы через него участвуют в одном BEGIN/COMMIT. При ошибке
// fn (или commit) транзакция откатывается. Ошибки БД маппятся errs.DB.
func (r *Repositories) WithTx(ctx context.Context, fn func(Store) error) error {
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
