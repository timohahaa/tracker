//go:build integration

package repository_test

import (
	"context"
	"testing"

	"github.com/timohahaa/tracker/internal/repository"
	"github.com/timohahaa/tracker/test/dbtest"
)

func TestTokenRepo_GetByID_And_UpdateLastUsed(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	db := dbtest.Open(t)
	repos := repository.New(db)

	userID := dbtest.SeedUser(t, db)
	wsID := dbtest.SeedWorkspace(t, db)
	tokenID := dbtest.SeedAPIToken(t, db, userID, wsID)

	token, err := repos.Token().GetByID(ctx, tokenID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if token.UserID != userID || token.WorkspaceID != wsID {
		t.Errorf("GetByID = user %q ws %q, want user %q ws %q", token.UserID, token.WorkspaceID, userID, wsID)
	}
	if token.LastUsedAt != nil {
		t.Errorf("last_used_at should be nil before use, got %v", token.LastUsedAt)
	}

	if err := repos.Token().UpdateLastUsed(ctx, tokenID); err != nil {
		t.Fatalf("UpdateLastUsed: %v", err)
	}
	after, err := repos.Token().GetByID(ctx, tokenID)
	if err != nil {
		t.Fatalf("GetByID after update: %v", err)
	}
	if after.LastUsedAt == nil {
		t.Error("last_used_at should be set after UpdateLastUsed")
	}
}
