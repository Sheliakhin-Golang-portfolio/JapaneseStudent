package services

import (
	"context"
	"fmt"

	"github.com/Sheliakhin-Golang-portfolio/JapaneseStudent/learn-service/internal/models"
)

// CharactersRepository is the interface that wraps methods for Characters table data access
type CharactersRepository interface {
	// Method GetAll retrieve all hiragana/katakana characters from a database.
	//
	// "alphabetType" and "locale" parameters are used to configure return type of characters (hiragana or katakana) and reading (russian or english).
	// Please reference AlphabetType and Locale constants for correct parameters values.
	//
	// If wrong parameters will be used or some error will occur during data retrieve, the error will be returned together with "nil" value.
	GetAll(ctx context.Context, alphabetType models.AlphabetType, locale models.Locale) ([]models.CharacterResponse, error)
	// Method GetByRowColumn retrieve hiragana/katakana characters from a database filtered by consonant or vowel group ("character" parameter).
	//
	// Please reference GetAll method for more information about parameters and error values.
	GetByRowColumn(ctx context.Context, alphabetType models.AlphabetType, locale models.Locale, character string) ([]models.CharacterResponse, error)
	// Method GetByID retrieve a character by its ID.
	//
	// "locale" parameter is used to configure return type of characters (hiragana or katakana) and reading (russian or english).
	// Please reference Locale constants for correct parameter values.
	//
	// If wrong parameters will be used or some error will occur during data retrieve, the error will be returned together with "nil" value.
	GetByID(ctx context.Context, id int, locale models.Locale) (*models.Character, error)
	// Method GetCharactersForReadingTest retrieve characters for reading test with defined IDs.
	//
	// This method returns a slice of ReadingTestItem objects with defined Id, each containing one correct character and two wrong characters.
	// "alphabetType" parameter is used to identify the alphabet type.
	// "locale" parameter is used to identify the locale.
	// "characterIDs" parameter is used to identify the character IDs.
	//
	// If some error will occur during data retrieval, the error will be returned together with "nil" value.
	GetCharactersForReadingTest(ctx context.Context, alphabetType models.AlphabetType, locale models.Locale, characterIDs []int) ([]models.ReadingTestItem, error)
	// Method GetCharactersForWritingTest retrieve characters for writing test with defined IDs.
	//
	// This method returns a slice of "characterIDs" WritingTestItem objects, each containing one correct character and two wrong characters.
	// "alphabetType" parameter is used to identify the alphabet type.
	// "locale" parameter is used to identify the locale.
	// "characterIDs" parameter is used to identify the character IDs.
	//
	// If some error will occur during data retrieval, the error will be returned together with "nil" value.
	GetCharactersForWritingTest(ctx context.Context, alphabetType models.AlphabetType, locale models.Locale, characterIDs []int) ([]models.WritingTestItem, error)
	// Method GetCharactersForListeningTest retrieve characters for listening test with defined IDs.
	//
	// Please reference GetCharactersForReadingTest method for more information about parameters and error values.
	GetCharactersForListeningTest(ctx context.Context, alphabetType models.AlphabetType, locale models.Locale, characterIDs []int) ([]models.ListeningTestItem, error)
	// Method GetCharactersWithoutHistory retrieve characters that don't have CharacterLearnHistory records for the user.
	//
	// "alphabetType" parameter is used to identify the alphabet type.
	// "userID" parameter is used to identify the user.
	// "count" parameter is used to identify the number of characters to retrieve.
	//
	// If some error will occur during data retrieval, the error will be returned together with "nil" value.
	GetCharactersWithoutHistory(ctx context.Context, userID int, alphabetType models.AlphabetType, count int) ([]int, error)
	// Method GetCharactersWithLowestResults retrieve characters with lowest result values for the user.
	//
	// "alphabetType" parameter is used to identify the alphabet type.
	// "userID" parameter is used to identify the user.
	// "testTypeResultField" parameter is used to identify the test type result field.
	// "count" parameter is used to identify the number of characters to retrieve.
	//
	// If some error will occur during data retrieval, the error will be returned together with "nil" value.
	GetCharactersWithLowestResults(ctx context.Context, userID int, alphabetType models.AlphabetType, testTypeResultField string, count int) ([]int, error)
}

type charactersService struct {
	repo        CharactersRepository
	historyRepo CharacterLearnHistoryRepository
}

// NewCharactersService creates a new character service
func NewCharactersService(repo CharactersRepository, historyRepo CharacterLearnHistoryRepository) *charactersService {
	return &charactersService{
		repo:        repo,
		historyRepo: historyRepo,
	}
}

