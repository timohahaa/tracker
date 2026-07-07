//go:build integration

package repository_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/timohahaa/tracker/internal/errs"
	"github.com/timohahaa/tracker/internal/repository"
	"github.com/timohahaa/tracker/test/dbtest"
)

func TestSessionRepo_CRUD(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	db := dbtest.Open(t)
	repos := repository.New(db)

	userID := dbtest.SeedUser(t, db)

	exp := time.Now().Add(time.Hour).Truncate(time.Millisecond)
	session, err := repos.Session.Create(ctx, userID, exp)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	t.Cleanup(func() { _, _ = db.Exec(`DELETE FROM core.session WHERE id = $1`, session.ID) })
	if session.ID == "" {
		t.Fatal("Create: empty session id")
	}

	got, err := repos.Session.GetByID(ctx, session.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.UserID != userID {
		t.Errorf("GetByID user_id = %q, want %q", got.UserID, userID)
	}

	newExp := time.Now().Add(2 * time.Hour)
	if err := repos.Session.Extend(ctx, session.ID, newExp); err != nil {
		t.Fatalf("Extend: %v", err)
	}
	extended, err := repos.Session.GetByID(ctx, session.ID)
	if err != nil {
		t.Fatalf("GetByID after extend: %v", err)
	}
	if !extended.ExpiresAt.After(got.ExpiresAt) {
		t.Errorf("Extend: expires_at not advanced (%v !> %v)", extended.ExpiresAt, got.ExpiresAt)
	}

	if err := repos.Session.Delete(ctx, session.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	_, err = repos.Session.GetByID(ctx, session.ID)
	var appErr *errs.AppError
	if !errors.As(err, &appErr) || appErr.Code != errs.CodeNotFound {
		t.Fatalf("GetByID after delete: want NotFound, got %v", err)
	}
}
