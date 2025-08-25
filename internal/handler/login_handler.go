package handler

import (
	"net/http"
	"time"

	"macwrite-auth-api/internal/models"
	"macwrite-auth-api/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

// LoginHandler handles login and OTP-related requests
type LoginHandler struct {
	otpService  *service.OTPService
	userService *service.UserService
	authService *service.AuthService
	validator   *validator.Validate
}

// NewLoginHandler creates a new login handler
func NewLoginHandler(
	otpService *service.OTPService,
	userService *service.UserService,
	authService *service.AuthService,
) *LoginHandler {
	return &LoginHandler{
		otpService:  otpService,
		userService: userService,
		authService: authService,
		validator:   validator.New(),
	}
}

// InitiateLogin sends OTP to user's email for login
// @Summary Initiate login with email
// @Description Send OTP code to user's email for authentication
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body models.LoginInitiateRequest true "Login initiate request"
// @Success 200 {object} models.LoginInitiateResponse
// @Failure 400 {object} models.APIResponse
// @Failure 429 {object} models.APIResponse "Too many requests"
// @Failure 500 {object} models.APIResponse
// @Router /api/v1/auth/login/initiate [post]
func (h *LoginHandler) InitiateLogin(c *gin.Context) {
	var req models.LoginInitiateRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Message: "Invalid request format",
			Error: &models.APIError{
				Code:    "INVALID_REQUEST",
				Message: err.Error(),
			},
		})
		return
	}

	// Validate request
	if err := h.validator.Struct(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Message: "Validation failed",
			Error: &models.APIError{
				Code:    "VALIDATION_ERROR",
				Message: err.Error(),
			},
		})
		return
	}

	// Send OTP
	otp, err := h.otpService.SendOTP(c.Request.Context(), req.Email, models.OTPPurposeLogin)
	if err != nil {
		statusCode := http.StatusInternalServerError
		errorCode := "OTP_SEND_FAILED"

		// Handle specific error cases
		if err.Error() == "no account found with this email address" {
			statusCode = http.StatusNotFound
			errorCode = "USER_NOT_FOUND"
		} else if err.Error() == "too many OTP requests. Please wait 10 minutes before requesting again" {
			statusCode = http.StatusTooManyRequests
			errorCode = "RATE_LIMIT_EXCEEDED"
		}

		c.JSON(statusCode, models.APIResponse{
			Success: false,
			Message: "Failed to send OTP",
			Error: &models.APIError{
				Code:    errorCode,
				Message: err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, models.LoginInitiateResponse{
		Message:   "OTP sent successfully to your email",
		Email:     req.Email,
		ExpiresAt: otp.ExpiresAt.Format(time.RFC3339),
	})
}

// VerifyLogin verifies OTP and returns authentication tokens
// @Summary Verify OTP and complete login
// @Description Verify the OTP code and return JWT tokens
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body models.LoginVerifyRequest true "Login verify request"
// @Success 200 {object} models.LoginVerifyResponse
// @Failure 400 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse "Invalid OTP"
// @Failure 500 {object} models.APIResponse
// @Router /api/v1/auth/login/verify [post]
func (h *LoginHandler) VerifyLogin(c *gin.Context) {
	var req models.LoginVerifyRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Message: "Invalid request format",
			Error: &models.APIError{
				Code:    "INVALID_REQUEST",
				Message: err.Error(),
			},
		})
		return
	}

	// Validate request
	if err := h.validator.Struct(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Message: "Validation failed",
			Error: &models.APIError{
				Code:    "VALIDATION_ERROR",
				Message: err.Error(),
			},
		})
		return
	}

	// Verify OTP
	otp, err := h.otpService.VerifyOTP(c.Request.Context(), req.Email, req.Code, models.OTPPurposeLogin)
	if err != nil {
		statusCode := http.StatusUnauthorized
		errorCode := "OTP_VERIFICATION_FAILED"

		// Handle specific error cases
		if err.Error() == "no active OTP found. Please request a new one" ||
			err.Error() == "OTP has expired. Please request a new one" {
			errorCode = "OTP_EXPIRED"
		} else if err.Error() == "invalid OTP code" {
			errorCode = "INVALID_OTP"
		} else if err.Error() == "too many failed attempts. Please request a new OTP" {
			errorCode = "TOO_MANY_ATTEMPTS"
		}

		c.JSON(statusCode, models.APIResponse{
			Success: false,
			Message: "OTP verification failed",
			Error: &models.APIError{
				Code:    errorCode,
				Message: err.Error(),
			},
		})
		return
	}

	// Get user details or create if doesn't exist
	user, err := h.userService.GetByEmail(c.Request.Context(), otp.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Message: "Failed to get user details",
			Error: &models.APIError{
				Code:    "USER_FETCH_FAILED",
				Message: err.Error(),
			},
		})
		return
	}

	// If user doesn't exist, create a new one with default data
	isNewUser := false
	if user == nil {
		user, err = h.userService.CreateUserFromEmail(c.Request.Context(), otp.Email)
		if err != nil {
			c.JSON(http.StatusInternalServerError, models.APIResponse{
				Success: false,
				Message: "Failed to create user account",
				Error: &models.APIError{
					Code:    "USER_CREATION_FAILED",
					Message: err.Error(),
				},
			})
			return
		}
		isNewUser = true
	}

	// Generate proper JWT tokens using auth service
	authResponse, err := h.authService.GenerateOTPTokens(c.Request.Context(), user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Message: "Failed to generate authentication tokens",
			Error: &models.APIError{
				Code:    "TOKEN_GENERATION_FAILED",
				Message: err.Error(),
			},
		})
		return
	}

	// Update user's last login
	if err := h.userService.UpdateLastLogin(c.Request.Context(), user.ID); err != nil {
		// Log this error but don't fail the login
		// logger.Error("Failed to update last login", "error", err, "user_id", user.ID)
	}

	message := "Login successful"
	if isNewUser {
		message = "Account created and login successful"
	}

	c.JSON(http.StatusOK, models.LoginVerifyResponse{
		Success:      true,
		Message:      message,
		AccessToken:  authResponse.AccessToken,
		RefreshToken: authResponse.RefreshToken,
		User:         authResponse.User,
		IsNewUser:    isNewUser,
	})
}

