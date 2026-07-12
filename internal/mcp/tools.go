package mcp

import (
	"context"

	"github.com/google/uuid"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/timohahaa/tracker/internal/errs"
	"github.com/timohahaa/tracker/internal/handler/httpx"
	"github.com/timohahaa/tracker/internal/handler/middleware"
	"github.com/timohahaa/tracker/internal/model"
	"github.com/timohahaa/tracker/internal/service"
)

// Допустимые значения enum-фильтров (зеркало internal/handler/issue/dto.go).
// Невалидное значение → validation, чтобы клиент не получал молча пустой список.
var (
	validPriorities = map[string]bool{
		"none": true, "urgent": true, "high": true, "medium": true, "low": true,
	}
	validStatusGroups = map[string]bool{
		"backlog": true, "planned": true, "in_progress": true, "completed": true, "canceled": true,
	}
)

// register регистрирует 7 tools MCP-сервера. Каждый tool — тонкая обёртка над
// service layer: объявляет input schema (выводится из типа In), достаёт actor из
// контекста, мапит аргументы в service-параметры и сериализует доменный
// результат. Схема input и валидация обязательных полей — на стороне SDK.
func (s *Server) register(srv *mcp.Server) {
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "create_issue",
		Description: "Create an issue in a team.",
	}, s.createIssue)
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "get_issue",
		Description: "Get an issue by UUID or identifier (e.g. ENG-123).",
	}, s.getIssue)
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "update_issue",
		Description: "Partially update an issue. Assignees and label_ids replace the whole set.",
	}, s.updateIssue)
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "delete_issue",
		Description: "Soft-delete an issue.",
	}, s.deleteIssue)
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "search_issues",
		Description: "Search issues with filters and cursor pagination.",
	}, s.searchIssues)
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "list_projects",
		Description: "List workspace projects with cursor pagination.",
	}, s.listProjects)
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "list_teams",
		Description: "List workspace teams.",
	}, s.listTeams)
}

// --- create_issue ---

type createIssueInput struct {
	TeamID      string    `json:"team_id" jsonschema:"UUID of the team the issue belongs to"`
	Title       string    `json:"title" jsonschema:"issue title"`
	Description *string   `json:"description,omitempty" jsonschema:"issue description"`
	StatusID    *string   `json:"status_id,omitempty" jsonschema:"status UUID; defaults to the team's backlog status"`
	ProjectID   *string   `json:"project_id,omitempty" jsonschema:"project UUID"`
	Priority    string    `json:"priority,omitempty" jsonschema:"one of none, urgent, high, medium, low"`
	DueDate     *string   `json:"due_date,omitempty" jsonschema:"due date in YYYY-MM-DD"`
	LabelIDs    *[]string `json:"label_ids,omitempty" jsonschema:"set of label UUIDs to attach"`
}

func (s *Server) createIssue(ctx context.Context, _ *mcp.CallToolRequest, in createIssueInput) (*mcp.CallToolResult, any, error) {
	member := middleware.MemberFromContext(ctx)

	due, err := parseDate("due_date", in.DueDate)
	if err != nil {
		return respondMCPError(err), nil, nil
	}

	issue, err := s.services.Issue.CreateIssue(ctx, member, service.CreateIssueParams{
		TeamID:      in.TeamID,
		Title:       in.Title,
		Description: in.Description,
		StatusID:    nilIfEmpty(in.StatusID),
		ProjectID:   nilIfEmpty(in.ProjectID),
		Priority:    model.Priority(in.Priority),
		DueDate:     due,
		LabelIDs:    in.LabelIDs,
	})
	if err != nil {
		return respondMCPError(err), nil, nil
	}
	return nil, toIssueOutput(issue), nil
}

// --- get_issue ---

type getIssueInput struct {
	Ref string `json:"ref" jsonschema:"issue UUID or identifier like ENG-123"`
}

func (s *Server) getIssue(ctx context.Context, _ *mcp.CallToolRequest, in getIssueInput) (*mcp.CallToolResult, any, error) {
	ws := middleware.WorkspaceFromContext(ctx)

	issue, err := s.services.Issue.GetIssue(ctx, ws.ID, in.Ref)
	if err != nil {
		return respondMCPError(err), nil, nil
	}
	return nil, toIssueOutput(issue), nil
}

// --- update_issue ---

type assigneeInput struct {
	UserID    string `json:"user_id" jsonschema:"assignee user UUID"`
	IsPrimary bool   `json:"is_primary,omitempty" jsonschema:"whether this assignee is the responsible one"`
}

type updateIssueInput struct {
	Ref         string           `json:"ref" jsonschema:"issue UUID or identifier like ENG-123"`
	Title       *string          `json:"title,omitempty" jsonschema:"new title"`
	Description *string          `json:"description,omitempty" jsonschema:"new description"`
	StatusID    *string          `json:"status_id,omitempty" jsonschema:"new status UUID"`
	ProjectID   *string          `json:"project_id,omitempty" jsonschema:"new project UUID"`
	Priority    *string          `json:"priority,omitempty" jsonschema:"one of none, urgent, high, medium, low"`
	DueDate     *string          `json:"due_date,omitempty" jsonschema:"due date in YYYY-MM-DD"`
	Assignees   *[]assigneeInput `json:"assignees,omitempty" jsonschema:"full replacement set of assignees"`
	LabelIDs    *[]string        `json:"label_ids,omitempty" jsonschema:"full replacement set of label UUIDs"`
}

