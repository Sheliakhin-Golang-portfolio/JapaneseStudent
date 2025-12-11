package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"slices"
	"strings"

	"github.com/japanesestudent/backend/internal/models"
	"go.uber.org/zap"
)

type charactersRepository struct {
	db     *sql.DB
	logger *zap.Logger
}

// NewCharactersRepository creates a new instance of the CharacterRepository interface
func NewCharactersRepository(db *sql.DB, logger *zap.Logger) *charactersRepository {
	return &charactersRepository{
		db:     db,
		logger: logger,
	}
}

// Method GetAll is a CharactersRepository implementation for retrieving all hiragana/katakana characters from a database.
func (r *charactersRepository) GetAll(ctx context.Context, alphabetType models.AlphabetType, locale models.Locale) ([]models.CharacterResponse, error) {
	var charField string
	switch alphabetType {
	case models.AlphabetTypeHiragana:
		charField = "hiragana"
	case models.AlphabetTypeKatakana:
		charField = "katakana"
	default:
		return nil, fmt.Errorf("invalid alphabet type: %s", alphabetType)
	}

	var readingField string
	switch locale {
	case models.LocaleEnglish:
		readingField = "english_reading"
	case models.LocaleRussian:
		readingField = "russian_reading"
	default:
		return nil, fmt.Errorf("invalid locale: %s", locale)
	}

	// Query to retrieve all characters from the database
	// Names of optional fields are specified in the parameters of the method
	query := fmt.Sprintf(`
		SELECT id, consonant, vowel, %s AS display_character, %s AS reading
		FROM characters
		ORDER BY id
	`, charField, readingField)

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		r.logger.Error("failed to query characters", zap.Error(err))
		return nil, fmt.Errorf("failed to query characters: %w", err)
	}
	defer rows.Close()

	var characters []models.CharacterResponse
	for rows.Next() {
		var char models.CharacterResponse
		if err := rows.Scan(&char.ID, &char.Consonant, &char.Vowel, &char.Character, &char.Reading); err != nil {
			r.logger.Error("failed to scan character", zap.Error(err))
			return nil, fmt.Errorf("failed to scan character: %w", err)
		}
		characters = append(characters, char)
	}

	if err := rows.Err(); err != nil {
		r.logger.Error("error iterating rows", zap.Error(err))
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return characters, nil
}

