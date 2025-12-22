// Package config provides configuration for the application
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all configuration for the application
type Config struct {
	Database      DatabaseConfig
	Server        ServerConfig
	Logging       LoggingConfig
	CORS          CORSConfig
	JWT           JWTConfig
	APIKey        string
	MediaBasePath string
	MediaBaseURL  string
}

// DatabaseConfig holds database connection settings
type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
}

// ServerConfig holds server settings
type ServerConfig struct {
	Port int
}

// LoggingConfig holds logging settings
type LoggingConfig struct {
	Level string
}

// CORSConfig holds CORS settings
type CORSConfig struct {
	AllowedOrigins []string
}

// JWTConfig holds JWT token configuration
type JWTConfig struct {
	Secret             string
	AccessTokenExpiry  time.Duration
	RefreshTokenExpiry time.Duration
}

// Load reads configuration from environment variables
func Load() (*Config, error) {
	// Try to load .env file (optional)
	godotenv.Load()

	cfg := &Config{}

	// Database configuration
	dbHost := os.Getenv("DB_HOST")
	if dbHost == "" {
		return nil, fmt.Errorf("DB_HOST is required")
	}
	cfg.Database.Host = dbHost

	dbPortStr := os.Getenv("DB_PORT")
	if dbPortStr == "" {
		return nil, fmt.Errorf("DB_PORT is required")
	}
	dbPort, err := strconv.Atoi(dbPortStr)
	if err != nil {
		return nil, fmt.Errorf("invalid DB_PORT: %w", err)
	}
	cfg.Database.Port = dbPort

	dbUser := os.Getenv("DB_USER")
	if dbUser == "" {
		return nil, fmt.Errorf("DB_USER is required")
	}
	cfg.Database.User = dbUser

	dbPassword := os.Getenv("DB_PASSWORD")
	if dbPassword == "" {
		return nil, fmt.Errorf("DB_PASSWORD is required")
	}
	cfg.Database.Password = dbPassword

	dbName := os.Getenv("DB_NAME")
	if dbName == "" {
		return nil, fmt.Errorf("DB_NAME is required")
	}
	cfg.Database.DBName = dbName

	// Server configuration
	serverPortStr := os.Getenv("SERVER_PORT")
	if serverPortStr == "" {
		serverPortStr = "8080" // default port
	}
	serverPort, err := strconv.Atoi(serverPortStr)
	if err != nil {
		return nil, fmt.Errorf("invalid SERVER_PORT: %w", err)
	}
	cfg.Server.Port = serverPort

	// Logging configuration
	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info" // default level
	}
	cfg.Logging.Level = logLevel

	// CORS configuration
	corsOrigins := os.Getenv("CORS_ALLOWED_ORIGINS")
	if corsOrigins == "" {
		// Default to allow all origins if not specified (for development)
		cfg.CORS.AllowedOrigins = []string{"*"}
	} else {
		// Parse comma-separated origins
		origins := strings.Split(corsOrigins, ",")
		cfg.CORS.AllowedOrigins = make([]string, 0, len(origins))
		for _, origin := range origins {
			origin = strings.TrimSpace(origin)
			if origin != "" {
				cfg.CORS.AllowedOrigins = append(cfg.CORS.AllowedOrigins, origin)
			}
		}
		// If no valid origins found, default to allow all
		if len(cfg.CORS.AllowedOrigins) == 0 {
			cfg.CORS.AllowedOrigins = []string{"*"}
		}
	}

	// JWT configuration
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required")
	}
	cfg.JWT.Secret = jwtSecret

	// Access token expiry (default: 1 hour)
	accessExpiryStr := os.Getenv("JWT_ACCESS_TOKEN_EXPIRY")
	if accessExpiryStr == "" {
		accessExpiryStr = "1h"
	}
	accessExpiry, err := time.ParseDuration(accessExpiryStr)
	if err != nil {
		return nil, fmt.Errorf("invalid JWT_ACCESS_TOKEN_EXPIRY: %w", err)
	}
	cfg.JWT.AccessTokenExpiry = accessExpiry

	// Refresh token expiry (default: 7 days)
	refreshExpiryStr := os.Getenv("JWT_REFRESH_TOKEN_EXPIRY")
	if refreshExpiryStr == "" {
		refreshExpiryStr = "168h" // 7 days
	}
	refreshExpiry, err := time.ParseDuration(refreshExpiryStr)
	if err != nil {
		return nil, fmt.Errorf("invalid JWT_REFRESH_TOKEN_EXPIRY: %w", err)
	}
	cfg.JWT.RefreshTokenExpiry = refreshExpiry

	// API Key configuration (optional, for service-to-service authentication)
	cfg.APIKey = os.Getenv("API_KEY")

	// Media base path configuration (optional, for media service)
	cfg.MediaBasePath = os.Getenv("MEDIA_BASE_PATH")

	// Media base URL configuration (optional, for media service)
	cfg.MediaBaseURL = os.Getenv("MEDIA_BASE_URL")

	return cfg, nil
}

// DSN returns the database connection string
func (c *Config) DSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true&charset=utf8mb4",
		c.Database.User,
		c.Database.Password,
		c.Database.Host,
		c.Database.Port,
		c.Database.DBName,
	)
}
