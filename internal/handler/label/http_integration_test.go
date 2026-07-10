//go:build integration

package label_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

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

// cleanupLabels удаляет метки workspace (и их junction-связи в FK-порядке),
// созданные через API-эндпоинты — они не покрыты seed-фабриками.
func cleanupLabels(t *testing.T, db *sqlx.DB, wsID string) {
	t.Helper()
	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM tracker.issue_label WHERE label_id IN (SELECT id FROM tracker.label WHERE workspace_id = $1)`, wsID)
		_, _ = db.Exec(`DELETE FROM tracker.label WHERE workspace_id = $1`, wsID)
	})
}

func TestLabelCRUD(t *testing.T) {
	t.Parallel()
	srv, cookieName := setupServer(t)
	db := dbtest.Open(t)
	client := srv.Client()

	wsID := dbtest.SeedWorkspace(t, db)
	cleanupLabels(t, db, wsID)
	m := seedMember(t, db, cookieName, wsID, model.WorkspaceRoleMember)
	teamID := dbtest.SeedTeam(t, db, wsID)

	labelsURL := srv.URL + "/api/workspaces/" + wsID + "/labels"

	// без cookie → 401
	resp := doJSON(t, client, http.MethodPost, labelsURL, map[string]any{"name": "bug", "color": "#3B82F6"})
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("no cookie: status = %d, want 401", resp.StatusCode)
	}
	_ = resp.Body.Close()

	// невалидное тело (нет name) → 400
	resp = doJSON(t, client, http.MethodPost, labelsURL, map[string]any{"color": "#3B82F6"}, m.cookie)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("missing name: status = %d, want 400", resp.StatusCode)
	}
	_ = resp.Body.Close()

	// кривой цвет → 400
	resp = doJSON(t, client, http.MethodPost, labelsURL, map[string]any{"name": "bug", "color": "red"}, m.cookie)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("bad color: status = %d, want 400", resp.StatusCode)
	}
	_ = resp.Body.Close()

	// happy-path: workspace-level метка → 201
	resp = doJSON(t, client, http.MethodPost, labelsURL, map[string]any{"name": "bug", "color": "#3B82F6"}, m.cookie)
	if resp.StatusCode != http.StatusCreated {
		_ = resp.Body.Close()
		t.Fatalf("create ws-level: status = %d, want 201", resp.StatusCode)
	}
	var wsLabel struct {
		ID     string  `json:"id"`
		TeamID *string `json:"team_id"`
		Name   string  `json:"name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&wsLabel); err != nil {
		t.Fatalf("decode ws-level: %v", err)
	}
	_ = resp.Body.Close()
	if wsLabel.ID == "" || wsLabel.TeamID != nil {
		t.Fatalf("ws-level = %+v, want id set / team_id nil", wsLabel)
	}

	// happy-path: team-level метка → 201
	resp = doJSON(t, client, http.MethodPost, labelsURL, map[string]any{"name": "feature", "color": "#00FF00", "team_id": teamID}, m.cookie)
	if resp.StatusCode != http.StatusCreated {
		_ = resp.Body.Close()
		t.Fatalf("create team-level: status = %d, want 201", resp.StatusCode)
	}
	var teamLabel struct {
		ID     string  `json:"id"`
		TeamID *string `json:"team_id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&teamLabel); err != nil {
		t.Fatalf("decode team-level: %v", err)
	}
	_ = resp.Body.Close()
	if teamLabel.TeamID == nil || *teamLabel.TeamID != teamID {
		t.Fatalf("team-level team_id = %v, want %s", teamLabel.TeamID, teamID)
	}

	// list → 200, обе метки
	resp = doJSON(t, client, http.MethodGet, labelsURL, nil, m.cookie)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list: status = %d, want 200", resp.StatusCode)
	}
	var list struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	_ = resp.Body.Close()
	if len(list.Data) != 2 {
		t.Fatalf("list len = %d, want 2", len(list.Data))
	}

	// дубль ws-level имени → 409
	resp = doJSON(t, client, http.MethodPost, labelsURL, map[string]any{"name": "bug", "color": "#123456"}, m.cookie)
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("duplicate ws-level: status = %d, want 409", resp.StatusCode)
	}
	_ = resp.Body.Close()

	// team-метка поверх существующей ws-метки "bug" → 409
	resp = doJSON(t, client, http.MethodPost, labelsURL, map[string]any{"name": "bug", "color": "#123456", "team_id": teamID}, m.cookie)
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("team over ws: status = %d, want 409", resp.StatusCode)
	}
	_ = resp.Body.Close()

	// get одиночной метки → 200
	labelURL := labelsURL + "/" + wsLabel.ID
	resp = doJSON(t, client, http.MethodGet, labelURL, nil, m.cookie)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get label: status = %d, want 200", resp.StatusCode)
	}
	var fetched struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&fetched); err != nil {
		t.Fatalf("decode fetched: %v", err)
	}
	_ = resp.Body.Close()
	if fetched.ID != wsLabel.ID || fetched.Name != "bug" {
		t.Fatalf("fetched = %+v, want id %s / bug", fetched, wsLabel.ID)
	}

	// get несуществующей метки → 404
	resp = doJSON(t, client, http.MethodGet, labelsURL+"/"+uuid.NewString(), nil, m.cookie)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("get missing label: status = %d, want 404", resp.StatusCode)
	}
	_ = resp.Body.Close()

	// patch: сменить name/color → 200
	resp = doJSON(t, client, http.MethodPatch, labelURL, map[string]any{"name": "critical", "color": "#FF0000"}, m.cookie)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("patch: status = %d, want 200", resp.StatusCode)
	}
	var patched struct {
		Name  string `json:"name"`
		Color string `json:"color"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&patched); err != nil {
		t.Fatalf("decode patched: %v", err)
	}
	_ = resp.Body.Close()
	if patched.Name != "critical" || patched.Color != "#FF0000" {
		t.Fatalf("patched = %+v, want critical / #FF0000", patched)
	}

	// delete → 204
	resp = doJSON(t, client, http.MethodDelete, labelURL, nil, m.cookie)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("delete: status = %d, want 204", resp.StatusCode)
	}
	_ = resp.Body.Close()

	// несуществующая метка → 404
	resp = doJSON(t, client, http.MethodPatch, labelsURL+"/"+uuid.NewString(), map[string]any{"name": "x"}, m.cookie)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("missing label: status = %d, want 404", resp.StatusCode)
	}
	_ = resp.Body.Close()
}

