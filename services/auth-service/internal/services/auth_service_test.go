package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/japanesestudent/auth-service/internal/models"
	"github.com/japanesestudent/libs/auth/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

// mockUserRepository is a mock implementation of UserRepository
type mockUserRepository struct {
	user                   *models.User
	err                    error
	existsByEmailResult    bool
	existsByEmailError     error
	existsByUsernameResult bool
	existsByUsernameError  error
}

func (m *mockUserRepository) Create(ctx context.Context, user *models.User) error {
	if m.err != nil {
		return m.err
	}
	user.ID = 1
	return nil
}

func (m *mockUserRepository) GetByEmailOrUsername(ctx context.Context, login string) (*models.User, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.user, nil
}

func (m *mockUserRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	if m.existsByEmailError != nil {
		return false, m.existsByEmailError
	}
	return m.existsByEmailResult, nil
}

func (m *mockUserRepository) ExistsByUsername(ctx context.Context, username string) (bool, error) {
	if m.existsByUsernameError != nil {
		return false, m.existsByUsernameError
	}
	return m.existsByUsernameResult, nil
}

// mockUserTokenRepository is a mock implementation of UserTokenRepository
type mockUserTokenRepository struct {
	token          *models.UserToken
	err            error
	updateTokenErr error
}

func (m *mockUserTokenRepository) Create(ctx context.Context, userToken *models.UserToken) error {
	return m.err
}

func (m *mockUserTokenRepository) GetByToken(ctx context.Context, token string) (*models.UserToken, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.token, nil
}

func (m *mockUserTokenRepository) UpdateToken(ctx context.Context, oldToken, newToken string, userID int) error {
	if m.updateTokenErr != nil {
		return m.updateTokenErr
	}
	return m.err
}

func (m *mockUserTokenRepository) DeleteByToken(ctx context.Context, token string) error {
	return m.err
}

func TestNewAuthService(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	userRepo := &mockUserRepository{}
	tokenRepo := &mockUserTokenRepository{}
	tokenGen := service.NewTokenGenerator("secret", 0, 0)

	svc := NewAuthService(userRepo, tokenRepo, tokenGen, logger)

	assert.NotNil(t, svc)
	assert.Equal(t, userRepo, svc.userRepo)
	assert.Equal(t, tokenRepo, svc.userTokenRepo)
	assert.Equal(t, tokenGen, svc.tokenGenerator)
	assert.Equal(t, logger, svc.logger)
}

