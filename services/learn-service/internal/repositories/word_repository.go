package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/japanesestudent/learn-service/internal/models"
	"go.uber.org/zap"
)

// wordRepository implements WordRepository
type wordRepository struct {
	db     *sql.DB
	logger *zap.Logger
}

// NewWordRepository creates a new word repository
func NewWordRepository(db *sql.DB, logger *zap.Logger) *wordRepository {
	return &wordRepository{
		db:     db,
		logger: logger,
	}
}

// GetByIDs retrieves words by their IDs
func (r *wordRepository) GetByIDs(ctx context.Context, wordIds []int, translationField, exampleTranslationField string) ([]models.WordResponse, error) {
	if len(wordIds) == 0 {
		return []models.WordResponse{}, nil
	}

	// Build query with placeholders
	placeholders := make([]string, len(wordIds))
	args := make([]any, len(wordIds))
	for i, id := range wordIds {
		placeholders[i] = "?"
		args[i] = id
	}

	query := fmt.Sprintf(`
		SELECT id, word, phonetic_clues, %s as translation, example, %s as example_translation,
		       easy_period, normal_period, hard_period, extra_hard_period
		FROM words
		WHERE id IN (%s)
	`, translationField, exampleTranslationField, strings.Join(placeholders, ","))

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		r.logger.Error("failed to query words by IDs", zap.Error(err))
		return nil, fmt.Errorf("failed to query words: %w", err)
	}
	defer rows.Close()

	var words []models.WordResponse
	for rows.Next() {
		var word models.WordResponse
		err := rows.Scan(
			&word.ID,
			&word.Word,
			&word.PhoneticClues,
			&word.Translation,
			&word.Example,
			&word.ExampleTranslation,
			&word.EasyPeriod,
			&word.NormalPeriod,
			&word.HardPeriod,
			&word.ExtraHardPeriod,
		)
		if err != nil {
			r.logger.Error("failed to scan word", zap.Error(err))
			return nil, fmt.Errorf("failed to scan word: %w", err)
		}
		words = append(words, word)
	}

	if err = rows.Err(); err != nil {
		r.logger.Error("error iterating rows", zap.Error(err))
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return words, nil
}

// GetExcludingIDs retrieves words not in the provided ID list
func (r *wordRepository) GetExcludingIDs(ctx context.Context, excludeIds []int, limit int, translationField, exampleTranslationField string) ([]models.WordResponse, error) {
	var query string
	var args []any

	if len(excludeIds) == 0 {
		// If no IDs to exclude, get random words
		query = fmt.Sprintf(`
			SELECT id, word, phonetic_clues, %s as translation, example, %s as example_translation,
			       easy_period, normal_period, hard_period, extra_hard_period
			FROM words
			ORDER BY RAND()
			LIMIT ?
		`, translationField, exampleTranslationField)
		args = []any{limit}
	} else {
		// Build query with placeholders for exclusion
		placeholders := make([]string, len(excludeIds))
		args = make([]any, len(excludeIds)+1)
		for i, id := range excludeIds {
			placeholders[i] = "?"
			args[i] = id
		}
		args[len(excludeIds)] = limit

		query = fmt.Sprintf(`
			SELECT id, word, phonetic_clues, %s as translation, example, %s as example_translation,
			       easy_period, normal_period, hard_period, extra_hard_period
			FROM words
			WHERE id NOT IN (%s)
			ORDER BY RAND()
			LIMIT ?
		`, translationField, exampleTranslationField, strings.Join(placeholders, ","))
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		r.logger.Error("failed to query words excluding IDs", zap.Error(err))
		return nil, fmt.Errorf("failed to query words: %w", err)
	}
	defer rows.Close()

	var words []models.WordResponse
	for rows.Next() {
		var word models.WordResponse
		err := rows.Scan(
			&word.ID,
			&word.Word,
			&word.PhoneticClues,
			&word.Translation,
			&word.Example,
			&word.ExampleTranslation,
			&word.EasyPeriod,
			&word.NormalPeriod,
			&word.HardPeriod,
			&word.ExtraHardPeriod,
		)
		if err != nil {
			r.logger.Error("failed to scan word", zap.Error(err))
			return nil, fmt.Errorf("failed to scan word: %w", err)
		}
		words = append(words, word)
	}

	if err = rows.Err(); err != nil {
		r.logger.Error("error iterating rows", zap.Error(err))
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return words, nil
}

// ValidateWordIDs checks if all word IDs exist in the database
func (r *wordRepository) ValidateWordIDs(ctx context.Context, wordIds []int) (bool, error) {
	if len(wordIds) == 0 {
		return false, fmt.Errorf("word IDs list cannot be empty")
	}

	// Build query with placeholders
	placeholders := make([]string, len(wordIds))
	args := make([]any, len(wordIds))
	for i, id := range wordIds {
		placeholders[i] = "?"
		args[i] = id
	}

	query := fmt.Sprintf(`
		SELECT COUNT(*) as count
		FROM words
		WHERE id IN (%s)
	`, strings.Join(placeholders, ","))

	var count int
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&count)
	if err != nil {
		r.logger.Error("failed to validate word IDs", zap.Error(err))
		return false, fmt.Errorf("failed to validate word IDs: %w", err)
	}

	return count == len(wordIds), nil
}
