package repositories

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Sheliakhin-Golang-portfolio/JapaneseStudent/learn-service/internal/models"
)

type lessonBlockRepository struct {
	db *sql.DB
}

// NewLessonBlockRepository creates a new lesson block repository
func NewLessonBlockRepository(db *sql.DB) *lessonBlockRepository {
	return &lessonBlockRepository{
		db: db,
	}
}

// GetByID retrieves a lesson block by its ID
func (r *lessonBlockRepository) GetByID(ctx context.Context, id int) (*models.LessonBlock, error) {
	query := `
		SELECT id, lesson_id, block_type, block_order, block_data
		FROM lesson_blocks
		WHERE id = ?
		LIMIT 1
	`

	var block models.LessonBlock
	var blockDataJSON string
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&block.ID,
		&block.LessonID,
		&block.BlockType,
		&block.BlockOrder,
		&blockDataJSON,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("lesson block not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get lesson block by id: %w", err)
	}

	block.BlockData = json.RawMessage(blockDataJSON)
	return &block, nil
}

// GetByLessonID retrieves all blocks for a lesson, sorted by order
func (r *lessonBlockRepository) GetByLessonID(ctx context.Context, lessonID int) ([]models.LessonBlockResponse, error) {
	query := `
		SELECT id, block_type, block_order, block_data
		FROM lesson_blocks
		WHERE lesson_id = ?
		ORDER BY block_order
	`

	rows, err := r.db.QueryContext(ctx, query, lessonID)
	if err != nil {
		return nil, fmt.Errorf("failed to query lesson blocks: %w", err)
	}
	defer rows.Close()

	var blocks []models.LessonBlockResponse
	for rows.Next() {
		var block models.LessonBlockResponse
		var blockDataJSON string
		err := rows.Scan(
			&block.ID,
			&block.BlockType,
			&block.BlockOrder,
			&blockDataJSON,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan lesson block: %w", err)
		}
		block.BlockData = json.RawMessage(blockDataJSON)
		blocks = append(blocks, block)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return blocks, nil
}

// ExistsByOrderInLesson checks if a block with the given order exists in the lesson
func (r *lessonBlockRepository) ExistsByOrderInLesson(ctx context.Context, lessonID int, order int) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM lesson_blocks WHERE lesson_id = ? AND block_order = ?)`

	var exists bool
	err := r.db.QueryRowContext(ctx, query, lessonID, order).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check block order existence: %w", err)
	}

	return exists, nil
}

// IncrementOrderForBlocks increments order for all blocks in a lesson with order >= given order
func (r *lessonBlockRepository) IncrementOrderForBlocks(ctx context.Context, lessonID, order int) error {
	query := `
		UPDATE lesson_blocks
		SET block_order = block_order + 1
		WHERE lesson_id = ? AND block_order >= ?
	`

	_, err := r.db.ExecContext(ctx, query, lessonID, order)
	if err != nil {
		return fmt.Errorf("failed to increment block order: %w", err)
	}

	return nil
}

// Create creates a new lesson block
func (r *lessonBlockRepository) Create(ctx context.Context, block *models.LessonBlock) error {
	blockDataJSON, err := json.Marshal(block.BlockData)
	if err != nil {
		return fmt.Errorf("failed to marshal block data: %w", err)
	}

	query := `
		INSERT INTO lesson_blocks (lesson_id, block_type, block_order, block_data)
		VALUES (?, ?, ?, ?)
	`

	result, err := r.db.ExecContext(ctx, query,
		block.LessonID,
		block.BlockType,
		block.BlockOrder,
		string(blockDataJSON),
	)
	if err != nil {
		return fmt.Errorf("failed to create lesson block: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}

	block.ID = int(id)
	return nil
}

// Update updates a lesson block (partial update)
func (r *lessonBlockRepository) Update(ctx context.Context, block *models.LessonBlock) error {
	var setParts []string
	var args []any

	if block.LessonID != 0 {
		setParts = append(setParts, "lesson_id = ?")
		args = append(args, block.LessonID)
	}
	if block.BlockType != "" {
		setParts = append(setParts, "block_type = ?")
		args = append(args, block.BlockType)
	}
	if block.BlockOrder != 0 {
		setParts = append(setParts, "block_order = ?")
		args = append(args, block.BlockOrder)
	}
	if block.BlockData != nil {
		blockDataJSON, err := json.Marshal(block.BlockData)
		if err != nil {
			return fmt.Errorf("failed to marshal block data: %w", err)
		}
		setParts = append(setParts, "block_data = ?")
		args = append(args, string(blockDataJSON))
	}

	if len(setParts) == 0 {
		return fmt.Errorf("no fields to update")
	}

	query := fmt.Sprintf(`
		UPDATE lesson_blocks
		SET %s
		WHERE id = ?
	`, strings.Join(setParts, ", "))
	args = append(args, block.ID)

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update lesson block: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("lesson block not found")
	}

	return nil
}

// Delete deletes a lesson block by ID
func (r *lessonBlockRepository) Delete(ctx context.Context, id int) error {
	query := `DELETE FROM lesson_blocks WHERE id = ?`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete lesson block: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("lesson block not found")
	}

	return nil
}

// CheckOwnership checks if the block belongs to the tutor
func (r *lessonBlockRepository) CheckOwnership(ctx context.Context, id, tutorID int) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM lesson_blocks WHERE id = ? AND lesson_id IN (SELECT id FROM lessons WHERE course_id IN (SELECT id FROM courses WHERE author_id = ?)))`

	var exists bool
	err := r.db.QueryRowContext(ctx, query, id, tutorID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check block ownership: %w", err)
	}

	return exists, nil
}
