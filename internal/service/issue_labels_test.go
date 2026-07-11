package service

import (
	"context"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/timohahaa/tracker/internal/errs"
	"github.com/timohahaa/tracker/internal/model"
	repomocks "github.com/timohahaa/tracker/internal/repository/mocks"
)

// Тесты label-захода написаны от контракта (docs/tech/data-model.md «labels»/
// «issue_labels», docs/tech/api.md «Labels»): метку можно навесить, только если
// она workspace-level (team_id IS NULL) ИЛИ принадлежит команде задачи; чужая
// team-метка → Validation. Набор дедуплицируется. replace-семантика PATCH:
// nil = не трогать, [] = снять все, набор = заменить (в транзакции). Enrichment
// наполняет IssueDetails.Labels на уровне сервиса. Наблюдаемое поведение — входы
// репозитория и коды ошибок; негативные инварианты — отсутствием ожиданий.

// wireLabelStore разводит моки и настраивает чтение задачи команды team-1.
func wireLabelStore(t *testing.T) (*repomocks.MockStore, *repomocks.MockTeamRepository, *repomocks.MockIssueRepository, *repomocks.MockLabelRepository) {
	t.Helper()
	ctrl := gomock.NewController(t)
	store := repomocks.NewMockStore(ctrl)
	teams := repomocks.NewMockTeamRepository(ctrl)
	issues := repomocks.NewMockIssueRepository(ctrl)
	labels := repomocks.NewMockLabelRepository(ctrl)

	store.EXPECT().Team().Return(teams).AnyTimes()
	store.EXPECT().Issue().Return(issues).AnyTimes()
	store.EXPECT().Label().Return(labels).AnyTimes()

	issues.EXPECT().GetByID(gomock.Any(), "ws-1", issueUUID).
		Return(model.IssueDetails{Issue: model.Issue{ID: issueUUID, TeamID: "team-1"}}, nil).AnyTimes()
	// GetIssue в начале UpdateIssue обогащает ассайни/метки; наборы задаёт тест.
	issues.EXPECT().ListAssignees(gomock.Any(), issueUUID).Return(nil, nil).AnyTimes()

	return store, teams, issues, labels
}

// wsLabel/teamLabel — фабрики меток workspace- и team-уровня.
func wsLabel(id string) model.Label { return model.Label{ID: id, WorkspaceID: "ws-1"} }

func teamLabel(id, teamID string) model.Label {
	return model.Label{ID: id, WorkspaceID: "ws-1", TeamID: &teamID}
}

// --- validateLabelSet (через UpdateIssue) ---

func TestUpdateIssue_LabelSetAcceptsWorkspaceAndOwnTeamLabels(t *testing.T) {
	store, teams, issues, labels := wireLabelStore(t)
	expectTx(store)
	teams.EXPECT().GetMember(gomock.Any(), "team-1", "user-1").Return(teamMember(), nil)

	// ws-level метка и метка команды задачи — обе допустимы.
	labels.EXPECT().GetByID(gomock.Any(), "ws-1", "lbl-ws").Return(wsLabel("lbl-ws"), nil)
	labels.EXPECT().GetByID(gomock.Any(), "ws-1", "lbl-team").Return(teamLabel("lbl-team", "team-1"), nil)

	issues.EXPECT().ListLabels(gomock.Any(), issueUUID).Return(nil, nil).AnyTimes()
	issues.EXPECT().Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, i model.Issue) (model.Issue, error) { return i, nil })

	// Набор привязывается: оба AddIssueLabel, ни одного лишнего RemoveIssueLabel.
	issues.EXPECT().AddIssueLabel(gomock.Any(), issueUUID, "lbl-ws").Return(nil)
	issues.EXPECT().AddIssueLabel(gomock.Any(), issueUUID, "lbl-team").Return(nil)

	set := []string{"lbl-ws", "lbl-team"}
	if _, err := newIssueService(store).UpdateIssue(context.Background(), issueActor(), issueUUID, UpdateIssueParams{LabelIDs: &set}); err != nil {
		t.Fatalf("UpdateIssue: %v", err)
	}
}

func TestUpdateIssue_LabelSetRejectsForeignTeamLabel(t *testing.T) {
	store, teams, issues, labels := wireLabelStore(t)
	issues.EXPECT().ListLabels(gomock.Any(), issueUUID).Return(nil, nil).AnyTimes()
	teams.EXPECT().GetMember(gomock.Any(), "team-1", "user-1").Return(teamMember(), nil)
	// Метка чужой команды → Validation. Ни транзакции, ни Update.
	labels.EXPECT().GetByID(gomock.Any(), "ws-1", "lbl-other").Return(teamLabel("lbl-other", "team-2"), nil)

	set := []string{"lbl-other"}
	_, err := newIssueService(store).UpdateIssue(context.Background(), issueActor(), issueUUID, UpdateIssueParams{LabelIDs: &set})
	assertAppErrCode(t, err, errs.CodeValidation)
}

func TestUpdateIssue_LabelSetRejectsMissingLabel(t *testing.T) {
	store, teams, issues, labels := wireLabelStore(t)
	issues.EXPECT().ListLabels(gomock.Any(), issueUUID).Return(nil, nil).AnyTimes()
	teams.EXPECT().GetMember(gomock.Any(), "team-1", "user-1").Return(teamMember(), nil)
	// Несуществующая метка (NotFound от репозитория) → Validation, не 404.
	labels.EXPECT().GetByID(gomock.Any(), "ws-1", "lbl-x").Return(model.Label{}, errs.NotFound("resource not found"))

	set := []string{"lbl-x"}
	_, err := newIssueService(store).UpdateIssue(context.Background(), issueActor(), issueUUID, UpdateIssueParams{LabelIDs: &set})
	assertAppErrCode(t, err, errs.CodeValidation)
}

