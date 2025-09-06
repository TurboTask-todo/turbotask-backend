package repository

import (
	"context"
	"fmt"
	"time"

	"quantumtask-auth-api/internal/models"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// OTPRepository handles OTP database operations
type OTPRepository struct {
	db *sqlx.DB
}

// NewOTPRepository creates a new OTP repository
func NewOTPRepository(db *sqlx.DB) *OTPRepository {
	return &OTPRepository{
		db: db,
	}
}

// Create creates a new OTP record
func (r *OTPRepository) Create(ctx context.Context, otp *models.OTP) error {
	otp.ID = uuid.New()
	otp.CreatedAt = time.Now()
	otp.IsVerified = false
	otp.AttemptCount = 0

	query := `
		INSERT INTO otps (
			id, email, code, purpose, is_verified, attempt_count, 
			expires_at, created_at
		) VALUES (
			:id, :email, :code, :purpose, :is_verified, :attempt_count,
			:expires_at, :created_at
		)`

	_, err := r.db.NamedExecContext(ctx, query, otp)
	if err != nil {
		return fmt.Errorf("failed to create OTP: %w", err)
	}

	return nil
}

// GetActiveOTP retrieves the most recent active OTP for email and purpose
func (r *OTPRepository) GetActiveOTP(ctx context.Context, email string, purpose models.OTPPurpose) (*models.OTP, error) {
	query := `
		SELECT id, email, code, purpose, is_verified, attempt_count,
		       expires_at, created_at, verified_at
		FROM otps 
		WHERE email = $1 AND purpose = $2 
		      AND is_verified = false 
		      AND expires_at > CURRENT_TIMESTAMP
		ORDER BY created_at DESC
		LIMIT 1`

	var otp models.OTP
	err := r.db.GetContext(ctx, &otp, query, email, purpose)
	if err != nil {
		return nil, fmt.Errorf("failed to get active OTP: %w", err)
	}

	return &otp, nil
}

// GetByID retrieves an OTP by ID
func (r *OTPRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.OTP, error) {
	query := `
		SELECT id, email, code, purpose, is_verified, attempt_count,
		       expires_at, created_at, verified_at
		FROM otps 
		WHERE id = $1`

	var otp models.OTP
	err := r.db.GetContext(ctx, &otp, query, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get OTP by ID: %w", err)
	}

	return &otp, nil
}

// VerifyOTP marks an OTP as verified
func (r *OTPRepository) VerifyOTP(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	query := `
		UPDATE otps 
		SET is_verified = true, verified_at = $1
		WHERE id = $2 AND is_verified = false AND expires_at > CURRENT_TIMESTAMP`

	result, err := r.db.ExecContext(ctx, query, now, id)
	if err != nil {
		return fmt.Errorf("failed to verify OTP: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("no valid OTP found to verify")
	}

	return nil
}

// IncrementAttempt increments the attempt count for an OTP
func (r *OTPRepository) IncrementAttempt(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE otps 
		SET attempt_count = attempt_count + 1
		WHERE id = $1`

	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to increment OTP attempt: %w", err)
	}

	return nil
}

// InvalidateOTPs marks all active OTPs for email and purpose as expired
func (r *OTPRepository) InvalidateOTPs(ctx context.Context, email string, purpose models.OTPPurpose) error {
	query := `
		UPDATE otps 
		SET expires_at = CURRENT_TIMESTAMP
		WHERE email = $1 AND purpose = $2 AND is_verified = false AND expires_at > CURRENT_TIMESTAMP`

	_, err := r.db.ExecContext(ctx, query, email, purpose)
	if err != nil {
		return fmt.Errorf("failed to invalidate OTPs: %w", err)
	}

	return nil
}

// DeleteExpired deletes expired OTPs (for cleanup)
func (r *OTPRepository) DeleteExpired(ctx context.Context) error {
	query := `
		DELETE FROM otps 
		WHERE expires_at < CURRENT_TIMESTAMP - INTERVAL '1 day'`

	_, err := r.db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to delete expired OTPs: %w", err)
	}

	return nil
}

// GetRecentAttempts gets the number of OTP requests in the last time period
func (r *OTPRepository) GetRecentAttempts(ctx context.Context, email string, purpose models.OTPPurpose, since time.Time) (int, error) {
	query := `
		SELECT COUNT(*) 
		FROM otps 
		WHERE email = $1 AND purpose = $2 AND created_at > $3`

	var count int
	err := r.db.GetContext(ctx, &count, query, email, purpose, since)
	if err != nil {
		return 0, fmt.Errorf("failed to get recent attempts: %w", err)
	}

	return count, nil
}
