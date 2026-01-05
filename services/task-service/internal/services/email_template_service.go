package services

import (
	"context"
	"fmt"

	"github.com/japanesestudent/task-service/internal/models"
	"go.uber.org/zap"
)

// EmailTemplateRepository is the interface that wraps methods for email template data access
type EmailAdminTemplateRepository interface {
	Create(ctx context.Context, template *models.EmailTemplate) error
	GetByID(ctx context.Context, id int) (*models.EmailTemplate, error)
	GetAll(ctx context.Context, page, count int, search string) ([]models.EmailTemplateListItem, error)
	Update(ctx context.Context, id int, template *models.EmailTemplate) error
	Delete(ctx context.Context, id int) error
	ExistsBySlug(ctx context.Context, slug string) (bool, error)
}

type emailTemplateService struct {
	repo   EmailAdminTemplateRepository
	logger *zap.Logger
}

// NewEmailTemplateService creates a new email template service
func NewEmailTemplateService(repo EmailAdminTemplateRepository, logger *zap.Logger) *emailTemplateService {
	return &emailTemplateService{
		repo:   repo,
		logger: logger,
	}
}

// Create creates a new email template
func (s *emailTemplateService) Create(ctx context.Context, req *models.CreateUpdateEmailTemplateRequest) (int, error) {
	// Check if slug is unique
	exists, err := s.repo.ExistsBySlug(ctx, req.Slug)
	if err != nil {
		return 0, err
	}
	if exists {
		return 0, fmt.Errorf("email template with slug '%s' already exists", req.Slug)
	}

	template := &models.EmailTemplate{
		Slug:            req.Slug,
		SubjectTemplate: req.SubjectTemplate,
		BodyTemplate:    req.BodyTemplate,
	}

	if err := s.repo.Create(ctx, template); err != nil {
		return 0, err
	}

	return template.ID, nil
}

// GetByID retrieves an email template by ID
func (s *emailTemplateService) GetByID(ctx context.Context, id int) (*models.EmailTemplate, error) {
	return s.repo.GetByID(ctx, id)
}

// GetAll retrieves a paginated list of email templates
func (s *emailTemplateService) GetAll(ctx context.Context, page, count int, search string) ([]models.EmailTemplateListItem, error) {
	if page < 1 {
		page = 1
	}
	if count < 1 {
		count = 20
	}

	return s.repo.GetAll(ctx, page, count, search)
}

// Update updates an email template
func (s *emailTemplateService) Update(ctx context.Context, id int, req *models.CreateUpdateEmailTemplateRequest) error {
	// Check if slug is unique (if provided)
	if req.Slug != "" {
		exists, err := s.repo.ExistsBySlug(ctx, req.Slug)
		if err != nil {
			return err
		}
		if exists {
			return fmt.Errorf("email template with slug '%s' already exists", req.Slug)
		}
	}

	// Update fields if provided
	template := &models.EmailTemplate{
		ID:              id,
		Slug:            req.Slug,
		SubjectTemplate: req.SubjectTemplate,
		BodyTemplate:    req.BodyTemplate,
	}

	return s.repo.Update(ctx, id, template)
}

// Delete deletes an email template
func (s *emailTemplateService) Delete(ctx context.Context, id int) error {
	return s.repo.Delete(ctx, id)
}
