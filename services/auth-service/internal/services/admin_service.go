package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"

	"github.com/Sheliakhin-Golang-portfolio/JapaneseStudent/services/auth-service/internal/models"
	"github.com/Sheliakhin-Golang-portfolio/JapaneseStudent/libs/auth/service"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

// UserRepository is the interface that wraps methods for User table data access
type AdminUserRepository interface {
	// Method Create inserts a new user into the database.
	//
	// "user" parameter is used to create a new user.
	//
	// If some error occurs during user creation, the error will be returned together with "nil" value.
	Create(ctx context.Context, user *models.User) error
	// Method GetByID retrieves a user by ID.
	//
	// "userID" parameter is used to retrieve a user by ID.
	//
	// If user with such ID does not exist, the error will be returned together with "nil" value.
	GetByID(ctx context.Context, userID int) (*models.User, error)
	// Method GetAll retrieves a paginated list of users with optional role and search filters.
	//
	// "page" parameter is used for pagination (default: 1).
	// "count" parameter is used for page size (default: 20).
	// "role" parameter is optional filter by role.
	// "search" parameter is optional search in email or username.
	//
	// If some error occurs, the error will be returned together with "nil" value.
	GetAll(ctx context.Context, page, count int, role *models.Role, search string) ([]models.User, error)
	// Method Update updates user fields.
	//
	// "userID" parameter is used to identify the user to update.
	// "user" parameter contains the fields to update.
	// "settings" parameter contains the fields to update.
	// "active" parameter contains the active status to update.
	//
	// If some error occurs, the error will be returned.
	Update(ctx context.Context, userID int, user *models.User, settings *models.UserSettings, active *bool) error
	// Method Delete deletes a user by ID.
	//
	// "userID" parameter is used to identify the user to delete.
	//
	// If some error occurs, the error will be returned.
	Delete(ctx context.Context, userID int) error
	// Method GetTutorsList retrieves a list of tutors (only ID and username).
	//
	// If some other error occurs, the error will be returned together with nil.
	GetTutorsList(ctx context.Context) ([]models.TutorListItem, error)
	// Method UpdateActive updates the active status of a user.
	//
	// "userID" parameter is used to identify the user to update.
	// "active" parameter is the new active status.
	//
	// If some error occurs, the error will be returned.
	UpdateActive(ctx context.Context, userID int, active bool) error
	// Method UpdatePasswordHash updates the password hash for a user.
	//
	// "userID" parameter is used to identify the user to update.
	// "passwordHash" parameter is the new password hash.
	//
	// If some error occurs, the error will be returned.
	UpdatePasswordHash(ctx context.Context, userID int, passwordHash string) error
}

// UserTokenRepository is the interface that wraps methods for UserToken table data access
type AdminUserTokenRepository interface {
	// Method Create inserts a new user token into the database.
	//
	// "userToken" parameter is used to create a new user token.
	//
	// If some error occurs during user token creation, the error will be returned together with "nil" value.
	Create(ctx context.Context, userToken *models.UserToken) error
}

// authService implements AuthService
type adminService struct {
	userRepo             AdminUserRepository
	userTokenRepo        AdminUserTokenRepository
	userSettingsRepo     UserSettingsRepository
	tokenGenerator       *service.TokenGenerator
	logger               *zap.Logger
	mediaBaseURL         string
	apiKey               string
	taskBaseURL          string
	learnServiceBaseURL  string
	isDockerContainer    bool
	scheduledTaskBaseURL string
}

