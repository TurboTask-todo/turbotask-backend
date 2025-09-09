# MacWrite Authentication API Makefile
# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=gofmt
BINARY_NAME=macwrite-auth-api
BINARY_UNIX=$(BINARY_NAME)_unix
MAIN_PATH=./cmd/server/main.go

# Air parameters
AIR_CMD=air
AIR_CONFIG=.air.toml

# Default target
.PHONY: help
help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# Development targets
.PHONY: init
init: ## Initialize project dependencies and Air
	@echo "📦 Installing Go dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy
	@echo "🌪️  Installing Air for live reloading..."
	$(GOCMD) install github.com/cosmtrek/air@v1.49.0
	@echo "✅ Checking Air configuration..."
	@$(MAKE) air-config
	@echo "✅ Initialization complete!"
	@echo ""
	@echo "Next steps:"
	@echo "  1. Copy .env.example to .env and configure"
	@echo "  2. Set up your PostgreSQL database"
	@echo "  3. Run 'make dev' to start development server"

.PHONY: air-config
air-config: ## Check Air configuration file
	@if [ ! -f $(AIR_CONFIG) ]; then \
		echo "❌ Air config file not found: $(AIR_CONFIG)"; \
		echo "ℹ️  The .air.toml file should already exist in the project root"; \
		exit 1; \
	else \
		echo "✅ Air config file found: $(AIR_CONFIG)"; \
	fi

.PHONY: dev
dev: ## Start development server with Air (live reload)
	@echo "🚀 Starting development server with live reload..."
	@if [ ! -f $(AIR_CONFIG) ]; then \
		echo "❌ Air config not found. Run 'make init' first."; \
		exit 1; \
	fi
	@mkdir -p tmp
	$(AIR_CMD) -c $(AIR_CONFIG)

.PHONY: air
air: dev ## Alias for dev target

.PHONY: run
run: ## Run the application directly
	@echo "🚀 Starting MacWrite Auth API..."
	$(GOCMD) run $(MAIN_PATH)

# Build targets
.PHONY: build
build: ## Build the application
	@echo "🔨 Building $(BINARY_NAME)..."
	$(GOBUILD) -o $(BINARY_NAME) -v $(MAIN_PATH)

.PHONY: build-linux
build-linux: ## Build for Linux
	@echo "🔨 Building $(BINARY_UNIX) for Linux..."
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(BINARY_UNIX) -v $(MAIN_PATH)

.PHONY: build-windows
build-windows: ## Build for Windows
	@echo "🔨 Building $(BINARY_NAME).exe for Windows..."
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 $(GOBUILD) -o $(BINARY_NAME).exe -v $(MAIN_PATH)

.PHONY: build-mac
build-mac: ## Build for macOS
	@echo "🔨 Building $(BINARY_NAME)_mac for macOS..."
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 $(GOBUILD) -o $(BINARY_NAME)_mac -v $(MAIN_PATH)

.PHONY: build-all
build-all: build-linux build-windows build-mac ## Build for all platforms

# Test targets
.PHONY: test
test: ## Run tests
	@echo "🧪 Running tests..."
	$(GOTEST) -v ./...

.PHONY: test-coverage
test-coverage: ## Run tests with coverage
	@echo "🧪 Running tests with coverage..."
	$(GOTEST) -v -cover ./...

.PHONY: test-coverage-html
test-coverage-html: ## Run tests with HTML coverage report
	@echo "🧪 Running tests with HTML coverage report..."
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "📊 Coverage report generated: coverage.html"

.PHONY: benchmark
benchmark: ## Run benchmarks
	@echo "📊 Running benchmarks..."
	$(GOTEST) -bench=. -benchmem ./...

# Code quality targets
.PHONY: fmt
fmt: ## Format Go code
	@echo "🎨 Formatting code..."
	$(GOFMT) -s -w .

.PHONY: lint
lint: ## Run linting
	@echo "🔍 Running linter..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed. Installing..."; \
		$(GOCMD) install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
		golangci-lint run; \
	fi

.PHONY: vet
vet: ## Run go vet
	@echo "🔍 Running go vet..."
	$(GOCMD) vet ./...

.PHONY: check
check: fmt vet lint test ## Run all code quality checks

