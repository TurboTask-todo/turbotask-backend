package models

import (
	"time"

	"github.com/google/uuid"
)

// APIResponse represents a standard API response wrapper
type APIResponse struct {
	Success   bool        `json:"success" example:"true"`
	Message   string      `json:"message" example:"Operation completed successfully"`
	Data      interface{} `json:"data,omitempty"`
	Error     *APIError   `json:"error,omitempty"`
	Timestamp time.Time   `json:"timestamp" example:"2023-12-01T10:00:00Z"`
	RequestID string      `json:"request_id,omitempty" example:"req_123456789"`
}

// APIError represents an API error
type APIError struct {
	Code    string            `json:"code" example:"INVALID_CREDENTIALS"`
	Message string            `json:"message" example:"Invalid email or password"`
	Details map[string]string `json:"details,omitempty"`
	Field   string            `json:"field,omitempty" example:"email"`
}

// AuthResponse represents authentication response with tokens
type AuthResponse struct {
	User         *UserResponse `json:"user"`
	AccessToken  string        `json:"access_token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
	RefreshToken string        `json:"refresh_token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
	TokenType    string        `json:"token_type" example:"Bearer"`
	ExpiresIn    int           `json:"expires_in" example:"900"` // seconds
	ExpiresAt    time.Time     `json:"expires_at" example:"2023-12-01T10:15:00Z"`
}

// UserResponse represents user data in API responses
type UserResponse struct {
	ID            uuid.UUID  `json:"id" example:"123e4567-e89b-12d3-a456-426614174000"`
	Email         string     `json:"email" example:"user@example.com"`
	Username      *string    `json:"username,omitempty" example:"john_doe"`
	EmailVerified bool       `json:"email_verified" example:"true"`
	FirstName     *string    `json:"first_name,omitempty" example:"John"`
	LastName      *string    `json:"last_name,omitempty" example:"Doe"`
	FullName      string     `json:"full_name" example:"John Doe"`
	AvatarURL     *string    `json:"avatar_url,omitempty" example:"https://example.com/avatar.jpg"`
	Timezone      string     `json:"timezone" example:"America/New_York"`
	Locale        string     `json:"locale" example:"en-US"`
	Preferences   JSONB      `json:"preferences,omitempty"`
	Status        UserStatus `json:"status" example:"active"`
	LastLoginAt   *time.Time `json:"last_login_at,omitempty" example:"2023-12-01T09:00:00Z"`
	CreatedAt     time.Time  `json:"created_at" example:"2023-11-01T10:00:00Z"`
	UpdatedAt     time.Time  `json:"updated_at" example:"2023-12-01T10:00:00Z"`
}

// TokenResponse represents token-only response
type TokenResponse struct {
	AccessToken  string    `json:"access_token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
	RefreshToken string    `json:"refresh_token,omitempty" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
	TokenType    string    `json:"token_type" example:"Bearer"`
	ExpiresIn    int       `json:"expires_in" example:"900"`
	ExpiresAt    time.Time `json:"expires_at" example:"2023-12-01T10:15:00Z"`
}

// MessageResponse represents a simple message response
type MessageResponse struct {
	Message string `json:"message" example:"Email verification sent successfully"`
}

// UserProfileResponse represents user profile response
type UserProfileResponse struct {
	User         *UserResponse    `json:"user"`
	AuthMethods  *AuthMethodsInfo `json:"auth_methods"`
	SecurityInfo *SecurityInfo    `json:"security_info"`
	SessionInfo  *SessionInfo     `json:"session_info,omitempty"`
}

// AuthMethodsInfo represents available authentication methods for a user
type AuthMethodsInfo struct {
	HasPassword    bool     `json:"has_password" example:"true"`
	OAuthProviders []string `json:"oauth_providers" example:"[\"google\", \"github\"]"`
	MFAMethods     []string `json:"mfa_methods" example:"[\"totp\", \"email\"]"`
	MFAEnabled     bool     `json:"mfa_enabled" example:"true"`
}

