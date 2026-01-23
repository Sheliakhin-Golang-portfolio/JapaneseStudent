package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-redis/redis/v8"
	"github.com/hibiken/asynq"
	"github.com/Sheliakhin-Golang-portfolio/JapaneseStudent/libs/handlers"
	"github.com/Sheliakhin-Golang-portfolio/JapaneseStudent/services/task-service/internal/models"
	"go.uber.org/zap"
)

// EmailAdminTemplateService is the interface that wraps methods for email template business logic
type EmailAdminTemplateService interface {
	GetAll(ctx context.Context, page, count int, search string) ([]models.EmailTemplateListItem, error)
	GetByID(ctx context.Context, id int) (*models.EmailTemplate, error)
	Create(ctx context.Context, req *models.CreateUpdateEmailTemplateRequest) (int, error)
	Update(ctx context.Context, id int, req *models.CreateUpdateEmailTemplateRequest) error
	Delete(ctx context.Context, id int) error
}

// ImmediateAdminTaskService is the interface that wraps methods for immediate task business logic
type ImmediateAdminTaskService interface {
	GetAll(ctx context.Context, page, count int, userID, templateID int, status string) ([]models.ImmediateTaskListItem, error)
	GetByID(ctx context.Context, id int) (*models.ImmediateTask, error)
	CreateAdmin(ctx context.Context, req *models.AdminCreateImmediateTaskRequest) (int, error)
	Update(ctx context.Context, id int, req *models.UpdateImmediateTaskRequest) error
	Delete(ctx context.Context, id int) error
}

// ScheduledAdminTaskService is the interface that wraps methods for scheduled task business logic
type ScheduledAdminTaskService interface {
	GetAll(ctx context.Context, page, count, userID, templateID int, active *bool) ([]models.ScheduledTaskListItem, error)
	GetByID(ctx context.Context, id int) (*models.ScheduledTask, error)
	CreateAdmin(ctx context.Context, req *models.AdminCreateScheduledTaskRequest) (int, error)
	Update(ctx context.Context, id int, req *models.UpdateScheduledTaskRequest) error
	Delete(ctx context.Context, id int) error
}

// AdminTaskLogService is the interface that wraps methods for task log business logic
type AdminTaskLogService interface {
	GetAll(ctx context.Context, page, count, taskID int, jobID, status string) ([]models.ScheduledTaskLogListItem, error)
	GetByID(ctx context.Context, id int) (*models.ScheduledTaskLog, error)
}

// AdminHandler handles admin-related HTTP requests
type AdminHandler struct {
	handlers.BaseHandler
	emailTemplateService EmailAdminTemplateService
	immediateTaskService ImmediateAdminTaskService
	scheduledTaskService ScheduledAdminTaskService
	taskLogService       AdminTaskLogService
	asynqClient          *asynq.Client
	redis                *redis.Client
}

// NewAdminHandler creates a new admin handler
func NewAdminHandler(
	emailTemplateService EmailAdminTemplateService,
	immediateTaskService ImmediateAdminTaskService,
	scheduledTaskService ScheduledAdminTaskService,
	taskLogService AdminTaskLogService,
	asynqClient *asynq.Client,
	redis *redis.Client,
	logger *zap.Logger,
) *AdminHandler {
	return &AdminHandler{
		BaseHandler:          handlers.BaseHandler{Logger: logger},
		emailTemplateService: emailTemplateService,
		immediateTaskService: immediateTaskService,
		scheduledTaskService: scheduledTaskService,
		taskLogService:       taskLogService,
		asynqClient:          asynqClient,
		redis:                redis,
	}
}

