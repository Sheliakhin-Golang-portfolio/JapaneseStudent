package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/japanesestudent/learn-service/internal/models"
	"github.com/japanesestudent/libs/handlers"
	"go.uber.org/zap"
)

// AdminWordsService is the interface that wraps methods for admin word operations
type AdminWordsService interface {
	// Method GetAllForAdmin retrieve a list of all words using configured repository.
	//
	// "page" parameter is used to specify the page number.
	// "count" parameter is used to specify the number of items per page.
	// "search" parameter is used to search words by word, phonetic clues, or translations.
	//
	// If some error will occur during data retrieve, the error will be returned together with "nil" value.
	GetAllForAdmin(ctx context.Context, page, count int, search string) ([]models.WordListItem, error)
	// Method GetByIDAdmin retrieve a word by its ID using configured repository.
	//
	// "id" parameter is used to identify the word.
	//
	// If some error will occur during data retrieve, the error will be returned together with "nil" value.
	GetByIDAdmin(ctx context.Context, id int) (*models.Word, error)
	// Method CreateWord creates a new word using configured repository.
	//
	// "word" parameter is used to create a new word.
	//
	// If some error will occur during data creation, the error will be returned together with 0 as word ID.
	CreateWord(ctx context.Context, word *models.CreateWordRequest) (int, error)
	// Method UpdateWord updates a word using configured repository.
	//
	// "id" parameter is used to identify the word.
	// "word" parameter is used to update the word.
	//
	// If some error will occur during data update, the error will be returned.
	UpdateWord(ctx context.Context, id int, word *models.UpdateWordRequest) error
	// Method DeleteWord deletes a word using configured repository.
	//
	// "id" parameter is used to identify the word.
	//
	// If some error will occur during data deletion, the error will be returned.
	DeleteWord(ctx context.Context, id int) error
}

// AdminWordsHandler handles admin-related HTTP requests for words
type AdminWordsHandler struct {
	handlers.BaseHandler
	service AdminWordsService
}

// NewAdminWordsHandler creates a new admin word handler
func NewAdminWordsHandler(svc AdminWordsService, logger *zap.Logger) *AdminWordsHandler {
	return &AdminWordsHandler{
		service:     svc,
		BaseHandler: handlers.BaseHandler{Logger: logger},
	}
}

// RegisterRoutes registers all admin word handler routes
// Note: This assumes the router is already scoped to /api/v3
func (h *AdminWordsHandler) RegisterRoutes(r chi.Router) {
	r.Route("/admin/words", func(r chi.Router) {
		r.Get("/", h.GetAll)
		r.Get("/{id}", h.GetByID)
		r.Post("/", h.Create)
		r.Patch("/{id}", h.Update)
		r.Delete("/{id}", h.Delete)
	})
}

// GetAll handles GET /admin/words
// @Summary Get list of words
// @Description Get paginated list of words with optional search filter
// @Tags admin
// @Accept json
// @Produce json
// @Param page query int false "Page number (default: 1)"
// @Param count query int false "Items per page (default: 20)"
// @Param search query string false "Search in word, phonetic clues, or translations"
// @Success 200 {array} models.WordListItem "List of words"
// @Failure 400 {object} map[string]string "Invalid request parameters"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /admin/words [get]
func (h *AdminWordsHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
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

	words, err := h.service.GetAllForAdmin(r.Context(), page, count, search)
	if err != nil {
		h.Logger.Error("failed to get words list", zap.Error(err))
		h.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.RespondJSON(w, http.StatusOK, words)
}

// GetByID handles GET /admin/words/{id}
// @Summary Get word by ID
// @Description Get full information about a word by ID
// @Tags admin
// @Accept json
// @Produce json
// @Param id path int true "Word ID"
// @Success 200 {object} models.Word "Word information"
// @Failure 400 {object} map[string]string "Invalid word ID"
// @Failure 404 {object} map[string]string "Word not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /admin/words/{id} [get]
func (h *AdminWordsHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	// Parse word ID
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		h.Logger.Error("failed to parse word ID", zap.Error(err))
		h.RespondError(w, http.StatusBadRequest, "invalid word ID")
		return
	}

	word, err := h.service.GetByIDAdmin(r.Context(), id)
	if err != nil {
		h.Logger.Error("failed to get word", zap.Error(err))
		errStatus := http.StatusInternalServerError
		if err.Error() == "invalid word id" || err.Error() == "word not found" {
			errStatus = http.StatusNotFound
		}
		h.RespondError(w, errStatus, err.Error())
		return
	}

	h.RespondJSON(w, http.StatusOK, word)
}

