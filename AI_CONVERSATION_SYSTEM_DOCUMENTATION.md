# AI Response History System - Complete Architecture

## Overview

This document provides a comprehensive architecture for storing AI conversation history with advanced features including compression, Redis caching, RabbitMQ queuing, and dynamic scaling capabilities.

## System Architecture

```
┌─────────────────┐    ┌──────────────┐    ┌─────────────────┐
│   Client App    │────│  API Gateway │────│  Queue Service  │
└─────────────────┘    └──────────────┘    └─────────────────┘
                              │                       │
                              ▼                       ▼
┌─────────────────┐    ┌──────────────┐    ┌─────────────────┐
│  Redis Cache    │◄───│  PostgreSQL  │◄───│   RabbitMQ      │
└─────────────────┘    └──────────────┘    └─────────────────┘
                              │                       │
                              ▼                       ▼
                    ┌──────────────┐    ┌─────────────────┐
                    │  Analytics   │    │  AI Processors  │
                    │   Service    │    │   (Workers)     │
                    └──────────────┘    └─────────────────┘
```

## Database Schema Design

### Core Tables

#### 1. `ai_conversations`

Stores conversation metadata and aggregated metrics.

```sql
Key Features:
- UUID primary keys for global uniqueness
- JSON metadata for flexible conversation context
- Cache keys for Redis integration
- Cost tracking with decimal precision
- Status management with constraints
```

#### 2. `ai_interactions`

Individual AI request/response pairs with compression support.

```sql
Key Features:
- Dual storage: TEXT for small responses, BYTEA for compressed large responses
- Multiple compression algorithms (gzip, lz4, brotli)
- Hash-based deduplication and caching
- Comprehensive performance metrics
- Queue integration tracking
```

#### 3. `cache_entries`

Redis cache management and tracking.

```sql
Key Features:
- Cache key pattern management
- Hit/miss ratio tracking
- TTL and expiration management
- Size-based cache optimization
```

#### 4. `queue_messages`

RabbitMQ message tracking and processing.

```sql
Key Features:
- Message lifecycle tracking
- Priority-based processing
- Retry logic with exponential backoff
- Dead letter queue integration
```

## Data Model Specifications

### Request Body Structure

```json
{
  "user_id": "uuid",
  "session_id": "string",
  "conversation_id": "uuid", // optional, for continuing conversations
  "model": {
    "name": "gpt-4",
    "version": "2024-01-15",
    "temperature": 0.7,
    "max_tokens": 2048,
    "top_p": 0.9
  },
  "input": {
    "user_message": "string",
    "system_prompt": "string", // optional
    "context": {} // optional metadata
  },
  "processing": {
    "priority": 5, // 1-10 scale
    "cache_enabled": true,
    "compression_enabled": true,
    "async": true
  },
  "metadata": {
    "client_info": {},
    "request_source": "web|mobile|api",
    "trace_id": "string"
  }
}
```

### Response Structure

```json
{
  "interaction_id": "uuid",
  "conversation_id": "uuid",
  "status": "completed|failed|processing",
  "response": {
    "ai_message": "string",
    "model_used": "gpt-4",
    "finish_reason": "stop|length|content_filter"
  },
  "metrics": {
    "input_tokens": 150,
    "output_tokens": 300,
    "processing_time_ms": 2500,
    "queue_time_ms": 150,
    "total_latency_ms": 2650,
    "cache_hit": false,
    "compression_used": "gzip",
    "cost": {
      "input_cost": 0.000045,
      "output_cost": 0.00009,
      "total_cost": 0.000135
    }
  },
  "queue_info": {
    "message_id": "string",
    "queue_name": "ai.conversation.standard",
    "priority": 5
  }
}
```

## Compression Strategy

### Intelligent Compression Selection

```sql
-- Compression logic in application layer
IF response_size < 1KB THEN
    store_as_text = true
    compression = 'none'
ELSIF response_size < 64KB THEN
    compression = 'gzip'  -- Good balance of speed/ratio
ELSIF response_size < 1MB THEN
    compression = 'lz4'   -- Faster decompression
ELSE
    compression = 'brotli' -- Best compression ratio
```

### Compression Implementation

