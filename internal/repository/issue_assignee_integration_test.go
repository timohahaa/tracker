//go:build integration

package repository_test

import (
	"context"
	"testing"

	"github.com/timohahaa/tracker/internal/errs"
	"github.com/timohahaa/tracker/internal/model"
	"github.com/timohahaa/tracker/internal/repository"
	"github.com/timohahaa/tracker/test/dbtest"
)

// assigneeUserIDs собирает user_id активных ассайни в порядке ответа.
func assigneeUserIDs(as []model.IssueAssignee) []string {
	out := make([]string, 0, len(as))
	for _, a := range as {
		out = append(out, a.UserID)
	}
	return out
}

func TestIssueAssigneeRepo_UpsertRestoreAndSoftDelete(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	db := dbtest.Open(t)
	repos := repository.New(db)

	wsID := dbtest.SeedWorkspace(t, db)
	teamID := dbtest.SeedTeam(t, db, wsID)
	statusID := dbtest.SeedStatus(t, db, teamID, "Backlog", "backlog", 0)
	issueID, _ := dbtest.SeedIssue(t, db, teamID, statusID)
	userID := dbtest.SeedUser(t, db)

	// Первое добавление (INSERT).
	added, err := repos.Issue().AddAssignee(ctx, model.IssueAssignee{IssueID: issueID, UserID: userID})
	if err != nil {
		t.Fatalf("AddAssignee: %v", err)
	}
	if added.DeletedAt != nil || added.IsPrimary {
		t.Fatalf("added = %+v, want active, non-primary", added)
	}

	// Soft delete → в активном списке нет.
	if err := repos.Issue().RemoveAssignee(ctx, issueID, userID); err != nil {
		t.Fatalf("RemoveAssignee: %v", err)
	}
	active, err := repos.Issue().ListAssignees(ctx, issueID)
	if err != nil {
		t.Fatalf("ListAssignees after remove: %v", err)
	}
	if len(active) != 0 {
		t.Fatalf("active after remove = %d, want 0 (deleted_at filtered)", len(active))
	}

	// Повторное добавление восстанавливает soft-deleted (deleted_at = NULL).
	restored, err := repos.Issue().AddAssignee(ctx, model.IssueAssignee{IssueID: issueID, UserID: userID})
	if err != nil {
		t.Fatalf("AddAssignee restore: %v", err)
	}
	if restored.DeletedAt != nil {
		t.Fatalf("restored.DeletedAt = %v, want nil (upsert restore)", restored.DeletedAt)
	}
	active, err = repos.Issue().ListAssignees(ctx, issueID)
	if err != nil {
		t.Fatalf("ListAssignees after restore: %v", err)
	}
	if len(active) != 1 || active[0].UserID != userID {
		t.Fatalf("active after restore = %+v, want [%s]", assigneeUserIDs(active), userID)
	}

	// Повторное удаление идемпотентно.
	if err := repos.Issue().RemoveAssignee(ctx, issueID, userID); err != nil {
		t.Fatalf("RemoveAssignee idempotent: %v", err)
	}
	if err := repos.Issue().RemoveAssignee(ctx, issueID, userID); err != nil {
		t.Fatalf("RemoveAssignee second call: %v", err)
	}
}

func TestIssueAssigneeRepo_ListByIssuesBatch(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	db := dbtest.Open(t)
	repos := repository.New(db)

	wsID := dbtest.SeedWorkspace(t, db)
	teamID := dbtest.SeedTeam(t, db, wsID)
	statusID := dbtest.SeedStatus(t, db, teamID, "Backlog", "backlog", 0)
	issueA, _ := dbtest.SeedIssue(t, db, teamID, statusID)
	issueB, _ := dbtest.SeedIssue(t, db, teamID, statusID)
	u1 := dbtest.SeedUser(t, db)
	u2 := dbtest.SeedUser(t, db)
	u3 := dbtest.SeedUser(t, db)

	dbtest.SeedIssueAssignee(t, db, issueA, u1, true)
	dbtest.SeedIssueAssignee(t, db, issueA, u2, false)
	dbtest.SeedIssueAssignee(t, db, issueB, u3, false)
	// Soft-deleted ассайни не попадает в батч.
	dbtest.SeedIssueAssignee(t, db, issueB, u1, false)
	if _, err := db.Exec(`UPDATE tracker.issue_assignee SET deleted_at = now() WHERE issue_id = $1 AND user_id = $2`, issueB, u1); err != nil {
		t.Fatalf("soft delete: %v", err)
	}

	byIssue, err := repos.Issue().ListAssigneesByIssues(ctx, []string{issueA, issueB})
	if err != nil {
		t.Fatalf("ListAssigneesByIssues: %v", err)
	}
	if len(byIssue[issueA]) != 2 {
		t.Errorf("issueA assignees = %v, want 2", assigneeUserIDs(byIssue[issueA]))
	}
	if len(byIssue[issueB]) != 1 || byIssue[issueB][0].UserID != u3 {
		t.Errorf("issueB assignees = %v, want [%s]", assigneeUserIDs(byIssue[issueB]), u3)
	}

	// Пустой вход → пустая map, без запроса.
	empty, err := repos.Issue().ListAssigneesByIssues(ctx, nil)
	if err != nil {
		t.Fatalf("ListAssigneesByIssues empty: %v", err)
	}
	if len(empty) != 0 {
		t.Errorf("empty input = %v, want empty map", empty)
	}
}

