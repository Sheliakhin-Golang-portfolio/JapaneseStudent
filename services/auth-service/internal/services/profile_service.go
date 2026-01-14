package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"strings"

	"github.com/japanesestudent/auth-service/internal/models"
	"github.com/japanesestudent/libs/auth/service"
	"golang.org/x/crypto/bcrypt"
)

// ProfileUserRepository is the interface that wraps methods for User table data access needed by profile service
type ProfileUserRepository interface {
	// GetByID retrieves a user by ID
	//
	// "userID" parameter is used to retrieve a user by ID.
	//
	// If user with such ID does not exist, the error will be returned together with "nil" value.
	GetByID(ctx context.Context, userID int) (*models.User, error)
	// Update updates user fields
	//
	// "userID" parameter is used to identify the user.
	// "user" parameter is used to update user fields.
	// "settings" parameter is used to update user settings.
	// "active" parameter is used to update the active status (if provided).
	//
	// If some error occurs during user update, the error will be returned.
	Update(ctx context.Context, userID int, user *models.User, settings *models.UserSettings, active *bool) error
	// UpdatePasswordHash updates the password hash for a user
	//
	// "userID" parameter is used to identify the user.
	// "passwordHash" parameter is used to update the password hash.
	//
	// If some error occurs during password hash update, the error will be returned.
	UpdatePasswordHash(ctx context.Context, userID int, passwordHash string) error
	// ExistsByEmail checks if a user exists with the given email
	//
	// "email" parameter is used to check if a user exists with the given email.
	//
	// If some error occurs during check, the error will be returned together with "false" value.
	ExistsByEmail(ctx context.Context, email string) (bool, error)
	// ExistsByUsername checks if a user exists with the given username
	//
	// "username" parameter is used to check if a user exists with the given username.
	//
	// If some error occurs during check, the error will be returned together with "false" value.
	ExistsByUsername(ctx context.Context, username string) (bool, error)
	// UpdateActive updates the active status of a user
	//
	// "userID" parameter is used to identify the user.
	// "active" parameter is the new active status.
	//
	// If some error occurs during active status update, the error will be returned.
	UpdateActive(ctx context.Context, userID int, active bool) error
}

// UserEmailRepository is the interface that wraps methods for User table data access needed for getUserEmail function
type UserEmailRepository interface {
	// GetEmail retrieves user email by user ID
	//
	// "userID" parameter is used to retrieve user email by user ID.
	//
	// If user with such ID does not exist, the error will be returned together with "nil" value.
	GetEmail(ctx context.Context, userID int) (string, error)
}

// profileService implements ProfileService
type profileService struct {
	userRepo             ProfileUserRepository
	userSettingsRepo     UserSettingsRepository
	tokenGenerator       *service.TokenGenerator
	mediaBaseURL         string
	apiKey               string
	taskBaseURL          string
	scheduledTaskBaseURL string
	verificationURL      string
	learnServiceBaseURL  string
	isDockerContainer    bool
}

// NewProfileService creates a new profile service
func NewProfileService(
	userRepo ProfileUserRepository,
	userSettingsRepo UserSettingsRepository,
	tokenGenerator *service.TokenGenerator,
	mediaBaseURL string,
	apiKey string,
	taskBaseURL string,
	verificationURL string,
	scheduledTaskBaseURL string,
	learnServiceBaseURL string,
	isDockerContainer bool,
) *profileService {
	return &profileService{
		userRepo:             userRepo,
		userSettingsRepo:     userSettingsRepo,
		tokenGenerator:       tokenGenerator,
		mediaBaseURL:         mediaBaseURL,
		apiKey:               apiKey,
		taskBaseURL:          taskBaseURL,
		scheduledTaskBaseURL: scheduledTaskBaseURL,
		verificationURL:      verificationURL,
		learnServiceBaseURL:  learnServiceBaseURL,
		isDockerContainer:    isDockerContainer,
	}
}

// GetUser retrieves user profile information
func (s *profileService) GetUser(ctx context.Context, userId int) (*models.ProfileResponse, error) {
	if userId <= 0 {
		return nil, fmt.Errorf("invalid user id")
	}

	user, err := s.userRepo.GetByID(ctx, userId)
	if err != nil {
		return nil, err
	}

	return &models.ProfileResponse{
		Username: user.Username,
		Email:    user.Email,
		Avatar:   user.Avatar,
	}, nil
}

