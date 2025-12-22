package services

import (
	"context"
	"fmt"
	"mime/multipart"
	"regexp"
	"strings"

	"github.com/japanesestudent/auth-service/internal/models"
	"github.com/japanesestudent/libs/auth/service"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

// UserSharedRepository is the interface that wraps methods for User table data access common for auth and admin services
type UserSharedRepository interface {
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
	// Method GetByID retrieves a user by ID.
	//
	// "userID" parameter is used to retrieve a user by ID.
	//
	// If user with such ID does not exist, the error will be returned together with "nil" value.
	GetByID(ctx context.Context, userID int) (*models.User, error)
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
	mediaBaseURL     string
	apiKey           string
}

// NewAuthService creates a new auth service
func NewAuthService(
	userRepo UserRepository,
	userTokenRepo UserTokenRepository,
	userSettingsRepo UserSettingsRepository,
	tokenGenerator *service.TokenGenerator,
	logger *zap.Logger,
	mediaBaseURL string,
	apiKey string,
) *authService {
	return &authService{
		userRepo:         userRepo,
		userTokenRepo:    userTokenRepo,
		userSettingsRepo: userSettingsRepo,
		tokenGenerator:   tokenGenerator,
		logger:           logger,
		mediaBaseURL:     mediaBaseURL,
		apiKey:           apiKey,
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
func (s *authService) Register(ctx context.Context, req *models.RegisterRequest, avatarFile multipart.File, avatarFilename string) (string, string, error) {
	// Check user credentials return normalized email and username
	normalizedEmail, normalizedUsername, err := checkRegisterCredentials(ctx, s.userRepo.(UserSharedRepository), req.Email, req.Username, req.Password)
	if err != nil {
		return "", "", err
	}

	// Upload avatar if provided (before creating user to maintain transaction safety)
	var avatarURL string
	if avatarFile != nil {
		avatarURL, err = uploadAvatar(ctx, s.mediaBaseURL, s.apiKey, avatarFile, avatarFilename)
		if err != nil {
			return "", "", fmt.Errorf("failed to upload avatar: %w", err)
		}
	}

	// Hash password
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return "", "", err
	}

	// Create user
	user := &models.User{
		Username:     normalizedUsername,
		Email:        normalizedEmail,
		PasswordHash: string(passwordHash),
		Role:         models.RoleUser, // Default role
		Avatar:       avatarURL,
	}

	err = s.userRepo.Create(ctx, user)
	if err != nil {
		return "", "", err
	}

	// Create default user settings
	// We don`t want to break the flow of the registration process, so we create user settings in a separate goroutine.
	// If some error occurs during user settings creation, we will log it and continue the registration process.
	go func() {
		if err := s.userSettingsRepo.Create(ctx, user.ID); err != nil {
			s.logger.Warn("failed to create user settings", zap.Int("userId", user.ID), zap.Error(err))
		}
	}()

	// Generate and save access and refresh tokens
	return generateAndSaveTokens(ctx, s.tokenGenerator, s.userTokenRepo, user.ID, user.Role)
}

// Login authenticates a user
func (s *authService) Login(ctx context.Context, req *models.LoginRequest) (string, string, error) {
	req.Login = strings.TrimSpace(req.Login)
	if req.Login == "" {
		return "", "", fmt.Errorf("login cannot be empty")
	}

	if req.Password == "" {
		return "", "", fmt.Errorf("password cannot be empty")
	}

	// Get user by email or username
	user, err := s.userRepo.GetByEmailOrUsername(ctx, req.Login)
	if err != nil {
		return "", "", err
	}

	// Verify password
	if err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return "", "", fmt.Errorf("invalid credentials")
	}

	// Generate and save access and refresh tokens
	return generateAndSaveTokens(ctx, s.tokenGenerator, s.userTokenRepo, user.ID, user.Role)
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
			return "", "", err
		}
	}
	userToken := <-userTokenChan
	if userToken == nil {
		return "", "", fmt.Errorf("failed to refresh token: failed to get user token")
	}

	// Get user to retrieve role
	user, err := s.userRepo.GetByID(ctx, userToken.UserID)
	if err != nil {
		return "", "", err
	}

	// Generate new tokens using userToken.UserID to ensure consistency with the token in database
	accessToken, newRefreshToken, err := s.tokenGenerator.GenerateTokens(userToken.UserID, int(user.Role))
	if err != nil {
		return "", "", err
	}

	// Update refresh token in database (replaces old token with new one)
	// Use userToken.UserID to ensure it matches the token that was retrieved from database
	if err := s.userTokenRepo.UpdateToken(ctx, refreshToken, newRefreshToken, userToken.UserID); err != nil {
		return "", "", err
	}

	return accessToken, newRefreshToken, nil
}

// Below is the methods with simple logic and shared between auth and admin services

// Method that generates and saves access and refresh tokens
//
// In current implementation of authentication, we do not track the lifespan or count of refresh tokens,
// we will make it possible in the future.
func generateAndSaveTokens(ctx context.Context, tokenGenerator *service.TokenGenerator,
	userTokenRepo UserTokenRepository, userID int, role models.Role) (string, string, error) {
	// Generate tokens
	accessToken, refreshToken, err := tokenGenerator.GenerateTokens(userID, int(role))
	if err != nil {
		return "", "", fmt.Errorf("failed to generate tokens: %w", err)
	}

	// Save refresh token
	userToken := &models.UserToken{
		UserID: userID,
		Token:  refreshToken,
	}
	if err := userTokenRepo.Create(ctx, userToken); err != nil {
		return "", "", fmt.Errorf("failed to save refresh token: %w", err)
	}

	return accessToken, refreshToken, nil
}

// Method that combines all checks for register credentials
//
// There is no need for check parts to wait each other, so I`m using goroutines to check all
// credentials in parallel to improve performance.
func checkRegisterCredentials(ctx context.Context, userRepo UserSharedRepository, email, username, password string) (string, string, error) {
	// Validation errors objects
	validationErrors := make(chan error, 3)
	// Normalize email and username
	normalizedEmail := strings.TrimSpace(strings.ToLower(email))
	normalizedUsername := strings.TrimSpace(username)

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

	// Validate email and check its uniqueness
	go func() {
		if !emailRegex.MatchString(normalizedEmail) {
			validationErrors <- fmt.Errorf("invalid email format")
			return
		}
		emailExists, err := userRepo.ExistsByEmail(ctx, normalizedEmail)
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

	// Validate username and check its uniqueness
	go func() {
		if normalizedUsername == "" {
			validationErrors <- fmt.Errorf("username cannot be empty")
			return
		}
		usernameExists, err := userRepo.ExistsByUsername(ctx, normalizedUsername)
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

	for range 3 {
		err := <-validationErrors
		if err != nil {
			return "", "", fmt.Errorf("failed to check user credentials: %w", err)
		}
	}

	return normalizedEmail, normalizedUsername, nil
}
