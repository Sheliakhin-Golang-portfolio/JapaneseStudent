package handlers

import (
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/japanesestudent/libs/handlers"
	"github.com/japanesestudent/media-service/internal/models"
	"go.uber.org/zap"
)

// MediaService defines the interface for media service operations
type MediaService interface {
	// Method GetMetadataByID retrieve a metadata by its ID using configured repository.
	//
	// If some error will occur during data retrieve, the error will be returned together with "nil" value.
	GetMetadataByID(ctx context.Context, id string) (*models.Metadata, error)
	// Method UploadFile uploads a file to the media service.
	//
	// "reader" parameter is the file to upload.
	// "contentType" parameter is the content type of the file.
	// "mediaType" parameter is the type of the media.
	// "baseURL" parameter is the base URL of the media service.
	// "extension" parameter is the extension of the file.
	//
	// If some error will occur during file upload, the error will be returned together with "nil" value.
	UploadFile(ctx context.Context, reader io.Reader, contentType, mediaType, baseURL, extension string) (string, error)
	// Method DeleteFile deletes a file from the media service.
	//
	// "filename" parameter is the name of the file to delete.
	// "mediaType" parameter is the type of the media.
	//
	// If some error will occur during file deletion, the error will be returned together with "nil" value.
	DeleteFile(ctx context.Context, filename, mediaType string) error
	// Method GetFileReader retrieves a reader for a file from the media service.
	//
	// "filename" parameter is the name of the file to retrieve.
	// "mediaType" parameter is the type of the media.
	//
	// If some error will occur during file retrieval, the error will be returned together with "nil" value.
	GetFileReader(filename, mediaType string) (io.ReadCloser, error)
	// Method GetFile retrieves a file from the media service.
	//
	// "filename" parameter is the name of the file to retrieve.
	// "mediaType" parameter is the type of the media.
	//
	// If some error will occur during file retrieval, the error will be returned together with "nil" value.
	GetFile(filename, mediaType string) (*os.File, error)
	// Method InferExtensionFromContentType infers the extension from the content type.
	//
	// "contentType" parameter is the content type to infer the extension from.
	//
	// If some error will occur during extension inference, the error will be returned together with "nil" value.
	InferExtensionFromContentType(contentType string) string
	// Method IsValidMediaType checks if the media type is valid.
	//
	// "mediaType" parameter is the media type to check.
	//
	// Returns true if the media type is valid, false otherwise.
	IsValidMediaType(mediaType string) bool
}

// MediaHandler handles media-related HTTP requests
type MediaHandler struct {
	handlers.BaseHandler
	mediaService MediaService
	baseURL      string
	authMw       func(http.Handler) http.Handler
}

// NewMediaHandler creates a new media handler
func NewMediaHandler(mediaService MediaService, logger *zap.Logger, baseURL string, authMw func(http.Handler) http.Handler) *MediaHandler {
	return &MediaHandler{
		BaseHandler:  handlers.BaseHandler{Logger: logger},
		mediaService: mediaService,
		baseURL:      baseURL,
		authMw:       authMw,
	}
}

// RegisterRoutes registers all media handler routes
func (h *MediaHandler) RegisterRoutes(r chi.Router) {
	r.Route("/media", func(r chi.Router) {
		r.Get("/{id}", h.GetMetadata)
		r.Get("/{mediaType}/{filename}", h.DownloadFile)
		r.Post("/{mediaType}", h.UploadFile)
		r.Delete("/{mediaType}/{filename}", h.DeleteFile)
	})
}

// GetMetadata handles GET /media/{id}
// @Summary Get file metadata
// @Description Retrieve metadata information for a file by its ID
// @Tags media
// @Accept json
// @Produce json
// @Param id path string true "File ID"
// @Success 200 {object} models.Metadata
// @Failure 404 {object} map[string]string "Metadata not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /media/{id} [get]
func (h *MediaHandler) GetMetadata(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	metadata, err := h.mediaService.GetMetadataByID(r.Context(), id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			h.Logger.Info("metadata not found", zap.String("id", id))
			h.RespondError(w, http.StatusNotFound, "metadata not found")
			return
		}
		h.Logger.Error("failed to get metadata", zap.Error(err), zap.String("id", id))
		h.RespondError(w, http.StatusInternalServerError, "failed to get metadata")
		return
	}

	h.RespondJSON(w, http.StatusOK, metadata)
}

