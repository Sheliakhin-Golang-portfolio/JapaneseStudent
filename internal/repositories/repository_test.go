package repositories

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/japanesestudent/backend/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// setupTestRepository creates a repository with a mock database
func setupTestRepository(t *testing.T) (*charactersRepository, sqlmock.Sqlmock, func()) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)

	logger, err := zap.NewDevelopment()
	require.NoError(t, err)

	repo := NewCharactersRepository(db, logger)

	cleanup := func() {
		db.Close()
	}

	return repo, mock, cleanup
}

func TestNewCharactersRepository(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	db := &sql.DB{}

	repo := NewCharactersRepository(db, logger)

	assert.NotNil(t, repo)
	assert.Equal(t, db, repo.db)
	assert.Equal(t, logger, repo.logger)
}

func TestCharactersRepository_GetAll(t *testing.T) {
	tests := []struct {
		name          string
		alphabetType  models.AlphabetType
		locale        models.Locale
		setupMock     func(sqlmock.Sqlmock)
		expectedError bool
		expectedCount int
	}{
		{
			name:         "success hiragana english",
			alphabetType: models.AlphabetTypeHiragana,
			locale:       models.LocaleEnglish,
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "consonant", "vowel", "hiragana", "english_reading"}).
					AddRow(1, "", "a", "あ", "a").
					AddRow(2, "k", "a", "か", "ka")
				mock.ExpectQuery(`SELECT id, consonant, vowel, hiragana AS display_character, english_reading AS reading FROM characters ORDER BY id`).
					WillReturnRows(rows)
			},
			expectedError: false,
			expectedCount: 2,
		},
		{
			name:         "success katakana russian",
			alphabetType: models.AlphabetTypeKatakana,
			locale:       models.LocaleRussian,
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "consonant", "vowel", "katakana", "russian_reading"}).
					AddRow(1, "", "a", "ア", "а").
					AddRow(2, "k", "a", "カ", "ка")
				mock.ExpectQuery(`SELECT id, consonant, vowel, katakana AS display_character, russian_reading AS reading FROM characters ORDER BY id`).
					WillReturnRows(rows)
			},
			expectedError: false,
			expectedCount: 2,
		},
		{
			name:         "invalid alphabet type",
			alphabetType: "invalid",
			locale:       models.LocaleEnglish,
			setupMock: func(mock sqlmock.Sqlmock) {
				// No query expected for invalid type
			},
			expectedError: true,
			expectedCount: 0,
		},
		{
			name:         "invalid locale",
			alphabetType: models.AlphabetTypeHiragana,
			locale:       "invalid",
			setupMock: func(mock sqlmock.Sqlmock) {
				// No query expected for invalid locale
			},
			expectedError: true,
			expectedCount: 0,
		},
		{
			name:         "database query error",
			alphabetType: models.AlphabetTypeHiragana,
			locale:       models.LocaleEnglish,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT id, consonant, vowel, hiragana AS display_character, english_reading AS reading FROM characters ORDER BY id`).
					WillReturnError(errors.New("database error"))
			},
			expectedError: true,
			expectedCount: 0,
		},
		{
			name:         "scan error",
			alphabetType: models.AlphabetTypeHiragana,
			locale:       models.LocaleEnglish,
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "consonant", "vowel", "hiragana", "english_reading"}).
					AddRow("invalid", "", "a", "あ", "a") // Invalid type for id
				mock.ExpectQuery(`SELECT id, consonant, vowel, hiragana AS display_character, english_reading AS reading FROM characters ORDER BY id`).
					WillReturnRows(rows)
			},
			expectedError: true,
			expectedCount: 0,
		},
		{
			name:         "rows iteration error",
			alphabetType: models.AlphabetTypeHiragana,
			locale:       models.LocaleEnglish,
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "consonant", "vowel", "hiragana", "english_reading"}).
					AddRow(1, "", "a", "あ", "a").
					RowError(0, errors.New("row error"))
				mock.ExpectQuery(`SELECT id, consonant, vowel, hiragana AS display_character, english_reading AS reading FROM characters ORDER BY id`).
					WillReturnRows(rows)
			},
			expectedError: true,
			expectedCount: 0,
		},
		{
			name:         "empty result",
			alphabetType: models.AlphabetTypeKatakana,
			locale:       models.LocaleEnglish,
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "consonant", "vowel", "katakana", "english_reading"})
				mock.ExpectQuery(`SELECT id, consonant, vowel, katakana AS display_character, english_reading AS reading FROM characters ORDER BY id`).
					WillReturnRows(rows)
			},
			expectedError: false,
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock, cleanup := setupTestRepository(t)
			defer cleanup()

			tt.setupMock(mock)

			result, err := repo.GetAll(context.Background(), tt.alphabetType, tt.locale)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				// Empty/nil slice is valid for empty results
				if tt.expectedCount == 0 {
					// In Go, nil slice and empty slice both have length 0
					assert.Len(t, result, 0)
				} else {
					assert.NotNil(t, result)
					assert.Len(t, result, tt.expectedCount)
				}
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestCharactersRepository_GetByRowColumn(t *testing.T) {
	tests := []struct {
		name          string
		alphabetType  models.AlphabetType
		locale        models.Locale
		character     string
		setupMock     func(sqlmock.Sqlmock)
		expectedError bool
		expectedCount int
	}{
		{
			name:         "success with vowel filter",
			alphabetType: models.AlphabetTypeHiragana,
			locale:       models.LocaleEnglish,
			character:    "a",
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "consonant", "vowel", "hiragana", "english_reading"}).
					AddRow(1, "", "a", "あ", "a").
					AddRow(2, "k", "a", "か", "ka")
				mock.ExpectQuery(`SELECT id, consonant, vowel, hiragana AS display_character, english_reading AS reading FROM characters WHERE \(consonant = \? OR vowel = \?\) ORDER BY id`).
					WithArgs("a", "a").
					WillReturnRows(rows)
			},
			expectedError: false,
			expectedCount: 2,
		},
		{
			name:         "success with consonant filter",
			alphabetType: models.AlphabetTypeKatakana,
			locale:       models.LocaleRussian,
			character:    "k",
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "consonant", "vowel", "katakana", "russian_reading"}).
					AddRow(1, "k", "a", "カ", "ка").
					AddRow(2, "k", "i", "キ", "ки")
				mock.ExpectQuery(`SELECT id, consonant, vowel, katakana AS display_character, russian_reading AS reading FROM characters WHERE \(consonant = \? OR vowel = \?\) ORDER BY id`).
					WithArgs("k", "k").
					WillReturnRows(rows)
			},
			expectedError: false,
			expectedCount: 2,
		},
		{
			name:         "invalid alphabet type",
			alphabetType: "invalid",
			locale:       models.LocaleEnglish,
			character:    "a",
			setupMock: func(mock sqlmock.Sqlmock) {
				// No query expected
			},
			expectedError: true,
			expectedCount: 0,
		},
		{
			name:         "invalid locale",
			alphabetType: models.AlphabetTypeHiragana,
			locale:       "invalid",
			character:    "a",
			setupMock: func(mock sqlmock.Sqlmock) {
				// No query expected
			},
			expectedError: true,
			expectedCount: 0,
		},
		{
			name:         "database query error",
			alphabetType: models.AlphabetTypeHiragana,
			locale:       models.LocaleEnglish,
			character:    "a",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT id, consonant, vowel, hiragana AS display_character, english_reading AS reading FROM characters WHERE \(consonant = \? OR vowel = \?\) ORDER BY id`).
					WithArgs("a", "a").
					WillReturnError(errors.New("database error"))
			},
			expectedError: true,
			expectedCount: 0,
		},
		{
			name:         "scan error",
			alphabetType: models.AlphabetTypeHiragana,
			locale:       models.LocaleEnglish,
			character:    "a",
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "consonant", "vowel", "hiragana", "english_reading"}).
					AddRow("invalid", "", "a", "あ", "a")
				mock.ExpectQuery(`SELECT id, consonant, vowel, hiragana AS display_character, english_reading AS reading FROM characters WHERE \(consonant = \? OR vowel = \?\) ORDER BY id`).
					WithArgs("a", "a").
					WillReturnRows(rows)
			},
			expectedError: true,
			expectedCount: 0,
		},
		{
			name:         "rows iteration error",
			alphabetType: models.AlphabetTypeHiragana,
			locale:       models.LocaleEnglish,
			character:    "a",
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "consonant", "vowel", "hiragana", "english_reading"}).
					AddRow(1, "", "a", "あ", "a").
					RowError(0, errors.New("row error"))
				mock.ExpectQuery(`SELECT id, consonant, vowel, hiragana AS display_character, english_reading AS reading FROM characters WHERE \(consonant = \? OR vowel = \?\) ORDER BY id`).
					WithArgs("a", "a").
					WillReturnRows(rows)
			},
			expectedError: true,
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock, cleanup := setupTestRepository(t)
			defer cleanup()

			tt.setupMock(mock)

			result, err := repo.GetByRowColumn(context.Background(), tt.alphabetType, tt.locale, tt.character)

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

