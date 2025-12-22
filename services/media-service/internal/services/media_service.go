package services

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/japanesestudent/media-service/internal/models"
	"github.com/japanesestudent/media-service/internal/storage"
)

// Storage defines the interface for file storage operations
type Storage interface {
	// Create creates a new file and returns a WriteCloser
	// The file path is generated based on id and mediaType
	Create(id, mediaType string) (io.WriteCloser, error)

	// Open opens a file for reading and returns a ReadCloser
	Open(id, mediaType string) (io.ReadCloser, error)

	// OpenFile opens a file and returns *os.File for use with http.ServeContent
	OpenFile(id, mediaType string) (*os.File, error)

	// Delete removes a file
	Delete(id, mediaType string) error
}

// MetadataRepository defines the interface for metadata data access
type MetadataRepository interface {
	Create(ctx context.Context, metadata *models.Metadata) error
	GetByID(ctx context.Context, id string) (*models.Metadata, error)
	DeleteByID(ctx context.Context, id string) error
}

// MediaService handles business logic for media operations
type MediaService struct {
	metadataRepo MetadataRepository
	storage      Storage
}

// NewMediaService creates a new media service
func NewMediaService(metadataRepo MetadataRepository, storage Storage) *MediaService {
	return &MediaService{
		metadataRepo: metadataRepo,
		storage:      storage,
	}
}

// GetMetadataByID retrieves metadata by ID
func (s *MediaService) GetMetadataByID(ctx context.Context, id string) (*models.Metadata, error) {
	return s.metadataRepo.GetByID(ctx, id)
}

// UploadFile handles file upload, creating both the file and metadata record
// Returns the generated download URL
// extension should include the leading dot (e.g., ".jpg")
func (s *MediaService) UploadFile(ctx context.Context, reader io.Reader, contentType, mediaType, baseURL, extension string) (string, error) {
	// Generate new filename with extension
	filename := storage.GenerateFileName(extension)

	// Create SizeWriter to track bytes
	sizeWriter := storage.NewSizeWriter()

	// Create TeeReader to count bytes while copying
	teeReader := io.TeeReader(reader, sizeWriter)

	// Create file in storage (convert string to MediaType)
	writeCloser, err := s.storage.Create(filename, mediaType)
	if err != nil {
		return "", fmt.Errorf("failed to create file: %w", err)
	}
	defer writeCloser.Close()

	// Copy data to file
	_, err = io.Copy(writeCloser, teeReader)
	if err != nil {
		// Cleanup: delete the file if copy fails
		s.storage.Delete(filename, mediaType)
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	// Build download URL
	downloadURL := fmt.Sprintf("%s/api/v4/media/%s/%s", baseURL, mediaType, filename)

	// Create metadata record
	metadata := &models.Metadata{
		ID:          filename,
		ContentType: contentType,
		Size:        sizeWriter.Size(),
		URL:         downloadURL,
		Type:        models.MediaType(mediaType),
	}

	if err := s.metadataRepo.Create(ctx, metadata); err != nil {
		// Cleanup: delete the file if metadata creation fails
		s.storage.Delete(filename, mediaType)
		return "", fmt.Errorf("failed to create metadata: %w", err)
	}

	// Return the metadata URL
	metadataURL := fmt.Sprintf("%s/api/v4/media/%s", baseURL, filename)
	return metadataURL, nil
}

// DeleteFile handles file deletion, removing both the file and metadata record
// Returns nil if successful, error otherwise
func (s *MediaService) DeleteFile(ctx context.Context, filename, mediaType string) error {
	// Delete file from storage
	err := s.storage.Delete(filename, mediaType)

	// Check if file didn't exist
	if err != nil && os.IsNotExist(err) {
		return fmt.Errorf("file not found: %w", err)
	}
	if err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	// Delete metadata record
	if err := s.metadataRepo.DeleteByID(ctx, filename); err != nil {
		return fmt.Errorf("failed to delete metadata: %w", err)
	}

	return nil
}

// GetFileReader returns a ReadCloser for the file
func (s *MediaService) GetFileReader(filename, mediaType string) (io.ReadCloser, error) {
	return s.storage.Open(filename, mediaType)
}

// GetFile returns an *os.File for use with http.ServeContent
func (s *MediaService) GetFile(filename, mediaType string) (*os.File, error) {
	return s.storage.OpenFile(filename, mediaType)
}

// InferExtensionFromContentType infers the extension from the content type
//
// "contentType" parameter is the content type to infer the extension from.
//
// Returns the inferred extension, or empty string if the extension cannot be inferred.
func (s *MediaService) InferExtensionFromContentType(contentType string) string {
	// Simple content type to extension mapping
	contentTypeMap := map[string]string{
		"image/jpeg":       ".jpg",
		"image/png":        ".png",
		"image/gif":        ".gif",
		"image/webp":       ".webp",
		"audio/mpeg":       ".mp3",
		"audio/wav":        ".wav",
		"audio/ogg":        ".ogg",
		"video/mp4":        ".mp4",
		"video/webm":       ".webm",
		"application/pdf":  ".pdf",
		"application/json": ".json",
		"text/plain":       ".txt",
		"text/html":        ".html",
	}

	if ext, ok := contentTypeMap[contentType]; ok {
		return ext
	}
	return ""
}

// IsValidMediaType checks if the media type is valid
//
// "mediaType" parameter is the media type to check.
//
// Returns true if the media type is valid, false otherwise.
func (s *MediaService) IsValidMediaType(mediaType string) bool {
	switch models.MediaType(mediaType) {
	case models.MediaTypeCharacter,
		models.MediaTypeWord,
		models.MediaTypeWordExample,
		models.MediaTypeLessonAudio,
		models.MediaTypeLessonVideo,
		models.MediaTypeLessonDoc,
		models.MediaTypeAvatar:
		return true
	default:
		return false
	}
}
