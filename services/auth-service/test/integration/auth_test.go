package integration

// NOTE: Email Verification Testing
//
// This integration test suite is configured to avoid sending emails to the task microservice
// by using empty taskBaseURL and apiKey values. This prevents actual HTTP requests to the
// task service during testing.
//
// Email verification functionality (sending verification emails, verifying tokens, etc.)
// should be tested on a real live server with the task microservice running, as it involves:
// - HTTP communication with the task microservice
// - Email delivery through the task service
// - End-to-end verification flow
//
// In these tests:
// - Users are created with active=false (as in production)
// - Registration fails when email sending is attempted (taskBaseURL is empty)
// - Test users are manually activated after registration to allow login tests
// - This approach allows testing the core registration/login logic without external dependencies

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	_ "github.com/go-sql-driver/mysql"
	"github.com/Sheliakhin-Golang-portfolio/JapaneseStudent/services/auth-service/internal/handlers"
	"github.com/Sheliakhin-Golang-portfolio/JapaneseStudent/services/auth-service/internal/models"
	"github.com/Sheliakhin-Golang-portfolio/JapaneseStudent/services/auth-service/internal/repositories"
	"github.com/Sheliakhin-Golang-portfolio/JapaneseStudent/services/auth-service/internal/services"
	"github.com/Sheliakhin-Golang-portfolio/JapaneseStudent/libs/auth/middleware"
	"github.com/Sheliakhin-Golang-portfolio/JapaneseStudent/libs/auth/service"
	"github.com/Sheliakhin-Golang-portfolio/JapaneseStudent/libs/config"
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
	// NOTE: Setting active=true for test users to allow login tests to work.
	// In real scenarios, users are created with active=false and must verify email first.
	// Email verification should be tested on a real live server with the task microservice running.
	passwordHash, err := bcrypt.GenerateFromPassword([]byte("Password123!"), bcrypt.DefaultCost)
	require.NoError(t, err, "Failed to hash password")

	query := `INSERT INTO users (username, email, password_hash, role, avatar, active) VALUES (?, ?, ?, ?, ?, ?)`
	_, err = db.Exec(query, "testuser", "test@example.com", string(passwordHash), models.RoleUser, "", true)
	require.NoError(t, err, "Failed to seed test user")

	// Insert test tutors
	tutorPasswordHash, err := bcrypt.GenerateFromPassword([]byte("Password123!"), bcrypt.DefaultCost)
	require.NoError(t, err, "Failed to hash tutor password")
	_, err = db.Exec(query, "tutor1", "tutor1@example.com", string(tutorPasswordHash), models.RoleTutor, "", true)
	require.NoError(t, err, "Failed to seed test tutor1")
	_, err = db.Exec(query, "tutor2", "tutor2@example.com", string(tutorPasswordHash), models.RoleTutor, "", true)
	require.NoError(t, err, "Failed to seed test tutor2")
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
// NOTE: taskBaseURL and apiKey are set to empty strings to avoid sending emails to task microservice in tests.
// Email verification functionality should be tested on a real live server with the task microservice running.
func setupTestRouter(db *sql.DB, logger *zap.Logger, cfg *config.Config) chi.Router {
	userRepo := repositories.NewUserRepository(db)
	tokenRepo := repositories.NewUserTokenRepository(db)
	userSettingsRepo := repositories.NewUserSettingsRepository(db)

	// Use JWT config from LoadTestConfig, with fallback defaults for tests
	jwtSecret := cfg.JWT.Secret
	if jwtSecret == "" {
		jwtSecret = "test-secret-key-for-integration-tests"
	}
	accessExpiry := cfg.JWT.AccessTokenExpiry
	if accessExpiry == 0 {
		accessExpiry = 1 * time.Hour
	}
	refreshExpiry := cfg.JWT.RefreshTokenExpiry
	if refreshExpiry == 0 {
		refreshExpiry = 7 * 24 * time.Hour
	}
	tokenGen := service.NewTokenGenerator(jwtSecret, accessExpiry, refreshExpiry)

	// Using empty taskBaseURL and apiKey to prevent email sending in tests
	verificationURL := "http://localhost:8080"
	apiKey := "test-api-key"
	authSvc := services.NewAuthService(userRepo, tokenRepo, userSettingsRepo, tokenGen, logger, verificationURL, apiKey, "", "")
	authHandler := handlers.NewAuthHandler(authSvc, logger)

	userSettingsSvc := services.NewUserSettingsService(userSettingsRepo)
	// Using empty scheduledTaskBaseURL to avoid calling task-service in tests
	// NOTE: Task-service integration (creating/deleting scheduled tasks) should be tested
	// on a live server with the task-service running.
	profileSvc := services.NewProfileService(userRepo, userSettingsRepo, tokenGen, "", "", "", "", "", "", false)
	profileHandler := handlers.NewProfileHandler(profileSvc, userSettingsSvc, logger)

	// Using empty scheduledTaskBaseURL and learnServiceBaseURL to avoid calling external services in tests
	// NOTE: External service integration should be tested on a live server with the services running.
	adminSvc := services.NewAdminService(userRepo, tokenRepo, userSettingsRepo, tokenGen, logger, "", "", "", "", false, "")
	adminHandler := handlers.NewAdminHandler(adminSvc, logger, "", false, "")

	tokenCleaningHandler := handlers.NewTokenCleaningHandler(tokenRepo, logger, refreshExpiry)

	r := chi.NewRouter()
	// Scope router to /api/v6 to match main.go setup
	r.Route("/api/v6", func(r chi.Router) {
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
		profileHandler.RegisterRoutes(r, authMiddleware)
		// Register admin routes without middleware for testing (we'll test the endpoint directly)
		adminHandler.RegisterRoutes(r)
		// Register token cleaning handler
		tokenCleaningHandler.RegisterRoutes(r)
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
	// NOTE: Using empty taskBaseURL and apiKey to avoid sending emails to task microservice in tests.
	// Email verification functionality should be tested on a real live server with the task microservice running.
	testRouter = setupTestRouter(testDB, testLogger, cfg)

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
	// Drop tables if they exist to ensure clean schema
	db.Exec("DROP TABLE IF EXISTS user_settings")
	db.Exec("DROP TABLE IF EXISTS user_tokens")
	db.Exec("DROP TABLE IF EXISTS users")

	usersTable := `
		CREATE TABLE users (
			id INT PRIMARY KEY AUTO_INCREMENT,
			username VARCHAR(255) NOT NULL UNIQUE,
			email VARCHAR(255) NOT NULL UNIQUE,
			password_hash VARCHAR(255) NOT NULL,
			role INT NOT NULL DEFAULT 1,
			avatar VARCHAR(500) NULL,
			active BOOLEAN NOT NULL DEFAULT FALSE,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			INDEX idx_email (email),
			INDEX idx_username (username)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
	`

	userTokensTable := `
		CREATE TABLE user_tokens (
			id INT PRIMARY KEY AUTO_INCREMENT,
			user_id INT NOT NULL,
			token TEXT NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
	`

	userSettingsTable := `
		CREATE TABLE user_settings (
			id INT PRIMARY KEY AUTO_INCREMENT,
			user_id INT NOT NULL UNIQUE,
			new_word_count INT NOT NULL DEFAULT 20,
			old_word_count INT NOT NULL DEFAULT 20,
			alphabet_learn_count INT NOT NULL DEFAULT 10,
			language VARCHAR(10) NOT NULL DEFAULT 'en',
			alphabet_repeat VARCHAR(20) NOT NULL DEFAULT 'in question',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
	`

	db.Exec(usersTable)
	db.Exec(userTokensTable)
	db.Exec(userSettingsTable)
}

// TestIntegration_Register tests user registration.
// NOTE: Email verification is skipped in integration tests (taskBaseURL is empty).
// After registration, users are manually activated in the database to allow login tests.
// Email verification functionality should be tested on a real live server with the task microservice running.
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
			expectedStatus: http.StatusAccepted, // 202 Accepted - user created but email sending failed (taskBaseURL is empty in tests)
			validateFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response map[string]string
				err := json.NewDecoder(w.Body).Decode(&response)
				require.NoError(t, err)
				// Registration fails because we can't send verification email (taskBaseURL is empty in tests)
				assert.Contains(t, response["error"], "cannot send verification email")

				// However, user should still be created in database (registration happens before email sending)
				var count int
				err = testDB.QueryRow("SELECT COUNT(*) FROM users WHERE email = ?", "newuser@example.com").Scan(&count)
				require.NoError(t, err)
				assert.Equal(t, 1, count, "user should be created even if email sending fails")

				// Verify password is hashed (not stored as plaintext)
				var passwordHash string
				err = testDB.QueryRow("SELECT password_hash FROM users WHERE email = ?", "newuser@example.com").Scan(&passwordHash)
				require.NoError(t, err)
				assert.NotEqual(t, "Password123!", passwordHash)
				assert.True(t, len(passwordHash) > 50) // bcrypt hashes are typically 60 characters

				// Verify user is inactive (email not verified)
				var active bool
				err = testDB.QueryRow("SELECT active FROM users WHERE email = ?", "newuser@example.com").Scan(&active)
				require.NoError(t, err)
				assert.False(t, active, "user should be inactive until email is verified")

				// Manually activate user for testing purposes (in real scenario, this happens via email verification)
				_, err = testDB.Exec("UPDATE users SET active = true WHERE email = ?", "newuser@example.com")
				require.NoError(t, err, "failed to activate user for testing")

				// Verify avatar is empty (not touching media service in tests)
				var avatar string
				err = testDB.QueryRow("SELECT avatar FROM users WHERE email = ?", "newuser@example.com").Scan(&avatar)
				require.NoError(t, err)
				assert.Empty(t, avatar, "avatar should be empty in tests")
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
				assert.Contains(t, response["error"], "email, username, and password are required")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanupTestData(t, testDB)
			seedTestData(t, testDB)

			// Create multipart form request
			var body bytes.Buffer
			writer := multipart.NewWriter(&body)

			for key, value := range tt.requestBody {
				writer.WriteField(key, value)
			}
			writer.Close()

			req := httptest.NewRequest(http.MethodPost, "/api/v6/auth/register", &body)
			req.Header.Set("Content-Type", writer.FormDataContentType())
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
			req := httptest.NewRequest(http.MethodPost, "/api/v6/auth/login", bytes.NewBuffer(body))
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

// NOTE: Refresh method is not tested in integration tests.
// The Refresh method uses goroutines for parallel validation (token database lookup and JWT validation),
// which makes it difficult to reliably test in integration tests due to timing and race condition issues.
// Refresh functionality is tested in unit tests (auth_service_test.go) instead.

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
			Avatar:       "", // Empty avatar for tests
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

	// Load test config for JWT settings
	cfg, err := config.LoadTestConfig()
	require.NoError(t, err)

	// Use JWT config from LoadTestConfig, with fallback defaults for tests
	jwtSecret := cfg.JWT.Secret
	if jwtSecret == "" {
		jwtSecret = "test-secret"
	}
	accessExpiry := cfg.JWT.AccessTokenExpiry
	if accessExpiry == 0 {
		accessExpiry = 1 * time.Hour
	}
	refreshExpiry := cfg.JWT.RefreshTokenExpiry
	if refreshExpiry == 0 {
		refreshExpiry = 7 * 24 * time.Hour
	}
	tokenGen := service.NewTokenGenerator(jwtSecret, accessExpiry, refreshExpiry)

	// Using empty taskBaseURL and apiKey to prevent email sending in tests
	// NOTE: Email verification functionality should be tested on a real live server with the task microservice running.
	authSvc := services.NewAuthService(userRepo, tokenRepo, userSettingsRepo, tokenGen, logger, "http://localhost:8080", "test-api-key", "", "")
	ctx := context.Background()

	t.Run("Register", func(t *testing.T) {
		req := &models.RegisterRequest{
			Email:    "servicetest@example.com",
			Username: "servicetest",
			Password: "Password123!",
		}
		// Register will fail because taskBaseURL is empty (email sending disabled in tests)
		err := authSvc.Register(ctx, req, nil, "") // Using nil avatarFile and empty avatarFilename to avoid touching media service
		require.Error(t, err, "registration should fail when taskBaseURL is empty")
		assert.Contains(t, err.Error(), "cannot send verification email")

		// Verify user was created despite email failure
		user, err := userRepo.GetByEmailOrUsername(ctx, "servicetest@example.com")
		require.NoError(t, err)
		assert.NotNil(t, user)
		assert.False(t, user.Active, "user should be inactive until email is verified")

		// Manually activate user for testing purposes (in real scenario, this happens via email verification)
		err = userRepo.UpdateActive(ctx, user.ID, true)
		require.NoError(t, err, "failed to activate user for testing")
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

	// NOTE: Refresh test is not included here.
	// The Refresh method uses goroutines for parallel validation (token database lookup and JWT validation),
	// which makes it difficult to reliably test in integration tests due to timing and race condition issues.
	// Refresh functionality is tested in unit tests (auth_service_test.go) instead.
}

func TestIntegration_UserSettings(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	cleanupTestData(t, testDB)
	seedTestData(t, testDB)
	defer cleanupTestData(t, testDB)

	// Create user settings for test user
	_, err := testDB.Exec("INSERT INTO user_settings (user_id, new_word_count, old_word_count, alphabet_learn_count, language, alphabet_repeat) VALUES (1, 20, 20, 10, 'en', 'in question')")
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
			url:            "/api/v6/profile/settings",
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
				assert.Equal(t, models.RepeatTypeInQuestion, response.AlphabetRepeat)
			},
		},
		{
			name:           "success update user settings - partial",
			userID:         1,
			method:         http.MethodPatch,
			url:            "/api/v6/profile/settings",
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
			url:            "/api/v6/profile/settings",
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
			url:            "/api/v6/profile/settings",
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
			url:            "/api/v6/profile/settings",
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
			url:            "/api/v6/profile/settings",
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
			url:            "/api/v6/profile/settings",
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
			url:            "/api/v6/profile/settings",
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

func TestIntegration_GetTutorsList(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	cleanupTestData(t, testDB)
	seedTestData(t, testDB)
	defer cleanupTestData(t, testDB)

	tests := []struct {
		name           string
		expectedStatus int
		validateFunc   func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:           "success get tutors list",
			expectedStatus: http.StatusOK,
			validateFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response []models.TutorListItem
				err := json.NewDecoder(w.Body).Decode(&response)
				require.NoError(t, err)
				assert.GreaterOrEqual(t, len(response), 2, "should return at least 2 tutors")
				// Check that all returned items have tutor role
				tutorIDs := make(map[int]bool)
				for _, tutor := range response {
					assert.Greater(t, tutor.ID, 0, "tutor ID should be positive")
					assert.NotEmpty(t, tutor.Username, "tutor username should not be empty")
					tutorIDs[tutor.ID] = true
				}
				// Verify tutors exist in database
				for tutorID := range tutorIDs {
					var role int
					err := testDB.QueryRow("SELECT role FROM users WHERE id = ?", tutorID).Scan(&role)
					require.NoError(t, err)
					assert.Equal(t, int(models.RoleTutor), role, "user should have tutor role")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v6/admin/tutors", nil)
			w := httptest.NewRecorder()

			testRouter.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.validateFunc != nil {
				tt.validateFunc(t, w)
			}
		})
	}
}

func TestIntegration_AdminGetUsersList(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	cleanupTestData(t, testDB)
	seedTestData(t, testDB)
	defer cleanupTestData(t, testDB)

	tests := []struct {
		name           string
		queryParams    string
		expectedStatus int
		validateFunc   func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:           "success get users list with defaults",
			queryParams:    "",
			expectedStatus: http.StatusOK,
			validateFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response []models.UserListItem
				err := json.NewDecoder(w.Body).Decode(&response)
				require.NoError(t, err)
				assert.GreaterOrEqual(t, len(response), 3, "should return at least 3 users")
			},
		},
		{
			name:           "success with pagination",
			queryParams:    "?page=1&count=2",
			expectedStatus: http.StatusOK,
			validateFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response []models.UserListItem
				err := json.NewDecoder(w.Body).Decode(&response)
				require.NoError(t, err)
				assert.LessOrEqual(t, len(response), 2)
			},
		},
		{
			name:           "success with role filter",
			queryParams:    "?role=2",
			expectedStatus: http.StatusOK,
			validateFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response []models.UserListItem
				err := json.NewDecoder(w.Body).Decode(&response)
				require.NoError(t, err)
				for _, user := range response {
					assert.Equal(t, models.RoleTutor, user.Role)
				}
			},
		},
		{
			name:           "success with search filter",
			queryParams:    "?search=test",
			expectedStatus: http.StatusOK,
			validateFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response []models.UserListItem
				err := json.NewDecoder(w.Body).Decode(&response)
				require.NoError(t, err)
				for _, user := range response {
					assert.True(t, strings.Contains(strings.ToLower(user.Username), "test") ||
						strings.Contains(strings.ToLower(user.Email), "test"))
				}
			},
		},
		{
			name:           "invalid page parameter",
			queryParams:    "?page=invalid",
			expectedStatus: http.StatusOK,
			validateFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response []models.UserListItem
				err := json.NewDecoder(w.Body).Decode(&response)
				require.NoError(t, err)
				// Should default to page 1
			},
		},
		{
			name:           "invalid count parameter",
			queryParams:    "?count=invalid",
			expectedStatus: http.StatusOK,
			validateFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response []models.UserListItem
				err := json.NewDecoder(w.Body).Decode(&response)
				require.NoError(t, err)
				// Should default to count 20
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v6/admin/users"+tt.queryParams, nil)
			w := httptest.NewRecorder()

			testRouter.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.validateFunc != nil {
				tt.validateFunc(t, w)
			}
		})
	}
}

