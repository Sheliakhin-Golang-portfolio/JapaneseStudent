package integration

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
	_ "github.com/go-sql-driver/mysql"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/mysql"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/hibiken/asynq"
	"github.com/japanesestudent/libs/config"
	"github.com/japanesestudent/task-service/internal/models"
	"github.com/japanesestudent/task-service/internal/repositories"
	"github.com/japanesestudent/task-service/internal/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

var (
	testDB     *sql.DB
	testRedis  *redis.Client
	testAsynq  *asynq.Client
	testLogger *zap.Logger
)

// cleanupTestData removes all test data
func cleanupTestData(t *testing.T, db *sql.DB) {
	t.Helper()
	// Delete in reverse order of dependencies to avoid foreign key constraints
	// Ignore errors for missing tables - they may not exist if migrations haven't been run
	_, _ = db.Exec("DELETE FROM scheduled_task_logs")
	_, _ = db.Exec("DELETE FROM scheduled_tasks")
	_, _ = db.Exec("DELETE FROM immediate_tasks")
	_, _ = db.Exec("DELETE FROM email_templates")
}

// seedTestData inserts test data into the database and returns the template ID
func seedTestData(t *testing.T, db *sql.DB) int {
	t.Helper()
	cleanupTestData(t, db)

	// Insert email template
	result, err := db.Exec(`
		INSERT INTO email_templates (slug, subject_template, body_template)
		VALUES ('test-template', 'Subject {{.Name}}', 'Body {{.Name}}')
	`)
	require.NoError(t, err, "Failed to seed email template")
	
	templateID, err := result.LastInsertId()
	require.NoError(t, err, "Failed to get template ID")
	
	return int(templateID)
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
	dsn := ""
	if err == nil {
		dsn = cfg.DSN()
	}
	if dsn == "" {
		// Default test database connection
		dsn = "root:password@tcp(localhost:3306)/japanesestudent_test?parseTime=true&charset=utf8mb4"
	}

	testDB, err = sql.Open("mysql", dsn)
	if err != nil {
		testLogger.Warn("Failed to connect to test database, skipping integration tests", zap.Error(err))
		os.Exit(0)
	}

	// Test connection
	if err = testDB.Ping(); err != nil {
		testLogger.Warn("Failed to ping test database, skipping integration tests", zap.Error(err))
		os.Exit(0)
	}

	// Drop all existing tables to ensure clean state
	if err = dropAllTables(testDB); err != nil {
		testLogger.Warn("Failed to drop tables, skipping integration tests", zap.Error(err))
		os.Exit(0)
	}

	// Run migrations
	if err = runMigrations(testDB); err != nil {
		testLogger.Warn("Failed to run migrations, skipping integration tests", zap.Error(err))
		os.Exit(0)
	}

	// Setup Redis using config from LoadTestConfig, with fallback defaults
	redisHost := "localhost"
	redisPort := 6379
	redisPassword := ""
	redisDB := 1
	if cfg != nil {
		if cfg.Redis.Host != "" {
			redisHost = cfg.Redis.Host
		}
		if cfg.Redis.Port != 0 {
			redisPort = cfg.Redis.Port
		}
		if cfg.Redis.Password != "" {
			redisPassword = cfg.Redis.Password
		}
		// Use DB 1 for tests (override config DB to avoid conflicts)
		redisDB = 1
	}

	testRedis = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", redisHost, redisPort),
		Password: redisPassword,
		DB:       redisDB,
	})
	ctx := context.Background()
	if err = testRedis.Ping(ctx).Err(); err != nil {
		testLogger.Warn("Failed to connect to Redis, skipping integration tests", zap.Error(err))
		testRedis = nil
	}

	// Setup Asynq
	if testRedis != nil {
		testAsynq = asynq.NewClient(asynq.RedisClientOpt{
			Addr:     fmt.Sprintf("%s:%d", redisHost, redisPort),
			Password: redisPassword,
			DB:       redisDB,
		})
	}

	// Run tests
	code := m.Run()

	// Cleanup
	if testDB != nil {
		testDB.Close()
	}
	if testRedis != nil {
		testRedis.Close()
	}
	if testAsynq != nil {
		testAsynq.Close()
	}

	os.Exit(code)
}