// SecurityInfo represents security information for a user
type SecurityInfo struct {
	FailedLoginAttempts int        `json:"failed_login_attempts" example:"0"`
	AccountLocked       bool       `json:"account_locked" example:"false"`
	AccountLockedUntil  *time.Time `json:"account_locked_until,omitempty" example:"2023-12-01T10:20:00Z"`
	PasswordChangedAt   *time.Time `json:"password_changed_at,omitempty" example:"2023-11-15T10:00:00Z"`
	LastSecurityEvent   *time.Time `json:"last_security_event,omitempty" example:"2023-12-01T09:00:00Z"`
}

// SessionInfo represents session information
type SessionInfo struct {
	SessionID      uuid.UUID `json:"session_id" example:"123e4567-e89b-12d3-a456-426614174000"`
	DeviceInfo     JSONB     `json:"device_info,omitempty"`
	IPAddress      *string   `json:"ip_address,omitempty" example:"192.168.1.1"`
	UserAgent      *string   `json:"user_agent,omitempty" example:"Mozilla/5.0..."`
	LocationData   JSONB     `json:"location_data,omitempty"`
	MFAVerified    bool      `json:"mfa_verified" example:"false"`
	ExpiresAt      time.Time `json:"expires_at" example:"2023-12-01T18:00:00Z"`
	LastActivityAt time.Time `json:"last_activity_at" example:"2023-12-01T10:00:00Z"`
}

// SessionListResponse represents a list of user sessions
type SessionListResponse struct {
	Sessions       []SessionInfo `json:"sessions"`
	CurrentSession uuid.UUID     `json:"current_session" example:"123e4567-e89b-12d3-a456-426614174000"`
	TotalCount     int           `json:"total_count" example:"3"`
}

// APIKeyResponse represents API key response
type APIKeyResponse struct {
	ID                 uuid.UUID  `json:"id" example:"123e4567-e89b-12d3-a456-426614174000"`
	ServiceName        string     `json:"service_name" example:"openai_gpt"`
	ServiceDisplayName string     `json:"service_display_name" example:"ChatGPT / GPT Models"`
	Provider           string     `json:"provider" example:"openai"`
	KeyName            *string    `json:"key_name,omitempty" example:"My OpenAI Key"`
	KeyPrefix          string     `json:"key_prefix" example:"sk-..."`
	RateLimitPerHour   int        `json:"rate_limit_per_hour" example:"1000"`
	RateLimitPerDay    int        `json:"rate_limit_per_day" example:"10000"`
	UsageCount         int        `json:"usage_count" example:"150"`
	LastUsedAt         *time.Time `json:"last_used_at,omitempty" example:"2023-12-01T09:30:00Z"`
	IsActive           bool       `json:"is_active" example:"true"`
	CreatedAt          time.Time  `json:"created_at" example:"2023-11-01T10:00:00Z"`
}

// APIKeyCreateResponse represents API key creation response
type APIKeyCreateResponse struct {
	APIKey   *APIKeyResponse `json:"api_key"`
	Message  string          `json:"message" example:"API key created successfully"`
	PlainKey string          `json:"plain_key,omitempty" example:"sk-actual-key-value"` // Only returned on creation
	Warning  string          `json:"warning,omitempty" example:"Save this key securely - it won't be shown again"`
}

// ServiceTypesResponse represents available service types
type ServiceTypesResponse struct {
	Services []APIServiceType `json:"services"`
	Count    int              `json:"count" example:"8"`
}

// SecurityEventResponse represents security event response
type SecurityEventResponse struct {
	ID            uuid.UUID `json:"id" example:"123e4567-e89b-12d3-a456-426614174000"`
	EventType     string    `json:"event_type" example:"login_success"`
	EventCategory string    `json:"event_category" example:"authentication"`
	IPAddress     *string   `json:"ip_address,omitempty" example:"192.168.1.1"`
	UserAgent     *string   `json:"user_agent,omitempty" example:"Mozilla/5.0..."`
	Success       bool      `json:"success" example:"true"`
	FailureReason *string   `json:"failure_reason,omitempty" example:"Invalid password"`
	RiskScore     int       `json:"risk_score" example:"10"`
	CreatedAt     time.Time `json:"created_at" example:"2023-12-01T10:00:00Z"`
}