```python
# Python example for compression handling
import gzip
import lz4.frame
import brotli

def compress_response(content: str, method: str) -> bytes:
    content_bytes = content.encode('utf-8')

    if method == 'gzip':
        return gzip.compress(content_bytes, compresslevel=6)
    elif method == 'lz4':
        return lz4.frame.compress(content_bytes)
    elif method == 'brotli':
        return brotli.compress(content_bytes, quality=6)
    else:
        return content_bytes

def decompress_response(data: bytes, method: str) -> str:
    if method == 'gzip':
        return gzip.decompress(data).decode('utf-8')
    elif method == 'lz4':
        return lz4.frame.decompress(data).decode('utf-8')
    elif method == 'brotli':
        return brotli.decompress(data).decode('utf-8')
    else:
        return data.decode('utf-8')
```

## Redis Cache Management

### Cache Key Patterns

```
conversation:{user_id}:{session_id}      # Full conversation data
interaction:{interaction_id}             # Individual responses
user_summary:{user_id}:{date}           # Daily analytics
model_response:{input_hash}             # Cached responses by input
queue_status:{queue_name}               # Queue monitoring
user_tokens:{user_id}:{model}:{date}    # Token usage tracking
```

### Cache Configuration

```python
# Redis configuration example
CACHE_CONFIG = {
    'conversation:*': {
        'ttl': 7200,      # 2 hours
        'compress': True,
        'max_size': '2MB'
    },
    'interaction:*': {
        'ttl': 3600,      # 1 hour
        'compress': True,
        'max_size': '1MB'
    },
    'model_response:*': {
        'ttl': 1800,      # 30 minutes
        'compress': True,
        'max_size': '2MB'
    },
    'user_summary:*': {
        'ttl': 86400,     # 24 hours
        'compress': False,
        'max_size': '64KB'
    }
}
```

### Cache Warming Strategy

```python
# Cache warming implementation
async def warm_cache_for_user(user_id: str):
    # Recent conversations
    conversations = await get_recent_conversations(user_id, limit=10)
    for conv in conversations:
        cache_key = f"conversation:{user_id}:{conv.session_id}"
        await redis.setex(cache_key, 7200, serialize_conversation(conv))

    # Frequently accessed responses
    common_responses = await get_common_responses_for_user(user_id)
    for response in common_responses:
        cache_key = f"model_response:{response.input_hash}"
        await redis.setex(cache_key, 1800, response.ai_response)
```

## RabbitMQ Queue Management

### Queue Architecture

```
High Priority Queue (ai.conversation.high_priority)
├── Max consumers: 20
├── Scale up threshold: 50 messages
├── Processing timeout: 3 minutes
└── Dead letter → ai.dlx

Standard Queue (ai.conversation.standard)
├── Max consumers: 50
├── Scale up threshold: 200 messages
├── Processing timeout: 5 minutes
└── Dead letter → ai.dlx

Low Priority Queue (ai.conversation.low_priority)
├── Max consumers: 10
├── Scale up threshold: 500 messages
├── Processing timeout: 10 minutes
└── Dead letter → ai.dlx

Analytics Queue (ai.analytics.processing)
├── Max consumers: 5
├── Batch processing enabled
└── Dead letter → ai.dlx
```

### Message Routing

```python
# Message routing logic
def get_queue_name(priority: int, user_tier: str) -> str:
    if priority >= 8 or user_tier == 'premium':
        return 'ai.conversation.high_priority'
    elif priority >= 4:
        return 'ai.conversation.standard'
    else:
        return 'ai.conversation.low_priority'

# Message publishing
async def publish_ai_request(request_data: dict):
    queue_name = get_queue_name(
        request_data.get('priority', 5),
        request_data.get('user_tier', 'standard')
    )

    message = {
        'interaction_id': str(uuid.uuid4()),
        'user_id': request_data['user_id'],
        'conversation_id': request_data.get('conversation_id'),
        'payload': request_data,
        'timestamp': datetime.utcnow().isoformat(),
        'retry_count': 0
    }

    await rabbitmq.publish(
        exchange='ai.conversation.direct',
        routing_key=queue_name.split('.')[-1],
        body=json.dumps(message),
        properties=pika.BasicProperties(
            priority=request_data.get('priority', 5),
            message_id=message['interaction_id'],
            timestamp=int(time.time()),
            headers={'user_id': request_data['user_id']}
        )
    )
```

### Auto-scaling Logic

