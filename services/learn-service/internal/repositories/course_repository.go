package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/Sheliakhin-Golang-portfolio/JapaneseStudent/learn-service/internal/models"
)

type courseRepository struct {
	db *sql.DB
}

// NewCourseRepository creates a new course repository
func NewCourseRepository(db *sql.DB) *courseRepository {
	return &courseRepository{
		db: db,
	}
}

// GetBySlug retrieves a course by its slug
func (r *courseRepository) GetBySlug(ctx context.Context, slug string, userID int) (*models.CourseDetailResponse, error) {
	query := `
		SELECT 
			c.id,
			c.short_summary,
			c.title,
			c.complexity_level,
			COUNT(DISTINCT l.id) as total_lessons,
			COUNT(DISTINCT luh.lesson_id) as completed_lessons
		FROM courses c
		LEFT JOIN lessons l ON l.course_id = c.id
		LEFT JOIN lesson_user_history luh ON luh.course_id = c.id AND luh.user_id = ? AND luh.lesson_id = l.id
		WHERE c.slug = ?
		GROUP BY c.id, c.slug, c.title, c.complexity_level
		LIMIT 1
	`

	var course models.CourseDetailResponse
	err := r.db.QueryRowContext(ctx, query, userID, slug).Scan(
		&course.ID,
		&course.ShortSummary,
		&course.Title,
		&course.ComplexityLevel,
		&course.TotalLessons,
		&course.CompletedLessons,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("course not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get course by slug: %w", err)
	}

	return &course, nil
}

// GetByID retrieves a course by its ID
func (r *courseRepository) GetByID(ctx context.Context, id int) (*models.Course, error) {
	query := `
		SELECT id, slug, author_id, title, short_summary, complexity_level
		FROM courses
		WHERE id = ?
		LIMIT 1
	`

	var course models.Course
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&course.ID,
		&course.Slug,
		&course.AuthorID,
		&course.Title,
		&course.ShortSummary,
		&course.ComplexityLevel,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("course not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get course by id: %w", err)
	}

	return &course, nil
}

