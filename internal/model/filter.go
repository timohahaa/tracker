package model

import "time"

// SortOrder — направление сортировки.
type SortOrder string

const (
	SortAsc  SortOrder = "asc"
	SortDesc SortOrder = "desc"
)

// Cursor — декодированное значение keyset-пагинации (sort value + id).
type Cursor struct {
	Value string
	ID    string
}

// Page — параметры пагинации, общие для repository и service.
type Page struct {
	Limit  int
	Sort   string
	Order  SortOrder
	Cursor *Cursor
}

// IssueFilter — фильтры списка задач (shared между service и repository).
type IssueFilter struct {
	TeamID        *string
	StatusID      *string
	StatusGroup   *StatusGroup
	Assignee      *string
	Responsible   *string
	Priority      *Priority
	ProjectID     *string
	LabelIDs      []string
	DueDateBefore *time.Time
	DueDateAfter  *time.Time
	Query         *string
}
