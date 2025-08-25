package handler

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"time"

	"macwrite-auth-api/internal/models"
	"macwrite-auth-api/internal/service"
	"macwrite-auth-api/pkg/oauth"

	"github.com/gin-gonic/gin"
)

// OAuthHandler handles OAuth authentication requests
type OAuthHandler struct {
	authService *service.AuthService
	googleOAuth *oauth.GoogleOAuth
}

// NewOAuthHandler creates a new OAuth handler
func NewOAuthHandler(authService *service.AuthService, googleOAuth *oauth.GoogleOAuth) *OAuthHandler {
	return &OAuthHandler{
		authService: authService,
		googleOAuth: googleOAuth,
	}
}

// GoogleAuth initiates Google OAuth flow
// @Summary Start Google OAuth
// @Description Initiate Google OAuth authentication flow
// @Tags OAuth
// @Accept json
// @Produce json
// @Success 200 {object} models.APIResponse{data=models.OAuthURLResponse} "OAuth URL generated successfully"
// @Failure 500 {object} models.APIResponse "Internal server error"
// @Router /auth/google [get]
func (h *OAuthHandler) GoogleAuth(c *gin.Context) {
	// Generate random state parameter for security
	state, err := generateRandomState()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.NewErrorResponse(
			"STATE_GENERATION_FAILED",
			"Failed to generate OAuth state",
			nil,
		))
		return
	}

	// Store state in session or cache (for now, we'll include it in response)
	// In production, you should store this in Redis or similar
	authURL := h.googleOAuth.GetAuthURL(state)

	response := &models.OAuthURLResponse{
		AuthURL: authURL,
		State:   state,
	}

	c.JSON(http.StatusOK, models.NewAPIResponse(
		"OAuth URL generated successfully",
		response,
	))
}

// GoogleCallback handles Google OAuth callback
// @Summary Google OAuth callback
// @Description Handle Google OAuth callback and authenticate user
// @Tags OAuth
// @Accept json
// @Produce json
// @Param code query string true "Authorization code from Google"
// @Param state query string true "State parameter for security"
// @Success 200 {object} models.APIResponse{data=models.AuthResponse} "OAuth authentication successful"
// @Failure 400 {object} models.APIResponse "Invalid request"
// @Failure 401 {object} models.APIResponse "OAuth authentication failed"
// @Failure 500 {object} models.APIResponse "Internal server error"
// @Router /auth/google/callback [get]
func (h *OAuthHandler) GoogleCallback(c *gin.Context) {
	code := c.Query("code")
	state := c.Query("state")

	if code == "" {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse(
			"MISSING_AUTH_CODE",
			"Authorization code is required",
			nil,
		))
		return
	}

	if state == "" {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse(
			"MISSING_STATE",
			"State parameter is required",
			nil,
		))
		return
	}

	// TODO: Validate state parameter against stored value
	// For now, we'll skip this validation, but in production this is crucial

	// Exchange code for tokens
	token, err := h.googleOAuth.ExchangeCode(c.Request.Context(), code)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.NewErrorResponse(
			"OAUTH_TOKEN_EXCHANGE_FAILED",
			"Failed to exchange authorization code for tokens",
			map[string]string{"details": err.Error()},
		))
		return
	}

	// Get user info from Google
	userInfo, err := h.googleOAuth.GetUserInfo(c.Request.Context(), token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.NewErrorResponse(
			"OAUTH_USER_INFO_FAILED",
			"Failed to get user information from Google",
			map[string]string{"details": err.Error()},
		))
		return
	}

	// TODO: Implement OAuth user handling in auth service
	// For now, return the user info we received
	response := map[string]interface{}{
		"provider":  "google",
		"user_info": userInfo,
		"message":   "OAuth authentication successful (user creation/login not yet implemented)",
	}

	c.JSON(http.StatusOK, models.NewAPIResponse(
		"OAuth authentication successful",
		response,
	))
}

// GoogleConnect connects Google account to existing user
// @Summary Connect Google account
// @Description Connect Google account to current authenticated user
// @Tags OAuth
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body models.OAuthCallbackRequest true "OAuth callback data"
// @Success 200 {object} models.APIResponse "Google account connected successfully"
// @Failure 400 {object} models.APIResponse "Invalid request"
// @Failure 401 {object} models.APIResponse "Unauthorized"
// @Failure 409 {object} models.APIResponse "Account already connected"
// @Failure 500 {object} models.APIResponse "Internal server error"
// @Router /auth/google/connect [post]
func (h *OAuthHandler) GoogleConnect(c *gin.Context) {
	// TODO: Implement connecting Google account to existing user
	c.JSON(http.StatusNotImplemented, models.NewErrorResponse(
		"NOT_IMPLEMENTED",
		"Google account connection not yet implemented",
		nil,
	))
}

// GoogleDisconnect disconnects Google account from user
// @Summary Disconnect Google account
// @Description Disconnect Google account from current authenticated user
// @Tags OAuth
// @Security BearerAuth
// @Accept json
// @Produce json
// @Success 200 {object} models.APIResponse "Google account disconnected successfully"
// @Failure 401 {object} models.APIResponse "Unauthorized"
// @Failure 404 {object} models.APIResponse "Google account not connected"
// @Failure 500 {object} models.APIResponse "Internal server error"
// @Router /auth/google/disconnect [post]
func (h *OAuthHandler) GoogleDisconnect(c *gin.Context) {
	// TODO: Implement disconnecting Google account from user
	c.JSON(http.StatusNotImplemented, models.NewErrorResponse(
		"NOT_IMPLEMENTED",
		"Google account disconnection not yet implemented",
		nil,
	))
}

