package repositories

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Sheliakhin-Golang-portfolio/JapaneseStudent/services/learn-service/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupDictionaryHistoryTestRepository creates a dictionary history repository with a mock database
func setupDictionaryHistoryTestRepository(t *testing.T) (*dictionaryHistoryRepository, sqlmock.Sqlmock, func()) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)

	repo := NewDictionaryHistoryRepository(db)

	cleanup := func() {
		db.Close()
	}

	return repo, mock, cleanup
}

func TestNewDictionaryHistoryRepository(t *testing.T) {
	db := &sql.DB{}

	repo := NewDictionaryHistoryRepository(db)

	assert.NotNil(t, repo)
	assert.Equal(t, db, repo.db)
}

func TestDictionaryHistoryRepository_GetOldWordIds(t *testing.T) {
	tests := []struct {
		name          string
		userId        int
		limit         int
		setupMock     func(sqlmock.Sqlmock)
		expectedError bool
		expectedCount int
	}{
		{
			name:   "success with multiple word IDs",
			userId: 1,
			limit:  10,
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"word_id"}).
					AddRow(1).
					AddRow(2).
					AddRow(3)
				mock.ExpectQuery(`SELECT word_id FROM dictionary_history WHERE user_id = \? AND next_appearance <= CURDATE\(\) ORDER BY next_appearance ASC LIMIT \?`).
					WithArgs(1, 10).
					WillReturnRows(rows)
			},
			expectedError: false,
			expectedCount: 3,
		},
		{
			name:   "success with single word ID",
			userId: 1,
			limit:  5,
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"word_id"}).
					AddRow(1)
				mock.ExpectQuery(`SELECT word_id FROM dictionary_history WHERE user_id = \? AND next_appearance <= CURDATE\(\) ORDER BY next_appearance ASC LIMIT \?`).
					WithArgs(1, 5).
					WillReturnRows(rows)
			},
			expectedError: false,
			expectedCount: 1,
		},
		{
			name:   "empty result - no old words",
			userId: 1,
			limit:  10,
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"word_id"})
				mock.ExpectQuery(`SELECT word_id FROM dictionary_history WHERE user_id = \? AND next_appearance <= CURDATE\(\) ORDER BY next_appearance ASC LIMIT \?`).
					WithArgs(1, 10).
					WillReturnRows(rows)
			},
			expectedError: false,
			expectedCount: 0,
		},
		{
			name:   "database query error",
			userId: 1,
			limit:  10,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT word_id FROM dictionary_history WHERE user_id = \? AND next_appearance <= CURDATE\(\) ORDER BY next_appearance ASC LIMIT \?`).
					WithArgs(1, 10).
					WillReturnError(errors.New("database error"))
			},
			expectedError: true,
			expectedCount: 0,
		},
		{
			name:   "scan error",
			userId: 1,
			limit:  10,
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"word_id"}).
					AddRow("invalid")
				mock.ExpectQuery(`SELECT word_id FROM dictionary_history WHERE user_id = \? AND next_appearance <= CURDATE\(\) ORDER BY next_appearance ASC LIMIT \?`).
					WithArgs(1, 10).
					WillReturnRows(rows)
			},
			expectedError: true,
			expectedCount: 0,
		},
		{
			name:   "rows iteration error",
			userId: 1,
			limit:  10,
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"word_id"}).
					AddRow(1).
					RowError(0, errors.New("row error"))
				mock.ExpectQuery(`SELECT word_id FROM dictionary_history WHERE user_id = \? AND next_appearance <= CURDATE\(\) ORDER BY next_appearance ASC LIMIT \?`).
					WithArgs(1, 10).
					WillReturnRows(rows)
			},
			expectedError: true,
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock, cleanup := setupDictionaryHistoryTestRepository(t)
			defer cleanup()

			tt.setupMock(mock)

			result, err := repo.GetOldWordIds(context.Background(), tt.userId, tt.limit)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				if tt.expectedCount == 0 {
					assert.Empty(t, result)
				} else {
					assert.NotNil(t, result)
					assert.Len(t, result, tt.expectedCount)
				}
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestDictionaryHistoryRepository_UpsertResults(t *testing.T) {
	tests := []struct {
		name          string
		userId        int
		results       []models.WordResult
		setupMock     func(sqlmock.Sqlmock)
		expectedError bool
	}{
		{
			name:   "success insert new records",
			userId: 1,
			results: []models.WordResult{
				{WordID: 1, Period: 3},
				{WordID: 2, Period: 7},
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(`(?s)INSERT INTO dictionary_history.*VALUES.*ON DUPLICATE KEY UPDATE.*`).
					WithArgs(1, 1, 3, 1, 2, 7).
					WillReturnResult(sqlmock.NewResult(1, 2))
				mock.ExpectCommit()
			},
			expectedError: false,
		},
		{
			name:   "success update existing records",
			userId: 1,
			results: []models.WordResult{
				{WordID: 1, Period: 5},
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(`(?s)INSERT INTO dictionary_history.*VALUES.*ON DUPLICATE KEY UPDATE.*`).
					WithArgs(1, 1, 5).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			expectedError: false,
		},
		{
			name:          "empty results slice",
			userId:        1,
			results:       []models.WordResult{},
			setupMock:     func(mock sqlmock.Sqlmock) {},
			expectedError: true,
		},
		{
			name:   "transaction begin error",
			userId: 1,
			results: []models.WordResult{
				{WordID: 1, Period: 3},
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin().WillReturnError(errors.New("begin error"))
			},
			expectedError: true,
		},
		{
			name:   "database error on insert",
			userId: 1,
			results: []models.WordResult{
				{WordID: 1, Period: 3},
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(`(?s)INSERT INTO dictionary_history.*`).
					WithArgs(1, 1, 3).
					WillReturnError(errors.New("insert error"))
				mock.ExpectRollback()
			},
			expectedError: true,
		},
		{
			name:   "transaction commit error",
			userId: 1,
			results: []models.WordResult{
				{WordID: 1, Period: 3},
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(`(?s)INSERT INTO dictionary_history.*`).
					WithArgs(1, 1, 3).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit().WillReturnError(errors.New("commit error"))
			},
			expectedError: true,
		},
		{
			name:   "success with multiple results and different periods",
			userId: 2,
			results: []models.WordResult{
				{WordID: 1, Period: 1},
				{WordID: 2, Period: 3},
				{WordID: 3, Period: 7},
				{WordID: 4, Period: 14},
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(`(?s)INSERT INTO dictionary_history.*`).
					WithArgs(2, 1, 1, 2, 2, 3, 2, 3, 7, 2, 4, 14).
					WillReturnResult(sqlmock.NewResult(1, 4))
				mock.ExpectCommit()
			},
			expectedError: false,
		},
		{
			name:   "success with maximum period (30 days)",
			userId: 1,
			results: []models.WordResult{
				{WordID: 1, Period: 30},
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(`(?s)INSERT INTO dictionary_history.*`).
					WithArgs(1, 1, 30).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			expectedError: false,
		},
		{
			name:   "success with minimum period (1 day)",
			userId: 1,
			results: []models.WordResult{
				{WordID: 1, Period: 1},
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(`(?s)INSERT INTO dictionary_history.*`).
					WithArgs(1, 1, 1).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock, cleanup := setupDictionaryHistoryTestRepository(t)
			defer cleanup()

			tt.setupMock(mock)

			err := repo.UpsertResults(context.Background(), tt.userId, tt.results)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
