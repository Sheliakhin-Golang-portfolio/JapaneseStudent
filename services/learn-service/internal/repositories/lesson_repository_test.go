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

// setupLessonTestRepository creates a lesson repository with a mock database
func setupLessonTestRepository(t *testing.T) (*lessonRepository, sqlmock.Sqlmock, func()) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)

	repo := NewLessonRepository(db)

	cleanup := func() {
		db.Close()
	}

	return repo, mock, cleanup
}

func TestNewLessonRepository(t *testing.T) {
	db := &sql.DB{}

	repo := NewLessonRepository(db)

	assert.NotNil(t, repo)
	assert.Equal(t, db, repo.db)
}

func TestLessonRepository_GetBySlug(t *testing.T) {
	tests := []struct {
		name          string
		slug          string
		userID        int
		setupMock     func(sqlmock.Sqlmock)
		expectedError bool
		errorContains string
	}{
		{
			name:   "success - lesson not completed",
			slug:   "test-lesson",
			userID: 1,
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "course_id", "title", "short_summary", "completed"}).
					AddRow(1, 1, "Test Lesson", "Summary", 0)
				mock.ExpectQuery(`SELECT.*FROM lessons l.*WHERE l.slug = \?`).
					WithArgs(1, "test-lesson").
					WillReturnRows(rows)
			},
			expectedError: false,
		},
		{
			name:   "success - lesson completed",
			slug:   "test-lesson",
			userID: 1,
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "course_id", "title", "short_summary", "completed"}).
					AddRow(1, 1, "Test Lesson", "Summary", 1)
				mock.ExpectQuery(`SELECT.*FROM lessons l.*WHERE l.slug = \?`).
					WithArgs(1, "test-lesson").
					WillReturnRows(rows)
			},
			expectedError: false,
		},
		{
			name:   "lesson not found",
			slug:   "nonexistent",
			userID: 1,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT.*FROM lessons l.*WHERE l.slug = \?`).
					WithArgs(1, "nonexistent").
					WillReturnError(sql.ErrNoRows)
			},
			expectedError: true,
			errorContains: "lesson not found",
		},
		{
			name:   "database error",
			slug:   "test-lesson",
			userID: 1,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT.*FROM lessons l.*WHERE l.slug = \?`).
					WithArgs(1, "test-lesson").
					WillReturnError(errors.New("database error"))
			},
			expectedError: true,
			errorContains: "failed to get lesson by slug",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock, cleanup := setupLessonTestRepository(t)
			defer cleanup()

			tt.setupMock(mock)

			result, err := repo.GetBySlug(context.Background(), tt.slug, tt.userID)

			if tt.expectedError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, 1, result.ID)
				assert.Equal(t, "Test Lesson", result.Title)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestLessonRepository_GetByID(t *testing.T) {
	tests := []struct {
		name          string
		id            int
		setupMock     func(sqlmock.Sqlmock)
		expectedError bool
		errorContains string
	}{
		{
			name: "success",
			id:   1,
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "slug", "course_id", "title", "short_summary", "order"}).
					AddRow(1, "test-lesson", 1, "Test Lesson", "Summary", 1)
				mock.ExpectQuery(`SELECT id, slug, course_id, title, short_summary, ` + "`order`" + ` FROM lessons WHERE id = \?`).
					WithArgs(1).
					WillReturnRows(rows)
			},
			expectedError: false,
		},
		{
			name: "lesson not found",
			id:   999,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT id, slug, course_id, title, short_summary, ` + "`order`" + ` FROM lessons WHERE id = \?`).
					WithArgs(999).
					WillReturnError(sql.ErrNoRows)
			},
			expectedError: true,
			errorContains: "lesson not found",
		},
		{
			name: "database error",
			id:   1,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT id, slug, course_id, title, short_summary, ` + "`order`" + ` FROM lessons WHERE id = \?`).
					WithArgs(1).
					WillReturnError(errors.New("database error"))
			},
			expectedError: true,
			errorContains: "failed to get lesson by id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock, cleanup := setupLessonTestRepository(t)
			defer cleanup()

			tt.setupMock(mock)

			result, err := repo.GetByID(context.Background(), tt.id)

			if tt.expectedError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, 1, result.ID)
				assert.Equal(t, "test-lesson", result.Slug)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestLessonRepository_GetByCourseID(t *testing.T) {
	tests := []struct {
		name          string
		courseID      int
		setupMock     func(sqlmock.Sqlmock)
		expectedError bool
		expectedCount int
	}{
		{
			name:     "success",
			courseID: 1,
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "slug", "title", "short_summary", "order"}).
					AddRow(1, "lesson-1", "Lesson 1", "Summary 1", 1).
					AddRow(2, "lesson-2", "Lesson 2", "Summary 2", 2)
				mock.ExpectQuery(`SELECT id, slug, title, short_summary, ` + "`order`" + ` FROM lessons WHERE course_id = \? ORDER BY ` + "`order`").
					WithArgs(1).
					WillReturnRows(rows)
			},
			expectedError: false,
			expectedCount: 2,
		},
		{
			name:     "empty results",
			courseID: 999,
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "slug", "title", "short_summary", "order"})
				mock.ExpectQuery(`SELECT id, slug, title, short_summary, ` + "`order`" + ` FROM lessons WHERE course_id = \? ORDER BY ` + "`order`").
					WithArgs(999).
					WillReturnRows(rows)
			},
			expectedError: false,
			expectedCount: 0,
		},
		{
			name:     "database query error",
			courseID: 1,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT id, slug, title, short_summary, ` + "`order`" + ` FROM lessons WHERE course_id = \? ORDER BY ` + "`order`").
					WithArgs(1).
					WillReturnError(errors.New("database error"))
			},
			expectedError: true,
			expectedCount: 0,
		},
		{
			name:     "scan error",
			courseID: 1,
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "slug", "title", "short_summary", "order"}).
					AddRow("invalid", "lesson-1", "Lesson 1", "Summary 1", 1)
				mock.ExpectQuery(`SELECT id, slug, title, short_summary, ` + "`order`" + ` FROM lessons WHERE course_id = \? ORDER BY ` + "`order`").
					WithArgs(1).
					WillReturnRows(rows)
			},
			expectedError: true,
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock, cleanup := setupLessonTestRepository(t)
			defer cleanup()

			tt.setupMock(mock)

			result, err := repo.GetByCourseID(context.Background(), tt.courseID)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.Len(t, result, tt.expectedCount)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestLessonRepository_GetByCourseIDWithCompletion(t *testing.T) {
	tests := []struct {
		name          string
		courseID      int
		userID        int
		setupMock     func(sqlmock.Sqlmock)
		expectedError bool
		expectedCount int
	}{
		{
			name:     "success",
			courseID: 1,
			userID:   1,
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"slug", "title", "order", "completed"}).
					AddRow("lesson-1", "Lesson 1", 1, 1).
					AddRow("lesson-2", "Lesson 2", 2, 0)
				mock.ExpectQuery(`SELECT.*FROM lessons l.*WHERE l.course_id = \?`).
					WithArgs(1, 1, 1).
					WillReturnRows(rows)
			},
			expectedError: false,
			expectedCount: 2,
		},
		{
			name:     "database query error",
			courseID: 1,
			userID:   1,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT.*FROM lessons l.*WHERE l.course_id = \?`).
					WithArgs(1, 1, 1).
					WillReturnError(errors.New("database error"))
			},
			expectedError: true,
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock, cleanup := setupLessonTestRepository(t)
			defer cleanup()

			tt.setupMock(mock)

			result, err := repo.GetByCourseIDWithCompletion(context.Background(), tt.courseID, tt.userID)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Len(t, result, tt.expectedCount)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestLessonRepository_GetShortInfoByCourseID(t *testing.T) {
	tests := []struct {
		name          string
		courseID      *int
		setupMock     func(sqlmock.Sqlmock)
		expectedError bool
		expectedCount int
	}{
		{
			name:     "success with course ID",
			courseID: func() *int { id := 1; return &id }(),
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "title"}).
					AddRow(1, "Lesson 1").
					AddRow(2, "Lesson 2")
				mock.ExpectQuery(`SELECT id, title FROM lessons WHERE course_id = \? ORDER BY ` + "`order`").
					WithArgs(1).
					WillReturnRows(rows)
			},
			expectedError: false,
			expectedCount: 2,
		},
		{
			name:     "success without course ID",
			courseID: nil,
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "title"}).
					AddRow(1, "Lesson 1").
					AddRow(2, "Lesson 2").
					AddRow(3, "Lesson 3")
				mock.ExpectQuery(`SELECT id, title FROM lessons.*ORDER BY ` + "`order`").
					WillReturnRows(rows)
			},
			expectedError: false,
			expectedCount: 3,
		},
		{
			name:     "database query error",
			courseID: func() *int { id := 1; return &id }(),
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT id, title FROM lessons WHERE course_id = \? ORDER BY ` + "`order`").
					WithArgs(1).
					WillReturnError(errors.New("database error"))
			},
			expectedError: true,
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock, cleanup := setupLessonTestRepository(t)
			defer cleanup()

			tt.setupMock(mock)

			result, err := repo.GetShortInfoByCourseID(context.Background(), tt.courseID)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Len(t, result, tt.expectedCount)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestLessonRepository_ExistsBySlug(t *testing.T) {
	tests := []struct {
		name          string
		slug          string
		setupMock     func(sqlmock.Sqlmock)
		expectedError bool
		expectedValue bool
	}{
		{
			name: "success - slug exists",
			slug: "test-lesson",
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"EXISTS(SELECT 1 FROM lessons WHERE slug = ?)"}).
					AddRow(true)
				mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM lessons WHERE slug = \?\)`).
					WithArgs("test-lesson").
					WillReturnRows(rows)
			},
			expectedError: false,
			expectedValue: true,
		},
		{
			name: "success - slug does not exist",
			slug: "nonexistent",
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"EXISTS(SELECT 1 FROM lessons WHERE slug = ?)"}).
					AddRow(false)
				mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM lessons WHERE slug = \?\)`).
					WithArgs("nonexistent").
					WillReturnRows(rows)
			},
			expectedError: false,
			expectedValue: false,
		},
		{
			name: "database error",
			slug: "test-lesson",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM lessons WHERE slug = \?\)`).
					WithArgs("test-lesson").
					WillReturnError(errors.New("database error"))
			},
			expectedError: true,
			expectedValue: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock, cleanup := setupLessonTestRepository(t)
			defer cleanup()

			tt.setupMock(mock)

			result, err := repo.ExistsBySlug(context.Background(), tt.slug)

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

