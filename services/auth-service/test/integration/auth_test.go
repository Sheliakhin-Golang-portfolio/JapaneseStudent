package integration

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	_ "github.com/go-sql-driver/mysql"
	"github.com/japanesestudent/auth-service/internal/handlers"
	"github.com/japanesestudent/auth-service/internal/models"
	"github.com/japanesestudent/auth-service/internal/repositories"
	"github.com/japanesestudent/auth-service/internal/services"
	"github.com/japanesestudent/libs/auth/middleware"
	"github.com/japanesestudent/libs/auth/service"
	"github.com/japanesestudent/libs/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

var (
	testDB     *sql.DB
	testRouter chi.Router
	testLogger *zap.Logger
)

// seedTestData inserts test data into the database
func seedTestData(t *testing.T, db *sql.DB) {
	t.Helper()

	// Clear existing data
	_, err := db.Exec("DELETE FROM user_settings")
	require.NoError(t, err, "Failed to clear user_settings")
	_, err = db.Exec("DELETE FROM user_tokens")
	require.NoError(t, err, "Failed to clear user_tokens")
	_, err = db.Exec("DELETE FROM users")
	require.NoError(t, err, "Failed to clear users")

	// Reset AUTO_INCREMENT
	_, err = db.Exec("ALTER TABLE users AUTO_INCREMENT = 1")
	require.NoError(t, err, "Failed to reset users AUTO_INCREMENT")
	_, err = db.Exec("ALTER TABLE user_tokens AUTO_INCREMENT = 1")
	require.NoError(t, err, "Failed to reset user_tokens AUTO_INCREMENT")

	// Insert test user with known password
	passwordHash, err := bcrypt.GenerateFromPassword([]byte("Password123!"), bcrypt.DefaultCost)
	require.NoError(t, err, "Failed to hash password")

	query := `INSERT INTO users (username, email, password_hash, role) VALUES (?, ?, ?, ?)`
	_, err = db.Exec(query, "testuser", "test@example.com", string(passwordHash), models.RoleUser)
	require.NoError(t, err, "Failed to seed test user")
}

// cleanupTestData removes all test data
func cleanupTestData(t *testing.T, db *sql.DB) {
	t.Helper()
	_, err := db.Exec("DELETE FROM user_settings")
	require.NoError(t, err, "Failed to cleanup user_settings")
	_, err = db.Exec("DELETE FROM user_tokens")
	require.NoError(t, err, "Failed to cleanup user_tokens")
	_, err = db.Exec("DELETE FROM users")
	require.NoError(t, err, "Failed to cleanup users")
}

// getCookieValue extracts a cookie value from the response
func getCookieValue(w *httptest.ResponseRecorder, name string) string {
	for _, cookie := range w.Result().Cookies() {
		if cookie.Name == name {
			return cookie.Value
		}
	}
	return ""
}

// setupTestRouter creates a test router with all handlers
func setupTestRouter(db *sql.DB, logger *zap.Logger) chi.Router {
	userRepo := repositories.NewUserRepository(db)
	tokenRepo := repositories.NewUserTokenRepository(db)
	userSettingsRepo := repositories.NewUserSettingsRepository(db)
	tokenGen := service.NewTokenGenerator("test-secret-key-for-integration-tests", 1*time.Hour, 7*24*time.Hour)
	authSvc := services.NewAuthService(userRepo, tokenRepo, userSettingsRepo, tokenGen, logger)
	authHandler := handlers.NewAuthHandler(authSvc, logger)

	userSettingsSvc := services.NewUserSettingsService(userSettingsRepo)
	userSettingsHandler := handlers.NewUserSettingsHandler(userSettingsSvc, logger)

	r := chi.NewRouter()
	// Scope router to /api/v3 to match main.go setup
	r.Route("/api/v3", func(r chi.Router) {
		authHandler.RegisterRoutes(r)
		// Mock auth middleware that extracts userID from context
		authMiddleware := func(h http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// For testing, extract userID from context if set (using middleware's SetUserID)
				// The test sets userID using middleware.SetUserID, so we just pass it through
				// If userID is already in context (set by middleware.SetUserID), it will be used
				// Otherwise, the handler will return 401
				h.ServeHTTP(w, r)
			})
		}
		userSettingsHandler.RegisterRoutes(r, authMiddleware)
	})

	return r
}

