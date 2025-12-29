package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/japanesestudent/learn-service/internal/models"
)

type lessonRepository struct {
	db *sql.DB
}

// NewLessonRepository creates a new lesson repository
func NewLessonRepository(db *sql.DB) *lessonRepository {
	return &lessonRepository{
		db: db,
	}
}

// GetBySlug retrieves a lesson by its slug
func (r *lessonRepository) GetBySlug(ctx context.Context, slug string, userID int) (*models.LessonListItem, error) {
	query := `
		SELECT 
			l.id,
			l.course_id,
			l.title,
			l.short_summary,
			CASE WHEN luh.id IS NOT NULL THEN 1 ELSE 0 END as completed
		FROM lessons l
		LEFT JOIN lesson_user_history luh ON luh.lesson_id = l.id AND luh.user_id = ? AND luh.course_id = l.course_id
		WHERE l.slug = ?
		LIMIT 1
	`

	var lesson models.LessonListItem
	var completed int
	err := r.db.QueryRowContext(ctx, query, userID, slug).Scan(
		&lesson.ID,
		&lesson.CourseID,
		&lesson.Title,
		&lesson.ShortSummary,
		&completed,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("lesson not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get lesson by slug: %w", err)
	}

	lesson.Completed = completed == 1
	return &lesson, nil
}

// GetByID retrieves a lesson by its ID
func (r *lessonRepository) GetByID(ctx context.Context, id int) (*models.Lesson, error) {
	query := `
		SELECT id, slug, course_id, title, short_summary, ` + "`order`" + `
		FROM lessons
		WHERE id = ?
		LIMIT 1
	`

	var lesson models.Lesson
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&lesson.ID,
		&lesson.Slug,
		&lesson.CourseID,
		&lesson.Title,
		&lesson.ShortSummary,
		&lesson.Order,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("lesson not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get lesson by id: %w", err)
	}

	return &lesson, nil
}

// GetByCourseID retrieves all lessons for a course, sorted by order
func (r *lessonRepository) GetByCourseID(ctx context.Context, courseID int) ([]models.Lesson, error) {
	query := `
		SELECT id, slug, title, short_summary, ` + "`order`" + `
		FROM lessons
		WHERE course_id = ?
		ORDER BY ` + "`order`" + `
	`

	rows, err := r.db.QueryContext(ctx, query, courseID)
	if err != nil {
		return nil, fmt.Errorf("failed to query lessons: %w", err)
	}
	defer rows.Close()

	var lessons []models.Lesson
	for rows.Next() {
		var lesson models.Lesson
		err := rows.Scan(
			&lesson.ID,
			&lesson.Slug,
			&lesson.Title,
			&lesson.ShortSummary,
			&lesson.Order,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan lesson: %w", err)
		}
		lessons = append(lessons, lesson)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return lessons, nil
}

// GetByCourseIDWithCompletion retrieves lessons for a course with completion status for a user
func (r *lessonRepository) GetByCourseIDWithCompletion(ctx context.Context, courseID, userID int) ([]models.LessonListItem, error) {
	query := `
		SELECT 
			l.slug,
			l.title,
			l.` + "`order`" + `,
			CASE WHEN luh.id IS NOT NULL THEN 1 ELSE 0 END as completed
		FROM lessons l
		LEFT JOIN lesson_user_history luh ON luh.lesson_id = l.id AND luh.user_id = ? AND luh.course_id = ?
		WHERE l.course_id = ?
		ORDER BY l.` + "`order`" + `
	`

	rows, err := r.db.QueryContext(ctx, query, userID, courseID, courseID)
	if err != nil {
		return nil, fmt.Errorf("failed to query lessons: %w", err)
	}
	defer rows.Close()

	var lessons []models.LessonListItem
	for rows.Next() {
		var lesson models.LessonListItem
		var completed int
		err := rows.Scan(
			&lesson.Slug,
			&lesson.Title,
			&lesson.Order,
			&completed,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan lesson: %w", err)
		}
		lesson.Completed = completed == 1
		lessons = append(lessons, lesson)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return lessons, nil
}

// GetShortInfoByCourseID retrieves lessons with only ID and Title for select options
func (r *lessonRepository) GetShortInfoByCourseID(ctx context.Context, courseID *int) ([]models.LessonShortInfo, error) {
	var whereClause string
	var args []any
	if courseID != nil {
		whereClause = "WHERE course_id = ?"
		args = append(args, *courseID)
	}
	query := fmt.Sprintf(`
		SELECT id, title
		FROM lessons
		%s
		ORDER BY `+"`order`"+`
	`, whereClause)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query lessons: %w", err)
	}
	defer rows.Close()

	var lessons []models.LessonShortInfo
	for rows.Next() {
		var lesson models.LessonShortInfo
		err := rows.Scan(&lesson.ID, &lesson.Title)
		if err != nil {
			return nil, fmt.Errorf("failed to scan lesson: %w", err)
		}
		lessons = append(lessons, lesson)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return lessons, nil
}

// ExistsBySlug checks if a lesson with the given slug exists
func (r *lessonRepository) ExistsBySlug(ctx context.Context, slug string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM lessons WHERE slug = ?)`

	var exists bool
	err := r.db.QueryRowContext(ctx, query, slug).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check lesson existence: %w", err)
	}

	return exists, nil
}

// ExistsByTitleInCourse checks if a lesson with the given title exists in the course
func (r *lessonRepository) ExistsByTitleInCourse(ctx context.Context, courseID int, title string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM lessons WHERE course_id = ? AND title = ?)`

	var exists bool
	err := r.db.QueryRowContext(ctx, query, courseID, title).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check lesson title existence: %w", err)
	}

	return exists, nil
}

