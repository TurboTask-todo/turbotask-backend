-- ===================================================================
-- OPTIMIZED QUERIES FOR AI CONVERSATION SYSTEM
-- ===================================================================
-- High-performance queries for common operations with caching support
-- and RabbitMQ queue management
-- ===================================================================

-- ===================================================================
-- 1. USER CONVERSATION QUERIES
-- ===================================================================

-- Get user's recent conversations with caching support
-- Performance: Uses partial index on active conversations and lateral join
SELECT 
    c.id,
    c.session_id,
    c.title,
    c.model_name,
    c.total_cost,
    c.interaction_count,
    c.created_at,
    c.cache_key,
    recent_interaction.last_response_preview,
    recent_interaction.last_interaction_at,
    CASE 
        WHEN c.cache_key IS NOT NULL 
        AND EXISTS (
            SELECT 1 FROM cache_entries 
            WHERE cache_key = c.cache_key 
            AND expires_at > NOW()
        ) THEN 'cached'
        ELSE 'fresh'
    END as cache_status
FROM ai_conversations c
LEFT JOIN LATERAL (
    SELECT 
        created_at as last_interaction_at,
        CASE 
            WHEN ai_response IS NOT NULL THEN LEFT(ai_response, 200)
            WHEN ai_response_compressed IS NOT NULL THEN 
                LEFT(convert_from(ai_response_compressed, 'UTF8'), 200)
        END as last_response_preview
    FROM ai_interactions 
    WHERE conversation_id = c.id 
    AND status = 'completed'
    ORDER BY sequence_number DESC 
    LIMIT 1
) recent_interaction ON true
WHERE c.user_id = $1 
AND c.status = 'active'
ORDER BY c.updated_at DESC
LIMIT $2 OFFSET $3;

-- ===================================================================
-- 2. CONVERSATION DETAIL QUERIES  
-- ===================================================================

-- Get full conversation with compressed response handling
-- Performance: Single query with optimized decompression
WITH conversation_interactions AS (
    SELECT 
        i.id,
        i.sequence_number,
        i.user_input,
        -- Smart decompression based on type
        CASE 
            WHEN i.ai_response IS NOT NULL THEN i.ai_response
            WHEN i.ai_response_compressed IS NOT NULL THEN 
                CASE i.output_compression_type
                    WHEN 'gzip' THEN convert_from(decompress(i.ai_response_compressed, 'gzip'), 'UTF8')
                    WHEN 'lz4' THEN convert_from(decompress(i.ai_response_compressed, 'lz4'), 'UTF8')
                    WHEN 'brotli' THEN convert_from(decompress(i.ai_response_compressed, 'brotli'), 'UTF8')
                    ELSE convert_from(i.ai_response_compressed, 'UTF8')
                END
        END as ai_response,
        i.input_token_count,
        i.output_token_count,
        i.total_cost,
        i.processing_time_ms,
        i.cache_hit,
        i.status,
        i.created_at,
        i.model_name,
        i.temperature
    FROM ai_interactions i
    WHERE i.conversation_id = $1
    ORDER BY i.sequence_number
)
SELECT 
    c.*,
    COALESCE(json_agg(
        json_build_object(
            'id', ci.id,
            'sequence_number', ci.sequence_number,
            'user_input', ci.user_input,
            'ai_response', ci.ai_response,
            'input_token_count', ci.input_token_count,
            'output_token_count', ci.output_token_count,
            'total_cost', ci.total_cost,
            'processing_time_ms', ci.processing_time_ms,
            'cache_hit', ci.cache_hit,
            'status', ci.status,
            'created_at', ci.created_at,
            'model_name', ci.model_name,
            'temperature', ci.temperature
        ) ORDER BY ci.sequence_number
    ) FILTER (WHERE ci.id IS NOT NULL), '[]'::json) as interactions
FROM ai_conversations c
LEFT JOIN conversation_interactions ci ON true
WHERE c.id = $1
GROUP BY c.id, c.user_id, c.session_id, c.title, c.model_name, c.model_version, 
         c.conversation_type, c.total_input_tokens, c.total_output_tokens, 
         c.total_cost, c.interaction_count, c.status, c.metadata, c.created_at, c.updated_at;

-- ===================================================================
-- 3. ANALYTICS AND REPORTING QUERIES
-- ===================================================================

