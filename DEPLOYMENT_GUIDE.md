# QuantumTask API Deployment Guide for Render

## Prerequisites

1. **GitHub Repository**: Ensure your code is pushed to GitHub
2. **Render Account**: Create account at [render.com](https://render.com)
3. **Environment Configuration**: Prepare production environment variables

## Step 1: Prepare Your Repository

### 1.1 Update Go Module
```bash
cd backend-go
go mod tidy
go mod vendor  # Optional: for faster builds
```

### 1.2 Create/Verify Required Files

Ensure these files exist in your `backend-go/` directory:
- `Dockerfile` ✅
- `render.yaml` ✅ 
- `.dockerignore` ✅
- `go.mod` and `go.sum`

### 1.3 Update Configuration for Production

Create a production configuration check in your `internal/config/config.go`:

```go
func LoadForProduction() (*Config, error) {
    cfg := &Config{}
    
    // Server Configuration
    cfg.Server.Port = getEnvAsInt("PORT", 8080)
    cfg.Server.GinMode = getEnv("GIN_MODE", "release")
    
    // Database from Render's DATABASE_URL
    cfg.Database.URL = getEnv("DATABASE_URL", "")
    if cfg.Database.URL == "" {
        return nil, fmt.Errorf("DATABASE_URL is required in production")
    }
    
    // Redis from Render's REDIS_URL
    redisURL := getEnv("REDIS_URL", "")
    if redisURL != "" {
        cfg.Redis.Addr = parseRedisURL(redisURL)
    }
    
    // JWT Configuration
    cfg.JWT.Secret = getEnv("JWT_SECRET", "")
    if cfg.JWT.Secret == "" {
        return nil, fmt.Errorf("JWT_SECRET is required in production")
    }
    
    return cfg, nil
}
```

## Step 2: Deploy Services on Render

### 2.1 Deploy PostgreSQL Database

1. **Go to Render Dashboard** → **New** → **PostgreSQL**
2. **Configure Database:**
   - **Name**: `quantumtask-postgres`
   - **Database Name**: `quantumtask`  
   - **User**: `quantumtask`
   - **Region**: Choose closest to your users
   - **Plan**: Start with **Starter** ($7/month)

3. **Note the Connection Details:**
   - Database URL will be auto-generated
   - Save the connection string for later

### 2.2 Deploy Redis Cache

1. **Go to Render Dashboard** → **New** → **Redis**
2. **Configure Redis:**
   - **Name**: `quantumtask-redis`
   - **Plan**: **Starter** ($7/month)
   - **Region**: Same as PostgreSQL

### 2.3 Setup External RabbitMQ (Optional)

Since Render doesn't provide managed RabbitMQ, you have options:

**Option A: CloudAMQP (Recommended)**
1. Sign up at [CloudAMQP](https://www.cloudamqp.com/)
2. Create a **Little Lemur** (Free) or **Tough Tiger** ($13/month) plan
3. Get the AMQP URL: `amqp://username:password@host:5672/vhost`

**Option B: Use Mock Client**
Your code already has fallback to mock client, so you can start without RabbitMQ.

## Step 3: Deploy Web Service

### 3.1 Create Web Service

1. **Go to Render Dashboard** → **New** → **Web Service**
2. **Connect Repository:**
   - Select your GitHub repository
   - Choose the repository containing your Go code
   - **Root Directory**: `backend-go` (if your Go code is in a subdirectory)

### 3.2 Configure Build Settings

**Basic Configuration:**
- **Name**: `quantumtask-api`
- **Region**: Same as your databases
- **Branch**: `main` (or your production branch)
- **Runtime**: `Go`

**Build & Deploy:**
- **Build Command**: `go build -o main ./cmd/server`
- **Start Command**: `./main`

**Pricing Plan:**
- **Starter**: $7/month (512MB RAM, 0.1 CPU)
- **Standard**: $25/month (2GB RAM, 1 CPU) - Recommended for production

### 3.3 Configure Environment Variables

Add these environment variables in Render:

**Application Settings:**
```bash
APP_NAME=QuantumTask API
APP_VERSION=1.0.0
APP_ENV=production
GIN_MODE=release
PORT=8080  # Render auto-assigns this
```

**Database Configuration:**
```bash
DATABASE_URL=postgresql://username:password@host:5432/database
# This will be auto-filled when you connect the PostgreSQL service
```

**Redis Configuration:**
```bash
REDIS_URL=redis://username:password@host:6379
# This will be auto-filled when you connect the Redis service
```

**Security Configuration:**
```bash
JWT_SECRET=your-super-secret-jwt-key-here-min-32-chars
JWT_ACCESS_TOKEN_DURATION=15m
JWT_REFRESH_TOKEN_DURATION=7d
```

**OAuth Configuration (if using):**
```bash
OAUTH_GOOGLE_CLIENT_ID=your-google-client-id
OAUTH_GOOGLE_CLIENT_SECRET=your-google-client-secret  
OAUTH_GOOGLE_REDIRECT_URL=https://your-app.onrender.com/api/v1/auth/google/callback
```

**Email Configuration:**
```bash
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USERNAME=your-email@gmail.com
SMTP_PASSWORD=your-app-password
EMAIL_FROM=your-email@gmail.com
```

**Rate Limiting:**
```bash
RATE_LIMIT_REQUESTS_PER_MINUTE=100
```

**External Services:**
```bash
QUEUE_URL=amqp://username:password@host:5672/vhost
AI_EXTERNAL_URL=https://your-ai-service.com
MIDDLEWARE_IO_API_KEY=your-middleware-io-key  # Optional
```

### 3.4 Connect Services

In the **Environment** tab:
1. **Connect PostgreSQL**: 
   - Click **Add from service**
   - Select your PostgreSQL database
   - Variable name: `DATABASE_URL`

2. **Connect Redis**:
   - Click **Add from service** 
   - Select your Redis instance
   - Variable name: `REDIS_URL`

## Step 4: Database Migration

### 4.1 Run Migrations

After deployment, you may need to run database migrations:

**Option A: Migration on Startup (Recommended)**
Your code already includes migration check in `main.go`:
```go
if err := db.RunMigrations(); err != nil {
    log.Fatalf("Database migration check failed: %v", err)
}
```

**Option B: Manual Migration**
1. Connect to your database using the connection string
2. Run your SQL migration files manually

### 4.2 Seed Data (Optional)

If you have seed data, create a separate script or endpoint to populate initial data.

## Step 5: Domain Configuration

### 5.1 Custom Domain (Optional)

1. **In Render Dashboard** → **Settings** → **Custom Domains**
2. **Add Domain**: `api.yourdomain.com`
3. **Configure DNS**: Add CNAME record pointing to Render URL
4. **SSL Certificate**: Render provides free SSL automatically

### 5.2 Update OAuth Redirect URLs

Update your OAuth providers (Google, etc.) with new domain:
- **Development**: `http://localhost:8080/api/v1/auth/google/callback`
- **Production**: `https://your-app.onrender.com/api/v1/auth/google/callback`

## Step 6: Monitoring & Maintenance

### 6.1 Health Checks

Render will automatically use your `/health` endpoint for health checks.

### 6.2 Logs

View logs in Render Dashboard → **Logs** tab:
```bash
# Your app logs will show:
# Starting QuantumTask API server on port 8080
# Environment: production
# Version: 1.0.0
```

### 6.3 Scaling

**Auto-scaling**: Available on Standard+ plans
**Manual scaling**: Upgrade plan or add more services

## Step 7: Production Checklist

### 7.1 Security
- ✅ Strong JWT secret (32+ characters)
- ✅ HTTPS enabled (automatic with Render)
- ✅ Rate limiting configured
- ✅ CORS properly configured
- ✅ Environment variables secured

### 7.2 Performance
- ✅ Database connection pooling
- ✅ Redis caching enabled
- ✅ Gin in release mode
- ✅ Proper error handling

### 7.3 Monitoring
- ✅ Health check endpoint working
- ✅ Logs properly configured
- ✅ Database metrics available
- ✅ APM integration (if using Middleware.io)

## Step 8: Testing Deployment

### 8.1 Verify Endpoints

Test your main endpoints:

```bash
# Health check
curl https://your-app.onrender.com/health

# API documentation
curl https://your-app.onrender.com/docs

# Test authentication
curl -X POST https://your-app.onrender.com/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"testpass123"}'
```

### 8.2 Database Connection

Verify database connectivity:
```bash
curl https://your-app.onrender.com/api/v1/monitoring/database/health
```

### 8.3 Performance Testing

Use tools like:
- **Apache Bench**: `ab -n 1000 -c 10 https://your-app.onrender.com/health`
- **Postman**: Import your API collection and run tests

## Troubleshooting

### Common Issues:

**1. Build Failures:**
- Check Go version compatibility
- Verify `go.mod` and `go.sum` are up to date
- Check build logs for missing dependencies

**2. Database Connection Issues:**
- Verify `DATABASE_URL` format
- Check database service is running
- Verify firewall/network connectivity

**3. High Memory Usage:**
- Monitor database connection pool size
- Check for memory leaks in goroutines
- Consider upgrading to Standard plan

**4. Slow Response Times:**
- Enable Redis caching
- Optimize database queries
- Check network latency between services

### Support Resources:
- **Render Documentation**: https://render.com/docs
- **Render Community**: https://community.render.com
- **GitHub Issues**: Create issues in your repository

## Cost Estimation

**Minimum Production Setup:**
- Web Service (Starter): $7/month
- PostgreSQL (Starter): $7/month  
- Redis (Starter): $7/month
- **Total**: ~$21/month

**Recommended Production Setup:**
- Web Service (Standard): $25/month
- PostgreSQL (Standard): $20/month
- Redis (Standard): $15/month
- **Total**: ~$60/month

## Next Steps

1. **Set up CI/CD**: Configure GitHub Actions for automated deployments
2. **Add monitoring**: Integrate with services like DataDog or New Relic
3. **Backup strategy**: Configure automated database backups
4. **Load balancing**: Add multiple instances for high availability
5. **CDN**: Use CloudFlare or similar for static assets

Your QuantumTask API should now be successfully deployed on Render! 🚀
