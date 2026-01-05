package models

import "time"

// ImmediateTaskStatus represents the status of an immediate task
type ImmediateTaskStatus string

const (
	ImmediateTaskStatusEnqueued  ImmediateTaskStatus = "Enqueued"
	ImmediateTaskStatusCompleted ImmediateTaskStatus = "Completed"
	ImmediateTaskStatusFailed    ImmediateTaskStatus = "Failed"
)

// ImmediateTask represents an immediate task
type ImmediateTask struct {
	ID         int                 `json:"id"`
	UserID     int                 `json:"user_id"`
	TemplateID *int                `json:"template_id,omitempty"`
	Content    string              `json:"content"`
	CreatedAt  time.Time           `json:"created_at"`
	Status     ImmediateTaskStatus `json:"status"`
	Error      string              `json:"error,omitempty"`
}

// CreateImmediateTaskRequest represents a request to create an immediate task
type CreateImmediateTaskRequest struct {
	UserID    int    `json:"user_id"`
	EmailSlug string `json:"email_slug"`
	Content   string `json:"content"`
}

// AdminCreateImmediateTaskRequest represents an admin request to create an immediate task
type AdminCreateImmediateTaskRequest struct {
	UserID     int    `json:"user_id"`
	TemplateID int    `json:"template_id"`
	Content    string `json:"content"`
}

// UpdateImmediateTaskRequest represents a request to update an immediate task
type UpdateImmediateTaskRequest struct {
	UserID     *int                `json:"user_id,omitempty"`
	TemplateID *int                `json:"template_id,omitempty"`
	Content    string              `json:"content,omitempty"`
	Status     ImmediateTaskStatus `json:"status,omitempty"`
	Error      string              `json:"error,omitempty"`
}

// ImmediateTaskListItem represents an immediate task in a list response
type ImmediateTaskListItem struct {
	ID         int                 `json:"id"`
	UserID     int                 `json:"user_id"`
	TemplateID *int                `json:"template_id,omitempty"`
	CreatedAt  time.Time           `json:"created_at"`
	Status     ImmediateTaskStatus `json:"status"`
}
