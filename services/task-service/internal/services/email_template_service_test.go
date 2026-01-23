package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Sheliakhin-Golang-portfolio/JapaneseStudent/services/task-service/internal/models"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

// mockEmailTemplateRepository is a mock implementation of EmailAdminTemplateRepository
type mockEmailTemplateRepository struct {
	template      *models.EmailTemplate
	templates     []models.EmailTemplateListItem
	exists        bool
	err           error
	createErr     error
	updateErr     error
	deleteErr     error
	existsErr     error
}

func (m *mockEmailTemplateRepository) Create(ctx context.Context, template *models.EmailTemplate) error {
	if m.createErr != nil {
		return m.createErr
	}
	if m.err != nil {
		return m.err
	}
	template.ID = 1
	return nil
}

func (m *mockEmailTemplateRepository) GetByID(ctx context.Context, id int) (*models.EmailTemplate, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.template, nil
}

func (m *mockEmailTemplateRepository) GetAll(ctx context.Context, page, count int, search string) ([]models.EmailTemplateListItem, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.templates, nil
}

func (m *mockEmailTemplateRepository) Update(ctx context.Context, id int, template *models.EmailTemplate) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	if m.err != nil {
		return m.err
	}
	return nil
}

func (m *mockEmailTemplateRepository) Delete(ctx context.Context, id int) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	if m.err != nil {
		return m.err
	}
	return nil
}

func (m *mockEmailTemplateRepository) ExistsBySlug(ctx context.Context, slug string) (bool, error) {
	if m.existsErr != nil {
		return false, m.existsErr
	}
	if m.err != nil {
		return false, m.err
	}
	return m.exists, nil
}

func TestNewEmailTemplateService(t *testing.T) {
	repo := &mockEmailTemplateRepository{}
	logger := zap.NewNop()

	svc := NewEmailTemplateService(repo, logger)

	assert.NotNil(t, svc)
	assert.Equal(t, repo, svc.repo)
}

