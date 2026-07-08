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

func TestIssueRepo_CRUDAndIdentifier(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	db := dbtest.Open(t)
	repos := repository.New(db)

	wsID := dbtest.SeedWorkspace(t, db)
	teamID := dbtest.SeedTeam(t, db, wsID)
	statusID := dbtest.SeedStatus(t, db, teamID, "Backlog", "backlog", 0)
	creatorID := dbtest.SeedUser(t, db)

	team, err := repos.Team().Get(ctx, wsID, teamID)
	if err != nil {
		t.Fatalf("Team.Get: %v", err)
	}
	key := team.Key

	// Первый инкремент → номер 1 (next_issue_number DEFAULT 0).
	n1, err := repos.Issue().IncrementIssueNumber(ctx, teamID)
	if err != nil {
		t.Fatalf("IncrementIssueNumber: %v", err)
	}
	if n1 != 1 {
		t.Fatalf("first issue number = %d, want 1", n1)
	}

	desc := "first issue body"
	created, err := repos.Issue().Create(ctx, model.Issue{
		TeamID:      teamID,
		StatusID:    statusID,
		CreatorID:   &creatorID,
		Title:       "First",
		Description: &desc,
		Priority:    model.PriorityHigh,
		IssueNumber: n1,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	t.Cleanup(func() { _, _ = db.Exec(`DELETE FROM tracker.issue WHERE id = $1`, created.ID) })
	if created.ID == "" || created.IssueNumber != 1 {
		t.Fatalf("Create = %+v, want id set, number 1", created)
	}

	// Get по UUID: identifier/team_key вычисляются JOIN'ом.
	byID, err := repos.Issue().GetByID(ctx, wsID, created.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if byID.Identifier != key+"-1" || byID.TeamKey != key {
		t.Errorf("GetByID identifier/key = (%q, %q), want (%q, %q)", byID.Identifier, byID.TeamKey, key+"-1", key)
	}
	if byID.CreatorID == nil || *byID.CreatorID != creatorID {
		t.Errorf("GetByID creator = %v, want %q", byID.CreatorID, creatorID)
	}

	// Get по identifier: lookup через team.key + issue_number.
	byIdent, err := repos.Issue().GetByIdentifier(ctx, wsID, key, 1)
	if err != nil {
		t.Fatalf("GetByIdentifier: %v", err)
	}
	if byIdent.ID != created.ID {
		t.Errorf("GetByIdentifier id = %q, want %q", byIdent.ID, created.ID)
	}

	// Update: partial (title/description/status/priority/due_date/project все в одном SET).
	otherStatus := dbtest.SeedStatus(t, db, teamID, "In Progress", "in_progress", 0)
	updIssue := created
	updIssue.Title = "First (renamed)"
	updIssue.StatusID = otherStatus
	updIssue.Priority = model.PriorityLow
	updated, err := repos.Issue().Update(ctx, updIssue)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Title != "First (renamed)" || updated.StatusID != otherStatus || updated.Priority != model.PriorityLow {
		t.Errorf("Update = %+v, want renamed/in_progress/low", updated)
	}

	// Soft delete → GetByID возвращает NotFound.
	if err := repos.Issue().Delete(ctx, created.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	_, err = repos.Issue().GetByID(ctx, wsID, created.ID)
	assertCode(t, err, errs.CodeNotFound)
}

func TestIssueRepo_IssueNumberPerTeam(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	db := dbtest.Open(t)
	repos := repository.New(db)

	wsID := dbtest.SeedWorkspace(t, db)
	teamA := dbtest.SeedTeam(t, db, wsID)
	teamB := dbtest.SeedTeam(t, db, wsID)
	statusA := dbtest.SeedStatus(t, db, teamA, "Backlog", "backlog", 0)

	// Последовательный инкремент в команде A: 1, 2.
	a1, err := repos.Issue().IncrementIssueNumber(ctx, teamA)
	if err != nil {
		t.Fatalf("increment A1: %v", err)
	}
	a2, err := repos.Issue().IncrementIssueNumber(ctx, teamA)
	if err != nil {
		t.Fatalf("increment A2: %v", err)
	}
	if a1 != 1 || a2 != 2 {
		t.Fatalf("team A numbers = (%d, %d), want (1, 2)", a1, a2)
	}

	// Команда B ведёт независимый счётчик: снова начинает с 1.
	b1, err := repos.Issue().IncrementIssueNumber(ctx, teamB)
	if err != nil {
		t.Fatalf("increment B1: %v", err)
	}
	if b1 != 1 {
		t.Fatalf("team B first number = %d, want 1", b1)
	}

	first, err := repos.Issue().Create(ctx, model.Issue{TeamID: teamA, StatusID: statusA, Title: "A-1", Priority: model.PriorityNone, IssueNumber: a1})
	if err != nil {
		t.Fatalf("Create first: %v", err)
	}
	t.Cleanup(func() { _, _ = db.Exec(`DELETE FROM tracker.issue WHERE id = $1`, first.ID) })

	// Дубликат (team_id, issue_number) среди активных → AlreadyExists (partial unique).
	_, err = repos.Issue().Create(ctx, model.Issue{TeamID: teamA, StatusID: statusA, Title: "A-1 dup", Priority: model.PriorityNone, IssueNumber: a1})
	assertCode(t, err, errs.CodeAlreadyExists)
}

func TestIssueRepo_CrossWorkspaceNotFound(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	db := dbtest.Open(t)
	repos := repository.New(db)

	wsID := dbtest.SeedWorkspace(t, db)
	teamID := dbtest.SeedTeam(t, db, wsID)
	statusID := dbtest.SeedStatus(t, db, teamID, "Backlog", "backlog", 0)

	n, err := repos.Issue().IncrementIssueNumber(ctx, teamID)
	if err != nil {
		t.Fatalf("IncrementIssueNumber: %v", err)
	}
	created, err := repos.Issue().Create(ctx, model.Issue{TeamID: teamID, StatusID: statusID, Title: "X", Priority: model.PriorityNone, IssueNumber: n})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	t.Cleanup(func() { _, _ = db.Exec(`DELETE FROM tracker.issue WHERE id = $1`, created.ID) })

	// Задача существует, но запрошена от лица чужого workspace → NotFound.
	_, err = repos.Issue().GetByID(ctx, uuid.NewString(), created.ID)
	assertCode(t, err, errs.CodeNotFound)
}

func TestIssueRepo_ProjectInWorkspace(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	db := dbtest.Open(t)
	repos := repository.New(db)

	wsID := dbtest.SeedWorkspace(t, db)
	otherWS := dbtest.SeedWorkspace(t, db)

	var projID string
	if err := db.Get(&projID, `INSERT INTO tracker.project (workspace_id, name) VALUES ($1, $2) RETURNING id`, wsID, "Roadmap"); err != nil {
		t.Fatalf("seed project: %v", err)
	}
	t.Cleanup(func() { _, _ = db.Exec(`DELETE FROM tracker.project WHERE id = $1`, projID) })

	ok, err := repos.Issue().ProjectInWorkspace(ctx, wsID, projID)
	if err != nil {
		t.Fatalf("ProjectInWorkspace same ws: %v", err)
	}
	if !ok {
		t.Errorf("ProjectInWorkspace same ws = false, want true")
	}

	// Проект другого workspace → false.
	ok, err = repos.Issue().ProjectInWorkspace(ctx, otherWS, projID)
	if err != nil {
		t.Fatalf("ProjectInWorkspace other ws: %v", err)
	}
	if ok {
		t.Errorf("ProjectInWorkspace other ws = true, want false")
	}

	// Несуществующий проект → false.
	ok, err = repos.Issue().ProjectInWorkspace(ctx, wsID, uuid.NewString())
	if err != nil {
		t.Fatalf("ProjectInWorkspace missing: %v", err)
	}
	if ok {
		t.Errorf("ProjectInWorkspace missing = true, want false")
	}
}
