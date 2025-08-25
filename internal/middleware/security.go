package middleware

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"macwrite-auth-api/internal/config"
	"macwrite-auth-api/internal/models"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// SecurityHeaders adds common security headers
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Prevent MIME type sniffing
		c.Header("X-Content-Type-Options", "nosniff")

		// Prevent clickjacking
		c.Header("X-Frame-Options", "DENY")

		// Enable XSS protection
		c.Header("X-XSS-Protection", "1; mode=block")

		// Referrer policy
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")

		// Content Security Policy
		c.Header("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data: https:; font-src 'self'; connect-src 'self'; frame-ancestors 'none';")

		// Permissions policy
		c.Header("Permissions-Policy", "camera=(), microphone=(), geolocation=()")

		// HSTS (only for HTTPS)
		if c.Request.TLS != nil {
			c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")
		}

		c.Next()
	}
}

// CORS configures Cross-Origin Resource Sharing
func CORS(cfg *config.Config) gin.HandlerFunc {
	config := cors.Config{
		AllowOrigins:     cfg.Server.CORSOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Length", "Content-Type", "Authorization", "X-Requested-With", "Accept", "Accept-Encoding", "Accept-Language", "Cache-Control"},
		ExposeHeaders:    []string{"Content-Length", "X-RateLimit-Limit", "X-RateLimit-Remaining", "X-RateLimit-Reset"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}

	// In development, allow all origins
	if cfg.IsDevelopment() {
		config.AllowAllOrigins = true
		config.AllowOrigins = nil
	}

	return cors.New(config)
}

// RequestID adds a unique request ID to each request
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := uuid.New().String()
		c.Header("X-Request-ID", requestID)
		c.Set("request_id", requestID)
		c.Next()
	}
}

// RequestLogger logs incoming requests
func RequestLogger() gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		return fmt.Sprintf("[%s] %s %s %d %s %s %s\n",
			param.TimeStamp.Format("2006/01/02 - 15:04:05"),
			param.Method,
			param.Path,
			param.StatusCode,
			param.Latency,
			param.ClientIP,
			param.ErrorMessage,
		)
	})
}

// Recovery middleware with custom error handling
func Recovery() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		if err, ok := recovered.(string); ok {
			c.JSON(http.StatusInternalServerError, models.NewErrorResponse(
				"INTERNAL_SERVER_ERROR",
				"An unexpected error occurred",
				map[string]string{
					"error": err,
				},
			))
		} else {
			c.JSON(http.StatusInternalServerError, models.NewErrorResponse(
				"INTERNAL_SERVER_ERROR",
				"An unexpected error occurred",
				nil,
			))
		}
		c.Abort()
	})
}

// DeviceFingerprint extracts device information for security tracking
func DeviceFingerprint() gin.HandlerFunc {
	return func(c *gin.Context) {
		deviceInfo := models.JSONB{
			"user_agent": c.GetHeader("User-Agent"),
			"ip_address": c.ClientIP(),
			"accept":     c.GetHeader("Accept"),
			"language":   c.GetHeader("Accept-Language"),
			"encoding":   c.GetHeader("Accept-Encoding"),
		}

		// Additional headers that might be useful for fingerprinting
		if dnt := c.GetHeader("DNT"); dnt != "" {
			deviceInfo["dnt"] = dnt
		}

		if timezone := c.GetHeader("X-Timezone"); timezone != "" {
			deviceInfo["timezone"] = timezone
		}

		c.Set("device_info", deviceInfo)
		c.Next()
	}
}

