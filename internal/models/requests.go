package models

import "github.com/google/uuid"

// AuthRequest represents common authentication request data
type AuthRequest struct {
	Email    string `json:"email" binding:"required,email" example:"user@example.com"`
	Password string `json:"password" binding:"required,min=8" example:"securepassword123"`
}

// RegisterRequest represents user registration request
type RegisterRequest struct {
	Email           string  `json:"email" binding:"required,email" example:"user@example.com"`
	Password        string  `json:"password" binding:"required,min=8" example:"securepassword123"`
	ConfirmPassword string  `json:"confirm_password" binding:"required" example:"securepassword123"`
	Username        *string `json:"username,omitempty" binding:"omitempty,min=3,max=50" example:"john_doe"`
	FirstName       *string `json:"first_name,omitempty" binding:"omitempty,max=100" example:"John"`
	LastName        *string `json:"last_name,omitempty" binding:"omitempty,max=100" example:"Doe"`
	Timezone        *string `json:"timezone,omitempty" example:"America/New_York"`
	Locale          *string `json:"locale,omitempty" example:"en-US"`
}

// LoginRequest represents user login request
type LoginRequest struct {
	Email      string `json:"email" binding:"required,email" example:"user@example.com"`
	Password   string `json:"password" binding:"required" example:"securepassword123"`
	RememberMe bool   `json:"remember_me,omitempty" example:"true"`
	DeviceInfo JSONB  `json:"device_info,omitempty"`
}

// RefreshTokenRequest represents refresh token request
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
}

// PaginationRequest represents pagination parameters
type PaginationRequest struct {
	Page     int    `form:"page" json:"page" example:"1" validate:"min=1"`
	PageSize int    `form:"page_size" json:"page_size" example:"20" validate:"min=1,max=100"`
	Sort     string `form:"sort" json:"sort,omitempty" example:"created_at"`
	Order    string `form:"order" json:"order,omitempty" example:"desc" validate:"omitempty,oneof=asc desc"`
}

// ProjectsQueryRequest represents project listing parameters
type ProjectsQueryRequest struct {
	PaginationRequest
	IncludeArchived bool   `form:"include_archived" json:"include_archived"`
	IncludeCounts   bool   `form:"include_counts" json:"include_counts"`
	Category        string `form:"category" json:"category,omitempty"`
	IsFavorite      *bool  `form:"is_favorite" json:"is_favorite,omitempty"`
	Search          string `form:"search" json:"search,omitempty"`
}

// ForgotPasswordRequest represents forgot password request
type ForgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email" example:"user@example.com"`
}

// ResetPasswordRequest represents reset password request
type ResetPasswordRequest struct {
	Token           string `json:"token" binding:"required" example:"reset-token-here"`
	Password        string `json:"password" binding:"required,min=8" example:"newsecurepassword123"`
	ConfirmPassword string `json:"confirm_password" binding:"required" example:"newsecurepassword123"`
}

// ChangePasswordRequest represents change password request
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required" example:"oldsecurepassword123"`
	NewPassword     string `json:"new_password" binding:"required,min=8" example:"newsecurepassword123"`
	ConfirmPassword string `json:"confirm_password" binding:"required" example:"newsecurepassword123"`
}

// VerifyEmailRequest represents email verification request
type VerifyEmailRequest struct {
	Token string `json:"token" binding:"required" example:"verification-token-here"`
}

// ResendVerificationRequest represents resend verification email request
type ResendVerificationRequest struct {
	Email string `json:"email" binding:"required,email" example:"user@example.com"`
}

// UpdateProfileRequest represents update profile request
type UpdateProfileRequest struct {
	Username    *string `json:"username,omitempty" binding:"omitempty,min=3,max=50" example:"john_doe"`
	FirstName   *string `json:"first_name,omitempty" binding:"omitempty,max=100" example:"John"`
	LastName    *string `json:"last_name,omitempty" binding:"omitempty,max=100" example:"Doe"`
	AvatarURL   *string `json:"avatar_url,omitempty" example:"https://example.com/avatar.jpg"`
	Timezone    *string `json:"timezone,omitempty" example:"America/New_York"`
	Locale      *string `json:"locale,omitempty" example:"en-US"`
	Preferences JSONB   `json:"preferences,omitempty"`
}