-- User daily analytics with cost breakdown and caching metrics
-- Performance: Optimized aggregation with proper indexing
SELECT 
    DATE(i.created_at) as date,
    i.model_name,
    COUNT(*) as total_interactions,
    COUNT(*) FILTER (WHERE i.cache_hit = true) as cached_interactions,
    ROUND(COUNT(*) FILTER (WHERE i.cache_hit = true) * 100.0 / COUNT(*), 2) as cache_hit_rate_percent,
    
    -- Token metrics
    SUM(i.input_token_count) as total_input_tokens,
    SUM(i.output_token_count) as total_output_tokens,
    ROUND(AVG(i.output_token_count::DECIMAL / NULLIF(i.input_token_count, 0)), 2) as avg_output_input_ratio,
    
    -- Cost metrics
    SUM(i.total_cost) as total_cost,
    AVG(i.total_cost) as avg_cost_per_interaction,
    SUM(i.input_cost) as total_input_cost,
    SUM(i.output_cost) as total_output_cost,
    
    -- Performance metrics
    ROUND(AVG(i.processing_time_ms)) as avg_processing_time_ms,
    ROUND(AVG(i.queue_time_ms)) as avg_queue_time_ms,
    ROUND(AVG(i.total_latency_ms)) as avg_total_latency_ms,
    
    -- Quality metrics
    COUNT(*) FILTER (WHERE i.status = 'completed') as successful_interactions,
    COUNT(*) FILTER (WHERE i.status = 'failed') as failed_interactions,
    ROUND(COUNT(*) FILTER (WHERE i.status = 'completed') * 100.0 / COUNT(*), 2) as success_rate_percent,
    
    -- Peak usage
    MODE() WITHIN GROUP (ORDER BY EXTRACT(HOUR FROM i.created_at)) as peak_hour
FROM ai_interactions i
WHERE i.user_id = $1 
AND i.created_at >= $2 
AND i.created_at < $3
GROUP BY DATE(i.created_at), i.model_name
ORDER BY date DESC, total_cost DESC;

-- Model performance comparison (last 30 days)
-- Performance: Efficient aggregation for model comparison
SELECT 
    i.model_name,
    COUNT(*) as total_interactions,
    COUNT(DISTINCT i.user_id) as unique_users,
    
    -- Performance metrics
    ROUND(AVG(i.processing_time_ms)) as avg_processing_time_ms,
    ROUND(PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY i.processing_time_ms)) as p95_processing_time_ms,
    ROUND(AVG(i.total_latency_ms)) as avg_total_latency_ms,
    
    -- Token efficiency
    ROUND(AVG(i.output_token_count::DECIMAL / NULLIF(i.input_token_count, 0)), 3) as avg_output_input_ratio,
    SUM(i.input_token_count) as total_input_tokens,
    SUM(i.output_token_count) as total_output_tokens,
    
    -- Cost analysis
    ROUND(AVG(i.total_cost), 6) as avg_cost_per_interaction,
    ROUND(SUM(i.total_cost), 2) as total_cost,
    ROUND(AVG(i.total_cost / NULLIF(i.output_token_count, 0) * 1000), 6) as cost_per_1k_output_tokens,
    
    -- Quality metrics
    ROUND(COUNT(*) FILTER (WHERE i.status = 'completed') * 100.0 / COUNT(*), 2) as success_rate_percent,
    COUNT(*) FILTER (WHERE i.retry_count > 0) as interactions_with_retries,
    ROUND(AVG(i.retry_count), 2) as avg_retry_count,
    
    -- Cache performance
    ROUND(COUNT(*) FILTER (WHERE i.cache_hit = true) * 100.0 / COUNT(*), 2) as cache_hit_rate_percent,
    
    -- User satisfaction (if feedback exists)
    ROUND(AVG(f.overall_rating), 2) as avg_user_rating
FROM ai_interactions i
LEFT JOIN ai_interaction_feedback f ON i.id = f.interaction_id
WHERE i.created_at >= NOW() - INTERVAL '30 days'
AND i.status IN ('completed', 'failed')
GROUP BY i.model_name
HAVING COUNT(*) >= 10  -- Only models with significant usage
ORDER BY total_interactions DESC;

-- ===================================================================
-- 4. QUEUE MANAGEMENT QUERIES
-- ===================================================================

