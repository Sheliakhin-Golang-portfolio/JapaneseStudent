package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// LoadTestConfig loads the configuration from the .env file or environment variables for integration tests
// If .env file doesn't exist or environment variables are not set, returns a Config with empty values
// which allows tests to use fallback DSN values
func LoadTestConfig() (*Config, error) {
	// Try to load .env file (ignore error if file doesn't exist - it's optional)
	// Try loading from project root
	_ = godotenv.Load("../../../../.env")
	_ = godotenv.Load()

	cfg := &Config{}
	dbHost := os.Getenv("TEST_DB_HOST")
	if dbHost == "" {
		// Return empty config to allow fallback DSN in tests
		return cfg, nil
	}
	cfg.Database.Host = dbHost

	dbPortStr := os.Getenv("TEST_DB_PORT")
	if dbPortStr == "" {
		// Return empty config to allow fallback DSN in tests
		return cfg, nil
	}
	dbPort, err := strconv.Atoi(dbPortStr)
	if err != nil {
		return nil, fmt.Errorf("invalid TEST_DB_PORT: %w", err)
	}
	cfg.Database.Port = dbPort

	dbUser := os.Getenv("TEST_DB_USER")
	if dbUser == "" {
		// Return empty config to allow fallback DSN in tests
		return cfg, nil
	}
	cfg.Database.User = dbUser

	dbPassword := os.Getenv("TEST_DB_PASSWORD")
	if dbPassword == "" {
		// Return empty config to allow fallback DSN in tests
		return cfg, nil
	}
	cfg.Database.Password = dbPassword

	dbName := os.Getenv("TEST_DB_NAME")
	if dbName == "" {
		// Return empty config to allow fallback DSN in tests
		return cfg, nil
	}
	cfg.Database.DBName = dbName

	// JWT configuration
	jwtSecret := os.Getenv("TEST_JWT_SECRET")
	if jwtSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required")
	}
	cfg.JWT.Secret = jwtSecret

	// Access token expiry (default: 1 hour)
	accessExpiryStr := os.Getenv("TEST_JWT_ACCESS_TOKEN_EXPIRY")
	if accessExpiryStr == "" {
		accessExpiryStr = "1h"
	}
	accessExpiry, err := time.ParseDuration(accessExpiryStr)
	if err != nil {
		return nil, fmt.Errorf("invalid JWT_ACCESS_TOKEN_EXPIRY: %w", err)
	}
	cfg.JWT.AccessTokenExpiry = accessExpiry

	// Refresh token expiry (default: 7 days)
	refreshExpiryStr := os.Getenv("TEST_JWT_REFRESH_TOKEN_EXPIRY")
	if refreshExpiryStr == "" {
		refreshExpiryStr = "168h" // 7 days
	}
	refreshExpiry, err := time.ParseDuration(refreshExpiryStr)
	if err != nil {
		return nil, fmt.Errorf("invalid JWT_REFRESH_TOKEN_EXPIRY: %w", err)
	}
	cfg.JWT.RefreshTokenExpiry = refreshExpiry

	// API Key configuration (optional, for service-to-service authentication)
	cfg.APIKey = os.Getenv("TEST_API_KEY")

	// Media base path configuration (optional, for media service)
	cfg.MediaBasePath = os.Getenv("TEST_MEDIA_BASE_PATH")

	return cfg, nil
}