```python
# Consumer auto-scaling
async def auto_scale_consumers():
    for queue in await get_active_queues():
        metrics = await get_queue_metrics(queue.name)

        # Scale up conditions
        if (metrics.queue_depth > queue.scale_up_threshold and
            metrics.active_consumers < queue.max_consumers and
            not_in_cooldown(queue.name, 'scale_up')):

            await scale_up_consumers(queue.name, 1)
            await record_scaling_event(queue.name, 'scale_up',
                f"Queue depth: {metrics.queue_depth}")

        # Scale down conditions
        elif (metrics.queue_depth < queue.scale_down_threshold and
              metrics.active_consumers > queue.min_consumers and
              not_in_cooldown(queue.name, 'scale_down')):

            await scale_down_consumers(queue.name, 1)
            await record_scaling_event(queue.name, 'scale_down',
                f"Queue depth: {metrics.queue_depth}")

# Worker health monitoring
async def monitor_worker_health():
    for worker in await get_active_workers():
        if worker.last_heartbeat < datetime.utcnow() - timedelta(minutes=5):
            await mark_worker_unhealthy(worker.id)
            await spawn_replacement_worker(worker.queue_name)
```

## Performance Optimization

### Best Practices

#### 1. Database Optimization

```sql
-- Partitioning strategy for large tables
CREATE TABLE ai_interactions_2024_q1 PARTITION OF ai_interactions
    FOR VALUES FROM ('2024-01-01') TO ('2024-04-01');

-- Optimized indexes for common queries
CREATE INDEX CONCURRENTLY idx_ai_interactions_user_recent
    ON ai_interactions(user_id, created_at DESC)
    WHERE status = 'completed';

-- Partial indexes for active data
CREATE INDEX CONCURRENTLY idx_ai_conversations_active
    ON ai_conversations(user_id, updated_at DESC)
    WHERE status = 'active';
```

#### 2. Query Optimization

```sql
-- Use LATERAL joins for correlated subqueries
SELECT c.*, recent.last_response
FROM ai_conversations c
LEFT JOIN LATERAL (
    SELECT ai_response as last_response
    FROM ai_interactions
    WHERE conversation_id = c.id
    ORDER BY sequence_number DESC
    LIMIT 1
) recent ON true
WHERE c.user_id = $1;

-- Use CTEs for complex aggregations
WITH user_metrics AS (
    SELECT user_id,
           SUM(total_cost) as total_spent,
           COUNT(*) as total_interactions
    FROM ai_interactions
    WHERE created_at >= NOW() - INTERVAL '30 days'
    GROUP BY user_id
)
SELECT * FROM user_metrics WHERE total_spent > 10.00;
```

#### 3. Caching Strategy

```python
# Multi-level caching
async def get_conversation(conversation_id: str):
    # L1: Application cache
    if conversation_id in app_cache:
        return app_cache[conversation_id]

    # L2: Redis cache
    cache_key = f"conversation:{conversation_id}"
    cached = await redis.get(cache_key)
    if cached:
        conversation = deserialize(cached)
        app_cache[conversation_id] = conversation
        return conversation

    # L3: Database
    conversation = await db.get_conversation(conversation_id)

    # Populate caches
    await redis.setex(cache_key, 3600, serialize(conversation))
    app_cache[conversation_id] = conversation

    return conversation
```

## Dynamic Scaling Configuration

### Horizontal Scaling

```yaml
# Kubernetes example for AI workers
apiVersion: apps/v1
kind: Deployment
metadata:
  name: ai-conversation-workers
spec:
  replicas: 3
  template:
    spec:
      containers:
        - name: ai-worker
          image: ai-conversation-worker:latest
          env:
            - name: RABBITMQ_URL
              value: "amqp://rabbitmq:5672"
            - name: REDIS_URL
              value: "redis://redis:6379"
          resources:
            requests:
              cpu: 500m
              memory: 1Gi
            limits:
              cpu: 2000m
              memory: 4Gi
---
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: ai-workers-hpa
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: ai-conversation-workers
  minReplicas: 3
  maxReplicas: 50
  metrics:
    - type: External
      external:
        metric:
          name: rabbitmq_queue_messages
        target:
          type: AverageValue
          averageValue: "100"
```

### Database Scaling

```python
# Database connection pooling
DATABASE_CONFIG = {
    'read_replicas': [
        'postgresql://user:pass@read-replica-1:5432/aidb',
        'postgresql://user:pass@read-replica-2:5432/aidb',
    ],
    'write_primary': 'postgresql://user:pass@primary:5432/aidb',
    'pool_size': 20,
    'max_overflow': 50,
    'pool_timeout': 30,
    'pool_recycle': 3600
}

# Read/write splitting
async def get_conversation(conversation_id: str):
    # Use read replica for queries
    return await read_db.query(
        "SELECT * FROM ai_conversations WHERE id = %s",
        conversation_id
    )

async def create_interaction(interaction_data: dict):
    # Use primary for writes
    return await write_db.execute(
        "INSERT INTO ai_interactions (...) VALUES (...)",
        interaction_data
    )
```

