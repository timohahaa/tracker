package service

import (
	"context"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/timohahaa/tracker/internal/errs"
	"github.com/timohahaa/tracker/internal/model"
	"github.com/timohahaa/tracker/internal/repository"
	repomocks "github.com/timohahaa/tracker/internal/repository/mocks"
)

// Тесты IssueService написаны от контракта слайса issue (PLAN.md 2.4 заход 1,
// docs/tech/data-model.md «Issue Identifier Generation», docs/tech/api.md):
// создание задачи в одной транзакции выдаёт issue_number атомарным инкрементом
// и вставляет задачу; дефолтный статус — первый backlog команды; валидации
// status_id/project_id; авторизация по team membership. Наблюдаемое поведение —
// входы репозиториев и коды ошибок; негативные инварианты — отсутствием
// ожиданий (лишний вызов роняет тест).

func newIssueService(store repository.Store) *IssueService {
	return &IssueService{repos: store}
}

func issueActor() model.WorkspaceMember {
	return model.WorkspaceMember{WorkspaceID: "ws-1", UserID: "user-1", Role: model.WorkspaceRoleMember}
}

func teamMember() model.TeamMember {
	return model.TeamMember{TeamID: "team-1", UserID: "user-1", Role: model.TeamRoleMember}
}

// --- CreateIssue ---

func TestCreateIssue_GeneratesNumberAndDefaultsToBacklogStatus(t *testing.T) {
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
	teams.EXPECT().GetMember(gomock.Any(), "team-1", "user-1").Return(teamMember(), nil)

	// List упорядочен по (group, position): первый backlog — дефолтный.
	statuses.EXPECT().List(gomock.Any(), "team-1").Return([]model.Status{
		{ID: "st-planned", TeamID: "team-1", Group: model.StatusGroupPlanned, Position: 0},
		{ID: "st-backlog", TeamID: "team-1", Group: model.StatusGroupBacklog, Position: 0},
		{ID: "st-backlog-2", TeamID: "team-1", Group: model.StatusGroupBacklog, Position: 1},
	}, nil)

	issues.EXPECT().IncrementIssueNumber(gomock.Any(), "team-1").Return(1, nil)

	var created model.Issue
	issues.EXPECT().Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, i model.Issue) (model.Issue, error) {
			created = i
			i.ID = "issue-1"
			return i, nil
		})

	got, err := newIssueService(store).CreateIssue(context.Background(), issueActor(), CreateIssueParams{
		TeamID: "team-1",
		Title:  "Fix bug",
	})
	if err != nil {
		t.Fatalf("CreateIssue: %v", err)
	}
	if created.IssueNumber != 1 {
		t.Errorf("issue_number = %d, want 1", created.IssueNumber)
	}
	if created.StatusID != "st-backlog" {
		t.Errorf("status_id = %q, want st-backlog", created.StatusID)
	}
	if created.CreatorID == nil || *created.CreatorID != "user-1" {
		t.Errorf("creator_id = %v, want user-1", created.CreatorID)
	}
	if created.Priority != model.PriorityNone {
		t.Errorf("priority = %q, want none", created.Priority)
	}
	if got.Identifier != "ENG-1" || got.TeamKey != "ENG" {
		t.Errorf("details = (%q, %q), want (ENG-1, ENG)", got.Identifier, got.TeamKey)
	}
}

func TestCreateIssue_UsesGivenStatus(t *testing.T) {
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
	teams.EXPECT().GetMember(gomock.Any(), "team-1", "user-1").Return(teamMember(), nil)

	statuses.EXPECT().Get(gomock.Any(), "team-1", "st-custom").
		Return(model.Status{ID: "st-custom", TeamID: "team-1", Group: model.StatusGroupInProgress}, nil)
	// Status().List не вызывается: статус задан явно.

	issues.EXPECT().IncrementIssueNumber(gomock.Any(), "team-1").Return(3, nil)

	var created model.Issue
	issues.EXPECT().Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, i model.Issue) (model.Issue, error) {
			created = i
			return i, nil
		})

	status := "st-custom"
	if _, err := newIssueService(store).CreateIssue(context.Background(), issueActor(), CreateIssueParams{
		TeamID:   "team-1",
		Title:    "Task",
		StatusID: &status,
	}); err != nil {
		t.Fatalf("CreateIssue: %v", err)
	}
	if created.StatusID != "st-custom" || created.IssueNumber != 3 {
		t.Errorf("created = (%q, %d), want (st-custom, 3)", created.StatusID, created.IssueNumber)
	}
}

