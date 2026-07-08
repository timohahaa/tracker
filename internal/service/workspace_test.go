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

func newWorkspaceService(store repository.Store) *WorkspaceService {
	return &WorkspaceService{repos: store}
}

func strPtr(s string) *string { return &s }

func rolePtr(r model.WorkspaceRole) *model.WorkspaceRole { return &r }

// --- canManageWorkspace ---

func TestCanManageWorkspace(t *testing.T) {
	cases := map[model.WorkspaceRole]bool{
		model.WorkspaceRoleOwner:  true,
		model.WorkspaceRoleAdmin:  true,
		model.WorkspaceRoleMember: false,
	}
	for role, want := range cases {
		if got := canManageWorkspace(role); got != want {
			t.Errorf("canManageWorkspace(%q) = %v, want %v", role, got, want)
		}
	}
}

// --- Load ---

func TestLoad_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := repomocks.NewMockStore(ctrl)
	workspaces := repomocks.NewMockWorkspaceRepository(ctrl)
	store.EXPECT().Workspace().Return(workspaces).AnyTimes()

	workspaces.EXPECT().GetByID(gomock.Any(), "ws-1").
		Return(model.Workspace{ID: "ws-1", Name: "Acme"}, nil)
	workspaces.EXPECT().GetMember(gomock.Any(), "ws-1", "user-1").
		Return(model.WorkspaceMember{WorkspaceID: "ws-1", UserID: "user-1", Role: model.WorkspaceRoleMember}, nil)

	ws, member, err := newWorkspaceService(store).Load(context.Background(), "ws-1", "user-1")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if ws.ID != "ws-1" || member.UserID != "user-1" {
		t.Errorf("got ws=%+v member=%+v", ws, member)
	}
}

func TestLoad_WorkspaceErrorPropagated(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := repomocks.NewMockStore(ctrl)
	workspaces := repomocks.NewMockWorkspaceRepository(ctrl)
	store.EXPECT().Workspace().Return(workspaces).AnyTimes()

	dbErr := errs.Internal("db unavailable")
	workspaces.EXPECT().GetByID(gomock.Any(), "ws-1").Return(model.Workspace{}, dbErr)
	// GetMember не вызывается, если workspace не загрузился.

	_, _, err := newWorkspaceService(store).Load(context.Background(), "ws-1", "user-1")
	if !errors.Is(err, dbErr) {
		t.Fatalf("error = %v, want %v", err, dbErr)
	}
}

func TestLoad_NonMemberMaskedAsNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := repomocks.NewMockStore(ctrl)
	workspaces := repomocks.NewMockWorkspaceRepository(ctrl)
	store.EXPECT().Workspace().Return(workspaces).AnyTimes()

	workspaces.EXPECT().GetByID(gomock.Any(), "ws-1").Return(model.Workspace{ID: "ws-1"}, nil)
	// Пользователь не участник: наличие workspace не раскрываем — отдаём not_found.
	workspaces.EXPECT().GetMember(gomock.Any(), "ws-1", "user-1").
		Return(model.WorkspaceMember{}, errs.NotFound("resource not found"))

	_, _, err := newWorkspaceService(store).Load(context.Background(), "ws-1", "user-1")
	assertAppErrCode(t, err, errs.CodeNotFound)
}

func TestLoad_MemberLookupErrorPropagated(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := repomocks.NewMockStore(ctrl)
	workspaces := repomocks.NewMockWorkspaceRepository(ctrl)
	store.EXPECT().Workspace().Return(workspaces).AnyTimes()

	workspaces.EXPECT().GetByID(gomock.Any(), "ws-1").Return(model.Workspace{ID: "ws-1"}, nil)
	dbErr := errs.Internal("db unavailable")
	workspaces.EXPECT().GetMember(gomock.Any(), "ws-1", "user-1").Return(model.WorkspaceMember{}, dbErr)

	_, _, err := newWorkspaceService(store).Load(context.Background(), "ws-1", "user-1")
	if !errors.Is(err, dbErr) {
		t.Fatalf("error = %v, want %v", err, dbErr)
	}
}