func (s *Server) updateIssue(ctx context.Context, _ *mcp.CallToolRequest, in updateIssueInput) (*mcp.CallToolResult, any, error) {
	member := middleware.MemberFromContext(ctx)

	due, err := parseDate("due_date", in.DueDate)
	if err != nil {
		return respondMCPError(err), nil, nil
	}

	var priority *model.Priority
	if in.Priority != nil {
		p := model.Priority(*in.Priority)
		priority = &p
	}

	var assignees *[]service.AssigneeInput
	if in.Assignees != nil {
		set := make([]service.AssigneeInput, 0, len(*in.Assignees))
		for _, a := range *in.Assignees {
			set = append(set, service.AssigneeInput{UserID: a.UserID, IsPrimary: a.IsPrimary})
		}
		assignees = &set
	}

	issue, err := s.services.Issue.UpdateIssue(ctx, member, in.Ref, service.UpdateIssueParams{
		Title:       in.Title,
		Description: in.Description,
		StatusID:    nilIfEmpty(in.StatusID),
		ProjectID:   nilIfEmpty(in.ProjectID),
		Priority:    priority,
		DueDate:     due,
		Assignees:   assignees,
		LabelIDs:    in.LabelIDs,
	})
	if err != nil {
		return respondMCPError(err), nil, nil
	}
	return nil, toIssueOutput(issue), nil
}

// --- delete_issue ---

type deleteIssueInput struct {
	Ref string `json:"ref" jsonschema:"issue UUID or identifier like ENG-123"`
}

type deleteIssueOutput struct {
	Ref     string `json:"ref"`
	Deleted bool   `json:"deleted"`
}

func (s *Server) deleteIssue(ctx context.Context, _ *mcp.CallToolRequest, in deleteIssueInput) (*mcp.CallToolResult, any, error) {
	member := middleware.MemberFromContext(ctx)

	if err := s.services.Issue.DeleteIssue(ctx, member, in.Ref); err != nil {
		return respondMCPError(err), nil, nil
	}
	return nil, deleteIssueOutput{Ref: in.Ref, Deleted: true}, nil
}

// --- search_issues ---

type searchIssuesInput struct {
	Team          *string  `json:"team,omitempty" jsonschema:"filter by team UUID"`
	Status        *string  `json:"status,omitempty" jsonschema:"filter by status UUID"`
	StatusGroup   *string  `json:"status_group,omitempty" jsonschema:"one of backlog, planned, in_progress, completed, canceled"`
	Assignee      *string  `json:"assignee,omitempty" jsonschema:"filter by assignee user UUID"`
	Responsible   *string  `json:"responsible,omitempty" jsonschema:"filter by responsible (primary assignee) user UUID"`
	Priority      *string  `json:"priority,omitempty" jsonschema:"one of none, urgent, high, medium, low"`
	Project       *string  `json:"project,omitempty" jsonschema:"filter by project UUID"`
	LabelIDs      []string `json:"label_ids,omitempty" jsonschema:"filter by label UUIDs"`
	Query         *string  `json:"q,omitempty" jsonschema:"full-text query over the title"`
	DueDateBefore *string  `json:"due_date_before,omitempty" jsonschema:"due date upper bound, YYYY-MM-DD"`
	DueDateAfter  *string  `json:"due_date_after,omitempty" jsonschema:"due date lower bound, YYYY-MM-DD"`
	Limit         int      `json:"limit,omitempty" jsonschema:"page size, max 100"`
	Cursor        string   `json:"cursor,omitempty" jsonschema:"opaque pagination cursor from a previous response"`
	Sort          string   `json:"sort,omitempty" jsonschema:"one of created_at, updated_at, priority"`
	Order         string   `json:"order,omitempty" jsonschema:"asc or desc"`
}

func (s *Server) searchIssues(ctx context.Context, _ *mcp.CallToolRequest, in searchIssuesInput) (*mcp.CallToolResult, any, error) {
	ws := middleware.WorkspaceFromContext(ctx)

	filter, err := s.buildIssueFilter(in)
	if err != nil {
		return respondMCPError(err), nil, nil
	}

	cursor, err := httpx.DecodeCursor(in.Cursor)
	if err != nil {
		return respondMCPError(errs.Validation("invalid cursor", errs.FieldError{
			Field:   "cursor",
			Reason:  "invalid",
			Message: "must be a valid cursor",
		})), nil, nil
	}

	page := model.Page{
		Limit:  in.Limit,
		Sort:   in.Sort,
		Order:  model.SortOrder(in.Order),
		Cursor: cursor,
	}

	res, err := s.services.Issue.ListIssues(ctx, ws.ID, filter, page)
	if err != nil {
		return respondMCPError(err), nil, nil
	}

	data := make([]issueOutput, 0, len(res.Issues))
	for _, d := range res.Issues {
		data = append(data, toIssueOutput(d))
	}
	return nil, searchIssuesOutput{
		Data: data,
		Pagination: paginationOutput{
			NextCursor: encodeNextCursor(res.NextCursor),
			HasMore:    res.HasMore,
		},
	}, nil
}

