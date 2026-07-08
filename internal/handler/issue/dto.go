package issue

import (
	"time"

	"github.com/timohahaa/tracker/internal/model"
	"github.com/timohahaa/tracker/internal/service"
)

// dateLayout — формат даты дедлайна (ISO 8601 date, без времени).
const dateLayout = "2006-01-02"

// createIssueRequest — тело POST /issues. team_id/title обязательны, остальное
// опционально. due_date — дата в формате YYYY-MM-DD.
type createIssueRequest struct {
	TeamID      string  `json:"team_id" validate:"required"`
	Title       string  `json:"title" validate:"required,min=1"`
	Description *string `json:"description"`
	StatusID    *string `json:"status_id"`
	ProjectID   *string `json:"project_id"`
	Priority    string  `json:"priority" validate:"omitempty,oneof=none urgent high medium low"`
	DueDate     *string `json:"due_date" validate:"omitempty,datetime=2006-01-02"`
}

// updateIssueRequest — тело PATCH /issues/{issueID}. Все поля опциональны
// (partial update): nil означает «не менять».
type updateIssueRequest struct {
	Title       *string `json:"title" validate:"omitempty,min=1"`
	Description *string `json:"description"`
	StatusID    *string `json:"status_id"`
	ProjectID   *string `json:"project_id"`
	Priority    *string `json:"priority" validate:"omitempty,oneof=none urgent high medium low"`
	DueDate     *string `json:"due_date" validate:"omitempty,datetime=2006-01-02"`
}

// assigneeResponse — участник задачи в ответе (user_id + признак ответственного).
// Заполняется в заходе assignees; сейчас всегда пусто.
type assigneeResponse struct {
	UserID    string `json:"user_id"`
	IsPrimary bool   `json:"is_primary"`
}

// labelResponse — label задачи в ответе. Заполняется в заходе label; сейчас пусто.
type labelResponse struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color"`
}

// issueResponse — публичное представление задачи (read-модель с identifier).
type issueResponse struct {
	ID          string             `json:"id"`
	Identifier  string             `json:"identifier"`
	TeamID      string             `json:"team_id"`
	TeamKey     string             `json:"team_key"`
	ProjectID   *string            `json:"project_id,omitempty"`
	StatusID    string             `json:"status_id"`
	CreatorID   *string            `json:"creator_id,omitempty"`
	Title       string             `json:"title"`
	Description *string            `json:"description,omitempty"`
	Priority    string             `json:"priority"`
	DueDate     *string            `json:"due_date,omitempty"`
	IssueNumber int                `json:"issue_number"`
	Assignees   []assigneeResponse `json:"assignees"`
	Labels      []labelResponse    `json:"labels"`
	CreatedAt   time.Time          `json:"created_at"`
	UpdatedAt   time.Time          `json:"updated_at"`
}

func (r createIssueRequest) toParams() (service.CreateIssueParams, error) {
	due, err := parseDueDate(r.DueDate)
	if err != nil {
		return service.CreateIssueParams{}, err
	}
	return service.CreateIssueParams{
		TeamID:      r.TeamID,
		Title:       r.Title,
		Description: r.Description,
		StatusID:    r.StatusID,
		ProjectID:   r.ProjectID,
		Priority:    model.Priority(r.Priority),
		DueDate:     due,
	}, nil
}

func (r updateIssueRequest) toParams() (service.UpdateIssueParams, error) {
	due, err := parseDueDate(r.DueDate)
	if err != nil {
		return service.UpdateIssueParams{}, err
	}
	var priority *model.Priority
	if r.Priority != nil {
		p := model.Priority(*r.Priority)
		priority = &p
	}
	return service.UpdateIssueParams{
		Title:       r.Title,
		Description: r.Description,
		StatusID:    r.StatusID,
		ProjectID:   r.ProjectID,
		Priority:    priority,
		DueDate:     due,
	}, nil
}

// parseDueDate парсит опциональную дату дедлайна (YYYY-MM-DD). nil → nil.
// Формат уже провалидирован тегом validate, поэтому ошибка здесь маловероятна.
func parseDueDate(s *string) (*time.Time, error) {
	if s == nil {
		return nil, nil
	}
	t, err := time.Parse(dateLayout, *s)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func toIssueResponse(d model.IssueDetails) issueResponse {
	assignees := make([]assigneeResponse, 0, len(d.Assignees))
	for _, a := range d.Assignees {
		assignees = append(assignees, assigneeResponse{UserID: a.UserID, IsPrimary: a.IsPrimary})
	}
	labels := make([]labelResponse, 0, len(d.Labels))
	for _, l := range d.Labels {
		labels = append(labels, labelResponse{ID: l.ID, Name: l.Name, Color: l.Color})
	}

	var due *string
	if d.DueDate != nil {
		s := d.DueDate.Format(dateLayout)
		due = &s
	}

	return issueResponse{
		ID:          d.ID,
		Identifier:  d.Identifier,
		TeamID:      d.TeamID,
		TeamKey:     d.TeamKey,
		ProjectID:   d.ProjectID,
		StatusID:    d.StatusID,
		CreatorID:   d.CreatorID,
		Title:       d.Title,
		Description: d.Description,
		Priority:    string(d.Priority),
		DueDate:     due,
		IssueNumber: d.IssueNumber,
		Assignees:   assignees,
		Labels:      labels,
		CreatedAt:   d.CreatedAt,
		UpdatedAt:   d.UpdatedAt,
	}
}
