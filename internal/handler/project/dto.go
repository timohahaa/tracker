package project

import (
	"net/http"
	"time"

	"github.com/timohahaa/tracker/internal/handler/httpx"
	"github.com/timohahaa/tracker/internal/model"
	"github.com/timohahaa/tracker/internal/service"
)

// createProjectRequest — тело POST /projects. name обязателен; status по
// умолчанию planned. team_ids сразу привязывает команды (nil/отсутствие = без
// команд).
type createProjectRequest struct {
	Name        string    `json:"name" validate:"required,min=1"`
	Description *string   `json:"description"`
	LeadID      *string   `json:"lead_id" validate:"omitempty,uuid"`
	Status      string    `json:"status" validate:"omitempty,oneof=planned in_progress completed canceled"`
	TeamIDs     *[]string `json:"team_ids" validate:"omitempty,dive,uuid"`
}

// updateProjectRequest — тело PATCH /projects/{projectID}. Все поля опциональны
// (partial update): nil означает «не менять». Для team_ids: nil (поле не
// прислано) = «не трогать команды»; присланный массив (в т.ч. пустой) = replace
// всего набора (пустой = отвязать все).
type updateProjectRequest struct {
	Name        *string   `json:"name" validate:"omitempty,min=1"`
	Description *string   `json:"description"`
	LeadID      *string   `json:"lead_id" validate:"omitempty,uuid"`
	Status      *string   `json:"status" validate:"omitempty,oneof=planned in_progress completed canceled"`
	TeamIDs     *[]string `json:"team_ids" validate:"omitempty,dive,uuid"`
}

// projectResponse — публичное представление проекта с привязанными командами.
type projectResponse struct {
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

// paginationResponse — метаданные keyset-пагинации (docs/tech/api.md).
// next_cursor пуст, когда следующей страницы нет.
type paginationResponse struct {
	NextCursor string `json:"next_cursor"`
	HasMore    bool   `json:"has_more"`
}

// projectListResponse — конверт ответа списка проектов.
type projectListResponse struct {
	Data       []projectResponse  `json:"data"`
	Pagination paginationResponse `json:"pagination"`
}

// parseListQuery собирает model.Page из query-параметров запроса. Битый cursor →
// errs.Validation (400). Нормализацию/валидацию sort/order/limit довершает
// service.
func parseListQuery(r *http.Request) (model.Page, error) {
	cursor, err := httpx.QueryCursor(r)
	if err != nil {
		return model.Page{}, err
	}
	return model.Page{
		Limit:  httpx.QueryLimit(r, service.DefaultProjectLimit, service.MaxProjectLimit),
		Sort:   httpx.QueryString(r, "sort"),
		Order:  model.SortOrder(httpx.QueryString(r, "order")),
		Cursor: cursor,
	}, nil
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

func (r createProjectRequest) toParams() service.CreateProjectParams {
	return service.CreateProjectParams{
		Name:        r.Name,
		Description: r.Description,
		LeadID:      nilIfEmpty(r.LeadID),
		Status:      model.ProjectStatus(r.Status),
		TeamIDs:     r.TeamIDs,
	}
}

func (r updateProjectRequest) toParams() service.UpdateProjectParams {
	var status *model.ProjectStatus
	if r.Status != nil {
		s := model.ProjectStatus(*r.Status)
		status = &s
	}
	return service.UpdateProjectParams{
		Name:        r.Name,
		Description: r.Description,
		LeadID:      nilIfEmpty(r.LeadID),
		Status:      status,
		TeamIDs:     r.TeamIDs,
	}
}

func toProjectResponse(d model.ProjectDetails) projectResponse {
	teamIDs := d.TeamIDs
	if teamIDs == nil {
		teamIDs = []string{}
	}
	return projectResponse{
		ID:          d.ID,
		WorkspaceID: d.WorkspaceID,
		LeadID:      d.LeadID,
		Name:        d.Name,
		Description: d.Description,
		Status:      string(d.Status),
		TeamIDs:     teamIDs,
		CreatedAt:   d.CreatedAt,
		UpdatedAt:   d.UpdatedAt,
	}
}

func toProjectListResponse(res service.ProjectListResult) projectListResponse {
	data := make([]projectResponse, 0, len(res.Projects))
	for _, d := range res.Projects {
		data = append(data, toProjectResponse(d))
	}
	var next string
	if res.NextCursor != nil {
		next = httpx.EncodeCursor(res.NextCursor.Value, res.NextCursor.ID)
	}
	return projectListResponse{
		Data:       data,
		Pagination: paginationResponse{NextCursor: next, HasMore: res.HasMore},
	}
}
