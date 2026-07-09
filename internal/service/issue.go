package service

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/timohahaa/tracker/internal/errs"
	"github.com/timohahaa/tracker/internal/model"
	"github.com/timohahaa/tracker/internal/repository"
)

// IssueService — CRUD задач, генерация identifier, assignees.
// Read-операции (List, фильтры) при росте выносятся в issue_queries.go.
type IssueService struct {
	repos repository.Store
}

// CreateIssueParams — входные данные создания задачи (граница service).
// StatusID/ProjectID/DueDate/Description опциональны (nil = не задано).
type CreateIssueParams struct {
	TeamID      string
	Title       string
	Description *string
	StatusID    *string
	ProjectID   *string
	Priority    model.Priority
	DueDate     *time.Time
}

// UpdateIssueParams — частичное обновление задачи. Nil-поле означает «не менять».
// Assignees — replace-семантика всего набора: nil = «не трогать ассайни»,
// непустой/пустой слайс = заменить набор целиком (пустой = снять всех).
type UpdateIssueParams struct {
	Title       *string
	Description *string
	StatusID    *string
	ProjectID   *string
	Priority    *model.Priority
	DueDate     *time.Time
	Assignees   *[]AssigneeInput
}

// AssigneeInput — участник задачи во входных данных (граница service).
type AssigneeInput struct {
	UserID    string
	IsPrimary bool
}

// CreateIssue создаёт задачу в команде: выдаёт issue_number атомарным
// инкрементом и вставляет задачу в одной транзакции. Доступно участнику команды.
// status_id по умолчанию — дефолтный статус группы backlog (первый по position).
// Проверяет, что team принадлежит workspace, а заданные status_id/project_id
// относятся к этой team/workspace, иначе errs.Validation.
func (s *IssueService) CreateIssue(ctx context.Context, actor model.WorkspaceMember, params CreateIssueParams) (model.IssueDetails, error) {
	team, err := s.repos.Team().Get(ctx, actor.WorkspaceID, params.TeamID)
	if err != nil {
		return model.IssueDetails{}, err
	}
	if err := s.requireTeamMember(ctx, team.ID, actor.UserID); err != nil {
		return model.IssueDetails{}, err
	}

	statusID, err := s.resolveStatusID(ctx, team.ID, params.StatusID)
	if err != nil {
		return model.IssueDetails{}, err
	}

	if err := s.validateProject(ctx, actor.WorkspaceID, params.ProjectID); err != nil {
		return model.IssueDetails{}, err
	}

	priority := params.Priority
	if priority == "" {
		priority = model.PriorityNone
	}

	creatorID := actor.UserID

	var (
		created model.Issue
		number  int
	)
	err = s.repos.WithTx(ctx, func(tx repository.Store) error {
		number, err = tx.Issue().IncrementIssueNumber(ctx, team.ID)
		if err != nil {
			return err
		}
		created, err = tx.Issue().Create(ctx, model.Issue{
			TeamID:      team.ID,
			ProjectID:   params.ProjectID,
			StatusID:    statusID,
			CreatorID:   &creatorID,
			Title:       params.Title,
			Description: params.Description,
			Priority:    priority,
			DueDate:     params.DueDate,
			IssueNumber: number,
		})
		return err
	})
	if err != nil {
		return model.IssueDetails{}, err
	}

	return model.IssueDetails{
		Issue:      created,
		TeamKey:    team.Key,
		Identifier: team.Key + "-" + strconv.Itoa(number),
	}, nil
}

// GetIssue возвращает задачу workspace по ссылке — UUID или identifier (ENG-123)
// с догруженными активными assignees. Доступно участнику workspace (проверено в
// middleware LoadWorkspace).
func (s *IssueService) GetIssue(ctx context.Context, workspaceID, ref string) (model.IssueDetails, error) {
	var (
		details model.IssueDetails
		err     error
	)
	if _, perr := uuid.Parse(ref); perr == nil {
		details, err = s.repos.Issue().GetByID(ctx, workspaceID, ref)
	} else {
		key, number, ok := parseIdentifier(ref)
		if !ok {
			return model.IssueDetails{}, errs.NotFound("issue not found")
		}
		details, err = s.repos.Issue().GetByIdentifier(ctx, workspaceID, key, number)
	}
	if err != nil {
		return model.IssueDetails{}, err
	}
	if err := s.enrichAssignees(ctx, &details); err != nil {
		return model.IssueDetails{}, err
	}
	return details, nil
}

// enrichAssignees догружает активных участников задачи (read-модель
// IssueDetails.Assignees заполняется на уровне сервиса, репо-мапперы её не
// трогают).
func (s *IssueService) enrichAssignees(ctx context.Context, d *model.IssueDetails) error {
	assignees, err := s.repos.Issue().ListAssignees(ctx, d.ID)
	if err != nil {
		return err
	}
	d.Assignees = assignees
	return nil
}

