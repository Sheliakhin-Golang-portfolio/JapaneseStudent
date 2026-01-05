package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/hibiken/asynq"
	"github.com/japanesestudent/task-service/internal/models"
	"github.com/japanesestudent/task-service/internal/services"
	"go.uber.org/zap"
	"gopkg.in/mail.v2"
)

// EmailTemplateRepository defines the interface for email template repository
type EmailTemplateRepository interface {
	// GetTemplateByID retrieves an email body and subject by its ID
	//
	// "id" parameter is used to retrieve an email body and subject by its ID.
	//
	// If some error occurs during data retrieve, the error will be returned together with "nil" value.
	GetTemplateByID(ctx context.Context, id int) (*models.EmailTemplateParts, error)
}

// ImmediateTaskRepository defines the interface for immediate task repository
type ImmediateTaskRepository interface {
	// GetByID retrieves an immediate task by its ID
	//
	// "id" parameter is used to retrieve an immediate task by its ID.
	//
	// If some error occurs during data retrieve, the error will be returned together with "nil" value.
	GetByID(ctx context.Context, id int) (*models.ImmediateTask, error)
	// UpdateStatus updates the status of an immediate task
	//
	// "id" parameter is used to update the status of an immediate task.
	// "status" parameter is used to update the status of an immediate task.
	// "errorMessage" parameter is used to update the error message of an immediate task.
	//
	// If some error occurs during data update, the error will be returned.
	UpdateStatus(ctx context.Context, id int, status models.ImmediateTaskStatus, errorMessage string) error
}

// ScheduledTaskRepository defines the interface for scheduled task repository
type ScheduledTaskRepository interface {
	// GetByID retrieves a scheduled task by its ID
	//
	// "id" parameter is used to retrieve a scheduled task by its ID.
	//
	// If some error occurs during data retrieve, the error will be returned together with "nil" value.
	GetByID(ctx context.Context, id int) (*models.ScheduledTask, error)
	// UpdatePreviousRunAndNextRun updates the previous_run and next_run fields of a scheduled task
	//
	// "id" parameter is used to update the previous_run and next_run fields of a scheduled task.
	// "previousRun" parameter is used to update the previous_run field of a scheduled task.
	// "nextRun" parameter is used to update the next_run field of a scheduled task.
	//
	// If some error occurs during data update, the error will be returned.
	UpdatePreviousRunAndNextRun(ctx context.Context, id int, previousRun time.Time, nextRun time.Time) error
	// UpdateURL updates the URL field of a scheduled task
	//
	// "id" parameter is used to update the URL field of a scheduled task.
	// "url" parameter is used to update the URL field of a scheduled task.
	//
	// If some error occurs during data update, the error will be returned.
	UpdateURL(ctx context.Context, id int, url string) error
}

// ScheduledTaskLogRepository defines the interface for scheduled task log repository
type ScheduledTaskLogRepository interface {
	// Create inserts a new scheduled task log
	//
	// "log" parameter is used to insert a new scheduled task log.
	//
	// If some error occurs during data insert, the error will be returned.
	Create(ctx context.Context, log *models.ScheduledTaskLog) error
}

// Worker handles task processing
type Worker struct {
	logger            *zap.Logger
	immediateTaskRepo ImmediateTaskRepository
	scheduledTaskRepo ScheduledTaskRepository
	taskLogRepo       ScheduledTaskLogRepository
	emailTemplateRepo EmailTemplateRepository
	smtpHost          string
	smtpPort          int
	smtpUsername      string
	smtpPassword      string
	smtpFrom          string
}

// NewWorker creates a new worker instance
func NewWorker(
	logger *zap.Logger,
	immediateTaskRepo ImmediateTaskRepository,
	scheduledTaskRepo ScheduledTaskRepository,
	taskLogRepo ScheduledTaskLogRepository,
	emailTemplateRepo EmailTemplateRepository,
	smtpHost string,
	smtpPort int,
	smtpUsername, smtpPassword, smtpFrom string,
) *Worker {
	return &Worker{
		logger:            logger,
		immediateTaskRepo: immediateTaskRepo,
		scheduledTaskRepo: scheduledTaskRepo,
		taskLogRepo:       taskLogRepo,
		emailTemplateRepo: emailTemplateRepo,
		smtpHost:          smtpHost,
		smtpPort:          smtpPort,
		smtpUsername:      smtpUsername,
		smtpPassword:      smtpPassword,
		smtpFrom:          smtpFrom,
	}
}

