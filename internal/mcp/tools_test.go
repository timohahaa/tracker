package mcp

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.uber.org/mock/gomock"

	"github.com/timohahaa/tracker/config"
	"github.com/timohahaa/tracker/internal/errs"
	"github.com/timohahaa/tracker/internal/handler/middleware"
	"github.com/timohahaa/tracker/internal/model"
	"github.com/timohahaa/tracker/internal/repository"
	repomocks "github.com/timohahaa/tracker/internal/repository/mocks"
	"github.com/timohahaa/tracker/internal/service"
)

// Тесты MCP tools проверяют роль слоя: маппинг аргументов tool в параметры
// сервиса, сериализацию доменного результата в content и трансляцию
// *errs.AppError в MCP tool-ошибку (IsError + доменный код). Бизнес-логика —
// в сервисах, поэтому tools подключены к реальным сервисам поверх мок-стора:
// наблюдаемое поведение — входы репозиториев и форма результата.

func newTestServer(store repository.Store) *Server {
	return &Server{services: service.New(store, config.Config{})}
}

// toolCtx кладёт actor/workspace в контекст теми же хелперами, что и TokenAuth,
// — так tools их и читают.
func toolCtx() context.Context {
	ctx := context.Background()
	ctx = middleware.WithWorkspace(ctx, model.Workspace{ID: "ws-1"})
	ctx = middleware.WithMember(ctx, model.WorkspaceMember{
		WorkspaceID: "ws-1", UserID: "user-1", Role: model.WorkspaceRoleMember,
	})
	return ctx
}

func expectTx(store *repomocks.MockStore) {
	store.EXPECT().WithTx(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, fn func(repository.Store) error) error {
			return fn(store)
		}).AnyTimes()
}

// errBody достаёт JSON-тело ошибки из tool-результата с IsError:true.
func errBody(t *testing.T, res *mcp.CallToolResult) mcpErrorBody {
	t.Helper()
	if res == nil || !res.IsError {
		t.Fatalf("expected error result, got %+v", res)
	}
	if len(res.Content) == 0 {
		t.Fatalf("error result has no content")
	}
	tc, ok := res.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("content is not text: %T", res.Content[0])
	}
	var body mcpErrorBody
	if err := json.Unmarshal([]byte(tc.Text), &body); err != nil {
		t.Fatalf("unmarshal error body: %v", err)
	}
	return body
}

// --- create_issue ---

func TestCreateIssue_MapsArgsToServiceParams(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := repomocks.NewMockStore(ctrl)
	teams := repomocks.NewMockTeamRepository(ctrl)
	statuses := repomocks.NewMockStatusRepository(ctrl)
	issues := repomocks.NewMockIssueRepository(ctrl)

	expectTx(store)
	store.EXPECT().Team().Return(teams).AnyTimes()
	store.EXPECT().Status().Return(statuses).AnyTimes()
	store.EXPECT().Issue().Return(issues).AnyTimes()

	teams.EXPECT().Get(gomock.Any(), "ws-1", "team-1").
		Return(model.Team{ID: "team-1", WorkspaceID: "ws-1", Key: "ENG"}, nil)
	teams.EXPECT().GetMember(gomock.Any(), "team-1", "user-1").
		Return(model.TeamMember{TeamID: "team-1", UserID: "user-1"}, nil)
	statuses.EXPECT().Get(gomock.Any(), "team-1", "st-1").
		Return(model.Status{ID: "st-1", TeamID: "team-1", Group: model.StatusGroupBacklog}, nil)
	issues.EXPECT().IncrementIssueNumber(gomock.Any(), "team-1").Return(7, nil)

	var created model.Issue
	issues.EXPECT().Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, i model.Issue) (model.Issue, error) {
			created = i
			i.ID = "issue-1"
			return i, nil
		})
	issues.EXPECT().ListLabels(gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()

	status := "st-1"
	desc := "details"
	due := "2026-03-01"
	res, out, err := newTestServer(store).createIssue(toolCtx(), nil, createIssueInput{
		TeamID:      "team-1",
		Title:       "Fix bug",
		Description: &desc,
		StatusID:    &status,
		Priority:    "high",
		DueDate:     &due,
	})
	if err != nil {
		t.Fatalf("createIssue: %v", err)
	}
	if res != nil {
		t.Fatalf("unexpected error result: %+v", res)
	}
	if created.Title != "Fix bug" || created.Priority != model.PriorityHigh {
		t.Errorf("mapped issue = %+v, want title=Fix bug priority=high", created)
	}
	if created.Description == nil || *created.Description != "details" {
		t.Errorf("description not mapped: %v", created.Description)
	}
	if created.DueDate == nil || created.DueDate.Format(dateLayout) != "2026-03-01" {
		t.Errorf("due_date not mapped: %v", created.DueDate)
	}
	got, ok := out.(issueOutput)
	if !ok {
		t.Fatalf("output type = %T, want issueOutput", out)
	}
	if got.Identifier != "ENG-7" || got.Title != "Fix bug" || got.Priority != "high" {
		t.Errorf("output = %+v, want identifier=ENG-7", got)
	}
}

