package models

import (
	"time"

	"github.com/google/uuid"
)

// APIServiceType represents the api_service_types table
type APIServiceType struct {
	ID                 uuid.UUID `json:"id" db:"id"`
	ServiceName        string    `json:"service_name" db:"service_name"`
	ServiceDisplayName string    `json:"service_display_name" db:"service_display_name"`
	Provider           string    `json:"provider" db:"provider"`
	APIEndpoint        *string   `json:"api_endpoint,omitempty" db:"api_endpoint"`
	DocumentationURL   *string   `json:"documentation_url,omitempty" db:"documentation_url"`
	PricingModel       string    `json:"pricing_model" db:"pricing_model"`
	IsActive           bool      `json:"is_active" db:"is_active"`
	CreatedAt          time.Time `json:"created_at" db:"created_at"`
	UpdatedAt          time.Time `json:"updated_at" db:"updated_at"`
}

// UserAPIKey represents the user_api_keys table
type UserAPIKey struct {
	ID                uuid.UUID  `json:"id" db:"id"`
	UserID            uuid.UUID  `json:"user_id" db:"user_id"`
	ServiceTypeID     uuid.UUID  `json:"service_type_id" db:"service_type_id"`
	KeyName           *string    `json:"key_name,omitempty" db:"key_name"`
	KeyHash           string     `json:"-" db:"key_hash"` // Hidden from JSON
	KeyPrefix         string     `json:"key_prefix" db:"key_prefix"`
	Permissions       JSONB      `json:"permissions" db:"permissions"`
	UsageStats        JSONB      `json:"usage_stats" db:"usage_stats"`
	RateLimitPerHour  int        `json:"rate_limit_per_hour" db:"rate_limit_per_hour"`
	RateLimitPerDay   int        `json:"rate_limit_per_day" db:"rate_limit_per_day"`
	RateLimitPerMonth int        `json:"rate_limit_per_month" db:"rate_limit_per_month"`
	CurrentUsage      JSONB      `json:"current_usage" db:"current_usage"`
	ExpiresAt         *time.Time `json:"expires_at,omitempty" db:"expires_at"`
	LastUsedAt        *time.Time `json:"last_used_at,omitempty" db:"last_used_at"`
	UsageCount        int        `json:"usage_count" db:"usage_count"`
	IsActive          bool       `json:"is_active" db:"is_active"`
	CreatedAt         time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at" db:"updated_at"`
}

// UserAuthSummary represents the v_user_auth_summary view
type UserAuthSummary struct {
	ID             uuid.UUID  `json:"id" db:"id"`
	Email          string     `json:"email" db:"email"`
	Username       *string    `json:"username,omitempty" db:"username"`
	Status         UserStatus `json:"status" db:"status"`
	EmailVerified  bool       `json:"email_verified" db:"email_verified"`
	LastLoginAt    *time.Time `json:"last_login_at,omitempty" db:"last_login_at"`
	CreatedAt      time.Time  `json:"created_at" db:"created_at"`
	HasPassword    bool       `json:"has_password" db:"has_password"`
	OAuthProviders []string   `json:"oauth_providers" db:"oauth_providers"`
	MFAMethods     []string   `json:"mfa_methods" db:"mfa_methods"`
	ActiveAPIKeys  int        `json:"active_api_keys" db:"active_api_keys"`
}

// APIKeyUsage represents the v_api_key_usage view
type APIKeyUsage struct {
	ID                 uuid.UUID  `json:"id" db:"id"`
	UserEmail          string     `json:"user_email" db:"user_email"`
	ServiceDisplayName string     `json:"service_display_name" db:"service_display_name"`
	Provider           string     `json:"provider" db:"provider"`
	KeyName            *string    `json:"key_name,omitempty" db:"key_name"`
	UsageCount         int        `json:"usage_count" db:"usage_count"`
	LastUsedAt         *time.Time `json:"last_used_at,omitempty" db:"last_used_at"`
	RateLimitPerHour   int        `json:"rate_limit_per_hour" db:"rate_limit_per_hour"`
	CurrentUsage       JSONB      `json:"current_usage" db:"current_usage"`
	IsActive           bool       `json:"is_active" db:"is_active"`
	CreatedAt          time.Time  `json:"created_at" db:"created_at"`
}
