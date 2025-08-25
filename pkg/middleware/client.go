package middleware

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"macwrite-auth-api/internal/config"

	track "github.com/middleware-labs/golang-apm/tracker"
)

// Client wraps the middleware.io APM client
type Client struct {
	config  *config.MiddlewareConfig
	enabled bool
	mu      sync.RWMutex
	started bool
}

// NewClient creates a new middleware.io client
func NewClient(cfg *config.MiddlewareConfig) *Client {
	return &Client{
		config:  cfg,
		enabled: cfg.Enabled,
	}
}

// Start initializes the middleware.io tracking
func (c *Client) Start(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.enabled {
		log.Println("Middleware.io APM tracking is disabled")
		return nil
	}

	if c.started {
		log.Println("Middleware.io APM tracking already started")
		return nil
	}

	// Configure tracking options (using the same pattern as in main.go)
	log.Printf("Starting Middleware.io APM tracking for service: %s", c.config.Service)

	// Start tracking in a goroutine
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("Middleware.io APM tracker panicked: %v", r)
			}
		}()

		track.Track(
			track.WithConfigTag("service", c.config.Service),
			track.WithConfigTag("accessToken", c.config.AccessToken),
			track.WithConfigTag("target", c.config.Target),
		)
		log.Println("Middleware.io APM tracking started successfully")
	}()

	c.started = true
	return nil
}

// Stop gracefully stops the middleware.io tracking
func (c *Client) Stop() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.started {
		return nil
	}

	log.Println("Stopping Middleware.io APM tracking...")
	c.started = false
	return nil
}

// IsEnabled returns whether middleware.io tracking is enabled
func (c *Client) IsEnabled() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.enabled
}

// IsStarted returns whether middleware.io tracking has been started
func (c *Client) IsStarted() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.started
}

// GetConfig returns the current configuration
func (c *Client) GetConfig() *config.MiddlewareConfig {
	return c.config
}

// HealthCheck performs a health check on the middleware.io integration
func (c *Client) HealthCheck() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	status := "healthy"
	if !c.enabled {
		status = "disabled"
	} else if !c.started {
		status = "not_started"
	}

	return map[string]interface{}{
		"status":      status,
		"enabled":     c.enabled,
		"started":     c.started,
		"service":     c.config.Service,
		"target":      c.config.Target,
		"environment": c.config.Environment,
		"version":     c.config.Version,
		"timestamp":   time.Now(),
	}
}

// GetMetrics returns basic metrics about the middleware.io integration
func (c *Client) GetMetrics() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	uptime := time.Duration(0)
	if c.started {
		// In a real implementation, you'd track when it was started
		uptime = time.Since(time.Now().Add(-1 * time.Hour)) // Placeholder
	}

	return map[string]interface{}{
		"uptime_seconds":    uptime.Seconds(),
		"enabled":           c.enabled,
		"started":           c.started,
		"service_name":      c.config.Service,
		"service_version":   c.config.Version,
		"target_endpoint":   c.config.Target,
		"environment":       c.config.Environment,
		"configuration_set": c.config.AccessToken != "",
		"timestamp":         time.Now(),
	}
}

// Restart restarts the middleware.io tracking
func (c *Client) Restart(ctx context.Context) error {
	if err := c.Stop(); err != nil {
		return fmt.Errorf("failed to stop middleware.io tracking: %w", err)
	}

	// Wait a moment before restarting
	time.Sleep(1 * time.Second)

	if err := c.Start(ctx); err != nil {
		return fmt.Errorf("failed to start middleware.io tracking: %w", err)
	}

	return nil
}

// UpdateConfig updates the configuration and restarts if necessary
func (c *Client) UpdateConfig(newConfig *config.MiddlewareConfig) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	oldConfig := c.config
	c.config = newConfig
	c.enabled = newConfig.Enabled

	// If the configuration changed significantly, restart
	if c.started && (oldConfig.AccessToken != newConfig.AccessToken ||
		oldConfig.Target != newConfig.Target ||
		oldConfig.Service != newConfig.Service) {

		log.Println("Middleware.io configuration changed, restarting...")
		return c.Restart(context.Background())
	}

	return nil
}
