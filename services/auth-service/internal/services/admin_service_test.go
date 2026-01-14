package services

import (
	"context"
	"errors"
	"testing"

	"github.com/japanesestudent/auth-service/internal/models"
	"github.com/japanesestudent/libs/auth/service"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zaptest"
)

// mockAdminUserRepository is a mock implementation of AdminUserRepository
type mockAdminUserRepository struct {
	user                   *models.User
	users                  []models.User
	tutorsList             []models.TutorListItem
	err                    error
	createErr              error
	updateErr              error
	deleteErr              error
	existsByEmailResult    bool
	existsByEmailError     error
	existsByUsernameResult bool
	existsByUsernameError  error
}

func (m *mockAdminUserRepository) Create(ctx context.Context, user *models.User) error {
	if m.createErr != nil {
		return m.createErr
	}
	if m.err != nil {
		return m.err
	}
	user.ID = 1
	return nil
}

func (m *mockAdminUserRepository) GetByID(ctx context.Context, userID int) (*models.User, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.user, nil
}

func (m *mockAdminUserRepository) GetAll(ctx context.Context, page, count int, role *models.Role, search string) ([]models.User, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.users, nil
}

func (m *mockAdminUserRepository) Update(ctx context.Context, userID int, user *models.User, settings *models.UserSettings, active *bool) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	return m.err
}

func (m *mockAdminUserRepository) Delete(ctx context.Context, userID int) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	return m.err
}

func (m *mockAdminUserRepository) GetTutorsList(ctx context.Context) ([]models.TutorListItem, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.tutorsList, nil
}

func (m *mockAdminUserRepository) UpdateActive(ctx context.Context, userID int, active bool) error {
	return m.err
}

func (m *mockAdminUserRepository) UpdatePasswordHash(ctx context.Context, userID int, passwordHash string) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	return m.err
}

// Implement UserSharedRepository interface methods
func (m *mockAdminUserRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	if m.existsByEmailError != nil {
		return false, m.existsByEmailError
	}
	return m.existsByEmailResult, nil
}

func (m *mockAdminUserRepository) ExistsByUsername(ctx context.Context, username string) (bool, error) {
	if m.existsByUsernameError != nil {
		return false, m.existsByUsernameError
	}
	return m.existsByUsernameResult, nil
}

// mockAdminUserTokenRepository is a mock implementation of AdminUserTokenRepository
type mockAdminUserTokenRepository struct {
	err error
}

func (m *mockAdminUserTokenRepository) Create(ctx context.Context, userToken *models.UserToken) error {
	return m.err
}

// mockUserSettingsRepository is a mock implementation of UserSettingsRepository
type mockUserSettingsRepository struct {
	settings  *models.UserSettings
	err       error
	getErr    error // Separate error for GetByUserId
	createErr error
}

func (m *mockUserSettingsRepository) Create(ctx context.Context, userID int) error {
	if m.createErr != nil {
		return m.createErr
	}
	return m.err
}

func (m *mockUserSettingsRepository) GetByUserId(ctx context.Context, userID int) (*models.UserSettings, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	if m.err != nil {
		return nil, m.err
	}
	return m.settings, nil
}

func (m *mockUserSettingsRepository) Update(ctx context.Context, userID int, settings *models.UserSettings) error {
	return m.err
}

func (m *mockUserSettingsRepository) ExistsByUserId(ctx context.Context, userID int) (bool, error) {
	return m.settings != nil, m.err
}

func TestNewAdminService(t *testing.T) {
	mockUserRepo := &mockAdminUserRepository{}
	mockTokenRepo := &mockAdminUserTokenRepository{}
	mockSettingsRepo := &mockUserSettingsRepository{}
	tokenGen := service.NewTokenGenerator("test-secret", 3600, 604800)
	logger := zaptest.NewLogger(t)

	svc := NewAdminService(mockUserRepo, mockTokenRepo, mockSettingsRepo, tokenGen, logger, "", "", "", "", false, "")

	assert.NotNil(t, svc)
	assert.Equal(t, mockUserRepo, svc.userRepo)
	assert.Equal(t, mockTokenRepo, svc.userTokenRepo)
	assert.Equal(t, mockSettingsRepo, svc.userSettingsRepo)
}