// GetAll retrieves courses with filtering and pagination
func (r *courseRepository) GetAll(ctx context.Context, userID int, complexityLevel *models.ComplexityLevel, search string, isMine bool, page, count int) ([]models.CourseDetailResponse, error) {
	var whereClauses []string
	args := []any{userID}

	// Build WHERE clause
	if isMine {
		whereClauses = append(whereClauses, `EXISTS (
			SELECT 1 FROM lesson_user_history 
			WHERE lesson_user_history.course_id = c.id 
			AND lesson_user_history.user_id = ?
		)`)
		args = append(args, userID)
	}

	if complexityLevel != nil {
		whereClauses = append(whereClauses, "complexity_level = ?")
		args = append(args, *complexityLevel)
	}

	if search != "" {
		whereClauses = append(whereClauses, "c.title LIKE ?")
		args = append(args, "%"+search+"%")
	}

	whereClause := ""
	if len(whereClauses) > 0 {
		whereClause = "WHERE " + strings.Join(whereClauses, " AND ")
	}

	// Calculate offset
	offset := (page - 1) * count

	query := fmt.Sprintf(`
		SELECT 
			c.slug,
			c.title,
			c.complexity_level,
			COUNT(DISTINCT l.id) as total_lessons,
			COUNT(DISTINCT luh.lesson_id) as completed_lessons
		FROM courses c
		LEFT JOIN lessons l ON l.course_id = c.id
		LEFT JOIN lesson_user_history luh ON luh.course_id = c.id AND luh.user_id = ? AND luh.lesson_id = l.id
		%s
		GROUP BY c.id, c.slug, c.title, c.complexity_level
		ORDER BY c.id
		LIMIT ? OFFSET ?
	`, whereClause)

	args = append(args, count, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query courses: %w", err)
	}
	defer rows.Close()

	var courses []models.CourseDetailResponse
	for rows.Next() {
		var course models.CourseDetailResponse
		err := rows.Scan(
			&course.Slug,
			&course.Title,
			&course.ComplexityLevel,
			&course.TotalLessons,
			&course.CompletedLessons,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan course: %w", err)
		}
		courses = append(courses, course)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return courses, nil
}

// GetByAuthorOrFull retrieves courses by author ID or full list with filtering and pagination
func (r *courseRepository) GetByAuthorOrFull(ctx context.Context, authorID *int, complexityLevel *models.ComplexityLevel, search string, page, count int) ([]models.CourseListItem, error) {
	whereClauses := []string{}
	args := []any{}
	if authorID != nil {
		whereClauses = append(whereClauses, "author_id = ?")
		args = append(args, *authorID)
	}

	if complexityLevel != nil {
		whereClauses = append(whereClauses, "complexity_level = ?")
		args = append(args, *complexityLevel)
	}

	if search != "" {
		whereClauses = append(whereClauses, "title LIKE ?")
		args = append(args, "%"+search+"%")
	}

	whereClause := ""
	if len(whereClauses) > 0 {
		whereClause = "WHERE " + strings.Join(whereClauses, " AND ")
	}

	// Calculate offset
	offset := (page - 1) * count

	query := fmt.Sprintf(`
		SELECT 
			id,
			slug,
			title,
			complexity_level,
			author_id
		FROM courses
		%s
		ORDER BY id
		LIMIT ? OFFSET ?
	`, whereClause)

	args = append(args, count, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query courses: %w", err)
	}
	defer rows.Close()

	var courses []models.CourseListItem
	for rows.Next() {
		var course models.CourseListItem
		err := rows.Scan(
			&course.ID,
			&course.Slug,
			&course.Title,
			&course.ComplexityLevel,
			&course.AuthorID,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan course: %w", err)
		}
		if authorID != nil {
			course.AuthorID = 0
		}
		courses = append(courses, course)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return courses, nil
}

// GetShortInfo retrieves courses with only ID and Title for select options
//
// If authorID is not nil, it will return courses by author ID.
func (r *courseRepository) GetShortInfo(ctx context.Context, authorID *int) ([]models.CourseShortInfo, error) {
	var rows *sql.Rows
	var err error
	if authorID != nil {
		query := `
			SELECT id, title
			FROM courses
			WHERE author_id = ?
			ORDER BY id
		`
		rows, err = r.db.QueryContext(ctx, query, *authorID)
	} else {
		query := `
			SELECT id, title
			FROM courses
			ORDER BY id
		`
		rows, err = r.db.QueryContext(ctx, query)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query courses: %w", err)
	}
	defer rows.Close()

	var courses []models.CourseShortInfo
	for rows.Next() {
		var course models.CourseShortInfo
		err := rows.Scan(&course.ID, &course.Title)
		if err != nil {
			return nil, fmt.Errorf("failed to scan course: %w", err)
		}
		courses = append(courses, course)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return courses, nil
}

// Create creates a new course
func (r *courseRepository) Create(ctx context.Context, course *models.Course) error {
	query := `
		INSERT INTO courses (slug, author_id, title, short_summary, complexity_level)
		VALUES (?, ?, ?, ?, ?)
	`

	result, err := r.db.ExecContext(ctx, query,
		course.Slug,
		course.AuthorID,
		course.Title,
		course.ShortSummary,
		course.ComplexityLevel,
	)
	if err != nil {
		return fmt.Errorf("failed to create course: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}

	course.ID = int(id)
	return nil
}

// Update updates a course (partial update)
func (r *courseRepository) Update(ctx context.Context, course *models.Course) error {
	var setParts []string
	var args []any

	if course.Slug != "" {
		setParts = append(setParts, "slug = ?")
		args = append(args, course.Slug)
	}
	if course.Title != "" {
		setParts = append(setParts, "title = ?")
		args = append(args, course.Title)
	}
	if course.ShortSummary != "" {
		setParts = append(setParts, "short_summary = ?")
		args = append(args, course.ShortSummary)
	}
	if course.ComplexityLevel != "" {
		setParts = append(setParts, "complexity_level = ?")
		args = append(args, course.ComplexityLevel)
	}
	if course.AuthorID != 0 {
		setParts = append(setParts, "author_id = ?")
		args = append(args, course.AuthorID)
	}

	if len(setParts) == 0 {
		return fmt.Errorf("no fields to update")
	}

	query := fmt.Sprintf(`
		UPDATE courses
		SET %s
		WHERE id = ?
	`, strings.Join(setParts, ", "))

	args = append(args, course.ID)

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update course: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("course not found")
	}

	return nil
}

// Delete deletes a course by ID
func (r *courseRepository) Delete(ctx context.Context, id int) error {
	query := "DELETE FROM courses WHERE id = ?"

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete course: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("course not found")
	}

	return nil
}

// CheckOwnership checks if a course belongs to a tutor
func (r *courseRepository) CheckOwnership(ctx context.Context, id, tutorID int) (bool, error) {
	query := "SELECT EXISTS(SELECT 1 FROM courses WHERE id = ? AND author_id = ?)"
	var exists bool
	err := r.db.QueryRowContext(ctx, query, id, tutorID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check course ownership: %w", err)
	}
	return exists, nil
}

// ExistsBySlug checks if a course with the given slug exists
func (r *courseRepository) ExistsBySlug(ctx context.Context, slug string) (bool, error) {
	query := "SELECT EXISTS(SELECT 1 FROM courses WHERE slug = ?)"
	var exists bool
	err := r.db.QueryRowContext(ctx, query, slug).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check course existence: %w", err)
	}
	return exists, nil
}

// ExistsByTitle checks if a course with the given title exists
func (r *courseRepository) ExistsByTitle(ctx context.Context, title string) (bool, error) {
	query := "SELECT EXISTS(SELECT 1 FROM courses WHERE title = ?)"
	var exists bool
	err := r.db.QueryRowContext(ctx, query, title).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check course existence: %w", err)
	}
	return exists, nil
}