// ExistsByOrderInCourse checks if a lesson with the given order exists in the course
func (r *lessonRepository) ExistsByOrderInCourse(ctx context.Context, courseID int, order int) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM lessons WHERE course_id = ? AND ` + "`order`" + ` = ?)`

	var exists bool
	err := r.db.QueryRowContext(ctx, query, courseID, order).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check lesson order existence: %w", err)
	}

	return exists, nil
}

// IncrementOrderForLessons increments order for all lessons in a course with order >= given order
func (r *lessonRepository) IncrementOrderForLessons(ctx context.Context, courseID, order int) error {
	query := `
		UPDATE lessons
		SET ` + "`order`" + ` = ` + "`order`" + ` + 1
		WHERE course_id = ? AND ` + "`order`" + ` >= ?
	`

	_, err := r.db.ExecContext(ctx, query, courseID, order)
	if err != nil {
		return fmt.Errorf("failed to increment lesson order: %w", err)
	}

	return nil
}

// Create creates a new lesson
func (r *lessonRepository) Create(ctx context.Context, lesson *models.Lesson) error {
	query := `
		INSERT INTO lessons (slug, course_id, title, short_summary, ` + "`order`" + `)
		VALUES (?, ?, ?, ?, ?)
	`

	result, err := r.db.ExecContext(ctx, query,
		lesson.Slug,
		lesson.CourseID,
		lesson.Title,
		lesson.ShortSummary,
		lesson.Order,
	)
	if err != nil {
		return fmt.Errorf("failed to create lesson: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}

	lesson.ID = int(id)
	return nil
}

// Update updates a lesson (partial update)
func (r *lessonRepository) Update(ctx context.Context, lesson *models.Lesson) error {
	var setParts []string
	var args []any

	if lesson.Slug != "" {
		setParts = append(setParts, "slug = ?")
		args = append(args, lesson.Slug)
	}
	if lesson.CourseID != 0 {
		setParts = append(setParts, "course_id = ?")
		args = append(args, lesson.CourseID)
	}
	if lesson.Title != "" {
		setParts = append(setParts, "title = ?")
		args = append(args, lesson.Title)
	}
	if lesson.ShortSummary != "" {
		setParts = append(setParts, "short_summary = ?")
		args = append(args, lesson.ShortSummary)
	}
	if lesson.Order != 0 {
		setParts = append(setParts, "`order` = ?")
		args = append(args, lesson.Order)
	}

	if len(setParts) == 0 {
		return fmt.Errorf("no fields to update")
	}

	query := fmt.Sprintf(`
		UPDATE lessons
		SET %s
		WHERE id = ?
	`, strings.Join(setParts, ", "))

	args = append(args, lesson.ID)

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update lesson: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("lesson not found")
	}

	return nil
}

// Delete deletes a lesson by ID
func (r *lessonRepository) Delete(ctx context.Context, id int) error {
	query := `DELETE FROM lessons WHERE id = ?`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete lesson: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("lesson not found")
	}

	return nil
}

// CheckOwnership checks if a lesson belongs to a tutor
func (r *lessonRepository) CheckOwnership(ctx context.Context, id, tutorID int) (bool, error) {
	query := "SELECT EXISTS(SELECT 1 FROM lessons WHERE id = ? AND course_id IN (SELECT id FROM courses WHERE author_id = ?))"
	var exists bool
	err := r.db.QueryRowContext(ctx, query, id, tutorID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check lesson ownership: %w", err)
	}
	return exists, nil
}
