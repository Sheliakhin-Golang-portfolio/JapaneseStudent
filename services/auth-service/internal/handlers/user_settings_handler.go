package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/japanesestudent/auth-service/internal/models"
	"github.com/japanesestudent/libs/auth/middleware"
	"github.com/japanesestudent/libs/handlers"
	"go.uber.org/zap"
)

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

// UserSettingsHandler handles user settings HTTP requests
type UserSettingsHandler struct {
	handlers.BaseHandler
	service UserSettingsService
}

// NewUserSettingsHandler creates a new user settings handler
func NewUserSettingsHandler(service UserSettingsService, logger *zap.Logger) *UserSettingsHandler {
	return &UserSettingsHandler{
		BaseHandler: handlers.BaseHandler{Logger: logger},
		service:     service,
	}
}

// RegisterRoutes registers all user settings handler routes
func (h *UserSettingsHandler) RegisterRoutes(r chi.Router, authMiddleware func(http.Handler) http.Handler) {
	r.Route("/settings", func(r chi.Router) {
		r.Use(authMiddleware)
		r.Get("/", h.GetUserSettings)
		r.Patch("/", h.UpdateUserSettings)
	})
}

// GetUserSettings handles GET /api/v1/settings
// @Summary Get user settings
// @Description Get user settings for the authenticated user. Requires authentication.
// @Tags settings
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} models.UserSettingsResponse "User settings"
// @Failure 401 {object} map[string]string "Unauthorized - authentication required"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/v1/settings [get]
func (h *UserSettingsHandler) GetUserSettings(w http.ResponseWriter, r *http.Request) {
	// Extract userID from auth middleware context
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		h.RespondError(w, http.StatusUnauthorized, "user ID not found in context")
		return
	}

	// Get user settings
	settings, err := h.service.GetUserSettings(r.Context(), userID)
	if err != nil {
		h.Logger.Error("failed to get user settings", zap.Error(err))
		h.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.RespondJSON(w, http.StatusOK, settings)
}

// UpdateUserSettings handles PATCH /api/v1/settings
// @Summary Update user settings
// @Description Update user settings for the authenticated user. Requires authentication.
// @Tags settings
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param settings body models.UpdateUserSettingsRequest true "User settings to update"
// @Success 204 "No Content"
// @Failure 400 {object} map[string]string "Bad request - invalid request body or validation error"
// @Failure 401 {object} map[string]string "Unauthorized - authentication required"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/v1/settings [patch]
func (h *UserSettingsHandler) UpdateUserSettings(w http.ResponseWriter, r *http.Request) {
	// Extract userID from auth middleware context
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		h.RespondError(w, http.StatusUnauthorized, "user ID not found in context")
		return
	}

	// Parse request body
	var req models.UpdateUserSettingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Validate that at least one field is provided
	if req.NewWordCount == nil && req.OldWordCount == nil && req.AlphabetLearnCount == nil && req.Language == nil {
		h.RespondError(w, http.StatusBadRequest, "at least one field must be provided")
		return
	}

	// Update user settings
	err := h.service.UpdateUserSettings(r.Context(), userID, &req)
	if err != nil {
		h.Logger.Error("failed to update user settings", zap.Error(err))
		statusCode := http.StatusInternalServerError
		// Check if it's a validation error
		if err.Error() == "newWordCount must be between 10 and 40" ||
			err.Error() == "oldWordCount must be between 10 and 40" ||
			err.Error() == "alphabetLearnCount must be between 5 and 15" ||
			len(err.Error()) > 0 && err.Error()[:15] == "invalid language" {
			statusCode = http.StatusBadRequest
		}
		h.RespondError(w, statusCode, err.Error())
		return
	}

	// Return 204 No Content
	w.WriteHeader(http.StatusNoContent)
}