// TestMain sets up and tears down the test environment
func TestMain(m *testing.M) {
	// Initialize logger
	var err error
	testLogger, err = zap.NewDevelopment()
	if err != nil {
		panic(fmt.Sprintf("Failed to initialize logger: %v", err))
	}

	// Setup test database
	cfg, err := config.LoadTestConfig()
	if err != nil {
		panic(fmt.Sprintf("Failed to load test config: %v", err))
	}
	dsn := cfg.DSN()
	if dsn == "" {
		// Default test database connection
		dsn = "root:password@tcp(localhost:3306)/japanesestudent_auth_test?parseTime=true&charset=utf8mb4"
	}

	testDB, err = sql.Open("mysql", dsn)
	if err != nil {
		panic(fmt.Sprintf("Failed to connect to test database: %v", err))
	}

	// Test connection
	if err = testDB.Ping(); err != nil {
		panic(fmt.Sprintf("Failed to ping test database: %v", err))
	}

	// Setup test schema
	setupTestSchemaForMain(testDB)

	// Setup test router
	testRouter = setupTestRouter(testDB, testLogger)

	// Run tests
	code := m.Run()

	// Cleanup
	if testDB != nil {
		testDB.Close()
	}
	os.Exit(code)
}

// setupTestSchemaForMain creates the test database schema (for TestMain)
func setupTestSchemaForMain(db *sql.DB) {
	usersTable := `
		CREATE TABLE IF NOT EXISTS users (
			id INT PRIMARY KEY AUTO_INCREMENT,
			username VARCHAR(255) NOT NULL UNIQUE,
			email VARCHAR(255) NOT NULL UNIQUE,
			password_hash VARCHAR(255) NOT NULL,
			role INT NOT NULL DEFAULT 1,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			INDEX idx_email (email),
			INDEX idx_username (username)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
	`

	userTokensTable := `
		CREATE TABLE IF NOT EXISTS user_tokens (
			id INT PRIMARY KEY AUTO_INCREMENT,
			user_id INT NOT NULL,
			token TEXT NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
	`

	userSettingsTable := `
		CREATE TABLE IF NOT EXISTS user_settings (
			id INT PRIMARY KEY AUTO_INCREMENT,
			user_id INT NOT NULL UNIQUE,
			new_word_count INT NOT NULL DEFAULT 20,
			old_word_count INT NOT NULL DEFAULT 20,
			alphabet_learn_count INT NOT NULL DEFAULT 10,
			language VARCHAR(10) NOT NULL DEFAULT 'en',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
	`

	db.Exec(usersTable)
	db.Exec(userTokensTable)
	db.Exec(userSettingsTable)
}

