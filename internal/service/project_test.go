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

// Тесты ProjectService написаны от контракта слайса project (PLAN.md 2.5,
// docs/tech/data-model.md, docs/tech/api.md): CRUD проектов; авторизация
// (create/get/list/update — участник workspace, delete — owner/admin);
// валидация status/lead_id/team_ids; replace-семантика команд в транзакции.
// Наблюдаемое поведение — входы репозиториев и коды ошибок; негативные
// инварианты — отсутствием ожиданий (лишний вызов роняет тест).

func newProjectService(store repository.Store) *ProjectService {
	return &ProjectService{repos: store}
}

func projectActor(role model.WorkspaceRole) model.WorkspaceMember {
	return model.WorkspaceMember{WorkspaceID: "ws-1", UserID: "user-1", Role: role}
}

// --- CreateProject ---

func TestCreateProject_DefaultsStatusAndBindsTeams(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := repomocks.NewMockStore(ctrl)
	teams := repomocks.NewMockTeamRepository(ctrl)
	projects := repomocks.NewMockProjectRepository(ctrl)

	expectTx(store)
	store.EXPECT().Team().Return(teams).AnyTimes()
	store.EXPECT().Project().Return(projects).AnyTimes()

	// team_ids валидируются до транзакции.
	teams.EXPECT().Get(gomock.Any(), "ws-1", "team-1").
		Return(model.Team{ID: "team-1", WorkspaceID: "ws-1"}, nil)

	var created model.Project
	projects.EXPECT().Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, p model.Project) (model.Project, error) {
			created = p
			p.ID = "proj-1"
			return p, nil
		})

	// replaceProjectTeams: текущий набор пуст → только AddTeam для целевого.
	projects.EXPECT().ListTeams(gomock.Any(), "proj-1").Return(nil, nil)
	projects.EXPECT().AddTeam(gomock.Any(), "proj-1", "team-1").Return(model.ProjectTeam{ProjectID: "proj-1", TeamID: "team-1"}, nil)
	// enrichTeams после транзакции.
	projects.EXPECT().ListTeams(gomock.Any(), "proj-1").Return([]model.ProjectTeam{{ProjectID: "proj-1", TeamID: "team-1"}}, nil)

	teamIDs := []string{"team-1"}
	got, err := newProjectService(store).CreateProject(context.Background(), projectActor(model.WorkspaceRoleMember), CreateProjectParams{
		Name:    "Roadmap",
		TeamIDs: &teamIDs,
	})
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}
	if created.WorkspaceID != "ws-1" {
		t.Errorf("created workspace_id = %q, want ws-1", created.WorkspaceID)
	}
	if created.Status != model.ProjectStatusPlanned {
		t.Errorf("created status = %q, want planned (default)", created.Status)
	}
	if len(got.TeamIDs) != 1 || got.TeamIDs[0] != "team-1" {
		t.Errorf("team_ids = %v, want [team-1]", got.TeamIDs)
	}
}

func TestCreateProject_InvalidStatusValidation(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := repomocks.NewMockStore(ctrl)
	// Никаких обращений к БД/транзакции: невалидный статус отсекается до WithTx.

	_, err := newProjectService(store).CreateProject(context.Background(), projectActor(model.WorkspaceRoleMember), CreateProjectParams{
		Name:   "Bad",
		Status: model.ProjectStatus("nonsense"),
	})
	assertAppErrCode(t, err, errs.CodeValidation)
}

func TestCreateProject_InvalidLeadValidation(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := repomocks.NewMockStore(ctrl)
	workspaces := repomocks.NewMockWorkspaceRepository(ctrl)
	store.EXPECT().Workspace().Return(workspaces).AnyTimes()

	workspaces.EXPECT().GetMember(gomock.Any(), "ws-1", "lead-x").
		Return(model.WorkspaceMember{}, errs.NotFound("resource not found"))
	// Ни транзакции, ни Create: lead не участник workspace.

	lead := "lead-x"
	_, err := newProjectService(store).CreateProject(context.Background(), projectActor(model.WorkspaceRoleMember), CreateProjectParams{
		Name:   "Roadmap",
		LeadID: &lead,
	})
	assertAppErrCode(t, err, errs.CodeValidation)
}