// OAuthCallbackRequest represents OAuth callback request
type OAuthCallbackRequest struct {
	Code  string `json:"code" form:"code" binding:"required" example:"oauth-authorization-code"`
	State string `json:"state" form:"state" binding:"required" example:"oauth-state-parameter"`
}

// DeviceInfo represents device information for security tracking
type DeviceInfo struct {
	Browser   string `json:"browser,omitempty" example:"Chrome"`
	OS        string `json:"os,omitempty" example:"macOS"`
	Device    string `json:"device,omitempty" example:"Desktop"`
	IP        string `json:"ip,omitempty" example:"192.168.1.1"`
	UserAgent string `json:"user_agent,omitempty" example:"Mozilla/5.0..."`
	Timezone  string `json:"timezone,omitempty" example:"America/New_York"`
	Language  string `json:"language,omitempty" example:"en-US"`
	Screen    string `json:"screen,omitempty" example:"1920x1080"`
	Platform  string `json:"platform,omitempty" example:"MacIntel"`
}

// SessionRequest represents session management request
type SessionRequest struct {
	SessionID string `json:"session_id" binding:"required" example:"session-uuid"`
}

// APIKeyCreateRequest represents API key creation request
type APIKeyCreateRequest struct {
	ServiceName      string `json:"service_name" binding:"required" example:"openai_gpt"`
	KeyName          string `json:"key_name" binding:"required" example:"My OpenAI Key"`
	APIKey           string `json:"api_key" binding:"required" example:"sk-..."`
	RateLimitPerHour *int   `json:"rate_limit_per_hour,omitempty" example:"1000"`
	RateLimitPerDay  *int   `json:"rate_limit_per_day,omitempty" example:"10000"`
}

// APIKeyUpdateRequest represents API key update request
type APIKeyUpdateRequest struct {
	KeyName          *string `json:"key_name,omitempty" example:"Updated Key Name"`
	APIKey           *string `json:"api_key,omitempty" example:"sk-new-key..."`
	RateLimitPerHour *int    `json:"rate_limit_per_hour,omitempty" example:"2000"`
	RateLimitPerDay  *int    `json:"rate_limit_per_day,omitempty" example:"20000"`
	IsActive         *bool   `json:"is_active,omitempty" example:"true"`
}

// UserActivationRequest represents user activation/deactivation request (admin only)
type UserActivationRequest struct {
	UserID uuid.UUID  `json:"user_id" binding:"required"`
	Status UserStatus `json:"status" binding:"required,oneof=active inactive suspended"`
	Reason string     `json:"reason,omitempty" example:"Account verified by admin"`
}

// Kanban Board Request Models

// MoveTodoRequest represents a request to move a todo to a different column/status
type MoveTodoRequest struct {
	TodoID    uuid.UUID  `json:"todo_id" binding:"required"`
	NewStatus TaskStatus `json:"new_status" binding:"required"`
	Position  *int       `json:"position,omitempty"` // Optional position in the new column
}

// ReorderTodosRequest represents a request to reorder todos within the same column
type ReorderTodosRequest struct {
	TodoIDs []uuid.UUID `json:"todo_ids" binding:"required,min=1"` // Array of todo IDs in new order
	Status  TaskStatus  `json:"status" binding:"required"`         // Column/status to reorder within
}

// BulkMoveTodosRequest represents a request to move multiple todos at once
type BulkMoveTodosRequest struct {
	Moves []MoveTodoRequest `json:"moves" binding:"required,min=1"`
}

// KanbanColumnRequest represents the structure for organizing kanban columns
type KanbanColumnRequest struct {
	ProjectID uuid.UUID `json:"project_id" binding:"required"`
	Columns   []string  `json:"columns,omitempty"` // Optional custom column order
}

// TaskQuickCreateRequest represents a simplified request for quick task creation in kanban
type TaskQuickCreateRequest struct {
	ProjectID       uuid.UUID  `json:"project_id" binding:"required"`
	TaskName        string     `json:"task_name" binding:"required,min=1,max=255"`
	TaskDescription *string    `json:"task_description,omitempty"`
	Status          TaskStatus `json:"status" binding:"required"`
	Priority        *string    `json:"priority,omitempty"`
	Position        *int       `json:"position,omitempty"` // Position in the column
}
