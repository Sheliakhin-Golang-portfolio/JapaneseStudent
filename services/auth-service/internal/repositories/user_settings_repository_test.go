package repositories

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Sheliakhin-Golang-portfolio/JapaneseStudent/services/auth-service/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupUserSettingsTestRepository creates a user settings repository with a mock database
func setupUserSettingsTestRepository(t *testing.T) (*userSettingsRepository, sqlmock.Sqlmock, func()) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)

	repo := NewUserSettingsRepository(db)

	cleanup := func() {
		db.Close()
	}

	return repo, mock, cleanup
}

func TestNewUserSettingsRepository(t *testing.T) {
	db := &sql.DB{}

	repo := NewUserSettingsRepository(db)

	assert.NotNil(t, repo)
	assert.Equal(t, db, repo.db)
}

func TestUserSettingsRepository_Create(t *testing.T) {
	tests := []struct {
		name          string
		userId        int
		setupMock     func(sqlmock.Sqlmock)
		expectedError bool
	}{
		{
			name:   "success",
			userId: 1,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`INSERT INTO user_settings`).
					WithArgs(1).
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			expectedError: false,
		},
		{
			name:   "database error",
			userId: 1,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`INSERT INTO user_settings`).
					WithArgs(1).
					WillReturnError(errors.New("database error"))
			},
			expectedError: true,
		},
		{
			name:   "foreign key constraint - invalid user_id",
			userId: 999,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`INSERT INTO user_settings`).
					WithArgs(999).
					WillReturnError(errors.New("Error 1452: Cannot add or update a child row: a foreign key constraint fails"))
			},
			expectedError: true,
		},
		{
			name:   "duplicate user_id",
			userId: 1,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`INSERT INTO user_settings`).
					WithArgs(1).
					WillReturnError(errors.New("Error 1062: Duplicate entry '1' for key 'user_id'"))
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock, cleanup := setupUserSettingsTestRepository(t)
			defer cleanup()

			tt.setupMock(mock)

			err := repo.Create(context.Background(), tt.userId)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestUserSettingsRepository_GetByUserId(t *testing.T) {
	tests := []struct {
		name          string
		userId        int
		setupMock     func(sqlmock.Sqlmock)
		expectedError bool
		expectedID    int
	}{
		{
			name:   "success",
			userId: 1,
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "user_id", "new_word_count", "old_word_count", "alphabet_learn_count", "language", "alphabet_repeat"}).
					AddRow(1, 1, 20, 20, 10, "en", "in question")
				mock.ExpectQuery(`SELECT id, user_id, new_word_count, old_word_count, alphabet_learn_count, language, alphabet_repeat FROM user_settings WHERE user_id = \? LIMIT 1`).
					WithArgs(1).
					WillReturnRows(rows)
			},
			expectedError: false,
			expectedID:    1,
		},
		{
			name:   "not found",
			userId: 999,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT id, user_id, new_word_count, old_word_count, alphabet_learn_count, language, alphabet_repeat FROM user_settings WHERE user_id = \? LIMIT 1`).
					WithArgs(999).
					WillReturnError(sql.ErrNoRows)
			},
			expectedError: true,
			expectedID:    0,
		},
		{
			name:   "database error",
			userId: 1,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT id, user_id, new_word_count, old_word_count, alphabet_learn_count, language, alphabet_repeat FROM user_settings WHERE user_id = \? LIMIT 1`).
					WithArgs(1).
					WillReturnError(errors.New("database error"))
			},
			expectedError: true,
			expectedID:    0,
		},
		{
			name:   "scan error - invalid data types",
			userId: 1,
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "user_id", "new_word_count", "old_word_count", "alphabet_learn_count", "language", "alphabet_repeat"}).
					AddRow("invalid", 1, 20, 20, 10, "en", "in question")
				mock.ExpectQuery(`SELECT id, user_id, new_word_count, old_word_count, alphabet_learn_count, language, alphabet_repeat FROM user_settings WHERE user_id = \? LIMIT 1`).
					WithArgs(1).
					WillReturnRows(rows)
			},
			expectedError: true,
			expectedID:    0,
		},
		{
			name:   "success with all language types",
			userId: 2,
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "user_id", "new_word_count", "old_word_count", "alphabet_learn_count", "language", "alphabet_repeat"}).
					AddRow(2, 2, 30, 25, 15, "ru", "in question")
				mock.ExpectQuery(`SELECT id, user_id, new_word_count, old_word_count, alphabet_learn_count, language, alphabet_repeat FROM user_settings WHERE user_id = \? LIMIT 1`).
					WithArgs(2).
					WillReturnRows(rows)
			},
			expectedError: false,
			expectedID:    2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock, cleanup := setupUserSettingsTestRepository(t)
			defer cleanup()

			tt.setupMock(mock)

			userSettings, err := repo.GetByUserId(context.Background(), tt.userId)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, userSettings)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, userSettings)
				assert.Equal(t, tt.expectedID, userSettings.ID)
				assert.Equal(t, tt.userId, userSettings.UserID)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestUserSettingsRepository_Update(t *testing.T) {
	tests := []struct {
		name          string
		userId        int
		settings      *models.UserSettings
		setupMock     func(sqlmock.Sqlmock)
		expectedError bool
	}{
		{
			name:   "success",
			userId: 1,
			settings: &models.UserSettings{
				NewWordCount:       25,
				OldWordCount:       30,
				AlphabetLearnCount: 12,
				Language:           models.LanguageEnglish,
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`UPDATE user_settings SET new_word_count = \?, old_word_count = \?, alphabet_learn_count = \?, language = \? WHERE user_id = \?`).
					WithArgs(25, 30, 12, "en", 1).
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
			expectedError: false,
		},
		{
			name:   "user settings not found - 0 rows affected",
			userId: 999,
			settings: &models.UserSettings{
				NewWordCount:       25,
				OldWordCount:       30,
				AlphabetLearnCount: 12,
				Language:           models.LanguageEnglish,
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`UPDATE user_settings SET new_word_count = \?, old_word_count = \?, alphabet_learn_count = \?, language = \? WHERE user_id = \?`).
					WithArgs(25, 30, 12, "en", 999).
					WillReturnResult(sqlmock.NewResult(0, 0))
			},
			expectedError: true,
		},
		{
			name:   "database error",
			userId: 1,
			settings: &models.UserSettings{
				NewWordCount:       25,
				OldWordCount:       30,
				AlphabetLearnCount: 12,
				Language:           models.LanguageEnglish,
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`UPDATE user_settings SET new_word_count = \?, old_word_count = \?, alphabet_learn_count = \?, language = \? WHERE user_id = \?`).
					WithArgs(25, 30, 12, "en", 1).
					WillReturnError(errors.New("database error"))
			},
			expectedError: true,
		},
		{
			name:   "error getting rows affected",
			userId: 1,
			settings: &models.UserSettings{
				NewWordCount:       25,
				OldWordCount:       30,
				AlphabetLearnCount: 12,
				Language:           models.LanguageEnglish,
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`UPDATE user_settings SET new_word_count = \?, old_word_count = \?, alphabet_learn_count = \?, language = \? WHERE user_id = \?`).
					WithArgs(25, 30, 12, "en", 1).
					WillReturnResult(sqlmock.NewErrorResult(errors.New("rows affected error")))
			},
			expectedError: true,
		},
		{
			name:   "success with russian language",
			userId: 2,
			settings: &models.UserSettings{
				NewWordCount:       30,
				OldWordCount:       35,
				AlphabetLearnCount: 15,
				Language:           models.LanguageRussian,
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`UPDATE user_settings SET new_word_count = \?, old_word_count = \?, alphabet_learn_count = \?, language = \? WHERE user_id = \?`).
					WithArgs(30, 35, 15, "ru", 2).
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
			expectedError: false,
		},
		{
			name:   "success with german language",
			userId: 3,
			settings: &models.UserSettings{
				NewWordCount:       20,
				OldWordCount:       20,
				AlphabetLearnCount: 10,
				Language:           models.LanguageGerman,
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`UPDATE user_settings SET new_word_count = \?, old_word_count = \?, alphabet_learn_count = \?, language = \? WHERE user_id = \?`).
					WithArgs(20, 20, 10, "de", 3).
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock, cleanup := setupUserSettingsTestRepository(t)
			defer cleanup()

			tt.setupMock(mock)

			err := repo.Update(context.Background(), tt.userId, tt.settings)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
