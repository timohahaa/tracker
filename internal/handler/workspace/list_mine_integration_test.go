//go:build integration

package workspace_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/timohahaa/tracker/internal/model"
	"github.com/timohahaa/tracker/test/dbtest"
)

func TestListMyWorkspaces(t *testing.T) {
	t.Parallel()
	srv, cookieName := setupServer(t)
	db := dbtest.Open(t)
	client := srv.Client()

	wsID := dbtest.SeedWorkspace(t, db)
	m := seedMember(t, db, cookieName, wsID, model.WorkspaceRoleOwner)

	// без cookie → 401
	resp := doJSON(t, client, http.MethodGet, srv.URL+"/api/workspaces", nil)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("no cookie: status = %d, want 401", resp.StatusCode)
	}
	_ = resp.Body.Close()

	// с cookie → 200, содержит workspace с ролью
	resp = doJSON(t, client, http.MethodGet, srv.URL+"/api/workspaces", nil, m.cookie)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list: status = %d, want 200", resp.StatusCode)
	}
	var body struct {
		Data []struct {
			ID   string `json:"id"`
			Slug string `json:"slug"`
			Role string `json:"role"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	_ = resp.Body.Close()

	var found bool
	for _, w := range body.Data {
		if w.ID == wsID {
			found = true
			if w.Role != string(model.WorkspaceRoleOwner) {
				t.Errorf("list: role = %q, want owner", w.Role)
			}
			if w.Slug == "" {
				t.Error("list: empty slug")
			}
		}
	}
	if !found {
		t.Fatalf("list: workspace %q not found in response", wsID)
	}
}
