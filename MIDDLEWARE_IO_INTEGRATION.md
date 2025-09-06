# Middleware.io APM Integration

This document describes the middleware.io APM (Application Performance Monitoring) integration implemented in the MacWrite backend application.

## Overview

The integration provides comprehensive application performance monitoring and tracking using middleware.io's APM service. It automatically tracks HTTP requests, errors, and custom metrics.

## Features

- ✅ **Automatic HTTP Request Tracking**: All HTTP requests are automatically tracked with metrics like response time, status code, and request details
- ✅ **Error Tracking**: Automatic capture and reporting of application errors  
- ✅ **Custom Metrics**: Support for sending custom application metrics
- ✅ **Health Monitoring**: Built-in health check endpoints for monitoring the APM integration
- ✅ **Graceful Shutdown**: Proper cleanup when the application shuts down
- ✅ **Environment-based Configuration**: Different configurations for development and production

## Configuration

### Environment Variables

The following environment variables can be used to configure the middleware.io integration:

```bash
# Enable/disable middleware.io tracking (default: true)
MIDDLEWARE_ENABLED=true

# Your middleware.io access token (required)
MIDDLEWARE_ACCESS_TOKEN=your_access_token_here

# Middleware.io target endpoint (default: dopem.middleware.io:443)
MIDDLEWARE_TARGET=dopem.middleware.io:443

# Service name for identification (default: mac)
MIDDLEWARE_SERVICE=mac

# Service version (defaults to app version)
MIDDLEWARE_VERSION=1.0.0

# Environment (defaults to app environment)
MIDDLEWARE_ENVIRONMENT=production
```

### Configuration in Code

The middleware.io configuration is automatically loaded from environment variables through the `config.MiddlewareConfig` struct:

```go
type MiddlewareConfig struct {
    Enabled     bool
    AccessToken string
    Target      string
    Service     string
    Version     string
    Environment string
}
```

## Usage

### Automatic Integration

The middleware.io APM tracking is automatically initialized when the application starts:

1. **Configuration Loading**: Environment variables are loaded into the config
2. **Client Initialization**: The middleware.io client is created and started
3. **Middleware Registration**: HTTP tracking middleware is registered with Gin
4. **Background Tracking**: The APM tracking runs in the background

### Manual Integration

If you need to manually work with the middleware.io client:

```go
import apmclient "quantumtask-auth-api/pkg/middleware"

// Create a new client
client := apmclient.NewClient(&cfg.Middleware)

// Start tracking
err := client.Start(context.Background())
if err != nil {
    log.Printf("Failed to start APM tracking: %v", err)
}

// Check if enabled/started
if client.IsEnabled() && client.IsStarted() {
    // APM is running
}

// Send custom metrics
client.CustomMetric("user_login", 1.0, map[string]string{
    "method": "email",
    "success": "true",
})

// Graceful shutdown
err = client.Stop()
```

### Custom Metrics

You can send custom metrics to middleware.io:

```go
// In your handler or service
func (h *Handler) SomeBusinessLogic(c *gin.Context) {
    // Your business logic here...
    
    // Track a custom metric
    middlewareClient.CustomMetric("business_operation", 1.0, map[string]string{
        "operation": "data_export",
        "user_id": userID,
        "success": "true",
    })
}
```

## Monitoring Endpoints

The integration provides several monitoring endpoints:

### APM Health Check
```bash
GET /api/v1/apm-monitoring/health
```

Response:
```json
{
    "status": "healthy",
    "enabled": true,
    "started": true,
    "service": "mac",
    "target": "dopem.middleware.io:443",
    "environment": "production",
    "version": "1.0.0",
    "timestamp": "2024-01-15T10:30:45Z"
}
```

### APM Metrics
```bash
GET /api/v1/apm-monitoring/metrics
```

Response:
```json
{
    "metrics": {
        "uptime_seconds": 3600,
        "enabled": true,
        "started": true,
        "service_name": "mac",
        "service_version": "1.0.0",
        "target_endpoint": "dopem.middleware.io:443",
        "environment": "production",
        "configuration_set": true,
        "timestamp": "2024-01-15T10:30:45Z"
    },
    "timestamp": "2024-01-15T10:30:45Z"
}
```