func TestIssueAssigneeRepo_AddAssigneePreservesAndResetsPrimary(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	db := dbtest.Open(t)
	repos := repository.New(db)

	wsID := dbtest.SeedWorkspace(t, db)
	teamID := dbtest.SeedTeam(t, db, wsID)
	statusID := dbtest.SeedStatus(t, db, teamID, "Backlog", "backlog", 0)
	issueID, _ := dbtest.SeedIssue(t, db, teamID, statusID)
	userID := dbtest.SeedUser(t, db)

	if _, err := repos.Issue().AddAssignee(ctx, model.IssueAssignee{IssueID: issueID, UserID: userID}); err != nil {
		t.Fatalf("AddAssignee: %v", err)
	}
	if err := repos.Issue().SetPrimary(ctx, issueID, userID); err != nil {
		t.Fatalf("SetPrimary: %v", err)
	}

	// Повторное добавление активного участника (is_primary=false) НЕ снимает
	// ответственность — активную строку апсерт не демоутит.
	readded, err := repos.Issue().AddAssignee(ctx, model.IssueAssignee{IssueID: issueID, UserID: userID})
	if err != nil {
		t.Fatalf("AddAssignee re-add: %v", err)
	}
	if !readded.IsPrimary {
		t.Fatalf("re-add active assignee = %+v, want is_primary preserved (true)", readded)
	}

	// Снятие ответственного и повторное добавление: восстановленная soft-deleted
	// строка возвращается БЕЗ is_primary (устаревший флаг не воскрешается).
	if err := repos.Issue().RemoveAssignee(ctx, issueID, userID); err != nil {
		t.Fatalf("RemoveAssignee: %v", err)
	}
	restored, err := repos.Issue().AddAssignee(ctx, model.IssueAssignee{IssueID: issueID, UserID: userID})
	if err != nil {
		t.Fatalf("AddAssignee restore: %v", err)
	}
	if restored.IsPrimary || restored.DeletedAt != nil {
		t.Fatalf("restored = %+v, want active, non-primary (is_primary reset)", restored)
	}
}

func TestIssueAssigneeRepo_SinglePrimaryEnforcedByIndex(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	db := dbtest.Open(t)
	repos := repository.New(db)

	wsID := dbtest.SeedWorkspace(t, db)
	teamID := dbtest.SeedTeam(t, db, wsID)
	statusID := dbtest.SeedStatus(t, db, teamID, "Backlog", "backlog", 0)
	issueID, _ := dbtest.SeedIssue(t, db, teamID, statusID)
	u1 := dbtest.SeedUser(t, db)
	u2 := dbtest.SeedUser(t, db)

	if _, err := repos.Issue().AddAssignee(ctx, model.IssueAssignee{IssueID: issueID, UserID: u1}); err != nil {
		t.Fatalf("AddAssignee u1: %v", err)
	}
	if _, err := repos.Issue().AddAssignee(ctx, model.IssueAssignee{IssueID: issueID, UserID: u2}); err != nil {
		t.Fatalf("AddAssignee u2: %v", err)
	}

	if err := repos.Issue().SetPrimary(ctx, issueID, u1); err != nil {
		t.Fatalf("SetPrimary u1: %v", err)
	}

	// Второй primary без предварительного ClearPrimary нарушает partial unique
	// idx_tracker_issue_assignee_uniq_primary → AlreadyExists.
	err := repos.Issue().SetPrimary(ctx, issueID, u2)
	assertCode(t, err, errs.CodeAlreadyExists)

	// Корректная смена: ClearPrimary + SetPrimary (как делает сервис в транзакции).
	if err := repos.Issue().ClearPrimary(ctx, issueID); err != nil {
		t.Fatalf("ClearPrimary: %v", err)
	}
	if err := repos.Issue().SetPrimary(ctx, issueID, u2); err != nil {
		t.Fatalf("SetPrimary u2 after clear: %v", err)
	}

	active, err := repos.Issue().ListAssignees(ctx, issueID)
	if err != nil {
		t.Fatalf("ListAssignees: %v", err)
	}
	primaries := 0
	var primaryUser string
	for _, a := range active {
		if a.IsPrimary {
			primaries++
			primaryUser = a.UserID
		}
	}
	if primaries != 1 || primaryUser != u2 {
		t.Errorf("primaries = %d (user %s), want exactly 1 = %s", primaries, primaryUser, u2)
	}
}
