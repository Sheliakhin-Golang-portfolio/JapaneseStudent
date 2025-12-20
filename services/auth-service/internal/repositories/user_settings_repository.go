package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/japanesestudent/auth-service/internal/models"
)

// userSettingsRepository implements UserSettingsRepository
type userSettingsRepository struct {
	db *sql.DB
}

// NewUserSettingsRepository creates a new user settings repository
func NewUserSettingsRepository(db *sql.DB) *userSettingsRepository {
	return &userSettingsRepository{
		db: db,
	}
}

// Create inserts a new user settings record
func (r *userSettingsRepository) Create(ctx context.Context, userId int) error {
	query := `
		INSERT INTO user_settings (user_id)
		VALUES (?)
	`

	if _, err := r.db.ExecContext(ctx, query, userId); err != nil {
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
		return nil, fmt.Errorf("failed to get user settings: %w", err)
	}

	userSettings.Language = models.Language(languageStr)
	return userSettings, nil
}

// Update updates user settings for a given user ID
func (r *userSettingsRepository) Update(ctx context.Context, userId int, settings *models.UserSettings) error {
	// Build SET clause
	setClauses := []string{}
	args := []any{}
	if settings.NewWordCount != 0 {
		setClauses = append(setClauses, "new_word_count = ?")
		args = append(args, settings.NewWordCount)
	}
	if settings.OldWordCount != 0 {
		setClauses = append(setClauses, "old_word_count = ?")
		args = append(args, settings.OldWordCount)
	}
	if settings.AlphabetLearnCount != 0 {
		setClauses = append(setClauses, "alphabet_learn_count = ?")
		args = append(args, settings.AlphabetLearnCount)
	}
	if settings.Language != "" {
		setClauses = append(setClauses, "language = ?")
		args = append(args, string(settings.Language))
	}
	if len(setClauses) == 0 {
		return fmt.Errorf("no fields to update")
	}

	args = append(args, userId)
	query := fmt.Sprintf(`
		UPDATE user_settings
		SET %s
		WHERE user_id = ?
	`, strings.Join(setClauses, ", "))

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update user settings: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("user settings not found")
	}

	return nil
}