func TestAuthService_Register(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	tokenGen := service.NewTokenGenerator("test-secret", 1, 1)

	tests := []struct {
		name          string
		email         string
		username      string
		password      string
		userRepo      *mockUserRepository
		tokenRepo     *mockUserTokenRepository
		expectedError bool
		errorContains string
	}{
		{
			name:     "success",
			email:    "test@example.com",
			username: "testuser",
			password: "Password123!",
			userRepo: &mockUserRepository{
				existsByEmailResult:    false,
				existsByUsernameResult: false,
			},
			tokenRepo:     &mockUserTokenRepository{},
			expectedError: false,
		},
		{
			name:     "invalid email format - missing @",
			email:    "invalid-email",
			username: "testuser",
			password: "Password123!",
			userRepo: &mockUserRepository{
				existsByEmailResult:    false,
				existsByUsernameResult: false,
			},
			tokenRepo:     &mockUserTokenRepository{},
			expectedError: true,
			errorContains: "invalid email format",
		},
		{
			name:     "invalid email format - missing domain",
			email:    "test@",
			username: "testuser",
			password: "Password123!",
			userRepo: &mockUserRepository{
				existsByEmailResult:    false,
				existsByUsernameResult: false,
			},
			tokenRepo:     &mockUserTokenRepository{},
			expectedError: true,
			errorContains: "invalid email format",
		},
		{
			name:     "invalid email format - missing local part",
			email:    "@example.com",
			username: "testuser",
			password: "Password123!",
			userRepo: &mockUserRepository{
				existsByEmailResult:    false,
				existsByUsernameResult: false,
			},
			tokenRepo:     &mockUserTokenRepository{},
			expectedError: true,
			errorContains: "invalid email format",
		},
		{
			name:     "password too short",
			email:    "test@example.com",
			username: "testuser",
			password: "Pass1!",
			userRepo: &mockUserRepository{
				existsByEmailResult:    false,
				existsByUsernameResult: false,
			},
			tokenRepo:     &mockUserTokenRepository{},
			expectedError: true,
			errorContains: "password must be at least 8 characters",
		},
		{
			name:     "password missing uppercase",
			email:    "test@example.com",
			username: "testuser",
			password: "password123!",
			userRepo: &mockUserRepository{
				existsByEmailResult:    false,
				existsByUsernameResult: false,
			},
			tokenRepo:     &mockUserTokenRepository{},
			expectedError: true,
			errorContains: "password must be at least 8 characters",
		},
		{
			name:     "password missing lowercase",
			email:    "test@example.com",
			username: "testuser",
			password: "PASSWORD123!",
			userRepo: &mockUserRepository{
				existsByEmailResult:    false,
				existsByUsernameResult: false,
			},
			tokenRepo:     &mockUserTokenRepository{},
			expectedError: true,
			errorContains: "password must be at least 8 characters",
		},
		{
			name:     "password missing number",
			email:    "test@example.com",
			username: "testuser",
			password: "Password!",
			userRepo: &mockUserRepository{
				existsByEmailResult:    false,
				existsByUsernameResult: false,
			},
			tokenRepo:     &mockUserTokenRepository{},
			expectedError: true,
			errorContains: "password must be at least 8 characters",
		},
		{
			name:     "password missing special character",
			email:    "test@example.com",
			username: "testuser",
			password: "Password123",
			userRepo: &mockUserRepository{
				existsByEmailResult:    false,
				existsByUsernameResult: false,
			},
			tokenRepo:     &mockUserTokenRepository{},
			expectedError: true,
			errorContains: "password must be at least 8 characters",
		},
		{
			name:     "password with invalid special character",
			email:    "test@example.com",
			username: "testuser",
			password: "Password123#",
			userRepo: &mockUserRepository{
				existsByEmailResult:    false,
				existsByUsernameResult: false,
			},
			tokenRepo:     &mockUserTokenRepository{},
			expectedError: true,
			errorContains: "password must be at least 8 characters",
		},
		{
			name:     "empty username",
			email:    "test@example.com",
			username: "",
			password: "Password123!",
			userRepo: &mockUserRepository{
				existsByEmailResult:    false,
				existsByUsernameResult: false,
			},
			tokenRepo:     &mockUserTokenRepository{},
			expectedError: true,
			errorContains: "username cannot be empty",
		},
		{
			name:     "email already exists",
			email:    "existing@example.com",
			username: "testuser",
			password: "Password123!",
			userRepo: &mockUserRepository{
				existsByEmailResult:    true,
				existsByUsernameResult: false,
			},
			tokenRepo:     &mockUserTokenRepository{},
			expectedError: true,
			errorContains: "email already exists",
		},
		{
			name:     "username already exists",
			email:    "test@example.com",
			username: "existinguser",
			password: "Password123!",
			userRepo: &mockUserRepository{
				existsByEmailResult:    false,
				existsByUsernameResult: true,
			},
			tokenRepo:     &mockUserTokenRepository{},
			expectedError: true,
			errorContains: "username already exists",
		},
		{
			name:     "email normalization - uppercase and spaces",
			email:    "  TEST@EXAMPLE.COM  ",
			username: "testuser",
			password: "Password123!",
			userRepo: &mockUserRepository{
				existsByEmailResult:    false,
				existsByUsernameResult: false,
			},
			tokenRepo:     &mockUserTokenRepository{},
			expectedError: false,
		},
		{
			name:     "username trimming - leading and trailing spaces",
			email:    "test@example.com",
			username: "  testuser  ",
			password: "Password123!",
			userRepo: &mockUserRepository{
				existsByEmailResult:    false,
				existsByUsernameResult: false,
			},
			tokenRepo:     &mockUserTokenRepository{},
			expectedError: false,
		},
		{
			name:     "database error on user creation",
			email:    "test@example.com",
			username: "testuser",
			password: "Password123!",
			userRepo: &mockUserRepository{
				existsByEmailResult:    false,
				existsByUsernameResult: false,
				err:                    errors.New("database error"),
			},
			tokenRepo:     &mockUserTokenRepository{},
			expectedError: true,
			errorContains: "failed to create user",
		},
		{
			name:     "database error on token creation",
			email:    "test@example.com",
			username: "testuser",
			password: "Password123!",
			userRepo: &mockUserRepository{
				existsByEmailResult:    false,
				existsByUsernameResult: false,
			},
			tokenRepo: &mockUserTokenRepository{
				err: errors.New("token creation error"),
			},
			expectedError: true,
			errorContains: "failed to save refresh token",
		},
		{
			name:     "database error checking email",
			email:    "test@example.com",
			username: "testuser",
			password: "Password123!",
			userRepo: &mockUserRepository{
				existsByEmailError:     errors.New("database error"),
				existsByUsernameResult: false,
			},
			tokenRepo:     &mockUserTokenRepository{},
			expectedError: true,
			errorContains: "failed to check email",
		},
		{
			name:     "database error checking username",
			email:    "test@example.com",
			username: "testuser",
			password: "Password123!",
			userRepo: &mockUserRepository{
				existsByEmailResult:   false,
				existsByUsernameError: errors.New("database error"),
			},
			tokenRepo:     &mockUserTokenRepository{},
			expectedError: true,
			errorContains: "failed to check username",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewAuthService(tt.userRepo, tt.tokenRepo, tokenGen, logger)

			accessToken, refreshToken, err := svc.Register(context.Background(), tt.email, tt.username, tt.password)

			if tt.expectedError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
				assert.Empty(t, accessToken)
				assert.Empty(t, refreshToken)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, accessToken)
				assert.NotEmpty(t, refreshToken)
			}
		})
	}
}

