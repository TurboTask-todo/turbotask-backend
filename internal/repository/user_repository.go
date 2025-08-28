package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"quantumtask-auth-api/internal/database"
	"quantumtask-auth-api/internal/models"

	"github.com/google/uuid"
)

// UserRepository handles user-related database operations
type UserRepository struct {
	db *database.DB
}

// NewUserRepository creates a new user repository
func NewUserRepository(db *database.DB) *UserRepository {
	return &UserRepository{db: db}
}

// Create creates a new user
func (r *UserRepository) Create(ctx context.Context, user *models.User) error {
	query := `
		INSERT INTO users (
			id, email, username, email_verified, first_name, last_name, 
			avatar_url, timezone, locale, preferences, status, created_at, updated_at
		) VALUES (
			:id, :email, :username, :email_verified, :first_name, :last_name,
			:avatar_url, :timezone, :locale, :preferences, :status, :created_at, :updated_at
		)`

	_, err := r.db.NamedExecContext(ctx, query, user)
	if err != nil {
		if database.IsUniqueViolation(err) {
			return fmt.Errorf("user with this email already exists")
		}
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

// GetByID retrieves a user by ID
func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	var user models.User
	query := `
		SELECT * FROM users 
		WHERE id = $1 AND deleted_at IS NULL`

	err := r.db.GetContext(ctx, &user, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get user by ID: %w", err)
	}

	return &user, nil
}

// GetByEmail retrieves a user by email
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	query := `
		SELECT * FROM users 
		WHERE email = $1 AND deleted_at IS NULL`

	err := r.db.GetContext(ctx, &user, query, email)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}

	return &user, nil
}

// GetByUsername retrieves a user by username
func (r *UserRepository) GetByUsername(ctx context.Context, username string) (*models.User, error) {
	var user models.User
	query := `
		SELECT * FROM users 
		WHERE username = $1 AND deleted_at IS NULL`

	err := r.db.GetContext(ctx, &user, query, username)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get user by username: %w", err)
	}

	return &user, nil
}

// Update updates a user
func (r *UserRepository) Update(ctx context.Context, user *models.User) error {
	user.UpdatedAt = time.Now()

	query := `
		UPDATE users SET
			email = :email,
			username = :username,
			email_verified = :email_verified,
			first_name = :first_name,
			last_name = :last_name,
			avatar_url = :avatar_url,
			timezone = :timezone,
			locale = :locale,
			preferences = :preferences,
			status = :status,
			last_login_at = :last_login_at,
			failed_login_attempts = :failed_login_attempts,
			account_locked_until = :account_locked_until,
			password_changed_at = :password_changed_at,
			updated_at = :updated_at
		WHERE id = :id AND deleted_at IS NULL`

	result, err := r.db.NamedExecContext(ctx, query, user)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("user not found or already deleted")
	}

	return nil
}

// IncrementFailedLoginAttempts increments the failed login attempts counter
func (r *UserRepository) IncrementFailedLoginAttempts(ctx context.Context, userID uuid.UUID) error {
	query := `
		UPDATE users SET
			failed_login_attempts = failed_login_attempts + 1,
			updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL`

	_, err := r.db.ExecContext(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("failed to increment failed login attempts: %w", err)
	}

	return nil
}

// LockAccount locks a user account until the specified time
func (r *UserRepository) LockAccount(ctx context.Context, userID uuid.UUID, lockUntil time.Time) error {
	query := `
		UPDATE users SET
			account_locked_until = $2,
			updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL`

	_, err := r.db.ExecContext(ctx, query, userID, lockUntil)
	if err != nil {
		return fmt.Errorf("failed to lock user account: %w", err)
	}

	return nil
}

// UnlockAccount unlocks a user account and resets failed login attempts
func (r *UserRepository) UnlockAccount(ctx context.Context, userID uuid.UUID) error {
	query := `
		UPDATE users SET
			account_locked_until = NULL,
			failed_login_attempts = 0,
			updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL`

	_, err := r.db.ExecContext(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("failed to unlock user account: %w", err)
	}

	return nil
}

