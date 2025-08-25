-- ===================================================================
-- AI Conversation Migration Script
-- ===================================================================
-- Run this script after auth.sql to add AI conversation functionality
-- This migration adds all tables needed for AI conversation history
-- ===================================================================

-- Check if users table exists (prerequisite)
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'users') THEN
        RAISE EXCEPTION 'Users table not found. Please run auth.sql migration first.';
    END IF;
END $$;

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_stat_statements";
CREATE EXTENSION IF NOT EXISTS "btree_gin";

-- ===================================================================
-- AI CONVERSATION TABLES
-- ===================================================================

-- AI conversation sessions
CREATE TABLE IF NOT EXISTS ai_conversations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    session_id VARCHAR(255) NOT NULL,
    title VARCHAR(500),
    model_name VARCHAR(100) NOT NULL,
    model_version VARCHAR(50),
    conversation_type VARCHAR(50) DEFAULT 'chat',
    
    -- Performance and cost tracking
    total_input_tokens BIGINT DEFAULT 0,
    total_output_tokens BIGINT DEFAULT 0,
    total_cost DECIMAL(12,8) DEFAULT 0.00,
    interaction_count INTEGER DEFAULT 0,
    
    -- Status and lifecycle
    status VARCHAR(20) DEFAULT 'active' CHECK (status IN ('active', 'archived', 'deleted', 'suspended')),
    priority INTEGER DEFAULT 5 CHECK (priority BETWEEN 1 AND 10),
    
    -- Metadata for caching and queuing
    cache_key VARCHAR(255),
    cache_ttl INTEGER DEFAULT 3600,
    last_accessed_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    -- Enhanced metadata
    metadata JSONB DEFAULT '{}',
    tags TEXT[] DEFAULT '{}',
    
    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    archived_at TIMESTAMP WITH TIME ZONE,
    
    -- Constraints
    CONSTRAINT valid_session_id CHECK (LENGTH(session_id) >= 8),
    CONSTRAINT valid_model_name CHECK (LENGTH(model_name) >= 3),
    CONSTRAINT valid_cost CHECK (total_cost >= 0)
);

