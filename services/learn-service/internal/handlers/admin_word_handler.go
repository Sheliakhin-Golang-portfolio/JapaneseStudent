package handlers

import (
	"context"
	"mime/multipart"
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
	// "wordAudioFile" and "wordAudioFilename" are optional parameters for word audio file upload.
	// "wordExampleAudioFile" and "wordExampleAudioFilename" are optional parameters for word example audio file upload.
	//
	// If some error will occur during data creation, the error will be returned together with 0 as word ID.
	CreateWord(ctx context.Context, word *models.CreateWordRequest, wordAudioFile multipart.File, wordAudioFilename string, wordExampleAudioFile multipart.File, wordExampleAudioFilename string) (int, error)
	// Method UpdateWord updates a word using configured repository.
	//
	// "id" parameter is used to identify the word.
	// "word" parameter is used to update the word.
	// "wordAudioFile" and "wordAudioFilename" are optional parameters for word audio file upload.
	// "wordExampleAudioFile" and "wordExampleAudioFilename" are optional parameters for word example audio file upload.
	//
	// If some error will occur during data update, the error will be returned.
	UpdateWord(ctx context.Context, id int, word *models.UpdateWordRequest, wordAudioFile multipart.File, wordAudioFilename string, wordExampleAudioFile multipart.File, wordExampleAudioFilename string) error
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
// @Description Create a new word with optional audio files
// @Tags admin
// @Accept multipart/form-data
// @Produce json
// @Param word formData string true "Word"
// @Param phoneticClues formData string true "Phonetic clues"
// @Param russianTranslation formData string true "Russian translation"
// @Param englishTranslation formData string true "English translation"
// @Param germanTranslation formData string true "German translation"
// @Param example formData string true "Example"
// @Param exampleRussianTranslation formData string true "Example Russian translation"
// @Param exampleEnglishTranslation formData string true "Example English translation"
// @Param exampleGermanTranslation formData string true "Example German translation"
// @Param easyPeriod formData int true "Easy period"
// @Param normalPeriod formData int true "Normal period"
// @Param hardPeriod formData int true "Hard period"
// @Param extraHardPeriod formData int true "Extra hard period"
// @Param wordAudio formData file false "Word audio file (optional)"
// @Param wordExampleAudio formData file false "Word example audio file (optional)"
// @Success 201 {object} map[string]string "Word created successfully"
// @Failure 400 {object} map[string]string "Invalid request body or word already exists"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /admin/words [post]
func (h *AdminWordsHandler) Create(w http.ResponseWriter, r *http.Request) {
	// Parse multipart form (30MB max)
	const maxMemory = 30 << 20 // 30MB
	if err := r.ParseMultipartForm(maxMemory); err != nil {
		h.Logger.Error("failed to parse multipart form", zap.Error(err))
		h.RespondError(w, http.StatusBadRequest, "failed to parse multipart form")
		return
	}

	// Extract word data from form fields
	req := &models.CreateWordRequest{
		Word:                      r.FormValue("word"),
		PhoneticClues:             r.FormValue("phoneticClues"),
		RussianTranslation:        r.FormValue("russianTranslation"),
		EnglishTranslation:        r.FormValue("englishTranslation"),
		GermanTranslation:         r.FormValue("germanTranslation"),
		Example:                   r.FormValue("example"),
		ExampleRussianTranslation: r.FormValue("exampleRussianTranslation"),
		ExampleEnglishTranslation: r.FormValue("exampleEnglishTranslation"),
		ExampleGermanTranslation:  r.FormValue("exampleGermanTranslation"),
	}

	if easyPeriodStr := r.FormValue("easyPeriod"); easyPeriodStr != "" {
		if p, err := strconv.Atoi(easyPeriodStr); err == nil {
			req.EasyPeriod = p
		}
	}
	if normalPeriodStr := r.FormValue("normalPeriod"); normalPeriodStr != "" {
		if p, err := strconv.Atoi(normalPeriodStr); err == nil {
			req.NormalPeriod = p
		}
	}
	if hardPeriodStr := r.FormValue("hardPeriod"); hardPeriodStr != "" {
		if p, err := strconv.Atoi(hardPeriodStr); err == nil {
			req.HardPeriod = p
		}
	}
	if extraHardPeriodStr := r.FormValue("extraHardPeriod"); extraHardPeriodStr != "" {
		if p, err := strconv.Atoi(extraHardPeriodStr); err == nil {
			req.ExtraHardPeriod = p
		}
	}

	// Extract word audio file (optional)
	var wordAudioFile multipart.File
	var wordAudioFilename string
	wordFile, header, err := r.FormFile("wordAudio")
	if err == nil {
		wordAudioFile = wordFile
		wordAudioFilename = header.Filename
		defer wordFile.Close()
	} else if err != http.ErrMissingFile {
		h.Logger.Error("failed to get word audio file from form", zap.Error(err))
		h.RespondError(w, http.StatusBadRequest, "failed to get word audio file")
		return
	}

	// Extract word example audio file (optional)
	var wordExampleAudioFile multipart.File
	var wordExampleAudioFilename string
	wordExampleFile, header, err := r.FormFile("wordExampleAudio")
	if err == nil {
		wordExampleAudioFile = wordExampleFile
		wordExampleAudioFilename = header.Filename
		defer wordExampleFile.Close()
	} else if err != http.ErrMissingFile {
		h.Logger.Error("failed to get word example audio file from form", zap.Error(err))
		h.RespondError(w, http.StatusBadRequest, "failed to get word example audio file")
		return
	}

	wordId, err := h.service.CreateWord(r.Context(), req, wordAudioFile, wordAudioFilename, wordExampleAudioFile, wordExampleAudioFilename)
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
// @Description Update word fields (partial update) with optional audio files
// @Tags admin
// @Accept multipart/form-data
// @Produce json
// @Param id path int true "Word ID"
// @Param word formData string false "Word"
// @Param phoneticClues formData string false "Phonetic clues"
// @Param russianTranslation formData string false "Russian translation"
// @Param englishTranslation formData string false "English translation"
// @Param germanTranslation formData string false "German translation"
// @Param example formData string false "Example"
// @Param exampleRussianTranslation formData string false "Example Russian translation"
// @Param exampleEnglishTranslation formData string false "Example English translation"
// @Param exampleGermanTranslation formData string false "Example German translation"
// @Param easyPeriod formData int false "Easy period"
// @Param normalPeriod formData int false "Normal period"
// @Param hardPeriod formData int false "Hard period"
// @Param extraHardPeriod formData int false "Extra hard period"
// @Param wordAudio formData file false "Word audio file (optional)"
// @Param wordExampleAudio formData file false "Word example audio file (optional)"
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

	// Parse multipart form (30MB max)
	const maxMemory = 30 << 20 // 30MB
	if err := r.ParseMultipartForm(maxMemory); err != nil {
		h.Logger.Error("failed to parse multipart form", zap.Error(err))
		h.RespondError(w, http.StatusBadRequest, "failed to parse multipart form")
		return
	}

	// Extract word data from form fields (all optional)
	req := &models.UpdateWordRequest{
		Word:                      r.FormValue("word"),
		PhoneticClues:             r.FormValue("phoneticClues"),
		RussianTranslation:        r.FormValue("russianTranslation"),
		EnglishTranslation:        r.FormValue("englishTranslation"),
		GermanTranslation:         r.FormValue("germanTranslation"),
		Example:                   r.FormValue("example"),
		ExampleRussianTranslation: r.FormValue("exampleRussianTranslation"),
		ExampleEnglishTranslation: r.FormValue("exampleEnglishTranslation"),
		ExampleGermanTranslation:  r.FormValue("exampleGermanTranslation"),
	}
	if easyPeriodStr := r.FormValue("easyPeriod"); easyPeriodStr != "" {
		if p, err := strconv.Atoi(easyPeriodStr); err == nil {
			req.EasyPeriod = &p
		}
	}
	if normalPeriodStr := r.FormValue("normalPeriod"); normalPeriodStr != "" {
		if p, err := strconv.Atoi(normalPeriodStr); err == nil {
			req.NormalPeriod = &p
		}
	}
	if hardPeriodStr := r.FormValue("hardPeriod"); hardPeriodStr != "" {
		if p, err := strconv.Atoi(hardPeriodStr); err == nil {
			req.HardPeriod = &p
		}
	}
	if extraHardPeriodStr := r.FormValue("extraHardPeriod"); extraHardPeriodStr != "" {
		if p, err := strconv.Atoi(extraHardPeriodStr); err == nil {
			req.ExtraHardPeriod = &p
		}
	}

	// Extract word audio file (optional)
	var wordAudioFile multipart.File
	var wordAudioFilename string
	wordFile, header, err := r.FormFile("wordAudio")
	if err == nil {
		wordAudioFile = wordFile
		wordAudioFilename = header.Filename
		defer wordFile.Close()
	} else if err != http.ErrMissingFile {
		h.Logger.Error("failed to get word audio file from form", zap.Error(err))
		h.RespondError(w, http.StatusBadRequest, "failed to get word audio file")
		return
	}

	// Extract word example audio file (optional)
	var wordExampleAudioFile multipart.File
	var wordExampleAudioFilename string
	wordExampleFile, header, err := r.FormFile("wordExampleAudio")
	if err == nil {
		wordExampleAudioFile = wordExampleFile
		wordExampleAudioFilename = header.Filename
		defer wordExampleFile.Close()
	} else if err != http.ErrMissingFile {
		h.Logger.Error("failed to get word example audio file from form", zap.Error(err))
		h.RespondError(w, http.StatusBadRequest, "failed to get word example audio file")
		return
	}

	err = h.service.UpdateWord(r.Context(), id, req, wordAudioFile, wordAudioFilename, wordExampleAudioFile, wordExampleAudioFilename)
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