// HandleImmediateTask handles immediate task processing
func (w *Worker) HandleImmediateTask(ctx context.Context, t *asynq.Task) error {
	// Parse task ID from payload
	taskIDStr := string(t.Payload())
	taskID := 0
	if _, err := fmt.Sscanf(taskIDStr, "%d", &taskID); err != nil {
		return fmt.Errorf("failed to parse task ID: %w", err)
	}

	// Get immediate task
	task, err := w.immediateTaskRepo.GetByID(ctx, taskID)
	if err != nil {
		// Task was deleted before processing, meaning we decided not to execute the task
		if err.Error() == "immediate task not found" {
			return nil
		}
		return err
	}

	// Check if template_id is set
	if task.TemplateID == nil {
		w.immediateTaskRepo.UpdateStatus(ctx, taskID, models.ImmediateTaskStatusFailed, "template_id is required")
		return fmt.Errorf("template_id is required")
	}

	parts, err := w.emailTemplateRepo.GetTemplateByID(ctx, *task.TemplateID)
	if err != nil {
		w.immediateTaskRepo.UpdateStatus(ctx, taskID, models.ImmediateTaskStatusFailed, err.Error())
		return err
	}

	// Split Content by ';' - first value is recipient email, rest are template variables
	contentParts := strings.Split(task.Content, ";")
	if len(contentParts) < 1 {
		w.immediateTaskRepo.UpdateStatus(ctx, taskID, models.ImmediateTaskStatusFailed, "content must contain at least recipient email")
		return fmt.Errorf("content must contain at least recipient email")
	}

	recipientEmail := strings.TrimSpace(contentParts[0])
	templateVars := contentParts[1:]

	// Replace template variables in body template
	bodyTemplate := parts.BodyTemplate
	for i, varValue := range templateVars {
		placeholder := fmt.Sprintf("{{%d}}", i+1)
		bodyTemplate = strings.ReplaceAll(bodyTemplate, placeholder, strings.TrimSpace(varValue))
	}

	// Send email
	if err := w.sendEmail(recipientEmail, parts.SubjectTemplate, bodyTemplate); err != nil {
		w.immediateTaskRepo.UpdateStatus(ctx, taskID, models.ImmediateTaskStatusFailed, err.Error())
		return err
	}

	// Update status to Completed
	if err := w.immediateTaskRepo.UpdateStatus(ctx, taskID, models.ImmediateTaskStatusCompleted, ""); err != nil {
		return err
	}

	w.logger.Info("Immediate task completed", zap.Int("task_id", taskID))
	return nil
}

