package services

import (
	"context"
	"errors"
	"testing"

	"github.com/hibiken/asynq"
	"github.com/japanesestudent/task-service/internal/models"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

// mockImmediateTaskRepository is a mock implementation of ImmediateTaskRepository
type mockImmediateTaskRepository struct {
	task  *models.ImmediateTask
	tasks []models.ImmediateTaskListItem
	err   error
}

func (m *mockImmediateTaskRepository) GetAll(ctx context.Context, page, count int, userID, templateID int, status string) ([]models.ImmediateTaskListItem, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.tasks, nil
}

func (m *mockImmediateTaskRepository) GetByID(ctx context.Context, id int) (*models.ImmediateTask, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.task, nil
}

func (m *mockImmediateTaskRepository) Create(ctx context.Context, task *models.ImmediateTask) error {
	if m.err != nil {
		return m.err
	}
	task.ID = 1
	return nil
}

func (m *mockImmediateTaskRepository) Update(ctx context.Context, task *models.ImmediateTask) error {
	if m.err != nil {
		return m.err
	}
	return nil
}

func (m *mockImmediateTaskRepository) Delete(ctx context.Context, id int) error {
	if m.err != nil {
		return m.err
	}
	return nil
}

// mockEmailTemplateRepositoryForImmediateTask is a mock implementation of EmailTemplateRepository for immediate tasks
type mockEmailTemplateRepositoryForImmediateTask struct {
	templateID int
	exists     bool
	err        error
}

func (m *mockEmailTemplateRepositoryForImmediateTask) GetIDBySlug(ctx context.Context, slug string) (int, error) {
	if m.err != nil {
		return 0, m.err
	}
	return m.templateID, nil
}

func (m *mockEmailTemplateRepositoryForImmediateTask) ExistsByID(ctx context.Context, id int) (bool, error) {
	if m.err != nil {
		return false, m.err
	}
	return m.exists, nil
}

// Note: asynq.Client is a concrete struct, so we can't easily mock it.
// For unit tests, we'll skip testing the enqueue functionality and test it in integration tests.
// The tests that require asynq will be marked to skip or use a real client.

func TestNewImmediateTaskService(t *testing.T) {
	repo := &mockImmediateTaskRepository{}
	templateRepo := &mockEmailTemplateRepositoryForImmediateTask{}
	asynqClient := (*asynq.Client)(nil) // Not used in constructor test
	logger := zap.NewNop()

	svc := NewImmediateTaskService(repo, templateRepo, asynqClient, logger)

	assert.NotNil(t, svc)
	assert.Equal(t, repo, svc.repo)
	assert.Equal(t, templateRepo, svc.templateRepo)
}

func TestImmediateTaskService_Create(t *testing.T) {
	templateID := 1
	tests := []struct {
		name          string
		req           *models.CreateImmediateTaskRequest
		repo          *mockImmediateTaskRepository
		templateRepo  *mockEmailTemplateRepositoryForImmediateTask
		asynqClient   *asynq.Client
		expectedError bool
		expectedID    int
	}{
		{
			name: "success",
			req: &models.CreateImmediateTaskRequest{
				UserID:    100,
				EmailSlug: "test-slug",
				Content:   "test@example.com;John",
			},
			repo: &mockImmediateTaskRepository{},
			templateRepo: &mockEmailTemplateRepositoryForImmediateTask{
				templateID: templateID,
			},
			asynqClient:   nil, // Will be tested in integration tests
			expectedError: false,
			expectedID:    1,
		},
		{
			name: "missing email slug",
			req: &models.CreateImmediateTaskRequest{
				UserID:  100,
				Content: "test@example.com;John",
			},
			repo:          &mockImmediateTaskRepository{},
			templateRepo:  &mockEmailTemplateRepositoryForImmediateTask{},
			asynqClient:   nil, // Will be tested in integration tests
			expectedError: true,
		},
		{
			name: "invalid user ID",
			req: &models.CreateImmediateTaskRequest{
				UserID:    -1, // Negative user ID should fail validation
				EmailSlug: "test-slug",
				Content:   "test@example.com;John",
			},
			repo:          &mockImmediateTaskRepository{},
			templateRepo:  &mockEmailTemplateRepositoryForImmediateTask{},
			asynqClient:   nil, // Will be tested in integration tests
			expectedError: true,
		},
		{
			name: "missing content",
			req: &models.CreateImmediateTaskRequest{
				UserID:    100,
				EmailSlug: "test-slug",
				Content:   "",
			},
			repo:          &mockImmediateTaskRepository{},
			templateRepo:  &mockEmailTemplateRepositoryForImmediateTask{},
			asynqClient:   nil, // Will be tested in integration tests
			expectedError: true,
		},
		{
			name: "invalid email",
			req: &models.CreateImmediateTaskRequest{
				UserID:    100,
				EmailSlug: "test-slug",
				Content:   "invalid-email;John",
			},
			repo:          &mockImmediateTaskRepository{},
			templateRepo:  &mockEmailTemplateRepositoryForImmediateTask{},
			asynqClient:   nil, // Will be tested in integration tests
			expectedError: true,
		},
		{
			name: "template not found",
			req: &models.CreateImmediateTaskRequest{
				UserID:    100,
				EmailSlug: "non-existent",
				Content:   "test@example.com;John",
			},
			repo: &mockImmediateTaskRepository{},
			templateRepo: &mockEmailTemplateRepositoryForImmediateTask{
				err: errors.New("email template not found"),
			},
			asynqClient:   nil, // Will be tested in integration tests
			expectedError: true,
		},
		{
			name: "repository error on create",
			req: &models.CreateImmediateTaskRequest{
				UserID:    100,
				EmailSlug: "test-slug",
				Content:   "test@example.com;John",
			},
			repo: &mockImmediateTaskRepository{
				err: errors.New("database error"),
			},
			templateRepo: &mockEmailTemplateRepositoryForImmediateTask{
				templateID: templateID,
			},
			asynqClient:   nil, // Will be tested in integration tests
			expectedError: true,
		},
		// Note: asynq error test is skipped - will be tested in integration tests
		// since asynq.Client is a concrete struct and can't be easily mocked
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.asynqClient == nil && !tt.expectedError {
				// Skip tests that require asynq client - will be tested in integration tests
				t.Skip("Skipping test that requires asynq client - test in integration tests")
				return
			}
			logger := zap.NewNop()
			svc := NewImmediateTaskService(tt.repo, tt.templateRepo, tt.asynqClient, logger)

			ctx := context.Background()
			id, err := svc.Create(ctx, tt.req)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Equal(t, 0, id)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedID, id)
			}
		})
	}
}

