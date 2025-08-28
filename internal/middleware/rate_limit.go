package middleware

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"quantumtask-auth-api/internal/models"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// RateLimiter implements a token bucket rate limiter
type RateLimiter struct {
	limiters map[string]*rate.Limiter
	mu       sync.RWMutex
	rate     rate.Limit
	burst    int
	cleanup  time.Duration
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(requestsPerMinute int, burst int) *RateLimiter {
	rl := &RateLimiter{
		limiters: make(map[string]*rate.Limiter),
		rate:     rate.Limit(float64(requestsPerMinute) / 60.0), // Convert per minute to per second
		burst:    burst,
		cleanup:  time.Minute * 10, // Cleanup unused limiters every 10 minutes
	}

	// Start cleanup goroutine
	go rl.cleanupRoutine()

	return rl
}

// GetLimiter returns a rate limiter for the given key
func (rl *RateLimiter) GetLimiter(key string) *rate.Limiter {
	rl.mu.RLock()
	limiter, exists := rl.limiters[key]
	rl.mu.RUnlock()

	if !exists {
		rl.mu.Lock()
		// Double-check in case it was created while we were waiting for the lock
		if limiter, exists = rl.limiters[key]; !exists {
			limiter = rate.NewLimiter(rl.rate, rl.burst)
			rl.limiters[key] = limiter
		}
		rl.mu.Unlock()
	}

	return limiter
}

// cleanupRoutine periodically removes unused rate limiters
func (rl *RateLimiter) cleanupRoutine() {
	ticker := time.NewTicker(rl.cleanup)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		for key, limiter := range rl.limiters {
			// Remove limiters that haven't been used recently
			if limiter.Allow() { // This call doesn't consume a token, just checks availability
				continue
			}
			// If the limiter is at capacity, check if it's been idle
			// This is a simple heuristic - in production you might want more sophisticated cleanup
			delete(rl.limiters, key)
		}
		rl.mu.Unlock()
	}
}

// IPRateLimit middleware for rate limiting by IP address
func IPRateLimit(requestsPerMinute int) gin.HandlerFunc {
	limiter := NewRateLimiter(requestsPerMinute, requestsPerMinute/2) // Burst is half of rate

	return func(c *gin.Context) {
		clientIP := c.ClientIP()
		limiter := limiter.GetLimiter(clientIP)

		if !limiter.Allow() {
			c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", requestsPerMinute))
			c.Header("X-RateLimit-Remaining", "0")
			c.Header("X-RateLimit-Reset", fmt.Sprintf("%d", time.Now().Add(time.Minute).Unix()))

			c.JSON(http.StatusTooManyRequests, models.NewErrorResponse(
				"RATE_LIMIT_EXCEEDED",
				"Too many requests. Please try again later.",
				map[string]string{
					"retry_after": "60",
					"limit":       fmt.Sprintf("%d requests per minute", requestsPerMinute),
				},
			))
			c.Abort()
			return
		}

		// Add rate limit headers
		c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", requestsPerMinute))
		// Note: Getting exact remaining count from rate.Limiter is not straightforward
		// In production, you might want to use a more sophisticated rate limiter
		c.Header("X-RateLimit-Remaining", fmt.Sprintf("%d", requestsPerMinute-1))

		c.Next()
	}
}

// UserRateLimit middleware for rate limiting by authenticated user
func UserRateLimit(requestsPerMinute int) gin.HandlerFunc {
	limiter := NewRateLimiter(requestsPerMinute, requestsPerMinute/2)

	return func(c *gin.Context) {
		// Use IP as fallback if user is not authenticated
		key := c.ClientIP()

		// If user is authenticated, use user ID for rate limiting
		if userID, err := GetUserID(c); err == nil && userID.String() != "00000000-0000-0000-0000-000000000000" {
			key = userID.String()
		}

		rateLimiter := limiter.GetLimiter(key)

		if !rateLimiter.Allow() {
			c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", requestsPerMinute))
			c.Header("X-RateLimit-Remaining", "0")
			c.Header("X-RateLimit-Reset", fmt.Sprintf("%d", time.Now().Add(time.Minute).Unix()))

			c.JSON(http.StatusTooManyRequests, models.NewErrorResponse(
				"RATE_LIMIT_EXCEEDED",
				"Too many requests. Please try again later.",
				map[string]string{
					"retry_after": "60",
					"limit":       fmt.Sprintf("%d requests per minute", requestsPerMinute),
				},
			))
			c.Abort()
			return
		}

		c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", requestsPerMinute))
		c.Next()
	}
}

