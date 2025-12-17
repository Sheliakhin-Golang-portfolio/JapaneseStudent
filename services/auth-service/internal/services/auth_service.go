package services

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/japanesestudent/auth-service/internal/models"
	"github.com/japanesestudent/libs/auth/service"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

// UserRepository is the interface that wraps methods for User table data access
type UserRepository interface {
	// Method Create inserts a new user into the database.
	//
	// "user" parameter is used to create a new user.
	//
	// If some error occurs during user creation, the error will be returned together with "nil" value.
	Create(ctx context.Context, user *models.User) error
	// Method GetByEmailOrUsername retrieves a user by email or username.
	//
	// "login" parameter is used to retrieve a user by email or username.
	//
	// If user with such email or username does not exist, the error will be returned together with "nil" value.
	GetByEmailOrUsername(ctx context.Context, login string) (*models.User, error)
	// Method ExistsByEmail checks if a user with such email exists.
	//
	// "email" parameter is used to check if a user with such email exists.
	//
	// If some error occurs during check, the error will be returned together with "false" value.
	ExistsByEmail(ctx context.Context, email string) (bool, error)
	// Method ExistsByUsername checks if a user with such username exists.
	//
	// "username" parameter is used to check if a user with such username exists.
	//
	// If some error occurs during check, the error will be returned together with "false" value.
	ExistsByUsername(ctx context.Context, username string) (bool, error)
}

// UserTokenRepository is the interface that wraps methods for UserToken table data access
type UserTokenRepository interface {
	// Method Create inserts a new user token into the database.
	//
	// "userToken" parameter is used to create a new user token.
	//
	// If some error occurs during user token creation, the error will be returned together with "nil" value.
	Create(ctx context.Context, userToken *models.UserToken) error
	// Method GetByToken retrieves a user token by token string.
	//
	// "token" parameter is used to retrieve a user token by token string.
	//
	// If user token with such token does not exist, the error will be returned together with "nil" value.
	GetByToken(ctx context.Context, token string) (*models.UserToken, error)
	// Method UpdateToken updates a user token by old token string and new token string.
	//
	// "oldToken" parameter is used to update a user token by old token string.
	// "newToken" parameter is used to update a user token by new token string.
	// "userID" parameter is used to update a user token by user ID.
	//
	// If some error occurs during user token update, the error will be returned together with "nil" value.
	UpdateToken(ctx context.Context, oldToken, newToken string, userID int) error
	// Method DeleteByToken deletes a user token by token string.
	//
	// "token" parameter is used to delete a user token by token string.
	//
	// If some error occurs during user token deletion, the error will be returned together with "nil" value.
	DeleteByToken(ctx context.Context, token string) error
}

// authService implements AuthService
type authService struct {
	userRepo         UserRepository
	userTokenRepo    UserTokenRepository
	userSettingsRepo UserSettingsRepository
	tokenGenerator   *service.TokenGenerator
	logger           *zap.Logger
}

// NewAuthService creates a new auth service
func NewAuthService(
	userRepo UserRepository,
	userTokenRepo UserTokenRepository,
	userSettingsRepo UserSettingsRepository,
	tokenGenerator *service.TokenGenerator,
	logger *zap.Logger,
) *authService {
	return &authService{
		userRepo:         userRepo,
		userTokenRepo:    userTokenRepo,
		userSettingsRepo: userSettingsRepo,
		tokenGenerator:   tokenGenerator,
		logger:           logger,
	}
}

// emailRegex validates email format
var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

// passwordRegex validates password: at least 8 chars, uppercase, lowercase, number, special: !_?^&+-=|
var passwordRegex = []*regexp.Regexp{
	regexp.MustCompile(`.{8,}`),
	regexp.MustCompile(`[a-z]`),
	regexp.MustCompile(`[A-Z]`),
	regexp.MustCompile(`[0-9]`),
	regexp.MustCompile(`[!_?^&+\-=|]`),
}

// Register creates a new user account
func (s *authService) Register(ctx context.Context, email, username, password string) (string, string, error) {
	// Check user credentials return normalized email and username
	normalizedEmail, normalizedUsername, err := s.checkRegisterCredentials(ctx, email, username, password)
	if err != nil {
		return "", "", err
	}

	// Hash password
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", "", fmt.Errorf("failed to hash password: %w", err)
	}

	// Create user
	user := &models.User{
		Username:     normalizedUsername,
		Email:        normalizedEmail,
		PasswordHash: string(passwordHash),
		Role:         models.RoleUser, // Default role
	}

	err = s.userRepo.Create(ctx, user)
	if err != nil {
		return "", "", fmt.Errorf("failed to create user: %w", err)
	}

	// Create default user settings
	// We don`t want to break the flow of the registration process, so we create user settings in a separate goroutine.
	// If some error occurs during user settings creation, we will log it and continue the registration process.
	go func() {
		userSettings := &models.UserSettings{
			UserID: user.ID,
		}
		if err := s.userSettingsRepo.Create(ctx, userSettings); err != nil {
			s.logger.Error("failed to create user settings", zap.Error(err), zap.Int("userId", user.ID))
		}
	}()

	// Generate and save access and refresh tokens
	return s.generateAndSaveTokens(ctx, user.ID)
}