func TestImmediateTaskService_CreateAdmin(t *testing.T) {
	templateID := 1
	tests := []struct {
		name          string
		req           *models.AdminCreateImmediateTaskRequest
		repo          *mockImmediateTaskRepository
		templateRepo  *mockEmailTemplateRepositoryForImmediateTask
		asynqClient   *asynq.Client
		expectedError bool
		expectedID    int
	}{
		{
			name: "success",
			req: &models.AdminCreateImmediateTaskRequest{
				UserID:     100,
				TemplateID: templateID,
				Content:    "test@example.com;John",
			},
			repo: &mockImmediateTaskRepository{},
			templateRepo: &mockEmailTemplateRepositoryForImmediateTask{
				exists: true,
			},
			asynqClient:   nil, // Will be tested in integration tests
			expectedError: false,
			expectedID:    1,
		},
		{
			name: "invalid template ID",
			req: &models.AdminCreateImmediateTaskRequest{
				UserID:     100,
				TemplateID: 0,
				Content:    "test@example.com;John",
			},
			repo:          &mockImmediateTaskRepository{},
			templateRepo:  &mockEmailTemplateRepositoryForImmediateTask{},
			asynqClient:   nil, // Will be tested in integration tests
			expectedError: true,
		},
		{
			name: "template not found",
			req: &models.AdminCreateImmediateTaskRequest{
				UserID:     100,
				TemplateID: templateID,
				Content:    "test@example.com;John",
			},
			repo: &mockImmediateTaskRepository{},
			templateRepo: &mockEmailTemplateRepositoryForImmediateTask{
				exists: false,
			},
			asynqClient:   nil, // Will be tested in integration tests
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.asynqClient == nil && !tt.expectedError {
				// Skip tests that require asynq client - will be tested in integration tests
				t.Skip("Skipping test that requires asynq client - test in integration tests")
				return
			}
			logger := zap.NewNop()
			svc := NewImmediateTaskService(tt.repo, tt.templateRepo, tt.asynqClient, logger)

			ctx := context.Background()
			id, err := svc.CreateAdmin(ctx, tt.req)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Equal(t, 0, id)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedID, id)
			}
		})
	}
}

