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

func TestTeamRepo_CRUD(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	db := dbtest.Open(t)
	repos := repository.New(db)

	wsID := dbtest.SeedWorkspace(t, db)
	key := "K" + uuid.NewString()[:8]

	created, err := repos.Team().Create(ctx, model.Team{WorkspaceID: wsID, Name: "Engineering", Key: key})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	t.Cleanup(func() { _, _ = db.Exec(`DELETE FROM org.team WHERE id = $1`, created.ID) })
	if created.ID == "" || created.Key != key || created.NextIssueNumber != 0 {
		t.Fatalf("Create = %+v, want id set, key %q, next=0", created, key)
	}

	got, err := repos.Team().Get(ctx, wsID, created.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.ID != created.ID {
		t.Errorf("Get id = %q, want %q", got.ID, created.ID)
	}

	// Get с чужим workspace → NotFound.
	_, err = repos.Team().Get(ctx, uuid.NewString(), created.ID)
	assertCode(t, err, errs.CodeNotFound)

	teams, err := repos.Team().List(ctx, wsID)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(teams) != 1 || teams[0].ID != created.ID {
		t.Fatalf("List = %+v, want [%q]", teams, created.ID)
	}

	desc := "Builds things"
	newKey := "U" + uuid.NewString()[:8]
	updated, err := repos.Team().Update(ctx, model.Team{ID: created.ID, Name: "Renamed", Description: &desc, Key: newKey})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Name != "Renamed" || updated.Key != newKey || updated.Description == nil || *updated.Description != desc {
		t.Errorf("Update = %+v, want renamed/%q/%q", updated, newKey, desc)
	}

	if err := repos.Team().Delete(ctx, created.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	_, err = repos.Team().Get(ctx, wsID, created.ID)
	assertCode(t, err, errs.CodeNotFound)
}

func TestTeamRepo_KeyConflict(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	db := dbtest.Open(t)
	repos := repository.New(db)

	wsID := dbtest.SeedWorkspace(t, db)
	key := "K" + uuid.NewString()[:8]

	first, err := repos.Team().Create(ctx, model.Team{WorkspaceID: wsID, Name: "One", Key: key})
	if err != nil {
		t.Fatalf("Create first: %v", err)
	}
	t.Cleanup(func() { _, _ = db.Exec(`DELETE FROM org.team WHERE id = $1`, first.ID) })

	_, err = repos.Team().Create(ctx, model.Team{WorkspaceID: wsID, Name: "Two", Key: key})
	assertCode(t, err, errs.CodeAlreadyExists)
}

func TestTeamRepo_KeyReuseAfterSoftDelete(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	db := dbtest.Open(t)
	repos := repository.New(db)

	wsID := dbtest.SeedWorkspace(t, db)
	key := "K" + uuid.NewString()[:8]

	first, err := repos.Team().Create(ctx, model.Team{WorkspaceID: wsID, Name: "One", Key: key})
	if err != nil {
		t.Fatalf("Create first: %v", err)
	}
	t.Cleanup(func() { _, _ = db.Exec(`DELETE FROM org.team WHERE id = $1`, first.ID) })

	// После soft delete партиальный unique index (WHERE deleted_at IS NULL)
	// освобождает key — команду с тем же key можно создать заново.
	if err := repos.Team().Delete(ctx, first.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	second, err := repos.Team().Create(ctx, model.Team{WorkspaceID: wsID, Name: "Two", Key: key})
	if err != nil {
		t.Fatalf("Create after soft delete: %v", err)
	}
	t.Cleanup(func() { _, _ = db.Exec(`DELETE FROM org.team WHERE id = $1`, second.ID) })
}

func TestTeamRepo_UpdateKeyConflict(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	db := dbtest.Open(t)
	repos := repository.New(db)

	wsID := dbtest.SeedWorkspace(t, db)
	keyA := "A" + uuid.NewString()[:8]
	keyB := "B" + uuid.NewString()[:8]

	a, err := repos.Team().Create(ctx, model.Team{WorkspaceID: wsID, Name: "A", Key: keyA})
	if err != nil {
		t.Fatalf("Create A: %v", err)
	}
	t.Cleanup(func() { _, _ = db.Exec(`DELETE FROM org.team WHERE id = $1`, a.ID) })

	b, err := repos.Team().Create(ctx, model.Team{WorkspaceID: wsID, Name: "B", Key: keyB})
	if err != nil {
		t.Fatalf("Create B: %v", err)
	}
	t.Cleanup(func() { _, _ = db.Exec(`DELETE FROM org.team WHERE id = $1`, b.ID) })

	// Переименование key команды B в уже занятый keyA → AlreadyExists.
	_, err = repos.Team().Update(ctx, model.Team{ID: b.ID, Name: "B", Key: keyA})
	assertCode(t, err, errs.CodeAlreadyExists)
}

func TestTeamRepo_Members(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	db := dbtest.Open(t)
	repos := repository.New(db)

	wsID := dbtest.SeedWorkspace(t, db)
	teamID := dbtest.SeedTeam(t, db, wsID)
	userID := dbtest.SeedUser(t, db)

	added, err := repos.Team().AddMember(ctx, model.TeamMember{TeamID: teamID, UserID: userID, Role: model.TeamRoleMember})
	if err != nil {
		t.Fatalf("AddMember: %v", err)
	}
	t.Cleanup(func() { _, _ = db.Exec(`DELETE FROM org.team_member WHERE team_id = $1 AND user_id = $2`, teamID, userID) })
	if added.Role != model.TeamRoleMember {
		t.Errorf("AddMember role = %q, want member", added.Role)
	}

	// Повторный AddMember с другой ролью — upsert (роль обновляется, дубля нет).
	upserted, err := repos.Team().AddMember(ctx, model.TeamMember{TeamID: teamID, UserID: userID, Role: model.TeamRoleOwner})
	if err != nil {
		t.Fatalf("AddMember upsert: %v", err)
	}
	if upserted.Role != model.TeamRoleOwner {
		t.Errorf("upsert role = %q, want owner", upserted.Role)
	}

	got, err := repos.Team().GetMember(ctx, teamID, userID)
	if err != nil {
		t.Fatalf("GetMember: %v", err)
	}
	if got.UserID != userID {
		t.Errorf("GetMember user = %q, want %q", got.UserID, userID)
	}

	members, err := repos.Team().ListMembers(ctx, teamID)
	if err != nil {
		t.Fatalf("ListMembers: %v", err)
	}
	if len(members) != 1 {
		t.Fatalf("ListMembers len = %d, want 1", len(members))
	}

	if err := repos.Team().RemoveMember(ctx, teamID, userID); err != nil {
		t.Fatalf("RemoveMember: %v", err)
	}
	_, err = repos.Team().GetMember(ctx, teamID, userID)
	assertCode(t, err, errs.CodeNotFound)

	// Повторное добавление после удаления — upsert восстанавливает связь
	// (deleted_at = NULL), а не создаёт дубль composite PK.
	restored, err := repos.Team().AddMember(ctx, model.TeamMember{TeamID: teamID, UserID: userID, Role: model.TeamRoleMember})
	if err != nil {
		t.Fatalf("AddMember restore: %v", err)
	}
	if restored.DeletedAt != nil {
		t.Errorf("restored member deleted_at = %v, want nil", restored.DeletedAt)
	}
	if _, err := repos.Team().GetMember(ctx, teamID, userID); err != nil {
		t.Fatalf("GetMember after restore: %v", err)
	}
}

func TestStatusRepo_CRUD(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	db := dbtest.Open(t)
	repos := repository.New(db)

	wsID := dbtest.SeedWorkspace(t, db)
	teamID := dbtest.SeedTeam(t, db, wsID)

	s1, err := repos.Status().Create(ctx, model.Status{TeamID: teamID, Name: "Backlog", Group: model.StatusGroupBacklog, Position: 0})
	if err != nil {
		t.Fatalf("Create s1: %v", err)
	}
	t.Cleanup(func() { _, _ = db.Exec(`DELETE FROM tracker.status WHERE id = $1`, s1.ID) })
	if s1.Group != model.StatusGroupBacklog {
		t.Errorf("Create group = %q, want backlog", s1.Group)
	}

	s2, err := repos.Status().Create(ctx, model.Status{TeamID: teamID, Name: "Triage", Group: model.StatusGroupBacklog, Position: 1})
	if err != nil {
		t.Fatalf("Create s2: %v", err)
	}
	t.Cleanup(func() { _, _ = db.Exec(`DELETE FROM tracker.status WHERE id = $1`, s2.ID) })

	// Конфликт позиции внутри группы → AlreadyExists.
	_, err = repos.Status().Create(ctx, model.Status{TeamID: teamID, Name: "Dup", Group: model.StatusGroupBacklog, Position: 0})
	assertCode(t, err, errs.CodeAlreadyExists)

	got, err := repos.Status().Get(ctx, teamID, s1.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Group != model.StatusGroupBacklog || got.Name != "Backlog" {
		t.Errorf("Get = %+v, want backlog/Backlog", got)
	}

	list, err := repos.Status().List(ctx, teamID)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("List len = %d, want 2", len(list))
	}

	count, err := repos.Status().CountByGroup(ctx, teamID, model.StatusGroupBacklog)
	if err != nil {
		t.Fatalf("CountByGroup: %v", err)
	}
	if count != 2 {
		t.Errorf("CountByGroup = %d, want 2", count)
	}

	updated, err := repos.Status().Update(ctx, model.Status{ID: s2.ID, Name: "In Triage", Position: 5})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Name != "In Triage" || updated.Position != 5 || updated.Group != model.StatusGroupBacklog {
		t.Errorf("Update = %+v, want In Triage/5/backlog", updated)
	}

	if err := repos.Status().Delete(ctx, s2.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if count, err = repos.Status().CountByGroup(ctx, teamID, model.StatusGroupBacklog); err != nil {
		t.Fatalf("CountByGroup after delete: %v", err)
	} else if count != 1 {
		t.Errorf("CountByGroup after delete = %d, want 1", count)
	}
}

func TestStatusRepo_UpdatePositionConflict(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	db := dbtest.Open(t)
	repos := repository.New(db)

	wsID := dbtest.SeedWorkspace(t, db)
	teamID := dbtest.SeedTeam(t, db, wsID)

	s1, err := repos.Status().Create(ctx, model.Status{TeamID: teamID, Name: "Backlog", Group: model.StatusGroupBacklog, Position: 0})
	if err != nil {
		t.Fatalf("Create s1: %v", err)
	}
	t.Cleanup(func() { _, _ = db.Exec(`DELETE FROM tracker.status WHERE id = $1`, s1.ID) })

	s2, err := repos.Status().Create(ctx, model.Status{TeamID: teamID, Name: "Triage", Group: model.StatusGroupBacklog, Position: 1})
	if err != nil {
		t.Fatalf("Create s2: %v", err)
	}
	t.Cleanup(func() { _, _ = db.Exec(`DELETE FROM tracker.status WHERE id = $1`, s2.ID) })

	// Перенос s2 на позицию, занятую s1 в той же группе → AlreadyExists.
	_, err = repos.Status().Update(ctx, model.Status{ID: s2.ID, Name: "Triage", Position: 0})
	assertCode(t, err, errs.CodeAlreadyExists)
}

func TestStatusRepo_PositionReuseAfterSoftDelete(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	db := dbtest.Open(t)
	repos := repository.New(db)

	wsID := dbtest.SeedWorkspace(t, db)
	teamID := dbtest.SeedTeam(t, db, wsID)

	s1, err := repos.Status().Create(ctx, model.Status{TeamID: teamID, Name: "Backlog", Group: model.StatusGroupBacklog, Position: 0})
	if err != nil {
		t.Fatalf("Create s1: %v", err)
	}
	t.Cleanup(func() { _, _ = db.Exec(`DELETE FROM tracker.status WHERE id = $1`, s1.ID) })

	// После soft delete партиальный unique index освобождает (group, position) —
	// статус с той же позицией в группе можно создать заново.
	if err := repos.Status().Delete(ctx, s1.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	s2, err := repos.Status().Create(ctx, model.Status{TeamID: teamID, Name: "New Backlog", Group: model.StatusGroupBacklog, Position: 0})
	if err != nil {
		t.Fatalf("Create after soft delete: %v", err)
	}
	t.Cleanup(func() { _, _ = db.Exec(`DELETE FROM tracker.status WHERE id = $1`, s2.ID) })
}

// assertCode проверяет, что err — *errs.AppError с ожидаемым кодом.
func assertCode(t *testing.T, err error, want errs.Code) {
	t.Helper()
	var appErr *errs.AppError
	if !errors.As(err, &appErr) || appErr.Code != want {
		t.Fatalf("error = %v, want code %s", err, want)
	}
}
