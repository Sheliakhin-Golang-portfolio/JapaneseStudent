package repositories

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/japanesestudent/auth-service/internal/models"
	"go.uber.org/zap"
)

// userSettingsRepository implements UserSettingsRepository
type userSettingsRepository struct {
	db     *sql.DB
	logger *zap.Logger
}

// NewUserSettingsRepository creates a new user settings repository
func NewUserSettingsRepository(db *sql.DB, logger *zap.Logger) *userSettingsRepository {
	return &userSettingsRepository{
		db:     db,
		logger: logger,
	}
}

// Create inserts a new user settings record
func (r *userSettingsRepository) Create(ctx context.Context, userSettings *models.UserSettings) error {
	query := `
		INSERT INTO user_settings (user_id)
		VALUES (?)
	`

	if _, err := r.db.ExecContext(ctx, query, userSettings.UserID); err != nil {
		r.logger.Error("failed to create user settings", zap.Error(err))
		return fmt.Errorf("failed to create user settings: %w", err)
	}
	return nil
}

// GetByUserId retrieves user settings by user ID
func (r *userSettingsRepository) GetByUserId(ctx context.Context, userId int) (*models.UserSettings, error) {
	query := `
		SELECT id, user_id, new_word_count, old_word_count, alphabet_learn_count, language
		FROM user_settings
		WHERE user_id = ?
		LIMIT 1
	`

	userSettings := &models.UserSettings{}
	var languageStr string
	err := r.db.QueryRowContext(ctx, query, userId).Scan(
		&userSettings.ID,
		&userSettings.UserID,
		&userSettings.NewWordCount,
		&userSettings.OldWordCount,
		&userSettings.AlphabetLearnCount,
		&languageStr,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user settings not found")
	}
	if err != nil {
		r.logger.Error("failed to get user settings by user ID", zap.Error(err), zap.Int("userId", userId))
		return nil, fmt.Errorf("failed to get user settings: %w", err)
	}

	userSettings.Language = models.Language(languageStr)
	return userSettings, nil
}

// Update updates user settings for a given user ID
func (r *userSettingsRepository) Update(ctx context.Context, userId int, settings *models.UserSettings) error {
	query := `
		UPDATE user_settings
		SET new_word_count = ?, old_word_count = ?, alphabet_learn_count = ?, language = ?
		WHERE user_id = ?
	`

	result, err := r.db.ExecContext(ctx, query,
		settings.NewWordCount,
		settings.OldWordCount,
		settings.AlphabetLearnCount,
		string(settings.Language),
		userId,
	)
	if err != nil {
		r.logger.Error("failed to update user settings", zap.Error(err), zap.Int("userId", userId))
		return fmt.Errorf("failed to update user settings: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		r.logger.Error("failed to get rows affected", zap.Error(err))
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("user settings not found")
	}

	return nil
}
