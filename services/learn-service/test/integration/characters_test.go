package integration

import (
	"bytes"
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
	"github.com/japanesestudent/learn-service/internal/handlers"
	"github.com/japanesestudent/learn-service/internal/models"
	"github.com/japanesestudent/learn-service/internal/repositories"
	"github.com/japanesestudent/learn-service/internal/services"
	"github.com/japanesestudent/libs/auth/middleware"
	"github.com/japanesestudent/libs/config"
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
	_, err := db.Exec("DELETE FROM dictionary_history")
	require.NoError(t, err, "Failed to clear dictionary_history")
	_, err = db.Exec("DELETE FROM words")
	require.NoError(t, err, "Failed to clear words")
	_, err = db.Exec("DELETE FROM character_learn_history")
	require.NoError(t, err, "Failed to clear character_learn_history")
	_, err = db.Exec("DELETE FROM characters")
	require.NoError(t, err, "Failed to clear test data")

	// Reset AUTO_INCREMENT to start from 1
	_, err = db.Exec("ALTER TABLE characters AUTO_INCREMENT = 1")
	require.NoError(t, err, "Failed to reset AUTO_INCREMENT")
	_, err = db.Exec("ALTER TABLE words AUTO_INCREMENT = 1")
	require.NoError(t, err, "Failed to reset AUTO_INCREMENT for words")

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
	_, err := db.Exec("DELETE FROM dictionary_history")
	require.NoError(t, err, "Failed to cleanup dictionary_history")
	_, err = db.Exec("DELETE FROM character_learn_history")
	require.NoError(t, err, "Failed to cleanup character_learn_history")
	_, err = db.Exec("DELETE FROM words")
	require.NoError(t, err, "Failed to cleanup words")
	_, err = db.Exec("DELETE FROM characters")
	require.NoError(t, err, "Failed to cleanup characters")
}