// NewAuthService creates a new auth service
func NewAdminService(
	userRepo AdminUserRepository,
	userTokenRepo AdminUserTokenRepository,
	userSettingsRepo UserSettingsRepository,
	tokenGenerator *service.TokenGenerator,
	logger *zap.Logger,
	mediaBaseURL string,
	apiKey string,
	taskBaseURL string,
	learnServiceBaseURL string,
	isDockerContainer bool,
	scheduledTaskBaseURL string,
) *adminService {
	return &adminService{
		userRepo:             userRepo,
		userTokenRepo:        userTokenRepo,
		userSettingsRepo:     userSettingsRepo,
		tokenGenerator:       tokenGenerator,
		logger:               logger,
		mediaBaseURL:         mediaBaseURL,
		apiKey:               apiKey,
		taskBaseURL:          taskBaseURL,
		learnServiceBaseURL:  learnServiceBaseURL,
		isDockerContainer:    isDockerContainer,
		scheduledTaskBaseURL: scheduledTaskBaseURL,
	}
}

// GetUsersList retrieves a paginated list of users with optional role and search filters
func (s *adminService) GetUsersList(ctx context.Context, page, count int, role *int, search string) ([]models.UserListItem, error) {
	if page < 1 {
		page = 1
	}
	if count < 1 {
		count = 20
	}

	var roleValid *models.Role
	if role != nil {
		roleValue := models.Role(*role)
		roleValid = &roleValue
		if *roleValid < models.RoleUser || *roleValid > models.RoleAdmin {
			return nil, fmt.Errorf("invalid role: %d", *role)
		}
	}

	users, err := s.userRepo.GetAll(ctx, page, count, roleValid, search)
	if err != nil {
		return nil, err
	}

	usersList := make([]models.UserListItem, len(users))
	for i, user := range users {
		usersList[i] = models.UserListItem{
			ID:       user.ID,
			Username: user.Username,
			Email:    user.Email,
			Role:     user.Role,
			Avatar:   user.Avatar,
		}
	}

	return usersList, nil
}

// GetUserWithSettings retrieves a user with their settings
func (s *adminService) GetUserWithSettings(ctx context.Context, userID int) (*models.UserWithSettingsResponse, error) {
	if userID <= 0 {
		return nil, fmt.Errorf("invalid user id: %d", userID)
	}

	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	settings, err := s.userSettingsRepo.GetByUserId(ctx, userID)
	if err != nil {
		// Settings not found is not a critical error, return user with nil settings
		s.logger.Warn("failed to get user settings", zap.Int("userID", userID), zap.Error(err))
		return &models.UserWithSettingsResponse{
			ID:       user.ID,
			Username: user.Username,
			Email:    user.Email,
			Role:     user.Role,
			Avatar:   user.Avatar,
			Active:   user.Active,
			Settings: nil,
			Message:  "Settings was not created",
		}, nil
	}

	return &models.UserWithSettingsResponse{
		ID:       user.ID,
		Username: user.Username,
		Email:    user.Email,
		Role:     user.Role,
		Active:   user.Active,
		Settings: settings,
		Avatar:   user.Avatar,
		Message:  "",
	}, nil
}

// CreateUser creates a new user with settings
func (s *adminService) CreateUser(ctx context.Context, user *models.CreateUserRequest, avatarFile multipart.File, avatarFilename string) (int, error) {
	// Check user credentials return normalized email and username
	normalizedEmail, normalizedUsername, err := checkRegisterCredentials(ctx, s.userRepo.(UserSharedRepository), user.Email, user.Username, user.Password)
	if err != nil {
		return 0, err
	}

	// Upload avatar if provided (before creating user to maintain transaction safety)
	var avatarURL string
	if avatarFile != nil {
		avatarURL, err = uploadAvatar(ctx, s.mediaBaseURL, s.apiKey, avatarFile, avatarFilename)
		if err != nil {
			return 0, fmt.Errorf("failed to upload avatar: %w", err)
		}
	}

	// Hash password
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return 0, fmt.Errorf("failed to hash password: %w", err)
	}

	// Create user
	userModel := &models.User{
		Username:     normalizedUsername,
		Email:        normalizedEmail,
		PasswordHash: string(passwordHash),
		Role:         user.Role,
		Avatar:       avatarURL,
		Active:       true,
	}

	if err := s.userRepo.Create(ctx, userModel); err != nil {
		return 0, err
	}

	// Create default user settings
	if err := s.userSettingsRepo.Create(ctx, userModel.ID); err != nil {
		// Return user but log the error - settings creation failure is not critical
		s.logger.Warn("failed to create user settings", zap.Error(err), zap.Int("userId", userModel.ID))
	}

	return userModel.ID, nil
}

