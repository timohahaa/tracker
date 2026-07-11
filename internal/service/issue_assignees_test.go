package service

import (
	"context"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/timohahaa/tracker/internal/errs"
	"github.com/timohahaa/tracker/internal/model"
	repomocks "github.com/timohahaa/tracker/internal/repository/mocks"
)

// Тесты assignee-захода написаны от контракта (docs/tech/data-model.md
// «issue_assignees», docs/tech/api.md «Assignees»): мутации доступны участнику
// команды задачи (requireTeamMember → Forbidden); ответственный ⊆ участников;
// смена ответственного идёт в одной транзакции (ClearPrimary + SetPrimary);
// replace-семантика PATCH; снятие ответственного не тянет авто-передачу.
// Наблюдаемое поведение — входы репозитория и коды ошибок; негативные инварианты
// — отсутствием ожиданий.

// issueUUID — валидный UUID, чтобы GetIssue уходил в GetByID (а не в identifier).
const issueUUID = "00000000-0000-0000-0000-000000000001"

// wireAssigneeStore разводит стандартный набор моков и настраивает чтение задачи
// команды team-1 (GetByID). ListAssignees (использует enrich в GetIssue, а также
// replace) задаёт каждый тест сам — так набор ассайни виден в тесте явно.
func wireAssigneeStore(t *testing.T) (*repomocks.MockStore, *repomocks.MockTeamRepository, *repomocks.MockIssueRepository) {
	t.Helper()
	ctrl := gomock.NewController(t)
	store := repomocks.NewMockStore(ctrl)
	teams := repomocks.NewMockTeamRepository(ctrl)
	issues := repomocks.NewMockIssueRepository(ctrl)

	store.EXPECT().Team().Return(teams).AnyTimes()
	store.EXPECT().Issue().Return(issues).AnyTimes()

	issues.EXPECT().GetByID(gomock.Any(), "ws-1", issueUUID).
		Return(model.IssueDetails{Issue: model.Issue{ID: issueUUID, TeamID: "team-1"}}, nil).AnyTimes()

	// UpdateIssue стартует с GetIssue, который теперь обогащает и метки; метки в
	// этих тестах не проверяются — разрешаем произвольный вызов без набора.
	issues.EXPECT().ListLabels(gomock.Any(), issueUUID).Return(nil, nil).AnyTimes()

	return store, teams, issues
}

// currentAssignees настраивает возврат активных участников задачи (для enrich в
// GetIssue и для diff в replace).
func currentAssignees(issues *repomocks.MockIssueRepository, set ...model.IssueAssignee) {
	issues.EXPECT().ListAssignees(gomock.Any(), issueUUID).Return(set, nil).AnyTimes()
}

// --- AddAssignee ---

func TestAddAssignee_ForbiddenWhenActorNotTeamMember(t *testing.T) {
	store, teams, issues := wireAssigneeStore(t)
	currentAssignees(issues)
	teams.EXPECT().GetMember(gomock.Any(), "team-1", "user-1").Return(model.TeamMember{}, errs.NotFound("resource not found"))
	// Ни AddAssignee, ни валидации ассайни: actor не участник команды.

	_, err := newIssueService(store).AddAssignee(context.Background(), issueActor(), issueUUID, "user-2", false)
	assertAppErrCode(t, err, errs.CodeForbidden)
}

func TestAddAssignee_ValidationWhenUserNotTeamMember(t *testing.T) {
	store, teams, issues := wireAssigneeStore(t)
	currentAssignees(issues)
	teams.EXPECT().GetMember(gomock.Any(), "team-1", "user-1").Return(teamMember(), nil)
	teams.EXPECT().GetMember(gomock.Any(), "team-1", "user-2").Return(model.TeamMember{}, errs.NotFound("resource not found"))
	// AddAssignee не вызывается: назначаемый не активный участник команды.

	_, err := newIssueService(store).AddAssignee(context.Background(), issueActor(), issueUUID, "user-2", false)
	assertAppErrCode(t, err, errs.CodeValidation)
}

func TestAddAssignee_NonPrimaryUpsertsWithoutTx(t *testing.T) {
	store, teams, issues := wireAssigneeStore(t)
	currentAssignees(issues)
	teams.EXPECT().GetMember(gomock.Any(), "team-1", "user-1").Return(teamMember(), nil)
	teams.EXPECT().GetMember(gomock.Any(), "team-1", "user-2").Return(teamMember(), nil)

	var added model.IssueAssignee
	issues.EXPECT().AddAssignee(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, a model.IssueAssignee) (model.IssueAssignee, error) {
			added = a
			return a, nil
		})
	// Ни WithTx, ни ClearPrimary/SetPrimary: primary=false.

	if _, err := newIssueService(store).AddAssignee(context.Background(), issueActor(), issueUUID, "user-2", false); err != nil {
		t.Fatalf("AddAssignee: %v", err)
	}
	if added.IssueID != issueUUID || added.UserID != "user-2" || added.IsPrimary {
		t.Errorf("added = %+v, want issue=%s user=user-2 is_primary=false", added, issueUUID)
	}
}

