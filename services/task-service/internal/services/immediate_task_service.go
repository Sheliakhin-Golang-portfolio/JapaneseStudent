package services

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/hibiken/asynq"
	"github.com/japanesestudent/task-service/internal/models"
	"go.uber.org/zap"
)

type ImmediateTaskRepository interface {
	GetAll(ctx context.Context, page, count int, userID, templateID int, status string) ([]models.ImmediateTaskListItem, error)
	GetByID(ctx context.Context, id int) (*models.ImmediateTask, error)
	Create(ctx context.Context, task *models.ImmediateTask) error
	Update(ctx context.Context, task *models.ImmediateTask) error
	Delete(ctx context.Context, id int) error
}

type EmailTemplateRepository interface {
	GetIDBySlug(ctx context.Context, slug string) (int, error)
	ExistsByID(ctx context.Context, id int) (bool, error)
}

// emailRegex validates email format
var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

type immediateTaskService struct {
	repo         ImmediateTaskRepository
	templateRepo EmailTemplateRepository
	asynqClient  *asynq.Client
	logger       *zap.Logger
}

// NewImmediateTaskService creates a new immediate task service
func NewImmediateTaskService(repo ImmediateTaskRepository, templateRepo EmailTemplateRepository, asynqClient *asynq.Client, logger *zap.Logger) *immediateTaskService {
	return &immediateTaskService{
		repo:         repo,
		templateRepo: templateRepo,
		asynqClient:  asynqClient,
		logger:       logger,
	}
}

// Create creates a new immediate task
func (s *immediateTaskService) Create(ctx context.Context, req *models.CreateImmediateTaskRequest) (int, error) {
	templateID, err := s.checkCreateImmediateTaskValidation(ctx, req)
	if err != nil {
		return 0, err
	}

	task := &models.ImmediateTask{
		UserID:     req.UserID,
		TemplateID: &templateID,
		Content:    req.Content,
	}

	if err := s.repo.Create(ctx, task); err != nil {
		return 0, fmt.Errorf("failed to create immediate task: %w", err)
	}

	// Enqueue task to immediate queue
	if err := s.addToQueue(ctx, task); err != nil {
		return 0, fmt.Errorf("failed to enqueue immediate task: %w", err)
	}

	return task.ID, nil
}

// checkCreateImmediateTaskValidation checks the validity of the immediate task creation request
func (s *immediateTaskService) checkCreateImmediateTaskValidation(ctx context.Context, req *models.CreateImmediateTaskRequest) (int, error) {
	errChan := make(chan error, 3)
	templateIDChan := make(chan int, 1)

	// Check if email slug is provided and get the template ID
	go func() {
		if req.EmailSlug == "" {
			errChan <- fmt.Errorf("email slug is required")
			templateIDChan <- 0
			return
		}

		templateID, err := s.templateRepo.GetIDBySlug(ctx, req.EmailSlug)
		if err != nil {
			errChan <- err
			templateIDChan <- 0
			return
		}

		errChan <- nil
		templateIDChan <- templateID
	}()

	// Check if user ID is provided and is not negative
	go func() {
		if req.UserID < 0 {
			errChan <- fmt.Errorf("user ID is required")
			return
		}
		errChan <- nil
	}()

	// Check if content is provided and is valid
	go func() {
		if req.Content == "" {
			errChan <- fmt.Errorf("content is required")
			return
		}

		templateVars := strings.Split(req.Content, ";")
		if !emailRegex.MatchString(templateVars[0]) {
			errChan <- fmt.Errorf("email is invalid")
			return
		}

		errChan <- nil
	}()

	for range 3 {
		err := <-errChan
		if err != nil {
			return 0, err
		}
	}

	return <-templateIDChan, nil
}

func (s *immediateTaskService) CreateAdmin(ctx context.Context, req *models.AdminCreateImmediateTaskRequest) (int, error) {
	err := s.checkCreateAdminImmediateTaskValidation(ctx, req)
	if err != nil {
		return 0, err
	}

	task := &models.ImmediateTask{
		UserID:     req.UserID,
		TemplateID: &req.TemplateID,
		Content:    req.Content,
	}

	if err := s.repo.Create(ctx, task); err != nil {
		return 0, fmt.Errorf("failed to create immediate task: %w", err)
	}

	// Enqueue task to immediate queue
	if err := s.addToQueue(ctx, task); err != nil {
		return 0, fmt.Errorf("failed to enqueue immediate task: %w", err)
	}

	return task.ID, nil
}