func TestLessonRepository_ExistsByTitleInCourse(t *testing.T) {
	tests := []struct {
		name          string
		courseID      int
		title         string
		setupMock     func(sqlmock.Sqlmock)
		expectedError bool
		expectedValue bool
	}{
		{
			name:     "success - title exists",
			courseID: 1,
			title:    "Test Lesson",
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"EXISTS(SELECT 1 FROM lessons WHERE course_id = ? AND title = ?)"}).
					AddRow(true)
				mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM lessons WHERE course_id = \? AND title = \?\)`).
					WithArgs(1, "Test Lesson").
					WillReturnRows(rows)
			},
			expectedError: false,
			expectedValue: true,
		},
		{
			name:     "success - title does not exist",
			courseID: 1,
			title:    "Nonexistent",
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"EXISTS(SELECT 1 FROM lessons WHERE course_id = ? AND title = ?)"}).
					AddRow(false)
				mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM lessons WHERE course_id = \? AND title = \?\)`).
					WithArgs(1, "Nonexistent").
					WillReturnRows(rows)
			},
			expectedError: false,
			expectedValue: false,
		},
		{
			name:     "database error",
			courseID: 1,
			title:    "Test Lesson",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM lessons WHERE course_id = \? AND title = \?\)`).
					WithArgs(1, "Test Lesson").
					WillReturnError(errors.New("database error"))
			},
			expectedError: true,
			expectedValue: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock, cleanup := setupLessonTestRepository(t)
			defer cleanup()

			tt.setupMock(mock)

			result, err := repo.ExistsByTitleInCourse(context.Background(), tt.courseID, tt.title)

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

func TestLessonRepository_ExistsByOrderInCourse(t *testing.T) {
	tests := []struct {
		name          string
		courseID      int
		order         int
		setupMock     func(sqlmock.Sqlmock)
		expectedError bool
		expectedValue bool
	}{
		{
			name:     "success - order exists",
			courseID: 1,
			order:    1,
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"EXISTS(SELECT 1 FROM lessons WHERE course_id = ? AND `order` = ?)"}).
					AddRow(true)
				mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM lessons WHERE course_id = \? AND ` + "`order`" + ` = \?\)`).
					WithArgs(1, 1).
					WillReturnRows(rows)
			},
			expectedError: false,
			expectedValue: true,
		},
		{
			name:     "success - order does not exist",
			courseID: 1,
			order:    999,
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"EXISTS(SELECT 1 FROM lessons WHERE course_id = ? AND `order` = ?)"}).
					AddRow(false)
				mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM lessons WHERE course_id = \? AND ` + "`order`" + ` = \?\)`).
					WithArgs(1, 999).
					WillReturnRows(rows)
			},
			expectedError: false,
			expectedValue: false,
		},
		{
			name:     "database error",
			courseID: 1,
			order:    1,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM lessons WHERE course_id = \? AND ` + "`order`" + ` = \?\)`).
					WithArgs(1, 1).
					WillReturnError(errors.New("database error"))
			},
			expectedError: true,
			expectedValue: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock, cleanup := setupLessonTestRepository(t)
			defer cleanup()

			tt.setupMock(mock)

			result, err := repo.ExistsByOrderInCourse(context.Background(), tt.courseID, tt.order)

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

