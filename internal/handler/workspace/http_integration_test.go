//go:build integration

package workspace_test

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
	userID   string
	username string
	cookie   *http.Cookie
}

// seedMember создаёт пользователя, его membership с указанной ролью и активную
// сессию; возвращает данные участника с готовым cookie.
func seedMember(t *testing.T, db *sqlx.DB, cookieName, wsID string, role model.WorkspaceRole) member {
	t.Helper()
	userID := dbtest.SeedUser(t, db)
	username := dbtest.SeedWorkspaceMember(t, db, wsID, userID, string(role))
	sessionID := dbtest.SeedSession(t, db, userID)
	return member{
		userID:   userID,
		username: username,
		cookie:   &http.Cookie{Name: cookieName, Value: sessionID},
	}
}

func TestLoadWorkspace(t *testing.T) {
	t.Parallel()
	srv, cookieName := setupServer(t)
	db := dbtest.Open(t)
	client := srv.Client()

	wsID := dbtest.SeedWorkspace(t, db)
	m := seedMember(t, db, cookieName, wsID, model.WorkspaceRoleOwner)

	// участник → 200
	resp := doJSON(t, client, http.MethodGet, srv.URL+"/api/workspaces/"+wsID, nil, m.cookie)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("member: status = %d, want 200", resp.StatusCode)
	}
	_ = resp.Body.Close()

	// не-участник (валидная сессия, но нет membership) → 404
	outsider := dbtest.SeedUser(t, db)
	outsiderSession := dbtest.SeedSession(t, db, outsider)
	resp = doJSON(t, client, http.MethodGet, srv.URL+"/api/workspaces/"+wsID, nil,
		&http.Cookie{Name: cookieName, Value: outsiderSession})
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("non-member: status = %d, want 404", resp.StatusCode)
	}
	_ = resp.Body.Close()

	// несуществующий workspace → 404
	resp = doJSON(t, client, http.MethodGet, srv.URL+"/api/workspaces/"+uuid.NewString(), nil, m.cookie)
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("missing workspace: status = %d, want 404", resp.StatusCode)
	}
	_ = resp.Body.Close()

	// без cookie → 401
	resp = doJSON(t, client, http.MethodGet, srv.URL+"/api/workspaces/"+wsID, nil)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("no cookie: status = %d, want 401", resp.StatusCode)
	}
	_ = resp.Body.Close()
}

func TestUpdateSettings(t *testing.T) {
	t.Parallel()
	srv, cookieName := setupServer(t)
	db := dbtest.Open(t)
	client := srv.Client()

	wsID := dbtest.SeedWorkspace(t, db)
	owner := seedMember(t, db, cookieName, wsID, model.WorkspaceRoleOwner)
	admin := seedMember(t, db, cookieName, wsID, model.WorkspaceRoleAdmin)
	plain := seedMember(t, db, cookieName, wsID, model.WorkspaceRoleMember)

	url := srv.URL + "/api/workspaces/" + wsID

	// owner → 200
	resp := doJSON(t, client, http.MethodPatch, url, map[string]string{"name": "By Owner"}, owner.cookie)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("owner update: status = %d, want 200", resp.StatusCode)
	}
	_ = resp.Body.Close()

	// admin → 200
	resp = doJSON(t, client, http.MethodPatch, url, map[string]string{"name": "By Admin"}, admin.cookie)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("admin update: status = %d, want 200", resp.StatusCode)
	}
	_ = resp.Body.Close()

	// member → 403
	resp = doJSON(t, client, http.MethodPatch, url, map[string]string{"name": "By Member"}, plain.cookie)
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("member update: status = %d, want 403", resp.StatusCode)
	}
	_ = resp.Body.Close()
}