func TestCreateIssue_InvalidStatusValidation(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := repomocks.NewMockStore(ctrl)
	teams := repomocks.NewMockTeamRepository(ctrl)
	statuses := repomocks.NewMockStatusRepository(ctrl)
	store.EXPECT().Team().Return(teams).AnyTimes()
	store.EXPECT().Status().Return(statuses).AnyTimes()

	teams.EXPECT().Get(gomock.Any(), "ws-1", "team-1").
		Return(model.Team{ID: "team-1", WorkspaceID: "ws-1", Key: "ENG"}, nil)
	teams.EXPECT().GetMember(gomock.Any(), "team-1", "user-1").Return(teamMember(), nil)
	statuses.EXPECT().Get(gomock.Any(), "team-1", "st-x").Return(model.Status{}, errs.NotFound("resource not found"))
	// Ни транзакции, ни Create: статус не принадлежит команде.

	status := "st-x"
	_, err := newIssueService(store).CreateIssue(context.Background(), issueActor(), CreateIssueParams{
		TeamID:   "team-1",
		Title:    "Task",
		StatusID: &status,
	})
	assertAppErrCode(t, err, errs.CodeValidation)
}

func TestCreateIssue_InvalidProjectValidation(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := repomocks.NewMockStore(ctrl)
	teams := repomocks.NewMockTeamRepository(ctrl)
	statuses := repomocks.NewMockStatusRepository(ctrl)
	issues := repomocks.NewMockIssueRepository(ctrl)
	store.EXPECT().Team().Return(teams).AnyTimes()
	store.EXPECT().Status().Return(statuses).AnyTimes()
	store.EXPECT().Issue().Return(issues).AnyTimes()

	teams.EXPECT().Get(gomock.Any(), "ws-1", "team-1").
		Return(model.Team{ID: "team-1", WorkspaceID: "ws-1", Key: "ENG"}, nil)
	teams.EXPECT().GetMember(gomock.Any(), "team-1", "user-1").Return(teamMember(), nil)
	statuses.EXPECT().Get(gomock.Any(), "team-1", "st-1").
		Return(model.Status{ID: "st-1", TeamID: "team-1"}, nil)
	issues.EXPECT().ProjectInWorkspace(gomock.Any(), "ws-1", "proj-x").Return(false, nil)
	// Ни транзакции, ни Create: проект не принадлежит workspace.

	status := "st-1"
	project := "proj-x"
	_, err := newIssueService(store).CreateIssue(context.Background(), issueActor(), CreateIssueParams{
		TeamID:    "team-1",
		Title:     "Task",
		StatusID:  &status,
		ProjectID: &project,
	})
	assertAppErrCode(t, err, errs.CodeValidation)
}

func TestCreateIssue_TeamNotFoundPropagated(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := repomocks.NewMockStore(ctrl)
	teams := repomocks.NewMockTeamRepository(ctrl)
	store.EXPECT().Team().Return(teams).AnyTimes()

	teams.EXPECT().Get(gomock.Any(), "ws-1", "missing").Return(model.Team{}, errs.NotFound("resource not found"))
	// GetMember не вызывается: команды нет.

	_, err := newIssueService(store).CreateIssue(context.Background(), issueActor(), CreateIssueParams{
		TeamID: "missing",
		Title:  "Task",
	})
	assertAppErrCode(t, err, errs.CodeNotFound)
}

func TestCreateIssue_ForbiddenWithoutTeamMembership(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := repomocks.NewMockStore(ctrl)
	teams := repomocks.NewMockTeamRepository(ctrl)
	store.EXPECT().Team().Return(teams).AnyTimes()

	teams.EXPECT().Get(gomock.Any(), "ws-1", "team-1").
		Return(model.Team{ID: "team-1", WorkspaceID: "ws-1", Key: "ENG"}, nil)
	teams.EXPECT().GetMember(gomock.Any(), "team-1", "user-1").Return(model.TeamMember{}, errs.NotFound("resource not found"))
	// Ни статусов, ни транзакции: actor не участник команды.

	_, err := newIssueService(store).CreateIssue(context.Background(), issueActor(), CreateIssueParams{
		TeamID: "team-1",
		Title:  "Task",
	})
	assertAppErrCode(t, err, errs.CodeForbidden)
}

