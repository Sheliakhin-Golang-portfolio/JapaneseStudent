package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/japanesestudent/learn-service/internal/models"
	"github.com/japanesestudent/libs/auth/middleware"
	"github.com/japanesestudent/libs/handlers"
	"go.uber.org/zap"
)

// TestResultService defines methods for test result business logic
type TestResultService interface {
	// SubmitTestResults updates or creates a list of learn history records for chosen characters.
	//
	// "userID" parameter is used to identify the user.
	// "alphabetType" parameter is used to identify the alphabet type.
	// "testType" parameter is used to identify the test type.
	// "results" parameter is used to update or create a list of learn history records.
	// "repeat" parameter indicates if user wants to repeat alphabet ("in question" by default).
	// Returns result with askForRepeat flag and error.
	// If some error occurs during data update or creation, the error will be returned.
	SubmitTestResults(ctx context.Context, userID int, alphabetType, testType string, results []models.TestResultItem, repeat string) (*models.SubmitTestResultsResult, error)
	// GetUserHistory retrieves all learn history records for a user.
	//
	// "userID" parameter is used to identify the user.
	// If no records are found, an empty slice will be returned.
	// If some error occurs during data retrieval, the error will be returned.
	GetUserHistory(ctx context.Context, userID int) ([]models.UserLearnHistory, error)
	// DropUserMarks lowers all CharacterLearnHistory results by 0.01 for a user
	//
	// "userID" parameter is used to identify the user.
	// If some error occurs during the update, the error will be returned.
	DropUserMarks(ctx context.Context, userID int) error
}

// TestResultHandler handles test result submission
type TestResultHandler struct {
	handlers.BaseHandler
	service TestResultService
}

// NewTestResultHandler creates a new test result handler
func NewTestResultHandler(service TestResultService, logger *zap.Logger) *TestResultHandler {
	return &TestResultHandler{
		BaseHandler: handlers.BaseHandler{Logger: logger},
		service:     service,
	}
}

// TestResultRequest represents a test result submission request
type TestResultRequest struct {
	Results []models.TestResultItem `json:"results"`
	Repeat  string                  `json:"repeat,omitempty"`
}

// RegisterRoutes registers all test result handler routes
func (h *TestResultHandler) RegisterRoutes(r chi.Router, authMiddleware func(http.Handler) http.Handler, apiKeyMiddleware func(http.Handler) http.Handler) {
	r.Route("/test-results", func(r chi.Router) {
		// Apply auth middleware to user-facing routes
		r.Group(func(r chi.Router) {
			r.Use(authMiddleware)
			r.Post("/{type}/{testType}", h.SubmitTestResult)
			r.Get("/history", h.GetUserHistory)
		})
		// Apply API key middleware to service-to-service routes
		r.Group(func(r chi.Router) {
			r.Use(apiKeyMiddleware)
			r.Get("/drop-marks/{userId}", h.DropUserMarks)
		})
	})
}

// SubmitTestResult handles POST /tests/{type}/{testType}
// @Summary Submit test results
// @Description Submit test results for hiragana or katakana reading, writing, or listening tests. Requires authentication.
// @Tags tests
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param type path string true "Alphabet type: hiragana or katakana"
// @Param testType path string true "Test type: reading, writing, or listening"
// @Param results body TestResultRequest true "Test results"
// @Success 200 {object} map[string]string "Test results submitted successfully"
// @Failure 400 {object} map[string]string "Bad request - invalid request body, empty results array, or invalid alphabet/test type"
// @Failure 401 {object} map[string]string "Unauthorized - authentication required, invalid/expired token, or user ID not found in context"
// @Failure 500 {object} map[string]string "Internal server error - failed to process or save test results"
// @Router /test-results/{type}/{testType} [post]
func (h *TestResultHandler) SubmitTestResult(w http.ResponseWriter, r *http.Request) {
	// Extract userID from auth middleware context
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		h.Logger.Error("user ID not found in context")
		h.RespondError(w, http.StatusUnauthorized, "user ID not found in context")
		return
	}

	// Get path parameters
	alphabetType := chi.URLParam(r, "type") // "hiragana" or "katakana"
	testType := chi.URLParam(r, "testType") // "reading", "writing", or "listening"

	// Parse request body
	var req TestResultRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.Logger.Error("failed to decode request body", zap.Error(err))
		h.RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if len(req.Results) == 0 {
		h.Logger.Error("results array cannot be empty")
		h.RespondError(w, http.StatusBadRequest, "results array cannot be empty")
		return
	}

	// Submit test results
	result, err := h.service.SubmitTestResults(r.Context(), userID, alphabetType, testType, req.Results, req.Repeat)
	if err != nil {
		h.Logger.Error("failed to submit test results", zap.Error(err))
		statusCode := http.StatusInternalServerError
		// Check if it's a validation error (should return 400)
		if err.Error() == "invalid alphabet type, must be 'hiragana' or 'katakana'" ||
			err.Error() == "invalid test type, must be 'reading', 'writing', or 'listening'" ||
			err.Error() == "invalid alphabet type or test type" {
			statusCode = http.StatusBadRequest
		}
		h.RespondError(w, statusCode, err.Error())
		return
	}

	h.RespondJSON(w, http.StatusOK, map[string]any{
		"message":      "test results submitted successfully",
		"askForRepeat": result.AskForRepeat,
	})
}

// GetUserHistory handles GET /tests/history
// @Summary Get user's learn history
// @Description Get all learn history records for the authenticated user. Requires authentication.
// @Tags tests
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {array} models.UserLearnHistory "List of learn history records (empty array if no records found)"
// @Failure 401 {object} map[string]string "Unauthorized - authentication required, invalid/expired token, or user ID not found in context"
// @Failure 500 {object} map[string]string "Internal server error - failed to retrieve learn history"
// @Router /test-results/history [get]
func (h *TestResultHandler) GetUserHistory(w http.ResponseWriter, r *http.Request) {
	// Extract userID from auth middleware context
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		h.Logger.Error("user ID not found in context")
		h.RespondError(w, http.StatusUnauthorized, "user ID not found in context")
		return
	}

	// Get user's learn history
	histories, err := h.service.GetUserHistory(r.Context(), userID)
	if err != nil {
		h.Logger.Error("failed to get user history", zap.Error(err))
		h.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.RespondJSON(w, http.StatusOK, histories)
}

// DropUserMarks handles GET /test-results/drop-marks/{userId}
// @Summary Drop user marks
// @Description Lowers all CharacterLearnHistory results by 0.01 for a user. Requires API key authentication.
// @Tags tests
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param userId path int true "User ID"
// @Success 200 {object} map[string]string "Marks dropped successfully"
// @Failure 400 {object} map[string]string "Bad request - invalid user ID"
// @Failure 401 {object} map[string]string "Unauthorized - invalid or missing API key"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /test-results/drop-marks/{userId} [get]
func (h *TestResultHandler) DropUserMarks(w http.ResponseWriter, r *http.Request) {
	// Extract userId from URL path
	userIDStr := chi.URLParam(r, "userId")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		h.Logger.Error("failed to parse user ID", zap.Error(err))
		h.RespondError(w, http.StatusBadRequest, "invalid user ID")
		return
	}

	// Drop user marks
	if err := h.service.DropUserMarks(r.Context(), userID); err != nil {
		h.Logger.Error("failed to drop user marks", zap.Error(err))
		h.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.RespondJSON(w, http.StatusOK, map[string]string{"message": "marks dropped successfully"})
}
