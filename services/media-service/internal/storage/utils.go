package storage

import (
	"github.com/google/uuid"
)

// GenerateFileName generates a new file name based on the file extension
// It creates a UUID-based filename with the provided extension
func GenerateFileName(extension string) string {
	newUUID := uuid.New().String()
	// Ensure extension starts with a dot if it doesn't already
	if extension != "" && extension[0] != '.' {
		return newUUID + "." + extension
	}
	return newUUID + extension
}

// SizeWriter wraps a writer and tracks the total number of bytes written
type sizeWriter struct {
	size int64
}

// Write implements io.Writer interface
// It tracks the size of data written and returns the length and nil error
func (sw *sizeWriter) Write(p []byte) (int, error) {
	n := len(p)
	sw.size += int64(n)
	return n, nil
}

// Size returns the total number of bytes written
func (sw *sizeWriter) Size() int64 {
	return sw.size
}

// NewSizeWriter creates a new SizeWriter instance
func NewSizeWriter() *sizeWriter {
	return &sizeWriter{
		size: 0,
	}
}
