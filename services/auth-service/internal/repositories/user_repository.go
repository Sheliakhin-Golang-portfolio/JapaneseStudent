package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/japanesestudent/auth-service/internal/models"
)

// userRepository implements UserRepository
type userRepository struct {
	db *sql.DB
}

// NewUserRepository creates a new user repository
func NewUserRepository(db *sql.DB) *userRepository {
	return &userRepository{
		db: db,
	}
}

// Create inserts a new user into the database
func (r *userRepository) Create(ctx context.Context, user *models.User) error {
	query := `
		INSERT INTO users (username, email, password_hash, role, avatar, active)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	result, err := r.db.ExecContext(ctx, query, user.Username, user.Email, user.PasswordHash, user.Role, user.Avatar, user.Active)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}

	user.ID = int(id)
	return nil
}

// GetByEmailOrUsername retrieves a user by email or username
func (r *userRepository) GetByEmailOrUsername(ctx context.Context, login string) (*models.User, error) {
	query := `
		SELECT id, username, email, password_hash, role, avatar, active
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
		&user.Avatar,
		&user.Active,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
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
		return false, fmt.Errorf("failed to check username existence: %w", err)
	}

	return exists, nil
}

// GetByID retrieves a user by ID
func (r *userRepository) GetByID(ctx context.Context, userID int) (*models.User, error) {
	query := `
		SELECT username, email, password_hash, role, avatar, active
		FROM users
		WHERE id = ?
		LIMIT 1
	`

	user := &models.User{}
	err := r.db.QueryRowContext(ctx, query, userID).Scan(
		&user.Username,
		&user.Email,
		&user.PasswordHash,
		&user.Role,
		&user.Avatar,
		&user.Active,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user by ID: %w", err)
	}

	user.ID = userID
	return user, nil
}

// GetAll retrieves a paginated list of users with optional role and search filters
func (r *userRepository) GetAll(ctx context.Context, page, count int, role *models.Role, search string) ([]models.User, error) {
	// Build WHERE clause
	var whereConditions []string
	var args []any

	if role != nil {
		whereConditions = append(whereConditions, "role = ?")
		args = append(args, *role)
	}

	if search != "" {
		whereConditions = append(whereConditions, "(email LIKE ? OR username LIKE ?)")
		whereValue := "%" + search + "%"
		args = append(args, whereValue, whereValue)
	}

	whereClause := ""
	if len(whereConditions) > 0 {
		whereClause = "WHERE " + strings.Join(whereConditions, " AND ")
	}

	// Calculate offset
	offset := (page - 1) * count

	query := fmt.Sprintf(`
		SELECT id, username, email, role, avatar
		FROM users
		%s
		ORDER BY email
		LIMIT ? OFFSET ?
	`, whereClause)

	args = append(args, count, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query users: %w", err)
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var user models.User
		err := rows.Scan(
			&user.ID,
			&user.Username,
			&user.Email,
			&user.Role,
			&user.Avatar,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return users, nil
}

// Update updates both user fields and settings
func (r *userRepository) Update(ctx context.Context, userID int, user *models.User, settings *models.UserSettings, active *bool) error {
	// Begin transaction
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Build SET clause for user
	if user != nil {
		setClauses := []string{}
		args := []any{}
		if user.Username != "" {
			setClauses = append(setClauses, "username = ?")
			args = append(args, user.Username)
		}
		if user.Email != "" {
			setClauses = append(setClauses, "email = ?")
			args = append(args, user.Email)
		}
		if user.Role != 0 {
			setClauses = append(setClauses, "role = ?")
			args = append(args, user.Role)
		}
		if user.Avatar != "" {
			setClauses = append(setClauses, "avatar = ?")
			args = append(args, user.Avatar)
		}
		// Update active status if provided
		if active != nil {
			setClauses = append(setClauses, "active = ?")
			args = append(args, *active)
		}

		if len(setClauses) != 0 {
			args = append(args, userID)
			query := fmt.Sprintf(`
				UPDATE users
				SET %s
				WHERE id = ?
				`, strings.Join(setClauses, ", "))

			result, err := tx.ExecContext(ctx, query, args...)
			if err != nil {
				return fmt.Errorf("failed to update user: %w", err)
			}

			rowsAffected, err := result.RowsAffected()
			if err != nil {
				return fmt.Errorf("failed to get rows affected: %w", err)
			}

			if rowsAffected == 0 {
				return fmt.Errorf("user not found")
			}
		}
	}

	// Build SET clause for settings
	if settings != nil {
		setClauses := []string{}
		args := []any{}
		if settings.Language != "" {
			setClauses = append(setClauses, "language = ?")
			args = append(args, settings.Language)
		}
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

		if len(setClauses) != 0 {
			args = append(args, userID)
			query := fmt.Sprintf(`
		UPDATE user_settings
		SET %s
		WHERE user_id = ?
		`, strings.Join(setClauses, ", "))

			result, err := tx.ExecContext(ctx, query, args...)
			if err != nil {
				return fmt.Errorf("failed to update settings: %w", err)
			}

			rowsAffected, err := result.RowsAffected()
			if err != nil {
				return fmt.Errorf("failed to get rows affected: %w", err)
			}

			if rowsAffected == 0 {
				return fmt.Errorf("settings not found")
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// Delete deletes a user by ID
func (r *userRepository) Delete(ctx context.Context, userID int) error {
	query := `DELETE FROM users WHERE id = ?`

	result, err := r.db.ExecContext(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("user not found")
	}

	return nil
}

// GetTutorsList retrieves a list of tutors (only ID and username)
func (r *userRepository) GetTutorsList(ctx context.Context) ([]models.TutorListItem, error) {
	query := `
		SELECT id, username
		FROM users
		WHERE role = 2
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get tutors list: %w", err)
	}
	defer rows.Close()

	var tutors []models.TutorListItem
	for rows.Next() {
		var tutor models.TutorListItem
		err := rows.Scan(&tutor.ID, &tutor.Username)
		if err != nil {
			return nil, fmt.Errorf("failed to scan tutor: %w", err)
		}
		tutors = append(tutors, tutor)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return tutors, nil
}

// UpdateActive updates the active status of a user
func (r *userRepository) UpdateActive(ctx context.Context, userID int, active bool) error {
	query := `
		UPDATE users
		SET active = ?
		WHERE id = ?
	`

	result, err := r.db.ExecContext(ctx, query, active, userID)
	if err != nil {
		return fmt.Errorf("failed to update user active status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("user not found")
	}

	return nil
}

// UpdatePasswordHash updates the password hash for a user
func (r *userRepository) UpdatePasswordHash(ctx context.Context, userID int, passwordHash string) error {
	query := `
		UPDATE users
		SET password_hash = ?
		WHERE id = ?
	`

	result, err := r.db.ExecContext(ctx, query, passwordHash, userID)
	if err != nil {
		return fmt.Errorf("failed to update password hash: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("user not found")
	}

	return nil
}
