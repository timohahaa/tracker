//go:build integration

package auth_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/timohahaa/tracker/internal/model"
	"github.com/timohahaa/tracker/test/dbtest"
)

type meBody struct {
	ID       string `json:"id"`
	Email    string `json:"email"`
	Settings struct {
		CurrentWorkspaceID *string `json:"current_workspace_id"`
	} `json:"settings"`
}

func TestMe(t *testing.T) {
	t.Parallel()
	srv, cookieName := setupServer(t)
	db := dbtest.Open(t)
	client := srv.Client()

	userID := dbtest.SeedUser(t, db)
	sessionID := dbtest.SeedSession(t, db, userID)
	cookie := &http.Cookie{Name: cookieName, Value: sessionID}

	// без cookie → 401
	resp := doJSON(t, client, http.MethodGet, srv.URL+"/api/auth/me", nil)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("no cookie: status = %d, want 401", resp.StatusCode)
	}
	_ = resp.Body.Close()

	// с cookie → 200, settings пустые
	resp = doJSON(t, client, http.MethodGet, srv.URL+"/api/auth/me", nil, cookie)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("me: status = %d, want 200", resp.StatusCode)
	}
	var body meBody
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode me: %v", err)
	}
	_ = resp.Body.Close()
	if body.ID != userID {
		t.Errorf("me: id = %q, want %q", body.ID, userID)
	}
	if body.Settings.CurrentWorkspaceID != nil {
		t.Errorf("me: current_workspace_id = %v, want nil", *body.Settings.CurrentWorkspaceID)
	}

	// PATCH current_workspace_id → 200 + персист
	wsID := dbtest.SeedWorkspace(t, db)
	dbtest.SeedWorkspaceMember(t, db, wsID, userID, string(model.WorkspaceRoleOwner))

	resp = doJSON(t, client, http.MethodPatch, srv.URL+"/api/auth/me",
		map[string]any{"current_workspace_id": wsID}, cookie)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("patch me: status = %d, want 200", resp.StatusCode)
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode patch me: %v", err)
	}
	_ = resp.Body.Close()
	if body.Settings.CurrentWorkspaceID == nil || *body.Settings.CurrentWorkspaceID != wsID {
		t.Fatalf("patch me: current_workspace_id = %v, want %q", body.Settings.CurrentWorkspaceID, wsID)
	}

	// GET снова → персист
	resp = doJSON(t, client, http.MethodGet, srv.URL+"/api/auth/me", nil, cookie)
	_ = json.NewDecoder(resp.Body).Decode(&body)
	_ = resp.Body.Close()
	if body.Settings.CurrentWorkspaceID == nil || *body.Settings.CurrentWorkspaceID != wsID {
		t.Fatalf("me after patch: current_workspace_id not persisted")
	}

	// невалидный uuid → 400
	resp = doJSON(t, client, http.MethodPatch, srv.URL+"/api/auth/me",
		map[string]any{"current_workspace_id": "not-a-uuid"}, cookie)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("patch invalid uuid: status = %d, want 400", resp.StatusCode)
	}
	_ = resp.Body.Close()
}