// HandleScheduledTask handles scheduled task processing
func (w *Worker) HandleScheduledTask(ctx context.Context, t *asynq.Task) error {
	// Parse task ID from payload
	taskIDStr := string(t.Payload())
	taskID := 0
	if _, err := fmt.Sscanf(taskIDStr, "%d", &taskID); err != nil {
		return fmt.Errorf("failed to parse task ID: %w", err)
	}

	// Get scheduled task
	task, err := w.scheduledTaskRepo.GetByID(ctx, taskID)
	if err != nil {
		// Task was deleted before processing, meaning we decided not to execute the task
		if err.Error() == "scheduled task not found" {
			return nil
		}
		return err
	}

	// Get job ID from context (asynq provides this)
	jobID, _ := asynq.GetTaskID(ctx)

	// Defer previous_run and next_run update
	defer func() {
		now := time.Now()
		nextRun, err := services.CalculateNextRun(task.Cron, now)
		if err == nil {
			w.scheduledTaskRepo.UpdatePreviousRunAndNextRun(ctx, taskID, now, nextRun)
		}
	}()

	// If URL is provided and doesn't start with "completed:", make HTTP request
	if task.URL != "" && !strings.HasPrefix(task.URL, "completed:") {
		url := task.URL
		if task.UserID != nil {
			url = fmt.Sprintf("%s/%d", url, *task.UserID)
		}

		resp, err := http.Get(url)
		if err != nil {
			// Create log entry in goroutine
			go func() {
				logEntry := &models.ScheduledTaskLog{
					TaskID:     taskID,
					JobID:      jobID,
					Status:     models.ScheduledTaskLogStatusFailed,
					HTTPStatus: 400,
					Error:      err.Error(),
				}
				w.taskLogRepo.Create(ctx, logEntry)
			}()

			return fmt.Errorf("failed to make HTTP request: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			// Read response body for error message
			bodyBytes, _ := io.ReadAll(resp.Body)
			errorMsg := string(bodyBytes)

			// Create log entry in goroutine
			go func() {
				logEntry := &models.ScheduledTaskLog{
					TaskID:     taskID,
					JobID:      jobID,
					Status:     models.ScheduledTaskLogStatusFailed,
					HTTPStatus: resp.StatusCode,
					Error:      errorMsg,
				}
				w.taskLogRepo.Create(ctx, logEntry)
			}()

			return fmt.Errorf("HTTP request returned status %d", resp.StatusCode)
		}
		// Status is 200, update URL with "completed:" prefix
		completedURL := "completed:" + task.URL
		w.scheduledTaskRepo.UpdateURL(ctx, taskID, completedURL)
	}

	// If Template ID is not null, send email
	if task.TemplateID != nil {
		parts, err := w.emailTemplateRepo.GetTemplateByID(ctx, *task.TemplateID)
		if err != nil {
			// Create log entry in goroutine
			go func() {
				logEntry := &models.ScheduledTaskLog{
					TaskID:     taskID,
					JobID:      jobID,
					Status:     models.ScheduledTaskLogStatusFailed,
					HTTPStatus: 400,
					Error:      err.Error(),
				}
				w.taskLogRepo.Create(ctx, logEntry)
			}()

			return fmt.Errorf("failed to get email template: %w", err)
		}

		// Split Content by ';' - first value is recipient email, rest are template variables
		contentParts := strings.Split(task.Content, ";")
		if len(contentParts) < 1 {
			// Create log entry in goroutine
			go func() {
				logEntry := &models.ScheduledTaskLog{
					TaskID:     taskID,
					JobID:      jobID,
					Status:     models.ScheduledTaskLogStatusFailed,
					HTTPStatus: 400,
					Error:      "content must contain at least recipient email",
				}
				w.taskLogRepo.Create(ctx, logEntry)
			}()

			return fmt.Errorf("content must contain at least recipient email")
		}

		recipientEmail := strings.TrimSpace(contentParts[0])
		templateVars := contentParts[1:]

		// Replace template variables in body template
		body := parts.BodyTemplate
		for i, varValue := range templateVars {
			placeholder := fmt.Sprintf("{{%d}}", i+1)
			body = strings.ReplaceAll(body, placeholder, strings.TrimSpace(varValue))
		}

		// Send email
		if err := w.sendEmail(recipientEmail, parts.SubjectTemplate, body); err != nil {
			// Create log entry in goroutine
			go func() {
				logEntry := &models.ScheduledTaskLog{
					TaskID:     taskID,
					JobID:      jobID,
					Status:     models.ScheduledTaskLogStatusFailed,
					HTTPStatus: 400,
					Error:      err.Error(),
				}
				w.taskLogRepo.Create(ctx, logEntry)
			}()

			return err
		}
	}

	// Create log entry with Completed status
	logEntry := &models.ScheduledTaskLog{
		TaskID:     taskID,
		JobID:      jobID,
		Status:     models.ScheduledTaskLogStatusCompleted,
		HTTPStatus: 200,
		Error:      "",
	}
	w.taskLogRepo.Create(ctx, logEntry)

	// Remove "completed:" prefix from URL if present
	if after, ok := strings.CutPrefix(task.URL, "completed:"); ok {
		w.scheduledTaskRepo.UpdateURL(ctx, taskID, after)
	}

	w.logger.Info("Scheduled task completed", zap.Int("task_id", taskID))
	return nil
}

// sendEmail sends an email using gopkg.in/mail.v2
func (w *Worker) sendEmail(to, subject, body string) error {
	m := mail.NewMessage()
	m.SetHeader("From", w.smtpFrom)
	m.SetHeader("To", to)
	m.SetHeader("Subject", subject)
	m.SetBody("text/html", body)

	d := mail.NewDialer(w.smtpHost, w.smtpPort, w.smtpUsername, w.smtpPassword)
	if err := d.DialAndSend(m); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}
