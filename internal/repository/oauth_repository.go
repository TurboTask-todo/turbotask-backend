package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"macwrite-auth-api/internal/database"
	"macwrite-auth-api/internal/models"

	"github.com/google/uuid"
)

// OAuthRepository handles OAuth provider database operations
type OAuthRepository struct {
	db *database.DB
}

// NewOAuthRepository creates a new OAuth repository
func NewOAuthRepository(db *database.DB) *OAuthRepository {
	return &OAuthRepository{db: db}
}

// Create creates a new OAuth provider record
func (r *OAuthRepository) Create(ctx context.Context, provider *models.OAuthProvider) error {
	query := `
		INSERT INTO oauth_providers (
			id, user_id, provider, provider_user_id, provider_email, provider_username,
			provider_data, access_token, refresh_token, token_type, scope, expires_at,
			is_active, created_at, updated_at
		) VALUES (
			:id, :user_id, :provider, :provider_user_id, :provider_email, :provider_username,
			:provider_data, :access_token, :refresh_token, :token_type, :scope, :expires_at,
			:is_active, :created_at, :updated_at
		)`

	_, err := r.db.NamedExecContext(ctx, query, provider)
	if err != nil {
		if database.IsUniqueViolation(err) {
			return fmt.Errorf("OAuth provider already linked to this user")
		}
		return fmt.Errorf("failed to create OAuth provider: %w", err)
	}

	return nil
}

// GetByUserAndProvider retrieves OAuth provider by user ID and provider name
func (r *OAuthRepository) GetByUserAndProvider(ctx context.Context, userID uuid.UUID, provider string) (*models.OAuthProvider, error) {
	var oauthProvider models.OAuthProvider
	query := `
		SELECT * FROM oauth_providers 
		WHERE user_id = $1 AND provider = $2 AND is_active = TRUE`

	err := r.db.GetContext(ctx, &oauthProvider, query, userID, provider)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get OAuth provider: %w", err)
	}

	return &oauthProvider, nil
}

// GetByProviderAndProviderUserID retrieves OAuth provider by provider name and provider user ID
func (r *OAuthRepository) GetByProviderAndProviderUserID(ctx context.Context, provider, providerUserID string) (*models.OAuthProvider, error) {
	var oauthProvider models.OAuthProvider
	query := `
		SELECT * FROM oauth_providers 
		WHERE provider = $1 AND provider_user_id = $2 AND is_active = TRUE`

	err := r.db.GetContext(ctx, &oauthProvider, query, provider, providerUserID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get OAuth provider: %w", err)
	}

	return &oauthProvider, nil
}

// GetByID retrieves OAuth provider by ID
func (r *OAuthRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.OAuthProvider, error) {
	var provider models.OAuthProvider
	query := `SELECT * FROM oauth_providers WHERE id = $1`

	err := r.db.GetContext(ctx, &provider, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get OAuth provider by ID: %w", err)
	}

	return &provider, nil
}

// GetByUserID retrieves all OAuth providers for a user
func (r *OAuthRepository) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*models.OAuthProvider, error) {
	var providers []*models.OAuthProvider
	query := `
		SELECT * FROM oauth_providers 
		WHERE user_id = $1 AND is_active = TRUE
		ORDER BY created_at DESC`

	err := r.db.SelectContext(ctx, &providers, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get OAuth providers: %w", err)
	}

	return providers, nil
}

