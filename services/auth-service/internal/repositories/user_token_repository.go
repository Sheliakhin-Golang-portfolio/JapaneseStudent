package repositories

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/japanesestudent/auth-service/internal/models"
	"go.uber.org/zap"
)

// userTokenRepository implements UserTokenRepository
type userTokenRepository struct {
	db     *sql.DB
	logger *zap.Logger
}

// NewUserTokenRepository creates a new user token repository
func NewUserTokenRepository(db *sql.DB, logger *zap.Logger) *userTokenRepository {
	return &userTokenRepository{
		db:     db,
		logger: logger,
	}
}

// Create inserts a new user token into the database
func (r *userTokenRepository) Create(ctx context.Context, userToken *models.UserToken) error {
	query := `
		INSERT INTO user_tokens (user_id, token)
		VALUES (?, ?)
	`

	if _, err := r.db.ExecContext(ctx, query, userToken.UserID, userToken.Token); err != nil {
		r.logger.Error("failed to create user token", zap.Error(err))
		return fmt.Errorf("failed to create user token: %w", err)
	}

	return nil
}

// GetByToken retrieves a user token by token string
func (r *userTokenRepository) GetByToken(ctx context.Context, token string) (*models.UserToken, error) {
	query := `
		SELECT id, user_id, token
		FROM user_tokens
		WHERE token = ?
		LIMIT 1
	`

	userToken := &models.UserToken{}
	err := r.db.QueryRowContext(ctx, query, token).Scan(
		&userToken.ID,
		&userToken.UserID,
		&userToken.Token,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("token not found")
	}
	if err != nil {
		r.logger.Error("failed to get user token by token", zap.Error(err))
		return nil, fmt.Errorf("failed to get user token by token: %w", err)
	}

	return userToken, nil
}

// UpdateToken updates an existing token record with a new token
func (r *userTokenRepository) UpdateToken(ctx context.Context, oldToken, newToken string, userID int) error {
	query := `
		UPDATE user_tokens
		SET token = ?
		WHERE token = ? AND user_id = ?
	`

	result, err := r.db.ExecContext(ctx, query, newToken, oldToken, userID)
	if err != nil {
		r.logger.Error("failed to update user token", zap.Error(err))
		return fmt.Errorf("failed to update user token: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		r.logger.Error("failed to get rows affected", zap.Error(err))
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("token not found or user mismatch")
	}

	return nil
}

// DeleteByToken deletes a token record by token string
func (r *userTokenRepository) DeleteByToken(ctx context.Context, token string) error {
	query := `DELETE FROM user_tokens WHERE token = ?`

	if _, err := r.db.ExecContext(ctx, query, token); err != nil {
		r.logger.Error("failed to delete user token", zap.Error(err))
		return fmt.Errorf("failed to delete user token: %w", err)
	}

	return nil
}
