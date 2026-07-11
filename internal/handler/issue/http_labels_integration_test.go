//go:build integration

package issue_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/google/uuid"

	"github.com/timohahaa/tracker/internal/model"
	"github.com/timohahaa/tracker/test/dbtest"
)

// labelBody — метка задачи в ответе issue.
type labelBody struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color"`
}

func decodeLabels(t *testing.T, resp *http.Response) []labelBody {
	t.Helper()
	var body struct {
		Labels []labelBody `json:"labels"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode issue: %v", err)
	}
	_ = resp.Body.Close()
	return body.Labels
}

func hasLabel(ls []labelBody, id string) bool {
	for _, l := range ls {
		if l.ID == id {
			return true
		}
	}
	return false
}

func TestIssueLabels_HappyPath(t *testing.T) {
	t.Parallel()
	srv, cookieName := setupServer(t)
	db := dbtest.Open(t)
	client := srv.Client()

	wsID := dbtest.SeedWorkspace(t, db)
	creator := seedMember(t, db, cookieName, wsID, model.WorkspaceRoleMember)

	teamsURL := srv.URL + "/api/workspaces/" + wsID + "/teams"
	teamID, _ := createTeam(t, client, db, teamsURL, creator.cookie)

	// ws-level метка и метка команды задачи — обе допустимы.
	wsLabel := dbtest.SeedLabel(t, db, wsID, nil)
	teamLabel := dbtest.SeedLabel(t, db, wsID, &teamID)

	issuesURL := srv.URL + "/api/workspaces/" + wsID + "/issues"

	// POST create с label_ids → 201, обе метки в ответе.
	resp := doJSON(t, client, http.MethodPost, issuesURL, map[string]any{
		"team_id":   teamID,
		"title":     "Labeled",
		"label_ids": []string{wsLabel, teamLabel},
	}, creator.cookie)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create with labels: status = %d, want 201", resp.StatusCode)
	}
	var created struct {
		ID     string      `json:"id"`
		Labels []labelBody `json:"labels"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		t.Fatalf("decode created: %v", err)
	}
	_ = resp.Body.Close()
	if !hasLabel(created.Labels, wsLabel) || !hasLabel(created.Labels, teamLabel) || len(created.Labels) != 2 {
		t.Fatalf("created labels = %+v, want ws+team", created.Labels)
	}
	issueID := created.ID
	t.Cleanup(func() { _, _ = db.Exec(`DELETE FROM tracker.issue_label WHERE issue_id = $1`, issueID) })

	// GET задачи → метки наполнены.
	resp = doJSON(t, client, http.MethodGet, issuesURL+"/"+issueID, nil, creator.cookie)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get: status = %d, want 200", resp.StatusCode)
	}
	if ls := decodeLabels(t, resp); len(ls) != 2 {
		t.Fatalf("get labels = %+v, want 2", ls)
	}

	// PATCH replace → остаётся только ws-метка.
	resp = doJSON(t, client, http.MethodPatch, issuesURL+"/"+issueID, map[string]any{
		"label_ids": []string{wsLabel},
	}, creator.cookie)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("patch replace: status = %d, want 200", resp.StatusCode)
	}
	ls := decodeLabels(t, resp)
	if len(ls) != 1 || ls[0].ID != wsLabel {
		t.Fatalf("after replace = %+v, want only ws label", ls)
	}

	// PATCH пустой набор → снять все.
	resp = doJSON(t, client, http.MethodPatch, issuesURL+"/"+issueID, map[string]any{
		"label_ids": []string{},
	}, creator.cookie)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("patch clear: status = %d, want 200", resp.StatusCode)
	}
	if ls := decodeLabels(t, resp); len(ls) != 0 {
		t.Fatalf("after clear = %+v, want empty", ls)
	}

	// PATCH без label_ids не трогает метки (навесим одну и обновим только title).
	resp = doJSON(t, client, http.MethodPatch, issuesURL+"/"+issueID, map[string]any{
		"label_ids": []string{teamLabel},
	}, creator.cookie)
	_ = resp.Body.Close()
	resp = doJSON(t, client, http.MethodPatch, issuesURL+"/"+issueID, map[string]any{
		"title": "Renamed",
	}, creator.cookie)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("patch title only: status = %d, want 200", resp.StatusCode)
	}
	if ls := decodeLabels(t, resp); len(ls) != 1 || ls[0].ID != teamLabel {
		t.Fatalf("after title-only patch = %+v, want team label untouched", ls)
	}
}

