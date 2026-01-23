package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-redis/redis/v8"
	"github.com/hibiken/asynq"
	"github.com/Sheliakhin-Golang-portfolio/JapaneseStudent/libs/handlers"
	"github.com/Sheliakhin-Golang-portfolio/JapaneseStudent/services/task-service/internal/models"
	"go.uber.org/zap"
)

// ImmediateTaskService is the interface that wraps methods for immediate task business logic
type ImmediateTaskService interface {
	// Create creates a new immediate task and register it in Asynq queue
	//
	// "ctx" parameter is used to specify the context.
	// "req" parameter is used to specify the immediate task creation request.
	//
	// If some error occurs during immediate task creation, the error will be returned.
	Create(ctx context.Context, req *models.CreateImmediateTaskRequest) (int, error)
}

// ScheduledTaskService is the interface that wraps methods for scheduled task business logic
type ScheduledTaskService interface {
	// Create creates a new scheduled task and register it in Redis ZSET
	//
	// "ctx" parameter is used to specify the context.
	// "req" parameter is used to specify the scheduled task creation request.
	//
	// If some error occurs during scheduled task creation, the error will be returned.
	Create(ctx context.Context, req *models.CreateScheduledTaskRequest) (int, error)
	// DeleteByUserID deletes all scheduled tasks for a user
	//
	// "ctx" parameter is used to specify the context.
	// "userID" parameter is used to identify the user.
	//
	// If some error occurs during task deletion, the error will be returned.
	DeleteByUserID(ctx context.Context, userID int) error
}

// TaskHandler handles task creation requests
type TaskHandler struct {
	handlers.BaseHandler
	immediateTaskService ImmediateTaskService
	scheduledTaskService ScheduledTaskService
	asynqClient          *asynq.Client
	redis                *redis.Client
}

// NewTaskHandler creates a new task handler
func NewTaskHandler(
	immediateTaskService ImmediateTaskService,
	scheduledTaskService ScheduledTaskService,
	asynqClient *asynq.Client,
	redis *redis.Client,
	logger *zap.Logger,
) *TaskHandler {
	return &TaskHandler{
		BaseHandler:          handlers.BaseHandler{Logger: logger},
		immediateTaskService: immediateTaskService,
		scheduledTaskService: scheduledTaskService,
		asynqClient:          asynqClient,
		redis:                redis,
	}
}

// RegisterRoutes registers task handler routes
func (h *TaskHandler) RegisterRoutes(r chi.Router) {
	r.Route("/tasks", func(r chi.Router) {
		r.Post("/immediate", h.CreateImmediateTask)
		r.Post("/scheduled", h.CreateScheduledTask)
		r.Delete("/scheduled/by-user", h.DeleteScheduledTaskByUserId)
	})
}

// CreateImmediateTask handles POST /tasks/immediate
// @Summary Create immediate task
// @Description Create a new immediate task and enqueue it for processing. Requires API key authentication.
// @Tags tasks
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param task body models.CreateImmediateTaskRequest true "Immediate task creation request"
// @Success 201 {object} map[string]any "Task created successfully"
// @Failure 400 {object} map[string]string "Bad request - invalid request body or task creation failed"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /tasks/immediate [post]
func (h *TaskHandler) CreateImmediateTask(w http.ResponseWriter, r *http.Request) {
	var req models.CreateImmediateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Create immediate task
	taskID, err := h.immediateTaskService.Create(r.Context(), &req)
	if err != nil {
		h.Logger.Error("failed to create immediate task", zap.Error(err))
		h.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}

	h.RespondJSON(w, http.StatusCreated, map[string]any{
		"message": "immediate task created successfully",
		"id":      taskID,
	})
}

// CreateScheduledTask handles POST /tasks/scheduled
// @Summary Create scheduled task
// @Description Create a new scheduled task with cron expression. Requires API key authentication.
// @Tags tasks
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param task body models.CreateScheduledTaskRequest true "Scheduled task creation request"
// @Success 201 {object} map[string]any "Task created successfully"
// @Failure 400 {object} map[string]string "Bad request - invalid request body or task creation failed"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /tasks/scheduled [post]
func (h *TaskHandler) CreateScheduledTask(w http.ResponseWriter, r *http.Request) {
	var req models.CreateScheduledTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Create scheduled task
	taskID, err := h.scheduledTaskService.Create(r.Context(), &req)
	if err != nil {
		h.Logger.Error("failed to create scheduled task", zap.Error(err))
		h.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}

	h.RespondJSON(w, http.StatusCreated, map[string]any{
		"message": "scheduled task created successfully",
		"id":      taskID,
	})
}

// DeleteScheduledTaskByUserIdRequest represents a request to delete scheduled tasks by user ID
type DeleteScheduledTaskByUserIdRequest struct {
	UserId int `json:"userId"`
}

// DeleteScheduledTaskByUserId handles DELETE /tasks/scheduled/by-user
// @Summary Delete scheduled tasks by user ID
// @Description Delete all scheduled tasks for a specific user. Requires API key authentication.
// @Tags tasks
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param request body DeleteScheduledTaskByUserIdRequest true "Delete scheduled tasks request"
// @Success 204 "No Content"
// @Failure 400 {object} map[string]string "Bad request - invalid request body"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /tasks/scheduled/by-user [delete]
func (h *TaskHandler) DeleteScheduledTaskByUserId(w http.ResponseWriter, r *http.Request) {
	var req DeleteScheduledTaskByUserIdRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Delete scheduled tasks by user ID
	if err := h.scheduledTaskService.DeleteByUserID(r.Context(), req.UserId); err != nil {
		h.Logger.Error("failed to delete scheduled tasks by user ID", zap.Error(err))
		h.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