// checkCreateAdminImmediateTaskValidation checks the validity of the immediate task creation request for admin endpoints
func (s *immediateTaskService) checkCreateAdminImmediateTaskValidation(ctx context.Context, req *models.AdminCreateImmediateTaskRequest) error {
	errChan := make(chan error, 3)

	// Check if email template Id is provided and exists
	go func() {
		if req.TemplateID <= 0 {
			errChan <- fmt.Errorf("email template ID is required")
			return
		}

		exists, err := s.templateRepo.ExistsByID(ctx, req.TemplateID)
		if err != nil {
			errChan <- err
			return
		}
		if !exists {
			errChan <- fmt.Errorf("email template not found")
			return
		}

		errChan <- nil
	}()

	// Check if user ID is provided and is not negative
	go func() {
		if req.UserID < 0 {
			errChan <- fmt.Errorf("user ID is required")
			return
		}
		errChan <- nil
	}()

	// Check if content is provided and is valid
	go func() {
		if req.Content == "" {
			errChan <- fmt.Errorf("content is required")
			return
		}

		templateVars := strings.Split(req.Content, ";")
		if !emailRegex.MatchString(templateVars[0]) {
			errChan <- fmt.Errorf("email is invalid")
			return
		}

		errChan <- nil
	}()

	for range 3 {
		err := <-errChan
		if err != nil {
			return err
		}
	}

	return nil
}

// GetByID retrieves an immediate task by ID
func (s *immediateTaskService) GetByID(ctx context.Context, id int) (*models.ImmediateTask, error) {
	return s.repo.GetByID(ctx, id)
}

// GetAll retrieves a paginated list of immediate tasks
func (s *immediateTaskService) GetAll(ctx context.Context, page, count int, userID, templateID int, status string) ([]models.ImmediateTaskListItem, error) {
	if page < 1 {
		page = 1
	}
	if count < 1 {
		count = 20
	}

	if status != "" && status != string(models.ImmediateTaskStatusEnqueued) &&
		status != string(models.ImmediateTaskStatusCompleted) &&
		status != string(models.ImmediateTaskStatusFailed) {
		status = ""
	}

	return s.repo.GetAll(ctx, page, count, userID, templateID, status)
}

// Update updates an immediate task
func (s *immediateTaskService) Update(ctx context.Context, id int, req *models.UpdateImmediateTaskRequest) error {
	if err := s.checkUpdateImmediateTaskValidation(ctx, req); err != nil {
		return err
	}

	task := &models.ImmediateTask{
		ID:         id,
		TemplateID: req.TemplateID,
		Content:    req.Content,
		Status:     req.Status,
		Error:      req.Error,
	}
	if req.UserID != nil {
		task.UserID = *req.UserID
	}

	return s.repo.Update(ctx, task)
}

// checkUpdateImmediateTaskValidation checks the validity of the immediate task update request
func (s *immediateTaskService) checkUpdateImmediateTaskValidation(ctx context.Context, req *models.UpdateImmediateTaskRequest) error {
	errChan := make(chan error, 4)

	// Check if user ID is provided and is not negative
	go func() {
		if req.UserID != nil && *req.UserID < 0 {
			errChan <- fmt.Errorf("user ID is invalid")
			return
		}
		errChan <- nil
	}()

	// Check if email template ID is provided and exists
	go func() {
		if req.TemplateID != nil {
			if *req.TemplateID < 0 {
				errChan <- fmt.Errorf("email template ID is required")
				return
			} else if *req.TemplateID == 0 { // 0 means we want to clear the template ID
				errChan <- nil
				return
			}
			exists, err := s.templateRepo.ExistsByID(ctx, *req.TemplateID)
			if err != nil {
				errChan <- err
				return
			}
			if !exists {
				errChan <- fmt.Errorf("email template not found")
				return
			}
		}
		errChan <- nil
	}()

	// Check if content is provided and is valid
	go func() {
		if req.Content != "" {
			templateVars := strings.Split(req.Content, ";")
			if !emailRegex.MatchString(templateVars[0]) {
				errChan <- fmt.Errorf("email is invalid")
				return
			}
		}

		errChan <- nil
	}()

	go func() {
		if req.Status != "" && req.Status != models.ImmediateTaskStatusEnqueued &&
			req.Status != models.ImmediateTaskStatusCompleted &&
			req.Status != models.ImmediateTaskStatusFailed {
			errChan <- fmt.Errorf("invalid status: %s", req.Status)
			return
		}
		errChan <- nil
	}()

	for range 4 {
		err := <-errChan
		if err != nil {
			return err
		}
	}

	return nil
}

// Delete deletes an immediate task
func (s *immediateTaskService) Delete(ctx context.Context, id int) error {
	return s.repo.Delete(ctx, id)
}

func (s *immediateTaskService) addToQueue(ctx context.Context, task *models.ImmediateTask) error {
	payload := []byte(strconv.Itoa(task.ID))
	asynqTask := asynq.NewTask("immediate:task", payload)
	if _, err := s.asynqClient.Enqueue(asynqTask, asynq.Queue("immediate")); err != nil {
		return err
	}
	return nil
}
