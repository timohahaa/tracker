package service

import (
	"context"
	"errors"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/timohahaa/tracker/internal/errs"
	"github.com/timohahaa/tracker/internal/model"
	"github.com/timohahaa/tracker/internal/repository"
	repomocks "github.com/timohahaa/tracker/internal/repository/mocks"
)

// Тесты TeamService написаны от контракта слайса team (PLAN.md 2.3, docs/tech/
// data-model.md, docs/tech/api.md): создание команды в одной транзакции создаёт
// 5 дефолтных статусов (по группе) и делает создателя участником-owner; правило
// «≥1 статус на группу» перед удалением; авторизация по membership/роли.
// Проверяется наблюдаемое поведение (входы репозиториев, коды ошибок), негативные
// инварианты — отсутствием ожиданий (лишний вызов роняет тест).

func newTeamService(store repository.Store) *TeamService {
	return &TeamService{repos: store}
}

// --- CreateTeam ---

func TestCreateTeam_CreatesTeamDefaultStatusesAndOwnerMembership(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := repomocks.NewMockStore(ctrl)
	teams := repomocks.NewMockTeamRepository(ctrl)
	statuses := repomocks.NewMockStatusRepository(ctrl)

	expectTx(store)
	store.EXPECT().Team().Return(teams).AnyTimes()
	store.EXPECT().Status().Return(statuses).AnyTimes()

	var created model.Team
	teams.EXPECT().Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, tm model.Team) (model.Team, error) {
			created = tm
			tm.ID = "team-1"
			return tm, nil
		})

	var groups []model.StatusGroup
	statuses.EXPECT().Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, st model.Status) (model.Status, error) {
			if st.TeamID != "team-1" {
				t.Errorf("status team_id = %q, want team-1", st.TeamID)
			}
			groups = append(groups, st.Group)
			return st, nil
		}).Times(len(defaultStatuses))

	var member model.TeamMember
	teams.EXPECT().AddMember(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, m model.TeamMember) (model.TeamMember, error) {
			member = m
			return m, nil
		})

	actor := model.WorkspaceMember{WorkspaceID: "ws-1", UserID: "user-1", Role: model.WorkspaceRoleMember}
	team, err := newTeamService(store).CreateTeam(context.Background(), actor, CreateTeamParams{
		Name: "Engineering",
		Key:  "eng",
	})
	if err != nil {
		t.Fatalf("CreateTeam: %v", err)
	}
	if team.ID != "team-1" {
		t.Errorf("team id = %q, want team-1", team.ID)
	}
	// key нормализуется в верхний регистр.
	if created.Key != "ENG" {
		t.Errorf("created key = %q, want ENG", created.Key)
	}
	if created.WorkspaceID != "ws-1" {
		t.Errorf("created workspace_id = %q, want ws-1", created.WorkspaceID)
	}
	// По одному дефолтному статусу на каждую группу.
	wantGroups := map[model.StatusGroup]bool{
		model.StatusGroupBacklog:    true,
		model.StatusGroupPlanned:    true,
		model.StatusGroupInProgress: true,
		model.StatusGroupCompleted:  true,
		model.StatusGroupCanceled:   true,
	}
	if len(groups) != len(wantGroups) {
		t.Fatalf("created %d statuses, want %d", len(groups), len(wantGroups))
	}
	for _, g := range groups {
		if !wantGroups[g] {
			t.Errorf("unexpected/duplicate status group %q", g)
		}
		delete(wantGroups, g)
	}
	// Создатель — участник-owner команды.
	if member.UserID != "user-1" || member.Role != model.TeamRoleOwner {
		t.Errorf("member = %+v, want user-1 owner", member)
	}
}

func TestCreateTeam_InvalidKeyValidation(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := repomocks.NewMockStore(ctrl)
	// Никаких обращений к БД/транзакции: невалидный key отсекается до WithTx.

	actor := model.WorkspaceMember{WorkspaceID: "ws-1", UserID: "user-1", Role: model.WorkspaceRoleMember}
	_, err := newTeamService(store).CreateTeam(context.Background(), actor, CreateTeamParams{
		Name: "No Key",
		Key:  "   ",
	})
	assertAppErrCode(t, err, errs.CodeValidation)
}

