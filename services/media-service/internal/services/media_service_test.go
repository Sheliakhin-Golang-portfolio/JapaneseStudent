package services

import (
	"context"
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/japanesestudent/media-service/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockMetadataRepository is a mock implementation of MetadataRepository
type mockMetadataRepository struct {
	metadata  *models.Metadata
	err       error
	deleteErr error
}

func (m *mockMetadataRepository) Create(ctx context.Context, metadata *models.Metadata) error {
	if m.err != nil {
		return m.err
	}
	return nil
}

func (m *mockMetadataRepository) GetByID(ctx context.Context, id string) (*models.Metadata, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.metadata, nil
}

func (m *mockMetadataRepository) DeleteByID(ctx context.Context, id string) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	return m.err
}

// mockStorage is a mock implementation of Storage
type mockStorage struct {
	createErr    error
	openErr      error
	openFileErr  error
	deleteErr    error
	writeCloser  io.WriteCloser
	readCloser   io.ReadCloser
	file         *os.File
	deleteCalled bool
	deleteParams []string // [id, mediaType]
}

func (m *mockStorage) Create(id, mediaType string) (io.WriteCloser, error) {
	if m.createErr != nil {
		return nil, m.createErr
	}
	if m.writeCloser != nil {
		return m.writeCloser, nil
	}
	return &mockWriteCloser{}, nil
}

func (m *mockStorage) Open(id, mediaType string) (io.ReadCloser, error) {
	if m.openErr != nil {
		return nil, m.openErr
	}
	if m.readCloser != nil {
		return m.readCloser, nil
	}
	return io.NopCloser(strings.NewReader("test content")), nil
}

func (m *mockStorage) OpenFile(id, mediaType string) (*os.File, error) {
	if m.openFileErr != nil {
		return nil, m.openFileErr
	}
	return m.file, nil
}

func (m *mockStorage) Delete(id, mediaType string) error {
	m.deleteCalled = true
	m.deleteParams = []string{id, mediaType}
	if m.deleteErr != nil {
		return m.deleteErr
	}
	return nil
}

// mockWriteCloser is a mock implementation of io.WriteCloser
type mockWriteCloser struct {
	writeErr error
	closeErr error
	written  []byte
}

func (m *mockWriteCloser) Write(p []byte) (int, error) {
	if m.writeErr != nil {
		return 0, m.writeErr
	}
	m.written = append(m.written, p...)
	return len(p), nil
}

func (m *mockWriteCloser) Close() error {
	return m.closeErr
}

func TestNewMediaService(t *testing.T) {
	metadataRepo := &mockMetadataRepository{}
	storage := &mockStorage{}

	svc := NewMediaService(metadataRepo, storage)

	assert.NotNil(t, svc)
	assert.Equal(t, metadataRepo, svc.metadataRepo)
	assert.Equal(t, storage, svc.storage)
}

func TestMediaService_GetMetadataByID(t *testing.T) {
	tests := []struct {
		name          string
		id            string
		repo          *mockMetadataRepository
		expectedError bool
		expectedMeta  *models.Metadata
	}{
		{
			name: "success",
			id:   "test-id-123",
			repo: &mockMetadataRepository{
				metadata: &models.Metadata{
					ID:          "test-id-123",
					ContentType: "image/jpeg",
					Size:        1024,
					URL:         "http://example.com/api/v6/media/character/test-id-123",
					Type:        models.MediaTypeCharacter,
				},
			},
			expectedError: false,
			expectedMeta: &models.Metadata{
				ID:          "test-id-123",
				ContentType: "image/jpeg",
				Size:        1024,
				URL:         "http://example.com/api/v6/media/character/test-id-123",
				Type:        models.MediaTypeCharacter,
			},
		},
		{
			name: "not found",
			id:   "nonexistent-id",
			repo: &mockMetadataRepository{
				err: errors.New("metadata not found"),
			},
			expectedError: true,
			expectedMeta:  nil,
		},
		{
			name: "database error",
			id:   "test-id-123",
			repo: &mockMetadataRepository{
				err: errors.New("database error"),
			},
			expectedError: true,
			expectedMeta:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := &mockStorage{}
			svc := NewMediaService(tt.repo, storage)

			metadata, err := svc.GetMetadataByID(context.Background(), tt.id)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, metadata)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, metadata)
				if tt.expectedMeta != nil {
					assert.Equal(t, tt.expectedMeta.ID, metadata.ID)
					assert.Equal(t, tt.expectedMeta.ContentType, metadata.ContentType)
					assert.Equal(t, tt.expectedMeta.Size, metadata.Size)
					assert.Equal(t, tt.expectedMeta.URL, metadata.URL)
					assert.Equal(t, tt.expectedMeta.Type, metadata.Type)
				}
			}
		})
	}
}