func TestImmediateTaskService_GetByID(t *testing.T) {
	tests := []struct {
		name          string
		id            int
		repo          *mockImmediateTaskRepository
		expectedError bool
	}{
		{
			name: "success",
			id:   1,
			repo: &mockImmediateTaskRepository{
				task: &models.ImmediateTask{
					ID:      1,
					UserID:  100,
					Content: "test@example.com;John",
					Status:  models.ImmediateTaskStatusEnqueued,
				},
			},
			expectedError: false,
		},
		{
			name: "not found",
			id:   999,
			repo: &mockImmediateTaskRepository{
				err: errors.New("immediate task not found"),
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := zap.NewNop()
			templateRepo := &mockEmailTemplateRepositoryForImmediateTask{}
			asynqClient := (*asynq.Client)(nil) // Not needed for GetByID
			svc := NewImmediateTaskService(tt.repo, templateRepo, asynqClient, logger)

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

func TestImmediateTaskService_GetAll(t *testing.T) {
	tests := []struct {
		name          string
		page          int
		count         int
		userID        int
		templateID    int
		status        string
		repo          *mockImmediateTaskRepository
		expectedError bool
		expectedCount int
	}{
		{
			name:       "success",
			page:       1,
			count:       10,
			userID:     0,
			templateID: 0,
			status:     "",
			repo: &mockImmediateTaskRepository{
				tasks: []models.ImmediateTaskListItem{
					{ID: 1, UserID: 100, Status: models.ImmediateTaskStatusEnqueued},
					{ID: 2, UserID: 101, Status: models.ImmediateTaskStatusCompleted},
				},
			},
			expectedError: false,
			expectedCount: 2,
		},
		{
			name:       "default page and count",
			page:       0,
			count:       0,
			userID:     0,
			templateID: 0,
			status:     "",
			repo: &mockImmediateTaskRepository{
				tasks: []models.ImmediateTaskListItem{},
			},
			expectedError: false,
			expectedCount: 0,
		},
		{
			name:       "invalid status filtered out",
			page:       1,
			count:       10,
			userID:     0,
			templateID: 0,
			status:     "InvalidStatus",
			repo: &mockImmediateTaskRepository{
				tasks: []models.ImmediateTaskListItem{},
			},
			expectedError: false,
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := zap.NewNop()
			templateRepo := &mockEmailTemplateRepositoryForImmediateTask{}
			asynqClient := (*asynq.Client)(nil) // Not needed for GetByID
			svc := NewImmediateTaskService(tt.repo, templateRepo, asynqClient, logger)

			ctx := context.Background()
			result, err := svc.GetAll(ctx, tt.page, tt.count, tt.userID, tt.templateID, tt.status)

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

func TestImmediateTaskService_Delete(t *testing.T) {
	tests := []struct {
		name          string
		id            int
		repo          *mockImmediateTaskRepository
		expectedError bool
	}{
		{
			name: "success",
			id:   1,
			repo: &mockImmediateTaskRepository{},
			expectedError: false,
		},
		{
			name: "repository error",
			id:   1,
			repo: &mockImmediateTaskRepository{
				err: errors.New("database error"),
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := zap.NewNop()
			templateRepo := &mockEmailTemplateRepositoryForImmediateTask{}
			asynqClient := (*asynq.Client)(nil) // Not needed for GetByID
			svc := NewImmediateTaskService(tt.repo, templateRepo, asynqClient, logger)

			ctx := context.Background()
			err := svc.Delete(ctx, tt.id)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
