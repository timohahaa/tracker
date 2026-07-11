//go:build integration

package repository_test

import (
	"context"
	"errors"
	"testing"

	"github.com/timohahaa/tracker/internal/errs"
	"github.com/timohahaa/tracker/internal/model"
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

func TestTokenRepo_Create_ListByUser_Delete(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	db := dbtest.Open(t)
	repos := repository.New(db)

	userID := dbtest.SeedUser(t, db)
	otherID := dbtest.SeedUser(t, db)
	wsID := dbtest.SeedWorkspace(t, db)

	// Create возвращает сгенерированные id (он же токен) и created_at.
	created, err := repos.Token().Create(ctx, model.APIToken{UserID: userID, WorkspaceID: wsID, Name: "ci"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	t.Cleanup(func() { _, _ = db.Exec(`DELETE FROM core.api_token WHERE id = $1`, created.ID) })
	if created.ID == "" {
		t.Fatal("Create: empty token id")
	}
	if created.CreatedAt.IsZero() {
		t.Error("Create: created_at not set")
	}
	if created.Name != "ci" {
		t.Errorf("Create: name = %q, want ci", created.Name)
	}

	// Токен другого пользователя в том же workspace — не должен попадать в выдачу.
	dbtest.SeedAPIToken(t, db, otherID, wsID)

	list, err := repos.Token().ListByUser(ctx, wsID, userID)
	if err != nil {
		t.Fatalf("ListByUser: %v", err)
	}
	if len(list) != 1 || list[0].ID != created.ID {
		t.Fatalf("ListByUser = %v, want only own token %q", list, created.ID)
	}

	// Delete мягко отзывает: пропадает из ListByUser и из GetByID.
	if err := repos.Token().Delete(ctx, created.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	after, err := repos.Token().ListByUser(ctx, wsID, userID)
	if err != nil {
		t.Fatalf("ListByUser after delete: %v", err)
	}
	if len(after) != 0 {
		t.Errorf("ListByUser after delete = %v, want empty (soft-deleted hidden)", after)
	}
	_, err = repos.Token().GetByID(ctx, created.ID)
	var appErr *errs.AppError
	if !errors.As(err, &appErr) || appErr.Code != errs.CodeNotFound {
		t.Fatalf("GetByID after delete: want NotFound, got %v", err)
	}

	// Повторный Delete идемпотентен.
	if err := repos.Token().Delete(ctx, created.ID); err != nil {
		t.Fatalf("Delete idempotent: %v", err)
	}
}
