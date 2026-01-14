package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
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
	// "avatarFile" parameter is an optional file reader for the avatar image.
	// "avatarFilename" parameter is the name of the avatar image file.
	//
	// If some other error occurs, the error will be returned together with 0 as user ID.
	CreateUser(ctx context.Context, user *models.CreateUserRequest, avatarFile multipart.File, avatarFilename string) (int, error)
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
	// "avatarFile" parameter is an optional file reader for the avatar image.
	// "avatarFilename" parameter is the name of the avatar image file.
	//
	// We cannot ignore error about settings not exists forever, so that`s where we will signal admin that it is not good.
	// If some other error occurs, the error will be returned.
	UpdateUserWithSettings(r *http.Request, userID int, userData *models.UpdateUserWithSettingsRequest, avatarFile multipart.File, avatarFilename string) error
	// Method DeleteUser deletes a user by ID.
	//
	// If some other error occurs, the error will be returned.
	DeleteUser(ctx context.Context, userID int) error
	// Method GetTutorsList gets a list of tutors (only ID and username).
	//
	// If some other error occurs, the error will be returned together with nil.
	GetTutorsList(ctx context.Context) ([]models.TutorListItem, error)
	// Method ScheduleTokenCleaningTask schedules a token cleaning task.
	//
	// "tokenCleaningURL" parameter is used to specify the token cleaning URL.
	//
	// If some other error occurs, the error will be returned.
	ScheduleTasks(ctx context.Context, tokenCleaningURL string) error
	// Method UpdateUserPassword updates a user's password.
	//
	// "userID" parameter is used to specify the user ID.
	// "password" parameter is the new password.
	//
	// If some other error occurs, the error will be returned.
	UpdateUserPassword(ctx context.Context, userID int, password string) error
}

// AdminHandler handles admin-related HTTP requests
type AdminHandler struct {
	handlers.BaseHandler
	adminService       AdminService
	mediaBaseURL       string
	isDockerContainer  bool
	authServiceBaseURL string
}

// NewAdminHandler creates a new admin handler
func NewAdminHandler(
	adminService AdminService,
	logger *zap.Logger,
	mediaBaseURL string,
	isDockerContainer bool,
	authServiceBaseURL string,
) *AdminHandler {
	return &AdminHandler{
		BaseHandler:        handlers.BaseHandler{Logger: logger},
		adminService:       adminService,
		mediaBaseURL:       mediaBaseURL,
		isDockerContainer:  isDockerContainer,
		authServiceBaseURL: authServiceBaseURL,
	}
}

