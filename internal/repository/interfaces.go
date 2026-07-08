package repository

import (
	"context"
	"time"

	"github.com/timohahaa/tracker/internal/model"
)

// Интерфейсы репозиториев — шов между service и repository. Конкретные *XRepo
// реализуют их структурно; сервисы работают через интерфейсы, что позволяет
// подменять реализацию в unit-тестах. Растут вместе с методами репозиториев.

// Store — шов между service и repository для всего набора репозиториев. Сервисы
// держат Store (а не конкретный *Repositories), поэтому в unit-тестах его можно
// целиком подменить моком, включая транзакционный путь через WithTx. Колбэк
// WithTx тоже получает Store — внутри транзакции нет конкретных типов, что делает
// транзакционную логику полностью мокаемой. *Repositories реализует Store.
type Store interface {
	User() UserRepository
	Session() SessionRepository
	Token() TokenRepository
	Workspace() WorkspaceRepository
	Team() *TeamRepo
	Status() *StatusRepo
	Issue() *IssueRepo
	Project() *ProjectRepo
	Label() *LabelRepo
	Integration() *IntegrationRepo
	Attachment() *AttachmentRepo
	Invite() *InviteRepo

	// WithTx выполняет fn в одной транзакции; переданный Store к ней привязан.
	WithTx(ctx context.Context, fn func(Store) error) error
}

// UserRepository — доступ к core.user.
type UserRepository interface {
	Create(ctx context.Context, u model.User) (model.User, error)
	GetByEmail(ctx context.Context, email string) (model.User, error)
	GetByID(ctx context.Context, id string) (model.User, error)
}

// SessionRepository — доступ к core.session.
type SessionRepository interface {
	Create(ctx context.Context, userID string, expiresAt time.Time) (model.Session, error)
	GetByID(ctx context.Context, id string) (model.Session, error)
	Delete(ctx context.Context, id string) error
	Extend(ctx context.Context, id string, expiresAt time.Time) error
}

// TokenRepository — доступ к core.api_token.
type TokenRepository interface {
	GetByID(ctx context.Context, id string) (model.APIToken, error)
	UpdateLastUsed(ctx context.Context, id string) error
}

// WorkspaceRepository — доступ к org.workspace и org.workspace_member.
type WorkspaceRepository interface {
	Create(ctx context.Context, ws model.Workspace) (model.Workspace, error)
	SlugTaken(ctx context.Context, slug string) (bool, error)
	CreateMember(ctx context.Context, m model.WorkspaceMember) (model.WorkspaceMember, error)
	GetByID(ctx context.Context, id string) (model.Workspace, error)
	Update(ctx context.Context, ws model.Workspace) (model.Workspace, error)
	ListMembers(ctx context.Context, workspaceID string) ([]model.WorkspaceMemberDetails, error)
	GetMember(ctx context.Context, workspaceID, userID string) (model.WorkspaceMember, error)
	UpdateMember(ctx context.Context, m model.WorkspaceMember) (model.WorkspaceMember, error)
	DeleteMember(ctx context.Context, workspaceID, userID string) error
	CountOwners(ctx context.Context, workspaceID string) (int, error)
}

// Компайл-тайм проверки соответствия конкретных репозиториев интерфейсам.
var (
	_ UserRepository      = (*UserRepo)(nil)
	_ SessionRepository   = (*SessionRepo)(nil)
	_ TokenRepository     = (*TokenRepo)(nil)
	_ WorkspaceRepository = (*WorkspaceRepo)(nil)
	_ Store               = (*Repositories)(nil)
)