func TestCreateIssue_InvalidDueDateIsValidationError(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := repomocks.NewMockStore(ctrl)
	// Никаких вызовов репозитория: маппинг падает на разборе даты до сервиса.

	bad := "01-03-2026"
	res, out, err := newTestServer(store).createIssue(toolCtx(), nil, createIssueInput{
		TeamID:  "team-1",
		Title:   "x",
		DueDate: &bad,
	})
	if err != nil {
		t.Fatalf("createIssue returned protocol error: %v", err)
	}
	if out != nil {
		t.Fatalf("unexpected output: %+v", out)
	}
	if body := errBody(t, res); body.Code != string(errs.CodeValidation) {
		t.Errorf("code = %q, want validation", body.Code)
	}
}

// --- get_issue ---

func TestGetIssue_ReturnsIssueContent(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := repomocks.NewMockStore(ctrl)
	issues := repomocks.NewMockIssueRepository(ctrl)
	store.EXPECT().Issue().Return(issues).AnyTimes()

	const ref = "11111111-1111-1111-1111-111111111111"
	issues.EXPECT().GetByID(gomock.Any(), "ws-1", ref).Return(model.IssueDetails{
		Issue:      model.Issue{ID: ref, TeamID: "team-1", Title: "Task", Priority: model.PriorityNone},
		Identifier: "ENG-1",
		TeamKey:    "ENG",
	}, nil)
	issues.EXPECT().ListAssignees(gomock.Any(), ref).Return(nil, nil)
	issues.EXPECT().ListLabels(gomock.Any(), ref).Return(nil, nil)

	res, out, err := newTestServer(store).getIssue(toolCtx(), nil, getIssueInput{Ref: ref})
	if err != nil {
		t.Fatalf("getIssue: %v", err)
	}
	if res != nil {
		t.Fatalf("unexpected error result: %+v", res)
	}
	got, ok := out.(issueOutput)
	if !ok {
		t.Fatalf("output type = %T, want issueOutput", out)
	}
	if got.ID != ref || got.Identifier != "ENG-1" {
		t.Errorf("output = %+v, want id=%s identifier=ENG-1", got, ref)
	}
	if got.Assignees == nil || got.Labels == nil {
		t.Errorf("assignees/labels must be non-nil slices: %+v", got)
	}
}

func TestGetIssue_NotFoundMapsToMCPError(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := repomocks.NewMockStore(ctrl)
	issues := repomocks.NewMockIssueRepository(ctrl)
	store.EXPECT().Issue().Return(issues).AnyTimes()

	const ref = "11111111-1111-1111-1111-111111111111"
	issues.EXPECT().GetByID(gomock.Any(), "ws-1", ref).
		Return(model.IssueDetails{}, errs.NotFound("issue not found"))

	res, out, err := newTestServer(store).getIssue(toolCtx(), nil, getIssueInput{Ref: ref})
	if err != nil {
		t.Fatalf("getIssue returned protocol error: %v", err)
	}
	if out != nil {
		t.Fatalf("unexpected output: %+v", out)
	}
	if body := errBody(t, res); body.Code != string(errs.CodeNotFound) {
		t.Errorf("code = %q, want not_found", body.Code)
	}
}

// --- update_issue ---

