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

// RefreshTokenRepository handles refresh token database operations
type RefreshTokenRepository struct {
	db *database.DB
}

// NewRefreshTokenRepository creates a new refresh token repository
func NewRefreshTokenRepository(db *database.DB) *RefreshTokenRepository {
	return &RefreshTokenRepository{db: db}
}

// Create creates a new refresh token
func (r *RefreshTokenRepository) Create(ctx context.Context, token *models.RefreshToken) error {
	query := `
		INSERT INTO refresh_tokens (
			id, user_id, token_hash, token_family, device_info, device_fingerprint,
			ip_address, user_agent, location_data, expires_at, last_used_at,
			usage_count, is_revoked, created_at, updated_at
		) VALUES (
			:id, :user_id, :token_hash, :token_family, :device_info, :device_fingerprint,
			:ip_address, :user_agent, :location_data, :expires_at, :last_used_at,
			:usage_count, :is_revoked, :created_at, :updated_at
		)`

	_, err := r.db.NamedExecContext(ctx, query, token)
	if err != nil {
		return fmt.Errorf("failed to create refresh token: %w", err)
	}

	return nil
}

// GetByTokenHash retrieves a refresh token by its hash
func (r *RefreshTokenRepository) GetByTokenHash(ctx context.Context, tokenHash string) (*models.RefreshToken, error) {
	var token models.RefreshToken
	query := `
		SELECT * FROM refresh_tokens 
		WHERE token_hash = $1 AND is_revoked = FALSE AND expires_at > NOW()`

	err := r.db.GetContext(ctx, &token, query, tokenHash)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get refresh token: %w", err)
	}

	return &token, nil
}

// GetByID retrieves a refresh token by ID
func (r *RefreshTokenRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.RefreshToken, error) {
	var token models.RefreshToken
	query := `SELECT * FROM refresh_tokens WHERE id = $1`

	err := r.db.GetContext(ctx, &token, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get refresh token by ID: %w", err)
	}

	return &token, nil
}

// GetActiveTokensByUserID retrieves active refresh tokens for a user
func (r *RefreshTokenRepository) GetActiveTokensByUserID(ctx context.Context, userID uuid.UUID) ([]*models.RefreshToken, error) {
	var tokens []*models.RefreshToken
	query := `
		SELECT * FROM refresh_tokens 
		WHERE user_id = $1 AND is_revoked = FALSE AND expires_at > NOW()
		ORDER BY created_at DESC`

	err := r.db.SelectContext(ctx, &tokens, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get active tokens: %w", err)
	}

	return tokens, nil
}

