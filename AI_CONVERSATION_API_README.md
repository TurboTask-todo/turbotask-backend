# AI Conversation API System

## Overview

A comprehensive AI conversation history system with advanced features including compression, Redis caching, RabbitMQ queuing, and dynamic scaling capabilities. This system provides professional-grade APIs for managing AI conversations, interactions, feedback, and analytics.

## 🚀 Features

### Core Functionality

- **Dynamic Conversation Management**: Create, update, delete, and archive AI conversation sessions
- **Intelligent Compression**: Automatic compression for large AI responses using gzip, lz4, and brotli
- **Redis Caching**: Multi-level caching for improved performance
- **RabbitMQ Queuing**: Asynchronous processing with priority queues
- **Real-time Analytics**: Comprehensive usage tracking and analytics
- **Full-text Search**: Search through conversation history
- **Feedback System**: User feedback and rating system for AI responses

### Performance Features

- **Optimized Database Queries**: Efficient pagination, filtering, and indexing
- **Intelligent Caching**: Response caching, conversation caching, and analytics caching
- **Queue Management**: Priority-based message processing with auto-scaling
- **Compression Strategy**: Smart compression based on content size
- **Partitioned Archives**: Automatic archiving of old interactions

## 📁 Project Structure

```
.
├── cmd/server/main.go                          # Main application entry point with AI routes
├── internal/
│   ├── entity/ai_conversation.go               # Data models and entities
│   ├── repository/ai_conversation_repository.go # Database layer
│   ├── service/ai_conversation_service.go      # Business logic layer
│   ├── handler/ai_conversation_handler.go      # HTTP handlers
│   └── config/config.go                        # Configuration (updated)
├── pkg/
│   ├── compression/compressor.go               # Compression utilities
│   ├── redis/client.go                         # Redis client implementation
│   └── queue/client.go                         # RabbitMQ client implementation
└── Query/
    ├── ai_conversation_schema.sql              # Complete database schema
    └── ai_conversation_migration.sql           # Database migration script
```

## 🛠 Installation & Setup

### 1. Database Setup

First, ensure you have PostgreSQL installed and the auth schema is set up:

```bash
# Run the auth migration first (if not already done)
psql -d your_database -f Query/auth.sql

# Run the AI conversation migration
psql -d your_database -f Query/ai_conversation_migration.sql
```

### 2. Environment Variables

Add these environment variables to your configuration:

```bash
# Redis Configuration
REDIS_ADDR=localhost:6379
REDIS_PASSWORD=
REDIS_DB=0

# RabbitMQ Configuration
RABBITMQ_URL=amqp://guest:guest@localhost:5672/
QUEUE_EXCHANGE_NAME=ai.conversation.direct
QUEUE_MAX_RETRIES=3
```

### 3. Start the Application

```bash
go run cmd/server/main.go
```

The AI conversation API will be available at `http://localhost:8080/api/v1/ai/`

## 📚 API Documentation

### Authentication

All AI conversation endpoints require authentication. Include the JWT token in the Authorization header:

```
Authorization: Bearer <your-jwt-token>
```

### Core Endpoints

#### Conversations

```http
POST   /api/v1/ai/conversations                    # Create new conversation
GET    /api/v1/ai/conversations                    # Get user conversations (paginated)
GET    /api/v1/ai/conversations/search             # Search conversations
GET    /api/v1/ai/conversations/:id                # Get conversation details
GET    /api/v1/ai/conversations/:id/interactions   # Get conversation with all interactions
PUT    /api/v1/ai/conversations/:id                # Update conversation
DELETE /api/v1/ai/conversations/:id                # Delete conversation
POST   /api/v1/ai/conversations/:id/archive        # Archive conversation
```

#### Interactions

```http
POST   /api/v1/ai/interactions                     # Create new AI interaction
GET    /api/v1/ai/interactions/:id                 # Get interaction details
```

#### Feedback

```http
POST   /api/v1/ai/feedback                         # Submit feedback
GET    /api/v1/ai/interactions/:id/feedback        # Get interaction feedback
```

#### Analytics

```http
GET    /api/v1/ai/analytics                        # Get user analytics
GET    /api/v1/ai/analytics/summary                # Get user summary
```