// GetAll retrieves all characters filtered by alphabet type and locale
//
// Please reference validateAlphabetType and validateLocale methods for more information about parameters and error values.
func (s *charactersService) GetAll(ctx context.Context, typeParam string, localeParam string) ([]models.CharacterResponse, error) {
	alphabetType := models.AlphabetType(typeParam)
	locale := models.Locale(localeParam)

	if err := s.validateAlphabetType(alphabetType); err != nil {
		return nil, err
	}
	// Normalize locale: treat "de" as "en"
	normalizedLocale := s.normalizeLocale(locale)
	if err := s.validateLocale(normalizedLocale); err != nil {
		return nil, err
	}

	return s.repo.GetAll(ctx, alphabetType, normalizedLocale)
}

// GetByRowColumn retrieves characters filtered by consonant or vowel
//
// Please reference validateAlphabetType and validateLocale methods for more information about parameters and error values.
func (s *charactersService) GetByRowColumn(ctx context.Context, typeParam string, localeParam string, character string) ([]models.CharacterResponse, error) {
	alphabetType := models.AlphabetType(typeParam)
	locale := models.Locale(localeParam)

	if err := s.validateAlphabetType(alphabetType); err != nil {
		return nil, err
	}
	// Normalize locale: treat "de" as "en"
	normalizedLocale := s.normalizeLocale(locale)
	if err := s.validateLocale(normalizedLocale); err != nil {
		return nil, err
	}
	if character == "" {
		return nil, fmt.Errorf("character parameter is required")
	}

	return s.repo.GetByRowColumn(ctx, alphabetType, normalizedLocale, character)
}

// GetByID retrieves a character by its ID
//
// For successful results localeParam must be either "ru" (Russian), "en" (English), or "de" (German - treated as English).
func (s *charactersService) GetByID(ctx context.Context, id int, localeParam string) (*models.Character, error) {
	locale := models.Locale(localeParam)

	if id <= 0 {
		return nil, fmt.Errorf("invalid character id")
	}
	// Normalize locale: treat "de" as "en"
	normalizedLocale := s.normalizeLocale(locale)
	if err := s.validateLocale(normalizedLocale); err != nil {
		return nil, err
	}

	return s.repo.GetByID(ctx, id, normalizedLocale)
}

// GetReadingTest retrieves random characters for reading test
//
// For successful results alphabetTypeStr must be either "hiragana" or "katakana" (from URL path).
// localeParam must be either "ru" (Russian), "en" (English), or "de" (German - treated as English).
// count must be a positive integer.
// userID is required - uses smart filtering based on user's learning history.
func (s *charactersService) GetReadingTest(ctx context.Context, alphabetTypeStr string, localeParam string, count int, userID int) ([]models.ReadingTestItem, error) {
	locale := models.Locale(localeParam)

	var at models.AlphabetType
	switch alphabetTypeStr {
	case "hiragana":
		at = models.AlphabetTypeHiragana
	case "katakana":
		at = models.AlphabetTypeKatakana
	default:
		return nil, fmt.Errorf("invalid alphabet type: %s, must be 'hiragana' or 'katakana'", alphabetTypeStr)
	}

	// Normalize locale: treat "de" as "en"
	normalizedLocale := s.normalizeLocale(locale)
	if err := s.validateLocale(normalizedLocale); err != nil {
		return nil, err
	}

	// Determine the result field based on alphabet type and test type
	var testTypeResultField string
	if alphabetTypeStr == "hiragana" {
		testTypeResultField = "hiragana_reading_result"
	} else {
		testTypeResultField = "katakana_reading_result"
	}

	// Get character IDs with smart filtering
	characterIDs, err := s.getCharacterIDsWithSmartFiltering(ctx, userID, at, testTypeResultField, count)
	if err != nil {
		return nil, fmt.Errorf("failed to get character IDs with smart filtering: %w", err)
	}

	// Get all test items
	return s.repo.GetCharactersForReadingTest(ctx, at, normalizedLocale, characterIDs)
}

