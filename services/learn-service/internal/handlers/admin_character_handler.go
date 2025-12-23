package handlers

import (
	"context"
	"mime/multipart"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/japanesestudent/learn-service/internal/models"
	"github.com/japanesestudent/libs/handlers"
	"go.uber.org/zap"
)

// AdminCharactersService is the interface that wraps methods for admin character operations
type AdminCharactersService interface {
	// Method GetAll retrieve a list of all hiragana/katakana characters using configured repository.
	//
	// If some error will occur during data retrieve, the error will be returned together with "nil" value.
	GetAllForAdmin(ctx context.Context) ([]models.CharacterListItem, error)
	// Method GetByIDAdmin retrieve a character by its ID using configured repository.
	//
	// "id" parameter is used to identify the character.
	//
	// If some error will occur during data retrieve, the error will be returned together with "nil" value.
	GetByIDAdmin(ctx context.Context, id int) (*models.Character, error)
	// Method CreateCharacter creates a new character using configured repository.
	//
	// "character" parameter is used to create a new character.
	// "audioFile" and "audioFilename" are optional parameters for audio file upload.
	//
	// If some error will occur during data creation, the error will be returned together with 0 as character ID.
	CreateCharacter(ctx context.Context, character *models.CreateCharacterRequest, audioFile multipart.File, audioFilename string) (int, error)
	// Method UpdateCharacter updates a character using configured repository.
	//
	// "id" parameter is used to identify the character.
	// "character" parameter is used to update the character.
	// "audioFile" and "audioFilename" are optional parameters for audio file upload.
	//
	// If some error will occur during data update, the error will be returned together with "nil" value.
	UpdateCharacter(ctx context.Context, id int, character *models.UpdateCharacterRequest, audioFile multipart.File, audioFilename string) error
	// Method DeleteCharacter deletes a character using configured repository.
	//
	// "id" parameter is used to identify the character.
	//
	// If some error will occur during data deletion, the error will be returned together with "nil" value.
	DeleteCharacter(ctx context.Context, id int) error
}

// AdminCharactersHandler handles admin-related HTTP requests for characters
type AdminCharactersHandler struct {
	handlers.BaseHandler
	service AdminCharactersService
}

// NewAdminCharactersHandler creates a new admin character handler
func NewAdminCharactersHandler(svc AdminCharactersService, logger *zap.Logger) *AdminCharactersHandler {
	return &AdminCharactersHandler{
		service:     svc,
		BaseHandler: handlers.BaseHandler{Logger: logger},
	}
}

// RegisterRoutes registers all admin character handler routes
// Note: This assumes the router is already scoped to /api/v4
func (h *AdminCharactersHandler) RegisterRoutes(r chi.Router) {
	r.Route("/admin/characters", func(r chi.Router) {
		r.Get("/", h.GetAll)
		r.Get("/{id}", h.GetByID)
		r.Post("/", h.Create)
		r.Patch("/{id}", h.Update)
		r.Delete("/{id}", h.Delete)
	})
}

// GetAll handles GET /admin/characters
// @Summary Get list of characters
// @Description Get full list of characters ordered by ID
// @Tags admin
// @Accept json
// @Produce json
// @Success 200 {array} models.CharacterListItem "List of characters"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /admin/characters [get]
func (h *AdminCharactersHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	characters, err := h.service.GetAllForAdmin(r.Context())
	if err != nil {
		h.Logger.Error("failed to get characters list", zap.Error(err))
		h.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.RespondJSON(w, http.StatusOK, characters)
}

// GetByID handles GET /admin/characters/{id}
// @Summary Get character by ID
// @Description Get full information about a character by ID
// @Tags admin
// @Accept json
// @Produce json
// @Param id path int true "Character ID"
// @Success 200 {object} models.Character "Character information"
// @Failure 400 {object} map[string]string "Invalid character ID"
// @Failure 404 {object} map[string]string "Character not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /admin/characters/{id} [get]
func (h *AdminCharactersHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	// Parse character ID
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		h.Logger.Error("failed to parse character ID", zap.Error(err))
		h.RespondError(w, http.StatusBadRequest, "invalid character ID")
		return
	}

	character, err := h.service.GetByIDAdmin(r.Context(), id)
	if err != nil {
		h.Logger.Error("failed to get character", zap.Error(err))
		errStatus := http.StatusInternalServerError
		if err.Error() == "invalid character id" || err.Error() == "character not found" {
			errStatus = http.StatusNotFound
		}
		h.RespondError(w, errStatus, err.Error())
		return
	}

	h.RespondJSON(w, http.StatusOK, character)
}

