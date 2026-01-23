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

// setupScheduledTaskTestRepository creates a scheduled task repository with a mock database
func setupScheduledTaskTestRepository(t *testing.T) (*scheduledTaskRepository, sqlmock.Sqlmock, func()) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)

	repo := NewScheduledTaskRepository(db)

	cleanup := func() {
		db.Close()
	}

	return repo, mock, cleanup
}

func TestNewScheduledTaskRepository(t *testing.T) {
	db := &sql.DB{}

	repo := NewScheduledTaskRepository(db)

	assert.NotNil(t, repo)
	assert.Equal(t, db, repo.db)
}

func TestScheduledTaskRepository_Create(t *testing.T) {
	userID := 100
	userIDPtr := &userID
	templateID := 1
	nextRun := time.Now().Add(1 * time.Hour)
	tests := []struct {
		name          string
		task          *models.ScheduledTask
		setupMock     func(sqlmock.Sqlmock)
		expectedError bool
		expectedID    int
	}{
		{
			name: "success",
			task: &models.ScheduledTask{
				UserID:     userIDPtr,
				TemplateID: &templateID,
				URL:        "http://example.com",
				Content:    "test@example.com;John",
				NextRun:    nextRun,
				Cron:       "0 0 * * *",
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`INSERT INTO scheduled_tasks`).
					WithArgs(userID, templateID, "http://example.com", "test@example.com;John", sqlmock.AnyArg(), "0 0 * * *").
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			expectedError: false,
			expectedID:    1,
		},
		{
			name: "database error",
			task: &models.ScheduledTask{
				UserID:     userIDPtr,
				TemplateID: &templateID,
				URL:        "http://example.com",
				Content:    "test@example.com;John",
				NextRun:    nextRun,
				Cron:       "0 0 * * *",
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`INSERT INTO scheduled_tasks`).
					WithArgs(userID, templateID, "http://example.com", "test@example.com;John", sqlmock.AnyArg(), "0 0 * * *").
					WillReturnError(errors.New("database error"))
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock, cleanup := setupScheduledTaskTestRepository(t)
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

func TestScheduledTaskRepository_GetByID(t *testing.T) {
	userID := 100
	userIDPtr := &userID
	templateID := 1
	nextRun := time.Now().Add(1 * time.Hour)
	tests := []struct {
		name          string
		id            int
		setupMock     func(sqlmock.Sqlmock)
		expectedError bool
		expected      *models.ScheduledTask
	}{
		{
			name: "success",
			id:   1,
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "user_id", "template_id", "url", "content", "created_at", "next_run", "previous_run", "active", "cron"}).
					AddRow(1, userID, templateID, "http://example.com", "test@example.com;John", time.Now(), nextRun, nil, true, "0 0 * * *")
				mock.ExpectQuery(`SELECT id, user_id, template_id, url, content, created_at, next_run, previous_run, active, cron`).
					WithArgs(1).
					WillReturnRows(rows)
			},
			expectedError: false,
			expected: &models.ScheduledTask{
				ID:         1,
				UserID:    userIDPtr,
				TemplateID: &templateID,
				URL:       "http://example.com",
				Content:   "test@example.com;John",
				NextRun:   nextRun,
				Active:    true,
				Cron:      "0 0 * * *",
			},
		},
		{
			name: "not found",
			id:   999,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT id, user_id, template_id, url, content, created_at, next_run, previous_run, active, cron`).
					WithArgs(999).
					WillReturnError(sql.ErrNoRows)
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock, cleanup := setupScheduledTaskTestRepository(t)
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
				assert.Equal(t, tt.expected.URL, result.URL)
				assert.Equal(t, tt.expected.Content, result.Content)
				assert.Equal(t, tt.expected.Active, result.Active)
				assert.Equal(t, tt.expected.Cron, result.Cron)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestScheduledTaskRepository_GetAll(t *testing.T) {
	userID := 100
	templateID := 1
	nextRun := time.Now().Add(1 * time.Hour)
	tests := []struct {
		name          string
		page          int
		count         int
		userID        int
		templateID    int
		active        *bool
		setupMock     func(sqlmock.Sqlmock)
		expectedError bool
		expectedCount  int
	}{
		{
			name:       "success - no filters",
			page:       1,
			count:       10,
			userID:     0,
			templateID: 0,
			active:     nil,
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "user_id", "template_id", "created_at", "url", "active", "next_run"}).
					AddRow(1, &userID, &templateID, time.Now(), "http://example.com", true, nextRun).
					AddRow(2, &userID, nil, time.Now(), "http://example2.com", false, nextRun)
				mock.ExpectQuery(`SELECT id, user_id, template_id, created_at, url, active, next_run`).
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
			templateID: 1,
			active:     func() *bool { b := true; return &b }(),
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "user_id", "template_id", "created_at", "url", "active", "next_run"}).
					AddRow(1, userID, templateID, time.Now(), "http://example.com", true, nextRun)
				mock.ExpectQuery(`SELECT id, user_id, template_id, created_at, url, active, next_run`).
					WithArgs(100, 1, true, 10, 0).
					WillReturnRows(rows)
			},
			expectedError: false,
			expectedCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock, cleanup := setupScheduledTaskTestRepository(t)
			defer cleanup()

			tt.setupMock(mock)

			ctx := context.Background()
			result, err := repo.GetAll(ctx, tt.page, tt.count, tt.userID, tt.templateID, tt.active)

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

func TestScheduledTaskRepository_GetActiveTasksForRestore(t *testing.T) {
	nextRun := time.Now().Add(1 * time.Hour)
	tests := []struct {
		name          string
		setupMock     func(sqlmock.Sqlmock)
		expectedError bool
		expectedCount int
	}{
		{
			name: "success",
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "next_run"}).
					AddRow(1, nextRun).
					AddRow(2, nextRun.Add(1*time.Hour))
				mock.ExpectQuery(`SELECT id, next_run`).
					WillReturnRows(rows)
			},
			expectedError: false,
			expectedCount: 2,
		},
		{
			name: "database error",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT id, next_run`).
					WillReturnError(errors.New("database error"))
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock, cleanup := setupScheduledTaskTestRepository(t)
			defer cleanup()

			tt.setupMock(mock)

			ctx := context.Background()
			result, err := repo.GetActiveTasksForRestore(ctx)

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

func TestScheduledTaskRepository_GetActiveTasksForNext24Hours(t *testing.T) {
	nextRun := time.Now().Add(1 * time.Hour)
	tests := []struct {
		name          string
		setupMock     func(sqlmock.Sqlmock)
		expectedError bool
		expectedCount int
	}{
		{
			name: "success",
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "next_run"}).
					AddRow(1, nextRun).
					AddRow(2, nextRun.Add(1*time.Hour))
				mock.ExpectQuery(`SELECT id, next_run`).
					WillReturnRows(rows)
			},
			expectedError: false,
			expectedCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock, cleanup := setupScheduledTaskTestRepository(t)
			defer cleanup()

			tt.setupMock(mock)

			ctx := context.Background()
			result, err := repo.GetActiveTasksForNext24Hours(ctx)

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

func TestScheduledTaskRepository_Update(t *testing.T) {
	userID := 100
	templateID := 1
	nextRun := time.Now().Add(1 * time.Hour)
	url := "http://example.com"
	content := "test@example.com;John"
	active := true
	tests := []struct {
		name          string
		id            int
		req           *models.UpdateScheduledTaskRequest
		setupMock     func(sqlmock.Sqlmock)
		expectedError bool
	}{
		{
			name: "success - update all fields",
			id:   1,
			req: &models.UpdateScheduledTaskRequest{
				UserID:     &userID,
				TemplateID: &templateID,
				URL:        &url,
				Content:    &content,
				NextRun:    &nextRun,
				Active:     &active,
				Cron:       "0 0 * * *",
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`UPDATE scheduled_tasks`).
					WithArgs(userID, templateID, url, content, sqlmock.AnyArg(), active, "0 0 * * *", 1).
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
			expectedError: false,
		},
		{
			name: "success - nullify user_id",
			id:   1,
			req: &models.UpdateScheduledTaskRequest{
				UserID: func() *int { id := 0; return &id }(),
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`UPDATE scheduled_tasks`).
					WithArgs(1).
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
			expectedError: false,
		},
		{
			name: "no fields to update",
			id:   1,
			req:  &models.UpdateScheduledTaskRequest{},
			setupMock: func(mock sqlmock.Sqlmock) {
				// No expectations
			},
			expectedError: true,
		},
		{
			name: "not found",
			id:   999,
			req: &models.UpdateScheduledTaskRequest{
				Active: &active,
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`UPDATE scheduled_tasks`).
					WithArgs(active, 999).
					WillReturnResult(sqlmock.NewResult(0, 0))
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock, cleanup := setupScheduledTaskTestRepository(t)
			defer cleanup()

			tt.setupMock(mock)

			ctx := context.Background()
			err := repo.Update(ctx, tt.id, tt.req)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestScheduledTaskRepository_UpdatePreviousRunAndNextRun(t *testing.T) {
	previousRun := time.Now()
	nextRun := time.Now().Add(1 * time.Hour)
	tests := []struct {
		name          string
		id            int
		previousRun   time.Time
		nextRun       time.Time
		setupMock     func(sqlmock.Sqlmock)
		expectedError bool
	}{
		{
			name:        "success",
			id:          1,
			previousRun: previousRun,
			nextRun:     nextRun,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`UPDATE scheduled_tasks SET previous_run = \?, next_run = \? WHERE id = \?`).
					WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), 1).
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
			expectedError: false,
		},
		{
			name:        "not found",
			id:          999,
			previousRun: previousRun,
			nextRun:     nextRun,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`UPDATE scheduled_tasks SET previous_run = \?, next_run = \? WHERE id = \?`).
					WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), 999).
					WillReturnResult(sqlmock.NewResult(0, 0))
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock, cleanup := setupScheduledTaskTestRepository(t)
			defer cleanup()

			tt.setupMock(mock)

			ctx := context.Background()
			err := repo.UpdatePreviousRunAndNextRun(ctx, tt.id, tt.previousRun, tt.nextRun)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestScheduledTaskRepository_UpdateURL(t *testing.T) {
	tests := []struct {
		name          string
		id            int
		url           string
		setupMock     func(sqlmock.Sqlmock)
		expectedError bool
	}{
		{
			name: "success",
			id:   1,
			url:  "http://example.com",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`UPDATE scheduled_tasks SET url = \? WHERE id = \?`).
					WithArgs("http://example.com", 1).
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
			expectedError: false,
		},
		{
			name: "not found",
			id:   999,
			url:  "http://example.com",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`UPDATE scheduled_tasks SET url = \? WHERE id = \?`).
					WithArgs("http://example.com", 999).
					WillReturnResult(sqlmock.NewResult(0, 0))
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock, cleanup := setupScheduledTaskTestRepository(t)
			defer cleanup()

			tt.setupMock(mock)

			ctx := context.Background()
			err := repo.UpdateURL(ctx, tt.id, tt.url)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestScheduledTaskRepository_Delete(t *testing.T) {
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
				mock.ExpectExec(`DELETE FROM scheduled_tasks WHERE id = \?`).
					WithArgs(1).
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
			expectedError: false,
		},
		{
			name: "not found",
			id:   999,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`DELETE FROM scheduled_tasks WHERE id = \?`).
					WithArgs(999).
					WillReturnResult(sqlmock.NewResult(0, 0))
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock, cleanup := setupScheduledTaskTestRepository(t)
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

func TestScheduledTaskRepository_GetURLByID(t *testing.T) {
	tests := []struct {
		name          string
		id            int
		setupMock     func(sqlmock.Sqlmock)
		expectedError bool
		expectedURL   string
	}{
		{
			name: "success",
			id:   1,
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"url"}).AddRow("http://example.com")
				mock.ExpectQuery(`SELECT url FROM scheduled_tasks WHERE id = \?`).
					WithArgs(1).
					WillReturnRows(rows)
			},
			expectedError: false,
			expectedURL:   "http://example.com",
		},
		{
			name: "not found",
			id:   999,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT url FROM scheduled_tasks WHERE id = \?`).
					WithArgs(999).
					WillReturnError(sql.ErrNoRows)
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock, cleanup := setupScheduledTaskTestRepository(t)
			defer cleanup()

			tt.setupMock(mock)

			ctx := context.Background()
			url, err := repo.GetURLByID(ctx, tt.id)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Empty(t, url)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedURL, url)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestScheduledTaskRepository_GetTemplateIDByID(t *testing.T) {
	templateID := 1
	tests := []struct {
		name          string
		id            int
		setupMock     func(sqlmock.Sqlmock)
		expectedError bool
		expectedID    *int
	}{
		{
			name: "success",
			id:   1,
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"template_id"}).AddRow(templateID)
				mock.ExpectQuery(`SELECT template_id FROM scheduled_tasks WHERE id = \?`).
					WithArgs(1).
					WillReturnRows(rows)
			},
			expectedError: false,
			expectedID:    &templateID,
		},
		{
			name: "not found",
			id:   999,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT template_id FROM scheduled_tasks WHERE id = \?`).
					WithArgs(999).
					WillReturnError(sql.ErrNoRows)
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock, cleanup := setupScheduledTaskTestRepository(t)
			defer cleanup()

			tt.setupMock(mock)

			ctx := context.Background()
			id, err := repo.GetTemplateIDByID(ctx, tt.id)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, id)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, id)
				assert.Equal(t, *tt.expectedID, *id)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestScheduledTaskRepository_GetContentByID(t *testing.T) {
	tests := []struct {
		name          string
		id            int
		setupMock     func(sqlmock.Sqlmock)
		expectedError bool
		expectedContent string
	}{
		{
			name: "success",
			id:   1,
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"content"}).AddRow("test@example.com;John")
				mock.ExpectQuery(`SELECT content FROM scheduled_tasks WHERE id = \?`).
					WithArgs(1).
					WillReturnRows(rows)
			},
			expectedError: false,
			expectedContent: "test@example.com;John",
		},
		{
			name: "not found",
			id:   999,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT content FROM scheduled_tasks WHERE id = \?`).
					WithArgs(999).
					WillReturnError(sql.ErrNoRows)
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock, cleanup := setupScheduledTaskTestRepository(t)
			defer cleanup()

			tt.setupMock(mock)

			ctx := context.Background()
			content, err := repo.GetContentByID(ctx, tt.id)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Empty(t, content)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedContent, content)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
