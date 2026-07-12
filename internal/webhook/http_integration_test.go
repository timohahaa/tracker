//go:build integration

package webhook_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/timohahaa/tracker/config"
	"github.com/timohahaa/tracker/internal/handler"
	"github.com/timohahaa/tracker/internal/repository"
	"github.com/timohahaa/tracker/internal/service"
	"github.com/timohahaa/tracker/test/dbtest"
)

// fakeGitLabClient фиксирует linkback-вызовы; реальных сетевых обращений нет.
type fakeGitLabClient struct {
	calls int
}

func (f *fakeGitLabClient) PostMRComment(_ context.Context, _, _ string, _, _ int, _ string) error {
	f.calls++
	return nil
}

// setup поднимает полный роутер поверх реальной БД и подменяет GitLab-клиент
// фейком (никаких сетевых вызовов в тестах).
func setup(t *testing.T) (*httptest.Server, *sqlx.DB, *fakeGitLabClient) {
	t.Helper()
	db := dbtest.Open(t)

	cfg := config.Config{SessionTTL: 24 * time.Hour}
	cfg.SetDefaults()
	repos := repository.New(db)
	services := service.New(repos, cfg)
	fake := &fakeGitLabClient{}
	services.GitLab.SetClient(fake)
	srv := httptest.NewServer(handler.New(services, cfg))
	t.Cleanup(srv.Close)
	return srv, db, fake
}

// seedTeamWithKey вставляет команду с заданным (uppercase) ключом. Нужен, чтобы
// identifier из webhook (после upper-нормализации) точно совпал с team.key.
func seedTeamWithKey(t *testing.T, db *sqlx.DB, workspaceID, key string) string {
	t.Helper()
	var id string
	if err := db.Get(&id, `
		INSERT INTO org.team (workspace_id, name, key)
		VALUES ($1, $2, $3) RETURNING id`, workspaceID, key, key); err != nil {
		t.Fatalf("seed team with key: %v", err)
	}
	t.Cleanup(func() { _, _ = db.Exec(`DELETE FROM org.team WHERE id = $1`, id) })
	return id
}

func uniqueKey(prefix string) string {
	return prefix + strings.ToUpper(uuid.NewString()[:6])
}

// mrPayload собирает merge_request webhook-событие.
func mrPayload(action, state, branch, title, url string) map[string]any {
	return map[string]any{
		"object_kind": "merge_request",
		"project":     map[string]any{"id": 42},
		"user":        map[string]any{"name": "Dev"},
		"object_attributes": map[string]any{
			"iid":           7,
			"title":         title,
			"description":   "",
			"source_branch": branch,
			"state":         state,
			"action":        action,
			"url":           url,
		},
	}
}

func postWebhook(t *testing.T, client *http.Client, url, token string, payload any) *http.Response {
	t.Helper()
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(payload); err != nil {
		t.Fatalf("encode payload: %v", err)
	}
	req, err := http.NewRequest(http.MethodPost, url, &buf)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("X-Gitlab-Token", token)
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	return resp
}

func TestGitLabWebhookHTTP(t *testing.T) {
	t.Parallel()
	srv, db, fake := setup(t)
	client := srv.Client()

	wsID := dbtest.SeedWorkspace(t, db)
	teamKey := uniqueKey("GL")
	teamID := seedTeamWithKey(t, db, wsID, teamKey)
	stBacklog := dbtest.SeedStatus(t, db, teamID, "Backlog", "backlog", 0)
	stInProgress := dbtest.SeedStatus(t, db, teamID, "In Progress", "in_progress", 0)
	_ = dbtest.SeedStatus(t, db, teamID, "Done", "completed", 0)
	issueID, issueNumber := dbtest.SeedIssue(t, db, teamID, stBacklog)
	t.Cleanup(func() { _, _ = db.Exec(`DELETE FROM tracker.attachment WHERE issue_id = $1`, issueID) })

	integrationID := dbtest.SeedIntegration(t, db, wsID, "gitlab",
		[]byte(`{"base_url":"https://gitlab.example.com"}`),
		[]byte(`{"access_token":"glpat-tok","webhook_secret":"whsecret"}`))

	webhookURL := srv.URL + "/webhooks/gitlab/" + integrationID
	identifier := teamKey + "-" + strconv.Itoa(issueNumber)
	// Ветка в нижнем регистре — проверка регистронезависимого матчинга.
	branch := "feature/" + strings.ToLower(identifier) + "-add-login"
	mrURL := "https://gitlab.example.com/g/p/-/merge_requests/7"

	// неверный X-Gitlab-Token → 401
	resp := postWebhook(t, client, webhookURL, "wrong", mrPayload("open", "opened", branch, "some title", mrURL))
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("bad token: status = %d, want 401", resp.StatusCode)
	}
	_ = resp.Body.Close()

	// несуществующий integrationID (валидный UUID) → 404
	resp = postWebhook(t, client, srv.URL+"/webhooks/gitlab/"+uuid.NewString(), "whsecret",
		mrPayload("open", "opened", branch, "some title", mrURL))
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("missing integration: status = %d, want 404", resp.StatusCode)
	}
	_ = resp.Body.Close()

	// валидный merge_request open → 200, attachment создан, статус → in_progress
	resp = postWebhook(t, client, webhookURL, "whsecret", mrPayload("open", "opened", branch, "some title", mrURL))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("valid open: status = %d, want 200", resp.StatusCode)
	}
	_ = resp.Body.Close()

	assertAttachmentCount(t, db, issueID, 1)
	assertIssueStatus(t, db, issueID, stInProgress)

	// повторная доставка (redelivery) → идемпотентно: attachment не дублируется
	resp = postWebhook(t, client, webhookURL, "whsecret", mrPayload("open", "opened", branch, "updated title", mrURL))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("redelivery: status = %d, want 200", resp.StatusCode)
	}
	_ = resp.Body.Close()
	assertAttachmentCount(t, db, issueID, 1)
	assertAttachmentTitle(t, db, issueID, "updated title")

	if fake.calls < 1 {
		t.Fatalf("linkback calls = %d, want >= 1", fake.calls)
	}
}

