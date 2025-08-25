package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

// PasswordHasher handles password hashing and verification
type PasswordHasher struct {
	memory      uint32
	iterations  uint32
	parallelism uint8
	saltLength  uint32
	keyLength   uint32
}

// NewPasswordHasher creates a new password hasher with secure defaults
func NewPasswordHasher() *PasswordHasher {
	return &PasswordHasher{
		memory:      64 * 1024, // 64 MB
		iterations:  3,         // 3 iterations
		parallelism: 2,         // Use 2 threads
		saltLength:  16,        // 16 bytes salt
		keyLength:   32,        // 32 bytes key
	}
}

// NewCustomPasswordHasher creates a password hasher with custom parameters
func NewCustomPasswordHasher(memory uint32, iterations uint32, parallelism uint8, saltLength, keyLength uint32) *PasswordHasher {
	return &PasswordHasher{
		memory:      memory,
		iterations:  iterations,
		parallelism: parallelism,
		saltLength:  saltLength,
		keyLength:   keyLength,
	}
}

// HashPassword hashes a password using Argon2id
func (h *PasswordHasher) HashPassword(password string) (string, string, error) {
	// Generate a random salt
	salt, err := h.generateRandomBytes(h.saltLength)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate salt: %w", err)
	}

	// Hash the password
	hash := argon2.IDKey([]byte(password), salt, h.iterations, h.memory, h.parallelism, h.keyLength)

	// Encode the salt and hash to base64
	saltB64 := base64.RawStdEncoding.EncodeToString(salt)
	hashB64 := base64.RawStdEncoding.EncodeToString(hash)

	// Create the encoded hash string with parameters
	encodedHash := fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, h.memory, h.iterations, h.parallelism, saltB64, hashB64)

	return encodedHash, saltB64, nil
}

// VerifyPassword verifies a password against its hash
func (h *PasswordHasher) VerifyPassword(password, encodedHash string) (bool, error) {
	// Parse the encoded hash
	params, salt, hash, err := h.decodeHash(encodedHash)
	if err != nil {
		return false, fmt.Errorf("failed to decode hash: %w", err)
	}

	// Hash the provided password with the same parameters
	otherHash := argon2.IDKey([]byte(password), salt, params.iterations, params.memory, params.parallelism, params.keyLength)

	// Compare the hashes using constant-time comparison
	return subtle.ConstantTimeCompare(hash, otherHash) == 1, nil
}

// generateRandomBytes generates random bytes of the specified length
func (h *PasswordHasher) generateRandomBytes(length uint32) ([]byte, error) {
	b := make([]byte, length)
	_, err := rand.Read(b)
	if err != nil {
		return nil, err
	}
	return b, nil
}

// hashParams holds the parameters used for hashing
type hashParams struct {
	memory      uint32
	iterations  uint32
	parallelism uint8
	saltLength  uint32
	keyLength   uint32
}

// decodeHash decodes an Argon2id hash string
func (h *PasswordHasher) decodeHash(encodedHash string) (*hashParams, []byte, []byte, error) {
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 {
		return nil, nil, nil, fmt.Errorf("invalid hash format")
	}

	if parts[1] != "argon2id" {
		return nil, nil, nil, fmt.Errorf("incompatible hash algorithm")
	}

	var version int
	_, err := fmt.Sscanf(parts[2], "v=%d", &version)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("invalid version: %w", err)
	}

	if version != argon2.Version {
		return nil, nil, nil, fmt.Errorf("incompatible version")
	}

	params := &hashParams{}
	_, err = fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &params.memory, &params.iterations, &params.parallelism)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("invalid parameters: %w", err)
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return nil, nil, nil, fmt.Errorf("invalid salt: %w", err)
	}
	params.saltLength = uint32(len(salt))

	hash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return nil, nil, nil, fmt.Errorf("invalid hash: %w", err)
	}
	params.keyLength = uint32(len(hash))

	return params, salt, hash, nil
}

// GetCostParams returns the cost parameters as a map for storing in the database
func (h *PasswordHasher) GetCostParams() map[string]interface{} {
	return map[string]interface{}{
		"memory":      h.memory,
		"iterations":  h.iterations,
		"parallelism": h.parallelism,
		"salt_length": h.saltLength,
		"key_length":  h.keyLength,
	}
}