// GetWritingTest retrieves random characters for writing test
//
// For successful results alphabetTypeStr must be either "hiragana" or "katakana" (from URL path).
// localeParam must be either "ru" (Russian), "en" (English), or "de" (German - treated as English).
// count must be a positive integer.
// userID is required - uses smart filtering based on user's learning history.
func (s *charactersService) GetWritingTest(ctx context.Context, alphabetTypeStr string, localeParam string, count int, userID int) ([]models.WritingTestItem, error) {
	locale := models.Locale(localeParam)

	var at models.AlphabetType
	switch alphabetTypeStr {
	case "hiragana":
		at = models.AlphabetTypeHiragana
	case "katakana":
		at = models.AlphabetTypeKatakana
	default:
		return nil, fmt.Errorf("invalid alphabet type: %s, must be 'hiragana' or 'katakana'", alphabetTypeStr)
	}

	// Normalize locale: treat "de" as "en"
	normalizedLocale := s.normalizeLocale(locale)
	if err := s.validateLocale(normalizedLocale); err != nil {
		return nil, err
	}

	// Determine the result field based on alphabet type and test type
	var testTypeResultField string
	if alphabetTypeStr == "hiragana" {
		testTypeResultField = "hiragana_writing_result"
	} else {
		testTypeResultField = "katakana_writing_result"
	}

	// Get character IDs with smart filtering
	characterIDs, err := s.getCharacterIDsWithSmartFiltering(ctx, userID, at, testTypeResultField, count)
	if err != nil {
		return nil, fmt.Errorf("failed to get character IDs with smart filtering: %w", err)
	}

	// Get all test items
	return s.repo.GetCharactersForWritingTest(ctx, at, normalizedLocale, characterIDs)
}

// GetListeningTest retrieves random characters for listening test
//
// For successful results alphabetTypeStr must be either "hiragana" or "katakana" (from URL path).
// localeParam must be either "ru" (Russian), "en" (English), or "de" (German - treated as English).
// count must be a positive integer.
// userID is required - uses smart filtering based on user's learning history.
func (s *charactersService) GetListeningTest(ctx context.Context, alphabetTypeStr string, localeParam string, count int, userID int) ([]models.ListeningTestItem, error) {
	locale := models.Locale(localeParam)

	var at models.AlphabetType
	switch alphabetTypeStr {
	case "hiragana":
		at = models.AlphabetTypeHiragana
	case "katakana":
		at = models.AlphabetTypeKatakana
	default:
		return nil, fmt.Errorf("invalid alphabet type: %s, must be 'hiragana' or 'katakana'", alphabetTypeStr)
	}

	// Normalize locale: treat "de" as "en"
	normalizedLocale := s.normalizeLocale(locale)
	if err := s.validateLocale(normalizedLocale); err != nil {
		return nil, err
	}

	// Determine the result field based on alphabet type and test type
	var testTypeResultField string
	if alphabetTypeStr == "hiragana" {
		testTypeResultField = "hiragana_listening_result"
	} else {
		testTypeResultField = "katakana_listening_result"
	}

	// Get character IDs with smart filtering
	characterIDs, err := s.getCharacterIDsWithSmartFiltering(ctx, userID, at, testTypeResultField, count)
	if err != nil {
		return nil, fmt.Errorf("failed to get character IDs with smart filtering: %w", err)
	}

	// Get all test items
	return s.repo.GetCharactersForListeningTest(ctx, at, normalizedLocale, characterIDs)
}

// validateAlphabetType validates the alphabet type
func (s *charactersService) validateAlphabetType(at models.AlphabetType) error {
	if at != models.AlphabetTypeHiragana && at != models.AlphabetTypeKatakana {
		return fmt.Errorf("invalid alphabet type: %s, must be 'hr' or 'kt'", at)
	}
	return nil
}

// normalizeLocale normalizes locale: treats "de" as "en"
func (s *charactersService) normalizeLocale(locale models.Locale) models.Locale {
	if locale == models.LocaleGerman {
		return models.LocaleEnglish
	}
	return locale
}

// validateLocale validates the locale
func (s *charactersService) validateLocale(locale models.Locale) error {
	if locale != models.LocaleEnglish && locale != models.LocaleRussian {
		return fmt.Errorf("invalid locale: %s, must be 'en' or 'ru'", locale)
	}
	return nil
}

// getCharacterIDsWithSmartFiltering retrieves character IDs with smart filtering based on user history
func (s *charactersService) getCharacterIDsWithSmartFiltering(ctx context.Context, userID int, alphabetType models.AlphabetType, testTypeResultField string, count int) ([]int, error) {
	// Get character IDs without history first
	characterIDs, err := s.repo.GetCharactersWithoutHistory(ctx, userID, alphabetType, count)
	if err != nil {
		return nil, fmt.Errorf("failed to get characters without history: %w", err)
	}

	// If not enough, get more from lowest results
	if len(characterIDs) < count {
		remainingCount := count - len(characterIDs)
		lowestResultIDs, err := s.repo.GetCharactersWithLowestResults(ctx, userID, alphabetType, testTypeResultField, remainingCount)
		if err != nil {
			return nil, fmt.Errorf("failed to get characters with lowest results: %w", err)
		}
		characterIDs = append(characterIDs, lowestResultIDs...)
	}
	return characterIDs, nil
}