-- RabbitMQ queue monitoring with performance metrics
-- Performance: Optimized for real-time queue monitoring
SELECT 
    q.queue_name,
    q.status,
    COUNT(*) as message_count,
    COUNT(*) FILTER (WHERE q.priority >= 8) as high_priority_count,
    COUNT(*) FILTER (WHERE q.priority <= 3) as low_priority_count,
    
    -- Age analysis
    ROUND(AVG(EXTRACT(EPOCH FROM (NOW() - q.created_at)))) as avg_age_seconds,
    ROUND(MAX(EXTRACT(EPOCH FROM (NOW() - q.created_at)))) as max_age_seconds,
    MIN(q.created_at) as oldest_message_at,
    
    -- Processing metrics
    ROUND(AVG(EXTRACT(EPOCH FROM (q.completed_at - q.started_at)))) FILTER (WHERE q.completed_at IS NOT NULL) as avg_processing_time_seconds,
    COUNT(*) FILTER (WHERE q.retry_count > 0) as retried_messages,
    ROUND(AVG(q.retry_count)) as avg_retry_count,
    
    -- Error analysis
    COUNT(*) FILTER (WHERE q.status = 'failed') as failed_messages,
    COUNT(*) FILTER (WHERE q.status = 'dead_letter') as dead_letter_messages,
    
    -- Recent throughput (last hour)
    COUNT(*) FILTER (WHERE q.created_at >= NOW() - INTERVAL '1 hour') as messages_last_hour,
    COUNT(*) FILTER (WHERE q.completed_at >= NOW() - INTERVAL '1 hour') as completed_last_hour
FROM queue_messages q
WHERE q.created_at >= NOW() - INTERVAL '24 hours'
GROUP BY q.queue_name, q.status
ORDER BY q.queue_name, 
    CASE q.status 
        WHEN 'processing' THEN 1 
        WHEN 'pending' THEN 2 
        WHEN 'failed' THEN 3 
        ELSE 4 
    END;

-- Queue processing efficiency by worker
SELECT 
    q.worker_id,
    q.queue_name,
    COUNT(*) as total_processed,
    COUNT(*) FILTER (WHERE q.status = 'completed') as successful_count,
    COUNT(*) FILTER (WHERE q.status = 'failed') as failed_count,
    ROUND(COUNT(*) FILTER (WHERE q.status = 'completed') * 100.0 / COUNT(*), 2) as success_rate_percent,
    
    -- Performance metrics
    ROUND(AVG(EXTRACT(EPOCH FROM (q.completed_at - q.started_at)))) as avg_processing_time_seconds,
    ROUND(AVG(EXTRACT(EPOCH FROM (q.started_at - q.created_at)))) as avg_queue_wait_time_seconds,
    
    -- Throughput analysis
    ROUND(COUNT(*)::DECIMAL / EXTRACT(EPOCH FROM (MAX(q.completed_at) - MIN(q.started_at)) / 3600), 2) as messages_per_hour,
    
    MIN(q.started_at) as first_processed_at,
    MAX(q.completed_at) as last_completed_at
FROM queue_messages q
WHERE q.worker_id IS NOT NULL
AND q.started_at >= NOW() - INTERVAL '24 hours'
GROUP BY q.worker_id, q.queue_name
HAVING COUNT(*) >= 5  -- Only workers with significant activity
ORDER BY success_rate_percent DESC, messages_per_hour DESC;

-- ===================================================================
-- 5. CACHE OPTIMIZATION QUERIES
-- ===================================================================

-- Cache performance analysis with hit/miss ratios
SELECT 
    SPLIT_PART(ce.cache_key, ':', 1) as cache_type,
    COUNT(*) as total_entries,
    SUM(ce.hit_count) as total_hits,
    SUM(ce.miss_count) as total_misses,
    ROUND(SUM(ce.hit_count)::DECIMAL / NULLIF(SUM(ce.hit_count + ce.miss_count), 0), 4) as overall_hit_rate,
    
    -- Size analysis
    ROUND(SUM(ce.size_bytes)::DECIMAL / 1048576, 2) as total_size_mb,
    ROUND(AVG(ce.size_bytes)) as avg_entry_size_bytes,
    MAX(ce.size_bytes) as max_entry_size_bytes,
    
    -- Access patterns
    ROUND(AVG(EXTRACT(EPOCH FROM (NOW() - ce.last_accessed_at)) / 3600), 2) as avg_hours_since_access,
    COUNT(*) FILTER (WHERE ce.last_accessed_at >= NOW() - INTERVAL '1 hour') as recently_accessed,
    COUNT(*) FILTER (WHERE ce.expires_at IS NOT NULL AND ce.expires_at < NOW()) as expired_entries,
    
    -- Efficiency metrics
    ROUND(SUM(ce.hit_count * ce.size_bytes)::DECIMAL / NULLIF(SUM(ce.size_bytes), 0), 2) as weighted_hit_rate,
    COUNT(*) FILTER (WHERE ce.hit_count = 0) as never_hit_entries
