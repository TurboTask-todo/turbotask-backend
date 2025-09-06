package database

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"quantumtask-auth-api/internal/config"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jmoiron/sqlx"
)

// HighPerformanceDB represents a high-performance database connection pool
type HighPerformanceDB struct {
	primary        *pgxpool.Pool
	readReplicas   []*ReplicaPool
	config         *config.Config
	metrics        *poolMetricsInternal
	circuitBreaker *CircuitBreaker

	// Connection state tracking
	isHealthy       int64 // 1 for healthy, 0 for unhealthy
	lastHealthCheck time.Time
	mu              sync.RWMutex
}

// ReplicaPool represents a read replica connection pool
type ReplicaPool struct {
	pool      *pgxpool.Pool
	config    config.DatabaseReplicaConfig
	weight    int
	priority  int
	isHealthy int64 // 1 for healthy, 0 for unhealthy
}

// PoolMetrics holds connection pool metrics (safe to copy)
type PoolMetrics struct {
	TotalConnections    int64
	ActiveConnections   int64
	IdleConnections     int64
	WaitingConnections  int64
	TotalQueries        int64
	SuccessfulQueries   int64
	FailedQueries       int64
	AverageQueryTime    float64
	ConnectionErrors    int64
	CircuitBreakerTrips int64
	LastUpdated         time.Time
}

// poolMetricsInternal holds the actual metrics with synchronization
type poolMetricsInternal struct {
	PoolMetrics
	mu sync.RWMutex
}

// CircuitBreaker implements circuit breaker pattern for database connections
type CircuitBreaker struct {
	failureThreshold int
	resetTimeout     time.Duration
	failures         int64
	lastFailureTime  time.Time
	state            int32 // 0: Closed, 1: Open, 2: Half-Open
	mu               sync.RWMutex
}

// NewHighPerformanceDB creates a new high-performance database connection pool
func NewHighPerformanceDB(cfg *config.Config) (*HighPerformanceDB, error) {
	db := &HighPerformanceDB{
		config: cfg,
		metrics: &poolMetricsInternal{
			PoolMetrics: PoolMetrics{LastUpdated: time.Now()},
		},
		circuitBreaker:  NewCircuitBreaker(10, 30*time.Second), // 10 failures, 30s reset
		isHealthy:       1,
		lastHealthCheck: time.Now(),
	}

	// Create primary connection pool
	primaryPool, err := db.createPrimaryPool()
	if err != nil {
		return nil, fmt.Errorf("failed to create primary pool: %w", err)
	}
	db.primary = primaryPool

	// Create read replica pools
	if len(cfg.Database.ReadReplicas) > 0 {
		db.readReplicas = make([]*ReplicaPool, 0, len(cfg.Database.ReadReplicas))
		for _, replicaConfig := range cfg.Database.ReadReplicas {
			replicaPool, err := db.createReplicaPool(replicaConfig)
			if err != nil {
				log.Printf("Warning: Failed to create replica pool for %s:%d: %v",
					replicaConfig.Host, replicaConfig.Port, err)
				continue
			}
			db.readReplicas = append(db.readReplicas, replicaPool)
		}
	}

	// Start health check goroutine
	if cfg.Database.Pool.EnableMetrics {
		go db.startMetricsCollection()
	}
	go db.startHealthChecks()

	return db, nil
}

// createPrimaryPool creates the primary database connection pool
func (db *HighPerformanceDB) createPrimaryPool() (*pgxpool.Pool, error) {
	cfg := db.config.Database

	// Build connection string
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Name, cfg.SSLMode)

	// Configure pgxpool
	poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database URL: %w", err)
	}

	// Apply high-performance settings
	poolConfig.MaxConns = cfg.Pool.MaxConns
	poolConfig.MinConns = cfg.Pool.MinConns
	poolConfig.MaxConnLifetime = cfg.Pool.MaxConnLifetime
	poolConfig.MaxConnIdleTime = cfg.Pool.MaxConnIdleTime
	poolConfig.HealthCheckPeriod = cfg.Pool.HealthCheckPeriod

	// Configure connection behavior
	poolConfig.ConnConfig.ConnectTimeout = cfg.Pool.ConnectTimeout

	// Enable prepared statement caching for performance
	if cfg.Pool.PreparedStatementCache {
		poolConfig.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeCacheStatement
		// Note: pgx v5 automatically manages prepared statement cache
	}

	// Set connection pool callbacks for monitoring
	poolConfig.BeforeAcquire = func(ctx context.Context, conn *pgx.Conn) bool {
		atomic.AddInt64(&db.metrics.ActiveConnections, 1)
		return true
	}

	poolConfig.AfterRelease = func(conn *pgx.Conn) bool {
		atomic.AddInt64(&db.metrics.ActiveConnections, -1)
		return true
	}

	poolConfig.BeforeClose = func(conn *pgx.Conn) {
		atomic.AddInt64(&db.metrics.TotalConnections, -1)
	}

	// Create the pool
	pool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Test the connection
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping primary database: %w", err)
	}

	return pool, nil
}

