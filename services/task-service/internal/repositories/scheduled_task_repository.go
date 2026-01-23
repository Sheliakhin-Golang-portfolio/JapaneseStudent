package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/Sheliakhin-Golang-portfolio/JapaneseStudent/services/task-service/internal/models"
)

type scheduledTaskRepository struct {
	db *sql.DB
}

// NewScheduledTaskRepository creates a new scheduled task repository
func NewScheduledTaskRepository(db *sql.DB) *scheduledTaskRepository {
	return &scheduledTaskRepository{db: db}
}

// Create inserts a new scheduled task
func (r *scheduledTaskRepository) Create(ctx context.Context, task *models.ScheduledTask) error {
	query := `
		INSERT INTO scheduled_tasks (user_id, template_id, url, content, next_run, cron)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	result, err := r.db.ExecContext(ctx, query,
		task.UserID, task.TemplateID, task.URL, task.Content, task.NextRun, task.Cron)
	if err != nil {
		return fmt.Errorf("failed to create scheduled task: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}

	task.ID = int(id)
	return nil
}

// GetByID retrieves a scheduled task by ID
func (r *scheduledTaskRepository) GetByID(ctx context.Context, id int) (*models.ScheduledTask, error) {
	query := `
		SELECT id, user_id, template_id, url, content, created_at, next_run, previous_run, active, cron
		FROM scheduled_tasks
		WHERE id = ?
		LIMIT 1
	`

	task := &models.ScheduledTask{}

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&task.ID,
		&task.UserID,
		&task.TemplateID,
		&task.URL,
		&task.Content,
		&task.CreatedAt,
		&task.NextRun,
		&task.PreviousRun,
		&task.Active,
		&task.Cron,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("scheduled task not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get scheduled task by ID: %w", err)
	}

	return task, nil
}

// GetAll retrieves a paginated list of scheduled tasks with optional filters
func (r *scheduledTaskRepository) GetAll(ctx context.Context, page, count, userID, templateID int, active *bool) ([]models.ScheduledTaskListItem, error) {
	var whereConditions []string
	var args []any

	if userID != 0 {
		whereConditions = append(whereConditions, "user_id = ?")
		args = append(args, userID)
	}

	if templateID != 0 {
		whereConditions = append(whereConditions, "template_id = ?")
		args = append(args, templateID)
	}

	if active != nil {
		whereConditions = append(whereConditions, "active = ?")
		args = append(args, *active)
	}

	whereClause := ""
	if len(whereConditions) > 0 {
		whereClause = "WHERE " + strings.Join(whereConditions, " AND ")
	}

	offset := (page - 1) * count

	query := fmt.Sprintf(`
		SELECT id, user_id, template_id, created_at, url, active, next_run
		FROM scheduled_tasks
		%s
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`, whereClause)

	args = append(args, count, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query scheduled tasks: %w", err)
	}
	defer rows.Close()

	var tasks []models.ScheduledTaskListItem
	for rows.Next() {
		var task models.ScheduledTaskListItem
		err := rows.Scan(&task.ID, &task.UserID, &task.TemplateID, &task.CreatedAt, &task.URL, &task.Active, &task.NextRun)
		if err != nil {
			return nil, fmt.Errorf("failed to scan scheduled task: %w", err)
		}
		tasks = append(tasks, task)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return tasks, nil
}

// GetActiveTasksForRestore retrieves active tasks where next_run >= NOW() - 1 minute
func (r *scheduledTaskRepository) GetActiveTasksForRestore(ctx context.Context) ([]models.ScheduledTaskSetItem, error) {
	query := `
		SELECT id, next_run
		FROM scheduled_tasks
		WHERE active = TRUE AND next_run >= DATE_SUB(NOW(), INTERVAL 1 MINUTE)
		ORDER BY next_run
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query scheduled tasks for restore: %w", err)
	}
	defer rows.Close()

	var tasks []models.ScheduledTaskSetItem
	for rows.Next() {
		var task models.ScheduledTaskSetItem

		err := rows.Scan(
			&task.ID,
			&task.NextRun,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan scheduled task: %w", err)
		}

		tasks = append(tasks, task)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return tasks, nil
}

