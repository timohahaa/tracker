//go:build integration

package issue_test

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

// createTeam создаёт команду через team API (нормализует key в верхний регистр,
// создаёт дефолтные статусы и делает создателя участником-owner) и возвращает
// её id и нормализованный key. Регистрирует cleanup задач/статусов/участников.
func createTeam(t *testing.T, client *http.Client, db *sqlx.DB, teamsURL string, cookie *http.Cookie) (string, string) {
	t.Helper()
	key := "K" + uuid.NewString()[:6]
	resp := doJSON(t, client, http.MethodPost, teamsURL, map[string]any{"name": "Engineering", "key": key}, cookie)
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
	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM tracker.issue WHERE team_id = $1`, body.ID)
		_, _ = db.Exec(`DELETE FROM tracker.status WHERE team_id = $1`, body.ID)
		_, _ = db.Exec(`DELETE FROM org.team_member WHERE team_id = $1`, body.ID)
		_, _ = db.Exec(`DELETE FROM org.team WHERE id = $1`, body.ID)
	})
	return body.ID, body.Key
}

func TestCreateAndGetIssue(t *testing.T) {
	t.Parallel()
	srv, cookieName := setupServer(t)
	db := dbtest.Open(t)
	client := srv.Client()

	wsID := dbtest.SeedWorkspace(t, db)
	creator := seedMember(t, db, cookieName, wsID, model.WorkspaceRoleMember)

	teamsURL := srv.URL + "/api/workspaces/" + wsID + "/teams"
	teamID, key := createTeam(t, client, db, teamsURL, creator.cookie)

	issuesURL := srv.URL + "/api/workspaces/" + wsID + "/issues"

	// без cookie → 401
	resp := doJSON(t, client, http.MethodPost, issuesURL, map[string]any{"team_id": teamID, "title": "X"})
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("no cookie: status = %d, want 401", resp.StatusCode)
	}
	_ = resp.Body.Close()

	// невалидное тело (нет title) → 400
	resp = doJSON(t, client, http.MethodPost, issuesURL, map[string]any{"team_id": teamID}, creator.cookie)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("missing title: status = %d, want 400", resp.StatusCode)
	}
	_ = resp.Body.Close()

	// happy-path создание → 201, identifier = KEY-1
	resp = doJSON(t, client, http.MethodPost, issuesURL, map[string]any{"team_id": teamID, "title": "First"}, creator.cookie)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create: status = %d, want 201", resp.StatusCode)
	}
	var created struct {
		ID         string `json:"id"`
		Identifier string `json:"identifier"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		t.Fatalf("decode created: %v", err)
	}
	_ = resp.Body.Close()
	if created.ID == "" || created.Identifier != key+"-1" {
		t.Fatalf("created = %+v, want id set + identifier %q", created, key+"-1")
	}

	// get по UUID → 200
	resp = doJSON(t, client, http.MethodGet, issuesURL+"/"+created.ID, nil, creator.cookie)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get by uuid: status = %d, want 200", resp.StatusCode)
	}
	var gotByID struct {
		Identifier string `json:"identifier"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&gotByID); err != nil {
		t.Fatalf("decode get by uuid: %v", err)
	}
	_ = resp.Body.Close()
	if gotByID.Identifier != key+"-1" {
		t.Errorf("get by uuid identifier = %q, want %q", gotByID.Identifier, key+"-1")
	}

	// get по identifier → 200
	resp = doJSON(t, client, http.MethodGet, issuesURL+"/"+created.Identifier, nil, creator.cookie)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get by identifier: status = %d, want 200", resp.StatusCode)
	}
	var gotByIdent struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&gotByIdent); err != nil {
		t.Fatalf("decode get by identifier: %v", err)
	}
	_ = resp.Body.Close()
	if gotByIdent.ID != created.ID {
		t.Errorf("get by identifier id = %q, want %q", gotByIdent.ID, created.ID)
	}

	// несуществующая задача → 404
	resp = doJSON(t, client, http.MethodGet, issuesURL+"/"+uuid.NewString(), nil, creator.cookie)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("missing issue: status = %d, want 404", resp.StatusCode)
	}
	_ = resp.Body.Close()
}

// TestIssueMalformedIDs проверяет, что мусорные UUID в теле и в path отсекаются
// на границе (400). Пустая строка project_id — тоже невалидный UUID (тег
// omitempty,uuid не пропускает не-nil указатель на ""), поэтому 400; очистка
// project_id делается отдельной ручкой DELETE /issues/{id}/project.
func TestIssueMalformedIDs(t *testing.T) {
	t.Parallel()
	srv, cookieName := setupServer(t)
	db := dbtest.Open(t)
	client := srv.Client()

	wsID := dbtest.SeedWorkspace(t, db)
	creator := seedMember(t, db, cookieName, wsID, model.WorkspaceRoleMember)

	teamsURL := srv.URL + "/api/workspaces/" + wsID + "/teams"
	teamID, _ := createTeam(t, client, db, teamsURL, creator.cookie)

	issuesURL := srv.URL + "/api/workspaces/" + wsID + "/issues"

	// team_id не-UUID → 400 (валидация тела)
	resp := doJSON(t, client, http.MethodPost, issuesURL, map[string]any{"team_id": "garbage", "title": "X"}, creator.cookie)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("garbage team_id: status = %d, want 400", resp.StatusCode)
	}
	_ = resp.Body.Close()

	// project_id не-UUID → 400
	resp = doJSON(t, client, http.MethodPost, issuesURL, map[string]any{"team_id": teamID, "title": "X", "project_id": "garbage"}, creator.cookie)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("garbage project_id: status = %d, want 400", resp.StatusCode)
	}
	_ = resp.Body.Close()

	// project_id пустая строка → невалидный UUID (omitempty не пропускает
	// не-nil указатель на "") → 400
	resp = doJSON(t, client, http.MethodPost, issuesURL, map[string]any{"team_id": teamID, "title": "EmptyProject", "project_id": ""}, creator.cookie)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("empty project_id: status = %d, want 400", resp.StatusCode)
	}
	_ = resp.Body.Close()

	// валидная задача без project_id для последующих проверок path/PATCH
	resp = doJSON(t, client, http.MethodPost, issuesURL, map[string]any{"team_id": teamID, "title": "NoProject"}, creator.cookie)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create issue: status = %d, want 201", resp.StatusCode)
	}
	var created struct {
		ID        string  `json:"id"`
		ProjectID *string `json:"project_id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		t.Fatalf("decode created: %v", err)
	}
	_ = resp.Body.Close()
	if created.ProjectID != nil {
		t.Fatalf("no project_id: project_id = %v, want nil", *created.ProjectID)
	}

	// PATCH с мусорным project_id → 400
	resp = doJSON(t, client, http.MethodPatch, issuesURL+"/"+created.ID, map[string]any{"project_id": "garbage"}, creator.cookie)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("patch garbage project_id: status = %d, want 400", resp.StatusCode)
	}
	_ = resp.Body.Close()

	// DELETE assignee с не-UUID userID в path → 400
	resp = doJSON(t, client, http.MethodDelete, issuesURL+"/"+created.ID+"/assignees/garbage", nil, creator.cookie)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("garbage assignee userID: status = %d, want 400", resp.StatusCode)
	}
	_ = resp.Body.Close()
}

