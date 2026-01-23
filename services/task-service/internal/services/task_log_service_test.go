package services

import (
	"context"
	"errors"
	"testing"

	"github.com/Sheliakhin-Golang-portfolio/JapaneseStudent/services/task-service/internal/models"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

// mockScheduledTaskLogRepository is a mock implementation of ScheduledTaskLogRepository
type mockScheduledTaskLogRepository struct {
	log  *models.ScheduledTaskLog
	logs []models.ScheduledTaskLogListItem
	err  error
}

func (m *mockScheduledTaskLogRepository) Create(ctx context.Context, log *models.ScheduledTaskLog) error {
	if m.err != nil {
		return m.err
	}
	log.ID = 1
	return nil
}

func (m *mockScheduledTaskLogRepository) GetByID(ctx context.Context, id int) (*models.ScheduledTaskLog, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.log, nil
}

func (m *mockScheduledTaskLogRepository) GetAll(ctx context.Context, page, count, taskID int, jobID, status string) ([]models.ScheduledTaskLogListItem, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.logs, nil
}

func TestNewTaskLogService(t *testing.T) {
	repo := &mockScheduledTaskLogRepository{}
	logger := zap.NewNop()

	svc := NewTaskLogService(repo, logger)

	assert.NotNil(t, svc)
	assert.Equal(t, repo, svc.repo)
}

func TestScheduledTaskLogService_Create(t *testing.T) {
	tests := []struct {
		name          string
		log           *models.ScheduledTaskLog
		repo          *mockScheduledTaskLogRepository
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
			repo:          &mockScheduledTaskLogRepository{},
			expectedError: false,
			expectedID:    1,
		},
		{
			name: "repository error",
			log: &models.ScheduledTaskLog{
				TaskID:     1,
				JobID:      "job-123",
				Status:     models.ScheduledTaskLogStatusCompleted,
				HTTPStatus: 200,
				Error:      "",
			},
			repo: &mockScheduledTaskLogRepository{
				err: errors.New("database error"),
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := zap.NewNop()
			svc := NewTaskLogService(tt.repo, logger)

			ctx := context.Background()
			err := svc.Create(ctx, tt.log)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedID, tt.log.ID)
			}
		})
	}
}

func TestScheduledTaskLogService_GetByID(t *testing.T) {
	tests := []struct {
		name          string
		id            int
		repo          *mockScheduledTaskLogRepository
		expectedError bool
	}{
		{
			name: "success",
			id:   1,
			repo: &mockScheduledTaskLogRepository{
				log: &models.ScheduledTaskLog{
					ID:         1,
					TaskID:     1,
					JobID:      "job-123",
					Status:     models.ScheduledTaskLogStatusCompleted,
					HTTPStatus: 200,
					Error:      "",
				},
			},
			expectedError: false,
		},
		{
			name: "not found",
			id:   999,
			repo: &mockScheduledTaskLogRepository{
				err: errors.New("scheduled task log not found"),
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := zap.NewNop()
			svc := NewTaskLogService(tt.repo, logger)

			ctx := context.Background()
			result, err := svc.GetByID(ctx, tt.id)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
			}
		})
	}
}

func TestScheduledTaskLogService_GetAll(t *testing.T) {
	tests := []struct {
		name          string
		page          int
		count         int
		taskID        int
		jobID         string
		status        string
		repo          *mockScheduledTaskLogRepository
		expectedError bool
		expectedCount int
	}{
		{
			name:   "success",
			page:   1,
			count:  10,
			taskID: 0,
			jobID:  "",
			status: "",
			repo: &mockScheduledTaskLogRepository{
				logs: []models.ScheduledTaskLogListItem{
					{ID: 1, TaskID: 1, JobID: "job-123", Status: models.ScheduledTaskLogStatusCompleted},
					{ID: 2, TaskID: 2, JobID: "job-456", Status: models.ScheduledTaskLogStatusFailed},
				},
			},
			expectedError: false,
			expectedCount: 2,
		},
		{
			name:   "default page and count",
			page:   0,
			count:  0,
			taskID: 0,
			jobID:  "",
			status: "",
			repo: &mockScheduledTaskLogRepository{
				logs: []models.ScheduledTaskLogListItem{},
			},
			expectedError: false,
			expectedCount: 0,
		},
		{
			name:   "invalid status filtered out",
			page:   1,
			count:  10,
			taskID: 0,
			jobID:  "",
			status: "InvalidStatus",
			repo: &mockScheduledTaskLogRepository{
				logs: []models.ScheduledTaskLogListItem{},
			},
			expectedError: false,
			expectedCount: 0,
		},
		{
			name:   "repository error",
			page:   1,
			count:  10,
			taskID: 0,
			jobID:  "",
			status: "",
			repo: &mockScheduledTaskLogRepository{
				err: errors.New("database error"),
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := zap.NewNop()
			svc := NewTaskLogService(tt.repo, logger)

			ctx := context.Background()
			result, err := svc.GetAll(ctx, tt.page, tt.count, tt.taskID, tt.jobID, tt.status)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Len(t, result, tt.expectedCount)
			}
		})
	}
}
