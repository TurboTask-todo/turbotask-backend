package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"quantumtask-auth-api/internal/config"
	"quantumtask-auth-api/internal/models"
	"quantumtask-auth-api/internal/repository"
	"quantumtask-auth-api/pkg/auth"
	"quantumtask-auth-api/pkg/validation"

	"github.com/google/uuid"
)

// AuthService handles authentication business logic
type AuthService struct {
	userRepo       *repository.UserRepository
	passwordRepo   *repository.PasswordRepository
	jwtManager     *auth.JWTManager
	passwordHasher *auth.PasswordHasher
	emailValidator *validation.EmailValidator
	config         *config.Config
}

// NewAuthService creates a new authentication service
func NewAuthService(
	userRepo *repository.UserRepository,
	passwordRepo *repository.PasswordRepository,
	jwtManager *auth.JWTManager,
	passwordHasher *auth.PasswordHasher,
	emailValidator *validation.EmailValidator,
	config *config.Config,
) *AuthService {
	return &AuthService{
		userRepo:       userRepo,
		passwordRepo:   passwordRepo,
		jwtManager:     jwtManager,
		passwordHasher: passwordHasher,
		emailValidator: emailValidator,
		config:         config,
	}
}

// RegisterUser registers a new user
func (s *AuthService) RegisterUser(ctx context.Context, req *models.RegisterRequest) (*models.AuthResponse, error) {
	// Validate input
	if err := s.validateRegisterRequest(req); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Normalize email
	normalizedEmail := validation.NormalizeEmail(req.Email)

	// Check if user already exists
	existingUser, err := s.userRepo.GetByEmail(ctx, normalizedEmail)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing user: %w", err)
	}
	if existingUser != nil {
		return nil, fmt.Errorf("user with this email already exists")
	}

	// Check username uniqueness if provided
	if req.Username != nil && *req.Username != "" {
		exists, err := s.userRepo.ExistsWithUsername(ctx, *req.Username)
		if err != nil {
			return nil, fmt.Errorf("failed to check username: %w", err)
		}
		if exists {
			return nil, fmt.Errorf("username already taken")
		}
	}

	// Create user
	userID := uuid.New()
	now := time.Now()

	user := &models.User{
		ID:            userID,
		Email:         normalizedEmail,
		Username:      req.Username,
		EmailVerified: false,
		FirstName:     req.FirstName,
		LastName:      req.LastName,
		Timezone:      "UTC",
		Locale:        "en-US",
		Preferences:   models.JSONB{},
		Status:        models.UserStatusPending,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	// Set timezone and locale from request if provided
	if req.Timezone != nil {
		user.Timezone = *req.Timezone
	}
	if req.Locale != nil {
		user.Locale = *req.Locale
	}

	// Create user in database
	err = s.userRepo.Create(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Hash password and create password record
	passwordHash, salt, err := s.passwordHasher.HashPassword(req.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	userPassword := &models.UserPassword{
		ID:           uuid.New(),
		UserID:       userID,
		PasswordHash: passwordHash,
		Salt:         &salt,
		Algorithm:    "argon2id",
		CostParams:   models.JSONB(s.passwordHasher.GetCostParams()),
		Version:      1,
		CreatedAt:    now,
		UpdatedAt:    now,
		IsActive:     true,
	}

	err = s.passwordRepo.Create(ctx, userPassword)
	if err != nil {
		return nil, fmt.Errorf("failed to create password: %w", err)
	}

	// Activate user account
	user.Status = models.UserStatusActive
	err = s.userRepo.Update(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("failed to activate user: %w", err)
	}

	// Generate session and tokens
	sessionID := uuid.New()
	accessToken, _, err := s.jwtManager.GenerateAccessToken(userID, user.Email, user.Username, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, _, err := s.jwtManager.GenerateRefreshToken(userID, user.Email, user.Username, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	return &models.AuthResponse{
		User:         user.ToUserResponse(),
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int(s.config.JWT.AccessTokenDuration.Seconds()),
		ExpiresAt:    time.Now().Add(s.config.JWT.AccessTokenDuration),
	}, nil
}

// LoginUser authenticates a user and returns tokens
func (s *AuthService) LoginUser(ctx context.Context, req *models.LoginRequest) (*models.AuthResponse, error) {
	// Validate input
	if req.Email == "" || req.Password == "" {
		return nil, fmt.Errorf("email and password are required")
	}

	// Normalize email
	normalizedEmail := validation.NormalizeEmail(req.Email)

	// Get user by email
	user, err := s.userRepo.GetByEmail(ctx, normalizedEmail)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	// Check if user can login
	if !user.CanLogin() {
		if user.IsAccountLocked() {
			return nil, fmt.Errorf("account is locked until %v", user.AccountLockedUntil.Format(time.RFC3339))
		}
		if user.Status != models.UserStatusActive {
			return nil, fmt.Errorf("account is not active")
		}
		return nil, fmt.Errorf("login not allowed")
	}

	// Get active password
	userPassword, err := s.passwordRepo.GetActiveByUserID(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user password: %w", err)
	}
	if userPassword == nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	// Verify password
	isValid, err := s.passwordHasher.VerifyPassword(req.Password, userPassword.PasswordHash)
	if err != nil {
		return nil, fmt.Errorf("failed to verify password: %w", err)
	}

	if !isValid {
		// Increment failed login attempts
		err = s.userRepo.IncrementFailedLoginAttempts(ctx, user.ID)
		if err != nil {
			// Log error but don't fail the login attempt
			fmt.Printf("Failed to increment login attempts: %v\n", err)
		}

		// Check if account should be locked
		if user.FailedLoginAttempts+1 >= s.config.Security.MaxLoginAttempts {
			lockUntil := time.Now().Add(time.Duration(s.config.Security.AccountLockDurationMinutes) * time.Minute)
			err = s.userRepo.LockAccount(ctx, user.ID, lockUntil)
			if err != nil {
				fmt.Printf("Failed to lock account: %v\n", err)
			}
			return nil, fmt.Errorf("account has been locked due to too many failed login attempts")
		}

		return nil, fmt.Errorf("invalid credentials")
	}

	// Reset failed login attempts and update last login
	err = s.userRepo.UpdateLastLogin(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to update last login: %w", err)
	}

	// Generate session and tokens
	sessionID := uuid.New()
	accessToken, _, err := s.jwtManager.GenerateAccessToken(user.ID, user.Email, user.Username, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, _, err := s.jwtManager.GenerateRefreshToken(user.ID, user.Email, user.Username, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	return &models.AuthResponse{
		User:         user.ToUserResponse(),
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int(s.config.JWT.AccessTokenDuration.Seconds()),
		ExpiresAt:    time.Now().Add(s.config.JWT.AccessTokenDuration),
	}, nil
}

// GenerateOTPTokens generates tokens for OTP-based login
func (s *AuthService) GenerateOTPTokens(ctx context.Context, user *models.User) (*models.AuthResponse, error) {
	if user == nil {
		return nil, fmt.Errorf("user cannot be nil")
	}

	// Generate session and tokens
	sessionID := uuid.New()
	accessToken, _, err := s.jwtManager.GenerateAccessToken(user.ID, user.Email, user.Username, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, _, err := s.jwtManager.GenerateRefreshToken(user.ID, user.Email, user.Username, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	return &models.AuthResponse{
		User:         user.ToUserResponse(),
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int(s.config.JWT.AccessTokenDuration.Seconds()),
		ExpiresAt:    time.Now().Add(s.config.JWT.AccessTokenDuration),
	}, nil
}

// RefreshTokens refreshes access and refresh tokens
func (s *AuthService) RefreshTokens(ctx context.Context, refreshToken string) (*models.TokenResponse, error) {
	// Validate refresh token
	claims, err := s.jwtManager.ValidateRefreshToken(refreshToken)
	if err != nil {
		return nil, fmt.Errorf("invalid refresh token: %w", err)
	}

	// Get user to ensure they still exist and are active
	user, err := s.userRepo.GetByID(ctx, claims.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil || !user.CanLogin() {
		return nil, fmt.Errorf("user not found or not active")
	}

	// Generate new tokens
	newAccessToken, newRefreshToken, _, _, err := s.jwtManager.RefreshTokens(claims)
	if err != nil {
		return nil, fmt.Errorf("failed to refresh tokens: %w", err)
	}

	return &models.TokenResponse{
		AccessToken:  newAccessToken,
		RefreshToken: newRefreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int(s.config.JWT.AccessTokenDuration.Seconds()),
		ExpiresAt:    time.Now().Add(s.config.JWT.AccessTokenDuration),
	}, nil
}

// ChangePassword changes a user's password
func (s *AuthService) ChangePassword(ctx context.Context, userID uuid.UUID, req *models.ChangePasswordRequest) error {
	// Validate input
	if req.CurrentPassword == "" || req.NewPassword == "" || req.ConfirmPassword == "" {
		return fmt.Errorf("all password fields are required")
	}

	if req.NewPassword != req.ConfirmPassword {
		return fmt.Errorf("new password and confirmation do not match")
	}

	// Validate new password strength
	passwordErrors := auth.ValidatePasswordStrength(req.NewPassword, s.config.Security.PasswordMinLength)
	if len(passwordErrors) > 0 {
		return fmt.Errorf("password validation failed: %s", strings.Join(passwordErrors, ", "))
	}

	// Get user
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return fmt.Errorf("user not found")
	}

	// Get current password
	currentPassword, err := s.passwordRepo.GetActiveByUserID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get current password: %w", err)
	}
	if currentPassword == nil {
		return fmt.Errorf("no active password found")
	}

	// Verify current password
	isValid, err := s.passwordHasher.VerifyPassword(req.CurrentPassword, currentPassword.PasswordHash)
	if err != nil {
		return fmt.Errorf("failed to verify current password: %w", err)
	}
	if !isValid {
		return fmt.Errorf("current password is incorrect")
	}

	// Hash new password
	newPasswordHash, salt, err := s.passwordHasher.HashPassword(req.NewPassword)
	if err != nil {
		return fmt.Errorf("failed to hash new password: %w", err)
	}

	// Check password reuse
	isReused, err := s.passwordRepo.IsPasswordReused(ctx, userID, newPasswordHash, 5)
	if err != nil {
		return fmt.Errorf("failed to check password reuse: %w", err)
	}
	if isReused {
		return fmt.Errorf("cannot reuse a recent password")
	}

	// Create new password record
	newPassword := &models.UserPassword{
		ID:           uuid.New(),
		UserID:       userID,
		PasswordHash: newPasswordHash,
		Salt:         &salt,
		Algorithm:    "argon2id",
		CostParams:   models.JSONB(s.passwordHasher.GetCostParams()),
		Version:      1,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		IsActive:     true,
	}

	// Create and set new password as active
	err = s.passwordRepo.CreateAndSetActive(ctx, newPassword)
	if err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	// Update password changed timestamp
	err = s.userRepo.ChangePassword(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to update password timestamp: %w", err)
	}

	return nil
}

// GetUserProfile returns user profile information
func (s *AuthService) GetUserProfile(ctx context.Context, userID uuid.UUID) (*models.UserProfileResponse, error) {
	// Get user
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return nil, fmt.Errorf("user not found")
	}

	// Get auth summary
	authSummary, err := s.userRepo.GetAuthSummary(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get auth summary: %w", err)
	}

	// Build auth methods info
	authMethods := &models.AuthMethodsInfo{
		HasPassword:    false,
		OAuthProviders: []string{},
		MFAMethods:     []string{},
		MFAEnabled:     false,
	}

	if authSummary != nil {
		authMethods.HasPassword = authSummary.HasPassword
		authMethods.OAuthProviders = authSummary.OAuthProviders
		authMethods.MFAMethods = authSummary.MFAMethods
		authMethods.MFAEnabled = len(authSummary.MFAMethods) > 0
	}

	// Build security info
	securityInfo := &models.SecurityInfo{
		FailedLoginAttempts: user.FailedLoginAttempts,
		AccountLocked:       user.IsAccountLocked(),
		AccountLockedUntil:  user.AccountLockedUntil,
		PasswordChangedAt:   user.PasswordChangedAt,
	}

	return &models.UserProfileResponse{
		User:         user.ToUserResponse(),
		AuthMethods:  authMethods,
		SecurityInfo: securityInfo,
	}, nil
}

// UpdateProfile updates user profile information
func (s *AuthService) UpdateProfile(ctx context.Context, userID uuid.UUID, req *models.UpdateProfileRequest) (*models.UserResponse, error) {
	// Get user
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	if user == nil {
		return nil, fmt.Errorf("user not found")
	}

	// Check username uniqueness if provided and different
	if req.Username != nil && *req.Username != "" {
		if user.Username == nil || *user.Username != *req.Username {
			exists, err := s.userRepo.ExistsWithUsername(ctx, *req.Username)
			if err != nil {
				return nil, fmt.Errorf("failed to check username: %w", err)
			}
			if exists {
				return nil, fmt.Errorf("username already taken")
			}
		}
	}

	// Update user fields
	if req.Username != nil {
		user.Username = req.Username
	}
	if req.FirstName != nil {
		user.FirstName = req.FirstName
	}
	if req.LastName != nil {
		user.LastName = req.LastName
	}
	if req.AvatarURL != nil {
		user.AvatarURL = req.AvatarURL
	}
	if req.Timezone != nil {
		user.Timezone = *req.Timezone
	}
	if req.Locale != nil {
		user.Locale = *req.Locale
	}
	if req.Preferences != nil {
		user.Preferences = req.Preferences
	}

	// Update user in database
	err = s.userRepo.Update(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	return user.ToUserResponse(), nil
}

// validateRegisterRequest validates user registration request
func (s *AuthService) validateRegisterRequest(req *models.RegisterRequest) error {
	// Validate email
	emailErrors := s.emailValidator.ValidateEmail(req.Email)
	if len(emailErrors) > 0 {
		return fmt.Errorf("email validation failed: %s", strings.Join(emailErrors, ", "))
	}

	// Validate password
	passwordErrors := auth.ValidatePasswordStrength(req.Password, s.config.Security.PasswordMinLength)
	if len(passwordErrors) > 0 {
		return fmt.Errorf("password validation failed: %s", strings.Join(passwordErrors, ", "))
	}

	// Validate password confirmation
	if req.Password != req.ConfirmPassword {
		return fmt.Errorf("password and confirmation do not match")
	}

	// Validate username if provided
	if req.Username != nil && *req.Username != "" {
		if len(*req.Username) < 3 {
			return fmt.Errorf("username must be at least 3 characters long")
		}
		if len(*req.Username) > 50 {
			return fmt.Errorf("username must be no more than 50 characters long")
		}
	}

	return nil
}
