# MacWrite Authentication API

A high-performance, scalable, and professionally structured Golang authentication API with PostgreSQL database, featuring JWT tokens, OAuth integration, comprehensive security measures, and production-ready architecture.

## 🚀 Features

### 🔐 Authentication & Security

- **Email/Password Authentication** with secure Argon2id hashing
- **JWT Access & Refresh Tokens** with automatic rotation
- **Google OAuth Integration** with extensible provider support
- **Account Locking** (20 minutes after 3 failed attempts)
- **Rate Limiting** and brute-force protection
- **Comprehensive Security Headers** (CORS, CSP, HSTS)
- **Device Fingerprinting** for enhanced security tracking

### 👤 User Management

- **User Registration & Verification**
- **Profile Management** with customizable preferences
- **Password Change** with history validation
- **Soft Delete** user accounts with restore capability
- **Multi-timezone & Locale Support**

### 🛡️ Security Features

- **Argon2id Password Hashing** (industry standard)
- **JWT Token Management** with secure signing
- **Rate Limiting** per IP and per user
- **Input Validation** with comprehensive error handling
- **SQL Injection Protection** via parameterized queries
- **CORS Configuration** with environment-based settings

### 📊 Database Design

- **PostgreSQL** with comprehensive schema
- **Audit Logging** for security events
- **Connection Pooling** and optimization
- **Migration Support** with version control
- **Performance Indexes** for all critical queries

## 🏗️ Architecture

```
cmd/
  server/           # Application entry point
internal/
  config/           # Configuration management
  database/         # Database connection & utilities
  handler/          # HTTP request handlers
  middleware/       # Authentication, rate limiting, security
  models/           # Data models and request/response structures
  repository/       # Data access layer
  service/          # Business logic layer
pkg/
  auth/             # JWT and password utilities
  oauth/            # OAuth provider integrations
  validation/       # Input validation utilities
docs/               # API documentation and Postman collection
migrations/         # Database migrations
```

## 🚦 Quick Start

### Prerequisites

- Go 1.21+
- PostgreSQL 13+
- Git

### 1. Clone Repository

```bash
git clone <repository-url>
cd backend-go
```

### 2. Install Dependencies

```bash
go mod download
```

### 3. Database Setup

```sql
-- Create database
CREATE DATABASE macwrite_auth;

-- Run the provided schema
psql -U postgres -d macwrite_auth -f Query/auth.sql
```

### 4. Environment Configuration

```bash
# Copy example environment file
cp .env.example .env

# Edit .env with your configuration
nano .env
```

Required environment variables:

```env
# Database
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=your_password
DB_NAME=macwrite_auth

# JWT Secret (generate a secure random string)
JWT_SECRET=your-super-secret-jwt-key-change-in-production

# Google OAuth (optional)
GOOGLE_CLIENT_ID=your-google-client-id
GOOGLE_CLIENT_SECRET=your-google-client-secret

# Encryption Key (32 characters)
ENCRYPTION_KEY=your-32-character-encryption-key
```

### 5. Run the Server

```bash
# Development mode
go run cmd/server/main.go

# Build and run
go build -o macwrite-auth-api cmd/server/main.go
./macwrite-auth-api
```

Server starts on `http://localhost:8080`

## 📚 API Documentation

### Base URL

```
http://localhost:8080/api/v1
```

### Authentication Endpoints

#### Register User

```http
POST /auth/register
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "SecurePass123!",
  "confirm_password": "SecurePass123!",
  "first_name": "John",
  "last_name": "Doe",
  "username": "johndoe"
}
```

#### Login

```http
POST /auth/login
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "SecurePass123!",
  "remember_me": true
}
```

#### Refresh Token

```http
POST /auth/refresh
Content-Type: application/json

{
  "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

#### Protected Endpoints

All protected endpoints require the `Authorization` header:

```http
Authorization: Bearer <access_token>
```

### User Management Endpoints

#### Get Current User

```http
GET /auth/me
Authorization: Bearer <access_token>
```

#### Update Profile

```http
PUT /auth/profile
Authorization: Bearer <access_token>
Content-Type: application/json