func TestCreateTeam_KeyConflictPropagated(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := repomocks.NewMockStore(ctrl)
	teams := repomocks.NewMockTeamRepository(ctrl)

	expectTx(store)
	store.EXPECT().Team().Return(teams).AnyTimes()

	conflict := errs.AlreadyExists("resource already exists")
	teams.EXPECT().Create(gomock.Any(), gomock.Any()).Return(model.Team{}, conflict)
	// Ни статусов, ни membership при провале вставки команды.

	actor := model.WorkspaceMember{WorkspaceID: "ws-1", UserID: "user-1", Role: model.WorkspaceRoleMember}
	_, err := newTeamService(store).CreateTeam(context.Background(), actor, CreateTeamParams{Name: "Dup", Key: "ENG"})
	if !errors.Is(err, conflict) {
		t.Fatalf("error = %v, want %v", err, conflict)
	}
}

// --- UpdateTeam ---

func TestUpdateTeam_ForbiddenWithoutTeamMembership(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := repomocks.NewMockStore(ctrl)
	teams := repomocks.NewMockTeamRepository(ctrl)
	store.EXPECT().Team().Return(teams).AnyTimes()

	teams.EXPECT().Get(gomock.Any(), "ws-1", "team-1").
		Return(model.Team{ID: "team-1", WorkspaceID: "ws-1"}, nil)
	teams.EXPECT().GetMember(gomock.Any(), "team-1", "user-1").
		Return(model.TeamMember{}, errs.NotFound("resource not found"))
	// Update не вызывается: не участник команды.

	actor := model.WorkspaceMember{WorkspaceID: "ws-1", UserID: "user-1", Role: model.WorkspaceRoleMember}
	_, err := newTeamService(store).UpdateTeam(context.Background(), actor, "team-1", UpdateTeamParams{Name: strPtr("New")})
	assertAppErrCode(t, err, errs.CodeForbidden)
}

func TestUpdateTeam_NotFoundPropagated(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := repomocks.NewMockStore(ctrl)
	teams := repomocks.NewMockTeamRepository(ctrl)
	store.EXPECT().Team().Return(teams).AnyTimes()

	teams.EXPECT().Get(gomock.Any(), "ws-1", "missing").Return(model.Team{}, errs.NotFound("resource not found"))
	// GetMember не вызывается, если команды нет.

	actor := model.WorkspaceMember{WorkspaceID: "ws-1", UserID: "user-1", Role: model.WorkspaceRoleMember}
	_, err := newTeamService(store).UpdateTeam(context.Background(), actor, "missing", UpdateTeamParams{Name: strPtr("New")})
	assertAppErrCode(t, err, errs.CodeNotFound)
}

func TestUpdateTeam_AppliesFieldsAndNormalizesKey(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := repomocks.NewMockStore(ctrl)
	teams := repomocks.NewMockTeamRepository(ctrl)
	store.EXPECT().Team().Return(teams).AnyTimes()

	teams.EXPECT().Get(gomock.Any(), "ws-1", "team-1").
		Return(model.Team{ID: "team-1", WorkspaceID: "ws-1", Name: "Old", Key: "OLD"}, nil)
	teams.EXPECT().GetMember(gomock.Any(), "team-1", "user-1").
		Return(model.TeamMember{TeamID: "team-1", UserID: "user-1", Role: model.TeamRoleMember}, nil)

	var saved model.Team
	teams.EXPECT().Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, tm model.Team) (model.Team, error) {
			saved = tm
			return tm, nil
		})

	actor := model.WorkspaceMember{WorkspaceID: "ws-1", UserID: "user-1", Role: model.WorkspaceRoleMember}
	_, err := newTeamService(store).UpdateTeam(context.Background(), actor, "team-1", UpdateTeamParams{
		Name: strPtr("New Name"),
		Key:  strPtr("new"),
	})
	if err != nil {
		t.Fatalf("UpdateTeam: %v", err)
	}
	if saved.Name != "New Name" || saved.Key != "NEW" {
		t.Errorf("saved = (%q, %q), want (New Name, NEW)", saved.Name, saved.Key)
	}
}

// --- DeleteTeam ---

func TestDeleteTeam_ForbiddenForNonManager(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := repomocks.NewMockStore(ctrl)
	// Проверка роли идёт первой — обращений к БД нет.

	actor := model.WorkspaceMember{WorkspaceID: "ws-1", UserID: "user-1", Role: model.WorkspaceRoleMember}
	err := newTeamService(store).DeleteTeam(context.Background(), actor, "team-1")
	assertAppErrCode(t, err, errs.CodeForbidden)
}