func TestAdminService_GetUsersList(t *testing.T) {
	tests := []struct {
		name          string
		page          int
		count         int
		role          *int
		search        string
		mockRepo      *mockAdminUserRepository
		expectedError bool
		expectedCount int
	}{
		{
			name:   "success with defaults",
			page:   0,
			count:  0,
			role:   nil,
			search: "",
			mockRepo: &mockAdminUserRepository{
				users: []models.User{
					{ID: 1, Username: "user1", Email: "user1@example.com", Role: models.RoleUser, Avatar: ""},
					{ID: 2, Username: "user2", Email: "user2@example.com", Role: models.RoleUser, Avatar: ""},
				},
			},
			expectedError: false,
			expectedCount: 2,
		},
		{
			name:   "success with pagination",
			page:   2,
			count:  10,
			role:   nil,
			search: "",
			mockRepo: &mockAdminUserRepository{
				users: []models.User{
					{ID: 1, Username: "user1", Email: "user1@example.com", Role: models.RoleUser, Avatar: ""},
				},
			},
			expectedError: false,
			expectedCount: 1,
		},
		{
			name:   "success with role filter",
			page:   1,
			count:  20,
			role:   intPtr(int(models.RoleAdmin)),
			search: "",
			mockRepo: &mockAdminUserRepository{
				users: []models.User{
					{ID: 1, Username: "admin", Email: "admin@example.com", Role: models.RoleAdmin, Avatar: ""},
				},
			},
			expectedError: false,
			expectedCount: 1,
		},
		{
			name:   "success with search",
			page:   1,
			count:  20,
			role:   nil,
			search: "user1",
			mockRepo: &mockAdminUserRepository{
				users: []models.User{
					{ID: 1, Username: "user1", Email: "user1@example.com", Role: models.RoleUser, Avatar: ""},
				},
			},
			expectedError: false,
			expectedCount: 1,
		},
		{
			name:          "invalid role - too low",
			page:          1,
			count:         20,
			role:          intPtr(0),
			search:        "",
			mockRepo:      &mockAdminUserRepository{},
			expectedError: true,
			expectedCount: 0,
		},
		{
			name:          "invalid role - too high",
			page:          1,
			count:         20,
			role:          intPtr(4),
			search:        "",
			mockRepo:      &mockAdminUserRepository{},
			expectedError: true,
			expectedCount: 0,
		},
		{
			name:   "repository error",
			page:   1,
			count:  20,
			role:   nil,
			search: "",
			mockRepo: &mockAdminUserRepository{
				err: errors.New("database error"),
			},
			expectedError: true,
			expectedCount: 0,
		},
		{
			name:   "empty result",
			page:   1,
			count:  20,
			role:   nil,
			search: "",
			mockRepo: &mockAdminUserRepository{
				users: []models.User{},
			},
			expectedError: false,
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenGen := service.NewTokenGenerator("test-secret", 3600, 604800)
			logger := zaptest.NewLogger(t)
			svc := NewAdminService(tt.mockRepo, &mockAdminUserTokenRepository{}, &mockUserSettingsRepository{}, tokenGen, logger, "", "", "", "", false, "")
			ctx := context.Background()

			result, err := svc.GetUsersList(ctx, tt.page, tt.count, tt.role, tt.search)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Len(t, result, tt.expectedCount)
			}
		})
	}
}

func TestAdminService_GetUserWithSettings(t *testing.T) {
	tests := []struct {
		name             string
		userID           int
		mockUserRepo     *mockAdminUserRepository
		mockSettingsRepo *mockUserSettingsRepository
		expectedError    bool
		expectedID       int
		hasSettings      bool
	}{
		{
			name:   "success with settings",
			userID: 1,
			mockUserRepo: &mockAdminUserRepository{
				user: &models.User{
					ID:       1,
					Username: "user1",
					Email:    "user1@example.com",
					Role:     models.RoleUser,
					Avatar:   "", // Empty avatar for tests
				},
			},
			mockSettingsRepo: &mockUserSettingsRepository{
				settings: &models.UserSettings{
					UserID:       1,
					NewWordCount: 20,
				},
			},
			expectedError: false,
			expectedID:    1,
			hasSettings:   true,
		},
		{
			name:   "success without settings",
			userID: 1,
			mockUserRepo: &mockAdminUserRepository{
				user: &models.User{
					ID:       1,
					Username: "user1",
					Email:    "user1@example.com",
					Role:     models.RoleUser,
					Avatar:   "", // Empty avatar for tests
				},
			},
			mockSettingsRepo: &mockUserSettingsRepository{
				err: errors.New("settings not found"),
			},
			expectedError: false,
			expectedID:    1,
			hasSettings:   false,
		},
		{
			name:             "invalid user id zero",
			userID:           0,
			mockUserRepo:     &mockAdminUserRepository{},
			mockSettingsRepo: &mockUserSettingsRepository{},
			expectedError:    true,
			expectedID:       0,
		},
		{
			name:             "invalid user id negative",
			userID:           -1,
			mockUserRepo:     &mockAdminUserRepository{},
			mockSettingsRepo: &mockUserSettingsRepository{},
			expectedError:    true,
			expectedID:       0,
		},
		{
			name:   "user not found",
			userID: 999,
			mockUserRepo: &mockAdminUserRepository{
				err: errors.New("user not found"),
			},
			mockSettingsRepo: &mockUserSettingsRepository{},
			expectedError:    true,
			expectedID:       0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenGen := service.NewTokenGenerator("test-secret", 3600, 604800)
			logger := zaptest.NewLogger(t)
			svc := NewAdminService(tt.mockUserRepo, &mockAdminUserTokenRepository{}, tt.mockSettingsRepo, tokenGen, logger, "", "", "", "", false, "")
			ctx := context.Background()

			result, err := svc.GetUserWithSettings(ctx, tt.userID)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.expectedID, result.ID)
				if tt.hasSettings {
					assert.NotNil(t, result.Settings)
					assert.Empty(t, result.Message)
				} else {
					assert.Nil(t, result.Settings)
					assert.Equal(t, "Settings was not created", result.Message)
				}
			}
		})
	}
}