func TestIssueLabels_ListAndFilter(t *testing.T) {
	t.Parallel()
	srv, cookieName := setupServer(t)
	db := dbtest.Open(t)
	client := srv.Client()

	wsID := dbtest.SeedWorkspace(t, db)
	creator := seedMember(t, db, cookieName, wsID, model.WorkspaceRoleMember)

	teamsURL := srv.URL + "/api/workspaces/" + wsID + "/teams"
	teamID, _ := createTeam(t, client, db, teamsURL, creator.cookie)

	labelA := dbtest.SeedLabel(t, db, wsID, nil)
	labelB := dbtest.SeedLabel(t, db, wsID, nil)

	issuesURL := srv.URL + "/api/workspaces/" + wsID + "/issues"

	// Задача с labelA и задача с labelB.
	withA := createIssueWithLabels(t, client, issuesURL, teamID, "HasA", creator.cookie, labelA)
	_ = createIssueWithLabels(t, client, issuesURL, teamID, "HasB", creator.cookie, labelB)
	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM tracker.issue_label WHERE issue_id = ANY(SELECT id FROM tracker.issue WHERE team_id = $1)`, teamID)
	})

	// Список без фильтра → метки наполнены батчем (без N+1).
	resp := doJSON(t, client, http.MethodGet, issuesURL+"?team="+teamID, nil, creator.cookie)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list: status = %d, want 200", resp.StatusCode)
	}
	var listBody struct {
		Data []struct {
			ID     string      `json:"id"`
			Labels []labelBody `json:"labels"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&listBody); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	_ = resp.Body.Close()
	if len(listBody.Data) != 2 {
		t.Fatalf("list count = %d, want 2", len(listBody.Data))
	}
	for _, d := range listBody.Data {
		if len(d.Labels) != 1 {
			t.Errorf("issue %s labels = %+v, want 1", d.ID, d.Labels)
		}
	}

	// Фильтр label=labelA → только задача с labelA.
	resp = doJSON(t, client, http.MethodGet, issuesURL+"?team="+teamID+"&label="+labelA, nil, creator.cookie)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list filtered: status = %d, want 200", resp.StatusCode)
	}
	if err := json.NewDecoder(resp.Body).Decode(&listBody); err != nil {
		t.Fatalf("decode filtered: %v", err)
	}
	_ = resp.Body.Close()
	if len(listBody.Data) != 1 || listBody.Data[0].ID != withA {
		t.Fatalf("label filter = %+v, want only %s", listBody.Data, withA)
	}
}

func TestIssueLabels_Errors(t *testing.T) {
	t.Parallel()
	srv, cookieName := setupServer(t)
	db := dbtest.Open(t)
	client := srv.Client()

	wsID := dbtest.SeedWorkspace(t, db)
	creator := seedMember(t, db, cookieName, wsID, model.WorkspaceRoleMember)

	teamsURL := srv.URL + "/api/workspaces/" + wsID + "/teams"
	teamID, _ := createTeam(t, client, db, teamsURL, creator.cookie)
	// Другая команда и её метка (чужая для задачи team-1).
	otherTeamID, _ := createTeam(t, client, db, teamsURL, creator.cookie)
	foreignLabel := dbtest.SeedLabel(t, db, wsID, &otherTeamID)

	issuesURL := srv.URL + "/api/workspaces/" + wsID + "/issues"

	// 400 — метка чужой команды.
	resp := doJSON(t, client, http.MethodPost, issuesURL, map[string]any{
		"team_id":   teamID,
		"title":     "Nope",
		"label_ids": []string{foreignLabel},
	}, creator.cookie)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("foreign team label: status = %d, want 400", resp.StatusCode)
	}
	_ = resp.Body.Close()

	// 400 — несуществующая метка.
	resp = doJSON(t, client, http.MethodPost, issuesURL, map[string]any{
		"team_id":   teamID,
		"title":     "Nope",
		"label_ids": []string{uuid.NewString()},
	}, creator.cookie)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("missing label: status = %d, want 400", resp.StatusCode)
	}
	_ = resp.Body.Close()

	// 400 — не-UUID в label_ids (валидация тела).
	resp = doJSON(t, client, http.MethodPost, issuesURL, map[string]any{
		"team_id":   teamID,
		"title":     "Nope",
		"label_ids": []string{"garbage"},
	}, creator.cookie)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("garbage label id: status = %d, want 400", resp.StatusCode)
	}
	_ = resp.Body.Close()
}

// createIssueWithLabels создаёт задачу с набором меток и возвращает её id.
func createIssueWithLabels(t *testing.T, client *http.Client, issuesURL, teamID, title string, cookie *http.Cookie, labelIDs ...string) string {
	t.Helper()
	resp := doJSON(t, client, http.MethodPost, issuesURL, map[string]any{
		"team_id":   teamID,
		"title":     title,
		"label_ids": labelIDs,
	}, cookie)
	if resp.StatusCode != http.StatusCreated {
		_ = resp.Body.Close()
		t.Fatalf("create issue with labels: status = %d, want 201", resp.StatusCode)
	}
	var body struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode issue: %v", err)
	}
	_ = resp.Body.Close()
	return body.ID
}