// Update updates an OAuth provider record
func (r *OAuthRepository) Update(ctx context.Context, provider *models.OAuthProvider) error {
	provider.UpdatedAt = time.Now()

	query := `
		UPDATE oauth_providers SET
			provider_email = :provider_email,
			provider_username = :provider_username,
			provider_data = :provider_data,
			access_token = :access_token,
			refresh_token = :refresh_token,
			token_type = :token_type,
			scope = :scope,
			expires_at = :expires_at,
			is_active = :is_active,
			updated_at = :updated_at
		WHERE id = :id`

	result, err := r.db.NamedExecContext(ctx, query, provider)
	if err != nil {
		return fmt.Errorf("failed to update OAuth provider: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("OAuth provider not found")
	}

	return nil
}

// UpdateTokens updates OAuth tokens for a provider
func (r *OAuthRepository) UpdateTokens(ctx context.Context, id uuid.UUID, accessToken, refreshToken *string, expiresAt *time.Time) error {
	query := `
		UPDATE oauth_providers SET
			access_token = $2,
			refresh_token = $3,
			expires_at = $4,
			updated_at = NOW()
		WHERE id = $1 AND is_active = TRUE`

	result, err := r.db.ExecContext(ctx, query, id, accessToken, refreshToken, expiresAt)
	if err != nil {
		return fmt.Errorf("failed to update OAuth tokens: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("OAuth provider not found or inactive")
	}

	return nil
}

// Deactivate deactivates an OAuth provider
func (r *OAuthRepository) Deactivate(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE oauth_providers SET
			is_active = FALSE,
			updated_at = NOW()
		WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to deactivate OAuth provider: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("OAuth provider not found")
	}

	return nil
}

// DeactivateByUserAndProvider deactivates OAuth provider by user and provider name
func (r *OAuthRepository) DeactivateByUserAndProvider(ctx context.Context, userID uuid.UUID, provider string) error {
	query := `
		UPDATE oauth_providers SET
			is_active = FALSE,
			updated_at = NOW()
		WHERE user_id = $1 AND provider = $2`

	result, err := r.db.ExecContext(ctx, query, userID, provider)
	if err != nil {
		return fmt.Errorf("failed to deactivate OAuth provider: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("OAuth provider not found")
	}

	return nil
}

// Delete hard deletes an OAuth provider
func (r *OAuthRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM oauth_providers WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete OAuth provider: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("OAuth provider not found")
	}

	return nil
}

// ExistsByProviderAndProviderUserID checks if OAuth provider exists
func (r *OAuthRepository) ExistsByProviderAndProviderUserID(ctx context.Context, provider, providerUserID string) (bool, error) {
	var exists bool
	query := `
		SELECT EXISTS(
			SELECT 1 FROM oauth_providers 
			WHERE provider = $1 AND provider_user_id = $2 AND is_active = TRUE
		)`

	err := r.db.GetContext(ctx, &exists, query, provider, providerUserID)
	if err != nil {
		return false, fmt.Errorf("failed to check OAuth provider existence: %w", err)
	}

	return exists, nil
}

// GetActiveProvidersByUser returns list of active OAuth provider names for a user
func (r *OAuthRepository) GetActiveProvidersByUser(ctx context.Context, userID uuid.UUID) ([]string, error) {
	var providers []string
	query := `
		SELECT DISTINCT provider FROM oauth_providers 
		WHERE user_id = $1 AND is_active = TRUE
		ORDER BY provider`

	err := r.db.SelectContext(ctx, &providers, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get active providers: %w", err)
	}

	return providers, nil
}

// GetUserByProviderEmail finds a user by OAuth provider email
func (r *OAuthRepository) GetUserByProviderEmail(ctx context.Context, provider, email string) (*models.OAuthProvider, error) {
	var oauthProvider models.OAuthProvider
	query := `
		SELECT * FROM oauth_providers 
		WHERE provider = $1 AND provider_email = $2 AND is_active = TRUE
		ORDER BY created_at DESC
		LIMIT 1`

	err := r.db.GetContext(ctx, &oauthProvider, query, provider, email)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get OAuth provider by email: %w", err)
	}

	return &oauthProvider, nil
}

// CleanupInactiveProviders removes inactive OAuth providers older than specified duration
func (r *OAuthRepository) CleanupInactiveProviders(ctx context.Context, olderThan time.Duration) (int, error) {
	query := `
		DELETE FROM oauth_providers 
		WHERE is_active = FALSE AND updated_at < $1`

	cutoffTime := time.Now().Add(-olderThan)
	result, err := r.db.ExecContext(ctx, query, cutoffTime)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup inactive providers: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get affected rows: %w", err)
	}

	return int(rowsAffected), nil
}

// GetExpiredTokens retrieves OAuth providers with expired tokens
func (r *OAuthRepository) GetExpiredTokens(ctx context.Context) ([]*models.OAuthProvider, error) {
	var providers []*models.OAuthProvider
	query := `
		SELECT * FROM oauth_providers 
		WHERE is_active = TRUE 
		AND expires_at IS NOT NULL 
		AND expires_at < NOW()`

	err := r.db.SelectContext(ctx, &providers, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get expired tokens: %w", err)
	}

	return providers, nil
}

// CountByProvider returns count of active OAuth providers by provider name
func (r *OAuthRepository) CountByProvider(ctx context.Context, provider string) (int, error) {
	var count int
	query := `
		SELECT COUNT(*) FROM oauth_providers 
		WHERE provider = $1 AND is_active = TRUE`

	err := r.db.GetContext(ctx, &count, query, provider)
	if err != nil {
		return 0, fmt.Errorf("failed to count providers: %w", err)
	}

	return count, nil
}

// GetProviderStats returns statistics about OAuth providers
func (r *OAuthRepository) GetProviderStats(ctx context.Context) (map[string]int, error) {
	stats := make(map[string]int)

	query := `
		SELECT provider, COUNT(*) as count 
		FROM oauth_providers 
		WHERE is_active = TRUE 
		GROUP BY provider`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get provider stats: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var provider string
		var count int
		if err := rows.Scan(&provider, &count); err != nil {
			return nil, fmt.Errorf("failed to scan provider stats: %w", err)
		}
		stats[provider] = count
	}

	return stats, nil
}
