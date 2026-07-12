//go:build integration

package integration_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/timohahaa/tracker/config"
	"github.com/timohahaa/tracker/internal/handler"
	"github.com/timohahaa/tracker/internal/model"
	"github.com/timohahaa/tracker/internal/repository"
	"github.com/timohahaa/tracker/internal/service"
	"github.com/timohahaa/tracker/test/dbtest"
)

func setupServer(t *testing.T) (*httptest.Server, string) {
	t.Helper()
	db := dbtest.Open(t)

	cfg := config.Config{SessionTTL: 24 * time.Hour}
	cfg.SetDefaults()
	repos := repository.New(db)
	services := service.New(repos, cfg)
	srv := httptest.NewServer(handler.New(services, cfg))
	t.Cleanup(srv.Close)
	return srv, cfg.SessionCookieName
}

func doJSON(t *testing.T, client *http.Client, method, url string, body any, cookies ...*http.Cookie) *http.Response {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatalf("encode body: %v", err)
		}
	}
	req, err := http.NewRequest(method, url, &buf)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	for _, c := range cookies {
		req.AddCookie(c)
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	return resp
}

type member struct {
	userID string
	cookie *http.Cookie
}

func seedMember(t *testing.T, db *sqlx.DB, cookieName, wsID string, role model.WorkspaceRole) member {
	t.Helper()
	userID := dbtest.SeedUser(t, db)
	dbtest.SeedWorkspaceMember(t, db, wsID, userID, string(role))
	sessionID := dbtest.SeedSession(t, db, userID)
	return member{userID: userID, cookie: &http.Cookie{Name: cookieName, Value: sessionID}}
}

func cleanupIntegrations(t *testing.T, db *sqlx.DB, wsID string) {
	t.Helper()
	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM integration.connection WHERE workspace_id = $1`, wsID)
	})
}

func TestGitLabIntegrationHTTP(t *testing.T) {
	t.Parallel()
	srv, cookieName := setupServer(t)
	db := dbtest.Open(t)
	client := srv.Client()

	wsID := dbtest.SeedWorkspace(t, db)
	cleanupIntegrations(t, db, wsID)
	admin := seedMember(t, db, cookieName, wsID, model.WorkspaceRoleAdmin)
	regular := seedMember(t, db, cookieName, wsID, model.WorkspaceRoleMember)

	gitlabURL := srv.URL + "/api/workspaces/" + wsID + "/integrations/gitlab"

	// без cookie → 401
	resp := doJSON(t, client, http.MethodGet, gitlabURL, nil)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("no cookie: status = %d, want 401", resp.StatusCode)
	}
	_ = resp.Body.Close()

	// GET участником до настройки → 200, connected=false
	resp = doJSON(t, client, http.MethodGet, gitlabURL, nil, regular.cookie)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get before setup: status = %d, want 200", resp.StatusCode)
	}
	var before struct {
		Connected bool `json:"connected"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&before)
	_ = resp.Body.Close()
	if before.Connected {
		t.Fatalf("connected = true before setup, want false")
	}

	// POST участником (member) → 403
	setupBody := map[string]any{
		"base_url":       "https://gitlab.example.com",
		"access_token":   "glpat-secret-token",
		"webhook_secret": "whsecret",
	}
	resp = doJSON(t, client, http.MethodPost, gitlabURL, setupBody, regular.cookie)
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("member setup: status = %d, want 403", resp.StatusCode)
	}
	_ = resp.Body.Close()

	// POST admin → 201, без утечки credentials
	resp = doJSON(t, client, http.MethodPost, gitlabURL, setupBody, admin.cookie)
	if resp.StatusCode != http.StatusCreated {
		_ = resp.Body.Close()
		t.Fatalf("admin setup: status = %d, want 201", resp.StatusCode)
	}
	raw := decodeRaw(t, resp)
	assertNoCredentials(t, raw)
	if settings, ok := raw["settings"].(map[string]any); !ok || settings["base_url"] != "https://gitlab.example.com" {
		t.Fatalf("setup response settings = %v, want base_url", raw["settings"])
	}

	// GET участником → 200, connected=true, credentials отсутствуют
	resp = doJSON(t, client, http.MethodGet, gitlabURL, nil, regular.cookie)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get after setup: status = %d, want 200", resp.StatusCode)
	}
	raw = decodeRaw(t, resp)
	if raw["connected"] != true {
		t.Fatalf("connected = %v, want true", raw["connected"])
	}
	assertNoCredentials(t, raw)

	// переподключение (POST ещё раз) → 201, идемпотентно (не дублирует)
	setupBody["base_url"] = "https://gitlab.reconnect.com"
	resp = doJSON(t, client, http.MethodPost, gitlabURL, setupBody, admin.cookie)
	if resp.StatusCode != http.StatusCreated {
		_ = resp.Body.Close()
		t.Fatalf("reconnect setup: status = %d, want 201", resp.StatusCode)
	}
	_ = resp.Body.Close()
	var activeCount int
	if err := db.Get(&activeCount, `
		SELECT count(*) FROM integration.connection
		WHERE workspace_id = $1 AND provider = 'gitlab' AND deleted_at IS NULL`, wsID); err != nil {
		t.Fatalf("count: %v", err)
	}
	if activeCount != 1 {
		t.Fatalf("active integrations = %d, want 1 (idempotent reconnect)", activeCount)
	}

	// DELETE участником (member) → 403
	resp = doJSON(t, client, http.MethodDelete, gitlabURL, nil, regular.cookie)
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("member delete: status = %d, want 403", resp.StatusCode)
	}
	_ = resp.Body.Close()

	// DELETE admin → 204
	resp = doJSON(t, client, http.MethodDelete, gitlabURL, nil, admin.cookie)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("admin delete: status = %d, want 204", resp.StatusCode)
	}
	_ = resp.Body.Close()

	// GET после удаления → connected=false
	resp = doJSON(t, client, http.MethodGet, gitlabURL, nil, regular.cookie)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get after delete: status = %d, want 200", resp.StatusCode)
	}
	raw = decodeRaw(t, resp)
	if raw["connected"] != false {
		t.Fatalf("connected = %v after delete, want false", raw["connected"])
	}
}

func decodeRaw(t *testing.T, resp *http.Response) map[string]any {
	t.Helper()
	var raw map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	_ = resp.Body.Close()
	return raw
}

// assertNoCredentials гарантирует, что секреты не протекли в тело ответа — ни на
// верхнем уровне, ни внутри settings.
func assertNoCredentials(t *testing.T, raw map[string]any) {
	t.Helper()
	for _, key := range []string{"credentials", "access_token", "webhook_secret"} {
		if _, ok := raw[key]; ok {
			t.Fatalf("response leaked %q: %v", key, raw)
		}
	}
	if settings, ok := raw["settings"].(map[string]any); ok {
		for _, key := range []string{"access_token", "webhook_secret"} {
			if _, ok := settings[key]; ok {
				t.Fatalf("settings leaked %q: %v", key, settings)
			}
		}
	}
}
