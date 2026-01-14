package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/japanesestudent/learn-service/internal/models"
)

// characterLearnHistoryRepository implements CharacterLearnHistoryRepository
type characterLearnHistoryRepository struct {
	db *sql.DB
}

// NewCharacterLearnHistoryRepository creates a new character learn history repository
func NewCharacterLearnHistoryRepository(db *sql.DB) *characterLearnHistoryRepository {
	return &characterLearnHistoryRepository{
		db: db,
	}
}

// GetByUserIDAndCharacterIDs retrieves learn history records for a user and set of character IDs
func (r *characterLearnHistoryRepository) GetByUserIDAndCharacterIDs(ctx context.Context, userID int, characterIDs []int) ([]models.CharacterLearnHistory, error) {
	if len(characterIDs) == 0 {
		return []models.CharacterLearnHistory{}, nil
	}

	// Build query with placeholders
	args := []any{userID}

	// Prepare the query for IN clause
	charPlaceholders := make([]string, len(characterIDs))
	for i := range charPlaceholders {
		charPlaceholders[i] = "?"
		args = append(args, characterIDs[i])
	}

	// Query to retrieve learn history records.
	// The query is prepared for IN clause to avoid multiple queries.
	// Placeholders are transformed into "?, ?, ?" string for slice insertion.
	query := fmt.Sprintf(`
		SELECT id, user_id, character_id, hiragana_reading_result, hiragana_writing_result,
		       hiragana_listening_result, katakana_reading_result, katakana_writing_result,
		       katakana_listening_result
		FROM character_learn_history
		WHERE user_id = ? AND character_id IN (%s)`, strings.Join(charPlaceholders, ","))

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query character learn history: %w", err)
	}
	defer rows.Close()

	var histories []models.CharacterLearnHistory
	for rows.Next() {
		var history models.CharacterLearnHistory
		err := rows.Scan(
			&history.ID,
			&history.UserID,
			&history.CharacterID,
			&history.HiraganaReadingResult,
			&history.HiraganaWritingResult,
			&history.HiraganaListeningResult,
			&history.KatakanaReadingResult,
			&history.KatakanaWritingResult,
			&history.KatakanaListeningResult,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan character learn history: %w", err)
		}
		histories = append(histories, history)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return histories, nil
}

// GetByUserID retrieves all learn history records for a user
func (r *characterLearnHistoryRepository) GetByUserID(ctx context.Context, userID int) ([]models.UserLearnHistory, error) {
	query := `
		SELECT
		character_learn_history.hiragana_reading_result,
		character_learn_history.hiragana_writing_result,
		character_learn_history.hiragana_listening_result,
		character_learn_history.katakana_reading_result,
		character_learn_history.katakana_writing_result,
		character_learn_history.katakana_listening_result,
		characters.hiragana,
		characters.katakana
		FROM character_learn_history
		JOIN characters ON character_learn_history.character_id = characters.id
		WHERE user_id = ?
		ORDER BY characters.id ASC
	`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query character learn history by user ID: %w", err)
	}
	defer rows.Close()

	var histories []models.UserLearnHistory
	for rows.Next() {
		var history models.UserLearnHistory
		err := rows.Scan(
			&history.HiraganaReadingResult,
			&history.HiraganaWritingResult,
			&history.HiraganaListeningResult,
			&history.KatakanaReadingResult,
			&history.KatakanaWritingResult,
			&history.KatakanaListeningResult,
			&history.CharacterHiragana,
			&history.CharacterKatakana,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan character learn history: %w", err)
		}
		histories = append(histories, history)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return histories, nil
}

// Upsert inserts or updates a list of character learn history records
func (r *characterLearnHistoryRepository) Upsert(ctx context.Context, histories []models.CharacterLearnHistory) error {
	if len(histories) == 0 {
		return fmt.Errorf("no histories to upsert")
	}

	charPlaceholders := make([]string, len(histories))
	args := []any{}
	for i := range charPlaceholders {
		charPlaceholders[i] = "(?, ?, ?, ?, ?, ?, ?, ?)"
		args = append(args, histories[i].UserID, histories[i].CharacterID, histories[i].HiraganaReadingResult,
			histories[i].HiraganaWritingResult, histories[i].HiraganaListeningResult, histories[i].KatakanaReadingResult,
			histories[i].KatakanaWritingResult, histories[i].KatakanaListeningResult)
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	query := fmt.Sprintf(`
		INSERT INTO character_learn_history
		(user_id, character_id, hiragana_reading_result, hiragana_writing_result,
		 hiragana_listening_result, katakana_reading_result, katakana_writing_result,
		 katakana_listening_result)
		VALUES %s
		ON DUPLICATE KEY UPDATE
			hiragana_reading_result = VALUES(hiragana_reading_result),
			hiragana_writing_result = VALUES(hiragana_writing_result),
			hiragana_listening_result = VALUES(hiragana_listening_result),
			katakana_reading_result = VALUES(katakana_reading_result),
			katakana_writing_result = VALUES(katakana_writing_result),
			katakana_listening_result = VALUES(katakana_listening_result)
	`, strings.Join(charPlaceholders, ","))

	if _, err = tx.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("failed to upsert character learn history: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// LowerResultsByUserID lowers all result values by 0.01 for all CharacterLearnHistory records for a user
//
// It allows to sort provided test data by mark in specified category.
func (r *characterLearnHistoryRepository) LowerResultsByUserID(ctx context.Context, userID int) error {
	query := `
		UPDATE character_learn_history 
		SET 
			hiragana_reading_result = GREATEST(0, hiragana_reading_result - 0.01),
			hiragana_writing_result = GREATEST(0, hiragana_writing_result - 0.01),
			hiragana_listening_result = GREATEST(0, hiragana_listening_result - 0.01),
			katakana_reading_result = GREATEST(0, katakana_reading_result - 0.01),
			katakana_writing_result = GREATEST(0, katakana_writing_result - 0.01),
			katakana_listening_result = GREATEST(0, katakana_listening_result - 0.01)
		WHERE user_id = ?
	`

	_, err := r.db.ExecContext(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("failed to lower results: %w", err)
	}

	// Return nil even if no rows affected (user has no history - not an error)
	return nil
}
