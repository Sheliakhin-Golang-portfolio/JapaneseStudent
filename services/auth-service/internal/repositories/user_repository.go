package repositories

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/japanesestudent/auth-service/internal/models"
	"go.uber.org/zap"
)

// userRepository implements UserRepository
type userRepository struct {
	db     *sql.DB
	logger *zap.Logger
}

// NewUserRepository creates a new user repository
func NewUserRepository(db *sql.DB, logger *zap.Logger) *userRepository {
	return &userRepository{
		db:     db,
		logger: logger,
	}
}

// Create inserts a new user into the database
func (r *userRepository) Create(ctx context.Context, user *models.User) error {
	query := `
		INSERT INTO users (username, email, password_hash, role)
		VALUES (?, ?, ?, ?)
	`

	result, err := r.db.ExecContext(ctx, query, user.Username, user.Email, user.PasswordHash, user.Role)
	if err != nil {
		r.logger.Error("failed to create user", zap.Error(err))
		return fmt.Errorf("failed to create user: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		r.logger.Error("failed to get last insert id", zap.Error(err))
		return fmt.Errorf("failed to get last insert id: %w", err)
	}

	user.ID = int(id)
	return nil
}

// GetByEmailOrUsername retrieves a user by email or username
func (r *userRepository) GetByEmailOrUsername(ctx context.Context, login string) (*models.User, error) {
	query := `
		SELECT id, username, email, password_hash, role
		FROM users
		WHERE email = ? OR username = ?
		LIMIT 1
	`

	user := &models.User{}
	err := r.db.QueryRowContext(ctx, query, login, login).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.Role,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		r.logger.Error("failed to get user by email or username", zap.Error(err), zap.String("login", login))
		return nil, fmt.Errorf("failed to get user by email or username: %w", err)
	}

	return user, nil
}

// ExistsByEmail checks if a user exists with the given email
func (r *userRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	query := `SELECT EXISTS(SELECT * FROM users WHERE email = ?)`

	var exists bool
	err := r.db.QueryRowContext(ctx, query, email).Scan(&exists)
	if err != nil {
		r.logger.Error("failed to check email existence", zap.Error(err), zap.String("email", email))
		return false, fmt.Errorf("failed to check email existence: %w", err)
	}

	return exists, nil
}

// ExistsByUsername checks if a user exists with the given username
func (r *userRepository) ExistsByUsername(ctx context.Context, username string) (bool, error) {
	query := `SELECT EXISTS(SELECT * FROM users WHERE username = ?)`

	var exists bool
	err := r.db.QueryRowContext(ctx, query, username).Scan(&exists)
	if err != nil {
		r.logger.Error("failed to check username existence", zap.Error(err), zap.String("username", username))
		return false, fmt.Errorf("failed to check username existence: %w", err)
	}

	return exists, nil
}