func TestUpdateIssue_LabelSetDeduplicates(t *testing.T) {
	store, teams, issues, labels := wireLabelStore(t)
	expectTx(store)
	teams.EXPECT().GetMember(gomock.Any(), "team-1", "user-1").Return(teamMember(), nil)

	// Один и тот же id дважды: валидируется и привязывается ровно один раз.
	labels.EXPECT().GetByID(gomock.Any(), "ws-1", "lbl-ws").Return(wsLabel("lbl-ws"), nil).Times(1)
	issues.EXPECT().ListLabels(gomock.Any(), issueUUID).Return(nil, nil).AnyTimes()
	issues.EXPECT().Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, i model.Issue) (model.Issue, error) { return i, nil })
	issues.EXPECT().AddIssueLabel(gomock.Any(), issueUUID, "lbl-ws").Return(nil).Times(1)

	set := []string{"lbl-ws", "lbl-ws"}
	if _, err := newIssueService(store).UpdateIssue(context.Background(), issueActor(), issueUUID, UpdateIssueParams{LabelIDs: &set}); err != nil {
		t.Fatalf("UpdateIssue: %v", err)
	}
}

// --- replace-семантика ---

func TestUpdateIssue_ReplaceLabelsInTx(t *testing.T) {
	store, teams, issues, labels := wireLabelStore(t)
	expectTx(store)
	teams.EXPECT().GetMember(gomock.Any(), "team-1", "user-1").Return(teamMember(), nil)
	labels.EXPECT().GetByID(gomock.Any(), "ws-1", "lbl-new").Return(wsLabel("lbl-new"), nil)

	// Текущий набор: lbl-old (будет снят, т.к. его нет в новом наборе).
	issues.EXPECT().ListLabels(gomock.Any(), issueUUID).
		Return([]model.Label{wsLabel("lbl-old")}, nil).AnyTimes()
	issues.EXPECT().Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, i model.Issue) (model.Issue, error) { return i, nil })

	issues.EXPECT().RemoveIssueLabel(gomock.Any(), issueUUID, "lbl-old").Return(nil)
	issues.EXPECT().AddIssueLabel(gomock.Any(), issueUUID, "lbl-new").Return(nil)

	set := []string{"lbl-new"}
	if _, err := newIssueService(store).UpdateIssue(context.Background(), issueActor(), issueUUID, UpdateIssueParams{LabelIDs: &set}); err != nil {
		t.Fatalf("UpdateIssue: %v", err)
	}
}

func TestUpdateIssue_ReplaceLabelsEmptyClearsAll(t *testing.T) {
	store, teams, issues, _ := wireLabelStore(t)
	expectTx(store)
	teams.EXPECT().GetMember(gomock.Any(), "team-1", "user-1").Return(teamMember(), nil)

	issues.EXPECT().ListLabels(gomock.Any(), issueUUID).
		Return([]model.Label{wsLabel("lbl-old")}, nil).AnyTimes()
	issues.EXPECT().Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, i model.Issue) (model.Issue, error) { return i, nil })
	// Пустой набор → снять все, ничего не добавлять.
	issues.EXPECT().RemoveIssueLabel(gomock.Any(), issueUUID, "lbl-old").Return(nil)

	empty := []string{}
	if _, err := newIssueService(store).UpdateIssue(context.Background(), issueActor(), issueUUID, UpdateIssueParams{LabelIDs: &empty}); err != nil {
		t.Fatalf("UpdateIssue: %v", err)
	}
}

func TestUpdateIssue_NilLabelsLeavesLinksUntouched(t *testing.T) {
	store, teams, issues, _ := wireLabelStore(t)
	teams.EXPECT().GetMember(gomock.Any(), "team-1", "user-1").Return(teamMember(), nil)

	// Простой путь (без транзакции): метки не заданы → ни Add/Remove, ни tx.
	issues.EXPECT().ListLabels(gomock.Any(), issueUUID).Return(nil, nil).AnyTimes()
	issues.EXPECT().Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, i model.Issue) (model.Issue, error) { return i, nil })

	if _, err := newIssueService(store).UpdateIssue(context.Background(), issueActor(), issueUUID, UpdateIssueParams{Title: strPtr("New")}); err != nil {
		t.Fatalf("UpdateIssue: %v", err)
	}
}

// --- enrichment ---

func TestGetIssue_EnrichesLabels(t *testing.T) {
	store, _, issues, _ := wireLabelStore(t)
	issues.EXPECT().ListLabels(gomock.Any(), issueUUID).
		Return([]model.Label{wsLabel("lbl-1"), teamLabel("lbl-2", "team-1")}, nil)

	got, err := newIssueService(store).GetIssue(context.Background(), "ws-1", issueUUID)
	if err != nil {
		t.Fatalf("GetIssue: %v", err)
	}
	if len(got.Labels) != 2 || got.Labels[0].ID != "lbl-1" || got.Labels[1].ID != "lbl-2" {
		t.Errorf("labels = %+v, want [lbl-1 lbl-2]", got.Labels)
	}
}
