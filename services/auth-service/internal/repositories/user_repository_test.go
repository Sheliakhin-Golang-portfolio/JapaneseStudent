package repositories

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/japanesestudent/auth-service/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// setupUserTestRepository creates a user repository with a mock database
func setupUserTestRepository(t *testing.T) (*userRepository, sqlmock.Sqlmock, func()) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)

	logger, err := zap.NewDevelopment()
	require.NoError(t, err)

	repo := NewUserRepository(db, logger)

	cleanup := func() {
		db.Close()
	}

	return repo, mock, cleanup
}

func TestNewUserRepository(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	db := &sql.DB{}

	repo := NewUserRepository(db, logger)

	assert.NotNil(t, repo)
	assert.Equal(t, db, repo.db)
	assert.Equal(t, logger, repo.logger)
}

func TestUserRepository_Create(t *testing.T) {
	tests := []struct {
		name          string
		user          *models.User
		setupMock     func(sqlmock.Sqlmock)
		expectedError bool
		expectedID    int
	}{
		{
			name: "success",
			user: &models.User{
				Username:     "testuser",
				Email:        "test@example.com",
				PasswordHash: "hashedpassword",
				Role:         models.RoleUser,
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`INSERT INTO users`).
					WithArgs("testuser", "test@example.com", "hashedpassword", models.RoleUser).
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			expectedError: false,
			expectedID:    1,
		},
		{
			name: "database error on insert",
			user: &models.User{
				Username:     "testuser",
				Email:        "test@example.com",
				PasswordHash: "hashedpassword",
				Role:         models.RoleUser,
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`INSERT INTO users`).
					WithArgs("testuser", "test@example.com", "hashedpassword", models.RoleUser).
					WillReturnError(errors.New("database error"))
			},
			expectedError: true,
			expectedID:    0,
		},
		{
			name: "error getting last insert id",
			user: &models.User{
				Username:     "testuser",
				Email:        "test@example.com",
				PasswordHash: "hashedpassword",
				Role:         models.RoleUser,
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`INSERT INTO users`).
					WithArgs("testuser", "test@example.com", "hashedpassword", models.RoleUser).
					WillReturnResult(sqlmock.NewErrorResult(errors.New("last insert id error")))
			},
			expectedError: true,
			expectedID:    0,
		},
		{
			name: "duplicate email",
			user: &models.User{
				Username:     "testuser",
				Email:        "duplicate@example.com",
				PasswordHash: "hashedpassword",
				Role:         models.RoleUser,
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`INSERT INTO users`).
					WithArgs("testuser", "duplicate@example.com", "hashedpassword", models.RoleUser).
					WillReturnError(errors.New("Error 1062: Duplicate entry 'duplicate@example.com' for key 'email'"))
			},
			expectedError: true,
			expectedID:    0,
		},
		{
			name: "duplicate username",
			user: &models.User{
				Username:     "duplicateuser",
				Email:        "test@example.com",
				PasswordHash: "hashedpassword",
				Role:         models.RoleUser,
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`INSERT INTO users`).
					WithArgs("duplicateuser", "test@example.com", "hashedpassword", models.RoleUser).
					WillReturnError(errors.New("Error 1062: Duplicate entry 'duplicateuser' for key 'username'"))
			},
			expectedError: true,
			expectedID:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock, cleanup := setupUserTestRepository(t)
			defer cleanup()

			tt.setupMock(mock)

			err := repo.Create(context.Background(), tt.user)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedID, tt.user.ID)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestUserRepository_GetByEmailOrUsername(t *testing.T) {
	tests := []struct {
		name          string
		login         string
		setupMock     func(sqlmock.Sqlmock)
		expectedError bool
		expectedUser  *models.User
	}{
		{
			name:  "success find by email",
			login: "test@example.com",
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "username", "email", "password_hash", "role"}).
					AddRow(1, "testuser", "test@example.com", "hashedpassword", models.RoleUser)
				mock.ExpectQuery(`SELECT id, username, email, password_hash, role FROM users WHERE email = \? OR username = \? LIMIT 1`).
					WithArgs("test@example.com", "test@example.com").
					WillReturnRows(rows)
			},
			expectedError: false,
			expectedUser: &models.User{
				ID:           1,
				Username:     "testuser",
				Email:        "test@example.com",
				PasswordHash: "hashedpassword",
				Role:         models.RoleUser,
			},
		},
		{
			name:  "success find by username",
			login: "testuser",
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "username", "email", "password_hash", "role"}).
					AddRow(2, "testuser", "test@example.com", "hashedpassword", models.RoleUser)
				mock.ExpectQuery(`SELECT id, username, email, password_hash, role FROM users WHERE email = \? OR username = \? LIMIT 1`).
					WithArgs("testuser", "testuser").
					WillReturnRows(rows)
			},
			expectedError: false,
			expectedUser: &models.User{
				ID:           2,
				Username:     "testuser",
				Email:        "test@example.com",
				PasswordHash: "hashedpassword",
				Role:         models.RoleUser,
			},
		},
		{
			name:  "not found",
			login: "nonexistent@example.com",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT id, username, email, password_hash, role FROM users WHERE email = \? OR username = \? LIMIT 1`).
					WithArgs("nonexistent@example.com", "nonexistent@example.com").
					WillReturnError(sql.ErrNoRows)
			},
			expectedError: true,
			expectedUser:  nil,
		},
		{
			name:  "database error",
			login: "test@example.com",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT id, username, email, password_hash, role FROM users WHERE email = \? OR username = \? LIMIT 1`).
					WithArgs("test@example.com", "test@example.com").
					WillReturnError(errors.New("database error"))
			},
			expectedError: true,
			expectedUser:  nil,
		},
		{
			name:  "scan error - invalid data types",
			login: "test@example.com",
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "username", "email", "password_hash", "role"}).
					AddRow("invalid", "testuser", "test@example.com", "hashedpassword", models.RoleUser)
				mock.ExpectQuery(`SELECT id, username, email, password_hash, role FROM users WHERE email = \? OR username = \? LIMIT 1`).
					WithArgs("test@example.com", "test@example.com").
					WillReturnRows(rows)
			},
			expectedError: true,
			expectedUser:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock, cleanup := setupUserTestRepository(t)
			defer cleanup()

			tt.setupMock(mock)

			user, err := repo.GetByEmailOrUsername(context.Background(), tt.login)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, user)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, user)
				assert.Equal(t, tt.expectedUser.ID, user.ID)
				assert.Equal(t, tt.expectedUser.Username, user.Username)
				assert.Equal(t, tt.expectedUser.Email, user.Email)
				assert.Equal(t, tt.expectedUser.PasswordHash, user.PasswordHash)
				assert.Equal(t, tt.expectedUser.Role, user.Role)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestUserRepository_ExistsByEmail(t *testing.T) {
	tests := []struct {
		name           string
		email          string
		setupMock      func(sqlmock.Sqlmock)
		expectedError  bool
		expectedExists bool
	}{
		{
			name:  "email exists",
			email: "existing@example.com",
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"exists"}).AddRow(true)
				mock.ExpectQuery(`SELECT EXISTS\(SELECT \* FROM users WHERE email = \?\)`).
					WithArgs("existing@example.com").
					WillReturnRows(rows)
			},
			expectedError:  false,
			expectedExists: true,
		},
		{
			name:  "email does not exist",
			email: "nonexistent@example.com",
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"exists"}).AddRow(false)
				mock.ExpectQuery(`SELECT EXISTS\(SELECT \* FROM users WHERE email = \?\)`).
					WithArgs("nonexistent@example.com").
					WillReturnRows(rows)
			},
			expectedError:  false,
			expectedExists: false,
		},
		{
			name:  "database error",
			email: "test@example.com",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT EXISTS\(SELECT \* FROM users WHERE email = \?\)`).
					WithArgs("test@example.com").
					WillReturnError(errors.New("database error"))
			},
			expectedError:  true,
			expectedExists: false,
		},
		{
			name:  "scan error",
			email: "test@example.com",
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"exists"}).AddRow("invalid")
				mock.ExpectQuery(`SELECT EXISTS\(SELECT \* FROM users WHERE email = \?\)`).
					WithArgs("test@example.com").
					WillReturnRows(rows)
			},
			expectedError:  true,
			expectedExists: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock, cleanup := setupUserTestRepository(t)
			defer cleanup()

			tt.setupMock(mock)

			exists, err := repo.ExistsByEmail(context.Background(), tt.email)

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

func TestUserRepository_ExistsByUsername(t *testing.T) {
	tests := []struct {
		name           string
		username       string
		setupMock      func(sqlmock.Sqlmock)
		expectedError  bool
		expectedExists bool
	}{
		{
			name:     "username exists",
			username: "existinguser",
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"exists"}).AddRow(true)
				mock.ExpectQuery(`SELECT EXISTS\(SELECT \* FROM users WHERE username = \?\)`).
					WithArgs("existinguser").
					WillReturnRows(rows)
			},
			expectedError:  false,
			expectedExists: true,
		},
		{
			name:     "username does not exist",
			username: "nonexistentuser",
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"exists"}).AddRow(false)
				mock.ExpectQuery(`SELECT EXISTS\(SELECT \* FROM users WHERE username = \?\)`).
					WithArgs("nonexistentuser").
					WillReturnRows(rows)
			},
			expectedError:  false,
			expectedExists: false,
		},
		{
			name:     "database error",
			username: "testuser",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT EXISTS\(SELECT \* FROM users WHERE username = \?\)`).
					WithArgs("testuser").
					WillReturnError(errors.New("database error"))
			},
			expectedError:  true,
			expectedExists: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock, cleanup := setupUserTestRepository(t)
			defer cleanup()

			tt.setupMock(mock)

			exists, err := repo.ExistsByUsername(context.Background(), tt.username)

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

