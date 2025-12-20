package services

import (
	"context"
	"errors"
	"testing"

	"github.com/japanesestudent/learn-service/internal/models"
	"github.com/stretchr/testify/assert"
)

// mockAdminCharactersRepository is a mock implementation of AdminCharactersRepository
type mockAdminCharactersRepository struct {
	characters                    []models.Character
	character                     *models.Character
	exists                        bool
	err                           error
	existsErr                     error // Separate error for ExistsByVowelConsonant
	existsByKatakanaOrHiragana    bool
	existsByKatakanaOrHiraganaErr error // Separate error for ExistsByKatakanaOrHiragana
	createErr                     error
	updateErr                     error
	deleteErr                     error
}

func (m *mockAdminCharactersRepository) GetAllForAdmin(ctx context.Context) ([]models.Character, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.characters, nil
}

func (m *mockAdminCharactersRepository) GetByIDAdmin(ctx context.Context, id int) (*models.Character, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.character, nil
}

func (m *mockAdminCharactersRepository) ExistsByVowelConsonant(ctx context.Context, vowel, consonant string) (bool, error) {
	if m.existsErr != nil {
		return false, m.existsErr
	}
	if m.err != nil {
		return false, m.err
	}
	return m.exists, nil
}

func (m *mockAdminCharactersRepository) ExistsByKatakanaOrHiragana(ctx context.Context, katakana, hiragana string) (bool, error) {
	if m.existsByKatakanaOrHiraganaErr != nil {
		return false, m.existsByKatakanaOrHiraganaErr
	}
	if m.err != nil {
		return false, m.err
	}
	return m.existsByKatakanaOrHiragana, nil
}

func (m *mockAdminCharactersRepository) Create(ctx context.Context, character *models.Character) error {
	if m.createErr != nil {
		return m.createErr
	}
	if m.err != nil {
		return m.err
	}
	character.ID = 1
	return nil
}

func (m *mockAdminCharactersRepository) Update(ctx context.Context, id int, character *models.Character) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	return m.err
}

func (m *mockAdminCharactersRepository) Delete(ctx context.Context, id int) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	return m.err
}

func TestNewAdminService(t *testing.T) {
	mockRepo := &mockAdminCharactersRepository{}

	svc := NewAdminService(mockRepo)

	assert.NotNil(t, svc)
	assert.Equal(t, mockRepo, svc.repo)
}

