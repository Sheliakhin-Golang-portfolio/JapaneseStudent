package integration

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/go-chi/chi/v5"
	_ "github.com/go-sql-driver/mysql"
	"github.com/japanesestudent/backend/internal/config"
	"github.com/japanesestudent/backend/internal/handlers"
	"github.com/japanesestudent/backend/internal/models"
	"github.com/japanesestudent/backend/internal/repositories"
	"github.com/japanesestudent/backend/internal/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

var (
	testDB     *sql.DB
	testRouter chi.Router
	testLogger *zap.Logger
)

// seedTestData inserts test data into the database
func seedTestData(t *testing.T, db *sql.DB) {
	t.Helper()

	// Clear existing data
	_, err := db.Exec("DELETE FROM characters")
	require.NoError(t, err, "Failed to clear test data")

	// Reset AUTO_INCREMENT to start from 1
	_, err = db.Exec("ALTER TABLE characters AUTO_INCREMENT = 1")
	require.NoError(t, err, "Failed to reset AUTO_INCREMENT")

	// Insert full hiragana/katakana alphabet
	query := `
		INSERT INTO characters (consonant, vowel, english_reading, russian_reading, katakana, hiragana) VALUES
		('', 'a', 'a', 'а', 'ア', 'あ'),
		('', 'i', 'i', 'и', 'イ', 'い'),
		('', 'u', 'u', 'у', 'ウ', 'う'),
		('', 'e', 'e', 'э', 'エ', 'え'),
		('', 'o', 'o', 'о', 'オ', 'お'),
		('k', 'a', 'ka', 'ка', 'カ', 'か'),
		('k', 'i', 'ki', 'ки', 'キ', 'き'),
		('k', 'u', 'ku', 'ку', 'ク', 'く'),
		('k', 'e', 'ke', 'кэ', 'ケ', 'け'),
		('k', 'o', 'ko', 'ко', 'コ', 'こ'),
		('s', 'a', 'sa', 'са', 'サ', 'さ'),
		('s', 'i', 'shi', 'си', 'シ', 'し'),
		('s', 'u', 'su', 'су', 'ス', 'す'),
		('s', 'e', 'se', 'сэ', 'セ', 'せ'),
		('s', 'o', 'so', 'со', 'ソ', 'そ'),
		('t', 'a', 'ta', 'та', 'タ', 'た'),
		('t', 'i', 'chi', 'ти', 'チ', 'ち'),
		('t', 'u', 'tsu', 'цу', 'ツ', 'つ'),
		('t', 'e', 'te', 'тэ', 'テ', 'て'),
		('t', 'o', 'to', 'то', 'ト', 'と'),
		('n', 'a', 'na', 'на', 'ナ', 'な'),
		('n', 'i', 'ni', 'ни', 'ニ', 'に'),
		('n', 'u', 'nu', 'ну', 'ヌ', 'ぬ'),
		('n', 'e', 'ne', 'нэ', 'ネ', 'ね'),
		('n', 'o', 'no', 'но', 'ノ', 'の'),
		('h', 'a', 'ha', 'ха', 'ハ', 'は'),
		('h', 'i', 'hi', 'хи', 'ヒ', 'ひ'),
		('h', 'u', 'fu', 'фу', 'フ', 'ふ'),
		('h', 'e', 'he', 'хэ', 'ヘ', 'へ'),
		('h', 'o', 'ho', 'хо', 'ホ', 'ほ'),
		('m', 'a', 'ma', 'ма', 'マ', 'ま'),
		('m', 'i', 'mi', 'ми', 'ミ', 'み'),
		('m', 'u', 'mu', 'му', 'ム', 'む'),
		('m', 'e', 'me', 'мэ', 'メ', 'め'),
		('m', 'o', 'mo', 'мо', 'モ', 'も'),
		('y', 'a', 'ya', 'я', 'ヤ', 'や'),
		('y', 'u', 'yu', 'ю', 'ユ', 'ゆ'),
		('y', 'o', 'yo', 'ё', 'ヨ', 'よ'),
		('r', 'a', 'ra', 'ра', 'ラ', 'ら'),
		('r', 'i', 'ri', 'ри', 'リ', 'り'),
		('r', 'u', 'ru', 'ру', 'ル', 'る'),
		('r', 'e', 're', 'рэ', 'レ', 'れ'),
		('r', 'o', 'ro', 'ро', 'ロ', 'ろ'),
		('w', 'a', 'wa', 'ва', 'ワ', 'わ'),
		('w', 'o', 'wo', 'во', 'ヲ', 'を'),
		('', 'n', 'n', 'н', 'ン', 'ん');
	`

	_, err = db.Exec(query)
	require.NoError(t, err, "Failed to seed test data")
}