// DownloadFile handles GET /media/{mediaType}/{filename}
// @Summary Download media file
// @Description Download a media file. Character files are public, others require authentication. Audio/video support range requests.
// @Tags media
// @Accept json
// @Produce application/octet-stream
// @Param mediaType path string true "Media type"
// @Param filename path string true "File name"
// @Param Range header string false "Range"
// @Success 200 "File content"
// @Success 206 "Partial file content (for range requests)"
// @Failure 401 {object} map[string]string "Authentication required"
// @Failure 404 {object} map[string]string "File not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /media/{mediaType}/{filename} [get]
func (h *MediaHandler) DownloadFile(w http.ResponseWriter, r *http.Request) {
	mediaTypeStr := chi.URLParam(r, "mediaType")
	filename := chi.URLParam(r, "filename")
	mediaType := models.MediaType(mediaTypeStr)

	// Apply auth middleware conditionally: character files are public, others (including avatar) require auth
	if mediaType != models.MediaTypeCharacter && h.authMw != nil {
		// Create a handler that will serve the file
		fileHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h.serveFile(w, r, mediaTypeStr, filename)
		})
		// Apply auth middleware
		h.authMw(fileHandler).ServeHTTP(w, r)
		return
	}

	// Character files are public, serve directly
	h.serveFile(w, r, mediaTypeStr, filename)
}

// serveFile serves the actual file content
func (h *MediaHandler) serveFile(w http.ResponseWriter, r *http.Request, mediaTypeStr, filename string) {
	mediaType := models.MediaType(mediaTypeStr)

	// Get metadata to determine content type
	metadata, err := h.mediaService.GetMetadataByID(r.Context(), filename)
	if err != nil {
		h.Logger.Error("failed to get metadata for download", zap.Error(err))
		h.RespondError(w, http.StatusNotFound, "file not found")
		return
	}

	// For lesson_audio and lesson_video, use range request support
	if mediaType == models.MediaTypeLessonAudio || mediaType == models.MediaTypeLessonVideo {
		file, err := h.mediaService.GetFile(filename, mediaTypeStr)
		if err != nil {
			if os.IsNotExist(err) {
				h.RespondError(w, http.StatusNotFound, "file not found")
				return
			}
			h.Logger.Error("failed to open file", zap.Error(err))
			h.RespondError(w, http.StatusInternalServerError, "failed to open file")
			return
		}
		defer file.Close()

		// Get file info for http.ServeContent
		fileInfo, err := file.Stat()
		if err != nil {
			h.Logger.Error("failed to get file info", zap.Error(err))
			h.RespondError(w, http.StatusInternalServerError, "failed to get file info")
			return
		}

		// Serve content with range support
		http.ServeContent(w, r, filename, fileInfo.ModTime(), file)
		return
	}

	// For other content types, serve full file
	reader, err := h.mediaService.GetFileReader(filename, mediaTypeStr)
	if err != nil {
		if os.IsNotExist(err) {
			h.RespondError(w, http.StatusNotFound, "file not found")
			return
		}
		h.Logger.Error("failed to open file", zap.Error(err))
		h.RespondError(w, http.StatusInternalServerError, "failed to open file")
		return
	}
	defer reader.Close()

	// Set content type
	w.Header().Set("Content-Type", metadata.ContentType)

	// Copy file to response
	_, err = io.Copy(w, reader)
	if err != nil {
		h.Logger.Error("failed to copy file to response", zap.Error(err))
		return
	}
}

