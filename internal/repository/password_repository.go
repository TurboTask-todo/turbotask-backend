package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"macwrite-auth-api/internal/database"
	"macwrite-auth-api/internal/models"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// PasswordRepository handles password-related database operations
type PasswordRepository struct {
	db *database.DB
}

// NewPasswordRepository creates a new password repository
func NewPasswordRepository(db *database.DB) *PasswordRepository {
	return &PasswordRepository{db: db}
}

// Create creates a new password record
func (r *PasswordRepository) Create(ctx context.Context, password *models.UserPassword) error {
	query := `
		INSERT INTO user_passwords (
			id, user_id, password_hash, salt, algorithm, cost_params, 
			version, created_at, updated_at, expires_at, is_temporary, is_active
		) VALUES (
			:id, :user_id, :password_hash, :salt, :algorithm, :cost_params,
			:version, :created_at, :updated_at, :expires_at, :is_temporary, :is_active
		)`

	_, err := r.db.NamedExecContext(ctx, query, password)
	if err != nil {
		return fmt.Errorf("failed to create password: %w", err)
	}

	return nil
}

// GetActiveByUserID retrieves the active password for a user
func (r *PasswordRepository) GetActiveByUserID(ctx context.Context, userID uuid.UUID) (*models.UserPassword, error) {
	var password models.UserPassword
	query := `
		SELECT * FROM user_passwords 
		WHERE user_id = $1 AND is_active = TRUE
		ORDER BY created_at DESC
		LIMIT 1`

	err := r.db.GetContext(ctx, &password, query, userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get active password: %w", err)
	}

	return &password, nil
}

// GetByID retrieves a password by ID
func (r *PasswordRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.UserPassword, error) {
	var password models.UserPassword
	query := `SELECT * FROM user_passwords WHERE id = $1`

	err := r.db.GetContext(ctx, &password, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get password by ID: %w", err)
	}

	return &password, nil
}

// Update updates a password record
func (r *PasswordRepository) Update(ctx context.Context, password *models.UserPassword) error {
	password.UpdatedAt = time.Now()

	query := `
		UPDATE user_passwords SET
			password_hash = :password_hash,
			salt = :salt,
			algorithm = :algorithm,
			cost_params = :cost_params,
			version = :version,
			expires_at = :expires_at,
			is_temporary = :is_temporary,
			is_active = :is_active,
			updated_at = :updated_at
		WHERE id = :id`

	result, err := r.db.NamedExecContext(ctx, query, password)
	if err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("password not found")
	}

	return nil
}

// DeactivateAllForUser deactivates all passwords for a user
func (r *PasswordRepository) DeactivateAllForUser(ctx context.Context, userID uuid.UUID) error {
	query := `
		UPDATE user_passwords SET
			is_active = FALSE,
			updated_at = NOW()
		WHERE user_id = $1 AND is_active = TRUE`

	_, err := r.db.ExecContext(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("failed to deactivate passwords: %w", err)
	}

	return nil
}

// SetAsActive sets a password as active and deactivates others for the user
func (r *PasswordRepository) SetAsActive(ctx context.Context, passwordID uuid.UUID, userID uuid.UUID) error {
	return r.db.WithTransaction(func(tx *sqlx.Tx) error {
		// Deactivate all other passwords for the user
		_, err := tx.ExecContext(ctx, `
			UPDATE user_passwords SET
				is_active = FALSE,
				updated_at = NOW()
			WHERE user_id = $1 AND id != $2`, userID, passwordID)
		if err != nil {
			return fmt.Errorf("failed to deactivate other passwords: %w", err)
		}

		// Activate the specified password
		_, err = tx.ExecContext(ctx, `
			UPDATE user_passwords SET
				is_active = TRUE,
				updated_at = NOW()
			WHERE id = $1`, passwordID)
		if err != nil {
			return fmt.Errorf("failed to activate password: %w", err)
		}

		return nil
	})
}

