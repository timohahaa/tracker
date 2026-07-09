//go:build integration

package team_test

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

// cleanupTeams удаляет команды workspace (и их статусы/участников в FK-порядке),
// созданные через API-эндпоинты — они не покрыты seed-фабриками.
func cleanupTeams(t *testing.T, db *sqlx.DB, wsID string) {
	t.Helper()
	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM tracker.status WHERE team_id IN (SELECT id FROM org.team WHERE workspace_id = $1)`, wsID)
		_, _ = db.Exec(`DELETE FROM org.team_member WHERE team_id IN (SELECT id FROM org.team WHERE workspace_id = $1)`, wsID)
		_, _ = db.Exec(`DELETE FROM org.team WHERE workspace_id = $1`, wsID)
	})
}

// createTeam создаёт команду через API и возвращает её id.
func createTeam(t *testing.T, client *http.Client, url, name, key string, cookie *http.Cookie) string {
	t.Helper()
	resp := doJSON(t, client, http.MethodPost, url, map[string]any{"name": name, "key": key}, cookie)
	if resp.StatusCode != http.StatusCreated {
		_ = resp.Body.Close()
		t.Fatalf("create team: status = %d, want 201", resp.StatusCode)
	}
	var body struct {
		ID  string `json:"id"`
		Key string `json:"key"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode team: %v", err)
	}
	_ = resp.Body.Close()
	return body.ID
}

