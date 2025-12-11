package services

import (
	"context"
	"errors"
	"testing"

	"github.com/japanesestudent/backend/internal/models"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

// mockRepository is a mock implementation of Repository
type mockRepository struct {
	characters          []models.CharacterResponse
	character           *models.Character
	readingTestItems    []models.ReadingTestItem
	writingTestItems    []models.WritingTestItem
	err                 error
	rowColumnCharacters []models.CharacterResponse
}

func (m *mockRepository) GetAll(ctx context.Context, alphabetType models.AlphabetType, locale models.Locale) ([]models.CharacterResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.characters, nil
}

func (m *mockRepository) GetByRowColumn(ctx context.Context, alphabetType models.AlphabetType, locale models.Locale, character string) ([]models.CharacterResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.rowColumnCharacters, nil
}

func (m *mockRepository) GetByID(ctx context.Context, id int, locale models.Locale) (*models.Character, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.character, nil
}

func (m *mockRepository) GetRandomForReadingTest(ctx context.Context, alphabetType models.AlphabetType, locale models.Locale, count int) ([]models.ReadingTestItem, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.readingTestItems, nil
}

func (m *mockRepository) GetRandomForWritingTest(ctx context.Context, alphabetType models.AlphabetType, locale models.Locale, count int) ([]models.WritingTestItem, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.writingTestItems, nil
}

func TestNewCharactersService(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	mockRepo := &mockRepository{}

	svc := NewCharactersService(mockRepo, logger)

	assert.NotNil(t, svc)
	assert.Equal(t, mockRepo, svc.repo)
	assert.Equal(t, logger, svc.logger)
}

func TestService_GetAll(t *testing.T) {
	tests := []struct {
		name          string
		alphabetType  string
		locale        string
		mockRepo      *mockRepository
		expectedError bool
		expectedCount int
	}{
		{
			name:         "success hiragana english",
			alphabetType: "hr",
			locale:       "en",
			mockRepo: &mockRepository{
				characters: []models.CharacterResponse{
					{ID: 1, Character: "あ", Reading: "a"},
					{ID: 2, Character: "い", Reading: "i"},
				},
			},
			expectedError: false,
			expectedCount: 2,
		},
		{
			name:         "success katakana russian",
			alphabetType: "kt",
			locale:       "ru",
			mockRepo: &mockRepository{
				characters: []models.CharacterResponse{
					{ID: 1, Character: "ア", Reading: "а"},
					{ID: 2, Character: "イ", Reading: "и"},
				},
			},
			expectedError: false,
			expectedCount: 2,
		},
		{
			name:          "invalid alphabet type",
			alphabetType:  "invalid",
			locale:        "en",
			mockRepo:      &mockRepository{},
			expectedError: true,
			expectedCount: 0,
		},
		{
			name:          "invalid locale",
			alphabetType:  "hr",
			locale:        "invalid",
			mockRepo:      &mockRepository{},
			expectedError: true,
			expectedCount: 0,
		},
		{
			name:         "repository error",
			alphabetType: "hr",
			locale:       "en",
			mockRepo: &mockRepository{
				err: errors.New("database error"),
			},
			expectedError: true,
			expectedCount: 0,
		},
		{
			name:         "empty result",
			alphabetType: "hr",
			locale:       "en",
			mockRepo: &mockRepository{
				characters: []models.CharacterResponse{},
			},
			expectedError: false,
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, _ := zap.NewDevelopment()
			svc := NewCharactersService(tt.mockRepo, logger)
			ctx := context.Background()

			result, err := svc.GetAll(ctx, tt.alphabetType, tt.locale)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Len(t, result, tt.expectedCount)
			}
		})
	}
}

