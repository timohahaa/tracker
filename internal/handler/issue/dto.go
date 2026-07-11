package issue

import (
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/timohahaa/tracker/internal/errs"
	"github.com/timohahaa/tracker/internal/handler/httpx"
	"github.com/timohahaa/tracker/internal/model"
	"github.com/timohahaa/tracker/internal/service"
)

// dateLayout — формат даты дедлайна (ISO 8601 date, без времени).
const dateLayout = "2006-01-02"

// createIssueRequest — тело POST /issues. team_id/title обязательны, остальное
// опционально. due_date — дата в формате YYYY-MM-DD. label_ids — набор меток
// (workspace-level или команды задачи); nil = без меток.
type createIssueRequest struct {
	TeamID      string    `json:"team_id" validate:"required,uuid"`
	Title       string    `json:"title" validate:"required,min=1"`
	Description *string   `json:"description"`
	StatusID    *string   `json:"status_id" validate:"omitempty,uuid"`
	ProjectID   *string   `json:"project_id" validate:"omitempty,uuid"`
	Priority    string    `json:"priority" validate:"omitempty,oneof=none urgent high medium low"`
	DueDate     *string   `json:"due_date" validate:"omitempty,datetime=2006-01-02"`
	LabelIDs    *[]string `json:"label_ids" validate:"omitempty,dive,uuid"`
}

// updateIssueRequest — тело PATCH /issues/{issueID}. Все поля опциональны
// (partial update): nil означает «не менять». Для assignees/label_ids: nil (поле
// не прислано) = «не трогать»; присланный массив (в т.ч. пустой) = replace всего
// набора (пустой = снять все).
type updateIssueRequest struct {
	Title       *string          `json:"title" validate:"omitempty,min=1"`
	Description *string          `json:"description"`
	StatusID    *string          `json:"status_id" validate:"omitempty,uuid"`
	ProjectID   *string          `json:"project_id" validate:"omitempty,uuid"`
	Priority    *string          `json:"priority" validate:"omitempty,oneof=none urgent high medium low"`
	DueDate     *string          `json:"due_date" validate:"omitempty,datetime=2006-01-02"`
	Assignees   *[]assigneeInput `json:"assignees" validate:"omitempty,dive"`
	LabelIDs    *[]string        `json:"label_ids" validate:"omitempty,dive,uuid"`
}

// assigneeInput — элемент набора assignees в PATCH-replace.
type assigneeInput struct {
	UserID    string `json:"user_id" validate:"required,uuid"`
	IsPrimary bool   `json:"is_primary"`
}