func TestEmailTemplateService_Create(t *testing.T) {
	tests := []struct {
		name          string
		req           *models.CreateUpdateEmailTemplateRequest
		repo          *mockEmailTemplateRepository
		expectedError bool
		expectedID    int
	}{
		{
			name: "success",
			req: &models.CreateUpdateEmailTemplateRequest{
				Slug:            "test-slug",
				SubjectTemplate: "Subject {{.Name}}",
				BodyTemplate:    "Body {{.Name}}",
			},
			repo: &mockEmailTemplateRepository{
				exists: false,
			},
			expectedError: false,
			expectedID:    1,
		},
		{
			name: "slug already exists",
			req: &models.CreateUpdateEmailTemplateRequest{
				Slug:            "existing-slug",
				SubjectTemplate: "Subject",
				BodyTemplate:    "Body",
			},
			repo: &mockEmailTemplateRepository{
				exists: true,
			},
			expectedError: true,
		},
		{
			name: "repository error on exists check",
			req: &models.CreateUpdateEmailTemplateRequest{
				Slug:            "test-slug",
				SubjectTemplate: "Subject",
				BodyTemplate:    "Body",
			},
			repo: &mockEmailTemplateRepository{
				existsErr: errors.New("database error"),
			},
			expectedError: true,
		},
		{
			name: "repository error on create",
			req: &models.CreateUpdateEmailTemplateRequest{
				Slug:            "test-slug",
				SubjectTemplate: "Subject",
				BodyTemplate:    "Body",
			},
			repo: &mockEmailTemplateRepository{
				exists:    false,
				createErr: errors.New("database error"),
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := zap.NewNop()
			svc := NewEmailTemplateService(tt.repo, logger)

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

func TestEmailTemplateService_GetByID(t *testing.T) {
	tests := []struct {
		name          string
		id            int
		repo          *mockEmailTemplateRepository
		expectedError bool
		expected      *models.EmailTemplate
	}{
		{
			name: "success",
			id:   1,
			repo: &mockEmailTemplateRepository{
				template: &models.EmailTemplate{
					ID:              1,
					Slug:            "test-slug",
					SubjectTemplate: "Subject",
					BodyTemplate:    "Body",
					CreatedAt:       time.Now(),
					UpdatedAt:       time.Now(),
				},
			},
			expectedError: false,
		},
		{
			name: "not found",
			id:   999,
			repo: &mockEmailTemplateRepository{
				err: errors.New("email template not found"),
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := zap.NewNop()
			svc := NewEmailTemplateService(tt.repo, logger)

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

func TestEmailTemplateService_GetAll(t *testing.T) {
	tests := []struct {
		name          string
		page          int
		count         int
		search        string
		repo          *mockEmailTemplateRepository
		expectedError bool
		expectedCount int
	}{
		{
			name:   "success",
			page:   1,
			count:  10,
			search: "",
			repo: &mockEmailTemplateRepository{
				templates: []models.EmailTemplateListItem{
					{ID: 1, Slug: "test-slug-1"},
					{ID: 2, Slug: "test-slug-2"},
				},
			},
			expectedError: false,
			expectedCount: 2,
		},
		{
			name:   "default page and count",
			page:   0,
			count:  0,
			search: "",
			repo: &mockEmailTemplateRepository{
				templates: []models.EmailTemplateListItem{},
			},
			expectedError: false,
			expectedCount: 0,
		},
		{
			name:   "repository error",
			page:   1,
			count:  10,
			search: "",
			repo: &mockEmailTemplateRepository{
				err: errors.New("database error"),
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := zap.NewNop()
			svc := NewEmailTemplateService(tt.repo, logger)

			ctx := context.Background()
			result, err := svc.GetAll(ctx, tt.page, tt.count, tt.search)

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

func TestEmailTemplateService_Update(t *testing.T) {
	tests := []struct {
		name          string
		id            int
		req           *models.CreateUpdateEmailTemplateRequest
		repo          *mockEmailTemplateRepository
		expectedError bool
	}{
		{
			name: "success",
			id:   1,
			req: &models.CreateUpdateEmailTemplateRequest{
				Slug:            "new-slug",
				SubjectTemplate: "New Subject",
				BodyTemplate:    "New Body",
			},
			repo: &mockEmailTemplateRepository{
				exists: false,
			},
			expectedError: false,
		},
		{
			name: "slug already exists",
			id:   1,
			req: &models.CreateUpdateEmailTemplateRequest{
				Slug:            "existing-slug",
				SubjectTemplate: "Subject",
				BodyTemplate:    "Body",
			},
			repo: &mockEmailTemplateRepository{
				exists: true,
			},
			expectedError: true,
		},
		{
			name: "update without slug check",
			id:   1,
			req: &models.CreateUpdateEmailTemplateRequest{
				SubjectTemplate: "New Subject",
				BodyTemplate:    "New Body",
			},
			repo: &mockEmailTemplateRepository{},
			expectedError: false,
		},
		{
			name: "repository error on update",
			id:   1,
			req: &models.CreateUpdateEmailTemplateRequest{
				Slug: "new-slug",
			},
			repo: &mockEmailTemplateRepository{
				exists:    false,
				updateErr: errors.New("database error"),
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := zap.NewNop()
			svc := NewEmailTemplateService(tt.repo, logger)

			ctx := context.Background()
			err := svc.Update(ctx, tt.id, tt.req)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestEmailTemplateService_Delete(t *testing.T) {
	tests := []struct {
		name          string
		id            int
		repo          *mockEmailTemplateRepository
		expectedError bool
	}{
		{
			name: "success",
			id:   1,
			repo: &mockEmailTemplateRepository{},
			expectedError: false,
		},
		{
			name: "repository error",
			id:   1,
			repo: &mockEmailTemplateRepository{
				deleteErr: errors.New("database error"),
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := zap.NewNop()
			svc := NewEmailTemplateService(tt.repo, logger)

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
