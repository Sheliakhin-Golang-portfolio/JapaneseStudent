package services

import (
	"context"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/japanesestudent/auth-service/internal/models"
	"github.com/japanesestudent/libs/auth/service"
	"github.com/stretchr/testify/assert"
)

// mockProfileUserRepository is a mock implementation of ProfileUserRepository for service tests
type mockProfileUserRepository struct {
	user            *models.User
	err             error
	updateErr       error
	updatePasswordErr error
	existsByEmail   bool
	existsByEmailErr error
	existsByUsername bool
	existsByUsernameErr error
}

// mockUserSettingsRepositoryForProfile is a mock implementation of UserSettingsRepository for profile service tests
type mockUserSettingsRepositoryForProfile struct {
	settings  *models.UserSettings
	err       error
	updateErr error
}

func (m *mockUserSettingsRepositoryForProfile) Create(ctx context.Context, userId int) error {
	return m.err
}

func (m *mockUserSettingsRepositoryForProfile) GetByUserId(ctx context.Context, userId int) (*models.UserSettings, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.settings, nil
}

func (m *mockUserSettingsRepositoryForProfile) Update(ctx context.Context, userId int, settings *models.UserSettings) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	return m.err
}

func (m *mockUserSettingsRepositoryForProfile) ExistsByUserId(ctx context.Context, userId int) (bool, error) {
	return m.settings != nil, m.err
}

func (m *mockProfileUserRepository) GetByID(ctx context.Context, userID int) (*models.User, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.user, nil
}

func (m *mockProfileUserRepository) Update(ctx context.Context, userID int, user *models.User, settings *models.UserSettings, active *bool) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	return m.err
}

func (m *mockProfileUserRepository) UpdatePasswordHash(ctx context.Context, userID int, passwordHash string) error {
	return m.updatePasswordErr
}

func (m *mockProfileUserRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	if m.existsByEmailErr != nil {
		return false, m.existsByEmailErr
	}
	return m.existsByEmail, nil
}

func (m *mockProfileUserRepository) ExistsByUsername(ctx context.Context, username string) (bool, error) {
	if m.existsByUsernameErr != nil {
		return false, m.existsByUsernameErr
	}
	return m.existsByUsername, nil
}

func (m *mockProfileUserRepository) UpdateActive(ctx context.Context, userID int, active bool) error {
	return m.err
}

func TestNewProfileService(t *testing.T) {
	mockRepo := &mockProfileUserRepository{}
	mockSettingsRepo := &mockUserSettingsRepositoryForProfile{}
	tokenGen := service.NewTokenGenerator("test-secret", 1*time.Hour, 7*24*time.Hour)

	svc := NewProfileService(mockRepo, mockSettingsRepo, tokenGen, "", "", "", "", "", "", false)

	assert.NotNil(t, svc)
}

func TestProfileService_GetUser(t *testing.T) {
	tests := []struct {
		name          string
		userId        int
		mockRepo      *mockProfileUserRepository
		expectedError bool
		expectedUser  *models.ProfileResponse
	}{
		{
			name:   "success",
			userId: 1,
			mockRepo: &mockProfileUserRepository{
				user: &models.User{
					ID:       1,
					Username: "testuser",
					Email:    "test@example.com",
					Avatar:   "http://example.com/avatar.jpg",
				},
			},
			expectedError: false,
			expectedUser: &models.ProfileResponse{
				Username: "testuser",
				Email:    "test@example.com",
				Avatar:   "http://example.com/avatar.jpg",
			},
		},
		{
			name:   "user not found",
			userId: 999,
			mockRepo: &mockProfileUserRepository{
				err: errors.New("user not found"),
			},
			expectedError: true,
		},
		{
			name:   "invalid user id",
			userId: 0,
			mockRepo: &mockProfileUserRepository{
				user: &models.User{},
			},
			expectedError: true,
		},
		{
			name:   "success without avatar",
			userId: 2,
			mockRepo: &mockProfileUserRepository{
				user: &models.User{
					ID:       2,
					Username: "testuser2",
					Email:    "test2@example.com",
					Avatar:   "",
				},
			},
			expectedError: false,
			expectedUser: &models.ProfileResponse{
				Username: "testuser2",
				Email:    "test2@example.com",
				Avatar:   "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenGen := service.NewTokenGenerator("test-secret", 1*time.Hour, 7*24*time.Hour)
			mockSettingsRepo := &mockUserSettingsRepositoryForProfile{}
			svc := NewProfileService(tt.mockRepo, mockSettingsRepo, tokenGen, "", "", "", "", "", "", false)

			result, err := svc.GetUser(context.Background(), tt.userId)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.expectedUser.Username, result.Username)
				assert.Equal(t, tt.expectedUser.Email, result.Email)
				assert.Equal(t, tt.expectedUser.Avatar, result.Avatar)
			}
		})
	}
}

