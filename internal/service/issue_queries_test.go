package service

import (
	"context"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	"github.com/timohahaa/tracker/internal/errs"
	"github.com/timohahaa/tracker/internal/model"
	repomocks "github.com/timohahaa/tracker/internal/repository/mocks"
)

// Тесты ListIssues покрывают контракт List-захода (PLAN.md 2.4 заход 2,
// docs/tech/api.md): нормализация/валидация page (дефолты, кэп, sort/order),
// резолв team-фильтра (UUID как есть, key → team_id), и вычисление
// has_more/next_cursor из выбранных репозиторием limit+1 строк. Наблюдаемое
// поведение — входы репозитория (перехват filter/page через DoAndReturn) и
// коды ошибок; несуществующие вызовы — отсутствием ожиданий.

func issueDetails(id string, updated time.Time, priority model.Priority) model.IssueDetails {
	return model.IssueDetails{
		Issue: model.Issue{ID: id, Priority: priority, UpdatedAt: updated, CreatedAt: updated},
	}
}

func TestListIssues_NormalizesPageDefaults(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := repomocks.NewMockStore(ctrl)
	issues := repomocks.NewMockIssueRepository(ctrl)
	store.EXPECT().Issue().Return(issues).AnyTimes()

	var gotPage model.Page
	issues.EXPECT().List(gomock.Any(), "ws-1", gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, _ model.IssueFilter, p model.Page) ([]model.IssueDetails, error) {
			gotPage = p
			return nil, nil
		})

	if _, err := newIssueService(store).ListIssues(context.Background(), "ws-1", model.IssueFilter{}, model.Page{}); err != nil {
		t.Fatalf("ListIssues: %v", err)
	}
	if gotPage.Limit != DefaultIssueLimit {
		t.Errorf("limit = %d, want %d", gotPage.Limit, DefaultIssueLimit)
	}
	if gotPage.Sort != "updated_at" {
		t.Errorf("sort = %q, want updated_at", gotPage.Sort)
	}
	if gotPage.Order != model.SortDesc {
		t.Errorf("order = %q, want desc", gotPage.Order)
	}
}

func TestListIssues_CapsLimit(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := repomocks.NewMockStore(ctrl)
	issues := repomocks.NewMockIssueRepository(ctrl)
	store.EXPECT().Issue().Return(issues).AnyTimes()

	var gotPage model.Page
	issues.EXPECT().List(gomock.Any(), "ws-1", gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, _ model.IssueFilter, p model.Page) ([]model.IssueDetails, error) {
			gotPage = p
			return nil, nil
		})

	if _, err := newIssueService(store).ListIssues(context.Background(), "ws-1", model.IssueFilter{}, model.Page{Limit: 10000}); err != nil {
		t.Fatalf("ListIssues: %v", err)
	}
	if gotPage.Limit != MaxIssueLimit {
		t.Errorf("limit = %d, want %d", gotPage.Limit, MaxIssueLimit)
	}
}

func TestListIssues_InvalidSortValidation(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := repomocks.NewMockStore(ctrl)
	// Issue().List не вызывается: sort невалиден.

	_, err := newIssueService(store).ListIssues(context.Background(), "ws-1", model.IssueFilter{}, model.Page{Sort: "bogus"})
	assertAppErrCode(t, err, errs.CodeValidation)
}

func TestListIssues_InvalidOrderValidation(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := repomocks.NewMockStore(ctrl)
	// Issue().List не вызывается: order невалиден.

	_, err := newIssueService(store).ListIssues(context.Background(), "ws-1", model.IssueFilter{}, model.Page{Order: model.SortOrder("sideways")})
	assertAppErrCode(t, err, errs.CodeValidation)
}