func TestIntegration_AdminGetUserWithSettings(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	cleanupTestData(t, testDB)
	seedTestData(t, testDB)
	defer cleanupTestData(t, testDB)

	// Create user settings for test user
	_, err := testDB.Exec("INSERT INTO user_settings (user_id, new_word_count, old_word_count, alphabet_learn_count, language, alphabet_repeat) VALUES (1, 20, 20, 10, 'en', 'in question')")
	require.NoError(t, err, "Failed to seed user settings")

	tests := []struct {
		name           string
		userID         string
		expectedStatus int
		validateFunc   func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:           "success get user with settings",
			userID:         "1",
			expectedStatus: http.StatusOK,
			validateFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response models.UserWithSettingsResponse
				err := json.NewDecoder(w.Body).Decode(&response)
				require.NoError(t, err)
				assert.Equal(t, 1, response.ID)
				assert.Equal(t, "testuser", response.Username)
				assert.NotNil(t, response.Settings)
				assert.Equal(t, 20, response.Settings.NewWordCount)
			},
		},
		{
			name:           "success get user without settings",
			userID:         "2",
			expectedStatus: http.StatusOK,
			validateFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response models.UserWithSettingsResponse
				err := json.NewDecoder(w.Body).Decode(&response)
				require.NoError(t, err)
				assert.Equal(t, 2, response.ID)
				assert.Nil(t, response.Settings, "settings should be nil if not created")
			},
		},
		{
			name:           "user not found",
			userID:         "999",
			expectedStatus: http.StatusNotFound,
			validateFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response map[string]string
				err := json.NewDecoder(w.Body).Decode(&response)
				require.NoError(t, err)
				assert.Contains(t, response["error"], "not found")
			},
		},
		{
			name:           "invalid user ID",
			userID:         "invalid",
			expectedStatus: http.StatusBadRequest,
			validateFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response map[string]string
				err := json.NewDecoder(w.Body).Decode(&response)
				require.NoError(t, err)
				assert.Contains(t, response["error"], "invalid user ID")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v6/admin/users/"+tt.userID, nil)
			w := httptest.NewRecorder()

			testRouter.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.validateFunc != nil {
				tt.validateFunc(t, w)
			}
		})
	}
}

