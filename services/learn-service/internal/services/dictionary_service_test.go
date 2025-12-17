package services

import (
	"context"
	"errors"
	"testing"

	"github.com/japanesestudent/learn-service/internal/models"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

// mockWordRepository is a mock implementation of WordRepository
type mockWordRepository struct {
	words      []models.WordResponse
	valid      bool
	err        error
	validateErr error
}

func (m *mockWordRepository) GetByIDs(ctx context.Context, wordIds []int, translationField, exampleTranslationField string) ([]models.WordResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.words, nil
}

func (m *mockWordRepository) GetExcludingIDs(ctx context.Context, excludeIds []int, limit int, translationField, exampleTranslationField string) ([]models.WordResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.words, nil
}

func (m *mockWordRepository) ValidateWordIDs(ctx context.Context, wordIds []int) (bool, error) {
	if m.validateErr != nil {
		return false, m.validateErr
	}
	return m.valid, nil
}

// mockDictionaryHistoryRepository is a mock implementation of DictionaryHistoryRepository
type mockDictionaryHistoryRepository struct {
	oldWordIds []int
	err        error
	upsertErr  error
}

func (m *mockDictionaryHistoryRepository) GetOldWordIds(ctx context.Context, userId int, limit int) ([]int, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.oldWordIds, nil
}

func (m *mockDictionaryHistoryRepository) UpsertResults(ctx context.Context, userId int, results []models.WordResult) error {
	return m.upsertErr
}

func TestNewDictionaryService(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	wordRepo := &mockWordRepository{}
	historyRepo := &mockDictionaryHistoryRepository{}

	svc := NewDictionaryService(wordRepo, historyRepo, logger)

	assert.NotNil(t, svc)
	assert.Equal(t, wordRepo, svc.wordRepo)
	assert.Equal(t, historyRepo, svc.dictionaryHistoryRepo)
	assert.Equal(t, logger, svc.logger)
}

func TestDictionaryService_GetWordList(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	tests := []struct {
		name          string
		userId        int
		newCount      int
		oldCount      int
		locale        string
		wordRepo      *mockWordRepository
		historyRepo   *mockDictionaryHistoryRepository
		expectedError bool
		errorContains string
		expectedCount int
	}{
		{
			name:     "success with english locale",
			userId:   1,
			newCount: 20,
			oldCount: 20,
			locale:   "en",
			wordRepo: &mockWordRepository{
				words: []models.WordResponse{
					{ID: 1, Word: "水", Translation: "water"},
					{ID: 2, Word: "火", Translation: "fire"},
				},
			},
			historyRepo: &mockDictionaryHistoryRepository{
				oldWordIds: []int{3, 4},
			},
			expectedError: false,
			expectedCount: 2,
		},
		{
			name:     "success with russian locale",
			userId:   1,
			newCount: 15,
			oldCount: 15,
			locale:   "ru",
			wordRepo: &mockWordRepository{
				words: []models.WordResponse{
					{ID: 1, Word: "水", Translation: "вода"},
				},
			},
			historyRepo: &mockDictionaryHistoryRepository{
				oldWordIds: []int{2},
			},
			expectedError: false,
			expectedCount: 1,
		},
		{
			name:     "success with german locale",
			userId:   1,
			newCount: 10,
			oldCount: 10,
			locale:   "de",
			wordRepo: &mockWordRepository{
				words: []models.WordResponse{
					{ID: 1, Word: "水", Translation: "Wasser"},
				},
			},
			historyRepo: &mockDictionaryHistoryRepository{
				oldWordIds: []int{},
			},
			expectedError: false,
			expectedCount: 1,
		},
		{
			name:     "invalid newCount - too low",
			userId:   1,
			newCount: 5,
			oldCount: 20,
			locale:   "en",
			wordRepo: &mockWordRepository{},
			historyRepo: &mockDictionaryHistoryRepository{},
			expectedError: true,
			errorContains: "newWordCount must be between 10 and 40",
		},
		{
			name:     "invalid newCount - too high",
			userId:   1,
			newCount: 50,
			oldCount: 20,
			locale:   "en",
			wordRepo: &mockWordRepository{},
			historyRepo: &mockDictionaryHistoryRepository{},
			expectedError: true,
			errorContains: "newWordCount must be between 10 and 40",
		},
		{
			name:     "invalid oldCount - too low",
			userId:   1,
			newCount: 20,
			oldCount: 5,
			locale:   "en",
			wordRepo: &mockWordRepository{},
			historyRepo: &mockDictionaryHistoryRepository{},
			expectedError: true,
			errorContains: "oldWordCount must be between 10 and 40",
		},
		{
			name:     "invalid oldCount - too high",
			userId:   1,
			newCount: 20,
			oldCount: 50,
			locale:   "en",
			wordRepo: &mockWordRepository{},
			historyRepo: &mockDictionaryHistoryRepository{},
			expectedError: true,
			errorContains: "oldWordCount must be between 10 and 40",
		},
		{
			name:     "invalid locale",
			userId:   1,
			newCount: 20,
			oldCount: 20,
			locale:   "fr",
			wordRepo: &mockWordRepository{},
			historyRepo: &mockDictionaryHistoryRepository{},
			expectedError: true,
			errorContains: "invalid locale",
		},
		{
			name:     "boundary values - newCount at minimum",
			userId:   1,
			newCount: 10,
			oldCount: 20,
			locale:   "en",
			wordRepo: &mockWordRepository{
				words: []models.WordResponse{{ID: 1, Word: "水"}},
			},
			historyRepo: &mockDictionaryHistoryRepository{
				oldWordIds: []int{},
			},
			expectedError: false,
			expectedCount: 1,
		},
		{
			name:     "boundary values - newCount at maximum",
			userId:   1,
			newCount: 40,
			oldCount: 20,
			locale:   "en",
			wordRepo: &mockWordRepository{
				words: []models.WordResponse{{ID: 1, Word: "水"}},
			},
			historyRepo: &mockDictionaryHistoryRepository{
				oldWordIds: []int{},
			},
			expectedError: false,
			expectedCount: 1,
		},
		{
			name:     "database error on get old word IDs",
			userId:   1,
			newCount: 20,
			oldCount: 20,
			locale:   "en",
			wordRepo: &mockWordRepository{},
			historyRepo: &mockDictionaryHistoryRepository{
				err: errors.New("database error"),
			},
			expectedError: true,
			errorContains: "failed to get old word IDs",
		},
		{
			name:     "database error on get old words",
			userId:   1,
			newCount: 20,
			oldCount: 20,
			locale:   "en",
			wordRepo: &mockWordRepository{
				err: errors.New("database error"),
			},
			historyRepo: &mockDictionaryHistoryRepository{
				oldWordIds: []int{1, 2},
			},
			expectedError: true,
		},
		{
			name:     "database error on get new words",
			userId:   1,
			newCount: 20,
			oldCount: 20,
			locale:   "en",
			wordRepo: &mockWordRepository{
				err: errors.New("database error"),
			},
			historyRepo: &mockDictionaryHistoryRepository{
				oldWordIds: []int{},
			},
			expectedError: true,
		},
		{
			name:     "success with empty old word IDs",
			userId:   1,
			newCount: 20,
			oldCount: 20,
			locale:   "en",
			wordRepo: &mockWordRepository{
				words: []models.WordResponse{
					{ID: 1, Word: "水", Translation: "water"},
					{ID: 2, Word: "火", Translation: "fire"},
				},
			},
			historyRepo: &mockDictionaryHistoryRepository{
				oldWordIds: []int{},
			},
			expectedError: false,
			expectedCount: 2,
		},
		{
			name:     "success with mixed old and new words",
			userId:   1,
			newCount: 10,
			oldCount: 10,
			locale:   "ru",
			wordRepo: &mockWordRepository{
				words: []models.WordResponse{
					{ID: 1, Word: "水", Translation: "вода"},
					{ID: 2, Word: "火", Translation: "огонь"},
				},
			},
			historyRepo: &mockDictionaryHistoryRepository{
				oldWordIds: []int{3, 4},
			},
			expectedError: false,
			expectedCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewDictionaryService(tt.wordRepo, tt.historyRepo, logger)

			result, err := svc.GetWordList(context.Background(), tt.userId, tt.newCount, tt.oldCount, tt.locale)

			if tt.expectedError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.LessOrEqual(t, len(result), tt.expectedCount*2) // Can be up to old + new
			}
		})
	}
}

