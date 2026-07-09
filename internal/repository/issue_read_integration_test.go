//go:build integration

package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/timohahaa/tracker/internal/model"
	"github.com/timohahaa/tracker/internal/repository"
	"github.com/timohahaa/tracker/test/dbtest"
)

// uptr парсит id из seed-фабрики в *uuid.UUID для типизированных фильтров.
func uptr(s string) *uuid.UUID {
	u := uuid.MustParse(s)
	return &u
}

// listIDs выполняет List и возвращает id задач в порядке ответа.
func listIDs(t *testing.T, repos *repository.Repositories, ws string, filter model.IssueFilter, page model.Page) []string {
	t.Helper()
	rows, err := repos.Issue().List(context.Background(), ws, filter, page)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	ids := make([]string, 0, len(rows))
	for _, r := range rows {
		ids = append(ids, r.ID)
	}
	return ids
}

func hasID(ids []string, id string) bool {
	for _, x := range ids {
		if x == id {
			return true
		}
	}
	return false
}

func date(y int, m time.Month, d int) *time.Time {
	t := time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
	return &t
}

// bigPage — страница, вмещающая весь тестовый набор (сортировка по умолчанию).
func bigPage() model.Page { return model.Page{Limit: 50} }

func TestIssueRepo_ListFilters(t *testing.T) {
	t.Parallel()
	db := dbtest.Open(t)
	repos := repository.New(db)

	ws := dbtest.SeedWorkspace(t, db)
	teamID := dbtest.SeedTeam(t, db, ws)
	backlog := dbtest.SeedStatus(t, db, teamID, "Backlog", "backlog", 0)
	inProg := dbtest.SeedStatus(t, db, teamID, "In Progress", "in_progress", 0)

	userA := dbtest.SeedUser(t, db)
	userB := dbtest.SeedUser(t, db)

	labelX := dbtest.SeedLabel(t, db, ws, nil)
	labelY := dbtest.SeedLabel(t, db, ws, nil)

	var projID string
	if err := db.Get(&projID, `INSERT INTO tracker.project (workspace_id, name) VALUES ($1, 'Roadmap') RETURNING id`, ws); err != nil {
		t.Fatalf("seed project: %v", err)
	}
	t.Cleanup(func() { _, _ = db.Exec(`DELETE FROM tracker.project WHERE id = $1`, projID) })

	// i1: backlog, high, "Login bug"/"auth flow", due 2026-06-01, userA (primary), labelX, project.
	i1, _ := dbtest.SeedIssueDetailed(t, db, teamID, backlog, dbtest.IssueSeed{
		Title: "Login bug", Description: "auth flow", Priority: "high", DueDate: date(2026, 6, 1),
	})
	if _, err := db.Exec(`UPDATE tracker.issue SET project_id = $1 WHERE id = $2`, projID, i1); err != nil {
		t.Fatalf("set project: %v", err)
	}
	dbtest.SeedIssueAssignee(t, db, i1, userA, true)
	dbtest.SeedIssueLabel(t, db, i1, labelX)

	// i2: in_progress, low, "Signup page"/"onboarding", due 2026-07-01, userB, labelX+labelY.
	i2, _ := dbtest.SeedIssueDetailed(t, db, teamID, inProg, dbtest.IssueSeed{
		Title: "Signup page", Description: "onboarding", Priority: "low", DueDate: date(2026, 7, 1),
	})
	dbtest.SeedIssueAssignee(t, db, i2, userB, false)
	dbtest.SeedIssueLabel(t, db, i2, labelX)
	dbtest.SeedIssueLabel(t, db, i2, labelY)

	// i3: backlog, urgent, "Payment"/"stripe", без due/assignee/label.
	i3, _ := dbtest.SeedIssueDetailed(t, db, teamID, backlog, dbtest.IssueSeed{
		Title: "Payment", Description: "stripe", Priority: "urgent",
	})

	strptr := func(s string) *string { return &s }
	grp := func(g model.StatusGroup) *model.StatusGroup { return &g }
	pri := func(p model.Priority) *model.Priority { return &p }

	cases := []struct {
		name   string
		filter model.IssueFilter
		want   []string
	}{
		{"team", model.IssueFilter{TeamID: uptr(teamID)}, []string{i1, i2, i3}},
		{"status backlog", model.IssueFilter{StatusID: uptr(backlog)}, []string{i1, i3}},
		{"status_group in_progress", model.IssueFilter{StatusGroup: grp(model.StatusGroupInProgress)}, []string{i2}},
		{"assignee userA", model.IssueFilter{Assignee: uptr(userA)}, []string{i1}},
		{"assignee userB", model.IssueFilter{Assignee: uptr(userB)}, []string{i2}},
		{"responsible userA", model.IssueFilter{Responsible: uptr(userA)}, []string{i1}},
		{"responsible userB none", model.IssueFilter{Responsible: uptr(userB)}, nil},
		{"priority urgent", model.IssueFilter{Priority: pri(model.PriorityUrgent)}, []string{i3}},
		{"project", model.IssueFilter{ProjectID: uptr(projID)}, []string{i1}},
		{"label X", model.IssueFilter{LabelIDs: []uuid.UUID{uuid.MustParse(labelX)}}, []string{i1, i2}},
		{"label X+Y (all)", model.IssueFilter{LabelIDs: []uuid.UUID{uuid.MustParse(labelX), uuid.MustParse(labelY)}}, []string{i2}},
		{"due before 2026-06-15", model.IssueFilter{DueDateBefore: date(2026, 6, 15)}, []string{i1}},
		{"due after 2026-06-15", model.IssueFilter{DueDateAfter: date(2026, 6, 15)}, []string{i2}},
		{"q title", model.IssueFilter{Query: strptr("pay")}, []string{i3}},
		{"q description", model.IssueFilter{Query: strptr("onboarding")}, []string{i2}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := listIDs(t, repos, ws, tc.filter, bigPage())
			if len(got) != len(tc.want) {
				t.Fatalf("%s: got %d issues %v, want %d %v", tc.name, len(got), got, len(tc.want), tc.want)
			}
			for _, id := range tc.want {
				if !hasID(got, id) {
					t.Errorf("%s: missing %s in %v", tc.name, id, got)
				}
			}
		})
	}
}