func TestIntegration_AdminCreateUser(t *testing.T) {
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
			name: "success create user",
			requestBody: map[string]string{
				"username": "newuser",
				"email":    "newuser@example.com",
				"password": "Password123!",
				"role":     "1",
			},
			expectedStatus: http.StatusCreated,
			validateFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response map[string]any
				err := json.NewDecoder(w.Body).Decode(&response)
				require.NoError(t, err)
				assert.Contains(t, response["message"], "user created successfully")
				assert.Greater(t, int(response["userId"].(float64)), 0)

				// Verify user was created in database
				var count int
				err = testDB.QueryRow("SELECT COUNT(*) FROM users WHERE email = ?", "newuser@example.com").Scan(&count)
				require.NoError(t, err)
				assert.Equal(t, 1, count)
			},
		},
		{
			name: "duplicate email",
			requestBody: map[string]string{
				"username": "anotheruser",
				"email":    "test@example.com",
				"password": "Password123!",
				"role":     "1",
			},
			expectedStatus: http.StatusBadRequest,
			validateFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response map[string]string
				err := json.NewDecoder(w.Body).Decode(&response)
				require.NoError(t, err)
				assert.Contains(t, response["error"], "already exists")
			},
		},
		{
			name: "missing required fields",
			requestBody: map[string]string{
				"username": "newuser",
				"email":    "newuser@example.com",
				// password and role missing
			},
			expectedStatus: http.StatusBadRequest,
			validateFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response map[string]string
				err := json.NewDecoder(w.Body).Decode(&response)
				require.NoError(t, err)
				assert.Contains(t, response["error"], "required")
			},
		},
		{
			name: "invalid role",
			requestBody: map[string]string{
				"username": "newuser",
				"email":    "newuser2@example.com",
				"password": "Password123!",
				"role":     "invalid",
			},
			expectedStatus: http.StatusBadRequest,
			validateFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response map[string]string
				err := json.NewDecoder(w.Body).Decode(&response)
				require.NoError(t, err)
				assert.Contains(t, response["error"], "invalid role")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create multipart form request
			var body bytes.Buffer
			writer := multipart.NewWriter(&body)

			for key, value := range tt.requestBody {
				writer.WriteField(key, value)
			}
			writer.Close()

			req := httptest.NewRequest(http.MethodPost, "/api/v6/admin/users", &body)
			req.Header.Set("Content-Type", writer.FormDataContentType())
			w := httptest.NewRecorder()

			testRouter.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.validateFunc != nil {
				tt.validateFunc(t, w)
			}
		})
	}
}

