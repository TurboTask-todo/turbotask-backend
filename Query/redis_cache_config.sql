-- ===================================================================
-- Redis Cache Management Configuration
-- ===================================================================
-- This file contains Redis-specific configurations and helper functions
-- for the AI conversation system
-- ===================================================================

-- ===================================================================
-- REDIS CACHE STRATEGIES
-- ===================================================================

-- Cache key patterns for different data types
/*
Cache Key Patterns:
- conversation:{user_id}:{session_id} -> Full conversation data
- interaction:{interaction_id} -> Individual interaction
- user_summary:{user_id}:{date} -> Daily user analytics
- model_response:{input_hash} -> Cached AI responses for duplicate inputs
- queue_status:{queue_name} -> Queue monitoring data
- user_tokens:{user_id}:{model} -> Token usage tracking
*/

-- ===================================================================
-- CACHE CONFIGURATION TABLE
-- ===================================================================

CREATE TABLE redis_cache_config (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    cache_pattern VARCHAR(255) NOT NULL UNIQUE,
    description TEXT,
    default_ttl_seconds INTEGER NOT NULL DEFAULT 3600,
    max_size_bytes BIGINT DEFAULT 1048576, -- 1MB default
    compression_enabled BOOLEAN DEFAULT TRUE,
    compression_type VARCHAR(20) DEFAULT 'gzip',
    eviction_policy VARCHAR(50) DEFAULT 'lru',
    
    -- Cache warming settings
    warm_on_create BOOLEAN DEFAULT FALSE,
    warm_on_access BOOLEAN DEFAULT FALSE,
    
    -- Performance settings
    async_writes BOOLEAN DEFAULT TRUE,
    batch_size INTEGER DEFAULT 100,
    
    -- Monitoring
    hit_rate_threshold DECIMAL(5,4) DEFAULT 0.8000,
    size_threshold_mb INTEGER DEFAULT 100,
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Insert default cache configurations
INSERT INTO redis_cache_config (cache_pattern, description, default_ttl_seconds, max_size_bytes, compression_enabled) VALUES
('conversation:*', 'Full conversation data with interactions', 7200, 2097152, TRUE),
('interaction:*', 'Individual AI interaction responses', 3600, 1048576, TRUE),
('user_summary:*', 'User daily analytics and summaries', 86400, 65536, FALSE),
('model_response:*', 'Cached AI model responses by input hash', 1800, 2097152, TRUE),
('queue_status:*', 'RabbitMQ queue monitoring data', 300, 32768, FALSE),
('user_tokens:*', 'User token usage tracking', 3600, 16384, FALSE),
('analytics:*', 'Aggregated analytics data', 14400, 131072, TRUE),
('feedback:*', 'User feedback and ratings cache', 7200, 65536, FALSE);

-- ===================================================================
-- CACHE MANAGEMENT FUNCTIONS
-- ===================================================================

-- Function to generate cache keys
CREATE OR REPLACE FUNCTION generate_cache_key(
    pattern TEXT,
    params TEXT[]
) RETURNS TEXT AS $$
DECLARE
    cache_key TEXT;
    param TEXT;
    i INTEGER := 1;
BEGIN
    cache_key := pattern;
    
    FOREACH param IN ARRAY params LOOP
        cache_key := REPLACE(cache_key, '{' || i || '}', param);
        i := i + 1;
    END LOOP;
    
    -- Replace any remaining placeholders with actual values
    cache_key := REPLACE(cache_key, '{user_id}', COALESCE(params[1], ''));
    cache_key := REPLACE(cache_key, '{conversation_id}', COALESCE(params[2], ''));
    cache_key := REPLACE(cache_key, '{date}', COALESCE(params[3], TO_CHAR(NOW(), 'YYYY-MM-DD')));
    
    RETURN cache_key;
END;
$$ LANGUAGE plpgsql;

-- Function to get cache configuration for a pattern
CREATE OR REPLACE FUNCTION get_cache_config(cache_key TEXT)
RETURNS redis_cache_config AS $$
DECLARE
    config redis_cache_config;
    pattern TEXT;
BEGIN
    -- Find matching pattern (most specific first)
    SELECT * INTO config
    FROM redis_cache_config
    WHERE cache_key LIKE (cache_pattern)
    ORDER BY LENGTH(cache_pattern) DESC
    LIMIT 1;
    
    -- Return default config if no match found
    IF config IS NULL THEN
        SELECT * INTO config
        FROM redis_cache_config
        WHERE cache_pattern = 'default:*'
        LIMIT 1;
    END IF;
    
    RETURN config;
END;
$$ LANGUAGE plpgsql;

-- ===================================================================
-- CACHE STATISTICS AND MONITORING
-- ===================================================================

CREATE TABLE redis_cache_stats (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    cache_pattern VARCHAR(255) NOT NULL,
    date DATE NOT NULL,
    hour INTEGER NOT NULL CHECK (hour BETWEEN 0 AND 23),
    
    -- Hit/Miss statistics
    hits BIGINT DEFAULT 0,
    misses BIGINT DEFAULT 0,
    hit_rate DECIMAL(5,4) GENERATED ALWAYS AS (
        CASE WHEN (hits + misses) > 0 
        THEN hits::DECIMAL / (hits + misses)
        ELSE 0 END
    ) STORED,
    
    -- Size and performance metrics
    avg_size_bytes BIGINT DEFAULT 0,
    max_size_bytes BIGINT DEFAULT 0,
    total_operations BIGINT DEFAULT 0,
    avg_access_time_ms INTEGER DEFAULT 0,
    
    -- Memory usage
    memory_used_bytes BIGINT DEFAULT 0,
    evictions_count BIGINT DEFAULT 0,
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    UNIQUE(cache_pattern, date, hour)
);

-- ===================================================================
-- CACHE WARMING STRATEGIES
-- ===================================================================

-- Function to warm conversation cache
CREATE OR REPLACE FUNCTION warm_conversation_cache(
    p_user_id UUID,
    p_limit INTEGER DEFAULT 10
) RETURNS INTEGER AS $$
DECLARE
    conversation_record RECORD;
    cache_key TEXT;
    cache_data JSONB;
    warmed_count INTEGER := 0;
BEGIN
    -- Get recent active conversations
    FOR conversation_record IN 
        SELECT c.*, 
               COUNT(i.id) as interaction_count,
               MAX(i.created_at) as last_interaction_at
        FROM ai_conversations c
        LEFT JOIN ai_interactions i ON c.id = i.conversation_id
        WHERE c.user_id = p_user_id 
        AND c.status = 'active'
        AND c.updated_at >= NOW() - INTERVAL '7 days'
        GROUP BY c.id
        ORDER BY c.updated_at DESC
        LIMIT p_limit
    LOOP
        -- Generate cache key
        cache_key := generate_cache_key(
            'conversation:{1}:{2}', 
            ARRAY[p_user_id::TEXT, conversation_record.session_id]
        );
        
        -- Prepare cache data
        cache_data := jsonb_build_object(
            'conversation', row_to_json(conversation_record),
            'cached_at', NOW(),
            'cache_type', 'warmed'
        );
        
        -- Insert/update cache entry
        INSERT INTO cache_entries (
            cache_key, 
            content_hash, 
            content_type, 
            size_bytes,
            created_at,
            expires_at
        ) VALUES (
            cache_key,
            md5(cache_data::TEXT),
            'application/json',
            length(cache_data::TEXT),
            NOW(),
            NOW() + INTERVAL '2 hours'
        ) ON CONFLICT (cache_key) DO UPDATE SET
            last_accessed_at = NOW(),
            hit_count = cache_entries.hit_count + 1;
            
        warmed_count := warmed_count + 1;
    END LOOP;
    
    RETURN warmed_count;
END;
$$ LANGUAGE plpgsql;

-- Function to warm model response cache for common inputs
CREATE OR REPLACE FUNCTION warm_model_response_cache(
    p_model_name VARCHAR(100),
    p_limit INTEGER DEFAULT 50
) RETURNS INTEGER AS $$
DECLARE
    response_record RECORD;
    cache_key TEXT;
    warmed_count INTEGER := 0;
BEGIN
    -- Get frequently accessed model responses
    FOR response_record IN
        SELECT input_hash,
               ai_response,
               ai_response_compressed,
               output_compression_type,
               COUNT(*) as access_count
        FROM ai_interactions
        WHERE model_name = p_model_name
        AND input_hash IS NOT NULL
        AND status = 'completed'
        AND created_at >= NOW() - INTERVAL '30 days'
        GROUP BY input_hash, ai_response, ai_response_compressed, output_compression_type
        HAVING COUNT(*) >= 3  -- Only cache responses requested 3+ times
        ORDER BY access_count DESC
        LIMIT p_limit
    LOOP
        cache_key := 'model_response:' || response_record.input_hash;
        
        INSERT INTO cache_entries (
            cache_key,
            content_hash,
            content_type,
            size_bytes,
            created_at,
            expires_at
        ) VALUES (
            cache_key,
            response_record.input_hash,
            'application/json',
            COALESCE(length(response_record.ai_response), length(response_record.ai_response_compressed)),
            NOW(),
            NOW() + INTERVAL '30 minutes'
        ) ON CONFLICT (cache_key) DO UPDATE SET
            hit_count = cache_entries.hit_count + 1;
            
        warmed_count := warmed_count + 1;
    END LOOP;
    
    RETURN warmed_count;
END;
$$ LANGUAGE plpgsql;

-- ===================================================================
-- CACHE INVALIDATION STRATEGIES
-- ===================================================================

-- Function to invalidate user-related caches
CREATE OR REPLACE FUNCTION invalidate_user_cache(p_user_id UUID)
RETURNS INTEGER AS $$
DECLARE
    invalidated_count INTEGER;
BEGIN
    -- Mark cache entries as expired
    UPDATE cache_entries 
    SET expires_at = NOW() - INTERVAL '1 second'
    WHERE cache_key LIKE 'conversation:' || p_user_id || ':%'
       OR cache_key LIKE 'user_summary:' || p_user_id || ':%'
       OR cache_key LIKE 'user_tokens:' || p_user_id || ':%';
    
    GET DIAGNOSTICS invalidated_count = ROW_COUNT;
    
    RETURN invalidated_count;
END;
$$ LANGUAGE plpgsql;

-- Function to invalidate conversation cache
CREATE OR REPLACE FUNCTION invalidate_conversation_cache(p_conversation_id UUID)
RETURNS INTEGER AS $$
DECLARE
    invalidated_count INTEGER;
    user_id_val UUID;
    session_id_val VARCHAR(255);
BEGIN
    -- Get conversation details
    SELECT user_id, session_id 
    INTO user_id_val, session_id_val
    FROM ai_conversations 
    WHERE id = p_conversation_id;
    
    IF user_id_val IS NOT NULL THEN
        -- Invalidate conversation cache
        UPDATE cache_entries 
        SET expires_at = NOW() - INTERVAL '1 second'
        WHERE cache_key = 'conversation:' || user_id_val || ':' || session_id_val;
        
        GET DIAGNOSTICS invalidated_count = ROW_COUNT;
    END IF;
    
    RETURN COALESCE(invalidated_count, 0);
END;
$$ LANGUAGE plpgsql;

-- ===================================================================
-- CACHE MAINTENANCE PROCEDURES
-- ===================================================================

-- Function to cleanup expired cache entries
CREATE OR REPLACE FUNCTION cleanup_expired_redis_cache()
RETURNS TABLE(cleaned_entries INTEGER, freed_bytes BIGINT) AS $$
DECLARE
    result_entries INTEGER;
    result_bytes BIGINT;
BEGIN
    -- Calculate total size of expired entries
    SELECT COUNT(*), COALESCE(SUM(size_bytes), 0)
    INTO result_entries, result_bytes
    FROM cache_entries
    WHERE expires_at IS NOT NULL AND expires_at < NOW();
    
    -- Delete expired entries
    DELETE FROM cache_entries 
    WHERE expires_at IS NOT NULL AND expires_at < NOW();
    
    RETURN QUERY SELECT result_entries, result_bytes;
END;
$$ LANGUAGE plpgsql;

-- Function to get cache health metrics
CREATE OR REPLACE FUNCTION get_cache_health_metrics()
RETURNS TABLE(
    total_entries BIGINT,
    total_size_mb DECIMAL(10,2),
    hit_rate DECIMAL(5,4),
    expired_entries BIGINT,
    top_patterns TEXT[]
) AS $$
BEGIN
    RETURN QUERY
    WITH cache_summary AS (
        SELECT 
            COUNT(*) as total_entries,
            ROUND(SUM(size_bytes)::DECIMAL / 1048576, 2) as total_size_mb,
            CASE WHEN SUM(hit_count + miss_count) > 0 
                 THEN ROUND(SUM(hit_count)::DECIMAL / SUM(hit_count + miss_count), 4)
                 ELSE 0 END as hit_rate,
            COUNT(*) FILTER (WHERE expires_at IS NOT NULL AND expires_at < NOW()) as expired_entries
        FROM cache_entries
    ),
    top_patterns AS (
        SELECT ARRAY_AGG(pattern ORDER BY entry_count DESC) as patterns
        FROM (
            SELECT 
                SPLIT_PART(cache_key, ':', 1) || ':*' as pattern,
                COUNT(*) as entry_count
            FROM cache_entries
            GROUP BY SPLIT_PART(cache_key, ':', 1)
            ORDER BY entry_count DESC
            LIMIT 5
        ) t
    )
    SELECT cs.*, tp.patterns
    FROM cache_summary cs
    CROSS JOIN top_patterns tp;
END;
$$ LANGUAGE plpgsql;

-- ===================================================================
-- INDEXES FOR CACHE TABLES
-- ===================================================================

CREATE INDEX idx_redis_cache_config_pattern ON redis_cache_config(cache_pattern);
CREATE INDEX idx_redis_cache_stats_pattern_date ON redis_cache_stats(cache_pattern, date DESC, hour);
CREATE INDEX idx_redis_cache_stats_hit_rate ON redis_cache_stats(hit_rate) WHERE hit_rate < 0.8;

-- ===================================================================
-- CACHE MONITORING VIEWS
-- ===================================================================

-- Real-time cache performance view
CREATE VIEW v_cache_performance AS
SELECT 
    cache_pattern,
    SUM(hits) as total_hits,
    SUM(misses) as total_misses,
    AVG(hit_rate) as avg_hit_rate,
    SUM(total_operations) as total_operations,
    AVG(avg_access_time_ms) as avg_access_time,
    SUM(memory_used_bytes) as total_memory_bytes
FROM redis_cache_stats
WHERE date >= CURRENT_DATE - INTERVAL '7 days'
GROUP BY cache_pattern
ORDER BY total_operations DESC;

-- Cache efficiency report
CREATE VIEW v_cache_efficiency AS
SELECT 
    ce.cache_key,
    SPLIT_PART(ce.cache_key, ':', 1) as cache_type,
    ce.hit_count,
    ce.miss_count,
    CASE WHEN (ce.hit_count + ce.miss_count) > 0 
         THEN ROUND(ce.hit_count::DECIMAL / (ce.hit_count + ce.miss_count), 4)
         ELSE 0 END as hit_rate,
    ce.size_bytes,
    EXTRACT(EPOCH FROM (NOW() - ce.last_accessed_at))/3600 as hours_since_access,
    rc.default_ttl_seconds,
    ce.expires_at
FROM cache_entries ce
LEFT JOIN redis_cache_config rc ON ce.cache_key LIKE rc.cache_pattern
WHERE ce.created_at >= NOW() - INTERVAL '24 hours';

COMMIT;