func TestService_GetByRowColumn(t *testing.T) {
	tests := []struct {
		name          string
		alphabetType  string
		locale        string
		character     string
		mockRepo      *mockRepository
		expectedError bool
		expectedCount int
	}{
		{
			name:         "success hiragana english with vowel",
			alphabetType: "hr",
			locale:       "en",
			character:    "a",
			mockRepo: &mockRepository{
				rowColumnCharacters: []models.CharacterResponse{
					{ID: 1, Character: "あ", Reading: "a", Vowel: "a"},
					{ID: 2, Character: "か", Reading: "ka", Vowel: "a"},
				},
			},
			expectedError: false,
			expectedCount: 2,
		},
		{
			name:         "success katakana russian with consonant",
			alphabetType: "kt",
			locale:       "ru",
			character:    "k",
			mockRepo: &mockRepository{
				rowColumnCharacters: []models.CharacterResponse{
					{ID: 1, Character: "カ", Reading: "ка", Consonant: "k"},
					{ID: 2, Character: "キ", Reading: "ки", Consonant: "k"},
				},
			},
			expectedError: false,
			expectedCount: 2,
		},
		{
			name:          "invalid alphabet type",
			alphabetType:  "invalid",
			locale:        "en",
			character:     "a",
			mockRepo:      &mockRepository{},
			expectedError: true,
			expectedCount: 0,
		},
		{
			name:          "invalid locale",
			alphabetType:  "hr",
			locale:        "invalid",
			character:     "a",
			mockRepo:      &mockRepository{},
			expectedError: true,
			expectedCount: 0,
		},
		{
			name:          "empty character parameter",
			alphabetType:  "hr",
			locale:        "en",
			character:     "",
			mockRepo:      &mockRepository{},
			expectedError: true,
			expectedCount: 0,
		},
		{
			name:         "repository error",
			alphabetType: "hr",
			locale:       "en",
			character:    "a",
			mockRepo: &mockRepository{
				err: errors.New("database error"),
			},
			expectedError: true,
			expectedCount: 0,
		},
		{
			name:         "empty result",
			alphabetType: "hr",
			locale:       "en",
			character:    "x",
			mockRepo: &mockRepository{
				rowColumnCharacters: []models.CharacterResponse{},
			},
			expectedError: false,
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, _ := zap.NewDevelopment()
			svc := NewCharactersService(tt.mockRepo, logger)
			ctx := context.Background()

			result, err := svc.GetByRowColumn(ctx, tt.alphabetType, tt.locale, tt.character)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Len(t, result, tt.expectedCount)
			}
		})
	}
}

func TestService_GetByID(t *testing.T) {
	tests := []struct {
		name          string
		id            int
		locale        string
		mockRepo      *mockRepository
		expectedError bool
		expectedID    int
	}{
		{
			name:   "success english locale",
			id:     1,
			locale: "en",
			mockRepo: &mockRepository{
				character: &models.Character{
					ID:             1,
					Consonant:      "",
					Vowel:          "a",
					EnglishReading: "a",
					Katakana:       "ア",
					Hiragana:       "あ",
				},
			},
			expectedError: false,
			expectedID:    1,
		},
		{
			name:   "success russian locale",
			id:     2,
			locale: "ru",
			mockRepo: &mockRepository{
				character: &models.Character{
					ID:             2,
					Consonant:      "k",
					Vowel:          "a",
					RussianReading: "ка",
					Katakana:       "カ",
					Hiragana:       "か",
				},
			},
			expectedError: false,
			expectedID:    2,
		},
		{
			name:          "invalid id zero",
			id:            0,
			locale:        "en",
			mockRepo:      &mockRepository{},
			expectedError: true,
			expectedID:    0,
		},
		{
			name:          "invalid id negative",
			id:            -1,
			locale:        "en",
			mockRepo:      &mockRepository{},
			expectedError: true,
			expectedID:    0,
		},
		{
			name:          "invalid locale",
			id:            1,
			locale:        "invalid",
			mockRepo:      &mockRepository{},
			expectedError: true,
			expectedID:    0,
		},
		{
			name:   "repository error not found",
			id:     999,
			locale: "en",
			mockRepo: &mockRepository{
				err: errors.New("character not found"),
			},
			expectedError: true,
			expectedID:    0,
		},
		{
			name:   "repository error database",
			id:     1,
			locale: "en",
			mockRepo: &mockRepository{
				err: errors.New("database error"),
			},
			expectedError: true,
			expectedID:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, _ := zap.NewDevelopment()
			svc := NewCharactersService(tt.mockRepo, logger)
			ctx := context.Background()

			result, err := svc.GetByID(ctx, tt.id, tt.locale)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.expectedID, result.ID)
			}
		})
	}
}

