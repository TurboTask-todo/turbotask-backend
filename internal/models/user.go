package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// UserStatus represents the status of a user account
type UserStatus string

const (
	UserStatusActive    UserStatus = "active"
	UserStatusInactive  UserStatus = "inactive"
	UserStatusSuspended UserStatus = "suspended"
	UserStatusPending   UserStatus = "pending"
	UserStatusDeleted   UserStatus = "deleted"
)

// User represents the users table
type User struct {
	ID                  uuid.UUID  `json:"id" db:"id"`
	Email               string     `json:"email" db:"email"`
	Username            *string    `json:"username,omitempty" db:"username"`
	EmailVerified       bool       `json:"email_verified" db:"email_verified"`
	FirstName           *string    `json:"first_name,omitempty" db:"first_name"`
	LastName            *string    `json:"last_name,omitempty" db:"last_name"`
	AvatarURL           *string    `json:"avatar_url,omitempty" db:"avatar_url"`
	Timezone            string     `json:"timezone" db:"timezone"`
	Locale              string     `json:"locale" db:"locale"`
	Preferences         JSONB      `json:"preferences" db:"preferences"`
	Status              UserStatus `json:"status" db:"status"`
	LastLoginAt         *time.Time `json:"last_login_at,omitempty" db:"last_login_at"`
	FailedLoginAttempts int        `json:"failed_login_attempts" db:"failed_login_attempts"`
	AccountLockedUntil  *time.Time `json:"account_locked_until,omitempty" db:"account_locked_until"`
	PasswordChangedAt   *time.Time `json:"password_changed_at,omitempty" db:"password_changed_at"`
	CreatedAt           time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at" db:"updated_at"`
	DeletedAt           *time.Time `json:"deleted_at,omitempty" db:"deleted_at"`
}

// UserPassword represents the user_passwords table
type UserPassword struct {
	ID           uuid.UUID  `json:"id" db:"id"`
	UserID       uuid.UUID  `json:"user_id" db:"user_id"`
	PasswordHash string     `json:"-" db:"password_hash"` // Hidden from JSON
	Salt         *string    `json:"-" db:"salt"`          // Hidden from JSON
	Algorithm    string     `json:"algorithm" db:"algorithm"`
	CostParams   JSONB      `json:"cost_params" db:"cost_params"`
	Version      int        `json:"version" db:"version"`
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at" db:"updated_at"`
	ExpiresAt    *time.Time `json:"expires_at,omitempty" db:"expires_at"`
	IsTemporary  bool       `json:"is_temporary" db:"is_temporary"`
	IsActive     bool       `json:"is_active" db:"is_active"`
}

// OAuthProvider represents the oauth_providers table
type OAuthProvider struct {
	ID               uuid.UUID      `json:"id" db:"id"`
	UserID           uuid.UUID      `json:"user_id" db:"user_id"`
	Provider         string         `json:"provider" db:"provider"`
	ProviderUserID   string         `json:"provider_user_id" db:"provider_user_id"`
	ProviderEmail    *string        `json:"provider_email,omitempty" db:"provider_email"`
	ProviderUsername *string        `json:"provider_username,omitempty" db:"provider_username"`
	ProviderData     JSONB          `json:"provider_data" db:"provider_data"`
	AccessToken      *string        `json:"-" db:"access_token"`  // Hidden from JSON
	RefreshToken     *string        `json:"-" db:"refresh_token"` // Hidden from JSON
	TokenType        string         `json:"token_type" db:"token_type"`
	Scope            pq.StringArray `json:"scope" db:"scope"`
	ExpiresAt        *time.Time     `json:"expires_at,omitempty" db:"expires_at"`
	IsActive         bool           `json:"is_active" db:"is_active"`
	CreatedAt        time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at" db:"updated_at"`
}

// RefreshToken represents the refresh_tokens table
type RefreshToken struct {
	ID                uuid.UUID  `json:"id" db:"id"`
	UserID            uuid.UUID  `json:"user_id" db:"user_id"`
	TokenHash         string     `json:"-" db:"token_hash"` // Hidden from JSON
	TokenFamily       uuid.UUID  `json:"token_family" db:"token_family"`
	DeviceInfo        JSONB      `json:"device_info" db:"device_info"`
	DeviceFingerprint *string    `json:"device_fingerprint,omitempty" db:"device_fingerprint"`
	IPAddress         *string    `json:"ip_address,omitempty" db:"ip_address"`
	UserAgent         *string    `json:"user_agent,omitempty" db:"user_agent"`
	LocationData      JSONB      `json:"location_data" db:"location_data"`
	ExpiresAt         time.Time  `json:"expires_at" db:"expires_at"`
	LastUsedAt        time.Time  `json:"last_used_at" db:"last_used_at"`
	UsageCount        int        `json:"usage_count" db:"usage_count"`
	IsRevoked         bool       `json:"is_revoked" db:"is_revoked"`
	RevokedAt         *time.Time `json:"revoked_at,omitempty" db:"revoked_at"`
	RevokedReason     *string    `json:"revoked_reason,omitempty" db:"revoked_reason"`
	CreatedAt         time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at" db:"updated_at"`
}

