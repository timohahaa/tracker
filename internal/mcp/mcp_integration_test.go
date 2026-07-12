//go:build integration

package mcp_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/timohahaa/tracker/config"
	"github.com/timohahaa/tracker/internal/handler"
	"github.com/timohahaa/tracker/internal/repository"
	"github.com/timohahaa/tracker/internal/service"
	"github.com/timohahaa/tracker/test/dbtest"
)

// Integration-тесты MCP-сервера идут через реальный mcp.Client поверх Streamable
// HTTP: хендшейк (initialize → tools/list → tools/call), happy-path tools с
// проверкой БД, авторизация по API-токену и скоуп на workspace токена.

// bearerRoundTripper добавляет Authorization: Bearer <token> ко всем запросам
// MCP-клиента — так токен доходит до chi TokenAuth перед /mcp.
type bearerRoundTripper struct {
	token string
	base  http.RoundTripper
}

func (b bearerRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	if b.token != "" {
		req.Header.Set("Authorization", "Bearer "+b.token)
	}
	return b.base.RoundTrip(req)
}

func setupServer(t *testing.T) (*httptest.Server, *sqlx.DB) {
	t.Helper()
	db := dbtest.Open(t)

	cfg := config.Config{SessionTTL: 24 * time.Hour}
	cfg.SetDefaults()
	repos := repository.New(db)
	services := service.New(repos, cfg)
	srv := httptest.NewServer(handler.New(services, cfg))
	t.Cleanup(srv.Close)
	return srv, db
}

// connect подключает MCP-клиент к /mcp с заданным Bearer-токеном.
func connect(t *testing.T, url, token string) *mcp.ClientSession {
	t.Helper()
	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "1.0.0"}, nil)
	httpClient := &http.Client{Transport: bearerRoundTripper{token: token, base: http.DefaultTransport}}
	transport := &mcp.StreamableClientTransport{Endpoint: url + "/mcp", HTTPClient: httpClient}
	sess, err := client.Connect(context.Background(), transport, nil)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(func() { _ = sess.Close() })
	return sess
}

// seedFixture создаёт workspace с участником-owner, команду с backlog-статусом и
// членством, а также валидный API-токен. Возвращает id-шники и токен.
type fixture struct {
	wsID     string
	userID   string
	teamID   string
	teamKey  string
	statusID string
	token    string
}

func seedFixture(t *testing.T, db *sqlx.DB) fixture {
	t.Helper()
	userID := dbtest.SeedUser(t, db)
	wsID := dbtest.SeedWorkspace(t, db)
	dbtest.SeedWorkspaceMember(t, db, wsID, userID, "owner")
	teamID := dbtest.SeedTeam(t, db, wsID)
	statusID := dbtest.SeedStatus(t, db, teamID, "Backlog", "backlog", 0)
	dbtest.SeedTeamMember(t, db, teamID, userID, "member")
	token := dbtest.SeedAPIToken(t, db, userID, wsID)

	var key string
	if err := db.Get(&key, `SELECT key FROM org.team WHERE id = $1`, teamID); err != nil {
		t.Fatalf("read team key: %v", err)
	}
	return fixture{wsID: wsID, userID: userID, teamID: teamID, teamKey: key, statusID: statusID, token: token}
}

func textContent(t *testing.T, res *mcp.CallToolResult) string {
	t.Helper()
	if len(res.Content) == 0 {
		t.Fatalf("result has no content")
	}
	tc, ok := res.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("content is not text: %T", res.Content[0])
	}
	return tc.Text
}

// callTool вызывает tool и (если out != nil и это не ошибка) декодирует текстовый
// content в out. Возвращает результат для проверки IsError.
func callTool(t *testing.T, sess *mcp.ClientSession, name string, args map[string]any, out any) *mcp.CallToolResult {
	t.Helper()
	res, err := sess.CallTool(context.Background(), &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		t.Fatalf("call %s: %v", name, err)
	}
	if out != nil && !res.IsError {
		if err := json.Unmarshal([]byte(textContent(t, res)), out); err != nil {
			t.Fatalf("decode %s result: %v", name, err)
		}
	}
	return res
}

func TestMCP_HandshakeAndListTools(t *testing.T) {
	srv, db := setupServer(t)
	fx := seedFixture(t, db)

	sess := connect(t, srv.URL, fx.token)
	res, err := sess.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("list tools: %v", err)
	}

	want := map[string]bool{
		"create_issue": true, "get_issue": true, "update_issue": true,
		"delete_issue": true, "search_issues": true, "list_projects": true, "list_teams": true,
	}
	got := make(map[string]bool)
	for _, tool := range res.Tools {
		got[tool.Name] = true
	}
	for name := range want {
		if !got[name] {
			t.Errorf("tool %q missing from tools/list", name)
		}
	}
	if len(res.Tools) != len(want) {
		t.Errorf("tools count = %d, want %d (%v)", len(res.Tools), len(want), got)
	}
}