// --- GetIssue (UUID vs identifier dispatch) ---

func TestGetIssue_DispatchesByUUID(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := repomocks.NewMockStore(ctrl)
	issues := repomocks.NewMockIssueRepository(ctrl)
	store.EXPECT().Issue().Return(issues).AnyTimes()

	const id = "00000000-0000-0000-0000-000000000001"
	issues.EXPECT().GetByID(gomock.Any(), "ws-1", id).Return(model.IssueDetails{Issue: model.Issue{ID: id}}, nil)
	// GetByIdentifier не вызывается: ref — валидный UUID.

	if _, err := newIssueService(store).GetIssue(context.Background(), "ws-1", id); err != nil {
		t.Fatalf("GetIssue: %v", err)
	}
}

func TestGetIssue_DispatchesByIdentifier(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := repomocks.NewMockStore(ctrl)
	issues := repomocks.NewMockIssueRepository(ctrl)
	store.EXPECT().Issue().Return(issues).AnyTimes()

	// key нормализуется в верхний регистр, номер парсится как int.
	issues.EXPECT().GetByIdentifier(gomock.Any(), "ws-1", "ENG", 7).Return(model.IssueDetails{}, nil)
	// GetByID не вызывается: ref — identifier.

	if _, err := newIssueService(store).GetIssue(context.Background(), "ws-1", "eng-7"); err != nil {
		t.Fatalf("GetIssue: %v", err)
	}
}

func TestGetIssue_BadIdentifierNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := repomocks.NewMockStore(ctrl)
	// Никаких обращений к репозиторию: ref не UUID и не валидный identifier.

	_, err := newIssueService(store).GetIssue(context.Background(), "ws-1", "not-an-identifier")
	assertAppErrCode(t, err, errs.CodeNotFound)
}

// --- UpdateIssue ---

func TestUpdateIssue_ForbiddenWithoutTeamMembership(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := repomocks.NewMockStore(ctrl)
	teams := repomocks.NewMockTeamRepository(ctrl)
	issues := repomocks.NewMockIssueRepository(ctrl)
	store.EXPECT().Team().Return(teams).AnyTimes()
	store.EXPECT().Issue().Return(issues).AnyTimes()

	const id = "00000000-0000-0000-0000-000000000001"
	issues.EXPECT().GetByID(gomock.Any(), "ws-1", id).
		Return(model.IssueDetails{Issue: model.Issue{ID: id, TeamID: "team-1"}}, nil)
	teams.EXPECT().GetMember(gomock.Any(), "team-1", "user-1").Return(model.TeamMember{}, errs.NotFound("resource not found"))
	// Update не вызывается: не участник команды.

	_, err := newIssueService(store).UpdateIssue(context.Background(), issueActor(), id, UpdateIssueParams{Title: strPtr("New")})
	assertAppErrCode(t, err, errs.CodeForbidden)
}

func TestUpdateIssue_AppliesFieldsAndValidatesStatus(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := repomocks.NewMockStore(ctrl)
	teams := repomocks.NewMockTeamRepository(ctrl)
	statuses := repomocks.NewMockStatusRepository(ctrl)
	issues := repomocks.NewMockIssueRepository(ctrl)
	store.EXPECT().Team().Return(teams).AnyTimes()
	store.EXPECT().Status().Return(statuses).AnyTimes()
	store.EXPECT().Issue().Return(issues).AnyTimes()

	const id = "00000000-0000-0000-0000-000000000001"
	issues.EXPECT().GetByID(gomock.Any(), "ws-1", id).
		Return(model.IssueDetails{
			Issue:      model.Issue{ID: id, TeamID: "team-1", Title: "Old", StatusID: "st-old"},
			TeamKey:    "ENG",
			Identifier: "ENG-1",
		}, nil)
	teams.EXPECT().GetMember(gomock.Any(), "team-1", "user-1").Return(teamMember(), nil)
	statuses.EXPECT().Get(gomock.Any(), "team-1", "st-new").
		Return(model.Status{ID: "st-new", TeamID: "team-1"}, nil)

	var saved model.Issue
	issues.EXPECT().Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, i model.Issue) (model.Issue, error) {
			saved = i
			return i, nil
		})

	got, err := newIssueService(store).UpdateIssue(context.Background(), issueActor(), id, UpdateIssueParams{
		Title:    strPtr("New Title"),
		StatusID: strPtr("st-new"),
	})
	if err != nil {
		t.Fatalf("UpdateIssue: %v", err)
	}
	if saved.Title != "New Title" || saved.StatusID != "st-new" {
		t.Errorf("saved = (%q, %q), want (New Title, st-new)", saved.Title, saved.StatusID)
	}
	if got.Identifier != "ENG-1" {
		t.Errorf("identifier = %q, want ENG-1", got.Identifier)
	}
}

