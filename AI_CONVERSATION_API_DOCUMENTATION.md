# AI Conversation History API - Complete Documentation

## Table of Contents

1. [Overview](#overview)
2. [Getting Started](#getting-started)
3. [Authentication](#authentication)
4. [Rate Limiting](#rate-limiting)
5. [Request/Response Format](#requestresponse-format)
6. [Error Handling](#error-handling)
7. [API Endpoints](#api-endpoints)
8. [Postman Collection](#postman-collection)
9. [Code Examples](#code-examples)
10. [Performance Features](#performance-features)
11. [Monitoring & Analytics](#monitoring--analytics)

## Overview

The AI Conversation History API provides a comprehensive system for managing AI conversations, interactions, and analytics. It features intelligent compression, Redis caching, RabbitMQ queuing, and real-time analytics.

### Key Features

- **Dynamic conversation management** with session tracking
- **Intelligent compression** for large AI responses
- **Multi-level caching** with Redis integration
- **Asynchronous processing** with RabbitMQ queues
- **Real-time analytics** and usage tracking
- **Feedback system** for AI interaction quality
- **Full-text search** across conversation history
- **Cost tracking** and budget management

### Base URL

```
http://localhost:8080/api/v1
```

## Getting Started

### 1. Prerequisites

- PostgreSQL database with migrations applied
- Redis server running (optional - falls back to in-memory cache)
- RabbitMQ server running (optional - uses mock client in development)
- Valid JWT authentication token

### 2. Quick Setup

1. **Run Database Migrations**:

   ```bash
   psql -d your_database -f Query/ai_conversation_migration.sql
   ```

2. **Import Postman Collection**:

   - Import `postman/AI_Conversation_API.postman_collection.json`
   - Import `postman/AI_Conversation_API.postman_environment.json`

3. **Authenticate**:
   ```bash
   # Login to get JWT token
   curl -X POST http://localhost:8080/api/v1/auth/login \
     -H "Content-Type: application/json" \
     -d '{"email":"user@example.com","password":"password"}'
   ```

## Authentication

All AI conversation endpoints require JWT authentication.

### Authentication Header

```http
Authorization: Bearer <your-jwt-token>
```

### Getting a JWT Token

1. **Register a new user**:

   ```http
   POST /api/v1/auth/register
   Content-Type: application/json

   {
     "email": "user@example.com",
     "password": "SecurePassword123!",
     "username": "testuser",
     "first_name": "Test",
     "last_name": "User"
   }
   ```

2. **Login to get tokens**:

   ```http
   POST /api/v1/auth/login
   Content-Type: application/json

   {
     "email": "user@example.com",
     "password": "SecurePassword123!"
   }
   ```

   **Response**:

   ```json
   {
     "success": true,
     "data": {
       "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
       "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
       "token_type": "Bearer",
       "expires_in": 900
     }
   }
   ```

## Rate Limiting

The API implements user-specific rate limiting to ensure fair usage:

| Endpoint Type    | Limit (per minute per user) |
| ---------------- | --------------------------- |
| Conversations    | 30 creations, 100 listings  |
| Interactions     | 60 AI requests              |
| Search           | 60 requests                 |
| Analytics        | 20 requests                 |
| Feedback         | 30 submissions              |
| Cache Operations | 5 invalidations, 3 warmings |

### Rate Limit Headers

```http
X-RateLimit-Limit: 60
X-RateLimit-Remaining: 45
X-RateLimit-Reset: 1642684800
```

## Request/Response Format

### Standard Response Structure

All API responses follow a consistent format:

```json
{
  "success": true|false,
  "data": <response_data>,
  "message": "Description of the result",
  "timestamp": "2024-01-15T10:30:00Z",
  "pagination": {  // Only for paginated responses
    "total": 150,
    "limit": 20,
    "offset": 0,
    "pages": 8
  }
}
```

### Error Response Structure

```json
{
  "success": false,
  "error": {
    "code": "ERROR_CODE",
    "message": "Human-readable error description"
  },
  "timestamp": "2024-01-15T10:30:00Z"
}
```

## Error Handling

### Common HTTP Status Codes

| Status Code | Description                             |
| ----------- | --------------------------------------- |
| 200         | Success                                 |
| 201         | Created                                 |
| 400         | Bad Request - Invalid input             |
| 401         | Unauthorized - Authentication required  |
| 403         | Forbidden - Insufficient permissions    |
| 404         | Not Found - Resource doesn't exist      |
| 429         | Too Many Requests - Rate limit exceeded |
| 500         | Internal Server Error                   |

### Error Codes

| Error Code                     | Description                   |
| ------------------------------ | ----------------------------- |
| `UNAUTHORIZED`                 | Authentication required       |
| `INVALID_REQUEST`              | Request validation failed     |
| `CONVERSATION_NOT_FOUND`       | Conversation doesn't exist    |
| `INTERACTION_NOT_FOUND`        | Interaction doesn't exist     |
| `RATE_LIMIT_EXCEEDED`          | Too many requests             |
| `CONVERSATION_CREATION_FAILED` | Failed to create conversation |
| `INTERACTION_CREATION_FAILED`  | Failed to create interaction  |

## API Endpoints

### Conversations

#### Create Conversation

```http
POST /api/v1/ai/conversations
Authorization: Bearer <token>
Content-Type: application/json

{
  "session_id": "unique-session-123",
  "title": "My AI Chat Session",
  "model_name": "gpt-4",
  "model_version": "2024-01-15",
  "conversation_type": "chat",
  "priority": 5,
  "metadata": {
    "source": "web_app",
    "environment": "production"
  },
  "tags": ["work", "research", "AI"]
}
```

**Response**: `201 Created`

```json
{
  "success": true,
  "data": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "user_id": "123e4567-e89b-12d3-a456-426614174000",
    "session_id": "unique-session-123",
    "title": "My AI Chat Session",
    "model_name": "gpt-4",
    "status": "active",
    "total_cost": 0.0,
    "interaction_count": 0,
    "created_at": "2024-01-15T10:30:00Z"
  },
  "message": "Conversation created successfully"
}
```

#### Get User Conversations

```http
GET /api/v1/ai/conversations?limit=20&offset=0&status=active&order_by=created_at&order=desc
Authorization: Bearer <token>
```

**Query Parameters**:

- `limit` (1-100): Number of results per page
- `offset`: Number of results to skip
- `status`: Filter by status (`active`, `archived`, `deleted`, `suspended`)
- `model`: Filter by model name
- `from_date`: Filter from date (YYYY-MM-DD)
- `to_date`: Filter to date (YYYY-MM-DD)
- `search`: Search in titles and content
- `tags`: Filter by tags (comma-separated)
- `order_by`: Sort field (`created_at`, `updated_at`, `total_cost`, `interaction_count`)
- `order`: Sort order (`asc`, `desc`)

**Response**: `200 OK`

```json
{
  "success": true,
  "data": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "title": "My AI Chat Session",
      "model_name": "gpt-4",
      "status": "active",
      "total_cost": 0.25,
      "interaction_count": 5,
      "last_interaction_at": "2024-01-15T11:30:00Z",
      "created_at": "2024-01-15T10:30:00Z"
    }
  ],
  "pagination": {
    "total": 25,
    "limit": 20,
    "offset": 0,
    "pages": 2
  }
}
```

#### Get Conversation with Interactions

```http
GET /api/v1/ai/conversations/{id}/interactions
Authorization: Bearer <token>
```

**Response**: `200 OK`

```json
{
  "success": true,
  "data": {
    "conversation": {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "title": "My AI Chat Session",
      "model_name": "gpt-4",
      "status": "active"
    },
    "interactions": [
      {
        "id": "interaction-id-1",
        "sequence_number": 1,
        "user_input": "Hello, how are you?",
        "ai_response": "I'm doing well, thank you for asking!",
        "input_token_count": 5,
        "output_token_count": 12,
        "total_cost": 0.000048,
        "processing_time_ms": 1250,
        "status": "completed",
        "created_at": "2024-01-15T10:31:00Z"
      }
    ]
  }
}
```

#### Update Conversation

```http
PUT /api/v1/ai/conversations/{id}
Authorization: Bearer <token>
Content-Type: application/json

{
  "title": "Updated Chat Session",
  "status": "active",
  "priority": 7,
  "metadata": {
    "updated_via": "api",
    "last_modified": "2024-01-15T10:35:00Z"
  },
  "tags": ["updated", "api"]
}
```

#### Search Conversations

```http
GET /api/v1/ai/conversations/search?query=machine learning&limit=10
Authorization: Bearer <token>
```

#### Archive Conversation

```http
POST /api/v1/ai/conversations/{id}/archive
Authorization: Bearer <token>
```

#### Delete Conversation

```http
DELETE /api/v1/ai/conversations/{id}
Authorization: Bearer <token>
```

### Interactions

#### Create AI Interaction

```http
POST /api/v1/ai/interactions
Authorization: Bearer <token>
Content-Type: application/json
X-Request-ID: unique-request-id

{
  "conversation_id": "550e8400-e29b-41d4-a716-446655440000",
  "user_input": "Explain how machine learning works",
  "model_name": "gpt-4",
  "model_version": "2024-01-15",
  "temperature": 0.7,
  "max_tokens": 2048,
  "top_p": 0.9,
  "frequency_penalty": 0.0,
  "presence_penalty": 0.0,
  "stop_sequences": [],
  "system_prompt": "You are a helpful AI assistant.",
  "priority": 5,
  "enable_compression": true,
  "enable_caching": true,
  "request_metadata": {
    "source": "web_app",
    "user_intent": "learning"
  }
}
```

**Response**: `201 Created`

```json
{
  "success": true,
  "data": {
    "interaction": {
      "id": "interaction-id-123",
      "conversation_id": "550e8400-e29b-41d4-a716-446655440000",
      "user_input": "Explain how machine learning works",
      "ai_response": "Machine learning is a subset of artificial intelligence...",
      "status": "completed",
      "input_token_count": 6,
      "output_token_count": 150,
      "total_cost": 0.000234,
      "processing_time_ms": 1500,
      "cache_hit": false,
      "created_at": "2024-01-15T10:35:00Z"
    },
    "queue_info": {
      "message_id": "msg-uuid-123",
      "queue_name": "ai.conversation.standard",
      "priority": 5,
      "estimated_wait": "1-2 min"
    },
    "cache_info": {
      "cache_key": "response:hash123",
      "cache_hit": false,
      "ttl": 1800
    }
  }
}
```

#### Get Interaction Details

```http
GET /api/v1/ai/interactions/{id}
Authorization: Bearer <token>
```

### Feedback

#### Submit Interaction Feedback

```http
POST /api/v1/ai/feedback
Authorization: Bearer <token>
Content-Type: application/json

{
  "interaction_id": "interaction-id-123",
  "overall_rating": 4,
  "accuracy_rating": 5,
  "helpfulness_rating": 4,
  "relevance_rating": 5,
  "feedback_type": "positive",
  "feedback_text": "Great explanation of machine learning concepts!",
  "improvement_suggestions": "Could include more examples",
  "tags": ["clear", "educational"],
  "session_quality": 4,
  "response_time_satisfaction": 5
}
```

#### Get Interaction Feedback

```http
GET /api/v1/ai/interactions/{interaction_id}/feedback
Authorization: Bearer <token>
```

### Analytics

#### Get User Analytics

```http
GET /api/v1/ai/analytics?from_date=2024-01-01&to_date=2024-01-31&group_by=day
Authorization: Bearer <token>
```

**Response**: `200 OK`

```json
{
  "success": true,
  "data": {
    "user_id": "123e4567-e89b-12d3-a456-426614174000",
    "date_range": {
      "from": "2024-01-01T00:00:00Z",
      "to": "2024-01-31T23:59:59Z"
    },
    "summary": {
      "total_interactions": 150,
      "total_cost": 2.45,
      "total_input_tokens": 12500,
      "total_output_tokens": 45000,
      "avg_processing_time": 1250,
      "success_rate": 98.5,
      "cache_hit_rate": 25.3,
      "unique_conversations": 25
    },
    "time_series": [
      {
        "timestamp": "2024-01-01T00:00:00Z",
        "interactions": 5,
        "cost": 0.08,
        "input_tokens": 250,
        "output_tokens": 850,
        "avg_processing_time_ms": 1100,
        "success_rate": 100.0,
        "cache_hit_rate": 20.0
      }
    ],
    "models": [
      {
        "model_name": "gpt-4",
        "interactions": 100,
        "cost": 1.8,
        "avg_cost_per_token": 0.00003,
        "success_rate": 99.0,
        "avg_response_time_ms": 1300
      }
    ]
  }
}
```

#### Get User Summary

```http
GET /api/v1/ai/analytics/summary
Authorization: Bearer <token>
```

### Cache Management

#### Invalidate User Cache

```http
POST /api/v1/ai/cache/invalidate
Authorization: Bearer <token>
```

#### Warm User Cache

```http
POST /api/v1/ai/cache/warm
Authorization: Bearer <token>
```

### System Health

#### System Health Check

```http
GET /api/v1/ai/health
```

**Response**: `200 OK`

```json
{
  "success": true,
  "data": {
    "redis": "healthy",
    "queue": "healthy",
    "database": "healthy",
    "timestamp": "2024-01-15T10:30:00Z",
    "version": "1.0.0"
  }
}
```

## Postman Collection

### Quick Start

1. **Import the Collection**:

   - Download `postman/AI_Conversation_API.postman_collection.json`
   - Import into Postman

2. **Import the Environment**:

   - Download `postman/AI_Conversation_API.postman_environment.json`
   - Import and select the environment

3. **Set Environment Variables**:

   - Update `base_url` if different from `localhost:8080`
   - Update `test_user_email` and `test_user_password`

4. **Authenticate**:

   - Run the "Login User" request
   - JWT tokens will be automatically saved to environment variables

5. **Test the Flow**:
   - Create Conversation → Create Interaction → Submit Feedback → View Analytics

### Environment Variables

| Variable          | Description             | Auto-Set             |
| ----------------- | ----------------------- | -------------------- |
| `base_url`        | API base URL            | No                   |
| `access_token`    | JWT access token        | Yes (after login)    |
| `conversation_id` | Current conversation ID | Yes (after creating) |
| `interaction_id`  | Current interaction ID  | Yes (after creating) |

## Code Examples

### JavaScript/Node.js

```javascript
const axios = require("axios");

class AIConversationAPI {
  constructor(baseURL, accessToken) {
    this.baseURL = baseURL;
    this.accessToken = accessToken;
    this.headers = {
      Authorization: `Bearer ${accessToken}`,
      "Content-Type": "application/json",
    };
  }

  // Create a new conversation
  async createConversation(sessionId, title, modelName) {
    const response = await axios.post(
      `${this.baseURL}/api/v1/ai/conversations`,
      {
        session_id: sessionId,
        title: title,
        model_name: modelName,
        conversation_type: "chat",
        priority: 5,
        enable_compression: true,
      },
      { headers: this.headers }
    );

    return response.data;
  }

  // Create an AI interaction
  async createInteraction(conversationId, userInput, options = {}) {
    const response = await axios.post(
      `${this.baseURL}/api/v1/ai/interactions`,
      {
        conversation_id: conversationId,
        user_input: userInput,
        model_name: options.modelName || "gpt-4",
        temperature: options.temperature || 0.7,
        max_tokens: options.maxTokens || 2048,
        enable_compression: true,
        enable_caching: true,
        priority: options.priority || 5,
      },
      { headers: this.headers }
    );

    return response.data;
  }

  // Get user analytics
  async getAnalytics(fromDate, toDate, groupBy = "day") {
    const response = await axios.get(`${this.baseURL}/api/v1/ai/analytics`, {
      params: { from_date: fromDate, to_date: toDate, group_by: groupBy },
      headers: this.headers,
    });

    return response.data;
  }
}

// Usage
const api = new AIConversationAPI("http://localhost:8080", "your-jwt-token");

(async () => {
  // Create conversation
  const conversation = await api.createConversation(
    "session-123",
    "My AI Chat",
    "gpt-4"
  );

  // Create interaction
  const interaction = await api.createInteraction(
    conversation.data.id,
    "Hello, how are you?"
  );

  // Get analytics
  const analytics = await api.getAnalytics("2024-01-01", "2024-01-31");

  console.log("Analytics:", analytics.data.summary);
})();
```

### Python

```python
import requests
from datetime import datetime
import json

class AIConversationAPI:
    def __init__(self, base_url, access_token):
        self.base_url = base_url
        self.headers = {
            'Authorization': f'Bearer {access_token}',
            'Content-Type': 'application/json'
        }

    def create_conversation(self, session_id, title, model_name):
        """Create a new AI conversation"""
        url = f"{self.base_url}/api/v1/ai/conversations"
        data = {
            'session_id': session_id,
            'title': title,
            'model_name': model_name,
            'conversation_type': 'chat',
            'priority': 5,
            'enable_compression': True
        }

        response = requests.post(url, json=data, headers=self.headers)
        return response.json()

    def create_interaction(self, conversation_id, user_input, **options):
        """Create an AI interaction"""
        url = f"{self.base_url}/api/v1/ai/interactions"
        data = {
            'conversation_id': conversation_id,
            'user_input': user_input,
            'model_name': options.get('model_name', 'gpt-4'),
            'temperature': options.get('temperature', 0.7),
            'max_tokens': options.get('max_tokens', 2048),
            'enable_compression': True,
            'enable_caching': True,
            'priority': options.get('priority', 5)
        }

        response = requests.post(url, json=data, headers=self.headers)
        return response.json()

    def get_analytics(self, from_date, to_date, group_by='day'):
        """Get user analytics"""
        url = f"{self.base_url}/api/v1/ai/analytics"
        params = {
            'from_date': from_date,
            'to_date': to_date,
            'group_by': group_by
        }

        response = requests.get(url, params=params, headers=self.headers)
        return response.json()

# Usage
api = AIConversationAPI('http://localhost:8080', 'your-jwt-token')

# Create conversation
conversation = api.create_conversation(
    session_id='session-123',
    title='My AI Chat',
    model_name='gpt-4'
)

# Create interaction
interaction = api.create_interaction(
    conversation_id=conversation['data']['id'],
    user_input='Hello, how are you?'
)

# Get analytics
analytics = api.get_analytics('2024-01-01', '2024-01-31')
print(f"Total interactions: {analytics['data']['summary']['total_interactions']}")
```

### cURL Examples

```bash
# Login to get token
ACCESS_TOKEN=$(curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"password"}' | \
  jq -r '.data.access_token')

# Create conversation
CONVERSATION_ID=$(curl -s -X POST http://localhost:8080/api/v1/ai/conversations \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "session_id": "session-123",
    "title": "My AI Chat",
    "model_name": "gpt-4",
    "conversation_type": "chat"
  }' | jq -r '.data.id')

# Create interaction
curl -X POST http://localhost:8080/api/v1/ai/interactions \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
    \"conversation_id\": \"$CONVERSATION_ID\",
    \"user_input\": \"Hello, how are you?\",
    \"model_name\": \"gpt-4\",
    \"temperature\": 0.7,
    \"enable_caching\": true
  }"

# Get analytics
curl -X GET "http://localhost:8080/api/v1/ai/analytics?from_date=2024-01-01&to_date=2024-01-31" \
  -H "Authorization: Bearer $ACCESS_TOKEN"
```

## Performance Features

### Intelligent Compression

The API automatically compresses large AI responses:

- **None**: Content < 1KB (no compression overhead)
- **Gzip**: Content 1KB-64KB (balanced speed/ratio)
- **LZ4**: Content 64KB-1MB (faster decompression)
- **Brotli**: Content > 1MB (best compression ratio)

### Multi-Level Caching

```
Cache Hierarchy:
1. Application Cache (in-memory) - 100ms access
2. Redis Cache (distributed) - 1-5ms access
3. Database (persistent) - 10-50ms access

Cache Types:
- Response Caching: By input hash (30 min TTL)
- Conversation Caching: By conversation ID (1 hour TTL)
- Analytics Caching: By date range (2 hour TTL)
- User Summary Caching: By user ID (30 min TTL)
```

### Queue Management

```
Queue Routing by Priority:
- High Priority (8-10): ai.conversation.high_priority (~30s processing)
- Standard (4-7): ai.conversation.standard (~1-2 min processing)
- Low Priority (1-3): ai.conversation.low_priority (~5-10 min processing)

Auto-scaling Rules:
- Scale up workers when queue depth > 100 messages
- Scale down workers when queue depth < 10 messages
- Minimum 2 workers, maximum 20 workers per queue
```

### Cost Optimization

- **Response Deduplication**: Identical requests return cached responses (no AI API cost)
- **Compression**: Reduces storage costs by 60-80%
- **Token Estimation**: Accurate cost prediction before API calls
- **Budget Controls**: Per-user daily/monthly limits

## Monitoring & Analytics

### Key Metrics

1. **Performance Metrics**:

   - Response time (P50, P95, P99)
   - Queue processing time
   - Cache hit rates
   - Database query performance

2. **Business Metrics**:

   - API usage per user/model
   - Cost per interaction
   - Success/error rates
   - User engagement patterns

3. **System Health**:
   - Database connection status
   - Redis availability
   - RabbitMQ queue depth
   - Worker health status

### Alerts & Notifications

- High error rates (> 5%)
- Slow response times (> 5s P95)
- Queue backup (> 500 messages)
- Budget threshold exceeded (80% of limit)
- System component failures

### Dashboard Endpoints

- `/api/v1/ai/health` - System health overview
- `/api/v1/ai/analytics/summary` - User summary dashboard
- `/api/v1/ai/analytics` - Detailed analytics with time series data

## Best Practices

### API Usage

1. **Authentication**: Always include valid JWT tokens
2. **Rate Limiting**: Respect rate limits and implement exponential backoff
3. **Error Handling**: Handle all error responses gracefully
4. **Request IDs**: Include unique request IDs for tracing
5. **Pagination**: Use appropriate page sizes (10-50 items)

### Performance Optimization

1. **Enable Caching**: Set `enable_caching: true` for repeated queries
2. **Use Compression**: Enable compression for large content
3. **Batch Operations**: Group related API calls when possible
4. **Monitor Usage**: Track analytics to optimize costs
5. **Cache Management**: Warm cache before high-usage periods

### Security

1. **Token Management**: Rotate JWT tokens regularly
2. **Input Validation**: Validate all inputs on client side
3. **Sensitive Data**: Never log sensitive conversation content
4. **HTTPS**: Always use HTTPS in production
5. **API Keys**: Store API credentials securely

---

## Support

For technical support or questions about the AI Conversation History API:

- **Documentation**: This comprehensive guide
- **Postman Collection**: Pre-built collection with examples
- **Health Endpoint**: `/api/v1/ai/health` for system status
- **Error Logging**: Detailed error responses with codes

The API is designed for production use with enterprise-grade features including comprehensive monitoring, caching, and scaling capabilities.
