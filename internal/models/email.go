package models

import "time"

// EmailMessage represents a queued email message
type EmailMessage struct {
	MessageID   string            `json:"message_id"`
	To          string            `json:"to"`
	Subject     string            `json:"subject"`
	Type        EmailMessageType  `json:"type"`
	Data        map[string]string `json:"data"`
	Priority    int               `json:"priority"`
	RetryCount  int               `json:"retry_count"`
	MaxRetries  int               `json:"max_retries"`
	QueuedAt    time.Time         `json:"queued_at"`
	ProcessedAt *time.Time        `json:"processed_at,omitempty"`
}

// EmailMessageType represents the type of email message
type EmailMessageType string

const (
	EmailTypeOTP     EmailMessageType = "otp"
	EmailTypeWelcome EmailMessageType = "welcome"
	EmailTypeReset   EmailMessageType = "reset"
)

// OTPEmailData represents data for OTP email
type OTPEmailData struct {
	Code      string     `json:"code"`
	Email     string     `json:"email"`
	Purpose   OTPPurpose `json:"purpose"`
	ExpiresIn string     `json:"expires_in"`
}
