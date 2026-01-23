package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/Sheliakhin-Golang-portfolio/JapaneseStudent/services/learn-service/internal/models"
)

// dictionaryHistoryRepository implements DictionaryHistoryRepository
type dictionaryHistoryRepository struct {
	db *sql.DB
}

// NewDictionaryHistoryRepository creates a new dictionary history repository
func NewDictionaryHistoryRepository(db *sql.DB) *dictionaryHistoryRepository {
	return &dictionaryHistoryRepository{
		db: db,
	}
}

// GetOldWordIds retrieves word IDs from dictionary history where NextAppearance <= current day
//
// "userId" parameter is used to identify the user.
// "limit" parameter is used to specify the number of words to return.
// Please reference GetByIDs method for more information about other parameters and error values.
func (r *dictionaryHistoryRepository) GetOldWordIds(ctx context.Context, userId int, limit int) ([]int, error) {
	query := `
		SELECT word_id
		FROM dictionary_history
		WHERE user_id = ? AND next_appearance <= CURDATE()
		ORDER BY next_appearance ASC
		LIMIT ?
	`

	rows, err := r.db.QueryContext(ctx, query, userId, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query old word IDs: %w", err)
	}
	defer rows.Close()

	var wordIds []int
	for rows.Next() {
		var wordId int
		if err := rows.Scan(&wordId); err != nil {
			return nil, fmt.Errorf("failed to scan word ID: %w", err)
		}
		wordIds = append(wordIds, wordId)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return wordIds, nil
}

// UpsertResults inserts or updates dictionary history records
//
// "userId" parameter is used to identify the user.
// "results" parameter is used to submit the word learning results.
func (r *dictionaryHistoryRepository) UpsertResults(ctx context.Context, userId int, results []models.WordResult) error {
	if len(results) == 0 {
		return fmt.Errorf("no results to upsert")
	}

	// Build placeholders and args for batch insert
	placeholders := make([]string, len(results))
	args := []any{}
	for i, result := range results {
		placeholders[i] = "(?, ?, DATE_ADD(CURDATE(), INTERVAL ? DAY))"
		args = append(args, userId, result.WordID, result.Period)
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	query := fmt.Sprintf(`
		INSERT INTO dictionary_history (user_id, word_id, next_appearance)
		VALUES %s
		ON DUPLICATE KEY UPDATE
			next_appearance = VALUES(next_appearance)
	`, strings.Join(placeholders, ","))

	if _, err = tx.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("failed to upsert dictionary history: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