func TestAdminService_CreateUser(t *testing.T) {
	tests := []struct {
		name             string
		request          *models.CreateUserRequest
		mockUserRepo     *mockAdminUserRepository
		mockSettingsRepo *mockUserSettingsRepository
		expectedError    bool
		expectedID       int
		errorContains    string
	}{
		{
			name: "success",
			request: &models.CreateUserRequest{
				Email:    "newuser@example.com",
				Username: "newuser",
				Password: "Password123!",
				Role:     models.RoleUser,
			},
			mockUserRepo: &mockAdminUserRepository{
				existsByEmailResult:    false,
				existsByUsernameResult: false,
			},
			mockSettingsRepo: &mockUserSettingsRepository{},
			expectedError:    false,
			expectedID:       1,
		},
		{
			name: "email already exists",
			request: &models.CreateUserRequest{
				Email:    "existing@example.com",
				Username: "newuser",
				Password: "Password123!",
				Role:     models.RoleUser,
			},
			mockUserRepo: &mockAdminUserRepository{
				existsByEmailResult:    true,
				existsByUsernameResult: false,
			},
			mockSettingsRepo: &mockUserSettingsRepository{},
			expectedError:    true,
			expectedID:       0,
			errorContains:    "email already exists",
		},
		{
			name: "username already exists",
			request: &models.CreateUserRequest{
				Email:    "newuser@example.com",
				Username: "existing",
				Password: "Password123!",
				Role:     models.RoleUser,
			},
			mockUserRepo: &mockAdminUserRepository{
				existsByEmailResult:    false,
				existsByUsernameResult: true,
			},
			mockSettingsRepo: &mockUserSettingsRepository{},
			expectedError:    true,
			expectedID:       0,
			errorContains:    "username already exists",
		},
		{
			name: "invalid email format",
			request: &models.CreateUserRequest{
				Email:    "invalid-email",
				Username: "newuser",
				Password: "Password123!",
				Role:     models.RoleUser,
			},
			mockUserRepo:     &mockAdminUserRepository{},
			mockSettingsRepo: &mockUserSettingsRepository{},
			expectedError:    true,
			expectedID:       0,
			errorContains:    "invalid email format",
		},
		{
			name: "invalid password - too short",
			request: &models.CreateUserRequest{
				Email:    "newuser@example.com",
				Username: "newuser",
				Password: "Short1!",
				Role:     models.RoleUser,
			},
			mockUserRepo:     &mockAdminUserRepository{},
			mockSettingsRepo: &mockUserSettingsRepository{},
			expectedError:    true,
			expectedID:       0,
			errorContains:    "password must be at least 8 characters",
		},
		{
			name: "failed to create user",
			request: &models.CreateUserRequest{
				Email:    "newuser@example.com",
				Username: "newuser",
				Password: "Password123!",
				Role:     models.RoleUser,
			},
			mockUserRepo: &mockAdminUserRepository{
				existsByEmailResult:    false,
				existsByUsernameResult: false,
				createErr:              errors.New("create error"),
			},
			mockSettingsRepo: &mockUserSettingsRepository{},
			expectedError:    true,
			expectedID:       0,
		},
		{
			name: "failed to create settings - non-critical",
			request: &models.CreateUserRequest{
				Email:    "newuser@example.com",
				Username: "newuser",
				Password: "Password123!",
				Role:     models.RoleUser,
			},
			mockUserRepo: &mockAdminUserRepository{
				existsByEmailResult:    false,
				existsByUsernameResult: false,
			},
			mockSettingsRepo: &mockUserSettingsRepository{
				createErr: errors.New("settings error"),
			},
			expectedError: false,
			expectedID:    1, // User is still created
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenGen := service.NewTokenGenerator("test-secret", 3600, 604800)
			logger := zaptest.NewLogger(t)
			svc := NewAdminService(tt.mockUserRepo, &mockAdminUserTokenRepository{}, tt.mockSettingsRepo, tokenGen, logger, "", "", "", "", false, "")
			ctx := context.Background()

			result, err := svc.CreateUser(ctx, tt.request, nil, "")

			if tt.expectedError {
				assert.Error(t, err)
				assert.Equal(t, 0, result)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedID, result)
			}
		})
	}
}

