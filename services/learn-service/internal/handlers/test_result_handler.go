package handlers

import (
	"context"
	"encoding/json"
	"net/http"

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
	// If some error occurs during data update or creation, the error will be returned.
	SubmitTestResults(ctx context.Context, userID int, alphabetType, testType string, results []models.TestResultItem) error
	// GetUserHistory retrieves all learn history records for a user.
	//
	// "userID" parameter is used to identify the user.
	// If no records are found, an empty slice will be returned.
	// If some error occurs during data retrieval, the error will be returned.
	GetUserHistory(ctx context.Context, userID int) ([]models.UserLearnHistory, error)
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
}

// RegisterRoutes registers all test result handler routes
func (h *TestResultHandler) RegisterRoutes(r chi.Router, authMiddleware func(http.Handler) http.Handler) {
	r.Route("/test-results", func(r chi.Router) {
		// Apply auth middleware to all test routes
		r.Use(authMiddleware)
		r.Post("/{type}/{testType}", h.SubmitTestResult)
		r.Get("/history", h.GetUserHistory)
	})
}

// SubmitTestResult handles POST /api/v1/tests/{type}/{testType}
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
// @Router /api/v1/test-results/{type}/{testType} [post]
func (h *TestResultHandler) SubmitTestResult(w http.ResponseWriter, r *http.Request) {
	// Extract userID from auth middleware context
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		// Fallback to context value for testing
		if ctxUserID, ok := r.Context().Value("userID").(int); ok {
			userID = ctxUserID
		} else {
			h.RespondError(w, http.StatusUnauthorized, "user ID not found in context")
			return
		}
	}

	// Get path parameters
	alphabetType := chi.URLParam(r, "type") // "hiragana" or "katakana"
	testType := chi.URLParam(r, "testType") // "reading", "writing", or "listening"

	// Parse request body
	var req TestResultRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if len(req.Results) == 0 {
		h.RespondError(w, http.StatusBadRequest, "results array cannot be empty")
		return
	}

	// Submit test results
	err := h.service.SubmitTestResults(r.Context(), userID, alphabetType, testType, req.Results)
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

	h.RespondJSON(w, http.StatusOK, map[string]string{"message": "test results submitted successfully"})
}

// GetUserHistory handles GET /api/v1/tests/history
// @Summary Get user's learn history
// @Description Get all learn history records for the authenticated user. Requires authentication.
// @Tags tests
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {array} models.UserLearnHistory "List of learn history records (empty array if no records found)"
// @Failure 401 {object} map[string]string "Unauthorized - authentication required, invalid/expired token, or user ID not found in context"
// @Failure 500 {object} map[string]string "Internal server error - failed to retrieve learn history"
// @Router /api/v1/test-results/history [get]
func (h *TestResultHandler) GetUserHistory(w http.ResponseWriter, r *http.Request) {
	// Extract userID from auth middleware context
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		// Fallback to context value for testing
		if ctxUserID, ok := r.Context().Value("userID").(int); ok {
			userID = ctxUserID
		} else {
			h.RespondError(w, http.StatusUnauthorized, "user ID not found in context")
			return
		}
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
