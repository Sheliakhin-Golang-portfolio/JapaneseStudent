package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	authMiddleware "github.com/Sheliakhin-Golang-portfolio/JapaneseStudent/libs/auth/middleware"
	authService "github.com/Sheliakhin-Golang-portfolio/JapaneseStudent/libs/auth/service"
	"github.com/Sheliakhin-Golang-portfolio/JapaneseStudent/libs/config"
	"github.com/Sheliakhin-Golang-portfolio/JapaneseStudent/libs/logger"
	loggerMiddleware "github.com/Sheliakhin-Golang-portfolio/JapaneseStudent/libs/logger/middleware"
	sharedMiddleware "github.com/Sheliakhin-Golang-portfolio/JapaneseStudent/libs/middlewares"
	_ "github.com/Sheliakhin-Golang-portfolio/JapaneseStudent/services/media-service/docs"
	"github.com/Sheliakhin-Golang-portfolio/JapaneseStudent/services/media-service/internal/handlers"
	"github.com/Sheliakhin-Golang-portfolio/JapaneseStudent/services/media-service/internal/repositories"
	"github.com/Sheliakhin-Golang-portfolio/JapaneseStudent/services/media-service/internal/services"
	"github.com/Sheliakhin-Golang-portfolio/JapaneseStudent/services/media-service/internal/storage"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/httprate"
	_ "github.com/go-sql-driver/mysql"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/mysql"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	httpSwagger "github.com/swaggo/http-swagger"
	"go.uber.org/zap"
)

const maxRequestSize = 50 * 1024 * 1024 // 50MB for file uploads

// @title JapaneseStudent Media API
// @version 4.0
// @description API for managing media files
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.email shelyahin.mihail@gmail.com

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:8082
// @BasePath /api/v6
// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name X-API-Key
// @description API key for service-to-service authentication
func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v\n", err)
	}

	// Validate media base path is set
	if cfg.MediaBasePath == "" {
		log.Fatalf("MEDIA_BASE_PATH is required")
	}

	// Initialize logger
	if err := logger.Init(cfg.Logging.Level); err != nil {
		log.Fatalf("Failed to initialize logger: %v\n", err)
	}
	defer logger.Sync()

	logger.Logger.Info("Starting JapaneseStudent Media Service")

	// Connect to database
	db, err := connectDB(cfg.DSN())
	if err != nil {
		logger.Logger.Fatal("Failed to connect to database", zap.Error(err))
		os.Exit(1)
	}
	defer db.Close()

	// Run migrations
	if err := runMigrations(db); err != nil {
		logger.Logger.Fatal("Failed to run migrations", zap.Error(err))
	}

	// Initialize JWT token generator (for auth middleware)
	tokenGenerator := authService.NewTokenGenerator(
		cfg.JWT.Secret,
		cfg.JWT.AccessTokenExpiry,
		cfg.JWT.RefreshTokenExpiry,
	)

	// Initialize storage
	fileStorage := storage.NewLocalStorage(cfg.MediaBasePath)

	// Initialize repositories
	metadataRepo := repositories.NewMetadataRepository(db)

	// Initialize services
	mediaService := services.NewMediaService(metadataRepo, fileStorage)

	// Initialize middleware
	authMw := authMiddleware.AuthMiddleware(tokenGenerator)
	apiKeyMw := authMiddleware.APIKeyMiddleware(cfg.APIKey)

	// Base URL for generating download URLs
	baseURL := os.Getenv("BASE_URL")
	if baseURL == "" {
		baseURL = fmt.Sprintf("http://localhost:%d", cfg.Server.Port)
	}

	// Initialize handlers
	mediaHandler := handlers.NewMediaHandler(mediaService, logger.Logger, baseURL, authMw)

	// Setup router
	r := chi.NewRouter()

	// Apply middleware
	r.Use(sharedMiddleware.RequestIDMiddleware)
	r.Use(loggerMiddleware.LoggerMiddleware(logger.Logger))
	r.Use(sharedMiddleware.RecoveryMiddleware(logger.Logger))
	r.Use(sharedMiddleware.CORSMiddleware(cfg.CORS.AllowedOrigins))
	r.Use(httprate.LimitByIP(100, time.Minute))
	r.Use(sharedMiddleware.RequestSizeLimitMiddleware(maxRequestSize))

	// Swagger documentation
	r.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL(fmt.Sprintf("http://localhost:%d/swagger/doc.json", cfg.Server.Port)),
	))

	// Scope router to /api/v6
	r.Route("/api/v6", func(r chi.Router) {
		// Public metadata endpoint
		r.Get("/media/{id}", mediaHandler.GetMetadata)

		// Download endpoint - auth is handled conditionally in the handler
		r.Get("/media/{mediaType}/{filename}", mediaHandler.DownloadFile)

		// Upload and delete endpoints require API key
		r.Group(func(r chi.Router) {
			r.Use(apiKeyMw)
			r.Post("/media/{mediaType}", mediaHandler.UploadFile)
			r.Delete("/media/{mediaType}/{filename}", mediaHandler.DeleteFile)
		})
	})

	// Start server
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      r,
		ReadTimeout:  30 * time.Second, // Longer timeout for file uploads
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		logger.Logger.Info("Server starting", zap.Int("port", cfg.Server.Port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Logger.Fatal("Server failed to start", zap.Error(err))
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Logger.Info("Shutting down server...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Logger.Error("Server forced to shutdown", zap.Error(err))
	}

	logger.Logger.Info("Server exited")
}

// connectDB connects to the database
func connectDB(dsn string) (*sql.DB, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}

// runMigrations runs database migrations
func runMigrations(db *sql.DB) error {
	// Use service-specific migration table name to avoid conflicts with other services
	driver, err := mysql.WithInstance(db, &mysql.Config{
		MigrationsTable: "media_schema_migrations",
	})
	if err != nil {
		return fmt.Errorf("failed to create migration driver: %w", err)
	}

	// Get the working directory or use migrations folder relative to the binary
	migrationPath := "file://migrations"
	if _, err := os.Stat("migrations"); os.IsNotExist(err) {
		// Try parent directory if running from cmd
		if _, err := os.Stat("../migrations"); err == nil {
			migrationPath = "file://../migrations"
		}
	}

	m, err := migrate.NewWithDatabaseInstance(
		migrationPath,
		"mysql",
		driver,
	)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}
