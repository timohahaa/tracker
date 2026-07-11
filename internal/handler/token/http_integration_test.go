//go:build integration

package token_test

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

// doBearer шлёт запрос с заголовком Authorization: Bearer <token> (без cookie).
func doBearer(t *testing.T, client *http.Client, method, url, token string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
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

// cleanupTokens удаляет токены workspace, созданные через API-эндпоинты — они не
// покрыты seed-фабриками.
func cleanupTokens(t *testing.T, db *sqlx.DB, wsID string) {
	t.Helper()
	t.Cleanup(func() { _, _ = db.Exec(`DELETE FROM core.api_token WHERE workspace_id = $1`, wsID) })
}

func TestTokenCRUD(t *testing.T) {
	t.Parallel()
	srv, cookieName := setupServer(t)
	db := dbtest.Open(t)
	client := srv.Client()

	wsID := dbtest.SeedWorkspace(t, db)
	cleanupTokens(t, db, wsID)
	owner := seedMember(t, db, cookieName, wsID, model.WorkspaceRoleMember)

	tokensURL := srv.URL + "/api/workspaces/" + wsID + "/tokens"

	// без cookie → 401
	resp := doJSON(t, client, http.MethodPost, tokensURL, map[string]any{"name": "ci"})
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("no cookie: status = %d, want 401", resp.StatusCode)
	}
	_ = resp.Body.Close()

	// невалидное тело (нет name) → 400
	resp = doJSON(t, client, http.MethodPost, tokensURL, map[string]any{}, owner.cookie)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("missing name: status = %d, want 400", resp.StatusCode)
	}
	_ = resp.Body.Close()

	// happy-path создание → 201, сырой токен присутствует
	resp = doJSON(t, client, http.MethodPost, tokensURL, map[string]any{"name": "ci"}, owner.cookie)
	if resp.StatusCode != http.StatusCreated {
		_ = resp.Body.Close()
		t.Fatalf("create token: status = %d, want 201", resp.StatusCode)
	}
	var created struct {
		ID    string `json:"id"`
		Token string `json:"token"`
		Name  string `json:"name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		t.Fatalf("decode token: %v", err)
	}
	_ = resp.Body.Close()
	if created.ID == "" || created.Token == "" || created.Token != created.ID {
		t.Fatalf("created = %+v, want token == id, both set", created)
	}

	// выданным токеном можно авторизоваться на защищённом Bearer-эндпоинте (/mcp):
	// auth проходит → не 401 (заглушка MCP отдаёт 501).
	resp = doBearer(t, client, http.MethodGet, srv.URL+"/mcp", created.Token)
	if resp.StatusCode == http.StatusUnauthorized {
		t.Fatalf("bearer auth with fresh token: status = 401, want authorized")
	}
	_ = resp.Body.Close()

	// GET → 200, только свои токены, без поля token
	resp = doJSON(t, client, http.MethodGet, tokensURL, nil, owner.cookie)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list tokens: status = %d, want 200", resp.StatusCode)
	}
	var listed struct {
		Data []map[string]any `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&listed); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	_ = resp.Body.Close()
	if len(listed.Data) != 1 {
		t.Fatalf("list len = %d, want 1 (own token only)", len(listed.Data))
	}
	if listed.Data[0]["id"] != created.ID {
		t.Fatalf("listed id = %v, want %s", listed.Data[0]["id"], created.ID)
	}
	if _, ok := listed.Data[0]["token"]; ok {
		t.Fatal("list must not expose the token field")
	}

	// revoke → 204
	tokenURL := tokensURL + "/" + created.ID
	resp = doJSON(t, client, http.MethodDelete, tokenURL, nil, owner.cookie)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("revoke token: status = %d, want 204", resp.StatusCode)
	}
	_ = resp.Body.Close()

	// после отзыва Bearer-авторизация ломается → 401
	resp = doBearer(t, client, http.MethodGet, srv.URL+"/mcp", created.Token)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("bearer auth after revoke: status = %d, want 401", resp.StatusCode)
	}
	_ = resp.Body.Close()

	// список пуст после отзыва
	resp = doJSON(t, client, http.MethodGet, tokensURL, nil, owner.cookie)
	if err := json.NewDecoder(resp.Body).Decode(&listed); err != nil {
		t.Fatalf("decode list after revoke: %v", err)
	}
	_ = resp.Body.Close()
	if len(listed.Data) != 0 {
		t.Fatalf("list after revoke len = %d, want 0", len(listed.Data))
	}
}

func TestTokenListIsolationAndRevokeForeign(t *testing.T) {
	t.Parallel()
	srv, cookieName := setupServer(t)
	db := dbtest.Open(t)
	client := srv.Client()

	wsID := dbtest.SeedWorkspace(t, db)
	cleanupTokens(t, db, wsID)
	owner := seedMember(t, db, cookieName, wsID, model.WorkspaceRoleMember)
	other := seedMember(t, db, cookieName, wsID, model.WorkspaceRoleMember)

	// чужой токен в том же workspace
	foreignID := dbtest.SeedAPIToken(t, db, other.userID, wsID)

	tokensURL := srv.URL + "/api/workspaces/" + wsID + "/tokens"

	// owner видит только свои токены (0), не чужой
	resp := doJSON(t, client, http.MethodGet, tokensURL, nil, owner.cookie)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list: status = %d, want 200", resp.StatusCode)
	}
	var listed struct {
		Data []map[string]any `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&listed); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	_ = resp.Body.Close()
	if len(listed.Data) != 0 {
		t.Fatalf("owner list len = %d, want 0 (foreign token hidden)", len(listed.Data))
	}

	// revoke чужого токена → 404 (не палим существование)
	resp = doJSON(t, client, http.MethodDelete, tokensURL+"/"+foreignID, nil, owner.cookie)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("revoke foreign: status = %d, want 404", resp.StatusCode)
	}
	_ = resp.Body.Close()

	// revoke несуществующего → 404
	resp = doJSON(t, client, http.MethodDelete, tokensURL+"/"+uuid.NewString(), nil, owner.cookie)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("revoke missing: status = %d, want 404", resp.StatusCode)
	}
	_ = resp.Body.Close()

	// малформед tokenID → 400 (парсинг на границе)
	resp = doJSON(t, client, http.MethodDelete, tokensURL+"/garbage", nil, owner.cookie)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("revoke malformed: status = %d, want 400", resp.StatusCode)
	}
	_ = resp.Body.Close()
}