func TestIntegration_AdminCreateUserSettings(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	cleanupTestData(t, testDB)
	seedTestData(t, testDB)
	defer cleanupTestData(t, testDB)

	// Ensure user 2 exists but has no settings
	_, err := testDB.Exec("DELETE FROM user_settings WHERE user_id = 2")
	require.NoError(t, err)

	// Create settings for user 1 to test "already exist" case
	_, err = testDB.Exec("INSERT INTO user_settings (user_id, new_word_count, old_word_count, alphabet_learn_count, language) VALUES (1, 20, 20, 10, 'en')")
	require.NoError(t, err, "Failed to create settings for user 1")

	tests := []struct {
		name           string
		userID         string
		expectedStatus int
		validateFunc   func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:           "success create user settings",
			userID:         "2",
			expectedStatus: http.StatusCreated,
			validateFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response map[string]string
				err := json.NewDecoder(w.Body).Decode(&response)
				require.NoError(t, err)
				assert.Contains(t, response["message"], "Settings created successfully")

				// Verify settings were created in database
				var count int
				err = testDB.QueryRow("SELECT COUNT(*) FROM user_settings WHERE user_id = ?", 2).Scan(&count)
				require.NoError(t, err)
				assert.Equal(t, 1, count)
			},
		},
		{
			name:           "settings already exist",
			userID:         "1",
			expectedStatus: http.StatusOK,
			validateFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response map[string]string
				err := json.NewDecoder(w.Body).Decode(&response)
				require.NoError(t, err)
				assert.Contains(t, response["message"], "Settings already exist")
			},
		},
		{
			name:           "invalid user ID",
			userID:         "invalid",
			expectedStatus: http.StatusBadRequest,
			validateFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response map[string]string
				err := json.NewDecoder(w.Body).Decode(&response)
				require.NoError(t, err)
				assert.Contains(t, response["error"], "invalid user ID")
			},
		},
		{
			name:           "user not found",
			userID:         "999",
			expectedStatus: http.StatusInternalServerError,
			validateFunc:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/v6/admin/users/"+tt.userID+"/settings", nil)
			w := httptest.NewRecorder()

			testRouter.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.validateFunc != nil {
				tt.validateFunc(t, w)
			}
		})
	}
}

func TestIntegration_AdminUpdateUserWithSettings(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	cleanupTestData(t, testDB)
	seedTestData(t, testDB)
	defer cleanupTestData(t, testDB)

	// Create user settings for test user
	_, err := testDB.Exec("INSERT INTO user_settings (user_id, new_word_count, old_word_count, alphabet_learn_count, language, alphabet_repeat) VALUES (1, 20, 20, 10, 'en', 'in question')")
	require.NoError(t, err, "Failed to seed user settings")

	tests := []struct {
		name           string
		userID         string
		requestBody    map[string]string
		expectedStatus int
		validateFunc   func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:   "success update user only",
			userID: "1",
			requestBody: map[string]string{
				"username": "updateduser",
			},
			expectedStatus: http.StatusNoContent,
			validateFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				// Verify database was updated
				var username string
				err := testDB.QueryRow("SELECT username FROM users WHERE id = ?", 1).Scan(&username)
				require.NoError(t, err)
				assert.Equal(t, "updateduser", username)
			},
		},
		{
			name:   "success update settings only",
			userID: "1",
			requestBody: map[string]string{
				"newWordCount": "25",
			},
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
			name:   "success update user and settings",
			userID: "1",
			requestBody: map[string]string{
				"email":        "updated@example.com",
				"newWordCount": "30",
				"oldWordCount": "35",
				"language":     "ru",
			},
			expectedStatus: http.StatusNoContent,
			validateFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				// Verify user was updated
				var email string
				err := testDB.QueryRow("SELECT email FROM users WHERE id = ?", 1).Scan(&email)
				require.NoError(t, err)
				assert.Equal(t, "updated@example.com", email)

				// Verify settings were updated
				var settings models.UserSettings
				err = testDB.QueryRow("SELECT new_word_count, old_word_count, language FROM user_settings WHERE user_id = ?", 1).
					Scan(&settings.NewWordCount, &settings.OldWordCount, &settings.Language)
				require.NoError(t, err)
				assert.Equal(t, 30, settings.NewWordCount)
				assert.Equal(t, 35, settings.OldWordCount)
				assert.Equal(t, models.LanguageRussian, settings.Language)
			},
		},
		{
			name:           "invalid user ID",
			userID:         "invalid",
			requestBody:    map[string]string{"username": "test"},
			expectedStatus: http.StatusBadRequest,
			validateFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response map[string]string
				err := json.NewDecoder(w.Body).Decode(&response)
				require.NoError(t, err)
				assert.Contains(t, response["error"], "invalid user ID")
			},
		},
		{
			name:           "user not found",
			userID:         "999",
			requestBody:    map[string]string{"username": "test"},
			expectedStatus: http.StatusNotFound,
			validateFunc:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create multipart form request
			var body bytes.Buffer
			writer := multipart.NewWriter(&body)

			for key, value := range tt.requestBody {
				writer.WriteField(key, value)
			}
			writer.Close()

			req := httptest.NewRequest(http.MethodPatch, "/api/v6/admin/users/"+tt.userID, &body)
			req.Header.Set("Content-Type", writer.FormDataContentType())
			w := httptest.NewRecorder()

			testRouter.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.validateFunc != nil {
				tt.validateFunc(t, w)
			}
		})
	}
}