// --- UpdateSettings ---

func TestUpdateSettings_ForbiddenForNonManager(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := repomocks.NewMockStore(ctrl)
	// Никаких обращений к repo: проверка прав идёт первой.

	actor := model.WorkspaceMember{WorkspaceID: "ws-1", Role: model.WorkspaceRoleMember}
	_, err := newWorkspaceService(store).UpdateSettings(context.Background(), actor, UpdateSettingsParams{
		Name: strPtr("New"),
	})
	assertAppErrCode(t, err, errs.CodeForbidden)
}

func TestUpdateSettings_GetByIDErrorPropagated(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := repomocks.NewMockStore(ctrl)
	workspaces := repomocks.NewMockWorkspaceRepository(ctrl)
	store.EXPECT().Workspace().Return(workspaces).AnyTimes()

	dbErr := errs.Internal("db unavailable")
	workspaces.EXPECT().GetByID(gomock.Any(), "ws-1").Return(model.Workspace{}, dbErr)

	actor := model.WorkspaceMember{WorkspaceID: "ws-1", Role: model.WorkspaceRoleAdmin}
	_, err := newWorkspaceService(store).UpdateSettings(context.Background(), actor, UpdateSettingsParams{Name: strPtr("New")})
	if !errors.Is(err, dbErr) {
		t.Fatalf("error = %v, want %v", err, dbErr)
	}
}

func TestUpdateSettings_InvalidSlugValidation(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := repomocks.NewMockStore(ctrl)
	workspaces := repomocks.NewMockWorkspaceRepository(ctrl)
	store.EXPECT().Workspace().Return(workspaces).AnyTimes()

	workspaces.EXPECT().GetByID(gomock.Any(), "ws-1").Return(model.Workspace{ID: "ws-1"}, nil)
	// Update не должен вызываться при невалидном slug.

	actor := model.WorkspaceMember{WorkspaceID: "ws-1", Role: model.WorkspaceRoleOwner}
	_, err := newWorkspaceService(store).UpdateSettings(context.Background(), actor, UpdateSettingsParams{Slug: strPtr("!!!")})
	assertAppErrCode(t, err, errs.CodeValidation)
}

func TestUpdateSettings_AppliesNameAndSlugifiedSlug(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := repomocks.NewMockStore(ctrl)
	workspaces := repomocks.NewMockWorkspaceRepository(ctrl)
	store.EXPECT().Workspace().Return(workspaces).AnyTimes()

	workspaces.EXPECT().GetByID(gomock.Any(), "ws-1").
		Return(model.Workspace{ID: "ws-1", Name: "Old", Slug: "old"}, nil)

	var updated model.Workspace
	workspaces.EXPECT().Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, ws model.Workspace) (model.Workspace, error) {
			updated = ws
			return ws, nil
		})

	actor := model.WorkspaceMember{WorkspaceID: "ws-1", Role: model.WorkspaceRoleAdmin}
	_, err := newWorkspaceService(store).UpdateSettings(context.Background(), actor, UpdateSettingsParams{
		Name: strPtr("New Name"),
		Slug: strPtr("New Slug"),
	})
	if err != nil {
		t.Fatalf("UpdateSettings: %v", err)
	}
	if updated.Name != "New Name" {
		t.Errorf("name = %q, want New Name", updated.Name)
	}
	if updated.Slug != "new-slug" {
		t.Errorf("slug = %q, want url-safe new-slug", updated.Slug)
	}
}

