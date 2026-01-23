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

// setupScheduledTaskLogTestRepository creates a scheduled task log repository with a mock database
func setupScheduledTaskLogTestRepository(t *testing.T) (*scheduledTaskLogRepository, sqlmock.Sqlmock, func()) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)

	repo := NewScheduledTaskLogRepository(db)

	cleanup := func() {
		db.Close()
	}

	return repo, mock, cleanup
}

func TestNewScheduledTaskLogRepository(t *testing.T) {
	db := &sql.DB{}

	repo := NewScheduledTaskLogRepository(db)

	assert.NotNil(t, repo)
	assert.Equal(t, db, repo.db)
}

func TestScheduledTaskLogRepository_Create(t *testing.T) {
	tests := []struct {
		name          string
		log           *models.ScheduledTaskLog
		setupMock     func(sqlmock.Sqlmock)
		expectedError bool
		expectedID    int
	}{
		{
			name: "success",
			log: &models.ScheduledTaskLog{
				TaskID:     1,
				JobID:      "job-123",
				Status:     models.ScheduledTaskLogStatusCompleted,
				HTTPStatus: 200,
				Error:      "",
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`INSERT INTO scheduled_task_logs`).
					WithArgs(1, "job-123", "Completed", 200, "").
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			expectedError: false,
			expectedID:    1,
		},
		{
			name: "success with error",
			log: &models.ScheduledTaskLog{
				TaskID:     1,
				JobID:      "job-123",
				Status:     models.ScheduledTaskLogStatusFailed,
				HTTPStatus: 500,
				Error:      "Internal server error",
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`INSERT INTO scheduled_task_logs`).
					WithArgs(1, "job-123", "Failed", 500, "Internal server error").
					WillReturnResult(sqlmock.NewResult(2, 1))
			},
			expectedError: false,
			expectedID:    2,
		},
		{
			name: "database error",
			log: &models.ScheduledTaskLog{
				TaskID:     1,
				JobID:      "job-123",
				Status:     models.ScheduledTaskLogStatusCompleted,
				HTTPStatus: 200,
				Error:      "",
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`INSERT INTO scheduled_task_logs`).
					WithArgs(1, "job-123", "Completed", 200, "").
					WillReturnError(errors.New("database error"))
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock, cleanup := setupScheduledTaskLogTestRepository(t)
			defer cleanup()

			tt.setupMock(mock)

			ctx := context.Background()
			err := repo.Create(ctx, tt.log)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedID, tt.log.ID)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestScheduledTaskLogRepository_GetByID(t *testing.T) {
	tests := []struct {
		name          string
		id            int
		setupMock     func(sqlmock.Sqlmock)
		expectedError bool
		expected      *models.ScheduledTaskLog
	}{
		{
			name: "success",
			id:   1,
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "task_id", "job_id", "status", "http_status", "error", "created_at"}).
					AddRow(1, 1, "job-123", "Completed", 200, "", time.Now())
				mock.ExpectQuery(`SELECT id, task_id, job_id, status, http_status, error, created_at`).
					WithArgs(1).
					WillReturnRows(rows)
			},
			expectedError: false,
			expected: &models.ScheduledTaskLog{
				ID:         1,
				TaskID:     1,
				JobID:      "job-123",
				Status:     models.ScheduledTaskLogStatusCompleted,
				HTTPStatus: 200,
				Error:      "",
			},
		},
		{
			name: "not found",
			id:   999,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT id, task_id, job_id, status, http_status, error, created_at`).
					WithArgs(999).
					WillReturnError(sql.ErrNoRows)
			},
			expectedError: true,
		},
		{
			name: "database error",
			id:   1,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT id, task_id, job_id, status, http_status, error, created_at`).
					WithArgs(1).
					WillReturnError(errors.New("database error"))
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock, cleanup := setupScheduledTaskLogTestRepository(t)
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
				assert.Equal(t, tt.expected.TaskID, result.TaskID)
				assert.Equal(t, tt.expected.JobID, result.JobID)
				assert.Equal(t, tt.expected.Status, result.Status)
				assert.Equal(t, tt.expected.HTTPStatus, result.HTTPStatus)
				assert.Equal(t, tt.expected.Error, result.Error)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestScheduledTaskLogRepository_GetAll(t *testing.T) {
	tests := []struct {
		name          string
		page          int
		count         int
		taskID        int
		jobID         string
		status        string
		setupMock     func(sqlmock.Sqlmock)
		expectedError bool
		expectedCount int
	}{
		{
			name:   "success - no filters",
			page:   1,
			count:  10,
			taskID: 0,
			jobID:  "",
			status: "",
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "task_id", "job_id", "status", "created_at"}).
					AddRow(1, 1, "job-123", "Completed", time.Now()).
					AddRow(2, 2, "job-456", "Failed", time.Now())
				mock.ExpectQuery(`SELECT id, task_id, job_id, status, created_at`).
					WithArgs(10, 0).
					WillReturnRows(rows)
			},
			expectedError: false,
			expectedCount: 2,
		},
		{
			name:   "success - with all filters",
			page:   1,
			count:  10,
			taskID: 1,
			jobID:  "job-123",
			status: "Completed",
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "task_id", "job_id", "status", "created_at"}).
					AddRow(1, 1, "job-123", "Completed", time.Now())
				mock.ExpectQuery(`SELECT id, task_id, job_id, status, created_at`).
					WithArgs(1, "%job-123%", "Completed", 10, 0).
					WillReturnRows(rows)
			},
			expectedError: false,
			expectedCount: 1,
		},
		{
			name:   "success - with task_id filter",
			page:   1,
			count:  10,
			taskID: 1,
			jobID:  "",
			status: "",
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "task_id", "job_id", "status", "created_at"}).
					AddRow(1, 1, "job-123", "Completed", time.Now()).
					AddRow(2, 1, "job-456", "Failed", time.Now())
				mock.ExpectQuery(`SELECT id, task_id, job_id, status, created_at`).
					WithArgs(1, 10, 0).
					WillReturnRows(rows)
			},
			expectedError: false,
			expectedCount: 2,
		},
		{
			name:   "database error",
			page:   1,
			count:  10,
			taskID: 0,
			jobID:  "",
			status: "",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT id, task_id, job_id, status, created_at`).
					WithArgs(10, 0).
					WillReturnError(errors.New("database error"))
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock, cleanup := setupScheduledTaskLogTestRepository(t)
			defer cleanup()

			tt.setupMock(mock)

			ctx := context.Background()
			result, err := repo.GetAll(ctx, tt.page, tt.count, tt.taskID, tt.jobID, tt.status)

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