func TestIntegration_AdminDeleteUser(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	cleanupTestData(t, testDB)
	seedTestData(t, testDB)
	defer cleanupTestData(t, testDB)

	// Create a separate user for deletion
	var newUserID int
	passwordHash, err := bcrypt.GenerateFromPassword([]byte("Password123!"), bcrypt.DefaultCost)
	require.NoError(t, err)
	err = testDB.QueryRow("INSERT INTO users (username, email, password_hash, role, avatar) VALUES (?, ?, ?, ?, ?) RETURNING id",
		"deleteme", "deleteme@example.com", string(passwordHash), models.RoleUser, "").Scan(&newUserID)
	if err != nil {
		// MySQL doesn't support RETURNING, use LastInsertId approach
		result, err := testDB.Exec("INSERT INTO users (username, email, password_hash, role, avatar) VALUES (?, ?, ?, ?, ?)",
			"deleteme", "deleteme@example.com", string(passwordHash), models.RoleUser, "")
		require.NoError(t, err)
		insertID, err := result.LastInsertId()
		require.NoError(t, err)
		newUserID = int(insertID)
	}

	tests := []struct {
		name           string
		userID         string
		expectedStatus int
		validateFunc   func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:           "success delete user",
			userID:         fmt.Sprintf("%d", newUserID),
			expectedStatus: http.StatusNoContent,
			validateFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				// Verify user was deleted from database
				var count int
				err := testDB.QueryRow("SELECT COUNT(*) FROM users WHERE id = ?", newUserID).Scan(&count)
				require.NoError(t, err)
				assert.Equal(t, 0, count)

				// Verify cascade delete worked (if settings existed)
				var settingsCount int
				err = testDB.QueryRow("SELECT COUNT(*) FROM user_settings WHERE user_id = ?", newUserID).Scan(&settingsCount)
				require.NoError(t, err)
				assert.Equal(t, 0, settingsCount)
			},
		},
		{
			name:           "invalid user ID",
			userID:         "invalid",
			expectedStatus: http.StatusBadRequest,
			validateFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response map[string]string
				err := json.NewDecoder(w.Body).Decode(&response)
				require.NoError(t, err)
				assert.Contains(t, response["error"], "invalid user ID")
			},
		},
		{
			name:           "user not found",
			userID:         "999",
			expectedStatus: http.StatusNotFound,
			validateFunc:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodDelete, "/api/v6/admin/users/"+tt.userID, nil)
			w := httptest.NewRecorder()

			testRouter.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.validateFunc != nil {
				tt.validateFunc(t, w)
			}
		})
	}
}

func TestIntegration_AdminUpdateUserPassword(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	cleanupTestData(t, testDB)
	seedTestData(t, testDB)
	defer cleanupTestData(t, testDB)

	// Create a separate user for password update
	var newUserID int
	passwordHash, err := bcrypt.GenerateFromPassword([]byte("OldPassword123!"), bcrypt.DefaultCost)
	require.NoError(t, err)
	err = testDB.QueryRow("INSERT INTO users (username, email, password_hash, role, avatar) VALUES (?, ?, ?, ?, ?) RETURNING id",
		"passworduser", "passworduser@example.com", string(passwordHash), models.RoleUser, "").Scan(&newUserID)
	if err != nil {
		// MySQL doesn't support RETURNING, use LastInsertId approach
		result, err := testDB.Exec("INSERT INTO users (username, email, password_hash, role, avatar) VALUES (?, ?, ?, ?, ?)",
			"passworduser", "passworduser@example.com", string(passwordHash), models.RoleUser, "")
		require.NoError(t, err)
		insertID, err := result.LastInsertId()
		require.NoError(t, err)
		newUserID = int(insertID)
	}

	tests := []struct {
		name           string
		userID         string
		requestBody    map[string]string
		expectedStatus int
		validateFunc   func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:   "success update password",
			userID: fmt.Sprintf("%d", newUserID),
			requestBody: map[string]string{
				"password": "NewPassword123!",
			},
			expectedStatus: http.StatusNoContent,
			validateFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				// Verify password was updated in database
				var storedHash string
				err := testDB.QueryRow("SELECT password_hash FROM users WHERE id = ?", newUserID).Scan(&storedHash)
				require.NoError(t, err)

				// Verify new password works
				err = bcrypt.CompareHashAndPassword([]byte(storedHash), []byte("NewPassword123!"))
				assert.NoError(t, err, "new password should match stored hash")

				// Verify old password doesn't work
				err = bcrypt.CompareHashAndPassword([]byte(storedHash), []byte("OldPassword123!"))
				assert.Error(t, err, "old password should not match stored hash")
			},
		},
		{
			name:   "success update password for existing user",
			userID: "1",
			requestBody: map[string]string{
				"password": "UpdatedPassword123!",
			},
			expectedStatus: http.StatusNoContent,
			validateFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				// Verify password was updated
				var storedHash string
				err := testDB.QueryRow("SELECT password_hash FROM users WHERE id = ?", 1).Scan(&storedHash)
				require.NoError(t, err)

				err = bcrypt.CompareHashAndPassword([]byte(storedHash), []byte("UpdatedPassword123!"))
				assert.NoError(t, err, "updated password should match stored hash")
			},
		},
		{
			name:   "invalid user ID",
			userID: "invalid",
			requestBody: map[string]string{
				"password": "NewPassword123!",
			},
			expectedStatus: http.StatusBadRequest,
			validateFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response map[string]string
				err := json.NewDecoder(w.Body).Decode(&response)
				require.NoError(t, err)
				assert.Contains(t, response["error"], "invalid user ID")
			},
		},
		{
			name:   "user not found",
			userID: "999",
			requestBody: map[string]string{
				"password": "NewPassword123!",
			},
			expectedStatus: http.StatusNotFound,
			validateFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response map[string]string
				err := json.NewDecoder(w.Body).Decode(&response)
				require.NoError(t, err)
				assert.Contains(t, response["error"], "not found")
			},
		},
		{
			name:   "invalid request body",
			userID: fmt.Sprintf("%d", newUserID),
			requestBody: map[string]string{
				"invalid": "field",
			},
			expectedStatus: http.StatusBadRequest,
			validateFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response map[string]string
				err := json.NewDecoder(w.Body).Decode(&response)
				require.NoError(t, err)
				// When request body has invalid field, JSON decodes successfully but password is empty,
				// so we get a password validation error, not "invalid request body"
				assert.Contains(t, response["error"], "password")
			},
		},
		{
			name:   "password too short",
			userID: fmt.Sprintf("%d", newUserID),
			requestBody: map[string]string{
				"password": "Short1!",
			},
			expectedStatus: http.StatusBadRequest,
			validateFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response map[string]string
				err := json.NewDecoder(w.Body).Decode(&response)
				require.NoError(t, err)
				assert.Contains(t, response["error"], "password")
			},
		},
		{
			name:   "password missing uppercase",
			userID: fmt.Sprintf("%d", newUserID),
			requestBody: map[string]string{
				"password": "password123!",
			},
			expectedStatus: http.StatusBadRequest,
			validateFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response map[string]string
				err := json.NewDecoder(w.Body).Decode(&response)
				require.NoError(t, err)
				assert.Contains(t, response["error"], "password")
			},
		},
		{
			name:   "password missing special character",
			userID: fmt.Sprintf("%d", newUserID),
			requestBody: map[string]string{
				"password": "Password123",
			},
			expectedStatus: http.StatusBadRequest,
			validateFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response map[string]string
				err := json.NewDecoder(w.Body).Decode(&response)
				require.NoError(t, err)
				assert.Contains(t, response["error"], "password")
			},
		},
		{
			name:   "password contains semicolon",
			userID: fmt.Sprintf("%d", newUserID),
			requestBody: map[string]string{
				"password": "Password123!;",
			},
			expectedStatus: http.StatusBadRequest,
			validateFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response map[string]string
				err := json.NewDecoder(w.Body).Decode(&response)
				require.NoError(t, err)
				assert.Contains(t, response["error"], "password cannot contain ';' character")
			},
		},
		{
			name:           "empty request body",
			userID:         fmt.Sprintf("%d", newUserID),
			requestBody:    nil,
			expectedStatus: http.StatusBadRequest,
			validateFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response map[string]string
				err := json.NewDecoder(w.Body).Decode(&response)
				require.NoError(t, err)
				assert.Contains(t, response["error"], "invalid request body")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req *http.Request
			if tt.requestBody != nil {
				body, _ := json.Marshal(tt.requestBody)
				req = httptest.NewRequest(http.MethodPatch, "/api/v6/admin/users/"+tt.userID+"/password", bytes.NewBuffer(body))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req = httptest.NewRequest(http.MethodPatch, "/api/v6/admin/users/"+tt.userID+"/password", nil)
				req.Header.Set("Content-Type", "application/json")
			}
			w := httptest.NewRecorder()

			testRouter.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.validateFunc != nil {
				tt.validateFunc(t, w)
			}
		})
	}
}