func TestCreateProject_InvalidTeamValidation(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := repomocks.NewMockStore(ctrl)
	teams := repomocks.NewMockTeamRepository(ctrl)
	store.EXPECT().Team().Return(teams).AnyTimes()

	teams.EXPECT().Get(gomock.Any(), "ws-1", "team-x").
		Return(model.Team{}, errs.NotFound("resource not found"))
	// Ни транзакции, ни Create: команда не принадлежит workspace.

	teamIDs := []string{"team-x"}
	_, err := newProjectService(store).CreateProject(context.Background(), projectActor(model.WorkspaceRoleMember), CreateProjectParams{
		Name:    "Roadmap",
		TeamIDs: &teamIDs,
	})
	assertAppErrCode(t, err, errs.CodeValidation)
}

// --- GetProject ---

func TestGetProject_EnrichesTeams(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := repomocks.NewMockStore(ctrl)
	projects := repomocks.NewMockProjectRepository(ctrl)
	store.EXPECT().Project().Return(projects).AnyTimes()

	projects.EXPECT().GetByID(gomock.Any(), "ws-1", "proj-1").
		Return(model.Project{ID: "proj-1", WorkspaceID: "ws-1", Name: "Roadmap"}, nil)
	projects.EXPECT().ListTeams(gomock.Any(), "proj-1").
		Return([]model.ProjectTeam{{ProjectID: "proj-1", TeamID: "team-1"}, {ProjectID: "proj-1", TeamID: "team-2"}}, nil)

	got, err := newProjectService(store).GetProject(context.Background(), "ws-1", "proj-1")
	if err != nil {
		t.Fatalf("GetProject: %v", err)
	}
	if len(got.TeamIDs) != 2 {
		t.Errorf("team_ids = %v, want 2 teams", got.TeamIDs)
	}
}

// --- UpdateProject ---

func TestUpdateProject_ReplacesTeamsInTx(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := repomocks.NewMockStore(ctrl)
	teams := repomocks.NewMockTeamRepository(ctrl)
	projects := repomocks.NewMockProjectRepository(ctrl)

	expectTx(store)
	store.EXPECT().Team().Return(teams).AnyTimes()
	store.EXPECT().Project().Return(projects).AnyTimes()

	projects.EXPECT().GetByID(gomock.Any(), "ws-1", "proj-1").
		Return(model.Project{ID: "proj-1", WorkspaceID: "ws-1", Name: "Old", Status: model.ProjectStatusPlanned}, nil)
	// team_ids валидируются до транзакции.
	teams.EXPECT().Get(gomock.Any(), "ws-1", "team-2").Return(model.Team{ID: "team-2", WorkspaceID: "ws-1"}, nil)

	var saved model.Project
	projects.EXPECT().Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, p model.Project) (model.Project, error) {
			saved = p
			return p, nil
		})

	// replaceProjectTeams: текущий набор {team-1}, целевой {team-2} →
	// отвязать team-1, привязать team-2.
	projects.EXPECT().ListTeams(gomock.Any(), "proj-1").
		Return([]model.ProjectTeam{{ProjectID: "proj-1", TeamID: "team-1"}}, nil)
	projects.EXPECT().RemoveTeam(gomock.Any(), "proj-1", "team-1").Return(nil)
	projects.EXPECT().AddTeam(gomock.Any(), "proj-1", "team-2").Return(model.ProjectTeam{ProjectID: "proj-1", TeamID: "team-2"}, nil)
	// enrichTeams после транзакции.
	projects.EXPECT().ListTeams(gomock.Any(), "proj-1").
		Return([]model.ProjectTeam{{ProjectID: "proj-1", TeamID: "team-2"}}, nil)

	teamIDs := []string{"team-2"}
	got, err := newProjectService(store).UpdateProject(context.Background(), projectActor(model.WorkspaceRoleMember), "proj-1", UpdateProjectParams{
		Name:    strPtr("New Name"),
		TeamIDs: &teamIDs,
	})
	if err != nil {
		t.Fatalf("UpdateProject: %v", err)
	}
	if saved.Name != "New Name" {
		t.Errorf("saved name = %q, want New Name", saved.Name)
	}
	if len(got.TeamIDs) != 1 || got.TeamIDs[0] != "team-2" {
		t.Errorf("team_ids = %v, want [team-2]", got.TeamIDs)
	}
}

func TestUpdateProject_SimplePathWithoutTeams(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := repomocks.NewMockStore(ctrl)
	projects := repomocks.NewMockProjectRepository(ctrl)
	store.EXPECT().Project().Return(projects).AnyTimes()
	// WithTx НЕ ожидается: team_ids не заданы → одиночный UPDATE без транзакции.

	projects.EXPECT().GetByID(gomock.Any(), "ws-1", "proj-1").
		Return(model.Project{ID: "proj-1", WorkspaceID: "ws-1", Status: model.ProjectStatusPlanned}, nil)

	var saved model.Project
	projects.EXPECT().Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, p model.Project) (model.Project, error) {
			saved = p
			return p, nil
		})
	projects.EXPECT().ListTeams(gomock.Any(), "proj-1").Return(nil, nil)

	if _, err := newProjectService(store).UpdateProject(context.Background(), projectActor(model.WorkspaceRoleMember), "proj-1", UpdateProjectParams{
		Status: statusPtr(model.ProjectStatusInProgress),
	}); err != nil {
		t.Fatalf("UpdateProject: %v", err)
	}
	if saved.Status != model.ProjectStatusInProgress {
		t.Errorf("saved status = %q, want in_progress", saved.Status)
	}
}

