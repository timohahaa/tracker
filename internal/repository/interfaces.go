package repository

import (
	"context"
	"time"

	"github.com/timohahaa/tracker/internal/model"
)

//go:generate go tool mockgen -destination=mocks/mocks.go -package=repomocks github.com/timohahaa/tracker/internal/repository Store,UserRepository,SessionRepository,TokenRepository,WorkspaceRepository,TeamRepository,StatusRepository,IssueRepository

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
	Team() TeamRepository
	Status() StatusRepository
	Issue() IssueRepository
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

// TeamRepository — доступ к org.team и org.team_member.
type TeamRepository interface {
	Create(ctx context.Context, t model.Team) (model.Team, error)
	Get(ctx context.Context, workspaceID, teamID string) (model.Team, error)
	List(ctx context.Context, workspaceID string) ([]model.Team, error)
	Update(ctx context.Context, t model.Team) (model.Team, error)
	Delete(ctx context.Context, teamID string) error

	AddMember(ctx context.Context, m model.TeamMember) (model.TeamMember, error)
	ListMembers(ctx context.Context, teamID string) ([]model.TeamMember, error)
	GetMember(ctx context.Context, teamID, userID string) (model.TeamMember, error)
	RemoveMember(ctx context.Context, teamID, userID string) error
}

// StatusRepository — доступ к tracker.status.
type StatusRepository interface {
	Create(ctx context.Context, s model.Status) (model.Status, error)
	Get(ctx context.Context, teamID, statusID string) (model.Status, error)
	List(ctx context.Context, teamID string) ([]model.Status, error)
	Update(ctx context.Context, s model.Status) (model.Status, error)
	Delete(ctx context.Context, statusID string) error
	CountByGroup(ctx context.Context, teamID string, group model.StatusGroup) (int, error)
}

// IssueRepository — доступ к tracker.issue. Identifier (ENG-123) не хранится:
// вычисляется при чтении из org.team.key + issue.issue_number. Инкремент номера
// (IncrementIssueNumber) обновляет org.team, но живёт здесь как часть выдачи
// identifier и вызывается в одной транзакции с Create.
type IssueRepository interface {
	Create(ctx context.Context, i model.Issue) (model.Issue, error)
	IncrementIssueNumber(ctx context.Context, teamID string) (int, error)
	GetByID(ctx context.Context, workspaceID, issueID string) (model.IssueDetails, error)
	GetByIdentifier(ctx context.Context, workspaceID, teamKey string, issueNumber int) (model.IssueDetails, error)
	Update(ctx context.Context, i model.Issue) (model.Issue, error)
	Delete(ctx context.Context, issueID string) error

	// ProjectInWorkspace — read-only проверка, что project активен и принадлежит
	// workspace. Позволяет валидировать project_id в issue-сервисе, не затрагивая
	// write-методы (будущего) project-слайса.
	ProjectInWorkspace(ctx context.Context, workspaceID, projectID string) (bool, error)
}

// Компайл-тайм проверки соответствия конкретных репозиториев интерфейсам.
var (
	_ UserRepository      = (*UserRepo)(nil)
	_ SessionRepository   = (*SessionRepo)(nil)
	_ TokenRepository     = (*TokenRepo)(nil)
	_ WorkspaceRepository = (*WorkspaceRepo)(nil)
	_ TeamRepository      = (*TeamRepo)(nil)
	_ StatusRepository    = (*StatusRepo)(nil)
	_ IssueRepository     = (*IssueRepo)(nil)
	_ Store               = (*Repositories)(nil)
)
