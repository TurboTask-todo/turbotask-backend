package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds all configuration for the application
type Config struct {
	// Database configuration
	Database DatabaseConfig

	// Server configuration
	Server ServerConfig

	// JWT configuration
	JWT JWTConfig

	// OAuth configuration
	OAuth OAuthConfig

	// Security configuration
	Security SecurityConfig

	// Email configuration
	Email EmailConfig

	// Application configuration
	App AppConfig

	// Redis configuration
	Redis RedisConfig

	// Queue configuration
	Queue QueueConfig

	// AI configuration
	AI AIConfig

	// Middleware.io configuration
	Middleware MiddlewareConfig
}

// DatabaseConfig holds database connection configuration
type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Name     string
	SSLMode  string
	// High-performance connection pool settings
	Pool DatabasePoolConfig
	// Read replica configuration for read/write splitting
	ReadReplicas []DatabaseReplicaConfig
}

// DatabasePoolConfig holds connection pool configuration for high performance
type DatabasePoolConfig struct {
	// Core pool settings
	MaxConns        int32         `yaml:"max_conns"`          // Maximum number of connections in the pool
	MinConns        int32         `yaml:"min_conns"`          // Minimum number of connections in the pool
	MaxConnLifetime time.Duration `yaml:"max_conn_lifetime"`  // Maximum lifetime of a connection
	MaxConnIdleTime time.Duration `yaml:"max_conn_idle_time"` // Maximum time a connection can be idle

	// Advanced performance settings
	HealthCheckPeriod time.Duration `yaml:"health_check_period"` // How often to health check idle connections
	ConnectTimeout    time.Duration `yaml:"connect_timeout"`     // Timeout for establishing new connections
	QueryTimeout      time.Duration `yaml:"query_timeout"`       // Default timeout for queries

	// High-throughput optimizations
	PreparedStatementCache bool `yaml:"prepared_statement_cache"` // Enable prepared statement caching
	StatementCacheCapacity int  `yaml:"statement_cache_capacity"` // Prepared statement cache size

	// Connection pool behavior
	AcquireTimeout   time.Duration `yaml:"acquire_timeout"`    // Timeout for acquiring connection from pool
	LazyConnect      bool          `yaml:"lazy_connect"`       // Don't establish connections until needed
	LoadBalanceHosts bool          `yaml:"load_balance_hosts"` // Load balance across multiple hosts

	// Monitoring and observability
	EnableMetrics   bool          `yaml:"enable_metrics"`   // Enable connection pool metrics
	MetricsInterval time.Duration `yaml:"metrics_interval"` // How often to collect metrics
	EnableTracing   bool          `yaml:"enable_tracing"`   // Enable connection tracing for debugging
	LogLevel        string        `yaml:"log_level"`        // Database logging level
}

// DatabaseReplicaConfig holds read replica configuration
type DatabaseReplicaConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Weight   int    `yaml:"weight"`   // Load balancing weight
	Priority int    `yaml:"priority"` // Failover priority (lower = higher priority)
}

// ServerConfig holds server configuration
type ServerConfig struct {
	Port        int
	GinMode     string
	CORSOrigins []string
}

// JWTConfig holds JWT configuration
type JWTConfig struct {
	Secret               string
	AccessTokenDuration  time.Duration
	RefreshTokenDuration time.Duration
}

// OAuthConfig holds OAuth provider configurations
type OAuthConfig struct {
	Google GoogleOAuthConfig
}

// GoogleOAuthConfig holds Google OAuth configuration
type GoogleOAuthConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
}

// SecurityConfig holds security-related configuration
type SecurityConfig struct {
	RateLimitRequestsPerMinute int
	AccountLockDurationMinutes int
	MaxLoginAttempts           int
	PasswordMinLength          int
	EncryptionKey              string
}

// EmailConfig holds email configuration
type EmailConfig struct {
	SMTPHost     string
	SMTPPort     int
	SMTPUsername string
	SMTPPassword string
	FromEmail    string
	FromName     string
	Enabled      bool
}

// AppConfig holds general application configuration
type AppConfig struct {
	Name    string
	Version string
	Env     string
}

// RedisConfig holds Redis configuration
type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}

// QueueConfig holds queue configuration
type QueueConfig struct {
	URL          string
	ExchangeName string
	MaxRetries   int
}

// AIConfig holds AI service configuration
type AIConfig struct {
	ExternalAIURL   string
	DefaultModel    string
	RequestTimeout  time.Duration
	MaxRetries      int
	RateLimitPerMin int
}