// Method that combines all checks for register credentials
//
// There is no need for check parts to wait each other, so I`m using goroutines to check all
// credentials in parallel to improve performance.
func (s *authService) checkRegisterCredentials(ctx context.Context, email, username, password string) (string, string, error) {
	// Validation errors objects
	validationErrors := make(chan error, 5)
	// Normalize email and username
	normalizedEmail := strings.TrimSpace(strings.ToLower(email))
	normalizedUsername := strings.TrimSpace(username)
	// Validate email
	go func() {
		if !emailRegex.MatchString(normalizedEmail) {
			validationErrors <- fmt.Errorf("invalid email format")
			return
		}
		validationErrors <- nil
	}()

	// Validate password
	go func() {
		for _, regex := range passwordRegex {
			if !regex.MatchString(password) {
				validationErrors <- fmt.Errorf("password must be at least 8 characters long and contain at least one uppercase letter, one lowercase letter, one number, and one special character (!_?^&+-=|)")
				return
			}
		}
		validationErrors <- nil
	}()

	// Validate username
	go func() {
		if normalizedUsername == "" {
			validationErrors <- fmt.Errorf("username cannot be empty")
			return
		}
		validationErrors <- nil
	}()

	// Check email uniqueness
	go func() {
		emailExists, err := s.userRepo.ExistsByEmail(ctx, normalizedEmail)
		if err != nil {
			validationErrors <- fmt.Errorf("failed to check email: %w", err)
			return
		}
		if emailExists {
			validationErrors <- fmt.Errorf("email already exists")
			return
		}
		validationErrors <- nil
	}()

	// Check username uniqueness
	go func() {
		usernameExists, err := s.userRepo.ExistsByUsername(ctx, normalizedUsername)
		if err != nil {
			validationErrors <- fmt.Errorf("failed to check username: %w", err)
			return
		}
		if usernameExists {
			validationErrors <- fmt.Errorf("username already exists")
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

// Method that generates and saves access and refresh tokens
//
// In current implementation of authentication, we do not track the lifespan or count of refresh tokens,
// we will make it possible in the future.
func (s *authService) generateAndSaveTokens(ctx context.Context, userID int) (string, string, error) {
	// Generate tokens
	accessToken, refreshToken, err := s.tokenGenerator.GenerateTokens(userID)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate tokens: %w", err)
	}

	// Save refresh token
	userToken := &models.UserToken{
		UserID: userID,
		Token:  refreshToken,
	}
	if err := s.userTokenRepo.Create(ctx, userToken); err != nil {
		return "", "", fmt.Errorf("failed to save refresh token: %w", err)
	}

	return accessToken, refreshToken, nil
}

// Login authenticates a user
func (s *authService) Login(ctx context.Context, login, password string) (string, string, error) {
	login = strings.TrimSpace(login)
	if login == "" {
		return "", "", fmt.Errorf("login cannot be empty")
	}

	if password == "" {
		return "", "", fmt.Errorf("password cannot be empty")
	}

	// Get user by email or username
	user, err := s.userRepo.GetByEmailOrUsername(ctx, login)
	if err != nil {
		return "", "", fmt.Errorf("invalid credentials")
	}

	// Verify password
	if err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", "", fmt.Errorf("invalid credentials")
	}

	// Generate and save access and refresh tokens
	return s.generateAndSaveTokens(ctx, user.ID)
}

// Refresh refreshes a user's access token
//
// There is no need for check parts to wait each other (because DELETE operation does not return error on 0 rows deleted),
// so I`m using goroutines to check validation parts in parallel to improve performance.
func (s *authService) Refresh(ctx context.Context, refreshToken string) (string, string, error) {
	refreshToken = strings.TrimSpace(refreshToken)
	errorChan := make(chan error, 2)
	userTokenChan := make(chan *models.UserToken, 1) // Buffered to prevent goroutine leak

	// Check if user token exists in database and return it
	go func() {
		userToken, err := s.userTokenRepo.GetByToken(ctx, refreshToken)
		if err != nil {
			errorChan <- fmt.Errorf("failed to get user token by refresh token: %w", err)
			userTokenChan <- nil
			return
		}
		userTokenChan <- userToken
		errorChan <- nil // Signal success
	}()

	// Validate refresh token
	go func() {
		if err := s.tokenGenerator.ValidateRefreshToken(refreshToken); err != nil {
			errorChan <- fmt.Errorf("invalid or expired refresh token")
			s.logger.Error("invalid refresh token", zap.Error(err))
			// Delete token if it exists in database
			s.userTokenRepo.DeleteByToken(ctx, refreshToken)
			return
		}
		errorChan <- nil // Signal success
	}()

	// Wait for both operations to complete
	for range 2 {
		err := <-errorChan
		if err != nil {
			return "", "", fmt.Errorf("failed to refresh token: %w", err)
		}
	}
	userToken := <-userTokenChan
	if userToken == nil {
		return "", "", fmt.Errorf("failed to refresh token: failed to get user token")
	}

	// Generate new tokens
	accessToken, newRefreshToken, err := s.tokenGenerator.GenerateTokens(userToken.UserID)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate tokens: %w", err)
	}

	// Update refresh token in database (replaces old token with new one)
	if err := s.userTokenRepo.UpdateToken(ctx, refreshToken, newRefreshToken, userToken.UserID); err != nil {
		return "", "", fmt.Errorf("failed to update refresh token: %w", err)
	}

	return accessToken, newRefreshToken, nil
}
