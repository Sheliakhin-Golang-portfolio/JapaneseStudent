package services

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/japanesestudent/task-service/internal/models"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
)

const (
	scheduledTasksZSet = "scheduled_tasks"
)

type ScheduledTaskRepository interface {
	Create(ctx context.Context, task *models.ScheduledTask) error
	GetByID(ctx context.Context, id int) (*models.ScheduledTask, error)
	GetAll(ctx context.Context, page, count, userID, templateID int, active *bool) ([]models.ScheduledTaskListItem, error)
	Update(ctx context.Context, id int, task *models.UpdateScheduledTaskRequest) error
	Delete(ctx context.Context, id int) error
	GetURLByID(ctx context.Context, id int) (string, error)
	GetTemplateIDByID(ctx context.Context, id int) (*int, error)
	GetContentByID(ctx context.Context, id int) (string, error)
}

type scheduledTaskService struct {
	repo         ScheduledTaskRepository
	templateRepo EmailTemplateRepository
	redis        *redis.Client
	logger       *zap.Logger
}

// NewScheduledTaskService creates a new scheduled task service
func NewScheduledTaskService(repo ScheduledTaskRepository, templateRepo EmailTemplateRepository, redis *redis.Client, logger *zap.Logger) *scheduledTaskService {
	return &scheduledTaskService{
		repo:         repo,
		templateRepo: templateRepo,
		redis:        redis,
		logger:       logger,
	}
}

// Create creates a new scheduled task
func (s *scheduledTaskService) Create(ctx context.Context, req *models.CreateScheduledTaskRequest) (int, error) {
	templateID, nextRun, err := s.checkCreateScheduledTaskValidation(ctx, req)
	if err != nil {
		return 0, err
	}

	task := &models.ScheduledTask{
		UserID:     req.UserID,
		TemplateID: templateID,
		URL:        req.URL,
		Content:    req.Content,
		NextRun:    nextRun,
		Cron:       req.Cron,
	}

	if err := s.repo.Create(ctx, task); err != nil {
		return 0, fmt.Errorf("failed to create scheduled task: %w", err)
	}

	// Add to Redis ZSET
	if err := s.addToRedisZSet(ctx, &task.NextRun, &task.ID); err != nil {
		return 0, fmt.Errorf("failed to add to Redis ZSET: %w", err)
	}

	return task.ID, nil
}

// checkCreateScheduledTaskValidation checks the validity of the scheduled task creation request
func (s *scheduledTaskService) checkCreateScheduledTaskValidation(ctx context.Context, req *models.CreateScheduledTaskRequest) (*int, time.Time, error) {
	// Validate that at least URL or EmailSlug is provided
	if req.URL == "" && req.EmailSlug == "" {
		return nil, time.Time{}, fmt.Errorf("either URL or EmailSlug must be provided")
	}

	errChan := make(chan error, 4)
	templateIDChan := make(chan *int, 1)
	nextRunChan := make(chan time.Time, 1)

	// Check if email slug is provided and get the template ID
	go func() {
		if req.EmailSlug != "" {
			templateID, err := s.templateRepo.GetIDBySlug(ctx, req.EmailSlug)
			if err != nil {
				errChan <- err
				templateIDChan <- nil
				return
			}
			if templateID == 0 {
				errChan <- fmt.Errorf("email template not found")
				templateIDChan <- nil
				return
			}
			errChan <- nil
			templateIDChan <- &templateID
			return
		}
		errChan <- nil
		templateIDChan <- nil
	}()

	// Check if content is provided and is valid
	go func() {
		if req.Content == "" && req.EmailSlug != "" {
			errChan <- fmt.Errorf("content is required")
			return
		} else if req.Content != "" {
			templateVars := strings.Split(req.Content, ";")
			if !emailRegex.MatchString(templateVars[0]) {
				errChan <- fmt.Errorf("email is invalid")
				return
			}
		}
		errChan <- nil
	}()

	// Check if cron expression is provided and is valid
	go func() {
		// Calculate next run from current time
		nextRun, err := CalculateNextRun(req.Cron, time.Now())
		if err != nil {
			errChan <- fmt.Errorf("failed to calculate next run: %w", err)
			nextRunChan <- time.Time{}
			return
		}
		nextRunChan <- nextRun
		errChan <- nil
	}()

	// Check if user id is provided and is valid
	go func() {
		if req.UserID != nil && *req.UserID <= 0 {
			errChan <- fmt.Errorf("user id is invalid")
			return
		}
		errChan <- nil
	}()

	for range 4 {
		err := <-errChan
		if err != nil {
			return nil, time.Time{}, err
		}
	}

	return <-templateIDChan, <-nextRunChan, nil
}