func TestUpdateSettings_NilParamsLeaveFieldsUnchanged(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := repomocks.NewMockStore(ctrl)
	workspaces := repomocks.NewMockWorkspaceRepository(ctrl)
	store.EXPECT().Workspace().Return(workspaces).AnyTimes()

	workspaces.EXPECT().GetByID(gomock.Any(), "ws-1").
		Return(model.Workspace{ID: "ws-1", Name: "Old", Slug: "old"}, nil)

	var updated model.Workspace
	workspaces.EXPECT().Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, ws model.Workspace) (model.Workspace, error) {
			updated = ws
			return ws, nil
		})

	actor := model.WorkspaceMember{WorkspaceID: "ws-1", Role: model.WorkspaceRoleOwner}
	if _, err := newWorkspaceService(store).UpdateSettings(context.Background(), actor, UpdateSettingsParams{}); err != nil {
		t.Fatalf("UpdateSettings: %v", err)
	}
	if updated.Name != "Old" || updated.Slug != "old" {
		t.Errorf("nil params must not change fields, got %+v", updated)
	}
}

// --- UpdateMember ---

func TestUpdateMember_ForbiddenForNonManager(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := repomocks.NewMockStore(ctrl)

	actor := model.WorkspaceMember{WorkspaceID: "ws-1", Role: model.WorkspaceRoleMember}
	_, err := newWorkspaceService(store).UpdateMember(context.Background(), actor, "target", UpdateMemberParams{
		Username: strPtr("x"),
	})
	assertAppErrCode(t, err, errs.CodeForbidden)
}

func TestUpdateMember_GetMemberErrorPropagated(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := repomocks.NewMockStore(ctrl)
	workspaces := repomocks.NewMockWorkspaceRepository(ctrl)
	store.EXPECT().Workspace().Return(workspaces).AnyTimes()

	dbErr := errs.NotFound("resource not found")
	workspaces.EXPECT().GetMember(gomock.Any(), "ws-1", "target").Return(model.WorkspaceMember{}, dbErr)

	actor := model.WorkspaceMember{WorkspaceID: "ws-1", Role: model.WorkspaceRoleAdmin}
	_, err := newWorkspaceService(store).UpdateMember(context.Background(), actor, "target", UpdateMemberParams{Username: strPtr("x")})
	if !errors.Is(err, dbErr) {
		t.Fatalf("error = %v, want %v", err, dbErr)
	}
}

func TestUpdateMember_CannotAssignOwnerRole(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := repomocks.NewMockStore(ctrl)
	workspaces := repomocks.NewMockWorkspaceRepository(ctrl)
	store.EXPECT().Workspace().Return(workspaces).AnyTimes()

	workspaces.EXPECT().GetMember(gomock.Any(), "ws-1", "target").
		Return(model.WorkspaceMember{WorkspaceID: "ws-1", UserID: "target", Role: model.WorkspaceRoleMember}, nil)
	// CountOwners/UpdateMember не вызываются.

	actor := model.WorkspaceMember{WorkspaceID: "ws-1", Role: model.WorkspaceRoleOwner}
	_, err := newWorkspaceService(store).UpdateMember(context.Background(), actor, "target", UpdateMemberParams{
		Role: rolePtr(model.WorkspaceRoleOwner),
	})
	assertAppErrCode(t, err, errs.CodeForbidden)
}

func TestUpdateMember_CannotDemoteLastOwner(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := repomocks.NewMockStore(ctrl)
	workspaces := repomocks.NewMockWorkspaceRepository(ctrl)
	store.EXPECT().Workspace().Return(workspaces).AnyTimes()

	workspaces.EXPECT().GetMember(gomock.Any(), "ws-1", "target").
		Return(model.WorkspaceMember{WorkspaceID: "ws-1", UserID: "target", Role: model.WorkspaceRoleOwner}, nil)
	workspaces.EXPECT().CountOwners(gomock.Any(), "ws-1").Return(1, nil)
	// UpdateMember не вызывается: последнего owner понижать нельзя.

	actor := model.WorkspaceMember{WorkspaceID: "ws-1", Role: model.WorkspaceRoleOwner}
	_, err := newWorkspaceService(store).UpdateMember(context.Background(), actor, "target", UpdateMemberParams{
		Role: rolePtr(model.WorkspaceRoleAdmin),
	})
	assertAppErrCode(t, err, errs.CodeForbidden)
}