// TestIssueRemoveProject проверяет отдельную ручку отвязки задачи от проекта:
// DELETE .../project очищает project_id (PATCH умеет только задавать),
// идемпотентно, с авторизацией по членству в команде.
func TestIssueRemoveProject(t *testing.T) {
	t.Parallel()
	srv, cookieName := setupServer(t)
	db := dbtest.Open(t)
	client := srv.Client()

	wsID := dbtest.SeedWorkspace(t, db)
	creator := seedMember(t, db, cookieName, wsID, model.WorkspaceRoleMember)
	outsider := seedMember(t, db, cookieName, wsID, model.WorkspaceRoleMember)

	projectID := dbtest.SeedProject(t, db, wsID)
	// «add team to project» при создании задачи с проектом создаёт junction —
	// подчищаем его до удаления проекта (FK-порядок).
	t.Cleanup(func() { _, _ = db.Exec(`DELETE FROM tracker.project_team WHERE project_id = $1`, projectID) })

	teamsURL := srv.URL + "/api/workspaces/" + wsID + "/teams"
	teamID, _ := createTeam(t, client, db, teamsURL, creator.cookie)

	issuesURL := srv.URL + "/api/workspaces/" + wsID + "/issues"

	// создать задачу с проектом → 201
	resp := doJSON(t, client, http.MethodPost, issuesURL, map[string]any{
		"team_id":    teamID,
		"title":      "Attached",
		"project_id": projectID,
	}, creator.cookie)
	if resp.StatusCode != http.StatusCreated {
		_ = resp.Body.Close()
		t.Fatalf("create issue: status = %d, want 201", resp.StatusCode)
	}
	var created struct {
		ID        string  `json:"id"`
		ProjectID *string `json:"project_id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		t.Fatalf("decode created: %v", err)
	}
	_ = resp.Body.Close()
	if created.ProjectID == nil || *created.ProjectID != projectID {
		t.Fatalf("created project_id = %v, want %s", created.ProjectID, projectID)
	}
	projectURL := issuesURL + "/" + created.ID + "/project"

	// без cookie → 401
	resp = doJSON(t, client, http.MethodDelete, projectURL, nil)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("no cookie: status = %d, want 401", resp.StatusCode)
	}
	_ = resp.Body.Close()

	// не участник команды → 403
	resp = doJSON(t, client, http.MethodDelete, projectURL, nil, outsider.cookie)
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("non-team-member: status = %d, want 403", resp.StatusCode)
	}
	_ = resp.Body.Close()

	// отвязать проект → 200, project_id очищен
	resp = doJSON(t, client, http.MethodDelete, projectURL, nil, creator.cookie)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("remove project: status = %d, want 200", resp.StatusCode)
	}
	var cleared struct {
		ProjectID *string `json:"project_id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&cleared); err != nil {
		t.Fatalf("decode cleared: %v", err)
	}
	_ = resp.Body.Close()
	if cleared.ProjectID != nil {
		t.Fatalf("project_id = %v, want nil", *cleared.ProjectID)
	}

	// повторный DELETE идемпотентен → 200
	resp = doJSON(t, client, http.MethodDelete, projectURL, nil, creator.cookie)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("idempotent remove project: status = %d, want 200", resp.StatusCode)
	}
	_ = resp.Body.Close()
}