func TestIntegration_Register(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	tests := []struct {
		name           string
		requestBody    map[string]string
		expectedStatus int
		validateFunc   func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name: "success valid registration",
			requestBody: map[string]string{
				"email":    "newuser@example.com",
				"username": "newuser",
				"password": "Password123!",
			},
			expectedStatus: http.StatusCreated,
			validateFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response map[string]string
				err := json.NewDecoder(w.Body).Decode(&response)
				require.NoError(t, err)

				// Tokens are in cookies, not in JSON response
				accessToken := getCookieValue(w, "access_token")
				refreshToken := getCookieValue(w, "refresh_token")
				assert.NotEmpty(t, accessToken, "access token should be set in cookie")
				assert.NotEmpty(t, refreshToken, "refresh token should be set in cookie")

				// Verify user was created in database
				var count int
				err = testDB.QueryRow("SELECT COUNT(*) FROM users WHERE email = ?", "newuser@example.com").Scan(&count)
				require.NoError(t, err)
				assert.Equal(t, 1, count)

				// Verify password is hashed (not stored as plaintext)
				var passwordHash string
				err = testDB.QueryRow("SELECT password_hash FROM users WHERE email = ?", "newuser@example.com").Scan(&passwordHash)
				require.NoError(t, err)
				assert.NotEqual(t, "Password123!", passwordHash)
				assert.True(t, len(passwordHash) > 50) // bcrypt hashes are typically 60 characters
			},
		},
		{
			name: "duplicate email",
			requestBody: map[string]string{
				"email":    "test@example.com",
				"username": "anotheruser",
				"password": "Password123!",
			},
			expectedStatus: http.StatusBadRequest,
			validateFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response map[string]string
				err := json.NewDecoder(w.Body).Decode(&response)
				require.NoError(t, err)
				assert.Contains(t, response["error"], "email already exists")
			},
		},
		{
			name: "duplicate username",
			requestBody: map[string]string{
				"email":    "unique@example.com",
				"username": "testuser",
				"password": "Password123!",
			},
			expectedStatus: http.StatusBadRequest,
			validateFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response map[string]string
				err := json.NewDecoder(w.Body).Decode(&response)
				require.NoError(t, err)
				assert.Contains(t, response["error"], "username already exists")
			},
		},
		{
			name: "invalid email format",
			requestBody: map[string]string{
				"email":    "invalid-email",
				"username": "validuser",
				"password": "Password123!",
			},
			expectedStatus: http.StatusBadRequest,
			validateFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response map[string]string
				err := json.NewDecoder(w.Body).Decode(&response)
				require.NoError(t, err)
				assert.Contains(t, response["error"], "invalid email format")
			},
		},
		{
			name: "invalid password - too short",
			requestBody: map[string]string{
				"email":    "valid@example.com",
				"username": "validuser",
				"password": "Pass1!",
			},
			expectedStatus: http.StatusBadRequest,
			validateFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response map[string]string
				err := json.NewDecoder(w.Body).Decode(&response)
				require.NoError(t, err)
				assert.Contains(t, response["error"], "password must be at least 8 characters")
			},
		},
		{
			name: "empty username",
			requestBody: map[string]string{
				"email":    "valid@example.com",
				"username": "",
				"password": "Password123!",
			},
			expectedStatus: http.StatusBadRequest,
			validateFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response map[string]string
				err := json.NewDecoder(w.Body).Decode(&response)
				require.NoError(t, err)
				assert.Contains(t, response["error"], "username cannot be empty")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanupTestData(t, testDB)
			seedTestData(t, testDB)

			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPost, "/api/v3/auth/register", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			testRouter.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.validateFunc != nil {
				tt.validateFunc(t, w)
			}
		})
	}
}

func TestIntegration_Login(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	cleanupTestData(t, testDB)
	seedTestData(t, testDB)
	defer cleanupTestData(t, testDB)

	tests := []struct {
		name           string
		requestBody    map[string]string
		expectedStatus int
		validateFunc   func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name: "success login with email",
			requestBody: map[string]string{
				"login":    "test@example.com",
				"password": "Password123!",
			},
			expectedStatus: http.StatusOK,
			validateFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response map[string]string
				err := json.NewDecoder(w.Body).Decode(&response)
				require.NoError(t, err)

				// Tokens are in cookies, not in JSON response
				accessToken := getCookieValue(w, "access_token")
				refreshToken := getCookieValue(w, "refresh_token")
				assert.NotEmpty(t, accessToken, "access token should be set in cookie")
				assert.NotEmpty(t, refreshToken, "refresh token should be set in cookie")
			},
		},
		{
			name: "success login with username",
			requestBody: map[string]string{
				"login":    "testuser",
				"password": "Password123!",
			},
			expectedStatus: http.StatusOK,
			validateFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response map[string]string
				err := json.NewDecoder(w.Body).Decode(&response)
				require.NoError(t, err)

				// Tokens are in cookies, not in JSON response
				accessToken := getCookieValue(w, "access_token")
				refreshToken := getCookieValue(w, "refresh_token")
				assert.NotEmpty(t, accessToken, "access token should be set in cookie")
				assert.NotEmpty(t, refreshToken, "refresh token should be set in cookie")
			},
		},
		{
			name: "wrong password",
			requestBody: map[string]string{
				"login":    "test@example.com",
				"password": "WrongPassword123!",
			},
			expectedStatus: http.StatusUnauthorized,
			validateFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response map[string]string
				err := json.NewDecoder(w.Body).Decode(&response)
				require.NoError(t, err)
				assert.Contains(t, response["error"], "invalid credentials")
			},
		},
		{
			name: "user not found",
			requestBody: map[string]string{
				"login":    "nonexistent@example.com",
				"password": "Password123!",
			},
			expectedStatus: http.StatusUnauthorized,
			validateFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response map[string]string
				err := json.NewDecoder(w.Body).Decode(&response)
				require.NoError(t, err)
				// The error can be either "user not found" or "invalid credentials"
				assert.True(t, strings.Contains(response["error"], "invalid credentials") || strings.Contains(response["error"], "user not found"),
					"error should contain 'invalid credentials' or 'user not found', got: %s", response["error"])
			},
		},
		{
			name: "case insensitive email",
			requestBody: map[string]string{
				"login":    "TEST@EXAMPLE.COM",
				"password": "Password123!",
			},
			expectedStatus: http.StatusOK,
			validateFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response map[string]string
				err := json.NewDecoder(w.Body).Decode(&response)
				require.NoError(t, err)

				// Tokens are in cookies, not in JSON response
				accessToken := getCookieValue(w, "access_token")
				assert.NotEmpty(t, accessToken, "access token should be set in cookie")
			},
		},
		{
			name: "empty credentials",
			requestBody: map[string]string{
				"login":    "",
				"password": "",
			},
			expectedStatus: http.StatusUnauthorized,
			validateFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response map[string]string
				err := json.NewDecoder(w.Body).Decode(&response)
				require.NoError(t, err)
				assert.Contains(t, response["error"], "cannot be empty")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest(http.MethodPost, "/api/v3/auth/login", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			testRouter.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.validateFunc != nil {
				tt.validateFunc(t, w)
			}
		})
	}
}