// GetActiveTasksForNext24Hours retrieves active tasks where next_run BETWEEN NOW() AND NOW() + 24 hours
func (r *scheduledTaskRepository) GetActiveTasksForNext24Hours(ctx context.Context) ([]models.ScheduledTaskSetItem, error) {
	query := `
		SELECT id, next_run
		FROM scheduled_tasks
		WHERE active = TRUE AND next_run >= NOW() AND next_run <= DATE_ADD(NOW(), INTERVAL 24 HOUR)
		ORDER BY next_run
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query scheduled tasks for next 24 hours: %w", err)
	}
	defer rows.Close()

	var tasks []models.ScheduledTaskSetItem
	for rows.Next() {
		var task models.ScheduledTaskSetItem

		err := rows.Scan(
			&task.ID,
			&task.NextRun,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan scheduled task: %w", err)
		}

		tasks = append(tasks, task)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return tasks, nil
}

// Update updates a scheduled task
//
// We use a separate request model to clearly define which fields are being nullified or ignored
func (r *scheduledTaskRepository) Update(ctx context.Context, id int, task *models.UpdateScheduledTaskRequest) error {
	setClauses := []string{}
	args := []any{}

	if task.UserID != nil {
		if *task.UserID == 0 {
			setClauses = append(setClauses, "user_id = NULL")
		} else {
			setClauses = append(setClauses, "user_id = ?")
			args = append(args, *task.UserID)
		}
	}
	if task.TemplateID != nil {
		if *task.TemplateID == 0 {
			setClauses = append(setClauses, "template_id = NULL")
		} else {
			setClauses = append(setClauses, "template_id = ?")
			args = append(args, *task.TemplateID)
		}
	}
	if task.URL != nil {
		setClauses = append(setClauses, "url = ?")
		args = append(args, *task.URL)
	}
	if task.Content != nil {
		setClauses = append(setClauses, "content = ?")
		args = append(args, *task.Content)
	}
	if task.NextRun != nil {
		setClauses = append(setClauses, "next_run = ?")
		args = append(args, *task.NextRun)
	}
	if task.Active != nil {
		setClauses = append(setClauses, "active = ?")
		args = append(args, *task.Active)
	}
	if task.Cron != "" {
		setClauses = append(setClauses, "cron = ?")
		args = append(args, task.Cron)
	}

	if len(setClauses) == 0 {
		return fmt.Errorf("no fields to update")
	}

	args = append(args, id)
	query := fmt.Sprintf(`
		UPDATE scheduled_tasks
		SET %s
		WHERE id = ?
	`, strings.Join(setClauses, ", "))

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update scheduled task: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("scheduled task not found")
	}

	return nil
}

// UpdatePreviousRunAndNextRun updates the previous_run and next_run fields of a scheduled task
func (r *scheduledTaskRepository) UpdatePreviousRunAndNextRun(ctx context.Context, id int, previousRun time.Time, nextRun time.Time) error {
	query := `UPDATE scheduled_tasks SET previous_run = ?, next_run = ? WHERE id = ?`
	result, err := r.db.ExecContext(ctx, query, previousRun, nextRun, id)
	if err != nil {
		return fmt.Errorf("failed to update previous_run and next_run: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("scheduled task not found")
	}

	return nil
}

// UpdateURL updates the URL field of a scheduled task
func (r *scheduledTaskRepository) UpdateURL(ctx context.Context, id int, url string) error {
	query := `UPDATE scheduled_tasks SET url = ? WHERE id = ?`

	result, err := r.db.ExecContext(ctx, query, url, id)
	if err != nil {
		return fmt.Errorf("failed to update url: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("scheduled task not found")
	}

	return nil
}

// Delete deletes a scheduled task by ID
func (r *scheduledTaskRepository) Delete(ctx context.Context, id int) error {
	query := `DELETE FROM scheduled_tasks WHERE id = ?`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete scheduled task: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("scheduled task not found")
	}

	return nil
}

// GetURLByID retrieves the URL of a scheduled task by ID
func (r *scheduledTaskRepository) GetURLByID(ctx context.Context, id int) (string, error) {
	query := `SELECT url FROM scheduled_tasks WHERE id = ?`

	var url string
	err := r.db.QueryRowContext(ctx, query, id).Scan(&url)
	if err != nil {
		return "", fmt.Errorf("failed to get url: %w", err)
	}

	return url, nil
}

// GetTemplateIDByID retrieves the TemplateID of a scheduled task by ID
func (r *scheduledTaskRepository) GetTemplateIDByID(ctx context.Context, id int) (*int, error) {
	query := `SELECT template_id FROM scheduled_tasks WHERE id = ?`

	var templateID *int
	if err := r.db.QueryRowContext(ctx, query, id).Scan(&templateID); err != nil {
		return nil, fmt.Errorf("failed to get template id: %w", err)
	}

	return templateID, nil
}

// GetContentByID retrieves the Content of a scheduled task by ID
func (r *scheduledTaskRepository) GetContentByID(ctx context.Context, id int) (string, error) {
	query := `SELECT content FROM scheduled_tasks WHERE id = ?`

	var content string
	err := r.db.QueryRowContext(ctx, query, id).Scan(&content)
	if err != nil {
		return "", fmt.Errorf("failed to get content: %w", err)
	}

	return content, nil
}

// ExistsByUserIDAndURL checks if an active scheduled task exists with the given user ID and URL
func (r *scheduledTaskRepository) ExistsByUserIDAndURL(ctx context.Context, userID int, url string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM scheduled_tasks WHERE user_id = ? AND url = ?)`

	var exists bool
	err := r.db.QueryRowContext(ctx, query, userID, url).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check task existence: %w", err)
	}

	return exists, nil
}

// DeleteByUserID deletes all scheduled tasks for a user and returns their IDs for Redis cleanup
func (r *scheduledTaskRepository) DeleteByUserID(ctx context.Context, userID int) ([]int, error) {
	// First, get all task IDs for this user (for Redis cleanup)
	selectQuery := `SELECT id FROM scheduled_tasks WHERE user_id = ?`
	rows, err := r.db.QueryContext(ctx, selectQuery, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query task IDs: %w", err)
	}
	defer rows.Close()

	var taskIDs []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("failed to scan task ID: %w", err)
		}
		taskIDs = append(taskIDs, id)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	// If no tasks found, return empty slice (not an error)
	if len(taskIDs) == 0 {
		return []int{}, nil
	}

	// Delete all tasks for this user
	deleteQuery := `DELETE FROM scheduled_tasks WHERE user_id = ?`
	_, err = r.db.ExecContext(ctx, deleteQuery, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to delete scheduled tasks: %w", err)
	}

	return taskIDs, nil
}