// TestLabelMerge проверяет merge: создание ws-level метки поверх одноимённой
// team-level метки перепривязывает issue_label и soft-удаляет team-метку.
func TestLabelMerge(t *testing.T) {
	t.Parallel()
	srv, cookieName := setupServer(t)
	db := dbtest.Open(t)
	client := srv.Client()

	wsID := dbtest.SeedWorkspace(t, db)
	cleanupLabels(t, db, wsID)
	m := seedMember(t, db, cookieName, wsID, model.WorkspaceRoleMember)
	teamID := dbtest.SeedTeam(t, db, wsID)
	statusID := dbtest.SeedStatus(t, db, teamID, "Todo", "planned", 0)
	issueID, _ := dbtest.SeedIssue(t, db, teamID, statusID)

	// team-level метка "urgent" висит на задаче (именованная — merge матчит по имени).
	var teamLabel string
	if err := db.Get(&teamLabel, `
		INSERT INTO tracker.label (workspace_id, team_id, name, color)
		VALUES ($1, $2, 'urgent', '#00FF00') RETURNING id`, wsID, teamID); err != nil {
		t.Fatalf("seed named team label: %v", err)
	}
	dbtest.SeedIssueLabel(t, db, issueID, teamLabel)

	labelsURL := srv.URL + "/api/workspaces/" + wsID + "/labels"

	// создать ws-level метку с тем же именем → merge → 201
	resp := doJSON(t, client, http.MethodPost, labelsURL, map[string]any{"name": "urgent", "color": "#3B82F6"}, m.cookie)
	if resp.StatusCode != http.StatusCreated {
		_ = resp.Body.Close()
		t.Fatalf("merge create: status = %d, want 201", resp.StatusCode)
	}
	var wsLabel struct {
		ID     string  `json:"id"`
		TeamID *string `json:"team_id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&wsLabel); err != nil {
		t.Fatalf("decode merge: %v", err)
	}
	_ = resp.Body.Close()
	if wsLabel.TeamID != nil {
		t.Fatalf("merged label team_id = %v, want nil (ws-level)", wsLabel.TeamID)
	}

	// старая team-метка soft-deleted.
	var teamActive int
	if err := db.Get(&teamActive, `SELECT count(*) FROM tracker.label WHERE id = $1 AND deleted_at IS NULL`, teamLabel); err != nil {
		t.Fatalf("count team label: %v", err)
	}
	if teamActive != 0 {
		t.Errorf("team label active = %d, want 0 (soft-deleted)", teamActive)
	}

	// issue_label перепривязан на новую ws-метку.
	var repointed int
	if err := db.Get(&repointed, `
		SELECT count(*) FROM tracker.issue_label
		WHERE issue_id = $1 AND label_id = $2 AND deleted_at IS NULL`, issueID, wsLabel.ID); err != nil {
		t.Fatalf("count repointed: %v", err)
	}
	if repointed != 1 {
		t.Errorf("repointed links = %d, want 1", repointed)
	}
}

// TestLabelUpdateMerge проверяет симметрию merge на rename: переименование
// ws-метки в имя существующих team-меток сливает их в эту ws-метку
// (перепривязка issue_label + soft-delete team-меток).
func TestLabelUpdateMerge(t *testing.T) {
	t.Parallel()
	srv, cookieName := setupServer(t)
	db := dbtest.Open(t)
	client := srv.Client()

	wsID := dbtest.SeedWorkspace(t, db)
	cleanupLabels(t, db, wsID)
	m := seedMember(t, db, cookieName, wsID, model.WorkspaceRoleMember)
	teamID := dbtest.SeedTeam(t, db, wsID)
	statusID := dbtest.SeedStatus(t, db, teamID, "Todo", "planned", 0)
	issueID, _ := dbtest.SeedIssue(t, db, teamID, statusID)

	labelsURL := srv.URL + "/api/workspaces/" + wsID + "/labels"

	// ws-level метка "A".
	resp := doJSON(t, client, http.MethodPost, labelsURL, map[string]any{"name": "A", "color": "#3B82F6"}, m.cookie)
	if resp.StatusCode != http.StatusCreated {
		_ = resp.Body.Close()
		t.Fatalf("create ws-level A: status = %d, want 201", resp.StatusCode)
	}
	var wsLabel struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&wsLabel); err != nil {
		t.Fatalf("decode A: %v", err)
	}
	_ = resp.Body.Close()

	// team-level метка "B" висит на задаче.
	var teamLabel string
	if err := db.Get(&teamLabel, `
		INSERT INTO tracker.label (workspace_id, team_id, name, color)
		VALUES ($1, $2, 'B', '#00FF00') RETURNING id`, wsID, teamID); err != nil {
		t.Fatalf("seed team label B: %v", err)
	}
	dbtest.SeedIssueLabel(t, db, issueID, teamLabel)

	// PATCH: переименовать ws-метку A → B → merge → 200
	resp = doJSON(t, client, http.MethodPatch, labelsURL+"/"+wsLabel.ID, map[string]any{"name": "B"}, m.cookie)
	if resp.StatusCode != http.StatusOK {
		_ = resp.Body.Close()
		t.Fatalf("rename merge: status = %d, want 200", resp.StatusCode)
	}
	_ = resp.Body.Close()

	// team-метка "B" soft-deleted.
	var teamActive int
	if err := db.Get(&teamActive, `SELECT count(*) FROM tracker.label WHERE id = $1 AND deleted_at IS NULL`, teamLabel); err != nil {
		t.Fatalf("count team label: %v", err)
	}
	if teamActive != 0 {
		t.Errorf("team label active = %d, want 0 (soft-deleted)", teamActive)
	}

	// issue_label перепривязан на переименованную ws-метку.
	var repointed int
	if err := db.Get(&repointed, `
		SELECT count(*) FROM tracker.issue_label
		WHERE issue_id = $1 AND label_id = $2 AND deleted_at IS NULL`, issueID, wsLabel.ID); err != nil {
		t.Fatalf("count repointed: %v", err)
	}
	if repointed != 1 {
		t.Errorf("repointed links = %d, want 1", repointed)
	}
}

// TestLabelMalformedID проверяет, что не-UUID в path/body парсится на границе и
// отдаётся как 400, а не доходит до БД (иначе 22P02 → 500).
func TestLabelMalformedID(t *testing.T) {
	t.Parallel()
	srv, cookieName := setupServer(t)
	db := dbtest.Open(t)
	client := srv.Client()

	wsID := dbtest.SeedWorkspace(t, db)
	cleanupLabels(t, db, wsID)
	m := seedMember(t, db, cookieName, wsID, model.WorkspaceRoleMember)

	labelsURL := srv.URL + "/api/workspaces/" + wsID + "/labels"

	// не-UUID labelID → 400
	badURL := labelsURL + "/garbage"
	for _, tc := range []struct {
		method string
		body   any
	}{
		{http.MethodGet, nil},
		{http.MethodPatch, map[string]any{"name": "x"}},
		{http.MethodDelete, nil},
	} {
		resp := doJSON(t, client, tc.method, badURL, tc.body, m.cookie)
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("%s malformed labelID: status = %d, want 400", tc.method, resp.StatusCode)
		}
		_ = resp.Body.Close()
	}

	// не-UUID team_id в теле → 400
	resp := doJSON(t, client, http.MethodPost, labelsURL, map[string]any{"name": "bug", "color": "#3B82F6", "team_id": "garbage"}, m.cookie)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("malformed team_id: status = %d, want 400", resp.StatusCode)
	}
	_ = resp.Body.Close()
}