FROM cache_entries ce
WHERE ce.created_at >= NOW() - INTERVAL '7 days'
GROUP BY SPLIT_PART(ce.cache_key, ':', 1)
ORDER BY total_size_mb DESC;

-- Identify cache entries for potential eviction
-- Performance: Finds least valuable cache entries
SELECT 
    ce.cache_key,
    ce.size_bytes,
    ce.hit_count,
    ce.miss_count,
    CASE WHEN (ce.hit_count + ce.miss_count) > 0 
         THEN ROUND(ce.hit_count::DECIMAL / (ce.hit_count + ce.miss_count), 4)
         ELSE 0 END as hit_rate,
    EXTRACT(EPOCH FROM (NOW() - ce.last_accessed_at)) / 3600 as hours_since_access,
    EXTRACT(EPOCH FROM (NOW() - ce.created_at)) / 3600 as age_hours,
    
    -- Efficiency score (higher = more valuable to keep)
    CASE 
        WHEN ce.hit_count = 0 THEN 0
        ELSE ROUND(
            (ce.hit_count::DECIMAL / NULLIF(ce.hit_count + ce.miss_count, 0)) * 
            (ce.hit_count::DECIMAL / GREATEST(EXTRACT(EPOCH FROM (NOW() - ce.last_accessed_at)) / 3600, 1)) *
            LOG(GREATEST(ce.hit_count, 1))
        , 4)
    END as efficiency_score,
    
    ce.expires_at
FROM cache_entries ce
WHERE ce.created_at >= NOW() - INTERVAL '30 days'
ORDER BY efficiency_score ASC, hours_since_access DESC
LIMIT 100;

-- ===================================================================
-- 6. REAL-TIME MONITORING QUERIES
-- ===================================================================

-- System health dashboard query
-- Performance: Single query for dashboard metrics
WITH recent_interactions AS (
    SELECT 
        COUNT(*) as total_interactions_24h,
        COUNT(*) FILTER (WHERE status = 'completed') as successful_interactions_24h,
        COUNT(*) FILTER (WHERE cache_hit = true) as cached_interactions_24h,
        ROUND(AVG(processing_time_ms)) as avg_processing_time_ms,
        ROUND(AVG(total_latency_ms)) as avg_total_latency_ms,
        SUM(total_cost) as total_cost_24h,
        COUNT(DISTINCT user_id) as active_users_24h,
        COUNT(DISTINCT conversation_id) as active_conversations_24h
    FROM ai_interactions
    WHERE created_at >= NOW() - INTERVAL '24 hours'
),
queue_status AS (
    SELECT 
        COUNT(*) FILTER (WHERE status = 'pending') as pending_messages,
        COUNT(*) FILTER (WHERE status = 'processing') as processing_messages,
        COUNT(*) FILTER (WHERE status = 'failed') as failed_messages,
        ROUND(AVG(EXTRACT(EPOCH FROM (NOW() - created_at)))) FILTER (WHERE status = 'pending') as avg_queue_wait_seconds
    FROM queue_messages
    WHERE created_at >= NOW() - INTERVAL '1 hour'
),
cache_status AS (
    SELECT 
        COUNT(*) as total_cache_entries,
        ROUND(SUM(size_bytes)::DECIMAL / 1048576, 2) as total_cache_size_mb,
        COUNT(*) FILTER (WHERE expires_at IS NOT NULL AND expires_at < NOW()) as expired_entries,
        ROUND(SUM(hit_count)::DECIMAL / NULLIF(SUM(hit_count + miss_count), 0), 4) as overall_cache_hit_rate
    FROM cache_entries
    WHERE created_at >= NOW() - INTERVAL '24 hours'
)
SELECT 
    -- Interaction metrics
    ri.total_interactions_24h,
    ri.successful_interactions_24h,
    ROUND(ri.successful_interactions_24h * 100.0 / NULLIF(ri.total_interactions_24h, 0), 2) as success_rate_percent,
    ri.cached_interactions_24h,
    ROUND(ri.cached_interactions_24h * 100.0 / NULLIF(ri.total_interactions_24h, 0), 2) as cache_hit_rate_percent,
    ri.avg_processing_time_ms,
    ri.avg_total_latency_ms,
    ri.total_cost_24h,
    ri.active_users_24h,
    ri.active_conversations_24h,
    
    -- Queue metrics
    qs.pending_messages,
    qs.processing_messages,
    qs.failed_messages,
    qs.avg_queue_wait_seconds,
    
    -- Cache metrics
    cs.total_cache_entries,
    cs.total_cache_size_mb,
    cs.expired_entries,
    cs.overall_cache_hit_rate,
    
    -- Calculated health scores (0-100)
    CASE 
        WHEN ri.successful_interactions_24h * 100.0 / NULLIF(ri.total_interactions_24h, 0) >= 95 THEN 100
        WHEN ri.successful_interactions_24h * 100.0 / NULLIF(ri.total_interactions_24h, 0) >= 90 THEN 80
        WHEN ri.successful_interactions_24h * 100.0 / NULLIF(ri.total_interactions_24h, 0) >= 85 THEN 60
        ELSE 40
    END as success_rate_health_score,
    
    CASE 
        WHEN qs.avg_queue_wait_seconds <= 5 THEN 100
        WHEN qs.avg_queue_wait_seconds <= 15 THEN 80
        WHEN qs.avg_queue_wait_seconds <= 30 THEN 60
        ELSE 40
    END as queue_health_score,
    
    CASE 
        WHEN cs.overall_cache_hit_rate >= 0.8 THEN 100
        WHEN cs.overall_cache_hit_rate >= 0.6 THEN 80
        WHEN cs.overall_cache_hit_rate >= 0.4 THEN 60
        ELSE 40
    END as cache_health_score
