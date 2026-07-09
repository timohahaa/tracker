package service

import (
	"context"
	"strconv"
	"time"

	"github.com/timohahaa/tracker/internal/errs"
	"github.com/timohahaa/tracker/internal/model"
)

// Дефолты и кэп пагинации списка задач.
const (
	DefaultIssueLimit = 50
	MaxIssueLimit     = 100
)

// cursorTimeLayout — формат таймстампа в курсоре. БД хранит timestamptz с
// микросекундной точностью, поэтому round-trip через строку стабилен.
const cursorTimeLayout = time.RFC3339Nano

// allowedSorts — поля сортировки списка задач (docs/tech/api.md).
var allowedSorts = map[string]bool{
	"created_at": true,
	"updated_at": true,
	"priority":   true,
}

// IssueListResult — страница списка задач с признаком продолжения и курсором на
// следующую страницу. NextCursor != nil только когда HasMore. Кодирование
// курсора в строку — на HTTP-границе.
type IssueListResult struct {
	Issues     []model.IssueDetails
	HasMore    bool
	NextCursor *model.Cursor
}

// ListIssues возвращает страницу задач workspace по фильтру. Нормализует page
// (дефолты sort=updated_at/order=desc/limit + кэп), валидирует sort/order,
// затем выбирает limit+1 строк у репозитория и по «лишней» вычисляет
// has_more/next_cursor. ID-фильтры уже провалидированы на HTTP-границе
// (типизированы как uuid.UUID). Доступно участнику workspace (гарантируется
// middleware LoadWorkspace).
func (s *IssueService) ListIssues(ctx context.Context, workspaceID string, filter model.IssueFilter, page model.Page) (IssueListResult, error) {
	page, err := normalizePage(page)
	if err != nil {
		return IssueListResult{}, err
	}

	rows, err := s.repos.Issue().List(ctx, workspaceID, filter, page)
	if err != nil {
		return IssueListResult{}, err
	}

	res := IssueListResult{Issues: rows}
	// Репозиторий выбирает limit+1: наличие «лишней» строки означает, что есть
	// следующая страница. Её отбрасываем, а по последнему оставленному элементу
	// строим курсор.
	if len(rows) > page.Limit {
		res.Issues = rows[:page.Limit]
		res.HasMore = true
		last := res.Issues[len(res.Issues)-1]
		res.NextCursor = &model.Cursor{Value: sortValue(last, page.Sort), ID: last.ID}
	}

	// Обогащаем assignees батчем (WHERE issue_id = ANY(...)) только для строк
	// возвращаемой страницы — без N+1.
	if err := s.enrichListAssignees(ctx, res.Issues); err != nil {
		return IssueListResult{}, err
	}
	return res, nil
}

// enrichListAssignees догружает активных участников для набора задач одним
// запросом и раскладывает их по строкам.
func (s *IssueService) enrichListAssignees(ctx context.Context, issues []model.IssueDetails) error {
	if len(issues) == 0 {
		return nil
	}
	ids := make([]string, len(issues))
	for i := range issues {
		ids[i] = issues[i].ID
	}
	byIssue, err := s.repos.Issue().ListAssigneesByIssues(ctx, ids)
	if err != nil {
		return err
	}
	for i := range issues {
		issues[i].Assignees = byIssue[issues[i].ID]
	}
	return nil
}

// normalizePage применяет дефолты и кэп к параметрам пагинации и валидирует
// sort/order. Невалидные sort/order → errs.Validation.
func normalizePage(page model.Page) (model.Page, error) {
	if page.Limit <= 0 {
		page.Limit = DefaultIssueLimit
	}
	if page.Limit > MaxIssueLimit {
		page.Limit = MaxIssueLimit
	}
	if page.Sort == "" {
		page.Sort = "updated_at"
	}
	if !allowedSorts[page.Sort] {
		return model.Page{}, errs.Validation("invalid sort", errs.FieldError{
			Field:   "sort",
			Reason:  "invalid",
			Message: "must be one of created_at, updated_at, priority",
		})
	}
	if page.Order == "" {
		page.Order = model.SortDesc
	}
	if page.Order != model.SortAsc && page.Order != model.SortDesc {
		return model.Page{}, errs.Validation("invalid order", errs.FieldError{
			Field:   "order",
			Reason:  "invalid",
			Message: "must be one of asc, desc",
		})
	}
	return page, nil
}

// sortValue возвращает значение поля сортировки последнего элемента для курсора.
// Для priority — ранг CASE (int-строка), для дат — RFC3339Nano-таймстамп.
func sortValue(d model.IssueDetails, sort string) string {
	switch sort {
	case "priority":
		return strconv.Itoa(priorityRank(d.Priority))
	case "created_at":
		return d.CreatedAt.Format(cursorTimeLayout)
	default: // updated_at
		return d.UpdatedAt.Format(cursorTimeLayout)
	}
}

// priorityRank ранжирует приоритет (urgent > high > medium > low > none).
// Должно совпадать с priorityRankSQL в репозитории.
func priorityRank(p model.Priority) int {
	switch p {
	case model.PriorityUrgent:
		return 4
	case model.PriorityHigh:
		return 3
	case model.PriorityMedium:
		return 2
	case model.PriorityLow:
		return 1
	default:
		return 0
	}
}
