package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/japanesestudent/auth-service/internal/models"
	"github.com/japanesestudent/libs/auth/service"
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
	//
	// If some error occurs, the error will be returned.
	Update(ctx context.Context, userID int, user *models.User, settings *models.UserSettings) error
	// Method Delete deletes a user by ID.
	//
	// "userID" parameter is used to identify the user to delete.
	//
	// If some error occurs, the error will be returned.
	Delete(ctx context.Context, userID int) error
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
	userRepo         AdminUserRepository
	userTokenRepo    AdminUserTokenRepository
	userSettingsRepo UserSettingsRepository
	tokenGenerator   *service.TokenGenerator
	logger           *zap.Logger
}

// NewAuthService creates a new auth service
func NewAdminService(
	userRepo AdminUserRepository,
	userTokenRepo AdminUserTokenRepository,
	userSettingsRepo UserSettingsRepository,
	tokenGenerator *service.TokenGenerator,
	logger *zap.Logger,
) *adminService {
	return &adminService{
		userRepo:         userRepo,
		userTokenRepo:    userTokenRepo,
		userSettingsRepo: userSettingsRepo,
		tokenGenerator:   tokenGenerator,
		logger:           logger,
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
			Settings: nil,
			Message:  "Settings was not created",
		}, nil
	}

	return &models.UserWithSettingsResponse{
		ID:       user.ID,
		Username: user.Username,
		Email:    user.Email,
		Role:     user.Role,
		Settings: settings,
		Message:  "",
	}, nil
}

// CreateUser creates a new user with settings
func (s *adminService) CreateUser(ctx context.Context, user *models.CreateUserRequest) (int, error) {
	// Check user credentials return normalized email and username
	normalizedEmail, normalizedUsername, err := checkRegisterCredentials(ctx, s.userRepo.(UserSharedRepository), user.Email, user.Username, user.Password)
	if err != nil {
		return 0, err
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
func (s *adminService) UpdateUserWithSettings(ctx context.Context, userID int, userData *models.UpdateUserWithSettingsRequest) error {
	if userID <= 0 {
		return fmt.Errorf("invalid user id")
	}

	// Check user credentials
	normalizedEmail, normalizedUsername, err := s.checkUpdateUserCredentials(ctx, userID, userData.Email, userData.Username, userData.Role, userData.Settings)
	if err != nil {
		return err
	}

	// Update user and settings if any data provided
	if normalizedUsername != "" || normalizedEmail != "" || userData.Role != nil || (userData.Settings != nil &&
		userData.Settings.Language != nil && userData.Settings.NewWordCount != nil &&
		userData.Settings.OldWordCount != nil && userData.Settings.AlphabetLearnCount != nil) {
		// Create user model for update
		userDataModel := &models.User{
			ID:       userID,
			Username: normalizedUsername,
			Email:    normalizedEmail,
		}
		if userData.Role != nil {
			userDataModel.Role = *userData.Role
		}

		// Create settings model for update
		settingsData := &models.UserSettings{
			UserID: userID,
		}
		if userData.Settings != nil {
			if userData.Settings.Language != nil {
				settingsData.Language = *userData.Settings.Language
			}
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

		return s.userRepo.Update(ctx, userID, userDataModel, settingsData)
	}

	return nil
}

// Method that checks the validity of the user's credentials for updating
//
// Almost the same as checkRegisterCredentials, but with optional fields, and role and settings checks.
func (s *adminService) checkUpdateUserCredentials(ctx context.Context, userID int, email, username string, role *models.Role, settings *models.UpdateUserSettingsRequest) (string, string, error) {
	// Validation errors objects
	validationErrors := make(chan error, 5)
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
			if settings.Language != nil && *settings.Language != models.LanguageEnglish &&
				*settings.Language != models.LanguageRussian && *settings.Language != models.LanguageGerman {
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

	for range 5 {
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

	err := s.userRepo.Delete(ctx, userID)
	if err != nil {
		return err
	}
	return nil
}
