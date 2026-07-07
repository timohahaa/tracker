//go:build integration

package repository_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/timohahaa/tracker/internal/errs"
	"github.com/timohahaa/tracker/internal/model"
	"github.com/timohahaa/tracker/internal/repository"
	"github.com/timohahaa/tracker/test/dbtest"
)

func TestUserRepo_CRUD(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	db := dbtest.Open(t)
	repos := repository.New(db)

	email := "user-" + uuid.NewString() + "@test.local"
	created, err := repos.User.Create(ctx, model.User{
		Email:        email,
		PasswordHash: "hash",
		DisplayName:  "Test User",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	t.Cleanup(func() { _, _ = db.Exec(`DELETE FROM core.user WHERE id = $1`, created.ID) })

	if created.ID == "" {
		t.Fatal("Create: empty id")
	}
	if created.CreatedAt.IsZero() {
		t.Error("Create: created_at not populated")
	}

	byEmail, err := repos.User.GetByEmail(ctx, email)
	if err != nil {
		t.Fatalf("GetByEmail: %v", err)
	}
	if byEmail.ID != created.ID {
		t.Errorf("GetByEmail id = %q, want %q", byEmail.ID, created.ID)
	}

	// Регистронезависимый поиск по email.
	if _, err := repos.User.GetByEmail(ctx, strings.ToUpper(email)); err != nil {
		t.Errorf("GetByEmail (upper): %v", err)
	}

	byID, err := repos.User.GetByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if byID.Email != email {
		t.Errorf("GetByID email = %q, want %q", byID.Email, email)
	}
}

func TestUserRepo_DuplicateEmail(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	db := dbtest.Open(t)
	repos := repository.New(db)

	email := "dup-" + uuid.NewString() + "@test.local"
	first, err := repos.User.Create(ctx, model.User{Email: email, PasswordHash: "h", DisplayName: "d"})
	if err != nil {
		t.Fatalf("Create first: %v", err)
	}
	t.Cleanup(func() { _, _ = db.Exec(`DELETE FROM core.user WHERE id = $1`, first.ID) })

	_, err = repos.User.Create(ctx, model.User{Email: email, PasswordHash: "h", DisplayName: "d"})
	if err == nil {
		t.Fatal("Create duplicate: expected error, got nil")
	}
	var appErr *errs.AppError
	if !errors.As(err, &appErr) || appErr.Code != errs.CodeAlreadyExists {
		t.Fatalf("Create duplicate: want AlreadyExists, got %v", err)
	}
}

func TestUserRepo_GetByID_NotFound(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	repos := repository.New(dbtest.Open(t))

	_, err := repos.User.GetByID(ctx, uuid.NewString())
	var appErr *errs.AppError
	if !errors.As(err, &appErr) || appErr.Code != errs.CodeNotFound {
		t.Fatalf("GetByID missing: want NotFound, got %v", err)
	}
}
