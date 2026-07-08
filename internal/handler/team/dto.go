package team

import (
	"time"

	"github.com/timohahaa/tracker/internal/model"
	"github.com/timohahaa/tracker/internal/service"
)

// createTeamRequest — тело POST /teams. name/key обязательны.
type createTeamRequest struct {
	Name        string  `json:"name" validate:"required,min=1"`
	Key         string  `json:"key" validate:"required,min=1"`
	Description *string `json:"description"`
}

// updateTeamRequest — тело PATCH /teams/{teamID}. Все поля опциональны
// (partial update): nil означает «не менять».
type updateTeamRequest struct {
	Name        *string `json:"name" validate:"omitempty,min=1"`
	Key         *string `json:"key" validate:"omitempty,min=1"`
	Description *string `json:"description"`
}

// addMemberRequest — тело POST /teams/{teamID}/members.
type addMemberRequest struct {
	UserID string `json:"user_id" validate:"required"`
	Role   string `json:"role" validate:"omitempty,oneof=owner member"`
}

// createStatusRequest — тело POST /teams/{teamID}/statuses.
type createStatusRequest struct {
	Name     string `json:"name" validate:"required,min=1"`
	Group    string `json:"group" validate:"required,oneof=backlog planned in_progress completed canceled"`
	Position int    `json:"position" validate:"gte=0"`
}

// updateStatusRequest — тело PATCH /teams/{teamID}/statuses/{statusID}.
type updateStatusRequest struct {
	Name     *string `json:"name" validate:"omitempty,min=1"`
	Position *int    `json:"position" validate:"omitempty,gte=0"`
}

// teamResponse — публичное представление команды.
type teamResponse struct {
	ID              string    `json:"id"`
	WorkspaceID     string    `json:"workspace_id"`
	Name            string    `json:"name"`
	Description     *string   `json:"description,omitempty"`
	Key             string    `json:"key"`
	NextIssueNumber int       `json:"next_issue_number"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// teamsResponse — ответ списка команд.
type teamsResponse struct {
	Data []teamResponse `json:"data"`
}

// memberResponse — публичное представление участника команды.
type memberResponse struct {
	TeamID    string    `json:"team_id"`
	UserID    string    `json:"user_id"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

// membersResponse — ответ списка участников команды.
type membersResponse struct {
	Data []memberResponse `json:"data"`
}

// statusResponse — публичное представление статуса.
type statusResponse struct {
	ID        string    `json:"id"`
	TeamID    string    `json:"team_id"`
	Name      string    `json:"name"`
	Group     string    `json:"group"`
	Position  int       `json:"position"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// statusesResponse — ответ списка статусов.
type statusesResponse struct {
	Data []statusResponse `json:"data"`
}

func (r createTeamRequest) toParams() service.CreateTeamParams {
	return service.CreateTeamParams{
		Name:        r.Name,
		Key:         r.Key,
		Description: r.Description,
	}
}

func (r updateTeamRequest) toParams() service.UpdateTeamParams {
	return service.UpdateTeamParams{
		Name:        r.Name,
		Key:         r.Key,
		Description: r.Description,
	}
}

func (r addMemberRequest) toParams() service.AddTeamMemberParams {
	return service.AddTeamMemberParams{
		UserID: r.UserID,
		Role:   model.TeamRole(r.Role),
	}
}

func (r createStatusRequest) toParams() service.CreateStatusParams {
	return service.CreateStatusParams{
		Name:     r.Name,
		Group:    model.StatusGroup(r.Group),
		Position: r.Position,
	}
}

func (r updateStatusRequest) toParams() service.UpdateStatusParams {
	return service.UpdateStatusParams{
		Name:     r.Name,
		Position: r.Position,
	}
}

func toTeamResponse(t model.Team) teamResponse {
	return teamResponse{
		ID:              t.ID,
		WorkspaceID:     t.WorkspaceID,
		Name:            t.Name,
		Description:     t.Description,
		Key:             t.Key,
		NextIssueNumber: t.NextIssueNumber,
		CreatedAt:       t.CreatedAt,
		UpdatedAt:       t.UpdatedAt,
	}
}

func toTeamsResponse(teams []model.Team) teamsResponse {
	data := make([]teamResponse, 0, len(teams))
	for _, t := range teams {
		data = append(data, toTeamResponse(t))
	}
	return teamsResponse{Data: data}
}

func toMemberResponse(m model.TeamMember) memberResponse {
	return memberResponse{
		TeamID:    m.TeamID,
		UserID:    m.UserID,
		Role:      string(m.Role),
		CreatedAt: m.CreatedAt,
	}
}

func toMembersResponse(members []model.TeamMember) membersResponse {
	data := make([]memberResponse, 0, len(members))
	for _, m := range members {
		data = append(data, toMemberResponse(m))
	}
	return membersResponse{Data: data}
}

func toStatusResponse(s model.Status) statusResponse {
	return statusResponse{
		ID:        s.ID,
		TeamID:    s.TeamID,
		Name:      s.Name,
		Group:     string(s.Group),
		Position:  s.Position,
		CreatedAt: s.CreatedAt,
		UpdatedAt: s.UpdatedAt,
	}
}

func toStatusesResponse(statuses []model.Status) statusesResponse {
	data := make([]statusResponse, 0, len(statuses))
	for _, s := range statuses {
		data = append(data, toStatusResponse(s))
	}
	return statusesResponse{Data: data}
}
