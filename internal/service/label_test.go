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

// Тесты LabelService написаны от контракта слайса label (PLAN.md 2.6,
// docs/tech/data-model.md, docs/tech/api.md): CRUD меток; авторизация — любой
// участник workspace; валидация hex-цвета; правила уникальности/merge
// (workspace- и team-level). Наблюдаемое поведение — входы репозиториев и коды
// ошибок; негативные инварианты — отсутствием ожиданий (лишний вызов роняет тест).

func newLabelService(store repository.Store) *LabelService {
	return &LabelService{repos: store}
}

func labelActor() model.WorkspaceMember {
	return model.WorkspaceMember{WorkspaceID: "ws-1", UserID: "user-1", Role: model.WorkspaceRoleMember}
}

// --- Валидация цвета (чистый хелпер) ---

func TestIsValidHexColor(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"#3B82F6", true},
		{"#000000", true},
		{"#ffffff", true},
		{"#abcABC", true},
		{"3B82F6", false},   // без #
		{"#3B82F", false},   // 5 цифр
		{"#3B82F60", false}, // 7 цифр
		{"#GGGGGG", false},  // не hex
		{"", false},
		{"#3b82f6 ", false}, // хвостовой пробел
	}
	for _, tc := range cases {
		if got := isValidHexColor(tc.in); got != tc.want {
			t.Errorf("isValidHexColor(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

func TestCreateLabel_InvalidColorValidation(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := repomocks.NewMockStore(ctrl)
	// Никаких обращений к БД: невалидный цвет отсекается первым.

	_, err := newLabelService(store).CreateLabel(context.Background(), labelActor(), CreateLabelParams{
		Name:  "bug",
		Color: "red",
	})
	assertAppErrCode(t, err, errs.CodeValidation)
}

// --- Workspace-level: обычная вставка (нет одноимённых team-меток) ---

func TestCreateLabel_WorkspaceLevelNoMerge(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := repomocks.NewMockStore(ctrl)
	labels := repomocks.NewMockLabelRepository(ctrl)
	store.EXPECT().Label().Return(labels).AnyTimes()

	labels.EXPECT().ListTeamLabelsByName(gomock.Any(), "ws-1", "bug").Return(nil, nil)

	var created model.Label
	labels.EXPECT().Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, l model.Label) (model.Label, error) {
			created = l
			l.ID = "label-1"
			return l, nil
		})

	got, err := newLabelService(store).CreateLabel(context.Background(), labelActor(), CreateLabelParams{
		Name:  "bug",
		Color: "#3B82F6",
	})
	if err != nil {
		t.Fatalf("CreateLabel: %v", err)
	}
	if created.WorkspaceID != "ws-1" || created.TeamID != nil {
		t.Errorf("created = %+v, want ws-1 / team_id nil", created)
	}
	if got.ID != "label-1" {
		t.Errorf("got id = %q, want label-1", got.ID)
	}
}

// --- Workspace-level: merge поверх одноимённых team-меток ---

func TestCreateLabel_WorkspaceLevelMerge(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := repomocks.NewMockStore(ctrl)
	labels := repomocks.NewMockLabelRepository(ctrl)

	expectTx(store)
	store.EXPECT().Label().Return(labels).AnyTimes()

	team1, team2 := "team-1", "team-2"
	labels.EXPECT().ListTeamLabelsByName(gomock.Any(), "ws-1", "bug").Return([]model.Label{
		{ID: "old-1", WorkspaceID: "ws-1", TeamID: &team1, Name: "bug"},
		{ID: "old-2", WorkspaceID: "ws-1", TeamID: &team2, Name: "bug"},
	}, nil)

	labels.EXPECT().Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, l model.Label) (model.Label, error) {
			l.ID = "new-ws"
			return l, nil
		})
	// Перепривязка issue_label со старых меток на новую + soft-delete старых.
	labels.EXPECT().RepointIssueLabels(gomock.Any(), []string{"old-1", "old-2"}, "new-ws").Return(nil)
	labels.EXPECT().Delete(gomock.Any(), "old-1").Return(nil)
	labels.EXPECT().Delete(gomock.Any(), "old-2").Return(nil)

	got, err := newLabelService(store).CreateLabel(context.Background(), labelActor(), CreateLabelParams{
		Name:  "bug",
		Color: "#3B82F6",
	})
	if err != nil {
		t.Fatalf("CreateLabel merge: %v", err)
	}
	if got.ID != "new-ws" || got.TeamID != nil {
		t.Errorf("got = %+v, want new-ws / ws-level", got)
	}
}

// --- Team-level: запрет поверх существующей workspace-level метки ---

func TestCreateLabel_TeamOverWorkspaceForbidden(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := repomocks.NewMockStore(ctrl)
	labels := repomocks.NewMockLabelRepository(ctrl)
	teams := repomocks.NewMockTeamRepository(ctrl)
	store.EXPECT().Label().Return(labels).AnyTimes()
	store.EXPECT().Team().Return(teams).AnyTimes()

	teams.EXPECT().Get(gomock.Any(), "ws-1", "team-1").Return(model.Team{ID: "team-1", WorkspaceID: "ws-1"}, nil)
	// Существует одноимённая ws-level метка → нельзя завести team-метку.
	labels.EXPECT().GetWorkspaceLabelByName(gomock.Any(), "ws-1", "bug").
		Return(model.Label{ID: "ws-label", Name: "bug"}, nil)
	// Create НЕ ожидается.

	teamID := "team-1"
	_, err := newLabelService(store).CreateLabel(context.Background(), labelActor(), CreateLabelParams{
		TeamID: &teamID,
		Name:   "bug",
		Color:  "#3B82F6",
	})
	assertAppErrCode(t, err, errs.CodeValidation)
}