func TestAdminService_CreateUserSettings(t *testing.T) {
	tests := []struct {
		name             string
		userID           int
		mockSettingsRepo *mockUserSettingsRepository
		expectedError    bool
		expectedMsg      string
	}{
		{
			name:   "success create settings",
			userID: 1,
			mockSettingsRepo: &mockUserSettingsRepository{
				getErr:    errors.New("settings not found"), // GetByUserId returns this error
				createErr: nil,                              // Create should succeed (err is nil)
			},
			expectedError: false,
			expectedMsg:   "Settings created successfully",
		},
		{
			name:   "settings already exist",
			userID: 1,
			mockSettingsRepo: &mockUserSettingsRepository{
				settings: &models.UserSettings{UserID: 1},
			},
			expectedError: false,
			expectedMsg:   "Settings already exist",
		},
		{
			name:   "failed to create settings",
			userID: 1,
			mockSettingsRepo: &mockUserSettingsRepository{
				getErr:    errors.New("settings not found"),
				createErr: errors.New("create error"),
			},
			expectedError: true,
			expectedMsg:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenGen := service.NewTokenGenerator("test-secret", 3600, 604800)
			logger := zaptest.NewLogger(t)
			svc := NewAdminService(&mockAdminUserRepository{}, &mockAdminUserTokenRepository{}, tt.mockSettingsRepo, tokenGen, logger, "", "", "", "", false, "")
			ctx := context.Background()

			result, err := svc.CreateUserSettings(ctx, tt.userID)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Empty(t, result)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedMsg, result)
			}
		})
	}
}

