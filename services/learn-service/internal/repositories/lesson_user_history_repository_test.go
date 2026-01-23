package repositories

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Sheliakhin-Golang-portfolio/JapaneseStudent/learn-service/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupLessonUserHistoryTestRepository creates a lesson user history repository with a mock database
func setupLessonUserHistoryTestRepository(t *testing.T) (*lessonUserHistoryRepository, sqlmock.Sqlmock, func()) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)

	repo := NewLessonUserHistoryRepository(db)

	cleanup := func() {
		db.Close()
	}

	return repo, mock, cleanup
}

func TestNewLessonUserHistoryRepository(t *testing.T) {
	db := &sql.DB{}

	repo := NewLessonUserHistoryRepository(db)

	assert.NotNil(t, repo)
	assert.Equal(t, db, repo.db)
}

func TestLessonUserHistoryRepository_Exists(t *testing.T) {
	tests := []struct {
		name          string
		userID        int
		courseID      int
		lessonID      int
		setupMock     func(sqlmock.Sqlmock)
		expectedError bool
		expectedValue bool
	}{
		{
			name:     "success - history exists",
			userID:   1,
			courseID: 1,
			lessonID: 1,
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"EXISTS(SELECT 1 FROM lesson_user_history WHERE user_id = ? AND course_id = ? AND lesson_id = ?)"}).
					AddRow(true)
				mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM lesson_user_history WHERE user_id = \? AND course_id = \? AND lesson_id = \?\)`).
					WithArgs(1, 1, 1).
					WillReturnRows(rows)
			},
			expectedError: false,
			expectedValue: true,
		},
		{
			name:     "success - history does not exist",
			userID:   1,
			courseID: 1,
			lessonID: 999,
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"EXISTS(SELECT 1 FROM lesson_user_history WHERE user_id = ? AND course_id = ? AND lesson_id = ?)"}).
					AddRow(false)
				mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM lesson_user_history WHERE user_id = \? AND course_id = \? AND lesson_id = \?\)`).
					WithArgs(1, 1, 999).
					WillReturnRows(rows)
			},
			expectedError: false,
			expectedValue: false,
		},
		{
			name:     "database error",
			userID:   1,
			courseID: 1,
			lessonID: 1,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM lesson_user_history WHERE user_id = \? AND course_id = \? AND lesson_id = \?\)`).
					WithArgs(1, 1, 1).
					WillReturnError(errors.New("database error"))
			},
			expectedError: true,
			expectedValue: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock, cleanup := setupLessonUserHistoryTestRepository(t)
			defer cleanup()

			tt.setupMock(mock)

			result, err := repo.Exists(context.Background(), tt.userID, tt.courseID, tt.lessonID)

			if tt.expectedError {
				assert.Error(t, err)
				assert.False(t, result)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedValue, result)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestLessonUserHistoryRepository_Create(t *testing.T) {
	tests := []struct {
		name          string
		history       *models.LessonUserHistory
		setupMock     func(sqlmock.Sqlmock)
		expectedError bool
		expectedID    int
	}{
		{
			name: "success",
			history: &models.LessonUserHistory{
				UserID:   1,
				CourseID: 1,
				LessonID: 1,
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`INSERT INTO lesson_user_history \(user_id, course_id, lesson_id\) VALUES \(\?, \?, \?\)`).
					WithArgs(1, 1, 1).
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			expectedError: false,
			expectedID:    1,
		},
		{
			name: "database error",
			history: &models.LessonUserHistory{
				UserID:   1,
				CourseID: 1,
				LessonID: 1,
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`INSERT INTO lesson_user_history`).
					WithArgs(1, 1, 1).
					WillReturnError(errors.New("database error"))
			},
			expectedError: true,
		},
		{
			name: "last insert id error",
			history: &models.LessonUserHistory{
				UserID:   1,
				CourseID: 1,
				LessonID: 1,
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`INSERT INTO lesson_user_history`).
					WithArgs(1, 1, 1).
					WillReturnResult(sqlmock.NewErrorResult(errors.New("last insert id error")))
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock, cleanup := setupLessonUserHistoryTestRepository(t)
			defer cleanup()

			tt.setupMock(mock)

			err := repo.Create(context.Background(), tt.history)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedID, tt.history.ID)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestLessonUserHistoryRepository_Delete(t *testing.T) {
	tests := []struct {
		name          string
		userID        int
		courseID      int
		lessonID      int
		setupMock     func(sqlmock.Sqlmock)
		expectedError bool
		errorContains string
	}{
		{
			name:     "success",
			userID:   1,
			courseID: 1,
			lessonID: 1,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`DELETE FROM lesson_user_history WHERE user_id = \? AND course_id = \? AND lesson_id = \?`).
					WithArgs(1, 1, 1).
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
			expectedError: false,
		},
		{
			name:     "history record not found",
			userID:   1,
			courseID: 1,
			lessonID: 999,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`DELETE FROM lesson_user_history WHERE user_id = \? AND course_id = \? AND lesson_id = \?`).
					WithArgs(1, 1, 999).
					WillReturnResult(sqlmock.NewResult(0, 0))
			},
			expectedError: true,
			errorContains: "history record not found",
		},
		{
			name:     "database error",
			userID:   1,
			courseID: 1,
			lessonID: 1,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`DELETE FROM lesson_user_history WHERE user_id = \? AND course_id = \? AND lesson_id = \?`).
					WithArgs(1, 1, 1).
					WillReturnError(errors.New("database error"))
			},
			expectedError: true,
			errorContains: "failed to delete history record",
		},
		{
			name:     "rows affected error",
			userID:   1,
			courseID: 1,
			lessonID: 1,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`DELETE FROM lesson_user_history WHERE user_id = \? AND course_id = \? AND lesson_id = \?`).
					WithArgs(1, 1, 1).
					WillReturnResult(sqlmock.NewErrorResult(errors.New("rows affected error")))
			},
			expectedError: true,
			errorContains: "failed to get rows affected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock, cleanup := setupLessonUserHistoryTestRepository(t)
			defer cleanup()

			tt.setupMock(mock)

			err := repo.Delete(context.Background(), tt.userID, tt.courseID, tt.lessonID)

			if tt.expectedError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestLessonUserHistoryRepository_CountCompletedLessonsByCourse(t *testing.T) {
	tests := []struct {
		name          string
		userID        int
		courseID      int
		setupMock     func(sqlmock.Sqlmock)
		expectedError bool
		expectedCount int
	}{
		{
			name:     "success",
			userID:   1,
			courseID: 1,
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"COUNT(DISTINCT lesson_id)"}).
					AddRow(5)
				mock.ExpectQuery(`SELECT COUNT\(DISTINCT lesson_id\) FROM lesson_user_history WHERE user_id = \? AND course_id = \?`).
					WithArgs(1, 1).
					WillReturnRows(rows)
			},
			expectedError: false,
			expectedCount: 5,
		},
		{
			name:     "zero completed lessons",
			userID:   1,
			courseID: 999,
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"COUNT(DISTINCT lesson_id)"}).
					AddRow(0)
				mock.ExpectQuery(`SELECT COUNT\(DISTINCT lesson_id\) FROM lesson_user_history WHERE user_id = \? AND course_id = \?`).
					WithArgs(1, 999).
					WillReturnRows(rows)
			},
			expectedError: false,
			expectedCount: 0,
		},
		{
			name:     "database error",
			userID:   1,
			courseID: 1,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT COUNT\(DISTINCT lesson_id\) FROM lesson_user_history WHERE user_id = \? AND course_id = \?`).
					WithArgs(1, 1).
					WillReturnError(errors.New("database error"))
			},
			expectedError: true,
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock, cleanup := setupLessonUserHistoryTestRepository(t)
			defer cleanup()

			tt.setupMock(mock)

			result, err := repo.CountCompletedLessonsByCourse(context.Background(), tt.userID, tt.courseID)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Equal(t, 0, result)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedCount, result)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