func TestService_GetReadingTest(t *testing.T) {
	tests := []struct {
		name          string
		alphabetType  string
		locale        string
		mockRepo      *mockRepository
		expectedError bool
		expectedCount int
	}{
		{
			name:         "success hiragana english",
			alphabetType: "hiragana",
			locale:       "en",
			mockRepo: &mockRepository{
				readingTestItems: []models.ReadingTestItem{
					{ID: 1, CorrectChar: "あ", Reading: "a", WrongOptions: []string{"い", "う"}},
					{ID: 2, CorrectChar: "い", Reading: "i", WrongOptions: []string{"う", "え"}},
				},
			},
			expectedError: false,
			expectedCount: 2,
		},
		{
			name:         "success katakana russian",
			alphabetType: "katakana",
			locale:       "ru",
			mockRepo: &mockRepository{
				readingTestItems: []models.ReadingTestItem{
					{ID: 1, CorrectChar: "ア", Reading: "а", WrongOptions: []string{"イ", "ウ"}},
				},
			},
			expectedError: false,
			expectedCount: 1,
		},
		{
			name:          "invalid alphabet type",
			alphabetType:  "invalid",
			locale:        "en",
			mockRepo:      &mockRepository{},
			expectedError: true,
			expectedCount: 0,
		},
		{
			name:          "invalid locale",
			alphabetType:  "hiragana",
			locale:        "invalid",
			mockRepo:      &mockRepository{},
			expectedError: true,
			expectedCount: 0,
		},
		{
			name:         "repository error",
			alphabetType: "hiragana",
			locale:       "en",
			mockRepo: &mockRepository{
				err: errors.New("database error"),
			},
			expectedError: true,
			expectedCount: 0,
		},
		{
			name:         "empty result",
			alphabetType: "hiragana",
			locale:       "en",
			mockRepo: &mockRepository{
				readingTestItems: []models.ReadingTestItem{},
			},
			expectedError: false,
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, _ := zap.NewDevelopment()
			svc := NewCharactersService(tt.mockRepo, logger)
			ctx := context.Background()

			result, err := svc.GetReadingTest(ctx, tt.alphabetType, tt.locale)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Len(t, result, tt.expectedCount)
			}
		})
	}
}

func TestService_GetWritingTest(t *testing.T) {
	tests := []struct {
		name          string
		alphabetType  string
		locale        string
		mockRepo      *mockRepository
		expectedError bool
		expectedCount int
	}{
		{
			name:         "success hiragana english",
			alphabetType: "hiragana",
			locale:       "en",
			mockRepo: &mockRepository{
				writingTestItems: []models.WritingTestItem{
					{ID: 1, CorrectReading: "a", Character: "あ"},
					{ID: 2, CorrectReading: "i", Character: "い"},
				},
			},
			expectedError: false,
			expectedCount: 2,
		},
		{
			name:         "success katakana russian",
			alphabetType: "katakana",
			locale:       "ru",
			mockRepo: &mockRepository{
				writingTestItems: []models.WritingTestItem{
					{ID: 1, CorrectReading: "а", Character: "ア"},
					{ID: 2, CorrectReading: "и", Character: "イ"},
					{ID: 3, CorrectReading: "у", Character: "ウ"},
				},
			},
			expectedError: false,
			expectedCount: 3,
		},
		{
			name:          "invalid alphabet type",
			alphabetType:  "invalid",
			locale:        "en",
			mockRepo:      &mockRepository{},
			expectedError: true,
			expectedCount: 0,
		},
		{
			name:          "invalid locale",
			alphabetType:  "hiragana",
			locale:        "invalid",
			mockRepo:      &mockRepository{},
			expectedError: true,
			expectedCount: 0,
		},
		{
			name:         "repository error",
			alphabetType: "hiragana",
			locale:       "en",
			mockRepo: &mockRepository{
				err: errors.New("database error"),
			},
			expectedError: true,
			expectedCount: 0,
		},
		{
			name:         "empty result",
			alphabetType: "hiragana",
			locale:       "en",
			mockRepo: &mockRepository{
				writingTestItems: []models.WritingTestItem{},
			},
			expectedError: false,
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, _ := zap.NewDevelopment()
			svc := NewCharactersService(tt.mockRepo, logger)
			ctx := context.Background()

			result, err := svc.GetWritingTest(ctx, tt.alphabetType, tt.locale)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Len(t, result, tt.expectedCount)
			}
		})
	}
}

func TestService_ValidateAlphabetType(t *testing.T) {
	svc := &charactersService{}

	tests := []struct {
		name          string
		alphabetType  models.AlphabetType
		expectedError bool
	}{
		{
			name:          "valid hiragana",
			alphabetType:  models.AlphabetTypeHiragana,
			expectedError: false,
		},
		{
			name:          "valid katakana",
			alphabetType:  models.AlphabetTypeKatakana,
			expectedError: false,
		},
		{
			name:          "invalid alphabet type",
			alphabetType:  "invalid",
			expectedError: true,
		},
		{
			name:          "empty alphabet type",
			alphabetType:  "",
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := svc.validateAlphabetType(tt.alphabetType)
			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestService_ValidateLocale(t *testing.T) {
	svc := &charactersService{}

	tests := []struct {
		name          string
		locale        models.Locale
		expectedError bool
	}{
		{
			name:          "valid english",
			locale:        models.LocaleEnglish,
			expectedError: false,
		},
		{
			name:          "valid russian",
			locale:        models.LocaleRussian,
			expectedError: false,
		},
		{
			name:          "invalid locale",
			locale:        "invalid",
			expectedError: true,
		},
		{
			name:          "empty locale",
			locale:        "",
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := svc.validateLocale(tt.locale)
			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