func TestCreateTeam(t *testing.T) {
	t.Parallel()
	srv, cookieName := setupServer(t)
	db := dbtest.Open(t)
	client := srv.Client()

	wsID := dbtest.SeedWorkspace(t, db)
	cleanupTeams(t, db, wsID)
	owner := seedMember(t, db, cookieName, wsID, model.WorkspaceRoleMember)

	teamsURL := srv.URL + "/api/workspaces/" + wsID + "/teams"

	// без cookie → 401
	resp := doJSON(t, client, http.MethodPost, teamsURL, map[string]any{"name": "X", "key": "X1"})
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("no cookie: status = %d, want 401", resp.StatusCode)
	}
	_ = resp.Body.Close()

	// невалидное тело (нет key) → 400
	resp = doJSON(t, client, http.MethodPost, teamsURL, map[string]any{"name": "No Key"}, owner.cookie)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("missing key: status = %d, want 400", resp.StatusCode)
	}
	_ = resp.Body.Close()

	// happy-path создание → 201
	key := "K" + uuid.NewString()[:6]
	teamID := createTeam(t, client, teamsURL, "Engineering", key, owner.cookie)
	if teamID == "" {
		t.Fatal("created team id is empty")
	}

	// дефолтные статусы созданы (по одному на каждую из 5 групп)
	resp = doJSON(t, client, http.MethodGet, teamsURL+"/"+teamID+"/statuses", nil, owner.cookie)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list statuses: status = %d, want 200", resp.StatusCode)
	}
	var statuses struct {
		Data []struct {
			Group string `json:"group"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&statuses); err != nil {
		t.Fatalf("decode statuses: %v", err)
	}
	_ = resp.Body.Close()
	groups := map[string]bool{}
	for _, s := range statuses.Data {
		groups[s.Group] = true
	}
	if len(statuses.Data) != 5 || len(groups) != 5 {
		t.Fatalf("statuses = %+v, want 5 distinct groups", statuses.Data)
	}

	// дубликат key → 409 (сервис нормализует key в верхний регистр)
	resp = doJSON(t, client, http.MethodPost, teamsURL, map[string]any{"name": "Dup", "key": key}, owner.cookie)
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("duplicate key: status = %d, want 409", resp.StatusCode)
	}
	_ = resp.Body.Close()
}

func TestTeamMutationsAuthorization(t *testing.T) {
	t.Parallel()
	srv, cookieName := setupServer(t)
	db := dbtest.Open(t)
	client := srv.Client()

	wsID := dbtest.SeedWorkspace(t, db)
	cleanupTeams(t, db, wsID)
	admin := seedMember(t, db, cookieName, wsID, model.WorkspaceRoleAdmin)
	creator := seedMember(t, db, cookieName, wsID, model.WorkspaceRoleMember)
	outsider := seedMember(t, db, cookieName, wsID, model.WorkspaceRoleMember)

	teamsURL := srv.URL + "/api/workspaces/" + wsID + "/teams"
	teamID := createTeam(t, client, teamsURL, "Engineering", "E"+uuid.NewString()[:6], creator.cookie)
	teamURL := teamsURL + "/" + teamID

	// участник команды (создатель) обновляет → 200
	resp := doJSON(t, client, http.MethodPatch, teamURL, map[string]any{"name": "Renamed"}, creator.cookie)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("member update: status = %d, want 200", resp.StatusCode)
	}
	_ = resp.Body.Close()

	// не-участник команды (обычный участник workspace) обновляет → 403
	resp = doJSON(t, client, http.MethodPatch, teamURL, map[string]any{"name": "Nope"}, outsider.cookie)
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("non-team-member update: status = %d, want 403", resp.StatusCode)
	}
	_ = resp.Body.Close()

	// несуществующая команда → 404
	resp = doJSON(t, client, http.MethodGet, teamsURL+"/"+uuid.NewString(), nil, creator.cookie)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("missing team: status = %d, want 404", resp.StatusCode)
	}
	_ = resp.Body.Close()

	// удаление обычным участником workspace → 403
	resp = doJSON(t, client, http.MethodDelete, teamURL, nil, creator.cookie)
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("member delete: status = %d, want 403", resp.StatusCode)
	}
	_ = resp.Body.Close()

	// удаление admin workspace → 204
	resp = doJSON(t, client, http.MethodDelete, teamURL, nil, admin.cookie)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("admin delete: status = %d, want 204", resp.StatusCode)
	}
	_ = resp.Body.Close()
}

// TestTeamMalformedID проверяет, что не-UUID teamID парсится на границе → 400
// (а не доходит до БД с 22P02 → 500). Валидный, но несуществующий id → 404.
func TestTeamMalformedID(t *testing.T) {
	t.Parallel()
	srv, cookieName := setupServer(t)
	db := dbtest.Open(t)
	client := srv.Client()

	wsID := dbtest.SeedWorkspace(t, db)
	cleanupTeams(t, db, wsID)
	member := seedMember(t, db, cookieName, wsID, model.WorkspaceRoleMember)

	teamsURL := srv.URL + "/api/workspaces/" + wsID + "/teams"

	// не-UUID teamID → 400
	resp := doJSON(t, client, http.MethodGet, teamsURL+"/garbage", nil, member.cookie)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("malformed teamID: status = %d, want 400", resp.StatusCode)
	}
	_ = resp.Body.Close()

	// не-UUID statusID во вложенном пути → 400
	resp = doJSON(t, client, http.MethodDelete, teamsURL+"/"+uuid.NewString()+"/statuses/garbage", nil, member.cookie)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("malformed statusID: status = %d, want 400", resp.StatusCode)
	}
	_ = resp.Body.Close()
}

func TestCreateStatusPositionConflict(t *testing.T) {
	t.Parallel()
	srv, cookieName := setupServer(t)
	db := dbtest.Open(t)
	client := srv.Client()

	wsID := dbtest.SeedWorkspace(t, db)
	cleanupTeams(t, db, wsID)
	creator := seedMember(t, db, cookieName, wsID, model.WorkspaceRoleMember)

	teamsURL := srv.URL + "/api/workspaces/" + wsID + "/teams"
	teamID := createTeam(t, client, teamsURL, "Engineering", "C"+uuid.NewString()[:6], creator.cookie)
	statusesURL := teamsURL + "/" + teamID + "/statuses"

	// первый статус на позиции 5 группы backlog → 201
	resp := doJSON(t, client, http.MethodPost, statusesURL, map[string]any{
		"name": "First", "group": "backlog", "position": 5,
	}, creator.cookie)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create first status: status = %d, want 201", resp.StatusCode)
	}
	_ = resp.Body.Close()

	// второй статус на той же (group, position) → 409
	resp = doJSON(t, client, http.MethodPost, statusesURL, map[string]any{
		"name": "Second", "group": "backlog", "position": 5,
	}, creator.cookie)
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("duplicate position: status = %d, want 409", resp.StatusCode)
	}
	_ = resp.Body.Close()
}

func TestDeleteLastStatusInGroup(t *testing.T) {
	t.Parallel()
	srv, cookieName := setupServer(t)
	db := dbtest.Open(t)
	client := srv.Client()

	wsID := dbtest.SeedWorkspace(t, db)
	cleanupTeams(t, db, wsID)
	creator := seedMember(t, db, cookieName, wsID, model.WorkspaceRoleMember)

	teamsURL := srv.URL + "/api/workspaces/" + wsID + "/teams"
	teamID := createTeam(t, client, teamsURL, "Engineering", "S"+uuid.NewString()[:6], creator.cookie)
	statusesURL := teamsURL + "/" + teamID + "/statuses"

	// найдём дефолтный статус группы backlog (единственный в группе)
	resp := doJSON(t, client, http.MethodGet, statusesURL, nil, creator.cookie)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list statuses: status = %d, want 200", resp.StatusCode)
	}
	var list struct {
		Data []struct {
			ID    string `json:"id"`
			Group string `json:"group"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		t.Fatalf("decode statuses: %v", err)
	}
	_ = resp.Body.Close()

	var backlogID string
	for _, s := range list.Data {
		if s.Group == "backlog" {
			backlogID = s.ID
		}
	}
	if backlogID == "" {
		t.Fatal("no backlog status found")
	}

	// удаление последнего статуса в группе → 400
	resp = doJSON(t, client, http.MethodDelete, statusesURL+"/"+backlogID, nil, creator.cookie)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("delete last status: status = %d, want 400", resp.StatusCode)
	}
	_ = resp.Body.Close()

	// добавим второй статус в группу backlog, тогда удаление первого разрешено → 204
	resp = doJSON(t, client, http.MethodPost, statusesURL, map[string]any{
		"name": "Triage", "group": "backlog", "position": 1,
	}, creator.cookie)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create second status: status = %d, want 201", resp.StatusCode)
	}
	_ = resp.Body.Close()

	resp = doJSON(t, client, http.MethodDelete, statusesURL+"/"+backlogID, nil, creator.cookie)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("delete status with sibling: status = %d, want 204", resp.StatusCode)
	}
	_ = resp.Body.Close()
}
