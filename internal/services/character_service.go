package services

import (
	"context"
	"fmt"

	"github.com/japanesestudent/backend/internal/models"
	"go.uber.org/zap"
)

// CharactersRepository is the interface that wraps methods for Characters table data access
type CharactersRepository interface {
	// Method GetAll retrieve all hiragana/katakana characters from a database.
	//
	// "alphabetType" and "locale" parameters are used to configure return type of characters (hiragana or katakana) and reading (russian or english).
	// Please reference AlphabetType and Locale constants for correct parameters values.
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
	// If wrong parameters will be used or some error will occur during data retrieve, the error will be returned together with "nil" value.
	GetByID(ctx context.Context, id int, locale models.Locale) (*models.Character, error)
	// Method GetRandomForReadingTest retrieve random characters for reading test.
	//
	// This method returns a slice of "count" random ReadingTestItem objects, each containing one correct character and two wrong characters.
	// lease reference GetAll method for more information about other parameters and error values.
	GetRandomForReadingTest(ctx context.Context, alphabetType models.AlphabetType, locale models.Locale, count int) ([]models.ReadingTestItem, error)
	// Method GetRandomForWritingTest retrieve random characters for writing test.
	//
	// Please reference GetAll method for more information about other parameters and error values.
	GetRandomForWritingTest(ctx context.Context, alphabetType models.AlphabetType, locale models.Locale, count int) ([]models.WritingTestItem, error)
}

const testCount = 10

type charactersService struct {
	repo   CharactersRepository
	logger *zap.Logger
}

// NewCharactersService creates a new character service
func NewCharactersService(repo CharactersRepository, logger *zap.Logger) *charactersService {
	return &charactersService{
		repo:   repo,
		logger: logger,
	}
}

// GetAll retrieves all characters filtered by alphabet type and locale
//
// For successful results typeParam must be either "hr" (hiragana) or "kt" (katakana).
// localeParam must be either "ru" (Russian) or "en" (English).
func (s *charactersService) GetAll(ctx context.Context, typeParam string, localeParam string) ([]models.CharacterResponse, error) {
	alphabetType := models.AlphabetType(typeParam)
	locale := models.Locale(localeParam)

	if err := s.validateAlphabetType(alphabetType); err != nil {
		return nil, err
	}
	if err := s.validateLocale(locale); err != nil {
		return nil, err
	}

	characters, err := s.repo.GetAll(ctx, alphabetType, locale)
	if err != nil {
		s.logger.Error("failed to get all characters", zap.Error(err))
		return nil, fmt.Errorf("failed to get characters: %w", err)
	}

	return characters, nil
}

// GetByRowColumn retrieves characters filtered by consonant or vowel
//
// For successful results typeParam must be either "hr" (hiragana) or "kt" (katakana).
// localeParam must be either "ru" (Russian) or "en" (English).
// "character" parameter must be of known vowel or consonant group.
func (s *charactersService) GetByRowColumn(ctx context.Context, typeParam string, localeParam string, character string) ([]models.CharacterResponse, error) {
	alphabetType := models.AlphabetType(typeParam)
	locale := models.Locale(localeParam)

	if err := s.validateAlphabetType(alphabetType); err != nil {
		return nil, err
	}
	if err := s.validateLocale(locale); err != nil {
		return nil, err
	}
	if character == "" {
		return nil, fmt.Errorf("character parameter is required")
	}

	characters, err := s.repo.GetByRowColumn(ctx, alphabetType, locale, character)
	if err != nil {
		s.logger.Error("failed to get characters by row/column", zap.Error(err))
		return nil, fmt.Errorf("failed to get characters: %w", err)
	}

	return characters, nil
}

// GetByID retrieves a character by its ID
//
// For successful results localeParam must be either "ru" (Russian) or "en" (English).
func (s *charactersService) GetByID(ctx context.Context, id int, localeParam string) (*models.Character, error) {
	locale := models.Locale(localeParam)

	if id <= 0 {
		return nil, fmt.Errorf("invalid character id")
	}
	if err := s.validateLocale(locale); err != nil {
		return nil, err
	}

	character, err := s.repo.GetByID(ctx, id, locale)
	if err != nil {
		s.logger.Error("failed to get character by id", zap.Error(err), zap.Int("id", id))
		return nil, fmt.Errorf("failed to get character: %w", err)
	}

	return character, nil
}

// GetReadingTest retrieves random characters for reading test
//
// For successful results alphabetTypeStr must be either "hiragana" or "katakana" (from URL path).
// localeParam must be either "ru" (Russian) or "en" (English).
func (s *charactersService) GetReadingTest(ctx context.Context, alphabetTypeStr string, localeParam string) ([]models.ReadingTestItem, error) {
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

	if err := s.validateLocale(locale); err != nil {
		return nil, err
	}

	items, err := s.repo.GetRandomForReadingTest(ctx, at, locale, testCount)
	if err != nil {
		s.logger.Error("failed to get reading test items", zap.Error(err))
		return nil, fmt.Errorf("failed to get reading test: %w", err)
	}

	return items, nil
}

// GetWritingTest retrieves random characters for writing test
//
// For successful results alphabetTypeStr must be either "hiragana" or "katakana" (from URL path).
// locale must be either "en" (English) or "ru" (Russian).
func (s *charactersService) GetWritingTest(ctx context.Context, alphabetTypeStr string, localeParam string) ([]models.WritingTestItem, error) {
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

	if err := s.validateLocale(locale); err != nil {
		return nil, err
	}

	items, err := s.repo.GetRandomForWritingTest(ctx, at, locale, testCount)
	if err != nil {
		s.logger.Error("failed to get writing test items", zap.Error(err))
		return nil, fmt.Errorf("failed to get writing test: %w", err)
	}

	return items, nil
}

// validateAlphabetType validates the alphabet type
func (s *charactersService) validateAlphabetType(at models.AlphabetType) error {
	if at != models.AlphabetTypeHiragana && at != models.AlphabetTypeKatakana {
		return fmt.Errorf("invalid alphabet type: %s, must be 'hr' or 'kt'", at)
	}
	return nil
}

// validateLocale validates the locale
func (s *charactersService) validateLocale(locale models.Locale) error {
	if locale != models.LocaleEnglish && locale != models.LocaleRussian {
		return fmt.Errorf("invalid locale: %s, must be 'en' or 'ru'", locale)
	}
	return nil
}