// dropAllTables drops all tables to ensure clean migration state
func dropAllTables(db *sql.DB) error {
	tables := []string{
		"scheduled_task_logs",
		"scheduled_tasks",
		"immediate_tasks",
		"email_templates",
	}

	// Disable foreign key checks temporarily
	_, err := db.Exec("SET FOREIGN_KEY_CHECKS = 0")
	if err != nil {
		return fmt.Errorf("failed to disable foreign key checks: %w", err)
	}
	defer db.Exec("SET FOREIGN_KEY_CHECKS = 1")

	// Drop tables in reverse dependency order
	for _, table := range tables {
		_, _ = db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", table))
	}

	// Drop migration table
	_, _ = db.Exec("DROP TABLE IF EXISTS task_schema_migrations")

	return nil
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

	// Get the working directory or use migrations folder relative to the test
	migrationPath := "file://../../migrations"
	if _, err := os.Stat("../../migrations"); os.IsNotExist(err) {
		// Try alternative path
		if _, err := os.Stat("../migrations"); err == nil {
			migrationPath = "file://../migrations"
		} else if _, err := os.Stat("migrations"); err == nil {
			migrationPath = "file://migrations"
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

	// Check for dirty migration state and fix it (though we drop tables first, this is a safety check)
	_, dirty, err := m.Version()
	if err != nil && err != migrate.ErrNilVersion {
		return fmt.Errorf("failed to get migration version: %w", err)
	}
	if dirty {
		// Force to version 0, then migrate up fresh
		if err := m.Force(0); err != nil {
			return fmt.Errorf("failed to force migration version to 0: %w", err)
		}
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}

func TestEmailTemplateRepository_Integration(t *testing.T) {
	if testDB == nil {
		t.Skip("Database not available")
	}

	repo := repositories.NewEmailTemplateRepository(testDB)
	ctx := context.Background()

	// Clean up before test
	cleanupTestData(t, testDB)

	t.Run("Create and Get", func(t *testing.T) {
		template := &models.EmailTemplate{
			Slug:            "integration-test",
			SubjectTemplate: "Test Subject",
			BodyTemplate:    "Test Body",
		}

		err := repo.Create(ctx, template)
		require.NoError(t, err)
		assert.Greater(t, template.ID, 0)

		retrieved, err := repo.GetByID(ctx, template.ID)
		require.NoError(t, err)
		assert.Equal(t, template.Slug, retrieved.Slug)
		assert.Equal(t, template.SubjectTemplate, retrieved.SubjectTemplate)
		assert.Equal(t, template.BodyTemplate, retrieved.BodyTemplate)
	})

	t.Run("GetAll", func(t *testing.T) {
		_ = seedTestData(t, testDB)

		templates, err := repo.GetAll(ctx, 1, 10, "")
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(templates), 1)
	})

	t.Run("Update", func(t *testing.T) {
		_ = seedTestData(t, testDB)

		// Get existing template
		templates, err := repo.GetAll(ctx, 1, 1, "test-template")
		require.NoError(t, err)
		require.Len(t, templates, 1)

		template := &models.EmailTemplate{
			Slug:            "updated-template",
			SubjectTemplate: "Updated Subject",
		}

		err = repo.Update(ctx, templates[0].ID, template)
		require.NoError(t, err)

		updated, err := repo.GetByID(ctx, templates[0].ID)
		require.NoError(t, err)
		assert.Equal(t, "updated-template", updated.Slug)
	})

	t.Run("Delete", func(t *testing.T) {
		_ = seedTestData(t, testDB)

		templates, err := repo.GetAll(ctx, 1, 1, "test-template")
		require.NoError(t, err)
		if len(templates) > 0 {
			err = repo.Delete(ctx, templates[0].ID)
			require.NoError(t, err)

			_, err = repo.GetByID(ctx, templates[0].ID)
			assert.Error(t, err)
		}
	})
}

func TestImmediateTaskRepository_Integration(t *testing.T) {
	if testDB == nil {
		t.Skip("Database not available")
	}

	repo := repositories.NewImmediateTaskRepository(testDB)
	ctx := context.Background()

	cleanupTestData(t, testDB)
	templateID := seedTestData(t, testDB)

	t.Run("Create and Get", func(t *testing.T) {
		task := &models.ImmediateTask{
			UserID:     100,
			TemplateID: &templateID,
			Content:    "test@example.com;John",
			Status:     models.ImmediateTaskStatusEnqueued,
		}

		err := repo.Create(ctx, task)
		require.NoError(t, err)
		assert.Greater(t, task.ID, 0)

		retrieved, err := repo.GetByID(ctx, task.ID)
		require.NoError(t, err)
		assert.Equal(t, task.UserID, retrieved.UserID)
		assert.Equal(t, task.Content, retrieved.Content)
		assert.Equal(t, task.Status, retrieved.Status)
	})

	t.Run("UpdateStatus", func(t *testing.T) {
		task := &models.ImmediateTask{
			UserID:     100,
			TemplateID: &templateID,
			Content:    "test@example.com;John",
			Status:     models.ImmediateTaskStatusEnqueued,
		}

		err := repo.Create(ctx, task)
		require.NoError(t, err)

		err = repo.UpdateStatus(ctx, task.ID, models.ImmediateTaskStatusCompleted, "")
		require.NoError(t, err)

		updated, err := repo.GetByID(ctx, task.ID)
		require.NoError(t, err)
		assert.Equal(t, models.ImmediateTaskStatusCompleted, updated.Status)
	})
}

func TestScheduledTaskRepository_Integration(t *testing.T) {
	if testDB == nil {
		t.Skip("Database not available")
	}

	repo := repositories.NewScheduledTaskRepository(testDB)
	ctx := context.Background()

	cleanupTestData(t, testDB)
	templateID := seedTestData(t, testDB)

	t.Run("Create and Get", func(t *testing.T) {
		userID := 100
		nextRun := time.Now().Add(1 * time.Hour)
		task := &models.ScheduledTask{
			UserID:     &userID,
			TemplateID: &templateID,
			URL:        "http://example.com",
			Content:    "test@example.com;John",
			NextRun:    nextRun,
			Active:     true,
			Cron:       "0 0 * * *",
		}

		err := repo.Create(ctx, task)
		require.NoError(t, err)
		assert.Greater(t, task.ID, 0)

		retrieved, err := repo.GetByID(ctx, task.ID)
		require.NoError(t, err)
		assert.Equal(t, task.URL, retrieved.URL)
		assert.Equal(t, task.Content, retrieved.Content)
		assert.Equal(t, task.Active, retrieved.Active)
	})

	t.Run("UpdatePreviousRunAndNextRun", func(t *testing.T) {
		userID := 100
		nextRun := time.Now().Add(1 * time.Hour)
		task := &models.ScheduledTask{
			UserID:     &userID,
			TemplateID: &templateID,
			URL:        "http://example.com",
			Content:    "test@example.com;John",
			NextRun:    nextRun,
			Active:     true,
			Cron:       "0 0 * * *",
		}

		err := repo.Create(ctx, task)
		require.NoError(t, err)

		previousRun := time.Now()
		newNextRun := time.Now().Add(2 * time.Hour)
		err = repo.UpdatePreviousRunAndNextRun(ctx, task.ID, previousRun, newNextRun)
		require.NoError(t, err)

		updated, err := repo.GetByID(ctx, task.ID)
		require.NoError(t, err)
		assert.NotNil(t, updated.PreviousRun)
	})
}

func TestEmailTemplateService_Integration(t *testing.T) {
	if testDB == nil {
		t.Skip("Database not available")
	}

	repo := repositories.NewEmailTemplateRepository(testDB)
	svc := services.NewEmailTemplateService(repo, testLogger)
	ctx := context.Background()

	cleanupTestData(t, testDB)

	t.Run("Create", func(t *testing.T) {
		req := &models.CreateUpdateEmailTemplateRequest{
			Slug:            "service-test",
			SubjectTemplate: "Service Subject",
			BodyTemplate:    "Service Body",
		}

		id, err := svc.Create(ctx, req)
		require.NoError(t, err)
		assert.Greater(t, id, 0)
	})

	t.Run("GetAll", func(t *testing.T) {
		_ = seedTestData(t, testDB)

		templates, err := svc.GetAll(ctx, 1, 10, "")
		require.NoError(t, err)
		assert.NotNil(t, templates)
	})
}

func TestImmediateTaskService_Integration(t *testing.T) {
	if testDB == nil || testAsynq == nil {
		t.Skip("Database or Asynq not available")
	}

	taskRepo := repositories.NewImmediateTaskRepository(testDB)
	templateRepo := repositories.NewEmailTemplateRepository(testDB)
	svc := services.NewImmediateTaskService(taskRepo, templateRepo, testAsynq, testLogger)
	ctx := context.Background()

	cleanupTestData(t, testDB)
	_ = seedTestData(t, testDB)

	t.Run("Create", func(t *testing.T) {
		req := &models.CreateImmediateTaskRequest{
			UserID:    100,
			EmailSlug: "test-template",
			Content:   "test@example.com;John",
		}

		id, err := svc.Create(ctx, req)
		require.NoError(t, err)
		assert.Greater(t, id, 0)
	})
}

func TestScheduledTaskService_Integration(t *testing.T) {
	if testDB == nil || testRedis == nil {
		t.Skip("Database or Redis not available")
	}

	taskRepo := repositories.NewScheduledTaskRepository(testDB)
	templateRepo := repositories.NewEmailTemplateRepository(testDB)
	svc := services.NewScheduledTaskService(taskRepo, templateRepo, testRedis, testLogger)
	ctx := context.Background()

	cleanupTestData(t, testDB)
	templateID := seedTestData(t, testDB)

	// Clean Redis
	if testRedis != nil {
		testRedis.Del(ctx, "scheduled_tasks")
	}

	t.Run("Create with URL", func(t *testing.T) {
		userID := 100
		req := &models.CreateScheduledTaskRequest{
			UserID: &userID,
			URL:    "http://example.com",
			Cron:   "0 0 * * *",
		}

		id, err := svc.Create(ctx, req)
		require.NoError(t, err)
		assert.Greater(t, id, 0)
	})

	t.Run("Create with email slug", func(t *testing.T) {
		userID := 100
		req := &models.CreateScheduledTaskRequest{
			UserID:    &userID,
			EmailSlug: "test-template",
			Content:   "test@example.com;John",
			Cron:      "0 0 * * *",
		}

		id, err := svc.Create(ctx, req)
		require.NoError(t, err)
		assert.Greater(t, id, 0)
	})

	t.Run("Create duplicate task returns 0", func(t *testing.T) {
		userID := 100
		url := "http://duplicate-test.com"
		req := &models.CreateScheduledTaskRequest{
			UserID: &userID,
			URL:    url,
			Cron:   "0 0 * * *",
		}

		// Create first task
		id1, err := svc.Create(ctx, req)
		require.NoError(t, err)
		assert.Greater(t, id1, 0)

		// Try to create duplicate
		id2, err := svc.Create(ctx, req)
		require.NoError(t, err)
		assert.Equal(t, 0, id2, "Duplicate task should return ID 0")
	})

	t.Run("CreateAdmin with URL", func(t *testing.T) {
		userID := 100
		req := &models.AdminCreateScheduledTaskRequest{
			UserID: &userID,
			URL:    "http://admin-example.com",
			Cron:   "0 0 * * *",
		}

		id, err := svc.CreateAdmin(ctx, req)
		require.NoError(t, err)
		assert.Greater(t, id, 0)
	})

	t.Run("CreateAdmin with template ID", func(t *testing.T) {
		userID := 100
		req := &models.AdminCreateScheduledTaskRequest{
			UserID:     &userID,
			TemplateID: &templateID,
			Content:    "admin@example.com;Admin User",
			Cron:       "0 0 * * *",
		}

		id, err := svc.CreateAdmin(ctx, req)
		require.NoError(t, err)
		assert.Greater(t, id, 0)
	})

	t.Run("Update task", func(t *testing.T) {
		userID := 100
		req := &models.CreateScheduledTaskRequest{
			UserID: &userID,
			URL:    "http://update-test.com",
			Cron:   "0 0 * * *",
		}

		id, err := svc.Create(ctx, req)
		require.NoError(t, err)
		assert.Greater(t, id, 0)

		newURL := "http://updated-url.com"
		updateReq := &models.UpdateScheduledTaskRequest{
			URL: &newURL,
		}

		err = svc.Update(ctx, id, updateReq)
		require.NoError(t, err)

		updated, err := svc.GetByID(ctx, id)
		require.NoError(t, err)
		assert.Equal(t, newURL, updated.URL)
	})

	t.Run("Update task active status", func(t *testing.T) {
		userID := 100
		req := &models.CreateScheduledTaskRequest{
			UserID: &userID,
			URL:    "http://active-test.com",
			Cron:   "0 0 * * *",
		}

		id, err := svc.Create(ctx, req)
		require.NoError(t, err)
		assert.Greater(t, id, 0)

		active := false
		updateReq := &models.UpdateScheduledTaskRequest{
			Active: &active,
		}

		err = svc.Update(ctx, id, updateReq)
		require.NoError(t, err)

		updated, err := svc.GetByID(ctx, id)
		require.NoError(t, err)
		assert.Equal(t, active, updated.Active)
	})

	t.Run("DeleteByUserID", func(t *testing.T) {
		userID := 200
		// Create multiple tasks for the user
		req1 := &models.CreateScheduledTaskRequest{
			UserID: &userID,
			URL:    "http://delete-user-test1.com",
			Cron:   "0 0 * * *",
		}
		req2 := &models.CreateScheduledTaskRequest{
			UserID: &userID,
			URL:    "http://delete-user-test2.com",
			Cron:   "0 0 * * *",
		}

		id1, err := svc.Create(ctx, req1)
		require.NoError(t, err)
		assert.Greater(t, id1, 0)

		id2, err := svc.Create(ctx, req2)
		require.NoError(t, err)
		assert.Greater(t, id2, 0)

		// Delete all tasks for the user
		err = svc.DeleteByUserID(ctx, userID)
		require.NoError(t, err)

		// Verify tasks are deleted
		_, err = svc.GetByID(ctx, id1)
		assert.Error(t, err, "Task should be deleted")

		_, err = svc.GetByID(ctx, id2)
		assert.Error(t, err, "Task should be deleted")
	})
}
