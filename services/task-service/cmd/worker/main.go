package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-redis/redis/v8"
	_ "github.com/go-sql-driver/mysql"
	"github.com/hibiken/asynq"
	"github.com/japanesestudent/libs/config"
	"github.com/japanesestudent/libs/logger"
	"github.com/japanesestudent/task-service/internal/repositories"
	"go.uber.org/zap"
)

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

	logger.Logger.Info("Starting Task Service Worker")

	// Connect to database
	db, err := connectDB(cfg.DSN())
	if err != nil {
		logger.Logger.Fatal("Failed to connect to database", zap.Error(err))
		os.Exit(1)
	}
	defer db.Close()

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

	// Initialize repositories
	emailTemplateRepo := repositories.NewEmailTemplateRepository(db)
	immediateTaskRepo := repositories.NewImmediateTaskRepository(db)
	scheduledTaskRepo := repositories.NewScheduledTaskRepository(db)
	taskLogRepo := repositories.NewScheduledTaskLogRepository(db)

	// Create Asynq server
	srv := asynq.NewServer(
		asynq.RedisClientOpt{
			Addr:     fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port),
			Password: cfg.Redis.Password,
			DB:       cfg.Redis.DB,
		},
		asynq.Config{
			Queues: map[string]int{
				"immediate": 5,
				"default":   1,
			},
		},
	)

	// Create worker instance
	worker := NewWorker(
		logger.Logger,
		immediateTaskRepo,
		scheduledTaskRepo,
		taskLogRepo,
		emailTemplateRepo,
		cfg.SMTP.Host,
		cfg.SMTP.Port,
		cfg.SMTP.Username,
		cfg.SMTP.Password,
		cfg.SMTP.From,
		cfg.APIKey,
	)

	// Register task handlers
	mux := asynq.NewServeMux()
	mux.HandleFunc("immediate:task", worker.HandleImmediateTask)
	mux.HandleFunc("scheduled:task", worker.HandleScheduledTask)

	// Start worker
	go func() {
		if err := srv.Run(mux); err != nil {
			logger.Logger.Fatal("Failed to start worker", zap.Error(err))
		}
	}()

	logger.Logger.Info("Worker started")

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Logger.Info("Shutting down worker...")
	srv.Shutdown()
	logger.Logger.Info("Worker exited")
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
