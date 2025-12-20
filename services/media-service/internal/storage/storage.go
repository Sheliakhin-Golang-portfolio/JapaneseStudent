package storage

import (
	"io"
	"os"
	"path/filepath"
	"strings"
)

// localStorage implements Storage interface using local filesystem
type localStorage struct {
	basePath string
}

// NewLocalStorage creates a new localStorage instance
func NewLocalStorage(basePath string) *localStorage {
	return &localStorage{
		basePath: basePath,
	}
}

// generatePath generates the full file path based on id and mediaType
// It converts underscores in mediaType to path separators
func (s *localStorage) generatePath(id, mediaType string) string {
	// Replace underscores with path separators based on operating system
	typePath := strings.ReplaceAll(mediaType, "_", string(filepath.Separator))

	// Combine base path, type path, and file id
	fullPath := filepath.Join(s.basePath, typePath, id)
	return fullPath
}

// Create creates a new file and returns a WriteCloser
func (s *localStorage) Create(id, mediaType string) (io.WriteCloser, error) {
	path := s.generatePath(id, mediaType)

	// Ensure the directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	// Create the file
	return os.Create(path)
}

// Open opens a file for reading and returns a ReadCloser
func (s *localStorage) Open(id, mediaType string) (io.ReadCloser, error) {
	path := s.generatePath(id, mediaType)
	return os.Open(path)
}

// OpenFile opens a file and returns *os.File
func (s *localStorage) OpenFile(id, mediaType string) (*os.File, error) {
	path := s.generatePath(id, mediaType)
	return os.Open(path)
}

// Delete removes a file
func (s *localStorage) Delete(id, mediaType string) error {
	path := s.generatePath(id, mediaType)
	return os.Remove(path)
}