// UpdateUser updates user profile with validation
func (s *profileService) UpdateUser(ctx context.Context, userId int, username, email string) error {
	// Normalize inputs
	normalizedEmail := strings.TrimSpace(strings.ToLower(email))
	normalizedUsername := strings.TrimSpace(username)

	// Validate user credentials
	if err := s.checkUpdateUserCredentials(ctx, userId, normalizedUsername, normalizedEmail); err != nil {
		return err
	}

	// Create user model for update
	userData := &models.User{
		Username: normalizedUsername,
		Email:    normalizedEmail,
	}

	// Update user (change active status to false if email has changed)
	active := normalizedEmail == ""
	if err := s.userRepo.Update(ctx, userId, userData, nil, &active); err != nil {
		return err
	}

	// Send verification email if email has been changed
	if !active {
		// Generate verification token with user_id (using access token format)
		verificationToken, _, err := s.tokenGenerator.GenerateTokens(userId, int(models.RoleUser))
		if err != nil {
			return err
		}

		// Build verification URL
		verificationURL := fmt.Sprintf("%s?validToken=%s", s.verificationURL, verificationToken)

		// Build content for email: email + ';' + verificationURL
		content := fmt.Sprintf("%s;%s", normalizedEmail, verificationURL)
		if err := createImmediateTask(ctx, s.taskBaseURL, s.apiKey, userId, "email_verification", content); err != nil {
			return err
		}
	}

	// Return success
	return nil
}

func (s *profileService) checkUpdateUserCredentials(ctx context.Context, userId int, username, email string) error {
	// Validate that at least one field is provided
	if email == "" && username == "" {
		return fmt.Errorf("at least one field must be provided")
	}

	errorChan := make(chan error, 3)
	// Validate that user ID is valid
	go func() {
		if userId <= 0 {
			errorChan <- fmt.Errorf("invalid user id")
			return
		}
		errorChan <- nil
	}()

	// Validate email if provided
	go func() {
		if email != "" {
			if !emailRegex.MatchString(email) {
				errorChan <- fmt.Errorf("invalid email format")
				return
			}

			// Check email uniqueness
			emailExists, err := s.userRepo.ExistsByEmail(ctx, email)
			if err != nil {
				errorChan <- err
				return
			}
			if emailExists {
				errorChan <- fmt.Errorf("email already exists")
				return
			}
		}
		errorChan <- nil
	}()

	// Validate username if provided
	go func() {
		if username != "" {
			usernameExists, err := s.userRepo.ExistsByUsername(ctx, username)
			if err != nil {
				errorChan <- err
				return
			}
			if usernameExists {
				errorChan <- fmt.Errorf("username already exists")
				return
			}
		}
		errorChan <- nil
	}()

	// Wait for all validations to complete
	for range 3 {
		err := <-errorChan
		if err != nil {
			return err
		}
	}

	return nil
}

// UpdateAvatar updates user avatar
func (s *profileService) UpdateAvatar(ctx context.Context, userId int, avatarFile multipart.File, avatarFilename string) (string, error) {
	if userId <= 0 {
		return "", fmt.Errorf("invalid user id")
	}

	// Get current user to check for existing avatar
	currentUser, err := s.userRepo.GetByID(ctx, userId)
	if err != nil {
		return "", fmt.Errorf("user not found")
	}

	// Delete old avatar if it exists
	if currentUser.Avatar != "" && s.mediaBaseURL != "" && s.apiKey != "" {
		fileID := extractFileIDFromAvatarURL(currentUser.Avatar)
		if fileID != "" {
			if err := deleteAvatarFromMediaService(ctx, s.mediaBaseURL, s.apiKey, fileID); err != nil {
				return "", err
			}
		}
	}

	// Upload new avatar
	uploadedURL, err := uploadAvatar(ctx, s.mediaBaseURL, s.apiKey, avatarFile, avatarFilename)
	if err != nil {
		return "", err
	}

	// Update user with new avatar URL
	userData := &models.User{
		Avatar: uploadedURL,
	}
	err = s.userRepo.Update(ctx, userId, userData, nil, nil)
	if err != nil {
		return "", err
	}

	return uploadedURL, nil
}

