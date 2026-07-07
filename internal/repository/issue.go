package repository

import "github.com/jmoiron/sqlx"

// IssueRepo — доступ к tracker.issues, issue_assignees, issue_labels.
// Read-операции (List с JOIN, фильтры) при росте выносятся в issue_read.go.
type IssueRepo struct {
	db *sqlx.DB
}