// NOTE: Task Scheduling Testing
//
// The ScheduleTasks method in admin_service.go schedules tasks via HTTP requests to the task-service.
// This functionality requires the task-service to be running and properly configured.
// Task scheduling should be tested in a live environment with the task-service running, as it involves:
// - HTTP communication with the task microservice
// - Task creation and scheduling through the task service
// - End-to-end task execution flow
//
// In these tests, we focus ONLY on the token cleaning process itself, not the task scheduling.

func TestIntegration_TokenCleaning(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	cleanupTestData(t, testDB)
	seedTestData(t, testDB)
	defer cleanupTestData(t, testDB)

	// Get refresh token expiry from config
	cfg, err := config.LoadTestConfig()
	require.NoError(t, err)
	refreshExpiry := cfg.JWT.RefreshTokenExpiry
	if refreshExpiry == 0 {
		refreshExpiry = 7 * 24 * time.Hour
	}

	// Create expired tokens (created more than refreshExpiry ago)
	expiredTime := time.Now().Add(-refreshExpiry).Add(-1 * time.Hour) // 1 hour older than expiry
	_, err = testDB.Exec("INSERT INTO user_tokens (user_id, token, created_at) VALUES (?, ?, ?)",
		1, "expired-token-1", expiredTime)
	require.NoError(t, err)
	_, err = testDB.Exec("INSERT INTO user_tokens (user_id, token, created_at) VALUES (?, ?, ?)",
		1, "expired-token-2", expiredTime)
	require.NoError(t, err)

	// Create valid tokens (created recently)
	_, err = testDB.Exec("INSERT INTO user_tokens (user_id, token, created_at) VALUES (?, ?, NOW())",
		1, "valid-token-1")
	require.NoError(t, err)
	_, err = testDB.Exec("INSERT INTO user_tokens (user_id, token, created_at) VALUES (?, ?, NOW())",
		1, "valid-token-2")
	require.NoError(t, err)

	tests := []struct {
		name           string
		expectedStatus int
		validateFunc   func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:           "success clean expired tokens",
			expectedStatus: http.StatusOK,
			validateFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response map[string]string
				err := json.NewDecoder(w.Body).Decode(&response)
				require.NoError(t, err)
				assert.Contains(t, response["message"], "token cleaning completed successfully")

				// Verify expired tokens were deleted
				var expiredCount int
				err = testDB.QueryRow("SELECT COUNT(*) FROM user_tokens WHERE token IN (?, ?)",
					"expired-token-1", "expired-token-2").Scan(&expiredCount)
				require.NoError(t, err)
				assert.Equal(t, 0, expiredCount, "expired tokens should be deleted")

				// Verify valid tokens still exist
				var validCount int
				err = testDB.QueryRow("SELECT COUNT(*) FROM user_tokens WHERE token IN (?, ?)",
					"valid-token-1", "valid-token-2").Scan(&validCount)
				require.NoError(t, err)
				assert.Equal(t, 2, validCount, "valid tokens should not be deleted")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v6/tokens/clean", nil)
			w := httptest.NewRecorder()

			testRouter.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.validateFunc != nil {
				tt.validateFunc(t, w)
			}
		})
	}
}

func TestIntegration_TokenCleaning_NoExpiredTokens(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	cleanupTestData(t, testDB)
	seedTestData(t, testDB)
	defer cleanupTestData(t, testDB)

	// Create only valid tokens (all recent)
	_, err := testDB.Exec("INSERT INTO user_tokens (user_id, token, created_at) VALUES (?, ?, NOW())",
		1, "valid-token-1")
	require.NoError(t, err)
	_, err = testDB.Exec("INSERT INTO user_tokens (user_id, token, created_at) VALUES (?, ?, NOW())",
		1, "valid-token-2")
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/api/v6/tokens/clean", nil)
	w := httptest.NewRecorder()

	testRouter.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]string
	err = json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)
	assert.Contains(t, response["message"], "token cleaning completed successfully")

	// Verify all tokens still exist (none were expired)
	var count int
	err = testDB.QueryRow("SELECT COUNT(*) FROM user_tokens WHERE token IN (?, ?)",
		"valid-token-1", "valid-token-2").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 2, count, "all tokens should still exist")
}

