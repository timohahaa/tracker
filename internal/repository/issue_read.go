package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"

	"github.com/timohahaa/tracker/internal/errs"
	"github.com/timohahaa/tracker/internal/model"
)

// priorityRankSQL — ранжирование приоритета для сортировки (urgent > high >
// medium > low > none). Должно совпадать с priorityRank в service.
const priorityRankSQL = `CASE i.priority
	WHEN 'urgent' THEN 4
	WHEN 'high' THEN 3
	WHEN 'medium' THEN 2
	WHEN 'low' THEN 1
	ELSE 0 END`

// List возвращает задачи workspace по фильтру с keyset-пагинацией. Строит
// динамический WHERE по заданным (не-nil) полям фильтра, применяет keyset-условие
// по (sort, id) при наличии курсора и упорядочивает по (sort, id) в направлении
// order. Выбирается page.Limit+1 строк — сервис по «лишней» строке вычисляет
// has_more. page.Sort/Order считаются уже нормализованными сервисом.
func (r *IssueRepo) List(ctx context.Context, workspaceID string, filter model.IssueFilter, page model.Page) ([]model.IssueDetails, error) {
	// b накапливает позиционные аргументы; next(v) регистрирует аргумент и
	// возвращает его плейсхолдер ($1, $2, ...).
	var args []any
	next := func(v any) string {
		args = append(args, v)
		return fmt.Sprintf("$%d", len(args))
	}

	where := []string{
		"i.deleted_at IS NULL",
		"t.deleted_at IS NULL",
		"t.workspace_id = " + next(workspaceID),
	}

	if filter.TeamID != nil {
		where = append(where, "i.team_id = "+next(*filter.TeamID)+"::uuid")
	}
	if filter.StatusID != nil {
		where = append(where, "i.status_id = "+next(*filter.StatusID)+"::uuid")
	}
	if filter.StatusGroup != nil {
		where = append(where, "EXISTS (SELECT 1 FROM tracker.status s WHERE s.id = i.status_id AND s.status_group = "+next(string(*filter.StatusGroup))+" AND s.deleted_at IS NULL)")
	}
	if filter.Assignee != nil {
		where = append(where, "EXISTS (SELECT 1 FROM tracker.issue_assignee ia WHERE ia.issue_id = i.id AND ia.user_id = "+next(*filter.Assignee)+"::uuid AND ia.deleted_at IS NULL)")
	}
	if filter.Responsible != nil {
		where = append(where, "EXISTS (SELECT 1 FROM tracker.issue_assignee ia WHERE ia.issue_id = i.id AND ia.user_id = "+next(*filter.Responsible)+"::uuid AND ia.is_primary AND ia.deleted_at IS NULL)")
	}
	if filter.Priority != nil {
		where = append(where, "i.priority = "+next(string(*filter.Priority)))
	}
	if filter.ProjectID != nil {
		where = append(where, "i.project_id = "+next(*filter.ProjectID)+"::uuid")
	}
	if len(filter.LabelIDs) > 0 {
		// pq.Array надёжно сериализует []string, поэтому UUID приводим к строкам.
		labelIDs := make([]string, len(filter.LabelIDs))
		for i, id := range filter.LabelIDs {
			labelIDs[i] = id.String()
		}
		// Задачи, у которых есть ВСЕ указанные метки (repeatable label-фильтр).
		where = append(where, "i.id IN (SELECT issue_id FROM tracker.issue_label WHERE label_id = ANY("+next(pq.Array(labelIDs))+"::uuid[]) AND deleted_at IS NULL GROUP BY issue_id HAVING count(DISTINCT label_id) = "+next(len(filter.LabelIDs))+")")
	}
	if filter.DueDateBefore != nil {
		where = append(where, "i.due_date <= "+next(*filter.DueDateBefore))
	}
	if filter.DueDateAfter != nil {
		where = append(where, "i.due_date >= "+next(*filter.DueDateAfter))
	}
	if filter.Query != nil {
		like := "%" + *filter.Query + "%"
		p := next(like)
		where = append(where, "(i.title ILIKE "+p+" OR i.description ILIKE "+p+")")
	}

	// Сортировка: выражение + приведение курсора к типу выражения.
	var sortExpr, cursorCast string
	switch page.Sort {
	case "created_at":
		sortExpr, cursorCast = "i.created_at", "::timestamptz"
	case "priority":
		sortExpr, cursorCast = priorityRankSQL, "::int"
	default: // updated_at
		sortExpr, cursorCast = "i.updated_at", "::timestamptz"
	}

	dir, op := "DESC", "<"
	if page.Order == model.SortAsc {
		dir, op = "ASC", ">"
	}

	if page.Cursor != nil {
		where = append(where, fmt.Sprintf("(%s, i.id) %s (%s%s, %s::uuid)",
			sortExpr, op, next(page.Cursor.Value), cursorCast, next(page.Cursor.ID)))
	}

	limit := next(page.Limit + 1)

	q := `
		SELECT` + issueSelectColumns + `
		FROM tracker.issue i
		JOIN org.team t ON t.id = i.team_id
		WHERE ` + strings.Join(where, "\n\t\t\t AND ") + `
		ORDER BY ` + sortExpr + ` ` + dir + `, i.id ` + dir + `
		LIMIT ` + limit

	var out []model.IssueDetails
	if err := sqlx.SelectContext(ctx, r.db, &out, q, args...); err != nil {
		return nil, errs.DB(err)
	}
	return out, nil
}
