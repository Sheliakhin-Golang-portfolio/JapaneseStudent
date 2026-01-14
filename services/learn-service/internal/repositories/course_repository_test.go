package repositories

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/japanesestudent/learn-service/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupCourseTestRepository creates a course repository with a mock database
func setupCourseTestRepository(t *testing.T) (*courseRepository, sqlmock.Sqlmock, func()) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)

	repo := NewCourseRepository(db)

	cleanup := func() {
		db.Close()
	}

	return repo, mock, cleanup
}

func TestNewCourseRepository(t *testing.T) {
	db := &sql.DB{}

	repo := NewCourseRepository(db)

	assert.NotNil(t, repo)
	assert.Equal(t, db, repo.db)
}

func TestCourseRepository_GetBySlug(t *testing.T) {
	tests := []struct {
		name          string
		slug          string
		userID        int
		setupMock     func(sqlmock.Sqlmock)
		expectedError bool
		errorContains string
	}{
		{
			name:   "success",
			slug:   "test-course",
			userID: 1,
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "short_summary", "title", "complexity_level", "total_lessons", "completed_lessons"}).
					AddRow(1, "Summary", "Test Course", "Beginner", 10, 5)
				mock.ExpectQuery(`SELECT.*FROM courses c.*WHERE c.slug = \?`).
					WithArgs(1, "test-course").
					WillReturnRows(rows)
			},
			expectedError: false,
		},
		{
			name:   "course not found",
			slug:   "nonexistent",
			userID: 1,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT.*FROM courses c.*WHERE c.slug = \?`).
					WithArgs(1, "nonexistent").
					WillReturnError(sql.ErrNoRows)
			},
			expectedError: true,
			errorContains: "course not found",
		},
		{
			name:   "database error",
			slug:   "test-course",
			userID: 1,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT.*FROM courses c.*WHERE c.slug = \?`).
					WithArgs(1, "test-course").
					WillReturnError(errors.New("database error"))
			},
			expectedError: true,
			errorContains: "failed to get course by slug",
		},
		{
			name:   "scan error",
			slug:   "test-course",
			userID: 1,
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "short_summary", "title", "complexity_level", "total_lessons", "completed_lessons"}).
					AddRow("invalid", "Summary", "Test Course", "Beginner", 10, 5)
				mock.ExpectQuery(`SELECT.*FROM courses c.*WHERE c.slug = \?`).
					WithArgs(1, "test-course").
					WillReturnRows(rows)
			},
			expectedError: true,
			errorContains: "failed to get course by slug",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock, cleanup := setupCourseTestRepository(t)
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
				assert.Equal(t, "Test Course", result.Title)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestCourseRepository_GetByID(t *testing.T) {
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
				rows := sqlmock.NewRows([]string{"id", "slug", "author_id", "title", "short_summary", "complexity_level"}).
					AddRow(1, "test-course", 1, "Test Course", "Summary", "Beginner")
				mock.ExpectQuery(`SELECT id, slug, author_id, title, short_summary, complexity_level FROM courses WHERE id = \?`).
					WithArgs(1).
					WillReturnRows(rows)
			},
			expectedError: false,
		},
		{
			name: "course not found",
			id:   999,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT id, slug, author_id, title, short_summary, complexity_level FROM courses WHERE id = \?`).
					WithArgs(999).
					WillReturnError(sql.ErrNoRows)
			},
			expectedError: true,
			errorContains: "course not found",
		},
		{
			name: "database error",
			id:   1,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT id, slug, author_id, title, short_summary, complexity_level FROM courses WHERE id = \?`).
					WithArgs(1).
					WillReturnError(errors.New("database error"))
			},
			expectedError: true,
			errorContains: "failed to get course by id",
		},
		{
			name: "scan error",
			id:   1,
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "slug", "author_id", "title", "short_summary", "complexity_level"}).
					AddRow("invalid", "test-course", 1, "Test Course", "Summary", "Beginner")
				mock.ExpectQuery(`SELECT id, slug, author_id, title, short_summary, complexity_level FROM courses WHERE id = \?`).
					WithArgs(1).
					WillReturnRows(rows)
			},
			expectedError: true,
			errorContains: "failed to get course by id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock, cleanup := setupCourseTestRepository(t)
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
				assert.Equal(t, "test-course", result.Slug)
				assert.Equal(t, "Test Course", result.Title)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestCourseRepository_GetAll(t *testing.T) {
	tests := []struct {
		name            string
		userID          int
		complexityLevel *models.ComplexityLevel
		search          string
		isMine          bool
		page            int
		count           int
		setupMock       func(sqlmock.Sqlmock)
		expectedError   bool
		expectedCount   int
	}{
		{
			name:   "success with defaults",
			userID: 1,
			page:   1,
			count:  10,
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"slug", "title", "complexity_level", "total_lessons", "completed_lessons"}).
					AddRow("course-1", "Course 1", "Beginner", 10, 5).
					AddRow("course-2", "Course 2", "Intermediate", 15, 8)
				mock.ExpectQuery(`SELECT.*FROM courses c.*ORDER BY c.id LIMIT \? OFFSET \?`).
					WithArgs(1, 10, 0).
					WillReturnRows(rows)
			},
			expectedError: false,
			expectedCount: 2,
		},
		{
			name:          "success with complexity filter",
			userID:        1,
			complexityLevel: func() *models.ComplexityLevel { level := models.ComplexityLevelBeginner; return &level }(),
			page:          1,
			count:         10,
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"slug", "title", "complexity_level", "total_lessons", "completed_lessons"}).
					AddRow("course-1", "Course 1", "Beginner", 10, 5)
				mock.ExpectQuery(`SELECT.*WHERE complexity_level = \?.*LIMIT \? OFFSET \?`).
					WithArgs(1, "Beginner", 10, 0).
					WillReturnRows(rows)
			},
			expectedError: false,
			expectedCount: 1,
		},
		{
			name:   "success with search filter",
			userID: 1,
			search: "test",
			page:   1,
			count:  10,
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"slug", "title", "complexity_level", "total_lessons", "completed_lessons"}).
					AddRow("test-course", "Test Course", "Beginner", 10, 5)
				mock.ExpectQuery(`SELECT.*FROM courses c.*WHERE c\.title LIKE \?.*GROUP BY.*ORDER BY.*LIMIT \? OFFSET \?`).
					WithArgs(1, "%test%", 10, 0).
					WillReturnRows(rows)
			},
			expectedError: false,
			expectedCount: 1,
		},
		{
			name:   "success with isMine filter",
			userID: 1,
			isMine: true,
			page:   1,
			count:  10,
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"slug", "title", "complexity_level", "total_lessons", "completed_lessons"}).
					AddRow("course-1", "Course 1", "Beginner", 10, 5)
				mock.ExpectQuery(`SELECT.*WHERE EXISTS.*LIMIT \? OFFSET \?`).
					WithArgs(1, 1, 10, 0).
					WillReturnRows(rows)
			},
			expectedError: false,
			expectedCount: 1,
		},
		{
			name:   "success with pagination",
			userID: 1,
			page:   2,
			count:  5,
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"slug", "title", "complexity_level", "total_lessons", "completed_lessons"}).
					AddRow("course-6", "Course 6", "Beginner", 10, 5)
				mock.ExpectQuery(`SELECT.*ORDER BY c.id LIMIT \? OFFSET \?`).
					WithArgs(1, 5, 5).
					WillReturnRows(rows)
			},
			expectedError: false,
			expectedCount: 1,
		},
		{
			name:   "empty results",
			userID: 1,
			page:   1,
			count:  10,
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"slug", "title", "complexity_level", "total_lessons", "completed_lessons"})
				mock.ExpectQuery(`SELECT.*FROM courses c.*ORDER BY c.id LIMIT \? OFFSET \?`).
					WithArgs(1, 10, 0).
					WillReturnRows(rows)
			},
			expectedError: false,
			expectedCount: 0,
		},
		{
			name:   "database query error",
			userID: 1,
			page:   1,
			count:  10,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT.*FROM courses c.*ORDER BY c.id LIMIT \? OFFSET \?`).
					WithArgs(1, 10, 0).
					WillReturnError(errors.New("database error"))
			},
			expectedError: true,
			expectedCount: 0,
		},
		{
			name:   "scan error",
			userID: 1,
			page:   1,
			count:  10,
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"slug", "title", "complexity_level", "total_lessons", "completed_lessons"}).
					AddRow("course-1", "Course 1", "Beginner", "invalid", 5)
				mock.ExpectQuery(`SELECT.*FROM courses c.*ORDER BY c.id LIMIT \? OFFSET \?`).
					WithArgs(1, 10, 0).
					WillReturnRows(rows)
			},
			expectedError: true,
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock, cleanup := setupCourseTestRepository(t)
			defer cleanup()

			tt.setupMock(mock)

			result, err := repo.GetAll(context.Background(), tt.userID, tt.complexityLevel, tt.search, tt.isMine, tt.page, tt.count)

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

func TestCourseRepository_GetByAuthorOrFull(t *testing.T) {
	tests := []struct {
		name            string
		authorID        *int
		complexityLevel *models.ComplexityLevel
		search          string
		page            int
		count           int
		setupMock       func(sqlmock.Sqlmock)
		expectedError   bool
		expectedCount   int
	}{
		{
			name:   "success with author ID",
			authorID: func() *int { id := 1; return &id }(),
			page:    1,
			count:   10,
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "slug", "title", "complexity_level", "author_id"}).
					AddRow(1, "course-1", "Course 1", "Beginner", 1).
					AddRow(2, "course-2", "Course 2", "Intermediate", 1)
				mock.ExpectQuery(`SELECT id, slug, title, complexity_level, author_id FROM courses WHERE author_id = \?.*LIMIT \? OFFSET \?`).
					WithArgs(1, 10, 0).
					WillReturnRows(rows)
			},
			expectedError: false,
			expectedCount: 2,
		},
		{
			name:   "success without author ID",
			authorID: nil,
			page:    1,
			count:   10,
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "slug", "title", "complexity_level", "author_id"}).
					AddRow(1, "course-1", "Course 1", "Beginner", 1)
				mock.ExpectQuery(`SELECT id, slug, title, complexity_level, author_id FROM courses.*LIMIT \? OFFSET \?`).
					WithArgs(10, 0).
					WillReturnRows(rows)
			},
			expectedError: false,
			expectedCount: 1,
		},
		{
			name:          "success with complexity filter",
			authorID:      nil,
			complexityLevel: func() *models.ComplexityLevel { level := models.ComplexityLevelBeginner; return &level }(),
			page:          1,
			count:         10,
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "slug", "title", "complexity_level", "author_id"}).
					AddRow(1, "course-1", "Course 1", "Beginner", 1)
				mock.ExpectQuery(`SELECT.*WHERE complexity_level = \?.*LIMIT \? OFFSET \?`).
					WithArgs("Beginner", 10, 0).
					WillReturnRows(rows)
			},
			expectedError: false,
			expectedCount: 1,
		},
		{
			name:   "success with search filter",
			authorID: nil,
			search: "test",
			page:   1,
			count:  10,
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "slug", "title", "complexity_level", "author_id"}).
					AddRow(1, "test-course", "Test Course", "Beginner", 1)
				mock.ExpectQuery(`SELECT.*WHERE title LIKE \?.*LIMIT \? OFFSET \?`).
					WithArgs("%test%", 10, 0).
					WillReturnRows(rows)
			},
			expectedError: false,
			expectedCount: 1,
		},
		{
			name:   "database query error",
			authorID: nil,
			page:    1,
			count:   10,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT id, slug, title, complexity_level, author_id FROM courses.*LIMIT \? OFFSET \?`).
					WithArgs(10, 0).
					WillReturnError(errors.New("database error"))
			},
			expectedError: true,
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock, cleanup := setupCourseTestRepository(t)
			defer cleanup()

			tt.setupMock(mock)

			result, err := repo.GetByAuthorOrFull(context.Background(), tt.authorID, tt.complexityLevel, tt.search, tt.page, tt.count)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Len(t, result, tt.expectedCount)
				if tt.authorID != nil && len(result) > 0 {
					assert.Equal(t, 0, result[0].AuthorID, "author ID should be zeroed for author-scoped queries")
				}
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestCourseRepository_GetShortInfo(t *testing.T) {
	tests := []struct {
		name          string
		authorID      *int
		setupMock     func(sqlmock.Sqlmock)
		expectedError bool
		expectedCount int
	}{
		{
			name:   "success with author ID",
			authorID: func() *int { id := 1; return &id }(),
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "title"}).
					AddRow(1, "Course 1").
					AddRow(2, "Course 2")
				mock.ExpectQuery(`SELECT id, title FROM courses WHERE author_id = \?`).
					WithArgs(1).
					WillReturnRows(rows)
			},
			expectedError: false,
			expectedCount: 2,
		},
		{
			name:   "success without author ID",
			authorID: nil,
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "title"}).
					AddRow(1, "Course 1").
					AddRow(2, "Course 2").
					AddRow(3, "Course 3")
				mock.ExpectQuery(`SELECT id, title FROM courses`).
					WillReturnRows(rows)
			},
			expectedError: false,
			expectedCount: 3,
		},
		{
			name:   "database query error",
			authorID: nil,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT id, title FROM courses`).
					WillReturnError(errors.New("database error"))
			},
			expectedError: true,
			expectedCount: 0,
		},
		{
			name:   "scan error",
			authorID: nil,
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "title"}).
					AddRow("invalid", "Course 1")
				mock.ExpectQuery(`SELECT id, title FROM courses`).
					WillReturnRows(rows)
			},
			expectedError: true,
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock, cleanup := setupCourseTestRepository(t)
			defer cleanup()

			tt.setupMock(mock)

			result, err := repo.GetShortInfo(context.Background(), tt.authorID)

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

func TestCourseRepository_Create(t *testing.T) {
	tests := []struct {
		name          string
		course        *models.Course
		setupMock     func(sqlmock.Sqlmock)
		expectedError bool
		expectedID    int
	}{
		{
			name: "success",
			course: &models.Course{
				Slug:            "test-course",
				AuthorID:        1,
				Title:           "Test Course",
				ShortSummary:    "Summary",
				ComplexityLevel: models.ComplexityLevelBeginner,
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`INSERT INTO courses \(slug, author_id, title, short_summary, complexity_level\) VALUES \(\?, \?, \?, \?, \?\)`).
					WithArgs("test-course", 1, "Test Course", "Summary", "Beginner").
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			expectedError: false,
			expectedID:    1,
		},
		{
			name: "database error",
			course: &models.Course{
				Slug:            "test-course",
				AuthorID:        1,
				Title:           "Test Course",
				ShortSummary:    "Summary",
				ComplexityLevel: models.ComplexityLevelBeginner,
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`INSERT INTO courses`).
					WithArgs("test-course", 1, "Test Course", "Summary", "Beginner").
					WillReturnError(errors.New("database error"))
			},
			expectedError: true,
		},
		{
			name: "last insert id error",
			course: &models.Course{
				Slug:            "test-course",
				AuthorID:        1,
				Title:           "Test Course",
				ShortSummary:    "Summary",
				ComplexityLevel: models.ComplexityLevelBeginner,
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`INSERT INTO courses`).
					WithArgs("test-course", 1, "Test Course", "Summary", "Beginner").
					WillReturnResult(sqlmock.NewErrorResult(errors.New("last insert id error")))
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock, cleanup := setupCourseTestRepository(t)
			defer cleanup()

			tt.setupMock(mock)

			err := repo.Create(context.Background(), tt.course)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedID, tt.course.ID)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestCourseRepository_Update(t *testing.T) {
	tests := []struct {
		name          string
		course        *models.Course
		setupMock     func(sqlmock.Sqlmock)
		expectedError bool
		errorContains string
	}{
		{
			name: "success partial update - slug and title",
			course: &models.Course{
				ID:    1,
				Slug:  "updated-slug",
				Title: "Updated Title",
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`UPDATE courses SET slug = \?, title = \? WHERE id = \?`).
					WithArgs("updated-slug", "Updated Title", 1).
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
			expectedError: false,
		},
		{
			name: "success partial update - all fields",
			course: &models.Course{
				ID:              1,
				Slug:            "updated-slug",
				AuthorID:        2,
				Title:           "Updated Title",
				ShortSummary:    "Updated Summary",
				ComplexityLevel: models.ComplexityLevelIntermediate,
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`UPDATE courses SET slug = \?, title = \?, short_summary = \?, complexity_level = \?, author_id = \? WHERE id = \?`).
					WithArgs("updated-slug", "Updated Title", "Updated Summary", "Intermediate", 2, 1).
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
			expectedError: false,
		},
		{
			name: "no fields to update",
			course: &models.Course{
				ID: 1,
			},
			setupMock:     func(mock sqlmock.Sqlmock) {},
			expectedError: true,
			errorContains: "no fields to update",
		},
		{
			name: "course not found",
			course: &models.Course{
				ID:    999,
				Title: "Updated Title",
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`UPDATE courses SET title = \? WHERE id = \?`).
					WithArgs("Updated Title", 999).
					WillReturnResult(sqlmock.NewResult(0, 0))
			},
			expectedError: true,
			errorContains: "course not found",
		},
		{
			name: "database error",
			course: &models.Course{
				ID:    1,
				Title: "Updated Title",
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`UPDATE courses SET title = \? WHERE id = \?`).
					WithArgs("Updated Title", 1).
					WillReturnError(errors.New("database error"))
			},
			expectedError: true,
			errorContains: "failed to update course",
		},
		{
			name: "rows affected error",
			course: &models.Course{
				ID:    1,
				Title: "Updated Title",
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`UPDATE courses SET title = \? WHERE id = \?`).
					WithArgs("Updated Title", 1).
					WillReturnResult(sqlmock.NewErrorResult(errors.New("rows affected error")))
			},
			expectedError: true,
			errorContains: "failed to get rows affected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock, cleanup := setupCourseTestRepository(t)
			defer cleanup()

			tt.setupMock(mock)

			err := repo.Update(context.Background(), tt.course)

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

func TestCourseRepository_Delete(t *testing.T) {
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
				mock.ExpectExec(`DELETE FROM courses WHERE id = \?`).
					WithArgs(1).
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
			expectedError: false,
		},
		{
			name: "course not found",
			id:   999,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`DELETE FROM courses WHERE id = \?`).
					WithArgs(999).
					WillReturnResult(sqlmock.NewResult(0, 0))
			},
			expectedError: true,
			errorContains: "course not found",
		},
		{
			name: "database error",
			id:   1,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`DELETE FROM courses WHERE id = \?`).
					WithArgs(1).
					WillReturnError(errors.New("database error"))
			},
			expectedError: true,
			errorContains: "failed to delete course",
		},
		{
			name: "rows affected error",
			id:   1,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`DELETE FROM courses WHERE id = \?`).
					WithArgs(1).
					WillReturnResult(sqlmock.NewErrorResult(errors.New("rows affected error")))
			},
			expectedError: true,
			errorContains: "failed to get rows affected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock, cleanup := setupCourseTestRepository(t)
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

func TestCourseRepository_CheckOwnership(t *testing.T) {
	tests := []struct {
		name          string
		id            int
		tutorID       int
		setupMock     func(sqlmock.Sqlmock)
		expectedError bool
		expectedValue bool
	}{
		{
			name:    "success - course belongs to tutor",
			id:      1,
			tutorID: 1,
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"EXISTS(SELECT 1 FROM courses WHERE id = ? AND author_id = ?)"}).
					AddRow(true)
				mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM courses WHERE id = \? AND author_id = \?\)`).
					WithArgs(1, 1).
					WillReturnRows(rows)
			},
			expectedError: false,
			expectedValue: true,
		},
		{
			name:    "success - course does not belong to tutor",
			id:      1,
			tutorID: 2,
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"EXISTS(SELECT 1 FROM courses WHERE id = ? AND author_id = ?)"}).
					AddRow(false)
				mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM courses WHERE id = \? AND author_id = \?\)`).
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
				mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM courses WHERE id = \? AND author_id = \?\)`).
					WithArgs(1, 1).
					WillReturnError(errors.New("database error"))
			},
			expectedError: true,
			expectedValue: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock, cleanup := setupCourseTestRepository(t)
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