// UpdateLastLogin updates the user's last login timestamp
func (r *UserRepository) UpdateLastLogin(ctx context.Context, userID uuid.UUID) error {
	query := `
		UPDATE users SET
			last_login_at = NOW(),
			failed_login_attempts = 0,
			updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL`

	_, err := r.db.ExecContext(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("failed to update last login: %w", err)
	}

	return nil
}

// VerifyEmail marks a user's email as verified
func (r *UserRepository) VerifyEmail(ctx context.Context, userID uuid.UUID) error {
	query := `
		UPDATE users SET
			email_verified = TRUE,
			updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL`

	_, err := r.db.ExecContext(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("failed to verify email: %w", err)
	}

	return nil
}

// ChangePassword updates the password changed timestamp
func (r *UserRepository) ChangePassword(ctx context.Context, userID uuid.UUID) error {
	query := `
		UPDATE users SET
			password_changed_at = NOW(),
			updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL`

	_, err := r.db.ExecContext(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("failed to update password changed timestamp: %w", err)
	}

	return nil
}

// SoftDelete soft deletes a user
func (r *UserRepository) SoftDelete(ctx context.Context, userID uuid.UUID) error {
	query := `SELECT soft_delete_user($1)`

	var success bool
	err := r.db.GetContext(ctx, &success, query, userID)
	if err != nil {
		return fmt.Errorf("failed to soft delete user: %w", err)
	}

	if !success {
		return fmt.Errorf("user not found or already deleted")
	}

	return nil
}

// Restore restores a soft-deleted user
func (r *UserRepository) Restore(ctx context.Context, userID uuid.UUID) error {
	query := `SELECT restore_user($1)`

	var success bool
	err := r.db.GetContext(ctx, &success, query, userID)
	if err != nil {
		return fmt.Errorf("failed to restore user: %w", err)
	}

	if !success {
		return fmt.Errorf("user not found or not deleted")
	}

	return nil
}

// List retrieves a paginated list of users
func (r *UserRepository) List(ctx context.Context, page, pageSize int, status string) ([]*models.User, int, error) {
	limit, offset := database.BuildLimitOffset(page, pageSize)

	// Build WHERE clause based on status filter
	whereClause := "WHERE deleted_at IS NULL"
	args := []interface{}{}
	argIndex := 1

	if status != "" {
		whereClause += fmt.Sprintf(" AND status = $%d", argIndex)
		args = append(args, status)
		argIndex++
	}

	// Get total count
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM users %s", whereClause)
	var totalCount int
	err := r.db.GetContext(ctx, &totalCount, countQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get user count: %w", err)
	}

	// Get users
	query := fmt.Sprintf(`
		SELECT * FROM users %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d`,
		whereClause, argIndex, argIndex+1)

	args = append(args, limit, offset)

	var users []*models.User
	err = r.db.SelectContext(ctx, &users, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get users: %w", err)
	}

	return users, totalCount, nil
}

// GetAuthSummary retrieves user authentication summary
func (r *UserRepository) GetAuthSummary(ctx context.Context, userID uuid.UUID) (*models.UserAuthSummary, error) {
	var summary models.UserAuthSummary
	query := `SELECT * FROM v_user_auth_summary WHERE id = $1`

	err := r.db.GetContext(ctx, &summary, query, userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get user auth summary: %w", err)
	}

	return &summary, nil
}

// ExistsWithEmail checks if a user exists with the given email
func (r *UserRepository) ExistsWithEmail(ctx context.Context, email string) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM users WHERE email = $1 AND deleted_at IS NULL)`

	err := r.db.GetContext(ctx, &exists, query, email)
	if err != nil {
		return false, fmt.Errorf("failed to check if user exists with email: %w", err)
	}

	return exists, nil
}

// ExistsWithUsername checks if a user exists with the given username
func (r *UserRepository) ExistsWithUsername(ctx context.Context, username string) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM users WHERE username = $1 AND deleted_at IS NULL)`

	err := r.db.GetContext(ctx, &exists, query, username)
	if err != nil {
		return false, fmt.Errorf("failed to check if user exists with username: %w", err)
	}

	return exists, nil
}