// createReplicaPool creates a read replica connection pool
func (db *HighPerformanceDB) createReplicaPool(replicaConfig config.DatabaseReplicaConfig) (*ReplicaPool, error) {
	cfg := db.config.Database

	// Build connection string for replica
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		cfg.User, cfg.Password, replicaConfig.Host, replicaConfig.Port, cfg.Name, cfg.SSLMode)

	// Configure pgxpool with reduced connection count for replicas
	poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to parse replica database URL: %w", err)
	}

	// Replica pools typically have fewer connections
	poolConfig.MaxConns = cfg.Pool.MaxConns / 2 // Half the connections of primary
	poolConfig.MinConns = cfg.Pool.MinConns / 2
	poolConfig.MaxConnLifetime = cfg.Pool.MaxConnLifetime
	poolConfig.MaxConnIdleTime = cfg.Pool.MaxConnIdleTime
	poolConfig.HealthCheckPeriod = cfg.Pool.HealthCheckPeriod
	poolConfig.ConnConfig.ConnectTimeout = cfg.Pool.ConnectTimeout

	// Create the pool
	pool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create replica pool: %w", err)
	}

	// Test the connection
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping replica database: %w", err)
	}

	return &ReplicaPool{
		pool:      pool,
		config:    replicaConfig,
		weight:    replicaConfig.Weight,
		priority:  replicaConfig.Priority,
		isHealthy: 1,
	}, nil
}

// GetPrimaryPool returns the primary database pool for write operations
func (db *HighPerformanceDB) GetPrimaryPool() *pgxpool.Pool {
	return db.primary
}

// GetReadPool returns an appropriate read replica pool using weighted round-robin
func (db *HighPerformanceDB) GetReadPool() *pgxpool.Pool {
	if len(db.readReplicas) == 0 {
		return db.primary // Fallback to primary if no replicas
	}

	// Filter healthy replicas
	healthyReplicas := make([]*ReplicaPool, 0, len(db.readReplicas))
	for _, replica := range db.readReplicas {
		if atomic.LoadInt64(&replica.isHealthy) == 1 {
			healthyReplicas = append(healthyReplicas, replica)
		}
	}

	if len(healthyReplicas) == 0 {
		return db.primary // Fallback to primary if no healthy replicas
	}

	// Weighted random selection
	totalWeight := 0
	for _, replica := range healthyReplicas {
		totalWeight += replica.weight
	}

	if totalWeight == 0 {
		// If all weights are 0, use random selection
		return healthyReplicas[rand.Intn(len(healthyReplicas))].pool
	}

	randomWeight := rand.Intn(totalWeight)
	currentWeight := 0
	for _, replica := range healthyReplicas {
		currentWeight += replica.weight
		if randomWeight < currentWeight {
			return replica.pool
		}
	}

	// Fallback (should not reach here)
	return healthyReplicas[0].pool
}

// ExecuteWithContext executes a query with context and automatic read/write routing
func (db *HighPerformanceDB) ExecuteWithContext(ctx context.Context, query string, args ...interface{}) (pgconn.CommandTag, error) {
	if !db.circuitBreaker.CanExecute() {
		return pgconn.CommandTag{}, errors.New("circuit breaker is open")
	}

	start := time.Now()
	defer func() {
		duration := time.Since(start)
		db.updateQueryMetrics(duration, true)
	}()

	// Use primary pool for write operations
	pool := db.GetPrimaryPool()

	result, err := pool.Exec(ctx, query, args...)
	if err != nil {
		db.circuitBreaker.RecordFailure()
		db.updateQueryMetrics(time.Since(start), false)
		atomic.AddInt64(&db.metrics.FailedQueries, 1)
		return pgconn.CommandTag{}, err
	}

	db.circuitBreaker.RecordSuccess()
	atomic.AddInt64(&db.metrics.SuccessfulQueries, 1)
	return result, nil
}