-- Individual AI interactions with enhanced compression
CREATE TABLE IF NOT EXISTS ai_interactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id UUID NOT NULL REFERENCES ai_conversations(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    sequence_number INTEGER NOT NULL,
    parent_interaction_id UUID REFERENCES ai_interactions(id) ON DELETE SET NULL,
    
    -- Input data with smart compression
    user_input TEXT,
    user_input_compressed BYTEA,
    input_compression_type VARCHAR(20) DEFAULT 'none' CHECK (input_compression_type IN ('none', 'gzip', 'lz4', 'brotli')),
    input_token_count INTEGER NOT NULL DEFAULT 0,
    input_character_count INTEGER NOT NULL DEFAULT 0,
    input_hash VARCHAR(64), -- SHA-256 for deduplication
    
    -- Output data with smart compression
    ai_response TEXT,
    ai_response_compressed BYTEA,
    output_compression_type VARCHAR(20) DEFAULT 'none' CHECK (output_compression_type IN ('none', 'gzip', 'lz4', 'brotli')),
    output_token_count INTEGER NOT NULL DEFAULT 0,
    output_character_count INTEGER NOT NULL DEFAULT 0,
    output_hash VARCHAR(64), -- SHA-256 for caching
    
    -- Model configuration
    model_name VARCHAR(100) NOT NULL,
    model_version VARCHAR(50),
    temperature DECIMAL(3,2) CHECK (temperature BETWEEN 0 AND 2),
    max_tokens INTEGER CHECK (max_tokens > 0),
    top_p DECIMAL(3,2) CHECK (top_p BETWEEN 0 AND 1),
    frequency_penalty DECIMAL(3,2) CHECK (frequency_penalty BETWEEN -2 AND 2),
    presence_penalty DECIMAL(3,2) CHECK (presence_penalty BETWEEN -2 AND 2),
    stop_sequences TEXT[],
    system_prompt TEXT,
    
    -- Performance metrics
    processing_time_ms INTEGER DEFAULT 0,
    queue_time_ms INTEGER DEFAULT 0,
    total_latency_ms INTEGER DEFAULT 0,
    retry_count INTEGER DEFAULT 0,
    
    -- Cost breakdown
    input_cost DECIMAL(12,10) DEFAULT 0.00,
    output_cost DECIMAL(12,10) DEFAULT 0.00,
    total_cost DECIMAL(12,8) DEFAULT 0.00,
    
    -- Queue and caching
    queue_message_id VARCHAR(255), -- RabbitMQ message ID
    queue_priority INTEGER DEFAULT 5 CHECK (queue_priority BETWEEN 1 AND 10),
    cache_key VARCHAR(255),
    cache_hit BOOLEAN DEFAULT FALSE,
    
    -- Status and error handling
    status VARCHAR(20) DEFAULT 'pending' CHECK (status IN ('pending', 'queued', 'processing', 'completed', 'failed', 'cancelled', 'cached')),
    error_code VARCHAR(50),
    error_message TEXT,
    
    -- Request context
    client_ip INET,
    user_agent TEXT,
    request_id VARCHAR(255),
    trace_id VARCHAR(255), -- For distributed tracing
    
    -- Enhanced metadata
    request_metadata JSONB DEFAULT '{}',
    response_metadata JSONB DEFAULT '{}',
    processing_metadata JSONB DEFAULT '{}',
    
    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    queued_at TIMESTAMP WITH TIME ZONE,
    started_processing_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    cached_at TIMESTAMP WITH TIME ZONE,
    
    -- Check constraints
    CONSTRAINT valid_sequence CHECK (sequence_number > 0),
    CONSTRAINT valid_token_counts CHECK (input_token_count >= 0 AND output_token_count >= 0),
    CONSTRAINT valid_costs CHECK (input_cost >= 0 AND output_cost >= 0 AND total_cost >= 0),
    CONSTRAINT valid_processing_times CHECK (processing_time_ms >= 0 AND queue_time_ms >= 0 AND total_latency_ms >= 0)
);

-- ===================================================================
-- CACHE MANAGEMENT TABLES
-- ===================================================================

-- Redis cache entries tracking
CREATE TABLE IF NOT EXISTS cache_entries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    cache_key VARCHAR(255) NOT NULL UNIQUE,
    content_hash VARCHAR(64) NOT NULL,
    content_type VARCHAR(50) NOT NULL,
    size_bytes BIGINT NOT NULL,
    hit_count BIGINT DEFAULT 0,
    miss_count BIGINT DEFAULT 0,
    ttl_seconds INTEGER DEFAULT 3600,
    tags TEXT[] DEFAULT '{}',
    
    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    last_accessed_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    expires_at TIMESTAMP WITH TIME ZONE,
    
    CONSTRAINT valid_ttl CHECK (ttl_seconds > 0),
    CONSTRAINT valid_size CHECK (size_bytes >= 0)
);

-- ===================================================================
-- QUEUE MANAGEMENT TABLES
-- ===================================================================

-- RabbitMQ queue management
CREATE TABLE IF NOT EXISTS queue_messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    message_id VARCHAR(255) NOT NULL UNIQUE,
    queue_name VARCHAR(100) NOT NULL,
    routing_key VARCHAR(255),
    exchange_name VARCHAR(100),
    
    -- Message content
    payload JSONB NOT NULL,
    content_type VARCHAR(50) DEFAULT 'application/json',
    content_encoding VARCHAR(20) DEFAULT 'utf-8',
    
    -- Queue properties
    priority INTEGER DEFAULT 5 CHECK (priority BETWEEN 1 AND 10),
    delay_seconds INTEGER DEFAULT 0,
    max_retries INTEGER DEFAULT 3,
    retry_count INTEGER DEFAULT 0,
    
    -- Processing state
    status VARCHAR(20) DEFAULT 'pending' CHECK (status IN ('pending', 'processing', 'completed', 'failed', 'dead_letter')),
    worker_id VARCHAR(255),
    error_message TEXT,
    
    -- Related entities
    interaction_id UUID REFERENCES ai_interactions(id) ON DELETE SET NULL,
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    
    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    scheduled_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    started_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    expires_at TIMESTAMP WITH TIME ZONE DEFAULT (NOW() + INTERVAL '24 hours')
);

