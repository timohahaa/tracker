//go:build integration

package repository_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/timohahaa/tracker/internal/errs"
	"github.com/timohahaa/tracker/internal/model"
	"github.com/timohahaa/tracker/internal/repository"
	"github.com/timohahaa/tracker/test/dbtest"
)

func TestWorkspaceRepo_GetByID(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	db := dbtest.Open(t)
	repos := repository.New(db)

	wsID := dbtest.SeedWorkspace(t, db)

	ws, err := repos.Workspace().GetByID(ctx, wsID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if ws.ID != wsID {
		t.Errorf("GetByID id = %q, want %q", ws.ID, wsID)
	}

	_, err = repos.Workspace().GetByID(ctx, uuid.NewString())
	var appErr *errs.AppError
	if !errors.As(err, &appErr) || appErr.Code != errs.CodeNotFound {
		t.Fatalf("GetByID missing: want NotFound, got %v", err)
	}
}

func TestWorkspaceRepo_Update(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	db := dbtest.Open(t)
	repos := repository.New(db)

	wsID := dbtest.SeedWorkspace(t, db)
	newSlug := "upd-" + uuid.NewString()

	updated, err := repos.Workspace().Update(ctx, model.Workspace{
		ID:   wsID,
		Name: "Renamed",
		Slug: newSlug,
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Name != "Renamed" || updated.Slug != newSlug {
		t.Errorf("Update = (%q, %q), want (Renamed, %q)", updated.Name, updated.Slug, newSlug)
	}
}

func TestWorkspaceRepo_Update_SlugConflict(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	db := dbtest.Open(t)
	repos := repository.New(db)

	// Первый workspace с известным slug.
	takenSlug := "taken-" + uuid.NewString()
	ws1 := dbtest.SeedWorkspace(t, db)
	if _, err := repos.Workspace().Update(ctx, model.Workspace{ID: ws1, Name: "One", Slug: takenSlug}); err != nil {
		t.Fatalf("Update ws1: %v", err)
	}

	// Второй workspace пытается занять тот же slug.
	ws2 := dbtest.SeedWorkspace(t, db)
	_, err := repos.Workspace().Update(ctx, model.Workspace{ID: ws2, Name: "Two", Slug: takenSlug})
	var appErr *errs.AppError
	if !errors.As(err, &appErr) || appErr.Code != errs.CodeAlreadyExists {
		t.Fatalf("Update slug conflict: want AlreadyExists, got %v", err)
	}
}

func TestWorkspaceRepo_Members(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	db := dbtest.Open(t)
	repos := repository.New(db)

	wsID := dbtest.SeedWorkspace(t, db)
	ownerID := dbtest.SeedUser(t, db)
	memberID := dbtest.SeedUser(t, db)
	dbtest.SeedWorkspaceMember(t, db, wsID, ownerID, string(model.WorkspaceRoleOwner))
	dbtest.SeedWorkspaceMember(t, db, wsID, memberID, string(model.WorkspaceRoleMember))

	// ListMembers наполняет email/display_name.
	members, err := repos.Workspace().ListMembers(ctx, wsID)
	if err != nil {
		t.Fatalf("ListMembers: %v", err)
	}
	if len(members) != 2 {
		t.Fatalf("ListMembers len = %d, want 2", len(members))
	}
	for _, m := range members {
		if m.Email == "" || m.DisplayName == "" {
			t.Errorf("ListMembers: empty email/display_name for %q", m.UserID)
		}
	}

	// GetMember.
	got, err := repos.Workspace().GetMember(ctx, wsID, memberID)
	if err != nil {
		t.Fatalf("GetMember: %v", err)
	}
	if got.Role != model.WorkspaceRoleMember {
		t.Errorf("GetMember role = %q, want member", got.Role)
	}

	// CountOwners.
	owners, err := repos.Workspace().CountOwners(ctx, wsID)
	if err != nil {
		t.Fatalf("CountOwners: %v", err)
	}
	if owners != 1 {
		t.Errorf("CountOwners = %d, want 1", owners)
	}
}

func TestWorkspaceRepo_UpdateMember(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	db := dbtest.Open(t)
	repos := repository.New(db)

	wsID := dbtest.SeedWorkspace(t, db)
	userID := dbtest.SeedUser(t, db)
	dbtest.SeedWorkspaceMember(t, db, wsID, userID, string(model.WorkspaceRoleMember))

	newUsername := "u-" + uuid.NewString()
	title := "Engineer"
	updated, err := repos.Workspace().UpdateMember(ctx, model.WorkspaceMember{
		WorkspaceID: wsID,
		UserID:      userID,
		Role:        model.WorkspaceRoleAdmin,
		Username:    newUsername,
		Title:       &title,
	})
	if err != nil {
		t.Fatalf("UpdateMember: %v", err)
	}
	if updated.Role != model.WorkspaceRoleAdmin || updated.Username != newUsername {
		t.Errorf("UpdateMember = (%q, %q), want (admin, %q)", updated.Role, updated.Username, newUsername)
	}
	if updated.Title == nil || *updated.Title != title {
		t.Errorf("UpdateMember title = %v, want %q", updated.Title, title)
	}
}

func TestWorkspaceRepo_UpdateMember_UsernameConflict(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	db := dbtest.Open(t)
	repos := repository.New(db)

	wsID := dbtest.SeedWorkspace(t, db)
	userA := dbtest.SeedUser(t, db)
	userB := dbtest.SeedUser(t, db)
	usernameA := dbtest.SeedWorkspaceMember(t, db, wsID, userA, string(model.WorkspaceRoleMember))
	dbtest.SeedWorkspaceMember(t, db, wsID, userB, string(model.WorkspaceRoleMember))

	// userB пытается занять username userA.
	_, err := repos.Workspace().UpdateMember(ctx, model.WorkspaceMember{
		WorkspaceID: wsID,
		UserID:      userB,
		Role:        model.WorkspaceRoleMember,
		Username:    usernameA,
	})
	var appErr *errs.AppError
	if !errors.As(err, &appErr) || appErr.Code != errs.CodeAlreadyExists {
		t.Fatalf("UpdateMember username conflict: want AlreadyExists, got %v", err)
	}
}

func TestWorkspaceRepo_DeleteMember(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	db := dbtest.Open(t)
	repos := repository.New(db)

	wsID := dbtest.SeedWorkspace(t, db)
	userID := dbtest.SeedUser(t, db)
	dbtest.SeedWorkspaceMember(t, db, wsID, userID, string(model.WorkspaceRoleMember))

	if err := repos.Workspace().DeleteMember(ctx, wsID, userID); err != nil {
		t.Fatalf("DeleteMember: %v", err)
	}

	// После soft delete участник не находится.
	_, err := repos.Workspace().GetMember(ctx, wsID, userID)
	var appErr *errs.AppError
	if !errors.As(err, &appErr) || appErr.Code != errs.CodeNotFound {
		t.Fatalf("GetMember after delete: want NotFound, got %v", err)
	}
}
