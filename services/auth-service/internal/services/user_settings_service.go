package services

import (
	"context"
	"fmt"

	"github.com/Sheliakhin-Golang-portfolio/JapaneseStudent/services/auth-service/internal/models"
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
	// Method ExistsByUserId checks if user settings exist for a given user ID
	//
	// "userId" parameter is used to check if user settings exist by user ID.
	//
	// If some error occurs during user settings existence check, the error will be returned together with "false" value.
	ExistsByUserId(ctx context.Context, userId int) (bool, error)
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
		AlphabetRepeat:     settings.AlphabetRepeat,
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
	// Validate update request
	if err := s.validateUpdateUserSettingsData(ctx, userId, updateRequest); err != nil {
		return err
	}

	// Update settings
	settings := &models.UserSettings{
		UserID:         userId,
		AlphabetRepeat: models.RepeatTypeInQuestion,
		Language:       models.LanguageEnglish,
	}
	if updateRequest.NewWordCount != nil {
		settings.NewWordCount = *updateRequest.NewWordCount
	}
	if updateRequest.OldWordCount != nil {
		settings.OldWordCount = *updateRequest.OldWordCount
	}
	if updateRequest.AlphabetLearnCount != nil {
		settings.AlphabetLearnCount = *updateRequest.AlphabetLearnCount
	}

	err := s.repo.Update(ctx, userId, settings)
	if err != nil {
		return err
	}
	return nil
}

// validateUpdateUserSettingsData validates the update request data
//
// For successful results:
//
// - newWordCount and oldWordCount must be between 10 and 40
//
// - alphabetLearnCount must be between 5 and 15
//
// - language must be "en", "ru", or "de"
func (s *userSettingsService) validateUpdateUserSettingsData(ctx context.Context, userId int, updateRequest *models.UpdateUserSettingsRequest) error {
	// Validate that at least one field is provided
	if updateRequest.NewWordCount == nil && updateRequest.OldWordCount == nil && updateRequest.AlphabetLearnCount == nil && updateRequest.Language == "" && updateRequest.AlphabetRepeat == "" {
		return fmt.Errorf("at least one field must be provided")
	}

	if userId <= 0 {
		return fmt.Errorf("invalid user id")
	}

	errorChan := make(chan error, 6)

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
		if updateRequest.Language != "" && updateRequest.Language != models.LanguageEnglish && updateRequest.Language != models.LanguageRussian && updateRequest.Language != models.LanguageGerman {
			errorChan <- fmt.Errorf("invalid language: %s, must be 'en', 'ru', or 'de'", updateRequest.Language)
			return
		}
		errorChan <- nil
	}()

	// Validate alphabetRepeat
	go func() {
		if updateRequest.AlphabetRepeat != "" && updateRequest.AlphabetRepeat != "in question" && updateRequest.AlphabetRepeat != "ignore" && updateRequest.AlphabetRepeat != "repeat" {
			errorChan <- fmt.Errorf("invalid alphabetRepeat: %s, must be 'in question', 'ignore', or 'repeat'", updateRequest.AlphabetRepeat)
			return
		}
		errorChan <- nil
	}()

	// Get existing settings to preserve unchanged fields
	go func() {
		exists, err := s.repo.ExistsByUserId(ctx, userId)
		if err != nil {
			errorChan <- err
			return
		}
		if !exists {
			errorChan <- fmt.Errorf("user settings not found")
			return
		}
		errorChan <- nil
	}()

	// Wait for all validations to complete
	for range 6 {
		err := <-errorChan
		if err != nil {
			return err
		}
	}

	return nil
}
