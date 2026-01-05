package repositories

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/japanesestudent/task-service/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupEmailTemplateTestRepository creates an email template repository with a mock database
func setupEmailTemplateTestRepository(t *testing.T) (*emailTemplateRepository, sqlmock.Sqlmock, func()) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)

	repo := NewEmailTemplateRepository(db)

	cleanup := func() {
		db.Close()
	}

	return repo, mock, cleanup
}

func TestNewEmailTemplateRepository(t *testing.T) {
	db := &sql.DB{}

	repo := NewEmailTemplateRepository(db)

	assert.NotNil(t, repo)
	assert.Equal(t, db, repo.db)
}

func TestEmailTemplateRepository_Create(t *testing.T) {
	tests := []struct {
		name          string
		template      *models.EmailTemplate
		setupMock     func(sqlmock.Sqlmock)
		expectedError bool
		expectedID    int
	}{
		{
			name: "success",
			template: &models.EmailTemplate{
				Slug:            "test-slug",
				SubjectTemplate: "Subject {{.Name}}",
				BodyTemplate:    "Body {{.Name}}",
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`INSERT INTO email_templates`).
					WithArgs("test-slug", "Subject {{.Name}}", "Body {{.Name}}").
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			expectedError: false,
			expectedID:   1,
		},
		{
			name: "database error on insert",
			template: &models.EmailTemplate{
				Slug:            "test-slug",
				SubjectTemplate: "Subject",
				BodyTemplate:    "Body",
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`INSERT INTO email_templates`).
					WithArgs("test-slug", "Subject", "Body").
					WillReturnError(errors.New("database error"))
			},
			expectedError: true,
		},
		{
			name: "error getting last insert id",
			template: &models.EmailTemplate{
				Slug:            "test-slug",
				SubjectTemplate: "Subject",
				BodyTemplate:    "Body",
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`INSERT INTO email_templates`).
					WithArgs("test-slug", "Subject", "Body").
					WillReturnResult(sqlmock.NewErrorResult(errors.New("error getting id")))
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock, cleanup := setupEmailTemplateTestRepository(t)
			defer cleanup()

			tt.setupMock(mock)

			ctx := context.Background()
			err := repo.Create(ctx, tt.template)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedID, tt.template.ID)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestEmailTemplateRepository_GetByID(t *testing.T) {
	tests := []struct {
		name          string
		id            int
		setupMock     func(sqlmock.Sqlmock)
		expectedError bool
		expected      *models.EmailTemplate
	}{
		{
			name: "success",
			id:   1,
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "slug", "subject_template", "body_template", "created_at", "updated_at"}).
					AddRow(1, "test-slug", "Subject", "Body", time.Now(), time.Now())
				mock.ExpectQuery(`SELECT id, slug, subject_template, body_template, created_at, updated_at`).
					WithArgs(1).
					WillReturnRows(rows)
			},
			expectedError: false,
			expected: &models.EmailTemplate{
				ID:              1,
				Slug:            "test-slug",
				SubjectTemplate: "Subject",
				BodyTemplate:    "Body",
			},
		},
		{
			name: "not found",
			id:   999,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT id, slug, subject_template, body_template, created_at, updated_at`).
					WithArgs(999).
					WillReturnError(sql.ErrNoRows)
			},
			expectedError: true,
		},
		{
			name: "database error",
			id:   1,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT id, slug, subject_template, body_template, created_at, updated_at`).
					WithArgs(1).
					WillReturnError(errors.New("database error"))
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock, cleanup := setupEmailTemplateTestRepository(t)
			defer cleanup()

			tt.setupMock(mock)

			ctx := context.Background()
			result, err := repo.GetByID(ctx, tt.id)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.expected.ID, result.ID)
				assert.Equal(t, tt.expected.Slug, result.Slug)
				assert.Equal(t, tt.expected.SubjectTemplate, result.SubjectTemplate)
				assert.Equal(t, tt.expected.BodyTemplate, result.BodyTemplate)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestEmailTemplateRepository_GetTemplateByID(t *testing.T) {
	tests := []struct {
		name          string
		id            int
		setupMock     func(sqlmock.Sqlmock)
		expectedError bool
		expected      *models.EmailTemplateParts
	}{
		{
			name: "success",
			id:   1,
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"subject_template", "body_template"}).
					AddRow("Subject", "Body")
				mock.ExpectQuery(`SELECT subject_template, body_template`).
					WithArgs(1).
					WillReturnRows(rows)
			},
			expectedError: false,
			expected: &models.EmailTemplateParts{
				SubjectTemplate: "Subject",
				BodyTemplate:    "Body",
			},
		},
		{
			name: "not found",
			id:   999,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT subject_template, body_template`).
					WithArgs(999).
					WillReturnError(sql.ErrNoRows)
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock, cleanup := setupEmailTemplateTestRepository(t)
			defer cleanup()

			tt.setupMock(mock)

			ctx := context.Background()
			result, err := repo.GetTemplateByID(ctx, tt.id)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.expected.SubjectTemplate, result.SubjectTemplate)
				assert.Equal(t, tt.expected.BodyTemplate, result.BodyTemplate)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestEmailTemplateRepository_GetIDBySlug(t *testing.T) {
	tests := []struct {
		name          string
		slug          string
		setupMock     func(sqlmock.Sqlmock)
		expectedError bool
		expectedID   int
	}{
		{
			name: "success",
			slug: "test-slug",
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id"}).AddRow(1)
				mock.ExpectQuery(`SELECT id FROM email_templates WHERE slug = \?`).
					WithArgs("test-slug").
					WillReturnRows(rows)
			},
			expectedError: false,
			expectedID:    1,
		},
		{
			name: "not found",
			slug: "non-existent",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT id FROM email_templates WHERE slug = \?`).
					WithArgs("non-existent").
					WillReturnError(sql.ErrNoRows)
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock, cleanup := setupEmailTemplateTestRepository(t)
			defer cleanup()

			tt.setupMock(mock)

			ctx := context.Background()
			id, err := repo.GetIDBySlug(ctx, tt.slug)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Equal(t, 0, id)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedID, id)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestEmailTemplateRepository_GetAll(t *testing.T) {
	tests := []struct {
		name          string
		page          int
		count         int
		search        string
		setupMock     func(sqlmock.Sqlmock)
		expectedError bool
		expectedCount int
	}{
		{
			name:   "success without search",
			page:   1,
			count:   10,
			search:  "",
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "slug"}).
					AddRow(1, "test-slug-1").
					AddRow(2, "test-slug-2")
				mock.ExpectQuery(`SELECT id, slug`).
					WithArgs(10, 0).
					WillReturnRows(rows)
			},
			expectedError: false,
			expectedCount: 2,
		},
		{
			name:   "success with search",
			page:   1,
			count:   10,
			search:  "test",
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "slug"}).
					AddRow(1, "test-slug")
				mock.ExpectQuery(`SELECT id, slug`).
					WithArgs("%test%", 10, 0).
					WillReturnRows(rows)
			},
			expectedError: false,
			expectedCount: 1,
		},
		{
			name:   "database error",
			page:   1,
			count:   10,
			search:  "",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT id, slug`).
					WithArgs(10, 0).
					WillReturnError(errors.New("database error"))
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock, cleanup := setupEmailTemplateTestRepository(t)
			defer cleanup()

			tt.setupMock(mock)

			ctx := context.Background()
			result, err := repo.GetAll(ctx, tt.page, tt.count, tt.search)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Len(t, result, tt.expectedCount)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestEmailTemplateRepository_Update(t *testing.T) {
	tests := []struct {
		name          string
		id            int
		template      *models.EmailTemplate
		setupMock     func(sqlmock.Sqlmock)
		expectedError bool
	}{
		{
			name: "success - update all fields",
			id:   1,
			template: &models.EmailTemplate{
				Slug:            "new-slug",
				SubjectTemplate: "New Subject",
				BodyTemplate:    "New Body",
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`UPDATE email_templates`).
					WithArgs("new-slug", "New Subject", "New Body", 1).
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
			expectedError: false,
		},
		{
			name: "success - update only slug",
			id:   1,
			template: &models.EmailTemplate{
				Slug: "new-slug",
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`UPDATE email_templates`).
					WithArgs("new-slug", 1).
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
			expectedError: false,
		},
		{
			name:     "nothing to update",
			id:       1,
			template: &models.EmailTemplate{},
			setupMock: func(mock sqlmock.Sqlmock) {
				// No expectations - nothing should be executed
			},
			expectedError: false,
		},
		{
			name: "not found",
			id:   999,
			template: &models.EmailTemplate{
				Slug: "new-slug",
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`UPDATE email_templates`).
					WithArgs("new-slug", 999).
					WillReturnResult(sqlmock.NewResult(0, 0))
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock, cleanup := setupEmailTemplateTestRepository(t)
			defer cleanup()

			tt.setupMock(mock)

			ctx := context.Background()
			err := repo.Update(ctx, tt.id, tt.template)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestEmailTemplateRepository_Delete(t *testing.T) {
	tests := []struct {
		name          string
		id            int
		setupMock     func(sqlmock.Sqlmock)
		expectedError bool
	}{
		{
			name: "success",
			id:   1,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`DELETE FROM email_templates WHERE id = \?`).
					WithArgs(1).
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
			expectedError: false,
		},
		{
			name: "not found",
			id:   999,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`DELETE FROM email_templates WHERE id = \?`).
					WithArgs(999).
					WillReturnResult(sqlmock.NewResult(0, 0))
			},
			expectedError: true,
		},
		{
			name: "database error",
			id:   1,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`DELETE FROM email_templates WHERE id = \?`).
					WithArgs(1).
					WillReturnError(errors.New("database error"))
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock, cleanup := setupEmailTemplateTestRepository(t)
			defer cleanup()

			tt.setupMock(mock)

			ctx := context.Background()
			err := repo.Delete(ctx, tt.id)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestEmailTemplateRepository_ExistsBySlug(t *testing.T) {
	tests := []struct {
		name          string
		slug          string
		setupMock     func(sqlmock.Sqlmock)
		expectedError bool
		expected      bool
	}{
		{
			name: "exists",
			slug: "test-slug",
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"EXISTS(SELECT 1 FROM email_templates WHERE slug = ?)"}).AddRow(1)
				mock.ExpectQuery(`SELECT EXISTS`).
					WithArgs("test-slug").
					WillReturnRows(rows)
			},
			expectedError: false,
			expected:      true,
		},
		{
			name: "does not exist",
			slug: "non-existent",
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"EXISTS(SELECT 1 FROM email_templates WHERE slug = ?)"}).AddRow(0)
				mock.ExpectQuery(`SELECT EXISTS`).
					WithArgs("non-existent").
					WillReturnRows(rows)
			},
			expectedError: false,
			expected:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock, cleanup := setupEmailTemplateTestRepository(t)
			defer cleanup()

			tt.setupMock(mock)

			ctx := context.Background()
			exists, err := repo.ExistsBySlug(ctx, tt.slug)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, exists)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestEmailTemplateRepository_ExistsByID(t *testing.T) {
	tests := []struct {
		name          string
		id            int
		setupMock     func(sqlmock.Sqlmock)
		expectedError bool
		expected      bool
	}{
		{
			name: "exists",
			id:   1,
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"EXISTS(SELECT 1 FROM email_templates WHERE id = ?)"}).AddRow(1)
				mock.ExpectQuery(`SELECT EXISTS`).
					WithArgs(1).
					WillReturnRows(rows)
			},
			expectedError: false,
			expected:      true,
		},
		{
			name: "does not exist",
			id:   999,
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"EXISTS(SELECT 1 FROM email_templates WHERE id = ?)"}).AddRow(0)
				mock.ExpectQuery(`SELECT EXISTS`).
					WithArgs(999).
					WillReturnRows(rows)
			},
			expectedError: false,
			expected:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock, cleanup := setupEmailTemplateTestRepository(t)
			defer cleanup()

			tt.setupMock(mock)

			ctx := context.Background()
			exists, err := repo.ExistsByID(ctx, tt.id)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, exists)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
