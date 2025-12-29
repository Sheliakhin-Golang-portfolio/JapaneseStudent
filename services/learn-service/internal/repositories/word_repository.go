package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/japanesestudent/learn-service/internal/models"
)

// wordRepository implements WordRepository
type wordRepository struct {
	db *sql.DB
}

// NewWordRepository creates a new word repository
func NewWordRepository(db *sql.DB) *wordRepository {
	return &wordRepository{
		db: db,
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
		       easy_period, normal_period, hard_period, extra_hard_period, word_audio, word_example_audio
		FROM words
		WHERE id IN (%s)
	`, translationField, exampleTranslationField, strings.Join(placeholders, ","))

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query words: %w", err)
	}
	defer rows.Close()

	var words []models.WordResponse
	for rows.Next() {
		var word models.WordResponse
		var wordAudio, wordExampleAudio sql.NullString
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
			&wordAudio,
			&wordExampleAudio,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan word: %w", err)
		}
		if wordAudio.Valid {
			word.WordAudio = wordAudio.String
		}
		if wordExampleAudio.Valid {
			word.WordExampleAudio = wordExampleAudio.String
		}
		words = append(words, word)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return words, nil
}

// GetExcludingIDs retrieves words not in the provided ID list
func (r *wordRepository) GetExcludingIDs(ctx context.Context, userId int, excludeIds []int, limit int, translationField, exampleTranslationField string) ([]models.WordResponse, error) {
	var query string
	var args []any

	if len(excludeIds) == 0 {
		// If no IDs to exclude, get random words
		query = fmt.Sprintf(`
			SELECT id, word, phonetic_clues, %s as translation, example, %s as example_translation,
			       easy_period, normal_period, hard_period, extra_hard_period, word_audio, word_example_audio
			FROM words
			ORDER BY (EXISTS (SELECT 1 FROM dictionary_history WHERE word_id = words.id AND user_id = ?)), RAND()
			LIMIT ?
		`, translationField, exampleTranslationField)
		args = []any{userId, limit}
	} else {
		// Build query with placeholders for exclusion
		placeholders := make([]string, len(excludeIds))
		args = make([]any, len(excludeIds)+2)
		for i, id := range excludeIds {
			placeholders[i] = "?"
			args[i] = id
		}
		args[len(excludeIds)] = userId
		args[len(excludeIds)+1] = limit

		query = fmt.Sprintf(`
			SELECT id, word, phonetic_clues, %s as translation, example, %s as example_translation,
			       easy_period, normal_period, hard_period, extra_hard_period, word_audio, word_example_audio
			FROM words
			WHERE id NOT IN (%s)
			ORDER BY (EXISTS (SELECT 1 FROM dictionary_history WHERE word_id = words.id AND user_id = ?)), RAND()
			LIMIT ?
		`, translationField, exampleTranslationField, strings.Join(placeholders, ","))
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query words: %w", err)
	}
	defer rows.Close()

	var words []models.WordResponse
	for rows.Next() {
		var word models.WordResponse
		var wordAudio, wordExampleAudio sql.NullString
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
			&wordAudio,
			&wordExampleAudio,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan word: %w", err)
		}
		if wordAudio.Valid {
			word.WordAudio = wordAudio.String
		}
		if wordExampleAudio.Valid {
			word.WordExampleAudio = wordExampleAudio.String
		}
		words = append(words, word)
	}

	if err = rows.Err(); err != nil {
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
		return false, fmt.Errorf("failed to validate word IDs: %w", err)
	}

	return count == len(wordIds), nil
}

// GetAllForAdmin retrieves a paginated list of words with optional search filter
func (r *wordRepository) GetAllForAdmin(ctx context.Context, page, count int, search string) ([]models.Word, error) {
	// Build WHERE clause
	var whereClause string
	var args []any

	if search != "" {
		whereClause = `WHERE word LIKE ? OR phonetic_clues LIKE ? OR english_translation LIKE ? OR russian_translation LIKE ? OR german_translation LIKE ?`
		searchValue := "%" + search + "%"
		args = append(args, searchValue, searchValue, searchValue, searchValue, searchValue)
	}

	// Calculate offset
	offset := (page - 1) * count

	query := fmt.Sprintf(`
		SELECT id, word, phonetic_clues, english_translation
		FROM words
		%s
		ORDER BY english_translation
		LIMIT ? OFFSET ?
	`, whereClause)

	args = append(args, count, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query words: %w", err)
	}
	defer rows.Close()

	var words []models.Word
	for rows.Next() {
		var word models.Word
		err := rows.Scan(
			&word.ID,
			&word.Word,
			&word.PhoneticClues,
			&word.EnglishTranslation,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan word: %w", err)
		}
		words = append(words, word)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return words, nil
}

// GetByID retrieves a word by ID
func (r *wordRepository) GetByIDAdmin(ctx context.Context, id int) (*models.Word, error) {
	query := `
		SELECT id, word, phonetic_clues, russian_translation, english_translation, german_translation,
		       example, example_russian_translation, example_english_translation, example_german_translation,
		       easy_period, normal_period, hard_period, extra_hard_period, word_audio, word_example_audio
		FROM words
		WHERE id = ?
		LIMIT 1
	`

	word := &models.Word{}
	var wordAudio, wordExampleAudio sql.NullString
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&word.ID,
		&word.Word,
		&word.PhoneticClues,
		&word.RussianTranslation,
		&word.EnglishTranslation,
		&word.GermanTranslation,
		&word.Example,
		&word.ExampleRussianTranslation,
		&word.ExampleEnglishTranslation,
		&word.ExampleGermanTranslation,
		&word.EasyPeriod,
		&word.NormalPeriod,
		&word.HardPeriod,
		&word.ExtraHardPeriod,
		&wordAudio,
		&wordExampleAudio,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("word not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get word by ID: %w", err)
	}

	if wordAudio.Valid {
		word.WordAudio = wordAudio.String
	}
	if wordExampleAudio.Valid {
		word.WordExampleAudio = wordExampleAudio.String
	}

	return word, nil
}

// ExistsByWord checks if a word with the given Word field exists
func (r *wordRepository) ExistsByWord(ctx context.Context, word string) (bool, error) {
	query := `SELECT EXISTS(SELECT * FROM words WHERE word = ?)`

	var exists bool
	err := r.db.QueryRowContext(ctx, query, word).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check word existence: %w", err)
	}

	return exists, nil
}

// Create inserts a new word into the database
func (r *wordRepository) Create(ctx context.Context, word *models.Word) error {
	query := `
		INSERT INTO words (word, phonetic_clues, russian_translation, english_translation, german_translation,
		                  example, example_russian_translation, example_english_translation, example_german_translation,
		                  easy_period, normal_period, hard_period, extra_hard_period, word_audio, word_example_audio)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := r.db.ExecContext(ctx, query,
		word.Word,
		word.PhoneticClues,
		word.RussianTranslation,
		word.EnglishTranslation,
		word.GermanTranslation,
		word.Example,
		word.ExampleRussianTranslation,
		word.ExampleEnglishTranslation,
		word.ExampleGermanTranslation,
		word.EasyPeriod,
		word.NormalPeriod,
		word.HardPeriod,
		word.ExtraHardPeriod,
		word.WordAudio,
		word.WordExampleAudio,
	)
	if err != nil {
		return fmt.Errorf("failed to create word: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}

	word.ID = int(id)
	return nil
}

// ExistsByWord checks if a word with the given Word field exists
func (r *wordRepository) ExistsByClues(ctx context.Context, clues string) (bool, error) {
	query := `SELECT EXISTS(SELECT * FROM words WHERE phonetic_clues = ?)`

	var exists bool
	err := r.db.QueryRowContext(ctx, query, clues).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check word existence: %w", err)
	}

	return exists, nil
}

// Update updates word fields (partial update)
func (r *wordRepository) Update(ctx context.Context, id int, word *models.Word) error {
	// Build dynamic UPDATE query based on provided fields
	var setParts []string
	var args []any

	if word.Word != "" {
		setParts = append(setParts, "word = ?")
		args = append(args, word.Word)
	}
	if word.PhoneticClues != "" {
		setParts = append(setParts, "phonetic_clues = ?")
		args = append(args, word.PhoneticClues)
	}
	if word.RussianTranslation != "" {
		setParts = append(setParts, "russian_translation = ?")
		args = append(args, word.RussianTranslation)
	}
	if word.EnglishTranslation != "" {
		setParts = append(setParts, "english_translation = ?")
		args = append(args, word.EnglishTranslation)
	}
	if word.GermanTranslation != "" {
		setParts = append(setParts, "german_translation = ?")
		args = append(args, word.GermanTranslation)
	}
	if word.Example != "" {
		setParts = append(setParts, "example = ?")
		args = append(args, word.Example)
	}
	if word.ExampleRussianTranslation != "" {
		setParts = append(setParts, "example_russian_translation = ?")
		args = append(args, word.ExampleRussianTranslation)
	}
	if word.ExampleEnglishTranslation != "" {
		setParts = append(setParts, "example_english_translation = ?")
		args = append(args, word.ExampleEnglishTranslation)
	}
	if word.ExampleGermanTranslation != "" {
		setParts = append(setParts, "example_german_translation = ?")
		args = append(args, word.ExampleGermanTranslation)
	}
	if word.EasyPeriod != 0 {
		setParts = append(setParts, "easy_period = ?")
		args = append(args, word.EasyPeriod)
	}
	if word.NormalPeriod != 0 {
		setParts = append(setParts, "normal_period = ?")
		args = append(args, word.NormalPeriod)
	}
	if word.HardPeriod != 0 {
		setParts = append(setParts, "hard_period = ?")
		args = append(args, word.HardPeriod)
	}
	if word.ExtraHardPeriod != 0 {
		setParts = append(setParts, "extra_hard_period = ?")
		args = append(args, word.ExtraHardPeriod)
	}
	if word.WordAudio != "" {
		setParts = append(setParts, "word_audio = ?")
		args = append(args, word.WordAudio)
	}
	if word.WordExampleAudio != "" {
		setParts = append(setParts, "word_example_audio = ?")
		args = append(args, word.WordExampleAudio)
	}

	if len(setParts) == 0 {
		return fmt.Errorf("no fields to update")
	}

	query := fmt.Sprintf(`
		UPDATE words
		SET %s
		WHERE id = ?
	`, strings.Join(setParts, ", "))

	args = append(args, id)

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update word: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("word not found")
	}

	return nil
}

// Delete deletes a word by ID
func (r *wordRepository) Delete(ctx context.Context, id int) error {
	query := `DELETE FROM words WHERE id = ?`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete word: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("word not found")
	}

	return nil
}
