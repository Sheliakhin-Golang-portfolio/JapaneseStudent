package handlers

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/japanesestudent/backend/internal/models"
	"go.uber.org/zap"
)

// CharactersService is the interface that wraps methods for Characters business logic.
type CharactersService interface {
	// Method GetAll retrieve a list of all hiragana/katakana characters using configured repository.
	//
	// "alphabetType" and "locale" parameters are used to configure return type of characters (hiragana or katakana) and reading (russian or english).
	// Please reference AlphabetType and Locale constants for correct parameters values.
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
	// If wrong parameters will be used or some error will occur during data retrieve, the error will be returned together with "nil" value.
	GetByID(ctx context.Context, id int, localeParam string) (*models.Character, error)
	// Method GetReadingTest retrieve a list of random characters for reading test using configured repository.
	//
	// "alphabetTypeStr" parameter is used to configure return type of characters (hiragana or katakana).
	// "localeParam" parameter is used to configure return type of reading (russian or english).
	// Please reference AlphabetType and Locale constants for correct parameters values.
	// If wrong parameters will be used or some error will occur during data retrieve, the error will be returned together with "nil" value.
	GetReadingTest(ctx context.Context, alphabetTypeStr string, localeParam string) ([]models.ReadingTestItem, error)
	// Method GetWritingTest retrieve a list of random characters for writing test using configured repository.
	//
	// "alphabetTypeStr" parameter is used to configure return type of characters (hiragana or katakana).
	// "localeParam" parameter is used to configure return type of reading (russian or english).
	// Please reference AlphabetType and Locale constants for correct parameters values.
	// If wrong parameters will be used or some error will occur during data retrieve, the error will be returned together with "nil" value.
	GetWritingTest(ctx context.Context, alphabetTypeStr string, localeParam string) ([]models.WritingTestItem, error)
}

// Handler handles HTTP requests for characters
type CharactersHandler struct {
	BaseHandler
	service CharactersService
}

// NewCharactersHandler creates a new character handler
func NewCharactersHandler(svc CharactersService, logger *zap.Logger) *CharactersHandler {
	return &CharactersHandler{
		service:     svc,
		BaseHandler: BaseHandler{logger: logger},
	}
}

// RegisterRoutes registers all character handler routes
func (h *CharactersHandler) RegisterRoutes(r chi.Router) {
	r.Route("/api/v1", func(r chi.Router) {
		r.Route("/characters", func(r chi.Router) {
			r.Get("/", h.GetAll)
			r.Get("/row-column", h.GetByRowColumn)
			r.Get("/{id}", h.GetByID)
		})
		r.Route("/tests", func(r chi.Router) {
			r.Get("/{type}/reading", h.GetReadingTest)
			r.Get("/{type}/writing", h.GetWritingTest)
		})
	})
}

// GetAll handles GET /api/v1/characters
// @Summary Get all characters
// @Description Get a list of all hiragana or katakana characters
// @Tags characters
// @Accept json
// @Produce json
// @Param type query string false "Alphabet type: hr (hiragana) or kt (katakana), default: hr"
// @Param locale query string false "Locale: en (English) or ru (Russian), default: en"
// @Success 200 {array} model.CharacterResponse
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
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
		h.logger.Error("failed to get all characters", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "failed to get characters")
		return
	}

	h.respondJSON(w, http.StatusOK, characters)
}

// GetByRowColumn handles GET /api/v1/characters/row-column
// @Summary Get characters by row or column
// @Description Get a row or column of hiragana or katakana characters filtered by consonant or vowel
// @Tags characters
// @Accept json
// @Produce json
// @Param type query string false "Alphabet type: hr (hiragana) or kt (katakana), default: hr"
// @Param locale query string false "Locale: en (English) or ru (Russian), default: en"
// @Param character query string true "Consonant or vowel character"
// @Success 200 {array} model.CharacterResponse
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
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
		h.respondError(w, http.StatusBadRequest, "character parameter is required")
		return
	}

	characters, err := h.service.GetByRowColumn(r.Context(), typeParam, localeParam, characterParam)
	if err != nil {
		h.logger.Error("failed to get characters by row/column", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "failed to get characters")
		return
	}

	h.respondJSON(w, http.StatusOK, characters)
}

// GetByID handles GET /api/v1/characters/{id}
// @Summary Get character by ID
// @Description Get a hiragana or katakana character by its ID
// @Tags characters
// @Accept json
// @Produce json
// @Param id path int true "Character ID"
// @Param locale query string false "Locale: en (English) or ru (Russian), default: en"
// @Success 200 {object} model.Character
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/characters/{id} [get]
func (h *CharactersHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	idParam := chi.URLParam(r, "id")
	localeParam := r.URL.Query().Get("locale")

	if idParam == "" {
		h.respondError(w, http.StatusBadRequest, "id parameter is required")
		return
	}
	if localeParam == "" {
		localeParam = "en"
	}

	id, err := strconv.Atoi(idParam)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid id parameter")
		return
	}

	character, err := h.service.GetByID(r.Context(), id, localeParam)
	if err != nil {
		// Check if error is "character not found" (may be wrapped)
		if strings.Contains(err.Error(), "character not found") {
			h.respondError(w, http.StatusNotFound, "character not found")
			return
		}
		h.logger.Error("failed to get character by id", zap.Error(err))
		h.respondError(w, http.StatusInternalServerError, "failed to get character")
		return
	}

	h.respondJSON(w, http.StatusOK, character)
}

// GetReadingTest handles GET /api/v1/tests/{type}/reading
// @Summary Get reading test
// @Description Get 20 random characters for reading test
// @Tags tests
// @Accept json
// @Produce json
// @Param type path string true "Alphabet type: hiragana or katakana"
// @Param locale query string false "Locale: en (English) or ru (Russian), default: en"
// @Success 200 {array} model.ReadingTestItem
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/tests/{type}/reading [get]
func (h *CharactersHandler) GetReadingTest(w http.ResponseWriter, r *http.Request) {
	typeParam := chi.URLParam(r, "type")
	localeParam := r.URL.Query().Get("locale")

	if typeParam == "" {
		h.respondError(w, http.StatusBadRequest, "type parameter is required")
		return
	}
	if localeParam == "" {
		localeParam = "en"
	}

	items, err := h.service.GetReadingTest(r.Context(), typeParam, localeParam)
	if err != nil {
		h.logger.Error("failed to get reading test", zap.Error(err))
		h.respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	h.respondJSON(w, http.StatusOK, items)
}

// GetWritingTest handles GET /api/v1/tests/{type}/writing
// @Summary Get writing test
// @Description Get 20 random characters for writing test with multiple choice options
// @Tags tests
// @Accept json
// @Produce json
// @Param type path string true "Alphabet type: hiragana or katakana"
// @Param locale query string false "Locale: en (English) or ru (Russian), default: en"
// @Success 200 {array} model.WritingTestItem
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/tests/{type}/writing [get]
func (h *CharactersHandler) GetWritingTest(w http.ResponseWriter, r *http.Request) {
	typeParam := chi.URLParam(r, "type")
	localeParam := r.URL.Query().Get("locale")

	if typeParam == "" {
		h.respondError(w, http.StatusBadRequest, "type parameter is required")
		return
	}
	if localeParam == "" {
		localeParam = "en"
	}

	items, err := h.service.GetWritingTest(r.Context(), typeParam, localeParam)
	if err != nil {
		h.logger.Error("failed to get writing test", zap.Error(err))
		h.respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	h.respondJSON(w, http.StatusOK, items)
}