### APM Restart
```bash
POST /api/v1/apm-monitoring/restart
```

Response:
```json
{
    "message": "Middleware.io APM tracking restarted successfully",
    "timestamp": "2024-01-15T10:30:45Z"
}
```

## What Gets Tracked

### HTTP Requests
- **Request Method**: GET, POST, PUT, DELETE, etc.
- **Request Path**: The endpoint being accessed
- **Response Status**: HTTP status codes (200, 404, 500, etc.)
- **Response Time**: How long the request took to process
- **Response Size**: Size of the response body
- **User Agent**: Browser/client information
- **Client IP**: IP address of the requesting client
- **Request ID**: Unique identifier for request tracing

### Errors
- **Error Messages**: Full error descriptions
- **Error Types**: Classification of error types
- **Request Context**: What request caused the error
- **Stack Traces**: Detailed error location information
- **Service Context**: Which service/component had the error

### Custom Metrics
- **Business Metrics**: Login attempts, data exports, feature usage
- **Performance Metrics**: Custom timing measurements
- **Security Metrics**: Failed authentication attempts, rate limiting triggers

## Development vs Production

### Development Mode
- **Verbose Logging**: Additional debug information is logged
- **Local Debugging**: Metrics are logged locally for debugging
- **Relaxed Configuration**: Some validations are relaxed

### Production Mode
- **Optimized Performance**: Minimal overhead on application performance
- **Secure Transmission**: All data is securely transmitted to middleware.io
- **Error Resilience**: APM failures don't affect application functionality

## Troubleshooting

### Common Issues

1. **APM Not Starting**
   - Check that `MIDDLEWARE_ENABLED=true`
   - Verify `MIDDLEWARE_ACCESS_TOKEN` is set correctly
   - Ensure network connectivity to `dopem.middleware.io:443`

2. **No Data in Dashboard**
   - Verify the access token is valid
   - Check that the service name matches your middleware.io configuration
   - Ensure requests are actually being made to the application

3. **Performance Impact**
   - The APM integration is designed to have minimal performance impact
   - If you notice issues, you can disable it temporarily with `MIDDLEWARE_ENABLED=false`

### Debug Commands

Check APM status:
```bash
curl http://localhost:8080/api/v1/apm-monitoring/health
```

View APM metrics:
```bash
curl http://localhost:8080/api/v1/apm-monitoring/metrics
```

Restart APM:
```bash
curl -X POST http://localhost:8080/api/v1/apm-monitoring/restart
```

### Logs

Look for these log messages to understand the APM status:

```
🔧 Initializing Middleware.io APM tracking...
✅ Middleware.io APM tracking initialized successfully
✅ Middleware.io APM tracking middleware applied
```

## Security Considerations

- **Access Token**: Keep your middleware.io access token secure and never commit it to version control
- **Network Traffic**: All APM data is transmitted over secure connections
- **Data Privacy**: Only performance and error data is sent, not sensitive business data
- **Rate Limiting**: The integration respects rate limits to avoid overwhelming the APM service

## Performance Impact

The middleware.io integration is designed to have minimal performance impact:

- **Asynchronous Processing**: Most APM operations happen in background goroutines
- **Efficient Middleware**: HTTP tracking middleware is optimized for speed
- **Conditional Processing**: APM overhead is only applied when enabled
- **Graceful Degradation**: Application continues to work even if APM fails

## Support

For issues related to:
- **Integration Code**: Check the implementation in `pkg/middleware/`
- **Configuration**: Review environment variables and `internal/config/config.go`
- **Middleware.io Service**: Contact middleware.io support
- **Performance Issues**: Use the monitoring endpoints to diagnose problems

## Files Structure

```
backend-go/
├── internal/config/config.go                 # Configuration including MiddlewareConfig
├── pkg/middleware/
│   ├── client.go                            # Main APM client implementation
│   └── gin_middleware.go                    # Gin HTTP middleware for request tracking
├── cmd/server/main.go                       # Integration and initialization
└── MIDDLEWARE_IO_INTEGRATION.md             # This documentation
```

This integration provides comprehensive APM capabilities while maintaining the performance and reliability of your MacWrite backend application.