func TestAuthService_Login(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	tokenGen := service.NewTokenGenerator("test-secret", 1, 1)

	// Create a valid password hash for testing
	validPasswordHash, _ := bcrypt.GenerateFromPassword([]byte("Password123!"), bcrypt.DefaultCost)

	tests := []struct {
		name          string
		login         string
		password      string
		userRepo      *mockUserRepository
		tokenRepo     *mockUserTokenRepository
		expectedError bool
		errorContains string
	}{
		{
			name:     "success with email",
			login:    "test@example.com",
			password: "Password123!",
			userRepo: &mockUserRepository{
				user: &models.User{
					ID:           1,
					Email:        "test@example.com",
					Username:     "testuser",
					PasswordHash: string(validPasswordHash),
					Role:         models.RoleUser,
				},
			},
			tokenRepo:     &mockUserTokenRepository{},
			expectedError: false,
		},
		{
			name:     "success with username",
			login:    "testuser",
			password: "Password123!",
			userRepo: &mockUserRepository{
				user: &models.User{
					ID:           1,
					Email:        "test@example.com",
					Username:     "testuser",
					PasswordHash: string(validPasswordHash),
					Role:         models.RoleUser,
				},
			},
			tokenRepo:     &mockUserTokenRepository{},
			expectedError: false,
		},
		{
			name:     "empty login",
			login:    "",
			password: "Password123!",
			userRepo: &mockUserRepository{
				user: &models.User{
					ID:           1,
					Email:        "test@example.com",
					Username:     "testuser",
					PasswordHash: string(validPasswordHash),
					Role:         models.RoleUser,
				},
			},
			tokenRepo:     &mockUserTokenRepository{},
			expectedError: true,
			errorContains: "login cannot be empty",
		},
		{
			name:     "empty password",
			login:    "test@example.com",
			password: "",
			userRepo: &mockUserRepository{
				user: &models.User{
					ID:           1,
					Email:        "test@example.com",
					Username:     "testuser",
					PasswordHash: string(validPasswordHash),
					Role:         models.RoleUser,
				},
			},
			tokenRepo:     &mockUserTokenRepository{},
			expectedError: true,
			errorContains: "password cannot be empty",
		},
		{
			name:     "user not found",
			login:    "nonexistent@example.com",
			password: "Password123!",
			userRepo: &mockUserRepository{
				err: errors.New("user not found"),
			},
			tokenRepo:     &mockUserTokenRepository{},
			expectedError: true,
			errorContains: "invalid credentials",
		},
		{
			name:     "wrong password",
			login:    "test@example.com",
			password: "WrongPassword123!",
			userRepo: &mockUserRepository{
				user: &models.User{
					ID:           1,
					Email:        "test@example.com",
					Username:     "testuser",
					PasswordHash: string(validPasswordHash),
					Role:         models.RoleUser,
				},
			},
			tokenRepo:     &mockUserTokenRepository{},
			expectedError: true,
			errorContains: "invalid credentials",
		},
		{
			name:     "login with spaces trimmed",
			login:    "  test@example.com  ",
			password: "Password123!",
			userRepo: &mockUserRepository{
				user: &models.User{
					ID:           1,
					Email:        "test@example.com",
					Username:     "testuser",
					PasswordHash: string(validPasswordHash),
					Role:         models.RoleUser,
				},
			},
			tokenRepo:     &mockUserTokenRepository{},
			expectedError: false,
		},
		{
			name:     "database error",
			login:    "test@example.com",
			password: "Password123!",
			userRepo: &mockUserRepository{
				err: errors.New("database error"),
			},
			tokenRepo:     &mockUserTokenRepository{},
			expectedError: true,
			errorContains: "invalid credentials",
		},
		{
			name:     "token generation failure",
			login:    "test@example.com",
			password: "Password123!",
			userRepo: &mockUserRepository{
				user: &models.User{
					ID:           1,
					Email:        "test@example.com",
					Username:     "testuser",
					PasswordHash: string(validPasswordHash),
					Role:         models.RoleUser,
				},
			},
			tokenRepo: &mockUserTokenRepository{
				err: errors.New("token creation error"),
			},
			expectedError: true,
			errorContains: "failed to save refresh token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewAuthService(tt.userRepo, tt.tokenRepo, tokenGen, logger)

			accessToken, refreshToken, err := svc.Login(context.Background(), tt.login, tt.password)

			if tt.expectedError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
				assert.Empty(t, accessToken)
				assert.Empty(t, refreshToken)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, accessToken)
				assert.NotEmpty(t, refreshToken)
			}
		})
	}
}