func TestAdminService_GetAllForAdmin(t *testing.T) {
	tests := []struct {
		name          string
		mockRepo      *mockAdminCharactersRepository
		expectedError bool
		expectedCount int
	}{
		{
			name: "success",
			mockRepo: &mockAdminCharactersRepository{
				characters: []models.Character{
					{ID: 1, Consonant: "", Vowel: "a", Katakana: "ア", Hiragana: "あ"},
					{ID: 2, Consonant: "k", Vowel: "a", Katakana: "カ", Hiragana: "か"},
				},
			},
			expectedError: false,
			expectedCount: 2,
		},
		{
			name: "empty result",
			mockRepo: &mockAdminCharactersRepository{
				characters: []models.Character{},
			},
			expectedError: false,
			expectedCount: 0,
		},
		{
			name: "repository error",
			mockRepo: &mockAdminCharactersRepository{
				err: errors.New("database error"),
			},
			expectedError: true,
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewAdminService(tt.mockRepo)
			ctx := context.Background()

			result, err := svc.GetAllForAdmin(ctx)

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

func TestAdminService_GetByIDAdmin(t *testing.T) {
	tests := []struct {
		name          string
		id            int
		mockRepo      *mockAdminCharactersRepository
		expectedError bool
		expectedID    int
	}{
		{
			name: "success",
			id:   1,
			mockRepo: &mockAdminCharactersRepository{
				character: &models.Character{
					ID:             1,
					Consonant:      "",
					Vowel:          "a",
					EnglishReading: "a",
					RussianReading: "а",
					Katakana:       "ア",
					Hiragana:       "あ",
				},
			},
			expectedError: false,
			expectedID:    1,
		},
		{
			name:          "invalid id zero",
			id:            0,
			mockRepo:      &mockAdminCharactersRepository{},
			expectedError: true,
			expectedID:    0,
		},
		{
			name:          "invalid id negative",
			id:            -1,
			mockRepo:      &mockAdminCharactersRepository{},
			expectedError: true,
			expectedID:    0,
		},
		{
			name: "repository error",
			id:   1,
			mockRepo: &mockAdminCharactersRepository{
				err: errors.New("character not found"),
			},
			expectedError: true,
			expectedID:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewAdminService(tt.mockRepo)
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

func TestAdminService_CreateCharacter(t *testing.T) {
	tests := []struct {
		name          string
		request       *models.CreateCharacterRequest
		mockRepo      *mockAdminCharactersRepository
		expectedError bool
		expectedID    int
		errorContains string
	}{
		{
			name: "success",
			request: &models.CreateCharacterRequest{
				Consonant:      "k",
				Vowel:          "a",
				EnglishReading: "ka",
				RussianReading: "ка",
				Katakana:       "カ",
				Hiragana:       "か",
			},
			mockRepo: &mockAdminCharactersRepository{
				exists: false,
			},
			expectedError: false,
			expectedID:    1,
		},
		{
			name: "character already exists",
			request: &models.CreateCharacterRequest{
				Consonant: "k",
				Vowel:     "a",
			},
			mockRepo: &mockAdminCharactersRepository{
				exists: true,
			},
			expectedError: true,
			expectedID:    0,
			errorContains: "already exists",
		},
		{
			name: "failed to check existence",
			request: &models.CreateCharacterRequest{
				Consonant: "k",
				Vowel:     "a",
			},
			mockRepo: &mockAdminCharactersRepository{
				err: errors.New("database error"),
			},
			expectedError: true,
			expectedID:    0,
			errorContains: "failed to check character existence",
		},
		{
			name: "failed to create - repository error",
			request: &models.CreateCharacterRequest{
				Consonant: "k",
				Vowel:     "a",
			},
			mockRepo: &mockAdminCharactersRepository{
				exists:    false,
				createErr: errors.New("create error"),
			},
			expectedError: true,
			expectedID:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewAdminService(tt.mockRepo)
			ctx := context.Background()

			result, err := svc.CreateCharacter(ctx, tt.request)

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

func TestAdminService_UpdateCharacter(t *testing.T) {
	tests := []struct {
		name          string
		id            int
		request       *models.UpdateCharacterRequest
		mockRepo      *mockAdminCharactersRepository
		expectedError bool
		errorContains string
	}{
		{
			name: "success partial update",
			id:   1,
			request: &models.UpdateCharacterRequest{
				EnglishReading: "ka",
			},
			mockRepo: &mockAdminCharactersRepository{
				character: &models.Character{
					ID:        1,
					Consonant: "k",
					Vowel:     "a",
				},
			},
			expectedError: false,
		},
		{
			name: "success update with consonant and vowel check",
			id:   1,
			request: &models.UpdateCharacterRequest{
				Consonant: "s",
				Vowel:     "a",
			},
			mockRepo: &mockAdminCharactersRepository{
				character: &models.Character{
					ID:        1,
					Consonant: "k",
					Vowel:     "a",
				},
				exists: false,
			},
			expectedError: false,
		},
		{
			name: "success update with only consonant",
			id:   1,
			request: &models.UpdateCharacterRequest{
				Consonant: "s",
			},
			mockRepo: &mockAdminCharactersRepository{
				character: &models.Character{
					ID:        1,
					Consonant: "k",
					Vowel:     "a",
				},
				exists: false,
			},
			expectedError: false,
		},
		{
			name: "success update with only vowel",
			id:   1,
			request: &models.UpdateCharacterRequest{
				Vowel: "i",
			},
			mockRepo: &mockAdminCharactersRepository{
				character: &models.Character{
					ID:        1,
					Consonant: "k",
					Vowel:     "a",
				},
				exists: false,
			},
			expectedError: false,
		},
		{
			name:          "invalid id zero",
			id:            0,
			request:       &models.UpdateCharacterRequest{},
			mockRepo:      &mockAdminCharactersRepository{},
			expectedError: true,
			errorContains: "invalid character id",
		},
		{
			name:          "invalid id negative",
			id:            -1,
			request:       &models.UpdateCharacterRequest{},
			mockRepo:      &mockAdminCharactersRepository{},
			expectedError: true,
			errorContains: "invalid character id",
		},
		{
			name: "character not found",
			id:   999,
			request: &models.UpdateCharacterRequest{
				Consonant: "k", // Add Consonant so GetByIDAdmin is called
			},
			mockRepo: &mockAdminCharactersRepository{
				err: errors.New("character not found"),
			},
			expectedError: true,
			errorContains: "failed to get character by id",
		},
		{
			name: "character with new vowel and consonant already exists",
			id:   1,
			request: &models.UpdateCharacterRequest{
				Consonant: "s",
				Vowel:     "a",
			},
			mockRepo: &mockAdminCharactersRepository{
				character: &models.Character{
					ID:        1,
					Consonant: "k",
					Vowel:     "a",
				},
				exists: true,
			},
			expectedError: true,
			errorContains: "already exists",
		},
		{
			name: "failed to check existence",
			id:   1,
			request: &models.UpdateCharacterRequest{
				Consonant: "s",
				Vowel:     "a",
			},
			mockRepo: &mockAdminCharactersRepository{
				character: &models.Character{
					ID:        1,
					Consonant: "k",
					Vowel:     "a",
				},
				existsErr: errors.New("database error"), // Use existsErr instead of err
			},
			expectedError: true,
			errorContains: "failed to check character existence",
		},
		{
			name: "failed to update",
			id:   1,
			request: &models.UpdateCharacterRequest{
				EnglishReading: "ka",
			},
			mockRepo: &mockAdminCharactersRepository{
				character: &models.Character{
					ID:        1,
					Consonant: "k",
					Vowel:     "a",
				},
				updateErr: errors.New("update error"),
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewAdminService(tt.mockRepo)
			ctx := context.Background()

			err := svc.UpdateCharacter(ctx, tt.id, tt.request)

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

func TestAdminService_DeleteCharacter(t *testing.T) {
	tests := []struct {
		name          string
		id            int
		mockRepo      *mockAdminCharactersRepository
		expectedError bool
		errorContains string
	}{
		{
			name:          "success",
			id:            1,
			mockRepo:      &mockAdminCharactersRepository{},
			expectedError: false,
		},
		{
			name:          "invalid id zero",
			id:            0,
			mockRepo:      &mockAdminCharactersRepository{},
			expectedError: true,
			errorContains: "invalid character id",
		},
		{
			name:          "invalid id negative",
			id:            -1,
			mockRepo:      &mockAdminCharactersRepository{},
			expectedError: true,
			errorContains: "invalid character id",
		},
		{
			name: "repository error",
			id:   1,
			mockRepo: &mockAdminCharactersRepository{
				deleteErr: errors.New("delete error"),
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewAdminService(tt.mockRepo)
			ctx := context.Background()

			err := svc.DeleteCharacter(ctx, tt.id)

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
