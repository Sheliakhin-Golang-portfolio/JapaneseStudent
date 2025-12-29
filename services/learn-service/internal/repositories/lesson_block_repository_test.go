package repositories

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/japanesestudent/learn-service/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupLessonBlockTestRepository creates a lesson block repository with a mock database
func setupLessonBlockTestRepository(t *testing.T) (*lessonBlockRepository, sqlmock.Sqlmock, func()) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)

	repo := NewLessonBlockRepository(db)

	cleanup := func() {
		db.Close()
	}

	return repo, mock, cleanup
}

func TestNewLessonBlockRepository(t *testing.T) {
	db := &sql.DB{}

	repo := NewLessonBlockRepository(db)

	assert.NotNil(t, repo)
	assert.Equal(t, db, repo.db)
}

func TestLessonBlockRepository_GetByID(t *testing.T) {
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
				blockData := json.RawMessage(`{"type":"text","content":"Hello"}`)
				blockDataStr, _ := json.Marshal(blockData)
				rows := sqlmock.NewRows([]string{"id", "lesson_id", "block_type", "block_order", "block_data"}).
					AddRow(1, 1, "text", 1, string(blockDataStr))
				mock.ExpectQuery(`SELECT id, lesson_id, block_type, block_order, block_data FROM lesson_blocks WHERE id = \?`).
					WithArgs(1).
					WillReturnRows(rows)
			},
			expectedError: false,
		},
		{
			name: "block not found",
			id:   999,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT id, lesson_id, block_type, block_order, block_data FROM lesson_blocks WHERE id = \?`).
					WithArgs(999).
					WillReturnError(sql.ErrNoRows)
			},
			expectedError: true,
			errorContains: "lesson block not found",
		},
		{
			name: "database error",
			id:   1,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT id, lesson_id, block_type, block_order, block_data FROM lesson_blocks WHERE id = \?`).
					WithArgs(1).
					WillReturnError(errors.New("database error"))
			},
			expectedError: true,
			errorContains: "failed to get lesson block by id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock, cleanup := setupLessonBlockTestRepository(t)
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
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestLessonBlockRepository_GetByLessonID(t *testing.T) {
	tests := []struct {
		name          string
		lessonID      int
		setupMock     func(sqlmock.Sqlmock)
		expectedError bool
		expectedCount int
	}{
		{
			name:     "success",
			lessonID: 1,
			setupMock: func(mock sqlmock.Sqlmock) {
				blockData := json.RawMessage(`{"type":"text","content":"Hello"}`)
				blockDataStr, _ := json.Marshal(blockData)
				rows := sqlmock.NewRows([]string{"id", "block_type", "block_order", "block_data"}).
					AddRow(1, "text", 1, string(blockDataStr)).
					AddRow(2, "image", 2, string(blockDataStr))
				mock.ExpectQuery(`SELECT id, block_type, block_order, block_data FROM lesson_blocks WHERE lesson_id = \? ORDER BY block_order`).
					WithArgs(1).
					WillReturnRows(rows)
			},
			expectedError: false,
			expectedCount: 2,
		},
		{
			name:     "empty results",
			lessonID: 999,
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "block_type", "block_order", "block_data"})
				mock.ExpectQuery(`SELECT id, block_type, block_order, block_data FROM lesson_blocks WHERE lesson_id = \? ORDER BY block_order`).
					WithArgs(999).
					WillReturnRows(rows)
			},
			expectedError: false,
			expectedCount: 0,
		},
		{
			name:     "database query error",
			lessonID: 1,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT id, block_type, block_order, block_data FROM lesson_blocks WHERE lesson_id = \? ORDER BY block_order`).
					WithArgs(1).
					WillReturnError(errors.New("database error"))
			},
			expectedError: true,
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock, cleanup := setupLessonBlockTestRepository(t)
			defer cleanup()

			tt.setupMock(mock)

			result, err := repo.GetByLessonID(context.Background(), tt.lessonID)

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

func TestLessonBlockRepository_ExistsByOrderInLesson(t *testing.T) {
	tests := []struct {
		name          string
		lessonID      int
		order         int
		setupMock     func(sqlmock.Sqlmock)
		expectedError bool
		expectedValue bool
	}{
		{
			name:     "success - order exists",
			lessonID: 1,
			order:    1,
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"EXISTS(SELECT 1 FROM lesson_blocks WHERE lesson_id = ? AND block_order = ?)"}).
					AddRow(true)
				mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM lesson_blocks WHERE lesson_id = \? AND block_order = \?\)`).
					WithArgs(1, 1).
					WillReturnRows(rows)
			},
			expectedError: false,
			expectedValue: true,
		},
		{
			name:     "success - order does not exist",
			lessonID: 1,
			order:    999,
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"EXISTS(SELECT 1 FROM lesson_blocks WHERE lesson_id = ? AND block_order = ?)"}).
					AddRow(false)
				mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM lesson_blocks WHERE lesson_id = \? AND block_order = \?\)`).
					WithArgs(1, 999).
					WillReturnRows(rows)
			},
			expectedError: false,
			expectedValue: false,
		},
		{
			name:     "database error",
			lessonID: 1,
			order:    1,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM lesson_blocks WHERE lesson_id = \? AND block_order = \?\)`).
					WithArgs(1, 1).
					WillReturnError(errors.New("database error"))
			},
			expectedError: true,
			expectedValue: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock, cleanup := setupLessonBlockTestRepository(t)
			defer cleanup()

			tt.setupMock(mock)

			result, err := repo.ExistsByOrderInLesson(context.Background(), tt.lessonID, tt.order)

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