// QueryWithContext executes a read query with automatic replica routing
func (db *HighPerformanceDB) QueryWithContext(ctx context.Context, query string, args ...interface{}) (pgx.Rows, error) {
	if !db.circuitBreaker.CanExecute() {
		return nil, errors.New("circuit breaker is open")
	}

	start := time.Now()
	defer func() {
		duration := time.Since(start)
		db.updateQueryMetrics(duration, true)
	}()

	// Use read replica for read operations
	pool := db.GetReadPool()

	rows, err := pool.Query(ctx, query, args...)
	if err != nil {
		db.circuitBreaker.RecordFailure()
		db.updateQueryMetrics(time.Since(start), false)
		atomic.AddInt64(&db.metrics.FailedQueries, 1)
		return nil, err
	}

	db.circuitBreaker.RecordSuccess()
	atomic.AddInt64(&db.metrics.SuccessfulQueries, 1)
	return rows, nil
}

// QueryRowWithContext executes a single-row read query with automatic replica routing
func (db *HighPerformanceDB) QueryRowWithContext(ctx context.Context, query string, args ...interface{}) pgx.Row {
	// Use read replica for read operations
	pool := db.GetReadPool()
	return pool.QueryRow(ctx, query, args...)
}

// BeginTx starts a new transaction on the primary database
func (db *HighPerformanceDB) BeginTx(ctx context.Context) (pgx.Tx, error) {
	if !db.circuitBreaker.CanExecute() {
		return nil, errors.New("circuit breaker is open")
	}

	tx, err := db.primary.BeginTx(ctx, pgx.TxOptions{
		IsoLevel: pgx.ReadCommitted,
	})

	if err != nil {
		db.circuitBreaker.RecordFailure()
		return nil, err
	}

	db.circuitBreaker.RecordSuccess()
	return tx, nil
}

// GetLegacyDB returns a sqlx.DB compatible interface for backward compatibility
func (db *HighPerformanceDB) GetLegacyDB() (*sqlx.DB, error) {
	// This is a compatibility layer for existing code that uses sqlx
	// In production, you should migrate to use the pgx interface directly
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		db.config.Database.User,
		db.config.Database.Password,
		db.config.Database.Host,
		db.config.Database.Port,
		db.config.Database.Name,
		db.config.Database.SSLMode,
	)

	return sqlx.Connect("postgres", dsn)
}

// GetMetrics returns current connection pool metrics
func (db *HighPerformanceDB) GetMetrics() PoolMetrics {
	db.metrics.mu.RLock()
	defer db.metrics.mu.RUnlock()

	// Update real-time metrics from pgxpool
	if db.primary != nil {
		stats := db.primary.Stat()
		atomic.StoreInt64(&db.metrics.TotalConnections, int64(stats.TotalConns()))
		atomic.StoreInt64(&db.metrics.IdleConnections, int64(stats.IdleConns()))
		atomic.StoreInt64(&db.metrics.WaitingConnections, int64(stats.AcquiredConns()))
	}

	// Return a copy to avoid lock copying issues
	return PoolMetrics{
		TotalConnections:    atomic.LoadInt64(&db.metrics.TotalConnections),
		ActiveConnections:   atomic.LoadInt64(&db.metrics.ActiveConnections),
		IdleConnections:     atomic.LoadInt64(&db.metrics.IdleConnections),
		WaitingConnections:  atomic.LoadInt64(&db.metrics.WaitingConnections),
		TotalQueries:        atomic.LoadInt64(&db.metrics.TotalQueries),
		SuccessfulQueries:   atomic.LoadInt64(&db.metrics.SuccessfulQueries),
		FailedQueries:       atomic.LoadInt64(&db.metrics.FailedQueries),
		AverageQueryTime:    db.metrics.AverageQueryTime,
		ConnectionErrors:    atomic.LoadInt64(&db.metrics.ConnectionErrors),
		CircuitBreakerTrips: atomic.LoadInt64(&db.metrics.CircuitBreakerTrips),
		LastUpdated:         db.metrics.LastUpdated,
	}
}

// IsHealthy returns the current health status of the database
func (db *HighPerformanceDB) IsHealthy() bool {
	return atomic.LoadInt64(&db.isHealthy) == 1
}

