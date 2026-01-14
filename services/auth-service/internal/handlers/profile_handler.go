package handlers

import (
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/japanesestudent/auth-service/internal/models"
	"github.com/japanesestudent/libs/auth/middleware"
	"github.com/japanesestudent/libs/handlers"
	"go.uber.org/zap"
)

// ProfileService is the interface that wraps methods for profile business logic
type ProfileService interface {
	// GetUser retrieves user profile public information (username, email, avatar)
	//
	// "userId" parameter is used to identify the user.
	//
	// If user with such ID does not exist, the error will be returned together with "nil" value.
	GetUser(ctx context.Context, userId int) (*models.ProfileResponse, error)
	// UpdateUser updates user profile (username and/or email) with validation
	//
	// "userId" parameter is used to identify the user.
	// "username" parameter is used to update the username.
	// "email" parameter is used to update the email.
	//
	// If some error occurs during user profile update, the error will be returned.
	UpdateUser(ctx context.Context, userId int, username, email string) error
	// UpdateAvatar updates user avatar
	//
	// "userId" parameter is used to identify the user.
	// "avatarFile" parameter is used to update the avatar.
	// "avatarFilename" parameter is used to update the avatar filename.
	//
	// If some error occurs during avatar update, the error will be returned together with "nil" value.
	UpdateAvatar(ctx context.Context, userId int, avatarFile multipart.File, avatarFilename string) (string, error)
	// UpdatePassword updates user password with validation
	//
	// "userId" parameter is used to identify the user.
	// "password" parameter is used to update the password.
	//
	// If some error occurs during password update, the error will be returned.
	UpdatePassword(ctx context.Context, userId int, password string) error
	// UpdateRepeatFlag updates the repeat flag
	//
	// "userId" parameter is used to update the repeat flag by user ID.
	// "flag" parameter is used to update the repeat flag.
	// "r" parameter is used to construct the URL for the drop marks endpoint.
	//
	// If some error occurs during repeat flag update, the error will be returned.
	UpdateRepeatFlag(ctx context.Context, userId int, flag string, r *http.Request) error
}

// UserSettingsService is the interface that wraps methods for user settings business logic
type UserSettingsService interface {
	// GetUserSettings retrieves user settings
	//
	// "userId" parameter is used to retrieve user settings by user ID.
	// If user settings are not found, the error will be returned together with "nil" value.
	GetUserSettings(ctx context.Context, userId int) (*models.UserSettingsResponse, error)
	// UpdateUserSettings updates user settings with validation
	//
	// Method validates that at least one field is provided.
	// "userId" parameter is used to update user settings by user ID.
	// "updateRequest" parameter is used to update only some fields of user settings.
	//
	// If some error occurs during user settings update, the error will be returned.
	UpdateUserSettings(ctx context.Context, userId int, updateRequest *models.UpdateUserSettingsRequest) error
}

// ProfileHandler handles profile HTTP requests
type ProfileHandler struct {
	handlers.BaseHandler
	profileService      ProfileService
	userSettingsService UserSettingsService
}

// NewProfileHandler creates a new profile handler
func NewProfileHandler(profileService ProfileService, userSettingsService UserSettingsService, logger *zap.Logger) *ProfileHandler {
	return &ProfileHandler{
		BaseHandler:         handlers.BaseHandler{Logger: logger},
		profileService:      profileService,
		userSettingsService: userSettingsService,
	}
}

// RegisterRoutes registers all profile handler routes
func (h *ProfileHandler) RegisterRoutes(r chi.Router, authMiddleware func(http.Handler) http.Handler) {
	r.Route("/profile", func(r chi.Router) {
		r.Use(authMiddleware)
		r.Get("/", h.GetUser)
		r.Patch("/", h.UpdateUser)
		r.Put("/avatar", h.UpdateAvatar)
		r.Put("/password", h.UpdatePassword)
		r.Get("/settings", h.GetUserSettings)
		r.Patch("/settings", h.UpdateUserSettings)
		r.Put("/repeat-flag", h.UpdateRepeatFlag)
	})
}

// GetUser handles GET /profile
// @Summary Get user profile
// @Description Get user profile information (username, email, avatar) for the authenticated user. Requires authentication.
// @Tags profile
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} models.ProfileResponse "User profile"
// @Failure 401 {object} map[string]string "Unauthorized - authentication required"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /profile [get]
func (h *ProfileHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	// Extract userID from auth middleware context
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		h.Logger.Error("user ID not found in context")
		h.RespondError(w, http.StatusUnauthorized, "user ID not found in context")
		return
	}

	// Get user profile
	profile, err := h.profileService.GetUser(r.Context(), userID)
	if err != nil {
		h.Logger.Error("failed to get user profile", zap.Error(err))
		statusCode := http.StatusInternalServerError
		if strings.Contains(err.Error(), "not found") {
			statusCode = http.StatusNotFound
		}
		h.RespondError(w, statusCode, err.Error())
		return
	}

	h.RespondJSON(w, http.StatusOK, profile)
}