// PaginationMeta represents pagination metadata
type PaginationMeta struct {
	Page        int  `json:"page" example:"1"`
	PageSize    int  `json:"page_size" example:"20"`
	TotalCount  int  `json:"total_count" example:"100"`
	TotalPages  int  `json:"total_pages" example:"5"`
	HasNextPage bool `json:"has_next_page" example:"true"`
	HasPrevPage bool `json:"has_prev_page" example:"false"`
	NextPage    *int `json:"next_page,omitempty" example:"2"`
	PrevPage    *int `json:"prev_page,omitempty" example:"1"`
}

// PaginatedResponse represents a paginated API response
type PaginatedResponse struct {
	Success    bool           `json:"success" example:"true"`
	Message    string         `json:"message" example:"Data retrieved successfully"`
	Data       interface{}    `json:"data"`
	Pagination PaginationMeta `json:"pagination"`
	Error      *APIError      `json:"error,omitempty"`
	Timestamp  time.Time      `json:"timestamp" example:"2023-12-01T10:00:00Z"`
	RequestID  string         `json:"request_id,omitempty" example:"req_123456789"`
}

// ProjectsResponse represents a paginated list of projects
type ProjectsResponse struct {
	Projects   []*Project     `json:"projects"`
	Pagination PaginationMeta `json:"pagination"`
}

// SecurityEventsResponse represents a list of security events
type SecurityEventsResponse struct {
	Events     []SecurityEventResponse `json:"events"`
	TotalCount int                     `json:"total_count" example:"25"`
	Page       int                     `json:"page" example:"1"`
	PageSize   int                     `json:"page_size" example:"20"`
}

// OAuthURLResponse represents OAuth authorization URL response
type OAuthURLResponse struct {
	AuthURL string `json:"auth_url" example:"https://accounts.google.com/oauth/authorize?..."`
	State   string `json:"state" example:"random-state-string"`
}

// HealthResponse represents health check response
type HealthResponse struct {
	Status    string                 `json:"status" example:"healthy"`
	Version   string                 `json:"version" example:"1.0.0"`
	Timestamp time.Time              `json:"timestamp" example:"2023-12-01T10:00:00Z"`
	Checks    map[string]interface{} `json:"checks,omitempty"`
}

// ValidationErrorResponse represents validation error details
type ValidationErrorResponse struct {
	Field   string `json:"field" example:"email"`
	Message string `json:"message" example:"Email format is invalid"`
	Value   string `json:"value,omitempty" example:"invalid-email"`
}

// ToUserResponse converts User model to UserResponse
func (u *User) ToUserResponse() *UserResponse {
	return &UserResponse{
		ID:            u.ID,
		Email:         u.Email,
		Username:      u.Username,
		EmailVerified: u.EmailVerified,
		FirstName:     u.FirstName,
		LastName:      u.LastName,
		FullName:      u.GetFullName(),
		AvatarURL:     u.AvatarURL,
		Timezone:      u.Timezone,
		Locale:        u.Locale,
		Preferences:   u.Preferences,
		Status:        u.Status,
		LastLoginAt:   u.LastLoginAt,
		CreatedAt:     u.CreatedAt,
		UpdatedAt:     u.UpdatedAt,
	}
}

// ToAPIKeyResponse converts UserAPIKey model to APIKeyResponse
func (ak *UserAPIKey) ToAPIKeyResponse(service *APIServiceType) *APIKeyResponse {
	return &APIKeyResponse{
		ID:                 ak.ID,
		ServiceName:        service.ServiceName,
		ServiceDisplayName: service.ServiceDisplayName,
		Provider:           service.Provider,
		KeyName:            ak.KeyName,
		KeyPrefix:          ak.KeyPrefix,
		RateLimitPerHour:   ak.RateLimitPerHour,
		RateLimitPerDay:    ak.RateLimitPerDay,
		UsageCount:         ak.UsageCount,
		LastUsedAt:         ak.LastUsedAt,
		IsActive:           ak.IsActive,
		CreatedAt:          ak.CreatedAt,
	}
}