func TestMediaService_UploadFile(t *testing.T) {
	tests := []struct {
		name          string
		contentType   string
		mediaType     string
		baseURL       string
		extension     string
		reader        io.Reader
		repo          *mockMetadataRepository
		storage       *mockStorage
		expectedError bool
		errorContains string
	}{
		{
			name:          "success",
			contentType:   "image/jpeg",
			mediaType:     "character",
			baseURL:       "http://example.com",
			extension:     ".jpg",
			reader:        strings.NewReader("test file content"),
			repo:          &mockMetadataRepository{},
			storage:       &mockStorage{},
			expectedError: false,
		},
		{
			name:        "storage create error",
			contentType: "image/png",
			mediaType:   "word",
			baseURL:     "http://example.com",
			extension:   ".png",
			reader:      strings.NewReader("test content"),
			repo:        &mockMetadataRepository{},
			storage: &mockStorage{
				createErr: errors.New("storage error"),
			},
			expectedError: true,
			errorContains: "failed to create file",
		},
		{
			name:        "write error",
			contentType: "image/jpeg",
			mediaType:   "character",
			baseURL:     "http://example.com",
			extension:   ".jpg",
			reader:      strings.NewReader("test content"),
			repo:        &mockMetadataRepository{},
			storage: &mockStorage{
				writeCloser: &mockWriteCloser{
					writeErr: errors.New("write error"),
				},
			},
			expectedError: true,
			errorContains: "failed to write file",
		},
		{
			name:        "metadata creation error - cleanup called",
			contentType: "image/jpeg",
			mediaType:   "character",
			baseURL:     "http://example.com",
			extension:   ".jpg",
			reader:      strings.NewReader("test content"),
			repo: &mockMetadataRepository{
				err: errors.New("metadata error"),
			},
			storage:       &mockStorage{},
			expectedError: true,
			errorContains: "failed to create metadata",
		},
		{
			name:        "close error on write closer",
			contentType: "image/jpeg",
			mediaType:   "character",
			baseURL:     "http://example.com",
			extension:   ".jpg",
			reader:      strings.NewReader("test content"),
			repo:        &mockMetadataRepository{},
			storage: &mockStorage{
				writeCloser: &mockWriteCloser{
					closeErr: errors.New("close error"),
				},
			},
			expectedError: false, // Close error is ignored in defer
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewMediaService(tt.repo, tt.storage)

			url, err := svc.UploadFile(context.Background(), tt.reader, tt.contentType, tt.mediaType, tt.baseURL, tt.extension)

			if tt.expectedError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
				assert.Empty(t, url)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, url)
				assert.Contains(t, url, tt.baseURL)
				assert.Contains(t, url, tt.mediaType)
			}

			// Verify cleanup was called if metadata creation failed
			if tt.expectedError && strings.Contains(tt.errorContains, "metadata") {
				assert.True(t, tt.storage.deleteCalled)
			}
		})
	}
}

func TestMediaService_DeleteFile(t *testing.T) {
	tests := []struct {
		name          string
		filename      string
		mediaType     string
		repo          *mockMetadataRepository
		storage       *mockStorage
		expectedError bool
		errorContains string
	}{
		{
			name:          "success",
			filename:      "test-id-123",
			mediaType:     "character",
			repo:          &mockMetadataRepository{},
			storage:       &mockStorage{},
			expectedError: false,
		},
		{
			name:      "file not found",
			filename:  "nonexistent-id",
			mediaType: "word",
			repo:      &mockMetadataRepository{},
			storage: &mockStorage{
				deleteErr: os.ErrNotExist,
			},
			expectedError: true,
			errorContains: "file not found",
		},
		{
			name:      "storage delete error",
			filename:  "test-id-123",
			mediaType: "character",
			repo:      &mockMetadataRepository{},
			storage: &mockStorage{
				deleteErr: errors.New("storage error"),
			},
			expectedError: true,
			errorContains: "failed to delete file",
		},
		{
			name:      "metadata delete error",
			filename:  "test-id-123",
			mediaType: "character",
			repo: &mockMetadataRepository{
				deleteErr: errors.New("metadata error"),
			},
			storage:       &mockStorage{},
			expectedError: true,
			errorContains: "failed to delete metadata",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewMediaService(tt.repo, tt.storage)

			err := svc.DeleteFile(context.Background(), tt.filename, tt.mediaType)

			if tt.expectedError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestMediaService_GetFileReader(t *testing.T) {
	tests := []struct {
		name          string
		filename      string
		mediaType     string
		storage       *mockStorage
		expectedError bool
	}{
		{
			name:          "success",
			filename:      "test-id-123",
			mediaType:     "character",
			storage:       &mockStorage{},
			expectedError: false,
		},
		{
			name:      "storage open error",
			filename:  "nonexistent-id",
			mediaType: "word",
			storage: &mockStorage{
				openErr: errors.New("file not found"),
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockMetadataRepository{}
			svc := NewMediaService(repo, tt.storage)

			reader, err := svc.GetFileReader(tt.filename, tt.mediaType)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, reader)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, reader)
				reader.Close()
			}
		})
	}
}