// RegisterRoutes registers all admin handler routes
// Note: This assumes the router is already scoped to /api/v6
func (h *AdminHandler) RegisterRoutes(r chi.Router) {
	r.Route("/admin", func(r chi.Router) {
		r.Get("/users", h.GetUsersList)
		r.Get("/users/{id}", h.GetUserWithSettings)
		r.Post("/users", h.CreateUser)
		r.Post("/users/{id}/settings", h.CreateUserSettings)
		r.Patch("/users/{id}", h.UpdateUserWithSettings)
		r.Patch("/users/{id}/password", h.UpdateUserPassword)
		r.Delete("/users/{id}", h.DeleteUser)
		r.Get("/tutors", h.GetTutorsList)
		r.Post("/tasks/schedule-token-cleaning", h.ScheduleTokenCleaningTask)
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
// @Description Create a new user with settings and optional avatar
// @Tags admin
// @Accept multipart/form-data
// @Produce json
// @Param username formData string true "Username"
// @Param email formData string true "Email"
// @Param password formData string true "Password"
// @Param role formData int true "Role (1=User, 2=Tutor, 3=Admin)"
// @Param avatar formData file false "Avatar image (optional)"
// @Success 201 {object} map[string]string "User created successfully"
// @Failure 400 {object} map[string]string "Invalid request body or user already exists"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /admin/users [post]
func (h *AdminHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	// Parse multipart form (limit to 20MB to match request size limit)
	err := r.ParseMultipartForm(20 << 20) // 20MB
	if err != nil {
		h.Logger.Error("failed to parse multipart form", zap.Error(err))
		h.RespondError(w, http.StatusBadRequest, "failed to parse request")
		return
	}

	// Extract form values
	req := models.CreateUserRequest{
		Username: r.FormValue("username"),
		Email:    r.FormValue("email"),
		Password: r.FormValue("password"),
	}
	roleStr := r.FormValue("role")

	if req.Username == "" || req.Email == "" || req.Password == "" || roleStr == "" {
		h.RespondError(w, http.StatusBadRequest, "username, email, password, and role are required")
		return
	}

	role, err := strconv.Atoi(roleStr)
	if err != nil {
		h.Logger.Error("failed to parse role", zap.Error(err))
		h.RespondError(w, http.StatusBadRequest, "invalid role")
		return
	}
	req.Role = models.Role(role)

	// Extract avatar file (optional)
	var avatarFile multipart.File
	var avatarFilename string
	file, fileHeader, err := r.FormFile("avatar")
	if err == nil && file != nil {
		// Validate file is actually provided (not just empty field)
		if fileHeader.Size > 0 {
			avatarFile = file
			avatarFilename = fileHeader.Filename
			defer file.Close()
		}
	} else if err != http.ErrMissingFile {
		// If error is not "missing file", it's a real error
		h.Logger.Error("failed to get avatar file from form", zap.Error(err))
		h.RespondError(w, http.StatusBadRequest, "failed to process avatar file")
		return
	}

	// Create user
	userID, err := h.adminService.CreateUser(r.Context(), &req, avatarFile, avatarFilename)
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

	// Return 200 if settings already exist, 201 if created
	statusCode := http.StatusCreated
	if message == "Settings already exist" {
		statusCode = http.StatusOK
	}
	h.RespondJSON(w, statusCode, map[string]string{"message": message})
}

// UpdateUserWithSettings handles PATCH /admin/users/{id}
// @Summary Update user with settings
// @Description Update user and/or settings fields (partial update). Supports optional avatar upload.
// @Tags admin
// @Accept multipart/form-data
// @Produce json
// @Param id path int true "User ID"
// @Param username formData string false "Username"
// @Param email formData string false "Email"
// @Param role formData int false "Role"
// @Param newWordCount formData int false "New word count"
// @Param oldWordCount formData int false "Old word count"
// @Param alphabetLearnCount formData int false "Alphabet learn count"
// @Param language formData string false "Language (en, ru, de)"
// @Param alphabetRepeat formData string false "Alphabet repeat (in question, ignore, repeat)"
// @Param avatar formData file false "Avatar image (optional)"
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

	// Parse multipart form (limit to 20MB to match request size limit)
	err = r.ParseMultipartForm(20 << 20) // 20MB
	if err != nil {
		h.Logger.Error("failed to parse multipart form", zap.Error(err))
		h.RespondError(w, http.StatusBadRequest, "failed to parse request")
		return
	}

	// Extract form values
	req := &models.UpdateUserWithSettingsRequest{
		Username: r.FormValue("username"),
		Email:    r.FormValue("email"),
	}

	if roleStr := r.FormValue("role"); roleStr != "" {
		if roleVal, err := strconv.Atoi(roleStr); err == nil {
			role := models.Role(roleVal)
			req.Role = &role
		}
	}

	// Extract settings if any setting field is provided
	var settings *models.UpdateUserSettingsRequest
	newWordCountStr := r.FormValue("newWordCount")
	oldWordCountStr := r.FormValue("oldWordCount")
	alphabetLearnCountStr := r.FormValue("alphabetLearnCount")
	languageStr := r.FormValue("language")
	alphabetRepeatStr := r.FormValue("alphabetRepeat")

	if newWordCountStr != "" || oldWordCountStr != "" ||
		alphabetLearnCountStr != "" || languageStr != "" ||
		alphabetRepeatStr != "" {
		settings = &models.UpdateUserSettingsRequest{
			Language:       models.Language(languageStr),
			AlphabetRepeat: models.RepeatType(alphabetRepeatStr),
		}

		if newWordCountStr != "" {
			if val, err := strconv.Atoi(newWordCountStr); err == nil {
				settings.NewWordCount = &val
			}
		}

		if oldWordCountStr != "" {
			if val, err := strconv.Atoi(oldWordCountStr); err == nil {
				settings.OldWordCount = &val
			}
		}

		if alphabetLearnCountStr != "" {
			if val, err := strconv.Atoi(alphabetLearnCountStr); err == nil {
				settings.AlphabetLearnCount = &val
			}
		}

		req.Settings = settings
	}

	// Extract avatar file (optional)
	var avatarFile multipart.File
	var avatarFilename string
	file, fileHeader, err := r.FormFile("avatar")
	if err == nil && file != nil {
		// Validate file is actually provided (not just empty field)
		if fileHeader.Size > 0 {
			avatarFile = file
			avatarFilename = fileHeader.Filename
			defer file.Close()
		}
	} else if err != http.ErrMissingFile {
		// If error is not "missing file", it's a real error
		h.Logger.Error("failed to get avatar file from form", zap.Error(err))
		h.RespondError(w, http.StatusBadRequest, "failed to process avatar file")
		return
	}

	// Update user and settings
	err = h.adminService.UpdateUserWithSettings(r, userID, req, avatarFile, avatarFilename)
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
// @Description Delete a user by ID and their avatar file from media service
// @Tags admin
// @Accept json
// @Produce json
// @Param id path int true "User ID"
// @Success 200 {object} map[string]string "User deleted successfully, but..."
// @Success 204 "No Content (when avatar deletion is successful)"
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
		if strings.Contains(err.Error(), "avatar file has not been deleted") {
			response := map[string]string{
				"message": fmt.Sprintf("user deleted successfully, but %s", err.Error()),
			}
			h.RespondJSON(w, http.StatusOK, response)
			return
		}
		if err.Error() == "invalid user id" || err.Error() == "user not found" {
			errStatus = http.StatusNotFound
		}
		h.RespondError(w, errStatus, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// UpdateUserPassword handles PATCH /admin/users/{id}/password
// @Summary Update user password
// @Description Update a user's password by user ID
// @Tags admin
// @Accept json
// @Produce json
// @Param id path int true "User ID"
// @Param request body models.UpdatePasswordRequest true "Password update request"
// @Success 204 "No Content"
// @Failure 400 {object} map[string]string "Invalid request or password validation failed"
// @Failure 404 {object} map[string]string "User not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /admin/users/{id}/password [patch]
func (h *AdminHandler) UpdateUserPassword(w http.ResponseWriter, r *http.Request) {
	// Parse user ID
	userIDStr := chi.URLParam(r, "id")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		h.Logger.Error("failed to parse user ID", zap.Error(err))
		h.RespondError(w, http.StatusBadRequest, "invalid user ID")
		return
	}

	// Parse request body
	var req models.UpdatePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.Logger.Error("failed to decode request body", zap.Error(err))
		h.RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Update user password
	err = h.adminService.UpdateUserPassword(r.Context(), userID, req.Password)
	if err != nil {
		h.Logger.Error("failed to update user password", zap.Error(err))
		errStatus := http.StatusInternalServerError
		if err.Error() == "invalid user id" || err.Error() == "user not found" {
			errStatus = http.StatusNotFound
		} else if strings.Contains(err.Error(), "password") || strings.Contains(err.Error(), "invalid") {
			errStatus = http.StatusBadRequest
		}
		h.RespondError(w, errStatus, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetTutorsList handles GET /admin/tutors
// @Summary Get list of tutors
// @Description Get list of tutors with only ID and username (for select options)
// @Tags admin
// @Accept json
// @Produce json
// @Success 200 {array} models.TutorListItem "List of tutors"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /admin/tutors [get]
func (h *AdminHandler) GetTutorsList(w http.ResponseWriter, r *http.Request) {
	// Get tutors list
	tutors, err := h.adminService.GetTutorsList(r.Context())
	if err != nil {
		h.Logger.Error("failed to get tutors list", zap.Error(err))
		h.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.RespondJSON(w, http.StatusOK, tutors)
}

// ScheduleTokenCleaningTask handles POST /admin/tasks/schedule-token-cleaning
// @Summary Schedule token cleaning task
// @Description Creates a scheduled task in task-service to call token cleaning endpoint twice daily
// @Tags admin
// @Accept json
// @Produce json
// @Success 201 {object} map[string]string "Task scheduled successfully"
// @Failure 400 {object} map[string]string "Bad request - invalid configuration or task creation failed"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /admin/tasks/schedule-token-cleaning [post]
func (h *AdminHandler) ScheduleTokenCleaningTask(w http.ResponseWriter, r *http.Request) {
	var tokenCleaningURL string
	// If all services are in the same docker network, we can use this network instead of constructing the URL from the request
	if h.isDockerContainer {
		tokenCleaningURL = fmt.Sprintf("%s/api/v6/tokens/clean", h.authServiceBaseURL)
	} else {
		// Construct the token cleaning endpoint URL from the request
		scheme := "http"
		if r.TLS != nil {
			scheme = "https"
		}
		tokenCleaningURL = fmt.Sprintf("%s://%s/api/v6/tokens/clean", scheme, r.Host)
	}

	if err := h.adminService.ScheduleTasks(r.Context(), tokenCleaningURL); err != nil {
		h.Logger.Error("failed to schedule token cleaning task", zap.Error(err))
		h.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.RespondJSON(w, http.StatusCreated, map[string]string{"message": "token cleaning task scheduled successfully"})
}
