//go:build integration

package issue_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/timohahaa/tracker/internal/model"
	"github.com/timohahaa/tracker/test/dbtest"
)

// assigneesOf декодирует поле assignees из ответа issue.
type assigneeBody struct {
	UserID    string `json:"user_id"`
	IsPrimary bool   `json:"is_primary"`
}

func decodeAssignees(t *testing.T, resp *http.Response) []assigneeBody {
	t.Helper()
	var body struct {
		Assignees []assigneeBody `json:"assignees"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode issue: %v", err)
	}
	_ = resp.Body.Close()
	return body.Assignees
}

// primaryOf возвращает user_id ответственного (is_primary) или "" если его нет.
func primaryOf(as []assigneeBody) string {
	for _, a := range as {
		if a.IsPrimary {
			return a.UserID
		}
	}
	return ""
}

func hasAssignee(as []assigneeBody, userID string) bool {
	for _, a := range as {
		if a.UserID == userID {
			return true
		}
	}
	return false
}

func TestIssueAssignees_HappyPath(t *testing.T) {
	t.Parallel()
	srv, cookieName := setupServer(t)
	db := dbtest.Open(t)
	client := srv.Client()

	wsID := dbtest.SeedWorkspace(t, db)
	creator := seedMember(t, db, cookieName, wsID, model.WorkspaceRoleMember)

	teamsURL := srv.URL + "/api/workspaces/" + wsID + "/teams"
	teamID, _ := createTeam(t, client, db, teamsURL, creator.cookie)

	issuesURL := srv.URL + "/api/workspaces/" + wsID + "/issues"
	issueID := createIssue(t, client, issuesURL, teamID, "Assignable", creator.cookie)
	t.Cleanup(func() { _, _ = db.Exec(`DELETE FROM tracker.issue_assignee WHERE issue_id = $1`, issueID) })

	// Два участника команды задачи.
	u1 := seedMember(t, db, cookieName, wsID, model.WorkspaceRoleMember)
	u2 := seedMember(t, db, cookieName, wsID, model.WorkspaceRoleMember)
	dbtest.SeedTeamMember(t, db, teamID, u1.userID, "member")
	dbtest.SeedTeamMember(t, db, teamID, u2.userID, "member")

	assigneesURL := issuesURL + "/" + issueID + "/assignees"

	// POST участника u1 → 200, u1 в наборе.
	resp := doJSON(t, client, http.MethodPost, assigneesURL, map[string]any{"user_id": u1.userID}, creator.cookie)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("add u1: status = %d, want 200", resp.StatusCode)
	}
	if as := decodeAssignees(t, resp); !hasAssignee(as, u1.userID) || primaryOf(as) != "" {
		t.Fatalf("after add u1 = %+v, want u1 present, no primary", as)
	}

	// PUT primary u1 → u1 ответственный.
	resp = doJSON(t, client, http.MethodPut, assigneesURL+"/"+u1.userID+"/primary", nil, creator.cookie)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("set primary u1: status = %d, want 200", resp.StatusCode)
	}
	if as := decodeAssignees(t, resp); primaryOf(as) != u1.userID {
		t.Fatalf("primary after set u1 = %q, want %s", primaryOf(as), u1.userID)
	}

	// POST u2, затем PUT primary u2 → старый (u1) перестаёт быть primary.
	resp = doJSON(t, client, http.MethodPost, assigneesURL, map[string]any{"user_id": u2.userID}, creator.cookie)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("add u2: status = %d, want 200", resp.StatusCode)
	}
	_ = resp.Body.Close()
	resp = doJSON(t, client, http.MethodPut, assigneesURL+"/"+u2.userID+"/primary", nil, creator.cookie)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("set primary u2: status = %d, want 200", resp.StatusCode)
	}
	as := decodeAssignees(t, resp)
	if primaryOf(as) != u2.userID {
		t.Fatalf("primary after set u2 = %q, want %s", primaryOf(as), u2.userID)
	}
	if len(as) != 2 {
		t.Fatalf("assignees after add u2 = %+v, want 2", as)
	}

	// DELETE u1 → 204; u1 больше нет в наборе.
	resp = doJSON(t, client, http.MethodDelete, assigneesURL+"/"+u1.userID, nil, creator.cookie)
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("delete u1: status = %d, want 204", resp.StatusCode)
	}
	_ = resp.Body.Close()

	resp = doJSON(t, client, http.MethodGet, issuesURL+"/"+issueID, nil, creator.cookie)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get after delete: status = %d, want 200", resp.StatusCode)
	}
	if as := decodeAssignees(t, resp); hasAssignee(as, u1.userID) || len(as) != 1 {
		t.Fatalf("after delete u1 = %+v, want only u2", as)
	}

	// PATCH replace всего набора → только u1 (ответственный).
	patchURL := issuesURL + "/" + issueID
	resp = doJSON(t, client, http.MethodPatch, patchURL, map[string]any{
		"assignees": []map[string]any{{"user_id": u1.userID, "is_primary": true}},
	}, creator.cookie)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("patch replace: status = %d, want 200", resp.StatusCode)
	}
	as = decodeAssignees(t, resp)
	if len(as) != 1 || as[0].UserID != u1.userID || !as[0].IsPrimary {
		t.Fatalf("after replace = %+v, want only u1 primary", as)
	}
}

func TestIssueAssignees_Errors(t *testing.T) {
	t.Parallel()
	srv, cookieName := setupServer(t)
	db := dbtest.Open(t)
	client := srv.Client()

	wsID := dbtest.SeedWorkspace(t, db)
	creator := seedMember(t, db, cookieName, wsID, model.WorkspaceRoleMember)
	// outsider — участник workspace, но НЕ участник команды.
	outsider := seedMember(t, db, cookieName, wsID, model.WorkspaceRoleMember)

	teamsURL := srv.URL + "/api/workspaces/" + wsID + "/teams"
	teamID, _ := createTeam(t, client, db, teamsURL, creator.cookie)

	issuesURL := srv.URL + "/api/workspaces/" + wsID + "/issues"
	issueID := createIssue(t, client, issuesURL, teamID, "Assignable", creator.cookie)
	t.Cleanup(func() { _, _ = db.Exec(`DELETE FROM tracker.issue_assignee WHERE issue_id = $1`, issueID) })

	assigneesURL := issuesURL + "/" + issueID + "/assignees"

	// 401 — без сессии.
	resp := doJSON(t, client, http.MethodPost, assigneesURL, map[string]any{"user_id": outsider.userID})
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("no cookie: status = %d, want 401", resp.StatusCode)
	}
	_ = resp.Body.Close()

	// 403 — actor не участник команды задачи.
	resp = doJSON(t, client, http.MethodPost, assigneesURL, map[string]any{"user_id": outsider.userID}, outsider.cookie)
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("non-team-member actor: status = %d, want 403", resp.StatusCode)
	}
	_ = resp.Body.Close()

	// 400 — добавление не-члена команды (outsider — только участник workspace).
	resp = doJSON(t, client, http.MethodPost, assigneesURL, map[string]any{"user_id": outsider.userID}, creator.cookie)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("add non-team-member: status = %d, want 400", resp.StatusCode)
	}
	_ = resp.Body.Close()

	// 400 — назначение ответственным не-участника задачи.
	resp = doJSON(t, client, http.MethodPut, assigneesURL+"/"+outsider.userID+"/primary", nil, creator.cookie)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("primary of non-assignee: status = %d, want 400", resp.StatusCode)
	}
	_ = resp.Body.Close()
}
