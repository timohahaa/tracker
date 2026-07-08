package model

import "time"

// StatusGroup — группа статусов.
type StatusGroup string

const (
	StatusGroupBacklog    StatusGroup = "backlog"
	StatusGroupPlanned    StatusGroup = "planned"
	StatusGroupInProgress StatusGroup = "in_progress"
	StatusGroupCompleted  StatusGroup = "completed"
	StatusGroupCanceled   StatusGroup = "canceled"
)

// Status — статус задачи внутри команды (schema tracker.statuses).
type Status struct {
	ID        string      `db:"id"`
	TeamID    string      `db:"team_id"`
	Name      string      `db:"name"`
	Group     StatusGroup `db:"status_group"`
	Position  int         `db:"position"`
	CreatedAt time.Time   `db:"created_at"`
	UpdatedAt time.Time   `db:"updated_at"`
	DeletedAt *time.Time  `db:"deleted_at"`
}
