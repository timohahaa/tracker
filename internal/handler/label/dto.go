package label

import (
	"github.com/timohahaa/tracker/internal/model"
	"github.com/timohahaa/tracker/internal/service"
)

// createLabelRequest — тело POST /labels. name/color обязательны; формат color
// ('#RRGGBB') валидирует service. team_id опционален (nil/пусто → workspace-level).
type createLabelRequest struct {
	TeamID *string `json:"team_id" validate:"omitempty,uuid"`
	Name   string  `json:"name" validate:"required,min=1"`
	Color  string  `json:"color" validate:"required"`
}

// updateLabelRequest — тело PATCH /labels/{labelID}. Все поля опциональны
// (partial update): nil означает «не менять».
type updateLabelRequest struct {
	Name  *string `json:"name" validate:"omitempty,min=1"`
	Color *string `json:"color" validate:"omitempty"`
}

// labelResponse — публичное представление метки. team_id пуст → workspace-level.
type labelResponse struct {
	ID          string  `json:"id"`
	WorkspaceID string  `json:"workspace_id"`
	TeamID      *string `json:"team_id,omitempty"`
	Name        string  `json:"name"`
	Color       string  `json:"color"`
}

// labelsResponse — ответ списка меток.
type labelsResponse struct {
	Data []labelResponse `json:"data"`
}

// nilIfEmpty нормализует опциональный id: пустая строка трактуется как «не задано»
// (nil), чтобы "" не дошёл до БД и не спровоцировал 22P02 → 500.
func nilIfEmpty(s *string) *string {
	if s == nil || *s == "" {
		return nil
	}
	return s
}

func (r createLabelRequest) toParams() service.CreateLabelParams {
	return service.CreateLabelParams{
		TeamID: nilIfEmpty(r.TeamID),
		Name:   r.Name,
		Color:  r.Color,
	}
}

func (r updateLabelRequest) toParams() service.UpdateLabelParams {
	return service.UpdateLabelParams{
		Name:  r.Name,
		Color: r.Color,
	}
}

func toLabelResponse(l model.Label) labelResponse {
	return labelResponse{
		ID:          l.ID,
		WorkspaceID: l.WorkspaceID,
		TeamID:      l.TeamID,
		Name:        l.Name,
		Color:       l.Color,
	}
}

func toLabelsResponse(labels []model.Label) labelsResponse {
	data := make([]labelResponse, 0, len(labels))
	for _, l := range labels {
		data = append(data, toLabelResponse(l))
	}
	return labelsResponse{Data: data}
}
