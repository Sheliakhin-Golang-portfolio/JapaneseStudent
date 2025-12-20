package services

import (
	"context"
	"fmt"

	"github.com/japanesestudent/auth-service/internal/models"
)

// UserSettingsRepository is the interface that wraps methods for UserSettings table data access
type UserSettingsRepository interface {
	// Method Create inserts a new user settings record
	//
	// "userId" parameter is used to create a new user settings record.
	//
	// If some error occurs during user settings creation, the error will be returned together with "nil" value.
	Create(ctx context.Context, userId int) error
	// GetByUserId retrieves user settings by user ID
	//
	// "userId" parameter is used to retrieve user settings by user ID.
	//
	// If user settings are not found, the error will be returned together with "nil" value.
	GetByUserId(ctx context.Context, userId int) (*models.UserSettings, error)
	// Method Update updates user settings for a given user ID
	//
	// "userId" parameter is used to update user settings by user ID.
	// "settings" parameter is used to update user settings.
	//
	// If some error occurs during user settings update, the error will be returned together with "nil" value.
	Update(ctx context.Context, userId int, settings *models.UserSettings) error
}

// userSettingsService implements UserSettingsService
type userSettingsService struct {
	repo UserSettingsRepository
}

// NewUserSettingsService creates a new user settings service
func NewUserSettingsService(repo UserSettingsRepository) *userSettingsService {
	return &userSettingsService{
		repo: repo,
	}
}

// GetUserSettings retrieves user settings without IDs
func (s *userSettingsService) GetUserSettings(ctx context.Context, userId int) (*models.UserSettingsResponse, error) {
	settings, err := s.repo.GetByUserId(ctx, userId)
	if err != nil {
		return nil, err
	}

	return &models.UserSettingsResponse{
		NewWordCount:       settings.NewWordCount,
		OldWordCount:       settings.OldWordCount,
		AlphabetLearnCount: settings.AlphabetLearnCount,
		Language:           settings.Language,
	}, nil
}

// UpdateUserSettings updates user settings with validation
//
// For successful results:
//
// - newWordCount and oldWordCount must be between 10 and 40
//
// - alphabetLearnCount must be between 5 and 15
//
// - language must be "en", "ru", or "de"
//
// "userId" parameter is used to update user settings by user ID.
// "updateRequest" parameter is used to update user settings.
//
// If some error occurs during user settings update, the error will be returned together with "nil" value.
func (s *userSettingsService) UpdateUserSettings(ctx context.Context, userId int, updateRequest *models.UpdateUserSettingsRequest) error {
	if userId <= 0 {
		return fmt.Errorf("invalid user id")
	}

	errorChan := make(chan error, 5)
	settingsChan := make(chan *models.UserSettings, 1)

	go func() {
		if updateRequest.NewWordCount != nil && (*updateRequest.NewWordCount < 10 || *updateRequest.NewWordCount > 40) {
			errorChan <- fmt.Errorf("newWordCount must be between 10 and 40")
			return
		}
		errorChan <- nil
	}()

	// Validate oldWordCount
	go func() {
		if updateRequest.OldWordCount != nil && (*updateRequest.OldWordCount < 10 || *updateRequest.OldWordCount > 40) {
			errorChan <- fmt.Errorf("oldWordCount must be between 10 and 40")
			return
		}
		errorChan <- nil
	}()

	// Validate alphabetLearnCount
	go func() {
		if updateRequest.AlphabetLearnCount != nil && (*updateRequest.AlphabetLearnCount < 5 || *updateRequest.AlphabetLearnCount > 15) {
			errorChan <- fmt.Errorf("alphabetLearnCount must be between 5 and 15")
			return
		}
		errorChan <- nil
	}()

	// Validate language
	go func() {
		if updateRequest.Language == nil {
			errorChan <- nil
			return
		}
		language := models.Language(string(*updateRequest.Language))
		if language != models.LanguageEnglish && language != models.LanguageRussian && language != models.LanguageGerman {
			errorChan <- fmt.Errorf("invalid language: %s, must be 'en', 'ru', or 'de'", language)
			return
		}
		errorChan <- nil
	}()

	// Get existing settings to preserve unchanged fields
	go func() {
		existingSettings, err := s.repo.GetByUserId(ctx, userId)
		if err != nil {
			errorChan <- err
			settingsChan <- nil
			return
		}
		settingsChan <- existingSettings
		errorChan <- nil
	}()

	// Wait for all validations to complete
	for range 5 {
		err := <-errorChan
		if err != nil {
			return err
		}
	}

	// Update settings
	settings := <-settingsChan
	if updateRequest.NewWordCount != nil {
		settings.NewWordCount = *updateRequest.NewWordCount
	}
	if updateRequest.OldWordCount != nil {
		settings.OldWordCount = *updateRequest.OldWordCount
	}
	if updateRequest.AlphabetLearnCount != nil {
		settings.AlphabetLearnCount = *updateRequest.AlphabetLearnCount
	}
	if updateRequest.Language != nil {
		settings.Language = *updateRequest.Language
	}

	err := s.repo.Update(ctx, userId, settings)
	if err != nil {
		return err
	}
	return nil
}
