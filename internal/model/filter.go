package model

import (
	"time"

	"github.com/google/uuid"
)

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
// ID-поля типизированы как uuid.UUID: разбор на HTTP-границе одновременно
// валидирует значение, поэтому в service/repository невалидный UUID недостижим.
type IssueFilter struct {
	TeamID        *uuid.UUID
	StatusID      *uuid.UUID
	StatusGroup   *StatusGroup
	Assignee      *uuid.UUID
	Responsible   *uuid.UUID
	Priority      *Priority
	ProjectID     *uuid.UUID
	LabelIDs      []uuid.UUID
	DueDateBefore *time.Time
	DueDateAfter  *time.Time
	Query         *string
}