// NewAPIResponse creates a new successful API response
func NewAPIResponse(message string, data interface{}) *APIResponse {
	return &APIResponse{
		Success:   true,
		Message:   message,
		Data:      data,
		Timestamp: time.Now(),
	}
}

// NewErrorResponse creates a new error API response
func NewErrorResponse(code, message string, details map[string]string) *APIResponse {
	return &APIResponse{
		Success: false,
		Error: &APIError{
			Code:    code,
			Message: message,
			Details: details,
		},
		Timestamp: time.Now(),
	}
}

// NewValidationErrorResponse creates a new validation error response
func NewValidationErrorResponse(field, message string) *APIResponse {
	return &APIResponse{
		Success: false,
		Error: &APIError{
			Code:    "VALIDATION_ERROR",
			Message: message,
			Field:   field,
		},
		Timestamp: time.Now(),
	}
}

// Kanban Board Response Models

// KanbanColumn represents a column in the kanban board
type KanbanColumn struct {
	Status    TaskStatus `json:"status" example:"backlog"`
	Title     string     `json:"title" example:"Backlog"`
	Color     string     `json:"color,omitempty" example:"#6c757d"`
	Todos     []Todo     `json:"todos"`
	TodoCount int        `json:"todo_count" example:"5"`
	MaxItems  *int       `json:"max_items,omitempty" example:"10"`
}

// KanbanBoard represents the complete kanban board for a project
type KanbanBoard struct {
	ProjectID    uuid.UUID      `json:"project_id"`
	ProjectTitle string         `json:"project_title" example:"Mobile App Development"`
	Columns      []KanbanColumn `json:"columns"`
	TotalTodos   int            `json:"total_todos" example:"20"`
	LastUpdated  time.Time      `json:"last_updated" example:"2023-12-01T10:00:00Z"`
}

// KanbanBoardResponse represents the response for kanban board API
type KanbanBoardResponse struct {
	Board   KanbanBoard `json:"board"`
	Stats   BoardStats  `json:"stats"`
	Message string      `json:"message" example:"Kanban board retrieved successfully"`
}

// BoardStats represents statistics for the kanban board
type BoardStats struct {
	TotalTodos      int `json:"total_todos" example:"20"`
	CompletedTodos  int `json:"completed_todos" example:"8"`
	InProgressTodos int `json:"in_progress_todos" example:"5"`
	BacklogTodos    int `json:"backlog_todos" example:"7"`
	CompletionRate  int `json:"completion_rate" example:"40"` // percentage
}

// TodoMoveResponse represents the response for moving a todo
type TodoMoveResponse struct {
	TodoID      uuid.UUID  `json:"todo_id"`
	OldStatus   TaskStatus `json:"old_status" example:"backlog"`
	NewStatus   TaskStatus `json:"new_status" example:"in_progress"`
	NewPosition int        `json:"new_position" example:"2"`
	Message     string     `json:"message" example:"Todo moved successfully"`
}

// BulkMoveResponse represents the response for bulk moving todos
type BulkMoveResponse struct {
	MovedTodos   []TodoMoveResponse `json:"moved_todos"`
	SuccessCount int                `json:"success_count" example:"3"`
	FailureCount int                `json:"failure_count" example:"0"`
	Failures     []MoveFailure      `json:"failures,omitempty"`
	Message      string             `json:"message" example:"Bulk move completed successfully"`
}

// MoveFailure represents a failed todo move in bulk operations
type MoveFailure struct {
	TodoID uuid.UUID `json:"todo_id"`
	Reason string    `json:"reason" example:"Todo not found"`
}

// ReorderResponse represents the response for reordering todos
type ReorderResponse struct {
	Status       TaskStatus  `json:"status" example:"in_progress"`
	ReorderedIDs []uuid.UUID `json:"reordered_ids"`
	Message      string      `json:"message" example:"Todos reordered successfully"`
}
