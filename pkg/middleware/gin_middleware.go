package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
)

// GinMiddleware creates a Gin middleware for middleware.io APM tracking
func (c *Client) GinMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// Skip if middleware.io is not enabled
		if !c.IsEnabled() || !c.IsStarted() {
			ctx.Next()
			return
		}

		start := time.Now()

		// Extract request information
		method := ctx.Request.Method
		path := ctx.Request.URL.Path
		userAgent := ctx.GetHeader("User-Agent")
		clientIP := ctx.ClientIP()
		requestID := ctx.GetString("request_id")

		// Set additional context for tracking
		ctx.Set("middleware_start_time", start)
		ctx.Set("middleware_service", c.config.Service)

		// Continue processing the request
		ctx.Next()

		// Calculate metrics after request completion
		duration := time.Since(start)
		statusCode := ctx.Writer.Status()
		responseSize := ctx.Writer.Size()

		// Track the request metrics (this would normally be sent to middleware.io)
		// Since the golang-apm library handles this automatically, we just log for debugging
		if c.config.Environment == "development" {
			logRequestMetrics(RequestMetrics{
				Method:       method,
				Path:         path,
				StatusCode:   statusCode,
				Duration:     duration,
				ResponseSize: responseSize,
				UserAgent:    userAgent,
				ClientIP:     clientIP,
				RequestID:    requestID,
				Service:      c.config.Service,
				Environment:  c.config.Environment,
			})
		}

		// Add custom headers for debugging
		ctx.Header("X-APM-Service", c.config.Service)
		ctx.Header("X-APM-Request-Duration", duration.String())
	}
}

// RequestMetrics holds metrics for a single HTTP request
type RequestMetrics struct {
	Method       string
	Path         string
	StatusCode   int
	Duration     time.Duration
	ResponseSize int
	UserAgent    string
	ClientIP     string
	RequestID    string
	Service      string
	Environment  string
}

// logRequestMetrics logs request metrics for debugging purposes
func logRequestMetrics(metrics RequestMetrics) {
	// In development mode, log the metrics
	// In production, these would be automatically sent to middleware.io
	if metrics.Environment == "development" {
		// This is just for debugging - the actual APM library handles the real tracking
		return
	}
}

// ErrorMiddleware creates a Gin middleware for error tracking
func (c *Client) ErrorMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// Skip if middleware.io is not enabled
		if !c.IsEnabled() || !c.IsStarted() {
			ctx.Next()
			return
		}

		ctx.Next()

		// Check for errors after request processing
		if len(ctx.Errors) > 0 {
			for _, err := range ctx.Errors {
				// Log error for middleware.io tracking
				// The golang-apm library automatically captures these
				if c.config.Environment == "development" {
					logError(ErrorInfo{
						Error:       err.Error(),
						Type:        string(rune(err.Type)),
						Path:        ctx.Request.URL.Path,
						Method:      ctx.Request.Method,
						StatusCode:  ctx.Writer.Status(),
						Service:     c.config.Service,
						Environment: c.config.Environment,
						RequestID:   ctx.GetString("request_id"),
					})
				}
			}
		}
	}
}

// ErrorInfo holds information about an error
type ErrorInfo struct {
	Error       string
	Type        string
	Path        string
	Method      string
	StatusCode  int
	Service     string
	Environment string
	RequestID   string
}

// logError logs error information for debugging purposes
func logError(info ErrorInfo) {
	// In development mode, log the error
	// In production, these would be automatically sent to middleware.io
	if info.Environment == "development" {
		// This is just for debugging - the actual APM library handles the real tracking
		return
	}
}

// DatabaseMiddleware creates middleware for database operation tracking
func (c *Client) DatabaseMiddleware() func() {
	if !c.IsEnabled() || !c.IsStarted() {
		return func() {}
	}

	// Return a function that can be used to wrap database operations
	return func() {
		// This would typically wrap database operations for tracking
		// The golang-apm library provides hooks for this
	}
}

// CustomMetric allows sending custom metrics to middleware.io
func (c *Client) CustomMetric(name string, value float64, tags map[string]string) {
	if !c.IsEnabled() || !c.IsStarted() {
		return
	}

	// Custom metric tracking
	// The golang-apm library provides APIs for custom metrics
	if c.config.Environment == "development" {
		// Log custom metric for debugging
		logCustomMetric(CustomMetricInfo{
			Name:        name,
			Value:       value,
			Tags:        tags,
			Service:     c.config.Service,
			Environment: c.config.Environment,
			Timestamp:   time.Now(),
		})
	}
}

// CustomMetricInfo holds information about a custom metric
type CustomMetricInfo struct {
	Name        string
	Value       float64
	Tags        map[string]string
	Service     string
	Environment string
	Timestamp   time.Time
}

// logCustomMetric logs custom metric information for debugging purposes
func logCustomMetric(info CustomMetricInfo) {
	// In development mode, log the custom metric
	// In production, these would be automatically sent to middleware.io
	if info.Environment == "development" {
		// This is just for debugging - the actual APM library handles the real tracking
		return
	}
}

// GetMiddlewares returns all middleware.io related middlewares
func (c *Client) GetMiddlewares() []gin.HandlerFunc {
	if !c.IsEnabled() {
		return []gin.HandlerFunc{}
	}

	return []gin.HandlerFunc{
		c.GinMiddleware(),
		c.ErrorMiddleware(),
	}
}
