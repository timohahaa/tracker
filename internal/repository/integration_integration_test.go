//go:build integration

package repository_test

import (
	"context"
	"strings"
	"testing"

	"github.com/jmoiron/sqlx"

	"github.com/timohahaa/tracker/internal/errs"
	"github.com/timohahaa/tracker/internal/model"
	"github.com/timohahaa/tracker/internal/repository"
	"github.com/timohahaa/tracker/test/dbtest"
)

func TestIntegrationRepo_UpsertIdempotentAndRestore(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	db := dbtest.Open(t)
	repos := repository.New(db)

	wsID := dbtest.SeedWorkspace(t, db)
	t.Cleanup(func() { _, _ = db.Exec(`DELETE FROM integration.connection WHERE workspace_id = $1`, wsID) })

	// Первичное подключение.
	created, err := repos.Integration().Upsert(ctx, model.Integration{
		WorkspaceID: wsID,
		Provider:    model.ProviderGitLab,
		Settings:    []byte(`{"base_url":"https://gitlab.example.com"}`),
		Credentials: []byte(`{"access_token":"tok1","webhook_secret":"sec1"}`),
	})
	if err != nil {
		t.Fatalf("Upsert create: %v", err)
	}
	if created.ID == "" {
		t.Fatalf("Upsert create id empty")
	}

	// Повторный upsert (активна) — тот же id, обновлённые settings/credentials,
	// без дубля активной строки.
	updated, err := repos.Integration().Upsert(ctx, model.Integration{
		WorkspaceID: wsID,
		Provider:    model.ProviderGitLab,
		Settings:    []byte(`{"base_url":"https://gitlab.new.com"}`),
		Credentials: []byte(`{"access_token":"tok2","webhook_secret":"sec2"}`),
	})
	if err != nil {
		t.Fatalf("Upsert update: %v", err)
	}
	if updated.ID != created.ID {
		t.Fatalf("Upsert update id = %s, want same %s (idempotent)", updated.ID, created.ID)
	}
	assertActiveIntegrations(t, db, wsID, 1)

	// GetByWorkspaceProvider / GetByID возвращают активную с обновлёнными данными.
	got, err := repos.Integration().GetByWorkspaceProvider(ctx, wsID, model.ProviderGitLab)
	if err != nil {
		t.Fatalf("GetByWorkspaceProvider: %v", err)
	}
	if !strings.Contains(string(got.Settings), "gitlab.new.com") {
		t.Errorf("settings = %s, want updated base_url", got.Settings)
	}
	byID, err := repos.Integration().GetByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if byID.ID != created.ID {
		t.Errorf("GetByID id = %s, want %s", byID.ID, created.ID)
	}

	// Soft delete → чтение больше не находит.
	if err := repos.Integration().Delete(ctx, created.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	_, err = repos.Integration().GetByWorkspaceProvider(ctx, wsID, model.ProviderGitLab)
	assertCode(t, err, errs.CodeNotFound)
	_, err = repos.Integration().GetByID(ctx, created.ID)
	assertCode(t, err, errs.CodeNotFound)
	assertActiveIntegrations(t, db, wsID, 0)

	// Переподключение после удаления: снова ровно одна активная интеграция
	// (без дубля активных строк по (workspace_id, provider)).
	reconnected, err := repos.Integration().Upsert(ctx, model.Integration{
		WorkspaceID: wsID,
		Provider:    model.ProviderGitLab,
		Settings:    []byte(`{"base_url":"https://gitlab.reconnect.com"}`),
		Credentials: []byte(`{"access_token":"tok3","webhook_secret":"sec3"}`),
	})
	if err != nil {
		t.Fatalf("Upsert reconnect: %v", err)
	}
	if reconnected.DeletedAt != nil {
		t.Errorf("reconnected deleted_at = %v, want nil (active)", reconnected.DeletedAt)
	}
	assertActiveIntegrations(t, db, wsID, 1)
}

func TestAttachmentRepo_UpsertIdempotent(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	db := dbtest.Open(t)
	repos := repository.New(db)

	wsID := dbtest.SeedWorkspace(t, db)
	teamID := dbtest.SeedTeam(t, db, wsID)
	statusID := dbtest.SeedStatus(t, db, teamID, "Backlog", "backlog", 0)
	issueID, _ := dbtest.SeedIssue(t, db, teamID, statusID)
	integrationID := dbtest.SeedIntegration(t, db, wsID, "gitlab",
		[]byte(`{"base_url":"https://gitlab.example.com"}`),
		[]byte(`{"access_token":"tok","webhook_secret":"sec"}`))
	t.Cleanup(func() { _, _ = db.Exec(`DELETE FROM tracker.attachment WHERE issue_id = $1`, issueID) })

	const url = "https://gitlab.example.com/g/p/-/merge_requests/1"

	first, err := repos.Attachment().Upsert(ctx, model.Attachment{
		IssueID:       issueID,
		IntegrationID: &integrationID,
		SourceType:    model.SourceTypeGitLabMR,
		URL:           url,
		Title:         "MR title v1",
		Metadata:      []byte(`{"state":"opened"}`),
	})
	if err != nil {
		t.Fatalf("Upsert first: %v", err)
	}

	// Повторная доставка того же MR (тот же url+issue) — тот же id, обновлённые
	// title/metadata, без дубля.
	second, err := repos.Attachment().Upsert(ctx, model.Attachment{
		IssueID:       issueID,
		IntegrationID: &integrationID,
		SourceType:    model.SourceTypeGitLabMR,
		URL:           url,
		Title:         "MR title v2",
		Metadata:      []byte(`{"state":"merged"}`),
	})
	if err != nil {
		t.Fatalf("Upsert second: %v", err)
	}
	if second.ID != first.ID {
		t.Fatalf("Upsert second id = %s, want same %s (idempotent)", second.ID, first.ID)
	}
	if second.Title != "MR title v2" {
		t.Errorf("title = %s, want updated", second.Title)
	}

	list, err := repos.Attachment().ListByIssue(ctx, issueID)
	if err != nil {
		t.Fatalf("ListByIssue: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("ListByIssue len = %d, want 1 (no dup)", len(list))
	}
	if list[0].IntegrationID == nil || *list[0].IntegrationID != integrationID {
		t.Errorf("attachment integration_id = %v, want %s", list[0].IntegrationID, integrationID)
	}
}

// assertActiveIntegrations проверяет число активных интеграций (ws, gitlab).
func assertActiveIntegrations(t *testing.T, db *sqlx.DB, wsID string, want int) {
	t.Helper()
	var n int
	if err := db.Get(&n, `
		SELECT count(*) FROM integration.connection
		WHERE workspace_id = $1 AND provider = 'gitlab' AND deleted_at IS NULL`, wsID); err != nil {
		t.Fatalf("count integrations: %v", err)
	}
	if n != want {
		t.Fatalf("active integrations = %d, want %d", n, want)
	}
}
