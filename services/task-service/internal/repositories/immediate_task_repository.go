package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/Sheliakhin-Golang-portfolio/JapaneseStudent/services/task-service/internal/models"
)

type immediateTaskRepository struct {
	db *sql.DB
}

// NewImmediateTaskRepository creates a new immediate task repository
func NewImmediateTaskRepository(db *sql.DB) *immediateTaskRepository {
	return &immediateTaskRepository{db: db}
}

// Create inserts a new immediate task
func (r *immediateTaskRepository) Create(ctx context.Context, task *models.ImmediateTask) error {
	query := `
		INSERT INTO immediate_tasks (user_id, template_id, content)
		VALUES (?, ?, ?)
	`

	result, err := r.db.ExecContext(ctx, query, task.UserID, task.TemplateID, task.Content)
	if err != nil {
		return fmt.Errorf("failed to create immediate task: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}

	task.ID = int(id)
	return nil
}

// GetByID retrieves an immediate task by ID
func (r *immediateTaskRepository) GetByID(ctx context.Context, id int) (*models.ImmediateTask, error) {
	query := `
		SELECT id, user_id, template_id, content, created_at, ` + "`status`" + `, COALESCE(error, '')
		FROM immediate_tasks
		WHERE id = ?
		LIMIT 1
	`

	task := &models.ImmediateTask{}
	var templateID sql.NullInt64
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&task.ID,
		&task.UserID,
		&templateID,
		&task.Content,
		&task.CreatedAt,
		&task.Status,
		&task.Error,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("immediate task not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get immediate task by ID: %w", err)
	}

	if templateID.Valid {
		templateIDInt := int(templateID.Int64)
		task.TemplateID = &templateIDInt
	}

	return task, nil
}

// GetAll retrieves a paginated list of immediate tasks with optional filters
func (r *immediateTaskRepository) GetAll(ctx context.Context, page, count int, userID, templateID int, status string) ([]models.ImmediateTaskListItem, error) {
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

	if status != "" {
		whereConditions = append(whereConditions, "`status` = ?")
		args = append(args, status)
	}

	whereClause := ""
	if len(whereConditions) > 0 {
		whereClause = "WHERE " + strings.Join(whereConditions, " AND ")
	}

	offset := (page - 1) * count

	query := fmt.Sprintf(`
		SELECT id, user_id, template_id, created_at, `+"`status`"+`
		FROM immediate_tasks
		%s
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`, whereClause)

	args = append(args, count, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query immediate tasks: %w", err)
	}
	defer rows.Close()

	var tasks []models.ImmediateTaskListItem
	for rows.Next() {
		var task models.ImmediateTaskListItem
		var templateID sql.NullInt64
		err := rows.Scan(&task.ID, &task.UserID, &templateID, &task.CreatedAt, &task.Status)
		if err != nil {
			return nil, fmt.Errorf("failed to scan immediate task: %w", err)
		}
		if templateID.Valid {
			templateIDInt := int(templateID.Int64)
			task.TemplateID = &templateIDInt
		}
		tasks = append(tasks, task)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return tasks, nil
}

// Update updates an immediate task
func (r *immediateTaskRepository) Update(ctx context.Context, task *models.ImmediateTask) error {
	setClauses := []string{}
	args := []any{}

	if task.UserID != 0 {
		setClauses = append(setClauses, "user_id = ?")
		args = append(args, task.UserID)
	}
	if task.TemplateID != nil {
		if *task.TemplateID == 0 {
			setClauses = append(setClauses, "template_id = NULL")
		} else {
			setClauses = append(setClauses, "template_id = ?")
			args = append(args, task.TemplateID)
		}
	}
	if task.Content != "" {
		setClauses = append(setClauses, "content = ?")
		args = append(args, task.Content)
	}
	if task.Status != "" {
		setClauses = append(setClauses, "`status` = ?")
		args = append(args, task.Status)
	}
	if task.Error != "" {
		setClauses = append(setClauses, "error = ?")
		args = append(args, task.Error)
	}

	if len(setClauses) == 0 {
		return nil // Nothing to update
	}

	args = append(args, task.ID)
	query := fmt.Sprintf(`
		UPDATE immediate_tasks
		SET %s
		WHERE id = ?
	`, strings.Join(setClauses, ", "))

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update immediate task: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("immediate task not found")
	}

	return nil
}

// UpdateStatus updates the status and error of an immediate task
func (r *immediateTaskRepository) UpdateStatus(ctx context.Context, id int, status models.ImmediateTaskStatus, errorMsg string) error {
	query := `
		UPDATE immediate_tasks
		SET ` + "`status`" + ` = ?, error = ?
		WHERE id = ?
	`

	result, err := r.db.ExecContext(ctx, query, status, errorMsg, id)
	if err != nil {
		return fmt.Errorf("failed to update immediate task status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("immediate task not found")
	}

	return nil
}

// Delete deletes an immediate task by ID
func (r *immediateTaskRepository) Delete(ctx context.Context, id int) error {
	query := `DELETE FROM immediate_tasks WHERE id = ?`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete immediate task: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("immediate task not found")
	}

	return nil
}