func TestUpdateIssue_MapsAssigneesReplaceSet(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := repomocks.NewMockStore(ctrl)
	issues := repomocks.NewMockIssueRepository(ctrl)
	teams := repomocks.NewMockTeamRepository(ctrl)
	store.EXPECT().Issue().Return(issues).AnyTimes()
	store.EXPECT().Team().Return(teams).AnyTimes()
	expectTx(store)

	const ref = "11111111-1111-1111-1111-111111111111"
	issues.EXPECT().GetByID(gomock.Any(), "ws-1", ref).Return(model.IssueDetails{
		Issue:      model.Issue{ID: ref, TeamID: "team-1", Title: "old", Priority: model.PriorityNone},
		Identifier: "ENG-1",
		TeamKey:    "ENG",
	}, nil)
	// GetIssue внутри UpdateIssue обогащает assignees/labels.
	issues.EXPECT().ListAssignees(gomock.Any(), ref).Return(nil, nil).AnyTimes()
	issues.EXPECT().ListLabels(gomock.Any(), ref).Return(nil, nil).AnyTimes()
	teams.EXPECT().GetMember(gomock.Any(), "team-1", "user-1").
		Return(model.TeamMember{TeamID: "team-1", UserID: "user-1"}, nil)
	// Валидация набора ассайни: каждый должен быть участником команды.
	teams.EXPECT().GetMember(gomock.Any(), "team-1", "user-2").
		Return(model.TeamMember{TeamID: "team-1", UserID: "user-2"}, nil)

	issues.EXPECT().Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, i model.Issue) (model.Issue, error) { return i, nil })
	// replace-семантика набора ассайни: upsert нового + primary отдельным шагом.
	var addedUser string
	var primaryUser string
	issues.EXPECT().AddAssignee(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, a model.IssueAssignee) (model.IssueAssignee, error) {
			addedUser = a.UserID
			return a, nil
		})
	issues.EXPECT().ClearPrimary(gomock.Any(), ref).Return(nil)
	issues.EXPECT().SetPrimary(gomock.Any(), ref, gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, userID string) error {
			primaryUser = userID
			return nil
		})

	title := "new title"
	res, out, err := newTestServer(store).updateIssue(toolCtx(), nil, updateIssueInput{
		Ref:       ref,
		Title:     &title,
		Assignees: &[]assigneeInput{{UserID: "user-2", IsPrimary: true}},
	})
	if err != nil {
		t.Fatalf("updateIssue: %v", err)
	}
	if res != nil {
		t.Fatalf("unexpected error result: %+v", res)
	}
	if addedUser != "user-2" || primaryUser != "user-2" {
		t.Errorf("assignee mapping: added=%q primary=%q, want user-2", addedUser, primaryUser)
	}
	if got, ok := out.(issueOutput); !ok || got.Title != "new title" {
		t.Errorf("output = %+v (%T), want title=new title", out, out)
	}
}

// --- delete_issue ---

func TestDeleteIssue_ReturnsDeleted(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := repomocks.NewMockStore(ctrl)
	issues := repomocks.NewMockIssueRepository(ctrl)
	teams := repomocks.NewMockTeamRepository(ctrl)
	store.EXPECT().Issue().Return(issues).AnyTimes()
	store.EXPECT().Team().Return(teams).AnyTimes()

	const ref = "11111111-1111-1111-1111-111111111111"
	issues.EXPECT().GetByID(gomock.Any(), "ws-1", ref).Return(model.IssueDetails{
		Issue:      model.Issue{ID: ref, TeamID: "team-1"},
		Identifier: "ENG-1",
		TeamKey:    "ENG",
	}, nil)
	issues.EXPECT().ListAssignees(gomock.Any(), ref).Return(nil, nil).AnyTimes()
	issues.EXPECT().ListLabels(gomock.Any(), ref).Return(nil, nil).AnyTimes()
	teams.EXPECT().GetMember(gomock.Any(), "team-1", "user-1").
		Return(model.TeamMember{TeamID: "team-1", UserID: "user-1"}, nil)
	issues.EXPECT().Delete(gomock.Any(), ref).Return(nil)

	res, out, err := newTestServer(store).deleteIssue(toolCtx(), nil, deleteIssueInput{Ref: ref})
	if err != nil {
		t.Fatalf("deleteIssue: %v", err)
	}
	if res != nil {
		t.Fatalf("unexpected error result: %+v", res)
	}
	got, ok := out.(deleteIssueOutput)
	if !ok || !got.Deleted || got.Ref != ref {
		t.Errorf("output = %+v (%T), want deleted ref=%s", out, out, ref)
	}
}