# Dependency management
.PHONY: deps
deps: ## Download dependencies
	@echo "📦 Downloading dependencies..."
	$(GOMOD) download

.PHONY: deps-update
deps-update: ## Update dependencies
	@echo "🔄 Updating dependencies..."
	$(GOMOD) tidy
	$(GOGET) -u ./...

.PHONY: deps-vendor
deps-vendor: ## Vendor dependencies
	@echo "📦 Vendoring dependencies..."
	$(GOMOD) vendor

# Database targets
.PHONY: db-setup
db-setup: ## Set up database (requires psql)
	@echo "🗄️  Setting up database..."
	@if [ -z "$(DB_NAME)" ]; then \
		echo "DB_NAME not set. Using default: macwrite_auth"; \
		export DB_NAME=macwrite_auth; \
	fi
	@createdb $(DB_NAME) || echo "Database may already exist"
	@psql -d $(DB_NAME) -f Query/auth.sql
	@echo "✅ Database setup complete!"

.PHONY: db-reset
db-reset: ## Reset database (WARNING: destructive)
	@echo "⚠️  WARNING: This will destroy all data!"
	@read -p "Are you sure? [y/N] " -n 1 -r; \
	if [[ $$REPLY =~ ^[Yy]$$ ]]; then \
		echo ""; \
		dropdb macwrite_auth || true; \
		$(MAKE) db-setup; \
	else \
		echo ""; \
		echo "Cancelled."; \
	fi

# Environment targets
.PHONY: env
env: ## Copy example environment file
	@echo "📄 Copying environment file..."
	@if [ ! -f .env ]; then \
		cp .env.example .env; \
		echo "✅ .env file created from .env.example"; \
		echo "📝 Please edit .env with your configuration"; \
	else \
		echo "⚠️  .env file already exists"; \
	fi

# Docker targets
.PHONY: docker-build
docker-build: ## Build Docker image
	@echo "🐳 Building Docker image..."
	docker build -t macwrite-auth-api:latest .

.PHONY: docker-run
docker-run: ## Run Docker container
	@echo "🐳 Running Docker container..."
	docker compose up -d

# Cleanup targets
.PHONY: clean
clean: ## Clean build artifacts
	@echo "🧹 Cleaning..."
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_UNIX)
	rm -f $(BINARY_NAME).exe
	rm -f $(BINARY_NAME)_mac
	rm -rf tmp/
	rm -f coverage.out
	rm -f coverage.html
	rm -f build-errors.log

.PHONY: clean-cache
clean-cache: ## Clean Go module cache
	@echo "🧹 Cleaning module cache..."
	$(GOCMD) clean -modcache

# Production targets
.PHONY: install
install: build ## Install binary to GOPATH
	@echo "📦 Installing $(BINARY_NAME)..."
	$(GOCMD) install $(MAIN_PATH)

.PHONY: release
release: clean test build ## Prepare release build
	@echo "🚀 Release build complete!"

# Development utilities
.PHONY: tools
tools: ## Install development tools
	@echo "🔧 Installing development tools..."
	$(GOCMD) install github.com/cosmtrek/air@v1.49.0
	$(GOCMD) install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	$(GOCMD) install github.com/swaggo/swag/cmd/swag@latest
	@echo "✅ Development tools installed!"

.PHONY: status
status: ## Show project status
	@echo "📊 MacWrite Auth API Status"
	@echo "=========================="
	@echo "Go version: $$(go version)"
	@echo "Project root: $$(pwd)"
	@echo "Binary name: $(BINARY_NAME)"
	@echo "Main file: $(MAIN_PATH)"
	@echo ""
	@echo "Dependencies:"
	@$(GOMOD) list -m all | head -10
	@echo ""
	@echo "Build status:"
	@if [ -f $(BINARY_NAME) ]; then \
		echo "✅ Binary exists: $(BINARY_NAME)"; \
		ls -la $(BINARY_NAME); \
	else \
		echo "❌ Binary not found: $(BINARY_NAME)"; \
	fi

# Convenience aliases
.PHONY: serve
serve: dev ## Alias for dev

.PHONY: start
start: run ## Alias for run

.PHONY: watch
watch: dev ## Alias for dev