// UpdatePassword updates user password with validation
func (s *profileService) UpdatePassword(ctx context.Context, userId int, password string) error {
	if userId <= 0 {
		return fmt.Errorf("invalid user id")
	}

	// Validate password
	if password == "" {
		return fmt.Errorf("password cannot be empty")
	}

	// Validate password against regex
	for _, regex := range passwordRegex {
		if !regex.MatchString(password) {
			return fmt.Errorf("password must be at least 8 characters long and contain at least one uppercase letter, one lowercase letter, one number, and one special character (!_?^&+-=|)")
		}
	}

	// Hash password
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Update password hash
	err = s.userRepo.UpdatePasswordHash(ctx, userId, string(passwordHash))
	if err != nil {
		return err
	}

	return nil
}

func (s *profileService) UpdateRepeatFlag(ctx context.Context, userId int, flag string, r *http.Request) error {
	// Validate user ID
	if userId <= 0 {
		return fmt.Errorf("invalid user id")
	}

	// Validate flag value
	if flag != "in question" && flag != "ignore" && flag != "repeat" {
		return fmt.Errorf("flag must be 'in question', 'ignore', or 'repeat'")
	}

	// Get current user settings to retrieve previous flag value
	currentSettings, err := s.userSettingsRepo.GetByUserId(ctx, userId)
	if err != nil {
		return err
	}

	previousFlag := currentSettings.AlphabetRepeat

	// Update the flag
	updateReq := &models.UserSettings{
		UserID:         userId,
		AlphabetRepeat: models.RepeatType(flag),
	}
	if err := s.userSettingsRepo.Update(ctx, userId, updateReq); err != nil {
		return err
	}

	// If new flag is "repeat", create scheduled task
	if flag == "repeat" {
		// Get user email
		userEmail, err := getUserEmail(ctx, s.userRepo, userId)
		if err != nil {
			return err
		}

		// Construct URL to drop marks endpoint
		dropMarksURL := fmt.Sprintf("%s/api/v6/test-results/drop-marks/%d", s.learnServiceBaseURL, userId)

		// Call task-service to create scheduled task
		return createScheduledTask(r.Context(), s.scheduledTaskBaseURL, s.apiKey, userId, "notify_alphabet", userEmail, dropMarksURL, "0 0 * * *")
	} else if previousFlag == "repeat" {
		// If new flag is NOT "repeat" AND previous flag was "repeat", delete scheduled task
		return deleteScheduledTaskByUserID(r.Context(), s.scheduledTaskBaseURL, s.apiKey, userId)
	}
	return nil
}

// getUserEmail retrieves user email by user ID
func getUserEmail(ctx context.Context, userRepo ProfileUserRepository, userID int) (string, error) {
	if userID <= 0 {
		return "", fmt.Errorf("invalid user id")
	}

	user, err := userRepo.GetByID(ctx, userID)
	if err != nil {
		return "", err
	}

	return user.Email, nil
}

// createScheduledTask creates a scheduled task in task-service
func createScheduledTask(ctx context.Context, scheduledTaskBaseURL, apiKey string, userID int, emailSlug, content, url, cron string) error {
	if scheduledTaskBaseURL == "" {
		return fmt.Errorf("SCHEDULED_TASK_BASE_URL is not configured")
	}
	if apiKey == "" {
		return fmt.Errorf("API_KEY is not configured")
	}

	// Create request body
	reqBody := map[string]any{
		"user_id":    userID,
		"email_slug": emailSlug,
		"content":    content,
		"url":        url,
		"cron":       cron,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request body: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, scheduledTaskBaseURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", apiKey)

	// Make HTTP request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to create scheduled task: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("task service returned status %d", resp.StatusCode)
	}

	return nil
}

// deleteScheduledTaskByUserID deletes scheduled tasks by user ID in task-service
func deleteScheduledTaskByUserID(ctx context.Context, scheduledTaskBaseURL, apiKey string, userID int) error {
	if scheduledTaskBaseURL == "" {
		return fmt.Errorf("SCHEDULED_TASK_BASE_URL is not configured")
	}
	if apiKey == "" {
		return fmt.Errorf("API_KEY is not configured")
	}

	// Create request body
	reqBody := map[string]int{
		"userId": userID,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request body: %w", err)
	}

	// Build task service URL
	taskBaseURL := strings.TrimSuffix(scheduledTaskBaseURL, "/")
	urlPath := fmt.Sprintf("%s/by-user", taskBaseURL)

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, urlPath, bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", apiKey)

	// Make HTTP request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete scheduled task: %w", err)
	}
	defer resp.Body.Close()

	// Check response status (204 No Content for successful deletion)
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("task service returned status %d", resp.StatusCode)
	}

	return nil
}