func TestIntegration_TokenCleaning_RepositoryLayer(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	cleanupTestData(t, testDB)
	seedTestData(t, testDB)
	defer cleanupTestData(t, testDB)

	tokenRepo := repositories.NewUserTokenRepository(testDB)
	ctx := context.Background()

	// Create tokens with specific creation times
	expiredTime := time.Now().Add(-8 * 24 * time.Hour) // 8 days ago (older than 7 day expiry)
	validTime := time.Now().Add(-1 * time.Hour)        // 1 hour ago (recent)

	// Insert expired tokens
	_, err := testDB.Exec("INSERT INTO user_tokens (user_id, token, created_at) VALUES (?, ?, ?)",
		1, "expired-1", expiredTime)
	require.NoError(t, err)
	_, err = testDB.Exec("INSERT INTO user_tokens (user_id, token, created_at) VALUES (?, ?, ?)",
		1, "expired-2", expiredTime)
	require.NoError(t, err)

	// Insert valid tokens
	_, err = testDB.Exec("INSERT INTO user_tokens (user_id, token, created_at) VALUES (?, ?, ?)",
		1, "valid-1", validTime)
	require.NoError(t, err)

	// Calculate expiry time (7 days ago)
	cfg, err := config.LoadTestConfig()
	require.NoError(t, err)
	refreshExpiry := cfg.JWT.RefreshTokenExpiry
	if refreshExpiry == 0 {
		refreshExpiry = 7 * 24 * time.Hour
	}
	expiryTime := time.Now().Add(-refreshExpiry)

	// Test DeleteExpiredTokens
	deletedCount, err := tokenRepo.DeleteExpiredTokens(ctx, expiryTime)
	require.NoError(t, err)
	assert.Equal(t, 2, deletedCount, "should delete 2 expired tokens")

	// Verify expired tokens were deleted
	var expiredCount int
	err = testDB.QueryRow("SELECT COUNT(*) FROM user_tokens WHERE token IN (?, ?)",
		"expired-1", "expired-2").Scan(&expiredCount)
	require.NoError(t, err)
	assert.Equal(t, 0, expiredCount, "expired tokens should be deleted")

	// Verify valid token still exists
	var validCount int
	err = testDB.QueryRow("SELECT COUNT(*) FROM user_tokens WHERE token = ?", "valid-1").Scan(&validCount)
	require.NoError(t, err)
	assert.Equal(t, 1, validCount, "valid token should still exist")
}

func TestIntegration_ProfileHandler(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	cleanupTestData(t, testDB)
	seedTestData(t, testDB)
	defer cleanupTestData(t, testDB)

	// Create user settings for test user
	_, err := testDB.Exec("INSERT INTO user_settings (user_id, new_word_count, old_word_count, alphabet_learn_count, language, alphabet_repeat) VALUES (1, 20, 20, 10, 'en', 'in question')")
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
			name:           "success get user profile",
			userID:         1,
			method:         http.MethodGet,
			url:            "/api/v6/profile",
			requestBody:    nil,
			expectedStatus: http.StatusOK,
			validateFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response models.ProfileResponse
				err := json.NewDecoder(w.Body).Decode(&response)
				require.NoError(t, err)
				assert.Equal(t, "testuser", response.Username)
				assert.Equal(t, "test@example.com", response.Email)
			},
		},
		{
			name:           "unauthorized get user profile",
			userID:         0, // No user ID in context
			method:         http.MethodGet,
			url:            "/api/v6/profile",
			requestBody:    nil,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:   "success update user profile - username only",
			userID: 1,
			method: http.MethodPatch,
			url:    "/api/v6/profile",
			requestBody: map[string]any{
				"username": "newusername",
			},
			expectedStatus: http.StatusNoContent,
		},
		{
			name:   "success update user profile - email only",
			userID: 1,
			method: http.MethodPatch,
			url:    "/api/v6/profile",
			requestBody: map[string]any{
				"email": "newemail@example.com",
			},
			expectedStatus: http.StatusInternalServerError, // Email update fails because TASK_BASE_URL is not configured in tests
			validateFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				// Verify email was updated in database despite email sending failure
				var email string
				err := testDB.QueryRow("SELECT email FROM users WHERE id = ?", 1).Scan(&email)
				require.NoError(t, err)
				assert.Equal(t, "newemail@example.com", email, "email should be updated in database")

				// Verify user is inactive (email not verified)
				var active bool
				err = testDB.QueryRow("SELECT active FROM users WHERE id = ?", 1).Scan(&active)
				require.NoError(t, err)
				assert.False(t, active, "user should be inactive until email is verified")

				// Manually reactivate user for testing purposes (in real scenario, this happens via email verification)
				_, err = testDB.Exec("UPDATE users SET active = true WHERE id = ?", 1)
				require.NoError(t, err, "failed to reactivate user for testing")
			},
		},
		{
			name:   "success update user profile - both fields",
			userID: 1,
			method: http.MethodPatch,
			url:    "/api/v6/profile",
			requestBody: map[string]any{
				"username": "updateduser",
				"email":    "updated@example.com",
			},
			expectedStatus: http.StatusInternalServerError, // Email update fails because TASK_BASE_URL is not configured in tests
			validateFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				// Verify both username and email were updated in database despite email sending failure
				var username, email string
				err := testDB.QueryRow("SELECT username, email FROM users WHERE id = ?", 1).Scan(&username, &email)
				require.NoError(t, err)
				assert.Equal(t, "updateduser", username, "username should be updated in database")
				assert.Equal(t, "updated@example.com", email, "email should be updated in database")

				// Verify user is inactive (email not verified)
				var active bool
				err = testDB.QueryRow("SELECT active FROM users WHERE id = ?", 1).Scan(&active)
				require.NoError(t, err)
				assert.False(t, active, "user should be inactive until email is verified")

				// Manually reactivate user for testing purposes (in real scenario, this happens via email verification)
				_, err = testDB.Exec("UPDATE users SET active = true WHERE id = ?", 1)
				require.NoError(t, err, "failed to reactivate user for testing")
			},
		},
		{
			name:           "validation error - no fields provided",
			userID:         1,
			method:         http.MethodPatch,
			url:            "/api/v6/profile",
			requestBody:    map[string]any{},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "validation error - invalid email format",
			userID: 1,
			method: http.MethodPatch,
			url:    "/api/v6/profile",
			requestBody: map[string]any{
				"email": "invalid-email",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "unauthorized update user profile",
			userID:         0, // No user ID in context
			method:         http.MethodPatch,
			url:            "/api/v6/profile",
			requestBody:    map[string]any{"username": "newuser"},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:   "success update password",
			userID: 1,
			method: http.MethodPut,
			url:    "/api/v6/profile/password",
			requestBody: map[string]any{
				"password": "NewPassword123!",
			},
			expectedStatus: http.StatusNoContent,
		},
		{
			name:   "validation error - empty password",
			userID: 1,
			method: http.MethodPut,
			url:    "/api/v6/profile/password",
			requestBody: map[string]any{
				"password": "",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "validation error - password too short",
			userID: 1,
			method: http.MethodPut,
			url:    "/api/v6/profile/password",
			requestBody: map[string]any{
				"password": "Pass1!",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "validation error - password missing uppercase",
			userID: 1,
			method: http.MethodPut,
			url:    "/api/v6/profile/password",
			requestBody: map[string]any{
				"password": "password123!",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "unauthorized update password",
			userID:         0, // No user ID in context
			method:         http.MethodPut,
			url:            "/api/v6/profile/password",
			requestBody:    map[string]any{"password": "NewPassword123!"},
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req *http.Request
			if tt.requestBody != nil {
				body, err := json.Marshal(tt.requestBody)
				require.NoError(t, err)
				req = httptest.NewRequest(tt.method, tt.url, bytes.NewBuffer(body))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req = httptest.NewRequest(tt.method, tt.url, nil)
			}

			// Set user ID in context if provided
			if tt.userID > 0 {
				ctx := middleware.SetUserID(req.Context(), tt.userID)
				req = req.WithContext(ctx)
			}

			w := httptest.NewRecorder()
			testRouter.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code, "Response status code should match")

			if tt.validateFunc != nil {
				tt.validateFunc(t, w)
			}
		})
	}

	// NOTE: Avatar update integration tests are not included here.
	// Avatar update uses an external media service and should be tested on a live server.
	// The media service integration involves:
	// - HTTP communication with the media microservice
	// - File upload and deletion through the media service
	// - End-to-end avatar update flow
}