func TestMCP_CreateAndGetIssue(t *testing.T) {
	srv, db := setupServer(t)
	fx := seedFixture(t, db)
	sess := connect(t, srv.URL, fx.token)

	var created struct {
		ID         string `json:"id"`
		Identifier string `json:"identifier"`
		Title      string `json:"title"`
		Priority   string `json:"priority"`
	}
	res := callTool(t, sess, "create_issue", map[string]any{
		"team_id":  fx.teamID,
		"title":    "MCP issue",
		"priority": "high",
	}, &created)
	if res.IsError {
		t.Fatalf("create_issue error: %s", textContent(t, res))
	}
	if created.ID == "" || created.Title != "MCP issue" || created.Priority != "high" {
		t.Fatalf("created = %+v", created)
	}
	t.Cleanup(func() { _, _ = db.Exec(`DELETE FROM tracker.issue WHERE id = $1`, created.ID) })

	// Задача действительно в БД, в правильной команде.
	var dbTitle, dbTeam string
	if err := db.QueryRow(`SELECT title, team_id FROM tracker.issue WHERE id = $1`, created.ID).Scan(&dbTitle, &dbTeam); err != nil {
		t.Fatalf("read issue: %v", err)
	}
	if dbTitle != "MCP issue" || dbTeam != fx.teamID {
		t.Errorf("db issue = (%q, %q), want (MCP issue, %s)", dbTitle, dbTeam, fx.teamID)
	}

	// get_issue по UUID возвращает ту же задачу.
	var fetched struct {
		ID         string `json:"id"`
		Identifier string `json:"identifier"`
	}
	res = callTool(t, sess, "get_issue", map[string]any{"ref": created.ID}, &fetched)
	if res.IsError {
		t.Fatalf("get_issue error: %s", textContent(t, res))
	}
	if fetched.ID != created.ID {
		t.Errorf("get_issue id = %q, want %q", fetched.ID, created.ID)
	}
}

func TestMCP_SearchListProjectsTeams(t *testing.T) {
	srv, db := setupServer(t)
	fx := seedFixture(t, db)
	sess := connect(t, srv.URL, fx.token)

	dbtest.SeedIssueDetailed(t, db, fx.teamID, fx.statusID, dbtest.IssueSeed{Title: "seeded"})
	dbtest.SeedProject(t, db, fx.wsID)

	var search struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	res := callTool(t, sess, "search_issues", map[string]any{"team": fx.teamID}, &search)
	if res.IsError {
		t.Fatalf("search_issues error: %s", textContent(t, res))
	}
	if len(search.Data) != 1 {
		t.Errorf("search_issues returned %d issues, want 1", len(search.Data))
	}

	var projects struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	res = callTool(t, sess, "list_projects", map[string]any{}, &projects)
	if res.IsError {
		t.Fatalf("list_projects error: %s", textContent(t, res))
	}
	if len(projects.Data) != 1 {
		t.Errorf("list_projects returned %d, want 1", len(projects.Data))
	}

	var teams struct {
		Data []struct {
			ID  string `json:"id"`
			Key string `json:"key"`
		} `json:"data"`
	}
	res = callTool(t, sess, "list_teams", map[string]any{}, &teams)
	if res.IsError {
		t.Fatalf("list_teams error: %s", textContent(t, res))
	}
	if len(teams.Data) != 1 || teams.Data[0].ID != fx.teamID {
		t.Errorf("list_teams = %+v, want team %s", teams.Data, fx.teamID)
	}
}

func TestMCP_AuthRequired(t *testing.T) {
	srv, db := setupServer(t)
	_ = seedFixture(t, db)

	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "1.0.0"}, nil)

	// Пустой токен и корректный по форме, но несуществующий (UUID) — оба должны
	// давать ошибку авторизации на хендшейке.
	for _, token := range []string{"", "00000000-0000-0000-0000-000000000000"} {
		httpClient := &http.Client{Transport: bearerRoundTripper{token: token, base: http.DefaultTransport}}
		transport := &mcp.StreamableClientTransport{Endpoint: srv.URL + "/mcp", HTTPClient: httpClient}
		sess, err := client.Connect(context.Background(), transport, nil)
		if err == nil {
			_ = sess.Close()
			t.Errorf("connect with token %q succeeded, want auth failure", token)
		}
	}
}

func TestMCP_TokenScopedToWorkspace(t *testing.T) {
	srv, db := setupServer(t)
	fx := seedFixture(t, db)
	sess := connect(t, srv.URL, fx.token)

	// Чужой workspace со своей задачей — токен fx на неё не должен смотреть.
	otherWS := dbtest.SeedWorkspace(t, db)
	otherTeam := dbtest.SeedTeam(t, db, otherWS)
	otherStatus := dbtest.SeedStatus(t, db, otherTeam, "Backlog", "backlog", 0)
	otherIssue, _ := dbtest.SeedIssueDetailed(t, db, otherTeam, otherStatus, dbtest.IssueSeed{Title: "foreign"})

	// get_issue чужой задачи по её UUID → not_found: GetByID скоупнут на
	// workspace токена, поэтому задача другого workspace недостижима.
	res := callTool(t, sess, "get_issue", map[string]any{"ref": otherIssue}, nil)
	if !res.IsError {
		t.Fatalf("get_issue of foreign workspace succeeded, want error")
	}
	if code := errCode(t, res); code != "not_found" {
		t.Errorf("get_issue foreign code = %q, want not_found", code)
	}

	// search_issues возвращает только задачи workspace токена (0 своих).
	var search struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	res = callTool(t, sess, "search_issues", map[string]any{}, &search)
	if res.IsError {
		t.Fatalf("search_issues error: %s", textContent(t, res))
	}
	if len(search.Data) != 0 {
		t.Errorf("search_issues returned %d issues from token workspace, want 0", len(search.Data))
	}
}

// errCode достаёт доменный код из tool-ошибки.
func errCode(t *testing.T, res *mcp.CallToolResult) string {
	t.Helper()
	var body struct {
		Code string `json:"code"`
	}
	if err := json.Unmarshal([]byte(textContent(t, res)), &body); err != nil {
		t.Fatalf("decode error body: %v", err)
	}
	return body.Code
}