func TestCharactersRepository_GetByID(t *testing.T) {
	tests := []struct {
		name          string
		id            int
		locale        models.Locale
		setupMock     func(sqlmock.Sqlmock)
		expectedError bool
		expectedID    int
	}{
		{
			name:   "success english locale",
			id:     1,
			locale: models.LocaleEnglish,
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "consonant", "vowel", "reading", "katakana", "hiragana"}).
					AddRow(1, "", "a", "a", "ア", "あ")
				mock.ExpectQuery(`SELECT id, consonant, vowel, english_reading as reading, katakana, hiragana FROM characters WHERE id = \?`).
					WithArgs(1).
					WillReturnRows(rows)
			},
			expectedError: false,
			expectedID:    1,
		},
		{
			name:   "success russian locale",
			id:     2,
			locale: models.LocaleRussian,
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "consonant", "vowel", "reading", "katakana", "hiragana"}).
					AddRow(2, "k", "a", "ка", "カ", "か")
				mock.ExpectQuery(`SELECT id, consonant, vowel, russian_reading as reading, katakana, hiragana FROM characters WHERE id = \?`).
					WithArgs(2).
					WillReturnRows(rows)
			},
			expectedError: false,
			expectedID:    2,
		},
		{
			name:   "not found",
			id:     999,
			locale: models.LocaleEnglish,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT id, consonant, vowel, english_reading as reading, katakana, hiragana FROM characters WHERE id = \?`).
					WithArgs(999).
					WillReturnError(sql.ErrNoRows)
			},
			expectedError: true,
			expectedID:    0,
		},
		{
			name:   "invalid locale",
			id:     1,
			locale: "invalid",
			setupMock: func(mock sqlmock.Sqlmock) {
				// No query expected
			},
			expectedError: true,
			expectedID:    0,
		},
		{
			name:   "database error",
			id:     1,
			locale: models.LocaleEnglish,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT id, consonant, vowel, english_reading as reading, katakana, hiragana FROM characters WHERE id = \?`).
					WithArgs(1).
					WillReturnError(errors.New("database error"))
			},
			expectedError: true,
			expectedID:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock, cleanup := setupTestRepository(t)
			defer cleanup()

			tt.setupMock(mock)

			result, err := repo.GetByID(context.Background(), tt.id, tt.locale)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.expectedID, result.ID)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestCharactersRepository_GetRandomForReadingTest(t *testing.T) {
	tests := []struct {
		name          string
		alphabetType  models.AlphabetType
		locale        models.Locale
		count         int
		setupMock     func(sqlmock.Sqlmock)
		expectedError bool
		expectedCount int
	}{
		{
			name:         "success hiragana english",
			alphabetType: models.AlphabetTypeHiragana,
			locale:       models.LocaleEnglish,
			count:        2,
			setupMock: func(mock sqlmock.Sqlmock) {
				// First query for correct characters
				rows1 := sqlmock.NewRows([]string{"id", "display_character", "reading"}).
					AddRow(1, "あ", "a").
					AddRow(2, "い", "i")
				mock.ExpectQuery(`SELECT id, hiragana AS display_character, english_reading AS reading FROM characters WHERE hiragana IS NOT NULL AND english_reading != '' ORDER BY RAND\(\) LIMIT \?`).
					WithArgs(2).
					WillReturnRows(rows1)

				// Second query for wrong options (filters by character, not ID)
				rows2 := sqlmock.NewRows([]string{"display_character"}).
					AddRow("う").
					AddRow("え").
					AddRow("お").
					AddRow("か")
				mock.ExpectQuery(`SELECT hiragana AS display_character FROM characters WHERE hiragana NOT IN \(\?,\?\) AND hiragana IS NOT NULL AND hiragana != '' ORDER BY RAND\(\)`).
					WithArgs("あ", "い").
					WillReturnRows(rows2)
			},
			expectedError: false,
			expectedCount: 2,
		},
		{
			name:         "invalid alphabet type",
			alphabetType: "invalid",
			locale:       models.LocaleEnglish,
			count:        2,
			setupMock: func(mock sqlmock.Sqlmock) {
				// No query expected
			},
			expectedError: true,
			expectedCount: 0,
		},
		{
			name:         "invalid locale",
			alphabetType: models.AlphabetTypeHiragana,
			locale:       "invalid",
			count:        2,
			setupMock: func(mock sqlmock.Sqlmock) {
				// No query expected
			},
			expectedError: true,
			expectedCount: 0,
		},
		{
			name:         "database query error on correct chars",
			alphabetType: models.AlphabetTypeHiragana,
			locale:       models.LocaleEnglish,
			count:        2,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT id, hiragana AS display_character, english_reading AS reading FROM characters WHERE hiragana IS NOT NULL AND english_reading != '' ORDER BY RAND\(\) LIMIT \?`).
					WithArgs(2).
					WillReturnError(errors.New("database error"))
			},
			expectedError: true,
			expectedCount: 0,
		},
		{
			name:         "database query error on wrong options",
			alphabetType: models.AlphabetTypeHiragana,
			locale:       models.LocaleEnglish,
			count:        1,
			setupMock: func(mock sqlmock.Sqlmock) {
				rows1 := sqlmock.NewRows([]string{"id", "display_character", "reading"}).
					AddRow(1, "あ", "a")
				mock.ExpectQuery(`SELECT id, hiragana AS display_character, english_reading AS reading FROM characters WHERE hiragana IS NOT NULL AND english_reading != '' ORDER BY RAND\(\) LIMIT \?`).
					WithArgs(1).
					WillReturnRows(rows1)

				mock.ExpectQuery(`SELECT hiragana AS display_character FROM characters WHERE hiragana NOT IN \(\?\) AND hiragana IS NOT NULL AND hiragana != '' ORDER BY RAND\(\)`).
					WithArgs("あ").
					WillReturnError(errors.New("database error"))
			},
			expectedError: true,
			expectedCount: 0,
		},
		{
			name:         "insufficient wrong options",
			alphabetType: models.AlphabetTypeHiragana,
			locale:       models.LocaleEnglish,
			count:        1,
			setupMock: func(mock sqlmock.Sqlmock) {
				rows1 := sqlmock.NewRows([]string{"id", "display_character", "reading"}).
					AddRow(1, "あ", "a")
				mock.ExpectQuery(`SELECT id, hiragana AS display_character, english_reading AS reading FROM characters WHERE hiragana IS NOT NULL AND english_reading != '' ORDER BY RAND\(\) LIMIT \?`).
					WithArgs(1).
					WillReturnRows(rows1)

				rows2 := sqlmock.NewRows([]string{"display_character"}).
					AddRow("い") // Only one wrong option, need 2
				mock.ExpectQuery(`SELECT hiragana AS display_character FROM characters WHERE hiragana NOT IN \(\?\) AND hiragana IS NOT NULL AND hiragana != '' ORDER BY RAND\(\)`).
					WithArgs("あ").
					WillReturnRows(rows2)
			},
			expectedError: true,
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock, cleanup := setupTestRepository(t)
			defer cleanup()

			tt.setupMock(mock)

			result, err := repo.GetRandomForReadingTest(context.Background(), tt.alphabetType, tt.locale, tt.count)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				// Empty slice is valid, not nil
				if tt.expectedCount == 0 {
					assert.NotNil(t, result)
					assert.Len(t, result, 0)
				} else {
					assert.NotNil(t, result)
					assert.Len(t, result, tt.expectedCount)
					// Verify each item has correct structure
					for _, item := range result {
						assert.NotEmpty(t, item.CorrectChar)
						assert.NotEmpty(t, item.Reading)
						assert.Len(t, item.WrongOptions, 2)
					}
				}
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestCharactersRepository_GetRandomForWritingTest(t *testing.T) {
	tests := []struct {
		name          string
		alphabetType  models.AlphabetType
		locale        models.Locale
		count         int
		setupMock     func(sqlmock.Sqlmock)
		expectedError bool
		expectedCount int
	}{
		{
			name:         "success katakana russian",
			alphabetType: models.AlphabetTypeKatakana,
			locale:       models.LocaleRussian,
			count:        3,
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "display_character", "reading"}).
					AddRow(1, "ア", "а").
					AddRow(2, "イ", "и").
					AddRow(3, "ウ", "у")
				mock.ExpectQuery(`SELECT id, katakana AS display_character, russian_reading AS reading FROM characters WHERE katakana IS NOT NULL AND russian_reading != '' ORDER BY RAND\(\) LIMIT \?`).
					WithArgs(3).
					WillReturnRows(rows)
			},
			expectedError: false,
			expectedCount: 3,
		},
		{
			name:         "invalid alphabet type",
			alphabetType: "invalid",
			locale:       models.LocaleEnglish,
			count:        2,
			setupMock: func(mock sqlmock.Sqlmock) {
				// No query expected
			},
			expectedError: true,
			expectedCount: 0,
		},
		{
			name:         "invalid locale",
			alphabetType: models.AlphabetTypeHiragana,
			locale:       "invalid",
			count:        2,
			setupMock: func(mock sqlmock.Sqlmock) {
				// No query expected
			},
			expectedError: true,
			expectedCount: 0,
		},
		{
			name:         "database query error",
			alphabetType: models.AlphabetTypeHiragana,
			locale:       models.LocaleEnglish,
			count:        2,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT id, hiragana AS display_character, english_reading AS reading FROM characters WHERE hiragana IS NOT NULL AND english_reading != '' ORDER BY RAND\(\) LIMIT \?`).
					WithArgs(2).
					WillReturnError(errors.New("database error"))
			},
			expectedError: true,
			expectedCount: 0,
		},
		{
			name:         "scan error",
			alphabetType: models.AlphabetTypeHiragana,
			locale:       models.LocaleEnglish,
			count:        1,
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "display_character", "reading"}).
					AddRow("invalid", "あ", "a")
				mock.ExpectQuery(`SELECT id, hiragana AS display_character, english_reading AS reading FROM characters WHERE hiragana IS NOT NULL AND english_reading != '' ORDER BY RAND\(\) LIMIT \?`).
					WithArgs(1).
					WillReturnRows(rows)
			},
			expectedError: true,
			expectedCount: 0,
		},
		{
			name:         "rows iteration error",
			alphabetType: models.AlphabetTypeHiragana,
			locale:       models.LocaleEnglish,
			count:        2,
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "display_character", "reading"}).
					AddRow(1, "あ", "a").
					RowError(0, errors.New("row error"))
				mock.ExpectQuery(`SELECT id, hiragana AS display_character, english_reading AS reading FROM characters WHERE hiragana IS NOT NULL AND english_reading != '' ORDER BY RAND\(\) LIMIT \?`).
					WithArgs(2).
					WillReturnRows(rows)
			},
			expectedError: true,
			expectedCount: 0,
		},
		{
			name:         "empty result",
			alphabetType: models.AlphabetTypeKatakana,
			locale:       models.LocaleEnglish,
			count:        5,
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "display_character", "reading"})
				mock.ExpectQuery(`SELECT id, katakana AS display_character, english_reading AS reading FROM characters WHERE katakana IS NOT NULL AND english_reading != '' ORDER BY RAND\(\) LIMIT \?`).
					WithArgs(5).
					WillReturnRows(rows)
			},
			expectedError: false,
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock, cleanup := setupTestRepository(t)
			defer cleanup()

			tt.setupMock(mock)

			result, err := repo.GetRandomForWritingTest(context.Background(), tt.alphabetType, tt.locale, tt.count)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				// Empty/nil slice is valid for empty results
				if tt.expectedCount == 0 {
					// In Go, nil slice and empty slice both have length 0
					assert.Len(t, result, 0)
				} else {
					assert.NotNil(t, result)
					assert.Len(t, result, tt.expectedCount)
				}
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestIsVowel(t *testing.T) {
	tests := []struct {
		name     string
		char     string
		expected bool
	}{
		{
			name:     "vowel a",
			char:     "a",
			expected: true,
		},
		{
			name:     "vowel i",
			char:     "i",
			expected: true,
		},
		{
			name:     "vowel u",
			char:     "u",
			expected: true,
		},
		{
			name:     "vowel e",
			char:     "e",
			expected: true,
		},
		{
			name:     "vowel o",
			char:     "o",
			expected: true,
		},
		{
			name:     "consonant k",
			char:     "k",
			expected: false,
		},
		{
			name:     "empty string",
			char:     "",
			expected: false,
		},
		{
			name:     "invalid character",
			char:     "x",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isVowel(tt.char)
			assert.Equal(t, tt.expected, result)
		})
	}
}
