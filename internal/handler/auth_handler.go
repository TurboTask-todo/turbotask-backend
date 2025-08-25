package handler

import (
	"net/http"
	"strings"

	"macwrite-auth-api/internal/middleware"
	"macwrite-auth-api/internal/models"
	"macwrite-auth-api/internal/service"

	"github.com/gin-gonic/gin"
)

// AuthHandler handles authentication HTTP requests
type AuthHandler struct {
	authService *service.AuthService
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(authService *service.AuthService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
	}
}

// Register handles user registration
// @Summary Register a new user
// @Description Register a new user with email and password
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body models.RegisterRequest true "Registration details"
// @Success 201 {object} models.APIResponse{data=models.AuthResponse} "User registered successfully"
// @Failure 400 {object} models.APIResponse "Invalid request"
// @Failure 409 {object} models.APIResponse "User already exists"
// @Failure 500 {object} models.APIResponse "Internal server error"
// @Router /auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	var req models.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse(
			"INVALID_REQUEST",
			"Invalid request data",
			map[string]string{"details": err.Error()},
		))
		return
	}

	// Register user
	authResponse, err := h.authService.RegisterUser(c.Request.Context(), &req)
	if err != nil {
		statusCode := http.StatusInternalServerError
		errorCode := "REGISTRATION_FAILED"

		if strings.Contains(err.Error(), "validation failed") {
			statusCode = http.StatusBadRequest
			errorCode = "VALIDATION_ERROR"
		} else if strings.Contains(err.Error(), "already exists") {
			statusCode = http.StatusConflict
			errorCode = "USER_EXISTS"
		}

		c.JSON(statusCode, models.NewErrorResponse(
			errorCode,
			err.Error(),
			nil,
		))
		return
	}

	c.JSON(http.StatusCreated, models.NewAPIResponse(
		"User registered successfully",
		authResponse,
	))
}

// Login handles user login
// @Summary User login
// @Description Authenticate user with email and password
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body models.LoginRequest true "Login credentials"
// @Success 200 {object} models.APIResponse{data=models.AuthResponse} "Login successful"
// @Failure 400 {object} models.APIResponse "Invalid request"
// @Failure 401 {object} models.APIResponse "Invalid credentials"
// @Failure 423 {object} models.APIResponse "Account locked"
// @Failure 500 {object} models.APIResponse "Internal server error"
// @Router /auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse(
			"INVALID_REQUEST",
			"Invalid request data",
			map[string]string{"details": err.Error()},
		))
		return
	}

	// Add device info from middleware if available
	if deviceInfo, exists := c.Get("device_info"); exists {
		req.DeviceInfo = deviceInfo.(models.JSONB)
	}

	// Login user
	authResponse, err := h.authService.LoginUser(c.Request.Context(), &req)
	if err != nil {
		statusCode := http.StatusInternalServerError
		errorCode := "LOGIN_FAILED"

		if strings.Contains(err.Error(), "invalid credentials") {
			statusCode = http.StatusUnauthorized
			errorCode = "INVALID_CREDENTIALS"
		} else if strings.Contains(err.Error(), "account is locked") {
			statusCode = http.StatusLocked
			errorCode = "ACCOUNT_LOCKED"
		} else if strings.Contains(err.Error(), "not active") {
			statusCode = http.StatusUnauthorized
			errorCode = "ACCOUNT_INACTIVE"
		}

		c.JSON(statusCode, models.NewErrorResponse(
			errorCode,
			err.Error(),
			nil,
		))
		return
	}

	c.JSON(http.StatusOK, models.NewAPIResponse(
		"Login successful",
		authResponse,
	))
}

// RefreshToken handles token refresh
// @Summary Refresh access token
// @Description Get a new access token using refresh token
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body models.RefreshTokenRequest true "Refresh token"
// @Success 200 {object} models.APIResponse{data=models.TokenResponse} "Token refreshed successfully"
// @Failure 400 {object} models.APIResponse "Invalid request"
// @Failure 401 {object} models.APIResponse "Invalid refresh token"
// @Failure 500 {object} models.APIResponse "Internal server error"
// @Router /auth/refresh [post]
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req models.RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse(
			"INVALID_REQUEST",
			"Invalid request data",
			map[string]string{"details": err.Error()},
		))
		return
	}

	// Refresh tokens
	tokenResponse, err := h.authService.RefreshTokens(c.Request.Context(), req.RefreshToken)
	if err != nil {
		statusCode := http.StatusUnauthorized
		errorCode := "INVALID_REFRESH_TOKEN"

		if strings.Contains(err.Error(), "user not found") {
			errorCode = "USER_NOT_FOUND"
		}

		c.JSON(statusCode, models.NewErrorResponse(
			errorCode,
			err.Error(),
			nil,
		))
		return
	}

	c.JSON(http.StatusOK, models.NewAPIResponse(
		"Token refreshed successfully",
		tokenResponse,
	))
}