// cleanupTestData removes all test data
func cleanupTestData(t *testing.T, db *sql.DB) {
	t.Helper()
	_, err := db.Exec("DELETE FROM characters")
	require.NoError(t, err, "Failed to cleanup test data")
}

// setupTestRouter creates a test router with all handlers
func setupTestRouter(db *sql.DB, logger *zap.Logger) chi.Router {
	repo := repositories.NewCharactersRepository(db, logger)
	svc := services.NewCharactersService(repo, logger)
	charHandler := handlers.NewCharactersHandler(svc, logger)

	r := chi.NewRouter()
	charHandler.RegisterRoutes(r)

	return r
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
	if err != nil {
		panic(fmt.Sprintf("Failed to load test config: %v", err))
	}
	dsn := cfg.DSN()
	if dsn == "" {
		// Default test database connection
		dsn = "root:password@tcp(localhost:3306)/japanesestudent_test?parseTime=true&charset=utf8mb4"
	}

	testDB, err = sql.Open("mysql", dsn)
	if err != nil {
		panic(fmt.Sprintf("Failed to connect to test database: %v", err))
	}

	// Test connection
	if err = testDB.Ping(); err != nil {
		panic(fmt.Sprintf("Failed to ping test database: %v", err))
	}

	// Setup test schema
	setupTestSchemaForMain(testDB)

	// Setup test router
	testRouter = setupTestRouter(testDB, testLogger)

	// Run tests
	code := m.Run()

	// Cleanup
	if testDB != nil {
		testDB.Close()
	}
	os.Exit(code)
}

// setupTestSchemaForMain creates the test database schema (for TestMain)
func setupTestSchemaForMain(db *sql.DB) {
	query := `
		CREATE TABLE IF NOT EXISTS characters (
			id INT PRIMARY KEY AUTO_INCREMENT,
			consonant VARCHAR(10) NOT NULL,
			vowel VARCHAR(10) NOT NULL,
			english_reading VARCHAR(50) NOT NULL,
			russian_reading VARCHAR(50) NOT NULL,
			katakana VARCHAR(10) NOT NULL,
			hiragana VARCHAR(10) NOT NULL,
			INDEX idx_consonant (consonant),
			INDEX idx_vowel (vowel),
			INDEX idx_katakana (katakana),
			INDEX idx_hiragana (hiragana)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
	`

	db.Exec(query)
}

func TestIntegration_GetAllCharacters(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	seedTestData(t, testDB)
	defer cleanupTestData(t, testDB)

	tests := []struct {
		name           string
		queryParams    string
		expectedStatus int
		expectedCount  int
		validateFunc   func(*testing.T, []models.CharacterResponse)
	}{
		{
			name:           "get all hiragana english",
			queryParams:    "?type=hr&locale=en",
			expectedStatus: http.StatusOK,
			expectedCount:  46,
			validateFunc: func(t *testing.T, chars []models.CharacterResponse) {
				assert.Greater(t, len(chars), 0)
				for _, char := range chars {
					assert.NotEmpty(t, char.Character)
					assert.NotEmpty(t, char.Reading)
					// Verify it's hiragana (contains hiragana characters)
					assert.Contains(t, "あいうえおかきくけこさしすせそたちつてとなにぬねのはひふへほまみむめもやゆよらりるれろわをん", char.Character)
				}
			},
		},
		{
			name:           "get all katakana russian",
			queryParams:    "?type=kt&locale=ru",
			expectedStatus: http.StatusOK,
			expectedCount:  46,
			validateFunc: func(t *testing.T, chars []models.CharacterResponse) {
				assert.Greater(t, len(chars), 0)
				for _, char := range chars {
					assert.NotEmpty(t, char.Character)
					assert.NotEmpty(t, char.Reading)
					// Verify it's katakana (contains katakana characters)
					assert.Contains(t, "アイウエオカキクケコサシスセソタチツテトナニヌネノハヒフヘホマミムメモヤユヨラリルレロワヲン", char.Character)
				}
			},
		},
		{
			name:           "default parameters",
			queryParams:    "",
			expectedStatus: http.StatusOK,
			expectedCount:  46,
			validateFunc: func(t *testing.T, chars []models.CharacterResponse) {
				assert.Greater(t, len(chars), 0)
			},
		},
		{
			name:           "invalid alphabet type",
			queryParams:    "?type=invalid&locale=en",
			expectedStatus: http.StatusInternalServerError,
			expectedCount:  0,
			validateFunc:   nil,
		},
		{
			name:           "invalid locale",
			queryParams:    "?type=hr&locale=invalid",
			expectedStatus: http.StatusInternalServerError,
			expectedCount:  0,
			validateFunc:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/characters"+tt.queryParams, nil)
			w := httptest.NewRecorder()

			testRouter.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus == http.StatusOK {
				var result []models.CharacterResponse
				err := json.NewDecoder(w.Body).Decode(&result)
				require.NoError(t, err)
				assert.Len(t, result, tt.expectedCount)

				if tt.validateFunc != nil {
					tt.validateFunc(t, result)
				}
			}
		})
	}
}