func TestDictionaryService_SubmitWordResults(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	tests := []struct {
		name          string
		userId        int
		results       []models.WordResult
		wordRepo      *mockWordRepository
		historyRepo   *mockDictionaryHistoryRepository
		expectedError bool
		errorContains string
	}{
		{
			name:   "success with valid results",
			userId: 1,
			results: []models.WordResult{
				{WordID: 1, Period: 3},
				{WordID: 2, Period: 7},
			},
			wordRepo: &mockWordRepository{
				valid: true,
			},
			historyRepo: &mockDictionaryHistoryRepository{},
			expectedError: false,
		},
		{
			name:          "empty results list",
			userId:        1,
			results:       []models.WordResult{},
			wordRepo:      &mockWordRepository{},
			historyRepo:   &mockDictionaryHistoryRepository{},
			expectedError: true,
			errorContains: "results list cannot be empty",
		},
		{
			name:   "invalid period - too low",
			userId: 1,
			results: []models.WordResult{
				{WordID: 1, Period: 0},
			},
			wordRepo:    &mockWordRepository{},
			historyRepo: &mockDictionaryHistoryRepository{},
			expectedError: true,
			errorContains: "period must be between 1 and 30",
		},
		{
			name:   "invalid period - too high",
			userId: 1,
			results: []models.WordResult{
				{WordID: 1, Period: 31},
			},
			wordRepo:    &mockWordRepository{},
			historyRepo: &mockDictionaryHistoryRepository{},
			expectedError: true,
			errorContains: "period must be between 1 and 30",
		},
		{
			name:   "boundary values - period at minimum",
			userId: 1,
			results: []models.WordResult{
				{WordID: 1, Period: 1},
			},
			wordRepo: &mockWordRepository{
				valid: true,
			},
			historyRepo: &mockDictionaryHistoryRepository{},
			expectedError: false,
		},
		{
			name:   "boundary values - period at maximum",
			userId: 1,
			results: []models.WordResult{
				{WordID: 1, Period: 30},
			},
			wordRepo: &mockWordRepository{
				valid: true,
			},
			historyRepo: &mockDictionaryHistoryRepository{},
			expectedError: false,
		},
		{
			name:   "database error on validate word IDs",
			userId: 1,
			results: []models.WordResult{
				{WordID: 1, Period: 3},
			},
			wordRepo: &mockWordRepository{
				validateErr: errors.New("database error"),
			},
			historyRepo: &mockDictionaryHistoryRepository{},
			expectedError: true,
			errorContains: "failed to validate word IDs",
		},
		{
			name:   "one or more word IDs do not exist",
			userId: 1,
			results: []models.WordResult{
				{WordID: 1, Period: 3},
				{WordID: 999, Period: 7},
			},
			wordRepo: &mockWordRepository{
				valid: false,
			},
			historyRepo: &mockDictionaryHistoryRepository{},
			expectedError: true,
			errorContains: "one or more word IDs do not exist",
		},
		{
			name:   "database error on upsert",
			userId: 1,
			results: []models.WordResult{
				{WordID: 1, Period: 3},
			},
			wordRepo: &mockWordRepository{
				valid: true,
			},
			historyRepo: &mockDictionaryHistoryRepository{
				upsertErr: errors.New("upsert error"),
			},
			expectedError: true,
		},
		{
			name:   "success with multiple results and different periods",
			userId: 1,
			results: []models.WordResult{
				{WordID: 1, Period: 1},
				{WordID: 2, Period: 3},
				{WordID: 3, Period: 7},
				{WordID: 4, Period: 14},
				{WordID: 5, Period: 30},
			},
			wordRepo: &mockWordRepository{
				valid: true,
			},
			historyRepo: &mockDictionaryHistoryRepository{},
			expectedError: false,
		},
		{
			name:   "mixed valid and invalid periods - should fail on first invalid",
			userId: 1,
			results: []models.WordResult{
				{WordID: 1, Period: 3},
				{WordID: 2, Period: 0}, // Invalid
			},
			wordRepo:    &mockWordRepository{},
			historyRepo: &mockDictionaryHistoryRepository{},
			expectedError: true,
			errorContains: "period must be between 1 and 30",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewDictionaryService(tt.wordRepo, tt.historyRepo, logger)

			err := svc.SubmitWordResults(context.Background(), tt.userId, tt.results)

			if tt.expectedError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

