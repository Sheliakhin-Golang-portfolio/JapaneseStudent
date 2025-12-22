package repositories

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/japanesestudent/media-service/internal/models"
)

// metadataRepository implements metadata repository operations
type metadataRepository struct {
	db *sql.DB
}

// NewMetadataRepository creates a new metadata repository
func NewMetadataRepository(db *sql.DB) *metadataRepository {
	return &metadataRepository{
		db: db,
	}
}

// Create inserts a new metadata record into the database
func (r *metadataRepository) Create(ctx context.Context, metadata *models.Metadata) error {
	query := `
		INSERT INTO metadata (id, content_type, size, url, type)
		VALUES (?, ?, ?, ?, ?)
	`

	_, err := r.db.ExecContext(ctx, query,
		metadata.ID,
		metadata.ContentType,
		metadata.Size,
		metadata.URL,
		metadata.Type,
	)
	if err != nil {
		return fmt.Errorf("failed to create metadata: %w", err)
	}

	return nil
}

// GetByID retrieves metadata by ID
func (r *metadataRepository) GetByID(ctx context.Context, id string) (*models.Metadata, error) {
	query := `
		SELECT content_type, size, url, type
		FROM metadata
		WHERE id = ?
		LIMIT 1
	`

	metadata := &models.Metadata{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&metadata.ContentType,
		&metadata.Size,
		&metadata.URL,
		&metadata.Type,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("metadata not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get metadata by id: %w", err)
	}

	metadata.ID = id
	return metadata, nil
}

// DeleteByID deletes metadata by ID
func (r *metadataRepository) DeleteByID(ctx context.Context, id string) error {
	query := `DELETE FROM metadata WHERE id = ?`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete metadata: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("metadata not found")
	}

	return nil
}