// VerificationToken represents the verification_tokens table
type VerificationToken struct {
	ID            uuid.UUID  `json:"id" db:"id"`
	UserID        uuid.UUID  `json:"user_id" db:"user_id"`
	TokenHash     string     `json:"-" db:"token_hash"` // Hidden from JSON
	TokenType     string     `json:"token_type" db:"token_type"`
	TargetValue   *string    `json:"target_value,omitempty" db:"target_value"`
	AttemptsCount int        `json:"attempts_count" db:"attempts_count"`
	MaxAttempts   int        `json:"max_attempts" db:"max_attempts"`
	ExpiresAt     time.Time  `json:"expires_at" db:"expires_at"`
	UsedAt        *time.Time `json:"used_at,omitempty" db:"used_at"`
	IPAddress     *string    `json:"ip_address,omitempty" db:"ip_address"`
	UserAgent     *string    `json:"user_agent,omitempty" db:"user_agent"`
	IsActive      bool       `json:"is_active" db:"is_active"`
	CreatedAt     time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at" db:"updated_at"`
}

// UserSecurityEvent represents the user_security_events table
type UserSecurityEvent struct {
	ID            uuid.UUID  `json:"id" db:"id"`
	UserID        *uuid.UUID `json:"user_id,omitempty" db:"user_id"`
	SessionID     *uuid.UUID `json:"session_id,omitempty" db:"session_id"`
	EventType     string     `json:"event_type" db:"event_type"`
	EventCategory string     `json:"event_category" db:"event_category"`
	IPAddress     *string    `json:"ip_address,omitempty" db:"ip_address"`
	UserAgent     *string    `json:"user_agent,omitempty" db:"user_agent"`
	DeviceInfo    JSONB      `json:"device_info" db:"device_info"`
	LocationData  JSONB      `json:"location_data" db:"location_data"`
	Success       bool       `json:"success" db:"success"`
	FailureReason *string    `json:"failure_reason,omitempty" db:"failure_reason"`
	RiskScore     int        `json:"risk_score" db:"risk_score"`
	Metadata      JSONB      `json:"metadata" db:"metadata"`
	CreatedAt     time.Time  `json:"created_at" db:"created_at"`
}

// User2FA represents the user_2fa table
type User2FA struct {
	ID                   uuid.UUID      `json:"id" db:"id"`
	UserID               uuid.UUID      `json:"user_id" db:"user_id"`
	Method               string         `json:"method" db:"method"`
	Label                *string        `json:"label,omitempty" db:"label"`
	SecretEncrypted      *string        `json:"-" db:"secret_encrypted"` // Hidden from JSON
	PhoneNumber          *string        `json:"phone_number,omitempty" db:"phone_number"`
	EmailAddress         *string        `json:"email_address,omitempty" db:"email_address"`
	BackupCodesEncrypted pq.StringArray `json:"-" db:"backup_codes_encrypted"` // Hidden from JSON
	DeviceData           JSONB          `json:"device_data" db:"device_data"`
	UsageCount           int            `json:"usage_count" db:"usage_count"`
	LastUsedAt           *time.Time     `json:"last_used_at,omitempty" db:"last_used_at"`
	IsEnabled            bool           `json:"is_enabled" db:"is_enabled"`
	IsPrimary            bool           `json:"is_primary" db:"is_primary"`
	VerifiedAt           *time.Time     `json:"verified_at,omitempty" db:"verified_at"`
	CreatedAt            time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt            time.Time      `json:"updated_at" db:"updated_at"`
}

// UserSession represents the user_sessions table
type UserSession struct {
	ID                uuid.UUID  `json:"id" db:"id"`
	UserID            uuid.UUID  `json:"user_id" db:"user_id"`
	SessionTokenHash  string     `json:"-" db:"session_token_hash"` // Hidden from JSON
	SessionName       *string    `json:"session_name,omitempty" db:"session_name"`
	DeviceInfo        JSONB      `json:"device_info" db:"device_info"`
	DeviceFingerprint *string    `json:"device_fingerprint,omitempty" db:"device_fingerprint"`
	IPAddress         *string    `json:"ip_address,omitempty" db:"ip_address"`
	UserAgent         *string    `json:"user_agent,omitempty" db:"user_agent"`
	LocationData      JSONB      `json:"location_data" db:"location_data"`
	MFAVerified       bool       `json:"mfa_verified" db:"mfa_verified"`
	RiskScore         int        `json:"risk_score" db:"risk_score"`
	ExpiresAt         time.Time  `json:"expires_at" db:"expires_at"`
	LastActivityAt    time.Time  `json:"last_activity_at" db:"last_activity_at"`
	ActivityCount     int        `json:"activity_count" db:"activity_count"`
	IsActive          bool       `json:"is_active" db:"is_active"`
	TerminatedAt      *time.Time `json:"terminated_at,omitempty" db:"terminated_at"`
	TerminationReason *string    `json:"termination_reason,omitempty" db:"termination_reason"`
	CreatedAt         time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at" db:"updated_at"`
}

// IsAccountLocked checks if the user account is currently locked
func (u *User) IsAccountLocked() bool {
	if u.AccountLockedUntil == nil {
		return false
	}
	return time.Now().Before(*u.AccountLockedUntil)
}

// IsDeleted checks if the user account has been soft deleted
func (u *User) IsDeleted() bool {
	return u.DeletedAt != nil
}

// CanLogin checks if the user can login (not deleted, not locked, active status)
func (u *User) CanLogin() bool {
	return !u.IsDeleted() && !u.IsAccountLocked() && u.Status == UserStatusActive
}

// GetFullName returns the user's full name
func (u *User) GetFullName() string {
	var name string
	if u.FirstName != nil {
		name = *u.FirstName
	}
	if u.LastName != nil {
		if name != "" {
			name += " "
		}
		name += *u.LastName
	}
	return name
}

// GetDisplayName returns a display name for the user (full name, username, or email)
func (u *User) GetDisplayName() string {
	fullName := u.GetFullName()
	if fullName != "" {
		return fullName
	}
	if u.Username != nil && *u.Username != "" {
		return *u.Username
	}
	return u.Email
}