{
  "first_name": "John Updated",
  "last_name": "Doe Updated",
  "timezone": "America/New_York",
  "preferences": {
    "theme": "dark",
    "notifications": true
  }
}
```

#### Change Password

```http
POST /auth/change-password
Authorization: Bearer <access_token>
Content-Type: application/json

{
  "current_password": "OldPass123!",
  "new_password": "NewSecurePass123!",
  "confirm_password": "NewSecurePass123!"
}
```

### OAuth Endpoints

#### Google OAuth

```http
# Start OAuth flow
GET /auth/google

# OAuth callback (handled by Google)
GET /auth/google/callback?code=<auth_code>&state=<state>

# Get OAuth providers
GET /auth/oauth/providers
```

### Validation Endpoints

#### Check Email Availability

```http
GET /auth/check-email?email=test@example.com
```

#### Check Username Availability

```http
GET /auth/check-username?username=johndoe
```

## 🧪 Testing with Postman

### Import Collection

1. Open Postman
2. Click "Import"
3. Select `docs/MacWrite_Auth_API.postman_collection.json`
4. Set environment variable `base_url` to `http://localhost:8080`

### Automated Testing

The collection includes:

- **Pre-request scripts** for dynamic test data
- **Test assertions** for response validation
- **Environment variable management** for tokens
- **Error scenario testing**

### Test Scenarios

- ✅ User Registration & Login
- ✅ Token Refresh & Validation
- ✅ Profile Management
- ✅ Password Changes
- ✅ OAuth Flow Initiation
- ✅ Rate Limiting Verification
- ✅ Error Handling

## 🔒 Security Features

### Password Security

- **Argon2id Hashing** with configurable parameters
- **Password Strength Validation** (length, complexity)
- **Password History** prevention (last 5 passwords)
- **Secure Random Salt** generation

### JWT Security

- **HS256 Signing** with secure secret
- **Short-lived Access Tokens** (15 minutes)
- **Long-lived Refresh Tokens** (7 days)
- **Token Rotation** on refresh
- **Audience & Issuer Validation**

### Rate Limiting

- **Global Rate Limiting**: 60 requests/minute
- **Login Rate Limiting**: 10 attempts/minute
- **Password Reset**: 3 attempts/minute
- **Registration**: 20 attempts/minute
- **Per-User Limits** for authenticated endpoints

### Account Security

- **Failed Login Tracking** per user
- **Account Locking** after 3 failed attempts
- **20-minute Lock Duration** with automatic unlock
- **Security Event Logging** for audit trails

### HTTP Security

- **CORS Configuration** with origin validation
- **Security Headers** (CSP, HSTS, X-Frame-Options)
- **Request Size Limits** (10MB max)
- **Content-Type Validation**
- **User-Agent Filtering**

## 🗄️ Database Schema

### Core Tables

- **users**: User accounts with soft delete
- **user_passwords**: Password history with versioning
- **oauth_providers**: OAuth account linking
- **refresh_tokens**: JWT refresh token management
- **user_security_events**: Comprehensive audit logging
- **user_sessions**: Session tracking and management

### Security Features

- **Soft Delete**: Users can be restored
- **Audit Logging**: All security events tracked
- **Token Families**: Refresh token rotation detection
- **Device Tracking**: Fingerprinting for security

### Performance Optimizations

- **Composite Indexes** for common queries
- **Partial Indexes** for active records
- **Connection Pooling** with optimal settings
- **Query Optimization** with prepared statements

## ⚡ Performance & Scalability

### Database Optimizations

- **Connection Pool**: 25 max connections, 10 idle
- **Query Timeout**: 30 seconds
- **Prepared Statements** for security and performance
- **Strategic Indexing** on all critical paths

### Caching Strategy

- **In-Memory Rate Limiting** with cleanup routines
- **JWT Validation** with minimal database calls
- **Connection Reuse** for HTTP clients