// ResendOTP resends OTP for login
// @Summary Resend OTP for login
// @Description Resend OTP code to user's email
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body models.LoginInitiateRequest true "Resend OTP request"
// @Success 200 {object} models.LoginInitiateResponse
// @Failure 400 {object} models.APIResponse
// @Failure 429 {object} models.APIResponse "Too many requests"
// @Failure 500 {object} models.APIResponse
// @Router /api/v1/auth/login/resend [post]
func (h *LoginHandler) ResendOTP(c *gin.Context) {
	// Resend OTP uses the same logic as InitiateLogin
	h.InitiateLogin(c)
}

// CheckEmail checks if an email exists in the system
// @Summary Check if email exists
// @Description Check if an email address is registered in the system
// @Tags Authentication
// @Accept json
// @Produce json
// @Param email query string true "Email address to check"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} models.APIResponse
// @Failure 500 {object} models.APIResponse
// @Router /api/v1/auth/check-email [get]
func (h *LoginHandler) CheckEmail(c *gin.Context) {
	email := c.Query("email")
	if email == "" {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Message: "Email parameter is required",
			Error: &models.APIError{
				Code:    "MISSING_EMAIL",
				Message: "Email query parameter is required",
			},
		})
		return
	}

	// Validate email format
	if err := h.validator.Var(email, "required,email"); err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Message: "Invalid email format",
			Error: &models.APIError{
				Code:    "INVALID_EMAIL",
				Message: err.Error(),
			},
		})
		return
	}

	// Check if user exists
	user, err := h.userService.GetByEmail(c.Request.Context(), email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Message: "Failed to check email",
			Error: &models.APIError{
				Code:    "EMAIL_CHECK_FAILED",
				Message: err.Error(),
			},
		})
		return
	}

	exists := user != nil

	response := map[string]interface{}{
		"success": true,
		"exists":  exists,
		"email":   email,
	}

	if exists {
		response["username"] = user.Username
		response["first_name"] = user.FirstName
		response["last_name"] = user.LastName
		response["user_id"] = user.ID
	}

	c.JSON(http.StatusOK, response)
}

// Logout handles user logout (invalidate tokens)
// @Summary Logout user
// @Description Logout user and invalidate tokens
// @Tags Authentication
// @Security Bearer
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} models.APIResponse
// @Router /api/v1/auth/logout [post]
func (h *LoginHandler) Logout(c *gin.Context) {
	// For JWT tokens, we typically just rely on token expiration
	// In a production system, you might want to maintain a blacklist
	// of invalidated tokens in Redis or database

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, models.APIResponse{
			Success: false,
			Message: "Unauthorized",
			Error: &models.APIError{
				Code:    "UNAUTHORIZED",
				Message: "No valid authentication token found",
			},
		})
		return
	}

	// Log the logout event (optional)
	// logger.Info("User logged out", "user_id", userID)

	c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Logged out successfully",
		"user_id": userID,
	})
}
