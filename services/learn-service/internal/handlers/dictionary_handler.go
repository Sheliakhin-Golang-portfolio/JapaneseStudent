package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/japanesestudent/learn-service/internal/models"
	"github.com/japanesestudent/libs/auth/middleware"
	"github.com/japanesestudent/libs/handlers"
	"go.uber.org/zap"
)

// DictionaryService is the interface that wraps methods for dictionary business logic
type DictionaryService interface {
	// GetWordList retrieves a mixed list of old and new words for the user
	//
	// "userId" parameter is used to identify the user.
	// "newCount" parameter is used to specify the number of new words to return.
	// "oldCount" parameter is used to specify the number of old words to return.
	// "locale" parameter is used to specify the locale of the words.
	// Please reference Locale constants for correct parameter values.
	//
	// If wrong parameters will be used or some error will occur during data retrieve, the error will be returned together with "nil" value.
	GetWordList(ctx context.Context, userId, newCount, oldCount int, locale string) ([]models.WordResponse, error)
	// SubmitWordResults validates and upserts word learning results
	//
	// "userId" parameter is used to identify the user.
	// "results" parameter is used to submit the word learning results.
	//
	// If wrong parameters will be used or some error will occur during data submission, the error will be returned together with "nil" value.
	SubmitWordResults(ctx context.Context, userId int, results []models.WordResult) error
}

// DictionaryHandler handles dictionary-related HTTP requests
type DictionaryHandler struct {
	handlers.BaseHandler
	service DictionaryService
}

// NewDictionaryHandler creates a new dictionary handler
func NewDictionaryHandler(service DictionaryService, logger *zap.Logger) *DictionaryHandler {
	return &DictionaryHandler{
		BaseHandler: handlers.BaseHandler{Logger: logger},
		service:     service,
	}
}

// RegisterRoutes registers all dictionary handler routes
func (h *DictionaryHandler) RegisterRoutes(r chi.Router, authMiddleware func(http.Handler) http.Handler) {
	r.Route("/words", func(r chi.Router) {
		r.Use(authMiddleware)
		r.Get("/", h.GetWordList)
		r.Post("/results", h.SubmitWordResults)
	})
}

// GetWordList handles GET /api/v1/words
// @Summary Get word list
// @Description Get a mixed list of old and new words for the authenticated user. Requires authentication.
// @Tags dictionary
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param newCount query int false "Number of new words (10-40), default: 20"
// @Param oldCount query int false "Number of old words (10-40), default: 20"
// @Param locale query string false "Locale: en, ru, or de, default: en"
// @Success 200 {array} models.WordResponse "List of words"
// @Failure 400 {object} map[string]string "Bad request - invalid parameters"
// @Failure 401 {object} map[string]string "Unauthorized - authentication required"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/v1/words [get]
func (h *DictionaryHandler) GetWordList(w http.ResponseWriter, r *http.Request) {
	// Extract userID from auth middleware context
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		h.Logger.Error("user ID not found in context")
		h.RespondError(w, http.StatusUnauthorized, "user ID not found in context")
		return
	}

	// Get query parameters
	newCountStr := r.URL.Query().Get("newCount")
	oldCountStr := r.URL.Query().Get("oldCount")
	locale := r.URL.Query().Get("locale")

	// Parse and validate newCount
	newCount := 20 // default
	if newCountStr != "" {
		parsed, err := strconv.Atoi(newCountStr)
		if err != nil {
			h.Logger.Error("failed to parse newCount parameter", zap.Error(err))
			h.RespondError(w, http.StatusBadRequest, "invalid newCount parameter")
			return
		}
		newCount = parsed
	}

	// Parse and validate oldCount
	oldCount := 20 // default
	if oldCountStr != "" {
		parsed, err := strconv.Atoi(oldCountStr)
		if err != nil {
			h.Logger.Error("failed to parse oldCount parameter", zap.Error(err))
			h.RespondError(w, http.StatusBadRequest, "invalid oldCount parameter")
			return
		}
		oldCount = parsed
	}

	// Default locale
	if locale == "" {
		locale = "en"
	}

	// Get word list
	words, err := h.service.GetWordList(r.Context(), userID, newCount, oldCount, locale)
	if err != nil {
		h.Logger.Error("failed to get word list", zap.Error(err))
		statusCode := http.StatusInternalServerError
		// Check if it's a validation error
		if err.Error() == "newWordCount must be between 10 and 40" ||
			err.Error() == "oldWordCount must be between 10 and 40" ||
			err.Error() == "invalid locale: "+locale+", must be 'en', 'ru', or 'de'" {
			statusCode = http.StatusBadRequest
		}
		h.RespondError(w, statusCode, err.Error())
		return
	}

	h.RespondJSON(w, http.StatusOK, words)
}

// SubmitWordResultsRequest represents a word results submission request
type SubmitWordResultsRequest struct {
	Results []models.WordResult `json:"results"`
}

// SubmitWordResults handles POST /api/v1/words/results
// @Summary Submit word learning results
// @Description Submit word learning results with period values. Requires authentication.
// @Tags dictionary
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param results body SubmitWordResultsRequest true "Word results"
// @Success 204 "No Content"
// @Failure 400 {object} map[string]string "Bad request - invalid request body or validation error"
// @Failure 401 {object} map[string]string "Unauthorized - authentication required"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /api/v1/words/results [post]
func (h *DictionaryHandler) SubmitWordResults(w http.ResponseWriter, r *http.Request) {
	// Extract userID from auth middleware context
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		h.Logger.Error("user ID not found in context")
		h.RespondError(w, http.StatusUnauthorized, "user ID not found in context")
		return
	}

	// Parse request body
	var req SubmitWordResultsRequest
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

	// Submit word results
	err := h.service.SubmitWordResults(r.Context(), userID, req.Results)
	if err != nil {
		h.Logger.Error("failed to submit word results", zap.Error(err))
		statusCode := http.StatusInternalServerError
		// Check if it's a validation error
		errMsg := err.Error()
		if errMsg == "results list cannot be empty" ||
			errMsg == "one or more word IDs do not exist" ||
			strings.Contains(errMsg, "period must be between") {
			statusCode = http.StatusBadRequest
		}
		h.RespondError(w, statusCode, errMsg)
		return
	}

	// Return 204 No Content
	w.WriteHeader(http.StatusNoContent)
}