func TestIntegration_GetByRowColumn(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	seedTestData(t, testDB)
	defer cleanupTestData(t, testDB)

	tests := []struct {
		name           string
		queryParams    string
		expectedStatus int
		expectedCount  int
		validateFunc   func(*testing.T, []models.CharacterResponse)
	}{
		{
			name:           "get by vowel a hiragana english",
			queryParams:    "?type=hr&locale=en&character=a",
			expectedStatus: http.StatusOK,
			expectedCount:  10, // 'a', 'ka', 'sa', 'ta', 'na', 'ha', 'ma', 'ya', 'ra', 'wa'
			validateFunc: func(t *testing.T, chars []models.CharacterResponse) {
				for _, char := range chars {
					assert.Equal(t, "a", char.Vowel)
				}
			},
		},
		{
			name:           "get by consonant k katakana russian",
			queryParams:    "?type=kt&locale=ru&character=k",
			expectedStatus: http.StatusOK,
			expectedCount:  5, // ka, ki, ku, ke, ko
			validateFunc: func(t *testing.T, chars []models.CharacterResponse) {
				for _, char := range chars {
					assert.Equal(t, "k", char.Consonant)
				}
			},
		},
		{
			name:           "missing character parameter",
			queryParams:    "?type=hr&locale=en",
			expectedStatus: http.StatusBadRequest,
			expectedCount:  0,
			validateFunc:   nil,
		},
		{
			name:           "invalid alphabet type",
			queryParams:    "?type=invalid&locale=en&character=a",
			expectedStatus: http.StatusInternalServerError,
			expectedCount:  0,
			validateFunc:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/characters/row-column"+tt.queryParams, nil)
			w := httptest.NewRecorder()

			testRouter.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus == http.StatusOK {
				var result []models.CharacterResponse
				err := json.NewDecoder(w.Body).Decode(&result)
				require.NoError(t, err)
				assert.Len(t, result, tt.expectedCount)

				if tt.validateFunc != nil {
					tt.validateFunc(t, result)
				}
			}
		})
	}
}

func TestIntegration_GetByID(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	seedTestData(t, testDB)
	defer cleanupTestData(t, testDB)

	tests := []struct {
		name           string
		id             string
		queryParams    string
		expectedStatus int
		validateFunc   func(*testing.T, *models.Character)
	}{
		{
			name:           "get character by id english",
			id:             "1",
			queryParams:    "?locale=en",
			expectedStatus: http.StatusOK,
			validateFunc: func(t *testing.T, char *models.Character) {
				assert.Equal(t, 1, char.ID)
				assert.Equal(t, "a", char.EnglishReading)
				assert.Equal(t, "あ", char.Hiragana)
				assert.Equal(t, "ア", char.Katakana)
			},
		},
		{
			name:           "get character by id russian",
			id:             "1",
			queryParams:    "?locale=ru",
			expectedStatus: http.StatusOK,
			validateFunc: func(t *testing.T, char *models.Character) {
				assert.Equal(t, 1, char.ID)
				assert.Equal(t, "а", char.RussianReading)
			},
		},
		{
			name:           "character not found",
			id:             "999",
			queryParams:    "?locale=en",
			expectedStatus: http.StatusNotFound,
			validateFunc:   nil,
		},
		{
			name:           "invalid id",
			id:             "invalid",
			queryParams:    "?locale=en",
			expectedStatus: http.StatusBadRequest,
			validateFunc:   nil,
		},
		{
			name:           "invalid locale",
			id:             "1",
			queryParams:    "?locale=invalid",
			expectedStatus: http.StatusInternalServerError,
			validateFunc:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/characters/"+tt.id+tt.queryParams, nil)
			w := httptest.NewRecorder()

			testRouter.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus == http.StatusOK {
				var result models.Character
				err := json.NewDecoder(w.Body).Decode(&result)
				require.NoError(t, err)

				if tt.validateFunc != nil {
					tt.validateFunc(t, &result)
				}
			}
		})
	}
}