// Logout handles user logout
// @Summary User logout
// @Description Logout user and invalidate tokens
// @Tags Authentication
// @Security BearerAuth
// @Accept json
// @Produce json
// @Success 200 {object} models.APIResponse "Logout successful"
// @Failure 401 {object} models.APIResponse "Unauthorized"
// @Failure 500 {object} models.APIResponse "Internal server error"
// @Router /auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	// TODO: Implement token revocation logic
	// For now, just return success - client should discard tokens
	c.JSON(http.StatusOK, models.NewAPIResponse(
		"Logout successful",
		nil,
	))
}

// ChangePassword handles password change
// @Summary Change user password
// @Description Change the password for authenticated user
// @Tags Authentication
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body models.ChangePasswordRequest true "Password change details"
// @Success 200 {object} models.APIResponse "Password changed successfully"
// @Failure 400 {object} models.APIResponse "Invalid request"
// @Failure 401 {object} models.APIResponse "Unauthorized"
// @Failure 500 {object} models.APIResponse "Internal server error"
// @Router /auth/change-password [post]
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.NewErrorResponse(
			"UNAUTHORIZED",
			"User ID not found in context",
			nil,
		))
		return
	}

	var req models.ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse(
			"INVALID_REQUEST",
			"Invalid request data",
			map[string]string{"details": err.Error()},
		))
		return
	}

	// Change password
	err = h.authService.ChangePassword(c.Request.Context(), userID, &req)
	if err != nil {
		statusCode := http.StatusInternalServerError
		errorCode := "PASSWORD_CHANGE_FAILED"

		if strings.Contains(err.Error(), "validation failed") ||
			strings.Contains(err.Error(), "do not match") ||
			strings.Contains(err.Error(), "incorrect") {
			statusCode = http.StatusBadRequest
			errorCode = "INVALID_PASSWORD"
		}

		c.JSON(statusCode, models.NewErrorResponse(
			errorCode,
			err.Error(),
			nil,
		))
		return
	}

	c.JSON(http.StatusOK, models.NewAPIResponse(
		"Password changed successfully",
		nil,
	))
}

// GetProfile handles getting user profile
// @Summary Get user profile
// @Description Get current user's profile information
// @Tags User
// @Security BearerAuth
// @Produce json
// @Success 200 {object} models.APIResponse{data=models.UserProfileResponse} "Profile retrieved successfully"
// @Failure 401 {object} models.APIResponse "Unauthorized"
// @Failure 500 {object} models.APIResponse "Internal server error"
// @Router /auth/profile [get]
func (h *AuthHandler) GetProfile(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.NewErrorResponse(
			"UNAUTHORIZED",
			"User ID not found in context",
			nil,
		))
		return
	}

	// Get user profile
	profile, err := h.authService.GetUserProfile(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.NewErrorResponse(
			"PROFILE_FETCH_FAILED",
			err.Error(),
			nil,
		))
		return
	}

	c.JSON(http.StatusOK, models.NewAPIResponse(
		"Profile retrieved successfully",
		profile,
	))
}

// UpdateProfile handles updating user profile
// @Summary Update user profile
// @Description Update current user's profile information
// @Tags User
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body models.UpdateProfileRequest true "Profile update details"
// @Success 200 {object} models.APIResponse{data=models.UserResponse} "Profile updated successfully"
// @Failure 400 {object} models.APIResponse "Invalid request"
// @Failure 401 {object} models.APIResponse "Unauthorized"
// @Failure 409 {object} models.APIResponse "Username already taken"
// @Failure 500 {object} models.APIResponse "Internal server error"
// @Router /auth/profile [put]
func (h *AuthHandler) UpdateProfile(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.NewErrorResponse(
			"UNAUTHORIZED",
			"User ID not found in context",
			nil,
		))
		return
	}

	var req models.UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse(
			"INVALID_REQUEST",
			"Invalid request data",
			map[string]string{"details": err.Error()},
		))
		return
	}

	// Update profile
	updatedUser, err := h.authService.UpdateProfile(c.Request.Context(), userID, &req)
	if err != nil {
		statusCode := http.StatusInternalServerError
		errorCode := "PROFILE_UPDATE_FAILED"

		if strings.Contains(err.Error(), "username already taken") {
			statusCode = http.StatusConflict
			errorCode = "USERNAME_TAKEN"
		}

		c.JSON(statusCode, models.NewErrorResponse(
			errorCode,
			err.Error(),
			nil,
		))
		return
	}

	c.JSON(http.StatusOK, models.NewAPIResponse(
		"Profile updated successfully",
		updatedUser,
	))
}