FROM recent_interactions ri
CROSS JOIN queue_status qs
CROSS JOIN cache_status cs;

-- ===================================================================
-- 7. SEARCH AND FILTERING QUERIES
-- ===================================================================

-- Advanced conversation search with full-text search
-- Performance: Uses GIN indexes for JSONB and text search
SELECT 
    c.id,
    c.session_id,
    c.title,
    c.model_name,
    c.total_cost,
    c.created_at,
    ts_rank(search_vector.tsv, plainto_tsquery($2)) as relevance_score,
    highlight(search_content, plainto_tsquery($2)) as highlighted_content
FROM ai_conversations c
JOIN LATERAL (
    SELECT 
        to_tsvector('english', 
            COALESCE(c.title, '') || ' ' ||
            string_agg(COALESCE(i.user_input, ''), ' ') || ' ' ||
            string_agg(
                CASE 
                    WHEN i.ai_response IS NOT NULL THEN i.ai_response
                    WHEN i.ai_response_compressed IS NOT NULL THEN 
                        convert_from(i.ai_response_compressed, 'UTF8')
                    ELSE ''
                END, ' '
            )
        ) as tsv,
        COALESCE(c.title, '') || ' ' ||
        string_agg(COALESCE(i.user_input, ''), ' ') || ' ' ||
        string_agg(
            CASE 
                WHEN i.ai_response IS NOT NULL THEN LEFT(i.ai_response, 500)
                WHEN i.ai_response_compressed IS NOT NULL THEN 
                    LEFT(convert_from(i.ai_response_compressed, 'UTF8'), 500)
                ELSE ''
            END, ' '
        ) as search_content
    FROM ai_interactions i
    WHERE i.conversation_id = c.id
    AND i.status = 'completed'
) search_vector ON true
WHERE c.user_id = $1
AND c.status = 'active'
AND search_vector.tsv @@ plainto_tsquery($2)
ORDER BY relevance_score DESC, c.updated_at DESC
LIMIT $3;

-- ===================================================================
-- 8. MAINTENANCE AND CLEANUP QUERIES
-- ===================================================================

-- Archive old completed interactions (run periodically)
WITH archived_interactions AS (
    INSERT INTO ai_interactions_archive 
    SELECT * FROM ai_interactions 
    WHERE created_at < $1  -- Usually NOW() - INTERVAL '90 days'
    AND status IN ('completed', 'failed', 'cancelled')
    RETURNING id, conversation_id, total_cost, created_at
)
DELETE FROM ai_interactions 
WHERE id IN (SELECT id FROM archived_interactions)
RETURNING 
    COUNT(*) as archived_count,
    SUM(total_cost) as archived_total_cost,
    MIN(created_at) as oldest_archived,
    MAX(created_at) as newest_archived;

