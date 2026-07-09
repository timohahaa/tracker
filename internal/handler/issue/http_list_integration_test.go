//go:build integration

package issue_test

import (
	"encoding/json"
	"net/http"
	"net/url"
	"testing"

	"github.com/timohahaa/tracker/internal/model"
	"github.com/timohahaa/tracker/test/dbtest"
)

// issueListBody — минимальная форма ответа списка для декодирования в тестах.
type issueListBody struct {
	Data []struct {
		ID string `json:"id"`
	} `json:"data"`
	Pagination struct {
		NextCursor string `json:"next_cursor"`
		HasMore    bool   `json:"has_more"`
	} `json:"pagination"`
}

// createIssue создаёт задачу через API и возвращает её id.
func createIssue(t *testing.T, client *http.Client, issuesURL, teamID, title string, cookie *http.Cookie) string {
	t.Helper()
	resp := doJSON(t, client, http.MethodPost, issuesURL, map[string]any{"team_id": teamID, "title": title}, cookie)
	if resp.StatusCode != http.StatusCreated {
		_ = resp.Body.Close()
		t.Fatalf("create issue: status = %d, want 201", resp.StatusCode)
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

func TestListIssuesPagination(t *testing.T) {
	t.Parallel()
	srv, cookieName := setupServer(t)
	db := dbtest.Open(t)
	client := srv.Client()

	wsID := dbtest.SeedWorkspace(t, db)
	creator := seedMember(t, db, cookieName, wsID, model.WorkspaceRoleMember)

	teamsURL := srv.URL + "/api/workspaces/" + wsID + "/teams"
	teamID, _ := createTeam(t, client, db, teamsURL, creator.cookie)

	issuesURL := srv.URL + "/api/workspaces/" + wsID + "/issues"

	// три задачи, страница по 2 → две страницы (2 + 1).
	want := map[string]bool{}
	want[createIssue(t, client, issuesURL, teamID, "A", creator.cookie)] = true
	want[createIssue(t, client, issuesURL, teamID, "B", creator.cookie)] = true
	want[createIssue(t, client, issuesURL, teamID, "C", creator.cookie)] = true

	// без cookie → 401
	resp := doJSON(t, client, http.MethodGet, issuesURL, nil)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("no cookie: status = %d, want 401", resp.StatusCode)
	}
	_ = resp.Body.Close()

	// page 1: limit=2, скоуп по team (изоляция от чужих задач в общей БД)
	page1URL := issuesURL + "?limit=2&team=" + teamID
	resp = doJSON(t, client, http.MethodGet, page1URL, nil, creator.cookie)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("page 1: status = %d, want 200", resp.StatusCode)
	}
	var p1 issueListBody
	if err := json.NewDecoder(resp.Body).Decode(&p1); err != nil {
		t.Fatalf("decode page 1: %v", err)
	}
	_ = resp.Body.Close()
	if len(p1.Data) != 2 || !p1.Pagination.HasMore || p1.Pagination.NextCursor == "" {
		t.Fatalf("page 1 = %+v, want 2 items + has_more + cursor", p1)
	}

	// page 2: по next_cursor → оставшаяся 1 задача, has_more=false
	page2URL := issuesURL + "?limit=2&team=" + teamID + "&cursor=" + url.QueryEscape(p1.Pagination.NextCursor)
	resp = doJSON(t, client, http.MethodGet, page2URL, nil, creator.cookie)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("page 2: status = %d, want 200", resp.StatusCode)
	}
	var p2 issueListBody
	if err := json.NewDecoder(resp.Body).Decode(&p2); err != nil {
		t.Fatalf("decode page 2: %v", err)
	}
	_ = resp.Body.Close()
	if len(p2.Data) != 1 || p2.Pagination.HasMore {
		t.Fatalf("page 2 = %+v, want 1 item + has_more=false", p2)
	}

	// объединение страниц = все три задачи ровно один раз
	got := map[string]int{}
	for _, d := range p1.Data {
		got[d.ID]++
	}
	for _, d := range p2.Data {
		got[d.ID]++
	}
	if len(got) != 3 {
		t.Fatalf("union has %d distinct issues, want 3", len(got))
	}
	for id := range want {
		if got[id] != 1 {
			t.Errorf("issue %s appeared %d times across pages, want 1", id, got[id])
		}
	}
}

func TestListIssuesBadParams(t *testing.T) {
	t.Parallel()
	srv, cookieName := setupServer(t)
	db := dbtest.Open(t)
	client := srv.Client()

	wsID := dbtest.SeedWorkspace(t, db)
	member := seedMember(t, db, cookieName, wsID, model.WorkspaceRoleMember)
	issuesURL := srv.URL + "/api/workspaces/" + wsID + "/issues"

	cases := []string{
		"?sort=bogus",
		"?order=sideways",
		"?cursor=!!!not-base64!!!",
		"?status=notauuid",
		"?assignee=notauuid",
		"?team=notauuid",
		"?label=notauuid",
	}
	for _, qs := range cases {
		resp := doJSON(t, client, http.MethodGet, issuesURL+qs, nil, member.cookie)
		if resp.StatusCode != http.StatusBadRequest {
			_ = resp.Body.Close()
			t.Fatalf("%s: status = %d, want 400", qs, resp.StatusCode)
		}
		_ = resp.Body.Close()
	}
}