// Create handles POST /admin/words
// @Summary Create a word
// @Description Create a new word
// @Tags admin
// @Accept json
// @Produce json
// @Param request body models.CreateWordRequest true "Word creation request"
// @Success 201 {object} map[string]string "Word created successfully"
// @Failure 400 {object} map[string]string "Invalid request body or word already exists"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /admin/words [post]
func (h *AdminWordsHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req models.CreateWordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.Logger.Error("failed to decode request body", zap.Error(err))
		h.RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	wordId, err := h.service.CreateWord(r.Context(), &req)
	if err != nil {
		h.Logger.Error("failed to create word", zap.Error(err))
		errStatus := http.StatusInternalServerError
		if strings.Contains(err.Error(), "already exists") || strings.Contains(err.Error(), "invalid") {
			errStatus = http.StatusBadRequest
		}
		h.RespondError(w, errStatus, err.Error())
		return
	}

	h.RespondJSON(w, http.StatusCreated, map[string]any{
		"message": "word created successfully",
		"wordId":  wordId,
	})
}

// Update handles PATCH /admin/words/{id}
// @Summary Update a word
// @Description Update word fields (partial update)
// @Tags admin
// @Accept json
// @Produce json
// @Param id path int true "Word ID"
// @Param request body models.UpdateWordRequest false "Update request"
// @Success 204 "No Content"
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 404 {object} map[string]string "Word not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /admin/words/{id} [patch]
func (h *AdminWordsHandler) Update(w http.ResponseWriter, r *http.Request) {
	// Parse word ID
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		h.Logger.Error("failed to parse word ID", zap.Error(err))
		h.RespondError(w, http.StatusBadRequest, "invalid word ID")
		return
	}

	var req models.UpdateWordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.Logger.Error("failed to decode request body", zap.Error(err))
		h.RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	err = h.service.UpdateWord(r.Context(), id, &req)
	if err != nil {
		h.Logger.Error("failed to update word", zap.Error(err))
		errStatus := http.StatusInternalServerError
		if err.Error() == "invalid word id" || err.Error() == "word not found" || err.Error() == "no fields to update" {
			if err.Error() == "word not found" {
				errStatus = http.StatusNotFound
			} else {
				errStatus = http.StatusBadRequest
			}
		}
		h.RespondError(w, errStatus, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Delete handles DELETE /admin/words/{id}
// @Summary Delete a word
// @Description Delete a word by ID
// @Tags admin
// @Accept json
// @Produce json
// @Param id path int true "Word ID"
// @Success 204 "No Content"
// @Failure 400 {object} map[string]string "Invalid word ID"
// @Failure 404 {object} map[string]string "Word not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /admin/words/{id} [delete]
func (h *AdminWordsHandler) Delete(w http.ResponseWriter, r *http.Request) {
	// Parse word ID
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		h.Logger.Error("failed to parse word ID", zap.Error(err))
		h.RespondError(w, http.StatusBadRequest, "invalid word ID")
		return
	}

	err = h.service.DeleteWord(r.Context(), id)
	if err != nil {
		h.Logger.Error("failed to delete word", zap.Error(err))
		errStatus := http.StatusInternalServerError
		if err.Error() == "invalid word id" || err.Error() == "word not found" {
			errStatus = http.StatusNotFound
		}
		h.RespondError(w, errStatus, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
