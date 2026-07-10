//go:build integration

package repository_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"github.com/timohahaa/tracker/internal/errs"
	"github.com/timohahaa/tracker/internal/model"
	"github.com/timohahaa/tracker/internal/repository"
	"github.com/timohahaa/tracker/test/dbtest"
)

func TestLabelRepo_CRUD(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	db := dbtest.Open(t)
	repos := repository.New(db)

	wsID := dbtest.SeedWorkspace(t, db)
	teamID := dbtest.SeedTeam(t, db, wsID)

	// workspace-level метка.
	wsLabel, err := repos.Label().Create(ctx, model.Label{
		WorkspaceID: wsID,
		Name:        "bug",
		Color:       "#3B82F6",
	})
	if err != nil {
		t.Fatalf("Create ws-level: %v", err)
	}
	t.Cleanup(func() { _, _ = db.Exec(`DELETE FROM tracker.label WHERE id = $1`, wsLabel.ID) })
	if wsLabel.ID == "" || wsLabel.TeamID != nil {
		t.Fatalf("Create ws-level = %+v, want id set / team_id nil", wsLabel)
	}

	// team-level метка.
	teamLabel, err := repos.Label().Create(ctx, model.Label{
		WorkspaceID: wsID,
		TeamID:      &teamID,
		Name:        "feature",
		Color:       "#00FF00",
	})
	if err != nil {
		t.Fatalf("Create team-level: %v", err)
	}
	t.Cleanup(func() { _, _ = db.Exec(`DELETE FROM tracker.label WHERE id = $1`, teamLabel.ID) })
	if teamLabel.TeamID == nil || *teamLabel.TeamID != teamID {
		t.Fatalf("Create team-level = %+v, want team_id %q", teamLabel, teamID)
	}

	// GetByID.
	got, err := repos.Label().GetByID(ctx, wsID, wsLabel.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Name != "bug" {
		t.Errorf("GetByID = %+v, want bug", got)
	}

	// GetByID с чужим workspace → NotFound.
	_, err = repos.Label().GetByID(ctx, uuid.NewString(), wsLabel.ID)
	assertCode(t, err, errs.CodeNotFound)

	// List: обе метки активны.
	list, err := repos.Label().List(ctx, wsID)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("List len = %d, want 2", len(list))
	}

	// Update.
	upd := wsLabel
	upd.Name = "critical"
	upd.Color = "#FF0000"
	updated, err := repos.Label().Update(ctx, upd)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Name != "critical" || updated.Color != "#FF0000" {
		t.Errorf("Update = %+v, want critical / #FF0000", updated)
	}

	// Soft delete → GetByID NotFound, а List возвращает только оставшуюся.
	if err := repos.Label().Delete(ctx, wsLabel.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	_, err = repos.Label().GetByID(ctx, wsID, wsLabel.ID)
	assertCode(t, err, errs.CodeNotFound)

	list, err = repos.Label().List(ctx, wsID)
	if err != nil {
		t.Fatalf("List after delete: %v", err)
	}
	if len(list) != 1 || list[0].ID != teamLabel.ID {
		t.Fatalf("List after delete = %+v, want [%s]", list, teamLabel.ID)
	}
}

func TestLabelRepo_PartialUniqueIndexes(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	db := dbtest.Open(t)
	repos := repository.New(db)

	wsID := dbtest.SeedWorkspace(t, db)
	teamID := dbtest.SeedTeam(t, db, wsID)
	otherTeamID := dbtest.SeedTeam(t, db, wsID)
	t.Cleanup(func() { _, _ = db.Exec(`DELETE FROM tracker.label WHERE workspace_id = $1`, wsID) })

	// Workspace-level дубль имени → AlreadyExists.
	if _, err := repos.Label().Create(ctx, model.Label{WorkspaceID: wsID, Name: "dup", Color: "#111111"}); err != nil {
		t.Fatalf("Create ws-level 1: %v", err)
	}
	_, err := repos.Label().Create(ctx, model.Label{WorkspaceID: wsID, Name: "dup", Color: "#222222"})
	assertCode(t, err, errs.CodeAlreadyExists)

	// Team-level дубль имени в той же команде → AlreadyExists.
	if _, err := repos.Label().Create(ctx, model.Label{WorkspaceID: wsID, TeamID: &teamID, Name: "tdup", Color: "#333333"}); err != nil {
		t.Fatalf("Create team-level 1: %v", err)
	}
	_, err = repos.Label().Create(ctx, model.Label{WorkspaceID: wsID, TeamID: &teamID, Name: "tdup", Color: "#444444"})
	assertCode(t, err, errs.CodeAlreadyExists)

	// Разные команды могут иметь одноимённые метки — не конфликт.
	if _, err := repos.Label().Create(ctx, model.Label{WorkspaceID: wsID, TeamID: &otherTeamID, Name: "tdup", Color: "#555555"}); err != nil {
		t.Fatalf("Create team-level other team: %v", err)
	}
}

func TestLabelRepo_RepointIssueLabels(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	db := dbtest.Open(t)
	repos := repository.New(db)

	wsID := dbtest.SeedWorkspace(t, db)
	teamID := dbtest.SeedTeam(t, db, wsID)
	statusID := dbtest.SeedStatus(t, db, teamID, "Todo", "planned", 0)
	issueID, _ := dbtest.SeedIssue(t, db, teamID, statusID)

	// Две старые team-level метки, обе висят на задаче.
	old1 := dbtest.SeedLabel(t, db, wsID, &teamID)
	old2 := dbtest.SeedLabel(t, db, wsID, &teamID)
	dbtest.SeedIssueLabel(t, db, issueID, old1)
	dbtest.SeedIssueLabel(t, db, issueID, old2)

	// Целевая workspace-level метка.
	target := dbtest.SeedLabel(t, db, wsID, nil)

	if err := repos.Label().RepointIssueLabels(ctx, []string{old1, old2}, target); err != nil {
		t.Fatalf("RepointIssueLabels: %v", err)
	}

	// Связь на целевую метку активна.
	var targetActive int
	if err := db.Get(&targetActive, `
		SELECT count(*) FROM tracker.issue_label
		WHERE issue_id = $1 AND label_id = $2 AND deleted_at IS NULL`, issueID, target); err != nil {
		t.Fatalf("count target: %v", err)
	}
	if targetActive != 1 {
		t.Errorf("target active links = %d, want 1", targetActive)
	}

	// Связи со старых меток помечены удалёнными.
	var oldActive int
	if err := db.Get(&oldActive, `
		SELECT count(*) FROM tracker.issue_label
		WHERE issue_id = $1 AND label_id = ANY($2::uuid[]) AND deleted_at IS NULL`, issueID, pq.Array([]string{old1, old2})); err != nil {
		t.Fatalf("count old: %v", err)
	}
	if oldActive != 0 {
		t.Errorf("old active links = %d, want 0", oldActive)
	}
}