// UpdateIssue частично обновляет задачу. Доступно участнику команды задачи.
// Заданные status_id/project_id валидируются относительно team/workspace.
func (s *IssueService) UpdateIssue(ctx context.Context, actor model.WorkspaceMember, ref string, params UpdateIssueParams) (model.IssueDetails, error) {
	details, err := s.GetIssue(ctx, actor.WorkspaceID, ref)
	if err != nil {
		return model.IssueDetails{}, err
	}
	if err := s.requireTeamMember(ctx, details.TeamID, actor.UserID); err != nil {
		return model.IssueDetails{}, err
	}

	issue := details.Issue
	if params.Title != nil {
		issue.Title = *params.Title
	}
	if params.Description != nil {
		issue.Description = params.Description
	}
	if params.StatusID != nil {
		if _, err := s.repos.Status().Get(ctx, issue.TeamID, *params.StatusID); err != nil {
			if isNotFound(err) {
				return model.IssueDetails{}, errs.Validation("invalid status", errs.FieldError{
					Field:   "status_id",
					Reason:  "invalid",
					Message: "must be a status of the issue team",
				})
			}
			return model.IssueDetails{}, err
		}
		issue.StatusID = *params.StatusID
	}
	if params.ProjectID != nil {
		if err := s.validateProject(ctx, actor.WorkspaceID, params.ProjectID); err != nil {
			return model.IssueDetails{}, err
		}
		issue.ProjectID = params.ProjectID
	}
	if params.Priority != nil {
		issue.Priority = *params.Priority
	}
	if params.DueDate != nil {
		issue.DueDate = params.DueDate
	}

	// Ассайни не трогаем — простой путь (одиночный UPDATE без транзакции).
	if params.Assignees == nil {
		updated, err := s.repos.Issue().Update(ctx, issue)
		if err != nil {
			return model.IssueDetails{}, err
		}
		out := model.IssueDetails{
			Issue:      updated,
			TeamKey:    details.TeamKey,
			Identifier: details.Identifier,
		}
		if err := s.enrichAssignees(ctx, &out); err != nil {
			return model.IssueDetails{}, err
		}
		return out, nil
	}

	// Replace-семантика набора: валидируем и заменяем в одной транзакции вместе
	// с обновлением полей задачи.
	set, err := s.validateAssigneeSet(ctx, issue.TeamID, *params.Assignees)
	if err != nil {
		return model.IssueDetails{}, err
	}

	var updated model.Issue
	err = s.repos.WithTx(ctx, func(tx repository.Store) error {
		var uerr error
		updated, uerr = tx.Issue().Update(ctx, issue)
		if uerr != nil {
			return uerr
		}
		return replaceAssignees(ctx, tx, issue.ID, set)
	})
	if err != nil {
		return model.IssueDetails{}, err
	}

	out := model.IssueDetails{
		Issue:      updated,
		TeamKey:    details.TeamKey,
		Identifier: details.Identifier,
	}
	if err := s.enrichAssignees(ctx, &out); err != nil {
		return model.IssueDetails{}, err
	}
	return out, nil
}

// DeleteIssue мягко удаляет задачу. Доступно участнику команды задачи.
func (s *IssueService) DeleteIssue(ctx context.Context, actor model.WorkspaceMember, ref string) error {
	details, err := s.GetIssue(ctx, actor.WorkspaceID, ref)
	if err != nil {
		return err
	}
	if err := s.requireTeamMember(ctx, details.TeamID, actor.UserID); err != nil {
		return err
	}
	return s.repos.Issue().Delete(ctx, details.ID)
}

// resolveStatusID возвращает status_id для новой задачи: заданный (после
// проверки принадлежности команде) либо дефолтный статус группы backlog (первый
// по position). Невалидный заданный статус → errs.Validation.
func (s *IssueService) resolveStatusID(ctx context.Context, teamID string, given *string) (string, error) {
	if given != nil {
		if _, err := s.repos.Status().Get(ctx, teamID, *given); err != nil {
			if isNotFound(err) {
				return "", errs.Validation("invalid status", errs.FieldError{
					Field:   "status_id",
					Reason:  "invalid",
					Message: "must be a status of the issue team",
				})
			}
			return "", err
		}
		return *given, nil
	}

	statuses, err := s.repos.Status().List(ctx, teamID)
	if err != nil {
		return "", err
	}
	// List упорядочен по (group, position), поэтому первый backlog-статус —
	// дефолтный для группы.
	for _, st := range statuses {
		if st.Group == model.StatusGroupBacklog {
			return st.ID, nil
		}
	}
	return "", errs.Validation("team has no default backlog status", errs.FieldError{
		Field:   "status_id",
		Reason:  "required",
		Message: "team has no backlog status to use as default",
	})
}

// validateProject проверяет, что заданный project активен и принадлежит
// workspace. nil-project — валиден (задача без проекта). Иначе errs.Validation.
func (s *IssueService) validateProject(ctx context.Context, workspaceID string, projectID *string) error {
	if projectID == nil {
		return nil
	}
	ok, err := s.repos.Issue().ProjectInWorkspace(ctx, workspaceID, *projectID)
	if err != nil {
		return err
	}
	if !ok {
		return errs.Validation("invalid project", errs.FieldError{
			Field:   "project_id",
			Reason:  "invalid",
			Message: "must be a project of the workspace",
		})
	}
	return nil
}

// requireTeamMember проверяет, что пользователь — активный участник команды.
// Нет membership → errs.Forbidden.
func (s *IssueService) requireTeamMember(ctx context.Context, teamID, userID string) error {
	if _, err := s.repos.Team().GetMember(ctx, teamID, userID); err != nil {
		if isNotFound(err) {
			return errs.Forbidden("not a team member")
		}
		return err
	}
	return nil
}

// parseIdentifier разбирает issue identifier вида "ENG-123" на team key и номер.
// key нормализуется (upper/trim, как при создании команды). ok=false, если
// формат не распознан (нет разделителя или номер не число).
func parseIdentifier(ref string) (key string, number int, ok bool) {
	idx := strings.LastIndex(ref, "-")
	if idx <= 0 || idx == len(ref)-1 {
		return "", 0, false
	}
	key = normalizeTeamKey(ref[:idx])
	n, err := strconv.Atoi(ref[idx+1:])
	if err != nil || n <= 0 {
		return "", 0, false
	}
	return key, n, true
}