func TestIntegration_GetReadingTest(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	seedTestData(t, testDB)
	defer cleanupTestData(t, testDB)

	tests := []struct {
		name           string
		alphabetType   string
		queryParams    string
		expectedStatus int
		expectedCount  int
		validateFunc   func(*testing.T, []models.ReadingTestItem)
	}{
		{
			name:           "get reading test hiragana english",
			alphabetType:   "hiragana",
			queryParams:    "?locale=en",
			expectedStatus: http.StatusOK,
			expectedCount:  10, // testCount constant
			validateFunc: func(t *testing.T, items []models.ReadingTestItem) {
				for _, item := range items {
					assert.NotEmpty(t, item.CorrectChar)
					assert.NotEmpty(t, item.Reading)
					assert.Len(t, item.WrongOptions, 2)
					// Verify wrong options are different from correct char
					for _, wrong := range item.WrongOptions {
						assert.NotEqual(t, item.CorrectChar, wrong)
					}
				}
			},
		},
		{
			name:           "get reading test katakana russian",
			alphabetType:   "katakana",
			queryParams:    "?locale=ru",
			expectedStatus: http.StatusOK,
			expectedCount:  10,
			validateFunc: func(t *testing.T, items []models.ReadingTestItem) {
				for _, item := range items {
					assert.NotEmpty(t, item.CorrectChar)
					assert.NotEmpty(t, item.Reading)
					assert.Len(t, item.WrongOptions, 2)
				}
			},
		},
		{
			name:           "invalid alphabet type",
			alphabetType:   "invalid",
			queryParams:    "?locale=en",
			expectedStatus: http.StatusBadRequest,
			expectedCount:  0,
			validateFunc:   nil,
		},
		{
			name:           "invalid locale",
			alphabetType:   "hiragana",
			queryParams:    "?locale=invalid",
			expectedStatus: http.StatusBadRequest,
			expectedCount:  0,
			validateFunc:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/tests/"+tt.alphabetType+"/reading"+tt.queryParams, nil)
			w := httptest.NewRecorder()

			testRouter.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus == http.StatusOK {
				var result []models.ReadingTestItem
				err := json.NewDecoder(w.Body).Decode(&result)
				require.NoError(t, err)
				assert.Len(t, result, tt.expectedCount)

				if tt.validateFunc != nil {
					tt.validateFunc(t, result)
				}
			}
		})
	}
}

func TestIntegration_GetWritingTest(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	seedTestData(t, testDB)
	defer cleanupTestData(t, testDB)

	tests := []struct {
		name           string
		alphabetType   string
		queryParams    string
		expectedStatus int
		expectedCount  int
		validateFunc   func(*testing.T, []models.WritingTestItem)
	}{
		{
			name:           "get writing test hiragana english",
			alphabetType:   "hiragana",
			queryParams:    "?locale=en",
			expectedStatus: http.StatusOK,
			expectedCount:  10, // testCount constant
			validateFunc: func(t *testing.T, items []models.WritingTestItem) {
				for _, item := range items {
					assert.NotEmpty(t, item.Character)
					assert.NotEmpty(t, item.CorrectReading)
				}
			},
		},
		{
			name:           "get writing test katakana russian",
			alphabetType:   "katakana",
			queryParams:    "?locale=ru",
			expectedStatus: http.StatusOK,
			expectedCount:  10,
			validateFunc: func(t *testing.T, items []models.WritingTestItem) {
				for _, item := range items {
					assert.NotEmpty(t, item.Character)
					assert.NotEmpty(t, item.CorrectReading)
				}
			},
		},
		{
			name:           "invalid alphabet type",
			alphabetType:   "invalid",
			queryParams:    "?locale=en",
			expectedStatus: http.StatusBadRequest,
			expectedCount:  0,
			validateFunc:   nil,
		},
		{
			name:           "invalid locale",
			alphabetType:   "hiragana",
			queryParams:    "?locale=invalid",
			expectedStatus: http.StatusBadRequest,
			expectedCount:  0,
			validateFunc:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/tests/"+tt.alphabetType+"/writing"+tt.queryParams, nil)
			w := httptest.NewRecorder()

			testRouter.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus == http.StatusOK {
				var result []models.WritingTestItem
				err := json.NewDecoder(w.Body).Decode(&result)
				require.NoError(t, err)
				assert.Len(t, result, tt.expectedCount)

				if tt.validateFunc != nil {
					tt.validateFunc(t, result)
				}
			}
		})
	}
}