-- ===================================================================
-- ANALYTICS AND FEEDBACK TABLES
-- ===================================================================

-- Enhanced feedback system
CREATE TABLE IF NOT EXISTS ai_interaction_feedback (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    interaction_id UUID NOT NULL REFERENCES ai_interactions(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    
    -- Rating system
    overall_rating INTEGER CHECK (overall_rating BETWEEN 1 AND 5),
    accuracy_rating INTEGER CHECK (accuracy_rating BETWEEN 1 AND 5),
    helpfulness_rating INTEGER CHECK (helpfulness_rating BETWEEN 1 AND 5),
    relevance_rating INTEGER CHECK (relevance_rating BETWEEN 1 AND 5),
    
    -- Feedback details
    feedback_type VARCHAR(50),
    feedback_text TEXT,
    improvement_suggestions TEXT,
    tags TEXT[] DEFAULT '{}',
    
    -- Context
    session_quality INTEGER CHECK (session_quality BETWEEN 1 AND 5),
    response_time_satisfaction INTEGER CHECK (response_time_satisfaction BETWEEN 1 AND 5),
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Real-time usage analytics (optimized for frequent updates)
CREATE TABLE IF NOT EXISTS ai_usage_analytics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    date DATE NOT NULL,
    hour INTEGER NOT NULL CHECK (hour BETWEEN 0 AND 23),
    model_name VARCHAR(100) NOT NULL,
    
    -- Aggregated metrics
    interaction_count INTEGER DEFAULT 0,
    total_input_tokens BIGINT DEFAULT 0,
    total_output_tokens BIGINT DEFAULT 0,
    total_cost DECIMAL(12,6) DEFAULT 0.00,
    
    -- Performance metrics
    avg_processing_time_ms INTEGER DEFAULT 0,
    avg_queue_time_ms INTEGER DEFAULT 0,
    success_rate DECIMAL(5,4) DEFAULT 1.0000,
    cache_hit_rate DECIMAL(5,4) DEFAULT 0.0000,
    
    -- Usage patterns
    conversation_types JSONB DEFAULT '{}',
    peak_usage_minute INTEGER CHECK (peak_usage_minute BETWEEN 0 AND 59),
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    UNIQUE(user_id, date, hour, model_name)
);

-- ===================================================================
-- PARTITIONED ARCHIVE TABLES
-- ===================================================================

-- Archive table for old interactions (partitioned by date)
CREATE TABLE IF NOT EXISTS ai_interactions_archive (
    LIKE ai_interactions INCLUDING DEFAULTS INCLUDING CONSTRAINTS INCLUDING INDEXES
) PARTITION BY RANGE (created_at);

-- Drop the original primary key and create a composite one that includes the partition key
ALTER TABLE ai_interactions_archive DROP CONSTRAINT IF EXISTS ai_interactions_archive_pkey;
ALTER TABLE ai_interactions_archive ADD CONSTRAINT ai_interactions_archive_pkey 
    PRIMARY KEY (id, created_at);

-- Create quarterly partitions for better management
DO $$
BEGIN
    -- Create partitions for 2024
    IF NOT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'ai_interactions_archive_2024_q1') THEN
        CREATE TABLE ai_interactions_archive_2024_q1 PARTITION OF ai_interactions_archive
            FOR VALUES FROM ('2024-01-01') TO ('2024-04-01');
    END IF;

    IF NOT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'ai_interactions_archive_2024_q2') THEN
        CREATE TABLE ai_interactions_archive_2024_q2 PARTITION OF ai_interactions_archive
            FOR VALUES FROM ('2024-04-01') TO ('2024-07-01');
    END IF;

    IF NOT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'ai_interactions_archive_2024_q3') THEN
        CREATE TABLE ai_interactions_archive_2024_q3 PARTITION OF ai_interactions_archive
            FOR VALUES FROM ('2024-07-01') TO ('2024-10-01');
    END IF;

    IF NOT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'ai_interactions_archive_2024_q4') THEN
        CREATE TABLE ai_interactions_archive_2024_q4 PARTITION OF ai_interactions_archive
            FOR VALUES FROM ('2024-10-01') TO ('2025-01-01');
    END IF;

    -- Create partitions for 2025
    IF NOT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'ai_interactions_archive_2025_q1') THEN
        CREATE TABLE ai_interactions_archive_2025_q1 PARTITION OF ai_interactions_archive
            FOR VALUES FROM ('2025-01-01') TO ('2025-04-01');
    END IF;
