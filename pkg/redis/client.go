package redis

import (
	"context"
	"errors"
	"sync"
	"time"
)

// Client defines the interface for Redis operations
type Client interface {
	Set(ctx context.Context, key string, value []byte, expiration time.Duration) error
	Get(ctx context.Context, key string) ([]byte, error)
	Delete(ctx context.Context, key string) error
	DeletePattern(ctx context.Context, pattern string) error
	Ping(ctx context.Context) error
	Close() error
}

// ErrKeyNotFound is returned when a key is not found
var ErrKeyNotFound = errors.New("key not found")

// cacheItem represents a cached item with expiration
type cacheItem struct {
	data      []byte
	expiresAt time.Time
}

// mockRedisClient implements the Client interface for development/testing
type mockRedisClient struct {
	cache map[string]cacheItem
	mutex sync.RWMutex
}

// NewClient creates a new Redis client (mock implementation for development)
func NewClient(addr, password string, db int) Client {
	return &mockRedisClient{
		cache: make(map[string]cacheItem),
	}
}

// Set stores a key-value pair with expiration
func (r *mockRedisClient) Set(ctx context.Context, key string, value []byte, expiration time.Duration) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	var expiresAt time.Time
	if expiration > 0 {
		expiresAt = time.Now().Add(expiration)
	}

	r.cache[key] = cacheItem{
		data:      value,
		expiresAt: expiresAt,
	}

	return nil
}

// Get retrieves a value by key
func (r *mockRedisClient) Get(ctx context.Context, key string) ([]byte, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	item, exists := r.cache[key]
	if !exists {
		return nil, ErrKeyNotFound
	}

	// Check if expired
	if !item.expiresAt.IsZero() && time.Now().After(item.expiresAt) {
		delete(r.cache, key)
		return nil, ErrKeyNotFound
	}

	return item.data, nil
}

// Delete removes a key
func (r *mockRedisClient) Delete(ctx context.Context, key string) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	delete(r.cache, key)
	return nil
}

// DeletePattern deletes all keys matching a pattern (simple implementation)
func (r *mockRedisClient) DeletePattern(ctx context.Context, pattern string) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Simple pattern matching for basic wildcards
	for key := range r.cache {
		if matchPattern(pattern, key) {
			delete(r.cache, key)
		}
	}

	return nil
}

// Ping checks if Redis is available
func (r *mockRedisClient) Ping(ctx context.Context) error {
	return nil // Always available for mock
}

// Close closes the Redis connection
func (r *mockRedisClient) Close() error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.cache = make(map[string]cacheItem)
	return nil
}

// matchPattern performs simple pattern matching
func matchPattern(pattern, key string) bool {
	if pattern == "*" {
		return true
	}

	if len(pattern) == 0 {
		return len(key) == 0
	}

	if pattern[len(pattern)-1] == '*' {
		prefix := pattern[:len(pattern)-1]
		return len(key) >= len(prefix) && key[:len(prefix)] == prefix
	}

	return pattern == key
}
