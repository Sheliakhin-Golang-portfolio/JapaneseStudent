package repositories

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// setupWordTestRepository creates a word repository with a mock database
func setupWordTestRepository(t *testing.T) (*wordRepository, sqlmock.Sqlmock, func()) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)

	logger, err := zap.NewDevelopment()
	require.NoError(t, err)

	repo := NewWordRepository(db, logger)

	cleanup := func() {
		db.Close()
	}

	return repo, mock, cleanup
}

func TestNewWordRepository(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	db := &sql.DB{}

	repo := NewWordRepository(db, logger)

	assert.NotNil(t, repo)
	assert.Equal(t, db, repo.db)
	assert.Equal(t, logger, repo.logger)
}

func TestWordRepository_GetByIDs(t *testing.T) {
	tests := []struct {
		name                    string
		wordIds                 []int
		translationField        string
		exampleTranslationField string
		setupMock               func(sqlmock.Sqlmock)
		expectedError           bool
		expectedCount           int
	}{
		{
			name:                    "success with multiple IDs",
			wordIds:                 []int{1, 2, 3},
			translationField:        "english_translation",
			exampleTranslationField: "example_english_translation",
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{
					"id", "word", "phonetic_clues", "translation", "example",
					"example_translation", "easy_period", "normal_period", "hard_period", "extra_hard_period",
				}).
					AddRow(1, "水", "みず", "water", "水を飲む", "drink water", 1, 3, 7, 14).
					AddRow(2, "火", "ひ", "fire", "火をつける", "light a fire", 1, 3, 7, 14).
					AddRow(3, "風", "かぜ", "wind", "風が吹く", "wind blows", 1, 3, 7, 14)
				mock.ExpectQuery(`SELECT id, word, phonetic_clues, english_translation as translation, example, example_english_translation as example_translation, easy_period, normal_period, hard_period, extra_hard_period FROM words WHERE id IN \(\?,\?,\?\)`).
					WithArgs(1, 2, 3).
					WillReturnRows(rows)
			},
			expectedError: false,
			expectedCount: 3,
		},
		{
			name:                    "success with single ID",
			wordIds:                 []int{1},
			translationField:        "russian_translation",
			exampleTranslationField: "example_russian_translation",
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{
					"id", "word", "phonetic_clues", "translation", "example",
					"example_translation", "easy_period", "normal_period", "hard_period", "extra_hard_period",
				}).
					AddRow(1, "水", "みず", "вода", "水を飲む", "пить воду", 1, 3, 7, 14)
				mock.ExpectQuery(`SELECT id, word, phonetic_clues, russian_translation as translation, example, example_russian_translation as example_translation, easy_period, normal_period, hard_period, extra_hard_period FROM words WHERE id IN \(\?\)`).
					WithArgs(1).
					WillReturnRows(rows)
			},
			expectedError: false,
			expectedCount: 1,
		},
		{
			name:                    "empty wordIds slice",
			wordIds:                 []int{},
			translationField:        "english_translation",
			exampleTranslationField: "example_english_translation",
			setupMock:               func(mock sqlmock.Sqlmock) {},
			expectedError:           false,
			expectedCount:           0,
		},
		{
			name:                    "database query error",
			wordIds:                 []int{1, 2},
			translationField:        "english_translation",
			exampleTranslationField: "example_english_translation",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT id, word, phonetic_clues, english_translation as translation, example, example_english_translation as example_translation, easy_period, normal_period, hard_period, extra_hard_period FROM words WHERE id IN \(\?,\?\)`).
					WithArgs(1, 2).
					WillReturnError(errors.New("database error"))
			},
			expectedError: true,
			expectedCount: 0,
		},
		{
			name:                    "scan error",
			wordIds:                 []int{1},
			translationField:        "english_translation",
			exampleTranslationField: "example_english_translation",
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{
					"id", "word", "phonetic_clues", "translation", "example",
					"example_translation", "easy_period", "normal_period", "hard_period", "extra_hard_period",
				}).
					AddRow("invalid", "水", "みず", "water", "水を飲む", "drink water", 1, 3, 7, 14)
				mock.ExpectQuery(`SELECT id, word, phonetic_clues, english_translation as translation, example, example_english_translation as example_translation, easy_period, normal_period, hard_period, extra_hard_period FROM words WHERE id IN \(\?\)`).
					WithArgs(1).
					WillReturnRows(rows)
			},
			expectedError: true,
			expectedCount: 0,
		},
		{
			name:                    "rows iteration error",
			wordIds:                 []int{1, 2},
			translationField:        "english_translation",
			exampleTranslationField: "example_english_translation",
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{
					"id", "word", "phonetic_clues", "translation", "example",
					"example_translation", "easy_period", "normal_period", "hard_period", "extra_hard_period",
				}).
					AddRow(1, "水", "みず", "water", "水を飲む", "drink water", 1, 3, 7, 14).
					RowError(0, errors.New("row error"))
				mock.ExpectQuery(`SELECT id, word, phonetic_clues, english_translation as translation, example, example_english_translation as example_translation, easy_period, normal_period, hard_period, extra_hard_period FROM words WHERE id IN \(\?,\?\)`).
					WithArgs(1, 2).
					WillReturnRows(rows)
			},
			expectedError: true,
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock, cleanup := setupWordTestRepository(t)
			defer cleanup()

			tt.setupMock(mock)

			result, err := repo.GetByIDs(context.Background(), tt.wordIds, tt.translationField, tt.exampleTranslationField)

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

func TestWordRepository_GetExcludingIDs(t *testing.T) {
	tests := []struct {
		name                    string
		excludeIds              []int
		limit                   int
		translationField        string
		exampleTranslationField string
		setupMock               func(sqlmock.Sqlmock)
		expectedError           bool
		expectedCount           int
	}{
		{
			name:                    "success with exclusion list",
			excludeIds:              []int{1, 2, 3},
			limit:                   5,
			translationField:        "english_translation",
			exampleTranslationField: "example_english_translation",
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{
					"id", "word", "phonetic_clues", "translation", "example",
					"example_translation", "easy_period", "normal_period", "hard_period", "extra_hard_period",
				}).
					AddRow(4, "木", "き", "tree", "木を植える", "plant a tree", 1, 3, 7, 14).
					AddRow(5, "土", "つち", "earth", "土を耕す", "till the earth", 1, 3, 7, 14)
				mock.ExpectQuery(`SELECT id, word, phonetic_clues, english_translation as translation, example, example_english_translation as example_translation, easy_period, normal_period, hard_period, extra_hard_period FROM words WHERE id NOT IN \(\?,\?,\?\) ORDER BY \(EXISTS \(SELECT 1 FROM dictionary_history WHERE word_id = words\.id AND user_id = \?\)\), RAND\(\) LIMIT \?`).
					WithArgs(1, 2, 3, 1, 5).
					WillReturnRows(rows)
			},
			expectedError: false,
			expectedCount: 2,
		},
		{
			name:                    "success with empty exclusion list",
			excludeIds:              []int{},
			limit:                   3,
			translationField:        "russian_translation",
			exampleTranslationField: "example_russian_translation",
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{
					"id", "word", "phonetic_clues", "translation", "example",
					"example_translation", "easy_period", "normal_period", "hard_period", "extra_hard_period",
				}).
					AddRow(1, "水", "みず", "вода", "水を飲む", "пить воду", 1, 3, 7, 14).
					AddRow(2, "火", "ひ", "огонь", "火をつける", "зажечь огонь", 1, 3, 7, 14).
					AddRow(3, "風", "かぜ", "ветер", "風が吹く", "дует ветер", 1, 3, 7, 14)
				mock.ExpectQuery(`SELECT id, word, phonetic_clues, russian_translation as translation, example, example_russian_translation as example_translation, easy_period, normal_period, hard_period, extra_hard_period FROM words ORDER BY \(EXISTS \(SELECT 1 FROM dictionary_history WHERE word_id = words\.id AND user_id = \?\)\), RAND\(\) LIMIT \?`).
					WithArgs(1, 3).
					WillReturnRows(rows)
			},
			expectedError: false,
			expectedCount: 3,
		},
		{
			name:                    "database query error",
			excludeIds:              []int{1},
			limit:                   5,
			translationField:        "english_translation",
			exampleTranslationField: "example_english_translation",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT id, word, phonetic_clues, english_translation as translation, example, example_english_translation as example_translation, easy_period, normal_period, hard_period, extra_hard_period FROM words WHERE id NOT IN \(\?\) ORDER BY \(EXISTS \(SELECT 1 FROM dictionary_history WHERE word_id = words\.id AND user_id = \?\)\), RAND\(\) LIMIT \?`).
					WithArgs(1, 1, 5).
					WillReturnError(errors.New("database error"))
			},
			expectedError: true,
			expectedCount: 0,
		},
		{
			name:                    "scan error",
			excludeIds:              []int{1},
			limit:                   2,
			translationField:        "english_translation",
			exampleTranslationField: "example_english_translation",
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{
					"id", "word", "phonetic_clues", "translation", "example",
					"example_translation", "easy_period", "normal_period", "hard_period", "extra_hard_period",
				}).
					AddRow("invalid", "水", "みず", "water", "水を飲む", "drink water", 1, 3, 7, 14)
				mock.ExpectQuery(`SELECT id, word, phonetic_clues, english_translation as translation, example, example_english_translation as example_translation, easy_period, normal_period, hard_period, extra_hard_period FROM words WHERE id NOT IN \(\?\) ORDER BY \(EXISTS \(SELECT 1 FROM dictionary_history WHERE word_id = words\.id AND user_id = \?\)\), RAND\(\) LIMIT \?`).
					WithArgs(1, 1, 2).
					WillReturnRows(rows)
			},
			expectedError: true,
			expectedCount: 0,
		},
		{
			name:                    "rows iteration error",
			excludeIds:              []int{1},
			limit:                   2,
			translationField:        "english_translation",
			exampleTranslationField: "example_english_translation",
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{
					"id", "word", "phonetic_clues", "translation", "example",
					"example_translation", "easy_period", "normal_period", "hard_period", "extra_hard_period",
				}).
					AddRow(2, "火", "ひ", "fire", "火をつける", "light a fire", 1, 3, 7, 14).
					RowError(0, errors.New("row error"))
				mock.ExpectQuery(`SELECT id, word, phonetic_clues, english_translation as translation, example, example_english_translation as example_translation, easy_period, normal_period, hard_period, extra_hard_period FROM words WHERE id NOT IN \(\?\) ORDER BY \(EXISTS \(SELECT 1 FROM dictionary_history WHERE word_id = words\.id AND user_id = \?\)\), RAND\(\) LIMIT \?`).
					WithArgs(1, 1, 2).
					WillReturnRows(rows)
			},
			expectedError: true,
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock, cleanup := setupWordTestRepository(t)
			defer cleanup()

			tt.setupMock(mock)

			result, err := repo.GetExcludingIDs(context.Background(), 1, tt.excludeIds, tt.limit, tt.translationField, tt.exampleTranslationField)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.LessOrEqual(t, len(result), tt.expectedCount)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestWordRepository_ValidateWordIDs(t *testing.T) {
	tests := []struct {
		name          string
		wordIds       []int
		setupMock     func(sqlmock.Sqlmock)
		expectedError bool
		expectedValid bool
	}{
		{
			name:    "success - all IDs exist",
			wordIds: []int{1, 2, 3},
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"count"}).AddRow(3)
				mock.ExpectQuery(`SELECT COUNT\(\*\) as count FROM words WHERE id IN \(\?,\?,\?\)`).
					WithArgs(1, 2, 3).
					WillReturnRows(rows)
			},
			expectedError: false,
			expectedValid: true,
		},
		{
			name:    "success - some IDs missing",
			wordIds: []int{1, 2, 3},
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"count"}).AddRow(2)
				mock.ExpectQuery(`SELECT COUNT\(\*\) as count FROM words WHERE id IN \(\?,\?,\?\)`).
					WithArgs(1, 2, 3).
					WillReturnRows(rows)
			},
			expectedError: false,
			expectedValid: false,
		},
		{
			name:    "empty wordIds slice",
			wordIds: []int{},
			setupMock: func(mock sqlmock.Sqlmock) {
				// No query expected
			},
			expectedError: true,
			expectedValid: false,
		},
		{
			name:    "database error",
			wordIds: []int{1, 2},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT COUNT\(\*\) as count FROM words WHERE id IN \(\?,\?\)`).
					WithArgs(1, 2).
					WillReturnError(errors.New("database error"))
			},
			expectedError: true,
			expectedValid: false,
		},
		{
			name:    "scan error",
			wordIds: []int{1},
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"count"}).AddRow("invalid")
				mock.ExpectQuery(`SELECT COUNT\(\*\) as count FROM words WHERE id IN \(\?\)`).
					WithArgs(1).
					WillReturnRows(rows)
			},
			expectedError: true,
			expectedValid: false,
		},
		{
			name:    "success - single ID exists",
			wordIds: []int{1},
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"count"}).AddRow(1)
				mock.ExpectQuery(`SELECT COUNT\(\*\) as count FROM words WHERE id IN \(\?\)`).
					WithArgs(1).
					WillReturnRows(rows)
			},
			expectedError: false,
			expectedValid: true,
		},
		{
			name:    "success - single ID missing",
			wordIds: []int{999},
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"count"}).AddRow(0)
				mock.ExpectQuery(`SELECT COUNT\(\*\) as count FROM words WHERE id IN \(\?\)`).
					WithArgs(999).
					WillReturnRows(rows)
			},
			expectedError: false,
			expectedValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock, cleanup := setupWordTestRepository(t)
			defer cleanup()

			tt.setupMock(mock)

			valid, err := repo.ValidateWordIDs(context.Background(), tt.wordIds)

			if tt.expectedError {
				assert.Error(t, err)
				assert.False(t, valid)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedValid, valid)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
