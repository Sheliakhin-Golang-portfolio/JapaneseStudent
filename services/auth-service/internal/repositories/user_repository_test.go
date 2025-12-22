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
)

// setupUserTestRepository creates a user repository with a mock database
func setupUserTestRepository(t *testing.T) (*userRepository, sqlmock.Sqlmock, func()) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)

	require.NoError(t, err)

	repo := NewUserRepository(db)

	cleanup := func() {
		db.Close()
	}

	return repo, mock, cleanup
}

func TestNewUserRepository(t *testing.T) {
	db := &sql.DB{}

	repo := NewUserRepository(db)

	assert.NotNil(t, repo)
	assert.Equal(t, db, repo.db)
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
					WithArgs("testuser", "test@example.com", "hashedpassword", models.RoleUser, "").
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
					WithArgs("testuser", "test@example.com", "hashedpassword", models.RoleUser, "").
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
					WithArgs("testuser", "test@example.com", "hashedpassword", models.RoleUser, "").
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
					WithArgs("testuser", "duplicate@example.com", "hashedpassword", models.RoleUser, "").
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
					WithArgs("duplicateuser", "test@example.com", "hashedpassword", models.RoleUser, "").
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
				rows := sqlmock.NewRows([]string{"id", "username", "email", "password_hash", "role", "avatar"}).
					AddRow(1, "testuser", "test@example.com", "hashedpassword", models.RoleUser, "")
				mock.ExpectQuery(`SELECT id, username, email, password_hash, role, avatar FROM users WHERE email = \? OR username = \? LIMIT 1`).
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
				Avatar:       "",
			},
		},
		{
			name:  "success find by username",
			login: "testuser",
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "username", "email", "password_hash", "role", "avatar"}).
					AddRow(2, "testuser", "test@example.com", "hashedpassword", models.RoleUser, "")
				mock.ExpectQuery(`SELECT id, username, email, password_hash, role, avatar FROM users WHERE email = \? OR username = \? LIMIT 1`).
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
				Avatar:       "",
			},
		},
		{
			name:  "not found",
			login: "nonexistent@example.com",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT id, username, email, password_hash, role, avatar FROM users WHERE email = \? OR username = \? LIMIT 1`).
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
				mock.ExpectQuery(`SELECT id, username, email, password_hash, role, avatar FROM users WHERE email = \? OR username = \? LIMIT 1`).
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
				rows := sqlmock.NewRows([]string{"id", "username", "email", "password_hash", "role", "avatar"}).
					AddRow("invalid", "testuser", "test@example.com", "hashedpassword", models.RoleUser, "")
				mock.ExpectQuery(`SELECT id, username, email, password_hash, role, avatar FROM users WHERE email = \? OR username = \? LIMIT 1`).
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
				assert.Equal(t, tt.expectedUser.Avatar, user.Avatar)
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

func TestUserRepository_GetByID(t *testing.T) {
	tests := []struct {
		name          string
		userID        int
		setupMock     func(sqlmock.Sqlmock)
		expectedError bool
		expectedUser  *models.User
	}{
		{
			name:   "success",
			userID: 1,
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"username", "email", "password_hash", "role", "avatar"}).
					AddRow("testuser", "test@example.com", "hashedpassword", models.RoleUser, "")
				mock.ExpectQuery(`SELECT username, email, password_hash, role, avatar FROM users WHERE id = \? LIMIT 1`).
					WithArgs(1).
					WillReturnRows(rows)
			},
			expectedError: false,
			expectedUser: &models.User{
				ID:           1,
				Username:     "testuser",
				Email:        "test@example.com",
				PasswordHash: "hashedpassword",
				Role:         models.RoleUser,
				Avatar:       "",
			},
		},
		{
			name:   "not found",
			userID: 999,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT username, email, password_hash, role, avatar FROM users WHERE id = \? LIMIT 1`).
					WithArgs(999).
					WillReturnError(sql.ErrNoRows)
			},
			expectedError: true,
			expectedUser:  nil,
		},
		{
			name:   "database error",
			userID: 1,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT username, email, password_hash, role, avatar FROM users WHERE id = \? LIMIT 1`).
					WithArgs(1).
					WillReturnError(errors.New("database error"))
			},
			expectedError: true,
			expectedUser:  nil,
		},
		{
			name:   "scan error",
			userID: 1,
			setupMock: func(mock sqlmock.Sqlmock) {
				// Provide invalid data: string "invalid" for role field which expects an int
				rows := sqlmock.NewRows([]string{"username", "email", "password_hash", "role", "avatar"}).
					AddRow("testuser", "test@example.com", "hashedpassword", "invalid", "")
				mock.ExpectQuery(`SELECT username, email, password_hash, role, avatar FROM users WHERE id = \? LIMIT 1`).
					WithArgs(1).
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

			user, err := repo.GetByID(context.Background(), tt.userID)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, user)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, user)
				if tt.expectedUser != nil {
					assert.Equal(t, tt.expectedUser.ID, user.ID)
					assert.Equal(t, tt.expectedUser.Username, user.Username)
					assert.Equal(t, tt.expectedUser.Email, user.Email)
					assert.Equal(t, tt.expectedUser.PasswordHash, user.PasswordHash)
					assert.Equal(t, tt.expectedUser.Role, user.Role)
					assert.Equal(t, tt.expectedUser.Avatar, user.Avatar)
				}
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestUserRepository_GetAll(t *testing.T) {
	tests := []struct {
		name          string
		page          int
		count         int
		role          *models.Role
		search        string
		setupMock     func(sqlmock.Sqlmock)
		expectedError bool
		expectedCount int
	}{
		{
			name:   "success without filters",
			page:   1,
			count:  10,
			role:   nil,
			search: "",
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "username", "email", "role", "avatar"}).
					AddRow(1, "user1", "user1@example.com", models.RoleUser, "").
					AddRow(2, "user2", "user2@example.com", models.RoleUser, "")
				mock.ExpectQuery(`SELECT id, username, email, role, avatar FROM users ORDER BY email LIMIT \? OFFSET \?`).
					WithArgs(10, 0).
					WillReturnRows(rows)
			},
			expectedError: false,
			expectedCount: 2,
		},
		{
			name:   "success with role filter",
			page:   1,
			count:  10,
			role:   func() *models.Role { r := models.RoleAdmin; return &r }(),
			search: "",
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "username", "email", "role", "avatar"}).
					AddRow(1, "admin1", "admin1@example.com", models.RoleAdmin, "")
				mock.ExpectQuery(`SELECT id, username, email, role, avatar FROM users WHERE role = \? ORDER BY email LIMIT \? OFFSET \?`).
					WithArgs(models.RoleAdmin, 10, 0).
					WillReturnRows(rows)
			},
			expectedError: false,
			expectedCount: 1,
		},
		{
			name:   "success with search filter",
			page:   1,
			count:  10,
			role:   nil,
			search: "test",
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "username", "email", "role", "avatar"}).
					AddRow(1, "testuser", "test@example.com", models.RoleUser, "")
				mock.ExpectQuery(`SELECT id, username, email, role, avatar FROM users WHERE \(email LIKE \? OR username LIKE \?\) ORDER BY email LIMIT \? OFFSET \?`).
					WithArgs("%test%", "%test%", 10, 0).
					WillReturnRows(rows)
			},
			expectedError: false,
			expectedCount: 1,
		},
		{
			name:   "success with role and search filters",
			page:   1,
			count:  10,
			role:   func() *models.Role { r := models.RoleUser; return &r }(),
			search: "test",
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "username", "email", "role", "avatar"}).
					AddRow(1, "testuser", "test@example.com", models.RoleUser, "")
				mock.ExpectQuery(`SELECT id, username, email, role, avatar FROM users WHERE role = \? AND \(email LIKE \? OR username LIKE \?\) ORDER BY email LIMIT \? OFFSET \?`).
					WithArgs(models.RoleUser, "%test%", "%test%", 10, 0).
					WillReturnRows(rows)
			},
			expectedError: false,
			expectedCount: 1,
		},
		{
			name:   "success with pagination",
			page:   2,
			count:  5,
			role:   nil,
			search: "",
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "username", "email", "role", "avatar"}).
					AddRow(6, "user6", "user6@example.com", models.RoleUser, "")
				mock.ExpectQuery(`SELECT id, username, email, role, avatar FROM users ORDER BY email LIMIT \? OFFSET \?`).
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
			role:   nil,
			search: "",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT id, username, email, role, avatar FROM users ORDER BY email LIMIT \? OFFSET \?`).
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
			role:   nil,
			search: "",
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "username", "email", "role", "avatar"}).
					AddRow("invalid", "user1", "user1@example.com", models.RoleUser, "")
				mock.ExpectQuery(`SELECT id, username, email, role, avatar FROM users ORDER BY email LIMIT \? OFFSET \?`).
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
			role:   nil,
			search: "",
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "username", "email", "role", "avatar"}).
					AddRow(1, "user1", "user1@example.com", models.RoleUser, "").
					RowError(0, errors.New("row error"))
				mock.ExpectQuery(`SELECT id, username, email, role, avatar FROM users ORDER BY email LIMIT \? OFFSET \?`).
					WithArgs(10, 0).
					WillReturnRows(rows)
			},
			expectedError: true,
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock, cleanup := setupUserTestRepository(t)
			defer cleanup()

			tt.setupMock(mock)

			result, err := repo.GetAll(context.Background(), tt.page, tt.count, tt.role, tt.search)

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

