package services

import (
	"context"
	"errors"
	"testing"

	"github.com/japanesestudent/learn-service/internal/models"
	"github.com/stretchr/testify/assert"
)

// mockAdminWordRepository is a mock implementation of AdminWordRepository
type mockAdminWordRepository struct {
	words        []models.Word
	word         *models.Word
	existsByWord bool
	existsByClues bool
	err          error
	createErr    error
	updateErr    error
	deleteErr    error
}

func (m *mockAdminWordRepository) GetAllForAdmin(ctx context.Context, page, count int, search string) ([]models.Word, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.words, nil
}

func (m *mockAdminWordRepository) GetByIDAdmin(ctx context.Context, id int) (*models.Word, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.word, nil
}

func (m *mockAdminWordRepository) ExistsByWord(ctx context.Context, word string) (bool, error) {
	if m.err != nil {
		return false, m.err
	}
	return m.existsByWord, nil
}

func (m *mockAdminWordRepository) ExistsByClues(ctx context.Context, clues string) (bool, error) {
	if m.err != nil {
		return false, m.err
	}
	return m.existsByClues, nil
}

func (m *mockAdminWordRepository) Create(ctx context.Context, word *models.Word) error {
	if m.createErr != nil {
		return m.createErr
	}
	if m.err != nil {
		return m.err
	}
	word.ID = 1
	return nil
}

func (m *mockAdminWordRepository) Update(ctx context.Context, id int, word *models.Word) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	return m.err
}

func (m *mockAdminWordRepository) Delete(ctx context.Context, id int) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	return m.err
}

func TestNewAdminWordService(t *testing.T) {
	mockWordRepo := &mockAdminWordRepository{}
	mockHistoryRepo := &mockDictionaryHistoryRepository{}

	svc := NewAdminWordService(mockWordRepo, mockHistoryRepo)

	assert.NotNil(t, svc)
	assert.Equal(t, mockWordRepo, svc.wordRepo)
	assert.Equal(t, mockHistoryRepo, svc.dictionaryHistoryRepo)
}

