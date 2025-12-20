package repositories

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/japanesestudent/media-service/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupMetadataTestRepository creates a metadata repository with a mock database
func setupMetadataTestRepository(t *testing.T) (*metadataRepository, sqlmock.Sqlmock, func()) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)

	repo := NewMetadataRepository(db)

	cleanup := func() {
		db.Close()
	}

	return repo, mock, cleanup
}

func TestNewMetadataRepository(t *testing.T) {
	db := &sql.DB{}

	repo := NewMetadataRepository(db)

	assert.NotNil(t, repo)
	assert.Equal(t, db, repo.db)
}

func TestMetadataRepository_Create(t *testing.T) {
	tests := []struct {
		name          string
		metadata      *models.Metadata
		setupMock     func(sqlmock.Sqlmock)
		expectedError bool
	}{
		{
			name: "success",
			metadata: &models.Metadata{
				ID:          "test-id-123",
				ContentType: "image/jpeg",
				Size:        1024,
				URL:         "http://example.com/api/v4/media/character/test-id-123",
				Type:        models.MediaTypeCharacter,
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`INSERT INTO metadata`).
					WithArgs("test-id-123", "image/jpeg", int64(1024), "http://example.com/api/v4/media/character/test-id-123", models.MediaTypeCharacter).
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
			expectedError: false,
		},
		{
			name: "database error on insert",
			metadata: &models.Metadata{
				ID:          "test-id-123",
				ContentType: "image/jpeg",
				Size:        1024,
				URL:         "http://example.com/api/v4/media/character/test-id-123",
				Type:        models.MediaTypeCharacter,
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`INSERT INTO metadata`).
					WithArgs("test-id-123", "image/jpeg", int64(1024), "http://example.com/api/v4/media/character/test-id-123", models.MediaTypeCharacter).
					WillReturnError(errors.New("database error"))
			},
			expectedError: true,
		},
		{
			name: "duplicate key error",
			metadata: &models.Metadata{
				ID:          "duplicate-id",
				ContentType: "image/png",
				Size:        2048,
				URL:         "http://example.com/api/v4/media/word/duplicate-id",
				Type:        models.MediaTypeWord,
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`INSERT INTO metadata`).
					WithArgs("duplicate-id", "image/png", int64(2048), "http://example.com/api/v4/media/word/duplicate-id", models.MediaTypeWord).
					WillReturnError(errors.New("Error 1062: Duplicate entry 'duplicate-id' for key 'PRIMARY'"))
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock, cleanup := setupMetadataTestRepository(t)
			defer cleanup()

			tt.setupMock(mock)

			err := repo.Create(context.Background(), tt.metadata)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestMetadataRepository_GetByID(t *testing.T) {
	tests := []struct {
		name          string
		id            string
		setupMock     func(sqlmock.Sqlmock)
		expectedError bool
		expectedMeta  *models.Metadata
	}{
		{
			name: "success",
			id:   "test-id-123",
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"content_type", "size", "url", "type"}).
					AddRow("image/jpeg", int64(1024), "http://example.com/api/v4/media/character/test-id-123", models.MediaTypeCharacter)
				mock.ExpectQuery(`SELECT content_type, size, url, type FROM metadata WHERE id = \? LIMIT 1`).
					WithArgs("test-id-123").
					WillReturnRows(rows)
			},
			expectedError: false,
			expectedMeta: &models.Metadata{
				ID:          "test-id-123",
				ContentType: "image/jpeg",
				Size:        1024,
				URL:         "http://example.com/api/v4/media/character/test-id-123",
				Type:        models.MediaTypeCharacter,
			},
		},
		{
			name: "not found",
			id:   "nonexistent-id",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT content_type, size, url, type FROM metadata WHERE id = \? LIMIT 1`).
					WithArgs("nonexistent-id").
					WillReturnError(sql.ErrNoRows)
			},
			expectedError: true,
			expectedMeta:  nil,
		},
		{
			name: "database error",
			id:   "test-id-123",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT content_type, size, url, type FROM metadata WHERE id = \? LIMIT 1`).
					WithArgs("test-id-123").
					WillReturnError(errors.New("database error"))
			},
			expectedError: true,
			expectedMeta:  nil,
		},
		{
			name: "scan error - invalid data types",
			id:   "test-id-123",
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"content_type", "size", "url", "type"}).
					AddRow("image/jpeg", "invalid", "http://example.com/api/v4/media/character/test-id-123", models.MediaTypeCharacter)
				mock.ExpectQuery(`SELECT content_type, size, url, type FROM metadata WHERE id = \? LIMIT 1`).
					WithArgs("test-id-123").
					WillReturnRows(rows)
			},
			expectedError: true,
			expectedMeta:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock, cleanup := setupMetadataTestRepository(t)
			defer cleanup()

			tt.setupMock(mock)

			metadata, err := repo.GetByID(context.Background(), tt.id)

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

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestMetadataRepository_DeleteByID(t *testing.T) {
	tests := []struct {
		name          string
		id            string
		setupMock     func(sqlmock.Sqlmock)
		expectedError bool
	}{
		{
			name: "success",
			id:   "test-id-123",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`DELETE FROM metadata WHERE id = \?`).
					WithArgs("test-id-123").
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
			expectedError: false,
		},
		{
			name: "metadata not found",
			id:   "nonexistent-id",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`DELETE FROM metadata WHERE id = \?`).
					WithArgs("nonexistent-id").
					WillReturnResult(sqlmock.NewResult(0, 0))
			},
			expectedError: true,
		},
		{
			name: "database error",
			id:   "test-id-123",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`DELETE FROM metadata WHERE id = \?`).
					WithArgs("test-id-123").
					WillReturnError(errors.New("database error"))
			},
			expectedError: true,
		},
		{
			name: "error getting rows affected",
			id:   "test-id-123",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`DELETE FROM metadata WHERE id = \?`).
					WithArgs("test-id-123").
					WillReturnResult(sqlmock.NewErrorResult(errors.New("rows affected error")))
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock, cleanup := setupMetadataTestRepository(t)
			defer cleanup()

			tt.setupMock(mock)

			err := repo.DeleteByID(context.Background(), tt.id)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