#### Cache Management

```http
POST   /api/v1/ai/cache/invalidate                 # Invalidate user cache
POST   /api/v1/ai/cache/warm                       # Warm user cache
```

#### System Health

```http
GET    /api/v1/ai/health                           # System health check
```

### Example API Calls

#### Create a Conversation

```bash
curl -X POST http://localhost:8080/api/v1/ai/conversations \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "session_id": "unique-session-123",
    "title": "My AI Chat Session",
    "model_name": "gpt-4",
    "conversation_type": "chat",
    "priority": 5
  }'
```

#### Create an AI Interaction

```bash
curl -X POST http://localhost:8080/api/v1/ai/interactions \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "conversation_id": "conversation-uuid",
    "user_input": "Hello, how are you?",
    "model_name": "gpt-4",
    "temperature": 0.7,
    "max_tokens": 2048,
    "enable_compression": true,
    "enable_caching": true,
    "priority": 5
  }'
```

#### Get User Analytics

```bash
curl -X GET "http://localhost:8080/api/v1/ai/analytics?from_date=2024-01-01&to_date=2024-01-31" \
  -H "Authorization: Bearer <token>"
```

## 🗄️ Database Schema

### Main Tables

#### `ai_conversations`

Stores conversation metadata and aggregated metrics.

- Tracks total costs, token counts, and interaction counts
- Supports conversation types: chat, completion, embedding
- Includes caching metadata and user-defined tags

#### `ai_interactions`

Individual AI request/response pairs with compression support.

- Smart compression for large responses (gzip, lz4, brotli)
- Hash-based deduplication and caching
- Comprehensive performance metrics
- Queue integration tracking

#### `cache_entries`

Redis cache management and tracking.

- Cache key pattern management
- Hit/miss ratio tracking
- TTL and expiration management

#### `queue_messages`

RabbitMQ message tracking and processing.

- Message lifecycle tracking
- Priority-based processing
- Retry logic with exponential backoff

#### `ai_usage_analytics`

Real-time usage analytics aggregated by time periods.

- Hourly and daily metrics
- Cost tracking per model
- Performance metrics
- Cache hit rates

### Optimized Indexes

- 25+ specialized indexes for query performance
- Composite indexes for common query patterns
- Partial indexes for active data
- GIN indexes for JSONB and array fields

## ⚡ Performance Features

### Intelligent Compression

```go
// Automatic compression selection based on content size
if size < 1KB:    compression = 'none'
if size < 64KB:   compression = 'gzip'   // Balance of speed/ratio
if size < 1MB:    compression = 'lz4'    // Faster decompression
else:             compression = 'brotli' // Best compression ratio
```

### Multi-Level Caching

```
Cache Hierarchy:
1. Application Cache (in-memory)
2. Redis Cache (distributed)
3. Database (persistent)

Cache Types:
- Response caching (by input hash)
- Conversation caching (by conversation ID)
- Analytics caching (by date range)
- User summary caching
```

### Queue Management

```
Queue Priorities:
- High Priority (8-10):   ai.conversation.high_priority
- Standard (4-7):         ai.conversation.standard
- Low Priority (1-3):     ai.conversation.low_priority

Auto-scaling:
- Scale up: >100 messages in queue
- Scale down: <10 messages in queue
- Cooldown periods to prevent thrashing
```

## 📊 Monitoring & Analytics

### System Health Metrics

- Database connection status
- Redis availability
- RabbitMQ queue health
- Processing latency
- Error rates
- Cache hit rates

### User Analytics

- Total interactions and costs
- Token usage by model
- Success rates and error patterns
- Peak usage patterns
- Cache efficiency
- Average response times

### Real-time Dashboards

Access comprehensive metrics at:

- `/api/v1/ai/health` - System health
- `/api/v1/ai/analytics/summary` - User summary
- `/api/v1/ai/analytics` - Detailed analytics

## 🚀 Advanced Features

### Queue Processing

```go
// Automatic queue routing based on priority
func getQueueName(priority int) string {
    if priority >= 8:  return "ai.conversation.high_priority"
    if priority >= 4:  return "ai.conversation.standard"
    return "ai.conversation.low_priority"
}
```

