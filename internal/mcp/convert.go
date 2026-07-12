package mcp

import (
	"time"

	"github.com/google/uuid"

	"github.com/timohahaa/tracker/internal/errs"
	"github.com/timohahaa/tracker/internal/handler/httpx"
	"github.com/timohahaa/tracker/internal/model"
)

// dateLayout — формат даты дедлайна (ISO 8601 date, без времени), общий с REST.
const dateLayout = "2006-01-02"

// Выходные структуры дублируют форму REST-DTO (internal/handler/*), которые
// unexported: MCP-слой не тянет транспортный пакет ради JSON-представления.

type assigneeOutput struct {
	UserID    string `json:"user_id"`
	IsPrimary bool   `json:"is_primary"`
}

type labelOutput struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color"`
}

type issueOutput struct {
	ID          string           `json:"id"`
	Identifier  string           `json:"identifier"`
	TeamID      string           `json:"team_id"`
	TeamKey     string           `json:"team_key"`
	ProjectID   *string          `json:"project_id,omitempty"`
	StatusID    string           `json:"status_id"`
	CreatorID   *string          `json:"creator_id,omitempty"`
	Title       string           `json:"title"`
	Description *string          `json:"description,omitempty"`
	Priority    string           `json:"priority"`
	DueDate     *string          `json:"due_date,omitempty"`
	IssueNumber int              `json:"issue_number"`
	Assignees   []assigneeOutput `json:"assignees"`
	Labels      []labelOutput    `json:"labels"`
	CreatedAt   time.Time        `json:"created_at"`
	UpdatedAt   time.Time        `json:"updated_at"`
}

type teamOutput struct {
	ID          string    `json:"id"`
	WorkspaceID string    `json:"workspace_id"`
	Name        string    `json:"name"`
	Description *string   `json:"description,omitempty"`
	Key         string    `json:"key"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type projectOutput struct {
	ID          string    `json:"id"`
	WorkspaceID string    `json:"workspace_id"`
	LeadID      *string   `json:"lead_id,omitempty"`
	Name        string    `json:"name"`
	Description *string   `json:"description,omitempty"`
	Status      string    `json:"status"`
	TeamIDs     []string  `json:"team_ids"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// paginationOutput — метаданные keyset-пагинации (как в REST). next_cursor пуст,
// когда следующей страницы нет.
type paginationOutput struct {
	NextCursor string `json:"next_cursor"`
	HasMore    bool   `json:"has_more"`
}

type searchIssuesOutput struct {
	Data       []issueOutput    `json:"data"`
	Pagination paginationOutput `json:"pagination"`
}

type listProjectsOutput struct {
	Data       []projectOutput  `json:"data"`
	Pagination paginationOutput `json:"pagination"`
}

type listTeamsOutput struct {
	Data []teamOutput `json:"data"`
}

func toIssueOutput(d model.IssueDetails) issueOutput {
	assignees := make([]assigneeOutput, 0, len(d.Assignees))
	for _, a := range d.Assignees {
		assignees = append(assignees, assigneeOutput{UserID: a.UserID, IsPrimary: a.IsPrimary})
	}
	labels := make([]labelOutput, 0, len(d.Labels))
	for _, l := range d.Labels {
		labels = append(labels, labelOutput{ID: l.ID, Name: l.Name, Color: l.Color})
	}
	var due *string
	if d.DueDate != nil {
		s := d.DueDate.Format(dateLayout)
		due = &s
	}
	return issueOutput{
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

func toTeamOutput(t model.Team) teamOutput {
	return teamOutput{
		ID:          t.ID,
		WorkspaceID: t.WorkspaceID,
		Name:        t.Name,
		Description: t.Description,
		Key:         t.Key,
		CreatedAt:   t.CreatedAt,
		UpdatedAt:   t.UpdatedAt,
	}
}

func toProjectOutput(p model.ProjectDetails) projectOutput {
	teamIDs := p.TeamIDs
	if teamIDs == nil {
		teamIDs = []string{}
	}
	return projectOutput{
		ID:          p.ID,
		WorkspaceID: p.WorkspaceID,
		LeadID:      p.LeadID,
		Name:        p.Name,
		Description: p.Description,
		Status:      string(p.Status),
		TeamIDs:     teamIDs,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
	}
}

// encodeNextCursor кодирует курсор следующей страницы в непрозрачную строку
// (тот же base64-формат, что и в REST). nil → "".
func encodeNextCursor(c *model.Cursor) string {
	if c == nil {
		return ""
	}
	return httpx.EncodeCursor(c.Value, c.ID)
}

// nilIfEmpty нормализует опциональный id: пустая строка → nil, чтобы мусор не
// дошёл до БД (симметрично REST-хендлерам).
func nilIfEmpty(s *string) *string {
	if s == nil || *s == "" {
		return nil
	}
	return s
}

// parseDate парсит опциональную дату (YYYY-MM-DD). nil/"" → nil; битый формат →
// errs.Validation с указанным полем.
func parseDate(field string, s *string) (*time.Time, error) {
	if s == nil || *s == "" {
		return nil, nil
	}
	t, err := time.Parse(dateLayout, *s)
	if err != nil {
		return nil, errs.Validation("invalid "+field, errs.FieldError{
			Field:   field,
			Reason:  "invalid",
			Message: "must be a date in format YYYY-MM-DD",
		})
	}
	return &t, nil
}

// parseFilterUUID разбирает опциональный UUID-фильтр. nil/"" → (nil, nil);
// невалидный UUID → errs.Validation (как optUUID на HTTP-границе).
func parseFilterUUID(field string, raw *string) (*uuid.UUID, error) {
	if raw == nil || *raw == "" {
		return nil, nil
	}
	id, err := uuid.Parse(*raw)
	if err != nil {
		return nil, errs.Validation("invalid "+field, errs.FieldError{
			Field:   field,
			Reason:  "invalid",
			Message: "must be a valid UUID",
		})
	}
	return &id, nil
}