func TestAddAssignee_PrimaryRunsSwapInTx(t *testing.T) {
	store, teams, issues := wireAssigneeStore(t)
	currentAssignees(issues)
	expectTx(store)
	teams.EXPECT().GetMember(gomock.Any(), "team-1", "user-1").Return(teamMember(), nil)
	teams.EXPECT().GetMember(gomock.Any(), "team-1", "user-2").Return(teamMember(), nil)

	// Ассайни апсертится с is_primary=false, primary выставляется отдельным шагом.
	var added model.IssueAssignee
	issues.EXPECT().AddAssignee(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, a model.IssueAssignee) (model.IssueAssignee, error) {
			added = a
			return a, nil
		})
	issues.EXPECT().ClearPrimary(gomock.Any(), issueUUID).Return(nil)
	issues.EXPECT().SetPrimary(gomock.Any(), issueUUID, "user-2").Return(nil)

	if _, err := newIssueService(store).AddAssignee(context.Background(), issueActor(), issueUUID, "user-2", true); err != nil {
		t.Fatalf("AddAssignee: %v", err)
	}
	if added.IsPrimary {
		t.Errorf("AddAssignee upsert must set is_primary=false, got true")
	}
}

// --- RemoveAssignee ---

func TestRemoveAssignee_SoftDeletesWithoutAutoTransfer(t *testing.T) {
	store, teams, issues := wireAssigneeStore(t)
	currentAssignees(issues)
	teams.EXPECT().GetMember(gomock.Any(), "team-1", "user-1").Return(teamMember(), nil)
	issues.EXPECT().RemoveAssignee(gomock.Any(), issueUUID, "user-2").Return(nil)
	// Ни ClearPrimary, ни SetPrimary: снятие участника (в т.ч. ответственного) не
	// переназначает ответственного автоматически.

	if err := newIssueService(store).RemoveAssignee(context.Background(), issueActor(), issueUUID, "user-2"); err != nil {
		t.Fatalf("RemoveAssignee: %v", err)
	}
}

func TestRemoveAssignee_ForbiddenWhenActorNotTeamMember(t *testing.T) {
	store, teams, issues := wireAssigneeStore(t)
	currentAssignees(issues)
	teams.EXPECT().GetMember(gomock.Any(), "team-1", "user-1").Return(model.TeamMember{}, errs.NotFound("resource not found"))
	// RemoveAssignee не вызывается: actor не участник команды.

	err := newIssueService(store).RemoveAssignee(context.Background(), issueActor(), issueUUID, "user-2")
	assertAppErrCode(t, err, errs.CodeForbidden)
}

// --- SetPrimary ---

func TestSetPrimary_ValidationWhenUserNotAssignee(t *testing.T) {
	store, teams, issues := wireAssigneeStore(t)
	currentAssignees(issues)
	teams.EXPECT().GetMember(gomock.Any(), "team-1", "user-1").Return(teamMember(), nil)
	issues.EXPECT().GetAssignee(gomock.Any(), issueUUID, "user-2").Return(model.IssueAssignee{}, errs.NotFound("resource not found"))
	// Ни транзакции, ни ClearPrimary/SetPrimary: назначаемый не участник задачи.

	_, err := newIssueService(store).SetPrimary(context.Background(), issueActor(), issueUUID, "user-2")
	assertAppErrCode(t, err, errs.CodeValidation)
}

func TestSetPrimary_RunsSwapInTx(t *testing.T) {
	store, teams, issues := wireAssigneeStore(t)
	currentAssignees(issues)
	expectTx(store)
	teams.EXPECT().GetMember(gomock.Any(), "team-1", "user-1").Return(teamMember(), nil)
	issues.EXPECT().GetAssignee(gomock.Any(), issueUUID, "user-2").
		Return(model.IssueAssignee{IssueID: issueUUID, UserID: "user-2"}, nil)
	issues.EXPECT().ClearPrimary(gomock.Any(), issueUUID).Return(nil)
	issues.EXPECT().SetPrimary(gomock.Any(), issueUUID, "user-2").Return(nil)

	if _, err := newIssueService(store).SetPrimary(context.Background(), issueActor(), issueUUID, "user-2"); err != nil {
		t.Fatalf("SetPrimary: %v", err)
	}
}

