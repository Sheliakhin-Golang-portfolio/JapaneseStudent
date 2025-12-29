package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/japanesestudent/learn-service/internal/models"
)

type tutorMediaRepository struct {
	db *sql.DB
}

// NewTutorMediaRepository creates a new tutor media repository
func NewTutorMediaRepository(db *sql.DB) *tutorMediaRepository {
	return &tutorMediaRepository{
		db: db,
	}
}

// GetByID retrieves tutor media by its ID
func (r *tutorMediaRepository) GetByID(ctx context.Context, id int) (*models.TutorMedia, error) {
	query := `
		SELECT id, tutor_id, slug, media_type, url
		FROM tutor_media
		WHERE id = ?
		LIMIT 1
	`

	var media models.TutorMedia
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&media.ID,
		&media.TutorID,
		&media.Slug,
		&media.MediaType,
		&media.URL,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("tutor media not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get tutor media by id: %w", err)
	}

	return &media, nil
}

// GetByTutorID retrieves tutor media by tutor ID with optional media type filter and pagination
func (r *tutorMediaRepository) GetByTutorID(ctx context.Context, tutorID *int, mediaType *models.MediaType, page, count int) ([]models.TutorMediaResponse, error) {
	var whereClauses []string
	var args []any

	if tutorID != nil {
		whereClauses = append(whereClauses, "tutor_id = ?")
		args = append(args, *tutorID)
	}

	if mediaType != nil {
		whereClauses = append(whereClauses, "media_type = ?")
		args = append(args, *mediaType)
	}

	whereClause := ""
	if len(whereClauses) > 0 {
		whereClause = "WHERE " + strings.Join(whereClauses, " AND ")
	}

	// Calculate offset
	offset := (page - 1) * count

	query := fmt.Sprintf(`
		SELECT id, slug, media_type, url
		FROM tutor_media
		%s
		ORDER BY id
		LIMIT ? OFFSET ?
	`, whereClause)

	args = append(args, count, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query tutor media: %w", err)
	}
	defer rows.Close()

	var mediaList []models.TutorMediaResponse
	for rows.Next() {
		var media models.TutorMediaResponse
		err := rows.Scan(
			&media.ID,
			&media.Slug,
			&media.MediaType,
			&media.URL,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan tutor media: %w", err)
		}
		mediaList = append(mediaList, media)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return mediaList, nil
}

// ExistsBySlug checks if tutor media with the given slug exists
func (r *tutorMediaRepository) ExistsBySlug(ctx context.Context, slug string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM tutor_media WHERE slug = ?)`

	var exists bool
	err := r.db.QueryRowContext(ctx, query, slug).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check tutor media existence: %w", err)
	}

	return exists, nil
}

// Create creates a new tutor media record
func (r *tutorMediaRepository) Create(ctx context.Context, media *models.TutorMedia) error {
	query := `
		INSERT INTO tutor_media (tutor_id, slug, media_type, url)
		VALUES (?, ?, ?, ?)
	`

	result, err := r.db.ExecContext(ctx, query,
		media.TutorID,
		media.Slug,
		media.MediaType,
		media.URL,
	)
	if err != nil {
		return fmt.Errorf("failed to create tutor media: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}

	media.ID = int(id)
	return nil
}

// Delete deletes tutor media by ID
func (r *tutorMediaRepository) Delete(ctx context.Context, id int) error {
	query := `DELETE FROM tutor_media WHERE id = ?`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete tutor media: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("tutor media not found")
	}

	return nil
}