// GetByRowColumn retrieves characters filtered by consonant or vowel
func (r *charactersRepository) GetByRowColumn(ctx context.Context, alphabetType models.AlphabetType, locale models.Locale, character string) ([]models.CharacterResponse, error) {
	var charField string
	switch alphabetType {
	case models.AlphabetTypeHiragana:
		charField = "hiragana"
	case models.AlphabetTypeKatakana:
		charField = "katakana"
	default:
		return nil, fmt.Errorf("invalid alphabet type: %s", alphabetType)
	}

	var readingField string
	switch locale {
	case models.LocaleEnglish:
		readingField = "english_reading"
	case models.LocaleRussian:
		readingField = "russian_reading"
	default:
		return nil, fmt.Errorf("invalid locale: %s", locale)
	}

	// Check if character is a vowel or consonant and filter accordingly
	// If it's a vowel, return consonant field; if consonant, return vowel field
	query := fmt.Sprintf(`
		SELECT id, consonant, vowel, %s AS display_character, %s AS reading
		FROM characters
		WHERE (consonant = ? OR vowel = ?)
		ORDER BY id
	`, charField, readingField)

	rows, err := r.db.QueryContext(ctx, query, character, character)
	if err != nil {
		r.logger.Error("failed to query characters by row/column", zap.Error(err))
		return nil, fmt.Errorf("failed to query characters: %w", err)
	}
	defer rows.Close()

	var characters []models.CharacterResponse
	for rows.Next() {
		var char models.CharacterResponse
		var consonant, vowel string
		if err := rows.Scan(&char.ID, &consonant, &vowel, &char.Character, &char.Reading); err != nil {
			r.logger.Error("failed to scan character", zap.Error(err))
			return nil, fmt.Errorf("failed to scan character: %w", err)
		}
		// Populate the field that matches the search parameter
		// When searching by vowel, populate Vowel field; when searching by consonant, populate Consonant field
		if isVowel(character) {
			char.Vowel = vowel
			// Don't set consonant field (it will be omitted in JSON due to omitempty)
		} else {
			char.Consonant = consonant
			// Don't set vowel field (it will be omitted in JSON due to omitempty)
		}
		characters = append(characters, char)
	}

	if err := rows.Err(); err != nil {
		r.logger.Error("error iterating rows", zap.Error(err))
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return characters, nil
}

// isVowel checks if a character is a vowel
func isVowel(char string) bool {
	vowels := []string{"a", "i", "u", "e", "o"}
	return slices.Contains(vowels, char)
}

// GetByID retrieves a character by its ID
func (r *charactersRepository) GetByID(ctx context.Context, id int, locale models.Locale) (*models.Character, error) {
	var readingField string
	switch locale {
	case models.LocaleEnglish:
		readingField = "english_reading"
	case models.LocaleRussian:
		readingField = "russian_reading"
	default:
		return nil, fmt.Errorf("invalid locale: %s", locale)
	}

	// Query to retrieve a character by its ID
	// Reading field is retrieved based on the locale parameter.
	query := fmt.Sprintf(`
		SELECT id, consonant, vowel, %s as reading, katakana, hiragana
		FROM characters
		WHERE id = ?
	`, readingField)

	var char models.Character
	var reading string
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&char.ID,
		&char.Consonant,
		&char.Vowel,
		&reading,
		&char.Katakana,
		&char.Hiragana,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("character not found")
		}
		r.logger.Error("failed to query character by id", zap.Error(err), zap.Int("id", id))
		return nil, fmt.Errorf("failed to query character: %w", err)
	}

	// Set the appropriate reading field
	if locale == models.LocaleEnglish {
		char.EnglishReading = reading
	} else {
		char.RussianReading = reading
	}

	return &char, nil
}

