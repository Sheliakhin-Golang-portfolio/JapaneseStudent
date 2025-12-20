package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/japanesestudent/auth-service/internal/models"
	"github.com/japanesestudent/libs/handlers"
	"go.uber.org/zap"
)

// AdminService is the interface that wraps methods for admin operations
type AdminService interface {
	// Method GetUsersList get a list of users with pagination, role and search filters.
	//
	// "page" parameter is used to specify the page number.
	// "count" parameter is used to specify the number of items per page.
	// "role" parameter is used to filter users by role.
	// "search" parameter is used to search users by email or username.
	//
	// If some other error occurs, the error will be returned together with nil.
	GetUsersList(ctx context.Context, page, count int, role *int, search string) ([]models.UserListItem, error)
	// Method GetUserWithSettings gets a user with settings by user ID.
	//
	// "userID" parameter is used to specify the user ID.
	//
	// If user not found, the error will be returned together with nil.
	// If settings were not created, the user will be returned together with nil settings.
	GetUserWithSettings(ctx context.Context, userID int) (*models.UserWithSettingsResponse, error)
	// Method CreateUser creates a new user with settings.
	//
	// "user" parameter is used to specify the user data.
	//
	// If some other error occurs, the error will be returned together with 0 as user ID.
	CreateUser(ctx context.Context, user *models.CreateUserRequest) (int, error)
	// Method CreateUserSettings creates settings for a user.
	//
	// "userID" parameter is used to specify the user ID.
	//
	// If some other error occurs, the error will be returned together with empty string.
	// If settings already exist, the message "Settings already exist" will be returned together with nil error.
	CreateUserSettings(ctx context.Context, userID int) (string, error)
	// Method UpdateUserWithSettings updates a user and settings.
	//
	// "userID" parameter is used to specify the user ID.
	// "userData" parameter is used to specify the user data and settings data.
	//
	// We cannot ignore error about settings not exists forever, so that`s where we will signal admin that it is not good.
	// If some other error occurs, the error will be returned.
	UpdateUserWithSettings(ctx context.Context, userID int, userData *models.UpdateUserWithSettingsRequest) error
	// Method DeleteUser deletes a user by ID.
	//
	// If some other error occurs, the error will be returned.
	DeleteUser(ctx context.Context, userID int) error
}

// AdminHandler handles admin-related HTTP requests
type AdminHandler struct {
	handlers.BaseHandler
	adminService AdminService
}

// NewAdminHandler creates a new admin handler
func NewAdminHandler(
	adminService AdminService,
	logger *zap.Logger,
) *AdminHandler {
	return &AdminHandler{
		BaseHandler:  handlers.BaseHandler{Logger: logger},
		adminService: adminService,
	}
}

// RegisterRoutes registers all admin handler routes
// Note: This assumes the router is already scoped to /api/v3
func (h *AdminHandler) RegisterRoutes(r chi.Router) {
	r.Route("/admin", func(r chi.Router) {
		r.Get("/users", h.GetUsersList)
		r.Get("/users/{id}", h.GetUserWithSettings)
		r.Post("/users", h.CreateUser)
		r.Post("/users/{id}/settings", h.CreateUserSettings)
		r.Patch("/users/{id}", h.UpdateUserWithSettings)
		r.Delete("/users/{id}", h.DeleteUser)
	})
}

// GetUsersList handles GET /admin/users
// @Summary Get list of users
// @Description Get paginated list of users with optional role and search filters
// @Tags admin
// @Accept json
// @Produce json
// @Param page query int false "Page number (default: 1)"
// @Param count query int false "Items per page (default: 20)"
// @Param role query int false "Filter by role"
// @Param search query string false "Search in email or username"
// @Success 200 {array} models.UserListItem "List of users"
// @Failure 400 {object} map[string]string "Invalid request parameters"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /admin/users [get]
func (h *AdminHandler) GetUsersList(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	page := 1
	count := 20
	var role *int
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

	if roleStr := r.URL.Query().Get("role"); roleStr != "" {
		if r, err := strconv.Atoi(roleStr); err == nil {
			role = &r
		}
	}

	if searchStr := r.URL.Query().Get("search"); searchStr != "" {
		search = strings.TrimSpace(searchStr)
	}

	// Get users list
	users, err := h.adminService.GetUsersList(r.Context(), page, count, role, search)
	if err != nil {
		h.Logger.Error("failed to get users list", zap.Error(err))
		h.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.RespondJSON(w, http.StatusOK, users)
}

// GetUserWithSettings handles GET /admin/users/{id}
// @Summary Get user with settings
// @Description Get user information and their settings by user ID
// @Tags admin
// @Accept json
// @Produce json
// @Param id path int true "User ID"
// @Success 200 {object} models.UserWithSettingsResponse "User with settings"
// @Failure 400 {object} map[string]string "Invalid user ID"
// @Failure 404 {object} map[string]string "User not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /admin/users/{id} [get]
func (h *AdminHandler) GetUserWithSettings(w http.ResponseWriter, r *http.Request) {
	// Parse user ID
	userIDStr := chi.URLParam(r, "id")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		h.Logger.Error("failed to parse user ID", zap.Error(err))
		h.RespondError(w, http.StatusBadRequest, "invalid user ID")
		return
	}

	// Get user with settings
	userWithSettings, err := h.adminService.GetUserWithSettings(r.Context(), userID)
	if err != nil {
		h.Logger.Error("failed to get user with settings", zap.Error(err))
		errStatus := http.StatusInternalServerError
		if err.Error() == "invalid user id" || err.Error() == "user not found" {
			errStatus = http.StatusNotFound
		}
		h.RespondError(w, errStatus, err.Error())
		return
	}

	h.RespondJSON(w, http.StatusOK, userWithSettings)
}

