package repositories

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Sheliakhin-Golang-portfolio/JapaneseStudent/services/task-service/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupImmediateTaskTestRepository creates an immediate task repository with a mock database
func setupImmediateTaskTestRepository(t *testing.T) (*immediateTaskRepository, sqlmock.Sqlmock, func()) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)

	repo := NewImmediateTaskRepository(db)

	cleanup := func() {
		db.Close()
	}

	return repo, mock, cleanup
}

func TestNewImmediateTaskRepository(t *testing.T) {
	db := &sql.DB{}

	repo := NewImmediateTaskRepository(db)

	assert.NotNil(t, repo)
	assert.Equal(t, db, repo.db)
}

func TestImmediateTaskRepository_Create(t *testing.T) {
	templateID := 1
	tests := []struct {
		name          string
		task          *models.ImmediateTask
		setupMock     func(sqlmock.Sqlmock)
		expectedError bool
		expectedID    int
	}{
		{
			name: "success",
			task: &models.ImmediateTask{
				UserID:     100,
				TemplateID: &templateID,
				Content:    "test@example.com;John",
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`INSERT INTO immediate_tasks`).
					WithArgs(100, 1, "test@example.com;John").
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			expectedError: false,
			expectedID:    1,
		},
		{
			name: "database error",
			task: &models.ImmediateTask{
				UserID:     100,
				TemplateID: &templateID,
				Content:    "test@example.com;John",
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`INSERT INTO immediate_tasks`).
					WithArgs(100, 1, "test@example.com;John").
					WillReturnError(errors.New("database error"))
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock, cleanup := setupImmediateTaskTestRepository(t)
			defer cleanup()

			tt.setupMock(mock)

			ctx := context.Background()
			err := repo.Create(ctx, tt.task)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedID, tt.task.ID)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestImmediateTaskRepository_GetByID(t *testing.T) {
	templateID := int64(1)
	tests := []struct {
		name          string
		id            int
		setupMock     func(sqlmock.Sqlmock)
		expectedError bool
		expected      *models.ImmediateTask
	}{
		{
			name: "success with template",
			id:   1,
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "user_id", "template_id", "content", "created_at", "status", "error"}).
					AddRow(1, 100, templateID, "test@example.com;John", time.Now(), "Enqueued", "")
				mock.ExpectQuery(`SELECT id, user_id, template_id, content, created_at, \` + "`status`" + `, error`).
					WithArgs(1).
					WillReturnRows(rows)
			},
			expectedError: false,
			expected: &models.ImmediateTask{
				ID:         1,
				UserID:    100,
				TemplateID: func() *int { id := 1; return &id }(),
				Content:    "test@example.com;John",
				Status:     "Enqueued",
			},
		},
		{
			name: "success without template",
			id:   2,
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "user_id", "template_id", "content", "created_at", "status", "error"}).
					AddRow(2, 100, nil, "test@example.com;John", time.Now(), "Enqueued", "")
				mock.ExpectQuery(`SELECT id, user_id, template_id, content, created_at, \` + "`status`" + `, error`).
					WithArgs(2).
					WillReturnRows(rows)
			},
			expectedError: false,
			expected: &models.ImmediateTask{
				ID:      2,
				UserID:  100,
				Content: "test@example.com;John",
				Status:  "Enqueued",
			},
		},
		{
			name: "not found",
			id:   999,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT id, user_id, template_id, content, created_at, \` + "`status`" + `, error`).
					WithArgs(999).
					WillReturnError(sql.ErrNoRows)
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock, cleanup := setupImmediateTaskTestRepository(t)
			defer cleanup()

			tt.setupMock(mock)

			ctx := context.Background()
			result, err := repo.GetByID(ctx, tt.id)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.expected.ID, result.ID)
				assert.Equal(t, tt.expected.UserID, result.UserID)
				assert.Equal(t, tt.expected.Content, result.Content)
				assert.Equal(t, tt.expected.Status, result.Status)
				if tt.expected.TemplateID != nil {
					assert.NotNil(t, result.TemplateID)
					assert.Equal(t, *tt.expected.TemplateID, *result.TemplateID)
				} else {
					assert.Nil(t, result.TemplateID)
				}
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestImmediateTaskRepository_GetAll(t *testing.T) {
	templateID := int64(1)
	tests := []struct {
		name          string
		page          int
		count         int
		userID        int
		templateID    int
		status        string
		setupMock     func(sqlmock.Sqlmock)
		expectedError bool
		expectedCount int
	}{
		{
			name:       "success - no filters",
			page:       1,
			count:       10,
			userID:     0,
			templateID:  0,
			status:     "",
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "user_id", "template_id", "created_at", "status"}).
					AddRow(1, 100, templateID, time.Now(), "Enqueued").
					AddRow(2, 101, nil, time.Now(), "Completed")
				mock.ExpectQuery(`SELECT id, user_id, template_id, created_at, \` + "`status`" + ``).
					WithArgs(10, 0).
					WillReturnRows(rows)
			},
			expectedError: false,
			expectedCount: 2,
		},
		{
			name:       "success - with filters",
			page:       1,
			count:       10,
			userID:     100,
			templateID:  1,
			status:     "Enqueued",
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "user_id", "template_id", "created_at", "status"}).
					AddRow(1, 100, templateID, time.Now(), "Enqueued")
				mock.ExpectQuery(`SELECT id, user_id, template_id, created_at, \` + "`status`" + ``).
					WithArgs(100, 1, "Enqueued", 10, 0).
					WillReturnRows(rows)
			},
			expectedError: false,
			expectedCount: 1,
		},
		{
			name:       "database error",
			page:       1,
			count:       10,
			userID:     0,
			templateID:  0,
			status:     "",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT id, user_id, template_id, created_at, \` + "`status`" + ``).
					WithArgs(10, 0).
					WillReturnError(errors.New("database error"))
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock, cleanup := setupImmediateTaskTestRepository(t)
			defer cleanup()

			tt.setupMock(mock)

			ctx := context.Background()
			result, err := repo.GetAll(ctx, tt.page, tt.count, tt.userID, tt.templateID, tt.status)

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

func TestImmediateTaskRepository_Update(t *testing.T) {
	templateID := 1
	tests := []struct {
		name          string
		task          *models.ImmediateTask
		setupMock     func(sqlmock.Sqlmock)
		expectedError bool
	}{
		{
			name: "success - update all fields",
			task: &models.ImmediateTask{
				ID:         1,
				UserID:     100,
				TemplateID: &templateID,
				Content:    "new@example.com;Jane",
				Status:     "Completed",
				Error:      "Some error", // Error must be non-empty to be included
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`UPDATE immediate_tasks`).
					WithArgs(100, 1, "new@example.com;Jane", "Completed", "Some error", 1).
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
			expectedError: false,
		},
		{
			name: "success - update only status",
			task: &models.ImmediateTask{
				ID:     1,
				Status: "Failed",
				Error:  "Error message",
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`UPDATE immediate_tasks`).
					WithArgs("Failed", "Error message", 1).
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
			expectedError: false,
		},
		{
			name: "success - nullify template_id",
			task: &models.ImmediateTask{
				ID:         1,
				TemplateID: func() *int { id := 0; return &id }(),
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`UPDATE immediate_tasks`).
					WithArgs(1).
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
			expectedError: false,
		},
		{
			name:     "nothing to update",
			task:     &models.ImmediateTask{ID: 1},
			setupMock: func(mock sqlmock.Sqlmock) {
				// No expectations
			},
			expectedError: false,
		},
		{
			name: "not found",
			task: &models.ImmediateTask{
				ID:     999,
				Status: "Completed",
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`UPDATE immediate_tasks`).
					WithArgs("Completed", 999).
					WillReturnResult(sqlmock.NewResult(0, 0))
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock, cleanup := setupImmediateTaskTestRepository(t)
			defer cleanup()

			tt.setupMock(mock)

			ctx := context.Background()
			err := repo.Update(ctx, tt.task)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestImmediateTaskRepository_UpdateStatus(t *testing.T) {
	tests := []struct {
		name          string
		id            int
		status        models.ImmediateTaskStatus
		errorMsg      string
		setupMock     func(sqlmock.Sqlmock)
		expectedError bool
	}{
		{
			name:     "success",
			id:       1,
			status:   models.ImmediateTaskStatusCompleted,
			errorMsg: "",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`UPDATE immediate_tasks`).
					WithArgs("Completed", "", 1).
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
			expectedError: false,
		},
		{
			name:     "not found",
			id:       999,
			status:   models.ImmediateTaskStatusFailed,
			errorMsg: "Error occurred",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`UPDATE immediate_tasks`).
					WithArgs("Failed", "Error occurred", 999).
					WillReturnResult(sqlmock.NewResult(0, 0))
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock, cleanup := setupImmediateTaskTestRepository(t)
			defer cleanup()

			tt.setupMock(mock)

			ctx := context.Background()
			err := repo.UpdateStatus(ctx, tt.id, tt.status, tt.errorMsg)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestImmediateTaskRepository_Delete(t *testing.T) {
	tests := []struct {
		name          string
		id            int
		setupMock     func(sqlmock.Sqlmock)
		expectedError bool
	}{
		{
			name: "success",
			id:   1,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`DELETE FROM immediate_tasks WHERE id = \?`).
					WithArgs(1).
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
			expectedError: false,
		},
		{
			name: "not found",
			id:   999,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`DELETE FROM immediate_tasks WHERE id = \?`).
					WithArgs(999).
					WillReturnResult(sqlmock.NewResult(0, 0))
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock, cleanup := setupImmediateTaskTestRepository(t)
			defer cleanup()

			tt.setupMock(mock)

			ctx := context.Background()
			err := repo.Delete(ctx, tt.id)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