func TestIntegration_Refresh(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	cleanupTestData(t, testDB)
	seedTestData(t, testDB)
	defer cleanupTestData(t, testDB)

	// First, login to get a valid refresh token
	loginBody, _ := json.Marshal(map[string]string{
		"login":    "test@example.com",
		"password": "Password123!",
	})
	loginReq := httptest.NewRequest(http.MethodPost, "/api/v3/auth/login", bytes.NewBuffer(loginBody))
	loginReq.Header.Set("Content-Type", "application/json")
	loginW := httptest.NewRecorder()
	testRouter.ServeHTTP(loginW, loginReq)

	// Verify login was successful
	require.Equal(t, http.StatusOK, loginW.Code, "login should succeed before testing refresh")

	// Extract refresh token from cookie
	validRefreshToken := getCookieValue(loginW, "refresh_token")
	require.NotEmpty(t, validRefreshToken, "refresh token should be set in cookie after login")

	tests := []struct {
		name           string
		refreshToken   string
		expectedStatus int
		validateFunc   func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:           "success valid refresh token",
			refreshToken:   validRefreshToken,
			expectedStatus: http.StatusOK,
			validateFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response map[string]string
				err := json.NewDecoder(w.Body).Decode(&response)
				require.NoError(t, err)

				// Tokens are in cookies, not in JSON response
				accessToken := getCookieValue(w, "access_token")
				newRefreshToken := getCookieValue(w, "refresh_token")
				assert.NotEmpty(t, accessToken, "access token should be set in cookie")
				assert.NotEmpty(t, newRefreshToken, "refresh token should be set in cookie")
				// New tokens should be different from old token
				assert.NotEqual(t, validRefreshToken, newRefreshToken, "new refresh token should be different from old one")
			},
		},
		{
			name:           "invalid token format",
			refreshToken:   "invalid-token",
			expectedStatus: http.StatusInternalServerError,
			validateFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response map[string]string
				err := json.NewDecoder(w.Body).Decode(&response)
				require.NoError(t, err)
				assert.Contains(t, response["error"], "invalid")
			},
		},
		{
			name:           "token not in database",
			refreshToken:   "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3MDAwMDAwMDAsImlhdCI6MTYwMDAwMDAwMCwidHlwZSI6InJlZnJlc2gifQ.test",
			expectedStatus: http.StatusInternalServerError,
			validateFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response map[string]string
				err := json.NewDecoder(w.Body).Decode(&response)
				require.NoError(t, err)
				assert.NotEmpty(t, response["error"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(map[string]string{
				"refresh_token": tt.refreshToken,
			})
			req := httptest.NewRequest(http.MethodPost, "/api/v3/auth/refresh", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			testRouter.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.validateFunc != nil {
				tt.validateFunc(t, w)
			}
		})
	}
}

