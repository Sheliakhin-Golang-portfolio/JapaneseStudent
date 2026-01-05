package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/japanesestudent/task-service/internal/models"
)

type scheduledTaskLogRepository struct {
	db *sql.DB
}

// NewScheduledTaskLogRepository creates a new scheduled task log repository
func NewScheduledTaskLogRepository(db *sql.DB) *scheduledTaskLogRepository {
	return &scheduledTaskLogRepository{db: db}
}

// Create inserts a new scheduled task log
func (r *scheduledTaskLogRepository) Create(ctx context.Context, log *models.ScheduledTaskLog) error {
	query := `
		INSERT INTO scheduled_task_logs (task_id, job_id, status, http_status, error)
		VALUES (?, ?, ?, ?, ?)
	`

	result, err := r.db.ExecContext(ctx, query, log.TaskID, log.JobID, log.Status, log.HTTPStatus, log.Error)
	if err != nil {
		return fmt.Errorf("failed to create scheduled task log: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}

	log.ID = int(id)
	return nil
}

// GetByID retrieves a scheduled task log by ID
func (r *scheduledTaskLogRepository) GetByID(ctx context.Context, id int) (*models.ScheduledTaskLog, error) {
	query := `
		SELECT id, task_id, job_id, status, http_status, error, created_at
		FROM scheduled_task_logs
		WHERE id = ?
		LIMIT 1
	`

	logEntry := &models.ScheduledTaskLog{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&logEntry.ID,
		&logEntry.TaskID,
		&logEntry.JobID,
		&logEntry.Status,
		&logEntry.HTTPStatus,
		&logEntry.Error,
		&logEntry.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("scheduled task log not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get scheduled task log by ID: %w", err)
	}

	return logEntry, nil
}

// GetAll retrieves a paginated list of scheduled task logs with optional filters
func (r *scheduledTaskLogRepository) GetAll(ctx context.Context, page, count, taskID int, jobID, status string) ([]models.ScheduledTaskLogListItem, error) {
	var whereConditions []string
	var args []any

	if taskID != 0 {
		whereConditions = append(whereConditions, "task_id = ?")
		args = append(args, taskID)
	}

	if jobID != "" {
		whereConditions = append(whereConditions, "job_id LIKE ?")
		args = append(args, "%"+jobID+"%")
	}

	if status != "" {
		whereConditions = append(whereConditions, "status = ?")
		args = append(args, status)
	}

	whereClause := ""
	if len(whereConditions) > 0 {
		whereClause = "WHERE " + strings.Join(whereConditions, " AND ")
	}

	offset := (page - 1) * count

	query := fmt.Sprintf(`
		SELECT id, task_id, job_id, status, created_at
		FROM scheduled_task_logs
		%s
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`, whereClause)

	args = append(args, count, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query scheduled task logs: %w", err)
	}
	defer rows.Close()

	var logs []models.ScheduledTaskLogListItem
	for rows.Next() {
		var logEntry models.ScheduledTaskLogListItem
		err := rows.Scan(&logEntry.ID, &logEntry.TaskID, &logEntry.JobID, &logEntry.Status, &logEntry.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan scheduled task log: %w", err)
		}
		logs = append(logs, logEntry)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return logs, nil
}