// Create creates a new scheduled task
func (s *scheduledTaskService) CreateAdmin(ctx context.Context, req *models.AdminCreateScheduledTaskRequest) (int, error) {
	nextRun, err := s.checkCreateAdminScheduledTaskValidation(ctx, req)
	if err != nil {
		return 0, err
	}

	task := &models.ScheduledTask{
		UserID:     req.UserID,
		TemplateID: req.TemplateID,
		URL:        req.URL,
		Content:    req.Content,
		NextRun:    nextRun,
		Cron:       req.Cron,
	}

	if err := s.repo.Create(ctx, task); err != nil {
		return 0, fmt.Errorf("failed to create scheduled task: %w", err)
	}

	// Add to Redis ZSET
	if err := s.addToRedisZSet(ctx, &task.NextRun, &task.ID); err != nil {
		return 0, fmt.Errorf("failed to add to Redis ZSET: %w", err)
	}

	return task.ID, nil
}

// checkCreateAdminScheduledTaskValidation checks the validity of the scheduled task creation request
func (s *scheduledTaskService) checkCreateAdminScheduledTaskValidation(ctx context.Context, req *models.AdminCreateScheduledTaskRequest) (time.Time, error) {
	// Validate that at least URL or EmailSlug is provided
	if req.URL == "" && req.TemplateID == nil {
		return time.Time{}, fmt.Errorf("either URL or TemplateID must be provided")
	}

	errChan := make(chan error, 4)
	nextRunChan := make(chan time.Time, 1)

	// Check if email slug is provided and get the template ID
	go func() {
		if req.TemplateID != nil {
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
		if req.Content == "" && req.TemplateID != nil {
			errChan <- fmt.Errorf("content is required")
			return
		} else if req.Content != "" {
			templateVars := strings.Split(req.Content, ";")
			if !emailRegex.MatchString(templateVars[0]) {
				errChan <- fmt.Errorf("email is invalid")
				return
			}
		}
		errChan <- nil
	}()

	// Check if cron expression is provided and is valid
	go func() {
		// Calculate next run from current time
		nextRun, err := CalculateNextRun(req.Cron, time.Now())
		if err != nil {
			errChan <- fmt.Errorf("failed to calculate next run: %w", err)
			nextRunChan <- time.Time{}
			return
		}
		nextRunChan <- nextRun
		errChan <- nil
	}()

	// Check if user id is provided and is valid
	go func() {
		if req.UserID != nil && *req.UserID <= 0 {
			errChan <- fmt.Errorf("user id is invalid")
			return
		}
		errChan <- nil
	}()

	for range 4 {
		err := <-errChan
		if err != nil {
			return time.Time{}, err
		}
	}

	return <-nextRunChan, nil
}

// GetByID retrieves a scheduled task by ID
func (s *scheduledTaskService) GetByID(ctx context.Context, id int) (*models.ScheduledTask, error) {
	return s.repo.GetByID(ctx, id)
}

// GetAll retrieves a paginated list of scheduled tasks
func (s *scheduledTaskService) GetAll(ctx context.Context, page, count, userID, templateID int, active *bool) ([]models.ScheduledTaskListItem, error) {
	if page < 1 {
		page = 1
	}
	if count < 1 {
		count = 20
	}

	return s.repo.GetAll(ctx, page, count, userID, templateID, active)
}

// Update updates a scheduled task
func (s *scheduledTaskService) Update(ctx context.Context, id int, req *models.UpdateScheduledTaskRequest) error {
	if err := s.checkUpdateScheduledTaskValidation(ctx, id, req); err != nil {
		return err
	}

	if err := s.repo.Update(ctx, id, req); err != nil {
		return err
	}

	// Handle ZSET updates
	if req.Active != nil {
		member := strconv.Itoa(id)
		if !*req.Active {
			// Remove from ZSET
			return s.redis.ZRem(ctx, scheduledTasksZSet, member).Err()
		} else {
			// Add to ZSET with NextRun as score
			var nextRun time.Time
			if req.NextRun != nil {
				nextRun = *req.NextRun
			} else {
				task, err := s.repo.GetByID(ctx, id)
				if err != nil {
					return err
				}
				nextRun = task.NextRun
			}
			score := float64(nextRun.Unix())
			return s.redis.ZAdd(ctx, scheduledTasksZSet, &redis.Z{
				Score:  score,
				Member: member,
			}).Err()
		}
	} else if req.NextRun != nil {
		// Update score in ZSET
		member := strconv.Itoa(id)
		score := float64(req.NextRun.Unix())
		return s.redis.ZAdd(ctx, scheduledTasksZSet, &redis.Z{
			Score:  score,
			Member: member,
		}).Err()
	}
	return nil
}

// checkUpdateScheduledTaskValidation checks the validity of the scheduled task update request
func (s *scheduledTaskService) checkUpdateScheduledTaskValidation(ctx context.Context, id int, req *models.UpdateScheduledTaskRequest) error {
	errChan := make(chan error, 5)

	// Get template id in advance to avoid multiple database queries
	templateID, err := s.repo.GetTemplateIDByID(ctx, id)
	if err != nil {
		return err
	}

	// Check if user id is provided and is valid (can be 0 if we want to nullify the user id)
	go func() {
		if req.UserID != nil && *req.UserID < 0 {
			errChan <- fmt.Errorf("user id is invalid")
			return
		}
		errChan <- nil
	}()

	// Check if template id is provided, valid and necessary
	go func() {
		if req.TemplateID != nil {
			if *req.TemplateID < 0 {
				errChan <- fmt.Errorf("template id is invalid")
				return
			} else if *req.TemplateID == 0 { // If template id set to 0, it means no template is used
				url, err := s.repo.GetURLByID(ctx, id)
				if err != nil {
					errChan <- err
					return
				}
				if url == "" && (req.URL == nil || *req.URL == "") {
					errChan <- fmt.Errorf("url is required if template id is not provided")
					return
				}
			} else {
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
		}
		errChan <- nil
	}()

	// Check if url is provided and necessary
	go func() {
		if req.URL != nil && *req.URL == "" {
			if templateID == nil && (req.TemplateID == nil || *req.TemplateID == 0) {
				errChan <- fmt.Errorf("template id is required if url is not provided")
				return
			}
		}
		errChan <- nil
	}()

	// Check if content is provided and necessary
	go func() {
		// If we assign a template id, we need to make sure that we have a content
		content, err := s.repo.GetContentByID(ctx, id)
		if err != nil {
			errChan <- err
			return
		}
		if req.Content == nil && content == "" && req.TemplateID != nil && *req.TemplateID > 0 {
			errChan <- fmt.Errorf("You provided a template id, but didn't provide a content")
			return
		}

		// If we want to nullify the content, we need to make sure that we will not use it
		if req.Content != nil && *req.Content == "" && (templateID != nil || (req.TemplateID != nil && *req.TemplateID > 0)) {
			errChan <- fmt.Errorf("content is required if template id is provided")
			return
		}

		// If we provided a content, we need to check if it is valid
		if req.Content != nil && *req.Content != "" {
			templateVars := strings.Split(*req.Content, ";")
			if !emailRegex.MatchString(templateVars[0]) {
				errChan <- fmt.Errorf("email is invalid")
				return
			}
		}
		errChan <- nil
	}()

	// Check if cron expression is provided and is valid
	go func() {
		if req.Cron != "" {
			// Calculate next run to check if cron expression is valid
			if _, err := CalculateNextRun(req.Cron, time.Now()); err != nil {
				errChan <- fmt.Errorf("invalid cron expression: %s", req.Cron)
				return
			}
		}
		errChan <- nil
	}()

	for range 5 {
		err := <-errChan
		if err != nil {
			return err
		}
	}
	return nil
}

// Delete deletes a scheduled task by ID from database and Redis ZSET
func (s *scheduledTaskService) Delete(ctx context.Context, id int) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
	return s.redis.ZRem(ctx, scheduledTasksZSet, strconv.Itoa(id)).Err()
}

// CalculateNextRun calculates the next run time from a cron expression
func CalculateNextRun(cronExpr string, fromTime time.Time) (time.Time, error) {
	schedule, err := cron.ParseStandard(cronExpr)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid cron expression: %w", err)
	}

	return schedule.Next(fromTime), nil
}

func (s *scheduledTaskService) addToRedisZSet(ctx context.Context, nextRun *time.Time, id *int) error {
	score := float64((*nextRun).Unix())
	member := strconv.Itoa(*id)
	return s.redis.ZAdd(ctx, scheduledTasksZSet, &redis.Z{
		Score:  score,
		Member: member,
	}).Err()
}