// TestGitLabWebhookHTTP_CrossWorkspaceIsolation проверяет, что webhook привязывает
// MR только к задачам workspace своей интеграции.
func TestGitLabWebhookHTTP_CrossWorkspaceIsolation(t *testing.T) {
	t.Parallel()
	srv, db, _ := setup(t)
	client := srv.Client()

	// Интеграция и её workspace.
	wsID := dbtest.SeedWorkspace(t, db)
	integrationID := dbtest.SeedIntegration(t, db, wsID, "gitlab",
		[]byte(`{"base_url":"https://gitlab.example.com"}`),
		[]byte(`{"access_token":"glpat-tok","webhook_secret":"whsecret"}`))

	// Чужой workspace с задачей, на которую MR НЕ должен привязаться.
	otherWsID := dbtest.SeedWorkspace(t, db)
	otherKey := uniqueKey("OT")
	otherTeamID := seedTeamWithKey(t, db, otherWsID, otherKey)
	otherStatus := dbtest.SeedStatus(t, db, otherTeamID, "Backlog", "backlog", 0)
	_ = dbtest.SeedStatus(t, db, otherTeamID, "In Progress", "in_progress", 0)
	otherIssueID, otherNumber := dbtest.SeedIssue(t, db, otherTeamID, otherStatus)
	t.Cleanup(func() { _, _ = db.Exec(`DELETE FROM tracker.attachment WHERE issue_id = $1`, otherIssueID) })

	identifier := otherKey + "-" + strconv.Itoa(otherNumber)
	branch := "feature/" + strings.ToLower(identifier)
	mrURL := "https://gitlab.example.com/g/p/-/merge_requests/9"

	// Webhook в интеграцию wsID ссылается на identifier чужого workspace.
	resp := postWebhook(t, client, srv.URL+"/webhooks/gitlab/"+integrationID, "whsecret",
		mrPayload("open", "opened", branch, "cross ws", mrURL))
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("cross-ws webhook: status = %d, want 200", resp.StatusCode)
	}
	_ = resp.Body.Close()

	// Задача чужого workspace не тронута: ни привязки, ни смены статуса.
	assertAttachmentCount(t, db, otherIssueID, 0)
	assertIssueStatus(t, db, otherIssueID, otherStatus)
}

func assertAttachmentCount(t *testing.T, db *sqlx.DB, issueID string, want int) {
	t.Helper()
	var n int
	if err := db.Get(&n, `
		SELECT count(*) FROM tracker.attachment
		WHERE issue_id = $1 AND deleted_at IS NULL`, issueID); err != nil {
		t.Fatalf("count attachments: %v", err)
	}
	if n != want {
		t.Fatalf("active attachments = %d, want %d", n, want)
	}
}

func assertAttachmentTitle(t *testing.T, db *sqlx.DB, issueID, want string) {
	t.Helper()
	var title string
	if err := db.Get(&title, `
		SELECT title FROM tracker.attachment
		WHERE issue_id = $1 AND deleted_at IS NULL`, issueID); err != nil {
		t.Fatalf("get attachment title: %v", err)
	}
	if title != want {
		t.Fatalf("attachment title = %q, want %q", title, want)
	}
}

func assertIssueStatus(t *testing.T, db *sqlx.DB, issueID, wantStatusID string) {
	t.Helper()
	var got string
	if err := db.Get(&got, `SELECT status_id FROM tracker.issue WHERE id = $1`, issueID); err != nil {
		t.Fatalf("get issue status: %v", err)
	}
	if got != wantStatusID {
		t.Fatalf("issue status = %s, want %s", got, wantStatusID)
	}
}
