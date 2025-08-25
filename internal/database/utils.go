package database

import (
	"context"
	"crypto/sha256"
	"fmt"
	"time"
)

// timeoutContext creates a context with timeout
func timeoutContext(timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), timeout)
}

// HashToken creates a SHA-256 hash of a token for database storage
func HashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return fmt.Sprintf("%x", hash)
}

// NullableString converts a string pointer to a nullable string for SQL queries
func NullableString(s *string) interface{} {
	if s == nil {
		return nil
	}
	return *s
}

// NullableTime converts a time pointer to a nullable time for SQL queries
func NullableTime(t *time.Time) interface{} {
	if t == nil {
		return nil
	}
	return *t
}

// StringPtr returns a pointer to the provided string
func StringPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// TimePtr returns a pointer to the provided time
func TimePtr(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}

// BoolPtr returns a pointer to the provided bool
func BoolPtr(b bool) *bool {
	return &b
}

// IntPtr returns a pointer to the provided int
func IntPtr(i int) *int {
	return &i
}

// DefaultString returns the value if not nil, otherwise returns the default
func DefaultString(value *string, defaultValue string) string {
	if value == nil {
		return defaultValue
	}
	return *value
}

// DefaultInt returns the value if not nil, otherwise returns the default
func DefaultInt(value *int, defaultValue int) int {
	if value == nil {
		return defaultValue
	}
	return *value
}

// DefaultBool returns the value if not nil, otherwise returns the default
func DefaultBool(value *bool, defaultValue bool) bool {
	if value == nil {
		return defaultValue
	}
	return *value
}

// BuildLimitOffset builds LIMIT and OFFSET clause for pagination
func BuildLimitOffset(page, pageSize int) (limit, offset int) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100 // Maximum page size
	}

	limit = pageSize
	offset = (page - 1) * pageSize
	return
}

// IsUniqueViolation checks if the error is a unique constraint violation
func IsUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	// PostgreSQL unique violation error code is 23505
	return fmt.Sprintf("%v", err) == "pq: duplicate key value violates unique constraint"
}

// IsForeignKeyViolation checks if the error is a foreign key constraint violation
func IsForeignKeyViolation(err error) bool {
	if err == nil {
		return false
	}
	// PostgreSQL foreign key violation error code is 23503
	return fmt.Sprintf("%v", err) == "pq: insert or update on table violates foreign key constraint"
}