// NOTE: Forgot password integration tests are not included here.
// Forgot password uses an external task service and should be tested on a live server.
// The forgot password functionality involves:
// - HTTP communication with the task microservice
// - Email delivery through the task service
// - Password generation and database update
// - End-to-end password reset flow

// NOTE: UpdateRepeatFlag Integration Testing
//
// The UpdateRepeatFlag method in profile_service.go calls task-service endpoints
// (CreateScheduledTask and DeleteScheduledTaskByUserID) when the flag is set to "repeat"
// or changed from "repeat" to another value. This functionality requires the task-service
// to be running and properly configured.
// Task-service integration should be tested in a live environment with the task-service running, as it involves:
// - HTTP communication with the task microservice
// - Scheduled task creation and deletion through the task service
// - End-to-end task scheduling flow
//
// In these tests, we focus ONLY on the flag update logic itself, not the task-service integration.
// Tests that would trigger task-service calls will fail with appropriate error messages indicating
// that SCHEDULED_TASK_BASE_URL is not configured.

func TestIntegration_UpdateRepeatFlag(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	cleanupTestData(t, testDB)
	seedTestData(t, testDB)
	defer cleanupTestData(t, testDB)

	// Create user settings for test user
	_, err := testDB.Exec("INSERT INTO user_settings (user_id, new_word_count, old_word_count, alphabet_learn_count, language, alphabet_repeat) VALUES (1, 20, 20, 10, 'en', 'in question')")
	require.NoError(t, err, "Failed to seed user settings")

	tests := []struct {
		name           string
		userID         int
		requestBody    map[string]string
		expectedStatus int
		validateFunc   func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:   "success update to 'ignore'",
			userID: 1,
			requestBody: map[string]string{
				"flag": "ignore",
			},
			expectedStatus: http.StatusNoContent,
			validateFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				// Verify database was updated
				var alphabetRepeat string
				err := testDB.QueryRow("SELECT alphabet_repeat FROM user_settings WHERE user_id = ?", 1).Scan(&alphabetRepeat)
				require.NoError(t, err)
				assert.Equal(t, "ignore", alphabetRepeat)
			},
		},
		{
			name:   "success update to 'in question'",
			userID: 1,
			requestBody: map[string]string{
				"flag": "in question",
			},
			expectedStatus: http.StatusNoContent,
			validateFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				// Verify database was updated
				var alphabetRepeat string
				err := testDB.QueryRow("SELECT alphabet_repeat FROM user_settings WHERE user_id = ?", 1).Scan(&alphabetRepeat)
				require.NoError(t, err)
				assert.Equal(t, "in question", alphabetRepeat)
			},
		},
		{
			name:   "update to 'repeat' - will fail due to empty scheduledTaskBaseURL",
			userID: 1,
			requestBody: map[string]string{
				"flag": "repeat",
			},
			expectedStatus: http.StatusInternalServerError,
			validateFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response map[string]string
				err := json.NewDecoder(w.Body).Decode(&response)
				require.NoError(t, err)
				assert.Contains(t, response["error"], "SCHEDULED_TASK_BASE_URL is not configured")

				// Verify flag was still updated in database despite task-service failure
				var alphabetRepeat string
				err = testDB.QueryRow("SELECT alphabet_repeat FROM user_settings WHERE user_id = ?", 1).Scan(&alphabetRepeat)
				require.NoError(t, err)
				assert.Equal(t, "repeat", alphabetRepeat, "flag should be updated in database even if task-service call fails")
			},
		},
		{
			name:   "update from 'repeat' to 'ignore' - will fail due to empty scheduledTaskBaseURL",
			userID: 1,
			requestBody: map[string]string{
				"flag": "ignore",
			},
			expectedStatus: http.StatusInternalServerError,
			validateFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response map[string]string
				err := json.NewDecoder(w.Body).Decode(&response)
				require.NoError(t, err)
				assert.Contains(t, response["error"], "SCHEDULED_TASK_BASE_URL is not configured")

				// Verify flag was still updated in database despite task-service failure
				var alphabetRepeat string
				err = testDB.QueryRow("SELECT alphabet_repeat FROM user_settings WHERE user_id = ?", 1).Scan(&alphabetRepeat)
				require.NoError(t, err)
				assert.Equal(t, "ignore", alphabetRepeat, "flag should be updated in database even if task-service call fails")
			},
		},
		{
			name:   "invalid flag value",
			userID: 1,
			requestBody: map[string]string{
				"flag": "invalid",
			},
			expectedStatus: http.StatusBadRequest,
			validateFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response map[string]string
				err := json.NewDecoder(w.Body).Decode(&response)
				require.NoError(t, err)
				assert.Contains(t, response["error"], "flag must be 'in question', 'ignore', or 'repeat'")
			},
		},
		{
			name:           "empty request body",
			userID:         1,
			requestBody:    nil,
			expectedStatus: http.StatusBadRequest,
			validateFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response map[string]string
				err := json.NewDecoder(w.Body).Decode(&response)
				require.NoError(t, err)
				assert.Contains(t, response["error"], "invalid request body")
			},
		},
		{
			name:           "unauthorized - no user ID",
			userID:         0,
			requestBody:    map[string]string{"flag": "ignore"},
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// For the "update from 'repeat' to 'ignore'" test, set flag to 'repeat' first
			if tt.name == "update from 'repeat' to 'ignore' - will fail due to empty scheduledTaskBaseURL" {
				_, err := testDB.Exec("UPDATE user_settings SET alphabet_repeat = 'repeat' WHERE user_id = ?", 1)
				require.NoError(t, err)
			}

			var req *http.Request
			if tt.requestBody != nil {
				body, err := json.Marshal(tt.requestBody)
				require.NoError(t, err)
				req = httptest.NewRequest(http.MethodPut, "/api/v6/profile/repeat-flag", bytes.NewBuffer(body))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req = httptest.NewRequest(http.MethodPut, "/api/v6/profile/repeat-flag", nil)
				req.Header.Set("Content-Type", "application/json")
			}

			// Set user ID in context if provided
			if tt.userID > 0 {
				ctx := middleware.SetUserID(req.Context(), tt.userID)
				req = req.WithContext(ctx)
			}

			w := httptest.NewRecorder()
			testRouter.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code, "Response status code should match")

			if tt.validateFunc != nil {
				tt.validateFunc(t, w)
			}
		})
	}
}
