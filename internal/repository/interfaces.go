package repository

import (
	"context"
	"time"

	"github.com/timohahaa/tracker/internal/model"
)

//go:generate go tool mockgen -destination=mocks/mocks.go -package=repomocks github.com/timohahaa/tracker/internal/repository Store,UserRepository,SessionRepository,TokenRepository,WorkspaceRepository,TeamRepository,StatusRepository,IssueRepository,ProjectRepository,LabelRepository

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
	Project() ProjectRepository
	Label() LabelRepository
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

	// List возвращает задачи workspace по фильтру с keyset-пагинацией. Read-only
	// (без транзакции). page.Limit — размер страницы; репозиторий выбирает
	// limit+1 строку, чтобы сервис мог вычислить has_more. Assignees/Labels не
	// заполняются (заход assignees).
	List(ctx context.Context, workspaceID string, filter model.IssueFilter, page model.Page) ([]model.IssueDetails, error)

	// ProjectInWorkspace — read-only проверка, что project активен и принадлежит
	// workspace. Позволяет валидировать project_id в issue-сервисе, не затрагивая
	// write-методы (будущего) project-слайса.
	ProjectInWorkspace(ctx context.Context, workspaceID, projectID string) (bool, error)

	// Assignees (tracker.issue_assignee). Гранулярные методы работают и на пуле,
	// и на транзакции; атомарные операции (смена ответственного, replace набора)
	// оркеструются в сервисе через WithTx. Все read-методы фильтруют
	// deleted_at IS NULL.
	ListAssignees(ctx context.Context, issueID string) ([]model.IssueAssignee, error)
	ListAssigneesByIssues(ctx context.Context, issueIDs []string) (map[string][]model.IssueAssignee, error)
	GetAssignee(ctx context.Context, issueID, userID string) (model.IssueAssignee, error)
	AddAssignee(ctx context.Context, a model.IssueAssignee) (model.IssueAssignee, error)
	RemoveAssignee(ctx context.Context, issueID, userID string) error
	ClearPrimary(ctx context.Context, issueID string) error
	SetPrimary(ctx context.Context, issueID, userID string) error

	// Labels (tracker.issue_label). Зеркало assignee-методов: гранулярные операции
	// работают и на пуле, и на транзакции; replace набора оркеструется сервисом
	// через WithTx. Read-методы фильтруют deleted_at IS NULL (обе стороны JOIN).
	ListLabels(ctx context.Context, issueID string) ([]model.Label, error)
	ListLabelsByIssues(ctx context.Context, issueIDs []string) (map[string][]model.Label, error)
	AddIssueLabel(ctx context.Context, issueID, labelID string) error
	RemoveIssueLabel(ctx context.Context, issueID, labelID string) error
}

// ProjectRepository — доступ к tracker.project и tracker.project_team.
type ProjectRepository interface {
	Create(ctx context.Context, p model.Project) (model.Project, error)
	GetByID(ctx context.Context, workspaceID, projectID string) (model.Project, error)
	Update(ctx context.Context, p model.Project) (model.Project, error)
	Delete(ctx context.Context, projectID string) error

	// List возвращает активные проекты workspace с keyset-пагинацией по
	// (sort, id). Выбирается page.Limit+1 строк — сервис по «лишней» строке
	// вычисляет has_more. Команды (project_team) не заполняются — их догружает
	// сервис батчем (ListTeamsByProjects).
	List(ctx context.Context, workspaceID string, page model.Page) ([]model.Project, error)

	// Junction tracker.project_team. Гранулярные методы работают и на пуле, и на
	// транзакции; атомарные операции (replace набора) оркеструются сервисом через
	// WithTx. Read-методы фильтруют deleted_at IS NULL.
	ListTeams(ctx context.Context, projectID string) ([]model.ProjectTeam, error)
	ListTeamsByProjects(ctx context.Context, projectIDs []string) (map[string][]model.ProjectTeam, error)
	AddTeam(ctx context.Context, projectID, teamID string) (model.ProjectTeam, error)
	RemoveTeam(ctx context.Context, projectID, teamID string) error
}

// LabelRepository — доступ к tracker.label и tracker.issue_label. Метки бывают
// workspace-level (team_id IS NULL) и team-level. Все read-методы фильтруют
// deleted_at IS NULL. RepointIssueLabels работает и на пуле, и на транзакции;
// атомарный merge (создание ws-метки + перепривязка + soft-delete старых)
// оркеструется сервисом через WithTx.
type LabelRepository interface {
	Create(ctx context.Context, l model.Label) (model.Label, error)
	GetByID(ctx context.Context, workspaceID, labelID string) (model.Label, error)
	List(ctx context.Context, workspaceID string) ([]model.Label, error)
	Update(ctx context.Context, l model.Label) (model.Label, error)
	Delete(ctx context.Context, labelID string) error

	GetWorkspaceLabelByName(ctx context.Context, workspaceID, name string) (model.Label, error)
	ListTeamLabelsByName(ctx context.Context, workspaceID, name string) ([]model.Label, error)
	RepointIssueLabels(ctx context.Context, fromLabelIDs []string, toLabelID string) error
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
	_ ProjectRepository   = (*ProjectRepo)(nil)
	_ LabelRepository     = (*LabelRepo)(nil)
	_ Store               = (*Repositories)(nil)
)
