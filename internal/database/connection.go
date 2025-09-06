package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"quantumtask-auth-api/internal/config"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq" // PostgreSQL driver for backward compatibility
)

// DB wraps the database connection with high-performance pool
type DB struct {
	*sqlx.DB
	highPerfPool *HighPerformanceDB // High-performance pool for new operations
}

// Connect establishes a connection to the PostgreSQL database with high-performance pool
func Connect(cfg *config.Config) (*DB, error) {
	// Create high-performance connection pool
	highPerfPool, err := NewHighPerformanceDB(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create high-performance pool: %w", err)
	}

	// Create legacy sqlx connection for backward compatibility
	legacyDB, err := highPerfPool.GetLegacyDB()
	if err != nil {
		highPerfPool.Close()
		return nil, fmt.Errorf("failed to create legacy database connection: %w", err)
	}

	// Configure legacy connection pool with high-performance settings
	legacyDB.SetMaxOpenConns(int(cfg.Database.Pool.MaxConns / 4)) // Use quarter for legacy connections
	legacyDB.SetMaxIdleConns(int(cfg.Database.Pool.MinConns / 4))
	legacyDB.SetConnMaxLifetime(cfg.Database.Pool.MaxConnLifetime)
	legacyDB.SetConnMaxIdleTime(cfg.Database.Pool.MaxConnIdleTime)

	// Test the connection
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := legacyDB.PingContext(ctx); err != nil {
		legacyDB.Close()
		highPerfPool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &DB{
		DB:           legacyDB,
		highPerfPool: highPerfPool,
	}, nil
}

// Close closes the database connection
func (db *DB) Close() error {
	// Close high-performance pool first
	if db.highPerfPool != nil {
		db.highPerfPool.Close()
	}
	// Close legacy connection
	return db.DB.Close()
}

// RunMigrations runs database migrations
func (db *DB) RunMigrations() error {
	// Check if the auth schema exists by trying to select from users table
	var authExists bool
	err := db.Get(&authExists, "SELECT EXISTS(SELECT 1 FROM information_schema.tables WHERE table_name = 'users')")
	if err != nil {
		return fmt.Errorf("failed to check if auth schema exists: %w", err)
	}

	if !authExists {
		return fmt.Errorf("auth schema not found. Please run the auth.sql migration file first")
	}

	// Check if AI conversation schema exists
	var aiExists bool
	err = db.Get(&aiExists, "SELECT EXISTS(SELECT 1 FROM information_schema.tables WHERE table_name = 'ai_conversations')")
	if err != nil {
		return fmt.Errorf("failed to check if ai conversation schema exists: %w", err)
	}

	if !aiExists {
		return fmt.Errorf("AI conversation schema not found. Please run the ai_conversation_migration.sql file")
	}

	return nil
}

// HealthCheck performs a health check on the database
func (db *DB) HealthCheck() error {
	ctx, cancel := timeoutContext(5 * time.Second)
	defer cancel()

	var result int
	err := db.GetContext(ctx, &result, "SELECT 1")
	if err != nil {
		return fmt.Errorf("database health check failed: %w", err)
	}

	return nil
}

// GetStats returns database connection statistics
func (db *DB) GetStats() sql.DBStats {
	return db.DB.Stats()
}

// Begin starts a new transaction
func (db *DB) Begin() (*sqlx.Tx, error) {
	return db.DB.Beginx()
}

// WithTransaction executes a function within a database transaction
func (db *DB) WithTransaction(fn func(*sqlx.Tx) error) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
	}()

	if err := fn(tx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("transaction failed: %v, rollback failed: %w", err, rbErr)
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// High-Performance Pool Access Methods

// GetHighPerfPool returns the high-performance database pool
func (db *DB) GetHighPerfPool() *HighPerformanceDB {
	return db.highPerfPool
}

// ExecuteWithReadWriteSplit executes a query with automatic read/write routing
func (db *DB) ExecuteWithReadWriteSplit(ctx context.Context, query string, args ...interface{}) error {
	if db.highPerfPool == nil {
		// Fallback to legacy connection
		_, err := db.DB.ExecContext(ctx, query, args...)
		return err
	}

	_, err := db.highPerfPool.ExecuteWithContext(ctx, query, args...)
	return err
}

// QueryWithReadWriteSplit executes a read query with automatic replica routing
func (db *DB) QueryWithReadWriteSplit(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	if db.highPerfPool == nil {
		// Fallback to legacy connection
		return db.DB.QueryContext(ctx, query, args...)
	}

	// For now, use the legacy connection for compatibility
	// In production, you should migrate to use the high-performance pool directly
	// and handle pgx.Rows instead of sql.Rows
	return db.DB.QueryContext(ctx, query, args...)
}

// GetPoolMetrics returns current connection pool metrics
func (db *DB) GetPoolMetrics() PoolMetrics {
	if db.highPerfPool == nil {
		return PoolMetrics{}
	}
	return db.highPerfPool.GetMetrics()
}

// IsHealthyAdvanced returns advanced health status with metrics
func (db *DB) IsHealthyAdvanced() (bool, PoolMetrics) {
	if db.highPerfPool == nil {
		// Fallback health check
		err := db.HealthCheck()
		return err == nil, PoolMetrics{}
	}

	isHealthy := db.highPerfPool.IsHealthy()
	metrics := db.highPerfPool.GetMetrics() // This already returns a copy
	return isHealthy, metrics
}