## Monitoring and Alerting

### Key Metrics

```python
# Prometheus metrics
METRICS = {
    'ai_requests_total': Counter('ai_requests_total', ['queue', 'status']),
    'ai_processing_time': Histogram('ai_processing_time_seconds', ['model']),
    'ai_queue_depth': Gauge('ai_queue_depth', ['queue']),
    'ai_cache_hit_rate': Gauge('ai_cache_hit_rate', ['cache_type']),
    'ai_cost_total': Counter('ai_cost_total', ['user_id', 'model']),
    'ai_error_rate': Gauge('ai_error_rate', ['error_type'])
}

# Health check endpoint
async def health_check():
    checks = {
        'database': await check_database_health(),
        'redis': await check_redis_health(),
        'rabbitmq': await check_rabbitmq_health(),
        'ai_models': await check_ai_models_health()
    }

    overall_health = all(checks.values())

    return {
        'status': 'healthy' if overall_health else 'unhealthy',
        'checks': checks,
        'timestamp': datetime.utcnow().isoformat()
    }
```

### Alerting Rules

```yaml
# Prometheus alerting rules
groups:
  - name: ai_conversation_alerts
    rules:
      - alert: HighQueueDepth
        expr: ai_queue_depth > 1000
        for: 5m
        annotations:
          summary: "AI queue depth is high"
          description: "Queue {{ $labels.queue }} has {{ $value }} messages"

      - alert: LowCacheHitRate
        expr: ai_cache_hit_rate < 0.7
        for: 10m
        annotations:
          summary: "Cache hit rate is low"
          description: "Cache hit rate is {{ $value | humanizePercentage }}"

      - alert: HighErrorRate
        expr: rate(ai_requests_total{status="failed"}[5m]) > 0.05
        for: 2m
        annotations:
          summary: "High AI request error rate"
          description: "Error rate is {{ $value | humanizePercentage }}"
```

## Security Considerations

### Data Protection

```python
# Data encryption at rest
def encrypt_sensitive_data(data: str, key: bytes) -> str:
    from cryptography.fernet import Fernet
    f = Fernet(key)
    return f.encrypt(data.encode()).decode()

def decrypt_sensitive_data(encrypted_data: str, key: bytes) -> str:
    from cryptography.fernet import Fernet
    f = Fernet(key)
    return f.decrypt(encrypted_data.encode()).decode()

# PII handling
async def store_interaction(interaction_data: dict):
    # Remove or encrypt PII before storage
    sanitized_data = sanitize_pii(interaction_data)

    # Store with encryption for sensitive fields
    if contains_sensitive_data(sanitized_data):
        sanitized_data['user_input'] = encrypt_sensitive_data(
            sanitized_data['user_input'],
            encryption_key
        )

    await db.insert_interaction(sanitized_data)
```

### Access Control

```sql
-- Row-level security
CREATE POLICY user_data_isolation ON ai_conversations
    FOR ALL TO app_user
    USING (user_id = current_setting('app.current_user_id')::uuid);

ALTER TABLE ai_conversations ENABLE ROW LEVEL SECURITY;
```

## Cost Optimization

### Usage Tracking

```python
# Cost tracking and budgets
async def track_usage_and_cost(user_id: str, interaction_data: dict):
    cost = calculate_interaction_cost(interaction_data)

    # Update user's daily usage
    await update_daily_usage(user_id, cost)

    # Check budget limits
    daily_usage = await get_daily_usage(user_id)
    user_limits = await get_user_limits(user_id)

    if daily_usage.total_cost > user_limits.daily_budget:
        await throttle_user_requests(user_id)
        await send_budget_alert(user_id, daily_usage.total_cost)

# Cost optimization through caching
async def get_ai_response(input_hash: str, model: str, params: dict):
    # Check cache first
    cache_key = f"model_response:{input_hash}:{model}"
    cached_response = await redis.get(cache_key)

    if cached_response:
        # No API cost for cached responses
        return {
            'response': cached_response,
            'cost': 0.0,
            'cache_hit': True
        }

    # Make API call if not cached
    response = await call_ai_model(model, params)
    cost = calculate_api_cost(response.usage)

    # Cache for future use
    await redis.setex(cache_key, 1800, response.content)

    return {
        'response': response.content,
        'cost': cost,
        'cache_hit': False
    }
```

This comprehensive architecture provides a robust, scalable, and efficient system for managing AI conversation history with advanced features for compression, caching, queuing, and dynamic scaling. The system is designed to handle high throughput while maintaining cost efficiency and providing detailed analytics and monitoring capabilities.