func TestLessonBlockRepository_IncrementOrderForBlocks(t *testing.T) {
	tests := []struct {
		name          string
		lessonID      int
		order         int
		setupMock     func(sqlmock.Sqlmock)
		expectedError bool
	}{
		{
			name:     "success",
			lessonID: 1,
			order:    2,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`UPDATE lesson_blocks SET block_order = block_order \+ 1 WHERE lesson_id = \? AND block_order >= \?`).
					WithArgs(1, 2).
					WillReturnResult(sqlmock.NewResult(0, 3))
			},
			expectedError: false,
		},
		{
			name:     "database error",
			lessonID: 1,
			order:    2,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`UPDATE lesson_blocks SET block_order = block_order \+ 1 WHERE lesson_id = \? AND block_order >= \?`).
					WithArgs(1, 2).
					WillReturnError(errors.New("database error"))
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock, cleanup := setupLessonBlockTestRepository(t)
			defer cleanup()

			tt.setupMock(mock)

			err := repo.IncrementOrderForBlocks(context.Background(), tt.lessonID, tt.order)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestLessonBlockRepository_Create(t *testing.T) {
	tests := []struct {
		name          string
		block         *models.LessonBlock
		setupMock     func(sqlmock.Sqlmock)
		expectedError bool
		expectedID    int
	}{
		{
			name: "success",
			block: &models.LessonBlock{
				LessonID:  1,
				BlockType: "text",
				BlockOrder: 1,
				BlockData: json.RawMessage(`{"type":"text","content":"Hello"}`),
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				blockDataStr := `{"type":"text","content":"Hello"}`
				mock.ExpectExec(`INSERT INTO lesson_blocks \(lesson_id, block_type, block_order, block_data\) VALUES \(\?, \?, \?, \?\)`).
					WithArgs(1, "text", 1, blockDataStr).
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			expectedError: false,
			expectedID:    1,
		},
		{
			name: "database error",
			block: &models.LessonBlock{
				LessonID:   1,
				BlockType:  "text",
				BlockOrder: 1,
				BlockData:  json.RawMessage(`{"type":"text","content":"Hello"}`),
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				blockDataStr := `{"type":"text","content":"Hello"}`
				mock.ExpectExec(`INSERT INTO lesson_blocks`).
					WithArgs(1, "text", 1, blockDataStr).
					WillReturnError(errors.New("database error"))
			},
			expectedError: true,
		},
		{
			name: "last insert id error",
			block: &models.LessonBlock{
				LessonID:   1,
				BlockType:  "text",
				BlockOrder: 1,
				BlockData:  json.RawMessage(`{"type":"text","content":"Hello"}`),
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				blockDataStr := `{"type":"text","content":"Hello"}`
				mock.ExpectExec(`INSERT INTO lesson_blocks`).
					WithArgs(1, "text", 1, blockDataStr).
					WillReturnResult(sqlmock.NewErrorResult(errors.New("last insert id error")))
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock, cleanup := setupLessonBlockTestRepository(t)
			defer cleanup()

			tt.setupMock(mock)

			err := repo.Create(context.Background(), tt.block)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedID, tt.block.ID)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestLessonBlockRepository_Update(t *testing.T) {
	tests := []struct {
		name          string
		block         *models.LessonBlock
		setupMock     func(sqlmock.Sqlmock)
		expectedError bool
		errorContains string
	}{
		{
			name: "success partial update - block_type",
			block: &models.LessonBlock{
				ID:        1,
				BlockType: "image",
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`UPDATE lesson_blocks SET block_type = \? WHERE id = \?`).
					WithArgs("image", 1).
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
			expectedError: false,
		},
		{
			name: "success partial update - all fields",
			block: &models.LessonBlock{
				ID:         1,
				LessonID:   2,
				BlockType:  "image",
				BlockOrder: 3,
				BlockData:  json.RawMessage(`{"type":"image","url":"test.jpg"}`),
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				blockDataStr := `{"type":"image","url":"test.jpg"}`
				mock.ExpectExec(`UPDATE lesson_blocks SET lesson_id = \?, block_type = \?, block_order = \?, block_data = \? WHERE id = \?`).
					WithArgs(2, "image", 3, blockDataStr, 1).
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
			expectedError: false,
		},
		{
			name: "no fields to update",
			block: &models.LessonBlock{
				ID: 1,
			},
			setupMock:     func(mock sqlmock.Sqlmock) {},
			expectedError: true,
			errorContains: "no fields to update",
		},
		{
			name: "block not found",
			block: &models.LessonBlock{
				ID:        999,
				BlockType: "text",
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`UPDATE lesson_blocks SET block_type = \? WHERE id = \?`).
					WithArgs("text", 999).
					WillReturnResult(sqlmock.NewResult(0, 0))
			},
			expectedError: true,
			errorContains: "lesson block not found",
		},
		{
			name: "database error",
			block: &models.LessonBlock{
				ID:        1,
				BlockType: "text",
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`UPDATE lesson_blocks SET block_type = \? WHERE id = \?`).
					WithArgs("text", 1).
					WillReturnError(errors.New("database error"))
			},
			expectedError: true,
			errorContains: "failed to update lesson block",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock, cleanup := setupLessonBlockTestRepository(t)
			defer cleanup()

			tt.setupMock(mock)

			err := repo.Update(context.Background(), tt.block)

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

func TestLessonBlockRepository_Delete(t *testing.T) {
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
				mock.ExpectExec(`DELETE FROM lesson_blocks WHERE id = \?`).
					WithArgs(1).
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
			expectedError: false,
		},
		{
			name: "block not found",
			id:   999,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`DELETE FROM lesson_blocks WHERE id = \?`).
					WithArgs(999).
					WillReturnResult(sqlmock.NewResult(0, 0))
			},
			expectedError: true,
			errorContains: "lesson block not found",
		},
		{
			name: "database error",
			id:   1,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`DELETE FROM lesson_blocks WHERE id = \?`).
					WithArgs(1).
					WillReturnError(errors.New("database error"))
			},
			expectedError: true,
			errorContains: "failed to delete lesson block",
		},
		{
			name: "rows affected error",
			id:   1,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`DELETE FROM lesson_blocks WHERE id = \?`).
					WithArgs(1).
					WillReturnResult(sqlmock.NewErrorResult(errors.New("rows affected error")))
			},
			expectedError: true,
			errorContains: "failed to get rows affected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock, cleanup := setupLessonBlockTestRepository(t)
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

func TestLessonBlockRepository_CheckOwnership(t *testing.T) {
	tests := []struct {
		name          string
		id            int
		tutorID       int
		setupMock     func(sqlmock.Sqlmock)
		expectedError bool
		expectedValue bool
	}{
		{
			name:    "success - block belongs to tutor",
			id:      1,
			tutorID: 1,
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"EXISTS(SELECT 1 FROM lesson_blocks WHERE id = ? AND lesson_id IN (SELECT id FROM lessons WHERE course_id IN (SELECT id FROM courses WHERE author_id = ?)))"}).
					AddRow(true)
				mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM lesson_blocks WHERE id = \? AND lesson_id.*\)`).
					WithArgs(1, 1).
					WillReturnRows(rows)
			},
			expectedError: false,
			expectedValue: true,
		},
		{
			name:    "success - block does not belong to tutor",
			id:      1,
			tutorID: 2,
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"EXISTS(SELECT 1 FROM lesson_blocks WHERE id = ? AND lesson_id IN (SELECT id FROM lessons WHERE course_id IN (SELECT id FROM courses WHERE author_id = ?)))"}).
					AddRow(false)
				mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM lesson_blocks WHERE id = \? AND lesson_id.*\)`).
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
				mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM lesson_blocks WHERE id = \? AND lesson_id.*\)`).
					WithArgs(1, 1).
					WillReturnError(errors.New("database error"))
			},
			expectedError: true,
			expectedValue: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock, cleanup := setupLessonBlockTestRepository(t)
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
