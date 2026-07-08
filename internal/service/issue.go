package service

import "github.com/timohahaa/tracker/internal/repository"

// IssueService — CRUD задач, генерация identifier, assignees.
// Read-операции (List, фильтры) при росте выносятся в issue_queries.go.
type IssueService struct {
	repos repository.Store
}