// UpdateLastUsed updates the last used timestamp and increments usage count
func (r *RefreshTokenRepository) UpdateLastUsed(ctx context.Context, tokenHash string) error {
	query := `
		UPDATE refresh_tokens SET
			last_used_at = NOW(),
			usage_count = usage_count + 1,
			updated_at = NOW()
		WHERE token_hash = $1 AND is_revoked = FALSE`

	result, err := r.db.ExecContext(ctx, query, tokenHash)
	if err != nil {
		return fmt.Errorf("failed to update last used: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("refresh token not found or already revoked")
	}

	return nil
}

// RevokeToken revokes a refresh token
func (r *RefreshTokenRepository) RevokeToken(ctx context.Context, tokenHash string, reason string) error {
	query := `
		UPDATE refresh_tokens SET
			is_revoked = TRUE,
			revoked_at = NOW(),
			revoked_reason = $2,
			updated_at = NOW()
		WHERE token_hash = $1`

	result, err := r.db.ExecContext(ctx, query, tokenHash, reason)
	if err != nil {
		return fmt.Errorf("failed to revoke token: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("refresh token not found")
	}

	return nil
}

// RevokeAllUserTokens revokes all refresh tokens for a user
func (r *RefreshTokenRepository) RevokeAllUserTokens(ctx context.Context, userID uuid.UUID, reason string) error {
	query := `
		UPDATE refresh_tokens SET
			is_revoked = TRUE,
			revoked_at = NOW(),
			revoked_reason = $2,
			updated_at = NOW()
		WHERE user_id = $1 AND is_revoked = FALSE`

	_, err := r.db.ExecContext(ctx, query, userID, reason)
	if err != nil {
		return fmt.Errorf("failed to revoke all user tokens: %w", err)
	}

	return nil
}

// RevokeTokenFamily revokes all tokens in a token family (for rotation detection)
func (r *RefreshTokenRepository) RevokeTokenFamily(ctx context.Context, tokenFamily uuid.UUID, reason string) error {
	query := `
		UPDATE refresh_tokens SET
			is_revoked = TRUE,
			revoked_at = NOW(),
			revoked_reason = $2,
			updated_at = NOW()
		WHERE token_family = $1 AND is_revoked = FALSE`

	_, err := r.db.ExecContext(ctx, query, tokenFamily, reason)
	if err != nil {
		return fmt.Errorf("failed to revoke token family: %w", err)
	}

	return nil
}

// CleanupExpiredTokens removes expired refresh tokens
func (r *RefreshTokenRepository) CleanupExpiredTokens(ctx context.Context) (int, error) {
	query := `DELETE FROM refresh_tokens WHERE expires_at < NOW() - INTERVAL '7 days'`

	result, err := r.db.ExecContext(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup expired tokens: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get affected rows: %w", err)
	}

	return int(rowsAffected), nil
}

// GetTokensByDeviceFingerprint retrieves tokens by device fingerprint
func (r *RefreshTokenRepository) GetTokensByDeviceFingerprint(ctx context.Context, fingerprint string) ([]*models.RefreshToken, error) {
	var tokens []*models.RefreshToken
	query := `
		SELECT * FROM refresh_tokens 
		WHERE device_fingerprint = $1 AND is_revoked = FALSE AND expires_at > NOW()
		ORDER BY created_at DESC`

	err := r.db.SelectContext(ctx, &tokens, query, fingerprint)
	if err != nil {
		return nil, fmt.Errorf("failed to get tokens by device fingerprint: %w", err)
	}

	return tokens, nil
}

// GetTokenStats returns statistics about refresh tokens
func (r *RefreshTokenRepository) GetTokenStats(ctx context.Context, userID uuid.UUID) (map[string]int, error) {
	stats := make(map[string]int)

	// Count active tokens
	var activeCount int
	err := r.db.GetContext(ctx, &activeCount, `
		SELECT COUNT(*) FROM refresh_tokens 
		WHERE user_id = $1 AND is_revoked = FALSE AND expires_at > NOW()`, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to count active tokens: %w", err)
	}
	stats["active"] = activeCount

	// Count expired tokens
	var expiredCount int
	err = r.db.GetContext(ctx, &expiredCount, `
		SELECT COUNT(*) FROM refresh_tokens 
		WHERE user_id = $1 AND expires_at <= NOW()`, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to count expired tokens: %w", err)
	}
	stats["expired"] = expiredCount

	// Count revoked tokens
	var revokedCount int
	err = r.db.GetContext(ctx, &revokedCount, `
		SELECT COUNT(*) FROM refresh_tokens 
		WHERE user_id = $1 AND is_revoked = TRUE`, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to count revoked tokens: %w", err)
	}
	stats["revoked"] = revokedCount

	return stats, nil
}

// Update updates a refresh token
func (r *RefreshTokenRepository) Update(ctx context.Context, token *models.RefreshToken) error {
	token.UpdatedAt = time.Now()

	query := `
		UPDATE refresh_tokens SET
			device_info = :device_info,
			device_fingerprint = :device_fingerprint,
			ip_address = :ip_address,
			user_agent = :user_agent,
			location_data = :location_data,
			expires_at = :expires_at,
			last_used_at = :last_used_at,
			usage_count = :usage_count,
			is_revoked = :is_revoked,
			revoked_at = :revoked_at,
			revoked_reason = :revoked_reason,
			updated_at = :updated_at
		WHERE id = :id`

	result, err := r.db.NamedExecContext(ctx, query, token)
	if err != nil {
		return fmt.Errorf("failed to update refresh token: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("refresh token not found")
	}

	return nil
}

// Delete hard deletes a refresh token
func (r *RefreshTokenRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM refresh_tokens WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete refresh token: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("refresh token not found")
	}

	return nil
}