// RegisterRoutes registers all admin handler routes
func (h *AdminHandler) RegisterRoutes(r chi.Router) {
	r.Route("/admin", func(r chi.Router) {
		// Email Templates
		r.Get("/email-templates", h.GetEmailTemplatesList)
		r.Get("/email-templates/{id}", h.GetEmailTemplate)
		r.Post("/email-templates", h.CreateEmailTemplate)
		r.Patch("/email-templates/{id}", h.UpdateEmailTemplate)
		r.Delete("/email-templates/{id}", h.DeleteEmailTemplate)

		// Immediate Tasks
		r.Get("/immediate-tasks", h.GetImmediateTasksList)
		r.Get("/immediate-tasks/{id}", h.GetImmediateTask)
		r.Post("/immediate-tasks", h.CreateImmediateTask)
		r.Patch("/immediate-tasks/{id}", h.UpdateImmediateTask)
		r.Delete("/immediate-tasks/{id}", h.DeleteImmediateTask)

		// Scheduled Tasks
		r.Get("/scheduled-tasks", h.GetScheduledTasksList)
		r.Get("/scheduled-tasks/{id}", h.GetScheduledTask)
		r.Post("/scheduled-tasks", h.CreateScheduledTask)
		r.Patch("/scheduled-tasks/{id}", h.UpdateScheduledTask)
		r.Delete("/scheduled-tasks/{id}", h.DeleteScheduledTask)

		// Scheduled Task Logs
		r.Get("/scheduled-task-logs", h.GetScheduledTaskLogsList)
		r.Get("/scheduled-task-logs/{id}", h.GetScheduledTaskLog)
	})
}

// Email Template Handlers

// GetEmailTemplatesList handles GET /admin/email-templates
// @Summary Get list of email templates
// @Description Get paginated list of email templates with optional search filter. Requires admin role (JWT authentication).
// @Tags admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number (default: 1)"
// @Param count query int false "Items per page (default: 20)"
// @Param search query string false "Search in template slug"
// @Success 200 {array} models.EmailTemplateListItem "List of email templates"
// @Failure 400 {object} map[string]string "Invalid request parameters"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /admin/email-templates [get]
func (h *AdminHandler) GetEmailTemplatesList(w http.ResponseWriter, r *http.Request) {
	page := 1
	count := 20
	search := ""

	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	if countStr := r.URL.Query().Get("count"); countStr != "" {
		if c, err := strconv.Atoi(countStr); err == nil && c > 0 {
			count = c
		}
	}

	if searchStr := r.URL.Query().Get("search"); searchStr != "" {
		search = strings.TrimSpace(searchStr)
	}

	templates, err := h.emailTemplateService.GetAll(r.Context(), page, count, search)
	if err != nil {
		h.Logger.Error("failed to get email templates list", zap.Error(err))
		h.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.RespondJSON(w, http.StatusOK, templates)
}

// GetEmailTemplate handles GET /admin/email-templates/{id}
// @Summary Get email template by ID
// @Description Get full email template information by ID. Requires admin role (JWT authentication).
// @Tags admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Email template ID"
// @Success 200 {object} models.EmailTemplate "Email template details"
// @Failure 400 {object} map[string]string "Invalid template ID"
// @Failure 404 {object} map[string]string "Email template not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /admin/email-templates/{id} [get]
func (h *AdminHandler) GetEmailTemplate(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		h.RespondError(w, http.StatusBadRequest, "invalid template ID")
		return
	}

	template, err := h.emailTemplateService.GetByID(r.Context(), id)
	if err != nil {
		h.Logger.Error("failed to get email template", zap.Error(err))
		h.RespondError(w, http.StatusNotFound, err.Error())
		return
	}

	h.RespondJSON(w, http.StatusOK, template)
}

// CreateEmailTemplate handles POST /admin/email-templates
// @Summary Create email template
// @Description Create a new email template. Requires admin role (JWT authentication).
// @Tags admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param template body models.CreateUpdateEmailTemplateRequest true "Email template creation request"
// @Success 201 {object} map[string]any "Email template created successfully"
// @Failure 400 {object} map[string]string "Bad request - invalid request body or template creation failed"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /admin/email-templates [post]
func (h *AdminHandler) CreateEmailTemplate(w http.ResponseWriter, r *http.Request) {
	var req models.CreateUpdateEmailTemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	templateID, err := h.emailTemplateService.Create(r.Context(), &req)
	if err != nil {
		h.Logger.Error("failed to create email template", zap.Error(err))
		h.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}

	h.RespondJSON(w, http.StatusCreated, map[string]any{
		"message": "email template created successfully",
		"id":      templateID,
	})
}