// MiddlewareConfig holds middleware.io APM configuration
type MiddlewareConfig struct {
	Enabled     bool
	AccessToken string
	Target      string
	Service     string
	Version     string
	Environment string
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	config := &Config{}

	// Load database configuration
	dbPort, err := strconv.Atoi(getEnv("DB_PORT", "5432"))
	if err != nil {
		return nil, fmt.Errorf("invalid DB_PORT: %v", err)
	}

	// Load database pool configuration with high-performance defaults
	maxConns, _ := strconv.ParseInt(getEnv("DB_POOL_MAX_CONNS", "1000"), 10, 32) // 1000 connections for high throughput
	minConns, _ := strconv.ParseInt(getEnv("DB_POOL_MIN_CONNS", "50"), 10, 32)   // 50 minimum connections
	maxConnLifetime, _ := time.ParseDuration(getEnv("DB_POOL_MAX_CONN_LIFETIME", "1h"))
	maxConnIdleTime, _ := time.ParseDuration(getEnv("DB_POOL_MAX_CONN_IDLE_TIME", "30m"))
	healthCheckPeriod, _ := time.ParseDuration(getEnv("DB_POOL_HEALTH_CHECK_PERIOD", "1m"))
	connectTimeout, _ := time.ParseDuration(getEnv("DB_POOL_CONNECT_TIMEOUT", "10s"))
	queryTimeout, _ := time.ParseDuration(getEnv("DB_POOL_QUERY_TIMEOUT", "30s"))
	preparedStmtCache, _ := strconv.ParseBool(getEnv("DB_POOL_PREPARED_STATEMENT_CACHE", "true"))
	stmtCacheCapacity, _ := strconv.Atoi(getEnv("DB_POOL_STATEMENT_CACHE_CAPACITY", "1000"))
	acquireTimeout, _ := time.ParseDuration(getEnv("DB_POOL_ACQUIRE_TIMEOUT", "5s"))
	lazyConnect, _ := strconv.ParseBool(getEnv("DB_POOL_LAZY_CONNECT", "false"))
	loadBalanceHosts, _ := strconv.ParseBool(getEnv("DB_POOL_LOAD_BALANCE_HOSTS", "true"))
	enableMetrics, _ := strconv.ParseBool(getEnv("DB_POOL_ENABLE_METRICS", "true"))
	metricsInterval, _ := time.ParseDuration(getEnv("DB_POOL_METRICS_INTERVAL", "30s"))
	enableTracing, _ := strconv.ParseBool(getEnv("DB_POOL_ENABLE_TRACING", "false"))

	config.Database = DatabaseConfig{
		Host:     getEnv("DB_HOST", "ep-nameless-morning-adw2ijdl-pooler.c-2.us-east-1.aws.neon.tech"),
		Port:     dbPort,
		User:     getEnv("DB_USER", "neondb_owner"),
		Password: getEnv("DB_PASSWORD", "npg_w5L4XCBkqDOJ"),
		Name:     getEnv("DB_NAME", "neondb"),
		SSLMode:  getEnv("DB_SSL_MODE", "require"),
		Pool: DatabasePoolConfig{
			MaxConns:               int32(maxConns),
			MinConns:               int32(minConns),
			MaxConnLifetime:        maxConnLifetime,
			MaxConnIdleTime:        maxConnIdleTime,
			HealthCheckPeriod:      healthCheckPeriod,
			ConnectTimeout:         connectTimeout,
			QueryTimeout:           queryTimeout,
			PreparedStatementCache: preparedStmtCache,
			StatementCacheCapacity: stmtCacheCapacity,
			AcquireTimeout:         acquireTimeout,
			LazyConnect:            lazyConnect,
			LoadBalanceHosts:       loadBalanceHosts,
			EnableMetrics:          enableMetrics,
			MetricsInterval:        metricsInterval,
			EnableTracing:          enableTracing,
			LogLevel:               getEnv("DB_POOL_LOG_LEVEL", "warn"),
		},
		ReadReplicas: loadReadReplicas(),
	}

	// Load server configuration
	serverPort, err := strconv.Atoi(getEnv("PORT", "8080"))
	if err != nil {
		return nil, fmt.Errorf("invalid PORT: %v", err)
	}

	corsOrigins := strings.Split(getEnv("CORS_ORIGINS", "http://localhost:3000"), ",")
	for i, origin := range corsOrigins {
		corsOrigins[i] = strings.TrimSpace(origin)
	}

	config.Server = ServerConfig{
		Port:        serverPort,
		GinMode:     getEnv("GIN_MODE", "debug"),
		CORSOrigins: corsOrigins,
	}

	// Load JWT configuration
	accessTokenDuration, err := time.ParseDuration(getEnv("JWT_ACCESS_TOKEN_DURATION", "1121212h"))
	if err != nil {
		return nil, fmt.Errorf("invalid JWT_ACCESS_TOKEN_DURATION: %v", err)
	}

	refreshTokenDuration, err := time.ParseDuration(getEnv("JWT_REFRESH_TOKEN_DURATION", "168h"))
	if err != nil {
		return nil, fmt.Errorf("invalid JWT_REFRESH_TOKEN_DURATION: %v", err)
	}

	config.JWT = JWTConfig{
		Secret:               getEnv("JWT_SECRET", "DEMO"),
		AccessTokenDuration:  accessTokenDuration,
		RefreshTokenDuration: refreshTokenDuration,
	}

	// Load OAuth configuration
	config.OAuth = OAuthConfig{
		Google: GoogleOAuthConfig{
			ClientID:     getEnv("GOOGLE_CLIENT_ID", ""),
			ClientSecret: getEnv("GOOGLE_CLIENT_SECRET", ""),
			RedirectURL:  getEnv("GOOGLE_REDIRECT_URL", ""),
		},
	}

	// Load security configuration
	rateLimitRPM, err := strconv.Atoi(getEnv("RATE_LIMIT_REQUESTS_PER_MINUTE", "12000"))
	if err != nil {
		return nil, fmt.Errorf("invalid RATE_LIMIT_REQUESTS_PER_MINUTE: %v", err)
	}

	lockDuration, err := strconv.Atoi(getEnv("ACCOUNT_LOCK_DURATION_MINUTES", "100020"))
	if err != nil {
		return nil, fmt.Errorf("invalid ACCOUNT_LOCK_DURATION_MINUTES: %v", err)
	}

	maxAttempts, err := strconv.Atoi(getEnv("MAX_LOGIN_ATTEMPTS", "6"))
	if err != nil {
		return nil, fmt.Errorf("invalid MAX_LOGIN_ATTEMPTS: %v", err)
	}

	passwordMinLength, err := strconv.Atoi(getEnv("PASSWORD_MIN_LENGTH", "8"))
	if err != nil {
		return nil, fmt.Errorf("invalid PASSWORD_MIN_LENGTH: %v", err)
	}

	config.Security = SecurityConfig{
		RateLimitRequestsPerMinute: rateLimitRPM,
		AccountLockDurationMinutes: lockDuration,
		MaxLoginAttempts:           maxAttempts,
		PasswordMinLength:          passwordMinLength,
		EncryptionKey:              getEnv("ENCRYPTION_KEY", "12345678901234567890123456789012"),
	}

	// Load email configuration
	smtpPort, err := strconv.Atoi(getEnv("SMTP_PORT", "587"))
	if err != nil {
		return nil, fmt.Errorf("invalid SMTP_PORT: %v", err)
	}

	emailEnabled, err := strconv.ParseBool(getEnv("EMAIL_ENABLED", "false"))
	if err != nil {
		return nil, fmt.Errorf("invalid EMAIL_ENABLED: %v", err)
	}

	config.Email = EmailConfig{
		SMTPHost:     getEnv("SMTP_HOST", "smtp.gmail.com"),
		SMTPPort:     smtpPort,
		SMTPUsername: getEnv("SMTP_USERNAME", "gokulakrishnanr812@gmail.com"),
		SMTPPassword: getEnv("SMTP_PASSWORD", "bgvpjbttfgvhoazr"),
		FromEmail:    getEnv("FROM_EMAIL", "gokulakrishnanr812@gmail.com"),
		FromName:     getEnv("FROM_NAME", "MacWrite Team"),
		Enabled:      emailEnabled,
	}

	// Load application configuration
	config.App = AppConfig{
		Name:    getEnv("APP_NAME", "MacWrite Auth API"),
		Version: getEnv("APP_VERSION", "1.0.0"),
		Env:     getEnv("APP_ENV", "development"),
	}

	// Load Redis configuration
	redisDB, err := strconv.Atoi(getEnv("REDIS_DB", "0"))
	if err != nil {
		return nil, fmt.Errorf("invalid REDIS_DB: %v", err)
	}

	config.Redis = RedisConfig{
		Addr:     getEnv("REDIS_ADDR", "localhost:6379"),
		Password: getEnv("REDIS_PASSWORD", ""),
		DB:       redisDB,
	}

	// Load Queue configuration
	queueMaxRetries, err := strconv.Atoi(getEnv("QUEUE_MAX_RETRIES", "10"))
	if err != nil {
		return nil, fmt.Errorf("invalid QUEUE_MAX_RETRIES: %v", err)
	}

	config.Queue = QueueConfig{
		URL:          getEnv("RABBITMQ_URL", "amqp://guest:guest@rabbitmq-jbos.onrender.com:5672/"),
		ExchangeName: getEnv("QUEUE_EXCHANGE_NAME", "ai.enhancement.direct"),
		MaxRetries:   queueMaxRetries,
	}

	// Load AI configuration
	aiRequestTimeout, err := time.ParseDuration(getEnv("AI_REQUEST_TIMEOUT", "60s"))
	if err != nil {
		return nil, fmt.Errorf("invalid AI_REQUEST_TIMEOUT: %v", err)
	}

	aiMaxRetries, err := strconv.Atoi(getEnv("AI_MAX_RETRIES", "3"))
	if err != nil {
		return nil, fmt.Errorf("invalid AI_MAX_RETRIES: %v", err)
	}

	aiRateLimit, err := strconv.Atoi(getEnv("AI_RATE_LIMIT_PER_MIN", "60"))
	if err != nil {
		return nil, fmt.Errorf("invalid AI_RATE_LIMIT_PER_MIN: %v", err)
	}

	config.AI = AIConfig{
		ExternalAIURL:   getEnv("EXTERNAL_AI_URL", "https://auto-comment.gokulakrishnanr812-492.workers.dev/"),
		DefaultModel:    getEnv("AI_DEFAULT_MODEL", "@cf/meta/llama-3.3-70b-instruct-fp8-fast"),
		RequestTimeout:  aiRequestTimeout,
		MaxRetries:      aiMaxRetries,
		RateLimitPerMin: aiRateLimit,
	}

	// Load middleware.io configuration
	middlewareEnabled, err := strconv.ParseBool(getEnv("MIDDLEWARE_ENABLED", "true"))
	if err != nil {
		return nil, fmt.Errorf("invalid MIDDLEWARE_ENABLED: %v", err)
	}

	config.Middleware = MiddlewareConfig{
		Enabled:     middlewareEnabled,
		AccessToken: getEnv("MIDDLEWARE_ACCESS_TOKEN", "mdskrxjjeqjmxgwqfpdnannqbyupukddffbu"),
		Target:      getEnv("MIDDLEWARE_TARGET", "dopem.middleware.io:443"),
		Service:     getEnv("MIDDLEWARE_SERVICE", "mac"),
		Version:     getEnv("MIDDLEWARE_VERSION", config.App.Version),
		Environment: getEnv("MIDDLEWARE_ENVIRONMENT", config.App.Env),
	}

	// Validate required fields
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %v", err)
	}

	return config, nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.Database.Password == "" {
		return fmt.Errorf("DB_PASSWORD is required")
	}

	if c.JWT.Secret == "" {
		return fmt.Errorf("JWT_SECRET is required")
	}

	if c.Security.EncryptionKey == "" {
		return fmt.Errorf("ENCRYPTION_KEY is required")
	}

	if len(c.Security.EncryptionKey) != 32 {
		return fmt.Errorf("ENCRYPTION_KEY must be exactly 32 characters")
	}

	return nil
}