// UpdateUserRequest represents a request to update user profile
type UpdateUserRequest struct {
	Username string `json:"username,omitempty"`
	Email    string `json:"email,omitempty"`
}

// UpdateUser handles PATCH /profile
// @Summary Update user profile
// @Description Update user profile (username and/or email) for the authenticated user. Requires authentication.
// @Tags profile
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param profile body UpdateUserRequest true "User profile to update"
// @Success 204 "No Content"
// @Failure 400 {object} map[string]string "Bad request - invalid request body or validation error"
// @Failure 401 {object} map[string]string "Unauthorized - authentication required"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /profile [patch]
func (h *ProfileHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	// Extract userID from auth middleware context
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		h.Logger.Error("user ID not found in context")
		h.RespondError(w, http.StatusUnauthorized, "user ID not found in context")
		return
	}

	// Parse request body
	var req UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.Logger.Error("failed to decode request body", zap.Error(err))
		h.RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Update user profile
	err := h.profileService.UpdateUser(r.Context(), userID, req.Username, req.Email)
	if err != nil {
		h.Logger.Error("failed to update user profile", zap.Error(err))
		statusCode := http.StatusInternalServerError
		// Check if it's a validation error
		errMsg := err.Error()
		if strings.Contains(errMsg, "at least one field") ||
			strings.Contains(errMsg, "invalid email") ||
			strings.Contains(errMsg, "already exists") {
			statusCode = http.StatusBadRequest
		}
		h.RespondError(w, statusCode, err.Error())
		return
	}

	// Return 204 No Content
	w.WriteHeader(http.StatusNoContent)
}

// UpdateAvatar handles PUT /profile/avatar
// @Summary Update user avatar
// @Description Update user avatar for the authenticated user. Requires authentication.
// @Tags profile
// @Accept multipart/form-data
// @Produce json
// @Security ApiKeyAuth
// @Param avatar formData file true "Avatar image file"
// @Success 200 {object} map[string]string "Avatar URL"
// @Failure 400 {object} map[string]string "Bad request - invalid file or validation error"
// @Failure 401 {object} map[string]string "Unauthorized - authentication required"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /profile/avatar [put]
func (h *ProfileHandler) UpdateAvatar(w http.ResponseWriter, r *http.Request) {
	// Extract userID from auth middleware context
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		h.Logger.Error("user ID not found in context")
		h.RespondError(w, http.StatusUnauthorized, "user ID not found in context")
		return
	}

	// Parse multipart form (limit to 20MB to match request size limit)
	err := r.ParseMultipartForm(20 << 20) // 20MB
	if err != nil {
		h.Logger.Error("failed to parse multipart form", zap.Error(err))
		h.RespondError(w, http.StatusBadRequest, "failed to parse request")
		return
	}

	// Extract avatar file
	file, fileHeader, err := r.FormFile("avatar")
	if err != nil {
		h.Logger.Error("failed to get avatar file from form", zap.Error(err))
		h.RespondError(w, http.StatusBadRequest, "avatar file is required")
		return
	}
	defer file.Close()

	// Validate that file is not empty
	if fileHeader.Size == 0 {
		h.Logger.Error("avatar file is empty")
		h.RespondError(w, http.StatusBadRequest, "avatar file cannot be empty")
		return
	}

	// Update avatar
	avatarURL, err := h.profileService.UpdateAvatar(r.Context(), userID, file, fileHeader.Filename)
	if err != nil {
		h.Logger.Error("failed to update avatar", zap.Error(err))
		statusCode := http.StatusInternalServerError
		if strings.Contains(err.Error(), "not found") {
			statusCode = http.StatusNotFound
		} else if strings.Contains(err.Error(), "invalid") || strings.Contains(err.Error(), "empty") {
			statusCode = http.StatusBadRequest
		}
		h.RespondError(w, statusCode, err.Error())
		return
	}

	// Return avatar URL
	h.RespondJSON(w, http.StatusOK, map[string]string{"avatar_url": avatarURL})
}

// UpdatePassword handles PUT /profile/password
// @Summary Update user password
// @Description Update user password for the authenticated user. Requires authentication.
// @Tags profile
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param password body models.UpdatePasswordRequest true "New password"
// @Success 204 "No Content"
// @Failure 400 {object} map[string]string "Bad request - invalid password or validation error"
// @Failure 401 {object} map[string]string "Unauthorized - authentication required"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /profile/password [put]
func (h *ProfileHandler) UpdatePassword(w http.ResponseWriter, r *http.Request) {
	// Extract userID from auth middleware context
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		h.Logger.Error("user ID not found in context")
		h.RespondError(w, http.StatusUnauthorized, "user ID not found in context")
		return
	}

	// Parse request body
	var req models.UpdatePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.Logger.Error("failed to decode request body", zap.Error(err))
		h.RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Update password
	err := h.profileService.UpdatePassword(r.Context(), userID, req.Password)
	if err != nil {
		h.Logger.Error("failed to update password", zap.Error(err))
		statusCode := http.StatusInternalServerError
		// Check if it's a validation error
		errMsg := err.Error()
		if strings.Contains(errMsg, "cannot be empty") ||
			strings.Contains(errMsg, "password must be") ||
			strings.Contains(errMsg, "invalid") {
			statusCode = http.StatusBadRequest
		}
		h.RespondError(w, statusCode, err.Error())
		return
	}

	// Return 204 No Content
	w.WriteHeader(http.StatusNoContent)
}

