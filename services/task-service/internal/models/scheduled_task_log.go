package models

import "time"

// ScheduledTaskLogStatus represents the status of a scheduled task log
type ScheduledTaskLogStatus string

const (
	ScheduledTaskLogStatusCompleted ScheduledTaskLogStatus = "Completed"
	ScheduledTaskLogStatusFailed    ScheduledTaskLogStatus = "Failed"
)

// ScheduledTaskLog represents a scheduled task log entry
type ScheduledTaskLog struct {
	ID         int                    `json:"id"`
	TaskID     int                    `json:"task_id"`
	JobID      string                 `json:"job_id"`
	Status     ScheduledTaskLogStatus `json:"status"`
	HTTPStatus int                    `json:"http_status"`
	Error      string                 `json:"error,omitempty"`
	CreatedAt  time.Time              `json:"created_at"`
}

// ScheduledTaskLogListItem represents a scheduled task log in a list response
type ScheduledTaskLogListItem struct {
	ID        int                    `json:"id"`
	TaskID    int                    `json:"task_id"`
	JobID     string                 `json:"job_id"`
	Status    ScheduledTaskLogStatus `json:"status"`
	CreatedAt time.Time              `json:"created_at"`
}
