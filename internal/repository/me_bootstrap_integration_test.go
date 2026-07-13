//go:build integration

package repository_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/timohahaa/tracker/internal/model"
	"github.com/timohahaa/tracker/internal/repository"
	"github.com/timohahaa/tracker/test/dbtest"
)

func TestUserRepo_Settings(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	db := dbtest.Open(t)
	repos := repository.New(db)

	email := "settings-" + uuid.NewString() + "@test.local"
	created, err := repos.User().Create(ctx, model.User{
		Email:        email,
		PasswordHash: "hash",
		DisplayName:  "Settings User",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	t.Cleanup(func() { _, _ = db.Exec(`DELETE FROM core.user WHERE id = $1`, created.ID) })

	// Дефолт '{}' → CurrentWorkspaceID nil.
	if created.Settings.CurrentWorkspaceID != nil {
		t.Errorf("Create: settings.current_workspace_id = %v, want nil", *created.Settings.CurrentWorkspaceID)
	}

	wsID := uuid.NewString()
	if err := repos.User().UpdateSettings(ctx, created.ID, model.UserSettings{CurrentWorkspaceID: &wsID}); err != nil {
		t.Fatalf("UpdateSettings: %v", err)
	}

	got, err := repos.User().GetByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Settings.CurrentWorkspaceID == nil || *got.Settings.CurrentWorkspaceID != wsID {
		t.Fatalf("GetByID: settings.current_workspace_id = %v, want %q", got.Settings.CurrentWorkspaceID, wsID)
	}
}

func TestWorkspaceRepo_ListByUser(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	db := dbtest.Open(t)
	repos := repository.New(db)

	userID := dbtest.SeedUser(t, db)

	// Активное членство → должен попасть в список с ролью.
	wsJoined := dbtest.SeedWorkspace(t, db)
	dbtest.SeedWorkspaceMember(t, db, wsJoined, userID, string(model.WorkspaceRoleOwner))

	// Workspace без членства → не должен попасть.
	dbtest.SeedWorkspace(t, db)

	// Soft-deleted членство → не должно попасть.
	wsDeleted := dbtest.SeedWorkspace(t, db)
	dbtest.SeedWorkspaceMember(t, db, wsDeleted, userID, string(model.WorkspaceRoleMember))
	if _, err := db.Exec(`UPDATE org.workspace_member SET deleted_at = now() WHERE workspace_id = $1 AND user_id = $2`, wsDeleted, userID); err != nil {
		t.Fatalf("soft-delete membership: %v", err)
	}

	list, err := repos.Workspace().ListByUser(ctx, userID)
	if err != nil {
		t.Fatalf("ListByUser: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("ListByUser: len = %d, want 1", len(list))
	}
	if list[0].ID != wsJoined {
		t.Errorf("ListByUser: id = %q, want %q", list[0].ID, wsJoined)
	}
	if list[0].Role != model.WorkspaceRoleOwner {
		t.Errorf("ListByUser: role = %q, want owner", list[0].Role)
	}
}
