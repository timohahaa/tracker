//go:build integration

package project_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/google/uuid"

	"github.com/timohahaa/tracker/config"
	"github.com/timohahaa/tracker/internal/handler"
	"github.com/timohahaa/tracker/internal/model"
	"github.com/timohahaa/tracker/internal/repository"
	"github.com/timohahaa/tracker/internal/service"
	"github.com/timohahaa/tracker/test/dbtest"
)

// setupServer собирает полный роутер поверх реальной БД и возвращает сервер и
// имя session-cookie из конфигурации.
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

// member описывает засиженного участника workspace + его session-cookie.
type member struct {
	userID string
	cookie *http.Cookie
}

// seedMember создаёт пользователя, его membership с указанной ролью и активную
// сессию; возвращает данные участника с готовым cookie.
func seedMember(t *testing.T, db *sqlx.DB, cookieName, wsID string, role model.WorkspaceRole) member {
	t.Helper()
	userID := dbtest.SeedUser(t, db)
	dbtest.SeedWorkspaceMember(t, db, wsID, userID, string(role))
	sessionID := dbtest.SeedSession(t, db, userID)
	return member{userID: userID, cookie: &http.Cookie{Name: cookieName, Value: sessionID}}
}

// cleanupProjects удаляет проекты workspace (и их junction-связи в FK-порядке),
// созданные через API-эндпоинты — они не покрыты seed-фабриками.
func cleanupProjects(t *testing.T, db *sqlx.DB, wsID string) {
	t.Helper()
	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM tracker.project_team WHERE project_id IN (SELECT id FROM tracker.project WHERE workspace_id = $1)`, wsID)
		_, _ = db.Exec(`DELETE FROM tracker.project WHERE workspace_id = $1`, wsID)
	})
}

func TestProjectCRUD(t *testing.T) {
	t.Parallel()
	srv, cookieName := setupServer(t)
	db := dbtest.Open(t)
	client := srv.Client()

	wsID := dbtest.SeedWorkspace(t, db)
	cleanupProjects(t, db, wsID)
	owner := seedMember(t, db, cookieName, wsID, model.WorkspaceRoleMember)
	teamID := dbtest.SeedTeam(t, db, wsID)

	projectsURL := srv.URL + "/api/workspaces/" + wsID + "/projects"

	// без cookie → 401
	resp := doJSON(t, client, http.MethodPost, projectsURL, map[string]any{"name": "X"})
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("no cookie: status = %d, want 401", resp.StatusCode)
	}
	_ = resp.Body.Close()

	// невалидное тело (нет name) → 400
	resp = doJSON(t, client, http.MethodPost, projectsURL, map[string]any{"description": "no name"}, owner.cookie)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("missing name: status = %d, want 400", resp.StatusCode)
	}
	_ = resp.Body.Close()

	// happy-path создание с привязкой команды → 201
	resp = doJSON(t, client, http.MethodPost, projectsURL, map[string]any{
		"name":     "Roadmap",
		"team_ids": []string{teamID},
	}, owner.cookie)
	if resp.StatusCode != http.StatusCreated {
		_ = resp.Body.Close()
		t.Fatalf("create project: status = %d, want 201", resp.StatusCode)
	}
	var created struct {
		ID      string   `json:"id"`
		Status  string   `json:"status"`
		TeamIDs []string `json:"team_ids"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		t.Fatalf("decode project: %v", err)
	}
	_ = resp.Body.Close()
	if created.ID == "" || created.Status != "planned" {
		t.Fatalf("created = %+v, want id set / planned", created)
	}
	if len(created.TeamIDs) != 1 || created.TeamIDs[0] != teamID {
		t.Fatalf("created team_ids = %v, want [%s]", created.TeamIDs, teamID)
	}
	projectURL := projectsURL + "/" + created.ID

	// get → 200, команды присутствуют
	resp = doJSON(t, client, http.MethodGet, projectURL, nil, owner.cookie)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get project: status = %d, want 200", resp.StatusCode)
	}
	_ = resp.Body.Close()

	// patch: сменить статус и отвязать все команды ([]) → 200
	resp = doJSON(t, client, http.MethodPatch, projectURL, map[string]any{
		"status":   "in_progress",
		"team_ids": []string{},
	}, owner.cookie)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("patch project: status = %d, want 200", resp.StatusCode)
	}
	var patched struct {
		Status  string   `json:"status"`
		TeamIDs []string `json:"team_ids"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&patched); err != nil {
		t.Fatalf("decode patched: %v", err)
	}
	_ = resp.Body.Close()
	if patched.Status != "in_progress" || len(patched.TeamIDs) != 0 {
		t.Fatalf("patched = %+v, want in_progress / no teams", patched)
	}

	// несуществующий проект → 404
	resp = doJSON(t, client, http.MethodGet, projectsURL+"/"+uuid.NewString(), nil, owner.cookie)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("missing project: status = %d, want 404", resp.StatusCode)
	}
	_ = resp.Body.Close()
}

// TestProjectMalformedID проверяет, что не-UUID в path парсится на границе и
// отдаётся как 400, а не доходит до БД (иначе 22P02 → 500).
func TestProjectMalformedID(t *testing.T) {
	t.Parallel()
	srv, cookieName := setupServer(t)
	db := dbtest.Open(t)
	client := srv.Client()

	wsID := dbtest.SeedWorkspace(t, db)
	cleanupProjects(t, db, wsID)
	owner := seedMember(t, db, cookieName, wsID, model.WorkspaceRoleMember)

	badURL := srv.URL + "/api/workspaces/" + wsID + "/projects/garbage"

	cases := []struct {
		method string
		body   any
	}{
		{http.MethodGet, nil},
		{http.MethodPatch, map[string]any{"status": "in_progress"}},
		{http.MethodDelete, nil},
	}
	for _, tc := range cases {
		resp := doJSON(t, client, tc.method, badURL, tc.body, owner.cookie)
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("%s malformed projectID: status = %d, want 400", tc.method, resp.StatusCode)
		}
		_ = resp.Body.Close()
	}

	// не-UUID workspaceID тоже отсекается на границе (middleware) → 400
	resp := doJSON(t, client, http.MethodGet, srv.URL+"/api/workspaces/garbage/projects", nil, owner.cookie)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("malformed workspaceID: status = %d, want 400", resp.StatusCode)
	}
	_ = resp.Body.Close()
}

func TestProjectDeleteAuthorization(t *testing.T) {
	t.Parallel()
	srv, cookieName := setupServer(t)
	db := dbtest.Open(t)
	client := srv.Client()

	wsID := dbtest.SeedWorkspace(t, db)
	cleanupProjects(t, db, wsID)
	member := seedMember(t, db, cookieName, wsID, model.WorkspaceRoleMember)
	admin := seedMember(t, db, cookieName, wsID, model.WorkspaceRoleAdmin)

	projectsURL := srv.URL + "/api/workspaces/" + wsID + "/projects"

	// создать проект (участник workspace) → 201
	resp := doJSON(t, client, http.MethodPost, projectsURL, map[string]any{"name": "Roadmap"}, member.cookie)
	if resp.StatusCode != http.StatusCreated {
		_ = resp.Body.Close()
		t.Fatalf("create project: status = %d, want 201", resp.StatusCode)
	}
	var created struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		t.Fatalf("decode project: %v", err)
	}
	_ = resp.Body.Close()
	projectURL := projectsURL + "/" + created.ID

	// удаление обычным участником workspace → 403
	resp = doJSON(t, client, http.MethodDelete, projectURL, nil, member.cookie)
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("member delete: status = %d, want 403", resp.StatusCode)
	}
	_ = resp.Body.Close()

	// удаление admin workspace → 204
	resp = doJSON(t, client, http.MethodDelete, projectURL, nil, admin.cookie)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("admin delete: status = %d, want 204", resp.StatusCode)
	}
	_ = resp.Body.Close()
}