### Response Deduplication

```go
// Hash-based caching for identical requests
inputHash := sha256(userInput + modelName + temperature + maxTokens)
if cachedResponse := getFromCache(inputHash); cachedResponse != nil {
    return cachedResponse // No API cost
}
```

### Automatic Archiving

```sql
-- Archive interactions older than 90 days
SELECT archive_old_interactions(90);
```

## 🔧 Configuration

### Rate Limits (per minute per user)

- Conversations: 30 creations, 100 listings
- Interactions: 60 AI requests
- Search: 60 requests
- Analytics: 20 requests
- Cache operations: 5 invalidations, 3 warmings

### Default Settings

- Cache TTL: 30 minutes - 2 hours
- Queue message TTL: 24 hours
- Archive threshold: 90 days
- Max request size: 10MB
- Connection pool: 25 max connections

## 🛡️ Security Features

- JWT authentication required for all endpoints
- Rate limiting per user and per IP
- Request size validation
- SQL injection prevention
- CORS configuration
- Security headers

## 📈 Cost Tracking

### Token-based Pricing

```go
// Cost calculation per model
costs := map[string]CostStructure{
    "gpt-4": {
        InputCost:  0.00003,  // per token
        OutputCost: 0.00006,  // per token
    },
    "gpt-3.5-turbo": {
        InputCost:  0.0000015,
        OutputCost: 0.000002,
    },
}
```

### Budget Management

- Daily and monthly budget limits
- Real-time cost tracking
- Budget alerts and throttling
- Cost optimization through caching

## 🔄 Async Processing Flow

```
1. User Request → HTTP Handler
2. Validate & Create Interaction Record
3. Check Cache (input hash)
4. If cached: Return immediately
5. If not cached: Queue for processing
6. RabbitMQ → Worker picks up message
7. Process with AI model
8. Update interaction record
9. Cache response for future use
10. Trigger analytics update
```

## 📝 Sample Response Structure

```json
{
  "success": true,
  "data": {
    "interaction": {
      "id": "interaction-uuid",
      "conversation_id": "conversation-uuid",
      "user_input": "Hello, how are you?",
      "ai_response": "I'm doing well, thank you for asking!",
      "status": "completed",
      "input_token_count": 5,
      "output_token_count": 12,
      "total_cost": 0.000048,
      "processing_time_ms": 1250,
      "cache_hit": false,
      "created_at": "2024-01-15T10:30:00Z"
    },
    "queue_info": {
      "message_id": "msg-uuid",
      "queue_name": "ai.conversation.standard",
      "priority": 5,
      "estimated_wait": "1-2 min"
    },
    "cache_info": {
      "cache_key": "response:hash123",
      "cache_hit": false,
      "ttl": 1800
    }
  },
  "message": "Interaction created successfully",
  "timestamp": "2024-01-15T10:30:00Z"
}
```

## 🚨 Error Handling

The API returns consistent error responses:

```json
{
  "success": false,
  "error": {
    "code": "CONVERSATION_NOT_FOUND",
    "message": "Conversation not found"
  },
  "timestamp": "2024-01-15T10:30:00Z"
}
```

Common error codes:

- `UNAUTHORIZED` - Authentication required
- `CONVERSATION_NOT_FOUND` - Conversation doesn't exist
- `INVALID_REQUEST` - Request validation failed
- `RATE_LIMIT_EXCEEDED` - Too many requests
- `INTERACTION_CREATION_FAILED` - Failed to create interaction

## 📚 Additional Resources

- **Database Schema**: `Query/ai_conversation_schema.sql`
- **Migration Script**: `Query/ai_conversation_migration.sql`
- **API Documentation**: `/docs` endpoint
- **Health Monitoring**: `/api/v1/ai/health`

## 🤝 Contributing

When extending the system:

1. Follow the existing layered architecture
2. Add appropriate rate limiting
3. Include comprehensive error handling
4. Update database migrations
5. Add monitoring and analytics
6. Include caching strategies
7. Document API changes

---

This AI conversation system provides enterprise-grade functionality with professional optimization for performance, scalability, and cost management.