func TestDeleteIssue_ForbiddenMapsToMCPError(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := repomocks.NewMockStore(ctrl)
	issues := repomocks.NewMockIssueRepository(ctrl)
	teams := repomocks.NewMockTeamRepository(ctrl)
	store.EXPECT().Issue().Return(issues).AnyTimes()
	store.EXPECT().Team().Return(teams).AnyTimes()

	const ref = "11111111-1111-1111-1111-111111111111"
	issues.EXPECT().GetByID(gomock.Any(), "ws-1", ref).Return(model.IssueDetails{
		Issue:      model.Issue{ID: ref, TeamID: "team-1"},
		Identifier: "ENG-1",
		TeamKey:    "ENG",
	}, nil)
	issues.EXPECT().ListAssignees(gomock.Any(), ref).Return(nil, nil).AnyTimes()
	issues.EXPECT().ListLabels(gomock.Any(), ref).Return(nil, nil).AnyTimes()
	// Не участник команды → requireTeamMember вернёт Forbidden.
	teams.EXPECT().GetMember(gomock.Any(), "team-1", "user-1").
		Return(model.TeamMember{}, errs.NotFound("resource not found"))

	res, _, err := newTestServer(store).deleteIssue(toolCtx(), nil, deleteIssueInput{Ref: ref})
	if err != nil {
		t.Fatalf("deleteIssue returned protocol error: %v", err)
	}
	if body := errBody(t, res); body.Code != string(errs.CodeForbidden) {
		t.Errorf("code = %q, want forbidden", body.Code)
	}
}

// --- search_issues ---

func TestSearchIssues_PassesFiltersAndPagination(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := repomocks.NewMockStore(ctrl)
	issues := repomocks.NewMockIssueRepository(ctrl)
	store.EXPECT().Issue().Return(issues).AnyTimes()

	var gotFilter model.IssueFilter
	var gotPage model.Page
	issues.EXPECT().List(gomock.Any(), "ws-1", gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, f model.IssueFilter, p model.Page) ([]model.IssueDetails, error) {
			gotFilter = f
			gotPage = p
			return nil, nil
		})

	team := "22222222-2222-2222-2222-222222222222"
	priority := "urgent"
	q := "login"
	res, out, err := newTestServer(store).searchIssues(toolCtx(), nil, searchIssuesInput{
		Team:     &team,
		Priority: &priority,
		Query:    &q,
		Limit:    25,
		Sort:     "created_at",
		Order:    "asc",
	})
	if err != nil {
		t.Fatalf("searchIssues: %v", err)
	}
	if res != nil {
		t.Fatalf("unexpected error result: %+v", res)
	}
	if gotFilter.TeamID == nil || gotFilter.TeamID.String() != team {
		t.Errorf("filter.TeamID = %v, want %s", gotFilter.TeamID, team)
	}
	if gotFilter.Priority == nil || *gotFilter.Priority != model.PriorityUrgent {
		t.Errorf("filter.Priority = %v, want urgent", gotFilter.Priority)
	}
	if gotFilter.Query == nil || *gotFilter.Query != "login" {
		t.Errorf("filter.Query = %v, want login", gotFilter.Query)
	}
	if gotPage.Limit != 25 || gotPage.Sort != "created_at" || gotPage.Order != model.SortAsc {
		t.Errorf("page = %+v, want limit=25 sort=created_at order=asc", gotPage)
	}
	if _, ok := out.(searchIssuesOutput); !ok {
		t.Fatalf("output type = %T, want searchIssuesOutput", out)
	}
}

func TestSearchIssues_InvalidPriorityIsValidationError(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := repomocks.NewMockStore(ctrl)
	// List не вызывается: фильтр падает на валидации приоритета.

	priority := "sky-high"
	res, _, err := newTestServer(store).searchIssues(toolCtx(), nil, searchIssuesInput{Priority: &priority})
	if err != nil {
		t.Fatalf("searchIssues returned protocol error: %v", err)
	}
	if body := errBody(t, res); body.Code != string(errs.CodeValidation) {
		t.Errorf("code = %q, want validation", body.Code)
	}
}