// CreateUserSettings creates user settings for a user if they don't exist
func (s *adminService) CreateUserSettings(ctx context.Context, userID int) (string, error) {
	// Check if settings already exist
	_, err := s.userSettingsRepo.GetByUserId(ctx, userID)
	if err == nil {
		// Settings already exist
		return "Settings already exist", nil
	}

	// Create settings
	if err := s.userSettingsRepo.Create(ctx, userID); err != nil {
		return "", err
	}

	return "Settings created successfully", nil
}

// UpdateUserWithSettings updates a user and their settings
//
// We cannot ignore error about settings not exists forever, so that`s where we will signal admin that it is not good.
func (s *adminService) UpdateUserWithSettings(r *http.Request, userID int, userData *models.UpdateUserWithSettingsRequest, avatarFile multipart.File, avatarFilename string) error {
	if userID <= 0 {
		return fmt.Errorf("invalid user id")
	}

	// Get current user to check for existing avatar
	currentUser, err := s.userRepo.GetByID(r.Context(), userID)
	if err != nil {
		return fmt.Errorf("user not found")
	}

	// Handle avatar upload if provided (before other updates)
	var newAvatarURL string
	if avatarFile != nil && avatarFilename != "" {
		// Delete old avatar if it exists
		if currentUser.Avatar != "" && s.mediaBaseURL != "" && s.apiKey != "" {
			fileID := extractFileIDFromAvatarURL(currentUser.Avatar)
			if fileID != "" {
				if err := deleteAvatarFromMediaService(r.Context(), s.mediaBaseURL, s.apiKey, fileID); err != nil {
					return fmt.Errorf("failed to delete old avatar: %w", err)
				}
			}
		}

		// Upload new avatar
		uploadedURL, err := uploadAvatar(r.Context(), s.mediaBaseURL, s.apiKey, avatarFile, avatarFilename)
		if err != nil {
			return fmt.Errorf("failed to upload avatar: %w", err)
		}
		newAvatarURL = uploadedURL
	}

	// Check user credentials
	normalizedEmail, normalizedUsername, err := s.checkUpdateUserCredentials(r.Context(), userID, userData.Email, userData.Username, userData.Role, userData.Settings)
	if err != nil {
		return err
	}

	// Update user and settings if any data provided
	hasUserUpdate := normalizedUsername != "" || normalizedEmail != "" || userData.Role != nil || newAvatarURL != "" || userData.Active != nil
	hasSettingsUpdate := userData.Settings != nil &&
		(userData.Settings.Language != "" || userData.Settings.NewWordCount != nil ||
			userData.Settings.OldWordCount != nil || userData.Settings.AlphabetLearnCount != nil || userData.Settings.AlphabetRepeat != "")

	if hasUserUpdate || hasSettingsUpdate {
		// Create user model for update
		userDataModel := &models.User{
			ID:       userID,
			Username: normalizedUsername,
			Email:    normalizedEmail,
		}
		if userData.Role != nil {
			userDataModel.Role = *userData.Role
		}
		if newAvatarURL != "" {
			userDataModel.Avatar = newAvatarURL
		}

		// Create settings model for update
		var settingsData *models.UserSettings
		if hasSettingsUpdate {
			settingsData = &models.UserSettings{
				UserID: userID,
			}
			if userData.Settings != nil {
				settingsData.Language = userData.Settings.Language
				settingsData.AlphabetRepeat = userData.Settings.AlphabetRepeat
				if userData.Settings.NewWordCount != nil {
					settingsData.NewWordCount = *userData.Settings.NewWordCount
				}
				if userData.Settings.OldWordCount != nil {
					settingsData.OldWordCount = *userData.Settings.OldWordCount
				}
				if userData.Settings.AlphabetLearnCount != nil {
					settingsData.AlphabetLearnCount = *userData.Settings.AlphabetLearnCount
				}
			}
		}

		if err := s.userRepo.Update(r.Context(), userID, userDataModel, settingsData, userData.Active); err != nil {
			return err
		}

		if hasSettingsUpdate && settingsData.AlphabetRepeat != "" {
			// If new flag is "repeat", create scheduled task
			if settingsData.AlphabetRepeat == "repeat" {
				// Construct URL to drop marks endpoint
				dropMarksURL := fmt.Sprintf("%s/api/v6/test-results/drop-marks/%d", s.learnServiceBaseURL, userID)

				// Call task-service to create scheduled task
				return createScheduledTask(r.Context(), s.scheduledTaskBaseURL, s.apiKey, userID, "notify_alphabet", currentUser.Email, dropMarksURL, "0 0 * * *")
			} else {
				// If new flag is NOT "repeat", delete scheduled task
				return deleteScheduledTaskByUserID(r.Context(), s.scheduledTaskBaseURL, s.apiKey, userID)
			}
		}
	}

	return nil
}