// ValidatePasswordStrength validates password strength
func ValidatePasswordStrength(password string, minLength int) []string {
	var errors []string

	if len(password) < minLength {
		errors = append(errors, fmt.Sprintf("Password must be at least %d characters long", minLength))
	}

	if len(password) > 128 {
		errors = append(errors, "Password must be no more than 128 characters long")
	}

	hasUpper := false
	hasLower := false
	hasDigit := false
	hasSpecial := false

	for _, char := range password {
		switch {
		case char >= 'A' && char <= 'Z':
			hasUpper = true
		case char >= 'a' && char <= 'z':
			hasLower = true
		case char >= '0' && char <= '9':
			hasDigit = true
		case strings.ContainsRune("!@#$%^&*()_+-=[]{}|;:,.<>?", char):
			hasSpecial = true
		}
	}

	if !hasUpper {
		errors = append(errors, "Password must contain at least one uppercase letter")
	}

	if !hasLower {
		errors = append(errors, "Password must contain at least one lowercase letter")
	}

	if !hasDigit {
		errors = append(errors, "Password must contain at least one digit")
	}

	if !hasSpecial {
		errors = append(errors, "Password must contain at least one special character")
	}

	// Check for common weak patterns
	if strings.Contains(strings.ToLower(password), "password") {
		errors = append(errors, "Password cannot contain the word 'password'")
	}

	if strings.Contains(strings.ToLower(password), "123456") {
		errors = append(errors, "Password cannot contain common sequences like '123456'")
	}

	if strings.Contains(strings.ToLower(password), "qwerty") {
		errors = append(errors, "Password cannot contain common patterns like 'qwerty'")
	}

	return errors
}

// GenerateSecurePassword generates a secure random password
func GenerateSecurePassword(length int) (string, error) {
	if length < 8 {
		length = 12 // Default to 12 characters
	}

	const (
		upperChars   = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
		lowerChars   = "abcdefghijklmnopqrstuvwxyz"
		digitChars   = "0123456789"
		specialChars = "!@#$%^&*()-_=+[]{}|;:,.<>?"
		allChars     = upperChars + lowerChars + digitChars + specialChars
	)

	password := make([]byte, length)

	// Ensure at least one character from each category
	password[0] = upperChars[randInt(len(upperChars))]
	password[1] = lowerChars[randInt(len(lowerChars))]
	password[2] = digitChars[randInt(len(digitChars))]
	password[3] = specialChars[randInt(len(specialChars))]

	// Fill the rest randomly
	for i := 4; i < length; i++ {
		password[i] = allChars[randInt(len(allChars))]
	}

	// Shuffle the password
	for i := len(password) - 1; i > 0; i-- {
		j := randInt(i + 1)
		password[i], password[j] = password[j], password[i]
	}

	return string(password), nil
}

// randInt generates a random integer in the range [0, n)
func randInt(n int) int {
	if n <= 0 {
		return 0
	}

	// Generate random bytes
	bytes := make([]byte, 4)
	_, err := rand.Read(bytes)
	if err != nil {
		// Fallback to a simple method if crypto/rand fails
		return 0
	}

	// Convert to int and mod by n
	num := int(bytes[0])<<24 | int(bytes[1])<<16 | int(bytes[2])<<8 | int(bytes[3])
	if num < 0 {
		num = -num
	}

	return num % n
}

// GenerateRandomToken generates a random token for various purposes
func GenerateRandomToken(length int) (string, error) {
	if length <= 0 {
		length = 32
	}

	bytes := make([]byte, length)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", fmt.Errorf("failed to generate random token: %w", err)
	}

	return base64.URLEncoding.EncodeToString(bytes), nil
}

// HashPlainToken hashes a plain token for secure storage
func HashPlainToken(token string) string {
	// Use a simple approach for token hashing - in production you might want something more sophisticated
	hash := argon2.IDKey([]byte(token), []byte("macwrite-token-salt"), 1, 64*1024, 2, 32)
	return base64.RawStdEncoding.EncodeToString(hash)
}