func TestCourseRepository_ExistsBySlug(t *testing.T) {
	tests := []struct {
		name          string
		slug          string
		setupMock     func(sqlmock.Sqlmock)
		expectedError bool
		expectedValue bool
	}{
		{
			name: "success - slug exists",
			slug: "test-course",
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"EXISTS(SELECT 1 FROM courses WHERE slug = ?)"}).
					AddRow(true)
				mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM courses WHERE slug = \?\)`).
					WithArgs("test-course").
					WillReturnRows(rows)
			},
			expectedError: false,
			expectedValue: true,
		},
		{
			name: "success - slug does not exist",
			slug: "nonexistent",
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"EXISTS(SELECT 1 FROM courses WHERE slug = ?)"}).
					AddRow(false)
				mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM courses WHERE slug = \?\)`).
					WithArgs("nonexistent").
					WillReturnRows(rows)
			},
			expectedError: false,
			expectedValue: false,
		},
		{
			name: "database error",
			slug: "test-course",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM courses WHERE slug = \?\)`).
					WithArgs("test-course").
					WillReturnError(errors.New("database error"))
			},
			expectedError: true,
			expectedValue: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock, cleanup := setupCourseTestRepository(t)
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

func TestCourseRepository_ExistsByTitle(t *testing.T) {
	tests := []struct {
		name          string
		title         string
		setupMock     func(sqlmock.Sqlmock)
		expectedError bool
		expectedValue bool
	}{
		{
			name:  "success - title exists",
			title: "Test Course",
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"EXISTS(SELECT 1 FROM courses WHERE title = ?)"}).
					AddRow(true)
				mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM courses WHERE title = \?\)`).
					WithArgs("Test Course").
					WillReturnRows(rows)
			},
			expectedError: false,
			expectedValue: true,
		},
		{
			name:  "success - title does not exist",
			title: "Nonexistent Course",
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"EXISTS(SELECT 1 FROM courses WHERE title = ?)"}).
					AddRow(false)
				mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM courses WHERE title = \?\)`).
					WithArgs("Nonexistent Course").
					WillReturnRows(rows)
			},
			expectedError: false,
			expectedValue: false,
		},
		{
			name:  "database error",
			title: "Test Course",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM courses WHERE title = \?\)`).
					WithArgs("Test Course").
					WillReturnError(errors.New("database error"))
			},
			expectedError: true,
			expectedValue: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock, cleanup := setupCourseTestRepository(t)
			defer cleanup()

			tt.setupMock(mock)

			result, err := repo.ExistsByTitle(context.Background(), tt.title)

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