func TestLessonRepository_IncrementOrderForLessons(t *testing.T) {
	tests := []struct {
		name          string
		courseID      int
		order         int
		setupMock     func(sqlmock.Sqlmock)
		expectedError bool
	}{
		{
			name:     "success",
			courseID: 1,
			order:    2,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`UPDATE lessons SET ` + "`order`" + ` = ` + "`order`" + ` \+ 1 WHERE course_id = \? AND ` + "`order`" + ` >= \?`).
					WithArgs(1, 2).
					WillReturnResult(sqlmock.NewResult(0, 3))
			},
			expectedError: false,
		},
		{
			name:     "database error",
			courseID: 1,
			order:    2,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`UPDATE lessons SET ` + "`order`" + ` = ` + "`order`" + ` \+ 1 WHERE course_id = \? AND ` + "`order`" + ` >= \?`).
					WithArgs(1, 2).
					WillReturnError(errors.New("database error"))
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock, cleanup := setupLessonTestRepository(t)
			defer cleanup()

			tt.setupMock(mock)

			err := repo.IncrementOrderForLessons(context.Background(), tt.courseID, tt.order)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestLessonRepository_Create(t *testing.T) {
	tests := []struct {
		name          string
		lesson        *models.Lesson
		setupMock     func(sqlmock.Sqlmock)
		expectedError bool
		expectedID    int
	}{
		{
			name: "success",
			lesson: &models.Lesson{
				Slug:         "test-lesson",
				CourseID:     1,
				Title:        "Test Lesson",
				ShortSummary: "Summary",
				Order:        1,
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`INSERT INTO lessons \(slug, course_id, title, short_summary, ` + "`order`" + `\) VALUES \(\?, \?, \?, \?, \?\)`).
					WithArgs("test-lesson", 1, "Test Lesson", "Summary", 1).
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			expectedError: false,
			expectedID:    1,
		},
		{
			name: "database error",
			lesson: &models.Lesson{
				Slug:         "test-lesson",
				CourseID:     1,
				Title:        "Test Lesson",
				ShortSummary: "Summary",
				Order:        1,
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`INSERT INTO lessons`).
					WithArgs("test-lesson", 1, "Test Lesson", "Summary", 1).
					WillReturnError(errors.New("database error"))
			},
			expectedError: true,
		},
		{
			name: "last insert id error",
			lesson: &models.Lesson{
				Slug:         "test-lesson",
				CourseID:     1,
				Title:        "Test Lesson",
				ShortSummary: "Summary",
				Order:        1,
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`INSERT INTO lessons`).
					WithArgs("test-lesson", 1, "Test Lesson", "Summary", 1).
					WillReturnResult(sqlmock.NewErrorResult(errors.New("last insert id error")))
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock, cleanup := setupLessonTestRepository(t)
			defer cleanup()

			tt.setupMock(mock)

			err := repo.Create(context.Background(), tt.lesson)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedID, tt.lesson.ID)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestLessonRepository_Update(t *testing.T) {
	tests := []struct {
		name          string
		lesson        *models.Lesson
		setupMock     func(sqlmock.Sqlmock)
		expectedError bool
		errorContains string
	}{
		{
			name: "success partial update - slug and title",
			lesson: &models.Lesson{
				ID:    1,
				Slug:  "updated-slug",
				Title: "Updated Title",
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`UPDATE lessons SET slug = \?, title = \? WHERE id = \?`).
					WithArgs("updated-slug", "Updated Title", 1).
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
			expectedError: false,
		},
		{
			name: "success partial update - all fields",
			lesson: &models.Lesson{
				ID:           1,
				Slug:         "updated-slug",
				CourseID:     2,
				Title:        "Updated Title",
				ShortSummary: "Updated Summary",
				Order:        2,
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`UPDATE lessons SET slug = \?, course_id = \?, title = \?, short_summary = \?, ` + "`order`" + ` = \? WHERE id = \?`).
					WithArgs("updated-slug", 2, "Updated Title", "Updated Summary", 2, 1).
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
			expectedError: false,
		},
		{
			name: "no fields to update",
			lesson: &models.Lesson{
				ID: 1,
			},
			setupMock:     func(mock sqlmock.Sqlmock) {},
			expectedError: true,
			errorContains: "no fields to update",
		},
		{
			name: "lesson not found",
			lesson: &models.Lesson{
				ID:    999,
				Title: "Updated Title",
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`UPDATE lessons SET title = \? WHERE id = \?`).
					WithArgs("Updated Title", 999).
					WillReturnResult(sqlmock.NewResult(0, 0))
			},
			expectedError: true,
			errorContains: "lesson not found",
		},
		{
			name: "database error",
			lesson: &models.Lesson{
				ID:    1,
				Title: "Updated Title",
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`UPDATE lessons SET title = \? WHERE id = \?`).
					WithArgs("Updated Title", 1).
					WillReturnError(errors.New("database error"))
			},
			expectedError: true,
			errorContains: "failed to update lesson",
		},
		{
			name: "rows affected error",
			lesson: &models.Lesson{
				ID:    1,
				Title: "Updated Title",
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`UPDATE lessons SET title = \? WHERE id = \?`).
					WithArgs("Updated Title", 1).
					WillReturnResult(sqlmock.NewErrorResult(errors.New("rows affected error")))
			},
			expectedError: true,
			errorContains: "failed to get rows affected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock, cleanup := setupLessonTestRepository(t)
			defer cleanup()

			tt.setupMock(mock)

			err := repo.Update(context.Background(), tt.lesson)

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

func TestLessonRepository_Delete(t *testing.T) {
	tests := []struct {
		name          string
		id            int
		setupMock     func(sqlmock.Sqlmock)
		expectedError bool
		errorContains string
	}{
		{
			name: "success",
			id:   1,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`DELETE FROM lessons WHERE id = \?`).
					WithArgs(1).
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
			expectedError: false,
		},
		{
			name: "lesson not found",
			id:   999,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`DELETE FROM lessons WHERE id = \?`).
					WithArgs(999).
					WillReturnResult(sqlmock.NewResult(0, 0))
			},
			expectedError: true,
			errorContains: "lesson not found",
		},
		{
			name: "database error",
			id:   1,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`DELETE FROM lessons WHERE id = \?`).
					WithArgs(1).
					WillReturnError(errors.New("database error"))
			},
			expectedError: true,
			errorContains: "failed to delete lesson",
		},
		{
			name: "rows affected error",
			id:   1,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`DELETE FROM lessons WHERE id = \?`).
					WithArgs(1).
					WillReturnResult(sqlmock.NewErrorResult(errors.New("rows affected error")))
			},
			expectedError: true,
			errorContains: "failed to get rows affected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock, cleanup := setupLessonTestRepository(t)
			defer cleanup()

			tt.setupMock(mock)

			err := repo.Delete(context.Background(), tt.id)

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

