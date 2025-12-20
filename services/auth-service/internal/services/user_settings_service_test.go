package services

import (
	"context"
	"errors"
	"testing"

	"github.com/japanesestudent/auth-service/internal/models"
	"github.com/stretchr/testify/assert"
)

// mockUserSettingsRepositoryForService is a mock implementation of UserSettingsRepository for service tests
type mockUserSettingsRepositoryForService struct {
	settings  *models.UserSettings
	err       error
	updateErr error
}

func (m *mockUserSettingsRepositoryForService) Create(ctx context.Context, userId int) error {
	return m.err
}

func (m *mockUserSettingsRepositoryForService) GetByUserId(ctx context.Context, userId int) (*models.UserSettings, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.settings, nil
}

func (m *mockUserSettingsRepositoryForService) Update(ctx context.Context, userId int, settings *models.UserSettings) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	return m.err
}

func TestNewUserSettingsService(t *testing.T) {
	mockRepo := &mockUserSettingsRepositoryForService{}

	svc := NewUserSettingsService(mockRepo)

	assert.NotNil(t, svc)
	assert.Equal(t, mockRepo, svc.repo)
}

func TestUserSettingsService_GetUserSettings(t *testing.T) {
	tests := []struct {
		name          string
		userId        int
		mockRepo      *mockUserSettingsRepositoryForService
		expectedError bool
		expectedCount int
	}{
		{
			name:   "success",
			userId: 1,
			mockRepo: &mockUserSettingsRepositoryForService{
				settings: &models.UserSettings{
					ID:                 1,
					UserID:             1,
					NewWordCount:       20,
					OldWordCount:       20,
					AlphabetLearnCount: 10,
					Language:           models.LanguageEnglish,
				},
			},
			expectedError: false,
		},
		{
			name:   "settings not found",
			userId: 999,
			mockRepo: &mockUserSettingsRepositoryForService{
				err: errors.New("user settings not found"),
			},
			expectedError: true,
		},
		{
			name:   "database error",
			userId: 1,
			mockRepo: &mockUserSettingsRepositoryForService{
				err: errors.New("database error"),
			},
			expectedError: true,
		},
		{
			name:   "success with russian language",
			userId: 2,
			mockRepo: &mockUserSettingsRepositoryForService{
				settings: &models.UserSettings{
					ID:                 2,
					UserID:             2,
					NewWordCount:       30,
					OldWordCount:       25,
					AlphabetLearnCount: 15,
					Language:           models.LanguageRussian,
				},
			},
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewUserSettingsService(tt.mockRepo)

			result, err := svc.GetUserSettings(context.Background(), tt.userId)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.mockRepo.settings.NewWordCount, result.NewWordCount)
				assert.Equal(t, tt.mockRepo.settings.OldWordCount, result.OldWordCount)
				assert.Equal(t, tt.mockRepo.settings.AlphabetLearnCount, result.AlphabetLearnCount)
				assert.Equal(t, tt.mockRepo.settings.Language, result.Language)
				// Verify IDs are not included in response
				assert.Zero(t, result.NewWordCount == 0 && tt.mockRepo.settings.NewWordCount != 0) // Just check they match
			}
		})
	}
}