func TestSetPrimary_ForbiddenWhenActorNotTeamMember(t *testing.T) {
	store, teams, issues := wireAssigneeStore(t)
	currentAssignees(issues)
	teams.EXPECT().GetMember(gomock.Any(), "team-1", "user-1").Return(model.TeamMember{}, errs.NotFound("resource not found"))
	// Ни GetAssignee, ни транзакции: actor не участник команды.

	_, err := newIssueService(store).SetPrimary(context.Background(), issueActor(), issueUUID, "user-2")
	assertAppErrCode(t, err, errs.CodeForbidden)
}

// --- UpdateIssue: replace-семантика assignees ---

func TestUpdateIssue_ReplaceAssigneesInTx(t *testing.T) {
	store, teams, issues := wireAssigneeStore(t)
	expectTx(store)
	// Любой пользователь — активный участник команды.
	teams.EXPECT().GetMember(gomock.Any(), "team-1", gomock.Any()).Return(teamMember(), nil).AnyTimes()

	// Текущий набор: user-3 (будет снят, т.к. его нет в новом наборе).
	currentAssignees(issues, model.IssueAssignee{IssueID: issueUUID, UserID: "user-3"})

	issues.EXPECT().Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, i model.Issue) (model.Issue, error) { return i, nil })

	issues.EXPECT().RemoveAssignee(gomock.Any(), issueUUID, "user-3").Return(nil)
	// Апсерт нового набора всегда с is_primary=false.
	issues.EXPECT().AddAssignee(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, a model.IssueAssignee) (model.IssueAssignee, error) {
			if a.IsPrimary {
				t.Errorf("replace upsert must set is_primary=false, got true for %s", a.UserID)
			}
			return a, nil
		}).Times(2)
	issues.EXPECT().ClearPrimary(gomock.Any(), issueUUID).Return(nil)
	issues.EXPECT().SetPrimary(gomock.Any(), issueUUID, "user-1").Return(nil)

	set := []AssigneeInput{{UserID: "user-1", IsPrimary: true}, {UserID: "user-2"}}
	if _, err := newIssueService(store).UpdateIssue(context.Background(), issueActor(), issueUUID, UpdateIssueParams{Assignees: &set}); err != nil {
		t.Fatalf("UpdateIssue: %v", err)
	}
}

func TestUpdateIssue_ReplaceAssigneesEmptyClearsAll(t *testing.T) {
	store, teams, issues := wireAssigneeStore(t)
	expectTx(store)
	teams.EXPECT().GetMember(gomock.Any(), "team-1", gomock.Any()).Return(teamMember(), nil).AnyTimes()

	currentAssignees(issues, model.IssueAssignee{IssueID: issueUUID, UserID: "user-3", IsPrimary: true})
	issues.EXPECT().Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, i model.Issue) (model.Issue, error) { return i, nil })
	issues.EXPECT().RemoveAssignee(gomock.Any(), issueUUID, "user-3").Return(nil)
	issues.EXPECT().ClearPrimary(gomock.Any(), issueUUID).Return(nil)
	// SetPrimary не вызывается: в наборе нет ответственного.

	empty := []AssigneeInput{}
	if _, err := newIssueService(store).UpdateIssue(context.Background(), issueActor(), issueUUID, UpdateIssueParams{Assignees: &empty}); err != nil {
		t.Fatalf("UpdateIssue: %v", err)
	}
}

func TestUpdateIssue_ReplaceRejectsMultiplePrimaries(t *testing.T) {
	store, teams, issues := wireAssigneeStore(t)
	currentAssignees(issues)
	teams.EXPECT().GetMember(gomock.Any(), "team-1", gomock.Any()).Return(teamMember(), nil).AnyTimes()
	// Ни транзакции, ни Update: два ответственных недопустимо.

	set := []AssigneeInput{{UserID: "user-1", IsPrimary: true}, {UserID: "user-2", IsPrimary: true}}
	_, err := newIssueService(store).UpdateIssue(context.Background(), issueActor(), issueUUID, UpdateIssueParams{Assignees: &set})
	assertAppErrCode(t, err, errs.CodeValidation)
}

func TestUpdateIssue_ReplaceRejectsNonTeamMember(t *testing.T) {
	store, teams, issues := wireAssigneeStore(t)
	currentAssignees(issues)
	teams.EXPECT().GetMember(gomock.Any(), "team-1", "user-1").Return(teamMember(), nil).AnyTimes()
	teams.EXPECT().GetMember(gomock.Any(), "team-1", "user-99").Return(model.TeamMember{}, errs.NotFound("resource not found"))
	// Ни транзакции, ни Update: назначаемый не участник команды.

	set := []AssigneeInput{{UserID: "user-99"}}
	_, err := newIssueService(store).UpdateIssue(context.Background(), issueActor(), issueUUID, UpdateIssueParams{Assignees: &set})
	assertAppErrCode(t, err, errs.CodeValidation)
}
