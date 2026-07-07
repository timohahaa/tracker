//go:build integration

package auth_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/timohahaa/tracker/config"
	"github.com/timohahaa/tracker/internal/handler"
	"github.com/timohahaa/tracker/internal/repository"
	"github.com/timohahaa/tracker/internal/service"
	"github.com/timohahaa/tracker/test/dbtest"
)

// setupServer собирает полный роутер поверх реальной БД и регистрирует очистку
// созданных регистрацией строк (session -> workspace_member -> workspace -> user).
// Возвращает сервер и имя session-cookie из конфигурации.
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

type registerResp struct {
	User      struct{ ID, Email string }
	Workspace struct{ ID, Slug string }
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

func sessionCookie(resp *http.Response, name string) *http.Cookie {
	for _, c := range resp.Cookies() {
		if c.Name == name {
			return c
		}
	}
	return nil
}

func TestAuthHappyPath(t *testing.T) {
	t.Parallel()
	srv, cookieName := setupServer(t)
	db := dbtest.Open(t)
	client := srv.Client()

	email := "http-" + uuid.NewString() + "@test.local"
	const password = "supersecret"

	// register
	resp := doJSON(t, client, http.MethodPost, srv.URL+"/api/auth/register", map[string]string{
		"email":          email,
		"password":       password,
		"display_name":   "HTTP User",
		"workspace_name": "Acme QA",
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("register: status = %d, want 201", resp.StatusCode)
	}
	var reg registerResp
	if err := json.NewDecoder(resp.Body).Decode(&reg); err != nil {
		t.Fatalf("decode register response: %v", err)
	}
	_ = resp.Body.Close()

	// очистка строк, созданных регистрацией.
	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM core.session WHERE user_id = $1`, reg.User.ID)
		_, _ = db.Exec(`DELETE FROM org.workspace_member WHERE user_id = $1`, reg.User.ID)
		_, _ = db.Exec(`DELETE FROM org.workspace WHERE id = $1`, reg.Workspace.ID)
		_, _ = db.Exec(`DELETE FROM core.user WHERE id = $1`, reg.User.ID)
	})

	regCookie := sessionCookie(resp, cookieName)
	if regCookie == nil || regCookie.Value == "" {
		t.Fatal("register: missing session_id cookie")
	}

	protectedURL := srv.URL + "/api/workspaces/" + reg.Workspace.ID + "/members"

	// доступ к защищённому эндпоинту под SessionAuth.
	resp = doJSON(t, client, http.MethodGet, protectedURL, nil, regCookie)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("protected (register cookie): status = %d, want 200", resp.StatusCode)
	}
	_ = resp.Body.Close()

	// login выдаёт рабочую сессию.
	resp = doJSON(t, client, http.MethodPost, srv.URL+"/api/auth/login", map[string]string{
		"email":    email,
		"password": password,
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("login: status = %d, want 200", resp.StatusCode)
	}
	loginCookie := sessionCookie(resp, cookieName)
	_ = resp.Body.Close()
	if loginCookie == nil || loginCookie.Value == "" {
		t.Fatal("login: missing session_id cookie")
	}

	resp = doJSON(t, client, http.MethodGet, protectedURL, nil, loginCookie)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("protected (login cookie): status = %d, want 200", resp.StatusCode)
	}
	_ = resp.Body.Close()
}

func TestProtectedUnauthorized(t *testing.T) {
	t.Parallel()
	srv, cookieName := setupServer(t)
	client := srv.Client()

	url := srv.URL + "/api/workspaces/" + uuid.NewString() + "/members"

	// нет cookie -> 401
	resp := doJSON(t, client, http.MethodGet, url, nil)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("no cookie: status = %d, want 401", resp.StatusCode)
	}
	_ = resp.Body.Close()

	// битая сессия -> 401
	resp = doJSON(t, client, http.MethodGet, url, nil, &http.Cookie{
		Name:  cookieName,
		Value: uuid.NewString(),
	})
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("bad cookie: status = %d, want 401", resp.StatusCode)
	}
	_ = resp.Body.Close()
}

func TestRegisterDuplicateEmail(t *testing.T) {
	t.Parallel()
	srv, _ := setupServer(t)
	db := dbtest.Open(t)
	client := srv.Client()

	email := "dup-http-" + uuid.NewString() + "@test.local"
	body := map[string]string{
		"email":          email,
		"password":       "supersecret",
		"display_name":   "Dup User",
		"workspace_name": "Dup Workspace",
	}

	resp := doJSON(t, client, http.MethodPost, srv.URL+"/api/auth/register", body)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("first register: status = %d, want 201", resp.StatusCode)
	}
	var reg registerResp
	_ = json.NewDecoder(resp.Body).Decode(&reg)
	_ = resp.Body.Close()
	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM core.session WHERE user_id = $1`, reg.User.ID)
		_, _ = db.Exec(`DELETE FROM org.workspace_member WHERE user_id = $1`, reg.User.ID)
		_, _ = db.Exec(`DELETE FROM org.workspace WHERE id = $1`, reg.Workspace.ID)
		_, _ = db.Exec(`DELETE FROM core.user WHERE id = $1`, reg.User.ID)
	})

	resp = doJSON(t, client, http.MethodPost, srv.URL+"/api/auth/register", body)
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("duplicate register: status = %d, want 409", resp.StatusCode)
	}
	_ = resp.Body.Close()
}
