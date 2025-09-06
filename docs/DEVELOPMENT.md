# MacWrite Auth API - Development Guide

## 🚀 Quick Start with Makefile

### Initial Setup

1. **Initialize the project** (one-time setup):

   ```bash
   make init
   ```

   This will:

   - Install Go dependencies
   - Install Air for live reloading
   - Verify Air configuration
   - Show next steps

2. **Set up environment**:

   ```bash
   make env  # Copy .env.example to .env
   # Then edit .env with your configuration
   ```

3. **Set up database** (if you have PostgreSQL installed):
   ```bash
   make db-setup
   ```

### Development Workflow

#### Live Reload Development Server

```bash
make dev          # Start development server with Air (live reload)
# OR
make air          # Alias for dev
# OR
make serve        # Another alias for dev
```

This will:

- Watch for file changes in `.go` files
- Automatically rebuild and restart the server
- Show build errors in real-time
- Clear screen on rebuild

#### Regular Development Server

```bash
make run          # Run server without live reload
# OR
make start        # Alias for run
```

#### Building the Application

```bash
make build        # Build binary for current platform
make build-all    # Build for Linux, Windows, and macOS
make build-linux  # Build for Linux only
make build-windows # Build for Windows only
make build-mac    # Build for macOS only
```

#### Testing

```bash
make test                 # Run all tests
make test-coverage        # Run tests with coverage
make test-coverage-html   # Generate HTML coverage report
make benchmark            # Run benchmarks
```

#### Code Quality

```bash
make fmt          # Format Go code
make lint         # Run linter (installs golangci-lint if needed)
make vet          # Run go vet
make check        # Run fmt + vet + lint + test
```

#### Database Management

```bash
make db-setup     # Set up database from schema
make db-reset     # Reset database (WARNING: destructive!)
```

#### Dependency Management

```bash
make deps         # Download dependencies
make deps-update  # Update all dependencies
make deps-vendor  # Vendor dependencies
```

#### Utility Commands

```bash
make clean        # Clean build artifacts
make clean-cache  # Clean Go module cache
make tools        # Install development tools
make status       # Show project status
make help         # Show all available commands
```

## 🔧 Air Configuration

The project includes a `.air.toml` configuration file that:

- Watches `.go` files for changes
- Excludes test files (`*_test.go`)
- Builds to `tmp/quantumtask-auth-api`
- Automatically restarts on file changes
- Shows colored output for different operations

### Customizing Air Behavior

Edit `.air.toml` to modify:

- **File watching patterns**: Change `include_ext` or `exclude_regex`
- **Build command**: Modify `cmd` parameter
- **Excluded directories**: Update `exclude_dir`
- **Rebuild delay**: Adjust `delay` (milliseconds)
- **Colors**: Customize `[color]` section

## 📁 Project Structure

```
quantumtask-auth-api/
├── cmd/server/main.go      # Application entry point
├── internal/               # Private application code
│   ├── config/            # Configuration management
│   ├── database/          # Database connection
│   ├── handler/           # HTTP request handlers
│   ├── middleware/        # Authentication, security, rate limiting
│   ├── models/            # Data models and request/response
│   ├── repository/        # Data access layer
│   └── service/           # Business logic layer
├── pkg/                   # Public/reusable packages
│   ├── auth/              # JWT and password utilities
│   ├── oauth/             # OAuth provider integrations
│   └── validation/        # Input validation utilities
├── docs/                  # Documentation
├── tmp/                   # Air build directory (auto-created)
├── .air.toml              # Air configuration
├── Makefile               # Build automation
└── README.md              # Main documentation
```

## 🌟 Development Tips

### 1. Live Reload Workflow

```bash
# Terminal 1: Start development server
make dev

# Terminal 2: Make code changes
# The server will automatically restart when you save files
```

### 2. Testing Workflow

```bash
# Run tests in watch mode (manual)
watch -n 2 'make test'

# Or run specific tests
go test ./internal/service/ -v
```

### 3. Database Development

```bash
# Reset database when schema changes
make db-reset

# Check database status
psql -d quantumtask_auth -c "\dt"
```

### 4. API Testing

```bash
# Start server
make dev

# In another terminal, test endpoints
curl http://localhost:8080/health

# Or use the Postman collection
# Import: docs/MacWrite_Auth_API.postman_collection.json
```

### 5. Debugging

```bash
# Build with debug info
go build -gcflags="all=-N -l" -o debug-server cmd/server/main.go

# Run with delve debugger
dlv exec ./debug-server
```

## 🔍 Troubleshooting

### Air Issues

```bash
# If Air doesn't start
make init          # Reinstall Air

# If builds fail
make clean         # Clean build artifacts
make deps          # Ensure dependencies are installed

# Check Air config
cat .air.toml      # Verify configuration
```

### Database Issues

```bash
# Check PostgreSQL connection
psql -d quantumtask_auth -c "SELECT 1;"

# Verify schema
psql -d quantumtask_auth -c "\dt"

# Reset if needed
make db-reset
```

### Build Issues

```bash
# Clean and rebuild
make clean build

# Check Go environment
go env
make status

# Update dependencies
make deps-update
```

### Port Already in Use

```bash
# Find process using port 8080
lsof -i :8080

# Kill process if needed
kill -9 <PID>
```

## 📝 Environment Variables

Key variables to configure in `.env`:

```env
# Database (required)
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=your_password
DB_NAME=quantumtask_auth

# JWT (required)
JWT_SECRET=your-super-secret-jwt-key

# Encryption (required)
ENCRYPTION_KEY=your-32-character-encryption-key

# Google OAuth (optional)
GOOGLE_CLIENT_ID=your-google-client-id
GOOGLE_CLIENT_SECRET=your-google-client-secret

# Server (optional)
PORT=8080
GIN_MODE=debug
```

## 🎯 Performance Tips

### Development

- Use `make dev` for instant feedback during development
- Use `make test-coverage` to identify untested code
- Use `make lint` to catch issues early

### Production

- Use `make build` for optimized binaries
- Set `GIN_MODE=release` in production
- Configure proper database connection pooling
- Enable HTTPS and security headers

---

**Happy coding! 🚀**
