package services

import (
	"context"
	"fmt"

	"github.com/Sheliakhin-Golang-portfolio/JapaneseStudent/services/task-service/internal/models"
	"go.uber.org/zap"
)

// ScheduledTaskLogRepository is the interface that wraps methods for ScheduledTaskLog table data access
type ScheduledTaskLogRepository interface {
	Create(ctx context.Context, log *models.ScheduledTaskLog) error
	GetByID(ctx context.Context, id int) (*models.ScheduledTaskLog, error)
	GetAll(ctx context.Context, page, count, taskID int, jobID, status string) ([]models.ScheduledTaskLogListItem, error)
}

type scheduledTaskLogService struct {
	repo   ScheduledTaskLogRepository
	logger *zap.Logger
}

// NewTaskLogService creates a new task log service
func NewTaskLogService(repo ScheduledTaskLogRepository, logger *zap.Logger) *scheduledTaskLogService {
	return &scheduledTaskLogService{
		repo:   repo,
		logger: logger,
	}
}

// Create creates a new scheduled task log
func (s *scheduledTaskLogService) Create(ctx context.Context, log *models.ScheduledTaskLog) error {
	if err := s.repo.Create(ctx, log); err != nil {
		return fmt.Errorf("failed to create scheduled task log: %w", err)
	}
	return nil
}

// GetByID retrieves a scheduled task log by ID
func (s *scheduledTaskLogService) GetByID(ctx context.Context, id int) (*models.ScheduledTaskLog, error) {
	return s.repo.GetByID(ctx, id)
}

// GetAll retrieves a paginated list of scheduled task logs
func (s *scheduledTaskLogService) GetAll(ctx context.Context, page, count int, taskID int, jobID, status string) ([]models.ScheduledTaskLogListItem, error) {
	if page < 1 {
		page = 1
	}
	if count < 1 {
		count = 20
	}

	if status != "" && status != string(models.ScheduledTaskLogStatusCompleted) &&
		status != string(models.ScheduledTaskLogStatusFailed) {
		status = ""
	}

	return s.repo.GetAll(ctx, page, count, taskID, jobID, status)
}
