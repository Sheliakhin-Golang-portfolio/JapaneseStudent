package repositories

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Sheliakhin-Golang-portfolio/JapaneseStudent/services/auth-service/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupUserTokenTestRepository creates a user token repository with a mock database
func setupUserTokenTestRepository(t *testing.T) (*userTokenRepository, sqlmock.Sqlmock, func()) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)

	repo := NewUserTokenRepository(db)

	cleanup := func() {
		db.Close()
	}

	return repo, mock, cleanup
}

func TestNewUserTokenRepository(t *testing.T) {
	db := &sql.DB{}

	repo := NewUserTokenRepository(db)

	assert.NotNil(t, repo)
	assert.Equal(t, db, repo.db)
}

func TestUserTokenRepository_Create(t *testing.T) {
	tests := []struct {
		name          string
		userToken     *models.UserToken
		setupMock     func(sqlmock.Sqlmock)
		expectedError bool
	}{
		{
			name: "success",
			userToken: &models.UserToken{
				UserID: 1,
				Token:  "test-refresh-token",
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`INSERT INTO user_tokens`).
					WithArgs(1, "test-refresh-token").
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			expectedError: false,
		},
		{
			name: "database error",
			userToken: &models.UserToken{
				UserID: 1,
				Token:  "test-refresh-token",
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`INSERT INTO user_tokens`).
					WithArgs(1, "test-refresh-token").
					WillReturnError(errors.New("database error"))
			},
			expectedError: true,
		},
		{
			name: "foreign key constraint - invalid user_id",
			userToken: &models.UserToken{
				UserID: 999,
				Token:  "test-refresh-token",
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`INSERT INTO user_tokens`).
					WithArgs(999, "test-refresh-token").
					WillReturnError(errors.New("Error 1452: Cannot add or update a child row: a foreign key constraint fails"))
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock, cleanup := setupUserTokenTestRepository(t)
			defer cleanup()

			tt.setupMock(mock)

			err := repo.Create(context.Background(), tt.userToken)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestUserTokenRepository_GetByToken(t *testing.T) {
	tests := []struct {
		name          string
		token         string
		setupMock     func(sqlmock.Sqlmock)
		expectedError bool
		expectedToken *models.UserToken
	}{
		{
			name:  "success",
			token: "test-refresh-token",
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "user_id", "token"}).
					AddRow(1, 10, "test-refresh-token")
				mock.ExpectQuery(`SELECT id, user_id, token FROM user_tokens WHERE token = \? LIMIT 1`).
					WithArgs("test-refresh-token").
					WillReturnRows(rows)
			},
			expectedError: false,
			expectedToken: &models.UserToken{
				ID:     1,
				UserID: 10,
				Token:  "test-refresh-token",
			},
		},
		{
			name:  "not found",
			token: "nonexistent-token",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT id, user_id, token FROM user_tokens WHERE token = \? LIMIT 1`).
					WithArgs("nonexistent-token").
					WillReturnError(sql.ErrNoRows)
			},
			expectedError: true,
			expectedToken: nil,
		},
		{
			name:  "database error",
			token: "test-refresh-token",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT id, user_id, token FROM user_tokens WHERE token = \? LIMIT 1`).
					WithArgs("test-refresh-token").
					WillReturnError(errors.New("database error"))
			},
			expectedError: true,
			expectedToken: nil,
		},
		{
			name:  "scan error - invalid data types",
			token: "test-refresh-token",
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "user_id", "token"}).
					AddRow("invalid", 10, "test-refresh-token")
				mock.ExpectQuery(`SELECT id, user_id, token FROM user_tokens WHERE token = \? LIMIT 1`).
					WithArgs("test-refresh-token").
					WillReturnRows(rows)
			},
			expectedError: true,
			expectedToken: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock, cleanup := setupUserTokenTestRepository(t)
			defer cleanup()

			tt.setupMock(mock)

			userToken, err := repo.GetByToken(context.Background(), tt.token)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, userToken)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, userToken)
				assert.Equal(t, tt.expectedToken.ID, userToken.ID)
				assert.Equal(t, tt.expectedToken.UserID, userToken.UserID)
				assert.Equal(t, tt.expectedToken.Token, userToken.Token)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestUserTokenRepository_UpdateToken(t *testing.T) {
	tests := []struct {
		name          string
		oldToken      string
		newToken      string
		userID        int
		setupMock     func(sqlmock.Sqlmock)
		expectedError bool
	}{
		{
			name:     "success",
			oldToken: "old-token",
			newToken: "new-token",
			userID:   1,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`UPDATE user_tokens SET token = \? WHERE token = \? AND user_id = \?`).
					WithArgs("new-token", "old-token", 1).
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
			expectedError: false,
		},
		{
			name:     "token not found - 0 rows affected",
			oldToken: "nonexistent-token",
			newToken: "new-token",
			userID:   1,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`UPDATE user_tokens SET token = \? WHERE token = \? AND user_id = \?`).
					WithArgs("new-token", "nonexistent-token", 1).
					WillReturnResult(sqlmock.NewResult(0, 0))
			},
			expectedError: true,
		},
		{
			name:     "user mismatch - wrong userID",
			oldToken: "old-token",
			newToken: "new-token",
			userID:   999,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`UPDATE user_tokens SET token = \? WHERE token = \? AND user_id = \?`).
					WithArgs("new-token", "old-token", 999).
					WillReturnResult(sqlmock.NewResult(0, 0))
			},
			expectedError: true,
		},
		{
			name:     "database error",
			oldToken: "old-token",
			newToken: "new-token",
			userID:   1,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`UPDATE user_tokens SET token = \? WHERE token = \? AND user_id = \?`).
					WithArgs("new-token", "old-token", 1).
					WillReturnError(errors.New("database error"))
			},
			expectedError: true,
		},
		{
			name:     "error getting rows affected",
			oldToken: "old-token",
			newToken: "new-token",
			userID:   1,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`UPDATE user_tokens SET token = \? WHERE token = \? AND user_id = \?`).
					WithArgs("new-token", "old-token", 1).
					WillReturnResult(sqlmock.NewErrorResult(errors.New("rows affected error")))
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock, cleanup := setupUserTokenTestRepository(t)
			defer cleanup()

			tt.setupMock(mock)

			err := repo.UpdateToken(context.Background(), tt.oldToken, tt.newToken, tt.userID)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestUserTokenRepository_DeleteByToken(t *testing.T) {
	tests := []struct {
		name          string
		token         string
		setupMock     func(sqlmock.Sqlmock)
		expectedError bool
	}{
		{
			name:  "success",
			token: "test-token",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`DELETE FROM user_tokens WHERE token = \?`).
					WithArgs("test-token").
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
			expectedError: false,
		},
		{
			name:  "token doesn't exist - should not error",
			token: "nonexistent-token",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`DELETE FROM user_tokens WHERE token = \?`).
					WithArgs("nonexistent-token").
					WillReturnResult(sqlmock.NewResult(0, 0))
			},
			expectedError: false,
		},
		{
			name:  "database error",
			token: "test-token",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`DELETE FROM user_tokens WHERE token = \?`).
					WithArgs("test-token").
					WillReturnError(errors.New("database error"))
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock, cleanup := setupUserTokenTestRepository(t)
			defer cleanup()

			tt.setupMock(mock)

			err := repo.DeleteByToken(context.Background(), tt.token)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestUserTokenRepository_DeleteExpiredTokens(t *testing.T) {
	tests := []struct {
		name            string
		expiryTime      time.Time
		setupMock       func(sqlmock.Sqlmock)
		expectedError   bool
		expectedCount   int
	}{
		{
			name:       "success - delete expired tokens",
			expiryTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`DELETE FROM user_tokens WHERE created_at <= \?`).
					WithArgs(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)).
					WillReturnResult(sqlmock.NewResult(0, 5))
			},
			expectedError: false,
			expectedCount: 5,
		},
		{
			name:       "success - no tokens to delete",
			expiryTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`DELETE FROM user_tokens WHERE created_at <= \?`).
					WithArgs(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)).
					WillReturnResult(sqlmock.NewResult(0, 0))
			},
			expectedError: false,
			expectedCount: 0,
		},
		{
			name:       "database error",
			expiryTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`DELETE FROM user_tokens WHERE created_at <= \?`).
					WithArgs(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)).
					WillReturnError(errors.New("database error"))
			},
			expectedError: true,
			expectedCount:  0,
		},
		{
			name:       "error getting rows affected",
			expiryTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`DELETE FROM user_tokens WHERE created_at <= \?`).
					WithArgs(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)).
					WillReturnResult(sqlmock.NewErrorResult(errors.New("rows affected error")))
			},
			expectedError: true,
			expectedCount:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock, cleanup := setupUserTokenTestRepository(t)
			defer cleanup()

			tt.setupMock(mock)

			count, err := repo.DeleteExpiredTokens(context.Background(), tt.expiryTime)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Equal(t, 0, count)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedCount, count)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