// --- Team-level: валидация принадлежности команды workspace ---

func TestCreateLabel_TeamNotInWorkspaceValidation(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := repomocks.NewMockStore(ctrl)
	teams := repomocks.NewMockTeamRepository(ctrl)
	store.EXPECT().Team().Return(teams).AnyTimes()

	teams.EXPECT().Get(gomock.Any(), "ws-1", "team-x").
		Return(model.Team{}, errs.NotFound("resource not found"))
	// Ни Label().Create, ни GetWorkspaceLabelByName: команда не в workspace.

	teamID := "team-x"
	_, err := newLabelService(store).CreateLabel(context.Background(), labelActor(), CreateLabelParams{
		TeamID: &teamID,
		Name:   "bug",
		Color:  "#3B82F6",
	})
	assertAppErrCode(t, err, errs.CodeValidation)
}

// --- Team-level happy-path ---

func TestCreateLabel_TeamLevelSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := repomocks.NewMockStore(ctrl)
	labels := repomocks.NewMockLabelRepository(ctrl)
	teams := repomocks.NewMockTeamRepository(ctrl)
	store.EXPECT().Label().Return(labels).AnyTimes()
	store.EXPECT().Team().Return(teams).AnyTimes()

	teams.EXPECT().Get(gomock.Any(), "ws-1", "team-1").Return(model.Team{ID: "team-1", WorkspaceID: "ws-1"}, nil)
	labels.EXPECT().GetWorkspaceLabelByName(gomock.Any(), "ws-1", "bug").
		Return(model.Label{}, errs.NotFound("resource not found"))

	var created model.Label
	labels.EXPECT().Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, l model.Label) (model.Label, error) {
			created = l
			l.ID = "label-1"
			return l, nil
		})

	teamID := "team-1"
	if _, err := newLabelService(store).CreateLabel(context.Background(), labelActor(), CreateLabelParams{
		TeamID: &teamID,
		Name:   "bug",
		Color:  "#3B82F6",
	}); err != nil {
		t.Fatalf("CreateLabel team-level: %v", err)
	}
	if created.TeamID == nil || *created.TeamID != "team-1" {
		t.Errorf("created team_id = %v, want team-1", created.TeamID)
	}
}

// --- UpdateLabel: невалидный цвет ---

func TestUpdateLabel_InvalidColorValidation(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := repomocks.NewMockStore(ctrl)
	labels := repomocks.NewMockLabelRepository(ctrl)
	store.EXPECT().Label().Return(labels).AnyTimes()

	labels.EXPECT().GetByID(gomock.Any(), "ws-1", "label-1").
		Return(model.Label{ID: "label-1", WorkspaceID: "ws-1", Name: "bug", Color: "#3B82F6"}, nil)
	// Update НЕ ожидается: невалидный цвет отсекается.

	bad := "not-a-color"
	_, err := newLabelService(store).UpdateLabel(context.Background(), labelActor(), "label-1", UpdateLabelParams{
		Color: &bad,
	})
	assertAppErrCode(t, err, errs.CodeValidation)
}

func TestUpdateLabel_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := repomocks.NewMockStore(ctrl)
	labels := repomocks.NewMockLabelRepository(ctrl)
	store.EXPECT().Label().Return(labels).AnyTimes()

	labels.EXPECT().GetByID(gomock.Any(), "ws-1", "label-1").
		Return(model.Label{ID: "label-1", WorkspaceID: "ws-1", Name: "bug", Color: "#3B82F6"}, nil)

	var saved model.Label
	labels.EXPECT().Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, l model.Label) (model.Label, error) {
			saved = l
			return l, nil
		})

	if _, err := newLabelService(store).UpdateLabel(context.Background(), labelActor(), "label-1", UpdateLabelParams{
		Name:  strPtr("feature"),
		Color: strPtr("#FF0000"),
	}); err != nil {
		t.Fatalf("UpdateLabel: %v", err)
	}
	if saved.Name != "feature" || saved.Color != "#FF0000" {
		t.Errorf("saved = %+v, want feature / #FF0000", saved)
	}
}

// --- DeleteLabel ---

func TestDeleteLabel_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := repomocks.NewMockStore(ctrl)
	labels := repomocks.NewMockLabelRepository(ctrl)
	store.EXPECT().Label().Return(labels).AnyTimes()

	labels.EXPECT().GetByID(gomock.Any(), "ws-1", "label-1").
		Return(model.Label{ID: "label-1", WorkspaceID: "ws-1"}, nil)
	labels.EXPECT().Delete(gomock.Any(), "label-1").Return(nil)

	if err := newLabelService(store).DeleteLabel(context.Background(), labelActor(), "label-1"); err != nil {
		t.Fatalf("DeleteLabel: %v", err)
	}
}