func TestUpdateMember_DemoteOwnerAllowedWhenOthersExist(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := repomocks.NewMockStore(ctrl)
	workspaces := repomocks.NewMockWorkspaceRepository(ctrl)
	store.EXPECT().Workspace().Return(workspaces).AnyTimes()

	workspaces.EXPECT().GetMember(gomock.Any(), "ws-1", "target").
		Return(model.WorkspaceMember{WorkspaceID: "ws-1", UserID: "target", Role: model.WorkspaceRoleOwner}, nil)
	workspaces.EXPECT().CountOwners(gomock.Any(), "ws-1").Return(2, nil)

	var saved model.WorkspaceMember
	workspaces.EXPECT().UpdateMember(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, m model.WorkspaceMember) (model.WorkspaceMember, error) {
			saved = m
			return m, nil
		})

	actor := model.WorkspaceMember{WorkspaceID: "ws-1", Role: model.WorkspaceRoleOwner}
	if _, err := newWorkspaceService(store).UpdateMember(context.Background(), actor, "target", UpdateMemberParams{
		Role: rolePtr(model.WorkspaceRoleAdmin),
	}); err != nil {
		t.Fatalf("UpdateMember: %v", err)
	}
	if saved.Role != model.WorkspaceRoleAdmin {
		t.Errorf("role = %q, want admin", saved.Role)
	}
}

func TestUpdateMember_UpdatesUsernameAndTitleWithoutRoleCheck(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := repomocks.NewMockStore(ctrl)
	workspaces := repomocks.NewMockWorkspaceRepository(ctrl)
	store.EXPECT().Workspace().Return(workspaces).AnyTimes()

	workspaces.EXPECT().GetMember(gomock.Any(), "ws-1", "target").
		Return(model.WorkspaceMember{WorkspaceID: "ws-1", UserID: "target", Role: model.WorkspaceRoleMember}, nil)
	// CountOwners не вызывается, т.к. роль не меняется.

	var saved model.WorkspaceMember
	workspaces.EXPECT().UpdateMember(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, m model.WorkspaceMember) (model.WorkspaceMember, error) {
			saved = m
			return m, nil
		})

	actor := model.WorkspaceMember{WorkspaceID: "ws-1", Role: model.WorkspaceRoleAdmin}
	if _, err := newWorkspaceService(store).UpdateMember(context.Background(), actor, "target", UpdateMemberParams{
		Username: strPtr("new-username"),
		Title:    strPtr("Engineer"),
	}); err != nil {
		t.Fatalf("UpdateMember: %v", err)
	}
	if saved.Username != "new-username" {
		t.Errorf("username = %q, want new-username", saved.Username)
	}
	if saved.Title == nil || *saved.Title != "Engineer" {
		t.Errorf("title = %v, want Engineer", saved.Title)
	}
}

// --- RemoveMember ---

func TestRemoveMember_ForbiddenForNonManager(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := repomocks.NewMockStore(ctrl)

	actor := model.WorkspaceMember{WorkspaceID: "ws-1", Role: model.WorkspaceRoleMember}
	err := newWorkspaceService(store).RemoveMember(context.Background(), actor, "target")
	assertAppErrCode(t, err, errs.CodeForbidden)
}

func TestRemoveMember_GetMemberErrorPropagated(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := repomocks.NewMockStore(ctrl)
	workspaces := repomocks.NewMockWorkspaceRepository(ctrl)
	store.EXPECT().Workspace().Return(workspaces).AnyTimes()

	dbErr := errs.NotFound("resource not found")
	workspaces.EXPECT().GetMember(gomock.Any(), "ws-1", "target").Return(model.WorkspaceMember{}, dbErr)

	actor := model.WorkspaceMember{WorkspaceID: "ws-1", Role: model.WorkspaceRoleAdmin}
	err := newWorkspaceService(store).RemoveMember(context.Background(), actor, "target")
	if !errors.Is(err, dbErr) {
		t.Fatalf("error = %v, want %v", err, dbErr)
	}
}