func TestLessonRepository_CheckOwnership(t *testing.T) {
	tests := []struct {
		name          string
		id            int
		tutorID       int
		setupMock     func(sqlmock.Sqlmock)
		expectedError bool
		expectedValue bool
	}{
		{
			name:    "success - lesson belongs to tutor",
			id:      1,
			tutorID: 1,
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"EXISTS(SELECT 1 FROM lessons WHERE id = ? AND course_id IN (SELECT id FROM courses WHERE author_id = ?))"}).
					AddRow(true)
				mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM lessons WHERE id = \? AND course_id IN \(SELECT id FROM courses WHERE author_id = \?\)\)`).
					WithArgs(1, 1).
					WillReturnRows(rows)
			},
			expectedError: false,
			expectedValue: true,
		},
		{
			name:    "success - lesson does not belong to tutor",
			id:      1,
			tutorID: 2,
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"EXISTS(SELECT 1 FROM lessons WHERE id = ? AND course_id IN (SELECT id FROM courses WHERE author_id = ?))"}).
					AddRow(false)
				mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM lessons WHERE id = \? AND course_id IN \(SELECT id FROM courses WHERE author_id = \?\)\)`).
					WithArgs(1, 2).
					WillReturnRows(rows)
			},
			expectedError: false,
			expectedValue: false,
		},
		{
			name:    "database error",
			id:      1,
			tutorID: 1,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM lessons WHERE id = \? AND course_id IN \(SELECT id FROM courses WHERE author_id = \?\)\)`).
					WithArgs(1, 1).
					WillReturnError(errors.New("database error"))
			},
			expectedError: true,
			expectedValue: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock, cleanup := setupLessonTestRepository(t)
			defer cleanup()

			tt.setupMock(mock)

			result, err := repo.CheckOwnership(context.Background(), tt.id, tt.tutorID)

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