func TestDeleteTeam_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := repomocks.NewMockStore(ctrl)
	teams := repomocks.NewMockTeamRepository(ctrl)
	store.EXPECT().Team().Return(teams).AnyTimes()

	teams.EXPECT().Get(gomock.Any(), "ws-1", "team-1").Return(model.Team{ID: "team-1", WorkspaceID: "ws-1"}, nil)
	teams.EXPECT().Delete(gomock.Any(), "team-1").Return(nil)

	actor := model.WorkspaceMember{WorkspaceID: "ws-1", UserID: "user-1", Role: model.WorkspaceRoleAdmin}
	if err := newTeamService(store).DeleteTeam(context.Background(), actor, "team-1"); err != nil {
		t.Fatalf("DeleteTeam: %v", err)
	}
}

// --- AddMember ---

func TestAddMember_ForbiddenWithoutTeamMembership(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := repomocks.NewMockStore(ctrl)
	teams := repomocks.NewMockTeamRepository(ctrl)
	store.EXPECT().Team().Return(teams).AnyTimes()

	teams.EXPECT().Get(gomock.Any(), "ws-1", "team-1").Return(model.Team{ID: "team-1", WorkspaceID: "ws-1"}, nil)
	teams.EXPECT().GetMember(gomock.Any(), "team-1", "actor").Return(model.TeamMember{}, errs.NotFound("resource not found"))
	// Ни Workspace().GetMember, ни AddMember: actor не участник команды.

	actor := model.WorkspaceMember{WorkspaceID: "ws-1", UserID: "actor", Role: model.WorkspaceRoleMember}
	_, err := newTeamService(store).AddMember(context.Background(), actor, "team-1", AddTeamMemberParams{UserID: "target"})
	assertAppErrCode(t, err, errs.CodeForbidden)
}

func TestAddMember_TargetNotWorkspaceMemberValidation(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := repomocks.NewMockStore(ctrl)
	teams := repomocks.NewMockTeamRepository(ctrl)
	workspaces := repomocks.NewMockWorkspaceRepository(ctrl)
	store.EXPECT().Team().Return(teams).AnyTimes()
	store.EXPECT().Workspace().Return(workspaces).AnyTimes()

	teams.EXPECT().Get(gomock.Any(), "ws-1", "team-1").Return(model.Team{ID: "team-1", WorkspaceID: "ws-1"}, nil)
	teams.EXPECT().GetMember(gomock.Any(), "team-1", "actor").
		Return(model.TeamMember{TeamID: "team-1", UserID: "actor", Role: model.TeamRoleOwner}, nil)
	workspaces.EXPECT().GetMember(gomock.Any(), "ws-1", "target").
		Return(model.WorkspaceMember{}, errs.NotFound("resource not found"))
	// AddMember не вызывается: кандидат не участник workspace.

	actor := model.WorkspaceMember{WorkspaceID: "ws-1", UserID: "actor", Role: model.WorkspaceRoleMember}
	_, err := newTeamService(store).AddMember(context.Background(), actor, "team-1", AddTeamMemberParams{UserID: "target"})
	assertAppErrCode(t, err, errs.CodeValidation)
}

func TestAddMember_DefaultsRoleToMember(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := repomocks.NewMockStore(ctrl)
	teams := repomocks.NewMockTeamRepository(ctrl)
	workspaces := repomocks.NewMockWorkspaceRepository(ctrl)
	store.EXPECT().Team().Return(teams).AnyTimes()
	store.EXPECT().Workspace().Return(workspaces).AnyTimes()

	teams.EXPECT().Get(gomock.Any(), "ws-1", "team-1").Return(model.Team{ID: "team-1", WorkspaceID: "ws-1"}, nil)
	teams.EXPECT().GetMember(gomock.Any(), "team-1", "actor").
		Return(model.TeamMember{TeamID: "team-1", UserID: "actor", Role: model.TeamRoleOwner}, nil)
	workspaces.EXPECT().GetMember(gomock.Any(), "ws-1", "target").
		Return(model.WorkspaceMember{WorkspaceID: "ws-1", UserID: "target", Role: model.WorkspaceRoleMember}, nil)

	var saved model.TeamMember
	teams.EXPECT().AddMember(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, m model.TeamMember) (model.TeamMember, error) {
			saved = m
			return m, nil
		})

	actor := model.WorkspaceMember{WorkspaceID: "ws-1", UserID: "actor", Role: model.WorkspaceRoleMember}
	if _, err := newTeamService(store).AddMember(context.Background(), actor, "team-1", AddTeamMemberParams{UserID: "target"}); err != nil {
		t.Fatalf("AddMember: %v", err)
	}
	if saved.UserID != "target" || saved.Role != model.TeamRoleMember {
		t.Errorf("saved = %+v, want target member", saved)
	}
}