// IsDevelopment returns true if the application is running in development mode
func (c *Config) IsDevelopment() bool {
	return c.App.Env == "development"
}

// IsProduction returns true if the application is running in production mode
func (c *Config) IsProduction() bool {
	return c.App.Env == "production"
}

// getEnv gets an environment variable with a fallback value
func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

// loadReadReplicas loads read replica configuration from environment variables
func loadReadReplicas() []DatabaseReplicaConfig {
	var replicas []DatabaseReplicaConfig

	// Support up to 10 read replicas
	for i := 1; i <= 10; i++ {
		hostKey := fmt.Sprintf("DB_REPLICA_%d_HOST", i)
		portKey := fmt.Sprintf("DB_REPLICA_%d_PORT", i)
		weightKey := fmt.Sprintf("DB_REPLICA_%d_WEIGHT", i)
		priorityKey := fmt.Sprintf("DB_REPLICA_%d_PRIORITY", i)

		host := getEnv(hostKey, "")
		if host == "" {
			continue // Skip if no host is configured
		}

		port, err := strconv.Atoi(getEnv(portKey, "5432"))
		if err != nil {
			port = 5432
		}

		weight, err := strconv.Atoi(getEnv(weightKey, "1"))
		if err != nil {
			weight = 1
		}

		priority, err := strconv.Atoi(getEnv(priorityKey, fmt.Sprintf("%d", i)))
		if err != nil {
			priority = i
		}

		replicas = append(replicas, DatabaseReplicaConfig{
			Host:     host,
			Port:     port,
			Weight:   weight,
			Priority: priority,
		})
	}

	return replicas
}