// CreateAndSetActive creates a new password and sets it as active
func (r *PasswordRepository) CreateAndSetActive(ctx context.Context, password *models.UserPassword) error {
	return r.db.WithTransaction(func(tx *sqlx.Tx) error {
		// First deactivate all existing passwords for the user
		_, err := tx.ExecContext(ctx, `
			UPDATE user_passwords SET
				is_active = FALSE,
				updated_at = NOW()
			WHERE user_id = $1`, password.UserID)
		if err != nil {
			return fmt.Errorf("failed to deactivate existing passwords: %w", err)
		}

		// Create the new password as active
		password.IsActive = true
		query := `
			INSERT INTO user_passwords (
				id, user_id, password_hash, salt, algorithm, cost_params, 
				version, created_at, updated_at, expires_at, is_temporary, is_active
			) VALUES (
				:id, :user_id, :password_hash, :salt, :algorithm, :cost_params,
				:version, :created_at, :updated_at, :expires_at, :is_temporary, :is_active
			)`

		_, err = tx.NamedExecContext(ctx, query, password)
		if err != nil {
			return fmt.Errorf("failed to create new password: %w", err)
		}

		return nil
	})
}

// Delete hard deletes a password record
func (r *PasswordRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM user_passwords WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete password: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("password not found")
	}

	return nil
}

// GetPasswordHistory retrieves password history for a user
func (r *PasswordRepository) GetPasswordHistory(ctx context.Context, userID uuid.UUID, limit int) ([]*models.UserPassword, error) {
	if limit <= 0 {
		limit = 10
	}

	var passwords []*models.UserPassword
	query := `
		SELECT * FROM user_passwords 
		WHERE user_id = $1 
		ORDER BY created_at DESC 
		LIMIT $2`

	err := r.db.SelectContext(ctx, &passwords, query, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get password history: %w", err)
	}

	return passwords, nil
}

// IsPasswordReused checks if a password hash has been used before by the user
func (r *PasswordRepository) IsPasswordReused(ctx context.Context, userID uuid.UUID, passwordHash string, limit int) (bool, error) {
	if limit <= 0 {
		limit = 5 // Default to checking last 5 passwords
	}

	var count int
	query := `
		SELECT COUNT(*) FROM (
			SELECT password_hash FROM user_passwords 
			WHERE user_id = $1 
			ORDER BY created_at DESC 
			LIMIT $3
		) recent_passwords
		WHERE password_hash = $2`

	err := r.db.GetContext(ctx, &count, query, userID, passwordHash, limit)
	if err != nil {
		return false, fmt.Errorf("failed to check password reuse: %w", err)
	}

	return count > 0, nil
}

// GetExpiredPasswords retrieves expired passwords
func (r *PasswordRepository) GetExpiredPasswords(ctx context.Context) ([]*models.UserPassword, error) {
	var passwords []*models.UserPassword
	query := `
		SELECT * FROM user_passwords 
		WHERE expires_at IS NOT NULL 
		AND expires_at < NOW() 
		AND is_active = TRUE`

	err := r.db.SelectContext(ctx, &passwords, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get expired passwords: %w", err)
	}

	return passwords, nil
}

// CountActivePasswords counts active passwords for a user
func (r *PasswordRepository) CountActivePasswords(ctx context.Context, userID uuid.UUID) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM user_passwords WHERE user_id = $1 AND is_active = TRUE`

	err := r.db.GetContext(ctx, &count, query, userID)
	if err != nil {
		return 0, fmt.Errorf("failed to count active passwords: %w", err)
	}

	return count, nil
}

// HasPassword checks if a user has any password (active or inactive)
func (r *PasswordRepository) HasPassword(ctx context.Context, userID uuid.UUID) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM user_passwords WHERE user_id = $1)`

	err := r.db.GetContext(ctx, &exists, query, userID)
	if err != nil {
		return false, fmt.Errorf("failed to check if user has password: %w", err)
	}

	return exists, nil
}