### Production Considerations

- **Graceful Shutdown** with 30-second timeout
- **Health Check Endpoint** for monitoring
- **Structured Logging** with request IDs
- **Environment-based Configuration**

## 🚀 Deployment

### Environment Variables

```env
# Production settings
APP_ENV=production
GIN_MODE=release
JWT_SECRET=<secure-random-256-bit-key>
ENCRYPTION_KEY=<secure-32-character-key>

# Database (use connection pooling)
DB_HOST=your-db-host
DB_PASSWORD=<secure-db-password>

# OAuth (production credentials)
GOOGLE_CLIENT_ID=<production-google-client-id>
GOOGLE_CLIENT_SECRET=<production-google-secret>
GOOGLE_REDIRECT_URL=https://yourdomain.com/auth/google/callback
```

### Docker Deployment

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o main cmd/server/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/main .
CMD ["./main"]
```

### Health Checks

```bash
# Application health
curl http://localhost:8080/health

# Database connectivity
curl http://localhost:8080/api/v1/auth/oauth/providers
```

## 🧪 Testing

### Unit Tests

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run specific package
go test ./internal/service/
```

### Integration Tests

```bash
# Database integration tests
go test -tags=integration ./internal/repository/

# API integration tests
go test -tags=integration ./internal/handler/
```

### Load Testing

```bash
# Using Apache Bench
ab -n 1000 -c 10 http://localhost:8080/health

# Using hey
hey -n 1000 -c 10 http://localhost:8080/api/v1/auth/oauth/providers
```

## 🛠️ Development

### Project Structure

```
macwrite-auth-api/
├── cmd/server/main.go          # Application entry point
├── internal/                   # Private application code
│   ├── config/                 # Configuration management
│   ├── database/               # Database connection
│   ├── handler/                # HTTP handlers
│   ├── middleware/             # HTTP middleware
│   ├── models/                 # Data models
│   ├── repository/             # Data access layer
│   └── service/                # Business logic
├── pkg/                        # Public libraries
│   ├── auth/                   # Authentication utilities
│   ├── oauth/                  # OAuth providers
│   └── validation/             # Input validation
├── docs/                       # Documentation
├── Query/auth.sql              # Database schema
└── README.md                   # This file
```

### Code Style

- **Standard Go formatting** with `gofmt`
- **Linting** with `golangci-lint`
- **Documentation** with Go comments
- **Error Handling** with wrapped errors

### Adding New Features

1. **Models**: Add to `internal/models/`
2. **Repository**: Database operations in `internal/repository/`
3. **Service**: Business logic in `internal/service/`
4. **Handler**: HTTP endpoints in `internal/handler/`
5. **Routes**: Register in `cmd/server/main.go`

## 🤝 Contributing

### Development Setup

```bash
# Clone repository
git clone <repository-url>
cd backend-go

# Install dependencies
go mod download

# Install development tools
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Run linting
golangci-lint run

# Run tests
go test ./...
```

### Pull Request Process

1. Fork the repository
2. Create feature branch (`git checkout -b feature/amazing-feature`)
3. Make changes with tests
4. Run linting and tests
5. Update documentation
6. Submit pull request

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙋‍♀️ Support

### Common Issues

**Database Connection Errors**

```bash
# Check PostgreSQL status
sudo systemctl status postgresql

# Verify database exists
psql -U postgres -l | grep macwrite_auth
```

**JWT Token Issues**

- Ensure `JWT_SECRET` is set and secure
- Check token expiration times
- Verify header format: `Bearer <token>`

**OAuth Configuration**

- Verify Google OAuth credentials
- Check redirect URL configuration
- Ensure proper scopes are requested

### Getting Help

- 📖 Check this README first
- 🐛 Report bugs via GitHub Issues
- 💬 Join our community discussions
- 📧 Email support: support@macwrite.com

---

**Built with ❤️ using Go, PostgreSQL, and modern security practices.**
