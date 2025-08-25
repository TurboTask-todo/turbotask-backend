package middleware

import (
	"net/http"
	"strings"

	"macwrite-auth-api/internal/models"
	"macwrite-auth-api/pkg/auth"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// AuthMiddleware provides JWT authentication middleware
type AuthMiddleware struct {
	jwtManager *auth.JWTManager
}

// NewAuthMiddleware creates a new auth middleware
func NewAuthMiddleware(jwtManager *auth.JWTManager) *AuthMiddleware {
	return &AuthMiddleware{
		jwtManager: jwtManager,
	}
}

// RequireAuth middleware that requires valid JWT authentication
func (m *AuthMiddleware) RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract token from Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, models.NewErrorResponse(
				"MISSING_AUTH_TOKEN",
				"Authorization header is required",
				nil,
			))
			c.Abort()
			return
		}

		// Extract token from "Bearer <token>" format
		token, err := auth.ExtractTokenFromHeader(authHeader)
		if err != nil {
			c.JSON(http.StatusUnauthorized, models.NewErrorResponse(
				"INVALID_AUTH_HEADER",
				err.Error(),
				nil,
			))
			c.Abort()
			return
		}

		// Validate access token
		claims, err := m.jwtManager.ValidateAccessToken(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, models.NewErrorResponse(
				"INVALID_ACCESS_TOKEN",
				"Invalid or expired access token",
				nil,
			))
			c.Abort()
			return
		}

		// Store user information in context
		c.Set("user_id", claims.UserID)
		c.Set("user_email", claims.Email)
		c.Set("session_id", claims.SessionID)
		c.Set("jwt_claims", claims)

		c.Next()
	}
}

// OptionalAuth middleware that allows both authenticated and unauthenticated requests
func (m *AuthMiddleware) OptionalAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.Next()
			return
		}

		token, err := auth.ExtractTokenFromHeader(authHeader)
		if err != nil {
			c.Next()
			return
		}

		claims, err := m.jwtManager.ValidateAccessToken(token)
		if err != nil {
			c.Next()
			return
		}

		// Store user information in context if token is valid
		c.Set("user_id", claims.UserID)
		c.Set("user_email", claims.Email)
		c.Set("session_id", claims.SessionID)
		c.Set("jwt_claims", claims)

		c.Next()
	}
}

// RequireRefreshToken middleware that requires valid refresh token
func (m *AuthMiddleware) RequireRefreshToken() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, models.NewErrorResponse(
				"MISSING_REFRESH_TOKEN",
				"Authorization header with refresh token is required",
				nil,
			))
			c.Abort()
			return
		}

		token, err := auth.ExtractTokenFromHeader(authHeader)
		if err != nil {
			c.JSON(http.StatusUnauthorized, models.NewErrorResponse(
				"INVALID_AUTH_HEADER",
				err.Error(),
				nil,
			))
			c.Abort()
			return
		}

		claims, err := m.jwtManager.ValidateRefreshToken(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, models.NewErrorResponse(
				"INVALID_REFRESH_TOKEN",
				"Invalid or expired refresh token",
				nil,
			))
			c.Abort()
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("user_email", claims.Email)
		c.Set("session_id", claims.SessionID)
		c.Set("jwt_claims", claims)
		c.Set("refresh_token", token)

		c.Next()
	}
}

// GetUserID extracts user ID from context
func GetUserID(c *gin.Context) (uuid.UUID, error) {
	userID, exists := c.Get("user_id")
	if !exists {
		return uuid.Nil, nil
	}

	id, ok := userID.(uuid.UUID)
	if !ok {
		return uuid.Nil, nil
	}

	return id, nil
}

// GetUserEmail extracts user email from context
func GetUserEmail(c *gin.Context) string {
	email, exists := c.Get("user_email")
	if !exists {
		return ""
	}

	emailStr, ok := email.(string)
	if !ok {
		return ""
	}

	return emailStr
}

// GetSessionID extracts session ID from context
func GetSessionID(c *gin.Context) (uuid.UUID, error) {
	sessionID, exists := c.Get("session_id")
	if !exists {
		return uuid.Nil, nil
	}

	id, ok := sessionID.(uuid.UUID)
	if !ok {
		return uuid.Nil, nil
	}

	return id, nil
}

// GetJWTClaims extracts JWT claims from context
func GetJWTClaims(c *gin.Context) (*auth.JWTClaims, error) {
	claims, exists := c.Get("jwt_claims")
	if !exists {
		return nil, nil
	}

	jwtClaims, ok := claims.(*auth.JWTClaims)
	if !ok {
		return nil, nil
	}

	return jwtClaims, nil
}

// IsAuthenticated checks if the current request is authenticated
func IsAuthenticated(c *gin.Context) bool {
	_, exists := c.Get("user_id")
	return exists
}

// RequireUser middleware that requires a specific user (admin functionality)
func (m *AuthMiddleware) RequireUser(requiredUserID uuid.UUID) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, err := GetUserID(c)
		if err != nil || userID != requiredUserID {
			c.JSON(http.StatusForbidden, models.NewErrorResponse(
				"ACCESS_DENIED",
				"Access denied for this user",
				nil,
			))
			c.Abort()
			return
		}

		c.Next()
	}
}

// ExtractBearerToken extracts bearer token from various sources
func ExtractBearerToken(c *gin.Context) string {
	// Try Authorization header first
	authHeader := c.GetHeader("Authorization")
	if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
		return strings.TrimPrefix(authHeader, "Bearer ")
	}

	// Try query parameter as fallback
	token := c.Query("token")
	if token != "" {
		return token
	}

	// Try form parameter as last resort
	token = c.PostForm("token")
	if token != "" {
		return token
	}

	return ""
}

// ValidateTokenMiddleware provides token validation without authentication requirement
func (m *AuthMiddleware) ValidateTokenMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := ExtractBearerToken(c)
		if token == "" {
			c.JSON(http.StatusBadRequest, models.NewErrorResponse(
				"MISSING_TOKEN",
				"Token is required",
				nil,
			))
			c.Abort()
			return
		}

		claims, err := m.jwtManager.ValidateToken(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, models.NewErrorResponse(
				"INVALID_TOKEN",
				"Invalid or expired token",
				nil,
			))
			c.Abort()
			return
		}

		c.Set("token", token)
		c.Set("jwt_claims", claims)
		c.Next()
	}
}