// GetUserSettings handles GET /settings
// @Summary Get user settings
// @Description Get user settings for the authenticated user. Requires authentication.
// @Tags profile
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} models.UserSettingsResponse "User settings"
// @Failure 401 {object} map[string]string "Unauthorized - authentication required"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /settings [get]
func (h *ProfileHandler) GetUserSettings(w http.ResponseWriter, r *http.Request) {
	// Extract userID from auth middleware context
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		h.Logger.Error("user ID not found in context")
		h.RespondError(w, http.StatusUnauthorized, "user ID not found in context")
		return
	}

	// Get user settings
	settings, err := h.userSettingsService.GetUserSettings(r.Context(), userID)
	if err != nil {
		h.Logger.Error("failed to get user settings", zap.Error(err))
		h.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.RespondJSON(w, http.StatusOK, settings)
}

// UpdateUserSettings handles PATCH /settings
// @Summary Update user settings
// @Description Update user settings for the authenticated user. Requires authentication.
// @Tags profile
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param settings body models.UpdateUserSettingsRequest true "User settings to update"
// @Success 204 "No Content"
// @Failure 400 {object} map[string]string "Bad request - invalid request body or validation error"
// @Failure 401 {object} map[string]string "Unauthorized - authentication required"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /settings [patch]
func (h *ProfileHandler) UpdateUserSettings(w http.ResponseWriter, r *http.Request) {
	// Extract userID from auth middleware context
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		h.Logger.Error("user ID not found in context")
		h.RespondError(w, http.StatusUnauthorized, "user ID not found in context")
		return
	}

	// Parse request body
	var req models.UpdateUserSettingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.Logger.Error("failed to decode request body", zap.Error(err))
		h.RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Because user should update Repeat flag in separate endpoint, we need to ensure that it is not provided here
	if req.AlphabetRepeat != "" {
		req.AlphabetRepeat = ""
	}

	// Update user settings
	err := h.userSettingsService.UpdateUserSettings(r.Context(), userID, &req)
	if err != nil {
		h.Logger.Error("failed to update user settings", zap.Error(err))
		statusCode := http.StatusInternalServerError
		// Check if it's a validation error
		errMsg := err.Error()
		if errMsg == "newWordCount must be between 10 and 40" ||
			errMsg == "oldWordCount must be between 10 and 40" ||
			errMsg == "alphabetLearnCount must be between 5 and 15" ||
			(len(errMsg) >= 16 && errMsg[:16] == "invalid language") {
			statusCode = http.StatusBadRequest
		}
		h.RespondError(w, statusCode, err.Error())
		return
	}

	// Return 204 No Content
	w.WriteHeader(http.StatusNoContent)
}

// UpdateRepeatFlagRequest represents a request to update repeat flag
type UpdateRepeatFlagRequest struct {
	Flag string `json:"flag"`
}

// UpdateRepeatFlag handles PUT /profile/repeat-flag
// @Summary Update repeat flag
// @Description Update alphabet repeat flag for the authenticated user. Requires authentication.
// @Tags profile
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param flag body UpdateRepeatFlagRequest true "Repeat flag: 'in question', 'ignore', or 'repeat'"
// @Success 204 "No Content"
// @Failure 400 {object} map[string]string "Bad request - invalid flag value"
// @Failure 401 {object} map[string]string "Unauthorized - authentication required"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /profile/repeat-flag [put]
func (h *ProfileHandler) UpdateRepeatFlag(w http.ResponseWriter, r *http.Request) {
	// Extract userID from auth middleware context
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		h.Logger.Error("user ID not found in context")
		h.RespondError(w, http.StatusUnauthorized, "user ID not found in context")
		return
	}

	// Parse request body
	var req UpdateRepeatFlagRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.Logger.Error("failed to decode request body", zap.Error(err))
		h.RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.profileService.UpdateRepeatFlag(r.Context(), userID, req.Flag, r); err != nil {
		h.Logger.Error("failed to update repeat flag", zap.Error(err))
		statusCode := http.StatusInternalServerError
		if strings.Contains(err.Error(), "invalid") {
			statusCode = http.StatusBadRequest
		}
		h.RespondError(w, statusCode, err.Error())
		return
	}

	// Return 204 No Content
	w.WriteHeader(http.StatusNoContent)
}