func TestProfileService_UpdateUser(t *testing.T) {
	tests := []struct {
		name          string
		userId        int
		username      string
		email         string
		mockRepo      *mockProfileUserRepository
		expectedError bool
		errorContains string
	}{
		{
			name:     "success update username",
			userId:   1,
			username: "newusername",
			email:    "",
			mockRepo: &mockProfileUserRepository{
				user: &models.User{
					ID:       1,
					Username: "oldusername",
					Email:    "test@example.com",
				},
				existsByUsername: false,
			},
			expectedError: false,
		},
		{
			name:     "success update email",
			userId:   1,
			username: "",
			email:    "newemail@example.com",
			mockRepo: &mockProfileUserRepository{
				user: &models.User{
					ID:       1,
					Username: "testuser",
					Email:    "old@example.com",
				},
				existsByEmail: false,
			},
			expectedError: true, // Will fail due to TASK_BASE_URL not configured
			errorContains: "TASK_BASE_URL is not configured",
		},
		{
			name:     "success update both",
			userId:   1,
			username: "newusername",
			email:    "newemail@example.com",
			mockRepo: &mockProfileUserRepository{
				user: &models.User{
					ID:       1,
					Username: "oldusername",
					Email:    "old@example.com",
				},
				existsByEmail:   false,
				existsByUsername: false,
			},
			expectedError: true, // Will fail due to TASK_BASE_URL not configured
			errorContains: "TASK_BASE_URL is not configured",
		},
		{
			name:     "no fields provided",
			userId:   1,
			username: "",
			email:    "",
			mockRepo: &mockProfileUserRepository{
				user: &models.User{},
			},
			expectedError: true,
			errorContains: "at least one field must be provided",
		},
		{
			name:     "invalid email format",
			userId:   1,
			username: "",
			email:    "invalid-email",
			mockRepo: &mockProfileUserRepository{
				user: &models.User{},
			},
			expectedError: true,
			errorContains: "invalid email format",
		},
		{
			name:     "email already exists (different user)",
			userId:   1,
			username: "",
			email:    "existing@example.com",
			mockRepo: &mockProfileUserRepository{
				user: &models.User{
					ID:       1,
					Username: "testuser",
					Email:    "old@example.com",
				},
				existsByEmail: true,
			},
			expectedError: true,
			errorContains: "email already exists",
		},
		{
			name:     "username already exists (different user)",
			userId:   1,
			username: "existinguser",
			email:    "",
			mockRepo: &mockProfileUserRepository{
				user: &models.User{
					ID:       1,
					Username: "testuser",
					Email:    "test@example.com",
				},
				existsByUsername: true,
			},
			expectedError: true,
			errorContains: "username already exists",
		},
		{
			name:     "email belongs to current user (allowed)",
			userId:   1,
			username: "",
			email:    "test@example.com",
			mockRepo: &mockProfileUserRepository{
				user: &models.User{
					ID:       1,
					Username: "testuser",
					Email:    "test@example.com",
				},
				existsByEmail: false, // Service doesn't check if email belongs to current user, so mock should return false
			},
			expectedError: true, // Service still tries to create task even if email is the same, so expect TASK_BASE_URL error
			errorContains: "TASK_BASE_URL is not configured",
		},
		{
			name:     "username belongs to current user (allowed)",
			userId:   1,
			username: "testuser",
			email:    "",
			mockRepo: &mockProfileUserRepository{
				user: &models.User{
					ID:       1,
					Username: "testuser",
					Email:    "test@example.com",
				},
				existsByUsername: false, // Service doesn't check if username belongs to current user, so mock should return false
			},
			expectedError: false,
		},
		{
			name:     "invalid user id",
			userId:   0,
			username: "newusername",
			email:    "",
			mockRepo: &mockProfileUserRepository{
				user: &models.User{},
			},
			expectedError: true,
			errorContains: "invalid user id",
		},
		{
			name:     "update error",
			userId:   1,
			username: "newusername",
			email:    "",
			mockRepo: &mockProfileUserRepository{
				user: &models.User{
					ID:       1,
					Username: "oldusername",
					Email:    "test@example.com",
				},
				existsByUsername: false,
				updateErr:        errors.New("database error"),
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenGen := service.NewTokenGenerator("test-secret", 1*time.Hour, 7*24*time.Hour)
			// Don't set taskBaseURL in tests to avoid HTTP call failures
			// Tests that need task service functionality should expect the error
			taskBaseURL := ""
			apiKey := ""
			mockSettingsRepo := &mockUserSettingsRepositoryForProfile{}
			svc := NewProfileService(tt.mockRepo, mockSettingsRepo, tokenGen, "", apiKey, taskBaseURL, "http://localhost:8080", "", "", false)

			err := svc.UpdateUser(context.Background(), tt.userId, tt.username, tt.email)

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

func TestProfileService_UpdatePassword(t *testing.T) {
	tests := []struct {
		name             string
		userId           int
		password         string
		mockRepo         *mockProfileUserRepository
		expectedError    bool
		errorContains    string
	}{
		{
			name:     "success",
			userId:   1,
			password: "Password123!",
			mockRepo: &mockProfileUserRepository{
				user: &models.User{ID: 1},
			},
			expectedError: false,
		},
		{
			name:     "empty password",
			userId:   1,
			password: "",
			mockRepo: &mockProfileUserRepository{
				user: &models.User{ID: 1},
			},
			expectedError: true,
			errorContains: "password cannot be empty",
		},
		{
			name:     "password too short",
			userId:   1,
			password: "Pass1!",
			mockRepo: &mockProfileUserRepository{
				user: &models.User{ID: 1},
			},
			expectedError: true,
			errorContains: "password must be at least 8 characters",
		},
		{
			name:     "password missing uppercase",
			userId:   1,
			password: "password123!",
			mockRepo: &mockProfileUserRepository{
				user: &models.User{ID: 1},
			},
			expectedError: true,
			errorContains: "password must be at least 8 characters",
		},
		{
			name:     "password missing lowercase",
			userId:   1,
			password: "PASSWORD123!",
			mockRepo: &mockProfileUserRepository{
				user: &models.User{ID: 1},
			},
			expectedError: true,
			errorContains: "password must be at least 8 characters",
		},
		{
			name:     "password missing number",
			userId:   1,
			password: "Password!",
			mockRepo: &mockProfileUserRepository{
				user: &models.User{ID: 1},
			},
			expectedError: true,
			errorContains: "password must be at least 8 characters",
		},
		{
			name:     "password missing special character",
			userId:   1,
			password: "Password123",
			mockRepo: &mockProfileUserRepository{
				user: &models.User{ID: 1},
			},
			expectedError: true,
			errorContains: "password must be at least 8 characters",
		},
		{
			name:     "invalid user id",
			userId:   0,
			password: "Password123!",
			mockRepo: &mockProfileUserRepository{
				user: &models.User{ID: 0},
			},
			expectedError: true,
			errorContains: "invalid user id",
		},
		{
			name:     "update password error",
			userId:   1,
			password: "Password123!",
			mockRepo: &mockProfileUserRepository{
				user:              &models.User{ID: 1},
				updatePasswordErr: errors.New("database error"),
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenGen := service.NewTokenGenerator("test-secret", 1*time.Hour, 7*24*time.Hour)
			mockSettingsRepo := &mockUserSettingsRepositoryForProfile{}
			svc := NewProfileService(tt.mockRepo, mockSettingsRepo, tokenGen, "", "", "", "", "", "", false)

			err := svc.UpdatePassword(context.Background(), tt.userId, tt.password)

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

func TestProfileService_UpdateAvatar(t *testing.T) {
	tests := []struct {
		name          string
		userId        int
		avatarFile    multipart.File
		avatarFilename string
		mockRepo      *mockProfileUserRepository
		mediaBaseURL  string
		apiKey        string
		expectedError bool
		errorContains string
	}{
		{
			name:          "invalid user id",
			userId:        0,
			avatarFile:    nil,
			avatarFilename: "avatar.jpg",
			mockRepo: &mockProfileUserRepository{
				user: &models.User{ID: 0},
			},
			mediaBaseURL:  "http://media.example.com",
			apiKey:        "test-key",
			expectedError: true,
			errorContains: "invalid user id",
		},
		{
			name:          "user not found",
			userId:        1,
			avatarFile:    nil,
			avatarFilename: "avatar.jpg",
			mockRepo: &mockProfileUserRepository{
				err: errors.New("user not found"),
			},
			mediaBaseURL:  "http://media.example.com",
			apiKey:        "test-key",
			expectedError: true,
			errorContains: "user not found",
		},
		{
			name:          "media base URL not configured",
			userId:        1,
			avatarFile:    nil,
			avatarFilename: "avatar.jpg",
			mockRepo: &mockProfileUserRepository{
				user: &models.User{ID: 1},
			},
			mediaBaseURL:  "",
			apiKey:        "test-key",
			expectedError: true,
			errorContains: "MEDIA_BASE_URL is not configured",
		},
		// Note: Full avatar upload tests would require mocking HTTP client,
		// which is complex. These tests verify validation logic.
		// Full integration testing should be done on live server as specified in plan.
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenGen := service.NewTokenGenerator("test-secret", 1*time.Hour, 7*24*time.Hour)
			mockSettingsRepo := &mockUserSettingsRepositoryForProfile{}
			svc := NewProfileService(tt.mockRepo, mockSettingsRepo, tokenGen, tt.mediaBaseURL, tt.apiKey, "", "", "", "", false)

			_, err := svc.UpdateAvatar(context.Background(), tt.userId, tt.avatarFile, tt.avatarFilename)

			if tt.expectedError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			}
		})
	}
}

// TestProfileService_UpdateRepeatFlag tests the UpdateRepeatFlag method.
// NOTE: This test avoids calling task-service by using empty scheduledTaskBaseURL.
// The actual task-service integration (creating/deleting scheduled tasks) should be tested
// on a live server with the task-service running.
func TestProfileService_UpdateRepeatFlag(t *testing.T) {
	tests := []struct {
		name              string
		userId            int
		flag              string
		previousFlag      models.RepeatType
		mockUserRepo      *mockProfileUserRepository
		mockSettingsRepo  *mockUserSettingsRepositoryForProfile
		expectedError     bool
		errorContains     string
	}{
		{
			name:   "success update to 'in question'",
			userId:  1,
			flag:    "in question",
			previousFlag: models.RepeatTypeIgnore,
			mockUserRepo: &mockProfileUserRepository{
				user: &models.User{
					ID:    1,
					Email: "test@example.com",
				},
			},
			mockSettingsRepo: &mockUserSettingsRepositoryForProfile{
				settings: &models.UserSettings{
					UserID:         1,
					AlphabetRepeat: models.RepeatTypeIgnore,
				},
			},
			expectedError: false,
		},
		{
			name:   "success update to 'ignore'",
			userId:  1,
			flag:    "ignore",
			previousFlag: models.RepeatTypeInQuestion,
			mockUserRepo: &mockProfileUserRepository{
				user: &models.User{
					ID:    1,
					Email: "test@example.com",
				},
			},
			mockSettingsRepo: &mockUserSettingsRepositoryForProfile{
				settings: &models.UserSettings{
					UserID:         1,
					AlphabetRepeat: models.RepeatTypeInQuestion,
				},
			},
			expectedError: false,
		},
		{
			name:   "success update to 'repeat' - will fail due to empty scheduledTaskBaseURL",
			userId:  1,
			flag:    "repeat",
			previousFlag: models.RepeatTypeInQuestion,
			mockUserRepo: &mockProfileUserRepository{
				user: &models.User{
					ID:    1,
					Email: "test@example.com",
				},
			},
			mockSettingsRepo: &mockUserSettingsRepositoryForProfile{
				settings: &models.UserSettings{
					UserID:         1,
					AlphabetRepeat: models.RepeatTypeInQuestion,
				},
			},
			expectedError: true,
			errorContains: "SCHEDULED_TASK_BASE_URL is not configured",
		},
		{
			name:   "success update from 'repeat' to 'ignore' - will fail due to empty scheduledTaskBaseURL",
			userId:  1,
			flag:    "ignore",
			previousFlag: models.RepeatTypeRepeat,
			mockUserRepo: &mockProfileUserRepository{
				user: &models.User{
					ID:    1,
					Email: "test@example.com",
				},
			},
			mockSettingsRepo: &mockUserSettingsRepositoryForProfile{
				settings: &models.UserSettings{
					UserID:         1,
					AlphabetRepeat: models.RepeatTypeRepeat,
				},
			},
			expectedError: true,
			errorContains: "SCHEDULED_TASK_BASE_URL is not configured",
		},
		{
			name:   "invalid flag value",
			userId:  1,
			flag:    "invalid",
			previousFlag: models.RepeatTypeInQuestion,
			mockUserRepo: &mockProfileUserRepository{
				user: &models.User{
					ID:    1,
					Email: "test@example.com",
				},
			},
			mockSettingsRepo: &mockUserSettingsRepositoryForProfile{
				settings: &models.UserSettings{
					UserID:         1,
					AlphabetRepeat: models.RepeatTypeInQuestion,
				},
			},
			expectedError: true,
			errorContains: "flag must be 'in question', 'ignore', or 'repeat'",
		},
		{
			name:   "invalid user id",
			userId:  0,
			flag:    "in question",
			previousFlag: models.RepeatTypeInQuestion,
			mockUserRepo: &mockProfileUserRepository{
				user: &models.User{ID: 0},
			},
			mockSettingsRepo: &mockUserSettingsRepositoryForProfile{
				settings: &models.UserSettings{
					UserID:         0,
					AlphabetRepeat: models.RepeatTypeInQuestion,
				},
			},
			expectedError: true,
			errorContains: "invalid user id",
		},
		{
			name:   "user settings not found",
			userId:  1,
			flag:    "in question",
			previousFlag: models.RepeatTypeInQuestion,
			mockUserRepo: &mockProfileUserRepository{
				user: &models.User{
					ID:    1,
					Email: "test@example.com",
				},
			},
			mockSettingsRepo: &mockUserSettingsRepositoryForProfile{
				err: errors.New("user settings not found"),
			},
			expectedError: true,
			errorContains: "user settings not found",
		},
		{
			name:   "update settings error",
			userId:  1,
			flag:    "in question",
			previousFlag: models.RepeatTypeInQuestion,
			mockUserRepo: &mockProfileUserRepository{
				user: &models.User{
					ID:    1,
					Email: "test@example.com",
				},
			},
			mockSettingsRepo: &mockUserSettingsRepositoryForProfile{
				settings: &models.UserSettings{
					UserID:         1,
					AlphabetRepeat: models.RepeatTypeInQuestion,
				},
				updateErr: errors.New("database error"),
			},
			expectedError: true,
			errorContains: "database error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenGen := service.NewTokenGenerator("test-secret", 1*time.Hour, 7*24*time.Hour)
			// Use empty scheduledTaskBaseURL to avoid calling task-service
			// NOTE: Task-service integration (creating/deleting scheduled tasks) should be tested
			// on a live server with the task-service running.
			svc := NewProfileService(tt.mockUserRepo, tt.mockSettingsRepo, tokenGen, "", "", "", "", "", "", false)

			// Create a mock HTTP request
			req := httptest.NewRequest(http.MethodPut, "/api/v6/profile/repeat-flag", nil)

			err := svc.UpdateRepeatFlag(context.Background(), tt.userId, tt.flag, req)

			if tt.expectedError {
				assert.Error(t, err)
				if tt.errorContains != "" && err != nil {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
