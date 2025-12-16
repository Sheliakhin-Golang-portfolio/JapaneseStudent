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
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	_ "github.com/go-sql-driver/mysql"
	"github.com/japanesestudent/auth-service/internal/handlers"
	"github.com/japanesestudent/auth-service/internal/models"
	"github.com/japanesestudent/auth-service/internal/repositories"
	"github.com/japanesestudent/auth-service/internal/services"
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
	_, err := db.Exec("DELETE FROM user_tokens")
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
	_, err := db.Exec("DELETE FROM user_tokens")
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
	userRepo := repositories.NewUserRepository(db, logger)
	tokenRepo := repositories.NewUserTokenRepository(db, logger)
	tokenGen := service.NewTokenGenerator("test-secret-key-for-integration-tests", 1*time.Hour, 7*24*time.Hour)
	authSvc := services.NewAuthService(userRepo, tokenRepo, tokenGen, logger)
	authHandler := handlers.NewAuthHandler(authSvc, logger)

	r := chi.NewRouter()
	// Scope router to /api/v1 to match main.go setup
	r.Route("/api/v1", func(r chi.Router) {
		authHandler.RegisterRoutes(r)
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

	db.Exec(usersTable)
	db.Exec(userTokensTable)
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
			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewBuffer(body))
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
				assert.Contains(t, response["error"], "invalid credentials")
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
			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(body))
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
	loginReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBuffer(loginBody))
	loginReq.Header.Set("Content-Type", "application/json")
	loginW := httptest.NewRecorder()
	testRouter.ServeHTTP(loginW, loginReq)

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
			req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewBuffer(body))
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

	logger, _ := zap.NewDevelopment()
	userRepo := repositories.NewUserRepository(testDB, logger)
	tokenRepo := repositories.NewUserTokenRepository(testDB, logger)
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
	userRepo := repositories.NewUserRepository(testDB, logger)
	tokenRepo := repositories.NewUserTokenRepository(testDB, logger)
	tokenGen := service.NewTokenGenerator("test-secret", 1*time.Hour, 7*24*time.Hour)
	authSvc := services.NewAuthService(userRepo, tokenRepo, tokenGen, logger)
	ctx := context.Background()

	t.Run("Register", func(t *testing.T) {
		accessToken, refreshToken, err := authSvc.Register(ctx, "servicetest@example.com", "servicetest", "Password123!")
		require.NoError(t, err)
		assert.NotEmpty(t, accessToken)
		assert.NotEmpty(t, refreshToken)
	})

	t.Run("Login", func(t *testing.T) {
		accessToken, refreshToken, err := authSvc.Login(ctx, "test@example.com", "Password123!")
		require.NoError(t, err)
		assert.NotEmpty(t, accessToken)
		assert.NotEmpty(t, refreshToken)
	})

	t.Run("Refresh", func(t *testing.T) {
		// First login to get a refresh token
		_, refreshToken, err := authSvc.Login(ctx, "test@example.com", "Password123!")
		require.NoError(t, err)

		// Refresh the token
		newAccessToken, newRefreshToken, err := authSvc.Refresh(ctx, refreshToken)
		require.NoError(t, err)
		assert.NotEmpty(t, newAccessToken)
		assert.NotEmpty(t, newRefreshToken)
		assert.NotEqual(t, refreshToken, newRefreshToken)
	})
}