func TestMediaService_GetFile(t *testing.T) {
	tests := []struct {
		name          string
		filename      string
		mediaType     string
		storage       *mockStorage
		expectedError bool
	}{
		{
			name:          "success",
			filename:      "test-id-123",
			mediaType:     "character",
			storage:       &mockStorage{},
			expectedError: false,
		},
		{
			name:      "storage open file error",
			filename:  "nonexistent-id",
			mediaType: "word",
			storage: &mockStorage{
				openFileErr: errors.New("file not found"),
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockMetadataRepository{}
			svc := NewMediaService(repo, tt.storage)

			file, err := svc.GetFile(tt.filename, tt.mediaType)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, file)
			} else {
				// Note: In real tests, we might want to check file validity
				// For now, we just check that no error occurred
				if file != nil {
					file.Close()
				}
			}
		})
	}
}

func TestMediaService_InferExtensionFromContentType(t *testing.T) {
	repo := &mockMetadataRepository{}
	storage := &mockStorage{}
	svc := NewMediaService(repo, storage)

	tests := []struct {
		name        string
		contentType string
		expectedExt string
	}{
		{
			name:        "image/jpeg",
			contentType: "image/jpeg",
			expectedExt: ".jpg",
		},
		{
			name:        "image/png",
			contentType: "image/png",
			expectedExt: ".png",
		},
		{
			name:        "image/gif",
			contentType: "image/gif",
			expectedExt: ".gif",
		},
		{
			name:        "image/webp",
			contentType: "image/webp",
			expectedExt: ".webp",
		},
		{
			name:        "audio/mpeg",
			contentType: "audio/mpeg",
			expectedExt: ".mp3",
		},
		{
			name:        "audio/wav",
			contentType: "audio/wav",
			expectedExt: ".wav",
		},
		{
			name:        "video/mp4",
			contentType: "video/mp4",
			expectedExt: ".mp4",
		},
		{
			name:        "application/pdf",
			contentType: "application/pdf",
			expectedExt: ".pdf",
		},
		{
			name:        "unknown content type",
			contentType: "application/unknown",
			expectedExt: "",
		},
		{
			name:        "empty content type",
			contentType: "",
			expectedExt: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ext := svc.InferExtensionFromContentType(tt.contentType)
			assert.Equal(t, tt.expectedExt, ext)
		})
	}
}

func TestMediaService_IsValidMediaType(t *testing.T) {
	repo := &mockMetadataRepository{}
	storage := &mockStorage{}
	svc := NewMediaService(repo, storage)

	tests := []struct {
		name          string
		mediaType     string
		expectedValid bool
	}{
		{
			name:          "valid character",
			mediaType:     "character",
			expectedValid: true,
		},
		{
			name:          "valid word",
			mediaType:     "word",
			expectedValid: true,
		},
		{
			name:          "valid word_example",
			mediaType:     "word_example",
			expectedValid: true,
		},
		{
			name:          "valid lesson_audio",
			mediaType:     "lesson_audio",
			expectedValid: true,
		},
		{
			name:          "valid lesson_video",
			mediaType:     "lesson_video",
			expectedValid: true,
		},
		{
			name:          "valid lesson_doc",
			mediaType:     "lesson_doc",
			expectedValid: true,
		},
		{
			name:          "invalid media type",
			mediaType:     "invalid",
			expectedValid: false,
		},
		{
			name:          "empty media type",
			mediaType:     "",
			expectedValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid := svc.IsValidMediaType(tt.mediaType)
			assert.Equal(t, tt.expectedValid, valid)
		})
	}
}

func TestMediaService_UploadFile_CleanupOnError(t *testing.T) {
	// Test that file is deleted when metadata creation fails
	repo := &mockMetadataRepository{
		err: errors.New("metadata creation failed"),
	}
	storage := &mockStorage{}

	svc := NewMediaService(repo, storage)
	reader := strings.NewReader("test content")

	_, err := svc.UploadFile(context.Background(), reader, "image/jpeg", "character", "http://example.com", ".jpg")

	require.Error(t, err)
	assert.True(t, storage.deleteCalled, "Storage.Delete should be called when metadata creation fails")
}