// addAssigneeRequest — тело POST /issues/{issueID}/assignees. is_primary
// опционален: true → участник сразу назначается ответственным.
type addAssigneeRequest struct {
	UserID    string `json:"user_id" validate:"required,uuid"`
	IsPrimary bool   `json:"is_primary"`
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

// validPriorities / validStatusGroups — допустимые значения enum-фильтров.
// Невалидное значение → 400, чтобы клиент не молча получал пустой список.
var (
	validPriorities = map[string]bool{
		"none": true, "urgent": true, "high": true, "medium": true, "low": true,
	}
	validStatusGroups = map[string]bool{
		"backlog": true, "planned": true, "in_progress": true, "completed": true, "canceled": true,
	}
)

// paginationResponse — метаданные keyset-пагинации (docs/tech/api.md).
// next_cursor пуст, когда следующей страницы нет.
type paginationResponse struct {
	NextCursor string `json:"next_cursor"`
	HasMore    bool   `json:"has_more"`
}

// issueListResponse — конверт ответа списка задач.
type issueListResponse struct {
	Data       []issueResponse    `json:"data"`
	Pagination paginationResponse `json:"pagination"`
}

// parseListQuery собирает model.IssueFilter и model.Page из query-параметров
// запроса. Битый cursor и невалидные enum/дата-фильтры → errs.Validation (400).
// Нормализацию/валидацию sort/order/limit довершает service.
func parseListQuery(r *http.Request) (model.IssueFilter, model.Page, error) {
	var filter model.IssueFilter
	var err error

	if filter.TeamID, err = optUUID(r, "team"); err != nil {
		return model.IssueFilter{}, model.Page{}, err
	}
	if filter.StatusID, err = optUUID(r, "status"); err != nil {
		return model.IssueFilter{}, model.Page{}, err
	}
	if filter.Assignee, err = optUUID(r, "assignee"); err != nil {
		return model.IssueFilter{}, model.Page{}, err
	}
	if filter.Responsible, err = optUUID(r, "responsible"); err != nil {
		return model.IssueFilter{}, model.Page{}, err
	}
	if filter.ProjectID, err = optUUID(r, "project"); err != nil {
		return model.IssueFilter{}, model.Page{}, err
	}
	filter.Query = optStr(httpx.QueryString(r, "q"))

	if g := httpx.QueryString(r, "status_group"); g != "" {
		if !validStatusGroups[g] {
			return model.IssueFilter{}, model.Page{}, errs.Validation("invalid status_group", errs.FieldError{
				Field:   "status_group",
				Reason:  "invalid",
				Message: "must be one of backlog, planned, in_progress, completed, canceled",
			})
		}
		sg := model.StatusGroup(g)
		filter.StatusGroup = &sg
	}
	if p := httpx.QueryString(r, "priority"); p != "" {
		if !validPriorities[p] {
			return model.IssueFilter{}, model.Page{}, errs.Validation("invalid priority", errs.FieldError{
				Field:   "priority",
				Reason:  "invalid",
				Message: "must be one of none, urgent, high, medium, low",
			})
		}
		pr := model.Priority(p)
		filter.Priority = &pr
	}

	for _, raw := range r.URL.Query()["label"] {
		if raw == "" {
			continue
		}
		id, perr := uuid.Parse(raw)
		if perr != nil {
			return model.IssueFilter{}, model.Page{}, errs.Validation("invalid label", errs.FieldError{
				Field:   "label",
				Reason:  "invalid",
				Message: "must be a valid UUID",
			})
		}
		filter.LabelIDs = append(filter.LabelIDs, id)
	}

	before, err := parseDueDate(optStr(httpx.QueryString(r, "due_date_before")))
	if err != nil {
		return model.IssueFilter{}, model.Page{}, errs.Validation("invalid due_date_before", errs.FieldError{
			Field:   "due_date_before",
			Reason:  "invalid",
			Message: "must be a date in format YYYY-MM-DD",
		})
	}
	filter.DueDateBefore = before

	after, err := parseDueDate(optStr(httpx.QueryString(r, "due_date_after")))
	if err != nil {
		return model.IssueFilter{}, model.Page{}, errs.Validation("invalid due_date_after", errs.FieldError{
			Field:   "due_date_after",
			Reason:  "invalid",
			Message: "must be a date in format YYYY-MM-DD",
		})
	}
	filter.DueDateAfter = after

	cursor, err := httpx.QueryCursor(r)
	if err != nil {
		return model.IssueFilter{}, model.Page{}, err
	}
	page := model.Page{
		Limit:  httpx.QueryLimit(r, service.DefaultIssueLimit, service.MaxIssueLimit),
		Sort:   httpx.QueryString(r, "sort"),
		Order:  model.SortOrder(httpx.QueryString(r, "order")),
		Cursor: cursor,
	}
	return filter, page, nil
}

// nilIfEmpty нормализует опциональный id: пустая строка трактуется как «не задано»
// (nil). Валидатор ловит только непустой мусор (omitempty,uuid), поэтому "" здесь
// приводим к nil, чтобы он не дошёл до БД и не спровоцировал 22P02 → 500.
func nilIfEmpty(s *string) *string {
	if s == nil || *s == "" {
		return nil
	}
	return s
}

// optStr возвращает *string для непустого значения, иначе nil (фильтр не задан).
func optStr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// optUUID читает необязательный UUID-фильтр из query. Пусто → (nil, nil);
// невалидный UUID → errs.Validation (400) с полем key.
func optUUID(r *http.Request, key string) (*uuid.UUID, error) {
	raw := httpx.QueryString(r, key)
	if raw == "" {
		return nil, nil
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		return nil, errs.Validation("invalid "+key, errs.FieldError{
			Field:   key,
			Reason:  "invalid",
			Message: "must be a valid UUID",
		})
	}
	return &id, nil
}

func toIssueListResponse(res service.IssueListResult) issueListResponse {
	data := make([]issueResponse, 0, len(res.Issues))
	for _, d := range res.Issues {
		data = append(data, toIssueResponse(d))
	}
	var next string
	if res.NextCursor != nil {
		next = httpx.EncodeCursor(res.NextCursor.Value, res.NextCursor.ID)
	}
	return issueListResponse{
		Data:       data,
		Pagination: paginationResponse{NextCursor: next, HasMore: res.HasMore},
	}
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
		StatusID:    nilIfEmpty(r.StatusID),
		ProjectID:   nilIfEmpty(r.ProjectID),
		Priority:    model.Priority(r.Priority),
		DueDate:     due,
		LabelIDs:    r.LabelIDs,
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
	var assignees *[]service.AssigneeInput
	if r.Assignees != nil {
		set := make([]service.AssigneeInput, 0, len(*r.Assignees))
		for _, a := range *r.Assignees {
			set = append(set, service.AssigneeInput{UserID: a.UserID, IsPrimary: a.IsPrimary})
		}
		assignees = &set
	}
	return service.UpdateIssueParams{
		Title:       r.Title,
		Description: r.Description,
		StatusID:    nilIfEmpty(r.StatusID),
		ProjectID:   nilIfEmpty(r.ProjectID),
		Priority:    priority,
		DueDate:     due,
		Assignees:   assignees,
		LabelIDs:    r.LabelIDs,
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