func TestListMembers(t *testing.T) {
	t.Parallel()
	srv, cookieName := setupServer(t)
	db := dbtest.Open(t)
	client := srv.Client()

	wsID := dbtest.SeedWorkspace(t, db)
	owner := seedMember(t, db, cookieName, wsID, model.WorkspaceRoleOwner)
	seedMember(t, db, cookieName, wsID, model.WorkspaceRoleMember)

	resp := doJSON(t, client, http.MethodGet, srv.URL+"/api/workspaces/"+wsID+"/members", nil, owner.cookie)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list members: status = %d, want 200", resp.StatusCode)
	}
	var body struct {
		Data []struct {
			UserID      string `json:"user_id"`
			Email       string `json:"email"`
			DisplayName string `json:"display_name"`
			Role        string `json:"role"`
			Username    string `json:"username"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode members: %v", err)
	}
	_ = resp.Body.Close()
	if len(body.Data) != 2 {
		t.Fatalf("list members len = %d, want 2", len(body.Data))
	}
	for _, m := range body.Data {
		if m.Email == "" || m.DisplayName == "" {
			t.Errorf("member missing email/display_name: %+v", m)
		}
	}
}

func TestUpdateMember(t *testing.T) {
	t.Parallel()
	srv, cookieName := setupServer(t)
	db := dbtest.Open(t)
	client := srv.Client()

	wsID := dbtest.SeedWorkspace(t, db)
	owner := seedMember(t, db, cookieName, wsID, model.WorkspaceRoleOwner)
	target := seedMember(t, db, cookieName, wsID, model.WorkspaceRoleMember)

	membersURL := srv.URL + "/api/workspaces/" + wsID + "/members/"

	// обновление role/username/title → 200
	newUsername := "u-" + uuid.NewString()
	resp := doJSON(t, client, http.MethodPatch, membersURL+target.userID, map[string]any{
		"role":     "admin",
		"username": newUsername,
		"title":    "Engineer",
	}, owner.cookie)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("update member: status = %d, want 200", resp.StatusCode)
	}
	_ = resp.Body.Close()

	// назначить owner → 403
	resp = doJSON(t, client, http.MethodPatch, membersURL+target.userID, map[string]any{
		"role": "owner",
	}, owner.cookie)
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("set owner: status = %d, want 403", resp.StatusCode)
	}
	_ = resp.Body.Close()

	// понизить последнего owner → 403
	resp = doJSON(t, client, http.MethodPatch, membersURL+owner.userID, map[string]any{
		"role": "member",
	}, owner.cookie)
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("demote last owner: status = %d, want 403", resp.StatusCode)
	}
	_ = resp.Body.Close()

	// конфликт username → 409 (target пытается занять username owner)
	resp = doJSON(t, client, http.MethodPatch, membersURL+target.userID, map[string]any{
		"username": owner.username,
	}, owner.cookie)
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("username conflict: status = %d, want 409", resp.StatusCode)
	}
	_ = resp.Body.Close()

	// невалидная роль → 400
	resp = doJSON(t, client, http.MethodPatch, membersURL+target.userID, map[string]any{
		"role": "superuser",
	}, owner.cookie)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("invalid role: status = %d, want 400", resp.StatusCode)
	}
	_ = resp.Body.Close()
}

func TestDeleteMember(t *testing.T) {
	t.Parallel()
	srv, cookieName := setupServer(t)
	db := dbtest.Open(t)
	client := srv.Client()

	wsID := dbtest.SeedWorkspace(t, db)
	owner := seedMember(t, db, cookieName, wsID, model.WorkspaceRoleOwner)
	target := seedMember(t, db, cookieName, wsID, model.WorkspaceRoleMember)

	membersURL := srv.URL + "/api/workspaces/" + wsID + "/members/"

	// удаление обычного участника → 204
	resp := doJSON(t, client, http.MethodDelete, membersURL+target.userID, nil, owner.cookie)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("delete member: status = %d, want 204", resp.StatusCode)
	}
	_ = resp.Body.Close()

	// удаление последнего owner → 403
	resp = doJSON(t, client, http.MethodDelete, membersURL+owner.userID, nil, owner.cookie)
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("delete last owner: status = %d, want 403", resp.StatusCode)
	}
	_ = resp.Body.Close()
}