// ValidateToken validates a JWT token
// @Summary Validate JWT token
// @Description Validate the provided JWT token
// @Tags Authentication
// @Security BearerAuth
// @Produce json
// @Success 200 {object} models.APIResponse "Token is valid"
// @Failure 401 {object} models.APIResponse "Invalid token"
// @Router /auth/validate [get]
func (h *AuthHandler) ValidateToken(c *gin.Context) {
	// If we reach here, the middleware has already validated the token
	claims, err := middleware.GetJWTClaims(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.NewErrorResponse(
			"INVALID_TOKEN",
			"Token validation failed",
			nil,
		))
		return
	}

	response := map[string]interface{}{
		"valid":      true,
		"user_id":    claims.UserID,
		"email":      claims.Email,
		"username":   claims.Username,
		"expires_at": claims.ExpiresAt.Time,
	}

	c.JSON(http.StatusOK, models.NewAPIResponse(
		"Token is valid",
		response,
	))
}

// Me returns current user information (alias for GetProfile)
// @Summary Get current user
// @Description Get current authenticated user's information
// @Tags User
// @Security BearerAuth
// @Produce json
// @Success 200 {object} models.APIResponse{data=models.UserResponse} "User information retrieved successfully"
// @Failure 401 {object} models.APIResponse "Unauthorized"
// @Failure 500 {object} models.APIResponse "Internal server error"
// @Router /auth/me [get]
func (h *AuthHandler) Me(c *gin.Context) {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.NewErrorResponse(
			"UNAUTHORIZED",
			"User ID not found in context",
			nil,
		))
		return
	}

	// Get user profile (just the user part)
	profile, err := h.authService.GetUserProfile(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.NewErrorResponse(
			"USER_FETCH_FAILED",
			err.Error(),
			nil,
		))
		return
	}

	c.JSON(http.StatusOK, models.NewAPIResponse(
		"User information retrieved successfully",
		profile.User,
	))
}

// CheckEmailAvailability checks if an email is available for registration
// @Summary Check email availability
// @Description Check if an email address is available for registration
// @Tags Authentication
// @Accept json
// @Produce json
// @Param email query string true "Email to check"
// @Success 200 {object} models.APIResponse "Email availability status"
// @Failure 400 {object} models.APIResponse "Invalid request"
// @Router /auth/check-email [get]
func (h *AuthHandler) CheckEmailAvailability(c *gin.Context) {
	email := c.Query("email")
	if email == "" {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse(
			"MISSING_EMAIL",
			"Email parameter is required",
			nil,
		))
		return
	}

	// TODO: Implement email availability check through service
	// For now, return a placeholder response
	response := map[string]interface{}{
		"email":     email,
		"available": true,
	}

	c.JSON(http.StatusOK, models.NewAPIResponse(
		"Email availability checked",
		response,
	))
}

// CheckUsernameAvailability checks if a username is available
// @Summary Check username availability
// @Description Check if a username is available for registration
// @Tags Authentication
// @Accept json
// @Produce json
// @Param username query string true "Username to check"
// @Success 200 {object} models.APIResponse "Username availability status"
// @Failure 400 {object} models.APIResponse "Invalid request"
// @Router /auth/check-username [get]
func (h *AuthHandler) CheckUsernameAvailability(c *gin.Context) {
	username := c.Query("username")
	if username == "" {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse(
			"MISSING_USERNAME",
			"Username parameter is required",
			nil,
		))
		return
	}

	// TODO: Implement username availability check through service
	// For now, return a placeholder response
	response := map[string]interface{}{
		"username":  username,
		"available": true,
	}

	c.JSON(http.StatusOK, models.NewAPIResponse(
		"Username availability checked",
		response,
	))
}
