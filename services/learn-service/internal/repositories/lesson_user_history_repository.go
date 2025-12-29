package repositories

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/japanesestudent/learn-service/internal/models"
)

type lessonUserHistoryRepository struct {
	db *sql.DB
}

// NewLessonUserHistoryRepository creates a new lesson user history repository
func NewLessonUserHistoryRepository(db *sql.DB) *lessonUserHistoryRepository {
	return &lessonUserHistoryRepository{
		db: db,
	}
}

// Exists checks if a history record exists for user, course, and lesson
func (r *lessonUserHistoryRepository) Exists(ctx context.Context, userID, courseID, lessonID int) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM lesson_user_history WHERE user_id = ? AND course_id = ? AND lesson_id = ?)`

	var exists bool
	err := r.db.QueryRowContext(ctx, query, userID, courseID, lessonID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check history existence: %w", err)
	}

	return exists, nil
}

// Create creates a new history record
func (r *lessonUserHistoryRepository) Create(ctx context.Context, history *models.LessonUserHistory) error {
	query := `
		INSERT INTO lesson_user_history (user_id, course_id, lesson_id)
		VALUES (?, ?, ?)
	`

	result, err := r.db.ExecContext(ctx, query,
		history.UserID,
		history.CourseID,
		history.LessonID,
	)
	if err != nil {
		return fmt.Errorf("failed to create history record: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}

	history.ID = int(id)
	return nil
}

// Delete deletes a history record
func (r *lessonUserHistoryRepository) Delete(ctx context.Context, userID, courseID, lessonID int) error {
	query := `
		DELETE FROM lesson_user_history
		WHERE user_id = ? AND course_id = ? AND lesson_id = ?
	`

	result, err := r.db.ExecContext(ctx, query, userID, courseID, lessonID)
	if err != nil {
		return fmt.Errorf("failed to delete history record: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("history record not found")
	}

	return nil
}

// CountCompletedLessonsByCourse counts completed lessons for a user in a course
func (r *lessonUserHistoryRepository) CountCompletedLessonsByCourse(ctx context.Context, userID, courseID int) (int, error) {
	query := `
		SELECT COUNT(DISTINCT lesson_id)
		FROM lesson_user_history
		WHERE user_id = ? AND course_id = ?
	`

	var count int
	err := r.db.QueryRowContext(ctx, query, userID, courseID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count completed lessons: %w", err)
	}

	return count, nil
}
