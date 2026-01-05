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
	"github.com/go-redis/redis/v8"
	_ "github.com/go-sql-driver/mysql"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/mysql"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/hibiken/asynq"
	"github.com/japanesestudent/libs/auth/middleware"
	"github.com/japanesestudent/libs/auth/service"
	"github.com/japanesestudent/libs/config"
	"github.com/japanesestudent/libs/logger"
	loggerMiddleware "github.com/japanesestudent/libs/logger/middleware"
	sharedMiddleware "github.com/japanesestudent/libs/middlewares"
	_ "github.com/japanesestudent/task-service/docs"
	"github.com/japanesestudent/task-service/internal/handlers"
	"github.com/japanesestudent/task-service/internal/repositories"
	"github.com/japanesestudent/task-service/internal/services"
	httpSwagger "github.com/swaggo/http-swagger"
	"go.uber.org/zap"
)

// @title JapaneseStudent Task API
// @version 6.0
// @description API for managing immediate and scheduled tasks
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.email shelyahin.mihail@gmail.com

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:8083
// @BasePath /api/v6
// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name X-API-Key
// @description API key for service-to-service authentication
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token. Required for admin endpoints.
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

	logger.Logger.Info("Starting Task Service API")

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

	// Connect to Redis
	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	defer rdb.Close()

	// Test Redis connection
	ctx := context.Background()
	if err := rdb.Ping(ctx).Err(); err != nil {
		logger.Logger.Fatal("Failed to connect to Redis", zap.Error(err))
		os.Exit(1)
	}

	// Create Asynq client
	asynqClient := asynq.NewClient(asynq.RedisClientOpt{
		Addr:     fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	defer asynqClient.Close()

	// Initialize JWT token generator
	tokenGenerator := service.NewTokenGenerator(
		cfg.JWT.Secret,
		cfg.JWT.AccessTokenExpiry,
		cfg.JWT.RefreshTokenExpiry,
	)

	// Initialize repositories
	emailTemplateRepo := repositories.NewEmailTemplateRepository(db)
	immediateTaskRepo := repositories.NewImmediateTaskRepository(db)
	scheduledTaskRepo := repositories.NewScheduledTaskRepository(db)
	taskLogRepo := repositories.NewScheduledTaskLogRepository(db)

	// Initialize services
	emailTemplateService := services.NewEmailTemplateService(emailTemplateRepo, logger.Logger)
	immediateTaskService := services.NewImmediateTaskService(immediateTaskRepo, emailTemplateRepo, asynqClient, logger.Logger)
	scheduledTaskService := services.NewScheduledTaskService(scheduledTaskRepo, emailTemplateRepo, rdb, logger.Logger)
	taskLogService := services.NewTaskLogService(taskLogRepo, logger.Logger)

	// Initialize handlers
	taskHandler := handlers.NewTaskHandler(
		immediateTaskService,
		scheduledTaskService,
		asynqClient,
		rdb,
		logger.Logger,
	)
	adminHandler := handlers.NewAdminHandler(
		emailTemplateService,
		immediateTaskService,
		scheduledTaskService,
		taskLogService,
		asynqClient,
		rdb,
		logger.Logger,
	)

	// Initialize auth middleware
	apiKeyMiddleware := middleware.APIKeyMiddleware(cfg.APIKey)
	adminMiddleware := middleware.RoleMiddleware(tokenGenerator, 3) // Admin role = 3

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
		// Public endpoints (API Key protected)
		r.Group(func(r chi.Router) {
			r.Use(apiKeyMiddleware)
			taskHandler.RegisterRoutes(r)
		})

		// Admin endpoints (Role 3, JWT protected)
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
		MigrationsTable: "task_schema_migrations",
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