func TestIntegration_RepositoryLayer(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	seedTestData(t, testDB)
	defer cleanupTestData(t, testDB)

	logger, _ := zap.NewDevelopment()
	repo := repositories.NewCharactersRepository(testDB, logger)
	ctx := context.Background()

	t.Run("GetAll hiragana english", func(t *testing.T) {
		result, err := repo.GetAll(ctx, models.AlphabetTypeHiragana, models.LocaleEnglish)
		require.NoError(t, err)
		assert.Greater(t, len(result), 0)
		assert.Equal(t, "あ", result[0].Character)
		assert.Equal(t, "a", result[0].Reading)
	})

	t.Run("GetByRowColumn with vowel", func(t *testing.T) {
		result, err := repo.GetByRowColumn(ctx, models.AlphabetTypeHiragana, models.LocaleEnglish, "a")
		require.NoError(t, err)
		assert.Greater(t, len(result), 0)
		for _, char := range result {
			assert.Equal(t, "a", char.Vowel)
		}
	})

	t.Run("GetByID", func(t *testing.T) {
		result, err := repo.GetByID(ctx, 1, models.LocaleEnglish)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, 1, result.ID)
		assert.Equal(t, "a", result.EnglishReading)
	})

	t.Run("GetRandomForReadingTest", func(t *testing.T) {
		result, err := repo.GetRandomForReadingTest(ctx, models.AlphabetTypeHiragana, models.LocaleEnglish, 5)
		require.NoError(t, err)
		assert.Len(t, result, 5)
		for _, item := range result {
			assert.NotEmpty(t, item.CorrectChar)
			assert.NotEmpty(t, item.Reading)
			assert.Len(t, item.WrongOptions, 2)
		}
	})

	t.Run("GetRandomForWritingTest", func(t *testing.T) {
		result, err := repo.GetRandomForWritingTest(ctx, models.AlphabetTypeKatakana, models.LocaleRussian, 5)
		require.NoError(t, err)
		assert.Len(t, result, 5)
		for _, item := range result {
			assert.NotEmpty(t, item.Character)
			assert.NotEmpty(t, item.CorrectReading)
		}
	})
}

func TestIntegration_ServiceLayer(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	seedTestData(t, testDB)
	defer cleanupTestData(t, testDB)

	logger, _ := zap.NewDevelopment()
	repo := repositories.NewCharactersRepository(testDB, logger)
	svc := services.NewCharactersService(repo, logger)
	ctx := context.Background()

	t.Run("GetAll", func(t *testing.T) {
		result, err := svc.GetAll(ctx, "hr", "en")
		require.NoError(t, err)
		assert.Greater(t, len(result), 0)
	})

	t.Run("GetByRowColumn", func(t *testing.T) {
		result, err := svc.GetByRowColumn(ctx, "hr", "en", "a")
		require.NoError(t, err)
		assert.Greater(t, len(result), 0)
	})

	t.Run("GetByID", func(t *testing.T) {
		result, err := svc.GetByID(ctx, 1, "en")
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, 1, result.ID)
	})

	t.Run("GetReadingTest", func(t *testing.T) {
		result, err := svc.GetReadingTest(ctx, "hiragana", "en")
		require.NoError(t, err)
		assert.Len(t, result, 10)
	})

	t.Run("GetWritingTest", func(t *testing.T) {
		result, err := svc.GetWritingTest(ctx, "katakana", "ru")
		require.NoError(t, err)
		assert.Len(t, result, 10)
	})
}

// Benchmark tests
func BenchmarkIntegration_GetAll(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping benchmarks in short mode")
	}

	seedTestData(&testing.T{}, testDB)
	defer cleanupTestData(&testing.T{}, testDB)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/characters?type=hr&locale=en", nil)

	for b.Loop() {
		w := httptest.NewRecorder()
		testRouter.ServeHTTP(w, req)
	}
}