// CreateUser handles POST /admin/users
// @Summary Create a user
// @Description Create a new user with settings
// @Tags admin
// @Accept json
// @Produce json
// @Param request body models.CreateUserRequest true "User creation request"
// @Success 201 {object} map[string]string "User created successfully"
// @Failure 400 {object} map[string]string "Invalid request body or user already exists"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /admin/users [post]
func (h *AdminHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var req models.CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.Logger.Error("failed to decode request body", zap.Error(err))
		h.RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Create user
	userID, err := h.adminService.CreateUser(r.Context(), &req)
	if err != nil {
		h.Logger.Error("failed to create user", zap.Error(err))
		errStatus := http.StatusInternalServerError
		if strings.Contains(err.Error(), "already exists") || strings.Contains(err.Error(), "invalid") {
			errStatus = http.StatusBadRequest
		}
		h.RespondError(w, errStatus, err.Error())
		return
	}

	// Check if settings were created (they should be, but handle gracefully)
	response := map[string]any{
		"message": "user created successfully",
		"userId":  userID,
	}
	_, settingsErr := h.adminService.GetUserWithSettings(r.Context(), userID)
	if settingsErr != nil {
		response["message"] = "user created successfully, but settings creation failed"
	}
	h.RespondJSON(w, http.StatusCreated, response)
}

// CreateUserSettings handles POST /admin/users/{id}/settings
// @Summary Create user settings
// @Description Create user settings for a user if they don't exist
// @Tags admin
// @Accept json
// @Produce json
// @Param id path int true "User ID"
// @Success 201 {object} map[string]string "Settings created successfully"
// @Success 200 {object} map[string]string "Settings already exist"
// @Failure 400 {object} map[string]string "Invalid user ID"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /admin/users/{id}/settings [post]
func (h *AdminHandler) CreateUserSettings(w http.ResponseWriter, r *http.Request) {
	// Parse user ID
	userIDStr := chi.URLParam(r, "id")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		h.Logger.Error("failed to parse user ID", zap.Error(err))
		h.RespondError(w, http.StatusBadRequest, "invalid user ID")
		return
	}

	// Create settings
	message, err := h.adminService.CreateUserSettings(r.Context(), userID)
	if err != nil {
		h.Logger.Error("failed to create user settings", zap.Error(err))
		h.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.RespondJSON(w, http.StatusCreated, map[string]string{"message": message})
}

// UpdateUserWithSettings handles PATCH /admin/users/{id}
// @Summary Update user with settings
// @Description Update user and/or settings fields (partial update)
// @Tags admin
// @Accept json
// @Produce json
// @Param id path int true "User ID"
// @Param request body models.UpdateUserWithSettingsRequest false "Update request"
// @Success 204 "No Content"
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /admin/users/{id} [patch]
func (h *AdminHandler) UpdateUserWithSettings(w http.ResponseWriter, r *http.Request) {
	// Parse user ID
	userIDStr := chi.URLParam(r, "id")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		h.Logger.Error("failed to parse user ID", zap.Error(err))
		h.RespondError(w, http.StatusBadRequest, "invalid user ID")
		return
	}

	var req models.UpdateUserWithSettingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.Logger.Error("failed to decode request body", zap.Error(err))
		h.RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Update user and settings
	err = h.adminService.UpdateUserWithSettings(r.Context(), userID, &req)
	if err != nil {
		h.Logger.Error("failed to update user with settings", zap.Error(err))
		errStatus := http.StatusBadRequest
		if err.Error() == "invalid user id" || err.Error() == "user not found" {
			errStatus = http.StatusNotFound
		}
		h.RespondError(w, errStatus, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// DeleteUser handles DELETE /admin/users/{id}
// @Summary Delete a user
// @Description Delete a user by ID
// @Tags admin
// @Accept json
// @Produce json
// @Param id path int true "User ID"
// @Success 204 "No Content"
// @Failure 400 {object} map[string]string "Invalid user ID"
// @Failure 404 {object} map[string]string "User not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /admin/users/{id} [delete]
func (h *AdminHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	// Parse user ID
	userIDStr := chi.URLParam(r, "id")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		h.Logger.Error("failed to parse user ID", zap.Error(err))
		h.RespondError(w, http.StatusBadRequest, "invalid user ID")
		return
	}

	// Delete user
	err = h.adminService.DeleteUser(r.Context(), userID)
	if err != nil {
		h.Logger.Error("failed to delete user", zap.Error(err))
		errStatus := http.StatusInternalServerError
		if err.Error() == "invalid user id" || err.Error() == "user not found" {
			errStatus = http.StatusNotFound
		}
		h.RespondError(w, errStatus, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