// --- RemoveLead ---

func TestRemoveLead_ClearsLead(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := repomocks.NewMockStore(ctrl)
	projects := repomocks.NewMockProjectRepository(ctrl)
	store.EXPECT().Project().Return(projects).AnyTimes()

	lead := "lead-1"
	projects.EXPECT().GetByID(gomock.Any(), "ws-1", "proj-1").
		Return(model.Project{ID: "proj-1", WorkspaceID: "ws-1", LeadID: &lead}, nil)

	var saved model.Project
	projects.EXPECT().Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, p model.Project) (model.Project, error) {
			saved = p
			return p, nil
		})
	projects.EXPECT().ListTeams(gomock.Any(), "proj-1").Return(nil, nil)

	got, err := newProjectService(store).RemoveLead(context.Background(), projectActor(model.WorkspaceRoleMember), "proj-1")
	if err != nil {
		t.Fatalf("RemoveLead: %v", err)
	}
	if saved.LeadID != nil {
		t.Errorf("saved lead_id = %v, want nil", *saved.LeadID)
	}
	if got.LeadID != nil {
		t.Errorf("returned lead_id = %v, want nil", *got.LeadID)
	}
}

func TestRemoveLead_IdempotentWhenAlreadyNull(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := repomocks.NewMockStore(ctrl)
	projects := repomocks.NewMockProjectRepository(ctrl)
	store.EXPECT().Project().Return(projects).AnyTimes()

	projects.EXPECT().GetByID(gomock.Any(), "ws-1", "proj-1").
		Return(model.Project{ID: "proj-1", WorkspaceID: "ws-1", LeadID: nil}, nil)

	var saved model.Project
	projects.EXPECT().Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, p model.Project) (model.Project, error) {
			saved = p
			return p, nil
		})
	projects.EXPECT().ListTeams(gomock.Any(), "proj-1").Return(nil, nil)

	if _, err := newProjectService(store).RemoveLead(context.Background(), projectActor(model.WorkspaceRoleMember), "proj-1"); err != nil {
		t.Fatalf("RemoveLead: %v", err)
	}
	if saved.LeadID != nil {
		t.Errorf("saved lead_id = %v, want nil", *saved.LeadID)
	}
}

func TestRemoveLead_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := repomocks.NewMockStore(ctrl)
	projects := repomocks.NewMockProjectRepository(ctrl)
	store.EXPECT().Project().Return(projects).AnyTimes()

	projects.EXPECT().GetByID(gomock.Any(), "ws-1", "missing").
		Return(model.Project{}, errs.NotFound("resource not found"))
	// Update не ожидается: проект не найден.

	_, err := newProjectService(store).RemoveLead(context.Background(), projectActor(model.WorkspaceRoleMember), "missing")
	assertAppErrCode(t, err, errs.CodeNotFound)
}

// --- DeleteProject ---

func TestDeleteProject_ForbiddenForNonManager(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := repomocks.NewMockStore(ctrl)
	// Проверка роли идёт первой — обращений к БД нет.

	err := newProjectService(store).DeleteProject(context.Background(), projectActor(model.WorkspaceRoleMember), "proj-1")
	assertAppErrCode(t, err, errs.CodeForbidden)
}

func TestDeleteProject_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := repomocks.NewMockStore(ctrl)
	projects := repomocks.NewMockProjectRepository(ctrl)
	store.EXPECT().Project().Return(projects).AnyTimes()

	projects.EXPECT().GetByID(gomock.Any(), "ws-1", "proj-1").
		Return(model.Project{ID: "proj-1", WorkspaceID: "ws-1"}, nil)
	projects.EXPECT().Delete(gomock.Any(), "proj-1").Return(nil)

	if err := newProjectService(store).DeleteProject(context.Background(), projectActor(model.WorkspaceRoleAdmin), "proj-1"); err != nil {
		t.Fatalf("DeleteProject: %v", err)
	}
}

func statusPtr(s model.ProjectStatus) *model.ProjectStatus { return &s }
