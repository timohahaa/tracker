//go:build integration

package repository_test

import (
	"context"
	"testing"

	"github.com/timohahaa/tracker/internal/model"
	"github.com/timohahaa/tracker/internal/repository"
	"github.com/timohahaa/tracker/test/dbtest"
)

// labelIDs собирает id активных меток в порядке ответа.
func labelIDs(ls []model.Label) []string {
	out := make([]string, 0, len(ls))
	for _, l := range ls {
		out = append(out, l.ID)
	}
	return out
}

func TestIssueLabelRepo_UpsertRestoreAndSoftDelete(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	db := dbtest.Open(t)
	repos := repository.New(db)

	wsID := dbtest.SeedWorkspace(t, db)
	teamID := dbtest.SeedTeam(t, db, wsID)
	statusID := dbtest.SeedStatus(t, db, teamID, "Backlog", "backlog", 0)
	issueID, _ := dbtest.SeedIssue(t, db, teamID, statusID)
	labelID := dbtest.SeedLabel(t, db, wsID, nil)

	// Первая привязка (INSERT).
	if err := repos.Issue().AddIssueLabel(ctx, issueID, labelID); err != nil {
		t.Fatalf("AddIssueLabel: %v", err)
	}
	active, err := repos.Issue().ListLabels(ctx, issueID)
	if err != nil {
		t.Fatalf("ListLabels: %v", err)
	}
	if len(active) != 1 || active[0].ID != labelID {
		t.Fatalf("active = %v, want [%s]", labelIDs(active), labelID)
	}

	// Soft delete → в активном списке нет.
	if err := repos.Issue().RemoveIssueLabel(ctx, issueID, labelID); err != nil {
		t.Fatalf("RemoveIssueLabel: %v", err)
	}
	active, err = repos.Issue().ListLabels(ctx, issueID)
	if err != nil {
		t.Fatalf("ListLabels after remove: %v", err)
	}
	if len(active) != 0 {
		t.Fatalf("active after remove = %d, want 0 (deleted_at filtered)", len(active))
	}

	// Повторная привязка восстанавливает soft-deleted (без дубля PK).
	if err := repos.Issue().AddIssueLabel(ctx, issueID, labelID); err != nil {
		t.Fatalf("AddIssueLabel restore: %v", err)
	}
	active, err = repos.Issue().ListLabels(ctx, issueID)
	if err != nil {
		t.Fatalf("ListLabels after restore: %v", err)
	}
	if len(active) != 1 || active[0].ID != labelID {
		t.Fatalf("active after restore = %v, want [%s]", labelIDs(active), labelID)
	}

	// Идемпотентный повторный Add активной связи не плодит дублей.
	if err := repos.Issue().AddIssueLabel(ctx, issueID, labelID); err != nil {
		t.Fatalf("AddIssueLabel idempotent: %v", err)
	}
	var count int
	if err := db.Get(&count, `SELECT count(*) FROM tracker.issue_label WHERE issue_id = $1 AND label_id = $2`, issueID, labelID); err != nil {
		t.Fatalf("count rows: %v", err)
	}
	if count != 1 {
		t.Fatalf("rows for (issue,label) = %d, want 1 (upsert, no PK dup)", count)
	}

	// Повторное удаление идемпотентно.
	if err := repos.Issue().RemoveIssueLabel(ctx, issueID, labelID); err != nil {
		t.Fatalf("RemoveIssueLabel idempotent: %v", err)
	}
	if err := repos.Issue().RemoveIssueLabel(ctx, issueID, labelID); err != nil {
		t.Fatalf("RemoveIssueLabel second call: %v", err)
	}
}

func TestIssueLabelRepo_ListByIssuesBatch(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	db := dbtest.Open(t)
	repos := repository.New(db)

	wsID := dbtest.SeedWorkspace(t, db)
	teamID := dbtest.SeedTeam(t, db, wsID)
	statusID := dbtest.SeedStatus(t, db, teamID, "Backlog", "backlog", 0)
	issueA, _ := dbtest.SeedIssue(t, db, teamID, statusID)
	issueB, _ := dbtest.SeedIssue(t, db, teamID, statusID)
	l1 := dbtest.SeedLabel(t, db, wsID, nil)
	l2 := dbtest.SeedLabel(t, db, wsID, &teamID)
	l3 := dbtest.SeedLabel(t, db, wsID, nil)

	dbtest.SeedIssueLabel(t, db, issueA, l1)
	dbtest.SeedIssueLabel(t, db, issueA, l2)
	dbtest.SeedIssueLabel(t, db, issueB, l3)
	// Soft-deleted связь не попадает в батч.
	dbtest.SeedIssueLabel(t, db, issueB, l1)
	if _, err := db.Exec(`UPDATE tracker.issue_label SET deleted_at = now() WHERE issue_id = $1 AND label_id = $2`, issueB, l1); err != nil {
		t.Fatalf("soft delete: %v", err)
	}

	byIssue, err := repos.Issue().ListLabelsByIssues(ctx, []string{issueA, issueB})
	if err != nil {
		t.Fatalf("ListLabelsByIssues: %v", err)
	}
	if len(byIssue[issueA]) != 2 {
		t.Errorf("issueA labels = %v, want 2", labelIDs(byIssue[issueA]))
	}
	if len(byIssue[issueB]) != 1 || byIssue[issueB][0].ID != l3 {
		t.Errorf("issueB labels = %v, want [%s]", labelIDs(byIssue[issueB]), l3)
	}

	// Soft-deleted метка исключается даже при активной связи.
	if _, err := db.Exec(`UPDATE tracker.label SET deleted_at = now() WHERE id = $1`, l2); err != nil {
		t.Fatalf("soft delete label: %v", err)
	}
	byIssue, err = repos.Issue().ListLabelsByIssues(ctx, []string{issueA})
	if err != nil {
		t.Fatalf("ListLabelsByIssues after label delete: %v", err)
	}
	if len(byIssue[issueA]) != 1 || byIssue[issueA][0].ID != l1 {
		t.Errorf("issueA labels after label delete = %v, want [%s]", labelIDs(byIssue[issueA]), l1)
	}

	// Пустой вход → пустая map, без запроса.
	empty, err := repos.Issue().ListLabelsByIssues(ctx, nil)
	if err != nil {
		t.Fatalf("ListLabelsByIssues empty: %v", err)
	}
	if len(empty) != 0 {
		t.Errorf("empty input = %v, want empty map", empty)
	}
}
