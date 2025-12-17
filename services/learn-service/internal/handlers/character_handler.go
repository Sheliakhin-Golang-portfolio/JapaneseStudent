package handlers

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/japanesestudent/learn-service/internal/models"
	"github.com/japanesestudent/libs/handlers"
	"go.uber.org/zap"
)

// CharactersService is the interface that wraps methods for Characters business logic.
type CharactersService interface {
	// Method GetAll retrieve a list of all hiragana/katakana characters using configured repository.
	//
	// "alphabetType" and "locale" parameters are used to configure return type of characters (hiragana or katakana) and reading (russian or english).
	// Please reference AlphabetType and Locale constants for correct parameters values.
	//
	// If wrong parameters will be used or some error will occur during data retrieve, the error will be returned together with "nil" value.
	GetAll(ctx context.Context, alphabetType string, locale string) ([]models.CharacterResponse, error)
	// Method GetByRowColumn retrieve hiragana/katakana characters of the same consonant or vowel group ("character" parameter) using configured repository.
	//
	// Please reference GetAll method for more information about parameters and error values.
	GetByRowColumn(ctx context.Context, typeParam string, localeParam string, character string) ([]models.CharacterResponse, error)
	// Method GetByID retrieve a character by its ID using configured repository.
	//
	// "id" parameter is used to identify the character.
	// "locale" parameter is used to configure return type of characters (hiragana or katakana) and reading (russian or english).
	// Please reference Locale constants for correct parameter values.
	//
	// If wrong parameters will be used or some error will occur during data retrieve, the error will be returned together with "nil" value.
	GetByID(ctx context.Context, id int, localeParam string) (*models.Character, error)
	// Method GetReadingTest retrieve a list of random characters for reading test using configured repository.
	//
	// "count" parameter is used to specify the number of characters to return (default: 10).
	//
	// Please reference GetAll method for more information about other parameters and error values.
	GetReadingTest(ctx context.Context, alphabetTypeStr string, localeParam string, count int) ([]models.ReadingTestItem, error)
	// Method GetWritingTest retrieve a list of random characters for writing test using configured repository.
	//
	// Please reference GetReadingTest method for more information about parameters and error values.
	GetWritingTest(ctx context.Context, alphabetTypeStr string, localeParam string, count int) ([]models.WritingTestItem, error)
}

// Handler handles HTTP requests for characters
type CharactersHandler struct {
	handlers.BaseHandler
	service CharactersService
}

// NewCharactersHandler creates a new character handler
func NewCharactersHandler(svc CharactersService, logger *zap.Logger) *CharactersHandler {
	return &CharactersHandler{
		service:     svc,
		BaseHandler: handlers.BaseHandler{Logger: logger},
	}
}

// RegisterRoutes registers all character handler routes
// Note: This assumes the router is already scoped to /api/v1
func (h *CharactersHandler) RegisterRoutes(r chi.Router, authMiddleware func(http.Handler) http.Handler) {
	r.Route("/characters", func(r chi.Router) {
		r.Get("/", h.GetAll)
		r.Get("/row-column", h.GetByRowColumn)
		r.Get("/{id}", h.GetByID)
	})
	r.Route("/tests", func(r chi.Router) {
		// Apply auth middleware to all test routes
		r.Use(authMiddleware)
		r.Get("/{type}/reading", h.GetReadingTest)
		r.Get("/{type}/writing", h.GetWritingTest)
	})
}