func TestSearchIssues_EncodesNextCursor(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := repomocks.NewMockStore(ctrl)
	issues := repomocks.NewMockIssueRepository(ctrl)
	store.EXPECT().Issue().Return(issues).AnyTimes()

	// limit=1 + «лишняя» строка → HasMore, курсор строится по последней строке.
	rows := []model.IssueDetails{
		{Issue: model.Issue{ID: "i-1"}, Identifier: "ENG-1"},
		{Issue: model.Issue{ID: "i-2"}, Identifier: "ENG-2"},
	}
	issues.EXPECT().List(gomock.Any(), "ws-1", gomock.Any(), gomock.Any()).Return(rows, nil)
	issues.EXPECT().ListAssigneesByIssues(gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()
	issues.EXPECT().ListLabelsByIssues(gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()

	_, out, err := newTestServer(store).searchIssues(toolCtx(), nil, searchIssuesInput{Limit: 1})
	if err != nil {
		t.Fatalf("searchIssues: %v", err)
	}
	got := out.(searchIssuesOutput)
	if !got.Pagination.HasMore || got.Pagination.NextCursor == "" {
		t.Errorf("pagination = %+v, want has_more + next_cursor", got.Pagination)
	}
	if len(got.Data) != 1 {
		t.Errorf("data len = %d, want 1", len(got.Data))
	}
}

// --- list_projects / list_teams ---

func TestListProjects_ReturnsData(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := repomocks.NewMockStore(ctrl)
	projects := repomocks.NewMockProjectRepository(ctrl)
	store.EXPECT().Project().Return(projects).AnyTimes()

	projects.EXPECT().List(gomock.Any(), "ws-1", gomock.Any()).Return([]model.Project{
		{ID: "p-1", WorkspaceID: "ws-1", Name: "Alpha", Status: model.ProjectStatusPlanned},
	}, nil)
	projects.EXPECT().ListTeamsByProjects(gomock.Any(), gomock.Any()).
		Return(map[string][]model.ProjectTeam{}, nil)

	res, out, err := newTestServer(store).listProjects(toolCtx(), nil, listProjectsInput{})
	if err != nil {
		t.Fatalf("listProjects: %v", err)
	}
	if res != nil {
		t.Fatalf("unexpected error result: %+v", res)
	}
	got, ok := out.(listProjectsOutput)
	if !ok || len(got.Data) != 1 || got.Data[0].ID != "p-1" || got.Data[0].TeamIDs == nil {
		t.Errorf("output = %+v (%T), want one project p-1 with non-nil team_ids", out, out)
	}
}

func TestListTeams_ReturnsData(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := repomocks.NewMockStore(ctrl)
	teams := repomocks.NewMockTeamRepository(ctrl)
	store.EXPECT().Team().Return(teams).AnyTimes()

	teams.EXPECT().List(gomock.Any(), "ws-1").Return([]model.Team{
		{ID: "team-1", WorkspaceID: "ws-1", Name: "Eng", Key: "ENG"},
	}, nil)

	res, out, err := newTestServer(store).listTeams(toolCtx(), nil, listTeamsInput{})
	if err != nil {
		t.Fatalf("listTeams: %v", err)
	}
	if res != nil {
		t.Fatalf("unexpected error result: %+v", res)
	}
	got, ok := out.(listTeamsOutput)
	if !ok || len(got.Data) != 1 || got.Data[0].Key != "ENG" {
		t.Errorf("output = %+v (%T), want one team ENG", out, out)
	}
}

// --- error mapping ---

func TestRespondMCPError_MapsDomainCodes(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want string
	}{
		{"validation", errs.Validation("bad"), string(errs.CodeValidation)},
		{"forbidden", errs.Forbidden("no"), string(errs.CodeForbidden)},
		{"not_found", errs.NotFound("missing"), string(errs.CodeNotFound)},
		{"unauthorized", errs.Unauthorized("nope"), string(errs.CodeUnauthorized)},
		{"non-app-error", context.Canceled, string(errs.CodeInternal)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			body := errBody(t, respondMCPError(tc.err))
			if body.Code != tc.want {
				t.Errorf("code = %q, want %q", body.Code, tc.want)
			}
		})
	}
}

func TestRespondMCPError_IncludesValidationFields(t *testing.T) {
	err := errs.Validation("invalid", errs.FieldError{Field: "title", Reason: "required", Message: "title is required"})
	body := errBody(t, respondMCPError(err))
	if len(body.Fields) != 1 || body.Fields[0].Field != "title" {
		t.Errorf("fields = %+v, want one field 'title'", body.Fields)
	}
}