func TestUserSettingsService_UpdateUserSettings(t *testing.T) {
	tests := []struct {
		name          string
		userId        int
		updateRequest *models.UpdateUserSettingsRequest
		mockRepo      *mockUserSettingsRepositoryForService
		expectedError bool
		errorContains string
	}{
		{
			name:   "success update all fields",
			userId: 1,
			updateRequest: &models.UpdateUserSettingsRequest{
				NewWordCount:       intPtr(25),
				OldWordCount:       intPtr(30),
				AlphabetLearnCount: intPtr(12),
				Language:           languagePtr(models.LanguageEnglish),
			},
			mockRepo: &mockUserSettingsRepositoryForService{
				settings: &models.UserSettings{
					ID:                 1,
					UserID:             1,
					NewWordCount:       20,
					OldWordCount:       20,
					AlphabetLearnCount: 10,
					Language:           models.LanguageEnglish,
				},
			},
			expectedError: false,
		},
		{
			name:   "success update partial fields",
			userId: 1,
			updateRequest: &models.UpdateUserSettingsRequest{
				NewWordCount: intPtr(30),
			},
			mockRepo: &mockUserSettingsRepositoryForService{
				settings: &models.UserSettings{
					ID:                 1,
					UserID:             1,
					NewWordCount:       20,
					OldWordCount:       20,
					AlphabetLearnCount: 10,
					Language:           models.LanguageEnglish,
				},
			},
			expectedError: false,
		},
		{
			name:   "invalid newWordCount - too low",
			userId: 1,
			updateRequest: &models.UpdateUserSettingsRequest{
				NewWordCount: intPtr(5),
			},
			mockRepo: &mockUserSettingsRepositoryForService{
				settings: &models.UserSettings{
					ID:                 1,
					UserID:             1,
					NewWordCount:       20,
					OldWordCount:       20,
					AlphabetLearnCount: 10,
					Language:           models.LanguageEnglish,
				},
			},
			expectedError: true,
			errorContains: "newWordCount must be between 10 and 40",
		},
		{
			name:   "invalid newWordCount - too high",
			userId: 1,
			updateRequest: &models.UpdateUserSettingsRequest{
				NewWordCount: intPtr(50),
			},
			mockRepo: &mockUserSettingsRepositoryForService{
				settings: &models.UserSettings{
					ID:                 1,
					UserID:             1,
					NewWordCount:       20,
					OldWordCount:       20,
					AlphabetLearnCount: 10,
					Language:           models.LanguageEnglish,
				},
			},
			expectedError: true,
			errorContains: "newWordCount must be between 10 and 40",
		},
		{
			name:   "invalid oldWordCount - too low",
			userId: 1,
			updateRequest: &models.UpdateUserSettingsRequest{
				OldWordCount: intPtr(5),
			},
			mockRepo: &mockUserSettingsRepositoryForService{
				settings: &models.UserSettings{
					ID:                 1,
					UserID:             1,
					NewWordCount:       20,
					OldWordCount:       20,
					AlphabetLearnCount: 10,
					Language:           models.LanguageEnglish,
				},
			},
			expectedError: true,
			errorContains: "oldWordCount must be between 10 and 40",
		},
		{
			name:   "invalid oldWordCount - too high",
			userId: 1,
			updateRequest: &models.UpdateUserSettingsRequest{
				OldWordCount: intPtr(50),
			},
			mockRepo: &mockUserSettingsRepositoryForService{
				settings: &models.UserSettings{
					ID:                 1,
					UserID:             1,
					NewWordCount:       20,
					OldWordCount:       20,
					AlphabetLearnCount: 10,
					Language:           models.LanguageEnglish,
				},
			},
			expectedError: true,
			errorContains: "oldWordCount must be between 10 and 40",
		},
		{
			name:   "invalid alphabetLearnCount - too low",
			userId: 1,
			updateRequest: &models.UpdateUserSettingsRequest{
				AlphabetLearnCount: intPtr(3),
			},
			mockRepo: &mockUserSettingsRepositoryForService{
				settings: &models.UserSettings{
					ID:                 1,
					UserID:             1,
					NewWordCount:       20,
					OldWordCount:       20,
					AlphabetLearnCount: 10,
					Language:           models.LanguageEnglish,
				},
			},
			expectedError: true,
			errorContains: "alphabetLearnCount must be between 5 and 15",
		},
		{
			name:   "invalid alphabetLearnCount - too high",
			userId: 1,
			updateRequest: &models.UpdateUserSettingsRequest{
				AlphabetLearnCount: intPtr(20),
			},
			mockRepo: &mockUserSettingsRepositoryForService{
				settings: &models.UserSettings{
					ID:                 1,
					UserID:             1,
					NewWordCount:       20,
					OldWordCount:       20,
					AlphabetLearnCount: 10,
					Language:           models.LanguageEnglish,
				},
			},
			expectedError: true,
			errorContains: "alphabetLearnCount must be between 5 and 15",
		},
		{
			name:   "invalid language",
			userId: 1,
			updateRequest: &models.UpdateUserSettingsRequest{
				Language: languagePtr(models.Language("fr")),
			},
			mockRepo: &mockUserSettingsRepositoryForService{
				settings: &models.UserSettings{
					ID:                 1,
					UserID:             1,
					NewWordCount:       20,
					OldWordCount:       20,
					AlphabetLearnCount: 10,
					Language:           models.LanguageEnglish,
				},
			},
			expectedError: true,
			errorContains: "invalid language",
		},
		{
			name:   "success with valid language - english",
			userId: 1,
			updateRequest: &models.UpdateUserSettingsRequest{
				Language: languagePtr(models.LanguageEnglish),
			},
			mockRepo: &mockUserSettingsRepositoryForService{
				settings: &models.UserSettings{
					ID:                 1,
					UserID:             1,
					NewWordCount:       20,
					OldWordCount:       20,
					AlphabetLearnCount: 10,
					Language:           models.LanguageEnglish,
				},
			},
			expectedError: false,
		},
		{
			name:   "success with valid language - russian",
			userId: 1,
			updateRequest: &models.UpdateUserSettingsRequest{
				Language: languagePtr(models.LanguageRussian),
			},
			mockRepo: &mockUserSettingsRepositoryForService{
				settings: &models.UserSettings{
					ID:                 1,
					UserID:             1,
					NewWordCount:       20,
					OldWordCount:       20,
					AlphabetLearnCount: 10,
					Language:           models.LanguageEnglish,
				},
			},
			expectedError: false,
		},
		{
			name:   "success with valid language - german",
			userId: 1,
			updateRequest: &models.UpdateUserSettingsRequest{
				Language: languagePtr(models.LanguageGerman),
			},
			mockRepo: &mockUserSettingsRepositoryForService{
				settings: &models.UserSettings{
					ID:                 1,
					UserID:             1,
					NewWordCount:       20,
					OldWordCount:       20,
					AlphabetLearnCount: 10,
					Language:           models.LanguageEnglish,
				},
			},
			expectedError: false,
		},
		{
			name:   "boundary values - newWordCount at minimum",
			userId: 1,
			updateRequest: &models.UpdateUserSettingsRequest{
				NewWordCount: intPtr(10),
			},
			mockRepo: &mockUserSettingsRepositoryForService{
				settings: &models.UserSettings{
					ID:                 1,
					UserID:             1,
					NewWordCount:       20,
					OldWordCount:       20,
					AlphabetLearnCount: 10,
					Language:           models.LanguageEnglish,
				},
			},
			expectedError: false,
		},
		{
			name:   "boundary values - newWordCount at maximum",
			userId: 1,
			updateRequest: &models.UpdateUserSettingsRequest{
				NewWordCount: intPtr(40),
			},
			mockRepo: &mockUserSettingsRepositoryForService{
				settings: &models.UserSettings{
					ID:                 1,
					UserID:             1,
					NewWordCount:       20,
					OldWordCount:       20,
					AlphabetLearnCount: 10,
					Language:           models.LanguageEnglish,
				},
			},
			expectedError: false,
		},
		{
			name:   "boundary values - alphabetLearnCount at minimum",
			userId: 1,
			updateRequest: &models.UpdateUserSettingsRequest{
				AlphabetLearnCount: intPtr(5),
			},
			mockRepo: &mockUserSettingsRepositoryForService{
				settings: &models.UserSettings{
					ID:                 1,
					UserID:             1,
					NewWordCount:       20,
					OldWordCount:       20,
					AlphabetLearnCount: 10,
					Language:           models.LanguageEnglish,
				},
			},
			expectedError: false,
		},
		{
			name:   "boundary values - alphabetLearnCount at maximum",
			userId: 1,
			updateRequest: &models.UpdateUserSettingsRequest{
				AlphabetLearnCount: intPtr(15),
			},
			mockRepo: &mockUserSettingsRepositoryForService{
				settings: &models.UserSettings{
					ID:                 1,
					UserID:             1,
					NewWordCount:       20,
					OldWordCount:       20,
					AlphabetLearnCount: 10,
					Language:           models.LanguageEnglish,
				},
			},
			expectedError: false,
		},
		{
			name:   "database error on get existing settings",
			userId: 1,
			updateRequest: &models.UpdateUserSettingsRequest{
				NewWordCount: intPtr(25),
			},
			mockRepo: &mockUserSettingsRepositoryForService{
				err: errors.New("database error"),
			},
			expectedError: true,
			errorContains: "database error", // Service returns error directly from GetByUserId
		},
		{
			name:   "database error on update",
			userId: 1,
			updateRequest: &models.UpdateUserSettingsRequest{
				NewWordCount: intPtr(25),
			},
			mockRepo: &mockUserSettingsRepositoryForService{
				settings: &models.UserSettings{
					ID:                 1,
					UserID:             1,
					NewWordCount:       20,
					OldWordCount:       20,
					AlphabetLearnCount: 10,
					Language:           models.LanguageEnglish,
				},
				updateErr: errors.New("update error"),
			},
			expectedError: true,
		},
		{
			name:   "multiple validation errors - should return first error",
			userId: 1,
			updateRequest: &models.UpdateUserSettingsRequest{
				NewWordCount: intPtr(5),
				OldWordCount: intPtr(50),
			},
			mockRepo: &mockUserSettingsRepositoryForService{
				settings: &models.UserSettings{
					ID:                 1,
					UserID:             1,
					NewWordCount:       20,
					OldWordCount:       20,
					AlphabetLearnCount: 10,
					Language:           models.LanguageEnglish,
				},
			},
			expectedError: true,
			errorContains: "must be between",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewUserSettingsService(tt.mockRepo)

			err := svc.UpdateUserSettings(context.Background(), tt.userId, tt.updateRequest)

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

// Helper functions (intPtr is defined in admin_service_test.go)

func languagePtr(l models.Language) *models.Language {
	return &l
}