// setupTestRouter creates a test router with all handlers
func setupTestRouter(db *sql.DB, logger *zap.Logger) chi.Router {
	repo := repositories.NewCharactersRepository(db)
	historyRepo := repositories.NewCharacterLearnHistoryRepository(db)
	svc := services.NewCharactersService(repo, historyRepo)
	charHandler := handlers.NewCharactersHandler(svc, logger)

	testResultSvc := services.NewTestResultService(historyRepo)
	testResultHandler := handlers.NewTestResultHandler(testResultSvc, logger)

	wordRepo := repositories.NewWordRepository(db)
	dictionaryHistoryRepo := repositories.NewDictionaryHistoryRepository(db)
	dictionarySvc := services.NewDictionaryService(wordRepo, dictionaryHistoryRepo)
	dictionaryHandler := handlers.NewDictionaryHandler(dictionarySvc, logger)

	r := chi.NewRouter()
	r.Route("/api/v3", func(r chi.Router) {
		// Register character routes (excluding /tests which we'll register together)
		r.Route("/characters", func(r chi.Router) {
			r.Get("/", charHandler.GetAll)
			r.Get("/row-column", charHandler.GetByRowColumn)
			r.Get("/{id}", charHandler.GetByID)
		})

		// Register all test routes together
		authMiddleware := func(h http.Handler) http.Handler { return h }
		r.Route("/tests", func(r chi.Router) {
			r.Use(authMiddleware)
			// Character test routes
			r.Get("/{type}/reading", charHandler.GetReadingTest)
			r.Get("/{type}/writing", charHandler.GetWritingTest)
			// Test result routes
			r.Post("/{type}/{testType}", testResultHandler.SubmitTestResult)
			r.Get("/history", testResultHandler.GetUserHistory)
		})

		// Register dictionary routes
		r.Route("/words", func(r chi.Router) {
			r.Use(authMiddleware)
			r.Get("/", dictionaryHandler.GetWordList)
			r.Post("/results", dictionaryHandler.SubmitWordResults)
		})
	})

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
	charactersTable := `
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

	historyTable := `
		CREATE TABLE IF NOT EXISTS character_learn_history (
			id INT PRIMARY KEY AUTO_INCREMENT,
			user_id INT NOT NULL,
			character_id INT NOT NULL,
			hiragana_reading_result FLOAT DEFAULT 0,
			hiragana_writing_result FLOAT DEFAULT 0,
			hiragana_listening_result FLOAT DEFAULT 0,
			katakana_reading_result FLOAT DEFAULT 0,
			katakana_writing_result FLOAT DEFAULT 0,
			katakana_listening_result FLOAT DEFAULT 0,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			UNIQUE KEY unique_user_character (user_id, character_id),
			FOREIGN KEY (character_id) REFERENCES characters(id) ON DELETE CASCADE
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
	`

	wordsTable := `
		CREATE TABLE IF NOT EXISTS words (
			id INT PRIMARY KEY AUTO_INCREMENT,
			word VARCHAR(255) NOT NULL,
			phonetic_clues VARCHAR(255) NOT NULL,
			russian_translation VARCHAR(255) NOT NULL,
			english_translation VARCHAR(255) NOT NULL,
			german_translation VARCHAR(255) NOT NULL,
			example TEXT NOT NULL,
			example_russian_translation TEXT NOT NULL,
			example_english_translation TEXT NOT NULL,
			example_german_translation TEXT NOT NULL,
			easy_period INT NOT NULL DEFAULT 1,
			normal_period INT NOT NULL DEFAULT 3,
			hard_period INT NOT NULL DEFAULT 7,
			extra_hard_period INT NOT NULL DEFAULT 14
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
	`

	dictionaryHistoryTable := `
		CREATE TABLE IF NOT EXISTS dictionary_history (
			id INT PRIMARY KEY AUTO_INCREMENT,
			user_id INT NOT NULL,
			word_id INT NOT NULL,
			next_appearance DATE NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			UNIQUE KEY unique_user_word (user_id, word_id),
			FOREIGN KEY (word_id) REFERENCES words(id) ON DELETE CASCADE
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
	`

	db.Exec(charactersTable)
	db.Exec(historyTable)
	db.Exec(wordsTable)
	db.Exec(dictionaryHistoryTable)
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
			req := httptest.NewRequest(http.MethodGet, "/api/v3/characters"+tt.queryParams, nil)
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
			req := httptest.NewRequest(http.MethodGet, "/api/v3/characters/row-column"+tt.queryParams, nil)
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
			req := httptest.NewRequest(http.MethodGet, "/api/v3/characters/"+tt.id+tt.queryParams, nil)
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
			req := httptest.NewRequest(http.MethodGet, "/api/v3/tests/"+tt.alphabetType+"/reading"+tt.queryParams, nil)
			req = req.WithContext(middleware.SetUserID(req.Context(), 1))
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
			req := httptest.NewRequest(http.MethodGet, "/api/v3/tests/"+tt.alphabetType+"/writing"+tt.queryParams, nil)
			req = req.WithContext(middleware.SetUserID(req.Context(), 1))
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

	repo := repositories.NewCharactersRepository(testDB)
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

	t.Run("GetCharactersForReadingTest", func(t *testing.T) {
		// Get some character IDs first
		allChars, err := repo.GetAll(ctx, models.AlphabetTypeHiragana, models.LocaleEnglish)
		require.NoError(t, err)
		require.GreaterOrEqual(t, len(allChars), 5)

		characterIDs := make([]int, 5)
		for i := 0; i < 5; i++ {
			characterIDs[i] = allChars[i].ID
		}

		result, err := repo.GetCharactersForReadingTest(ctx, models.AlphabetTypeHiragana, models.LocaleEnglish, characterIDs)
		require.NoError(t, err)
		assert.Len(t, result, 5)
		for _, item := range result {
			assert.NotEmpty(t, item.CorrectChar)
			assert.NotEmpty(t, item.Reading)
			assert.Len(t, item.WrongOptions, 2)
		}
	})

	t.Run("GetCharactersForWritingTest", func(t *testing.T) {
		// Get some character IDs first
		allChars, err := repo.GetAll(ctx, models.AlphabetTypeKatakana, models.LocaleRussian)
		require.NoError(t, err)
		require.GreaterOrEqual(t, len(allChars), 5)

		characterIDs := make([]int, 5)
		for i := 0; i < 5; i++ {
			characterIDs[i] = allChars[i].ID
		}

		result, err := repo.GetCharactersForWritingTest(ctx, models.AlphabetTypeKatakana, models.LocaleRussian, characterIDs)
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

	repo := repositories.NewCharactersRepository(testDB)
	historyRepo := repositories.NewCharacterLearnHistoryRepository(testDB)
	svc := services.NewCharactersService(repo, historyRepo)
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
		result, err := svc.GetReadingTest(ctx, "hiragana", "en", 10, 1)
		require.NoError(t, err)
		assert.Len(t, result, 10)
	})

	t.Run("GetWritingTest", func(t *testing.T) {
		result, err := svc.GetWritingTest(ctx, "katakana", "ru", 10, 1)
		require.NoError(t, err)
		assert.Len(t, result, 10)
	})
}

func TestIntegration_SubmitTestResults(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	cleanupTestData(t, testDB)
	seedTestData(t, testDB)
	defer cleanupTestData(t, testDB)

	// Clean history table
	_, err := testDB.Exec("DELETE FROM character_learn_history")
	require.NoError(t, err)

	tests := []struct {
		name           string
		userID         int
		alphabetType   string
		testType       string
		results        []map[string]any
		expectedStatus int
		validateFunc   func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:         "success submit hiragana reading results",
			userID:       1,
			alphabetType: "hiragana",
			testType:     "reading",
			results: []map[string]any{
				{"characterId": 1, "passed": true},
				{"characterId": 2, "passed": false},
			},
			expectedStatus: http.StatusOK,
			validateFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				// Verify database records were created
				var count int
				err := testDB.QueryRow("SELECT COUNT(*) FROM character_learn_history WHERE user_id = ?", 1).Scan(&count)
				require.NoError(t, err)
				assert.Equal(t, 2, count)

				// Verify result values
				var result float64
				err = testDB.QueryRow("SELECT hiragana_reading_result FROM character_learn_history WHERE user_id = ? AND character_id = ?", 1, 1).Scan(&result)
				require.NoError(t, err)
				assert.Equal(t, 1.0, result)

				err = testDB.QueryRow("SELECT hiragana_reading_result FROM character_learn_history WHERE user_id = ? AND character_id = ?", 1, 2).Scan(&result)
				require.NoError(t, err)
				assert.Equal(t, 0.0, result)
			},
		},
		{
			name:         "success submit katakana writing results",
			userID:       2,
			alphabetType: "katakana",
			testType:     "writing",
			results: []map[string]any{
				{"characterId": 1, "passed": true},
			},
			expectedStatus: http.StatusOK,
			validateFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				var result float64
				err := testDB.QueryRow("SELECT katakana_writing_result FROM character_learn_history WHERE user_id = ? AND character_id = ?", 2, 1).Scan(&result)
				require.NoError(t, err)
				assert.Equal(t, 1.0, result)
			},
		},
		{
			name:         "success submit listening results",
			userID:       3,
			alphabetType: "hiragana",
			testType:     "listening",
			results: []map[string]any{
				{"characterId": 1, "passed": true},
			},
			expectedStatus: http.StatusOK,
			validateFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				var result float64
				err := testDB.QueryRow("SELECT hiragana_listening_result FROM character_learn_history WHERE user_id = ? AND character_id = ?", 3, 1).Scan(&result)
				require.NoError(t, err)
				assert.Equal(t, 1.0, result)
			},
		},
		{
			name:         "success update existing history",
			userID:       4,
			alphabetType: "hiragana",
			testType:     "reading",
			results: []map[string]any{
				{"characterId": 1, "passed": true},
			},
			expectedStatus: http.StatusOK,
			validateFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				// First submission should create record with 1.0
				var result float64
				err := testDB.QueryRow("SELECT hiragana_reading_result FROM character_learn_history WHERE user_id = ? AND character_id = ?", 4, 1).Scan(&result)
				require.NoError(t, err)
				assert.Equal(t, 1.0, result)

				// Submit again with failed result
				body, _ := json.Marshal(map[string]any{
					"results": []map[string]any{{"characterId": 1, "passed": false}},
				})
				req := httptest.NewRequest(http.MethodPost, "/api/v3/tests/hiragana/reading", bytes.NewBuffer(body))
				req.Header.Set("Content-Type", "application/json")
				req = req.WithContext(middleware.SetUserID(req.Context(), 4))
				w2 := httptest.NewRecorder()
				testRouter.ServeHTTP(w2, req)

				// Should update to new value (replaces existing, not averaging)
				err = testDB.QueryRow("SELECT hiragana_reading_result FROM character_learn_history WHERE user_id = ? AND character_id = ?", 4, 1).Scan(&result)
				require.NoError(t, err)
				assert.Equal(t, 0.0, result)
			},
		},
		{
			name:         "invalid alphabet type",
			userID:       5,
			alphabetType: "kanji",
			testType:     "reading",
			results: []map[string]any{
				{"characterId": 1, "passed": true},
			},
			expectedStatus: http.StatusBadRequest,
			validateFunc:   nil,
		},
		{
			name:         "invalid test type",
			userID:       5,
			alphabetType: "hiragana",
			testType:     "speaking",
			results: []map[string]any{
				{"characterId": 1, "passed": true},
			},
			expectedStatus: http.StatusBadRequest,
			validateFunc:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(map[string]any{
				"results": tt.results,
			})
			url := fmt.Sprintf("/api/v3/tests/%s/%s", tt.alphabetType, tt.testType)
			req := httptest.NewRequest(http.MethodPost, url, bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			req = req.WithContext(middleware.SetUserID(req.Context(), tt.userID))
			w := httptest.NewRecorder()

			testRouter.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.validateFunc != nil {
				tt.validateFunc(t, w)
			}
		})
	}
}

func TestIntegration_GetUserHistory(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	cleanupTestData(t, testDB)
	seedTestData(t, testDB)
	defer cleanupTestData(t, testDB)

	// Clean and seed history
	_, err := testDB.Exec("DELETE FROM character_learn_history")
	require.NoError(t, err)

	// Insert test history
	_, err = testDB.Exec(`
		INSERT INTO character_learn_history 
		(user_id, character_id, hiragana_reading_result, hiragana_writing_result, katakana_reading_result) 
		VALUES (1, 1, 1.0, 0.8, 0.5), (1, 2, 0.9, 0.7, 0.6)
	`)
	require.NoError(t, err)

	tests := []struct {
		name           string
		userID         int
		expectedStatus int
		validateFunc   func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:           "success get history for user with data",
			userID:         1,
			expectedStatus: http.StatusOK,
			validateFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response []map[string]interface{}
				err := json.NewDecoder(w.Body).Decode(&response)
				require.NoError(t, err)
				assert.Len(t, response, 2)

				// Verify JOIN with characters table works
				assert.NotEmpty(t, response[0]["characterHiragana"])
				assert.NotEmpty(t, response[0]["characterKatakana"])
			},
		},
		{
			name:           "empty history for new user",
			userID:         999,
			expectedStatus: http.StatusOK,
			validateFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response []map[string]interface{}
				err := json.NewDecoder(w.Body).Decode(&response)
				require.NoError(t, err)
				assert.Len(t, response, 0)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v3/tests/history", nil)
			req = req.WithContext(middleware.SetUserID(req.Context(), tt.userID))
			w := httptest.NewRecorder()

			testRouter.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.validateFunc != nil {
				tt.validateFunc(t, w)
			}
		})
	}
}

func TestIntegration_CharacterLearnHistoryRepository(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	cleanupTestData(t, testDB)
	seedTestData(t, testDB)
	defer cleanupTestData(t, testDB)

	// Clean history
	_, err := testDB.Exec("DELETE FROM character_learn_history")
	require.NoError(t, err)

	historyRepo := repositories.NewCharacterLearnHistoryRepository(testDB)
	ctx := context.Background()

	t.Run("GetByUserIDAndCharacterIDs with real data", func(t *testing.T) {
		// Insert test data
		_, err := testDB.Exec(`
			INSERT INTO character_learn_history 
			(user_id, character_id, hiragana_reading_result) 
			VALUES (1, 1, 1.0), (1, 2, 0.8)
		`)
		require.NoError(t, err)

		histories, err := historyRepo.GetByUserIDAndCharacterIDs(ctx, 1, []int{1, 2})
		require.NoError(t, err)
		assert.Len(t, histories, 2)
	})

	t.Run("GetByUserID with real data", func(t *testing.T) {
		histories, err := historyRepo.GetByUserID(ctx, 1)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(histories), 2)

		// Verify JOIN works
		for _, h := range histories {
			assert.NotEmpty(t, h.CharacterHiragana)
			assert.NotEmpty(t, h.CharacterKatakana)
		}
	})

	t.Run("Upsert new records", func(t *testing.T) {
		histories := []models.CharacterLearnHistory{
			{UserID: 2, CharacterID: 1, HiraganaReadingResult: 1.0},
			{UserID: 2, CharacterID: 2, HiraganaWritingResult: 0.9},
		}
		err := historyRepo.Upsert(ctx, histories)
		require.NoError(t, err)

		// Verify records were created
		var count int
		err = testDB.QueryRow("SELECT COUNT(*) FROM character_learn_history WHERE user_id = ?", 2).Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 2, count)
	})

	t.Run("Upsert existing records", func(t *testing.T) {
		// Update existing records
		histories := []models.CharacterLearnHistory{
			{UserID: 2, CharacterID: 1, HiraganaReadingResult: 0.5},
		}
		err := historyRepo.Upsert(ctx, histories)
		require.NoError(t, err)

		// Verify record was updated
		var result float64
		err = testDB.QueryRow("SELECT hiragana_reading_result FROM character_learn_history WHERE user_id = ? AND character_id = ?", 2, 1).Scan(&result)
		require.NoError(t, err)
		assert.Equal(t, 0.5, result)
	})

	t.Run("Transaction rollback on error", func(t *testing.T) {
		// Try to insert with invalid character_id (foreign key constraint)
		histories := []models.CharacterLearnHistory{
			{UserID: 3, CharacterID: 99999, HiraganaReadingResult: 1.0},
		}
		err := historyRepo.Upsert(ctx, histories)
		assert.Error(t, err)

		// Verify no record was created
		var count int
		err = testDB.QueryRow("SELECT COUNT(*) FROM character_learn_history WHERE user_id = ?", 3).Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 0, count)
	})
}

// Benchmark tests
func BenchmarkIntegration_GetAll(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping benchmarks in short mode")
	}

	seedTestData(&testing.T{}, testDB)
	defer cleanupTestData(&testing.T{}, testDB)

	req := httptest.NewRequest(http.MethodGet, "/api/v3/characters?type=hr&locale=en", nil)

	for b.Loop() {
		w := httptest.NewRecorder()
		testRouter.ServeHTTP(w, req)
	}
}

func TestIntegration_Dictionary(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	cleanupTestData(t, testDB)
	seedTestData(t, testDB)
	defer cleanupTestData(t, testDB)

	// Seed words
	_, err := testDB.Exec(`
		INSERT INTO words (word, phonetic_clues, russian_translation, english_translation, german_translation, 
		                   example, example_russian_translation, example_english_translation, example_german_translation,
		                   easy_period, normal_period, hard_period, extra_hard_period) VALUES
		('水', 'みず', 'вода', 'water', 'Wasser', '水を飲む', 'пить воду', 'drink water', 'Wasser trinken', 1, 3, 7, 14),
		('火', 'ひ', 'огонь', 'fire', 'Feuer', '火をつける', 'зажечь огонь', 'light a fire', 'Feuer anzünden', 1, 3, 7, 14),
		('風', 'かぜ', 'ветер', 'wind', 'Wind', '風が吹く', 'дует ветер', 'wind blows', 'Wind weht', 1, 3, 7, 14),
		('木', 'き', 'дерево', 'tree', 'Baum', '木を植える', 'посадить дерево', 'plant a tree', 'Baum pflanzen', 1, 3, 7, 14),
		('土', 'つち', 'земля', 'earth', 'Erde', '土を耕す', 'обрабатывать землю', 'till the earth', 'Erde bestellen', 1, 3, 7, 14)
	`)
	require.NoError(t, err, "Failed to seed words")

	tests := []struct {
		name           string
		userID         int
		method         string
		url            string
		requestBody    map[string]any
		expectedStatus int
		validateFunc   func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:           "success get word list with defaults",
			userID:         1,
			method:         http.MethodGet,
			url:            "/api/v3/words",
			requestBody:    nil,
			expectedStatus: http.StatusOK,
			validateFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response []models.WordResponse
				err := json.NewDecoder(w.Body).Decode(&response)
				require.NoError(t, err)
				assert.Greater(t, len(response), 0)
			},
		},
		{
			name:           "success get word list with parameters",
			userID:         1,
			method:         http.MethodGet,
			url:            "/api/v3/words?newCount=10&oldCount=10&locale=en",
			requestBody:    nil,
			expectedStatus: http.StatusOK,
			validateFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response []models.WordResponse
				err := json.NewDecoder(w.Body).Decode(&response)
				require.NoError(t, err)
				assert.LessOrEqual(t, len(response), 20) // newCount + oldCount
			},
		},
		{
			name:           "success get word list with russian locale",
			userID:         1,
			method:         http.MethodGet,
			url:            "/api/v3/words?newCount=10&oldCount=10&locale=ru",
			requestBody:    nil,
			expectedStatus: http.StatusOK,
			validateFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response []models.WordResponse
				err := json.NewDecoder(w.Body).Decode(&response)
				require.NoError(t, err)
				if len(response) > 0 {
					// Verify translation is in Russian
					assert.NotEmpty(t, response[0].Translation)
				}
			},
		},
		{
			name:           "invalid newCount - too low",
			userID:         1,
			method:         http.MethodGet,
			url:            "/api/v3/words?newCount=5&oldCount=20&locale=en",
			requestBody:    nil,
			expectedStatus: http.StatusBadRequest,
			validateFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response map[string]string
				err := json.NewDecoder(w.Body).Decode(&response)
				require.NoError(t, err)
				assert.Contains(t, response["error"], "newWordCount must be between 10 and 40")
			},
		},
		{
			name:           "invalid locale",
			userID:         1,
			method:         http.MethodGet,
			url:            "/api/v3/words?newCount=20&oldCount=20&locale=fr",
			requestBody:    nil,
			expectedStatus: http.StatusBadRequest,
			validateFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response map[string]string
				err := json.NewDecoder(w.Body).Decode(&response)
				require.NoError(t, err)
				assert.Contains(t, response["error"], "invalid locale")
			},
		},
		{
			name:   "success submit word results",
			userID: 1,
			method: http.MethodPost,
			url:    "/api/v3/words/results",
			requestBody: map[string]any{
				"results": []map[string]any{
					{"wordId": 1, "period": 3},
					{"wordId": 2, "period": 7},
				},
			},
			expectedStatus: http.StatusNoContent,
			validateFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				// Verify database records were created
				var count int
				err := testDB.QueryRow("SELECT COUNT(*) FROM dictionary_history WHERE user_id = ?", 1).Scan(&count)
				require.NoError(t, err)
				assert.Equal(t, 2, count)
			},
		},
		{
			name:   "invalid period - too low",
			userID: 1,
			method: http.MethodPost,
			url:    "/api/v3/words/results",
			requestBody: map[string]any{
				"results": []map[string]any{
					{"wordId": 1, "period": 0},
				},
			},
			expectedStatus: http.StatusBadRequest,
			validateFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response map[string]string
				err := json.NewDecoder(w.Body).Decode(&response)
				require.NoError(t, err)
				assert.Contains(t, response["error"], "period must be between 1 and 30")
			},
		},
		{
			name:   "invalid word ID - does not exist",
			userID: 1,
			method: http.MethodPost,
			url:    "/api/v3/words/results",
			requestBody: map[string]any{
				"results": []map[string]any{
					{"wordId": 999, "period": 3},
				},
			},
			expectedStatus: http.StatusBadRequest,
			validateFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response map[string]string
				err := json.NewDecoder(w.Body).Decode(&response)
				require.NoError(t, err)
				assert.Contains(t, response["error"], "word IDs do not exist")
			},
		},
		{
			name:   "empty results array",
			userID: 1,
			method: http.MethodPost,
			url:    "/api/v3/words/results",
			requestBody: map[string]any{
				"results": []map[string]any{},
			},
			expectedStatus: http.StatusBadRequest,
			validateFunc: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response map[string]string
				err := json.NewDecoder(w.Body).Decode(&response)
				require.NoError(t, err)
				assert.Contains(t, response["error"], "results array cannot be empty")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req *http.Request
			if tt.requestBody != nil {
				body, _ := json.Marshal(tt.requestBody)
				req = httptest.NewRequest(tt.method, tt.url, bytes.NewBuffer(body))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req = httptest.NewRequest(tt.method, tt.url, nil)
			}
			// Set userID in context for auth middleware
			req = req.WithContext(middleware.SetUserID(req.Context(), tt.userID))
			w := httptest.NewRecorder()

			testRouter.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.validateFunc != nil {
				tt.validateFunc(t, w)
			}
		})
	}
}

func TestIntegration_DictionaryRepositoryLayer(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	cleanupTestData(t, testDB)
	seedTestData(t, testDB)
	defer cleanupTestData(t, testDB)

	// Seed words
	_, err := testDB.Exec(`
		INSERT INTO words (word, phonetic_clues, russian_translation, english_translation, german_translation, 
		                   example, example_russian_translation, example_english_translation, example_german_translation,
		                   easy_period, normal_period, hard_period, extra_hard_period) VALUES
		('水', 'みず', 'вода', 'water', 'Wasser', '水を飲む', 'пить воду', 'drink water', 'Wasser trinken', 1, 3, 7, 14),
		('火', 'ひ', 'огонь', 'fire', 'Feuer', '火をつける', 'зажечь огонь', 'light a fire', 'Feuer anzünden', 1, 3, 7, 14)
	`)
	require.NoError(t, err)

	wordRepo := repositories.NewWordRepository(testDB)
	historyRepo := repositories.NewDictionaryHistoryRepository(testDB)
	ctx := context.Background()

	t.Run("WordRepository GetByIDs", func(t *testing.T) {
		words, err := wordRepo.GetByIDs(ctx, []int{1, 2}, "english_translation", "example_english_translation")
		require.NoError(t, err)
		assert.Len(t, words, 2)
	})

	t.Run("WordRepository GetExcludingIDs", func(t *testing.T) {
		words, err := wordRepo.GetExcludingIDs(ctx, 1, []int{1}, 1, "english_translation", "example_english_translation")
		require.NoError(t, err)
		assert.LessOrEqual(t, len(words), 1)
	})

	t.Run("WordRepository ValidateWordIDs", func(t *testing.T) {
		valid, err := wordRepo.ValidateWordIDs(ctx, []int{1, 2})
		require.NoError(t, err)
		assert.True(t, valid)

		valid, err = wordRepo.ValidateWordIDs(ctx, []int{999})
		require.NoError(t, err)
		assert.False(t, valid)
	})

	t.Run("DictionaryHistoryRepository GetOldWordIds", func(t *testing.T) {
		// Insert history
		_, err := testDB.Exec(`
			INSERT INTO dictionary_history (user_id, word_id, next_appearance) VALUES
			(1, 1, CURDATE()),
			(1, 2, DATE_ADD(CURDATE(), INTERVAL 1 DAY))
		`)
		require.NoError(t, err)

		wordIds, err := historyRepo.GetOldWordIds(ctx, 1, 10)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(wordIds), 1)
	})

	t.Run("DictionaryHistoryRepository UpsertResults", func(t *testing.T) {
		results := []models.WordResult{
			{WordID: 1, Period: 3},
			{WordID: 2, Period: 7},
		}
		err := historyRepo.UpsertResults(ctx, 2, results)
		require.NoError(t, err)

		// Verify records were created
		var count int
		err = testDB.QueryRow("SELECT COUNT(*) FROM dictionary_history WHERE user_id = ?", 2).Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 2, count)
	})
}

func TestIntegration_DictionaryServiceLayer(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	cleanupTestData(t, testDB)
	seedTestData(t, testDB)
	defer cleanupTestData(t, testDB)

	// Seed words
	_, err := testDB.Exec(`
		INSERT INTO words (word, phonetic_clues, russian_translation, english_translation, german_translation, 
		                   example, example_russian_translation, example_english_translation, example_german_translation,
		                   easy_period, normal_period, hard_period, extra_hard_period) VALUES
		('水', 'みず', 'вода', 'water', 'Wasser', '水を飲む', 'пить воду', 'drink water', 'Wasser trinken', 1, 3, 7, 14),
		('火', 'ひ', 'огонь', 'fire', 'Feuer', '火をつける', 'зажечь огонь', 'light a fire', 'Feuer anzünden', 1, 3, 7, 14),
		('風', 'かぜ', 'ветер', 'wind', 'Wind', '風が吹く', 'дует ветер', 'wind blows', 'Wind weht', 1, 3, 7, 14)
	`)
	require.NoError(t, err)

	wordRepo := repositories.NewWordRepository(testDB)
	historyRepo := repositories.NewDictionaryHistoryRepository(testDB)
	dictionarySvc := services.NewDictionaryService(wordRepo, historyRepo)
	ctx := context.Background()

	t.Run("GetWordList", func(t *testing.T) {
		words, err := dictionarySvc.GetWordList(ctx, 1, 10, 10, "en")
		require.NoError(t, err)
		assert.LessOrEqual(t, len(words), 20)
	})

	t.Run("SubmitWordResults", func(t *testing.T) {
		results := []models.WordResult{
			{WordID: 1, Period: 3},
			{WordID: 2, Period: 7},
		}
		err := dictionarySvc.SubmitWordResults(ctx, 1, results)
		require.NoError(t, err)
	})
}
