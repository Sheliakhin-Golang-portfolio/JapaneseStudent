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

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/httprate"
	_ "github.com/go-sql-driver/mysql"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/mysql"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/japanesestudent/auth-service/docs"
	"github.com/japanesestudent/auth-service/internal/handlers"
	"github.com/japanesestudent/auth-service/internal/repositories"
	"github.com/japanesestudent/auth-service/internal/services"
	"github.com/japanesestudent/libs/auth/middleware"
	"github.com/japanesestudent/libs/auth/service"
	"github.com/japanesestudent/libs/config"
	"github.com/japanesestudent/libs/logger"
	loggerMiddleware "github.com/japanesestudent/libs/logger/middleware"
	sharedMiddleware "github.com/japanesestudent/libs/middlewares"
	httpSwagger "github.com/swaggo/http-swagger"
	"go.uber.org/zap"
)

// @title JapaneseStudent Auth API
// @version 4.0
// @description API for user authentication and authorization
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.email shelyahin.mihail@gmail.com

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:8081
// @BasePath /api/v6
// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.
func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v\n", err)
	}

	// Initialize logger
	if err := logger.Init(cfg.Logging.Level); err != nil {
		log.Fatalf("Failed to initialize logger: %v\n", err)
	}
	defer logger.Sync()

	logger.Logger.Info("Starting JapaneseStudent Auth Service")

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

	// Initialize JWT token generator
	tokenGenerator := service.NewTokenGenerator(
		cfg.JWT.Secret,
		cfg.JWT.AccessTokenExpiry,
		cfg.JWT.RefreshTokenExpiry,
	)

	// Initialize repositories
	userRepo := repositories.NewUserRepository(db)
	userTokenRepo := repositories.NewUserTokenRepository(db)
	userSettingsRepo := repositories.NewUserSettingsRepository(db)

	// Initialize services
	authService := services.NewAuthService(userRepo, userTokenRepo, userSettingsRepo, tokenGenerator, logger.Logger, cfg.MediaBaseURL, cfg.APIKey, cfg.ImmediateTaskBaseURL, cfg.VerificationURL)
	userSettingsService := services.NewUserSettingsService(userSettingsRepo)
	profileService := services.NewProfileService(userRepo, userSettingsRepo, tokenGenerator, cfg.MediaBaseURL, cfg.APIKey, cfg.ImmediateTaskBaseURL, cfg.VerificationURL, cfg.ScheduledTaskBaseURL, cfg.LearnServiceBaseURL, cfg.IsDockerContainer)

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(authService, logger.Logger)
	adminService := services.NewAdminService(userRepo, userTokenRepo, userSettingsRepo, tokenGenerator, logger.Logger, cfg.MediaBaseURL, cfg.APIKey, cfg.ScheduledTaskBaseURL, cfg.LearnServiceBaseURL, cfg.IsDockerContainer, cfg.ScheduledTaskBaseURL)
	profileHandler := handlers.NewProfileHandler(profileService, userSettingsService, logger.Logger)
	adminHandler := handlers.NewAdminHandler(adminService, logger.Logger, cfg.MediaBaseURL, cfg.IsDockerContainer, cfg.AuthServiceBaseURL)
	tokenCleaningHandler := handlers.NewTokenCleaningHandler(userTokenRepo, logger.Logger, cfg.JWT.RefreshTokenExpiry)

	// Initialize auth middleware
	authMiddleware := middleware.AuthMiddleware(tokenGenerator)
	adminMiddleware := middleware.RoleMiddleware(tokenGenerator, 3) // Admin role = 3
	apiKeyMiddleware := middleware.APIKeyMiddleware(cfg.APIKey)

	// Setup router
	r := chi.NewRouter()

	// Apply middleware
	r.Use(sharedMiddleware.RequestIDMiddleware)
	r.Use(loggerMiddleware.LoggerMiddleware(logger.Logger))
	r.Use(sharedMiddleware.RecoveryMiddleware(logger.Logger))
	r.Use(sharedMiddleware.CORSMiddleware(cfg.CORS.AllowedOrigins))
	r.Use(httprate.LimitByIP(100, time.Minute))
	r.Use(sharedMiddleware.RequestSizeLimitMiddleware(10 * 1024 * 1024)) // 10MB

	// Swagger documentation
	r.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL(fmt.Sprintf("http://localhost:%d/swagger/doc.json", cfg.Server.Port)),
	))

	// Scope router to /api/v6
	r.Route("/api/v6", func(r chi.Router) {
		// Register auth routes
		authHandler.RegisterRoutes(r)
		// Register profile routes
		profileHandler.RegisterRoutes(r, authMiddleware)
		// Register token cleaning routes with API key middleware
		r.Group(func(r chi.Router) {
			r.Use(apiKeyMiddleware)
			tokenCleaningHandler.RegisterRoutes(r)
		})
		// Register admin routes with role middleware
		r.Group(func(r chi.Router) {
			r.Use(adminMiddleware)
			adminHandler.RegisterRoutes(r)
		})
	})

	// Start server
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
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
		MigrationsTable: "auth_schema_migrations",
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