// GetRandomForReadingTest retrieves a slice of random characters for reading test with multiple choice options
func (r *charactersRepository) GetRandomForReadingTest(ctx context.Context, alphabetType models.AlphabetType, locale models.Locale, count int) ([]models.ReadingTestItem, error) {
	var charField string
	switch alphabetType {
	case models.AlphabetTypeHiragana:
		charField = "hiragana"
	case models.AlphabetTypeKatakana:
		charField = "katakana"
	default:
		return nil, fmt.Errorf("invalid alphabet type: %s", alphabetType)
	}

	var readingField string
	switch locale {
	case models.LocaleEnglish:
		readingField = "english_reading"
	case models.LocaleRussian:
		readingField = "russian_reading"
	default:
		return nil, fmt.Errorf("invalid locale: %s", locale)
	}

	query := fmt.Sprintf(`
		SELECT id, %s AS display_character, %s AS reading
		FROM characters
		WHERE %s IS NOT NULL AND %s != ''
		ORDER BY RAND()
		LIMIT ?
	`, charField, readingField, charField, readingField)

	rows, err := r.db.QueryContext(ctx, query, count)
	if err != nil {
		r.logger.Error("failed to query random characters for reading test", zap.Error(err))
		return nil, fmt.Errorf("failed to query characters: %w", err)
	}
	defer rows.Close()

	var items []models.ReadingTestItem
	// Prepare the slice for IN clause.
	// Slice is of type interface{} to avoid type assertion errors.
	var correctChars []interface{}
	for rows.Next() {
		var testItem models.ReadingTestItem
		if err := rows.Scan(&testItem.ID, &testItem.CorrectChar, &testItem.Reading); err != nil {
			r.logger.Error("failed to scan reading test item", zap.Error(err))
			return nil, fmt.Errorf("failed to scan item: %w", err)
		}
		testItem.WrongOptions = make([]string, 2)
		items = append(items, testItem)
		correctChars = append(correctChars, testItem.CorrectChar)
	}

	if err := rows.Err(); err != nil {
		r.logger.Error("error iterating rows", zap.Error(err))
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	// Prepare the query for IN clause
	charPlaceholders := make([]string, len(correctChars))
	for i := range charPlaceholders {
		charPlaceholders[i] = "?"
	}
	// Query to retrieve wrong options.
	// The query is prepared for IN clause to avoid multiple queries.
	// Placeholders are transformed into "?, ?, ?" string for slice insertion.
	wrongQuery := fmt.Sprintf(`
		SELECT %s AS display_character
		FROM characters
		WHERE %s NOT IN (%s) AND %s IS NOT NULL AND %s != ''
		ORDER BY RAND()
		`, charField, charField, strings.Join(charPlaceholders, ","), charField, charField)
	wrongRows, err := r.db.QueryContext(ctx, wrongQuery, correctChars...)
	if err != nil {
		r.logger.Error("failed to query wrong options", zap.Error(err))
		return nil, fmt.Errorf("failed to query wrong options: %w", err)
	}
	defer wrongRows.Close()

	for _, item := range items {
		var wrongCharFirst, wrongCharSecond string
		if hasNext, err := wrongRows.Next(), wrongRows.Scan(&wrongCharFirst); !hasNext || err != nil {
			wrongRows.Close()
			if !hasNext {
				return nil, fmt.Errorf("failed to scan wrong option: %w", fmt.Errorf("no more rows"))
			}
			return nil, fmt.Errorf("failed to scan wrong option: %w", err)
		}
		if hasNext, err := wrongRows.Next(), wrongRows.Scan(&wrongCharSecond); !hasNext || err != nil {
			wrongRows.Close()
			if !hasNext {
				return nil, fmt.Errorf("failed to scan wrong option: %w", fmt.Errorf("no more rows"))
			}
			return nil, fmt.Errorf("failed to scan wrong option: %w", err)
		}
		item.WrongOptions = append(item.WrongOptions, wrongCharFirst, wrongCharSecond)
	}

	return items, nil
}

// GetRandomForWritingTest retrieves random characters for writing test with correct reading
func (r *charactersRepository) GetRandomForWritingTest(ctx context.Context, alphabetType models.AlphabetType, locale models.Locale, count int) ([]models.WritingTestItem, error) {
	var charField string
	switch alphabetType {
	case models.AlphabetTypeHiragana:
		charField = "hiragana"
	case models.AlphabetTypeKatakana:
		charField = "katakana"
	default:
		return nil, fmt.Errorf("invalid alphabet type: %s", alphabetType)
	}

	var readingField string
	switch locale {
	case models.LocaleEnglish:
		readingField = "english_reading"
	case models.LocaleRussian:
		readingField = "russian_reading"
	default:
		return nil, fmt.Errorf("invalid locale: %s", locale)
	}

	// Get random characters
	query := fmt.Sprintf(`
		SELECT id, %s AS display_character, %s AS reading
		FROM characters
		WHERE %s IS NOT NULL AND %s != ''
		ORDER BY RAND()
		LIMIT ?
	`, charField, readingField, charField, readingField)

	rows, err := r.db.QueryContext(ctx, query, count)
	if err != nil {
		r.logger.Error("failed to query random characters for writing test", zap.Error(err))
		return nil, fmt.Errorf("failed to query characters: %w", err)
	}
	defer rows.Close()

	var items []models.WritingTestItem
	for rows.Next() {
		var item models.WritingTestItem
		if err := rows.Scan(&item.ID, &item.Character, &item.CorrectReading); err != nil {
			r.logger.Error("failed to scan writing test item", zap.Error(err))
			return nil, fmt.Errorf("failed to scan item: %w", err)
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		r.logger.Error("error iterating rows", zap.Error(err))
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return items, nil
}