// buildIssueFilter собирает model.IssueFilter из аргументов tool, валидируя
// UUID-фильтры и enum-значения (status_group/priority) до вызова сервиса.
func (s *Server) buildIssueFilter(in searchIssuesInput) (model.IssueFilter, error) {
	var filter model.IssueFilter
	var err error

	if filter.TeamID, err = parseFilterUUID("team", in.Team); err != nil {
		return model.IssueFilter{}, err
	}
	if filter.StatusID, err = parseFilterUUID("status", in.Status); err != nil {
		return model.IssueFilter{}, err
	}
	if filter.Assignee, err = parseFilterUUID("assignee", in.Assignee); err != nil {
		return model.IssueFilter{}, err
	}
	if filter.Responsible, err = parseFilterUUID("responsible", in.Responsible); err != nil {
		return model.IssueFilter{}, err
	}
	if filter.ProjectID, err = parseFilterUUID("project", in.Project); err != nil {
		return model.IssueFilter{}, err
	}
	filter.Query = in.Query

	if in.StatusGroup != nil && *in.StatusGroup != "" {
		if !validStatusGroups[*in.StatusGroup] {
			return model.IssueFilter{}, errs.Validation("invalid status_group", errs.FieldError{
				Field:   "status_group",
				Reason:  "invalid",
				Message: "must be one of backlog, planned, in_progress, completed, canceled",
			})
		}
		sg := model.StatusGroup(*in.StatusGroup)
		filter.StatusGroup = &sg
	}
	if in.Priority != nil && *in.Priority != "" {
		if !validPriorities[*in.Priority] {
			return model.IssueFilter{}, errs.Validation("invalid priority", errs.FieldError{
				Field:   "priority",
				Reason:  "invalid",
				Message: "must be one of none, urgent, high, medium, low",
			})
		}
		pr := model.Priority(*in.Priority)
		filter.Priority = &pr
	}

	for _, raw := range in.LabelIDs {
		if raw == "" {
			continue
		}
		id, perr := uuid.Parse(raw)
		if perr != nil {
			return model.IssueFilter{}, errs.Validation("invalid label", errs.FieldError{
				Field:   "label_ids",
				Reason:  "invalid",
				Message: "must be valid UUIDs",
			})
		}
		filter.LabelIDs = append(filter.LabelIDs, id)
	}

	if filter.DueDateBefore, err = parseDate("due_date_before", in.DueDateBefore); err != nil {
		return model.IssueFilter{}, err
	}
	if filter.DueDateAfter, err = parseDate("due_date_after", in.DueDateAfter); err != nil {
		return model.IssueFilter{}, err
	}
	return filter, nil
}

// --- list_projects ---

type listProjectsInput struct {
	Limit  int    `json:"limit,omitempty" jsonschema:"page size, max 100"`
	Cursor string `json:"cursor,omitempty" jsonschema:"opaque pagination cursor from a previous response"`
	Sort   string `json:"sort,omitempty" jsonschema:"one of created_at, updated_at"`
	Order  string `json:"order,omitempty" jsonschema:"asc or desc"`
}

func (s *Server) listProjects(ctx context.Context, _ *mcp.CallToolRequest, in listProjectsInput) (*mcp.CallToolResult, any, error) {
	ws := middleware.WorkspaceFromContext(ctx)

	cursor, err := httpx.DecodeCursor(in.Cursor)
	if err != nil {
		return respondMCPError(errs.Validation("invalid cursor", errs.FieldError{
			Field:   "cursor",
			Reason:  "invalid",
			Message: "must be a valid cursor",
		})), nil, nil
	}

	res, err := s.services.Project.ListProjects(ctx, ws.ID, model.Page{
		Limit:  in.Limit,
		Sort:   in.Sort,
		Order:  model.SortOrder(in.Order),
		Cursor: cursor,
	})
	if err != nil {
		return respondMCPError(err), nil, nil
	}

	data := make([]projectOutput, 0, len(res.Projects))
	for _, p := range res.Projects {
		data = append(data, toProjectOutput(p))
	}
	return nil, listProjectsOutput{
		Data: data,
		Pagination: paginationOutput{
			NextCursor: encodeNextCursor(res.NextCursor),
			HasMore:    res.HasMore,
		},
	}, nil
}

// --- list_teams ---

type listTeamsInput struct{}

func (s *Server) listTeams(ctx context.Context, _ *mcp.CallToolRequest, _ listTeamsInput) (*mcp.CallToolResult, any, error) {
	ws := middleware.WorkspaceFromContext(ctx)

	teams, err := s.services.Team.ListTeams(ctx, ws.ID)
	if err != nil {
		return respondMCPError(err), nil, nil
	}

	data := make([]teamOutput, 0, len(teams))
	for _, t := range teams {
		data = append(data, toTeamOutput(t))
	}
	return nil, listTeamsOutput{Data: data}, nil
}