// Method that checks the validity of the user's credentials for updating
//
// Almost the same as checkRegisterCredentials, but with optional fields, and role and settings checks.
func (s *adminService) checkUpdateUserCredentials(ctx context.Context, userID int, email, username string, role *models.Role, settings *models.UpdateUserSettingsRequest) (string, string, error) {
	// Validation errors objects
	validationErrors := make(chan error, 6)
	// Normalize email and username
	normalizedEmail := strings.TrimSpace(strings.ToLower(email))
	normalizedUsername := strings.TrimSpace(username)

	// Check email uniqueness and validity
	go func() {
		if normalizedEmail != "" {
			if !emailRegex.MatchString(normalizedEmail) {
				validationErrors <- fmt.Errorf("invalid email format")
				return
			}
			emailExists, err := s.userRepo.(UserSharedRepository).ExistsByEmail(ctx, normalizedEmail)
			if err != nil {
				validationErrors <- fmt.Errorf("failed to check email: %w", err)
				return
			}
			if emailExists {
				validationErrors <- fmt.Errorf("email already exists")
				return
			}
		}
		validationErrors <- nil
	}()

	// Check username uniqueness
	go func() {
		if normalizedUsername != "" {
			usernameExists, err := s.userRepo.(UserSharedRepository).ExistsByUsername(ctx, normalizedUsername)
			if err != nil {
				validationErrors <- fmt.Errorf("failed to check username: %w", err)
				return
			}
			if usernameExists {
				validationErrors <- fmt.Errorf("username already exists")
				return
			}
		}
		validationErrors <- nil
	}()

	// Check role validity
	go func() {
		if role != nil && (*role < models.RoleUser || *role > models.RoleAdmin) {
			validationErrors <- fmt.Errorf("invalid role")
			return
		}
		validationErrors <- nil
	}()

	// Check settings validity
	go func() {
		if settings != nil {
			if settings.Language != "" && settings.Language != models.LanguageEnglish &&
				settings.Language != models.LanguageRussian && settings.Language != models.LanguageGerman {
				validationErrors <- fmt.Errorf("invalid language")
				return
			}
			if settings.NewWordCount != nil && (*settings.NewWordCount < 1 || *settings.NewWordCount > 40) {
				validationErrors <- fmt.Errorf("invalid new word count")
				return
			}
			if settings.OldWordCount != nil && (*settings.OldWordCount < 1 || *settings.OldWordCount > 40) {
				validationErrors <- fmt.Errorf("invalid old word count")
				return
			}
			if settings.AlphabetLearnCount != nil && (*settings.AlphabetLearnCount < 5 || *settings.AlphabetLearnCount > 15) {
				validationErrors <- fmt.Errorf("invalid alphabet learn count")
				return
			}
			if settings.AlphabetRepeat != "" && settings.AlphabetRepeat != models.RepeatTypeInQuestion &&
				settings.AlphabetRepeat != models.RepeatTypeIgnore &&
				settings.AlphabetRepeat != models.RepeatTypeRepeat {
				validationErrors <- fmt.Errorf("invalid alphabet repeat")
				return
			}
		}
		validationErrors <- nil
	}()

	go func() {
		_, err := s.userSettingsRepo.GetByUserId(ctx, userID)
		if err != nil {
			validationErrors <- err
			return
		}
		validationErrors <- nil
	}()

	go func() {
		if settings.AlphabetRepeat != "" {
			if settings.AlphabetRepeat != "in question" && settings.AlphabetRepeat != "ignore" && settings.AlphabetRepeat != "repeat" {
				validationErrors <- fmt.Errorf("invalid alphabet repeat")
				return
			}
		}
		validationErrors <- nil
	}()

	for range 6 {
		err := <-validationErrors
		if err != nil {
			return "", "", fmt.Errorf("failed to check user credentials: %w", err)
		}
	}

	return normalizedEmail, normalizedUsername, nil
}

