package repositories

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/japanesestudent/learn-service/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupWordTestRepository creates a word repository with a mock database
func setupWordTestRepository(t *testing.T) (*wordRepository, sqlmock.Sqlmock, func()) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)

	repo := NewWordRepository(db)

	cleanup := func() {
		db.Close()
	}

	return repo, mock, cleanup
}

func TestNewWordRepository(t *testing.T) {
	db := &sql.DB{}

	repo := NewWordRepository(db)

	assert.NotNil(t, repo)
	assert.Equal(t, db, repo.db)
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

func TestWordRepository_GetAllForAdmin(t *testing.T) {
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
			count:  10,
			search: "",
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "word", "phonetic_clues", "english_translation"}).
					AddRow(1, "水", "みず", "water").
					AddRow(2, "火", "ひ", "fire")
				mock.ExpectQuery(`SELECT id, word, phonetic_clues, english_translation FROM words ORDER BY english_translation LIMIT \? OFFSET \?`).
					WithArgs(10, 0).
					WillReturnRows(rows)
			},
			expectedError: false,
			expectedCount: 2,
		},
		{
			name:   "success with search",
			page:   1,
			count:  10,
			search: "water",
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "word", "phonetic_clues", "english_translation"}).
					AddRow(1, "水", "みず", "water")
				mock.ExpectQuery(`SELECT id, word, phonetic_clues, english_translation FROM words WHERE word LIKE \? OR phonetic_clues LIKE \? OR english_translation LIKE \? OR russian_translation LIKE \? OR german_translation LIKE \? ORDER BY english_translation LIMIT \? OFFSET \?`).
					WithArgs("%water%", "%water%", "%water%", "%water%", "%water%", 10, 0).
					WillReturnRows(rows)
			},
			expectedError: false,
			expectedCount: 1,
		},
		{
			name:   "success with pagination",
			page:   2,
			count:  5,
			search: "",
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "word", "phonetic_clues", "english_translation"}).
					AddRow(6, "木", "き", "tree")
				mock.ExpectQuery(`SELECT id, word, phonetic_clues, english_translation FROM words ORDER BY english_translation LIMIT \? OFFSET \?`).
					WithArgs(5, 5).
					WillReturnRows(rows)
			},
			expectedError: false,
			expectedCount: 1,
		},
		{
			name:   "database query error",
			page:   1,
			count:  10,
			search: "",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT id, word, phonetic_clues, english_translation FROM words ORDER BY english_translation LIMIT \? OFFSET \?`).
					WithArgs(10, 0).
					WillReturnError(errors.New("database error"))
			},
			expectedError: true,
			expectedCount: 0,
		},
		{
			name:   "scan error",
			page:   1,
			count:  10,
			search: "",
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "word", "phonetic_clues", "english_translation"}).
					AddRow("invalid", "水", "みず", "water")
				mock.ExpectQuery(`SELECT id, word, phonetic_clues, english_translation FROM words ORDER BY english_translation LIMIT \? OFFSET \?`).
					WithArgs(10, 0).
					WillReturnRows(rows)
			},
			expectedError: true,
			expectedCount: 0,
		},
		{
			name:   "rows iteration error",
			page:   1,
			count:  10,
			search: "",
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "word", "phonetic_clues", "english_translation"}).
					AddRow(1, "水", "みず", "water").
					RowError(0, errors.New("row error"))
				mock.ExpectQuery(`SELECT id, word, phonetic_clues, english_translation FROM words ORDER BY english_translation LIMIT \? OFFSET \?`).
					WithArgs(10, 0).
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

			result, err := repo.GetAllForAdmin(context.Background(), tt.page, tt.count, tt.search)

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

func TestWordRepository_GetByIDAdmin(t *testing.T) {
	tests := []struct {
		name          string
		id            int
		setupMock     func(sqlmock.Sqlmock)
		expectedError bool
		expectedWord  *models.Word
	}{
		{
			name: "success",
			id:   1,
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{
					"id", "word", "phonetic_clues", "russian_translation", "english_translation", "german_translation",
					"example", "example_russian_translation", "example_english_translation", "example_german_translation",
					"easy_period", "normal_period", "hard_period", "extra_hard_period",
				}).
					AddRow(1, "水", "みず", "вода", "water", "Wasser", "水を飲む", "пить воду", "drink water", "Wasser trinken", 1, 3, 7, 14)
				mock.ExpectQuery(`SELECT id, word, phonetic_clues, russian_translation, english_translation, german_translation, example, example_russian_translation, example_english_translation, example_german_translation, easy_period, normal_period, hard_period, extra_hard_period FROM words WHERE id = \? LIMIT 1`).
					WithArgs(1).
					WillReturnRows(rows)
			},
			expectedError: false,
			expectedWord: &models.Word{
				ID:                        1,
				Word:                      "水",
				PhoneticClues:             "みず",
				RussianTranslation:        "вода",
				EnglishTranslation:        "water",
				GermanTranslation:         "Wasser",
				Example:                   "水を飲む",
				ExampleRussianTranslation: "пить воду",
				ExampleEnglishTranslation: "drink water",
				ExampleGermanTranslation:  "Wasser trinken",
				EasyPeriod:                1,
				NormalPeriod:              3,
				HardPeriod:                7,
				ExtraHardPeriod:           14,
			},
		},
		{
			name: "not found",
			id:   999,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT id, word, phonetic_clues, russian_translation, english_translation, german_translation, example, example_russian_translation, example_english_translation, example_german_translation, easy_period, normal_period, hard_period, extra_hard_period FROM words WHERE id = \? LIMIT 1`).
					WithArgs(999).
					WillReturnError(sql.ErrNoRows)
			},
			expectedError: true,
			expectedWord:  nil,
		},
		{
			name: "database error",
			id:   1,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT id, word, phonetic_clues, russian_translation, english_translation, german_translation, example, example_russian_translation, example_english_translation, example_german_translation, easy_period, normal_period, hard_period, extra_hard_period FROM words WHERE id = \? LIMIT 1`).
					WithArgs(1).
					WillReturnError(errors.New("database error"))
			},
			expectedError: true,
			expectedWord:  nil,
		},
		{
			name: "scan error",
			id:   1,
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{
					"id", "word", "phonetic_clues", "russian_translation", "english_translation", "german_translation",
					"example", "example_russian_translation", "example_english_translation", "example_german_translation",
					"easy_period", "normal_period", "hard_period", "extra_hard_period",
				}).
					AddRow("invalid", "水", "みず", "вода", "water", "Wasser", "水を飲む", "пить воду", "drink water", "Wasser trinken", 1, 3, 7, 14)
				mock.ExpectQuery(`SELECT id, word, phonetic_clues, russian_translation, english_translation, german_translation, example, example_russian_translation, example_english_translation, example_german_translation, easy_period, normal_period, hard_period, extra_hard_period FROM words WHERE id = \? LIMIT 1`).
					WithArgs(1).
					WillReturnRows(rows)
			},
			expectedError: true,
			expectedWord:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock, cleanup := setupWordTestRepository(t)
			defer cleanup()

			tt.setupMock(mock)

			result, err := repo.GetByIDAdmin(context.Background(), tt.id)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				if tt.expectedWord != nil {
					assert.Equal(t, tt.expectedWord.ID, result.ID)
					assert.Equal(t, tt.expectedWord.Word, result.Word)
					assert.Equal(t, tt.expectedWord.PhoneticClues, result.PhoneticClues)
				}
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestWordRepository_ExistsByWord(t *testing.T) {
	tests := []struct {
		name           string
		word           string
		setupMock      func(sqlmock.Sqlmock)
		expectedError  bool
		expectedExists bool
	}{
		{
			name: "word exists",
			word: "水",
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"exists"}).AddRow(true)
				mock.ExpectQuery(`SELECT EXISTS\(SELECT \* FROM words WHERE word = \?\)`).
					WithArgs("水").
					WillReturnRows(rows)
			},
			expectedError:  false,
			expectedExists: true,
		},
		{
			name: "word does not exist",
			word: "nonexistent",
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"exists"}).AddRow(false)
				mock.ExpectQuery(`SELECT EXISTS\(SELECT \* FROM words WHERE word = \?\)`).
					WithArgs("nonexistent").
					WillReturnRows(rows)
			},
			expectedError:  false,
			expectedExists: false,
		},
		{
			name: "database error",
			word: "水",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT EXISTS\(SELECT \* FROM words WHERE word = \?\)`).
					WithArgs("水").
					WillReturnError(errors.New("database error"))
			},
			expectedError:  true,
			expectedExists: false,
		},
		{
			name: "scan error",
			word: "水",
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"exists"}).AddRow("invalid")
				mock.ExpectQuery(`SELECT EXISTS\(SELECT \* FROM words WHERE word = \?\)`).
					WithArgs("水").
					WillReturnRows(rows)
			},
			expectedError:  true,
			expectedExists: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock, cleanup := setupWordTestRepository(t)
			defer cleanup()

			tt.setupMock(mock)

			exists, err := repo.ExistsByWord(context.Background(), tt.word)

			if tt.expectedError {
				assert.Error(t, err)
				assert.False(t, exists)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedExists, exists)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestWordRepository_Create(t *testing.T) {
	tests := []struct {
		name          string
		word          *models.Word
		setupMock     func(sqlmock.Sqlmock)
		expectedError bool
		expectedID    int
	}{
		{
			name: "success",
			word: &models.Word{
				Word:                      "水",
				PhoneticClues:             "みず",
				RussianTranslation:        "вода",
				EnglishTranslation:        "water",
				GermanTranslation:         "Wasser",
				Example:                   "水を飲む",
				ExampleRussianTranslation: "пить воду",
				ExampleEnglishTranslation: "drink water",
				ExampleGermanTranslation:  "Wasser trinken",
				EasyPeriod:                1,
				NormalPeriod:              3,
				HardPeriod:                7,
				ExtraHardPeriod:           14,
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`INSERT INTO words`).
					WithArgs("水", "みず", "вода", "water", "Wasser", "水を飲む", "пить воду", "drink water", "Wasser trinken", 1, 3, 7, 14).
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			expectedError: false,
			expectedID:    1,
		},
		{
			name: "database error",
			word: &models.Word{
				Word:                      "水",
				PhoneticClues:             "みず",
				RussianTranslation:        "вода",
				EnglishTranslation:        "water",
				GermanTranslation:         "Wasser",
				Example:                   "水を飲む",
				ExampleRussianTranslation: "пить воду",
				ExampleEnglishTranslation: "drink water",
				ExampleGermanTranslation:  "Wasser trinken",
				EasyPeriod:                1,
				NormalPeriod:              3,
				HardPeriod:                7,
				ExtraHardPeriod:           14,
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`INSERT INTO words`).
					WithArgs("水", "みず", "вода", "water", "Wasser", "水を飲む", "пить воду", "drink water", "Wasser trinken", 1, 3, 7, 14).
					WillReturnError(errors.New("database error"))
			},
			expectedError: true,
			expectedID:    0,
		},
		{
			name: "error getting last insert id",
			word: &models.Word{
				Word:                      "水",
				PhoneticClues:             "みず",
				RussianTranslation:        "вода",
				EnglishTranslation:        "water",
				GermanTranslation:         "Wasser",
				Example:                   "水を飲む",
				ExampleRussianTranslation: "пить воду",
				ExampleEnglishTranslation: "drink water",
				ExampleGermanTranslation:  "Wasser trinken",
				EasyPeriod:                1,
				NormalPeriod:              3,
				HardPeriod:                7,
				ExtraHardPeriod:           14,
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`INSERT INTO words`).
					WithArgs("水", "みず", "вода", "water", "Wasser", "水を飲む", "пить воду", "drink water", "Wasser trinken", 1, 3, 7, 14).
					WillReturnResult(sqlmock.NewErrorResult(errors.New("last insert id error")))
			},
			expectedError: true,
			expectedID:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock, cleanup := setupWordTestRepository(t)
			defer cleanup()

			tt.setupMock(mock)

			err := repo.Create(context.Background(), tt.word)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedID, tt.word.ID)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestWordRepository_ExistsByClues(t *testing.T) {
	tests := []struct {
		name           string
		clues          string
		setupMock      func(sqlmock.Sqlmock)
		expectedError  bool
		expectedExists bool
	}{
		{
			name:  "clues exists",
			clues: "みず",
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"exists"}).AddRow(true)
				mock.ExpectQuery(`SELECT EXISTS\(SELECT \* FROM words WHERE phonetic_clues = \?\)`).
					WithArgs("みず").
					WillReturnRows(rows)
			},
			expectedError:  false,
			expectedExists: true,
		},
		{
			name:  "clues does not exist",
			clues: "nonexistent",
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"exists"}).AddRow(false)
				mock.ExpectQuery(`SELECT EXISTS\(SELECT \* FROM words WHERE phonetic_clues = \?\)`).
					WithArgs("nonexistent").
					WillReturnRows(rows)
			},
			expectedError:  false,
			expectedExists: false,
		},
		{
			name:  "database error",
			clues: "みず",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT EXISTS\(SELECT \* FROM words WHERE phonetic_clues = \?\)`).
					WithArgs("みず").
					WillReturnError(errors.New("database error"))
			},
			expectedError:  true,
			expectedExists: false,
		},
		{
			name:  "scan error",
			clues: "みず",
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"exists"}).AddRow("invalid")
				mock.ExpectQuery(`SELECT EXISTS\(SELECT \* FROM words WHERE phonetic_clues = \?\)`).
					WithArgs("みず").
					WillReturnRows(rows)
			},
			expectedError:  true,
			expectedExists: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock, cleanup := setupWordTestRepository(t)
			defer cleanup()

			tt.setupMock(mock)

			exists, err := repo.ExistsByClues(context.Background(), tt.clues)

			if tt.expectedError {
				assert.Error(t, err)
				assert.False(t, exists)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedExists, exists)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestWordRepository_Update(t *testing.T) {
	tests := []struct {
		name          string
		id            int
		word          *models.Word
		setupMock     func(sqlmock.Sqlmock)
		expectedError bool
	}{
		{
			name: "success - update all fields",
			id:   1,
			word: &models.Word{
				Word:                      "火",
				PhoneticClues:             "ひ",
				RussianTranslation:        "огонь",
				EnglishTranslation:        "fire",
				GermanTranslation:         "Feuer",
				Example:                   "火をつける",
				ExampleRussianTranslation: "зажечь огонь",
				ExampleEnglishTranslation: "light a fire",
				ExampleGermanTranslation:  "Feuer anzünden",
				EasyPeriod:                2,
				NormalPeriod:              4,
				HardPeriod:                8,
				ExtraHardPeriod:           16,
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`UPDATE words SET word = \?, phonetic_clues = \?, russian_translation = \?, english_translation = \?, german_translation = \?, example = \?, example_russian_translation = \?, example_english_translation = \?, example_german_translation = \?, easy_period = \?, normal_period = \?, hard_period = \?, extra_hard_period = \? WHERE id = \?`).
					WithArgs("火", "ひ", "огонь", "fire", "Feuer", "火をつける", "зажечь огонь", "light a fire", "Feuer anzünden", 2, 4, 8, 16, 1).
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
			expectedError: false,
		},
		{
			name: "success - partial update",
			id:   1,
			word: &models.Word{
				Word:               "火",
				EnglishTranslation: "fire",
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`UPDATE words SET word = \?, english_translation = \? WHERE id = \?`).
					WithArgs("火", "fire", 1).
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
			expectedError: false,
		},
		{
			name: "no fields to update",
			id:   1,
			word: &models.Word{},
			setupMock: func(mock sqlmock.Sqlmock) {
				// No query expected
			},
			expectedError: true,
		},
		{
			name: "word not found",
			id:   999,
			word: &models.Word{
				Word: "火",
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`UPDATE words SET word = \? WHERE id = \?`).
					WithArgs("火", 999).
					WillReturnResult(sqlmock.NewResult(0, 0))
			},
			expectedError: true,
		},
		{
			name: "database error",
			id:   1,
			word: &models.Word{
				Word: "火",
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`UPDATE words SET word = \? WHERE id = \?`).
					WithArgs("火", 1).
					WillReturnError(errors.New("database error"))
			},
			expectedError: true,
		},
		{
			name: "error getting rows affected",
			id:   1,
			word: &models.Word{
				Word: "火",
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`UPDATE words SET word = \? WHERE id = \?`).
					WithArgs("火", 1).
					WillReturnResult(sqlmock.NewErrorResult(errors.New("rows affected error")))
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock, cleanup := setupWordTestRepository(t)
			defer cleanup()

			tt.setupMock(mock)

			err := repo.Update(context.Background(), tt.id, tt.word)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestWordRepository_Delete(t *testing.T) {
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
				mock.ExpectExec(`DELETE FROM words WHERE id = \?`).
					WithArgs(1).
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
			expectedError: false,
		},
		{
			name: "word not found",
			id:   999,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`DELETE FROM words WHERE id = \?`).
					WithArgs(999).
					WillReturnResult(sqlmock.NewResult(0, 0))
			},
			expectedError: true,
		},
		{
			name: "database error",
			id:   1,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`DELETE FROM words WHERE id = \?`).
					WithArgs(1).
					WillReturnError(errors.New("database error"))
			},
			expectedError: true,
		},
		{
			name: "error getting rows affected",
			id:   1,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`DELETE FROM words WHERE id = \?`).
					WithArgs(1).
					WillReturnResult(sqlmock.NewErrorResult(errors.New("rows affected error")))
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock, cleanup := setupWordTestRepository(t)
			defer cleanup()

			tt.setupMock(mock)

			err := repo.Delete(context.Background(), tt.id)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