func TestAdminService_DeleteUser(t *testing.T) {
	tests := []struct {
		name          string
		userID        int
		mockUserRepo  *mockAdminUserRepository
		expectedError bool
		errorContains string
	}{
		{
			name:   "success",
			userID: 1,
			mockUserRepo: &mockAdminUserRepository{
				user: &models.User{
					ID:       1,
					Username: "testuser",
					Email:    "test@example.com",
					Role:     models.RoleUser,
					Avatar:   "", // Empty avatar for tests
				},
			},
			expectedError: false,
		},
		{
			name:          "invalid user id zero",
			userID:        0,
			mockUserRepo:  &mockAdminUserRepository{},
			expectedError: true,
			errorContains: "invalid user id",
		},
		{
			name:          "invalid user id negative",
			userID:        -1,
			mockUserRepo:  &mockAdminUserRepository{},
			expectedError: true,
			errorContains: "invalid user id",
		},
		{
			name:   "repository error",
			userID: 1,
			mockUserRepo: &mockAdminUserRepository{
				user: &models.User{
					ID:       1,
					Username: "testuser",
					Email:    "test@example.com",
					Role:     models.RoleUser,
					Avatar:   "", // Empty avatar for tests
				},
				deleteErr: errors.New("delete error"),
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenGen := service.NewTokenGenerator("test-secret", 3600, 604800)
			logger := zaptest.NewLogger(t)
			svc := NewAdminService(tt.mockUserRepo, &mockAdminUserTokenRepository{}, &mockUserSettingsRepository{}, tokenGen, logger, "", "", "", "", false, "")
			ctx := context.Background()

			err := svc.DeleteUser(ctx, tt.userID)

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

func TestAdminService_GetTutorsList(t *testing.T) {
	tests := []struct {
		name           string
		mockUserRepo   *mockAdminUserRepository
		expectedError  bool
		expectedCount  int
		expectedTutors []models.TutorListItem
	}{
		{
			name: "success with tutors",
			mockUserRepo: &mockAdminUserRepository{
				tutorsList: []models.TutorListItem{
					{ID: 1, Username: "tutor1"},
					{ID: 2, Username: "tutor2"},
					{ID: 3, Username: "tutor3"},
				},
			},
			expectedError: false,
			expectedCount: 3,
			expectedTutors: []models.TutorListItem{
				{ID: 1, Username: "tutor1"},
				{ID: 2, Username: "tutor2"},
				{ID: 3, Username: "tutor3"},
			},
		},
		{
			name: "success with empty list",
			mockUserRepo: &mockAdminUserRepository{
				tutorsList: []models.TutorListItem{},
			},
			expectedError: false,
			expectedCount: 0,
			expectedTutors: []models.TutorListItem{},
		},
		{
			name: "repository error",
			mockUserRepo: &mockAdminUserRepository{
				err: errors.New("database error"),
			},
			expectedError: true,
			expectedCount: 0,
			expectedTutors: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenGen := service.NewTokenGenerator("test-secret", 3600, 604800)
			logger := zaptest.NewLogger(t)
			svc := NewAdminService(tt.mockUserRepo, &mockAdminUserTokenRepository{}, &mockUserSettingsRepository{}, tokenGen, logger, "", "", "", "", false, "")
			ctx := context.Background()

			result, err := svc.GetTutorsList(ctx)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Len(t, result, tt.expectedCount)
				assert.Equal(t, tt.expectedTutors, result)
			}
		})
	}
}

func TestAdminService_UpdateUserPassword(t *testing.T) {
	tests := []struct {
		name          string
		userID        int
		password      string
		mockUserRepo  *mockAdminUserRepository
		expectedError bool
		errorContains string
	}{
		{
			name:     "success",
			userID:   1,
			password: "Password123!",
			mockUserRepo: &mockAdminUserRepository{
				err: nil,
			},
			expectedError: false,
		},
		{
			name:          "invalid user id zero",
			userID:        0,
			password:      "Password123!",
			mockUserRepo:  &mockAdminUserRepository{},
			expectedError: true,
			errorContains: "invalid user id",
		},
		{
			name:          "invalid user id negative",
			userID:        -1,
			password:      "Password123!",
			mockUserRepo:  &mockAdminUserRepository{},
			expectedError: true,
			errorContains: "invalid user id",
		},
		{
			name:          "password too short",
			userID:        1,
			password:      "Pass1!",
			mockUserRepo:  &mockAdminUserRepository{},
			expectedError: true,
			errorContains: "password must be at least 8 characters",
		},
		{
			name:          "password missing uppercase",
			userID:        1,
			password:      "password123!",
			mockUserRepo:  &mockAdminUserRepository{},
			expectedError: true,
			errorContains: "password must be at least 8 characters",
		},
		{
			name:          "password missing lowercase",
			userID:        1,
			password:      "PASSWORD123!",
			mockUserRepo:  &mockAdminUserRepository{},
			expectedError: true,
			errorContains: "password must be at least 8 characters",
		},
		{
			name:          "password missing number",
			userID:        1,
			password:      "Password!",
			mockUserRepo:  &mockAdminUserRepository{},
			expectedError: true,
			errorContains: "password must be at least 8 characters",
		},
		{
			name:          "password missing special character",
			userID:        1,
			password:      "Password123",
			mockUserRepo:  &mockAdminUserRepository{},
			expectedError: true,
			errorContains: "password must be at least 8 characters",
		},
		{
			name:          "password contains semicolon",
			userID:        1,
			password:      "Password123!;",
			mockUserRepo:  &mockAdminUserRepository{},
			expectedError: true,
			errorContains: "password cannot contain ';' character",
		},
		{
			name:     "repository error",
			userID:   1,
			password: "Password123!",
			mockUserRepo: &mockAdminUserRepository{
				updateErr: errors.New("database error"),
			},
			expectedError: true,
		},
		{
			name:     "user not found",
			userID:   1,
			password: "Password123!",
			mockUserRepo: &mockAdminUserRepository{
				updateErr: errors.New("user not found"),
			},
			expectedError: true,
			errorContains: "user not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenGen := service.NewTokenGenerator("test-secret", 3600, 604800)
			logger := zaptest.NewLogger(t)
			svc := NewAdminService(tt.mockUserRepo, &mockAdminUserTokenRepository{}, &mockUserSettingsRepository{}, tokenGen, logger, "", "", "", "", false, "")
			ctx := context.Background()

			err := svc.UpdateUserPassword(ctx, tt.userID, tt.password)

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

// Helper function to create int pointer
func intPtr(i int) *int {
	return &i
}
