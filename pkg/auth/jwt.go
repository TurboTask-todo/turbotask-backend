package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// JWTClaims represents the JWT claims
type JWTClaims struct {
	UserID    uuid.UUID `json:"user_id"`
	Email     string    `json:"email"`
	Username  *string   `json:"username,omitempty"`
	TokenType string    `json:"token_type"` // "access" or "refresh"
	SessionID uuid.UUID `json:"session_id,omitempty"`
	jwt.RegisteredClaims
}

// JWTManager handles JWT token operations
type JWTManager struct {
	secretKey            []byte
	accessTokenDuration  time.Duration
	refreshTokenDuration time.Duration
	issuer               string
}

// NewJWTManager creates a new JWT manager
func NewJWTManager(secretKey string, accessTokenDuration, refreshTokenDuration time.Duration, issuer string) *JWTManager {
	return &JWTManager{
		secretKey:            []byte(secretKey),
		accessTokenDuration:  accessTokenDuration,
		refreshTokenDuration: refreshTokenDuration,
		issuer:               issuer,
	}
}

// GenerateAccessToken generates an access token for a user
func (m *JWTManager) GenerateAccessToken(userID uuid.UUID, email string, username *string, sessionID uuid.UUID) (string, *JWTClaims, error) {
	now := time.Now()
	expiresAt := now.Add(m.accessTokenDuration)

	claims := &JWTClaims{
		UserID:    userID,
		Email:     email,
		Username:  username,
		TokenType: "access",
		SessionID: sessionID,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.New().String(),
			Issuer:    m.issuer,
			Subject:   userID.String(),
			Audience:  []string{"macwrite-api"},
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			NotBefore: jwt.NewNumericDate(now),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(m.secretKey)
	if err != nil {
		return "", nil, fmt.Errorf("failed to sign access token: %w", err)
	}

	return tokenString, claims, nil
}

// GenerateRefreshToken generates a refresh token for a user
func (m *JWTManager) GenerateRefreshToken(userID uuid.UUID, email string, username *string, sessionID uuid.UUID) (string, *JWTClaims, error) {
	now := time.Now()
	expiresAt := now.Add(m.refreshTokenDuration)

	claims := &JWTClaims{
		UserID:    userID,
		Email:     email,
		Username:  username,
		TokenType: "refresh",
		SessionID: sessionID,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.New().String(),
			Issuer:    m.issuer,
			Subject:   userID.String(),
			Audience:  []string{"macwrite-api"},
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			NotBefore: jwt.NewNumericDate(now),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(m.secretKey)
	if err != nil {
		return "", nil, fmt.Errorf("failed to sign refresh token: %w", err)
	}

	return tokenString, claims, nil
}

// ValidateToken validates a JWT token and returns the claims
func (m *JWTManager) ValidateToken(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return m.secretKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	claims, ok := token.Claims.(*JWTClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	// Check if token is expired
	if claims.ExpiresAt != nil && time.Now().After(claims.ExpiresAt.Time) {
		return nil, fmt.Errorf("token has expired")
	}

	// Check if token is not yet valid
	if claims.NotBefore != nil && time.Now().Before(claims.NotBefore.Time) {
		return nil, fmt.Errorf("token is not yet valid")
	}

	return claims, nil
}

// ValidateAccessToken validates an access token specifically
func (m *JWTManager) ValidateAccessToken(tokenString string) (*JWTClaims, error) {
	claims, err := m.ValidateToken(tokenString)
	if err != nil {
		return nil, err
	}

	if claims.TokenType != "access" {
		return nil, fmt.Errorf("invalid token type: expected access, got %s", claims.TokenType)
	}

	return claims, nil
}

// ValidateRefreshToken validates a refresh token specifically
func (m *JWTManager) ValidateRefreshToken(tokenString string) (*JWTClaims, error) {
	claims, err := m.ValidateToken(tokenString)
	if err != nil {
		return nil, err
	}

	if claims.TokenType != "refresh" {
		return nil, fmt.Errorf("invalid token type: expected refresh, got %s", claims.TokenType)
	}

	return claims, nil
}

// ExtractTokenFromHeader extracts a JWT token from Authorization header
func ExtractTokenFromHeader(authHeader string) (string, error) {
	if authHeader == "" {
		return "", fmt.Errorf("authorization header is required")
	}

	const bearerPrefix = "Bearer "
	if len(authHeader) < len(bearerPrefix) || authHeader[:len(bearerPrefix)] != bearerPrefix {
		return "", fmt.Errorf("authorization header must start with 'Bearer '")
	}

	token := authHeader[len(bearerPrefix):]
	if token == "" {
		return "", fmt.Errorf("token is required")
	}

	return token, nil
}

// GetTokenExpiration returns the expiration time for a token type
func (m *JWTManager) GetTokenExpiration(tokenType string) time.Duration {
	switch tokenType {
	case "access":
		return m.accessTokenDuration
	case "refresh":
		return m.refreshTokenDuration
	default:
		return 0
	}
}

// IsTokenExpiringSoon checks if a token is expiring within the specified threshold
func (m *JWTManager) IsTokenExpiringSoon(claims *JWTClaims, threshold time.Duration) bool {
	if claims.ExpiresAt == nil {
		return false
	}

	timeUntilExpiry := time.Until(claims.ExpiresAt.Time)
	return timeUntilExpiry <= threshold
}

// RefreshTokens generates new access and refresh tokens
func (m *JWTManager) RefreshTokens(refreshClaims *JWTClaims) (accessToken, newRefreshToken string, accessClaims, refreshClaims2 *JWTClaims, err error) {
	// Generate new access token
	accessToken, accessClaims, err = m.GenerateAccessToken(
		refreshClaims.UserID,
		refreshClaims.Email,
		refreshClaims.Username,
		refreshClaims.SessionID,
	)
	if err != nil {
		return "", "", nil, nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	// Generate new refresh token
	newRefreshToken, refreshClaims2, err = m.GenerateRefreshToken(
		refreshClaims.UserID,
		refreshClaims.Email,
		refreshClaims.Username,
		refreshClaims.SessionID,
	)
	if err != nil {
		return "", "", nil, nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	return accessToken, newRefreshToken, accessClaims, refreshClaims2, nil
}