func TestUpdateIssue_InvalidStatusValidation(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := repomocks.NewMockStore(ctrl)
	teams := repomocks.NewMockTeamRepository(ctrl)
	statuses := repomocks.NewMockStatusRepository(ctrl)
	issues := repomocks.NewMockIssueRepository(ctrl)
	store.EXPECT().Team().Return(teams).AnyTimes()
	store.EXPECT().Status().Return(statuses).AnyTimes()
	store.EXPECT().Issue().Return(issues).AnyTimes()

	const id = "00000000-0000-0000-0000-000000000001"
	issues.EXPECT().GetByID(gomock.Any(), "ws-1", id).
		Return(model.IssueDetails{Issue: model.Issue{ID: id, TeamID: "team-1"}}, nil)
	teams.EXPECT().GetMember(gomock.Any(), "team-1", "user-1").Return(teamMember(), nil)
	statuses.EXPECT().Get(gomock.Any(), "team-1", "st-x").Return(model.Status{}, errs.NotFound("resource not found"))
	// Update не вызывается: новый статус не принадлежит команде.

	_, err := newIssueService(store).UpdateIssue(context.Background(), issueActor(), id, UpdateIssueParams{StatusID: strPtr("st-x")})
	assertAppErrCode(t, err, errs.CodeValidation)
}

// --- DeleteIssue ---

func TestDeleteIssue_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := repomocks.NewMockStore(ctrl)
	teams := repomocks.NewMockTeamRepository(ctrl)
	issues := repomocks.NewMockIssueRepository(ctrl)
	store.EXPECT().Team().Return(teams).AnyTimes()
	store.EXPECT().Issue().Return(issues).AnyTimes()

	const id = "00000000-0000-0000-0000-000000000001"
	issues.EXPECT().GetByID(gomock.Any(), "ws-1", id).
		Return(model.IssueDetails{Issue: model.Issue{ID: id, TeamID: "team-1"}}, nil)
	teams.EXPECT().GetMember(gomock.Any(), "team-1", "user-1").Return(teamMember(), nil)
	issues.EXPECT().Delete(gomock.Any(), id).Return(nil)

	if err := newIssueService(store).DeleteIssue(context.Background(), issueActor(), id); err != nil {
		t.Fatalf("DeleteIssue: %v", err)
	}
}

func TestDeleteIssue_ForbiddenWithoutTeamMembership(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := repomocks.NewMockStore(ctrl)
	teams := repomocks.NewMockTeamRepository(ctrl)
	issues := repomocks.NewMockIssueRepository(ctrl)
	store.EXPECT().Team().Return(teams).AnyTimes()
	store.EXPECT().Issue().Return(issues).AnyTimes()

	const id = "00000000-0000-0000-0000-000000000001"
	issues.EXPECT().GetByID(gomock.Any(), "ws-1", id).
		Return(model.IssueDetails{Issue: model.Issue{ID: id, TeamID: "team-1"}}, nil)
	teams.EXPECT().GetMember(gomock.Any(), "team-1", "user-1").Return(model.TeamMember{}, errs.NotFound("resource not found"))
	// Delete не вызывается: не участник команды.

	err := newIssueService(store).DeleteIssue(context.Background(), issueActor(), id)
	assertAppErrCode(t, err, errs.CodeForbidden)
}