// UpdateEmailTemplate handles PATCH /admin/email-templates/{id}
// @Summary Update email template
// @Description Update an email template (partial update). Requires admin role (JWT authentication).
// @Tags admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Email template ID"
// @Param template body models.CreateUpdateEmailTemplateRequest true "Email template update request"
// @Success 204 "No Content"
// @Failure 400 {object} map[string]string "Bad request - invalid template ID or request body"
// @Failure 404 {object} map[string]string "Email template not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /admin/email-templates/{id} [patch]
func (h *AdminHandler) UpdateEmailTemplate(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		h.RespondError(w, http.StatusBadRequest, "invalid template ID")
		return
	}

	var req models.CreateUpdateEmailTemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.emailTemplateService.Update(r.Context(), id, &req); err != nil {
		h.Logger.Error("failed to update email template", zap.Error(err))
		h.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// DeleteEmailTemplate handles DELETE /admin/email-templates/{id}
// @Summary Delete email template
// @Description Delete an email template by ID. Requires admin role (JWT authentication).
// @Tags admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Email template ID"
// @Success 204 "No Content"
// @Failure 400 {object} map[string]string "Invalid template ID"
// @Failure 404 {object} map[string]string "Email template not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /admin/email-templates/{id} [delete]
func (h *AdminHandler) DeleteEmailTemplate(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		h.RespondError(w, http.StatusBadRequest, "invalid template ID")
		return
	}

	if err := h.emailTemplateService.Delete(r.Context(), id); err != nil {
		h.Logger.Error("failed to delete email template", zap.Error(err))
		h.RespondError(w, http.StatusNotFound, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Immediate Task Handlers

// GetImmediateTasksList handles GET /admin/immediate-tasks
// @Summary Get list of immediate tasks
// @Description Get paginated list of immediate tasks with optional filters (user ID, template ID, status). Requires admin role (JWT authentication).
// @Tags admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number (default: 1)"
// @Param count query int false "Items per page (default: 20)"
// @Param user_id query int false "Filter by user ID"
// @Param template_id query int false "Filter by template ID"
// @Param status query string false "Filter by status (Enqueued, Completed, Failed)"
// @Success 200 {array} models.ImmediateTaskListItem "List of immediate tasks"
// @Failure 400 {object} map[string]string "Invalid request parameters"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /admin/immediate-tasks [get]
func (h *AdminHandler) GetImmediateTasksList(w http.ResponseWriter, r *http.Request) {
	page := 1
	count := 20
	userID := 0
	templateID := 0
	status := ""

	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	if countStr := r.URL.Query().Get("count"); countStr != "" {
		if c, err := strconv.Atoi(countStr); err == nil && c > 0 {
			count = c
		}
	}

	if userIDStr := r.URL.Query().Get("user_id"); userIDStr != "" {
		if u, err := strconv.Atoi(userIDStr); err == nil {
			userID = u
		}
	}

	if templateIDStr := r.URL.Query().Get("template_id"); templateIDStr != "" {
		if t, err := strconv.Atoi(templateIDStr); err == nil {
			templateID = t
		}
	}

	if statusStr := r.URL.Query().Get("status"); statusStr != "" {
		status = strings.TrimSpace(statusStr)
	}

	tasks, err := h.immediateTaskService.GetAll(r.Context(), page, count, userID, templateID, status)
	if err != nil {
		h.Logger.Error("failed to get immediate tasks list", zap.Error(err))
		h.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.RespondJSON(w, http.StatusOK, tasks)
}

// GetImmediateTask handles GET /admin/immediate-tasks/{id}
// @Summary Get immediate task by ID
// @Description Get full immediate task information by ID. Requires admin role (JWT authentication).
// @Tags admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Immediate task ID"
// @Success 200 {object} models.ImmediateTask "Immediate task details"
// @Failure 400 {object} map[string]string "Invalid task ID"
// @Failure 404 {object} map[string]string "Immediate task not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /admin/immediate-tasks/{id} [get]
func (h *AdminHandler) GetImmediateTask(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		h.RespondError(w, http.StatusBadRequest, "invalid task ID")
		return
	}

	task, err := h.immediateTaskService.GetByID(r.Context(), id)
	if err != nil {
		h.Logger.Error("failed to get immediate task", zap.Error(err))
		h.RespondError(w, http.StatusNotFound, err.Error())
		return
	}

	h.RespondJSON(w, http.StatusOK, task)
}

// CreateImmediateTask handles POST /admin/immediate-tasks
// @Summary Create immediate task
// @Description Create a new immediate task and enqueue it for processing. Requires admin role (JWT authentication).
// @Tags admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param task body models.AdminCreateImmediateTaskRequest true "Immediate task creation request"
// @Success 201 {object} map[string]any "Immediate task created successfully"
// @Failure 400 {object} map[string]string "Bad request - invalid request body or task creation failed"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /admin/immediate-tasks [post]
func (h *AdminHandler) CreateImmediateTask(w http.ResponseWriter, r *http.Request) {
	var req models.AdminCreateImmediateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Create immediate task and enqueue it
	taskID, err := h.immediateTaskService.CreateAdmin(r.Context(), &req)
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

// UpdateImmediateTask handles PATCH /admin/immediate-tasks/{id}
// @Summary Update immediate task
// @Description Update an immediate task (partial update). Requires admin role (JWT authentication).
// @Tags admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Immediate task ID"
// @Param task body models.UpdateImmediateTaskRequest true "Immediate task update request"
// @Success 204 "No Content"
// @Failure 400 {object} map[string]string "Bad request - invalid task ID or request body"
// @Failure 404 {object} map[string]string "Immediate task not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /admin/immediate-tasks/{id} [patch]
func (h *AdminHandler) UpdateImmediateTask(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		h.RespondError(w, http.StatusBadRequest, "invalid task ID")
		return
	}

	var req models.UpdateImmediateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.immediateTaskService.Update(r.Context(), id, &req); err != nil {
		h.Logger.Error("failed to update immediate task", zap.Error(err))
		h.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// DeleteImmediateTask handles DELETE /admin/immediate-tasks/{id}
// @Summary Delete immediate task
// @Description Delete an immediate task by ID. Requires admin role (JWT authentication).
// @Tags admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Immediate task ID"
// @Success 204 "No Content"
// @Failure 400 {object} map[string]string "Invalid task ID"
// @Failure 404 {object} map[string]string "Immediate task not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /admin/immediate-tasks/{id} [delete]
func (h *AdminHandler) DeleteImmediateTask(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		h.RespondError(w, http.StatusBadRequest, "invalid task ID")
		return
	}

	if err := h.immediateTaskService.Delete(r.Context(), id); err != nil {
		h.Logger.Error("failed to delete immediate task", zap.Error(err))
		errStatus := http.StatusInternalServerError
		if err.Error() == "immediate task not found" {
			errStatus = http.StatusNotFound
		}
		h.RespondError(w, errStatus, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Scheduled Task Handlers

// GetScheduledTasksList handles GET /admin/scheduled-tasks
// @Summary Get list of scheduled tasks
// @Description Get paginated list of scheduled tasks with optional filters (user ID, template ID, active status). Requires admin role (JWT authentication).
// @Tags admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number (default: 1)"
// @Param count query int false "Items per page (default: 20)"
// @Param user_id query int false "Filter by user ID"
// @Param template_id query int false "Filter by template ID"
// @Param active query bool false "Filter by active status"
// @Success 200 {array} models.ScheduledTaskListItem "List of scheduled tasks"
// @Failure 400 {object} map[string]string "Invalid request parameters"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /admin/scheduled-tasks [get]
func (h *AdminHandler) GetScheduledTasksList(w http.ResponseWriter, r *http.Request) {
	page := 1
	count := 20
	userID := 0
	templateID := 0
	var active *bool

	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	if countStr := r.URL.Query().Get("count"); countStr != "" {
		if c, err := strconv.Atoi(countStr); err == nil && c > 0 {
			count = c
		}
	}

	if userIDStr := r.URL.Query().Get("user_id"); userIDStr != "" {
		if u, err := strconv.Atoi(userIDStr); err == nil {
			userID = u
		}
	}

	if templateIDStr := r.URL.Query().Get("template_id"); templateIDStr != "" {
		if t, err := strconv.Atoi(templateIDStr); err == nil {
			templateID = t
		}
	}

	if activeStr := r.URL.Query().Get("active"); activeStr != "" {
		if a, err := strconv.ParseBool(activeStr); err == nil {
			active = &a
		}
	}

	tasks, err := h.scheduledTaskService.GetAll(r.Context(), page, count, userID, templateID, active)
	if err != nil {
		h.Logger.Error("failed to get scheduled tasks list", zap.Error(err))
		h.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.RespondJSON(w, http.StatusOK, tasks)
}

// GetScheduledTask handles GET /admin/scheduled-tasks/{id}
// @Summary Get scheduled task by ID
// @Description Get full scheduled task information by ID. Requires admin role (JWT authentication).
// @Tags admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Scheduled task ID"
// @Success 200 {object} models.ScheduledTask "Scheduled task details"
// @Failure 400 {object} map[string]string "Invalid task ID"
// @Failure 404 {object} map[string]string "Scheduled task not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /admin/scheduled-tasks/{id} [get]
func (h *AdminHandler) GetScheduledTask(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		h.RespondError(w, http.StatusBadRequest, "invalid task ID")
		return
	}

	task, err := h.scheduledTaskService.GetByID(r.Context(), id)
	if err != nil {
		h.Logger.Error("failed to get scheduled task", zap.Error(err))
		h.RespondError(w, http.StatusNotFound, err.Error())
		return
	}

	h.RespondJSON(w, http.StatusOK, task)
}

// CreateScheduledTask handles POST /admin/scheduled-tasks
// @Summary Create scheduled task
// @Description Create a new scheduled task with cron expression. Requires admin role (JWT authentication).
// @Tags admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param task body models.AdminCreateScheduledTaskRequest true "Scheduled task creation request"
// @Success 201 {object} map[string]any "Scheduled task created successfully"
// @Failure 400 {object} map[string]string "Bad request - invalid request body or task creation failed"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /admin/scheduled-tasks [post]
func (h *AdminHandler) CreateScheduledTask(w http.ResponseWriter, r *http.Request) {
	var req models.AdminCreateScheduledTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	taskID, err := h.scheduledTaskService.CreateAdmin(r.Context(), &req)
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

// UpdateScheduledTask handles PATCH /admin/scheduled-tasks/{id}
// @Summary Update scheduled task
// @Description Update a scheduled task (partial update). Requires admin role (JWT authentication).
// @Tags admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Scheduled task ID"
// @Param task body models.UpdateScheduledTaskRequest true "Scheduled task update request"
// @Success 204 "No Content"
// @Failure 400 {object} map[string]string "Bad request - invalid task ID or request body"
// @Failure 404 {object} map[string]string "Scheduled task not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /admin/scheduled-tasks/{id} [patch]
func (h *AdminHandler) UpdateScheduledTask(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		h.RespondError(w, http.StatusBadRequest, "invalid task ID")
		return
	}

	var req models.UpdateScheduledTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Update task
	if err := h.scheduledTaskService.Update(r.Context(), id, &req); err != nil {
		h.Logger.Error("failed to update scheduled task", zap.Error(err))
		h.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// DeleteScheduledTask handles DELETE /admin/scheduled-tasks/{id}
// @Summary Delete scheduled task
// @Description Delete a scheduled task by ID from database and Redis ZSET. Requires admin role (JWT authentication).
// @Tags admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Scheduled task ID"
// @Success 204 "No Content"
// @Failure 400 {object} map[string]string "Invalid task ID"
// @Failure 404 {object} map[string]string "Scheduled task not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /admin/scheduled-tasks/{id} [delete]
func (h *AdminHandler) DeleteScheduledTask(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		h.RespondError(w, http.StatusBadRequest, "invalid task ID")
		return
	}

	// Delete task from database and Redis ZSET
	if err := h.scheduledTaskService.Delete(r.Context(), id); err != nil {
		h.Logger.Error("failed to delete scheduled task", zap.Error(err))
		h.RespondError(w, http.StatusNotFound, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Scheduled Task Log Handlers

// GetScheduledTaskLogsList handles GET /admin/scheduled-task-logs
// @Summary Get list of scheduled task logs
// @Description Get paginated list of scheduled task logs with optional filters (task ID, job ID, status). Requires admin role (JWT authentication).
// @Tags admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number (default: 1)"
// @Param count query int false "Items per page (default: 20)"
// @Param task_id query int false "Filter by task ID"
// @Param job_id query string false "Filter by job ID"
// @Param status query string false "Filter by status (Completed, Failed)"
// @Success 200 {array} models.ScheduledTaskLogListItem "List of scheduled task logs"
// @Failure 400 {object} map[string]string "Invalid request parameters"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /admin/scheduled-task-logs [get]
func (h *AdminHandler) GetScheduledTaskLogsList(w http.ResponseWriter, r *http.Request) {
	page := 1
	count := 20
	taskID := 0
	jobID := ""
	status := ""

	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	if countStr := r.URL.Query().Get("count"); countStr != "" {
		if c, err := strconv.Atoi(countStr); err == nil && c > 0 {
			count = c
		}
	}

	if taskIDStr := r.URL.Query().Get("task_id"); taskIDStr != "" {
		if t, err := strconv.Atoi(taskIDStr); err == nil {
			taskID = t
		}
	}

	if jobIDStr := r.URL.Query().Get("job_id"); jobIDStr != "" {
		jobID = strings.TrimSpace(jobIDStr)
	}

	if statusStr := r.URL.Query().Get("status"); statusStr != "" {
		status = strings.TrimSpace(statusStr)
	}

	logs, err := h.taskLogService.GetAll(r.Context(), page, count, taskID, jobID, status)
	if err != nil {
		h.Logger.Error("failed to get scheduled task logs list", zap.Error(err))
		h.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.RespondJSON(w, http.StatusOK, logs)
}

// GetScheduledTaskLog handles GET /admin/scheduled-task-logs/{id}
// @Summary Get scheduled task log by ID
// @Description Get full scheduled task log information by ID. Requires admin role (JWT authentication).
// @Tags admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Scheduled task log ID"
// @Success 200 {object} models.ScheduledTaskLog "Scheduled task log details"
// @Failure 400 {object} map[string]string "Invalid log ID"
// @Failure 404 {object} map[string]string "Scheduled task log not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /admin/scheduled-task-logs/{id} [get]
func (h *AdminHandler) GetScheduledTaskLog(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		h.RespondError(w, http.StatusBadRequest, "invalid log ID")
		return
	}

	logEntry, err := h.taskLogService.GetByID(r.Context(), id)
	if err != nil {
		h.Logger.Error("failed to get scheduled task log", zap.Error(err))
		h.RespondError(w, http.StatusNotFound, err.Error())
		return
	}

	h.RespondJSON(w, http.StatusOK, logEntry)
}