-- Cleanup orphaned cache entries and update statistics
WITH cleanup_stats AS (
    DELETE FROM cache_entries 
    WHERE expires_at IS NOT NULL 
    AND expires_at < NOW()
    RETURNING size_bytes
)
SELECT 
    COUNT(*) as cleaned_entries,
    ROUND(SUM(size_bytes)::DECIMAL / 1048576, 2) as freed_mb
FROM cleanup_stats;

-- Update daily analytics (run via cron)
INSERT INTO ai_usage_analytics (
    user_id, date, hour, model_name,
    interaction_count, total_input_tokens, total_output_tokens, total_cost,
    avg_processing_time_ms, avg_queue_time_ms, success_rate, cache_hit_rate,
    conversation_types, peak_usage_minute
)
SELECT 
    user_id,
    DATE(created_at) as date,
    EXTRACT(HOUR FROM created_at) as hour,
    model_name,
    COUNT(*) as interaction_count,
    SUM(input_token_count) as total_input_tokens,
    SUM(output_token_count) as total_output_tokens,
    SUM(total_cost) as total_cost,
    ROUND(AVG(processing_time_ms)) as avg_processing_time_ms,
    ROUND(AVG(queue_time_ms)) as avg_queue_time_ms,
    ROUND(COUNT(*) FILTER (WHERE status = 'completed') * 1.0 / COUNT(*), 4) as success_rate,
    ROUND(COUNT(*) FILTER (WHERE cache_hit = true) * 1.0 / COUNT(*), 4) as cache_hit_rate,
    jsonb_object_agg(
        conversation_type, 
        COUNT(*) 
    ) FILTER (WHERE conversation_type IS NOT NULL) as conversation_types,
    MODE() WITHIN GROUP (ORDER BY EXTRACT(MINUTE FROM created_at)) as peak_usage_minute
FROM ai_interactions
WHERE DATE(created_at) = $1  -- Target date for analytics
AND user_id = $2  -- Optional user filter
GROUP BY user_id, DATE(created_at), EXTRACT(HOUR FROM created_at), model_name
ON CONFLICT (user_id, date, hour, model_name) 
DO UPDATE SET
    interaction_count = EXCLUDED.interaction_count,
    total_input_tokens = EXCLUDED.total_input_tokens,
    total_output_tokens = EXCLUDED.total_output_tokens,
    total_cost = EXCLUDED.total_cost,
    avg_processing_time_ms = EXCLUDED.avg_processing_time_ms,
    avg_queue_time_ms = EXCLUDED.avg_queue_time_ms,
    success_rate = EXCLUDED.success_rate,
    cache_hit_rate = EXCLUDED.cache_hit_rate,
    conversation_types = EXCLUDED.conversation_types,
    peak_usage_minute = EXCLUDED.peak_usage_minute,
    updated_at = NOW();

-- ===================================================================
-- QUERY PERFORMANCE NOTES
-- ===================================================================

/*
Performance Optimization Guidelines:

1. Always use parameterized queries ($1, $2, etc.) for security and plan caching
2. Use appropriate LIMIT clauses for pagination
3. Leverage partial indexes on status columns
4. Use LATERAL joins for correlated subqueries
5. Implement proper connection pooling
6. Monitor query performance with pg_stat_statements
7. Use EXPLAIN ANALYZE for query optimization
8. Consider materialized views for heavy aggregations
9. Partition large tables by date ranges
10. Use appropriate data types for better performance

Index Usage:
- User-based queries: Use idx_ai_interactions_user_created
- Conversation queries: Use idx_ai_interactions_conversation_seq
- Status filtering: Use partial indexes on status
- Cache lookups: Use idx_cache_entries_key
- Queue operations: Use idx_queue_messages_queue

Cache Strategy:
- Cache conversation summaries for 2-4 hours
- Cache user analytics for 1-24 hours based on frequency
- Cache model responses for 30 minutes to 2 hours
- Implement cache warming for frequently accessed data
- Use Redis for session-based caching

Queue Optimization:
- Use priority queues for time-sensitive requests
- Implement exponential backoff for retries
- Monitor queue depth and processing times
- Use dead letter queues for failed messages
- Scale workers based on queue depth
*/