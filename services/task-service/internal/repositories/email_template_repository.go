package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/Sheliakhin-Golang-portfolio/JapaneseStudent/services/task-service/internal/models"
)

type emailTemplateRepository struct {
	db *sql.DB
}

// NewEmailTemplateRepository creates a new email template repository
func NewEmailTemplateRepository(db *sql.DB) *emailTemplateRepository {
	return &emailTemplateRepository{db: db}
}

// Create inserts a new email template
func (r *emailTemplateRepository) Create(ctx context.Context, template *models.EmailTemplate) error {
	query := `
		INSERT INTO email_templates (slug, subject_template, body_template)
		VALUES (?, ?, ?)
	`

	result, err := r.db.ExecContext(ctx, query, template.Slug, template.SubjectTemplate, template.BodyTemplate)
	if err != nil {
		return fmt.Errorf("failed to create email template: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}

	template.ID = int(id)
	return nil
}

// GetByID retrieves an email template by ID
func (r *emailTemplateRepository) GetByID(ctx context.Context, id int) (*models.EmailTemplate, error) {
	query := `
		SELECT id, slug, subject_template, body_template, created_at, updated_at
		FROM email_templates
		WHERE id = ?
		LIMIT 1
	`

	template := &models.EmailTemplate{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&template.ID,
		&template.Slug,
		&template.SubjectTemplate,
		&template.BodyTemplate,
		&template.CreatedAt,
		&template.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("email template not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get email template by ID: %w", err)
	}

	return template, nil
}

// GetTemplateByID retrieves an email template by ID
func (r *emailTemplateRepository) GetTemplateByID(ctx context.Context, id int) (*models.EmailTemplateParts, error) {
	query := `
		SELECT subject_template, body_template
		FROM email_templates
		WHERE id = ?
		LIMIT 1
	`
	parts := &models.EmailTemplateParts{}
	if err := r.db.QueryRowContext(ctx, query, id).Scan(
		&parts.SubjectTemplate,
		&parts.BodyTemplate,
	); err != nil {
		return nil, fmt.Errorf("failed to get email template parts by ID: %w", err)
	}
	return parts, nil
}

// GetIDBySlug retrieves an email template ID by slug
func (r *emailTemplateRepository) GetIDBySlug(ctx context.Context, slug string) (int, error) {
	query := "SELECT id FROM email_templates WHERE slug = ? LIMIT 1"

	var id int
	err := r.db.QueryRowContext(ctx, query, slug).Scan(&id)

	if err == sql.ErrNoRows {
		return 0, fmt.Errorf("email template not found")
	}
	if err != nil {
		return 0, fmt.Errorf("failed to get email template by slug: %w", err)
	}

	return id, nil
}

// GetAll retrieves a paginated list of email templates with optional search
func (r *emailTemplateRepository) GetAll(ctx context.Context, page, count int, search string) ([]models.EmailTemplateListItem, error) {
	var args []any
	whereClause := ""

	if search != "" {
		whereClause = "WHERE slug LIKE ?"
		args = append(args, "%"+search+"%")
	}

	offset := (page - 1) * count

	query := fmt.Sprintf(`
		SELECT id, slug
		FROM email_templates
		%s
		ORDER BY slug
		LIMIT ? OFFSET ?
	`, whereClause)

	args = append(args, count, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query email templates: %w", err)
	}
	defer rows.Close()

	var templates []models.EmailTemplateListItem
	for rows.Next() {
		var template models.EmailTemplateListItem
		err := rows.Scan(&template.ID, &template.Slug)
		if err != nil {
			return nil, fmt.Errorf("failed to scan email template: %w", err)
		}
		templates = append(templates, template)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return templates, nil
}

// Update updates an email template
func (r *emailTemplateRepository) Update(ctx context.Context, id int, template *models.EmailTemplate) error {
	setClauses := []string{}
	args := []any{}

	if template.Slug != "" {
		setClauses = append(setClauses, "slug = ?")
		args = append(args, template.Slug)
	}
	if template.SubjectTemplate != "" {
		setClauses = append(setClauses, "subject_template = ?")
		args = append(args, template.SubjectTemplate)
	}
	if template.BodyTemplate != "" {
		setClauses = append(setClauses, "body_template = ?")
		args = append(args, template.BodyTemplate)
	}

	if len(setClauses) == 0 {
		return nil // Nothing to update
	}

	args = append(args, id)
	query := fmt.Sprintf(`
		UPDATE email_templates
		SET %s
		WHERE id = ?
	`, strings.Join(setClauses, ", "))

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update email template: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("email template not found")
	}

	return nil
}

// Delete deletes an email template by ID
func (r *emailTemplateRepository) Delete(ctx context.Context, id int) error {
	query := `DELETE FROM email_templates WHERE id = ?`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete email template: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("email template not found")
	}

	return nil
}

// ExistsBySlug checks if an email template exists with the given slug
func (r *emailTemplateRepository) ExistsBySlug(ctx context.Context, slug string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM email_templates WHERE slug = ?)`

	var exists bool
	err := r.db.QueryRowContext(ctx, query, slug).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check slug existence: %w", err)
	}

	return exists, nil
}

// ExistsBySlug checks if an email template exists with the given slug
func (r *emailTemplateRepository) ExistsByID(ctx context.Context, id int) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM email_templates WHERE id = ?)`

	var exists bool
	err := r.db.QueryRowContext(ctx, query, id).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check ID existence: %w", err)
	}

	return exists, nil
}