func TestRemoveMember_CannotRemoveLastOwner(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := repomocks.NewMockStore(ctrl)
	workspaces := repomocks.NewMockWorkspaceRepository(ctrl)
	store.EXPECT().Workspace().Return(workspaces).AnyTimes()

	workspaces.EXPECT().GetMember(gomock.Any(), "ws-1", "target").
		Return(model.WorkspaceMember{WorkspaceID: "ws-1", UserID: "target", Role: model.WorkspaceRoleOwner}, nil)
	workspaces.EXPECT().CountOwners(gomock.Any(), "ws-1").Return(1, nil)
	// DeleteMember не вызывается.

	actor := model.WorkspaceMember{WorkspaceID: "ws-1", Role: model.WorkspaceRoleOwner}
	err := newWorkspaceService(store).RemoveMember(context.Background(), actor, "target")
	assertAppErrCode(t, err, errs.CodeForbidden)
}

func TestRemoveMember_RemovesOwnerWhenOthersExist(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := repomocks.NewMockStore(ctrl)
	workspaces := repomocks.NewMockWorkspaceRepository(ctrl)
	store.EXPECT().Workspace().Return(workspaces).AnyTimes()

	workspaces.EXPECT().GetMember(gomock.Any(), "ws-1", "target").
		Return(model.WorkspaceMember{WorkspaceID: "ws-1", UserID: "target", Role: model.WorkspaceRoleOwner}, nil)
	workspaces.EXPECT().CountOwners(gomock.Any(), "ws-1").Return(2, nil)
	workspaces.EXPECT().DeleteMember(gomock.Any(), "ws-1", "target").Return(nil)

	actor := model.WorkspaceMember{WorkspaceID: "ws-1", Role: model.WorkspaceRoleOwner}
	if err := newWorkspaceService(store).RemoveMember(context.Background(), actor, "target"); err != nil {
		t.Fatalf("RemoveMember: %v", err)
	}
}

func TestRemoveMember_RemovesNonOwnerWithoutOwnerCheck(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := repomocks.NewMockStore(ctrl)
	workspaces := repomocks.NewMockWorkspaceRepository(ctrl)
	store.EXPECT().Workspace().Return(workspaces).AnyTimes()

	workspaces.EXPECT().GetMember(gomock.Any(), "ws-1", "target").
		Return(model.WorkspaceMember{WorkspaceID: "ws-1", UserID: "target", Role: model.WorkspaceRoleMember}, nil)
	// CountOwners не вызывается для не-owner.
	workspaces.EXPECT().DeleteMember(gomock.Any(), "ws-1", "target").Return(nil)

	actor := model.WorkspaceMember{WorkspaceID: "ws-1", Role: model.WorkspaceRoleAdmin}
	if err := newWorkspaceService(store).RemoveMember(context.Background(), actor, "target"); err != nil {
		t.Fatalf("RemoveMember: %v", err)
	}
}

// --- ListMembers ---

func TestListMembers_DelegatesToRepo(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := repomocks.NewMockStore(ctrl)
	workspaces := repomocks.NewMockWorkspaceRepository(ctrl)
	store.EXPECT().Workspace().Return(workspaces).AnyTimes()

	want := []model.WorkspaceMemberDetails{
		{WorkspaceMember: model.WorkspaceMember{WorkspaceID: "ws-1", UserID: "user-1"}, Email: "ada@example.com"},
	}
	workspaces.EXPECT().ListMembers(gomock.Any(), "ws-1").Return(want, nil)

	got, err := newWorkspaceService(store).ListMembers(context.Background(), "ws-1")
	if err != nil {
		t.Fatalf("ListMembers: %v", err)
	}
	if len(got) != 1 || got[0].Email != "ada@example.com" {
		t.Errorf("got %+v, want %+v", got, want)
	}
}
