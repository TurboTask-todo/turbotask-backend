package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// GoogleOAuth handles Google OAuth integration
type GoogleOAuth struct {
	config *oauth2.Config
}

// GoogleUserInfo represents user information from Google
type GoogleUserInfo struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	VerifiedEmail bool   `json:"verified_email"`
	Name          string `json:"name"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	Picture       string `json:"picture"`
	Locale        string `json:"locale"`
}

// NewGoogleOAuth creates a new Google OAuth provider
func NewGoogleOAuth(clientID, clientSecret, redirectURL string) *GoogleOAuth {
	config := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		},
		Endpoint: google.Endpoint,
	}

	return &GoogleOAuth{
		config: config,
	}
}

// GetAuthURL generates the authorization URL for Google OAuth
func (g *GoogleOAuth) GetAuthURL(state string) string {
	return g.config.AuthCodeURL(state, oauth2.AccessTypeOffline)
}

// ExchangeCode exchanges the authorization code for tokens
func (g *GoogleOAuth) ExchangeCode(ctx context.Context, code string) (*oauth2.Token, error) {
	token, err := g.config.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code for token: %w", err)
	}

	return token, nil
}

// GetUserInfo retrieves user information from Google using the access token
func (g *GoogleOAuth) GetUserInfo(ctx context.Context, token *oauth2.Token) (*GoogleUserInfo, error) {
	client := g.config.Client(ctx, token)

	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		return nil, fmt.Errorf("failed to get user info from Google: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Google API returned status %d", resp.StatusCode)
	}

	var userInfo GoogleUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, fmt.Errorf("failed to decode user info: %w", err)
	}

	return &userInfo, nil
}

// RefreshToken refreshes an OAuth token
func (g *GoogleOAuth) RefreshToken(ctx context.Context, token *oauth2.Token) (*oauth2.Token, error) {
	if token.RefreshToken == "" {
		return nil, fmt.Errorf("no refresh token available")
	}

	tokenSource := g.config.TokenSource(ctx, token)
	newToken, err := tokenSource.Token()
	if err != nil {
		return nil, fmt.Errorf("failed to refresh token: %w", err)
	}

	return newToken, nil
}

// ValidateToken validates an OAuth token by making a request to Google's tokeninfo endpoint
func (g *GoogleOAuth) ValidateToken(ctx context.Context, accessToken string) (*GoogleTokenInfo, error) {
	url := fmt.Sprintf("https://www.googleapis.com/oauth2/v1/tokeninfo?access_token=%s", accessToken)

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to validate token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token validation failed with status %d", resp.StatusCode)
	}

	var tokenInfo GoogleTokenInfo
	if err := json.NewDecoder(resp.Body).Decode(&tokenInfo); err != nil {
		return nil, fmt.Errorf("failed to decode token info: %w", err)
	}

	// Verify the token is for our application
	if tokenInfo.Audience != g.config.ClientID {
		return nil, fmt.Errorf("token audience mismatch")
	}

	return &tokenInfo, nil
}

// GoogleTokenInfo represents token information from Google's tokeninfo endpoint
type GoogleTokenInfo struct {
	Audience      string `json:"aud"`
	UserID        string `json:"user_id"`
	Scope         string `json:"scope"`
	ExpiresIn     int    `json:"expires_in"`
	Email         string `json:"email,omitempty"`
	VerifiedEmail bool   `json:"verified_email,omitempty"`
}

// RevokeToken revokes an OAuth token
func (g *GoogleOAuth) RevokeToken(ctx context.Context, token string) error {
	url := fmt.Sprintf("https://oauth2.googleapis.com/revoke?token=%s", token)

	resp, err := http.Post(url, "application/x-www-form-urlencoded", nil)
	if err != nil {
		return fmt.Errorf("failed to revoke token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("token revocation failed with status %d", resp.StatusCode)
	}

	return nil
}

// IsTokenExpired checks if a token is expired
func (g *GoogleOAuth) IsTokenExpired(token *oauth2.Token) bool {
	return !token.Valid()
}

// GetScopes returns the configured OAuth scopes
func (g *GoogleOAuth) GetScopes() []string {
	return g.config.Scopes
}

// SetScopes updates the OAuth scopes
func (g *GoogleOAuth) SetScopes(scopes []string) {
	g.config.Scopes = scopes
}

// GetConfig returns the underlying OAuth2 config
func (g *GoogleOAuth) GetConfig() *oauth2.Config {
	return g.config
}