// DeleteUser deletes a user by ID
func (s *adminService) DeleteUser(ctx context.Context, userID int) error {
	if userID <= 0 {
		return fmt.Errorf("invalid user id")
	}

	userWithSettings, err := s.GetUserWithSettings(ctx, userID)
	if err != nil {
		return err
	}

	err = s.userRepo.Delete(ctx, userID)
	if err != nil {
		return err
	}

	// Delete avatar file from media service if avatar exists
	if userWithSettings.Avatar != "" && s.mediaBaseURL != "" && s.apiKey != "" {
		// Extract file ID from avatar URL (last part of the path)
		fileID := extractFileIDFromAvatarURL(userWithSettings.Avatar)
		if fileID != "" {
			if err := deleteAvatarFromMediaService(ctx, s.mediaBaseURL, s.apiKey, fileID); err != nil {
				return fmt.Errorf("avatar file has not been deleted: %w", err)
			}
		}
	}
	return nil
}

// extractFileIDFromAvatarURL extracts the file ID (filename) from the avatar URL
// The avatar URL format is expected to be like: http://.../media/avatar/{fileID}
// Returns the last part of the URL path as the file ID
func extractFileIDFromAvatarURL(avatarURL string) string {
	if avatarURL == "" {
		return ""
	}

	// Parse URL to handle it properly
	parsedURL, err := url.Parse(avatarURL)
	if err != nil {
		// If URL parsing fails, try to extract from string directly
		// Remove query parameters and fragments
		parts := strings.Split(avatarURL, "?")
		parts = strings.Split(parts[0], "#")
		urlPath := parts[0]

		// Extract the last part of the path by splitting on "/"
		pathParts := strings.Split(strings.Trim(urlPath, "/"), "/")
		if len(pathParts) > 0 {
			return pathParts[len(pathParts)-1]
		}
		return ""
	}

	// Extract the last part of the path
	pathParts := strings.Split(strings.Trim(parsedURL.Path, "/"), "/")
	if len(pathParts) > 0 {
		return pathParts[len(pathParts)-1]
	}

	return ""
}

// deleteAvatarFromMediaService sends a DELETE request to media service to delete the avatar file
func deleteAvatarFromMediaService(ctx context.Context, mediaBaseURL, apiKey, fileID string) error {
	if mediaBaseURL == "" || apiKey == "" {
		return nil // Skip if media service is not configured
	}

	// Construct the delete URL: {mediaBaseURL}/media/avatar/{fileID}
	deleteURL := strings.TrimSuffix(mediaBaseURL, "/") + "/media/avatar/" + fileID

	// Create DELETE request
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, deleteURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create delete request: %w", err)
	}

	// Set API key header
	req.Header.Set("X-API-Key", apiKey)

	// Execute request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send delete request: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusNotFound {
		return fmt.Errorf("media service returned status %d", resp.StatusCode)
	}

	return nil
}