func TestIssueCreateAuthorization(t *testing.T) {
	t.Parallel()
	srv, cookieName := setupServer(t)
	db := dbtest.Open(t)
	client := srv.Client()

	wsID := dbtest.SeedWorkspace(t, db)
	// creator создаёт команду (становится её owner), outsider — только участник
	// workspace (НЕ участник команды).
	creator := seedMember(t, db, cookieName, wsID, model.WorkspaceRoleMember)
	outsider := seedMember(t, db, cookieName, wsID, model.WorkspaceRoleMember)

	teamsURL := srv.URL + "/api/workspaces/" + wsID + "/teams"
	teamID, _ := createTeam(t, client, db, teamsURL, creator.cookie)

	issuesURL := srv.URL + "/api/workspaces/" + wsID + "/issues"

	// создание не-участником команды → 403
	resp := doJSON(t, client, http.MethodPost, issuesURL, map[string]any{"team_id": teamID, "title": "Nope"}, outsider.cookie)
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("non-team-member create: status = %d, want 403", resp.StatusCode)
	}
	_ = resp.Body.Close()

	// участник команды создаёт задачу, outsider (участник workspace) может её прочитать → 200
	resp = doJSON(t, client, http.MethodPost, issuesURL, map[string]any{"team_id": teamID, "title": "Readable"}, creator.cookie)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create: status = %d, want 201", resp.StatusCode)
	}
	var created struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		t.Fatalf("decode created: %v", err)
	}
	_ = resp.Body.Close()

	resp = doJSON(t, client, http.MethodGet, issuesURL+"/"+created.ID, nil, outsider.cookie)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("workspace-member get: status = %d, want 200", resp.StatusCode)
	}
	_ = resp.Body.Close()
}