END $$;

-- ===================================================================
-- OPTIMIZED INDEXES
-- ===================================================================

-- ai_conversations indexes
CREATE INDEX IF NOT EXISTS idx_ai_conversations_user_id ON ai_conversations(user_id) WHERE status = 'active';
CREATE INDEX IF NOT EXISTS idx_ai_conversations_session_id ON ai_conversations(session_id);
CREATE INDEX IF NOT EXISTS idx_ai_conversations_created_at ON ai_conversations(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_ai_conversations_status ON ai_conversations(status) WHERE status IN ('active', 'archived');
CREATE INDEX IF NOT EXISTS idx_ai_conversations_cache_key ON ai_conversations(cache_key) WHERE cache_key IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_ai_conversations_last_accessed ON ai_conversations(last_accessed_at DESC);
CREATE INDEX IF NOT EXISTS idx_ai_conversations_model_cost ON ai_conversations(model_name, total_cost DESC);

-- ai_interactions indexes
CREATE INDEX IF NOT EXISTS idx_ai_interactions_conversation_id ON ai_interactions(conversation_id);
CREATE INDEX IF NOT EXISTS idx_ai_interactions_user_id ON ai_interactions(user_id);
CREATE INDEX IF NOT EXISTS idx_ai_interactions_created_at ON ai_interactions(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_ai_interactions_status ON ai_interactions(status) WHERE status IN ('pending', 'processing', 'failed');
CREATE INDEX IF NOT EXISTS idx_ai_interactions_model_name ON ai_interactions(model_name);
CREATE INDEX IF NOT EXISTS idx_ai_interactions_sequence ON ai_interactions(conversation_id, sequence_number);
CREATE INDEX IF NOT EXISTS idx_ai_interactions_cache_key ON ai_interactions(cache_key) WHERE cache_key IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_ai_interactions_queue_message ON ai_interactions(queue_message_id) WHERE queue_message_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_ai_interactions_hash ON ai_interactions(input_hash, output_hash) WHERE input_hash IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_ai_interactions_cost ON ai_interactions(total_cost DESC) WHERE total_cost > 0;

-- Composite indexes for common query patterns
CREATE INDEX IF NOT EXISTS idx_ai_interactions_user_created ON ai_interactions(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_ai_interactions_conversation_seq ON ai_interactions(conversation_id, sequence_number);
CREATE INDEX IF NOT EXISTS idx_ai_interactions_model_status ON ai_interactions(model_name, status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_ai_interactions_user_model_date ON ai_interactions(user_id, model_name, created_at DESC);

-- Cache and queue indexes
CREATE INDEX IF NOT EXISTS idx_cache_entries_key ON cache_entries(cache_key);
CREATE INDEX IF NOT EXISTS idx_cache_entries_hash ON cache_entries(content_hash);
CREATE INDEX IF NOT EXISTS idx_cache_entries_expires ON cache_entries(expires_at) WHERE expires_at IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_cache_entries_accessed ON cache_entries(last_accessed_at DESC);

CREATE INDEX IF NOT EXISTS idx_queue_messages_id ON queue_messages(message_id);
CREATE INDEX IF NOT EXISTS idx_queue_messages_queue ON queue_messages(queue_name, status, priority DESC, created_at);
CREATE INDEX IF NOT EXISTS idx_queue_messages_interaction ON queue_messages(interaction_id) WHERE interaction_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_queue_messages_status ON queue_messages(status, scheduled_at) WHERE status IN ('pending', 'processing');
CREATE INDEX IF NOT EXISTS idx_queue_messages_expires ON queue_messages(expires_at) WHERE expires_at < NOW();

-- Analytics indexes
CREATE INDEX IF NOT EXISTS idx_ai_feedback_interaction ON ai_interaction_feedback(interaction_id);
CREATE INDEX IF NOT EXISTS idx_ai_feedback_user ON ai_interaction_feedback(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_ai_feedback_rating ON ai_interaction_feedback(overall_rating, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_ai_analytics_user_date ON ai_usage_analytics(user_id, date DESC, hour);
CREATE INDEX IF NOT EXISTS idx_ai_analytics_model_date ON ai_usage_analytics(model_name, date DESC);
CREATE INDEX IF NOT EXISTS idx_ai_analytics_cost ON ai_usage_analytics(total_cost DESC) WHERE total_cost > 0;

-- ===================================================================
-- PERFORMANCE VIEWS
-- ===================================================================

-- Active conversations with latest interaction
CREATE OR REPLACE VIEW v_active_conversations AS
SELECT 
    c.*,
    i.created_at as last_interaction_at,
    i.ai_response as last_response,
    i.status as last_interaction_status
FROM ai_conversations c
LEFT JOIN LATERAL (
    SELECT * FROM ai_interactions 
    WHERE conversation_id = c.id 
    ORDER BY sequence_number DESC 
    LIMIT 1
) i ON true
WHERE c.status = 'active';

-- User interaction summary
CREATE OR REPLACE VIEW v_user_interaction_summary AS
SELECT 
    user_id,
    COUNT(*) as total_interactions,
    SUM(input_token_count) as total_input_tokens,
    SUM(output_token_count) as total_output_tokens,
    SUM(total_cost) as total_cost,
    AVG(processing_time_ms) as avg_processing_time,
    COUNT(*) FILTER (WHERE cache_hit = true) as cache_hits,
    COUNT(*) FILTER (WHERE status = 'completed') as successful_interactions,
    MAX(created_at) as last_interaction_at
FROM ai_interactions
GROUP BY user_id;

-- Model performance metrics
CREATE OR REPLACE VIEW v_model_performance AS
SELECT 
    model_name,
    COUNT(*) as total_interactions,
    AVG(processing_time_ms) as avg_processing_time,
    AVG(total_cost) as avg_cost_per_interaction,
    AVG(output_token_count::DECIMAL / NULLIF(input_token_count, 0)) as avg_output_input_ratio,
    COUNT(*) FILTER (WHERE status = 'completed') * 100.0 / COUNT(*) as success_rate_percent
FROM ai_interactions
WHERE created_at >= NOW() - INTERVAL '30 days'
GROUP BY model_name;

-- ===================================================================
-- TRIGGERS AND FUNCTIONS
-- ===================================================================

-- Function to update conversation totals
CREATE OR REPLACE FUNCTION update_conversation_totals()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' OR TG_OP = 'UPDATE' THEN
        UPDATE ai_conversations SET
            total_input_tokens = (
                SELECT COALESCE(SUM(input_token_count), 0)
                FROM ai_interactions 
                WHERE conversation_id = NEW.conversation_id
            ),
            total_output_tokens = (
                SELECT COALESCE(SUM(output_token_count), 0)
                FROM ai_interactions 
                WHERE conversation_id = NEW.conversation_id
            ),
            total_cost = (
                SELECT COALESCE(SUM(total_cost), 0)
                FROM ai_interactions 
                WHERE conversation_id = NEW.conversation_id
            ),
            interaction_count = (
                SELECT COUNT(*)
                FROM ai_interactions 
                WHERE conversation_id = NEW.conversation_id
            ),
            updated_at = NOW()
        WHERE id = NEW.conversation_id;
    END IF;
    
    RETURN COALESCE(NEW, OLD);
END;
$$ LANGUAGE plpgsql;

-- Create trigger only if it doesn't exist
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'trg_update_conversation_totals') THEN
        CREATE TRIGGER trg_update_conversation_totals
            AFTER INSERT OR UPDATE OR DELETE ON ai_interactions
            FOR EACH ROW
            EXECUTE FUNCTION update_conversation_totals();
    END IF;
END $$;

-- Function to manage cache TTL
CREATE OR REPLACE FUNCTION cleanup_expired_cache()
RETURNS void AS $$
BEGIN
    DELETE FROM cache_entries 
    WHERE expires_at IS NOT NULL AND expires_at < NOW();
END;
$$ LANGUAGE plpgsql;

-- Function to archive old interactions
CREATE OR REPLACE FUNCTION archive_old_interactions(days_old INTEGER DEFAULT 90)
RETURNS INTEGER AS $$
DECLARE
    archived_count INTEGER;
BEGIN
    WITH archived AS (
        INSERT INTO ai_interactions_archive 
        SELECT * FROM ai_interactions 
        WHERE created_at < NOW() - INTERVAL '1 day' * days_old
        AND status IN ('completed', 'failed', 'cancelled')
        RETURNING id
    )
    SELECT COUNT(*) INTO archived_count FROM archived;
    
    DELETE FROM ai_interactions 
    WHERE created_at < NOW() - INTERVAL '1 day' * days_old
    AND status IN ('completed', 'failed', 'cancelled');
    
    RETURN archived_count;
END;
$$ LANGUAGE plpgsql;

-- ===================================================================
-- INITIAL DATA AND CONFIGURATION
-- ===================================================================

-- Insert default cache configuration
INSERT INTO cache_entries (cache_key, content_hash, content_type, size_bytes, ttl_seconds)
VALUES 
    ('system:models', 'default', 'application/json', 0, 3600),
    ('system:config', 'default', 'application/json', 0, 1800)
ON CONFLICT (cache_key) DO NOTHING;

-- Grant permissions (commented out for security - adjust as needed)
-- GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO ai_app_user;
-- GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO ai_app_user;

-- Log migration completion
DO $$
BEGIN
    RAISE NOTICE 'AI Conversation migration completed successfully!';
    RAISE NOTICE 'Tables created:';
    RAISE NOTICE '- ai_conversations';
    RAISE NOTICE '- ai_interactions';
    RAISE NOTICE '- ai_interaction_feedback';
    RAISE NOTICE '- ai_usage_analytics';
    RAISE NOTICE '- cache_entries';
    RAISE NOTICE '- queue_messages';
    RAISE NOTICE '- ai_interactions_archive (partitioned)';
    RAISE NOTICE 'Views created:';
    RAISE NOTICE '- v_active_conversations';
    RAISE NOTICE '- v_user_interaction_summary';
    RAISE NOTICE '- v_model_performance';
    RAISE NOTICE 'Functions created:';
    RAISE NOTICE '- update_conversation_totals()';
    RAISE NOTICE '- cleanup_expired_cache()';
    RAISE NOTICE '- archive_old_interactions()';
END $$;

COMMIT;