// Close closes all database connections
func (db *HighPerformanceDB) Close() {
	if db.primary != nil {
		db.primary.Close()
	}

	for _, replica := range db.readReplicas {
		if replica.pool != nil {
			replica.pool.Close()
		}
	}
}

// updateQueryMetrics updates query execution metrics
func (db *HighPerformanceDB) updateQueryMetrics(duration time.Duration, success bool) {
	atomic.AddInt64(&db.metrics.TotalQueries, 1)

	// Update average query time using exponential moving average
	currentAvg := db.metrics.AverageQueryTime
	newAvg := currentAvg + (duration.Seconds()-currentAvg)*0.1 // 10% weight for new value

	db.metrics.mu.Lock()
	db.metrics.AverageQueryTime = newAvg
	db.metrics.LastUpdated = time.Now()
	db.metrics.mu.Unlock()
}

// startMetricsCollection starts the metrics collection goroutine
func (db *HighPerformanceDB) startMetricsCollection() {
	ticker := time.NewTicker(db.config.Database.Pool.MetricsInterval)
	defer ticker.Stop()

	for range ticker.C {
		if db.primary != nil {
			stats := db.primary.Stat()
			atomic.StoreInt64(&db.metrics.TotalConnections, int64(stats.TotalConns()))
			atomic.StoreInt64(&db.metrics.IdleConnections, int64(stats.IdleConns()))

			// Log metrics if tracing is enabled
			if db.config.Database.Pool.EnableTracing {
				log.Printf("DB Metrics - Total: %d, Idle: %d, Active: %d, Queries: %d, Avg Time: %.3fs",
					stats.TotalConns(),
					stats.IdleConns(),
					atomic.LoadInt64(&db.metrics.ActiveConnections),
					atomic.LoadInt64(&db.metrics.TotalQueries),
					db.metrics.AverageQueryTime,
				)
			}
		}
	}
}

// startHealthChecks starts the health check goroutine
func (db *HighPerformanceDB) startHealthChecks() {
	ticker := time.NewTicker(db.config.Database.Pool.HealthCheckPeriod)
	defer ticker.Stop()

	for range ticker.C {
		db.performHealthCheck()
	}
}

// performHealthCheck performs health checks on all database connections
func (db *HighPerformanceDB) performHealthCheck() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Check primary database
	primaryHealthy := true
	if db.primary != nil {
		if err := db.primary.Ping(ctx); err != nil {
			log.Printf("Primary database health check failed: %v", err)
			primaryHealthy = false
			atomic.AddInt64(&db.metrics.ConnectionErrors, 1)
		}
	}

	if primaryHealthy {
		atomic.StoreInt64(&db.isHealthy, 1)
	} else {
		atomic.StoreInt64(&db.isHealthy, 0)
	}

	// Check read replicas
	for _, replica := range db.readReplicas {
		if replica.pool != nil {
			if err := replica.pool.Ping(ctx); err != nil {
				log.Printf("Replica %s:%d health check failed: %v",
					replica.config.Host, replica.config.Port, err)
				atomic.StoreInt64(&replica.isHealthy, 0)
				atomic.AddInt64(&db.metrics.ConnectionErrors, 1)
			} else {
				atomic.StoreInt64(&replica.isHealthy, 1)
			}
		}
	}

	db.mu.Lock()
	db.lastHealthCheck = time.Now()
	db.mu.Unlock()
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(failureThreshold int, resetTimeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		failureThreshold: failureThreshold,
		resetTimeout:     resetTimeout,
		state:            0, // Closed
	}
}

// CanExecute checks if the circuit breaker allows execution
func (cb *CircuitBreaker) CanExecute() bool {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	switch cb.state {
	case 0: // Closed
		return true
	case 1: // Open
		if time.Since(cb.lastFailureTime) > cb.resetTimeout {
			cb.mu.RUnlock()
			cb.mu.Lock()
			cb.state = 2 // Half-Open
			cb.mu.Unlock()
			cb.mu.RLock()
			return true
		}
		return false
	case 2: // Half-Open
		return true
	default:
		return false
	}
}

// RecordSuccess records a successful operation
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures = 0
	if cb.state == 2 { // Half-Open -> Closed
		cb.state = 0
	}
}

// RecordFailure records a failed operation
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures++
	cb.lastFailureTime = time.Now()

	if cb.failures >= int64(cb.failureThreshold) {
		cb.state = 1 // Open
	}
}
