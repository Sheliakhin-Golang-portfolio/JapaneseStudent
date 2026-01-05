package integration

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/hibiken/asynq"
	_ "github.com/go-sql-driver/mysql"
	"github.com/japanesestudent/task-service/internal/models"
	"github.com/japanesestudent/task-service/internal/repositories"
	"github.com/japanesestudent/task-service/internal/services"
	"github.com/japanesestudent/libs/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

var (
	testDB       *sql.DB
	testRedis    *redis.Client
	testAsynq    *asynq.Client
	testLogger   *zap.Logger
)

// cleanupTestData removes all test data
func cleanupTestData(t *testing.T, db *sql.DB) {
	t.Helper()
	_, err := db.Exec("DELETE FROM scheduled_task_logs")
	require.NoError(t, err, "Failed to cleanup scheduled_task_logs")
	_, err = db.Exec("DELETE FROM scheduled_tasks")
	require.NoError(t, err, "Failed to cleanup scheduled_tasks")
	_, err = db.Exec("DELETE FROM immediate_tasks")
	require.NoError(t, err, "Failed to cleanup immediate_tasks")
	_, err = db.Exec("DELETE FROM email_templates")
	require.NoError(t, err, "Failed to cleanup email_templates")
}

// seedTestData inserts test data into the database
func seedTestData(t *testing.T, db *sql.DB) {
	t.Helper()
	cleanupTestData(t, db)

	// Insert email template
	_, err := db.Exec(`
		INSERT INTO email_templates (slug, subject_template, body_template)
		VALUES ('test-template', 'Subject {{.Name}}', 'Body {{.Name}}')
	`)
	require.NoError(t, err, "Failed to seed email template")
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

	// Setup Redis
	testRedis = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       1, // Use DB 1 for tests
	})
	ctx := context.Background()
	if err = testRedis.Ping(ctx).Err(); err != nil {
		testLogger.Warn("Failed to connect to Redis, skipping integration tests", zap.Error(err))
		testRedis = nil
	}

	// Setup Asynq
	if testRedis != nil {
		testAsynq = asynq.NewClient(asynq.RedisClientOpt{
			Addr:     "localhost:6379",
			Password: "",
			DB:       1,
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
		seedTestData(t, testDB)

		templates, err := repo.GetAll(ctx, 1, 10, "")
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(templates), 1)
	})

	t.Run("Update", func(t *testing.T) {
		seedTestData(t, testDB)

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
		seedTestData(t, testDB)

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
	seedTestData(t, testDB)

	t.Run("Create and Get", func(t *testing.T) {
		templateID := 1
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
		templateID := 1
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
	seedTestData(t, testDB)

	t.Run("Create and Get", func(t *testing.T) {
		userID := 100
		templateID := 1
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
		templateID := 1
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
		seedTestData(t, testDB)

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
	seedTestData(t, testDB)

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
	seedTestData(t, testDB)

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
}