func TestAuthService_Refresh(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	tokenGen := service.NewTokenGenerator("test-secret", 1*time.Hour, 1*time.Hour)

	// Generate a valid refresh token for testing
	_, validRefreshToken, _ := tokenGen.GenerateTokens(1)

	tests := []struct {
		name          string
		refreshToken  string
		userRepo      *mockUserRepository
		tokenRepo     *mockUserTokenRepository
		expectedError bool
		errorContains string
	}{
		{
			name:         "success",
			refreshToken: validRefreshToken,
			userRepo:     &mockUserRepository{},
			tokenRepo: &mockUserTokenRepository{
				token: &models.UserToken{
					ID:     1,
					UserID: 1,
					Token:  validRefreshToken,
				},
			},
			expectedError: false,
		},
		{
			name:          "empty token",
			refreshToken:  "",
			userRepo:      &mockUserRepository{},
			tokenRepo:     &mockUserTokenRepository{},
			expectedError: true,
			errorContains: "failed to refresh token",
		},
		{
			name:          "whitespace token",
			refreshToken:  "   ",
			userRepo:      &mockUserRepository{},
			tokenRepo:     &mockUserTokenRepository{},
			expectedError: true,
			errorContains: "failed to refresh token",
		},
		{
			name:         "token not in database",
			refreshToken: validRefreshToken,
			userRepo:     &mockUserRepository{},
			tokenRepo: &mockUserTokenRepository{
				err: errors.New("token not found"),
			},
			expectedError: true,
			errorContains: "failed to get user token",
		},
		{
			name:         "invalid token format",
			refreshToken: "invalid-token-format",
			userRepo:     &mockUserRepository{},
			tokenRepo: &mockUserTokenRepository{
				token: &models.UserToken{
					ID:     1,
					UserID: 1,
					Token:  "invalid-token-format",
				},
			},
			expectedError: true,
			errorContains: "invalid or expired refresh token",
		},
		{
			name:         "database error updating token",
			refreshToken: validRefreshToken,
			userRepo:     &mockUserRepository{},
			tokenRepo: &mockUserTokenRepository{
				token: &models.UserToken{
					ID:     1,
					UserID: 1,
					Token:  validRefreshToken,
				},
				updateTokenErr: errors.New("update error"),
			},
			expectedError: true,
			errorContains: "failed to update refresh token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewAuthService(tt.userRepo, tt.tokenRepo, tokenGen, logger)

			// Add small delay for success case to ensure different token timestamps
			if !tt.expectedError && tt.name == "success" {
				time.Sleep(1100 * time.Millisecond) // Wait more than 1 second to ensure different iat
			}

			accessToken, refreshToken, err := svc.Refresh(context.Background(), tt.refreshToken)

			if tt.expectedError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
				assert.Empty(t, accessToken)
				assert.Empty(t, refreshToken)
			} else {
				require.NoError(t, err)
				assert.NotEmpty(t, accessToken)
				assert.NotEmpty(t, refreshToken)
				// New tokens should be different from old token
				assert.NotEqual(t, tt.refreshToken, refreshToken)
			}
		})
	}
}