// Create handles POST /admin/characters
// @Summary Create a character
// @Description Create a new character with optional audio file
// @Tags admin
// @Accept multipart/form-data
// @Produce json
// @Param consonant formData string true "Consonant"
// @Param vowel formData string true "Vowel"
// @Param englishReading formData string true "English reading"
// @Param russianReading formData string true "Russian reading"
// @Param katakana formData string true "Katakana character"
// @Param hiragana formData string true "Hiragana character"
// @Param audio formData file false "Audio file (optional)"
// @Success 201 {object} map[string]string "Character created successfully"
// @Failure 400 {object} map[string]string "Invalid request body or character already exists"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /admin/characters [post]
func (h *AdminCharactersHandler) Create(w http.ResponseWriter, r *http.Request) {
	// Parse multipart form (20MB max)
	const maxMemory = 20 << 20 // 20MB
	if err := r.ParseMultipartForm(maxMemory); err != nil {
		h.Logger.Error("failed to parse multipart form", zap.Error(err))
		h.RespondError(w, http.StatusBadRequest, "failed to parse multipart form")
		return
	}

	// Extract character data from form fields
	req := models.CreateCharacterRequest{
		Consonant:      r.FormValue("consonant"),
		Vowel:          r.FormValue("vowel"),
		EnglishReading: r.FormValue("englishReading"),
		RussianReading: r.FormValue("russianReading"),
		Katakana:       r.FormValue("katakana"),
		Hiragana:       r.FormValue("hiragana"),
	}

	// Extract audio file (optional)
	var audioFile multipart.File
	var audioFilename string
	file, header, err := r.FormFile("audio")
	if err == nil {
		audioFile = file
		audioFilename = header.Filename
		defer file.Close()
	} else if err != http.ErrMissingFile {
		h.Logger.Error("failed to get audio file from form", zap.Error(err))
		h.RespondError(w, http.StatusBadRequest, "failed to get audio file")
		return
	}

	characterID, err := h.service.CreateCharacter(r.Context(), &req, audioFile, audioFilename)
	if err != nil {
		h.Logger.Error("failed to create character", zap.Error(err))
		errStatus := http.StatusInternalServerError
		if err.Error() == "character with vowel" || err.Error() == "invalid" {
			errStatus = http.StatusBadRequest
		}
		h.RespondError(w, errStatus, err.Error())
		return
	}

	h.RespondJSON(w, http.StatusCreated, map[string]any{
		"message":     "character created successfully",
		"characterId": characterID,
	})
}

// Update handles PATCH /admin/characters/{id}
// @Summary Update a character
// @Description Update character fields (partial update) with optional audio file
// @Tags admin
// @Accept multipart/form-data
// @Produce json
// @Param id path int true "Character ID"
// @Param consonant formData string false "Consonant"
// @Param vowel formData string false "Vowel"
// @Param englishReading formData string false "English reading"
// @Param russianReading formData string false "Russian reading"
// @Param katakana formData string false "Katakana character"
// @Param hiragana formData string false "Hiragana character"
// @Param audio formData file false "Audio file (optional)"
// @Success 204 "No Content"
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 404 {object} map[string]string "Character not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /admin/characters/{id} [patch]
func (h *AdminCharactersHandler) Update(w http.ResponseWriter, r *http.Request) {
	// Parse character ID
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		h.Logger.Error("failed to parse character ID", zap.Error(err))
		h.RespondError(w, http.StatusBadRequest, "invalid character ID")
		return
	}

	// Parse multipart form (20MB max)
	const maxMemory = 20 << 20 // 20MB
	if err := r.ParseMultipartForm(maxMemory); err != nil {
		h.Logger.Error("failed to parse multipart form", zap.Error(err))
		h.RespondError(w, http.StatusBadRequest, "failed to parse multipart form")
		return
	}

	// Extract character data from form fields (all optional)
	var req models.UpdateCharacterRequest
	if consonant := r.FormValue("consonant"); consonant != "" {
		req.Consonant = consonant
	}
	if vowel := r.FormValue("vowel"); vowel != "" {
		req.Vowel = vowel
	}
	if englishReading := r.FormValue("englishReading"); englishReading != "" {
		req.EnglishReading = englishReading
	}
	if russianReading := r.FormValue("russianReading"); russianReading != "" {
		req.RussianReading = russianReading
	}
	if katakana := r.FormValue("katakana"); katakana != "" {
		req.Katakana = katakana
	}
	if hiragana := r.FormValue("hiragana"); hiragana != "" {
		req.Hiragana = hiragana
	}

	// Extract audio file (optional)
	var audioFile multipart.File
	var audioFilename string
	file, header, err := r.FormFile("audio")
	if err == nil {
		audioFile = file
		audioFilename = header.Filename
		defer file.Close()
	} else if err != http.ErrMissingFile {
		h.Logger.Error("failed to get audio file from form", zap.Error(err))
		h.RespondError(w, http.StatusBadRequest, "failed to get audio file")
		return
	}

	err = h.service.UpdateCharacter(r.Context(), id, &req, audioFile, audioFilename)
	if err != nil {
		h.Logger.Error("failed to update character", zap.Error(err))
		errStatus := http.StatusBadRequest
		if err.Error() == "invalid character id" || err.Error() == "character not found" {
			errStatus = http.StatusNotFound
		}
		h.RespondError(w, errStatus, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Delete handles DELETE /admin/characters/{id}
// @Summary Delete a character
// @Description Delete a character by ID
// @Tags admin
// @Accept json
// @Produce json
// @Param id path int true "Character ID"
// @Success 204 "No Content"
// @Failure 400 {object} map[string]string "Invalid character ID"
// @Failure 404 {object} map[string]string "Character not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /admin/characters/{id} [delete]
func (h *AdminCharactersHandler) Delete(w http.ResponseWriter, r *http.Request) {
	// Parse character ID
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		h.Logger.Error("failed to parse character ID", zap.Error(err))
		h.RespondError(w, http.StatusBadRequest, "invalid character ID")
		return
	}

	err = h.service.DeleteCharacter(r.Context(), id)
	if err != nil {
		h.Logger.Error("failed to delete character", zap.Error(err))
		errStatus := http.StatusInternalServerError
		if err.Error() == "invalid character id" || err.Error() == "character not found" {
			errStatus = http.StatusNotFound
		}
		h.RespondError(w, errStatus, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
