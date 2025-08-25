package service

import (
	"context"
	"crypto/rand"
	"database/sql"
	"fmt"
	"math/big"
	"strings"
	"time"

	"macwrite-auth-api/internal/models"
	"macwrite-auth-api/internal/repository"
	"macwrite-auth-api/pkg/email"
)

// OTPService handles OTP business logic
type OTPService struct {
	otpRepo     *repository.OTPRepository
	userRepo    *repository.UserRepository
	emailClient *email.Client
}

// NewOTPService creates a new OTP service
func NewOTPService(
	otpRepo *repository.OTPRepository,
	userRepo *repository.UserRepository,
	emailClient *email.Client,
) *OTPService {
	return &OTPService{
		otpRepo:     otpRepo,
		userRepo:    userRepo,
		emailClient: emailClient,
	}
}

// SendOTP generates and sends an OTP for the specified purpose
func (s *OTPService) SendOTP(ctx context.Context, email string, purpose models.OTPPurpose) (*models.OTP, error) {
	// Validate email format
	if !isValidEmail(email) {
		return nil, fmt.Errorf("invalid email format")
	}

	// Check rate limiting - max 3 OTPs per 10 minutes
	since := time.Now().Add(-10 * time.Minute)
	recentAttempts, err := s.otpRepo.GetRecentAttempts(ctx, email, purpose, since)
	if err != nil {
		return nil, fmt.Errorf("failed to check rate limit: %w", err)
	}

	if recentAttempts >= 3 {
		return nil, fmt.Errorf("too many OTP requests. Please wait 10 minutes before requesting again")
	}

	// For login purpose, we allow OTP for any email
	// User will be created automatically if they don't exist during verification

	// Invalidate any existing active OTPs
	if err := s.otpRepo.InvalidateOTPs(ctx, email, purpose); err != nil {
		return nil, fmt.Errorf("failed to invalidate existing OTPs: %w", err)
	}

	// Generate OTP code
	code, err := s.generateOTPCode()
	if err != nil {
		return nil, fmt.Errorf("failed to generate OTP code: %w", err)
	}

	// Create OTP record
	otp := &models.OTP{
		Email:     email,
		Code:      code,
		Purpose:   purpose,
		ExpiresAt: time.Now().Add(10 * time.Minute), // 10 minutes expiry
	}

	if err := s.otpRepo.Create(ctx, otp); err != nil {
		return nil, fmt.Errorf("failed to create OTP record: %w", err)
	}

	// Send email
	if err := s.emailClient.SendOTP(email, code, purpose, "10 minutes"); err != nil {
		return nil, fmt.Errorf("failed to send OTP email: %w", err)
	}

	return otp, nil
}

// VerifyOTP verifies an OTP code
func (s *OTPService) VerifyOTP(ctx context.Context, email, code string, purpose models.OTPPurpose) (*models.OTP, error) {
	// Validate inputs
	if !isValidEmail(email) {
		return nil, fmt.Errorf("invalid email format")
	}

	if len(code) != 6 {
		return nil, fmt.Errorf("invalid OTP code format")
	}

	// Get active OTP
	otp, err := s.otpRepo.GetActiveOTP(ctx, email, purpose)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no active OTP found. Please request a new one")
		}
		return nil, fmt.Errorf("failed to get active OTP: %w", err)
	}

	// Check if OTP is expired
	if time.Now().After(otp.ExpiresAt) {
		return nil, fmt.Errorf("OTP has expired. Please request a new one")
	}

	// Check attempt limit
	if otp.AttemptCount >= 5 {
		return nil, fmt.Errorf("too many failed attempts. Please request a new OTP")
	}

	// Increment attempt count
	if err := s.otpRepo.IncrementAttempt(ctx, otp.ID); err != nil {
		return nil, fmt.Errorf("failed to increment attempt count: %w", err)
	}

	// Verify code
	if !constantTimeCompare(otp.Code, code) {
		return nil, fmt.Errorf("invalid OTP code")
	}

	// Mark as verified
	if err := s.otpRepo.VerifyOTP(ctx, otp.ID); err != nil {
		return nil, fmt.Errorf("failed to verify OTP: %w", err)
	}

	// Update the OTP object
	otp.IsVerified = true
	now := time.Now()
	otp.VerifiedAt = &now

	return otp, nil
}

// generateOTPCode generates a 6-digit OTP code
func (s *OTPService) generateOTPCode() (string, error) {
	// Generate a 6-digit number
	max := big.NewInt(999999)
	min := big.NewInt(100000)

	n, err := rand.Int(rand.Reader, new(big.Int).Sub(max, min))
	if err != nil {
		return "", err
	}

	n.Add(n, min)
	return fmt.Sprintf("%06d", n.Int64()), nil
}

// isValidEmail performs basic email validation
func isValidEmail(email string) bool {
	// Basic email validation
	if len(email) < 3 || len(email) > 255 {
		return false
	}

	if !strings.Contains(email, "@") {
		return false
	}

	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return false
	}

	if len(parts[0]) == 0 || len(parts[1]) == 0 {
		return false
	}

	if !strings.Contains(parts[1], ".") {
		return false
	}

	return true
}

// constantTimeCompare performs constant-time string comparison to prevent timing attacks
func constantTimeCompare(a, b string) bool {
	if len(a) != len(b) {
		return false
	}

	result := 0
	for i := 0; i < len(a); i++ {
		result |= int(a[i] ^ b[i])
	}

	return result == 0
}

// CleanupExpiredOTPs removes expired OTP records (should be called periodically)
func (s *OTPService) CleanupExpiredOTPs(ctx context.Context) error {
	return s.otpRepo.DeleteExpired(ctx)
}