// ValidateContentType ensures requests have the correct content type
func ValidateContentType(allowedTypes ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Only validate for non-GET requests
		if c.Request.Method == "GET" || c.Request.Method == "DELETE" {
			c.Next()
			return
		}

		contentType := c.GetHeader("Content-Type")
		if contentType == "" {
			c.JSON(http.StatusBadRequest, models.NewErrorResponse(
				"MISSING_CONTENT_TYPE",
				"Content-Type header is required",
				nil,
			))
			c.Abort()
			return
		}

		// Check if content type is allowed
		for _, allowedType := range allowedTypes {
			if strings.Contains(contentType, allowedType) {
				c.Next()
				return
			}
		}

		c.JSON(http.StatusUnsupportedMediaType, models.NewErrorResponse(
			"UNSUPPORTED_MEDIA_TYPE",
			"Content-Type not supported",
			map[string]string{
				"received": contentType,
				"allowed":  strings.Join(allowedTypes, ", "),
			},
		))
		c.Abort()
	}
}

// MethodOverride allows method override via header or form field
func MethodOverride() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method != "POST" {
			c.Next()
			return
		}

		// Check for method override in header
		if method := c.GetHeader("X-HTTP-Method-Override"); method != "" {
			c.Request.Method = method
		} else if method := c.PostForm("_method"); method != "" {
			// Check for method override in form field
			c.Request.Method = method
		}

		c.Next()
	}
}

// MaxRequestSize limits the size of request bodies
func MaxRequestSize(maxSize int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.ContentLength > maxSize {
			c.JSON(http.StatusRequestEntityTooLarge, models.NewErrorResponse(
				"REQUEST_TOO_LARGE",
				"Request body too large",
				map[string]string{
					"max_size": fmt.Sprintf("%d bytes", maxSize),
					"received": fmt.Sprintf("%d bytes", c.Request.ContentLength),
				},
			))
			c.Abort()
			return
		}

		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxSize)
		c.Next()
	}
}

// NoCache adds cache control headers to prevent caching
func NoCache() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
		c.Header("Pragma", "no-cache")
		c.Header("Expires", "0")
		c.Next()
	}
}

// SecureAPIResponse ensures API responses have security headers
func SecureAPIResponse() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// Add security headers for API responses
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")

		// Ensure JSON responses have correct content type
		if c.Writer.Header().Get("Content-Type") == "" &&
			c.Writer.Status() >= 200 && c.Writer.Status() < 300 {
			c.Header("Content-Type", "application/json; charset=utf-8")
		}
	}
}

// IPWhitelist restricts access to specific IP addresses
func IPWhitelist(allowedIPs []string) gin.HandlerFunc {
	allowedIPMap := make(map[string]bool)
	for _, ip := range allowedIPs {
		allowedIPMap[ip] = true
	}

	return func(c *gin.Context) {
		clientIP := c.ClientIP()

		if !allowedIPMap[clientIP] {
			c.JSON(http.StatusForbidden, models.NewErrorResponse(
				"IP_NOT_ALLOWED",
				"Access denied from this IP address",
				map[string]string{
					"ip": clientIP,
				},
			))
			c.Abort()
			return
		}

		c.Next()
	}
}

// UserAgent validates and tracks user agents
func UserAgent() gin.HandlerFunc {
	return func(c *gin.Context) {
		userAgent := c.GetHeader("User-Agent")
		if userAgent == "" {
			c.JSON(http.StatusBadRequest, models.NewErrorResponse(
				"MISSING_USER_AGENT",
				"User-Agent header is required",
				nil,
			))
			c.Abort()
			return
		}

		// Block known bad user agents (bots, scrapers, etc.)
		blockedAgents := []string{
			"curl",
			"wget",
			"python-requests",
			"postman",
		}

		userAgentLower := strings.ToLower(userAgent)
		for _, blocked := range blockedAgents {
			if strings.Contains(userAgentLower, blocked) {
				c.JSON(http.StatusForbidden, models.NewErrorResponse(
					"USER_AGENT_BLOCKED",
					"This user agent is not allowed",
					map[string]string{
						"user_agent": userAgent,
					},
				))
				c.Abort()
				return
			}
		}

		c.Set("user_agent", userAgent)
		c.Next()
	}
}