// GetAll handles GET /api/v1/characters
// @Summary Get all characters
// @Description Get a list of all hiragana or katakana characters
// @Tags characters
// @Accept json
// @Produce json
// @Param type query string false "Alphabet type: hr (hiragana) or kt (katakana), default: hr"
// @Param locale query string false "Locale: en (English), ru (Russian), or de (German - treated as English), default: en"
// @Success 200 {array} models.CharacterResponse "List of characters"
// @Failure 500 {object} map[string]string "Internal server error - failed to retrieve characters"
// @Router /api/v1/characters [get]
func (h *CharactersHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	typeParam := r.URL.Query().Get("type")
	localeParam := r.URL.Query().Get("locale")

	if typeParam == "" {
		typeParam = "hr"
	}
	if localeParam == "" {
		localeParam = "en"
	}

	characters, err := h.service.GetAll(r.Context(), typeParam, localeParam)
	if err != nil {
		h.Logger.Error("failed to get all characters", zap.Error(err))
		h.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.RespondJSON(w, http.StatusOK, characters)
}

// GetByRowColumn handles GET /api/v1/characters/row-column
// @Summary Get characters by row or column
// @Description Get a row or column of hiragana or katakana characters filtered by consonant or vowel
// @Tags characters
// @Accept json
// @Produce json
// @Param type query string false "Alphabet type: hr (hiragana) or kt (katakana), default: hr"
// @Param locale query string false "Locale: en (English), ru (Russian), or de (German - treated as English), default: en"
// @Param character query string true "Consonant or vowel character"
// @Success 200 {array} models.CharacterResponse "List of characters matching the filter"
// @Failure 400 {object} map[string]string "Bad request - character parameter is required"
// @Failure 500 {object} map[string]string "Internal server error - failed to retrieve characters"
// @Router /api/v1/characters/row-column [get]
func (h *CharactersHandler) GetByRowColumn(w http.ResponseWriter, r *http.Request) {
	typeParam := r.URL.Query().Get("type")
	localeParam := r.URL.Query().Get("locale")
	characterParam := r.URL.Query().Get("character")

	if typeParam == "" {
		typeParam = "hr"
	}
	if localeParam == "" {
		localeParam = "en"
	}
	if characterParam == "" {
		h.RespondError(w, http.StatusBadRequest, "character parameter is required")
		return
	}

	characters, err := h.service.GetByRowColumn(r.Context(), typeParam, localeParam, characterParam)
	if err != nil {
		h.Logger.Error("failed to get characters by row/column", zap.Error(err))
		h.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.RespondJSON(w, http.StatusOK, characters)
}

// GetByID handles GET /api/v1/characters/{id}
// @Summary Get character by ID
// @Description Get a hiragana or katakana character by its ID
// @Tags characters
// @Accept json
// @Produce json
// @Param id path int true "Character ID"
// @Param locale query string false "Locale: en (English), ru (Russian), or de (German - treated as English), default: en"
// @Success 200 {object} models.Character "Character details"
// @Failure 400 {object} map[string]string "Bad request - id parameter is required or invalid id parameter"
// @Failure 404 {object} map[string]string "Not found - character not found"
// @Failure 500 {object} map[string]string "Internal server error - failed to retrieve character"
// @Router /api/v1/characters/{id} [get]
func (h *CharactersHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	idParam := chi.URLParam(r, "id")
	localeParam := r.URL.Query().Get("locale")

	if idParam == "" {
		h.RespondError(w, http.StatusBadRequest, "id parameter is required")
		return
	}
	if localeParam == "" {
		localeParam = "en"
	}

	id, err := strconv.Atoi(idParam)
	if err != nil {
		h.RespondError(w, http.StatusBadRequest, "invalid id parameter")
		return
	}

	character, err := h.service.GetByID(r.Context(), id, localeParam)
	if err != nil {
		errStatus := http.StatusInternalServerError
		// Check if error is "character not found" (may be wrapped)
		if strings.Contains(err.Error(), "character not found") {
			errStatus = http.StatusNotFound
		}
		h.Logger.Error("failed to get character by id", zap.Error(err))
		h.RespondError(w, errStatus, err.Error())
		return
	}

	h.RespondJSON(w, http.StatusOK, character)
}

// GetReadingTest handles GET /api/v1/tests/{type}/reading
// @Summary Get reading test
// @Description Get random characters for reading test. Requires authentication.
// @Tags tests
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param type path string true "Alphabet type: hiragana or katakana"
// @Param locale query string false "Locale: en (English), ru (Russian), or de (German - treated as English), default: en"
// @Param count query int false "Number of characters to return, default: 10"
// @Success 200 {array} models.ReadingTestItem "List of random characters for reading test"
// @Failure 400 {object} map[string]string "Bad request - type parameter is required or invalid alphabet type/locale/count"
// @Failure 401 {object} map[string]string "Unauthorized - authentication required or invalid/expired token"
// @Failure 500 {object} map[string]string "Internal server error - failed to retrieve test characters"
// @Router /api/v1/tests/{type}/reading [get]
func (h *CharactersHandler) GetReadingTest(w http.ResponseWriter, r *http.Request) {
	typeParam := chi.URLParam(r, "type")
	localeParam := r.URL.Query().Get("locale")
	countStr := r.URL.Query().Get("count")

	if typeParam == "" {
		h.RespondError(w, http.StatusBadRequest, "type parameter is required")
		return
	}
	if localeParam == "" {
		localeParam = "en"
	}

	// Parse count parameter
	count := 10 // default
	if countStr != "" {
		parsed, err := strconv.Atoi(countStr)
		if err != nil || parsed <= 0 {
			h.RespondError(w, http.StatusBadRequest, "invalid count parameter")
			return
		}
		count = parsed
	}

	items, err := h.service.GetReadingTest(r.Context(), typeParam, localeParam, count)
	if err != nil {
		h.Logger.Error("failed to get reading test", zap.Error(err))
		h.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}

	h.RespondJSON(w, http.StatusOK, items)
}

// GetWritingTest handles GET /api/v1/tests/{type}/writing
// @Summary Get writing test
// @Description Get random characters for writing test with multiple choice options. Requires authentication.
// @Tags tests
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param type path string true "Alphabet type: hiragana or katakana"
// @Param locale query string false "Locale: en (English), ru (Russian), or de (German - treated as English), default: en"
// @Param count query int false "Number of characters to return, default: 10"
// @Success 200 {array} models.WritingTestItem "List of random characters for writing test with multiple choice options"
// @Failure 400 {object} map[string]string "Bad request - type parameter is required or invalid alphabet type/locale/count"
// @Failure 401 {object} map[string]string "Unauthorized - authentication required or invalid/expired token"
// @Failure 500 {object} map[string]string "Internal server error - failed to retrieve test characters"
// @Router /api/v1/tests/{type}/writing [get]
func (h *CharactersHandler) GetWritingTest(w http.ResponseWriter, r *http.Request) {
	typeParam := chi.URLParam(r, "type")
	localeParam := r.URL.Query().Get("locale")
	countStr := r.URL.Query().Get("count")

	if typeParam == "" {
		h.RespondError(w, http.StatusBadRequest, "type parameter is required")
		return
	}
	if localeParam == "" {
		localeParam = "en"
	}

	// Parse count parameter
	count := 10 // default
	if countStr != "" {
		parsed, err := strconv.Atoi(countStr)
		if err != nil || parsed <= 0 {
			h.RespondError(w, http.StatusBadRequest, "invalid count parameter")
			return
		}
		count = parsed
	}

	items, err := h.service.GetWritingTest(r.Context(), typeParam, localeParam, count)
	if err != nil {
		h.Logger.Error("failed to get writing test", zap.Error(err))
		h.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}

	h.RespondJSON(w, http.StatusOK, items)
}
