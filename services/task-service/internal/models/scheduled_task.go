package models

import "time"

// ScheduledTask represents a scheduled task
type ScheduledTask struct {
	ID          int        `json:"id"`
	UserID      *int       `json:"user_id,omitempty"`
	TemplateID  *int       `json:"template_id,omitempty"`
	URL         string     `json:"url,omitempty"`
	Content     string     `json:"content"`
	CreatedAt   time.Time  `json:"created_at"`
	NextRun     time.Time  `json:"next_run"`
	PreviousRun *time.Time `json:"previous_run,omitempty"`
	Active      bool       `json:"active"`
	Cron        string     `json:"cron"`
}

// CreateScheduledTaskRequest represents a request to create a scheduled task
type CreateScheduledTaskRequest struct {
	UserID    *int   `json:"user_id,omitempty"`
	EmailSlug string `json:"email_slug"` // empty means no template
	URL       string `json:"url"`        // empty means no URL
	Content   string `json:"content"`
	Cron      string `json:"cron"`
}

// AdminCreateScheduledTaskRequest represents an admin request to create a scheduled task
type AdminCreateScheduledTaskRequest struct {
	UserID     *int   `json:"user_id,omitempty"`
	TemplateID *int   `json:"template_id,omitempty"`
	URL        string `json:"url,omitempty"`
	Content    string `json:"content,omitempty"`
	Cron       string `json:"cron"`
}

// UpdateScheduledTaskRequest represents a request to update a scheduled task
type UpdateScheduledTaskRequest struct {
	UserID     *int       `json:"user_id,omitempty"`
	TemplateID *int       `json:"template_id,omitempty"`
	URL        *string    `json:"url,omitempty"`
	Content    *string    `json:"content,omitempty"`
	NextRun    *time.Time `json:"next_run,omitempty"`
	Active     *bool      `json:"active,omitempty"`
	Cron       string     `json:"cron,omitempty"`
}

// ScheduledTaskListItem represents a scheduled task in a list response
type ScheduledTaskListItem struct {
	ID         int       `json:"id"`
	UserID     *int      `json:"user_id,omitempty"`
	TemplateID *int      `json:"template_id,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	URL        string    `json:"url,omitempty"`
	Active     bool      `json:"active"`
	NextRun    time.Time `json:"next_run"`
}

// ScheduledTaskSetItem represents a scheduled task in the Redis set
type ScheduledTaskSetItem struct {
	ID      int
	NextRun time.Time
}