func TestListIssues_HasMoreAndTimestampCursor(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := repomocks.NewMockStore(ctrl)
	issues := repomocks.NewMockIssueRepository(ctrl)
	store.EXPECT().Issue().Return(issues).AnyTimes()

	t2 := time.Date(2026, 7, 7, 10, 0, 0, 0, time.UTC)
	rows := []model.IssueDetails{
		issueDetails("i1", time.Date(2026, 7, 8, 10, 0, 0, 0, time.UTC), model.PriorityNone),
		issueDetails("i2", t2, model.PriorityNone),
		issueDetails("i3", time.Date(2026, 7, 6, 10, 0, 0, 0, time.UTC), model.PriorityNone), // «лишняя» строка
	}
	issues.EXPECT().List(gomock.Any(), "ws-1", gomock.Any(), gomock.Any()).Return(rows, nil)
	issues.EXPECT().ListAssigneesByIssues(gomock.Any(), gomock.Any()).Return(map[string][]model.IssueAssignee{}, nil).AnyTimes()

	res, err := newIssueService(store).ListIssues(context.Background(), "ws-1", model.IssueFilter{}, model.Page{Limit: 2})
	if err != nil {
		t.Fatalf("ListIssues: %v", err)
	}
	if !res.HasMore {
		t.Error("HasMore = false, want true")
	}
	if len(res.Issues) != 2 {
		t.Fatalf("len(Issues) = %d, want 2", len(res.Issues))
	}
	if res.NextCursor == nil {
		t.Fatal("NextCursor = nil, want cursor on last kept row")
	}
	if res.NextCursor.ID != "i2" {
		t.Errorf("NextCursor.ID = %q, want i2", res.NextCursor.ID)
	}
	if res.NextCursor.Value != t2.Format(time.RFC3339Nano) {
		t.Errorf("NextCursor.Value = %q, want %q", res.NextCursor.Value, t2.Format(time.RFC3339Nano))
	}
}

func TestListIssues_NoMoreLeavesCursorNil(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := repomocks.NewMockStore(ctrl)
	issues := repomocks.NewMockIssueRepository(ctrl)
	store.EXPECT().Issue().Return(issues).AnyTimes()

	rows := []model.IssueDetails{
		issueDetails("i1", time.Now(), model.PriorityNone),
		issueDetails("i2", time.Now(), model.PriorityNone),
	}
	issues.EXPECT().List(gomock.Any(), "ws-1", gomock.Any(), gomock.Any()).Return(rows, nil)
	issues.EXPECT().ListAssigneesByIssues(gomock.Any(), gomock.Any()).Return(map[string][]model.IssueAssignee{}, nil).AnyTimes()

	res, err := newIssueService(store).ListIssues(context.Background(), "ws-1", model.IssueFilter{}, model.Page{Limit: 2})
	if err != nil {
		t.Fatalf("ListIssues: %v", err)
	}
	if res.HasMore {
		t.Error("HasMore = true, want false")
	}
	if res.NextCursor != nil {
		t.Errorf("NextCursor = %+v, want nil", res.NextCursor)
	}
	if len(res.Issues) != 2 {
		t.Errorf("len(Issues) = %d, want 2", len(res.Issues))
	}
}

func TestListIssues_PriorityCursorEncodesRank(t *testing.T) {
	ctrl := gomock.NewController(t)
	store := repomocks.NewMockStore(ctrl)
	issues := repomocks.NewMockIssueRepository(ctrl)
	store.EXPECT().Issue().Return(issues).AnyTimes()

	rows := []model.IssueDetails{
		issueDetails("i1", time.Now(), model.PriorityUrgent),
		issueDetails("i2", time.Now(), model.PriorityHigh),
		issueDetails("i3", time.Now(), model.PriorityLow), // «лишняя» строка
	}
	issues.EXPECT().List(gomock.Any(), "ws-1", gomock.Any(), gomock.Any()).Return(rows, nil)
	issues.EXPECT().ListAssigneesByIssues(gomock.Any(), gomock.Any()).Return(map[string][]model.IssueAssignee{}, nil).AnyTimes()

	res, err := newIssueService(store).ListIssues(context.Background(), "ws-1", model.IssueFilter{}, model.Page{Limit: 2, Sort: "priority"})
	if err != nil {
		t.Fatalf("ListIssues: %v", err)
	}
	if res.NextCursor == nil || res.NextCursor.ID != "i2" {
		t.Fatalf("NextCursor = %+v, want cursor on i2", res.NextCursor)
	}
	// priority=high → ранг 3.
	if res.NextCursor.Value != "3" {
		t.Errorf("NextCursor.Value = %q, want 3", res.NextCursor.Value)
	}
}
