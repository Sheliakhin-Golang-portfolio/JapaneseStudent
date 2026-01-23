package main

import (
	"context"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/hibiken/asynq"
	"github.com/Sheliakhin-Golang-portfolio/JapaneseStudent/services/task-service/internal/models"
	"go.uber.org/zap"
)

const (
	scheduledTasksZSet = "scheduled_tasks"
)

// ScheduledTaskRepository defines methods for scheduled task operations
type ScheduledTaskRepository interface {
	// GetActiveTasksForRestore takes active due tasks from database
	GetActiveTasksForRestore(ctx context.Context) ([]models.ScheduledTaskSetItem, error)
	// GetActiveTasksForNext24Hours takes active tasks for next 24 hours from database
	GetActiveTasksForNext24Hours(ctx context.Context) ([]models.ScheduledTaskSetItem, error)
}

// Scheduler manages scheduled task operations
type Scheduler struct {
	redis       *redis.Client
	asynqClient *asynq.Client
	logger      *zap.Logger
	ticker      *time.Ticker
	stopChan    chan struct{}
	taskRepo    ScheduledTaskRepository
}

// NewScheduler creates a new scheduler instance
func NewScheduler(redis *redis.Client, asynqClient *asynq.Client, logger *zap.Logger, taskRepo ScheduledTaskRepository) *Scheduler {
	return &Scheduler{
		redis:       redis,
		asynqClient: asynqClient,
		logger:      logger,
		ticker:      time.NewTicker(10 * time.Second),
		stopChan:    make(chan struct{}),
		taskRepo:    taskRepo,
	}
}

// Start starts the scheduler
func (s *Scheduler) Start() {
	s.logger.Info("Scheduler started")
	go s.run()
}

// Stop stops the scheduler
func (s *Scheduler) Stop() {
	s.ticker.Stop()
	close(s.stopChan)
	s.logger.Info("Scheduler stopped")
}

// run executes the scheduler loop
func (s *Scheduler) run() {
	ctx := context.Background()

	// Run immediately on start
	s.executeTasks(ctx)

	for {
		select {
		case <-s.ticker.C:
			s.executeTasks(ctx)
		case <-s.stopChan:
			return
		}
	}
}

// executeTasks runs all three scheduler functions
func (s *Scheduler) executeTasks(ctx context.Context) {
	s.restoreScheduledTasks(ctx)
	s.populateRedisSet(ctx)
	s.enqueueDueScheduledTasks(ctx)
}

// restoreScheduledTasks checks if Redis ZCard is empty, and if it is, method fills it with records from database
func (s *Scheduler) restoreScheduledTasks(ctx context.Context) {
	// Check if ZSET is empty
	card, err := s.redis.ZCard(ctx, scheduledTasksZSet).Result()
	if err != nil {
		s.logger.Error("Failed to check ZSET cardinality", zap.Error(err))
		return
	}

	if card > 0 {
		return // ZSET is not empty, skip restoration
	}

	// Query tasks for restoration
	tasks, err := s.taskRepo.GetActiveTasksForRestore(ctx)
	if err != nil {
		s.logger.Error("Failed to get tasks for restore", zap.Error(err))
		return
	}

	// Add tasks to ZSET
	for _, task := range tasks {
		score := float64(task.NextRun.Unix())
		member := strconv.Itoa(task.ID)
		err := s.redis.ZAdd(ctx, scheduledTasksZSet, &redis.Z{
			Score:  score,
			Member: member,
		}).Err()
		if err != nil {
			s.logger.Error("Failed to add task to ZSET", zap.Int("task_id", task.ID), zap.Error(err))
			continue
		}
		s.logger.Debug("Restored task to ZSET", zap.Int("task_id", task.ID), zap.Time("next_run", task.NextRun))
	}

	if len(tasks) > 0 {
		s.logger.Info("Restored scheduled tasks", zap.Int("count", len(tasks)))
	}
}

// PopulateRedisSet takes new or existing tasks for next 24 hours and adds them to ZSET
func (s *Scheduler) populateRedisSet(ctx context.Context) {
	// Query tasks for next 24 hours
	tasks, err := s.taskRepo.GetActiveTasksForNext24Hours(ctx)
	if err != nil {
		s.logger.Error("Failed to get tasks for next 24 hours", zap.Error(err))
		return
	}

	// Add tasks to ZSET
	for _, task := range tasks {
		score := float64(task.NextRun.Unix())
		member := strconv.Itoa(task.ID)
		err := s.redis.ZAdd(ctx, scheduledTasksZSet, &redis.Z{
			Score:  score,
			Member: member,
		}).Err()
		if err != nil {
			s.logger.Error("Failed to add task to ZSET", zap.Int("task_id", task.ID), zap.Error(err))
			continue
		}
		s.logger.Debug("Populated task to ZSET", zap.Int("task_id", task.ID), zap.Time("next_run", task.NextRun))
	}

	if len(tasks) > 0 {
		s.logger.Debug("Populated scheduled tasks", zap.Int("count", len(tasks)))
	}
}

// EnqueueDueScheduledTasks will move due tasks from ZSET to "default" queue
func (s *Scheduler) enqueueDueScheduledTasks(ctx context.Context) {
	now := time.Now()
	currentTimestamp := float64(now.Unix())

	// Get due tasks from ZSET
	members, err := s.redis.ZRangeByScore(ctx, scheduledTasksZSet, &redis.ZRangeBy{
		Min: "-inf",
		Max: strconv.FormatFloat(currentTimestamp, 'f', 0, 64),
	}).Result()
	if err != nil {
		s.logger.Error("Failed to get due tasks from ZSET", zap.Error(err))
		return
	}

	if len(members) == 0 {
		return // No due tasks
	}

	// Enqueue each task and remove from ZSET
	for _, member := range members {
		if err != nil {
			s.logger.Error("Failed to parse task ID", zap.String("member", member), zap.Error(err))
			continue
		}

		// Create task payload
		payload := []byte(member)
		task := asynq.NewTask("scheduled:task", payload)

		// Enqueue to default queue
		_, err = s.asynqClient.Enqueue(task, asynq.Queue("default"))
		if err != nil {
			s.logger.Error("Failed to enqueue task", zap.String("member", member), zap.Error(err))
			continue
		}

		// Remove from ZSET
		err = s.redis.ZRem(ctx, scheduledTasksZSet, member).Err()
		if err != nil {
			s.logger.Error("Failed to remove task from ZSET", zap.String("member", member), zap.Error(err))
			continue
		}

		s.logger.Info("Enqueued due scheduled task", zap.String("member", member))
	}
}