// uploadAvatar uploads an avatar file to the media-service using io.Pipe for streaming
// It is a shared function for both auth and admin services.
func uploadAvatar(ctx context.Context, mediaBaseURL, apiKey string, avatarFile multipart.File, avatarFilename string) (string, error) {
	if mediaBaseURL == "" {
		return "", fmt.Errorf("MEDIA_BASE_URL is not configured")
	}
	if apiKey == "" {
		return "", fmt.Errorf("API_KEY is not configured")
	}

	// Create a pipe for streaming
	pr, pw := io.Pipe()
	defer pr.Close()

	// Create multipart writer
	writer := multipart.NewWriter(pw)

	// Start goroutine to write file to pipe
	errChan := make(chan error, 1)
	go func() {
		defer pw.Close()
		defer writer.Close()

		// Create form field for file
		part, err := writer.CreateFormFile("file", avatarFilename)
		if err != nil {
			errChan <- fmt.Errorf("failed to create form file: %w", err)
			return
		}

		// Copy file content to form
		_, err = io.Copy(part, avatarFile)
		if err != nil {
			errChan <- fmt.Errorf("failed to copy file: %w", err)
			return
		}

		errChan <- nil
	}()

	// Build upload URL
	uploadURL := fmt.Sprintf("%s/media/avatar", mediaBaseURL)

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, uploadURL, pr)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("X-API-Key", apiKey)

	// Make HTTP request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		// We will not wait for goroutine to complete. Instead it will finish when we close the pipe.
		return "", fmt.Errorf("failed to upload file: %w", err)
	}
	defer resp.Body.Close()

	// Check for errors from goroutine
	if err := <-errChan; err != nil {
		return "", err
	}

	// Check response status
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("media-service returned error: status %d, body: %s", resp.StatusCode, string(body))
	}

	// Read avatar URL from response
	avatarURL, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	return strings.TrimSpace(string(avatarURL)), nil
}

// GetTutorsList retrieves a list of tutors (only ID and username)
func (s *adminService) GetTutorsList(ctx context.Context) ([]models.TutorListItem, error) {
	return s.userRepo.GetTutorsList(ctx)
}

// UpdateUserPassword updates a user's password
func (s *adminService) UpdateUserPassword(ctx context.Context, userID int, password string) error {
	if userID <= 0 {
		return fmt.Errorf("invalid user id")
	}

	// Validate password against regex (passwordRegex is from auth_service.go in the same package)
	for _, regex := range passwordRegex {
		if !regex.MatchString(password) {
			return fmt.Errorf("password must be at least 8 characters long and contain at least one uppercase letter, one lowercase letter, one number, and one special character (!_?^&+-=|)")
		}
	}

	// Check if password contains ';' character (not allowed)
	if strings.Contains(password, ";") {
		return fmt.Errorf("password cannot contain ';' character")
	}

	// Hash password
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Update password hash
	err = s.userRepo.UpdatePasswordHash(ctx, userID, string(passwordHash))
	if err != nil {
		return err
	}

	return nil
}

// ScheduleTasks schedules tasks for admin
func (s *adminService) ScheduleTasks(ctx context.Context, tokenCleaningURL string) error {
	if s.taskBaseURL == "" {
		return fmt.Errorf("TASK_BASE_URL is not configured")
	}

	if s.apiKey == "" {
		return fmt.Errorf("API_KEY is not configured")
	}
	// Create request body
	reqBody := map[string]any{
		"user_id":    nil,
		"url":        tokenCleaningURL,
		"email_slug": "",
		"content":    "",
		"cron":       "0 0,12 * * *",
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request body: %w", err)
	}

	// Create HTTP POST request to task-service
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.taskBaseURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", s.apiKey)

	// Execute request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("task service returned status %d", resp.StatusCode)
	}

	return nil
}