func TestUserRepository_Update(t *testing.T) {
	tests := []struct {
		name          string
		userID        int
		user          *models.User
		settings      *models.UserSettings
		setupMock     func(sqlmock.Sqlmock)
		expectedError bool
	}{
		{
			name:   "success - update user only",
			userID: 1,
			user: &models.User{
				Username: "updateduser",
				Email:    "updated@example.com",
				Role:     models.RoleAdmin,
			},
			settings: nil,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(`UPDATE users SET username = \?, email = \?, role = \? WHERE id = \?`).
					WithArgs("updateduser", "updated@example.com", models.RoleAdmin, 1).
					WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectCommit()
			},
			expectedError: false,
		},
		{
			name:   "success - update settings only",
			userID: 1,
			user:   nil,
			settings: &models.UserSettings{
				NewWordCount:       25,
				OldWordCount:       30,
				AlphabetLearnCount: 12,
				Language:           models.LanguageEnglish,
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(`UPDATE user_settings SET language = \?, new_word_count = \?, old_word_count = \?, alphabet_learn_count = \? WHERE user_id = \?`).
					WithArgs("en", 25, 30, 12, 1).
					WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectCommit()
			},
			expectedError: false,
		},
		{
			name:   "success - update both user and settings",
			userID: 1,
			user: &models.User{
				Username: "updateduser",
				Email:    "updated@example.com",
			},
			settings: &models.UserSettings{
				NewWordCount: 25,
				Language:     models.LanguageEnglish,
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(`UPDATE users SET username = \?, email = \? WHERE id = \?`).
					WithArgs("updateduser", "updated@example.com", 1).
					WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectExec(`UPDATE user_settings SET language = \?, new_word_count = \? WHERE user_id = \?`).
					WithArgs("en", 25, 1).
					WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectCommit()
			},
			expectedError: false,
		},
		{
			name:     "success - no fields to update",
			userID:   1,
			user:     &models.User{},
			settings: &models.UserSettings{},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectCommit()
			},
			expectedError: false,
		},
		{
			name:   "user not found",
			userID: 999,
			user: &models.User{
				Username: "updateduser",
			},
			settings: nil,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(`UPDATE users SET username = \? WHERE id = \?`).
					WithArgs("updateduser", 999).
					WillReturnResult(sqlmock.NewResult(0, 0))
				mock.ExpectRollback()
			},
			expectedError: true,
		},
		{
			name:   "settings not found",
			userID: 999,
			user:   nil,
			settings: &models.UserSettings{
				NewWordCount: 25,
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(`UPDATE user_settings SET new_word_count = \? WHERE user_id = \?`).
					WithArgs(25, 999).
					WillReturnResult(sqlmock.NewResult(0, 0))
				mock.ExpectRollback()
			},
			expectedError: true,
		},
		{
			name:   "database error on begin transaction",
			userID: 1,
			user: &models.User{
				Username: "updateduser",
			},
			settings: nil,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin().WillReturnError(errors.New("transaction error"))
			},
			expectedError: true,
		},
		{
			name:   "database error on user update",
			userID: 1,
			user: &models.User{
				Username: "updateduser",
			},
			settings: nil,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(`UPDATE users SET username = \? WHERE id = \?`).
					WithArgs("updateduser", 1).
					WillReturnError(errors.New("database error"))
				mock.ExpectRollback()
			},
			expectedError: true,
		},
		{
			name:   "database error on commit",
			userID: 1,
			user: &models.User{
				Username: "updateduser",
			},
			settings: nil,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(`UPDATE users SET username = \? WHERE id = \?`).
					WithArgs("updateduser", 1).
					WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectCommit().WillReturnError(errors.New("commit error"))
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock, cleanup := setupUserTestRepository(t)
			defer cleanup()

			tt.setupMock(mock)

			err := repo.Update(context.Background(), tt.userID, tt.user, tt.settings)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestUserRepository_Delete(t *testing.T) {
	tests := []struct {
		name          string
		userID        int
		setupMock     func(sqlmock.Sqlmock)
		expectedError bool
	}{
		{
			name:   "success",
			userID: 1,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`DELETE FROM users WHERE id = \?`).
					WithArgs(1).
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
			expectedError: false,
		},
		{
			name:   "user not found",
			userID: 999,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`DELETE FROM users WHERE id = \?`).
					WithArgs(999).
					WillReturnResult(sqlmock.NewResult(0, 0))
			},
			expectedError: true,
		},
		{
			name:   "database error",
			userID: 1,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`DELETE FROM users WHERE id = \?`).
					WithArgs(1).
					WillReturnError(errors.New("database error"))
			},
			expectedError: true,
		},
		{
			name:   "error getting rows affected",
			userID: 1,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`DELETE FROM users WHERE id = \?`).
					WithArgs(1).
					WillReturnResult(sqlmock.NewErrorResult(errors.New("rows affected error")))
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock, cleanup := setupUserTestRepository(t)
			defer cleanup()

			tt.setupMock(mock)

			err := repo.Delete(context.Background(), tt.userID)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
