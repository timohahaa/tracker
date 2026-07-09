//go:build integration

package repository_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/timohahaa/tracker/internal/errs"
	"github.com/timohahaa/tracker/internal/model"
	"github.com/timohahaa/tracker/internal/repository"
	"github.com/timohahaa/tracker/test/dbtest"
)

func TestProjectRepo_CRUD(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	db := dbtest.Open(t)
	repos := repository.New(db)

	wsID := dbtest.SeedWorkspace(t, db)
	leadID := dbtest.SeedUser(t, db)

	desc := "Q3 deliverables"
	created, err := repos.Project().Create(ctx, model.Project{
		WorkspaceID: wsID,
		LeadID:      &leadID,
		Name:        "Roadmap",
		Description: &desc,
		Status:      model.ProjectStatusPlanned,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	t.Cleanup(func() { _, _ = db.Exec(`DELETE FROM tracker.project WHERE id = $1`, created.ID) })
	if created.ID == "" || created.Status != model.ProjectStatusPlanned || created.LeadID == nil || *created.LeadID != leadID {
		t.Fatalf("Create = %+v, want id/planned/lead set", created)
	}

	got, err := repos.Project().GetByID(ctx, wsID, created.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.ID != created.ID || got.Name != "Roadmap" {
		t.Errorf("GetByID = %+v, want id %q / Roadmap", got, created.ID)
	}

	// GetByID с чужим workspace → NotFound.
	_, err = repos.Project().GetByID(ctx, uuid.NewString(), created.ID)
	assertCode(t, err, errs.CodeNotFound)

	// Update: lead снимается, статус меняется.
	upd := created
	upd.Name = "Roadmap 2"
	upd.Status = model.ProjectStatusInProgress
	upd.LeadID = nil
	updated, err := repos.Project().Update(ctx, upd)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Name != "Roadmap 2" || updated.Status != model.ProjectStatusInProgress || updated.LeadID != nil {
		t.Errorf("Update = %+v, want Roadmap 2/in_progress/no lead", updated)
	}

	// Soft delete → GetByID возвращает NotFound.
	if err := repos.Project().Delete(ctx, created.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	_, err = repos.Project().GetByID(ctx, wsID, created.ID)
	assertCode(t, err, errs.CodeNotFound)
}

func TestProjectRepo_List(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	db := dbtest.Open(t)
	repos := repository.New(db)

	wsID := dbtest.SeedWorkspace(t, db)
	p1 := dbtest.SeedProject(t, db, wsID)
	p2 := dbtest.SeedProject(t, db, wsID)
	p3 := dbtest.SeedProject(t, db, wsID)

	// created_at asc: порядок вставки p1, p2, p3.
	rows, err := repos.Project().List(ctx, wsID, model.Page{Limit: 50, Sort: "created_at", Order: model.SortAsc})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(rows) != 3 || rows[0].ID != p1 || rows[1].ID != p2 || rows[2].ID != p3 {
		t.Fatalf("List = %+v, want [%s %s %s]", rows, p1, p2, p3)
	}

	// keyset: limit=2 → страница из 2 + «лишняя» строка для has_more.
	page1, err := repos.Project().List(ctx, wsID, model.Page{Limit: 2, Sort: "created_at", Order: model.SortAsc})
	if err != nil {
		t.Fatalf("List page1: %v", err)
	}
	if len(page1) != 3 {
		t.Fatalf("List page1 len = %d, want 3 (limit+1)", len(page1))
	}
	last := page1[1]
	cursor := &model.Cursor{Value: last.CreatedAt.Format("2006-01-02T15:04:05.999999999Z07:00"), ID: last.ID}
	page2, err := repos.Project().List(ctx, wsID, model.Page{Limit: 2, Sort: "created_at", Order: model.SortAsc, Cursor: cursor})
	if err != nil {
		t.Fatalf("List page2: %v", err)
	}
	if len(page2) != 1 || page2[0].ID != p3 {
		t.Fatalf("List page2 = %+v, want [%s]", page2, p3)
	}
}

func TestProjectRepo_Teams(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	db := dbtest.Open(t)
	repos := repository.New(db)

	wsID := dbtest.SeedWorkspace(t, db)
	projID := dbtest.SeedProject(t, db, wsID)
	teamA := dbtest.SeedTeam(t, db, wsID)
	teamB := dbtest.SeedTeam(t, db, wsID)

	if _, err := repos.Project().AddTeam(ctx, projID, teamA); err != nil {
		t.Fatalf("AddTeam A: %v", err)
	}
	t.Cleanup(func() { _, _ = db.Exec(`DELETE FROM tracker.project_team WHERE project_id = $1`, projID) })
	if _, err := repos.Project().AddTeam(ctx, projID, teamB); err != nil {
		t.Fatalf("AddTeam B: %v", err)
	}

	teams, err := repos.Project().ListTeams(ctx, projID)
	if err != nil {
		t.Fatalf("ListTeams: %v", err)
	}
	if len(teams) != 2 {
		t.Fatalf("ListTeams len = %d, want 2", len(teams))
	}

	// RemoveTeam + повторный AddTeam восстанавливает связь (deleted_at = NULL),
	// а не создаёт дубль composite PK.
	if err := repos.Project().RemoveTeam(ctx, projID, teamA); err != nil {
		t.Fatalf("RemoveTeam: %v", err)
	}
	teams, err = repos.Project().ListTeams(ctx, projID)
	if err != nil {
		t.Fatalf("ListTeams after remove: %v", err)
	}
	if len(teams) != 1 || teams[0].TeamID != teamB {
		t.Fatalf("ListTeams after remove = %+v, want [%s]", teams, teamB)
	}

	restored, err := repos.Project().AddTeam(ctx, projID, teamA)
	if err != nil {
		t.Fatalf("AddTeam restore: %v", err)
	}
	if restored.DeletedAt != nil {
		t.Errorf("restored team deleted_at = %v, want nil", restored.DeletedAt)
	}

	// ListTeamsByProjects: батч по нескольким проектам.
	other := dbtest.SeedProject(t, db, wsID)
	if _, err := repos.Project().AddTeam(ctx, other, teamB); err != nil {
		t.Fatalf("AddTeam other: %v", err)
	}
	t.Cleanup(func() { _, _ = db.Exec(`DELETE FROM tracker.project_team WHERE project_id = $1`, other) })

	byProject, err := repos.Project().ListTeamsByProjects(ctx, []string{projID, other})
	if err != nil {
		t.Fatalf("ListTeamsByProjects: %v", err)
	}
	if len(byProject[projID]) != 2 {
		t.Errorf("byProject[%s] len = %d, want 2", projID, len(byProject[projID]))
	}
	if len(byProject[other]) != 1 {
		t.Errorf("byProject[%s] len = %d, want 1", other, len(byProject[other]))
	}
}