func TestAdminWordService_GetAllForAdmin(t *testing.T) {
	tests := []struct {
		name          string
		page          int
		count         int
		search        string
		mockRepo      *mockAdminWordRepository
		expectedError bool
		expectedCount int
	}{
		{
			name:   "success with defaults",
			page:   0,
			count:  0,
			search: "",
			mockRepo: &mockAdminWordRepository{
				words: []models.Word{
					{ID: 1, Word: "水", PhoneticClues: "みず", EnglishTranslation: "water"},
					{ID: 2, Word: "火", PhoneticClues: "ひ", EnglishTranslation: "fire"},
				},
			},
			expectedError: false,
			expectedCount: 2,
		},
		{
			name:   "success with pagination",
			page:   2,
			count:  10,
			search: "",
			mockRepo: &mockAdminWordRepository{
				words: []models.Word{
					{ID: 1, Word: "水", PhoneticClues: "みず", EnglishTranslation: "water"},
				},
			},
			expectedError: false,
			expectedCount: 1,
		},
		{
			name:   "success with search",
			page:   1,
			count:  20,
			search: "water",
			mockRepo: &mockAdminWordRepository{
				words: []models.Word{
					{ID: 1, Word: "水", PhoneticClues: "みず", EnglishTranslation: "water"},
				},
			},
			expectedError: false,
			expectedCount: 1,
		},
		{
			name:   "repository error",
			page:   1,
			count:  20,
			search: "",
			mockRepo: &mockAdminWordRepository{
				err: errors.New("database error"),
			},
			expectedError: true,
			expectedCount: 0,
		},
		{
			name:   "empty result",
			page:   1,
			count:  20,
			search: "",
			mockRepo: &mockAdminWordRepository{
				words: []models.Word{},
			},
			expectedError: false,
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewAdminWordService(tt.mockRepo, &mockDictionaryHistoryRepository{})
			ctx := context.Background()

			result, err := svc.GetAllForAdmin(ctx, tt.page, tt.count, tt.search)

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

func TestAdminWordService_GetByIDAdmin(t *testing.T) {
	tests := []struct {
		name          string
		id            int
		mockRepo      *mockAdminWordRepository
		expectedError bool
		expectedID    int
	}{
		{
			name: "success",
			id:   1,
			mockRepo: &mockAdminWordRepository{
				word: &models.Word{
					ID:                1,
					Word:              "水",
					PhoneticClues:     "みず",
					EnglishTranslation: "water",
				},
			},
			expectedError: false,
			expectedID:    1,
		},
		{
			name:          "invalid id zero",
			id:            0,
			mockRepo:      &mockAdminWordRepository{},
			expectedError: true,
			expectedID:    0,
		},
		{
			name:          "invalid id negative",
			id:            -1,
			mockRepo:      &mockAdminWordRepository{},
			expectedError: true,
			expectedID:    0,
		},
		{
			name: "repository error",
			id:   1,
			mockRepo: &mockAdminWordRepository{
				err: errors.New("word not found"),
			},
			expectedError: true,
			expectedID:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewAdminWordService(tt.mockRepo, &mockDictionaryHistoryRepository{})
			ctx := context.Background()

			result, err := svc.GetByIDAdmin(ctx, tt.id)

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

func TestAdminWordService_CreateWord(t *testing.T) {
	tests := []struct {
		name          string
		request       *models.CreateWordRequest
		mockRepo      *mockAdminWordRepository
		expectedError bool
		expectedID    int
		errorContains string
	}{
		{
			name: "success",
			request: &models.CreateWordRequest{
				Word:               "水",
				PhoneticClues:       "みず",
				EnglishTranslation: "water",
			},
			mockRepo: &mockAdminWordRepository{
				existsByWord: false,
			},
			expectedError: false,
			expectedID:    1,
		},
		{
			name: "word already exists",
			request: &models.CreateWordRequest{
				Word: "水",
			},
			mockRepo: &mockAdminWordRepository{
				existsByWord: true,
			},
			expectedError: true,
			expectedID:    0,
			errorContains: "already exists",
		},
		{
			name: "failed to check word existence",
			request: &models.CreateWordRequest{
				Word: "水",
			},
			mockRepo: &mockAdminWordRepository{
				err: errors.New("database error"),
			},
			expectedError: true,
			expectedID:    0,
			errorContains: "failed to check word existence",
		},
		{
			name: "failed to create",
			request: &models.CreateWordRequest{
				Word: "水",
			},
			mockRepo: &mockAdminWordRepository{
				existsByWord: false,
				createErr:    errors.New("create error"),
			},
			expectedError: true,
			expectedID:    0,
			errorContains: "failed to create word",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewAdminWordService(tt.mockRepo, &mockDictionaryHistoryRepository{})
			ctx := context.Background()

			result, err := svc.CreateWord(ctx, tt.request)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Equal(t, 0, result)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedID, result)
			}
		})
	}
}

func TestAdminWordService_UpdateWord(t *testing.T) {
	tests := []struct {
		name          string
		id            int
		request       *models.UpdateWordRequest
		mockRepo      *mockAdminWordRepository
		expectedError bool
		errorContains string
	}{
		{
			name: "success partial update",
			id:   1,
			request: &models.UpdateWordRequest{
				EnglishTranslation: "updated water",
			},
			mockRepo: &mockAdminWordRepository{},
			expectedError: false,
		},
		{
			name: "success update with word field - word exists",
			id:   1,
			request: &models.UpdateWordRequest{
				Word: "新",
			},
			mockRepo: &mockAdminWordRepository{
				existsByWord: true,
			},
			expectedError: true,
			errorContains: "already exists",
		},
		{
			name: "success update with clues field - clues exists",
			id:   1,
			request: &models.UpdateWordRequest{
				PhoneticClues: "あたらしい",
			},
			mockRepo: &mockAdminWordRepository{
				existsByClues: true,
			},
			expectedError: true,
			errorContains: "already exists",
		},
		{
			name: "invalid period - easy too low",
			id:   1,
			request: &models.UpdateWordRequest{
				EasyPeriod: intPtr(0),
			},
			mockRepo:      &mockAdminWordRepository{},
			expectedError: true,
			errorContains: "easy period must be between 1 and 30",
		},
		{
			name: "invalid period - easy too high",
			id:   1,
			request: &models.UpdateWordRequest{
				EasyPeriod: intPtr(31),
			},
			mockRepo:      &mockAdminWordRepository{},
			expectedError: true,
			errorContains: "easy period must be between 1 and 30",
		},
		{
			name: "invalid period - normal too low",
			id:   1,
			request: &models.UpdateWordRequest{
				NormalPeriod: intPtr(0),
			},
			mockRepo:      &mockAdminWordRepository{},
			expectedError: true,
			errorContains: "normal period must be between 1 and 30",
		},
		{
			name: "invalid period - hard too high",
			id:   1,
			request: &models.UpdateWordRequest{
				HardPeriod: intPtr(31),
			},
			mockRepo:      &mockAdminWordRepository{},
			expectedError: true,
			errorContains: "hard period must be between 1 and 30",
		},
		{
			name: "invalid period - extra hard too low",
			id:   1,
			request: &models.UpdateWordRequest{
				ExtraHardPeriod: intPtr(0),
			},
			mockRepo:      &mockAdminWordRepository{},
			expectedError: true,
			errorContains: "extra hard period must be between 1 and 30",
		},
		{
			name:          "invalid id zero",
			id:            0,
			request:       &models.UpdateWordRequest{},
			mockRepo:      &mockAdminWordRepository{},
			expectedError: true,
			errorContains: "invalid word id",
		},
		{
			name: "failed to check word existence",
			id:   1,
			request: &models.UpdateWordRequest{
				Word: "新",
			},
			mockRepo: &mockAdminWordRepository{
				err: errors.New("database error"),
			},
			expectedError: true,
			errorContains: "failed to check word existence",
		},
		{
			name: "failed to check clues existence",
			id:   1,
			request: &models.UpdateWordRequest{
				PhoneticClues: "あたらしい",
			},
			mockRepo: &mockAdminWordRepository{
				err: errors.New("database error"),
			},
			expectedError: true,
			errorContains: "failed to check clues existence",
		},
		{
			name: "failed to update",
			id:   1,
			request: &models.UpdateWordRequest{
				EnglishTranslation: "updated",
			},
			mockRepo: &mockAdminWordRepository{
				updateErr: errors.New("update error"),
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewAdminWordService(tt.mockRepo, &mockDictionaryHistoryRepository{})
			ctx := context.Background()

			err := svc.UpdateWord(ctx, tt.id, tt.request)

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

func TestAdminWordService_DeleteWord(t *testing.T) {
	tests := []struct {
		name          string
		id            int
		mockRepo      *mockAdminWordRepository
		expectedError bool
		errorContains string
	}{
		{
			name:          "success",
			id:            1,
			mockRepo:      &mockAdminWordRepository{},
			expectedError: false,
		},
		{
			name:          "invalid id zero",
			id:            0,
			mockRepo:      &mockAdminWordRepository{},
			expectedError: true,
			errorContains: "invalid word id",
		},
		{
			name:          "invalid id negative",
			id:            -1,
			mockRepo:      &mockAdminWordRepository{},
			expectedError: true,
			errorContains: "invalid word id",
		},
		{
			name: "repository error",
			id:   1,
			mockRepo: &mockAdminWordRepository{
				deleteErr: errors.New("delete error"),
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewAdminWordService(tt.mockRepo, &mockDictionaryHistoryRepository{})
			ctx := context.Background()

			err := svc.DeleteWord(ctx, tt.id)

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

// Helper function to create int pointer
func intPtr(i int) *int {
	return &i
}

