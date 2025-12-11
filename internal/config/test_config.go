package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// LoadTestConfig loads the configuration from the .env file or environment variables for integration tests
// If .env file doesn't exist or environment variables are not set, returns a Config with empty values
// which allows tests to use fallback DSN values
func LoadTestConfig() (*Config, error) {
	// Try to load .env file (ignore error if file doesn't exist - it's optional)
	// Try both possible paths
	_ = godotenv.Load("./../../configs/.env")
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

	return cfg, nil
}