func TestIntegration_RepositoryLayer(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	cleanupTestData(t, testDB)
	seedTestData(t, testDB)
	defer cleanupTestData(t, testDB)

	userRepo := repositories.NewUserRepository(testDB)
	tokenRepo := repositories.NewUserTokenRepository(testDB)
	ctx := context.Background()

	t.Run("UserRepository Create", func(t *testing.T) {
		user := &models.User{
			Username:     "repotest",
			Email:        "repotest@example.com",
			PasswordHash: "hashedpassword",
			Role:         models.RoleUser,
		}
		err := userRepo.Create(ctx, user)
		require.NoError(t, err)
		assert.Greater(t, user.ID, 0)
	})

	t.Run("UserRepository GetByEmailOrUsername", func(t *testing.T) {
		user, err := userRepo.GetByEmailOrUsername(ctx, "test@example.com")
		require.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, "test@example.com", user.Email)
		assert.Equal(t, "testuser", user.Username)
	})

	t.Run("UserRepository ExistsByEmail", func(t *testing.T) {
		exists, err := userRepo.ExistsByEmail(ctx, "test@example.com")
		require.NoError(t, err)
		assert.True(t, exists)

		exists, err = userRepo.ExistsByEmail(ctx, "nonexistent@example.com")
		require.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("UserRepository ExistsByUsername", func(t *testing.T) {
		exists, err := userRepo.ExistsByUsername(ctx, "testuser")
		require.NoError(t, err)
		assert.True(t, exists)

		exists, err = userRepo.ExistsByUsername(ctx, "nonexistentuser")
		require.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("UserTokenRepository Create and GetByToken", func(t *testing.T) {
		token := &models.UserToken{
			UserID: 1,
			Token:  "test-refresh-token-123",
		}
		err := tokenRepo.Create(ctx, token)
		require.NoError(t, err)

		retrieved, err := tokenRepo.GetByToken(ctx, "test-refresh-token-123")
		require.NoError(t, err)
		assert.NotNil(t, retrieved)
		assert.Equal(t, 1, retrieved.UserID)
	})

	t.Run("UserTokenRepository UpdateToken", func(t *testing.T) {
		// Create initial token
		token := &models.UserToken{
			UserID: 1,
			Token:  "old-token",
		}
		err := tokenRepo.Create(ctx, token)
		require.NoError(t, err)

		// Update token
		err = tokenRepo.UpdateToken(ctx, "old-token", "new-token", 1)
		require.NoError(t, err)

		// Verify old token doesn't exist
		_, err = tokenRepo.GetByToken(ctx, "old-token")
		assert.Error(t, err)

		// Verify new token exists
		retrieved, err := tokenRepo.GetByToken(ctx, "new-token")
		require.NoError(t, err)
		assert.Equal(t, 1, retrieved.UserID)
	})

	t.Run("UserTokenRepository DeleteByToken", func(t *testing.T) {
		// Create token
		token := &models.UserToken{
			UserID: 1,
			Token:  "token-to-delete",
		}
		err := tokenRepo.Create(ctx, token)
		require.NoError(t, err)

		// Delete token
		err = tokenRepo.DeleteByToken(ctx, "token-to-delete")
		require.NoError(t, err)

		// Verify token doesn't exist
		_, err = tokenRepo.GetByToken(ctx, "token-to-delete")
		assert.Error(t, err)
	})
}

func TestIntegration_ServiceLayer(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	cleanupTestData(t, testDB)
	seedTestData(t, testDB)
	defer cleanupTestData(t, testDB)

	logger, _ := zap.NewDevelopment()
	userRepo := repositories.NewUserRepository(testDB)
	tokenRepo := repositories.NewUserTokenRepository(testDB)
	userSettingsRepo := repositories.NewUserSettingsRepository(testDB)
	tokenGen := service.NewTokenGenerator("test-secret", 1*time.Hour, 7*24*time.Hour)
	authSvc := services.NewAuthService(userRepo, tokenRepo, userSettingsRepo, tokenGen, logger)
	ctx := context.Background()

	t.Run("Register", func(t *testing.T) {
		req := &models.RegisterRequest{
			Email:    "servicetest@example.com",
			Username: "servicetest",
			Password: "Password123!",
		}
		accessToken, refreshToken, err := authSvc.Register(ctx, req)
		require.NoError(t, err)
		assert.NotEmpty(t, accessToken)
		assert.NotEmpty(t, refreshToken)
	})

	t.Run("Login", func(t *testing.T) {
		req := &models.LoginRequest{
			Login:    "test@example.com",
			Password: "Password123!",
		}
		accessToken, refreshToken, err := authSvc.Login(ctx, req)
		require.NoError(t, err)
		assert.NotEmpty(t, accessToken)
		assert.NotEmpty(t, refreshToken)
	})

	t.Run("Refresh", func(t *testing.T) {
		// First login to get a refresh token
		loginReq := &models.LoginRequest{
			Login:    "test@example.com",
			Password: "Password123!",
		}
		_, refreshToken, err := authSvc.Login(ctx, loginReq)
		require.NoError(t, err)

		// Refresh the token
		newAccessToken, newRefreshToken, err := authSvc.Refresh(ctx, refreshToken)
		require.NoError(t, err)
		assert.NotEmpty(t, newAccessToken)
		assert.NotEmpty(t, newRefreshToken)
		assert.NotEqual(t, refreshToken, newRefreshToken)
	})
}

func TestIntegration_UserSettings(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	cleanupTestData(t, testDB)
	seedTestData(t, testDB)
	defer cleanupTestData(t, testDB)

	// Create user settings for test user
	_, err := testDB.Exec("INSERT INTO user_settings (user_id, new_word_count, old_word_count, alphabet_learn_count, language) VALUES (1, 20, 20, 10, 'en')")
	require.NoError(t, err, "Failed to seed user settings")

	tests := []struct {
		name           string
		userID         int
		method         string
		url            string
		requestBody    map[string]any
		expectedStatus int
		validateFunc   func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:           "success get user settings",
			userID:         1,
			method:         http.MethodGet,
			url:            "/api/v3/settings",
			requestBody:    nil,
			expectedStatus: http.StatusOK,
			validateFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response models.UserSettingsResponse
				err := json.NewDecoder(w.Body).Decode(&response)
				require.NoError(t, err)
				assert.Equal(t, 20, response.NewWordCount)
				assert.Equal(t, 20, response.OldWordCount)
				assert.Equal(t, 10, response.AlphabetLearnCount)
				assert.Equal(t, models.LanguageEnglish, response.Language)
			},
		},
		{
			name:           "success update user settings - partial",
			userID:         1,
			method:         http.MethodPatch,
			url:            "/api/v3/settings",
			requestBody:    map[string]any{"newWordCount": 25},
			expectedStatus: http.StatusNoContent,
			validateFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				// Verify database was updated
				var newWordCount int
				err := testDB.QueryRow("SELECT new_word_count FROM user_settings WHERE user_id = ?", 1).Scan(&newWordCount)
				require.NoError(t, err)
				assert.Equal(t, 25, newWordCount)
			},
		},
		{
			name:           "success update user settings - all fields",
			userID:         1,
			method:         http.MethodPatch,
			url:            "/api/v3/settings",
			requestBody:    map[string]any{"newWordCount": 30, "oldWordCount": 35, "alphabetLearnCount": 12, "language": "ru"},
			expectedStatus: http.StatusNoContent,
			validateFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				// Verify database was updated
				var settings models.UserSettings
				err := testDB.QueryRow("SELECT new_word_count, old_word_count, alphabet_learn_count, language FROM user_settings WHERE user_id = ?", 1).
					Scan(&settings.NewWordCount, &settings.OldWordCount, &settings.AlphabetLearnCount, &settings.Language)
				require.NoError(t, err)
				assert.Equal(t, 30, settings.NewWordCount)
				assert.Equal(t, 35, settings.OldWordCount)
				assert.Equal(t, 12, settings.AlphabetLearnCount)
				assert.Equal(t, models.LanguageRussian, settings.Language)
			},
		},
		{
			name:           "invalid newWordCount - too low",
			userID:         1,
			method:         http.MethodPatch,
			url:            "/api/v3/settings",
			requestBody:    map[string]any{"newWordCount": 5},
			expectedStatus: http.StatusBadRequest,
			validateFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response map[string]string
				err := json.NewDecoder(w.Body).Decode(&response)
				require.NoError(t, err)
				assert.Contains(t, response["error"], "newWordCount must be between 10 and 40")
			},
		},
		{
			name:           "invalid newWordCount - too high",
			userID:         1,
			method:         http.MethodPatch,
			url:            "/api/v3/settings",
			requestBody:    map[string]any{"newWordCount": 50},
			expectedStatus: http.StatusBadRequest,
			validateFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response map[string]string
				err := json.NewDecoder(w.Body).Decode(&response)
				require.NoError(t, err)
				assert.Contains(t, response["error"], "newWordCount must be between 10 and 40")
			},
		},
		{
			name:           "invalid language",
			userID:         1,
			method:         http.MethodPatch,
			url:            "/api/v3/settings",
			requestBody:    map[string]any{"language": "fr"},
			expectedStatus: http.StatusBadRequest,
			validateFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response map[string]string
				err := json.NewDecoder(w.Body).Decode(&response)
				require.NoError(t, err)
				assert.Contains(t, response["error"], "invalid language")
			},
		},
		{
			name:           "empty request body",
			userID:         1,
			method:         http.MethodPatch,
			url:            "/api/v3/settings",
			requestBody:    map[string]any{},
			expectedStatus: http.StatusBadRequest,
			validateFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response map[string]string
				err := json.NewDecoder(w.Body).Decode(&response)
				require.NoError(t, err)
				assert.Contains(t, response["error"], "at least one field must be provided")
			},
		},
		{
			name:           "user settings not found",
			userID:         999,
			method:         http.MethodGet,
			url:            "/api/v3/settings",
			requestBody:    nil,
			expectedStatus: http.StatusInternalServerError,
			validateFunc:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req *http.Request
			if tt.requestBody != nil {
				body, _ := json.Marshal(tt.requestBody)
				req = httptest.NewRequest(tt.method, tt.url, bytes.NewBuffer(body))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req = httptest.NewRequest(tt.method, tt.url, nil)
			}
			// Set userID in context for auth middleware (using middleware's SetUserID)
			req = req.WithContext(middleware.SetUserID(req.Context(), tt.userID))
			w := httptest.NewRecorder()

			testRouter.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.validateFunc != nil {
				tt.validateFunc(t, w)
			}
		})
	}
}