func TestIssueRepo_ListSorts(t *testing.T) {
	t.Parallel()
	db := dbtest.Open(t)
	repos := repository.New(db)

	ws := dbtest.SeedWorkspace(t, db)
	teamID := dbtest.SeedTeam(t, db, ws)
	st := dbtest.SeedStatus(t, db, teamID, "Backlog", "backlog", 0)

	// Порядок вставки задаёт возрастание created_at и id (uuidv7 монотонен).
	i1, _ := dbtest.SeedIssueDetailed(t, db, teamID, st, dbtest.IssueSeed{Title: "one", Priority: "low"})
	i2, _ := dbtest.SeedIssueDetailed(t, db, teamID, st, dbtest.IssueSeed{Title: "two", Priority: "urgent"})
	i3, _ := dbtest.SeedIssueDetailed(t, db, teamID, st, dbtest.IssueSeed{Title: "three", Priority: "high"})

	equal := func(name string, got, want []string) {
		if len(got) != len(want) {
			t.Fatalf("%s: got %v, want %v", name, got, want)
		}
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("%s: got %v, want %v", name, got, want)
			}
		}
	}

	// created_at asc/desc.
	equal("created_at asc",
		listIDs(t, repos, ws, model.IssueFilter{}, model.Page{Limit: 50, Sort: "created_at", Order: model.SortAsc}),
		[]string{i1, i2, i3})
	equal("created_at desc",
		listIDs(t, repos, ws, model.IssueFilter{}, model.Page{Limit: 50, Sort: "created_at", Order: model.SortDesc}),
		[]string{i3, i2, i1})

	// updated_at asc (сразу после вставки == created_at, tiebreaker по id).
	equal("updated_at asc",
		listIDs(t, repos, ws, model.IssueFilter{}, model.Page{Limit: 50, Sort: "updated_at", Order: model.SortAsc}),
		[]string{i1, i2, i3})

	// priority desc: urgent(i2) > high(i3) > low(i1).
	equal("priority desc",
		listIDs(t, repos, ws, model.IssueFilter{}, model.Page{Limit: 50, Sort: "priority", Order: model.SortDesc}),
		[]string{i2, i3, i1})
	// priority asc: low(i1) < high(i3) < urgent(i2).
	equal("priority asc",
		listIDs(t, repos, ws, model.IssueFilter{}, model.Page{Limit: 50, Sort: "priority", Order: model.SortAsc}),
		[]string{i1, i3, i2})
}

func TestIssueRepo_ListKeysetNoDuplicatesOrGaps(t *testing.T) {
	t.Parallel()
	db := dbtest.Open(t)
	repos := repository.New(db)

	ws := dbtest.SeedWorkspace(t, db)
	teamID := dbtest.SeedTeam(t, db, ws)
	st := dbtest.SeedStatus(t, db, teamID, "Backlog", "backlog", 0)

	const n = 7
	full := make(map[string]bool, n)
	for i := 0; i < n; i++ {
		id, _ := dbtest.SeedIssueDetailed(t, db, teamID, st, dbtest.IssueSeed{Title: "keyset"})
		full[id] = true
	}

	const limit = 3
	seen := map[string]int{}
	var order []string
	var cursor *model.Cursor
	pages := 0
	for {
		pages++
		if pages > n+2 {
			t.Fatal("pagination did not terminate")
		}
		rows, err := repos.Issue().List(context.Background(), ws, model.IssueFilter{},
			model.Page{Limit: limit, Sort: "created_at", Order: model.SortAsc, Cursor: cursor})
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		hasMore := len(rows) > limit
		if hasMore {
			rows = rows[:limit]
		}
		for _, r := range rows {
			seen[r.ID]++
			order = append(order, r.ID)
		}
		if !hasMore {
			break
		}
		last := rows[len(rows)-1]
		cursor = &model.Cursor{Value: last.CreatedAt.Format(time.RFC3339Nano), ID: last.ID}
	}

	// Объединение страниц = полный набор ровно один раз (без дублей/пропусков).
	if len(order) != n {
		t.Fatalf("collected %d issues, want %d", len(order), n)
	}
	for id := range full {
		if seen[id] != 1 {
			t.Errorf("issue %s seen %d times, want 1", id, seen[id])
		}
	}
	for id, c := range seen {
		if !full[id] {
			t.Errorf("unexpected issue %s in results (seen %d)", id, c)
		}
	}
}