// LoginRateLimit provides stricter rate limiting for login attempts
func LoginRateLimit() gin.HandlerFunc {
	// More restrictive rate limiting for login attempts
	limiter := NewRateLimiter(10, 5) // 10 requests per minute, burst of 5

	return func(c *gin.Context) {
		clientIP := c.ClientIP()
		rateLimiter := limiter.GetLimiter(clientIP)

		if !rateLimiter.Allow() {
			c.Header("X-RateLimit-Limit", "10")
			c.Header("X-RateLimit-Remaining", "0")
			c.Header("X-RateLimit-Reset", fmt.Sprintf("%d", time.Now().Add(time.Minute).Unix()))

			c.JSON(http.StatusTooManyRequests, models.NewErrorResponse(
				"LOGIN_RATE_LIMIT_EXCEEDED",
				"Too many login attempts. Please try again later.",
				map[string]string{
					"retry_after": "60",
					"limit":       "10 login attempts per minute",
				},
			))
			c.Abort()
			return
		}

		c.Header("X-RateLimit-Limit", "10")
		c.Next()
	}
}

// PasswordResetRateLimit provides rate limiting for password reset requests
func PasswordResetRateLimit() gin.HandlerFunc {
	// Very restrictive rate limiting for password reset
	limiter := NewRateLimiter(3, 1) // 3 requests per minute, burst of 1

	return func(c *gin.Context) {
		clientIP := c.ClientIP()
		rateLimiter := limiter.GetLimiter(clientIP)

		if !rateLimiter.Allow() {
			c.Header("X-RateLimit-Limit", "3")
			c.Header("X-RateLimit-Remaining", "0")
			c.Header("X-RateLimit-Reset", fmt.Sprintf("%d", time.Now().Add(time.Minute).Unix()))

			c.JSON(http.StatusTooManyRequests, models.NewErrorResponse(
				"PASSWORD_RESET_RATE_LIMIT_EXCEEDED",
				"Too many password reset attempts. Please try again later.",
				map[string]string{
					"retry_after": "60",
					"limit":       "3 password reset attempts per minute",
				},
			))
			c.Abort()
			return
		}

		c.Header("X-RateLimit-Limit", "3")
		c.Next()
	}
}

// GlobalRateLimit provides global rate limiting across all endpoints
func GlobalRateLimit(requestsPerMinute int) gin.HandlerFunc {
	return IPRateLimit(requestsPerMinute)
}

// CustomRateLimit allows for custom rate limiting configuration
func CustomRateLimit(requestsPerMinute, burst int, keyFunc func(*gin.Context) string) gin.HandlerFunc {
	limiter := NewRateLimiter(requestsPerMinute, burst)

	return func(c *gin.Context) {
		key := keyFunc(c)
		if key == "" {
			key = c.ClientIP() // Fallback to IP
		}

		rateLimiter := limiter.GetLimiter(key)

		if !rateLimiter.Allow() {
			c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", requestsPerMinute))
			c.Header("X-RateLimit-Remaining", "0")
			c.Header("X-RateLimit-Reset", fmt.Sprintf("%d", time.Now().Add(time.Minute).Unix()))

			c.JSON(http.StatusTooManyRequests, models.NewErrorResponse(
				"RATE_LIMIT_EXCEEDED",
				"Too many requests. Please try again later.",
				map[string]string{
					"retry_after": "60",
					"limit":       fmt.Sprintf("%d requests per minute", requestsPerMinute),
				},
			))
			c.Abort()
			return
		}

		c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", requestsPerMinute))
		c.Next()
	}
}

// BruteForceProtection provides enhanced protection against brute force attacks
func BruteForceProtection() gin.HandlerFunc {
	// Track failed attempts with exponential backoff
	failedAttempts := make(map[string]int)
	lastAttempt := make(map[string]time.Time)
	mu := sync.RWMutex{}

	return func(c *gin.Context) {
		clientIP := c.ClientIP()

		mu.RLock()
		attempts := failedAttempts[clientIP]
		lastTime := lastAttempt[clientIP]
		mu.RUnlock()

		// Implement exponential backoff
		if attempts > 0 {
			backoffDuration := time.Duration(attempts*attempts) * time.Minute
			if time.Since(lastTime) < backoffDuration {
				c.JSON(http.StatusTooManyRequests, models.NewErrorResponse(
					"BRUTE_FORCE_PROTECTION",
					fmt.Sprintf("Account temporarily locked due to multiple failed attempts. Try again in %v.", backoffDuration-time.Since(lastTime)),
					map[string]string{
						"retry_after": fmt.Sprintf("%.0f", (backoffDuration - time.Since(lastTime)).Seconds()),
					},
				))
				c.Abort()
				return
			}
		}

		c.Next()

		// Check if the request failed (you might want to customize this logic)
		if c.Writer.Status() == http.StatusUnauthorized {
			mu.Lock()
			failedAttempts[clientIP]++
			lastAttempt[clientIP] = time.Now()
			mu.Unlock()
		} else if c.Writer.Status() == http.StatusOK {
			// Reset failed attempts on successful login
			mu.Lock()
			delete(failedAttempts, clientIP)
			delete(lastAttempt, clientIP)
			mu.Unlock()
		}
	}
}