// --- DeleteStatus ---

func TestDeleteStatus_LastInGroupValidation(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := repomocks.NewMockStore(ctrl)
	teams := repomocks.NewMockTeamRepository(ctrl)
	statuses := repomocks.NewMockStatusRepository(ctrl)
	store.EXPECT().Team().Return(teams).AnyTimes()
	store.EXPECT().Status().Return(statuses).AnyTimes()

	teams.EXPECT().Get(gomock.Any(), "ws-1", "team-1").Return(model.Team{ID: "team-1", WorkspaceID: "ws-1"}, nil)
	teams.EXPECT().GetMember(gomock.Any(), "team-1", "user-1").
		Return(model.TeamMember{TeamID: "team-1", UserID: "user-1", Role: model.TeamRoleMember}, nil)
	statuses.EXPECT().Get(gomock.Any(), "team-1", "st-1").
		Return(model.Status{ID: "st-1", TeamID: "team-1", Group: model.StatusGroupBacklog}, nil)
	statuses.EXPECT().CountByGroup(gomock.Any(), "team-1", model.StatusGroupBacklog).Return(1, nil)
	// Delete не вызывается: последний статус в группе.

	actor := model.WorkspaceMember{WorkspaceID: "ws-1", UserID: "user-1", Role: model.WorkspaceRoleMember}
	err := newTeamService(store).DeleteStatus(context.Background(), actor, "team-1", "st-1")
	assertAppErrCode(t, err, errs.CodeValidation)
}

func TestDeleteStatus_AllowedWhenGroupHasOthers(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := repomocks.NewMockStore(ctrl)
	teams := repomocks.NewMockTeamRepository(ctrl)
	statuses := repomocks.NewMockStatusRepository(ctrl)
	store.EXPECT().Team().Return(teams).AnyTimes()
	store.EXPECT().Status().Return(statuses).AnyTimes()

	teams.EXPECT().Get(gomock.Any(), "ws-1", "team-1").Return(model.Team{ID: "team-1", WorkspaceID: "ws-1"}, nil)
	teams.EXPECT().GetMember(gomock.Any(), "team-1", "user-1").
		Return(model.TeamMember{TeamID: "team-1", UserID: "user-1", Role: model.TeamRoleMember}, nil)
	statuses.EXPECT().Get(gomock.Any(), "team-1", "st-1").
		Return(model.Status{ID: "st-1", TeamID: "team-1", Group: model.StatusGroupBacklog}, nil)
	statuses.EXPECT().CountByGroup(gomock.Any(), "team-1", model.StatusGroupBacklog).Return(2, nil)
	statuses.EXPECT().Delete(gomock.Any(), "st-1").Return(nil)

	actor := model.WorkspaceMember{WorkspaceID: "ws-1", UserID: "user-1", Role: model.WorkspaceRoleMember}
	if err := newTeamService(store).DeleteStatus(context.Background(), actor, "team-1", "st-1"); err != nil {
		t.Fatalf("DeleteStatus: %v", err)
	}
}

// --- CreateStatus ---

func TestCreateStatus_ForbiddenWithoutTeamMembership(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := repomocks.NewMockStore(ctrl)
	teams := repomocks.NewMockTeamRepository(ctrl)
	store.EXPECT().Team().Return(teams).AnyTimes()

	teams.EXPECT().Get(gomock.Any(), "ws-1", "team-1").Return(model.Team{ID: "team-1", WorkspaceID: "ws-1"}, nil)
	teams.EXPECT().GetMember(gomock.Any(), "team-1", "user-1").Return(model.TeamMember{}, errs.NotFound("resource not found"))
	// Status().Create не вызывается.

	actor := model.WorkspaceMember{WorkspaceID: "ws-1", UserID: "user-1", Role: model.WorkspaceRoleMember}
	_, err := newTeamService(store).CreateStatus(context.Background(), actor, "team-1", CreateStatusParams{
		Name:  "Blocked",
		Group: model.StatusGroupInProgress,
	})
	assertAppErrCode(t, err, errs.CodeForbidden)
}