// UploadFile handles POST /media/{mediaType}
// @Summary Upload media file
// @Description Upload a media file. Requires API key authentication.
// @Tags media
// @Accept multipart/form-data
// @Produce text/plain
// @Param mediaType path string true "Media type"
// @Param file formData file true "File to upload"
// @Param X-API-Key header string true "API Key"
// @Success 200 {string} string "Download URL"
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 401 {object} map[string]string "Authentication required"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /media/{mediaType} [post]
func (h *MediaHandler) UploadFile(w http.ResponseWriter, r *http.Request) {
	mediaTypeStr := chi.URLParam(r, "mediaType")

	// Validate media type
	if !h.mediaService.IsValidMediaType(mediaTypeStr) {
		h.RespondError(w, http.StatusBadRequest, "invalid media type")
		return
	}

	// Parse multipart form (limit to 50MB)
	err := r.ParseMultipartForm(50 << 20) // 50MB
	if err != nil {
		h.Logger.Error("failed to parse multipart form", zap.Error(err))
		h.RespondError(w, http.StatusBadRequest, "failed to parse request")
		return
	}

	// Get file from form
	file, fileHeader, err := r.FormFile("file")
	if err != nil {
		h.Logger.Error("failed to get file from form", zap.Error(err))
		h.RespondError(w, http.StatusBadRequest, "file is required")
		return
	}
	defer file.Close()

	// Get content type from form file header
	contentType := fileHeader.Header.Get("Content-Type")
	if contentType == "" {
		// Try to infer from request header
		contentType = r.Header.Get("Content-Type")
		// Remove multipart boundary if present
		if strings.HasPrefix(contentType, "multipart/") {
			contentType = ""
		}
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	// Extract extension from original filename
	ext := filepath.Ext(fileHeader.Filename)
	if ext == "" {
		// Try to infer from content type
		ext = h.mediaService.InferExtensionFromContentType(contentType)
	}

	// Upload file using service
	downloadURL, err := h.mediaService.UploadFile(r.Context(), file, contentType, mediaTypeStr, h.baseURL, ext)
	if err != nil {
		h.Logger.Error("failed to upload file", zap.Error(err))
		h.RespondError(w, http.StatusInternalServerError, "failed to upload file")
		return
	}

	// Return download URL as plain text (not JSON)
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(downloadURL))
}

// DeleteFile handles DELETE /media/{mediaType}/{filename}
// @Summary Delete media file
// @Description Delete a media file and its metadata. Requires API key authentication.
// @Tags media
// @Accept json
// @Produce json
// @Param mediaType path string true "Media type"
// @Param filename path string true "File name"
// @Param X-API-Key header string true "API Key"
// @Success 204 "File and metadata deleted successfully"
// @Failure 401 {object} map[string]string "Authentication required"
// @Failure 404 {object} map[string]string "File not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /media/{mediaType}/{filename} [delete]
func (h *MediaHandler) DeleteFile(w http.ResponseWriter, r *http.Request) {
	mediaTypeStr := chi.URLParam(r, "mediaType")
	filename := chi.URLParam(r, "filename")

	// Validate media type
	if !h.mediaService.IsValidMediaType(mediaTypeStr) {
		h.Logger.Error("invalid media type", zap.String("mediaType", mediaTypeStr))
		h.RespondError(w, http.StatusBadRequest, "invalid media type")
		return
	}

	// Delete file using service
	err := h.mediaService.DeleteFile(r.Context(), filename, mediaTypeStr)
	if err != nil {
		if strings.Contains(err.Error(), "not found") || os.IsNotExist(err) {
			h.Logger.Error("file not found", zap.String("filename", filename), zap.String("mediaType", mediaTypeStr))
			h.RespondError(w, http.StatusNotFound, "file not found")
			return
		}
		h.Logger.Error("failed to delete file", zap.Error(err))
		h.RespondError(w, http.StatusInternalServerError, "failed to delete file")
		return
	}

	// Return 204 No Content
	w.WriteHeader(http.StatusNoContent)
}