// GetOAuthProviders returns list of available OAuth providers
// @Summary Get OAuth providers
// @Description Get list of available OAuth providers and their status
// @Tags OAuth
// @Accept json
// @Produce json
// @Success 200 {object} models.APIResponse "OAuth providers retrieved successfully"
// @Router /auth/oauth/providers [get]
func (h *OAuthHandler) GetOAuthProviders(c *gin.Context) {
	providers := []map[string]interface{}{
		{
			"name":         "google",
			"display_name": "Google",
			"enabled":      true,
			"scopes":       h.googleOAuth.GetScopes(),
		},
		{
			"name":         "github",
			"display_name": "GitHub",
			"enabled":      false,
			"message":      "Coming soon",
		},
		{
			"name":         "microsoft",
			"display_name": "Microsoft",
			"enabled":      false,
			"message":      "Coming soon",
		},
	}

	c.JSON(http.StatusOK, models.NewAPIResponse(
		"OAuth providers retrieved successfully",
		map[string]interface{}{
			"providers": providers,
			"count":     len(providers),
		},
	))
}

// RefreshOAuthToken refreshes an OAuth token
// @Summary Refresh OAuth token
// @Description Refresh an OAuth access token using refresh token
// @Tags OAuth
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param provider path string true "OAuth provider (e.g., google)"
// @Success 200 {object} models.APIResponse "OAuth token refreshed successfully"
// @Failure 400 {object} models.APIResponse "Invalid request"
// @Failure 401 {object} models.APIResponse "Unauthorized"
// @Failure 404 {object} models.APIResponse "OAuth account not found"
// @Failure 500 {object} models.APIResponse "Internal server error"
// @Router /auth/oauth/{provider}/refresh [post]
func (h *OAuthHandler) RefreshOAuthToken(c *gin.Context) {
	provider := c.Param("provider")
	if provider == "" {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse(
			"MISSING_PROVIDER",
			"OAuth provider is required",
			nil,
		))
		return
	}

	// TODO: Implement OAuth token refresh
	c.JSON(http.StatusNotImplemented, models.NewErrorResponse(
		"NOT_IMPLEMENTED",
		"OAuth token refresh not yet implemented",
		nil,
	))
}

// RevokeOAuthToken revokes an OAuth token
// @Summary Revoke OAuth token
// @Description Revoke an OAuth access token
// @Tags OAuth
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param provider path string true "OAuth provider (e.g., google)"
// @Success 200 {object} models.APIResponse "OAuth token revoked successfully"
// @Failure 400 {object} models.APIResponse "Invalid request"
// @Failure 401 {object} models.APIResponse "Unauthorized"
// @Failure 404 {object} models.APIResponse "OAuth account not found"
// @Failure 500 {object} models.APIResponse "Internal server error"
// @Router /auth/oauth/{provider}/revoke [post]
func (h *OAuthHandler) RevokeOAuthToken(c *gin.Context) {
	provider := c.Param("provider")
	if provider == "" {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse(
			"MISSING_PROVIDER",
			"OAuth provider is required",
			nil,
		))
		return
	}

	// TODO: Implement OAuth token revocation
	c.JSON(http.StatusNotImplemented, models.NewErrorResponse(
		"NOT_IMPLEMENTED",
		"OAuth token revocation not yet implemented",
		nil,
	))
}

// GetConnectedAccounts returns list of connected OAuth accounts for user
// @Summary Get connected OAuth accounts
// @Description Get list of OAuth accounts connected to current user
// @Tags OAuth
// @Security BearerAuth
// @Accept json
// @Produce json
// @Success 200 {object} models.APIResponse "Connected accounts retrieved successfully"
// @Failure 401 {object} models.APIResponse "Unauthorized"
// @Failure 500 {object} models.APIResponse "Internal server error"
// @Router /auth/oauth/accounts [get]
func (h *OAuthHandler) GetConnectedAccounts(c *gin.Context) {
	// TODO: Implement getting connected OAuth accounts
	c.JSON(http.StatusNotImplemented, models.NewErrorResponse(
		"NOT_IMPLEMENTED",
		"Getting connected OAuth accounts not yet implemented",
		nil,
	))
}

// generateRandomState generates a random state parameter for OAuth security
func generateRandomState() (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// validateState validates the OAuth state parameter
func validateState(receivedState, expectedState string) bool {
	return receivedState == expectedState && receivedState != ""
}

// extractDeviceInfo extracts device information from request
func extractDeviceInfo(c *gin.Context) models.JSONB {
	deviceInfo := models.JSONB{
		"user_agent": c.GetHeader("User-Agent"),
		"ip_address": c.ClientIP(),
		"timestamp":  time.Now().Unix(),
	}

	// Add additional device info if available
	if info, exists := c.Get("device_info"); exists {
		if existingInfo, ok := info.(models.JSONB); ok {
			for k, v := range existingInfo {
				deviceInfo[k] = v
			}
		}
	}

	return deviceInfo
}
