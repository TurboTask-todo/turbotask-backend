# QuantumTask API

A comprehensive Go-based REST API for the QuantumTask productivity application, featuring AI-enhanced task management, real-time collaboration, and advanced analytics.

## 🚀 Features

- **Authentication & Authorization**: JWT-based auth with OAuth integration
- **AI-Enhanced Task Management**: Intelligent task creation and optimization
- **Real-time Collaboration**: WebSocket-powered live updates
- **Advanced Analytics**: Comprehensive reports and insights
- **Email Integration**: OTP authentication and notifications
- **Rate Limiting**: Comprehensive request throttling
- **Database Optimization**: High-performance connection pooling
- **Message Queuing**: RabbitMQ integration for background processing
- **Caching**: Redis-powered caching layer
- **Monitoring**: Built-in health checks and metrics

## 🏗️ Architecture

```
├── cmd/
│   └── server/          # Application entry point
├── internal/
│   ├── config/          # Configuration management
│   ├── database/        # Database connection & migrations
│   ├── handler/         # HTTP handlers
│   ├── middleware/      # HTTP middleware
│   ├── models/          # Data models
│   ├── repository/      # Data access layer
│   └── service/         # Business logic
├── pkg/
│   ├── ai/             # AI integration
│   ├── auth/           # Authentication utilities
│   ├── email/          # Email services
│   ├── oauth/          # OAuth providers
│   ├── queue/          # Message queue client
│   ├── redis/          # Redis client
│   ├── validation/     # Input validation
│   └── websocket/      # WebSocket infrastructure
└── migrations/         # Database migration files
```

## 🛠️ Tech Stack

- **Language**: Go 1.21+
- **Framework**: Gin Web Framework
- **Database**: PostgreSQL
- **Cache**: Redis
- **Message Queue**: RabbitMQ
- **Authentication**: JWT + OAuth (Google)
- **Real-time**: WebSocket
- **Deployment**: Docker + Render

## 🚀 Quick Start

### Local Development

1. **Clone the repository**
```bash
git clone <your-repo-url>
cd backend-go
```

2. **Install dependencies**
```bash
go mod download
```

3. **Set up environment variables**
```bash
cp .env.example .env
# Edit .env with your configuration
```

4. **Start dependencies with Docker**
```bash
# PostgreSQL
docker run --name postgres-dev -e POSTGRES_PASSWORD=password -e POSTGRES_DB=quantumtask -p 5432:5432 -d postgres:15

# Redis
docker run --name redis-dev -p 6379:6379 -d redis:7

# RabbitMQ (optional)
docker run --name rabbitmq-dev -p 5672:5672 -p 15672:15672 -d rabbitmq:3-management
```

5. **Run database migrations**
```bash
# Migrations are automatically applied on startup
go run cmd/server/main.go
```

6. **Start the server**
```bash
go run cmd/server/main.go
```

The API will be available at `http://localhost:8080`

### API Documentation

Visit `http://localhost:8080/docs` for interactive API documentation.

Key endpoints:
- **Health Check**: `GET /health`
- **Authentication**: `POST /api/v1/auth/*`
- **Projects**: `GET /api/v1/todo/projects`
- **Tasks**: `GET /api/v1/todo/todos`
- **AI Enhancement**: `POST /api/v1/ai/tasks`
- **WebSocket**: `WS /api/v1/ws/connect`

## 🚀 Deployment

### Deploy to Render

1. **Pre-deployment Check**
```bash
./scripts/deploy-check.sh
```

2. **Follow the comprehensive deployment guide**
See [DEPLOYMENT_GUIDE.md](./DEPLOYMENT_GUIDE.md) for detailed instructions.

3. **Quick Deploy with Render Blueprint**
```bash
# Push to GitHub first
git add .
git commit -m "Deploy to Render"
git push origin main

# Then use render.yaml for one-click deploy
```

### Environment Variables

Copy `.env.render.example` and configure these key variables:

```bash
# Required for production
JWT_SECRET=your-32-character-secret
DATABASE_URL=postgresql://...
REDIS_URL=redis://...

# OAuth (optional)
OAUTH_GOOGLE_CLIENT_ID=...
OAUTH_GOOGLE_CLIENT_SECRET=...

# Email (for OTP)
SMTP_USERNAME=...
SMTP_PASSWORD=...
```

## 📊 Monitoring

### Health Checks

- **Application**: `GET /health`
- **Database**: `GET /api/v1/monitoring/database/health`
- **AI Consumer**: `GET /api/v1/ai-monitoring/consumer/health`

### Metrics

- **Database Pool**: `GET /api/v1/monitoring/database/metrics`
- **Connection Status**: `GET /api/v1/monitoring/database/pool-status`
- **AI Metrics**: `GET /api/v1/ai-monitoring/consumer/metrics`

## 🧪 Testing

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run specific package tests
go test ./internal/service/...
```

### API Testing

Use the provided Postman collections:
- `postman/Todo_App_API.postman_collection.json`
- `postman/AI_Conversation_API.postman_collection.json`

## 🔧 Development

### Project Structure

- **cmd/**: Application entrypoints
- **internal/**: Private application code
- **pkg/**: Public packages (can be imported by other projects)
- **migrations/**: Database migration files
- **scripts/**: Utility scripts
- **postman/**: API testing collections

### Code Style

- Follow standard Go conventions
- Use `gofmt` and `golint`
- Write tests for new features
- Document public functions

### Database Migrations

Add new migrations to the `migrations/` directory:
```sql
-- migrations/001_create_users.sql
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);
```

## 🔐 Security

- **JWT Authentication**: Secure token-based auth
- **Rate Limiting**: Prevent abuse
- **CORS Protection**: Configure allowed origins
- **SQL Injection Protection**: Parameterized queries
- **Input Validation**: Comprehensive request validation
- **HTTPS Enforced**: Secure connections only

## 📈 Performance

- **Connection Pooling**: Optimized database connections
- **Redis Caching**: Fast data retrieval
- **Background Processing**: Queue-based task processing
- **WebSocket**: Efficient real-time updates
- **Database Optimization**: Indexed queries and efficient schemas

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## 📜 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🆘 Support

- **Documentation**: Check the [DEPLOYMENT_GUIDE.md](./DEPLOYMENT_GUIDE.md)
- **Issues**: Open a GitHub issue
- **Email**: Contact the development team

## 🎯 Roadmap

- [ ] GraphQL API support
- [ ] Microservices architecture
- [ ] Advanced AI features
- [ ] Mobile app integration
- [ ] Team collaboration features
- [ ] Advanced reporting dashboard

---

**QuantumTask API** - Empowering productivity with AI-enhanced task management 🚀