func TestIntegration_UserSettingsRepositoryLayer(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	cleanupTestData(t, testDB)
	seedTestData(t, testDB)
	defer cleanupTestData(t, testDB)

	userSettingsRepo := repositories.NewUserSettingsRepository(testDB)
	ctx := context.Background()

	t.Run("Create user settings", func(t *testing.T) {
		err := userSettingsRepo.Create(ctx, 1)
		require.NoError(t, err)
	})

	t.Run("GetByUserId", func(t *testing.T) {
		settings, err := userSettingsRepo.GetByUserId(ctx, 1)
		require.NoError(t, err)
		assert.NotNil(t, settings)
		assert.Equal(t, 1, settings.UserID)
	})

	t.Run("Update user settings", func(t *testing.T) {
		settings := &models.UserSettings{
			NewWordCount:       25,
			OldWordCount:       30,
			AlphabetLearnCount: 12,
			Language:           models.LanguageRussian,
		}
		err := userSettingsRepo.Update(ctx, 1, settings)
		require.NoError(t, err)

		// Verify update
		updated, err := userSettingsRepo.GetByUserId(ctx, 1)
		require.NoError(t, err)
		assert.Equal(t, 25, updated.NewWordCount)
		assert.Equal(t, 30, updated.OldWordCount)
		assert.Equal(t, 12, updated.AlphabetLearnCount)
		assert.Equal(t, models.LanguageRussian, updated.Language)
	})
}
