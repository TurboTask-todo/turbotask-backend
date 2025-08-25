package models

import (
	"time"

	"github.com/google/uuid"
)

// OTPPurpose represents the purpose of OTP
type OTPPurpose string

const (
	OTPPurposeLogin    OTPPurpose = "login"
	OTPPurposeRegister OTPPurpose = "register"
	OTPPurposeReset    OTPPurpose = "password_reset"
)

// OTP represents the email OTP verification system
type OTP struct {
	ID           uuid.UUID  `json:"id" db:"id"`
	Email        string     `json:"email" db:"email" validate:"required,email"`
	Code         string     `json:"-" db:"code"` // Never expose in JSON
	Purpose      OTPPurpose `json:"purpose" db:"purpose"`
	IsVerified   bool       `json:"is_verified" db:"is_verified"`
	AttemptCount int        `json:"attempt_count" db:"attempt_count"`
	ExpiresAt    time.Time  `json:"expires_at" db:"expires_at"`
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`
	VerifiedAt   *time.Time `json:"verified_at,omitempty" db:"verified_at"`
}

// Login request models
type LoginInitiateRequest struct {
	Email string `json:"email" validate:"required,email"`
}

type LoginVerifyRequest struct {
	Email string `json:"email" validate:"required,email"`
	Code  string `json:"code" validate:"required,len=6"`
}

// Login response models
type LoginInitiateResponse struct {
	Message   string `json:"message"`
	Email     string `json:"email"`
	ExpiresAt string `json:"expires_at"`
}

type LoginVerifyResponse struct {
	Success      bool          `json:"success"`
	Message      string        `json:"message"`
	AccessToken  string        `json:"access_token,omitempty"`
	RefreshToken string        `json:"refresh_token,omitempty"`
	User         *UserResponse `json:"user,omitempty"`
	IsNewUser    bool          `json:"is_new_user"`
}
