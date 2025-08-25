-- ===================================================================
-- RABBITMQ QUEUE MANAGEMENT SCHEMA
-- ===================================================================
-- Enhanced queue management system for AI conversation processing
-- with dynamic scaling and comprehensive monitoring
-- ===================================================================

-- ===================================================================
-- QUEUE CONFIGURATION TABLES
-- ===================================================================

-- RabbitMQ exchange and queue definitions
CREATE TABLE rabbitmq_exchanges (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    exchange_name VARCHAR(100) NOT NULL UNIQUE,
    exchange_type VARCHAR(20) NOT NULL CHECK (exchange_type IN ('direct', 'topic', 'fanout', 'headers')),
    durable BOOLEAN DEFAULT TRUE,
    auto_delete BOOLEAN DEFAULT FALSE,
    arguments JSONB DEFAULT '{}',
    
    -- Management
    is_active BOOLEAN DEFAULT TRUE,
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Queue definitions with routing and scaling config
CREATE TABLE rabbitmq_queues (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    queue_name VARCHAR(100) NOT NULL UNIQUE,
    exchange_name VARCHAR(100) NOT NULL,
    routing_key VARCHAR(255),
    
    -- Queue properties
    durable BOOLEAN DEFAULT TRUE,
    exclusive BOOLEAN DEFAULT FALSE,
    auto_delete BOOLEAN DEFAULT FALSE,
    
    -- Processing configuration
    max_priority INTEGER DEFAULT 10,
    message_ttl_ms INTEGER DEFAULT 3600000, -- 1 hour default
    max_length INTEGER DEFAULT 10000,
    max_length_bytes BIGINT DEFAULT 104857600, -- 100MB
    
    -- Consumer configuration
    prefetch_count INTEGER DEFAULT 10,
    consumer_timeout_ms INTEGER DEFAULT 300000, -- 5 minutes
    ack_timeout_ms INTEGER DEFAULT 30000, -- 30 seconds
    
    -- Dead letter configuration
    dead_letter_exchange VARCHAR(100),
    dead_letter_routing_key VARCHAR(255),
    max_retries INTEGER DEFAULT 3,
    retry_delay_ms INTEGER DEFAULT 5000,
    
    -- Auto-scaling configuration
    min_consumers INTEGER DEFAULT 1,
    max_consumers INTEGER DEFAULT 10,
    scale_up_threshold INTEGER DEFAULT 100, -- Messages in queue
    scale_down_threshold INTEGER DEFAULT 10,
    scale_up_cooldown_minutes INTEGER DEFAULT 5,
    scale_down_cooldown_minutes INTEGER DEFAULT 15,
    
    -- Monitoring thresholds
    latency_threshold_ms INTEGER DEFAULT 30000,
    error_rate_threshold DECIMAL(5,4) DEFAULT 0.05, -- 5%
    
    -- Queue metadata
    description TEXT,
    is_active BOOLEAN DEFAULT TRUE,
    arguments JSONB DEFAULT '{}',
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    FOREIGN KEY (exchange_name) REFERENCES rabbitmq_exchanges(exchange_name) ON UPDATE CASCADE
);

-- ===================================================================
-- MESSAGE PROCESSING TABLES
-- ===================================================================

-- Enhanced message tracking with processing states
CREATE TABLE queue_message_processing (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    message_id VARCHAR(255) NOT NULL,
    queue_name VARCHAR(100) NOT NULL,
    
    -- Message identification
    correlation_id VARCHAR(255),
    reply_to VARCHAR(255),
    delivery_tag BIGINT,
    redelivered BOOLEAN DEFAULT FALSE,
    
    -- Content and routing
    exchange_name VARCHAR(100),
    routing_key VARCHAR(255),
    content_type VARCHAR(100) DEFAULT 'application/json',
    content_encoding VARCHAR(50) DEFAULT 'utf-8',
    
    -- Message properties
    priority INTEGER DEFAULT 5 CHECK (priority BETWEEN 0 AND 10),
    timestamp_utc TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    expiration_ms INTEGER,
    message_id_original VARCHAR(255), -- Original message ID for retries
    
    -- Processing state
    status VARCHAR(20) DEFAULT 'received' CHECK (status IN (
        'received', 'queued', 'processing', 'completed', 
        'failed', 'requeued', 'dead_letter', 'expired'
    )),
    processing_attempts INTEGER DEFAULT 0,
    last_error TEXT,
    error_code VARCHAR(50),
    
    -- Consumer information
    consumer_tag VARCHAR(255),
    worker_id VARCHAR(255),
    worker_version VARCHAR(50),
    processing_node VARCHAR(100),
    
    -- Timing information
    received_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    queued_at TIMESTAMP WITH TIME ZONE,
    started_processing_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    acknowledged_at TIMESTAMP WITH TIME ZONE,
    
    -- Performance metrics
    queue_wait_time_ms INTEGER,
    processing_time_ms INTEGER,
    total_latency_ms INTEGER,
    
    -- Related entities
    interaction_id UUID,
    user_id UUID,
    conversation_id UUID,
    
    -- Message payload (for debugging/retry)
    payload JSONB,
    headers JSONB DEFAULT '{}',
    
    FOREIGN KEY (queue_name) REFERENCES rabbitmq_queues(queue_name) ON UPDATE CASCADE,
    FOREIGN KEY (interaction_id) REFERENCES ai_interactions(id) ON DELETE SET NULL
);

-- ===================================================================
-- CONSUMER MANAGEMENT
-- ===================================================================

-- Consumer/worker registration and health tracking
CREATE TABLE rabbitmq_consumers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    consumer_tag VARCHAR(255) NOT NULL,
    queue_name VARCHAR(100) NOT NULL,
    worker_id VARCHAR(255) NOT NULL,
    
    -- Worker information
    hostname VARCHAR(255),
    process_id INTEGER,
    worker_version VARCHAR(50),
    capabilities JSONB DEFAULT '{}',
    
    -- Configuration
    prefetch_count INTEGER DEFAULT 10,
    auto_ack BOOLEAN DEFAULT FALSE,
    exclusive BOOLEAN DEFAULT FALSE,
    
    -- Health and status
    status VARCHAR(20) DEFAULT 'starting' CHECK (status IN (
        'starting', 'active', 'idle', 'busy', 'paused', 'stopping', 'stopped', 'failed'
    )),
    last_heartbeat TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    health_score INTEGER DEFAULT 100 CHECK (health_score BETWEEN 0 AND 100),
    
    -- Performance metrics
    messages_processed_total BIGINT DEFAULT 0,
    messages_failed_total BIGINT DEFAULT 0,
    avg_processing_time_ms INTEGER DEFAULT 0,
    current_load DECIMAL(5,4) DEFAULT 0.0000,
    
    -- Lifecycle
    started_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    stopped_at TIMESTAMP WITH TIME ZONE,
    last_activity_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    FOREIGN KEY (queue_name) REFERENCES rabbitmq_queues(queue_name) ON UPDATE CASCADE,
    UNIQUE(consumer_tag, queue_name)
);

-- ===================================================================
-- MONITORING AND METRICS
-- ===================================================================

-- Real-time queue metrics
CREATE TABLE queue_metrics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    queue_name VARCHAR(100) NOT NULL,
    metric_timestamp TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    -- Queue depth metrics
    messages_ready INTEGER DEFAULT 0,
    messages_unacknowledged INTEGER DEFAULT 0,
    messages_total INTEGER DEFAULT 0,
    
    -- Consumer metrics
    active_consumers INTEGER DEFAULT 0,
    idle_consumers INTEGER DEFAULT 0,
    
    -- Throughput metrics (per minute)
    messages_published_rate DECIMAL(10,4) DEFAULT 0,
    messages_delivered_rate DECIMAL(10,4) DEFAULT 0,
    messages_acknowledged_rate DECIMAL(10,4) DEFAULT 0,
    messages_redelivered_rate DECIMAL(10,4) DEFAULT 0,
    
    -- Performance metrics
    avg_processing_time_ms INTEGER DEFAULT 0,
    avg_queue_wait_time_ms INTEGER DEFAULT 0,
    p95_processing_time_ms INTEGER DEFAULT 0,
    
    -- Error metrics
    error_rate DECIMAL(5,4) DEFAULT 0.0000,
    timeout_rate DECIMAL(5,4) DEFAULT 0.0000,
    dead_letter_rate DECIMAL(5,4) DEFAULT 0.0000,
    
    -- Memory and disk usage
    queue_memory_bytes BIGINT DEFAULT 0,
    queue_disk_bytes BIGINT DEFAULT 0,
    
    FOREIGN KEY (queue_name) REFERENCES rabbitmq_queues(queue_name) ON UPDATE CASCADE
);

-- ===================================================================
-- AUTO-SCALING MANAGEMENT
-- ===================================================================

-- Consumer scaling decisions and history
CREATE TABLE consumer_scaling_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    queue_name VARCHAR(100) NOT NULL,
    
    -- Scaling decision
    scaling_action VARCHAR(20) NOT NULL CHECK (scaling_action IN ('scale_up', 'scale_down', 'no_action')),
    trigger_reason VARCHAR(100),
    
    -- Before and after state
    consumers_before INTEGER NOT NULL,
    consumers_after INTEGER NOT NULL,
    queue_depth_before INTEGER NOT NULL,
    avg_processing_time_before INTEGER,
    
    -- Metrics that triggered scaling
    queue_depth_threshold INTEGER,
    processing_time_threshold INTEGER,
    error_rate_threshold DECIMAL(5,4),
    
    -- Scaling constraints
    min_consumers INTEGER,
    max_consumers INTEGER,
    cooldown_remaining_minutes INTEGER DEFAULT 0,
    
    -- Execution details
    executed BOOLEAN DEFAULT FALSE,
    execution_error TEXT,
    execution_time_ms INTEGER,
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    executed_at TIMESTAMP WITH TIME ZONE,
    
    FOREIGN KEY (queue_name) REFERENCES rabbitmq_queues(queue_name) ON UPDATE CASCADE
);

-- ===================================================================
-- DEAD LETTER MANAGEMENT
-- ===================================================================

-- Dead letter queue messages for analysis and recovery
CREATE TABLE dead_letter_messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    original_message_id VARCHAR(255) NOT NULL,
    original_queue_name VARCHAR(100) NOT NULL,
    
    -- Original message context
    original_exchange VARCHAR(100),
    original_routing_key VARCHAR(255),
    death_reason VARCHAR(100), -- 'rejected', 'expired', 'maxlen', 'retry_exceeded'
    
    -- Failure analysis
    failure_count INTEGER DEFAULT 1,
    last_error_message TEXT,
    last_error_code VARCHAR(50),
    error_stack_trace TEXT,
    
    -- Message content for recovery
    original_payload JSONB,
    original_headers JSONB DEFAULT '{}',
    original_properties JSONB DEFAULT '{}',
    
    -- Recovery status
    recovery_status VARCHAR(20) DEFAULT 'pending' CHECK (recovery_status IN (
        'pending', 'analyzing', 'recoverable', 'non_recoverable', 'recovered', 'discarded'
    )),
    recovery_attempts INTEGER DEFAULT 0,
    recovery_notes TEXT,
    
    -- Related entities for context
    interaction_id UUID,
    user_id UUID,
    conversation_id UUID,
    
    -- Timestamps
    first_death_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    last_death_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    analyzed_at TIMESTAMP WITH TIME ZONE,
    recovered_at TIMESTAMP WITH TIME ZONE,
    
    FOREIGN KEY (interaction_id) REFERENCES ai_interactions(id) ON DELETE SET NULL
);

-- ===================================================================
-- QUEUE PERFORMANCE VIEWS
-- ===================================================================

-- Real-time queue health dashboard
CREATE VIEW v_queue_health AS
SELECT 
    q.queue_name,
    q.is_active,
    
    -- Current state
    COALESCE(qm.messages_total, 0) as current_queue_depth,
    COALESCE(qm.active_consumers, 0) as active_consumers,
    q.min_consumers,
    q.max_consumers,
    
    -- Performance metrics
    COALESCE(qm.avg_processing_time_ms, 0) as avg_processing_time_ms,
    COALESCE(qm.avg_queue_wait_time_ms, 0) as avg_queue_wait_time_ms,
    COALESCE(qm.error_rate, 0) as error_rate,
    
    -- Health indicators
    CASE 
        WHEN qm.messages_total > q.scale_up_threshold THEN 'overloaded'
        WHEN qm.messages_total > q.scale_up_threshold * 0.7 THEN 'busy'
        WHEN qm.messages_total < q.scale_down_threshold THEN 'idle'
        ELSE 'normal'
    END as load_status,
    
    CASE 
        WHEN qm.error_rate > q.error_rate_threshold THEN 'unhealthy'
        WHEN qm.avg_processing_time_ms > q.latency_threshold_ms THEN 'slow'
        WHEN qm.active_consumers = 0 THEN 'no_consumers'
        ELSE 'healthy'
    END as health_status,
    
    -- Scaling recommendations
    CASE 
        WHEN qm.messages_total > q.scale_up_threshold AND qm.active_consumers < q.max_consumers THEN 'scale_up'
        WHEN qm.messages_total < q.scale_down_threshold AND qm.active_consumers > q.min_consumers THEN 'scale_down'
        ELSE 'no_action'
    END as scaling_recommendation,
    
    qm.metric_timestamp as last_updated
FROM rabbitmq_queues q
LEFT JOIN LATERAL (
    SELECT * FROM queue_metrics 
    WHERE queue_name = q.queue_name 
    ORDER BY metric_timestamp DESC 
    LIMIT 1
) qm ON true
WHERE q.is_active = true;

-- Consumer performance view
CREATE VIEW v_consumer_performance AS
SELECT 
    c.queue_name,
    c.worker_id,
    c.status,
    c.health_score,
    
    -- Performance metrics
    c.messages_processed_total,
    c.messages_failed_total,
    CASE WHEN c.messages_processed_total > 0 
         THEN ROUND(c.messages_failed_total * 100.0 / c.messages_processed_total, 2)
         ELSE 0 END as failure_rate_percent,
    c.avg_processing_time_ms,
    c.current_load,
    
    -- Activity metrics
    EXTRACT(EPOCH FROM (NOW() - c.last_activity_at)) / 60 as minutes_since_activity,
    EXTRACT(EPOCH FROM (NOW() - c.last_heartbeat)) as seconds_since_heartbeat,
    EXTRACT(EPOCH FROM (NOW() - c.started_at)) / 3600 as uptime_hours,
    
    -- Health indicators
    CASE 
        WHEN c.status IN ('failed', 'stopped') THEN 'down'
        WHEN EXTRACT(EPOCH FROM (NOW() - c.last_heartbeat)) > 300 THEN 'stale'
        WHEN c.health_score < 50 THEN 'unhealthy'
        WHEN c.current_load > 0.9 THEN 'overloaded'
        ELSE 'healthy'
    END as health_status
FROM rabbitmq_consumers c
WHERE c.status != 'stopped'
ORDER BY c.queue_name, c.health_score DESC;

-- ===================================================================
-- QUEUE MANAGEMENT FUNCTIONS
-- ===================================================================

-- Function to calculate queue scaling decision
CREATE OR REPLACE FUNCTION calculate_scaling_decision(
    p_queue_name VARCHAR(100)
) RETURNS TABLE(
    action VARCHAR(20),
    reason TEXT,
    consumers_target INTEGER
) AS $$
DECLARE
    queue_config rabbitmq_queues%ROWTYPE;
    current_metrics queue_metrics%ROWTYPE;
    current_consumers INTEGER;
    last_scaling_event TIMESTAMP WITH TIME ZONE;
    cooldown_remaining INTEGER;
BEGIN
    -- Get queue configuration
    SELECT * INTO queue_config
    FROM rabbitmq_queues
    WHERE queue_name = p_queue_name AND is_active = true;
    
    IF queue_config IS NULL THEN
        RETURN QUERY SELECT 'no_action'::VARCHAR(20), 'Queue not found or inactive'::TEXT, 0::INTEGER;
        RETURN;
    END IF;
    
    -- Get latest metrics
    SELECT * INTO current_metrics
    FROM queue_metrics
    WHERE queue_name = p_queue_name
    ORDER BY metric_timestamp DESC
    LIMIT 1;
    
    IF current_metrics IS NULL THEN
        RETURN QUERY SELECT 'no_action'::VARCHAR(20), 'No metrics available'::TEXT, queue_config.min_consumers;
        RETURN;
    END IF;
    
    -- Get current consumer count
    current_consumers := current_metrics.active_consumers;
    
    -- Check cooldown period
    SELECT MAX(created_at) INTO last_scaling_event
    FROM consumer_scaling_events
    WHERE queue_name = p_queue_name
    AND scaling_action IN ('scale_up', 'scale_down')
    AND executed = true;
    
    -- Determine scaling action
    IF current_metrics.messages_total > queue_config.scale_up_threshold THEN
        -- Check if we can scale up
        IF current_consumers >= queue_config.max_consumers THEN
            RETURN QUERY SELECT 'no_action'::VARCHAR(20), 'Already at maximum consumers'::TEXT, current_consumers;
        ELSIF last_scaling_event IS NOT NULL AND 
              last_scaling_event > NOW() - INTERVAL '1 minute' * queue_config.scale_up_cooldown_minutes THEN
            cooldown_remaining := EXTRACT(EPOCH FROM (
                last_scaling_event + INTERVAL '1 minute' * queue_config.scale_up_cooldown_minutes - NOW()
            )) / 60;
            RETURN QUERY SELECT 'no_action'::VARCHAR(20), 
                format('Scale up cooldown: %s minutes remaining', cooldown_remaining)::TEXT, 
                current_consumers;
        ELSE
            RETURN QUERY SELECT 'scale_up'::VARCHAR(20), 
                format('Queue depth %s > threshold %s', current_metrics.messages_total, queue_config.scale_up_threshold)::TEXT,
                LEAST(current_consumers + 1, queue_config.max_consumers);
        END IF;
        
    ELSIF current_metrics.messages_total < queue_config.scale_down_threshold THEN
        -- Check if we can scale down
        IF current_consumers <= queue_config.min_consumers THEN
            RETURN QUERY SELECT 'no_action'::VARCHAR(20), 'Already at minimum consumers'::TEXT, current_consumers;
        ELSIF last_scaling_event IS NOT NULL AND 
              last_scaling_event > NOW() - INTERVAL '1 minute' * queue_config.scale_down_cooldown_minutes THEN
            cooldown_remaining := EXTRACT(EPOCH FROM (
                last_scaling_event + INTERVAL '1 minute' * queue_config.scale_down_cooldown_minutes - NOW()
            )) / 60;
            RETURN QUERY SELECT 'no_action'::VARCHAR(20), 
                format('Scale down cooldown: %s minutes remaining', cooldown_remaining)::TEXT, 
                current_consumers;
        ELSE
            RETURN QUERY SELECT 'scale_down'::VARCHAR(20), 
                format('Queue depth %s < threshold %s', current_metrics.messages_total, queue_config.scale_down_threshold)::TEXT,
                GREATEST(current_consumers - 1, queue_config.min_consumers);
        END IF;
    ELSE
        RETURN QUERY SELECT 'no_action'::VARCHAR(20), 'Queue depth within normal range'::TEXT, current_consumers;
    END IF;
END;
$$ LANGUAGE plpgsql;

-- Function to record scaling event
CREATE OR REPLACE FUNCTION record_scaling_event(
    p_queue_name VARCHAR(100),
    p_action VARCHAR(20),
    p_reason TEXT,
    p_consumers_before INTEGER,
    p_consumers_after INTEGER
) RETURNS UUID AS $$
DECLARE
    event_id UUID;
    queue_config rabbitmq_queues%ROWTYPE;
    current_metrics queue_metrics%ROWTYPE;
BEGIN
    -- Get queue config and metrics
    SELECT * INTO queue_config FROM rabbitmq_queues WHERE queue_name = p_queue_name;
    SELECT * INTO current_metrics FROM queue_metrics 
    WHERE queue_name = p_queue_name ORDER BY metric_timestamp DESC LIMIT 1;
    
    -- Insert scaling event
    INSERT INTO consumer_scaling_events (
        queue_name, scaling_action, trigger_reason,
        consumers_before, consumers_after,
        queue_depth_before, avg_processing_time_before,
        queue_depth_threshold, processing_time_threshold, error_rate_threshold,
        min_consumers, max_consumers
    ) VALUES (
        p_queue_name, p_action, p_reason,
        p_consumers_before, p_consumers_after,
        COALESCE(current_metrics.messages_total, 0),
        COALESCE(current_metrics.avg_processing_time_ms, 0),
        CASE WHEN p_action = 'scale_up' THEN queue_config.scale_up_threshold 
             ELSE queue_config.scale_down_threshold END,
        queue_config.latency_threshold_ms,
        queue_config.error_rate_threshold,
        queue_config.min_consumers,
        queue_config.max_consumers
    ) RETURNING id INTO event_id;
    
    RETURN event_id;
END;
$$ LANGUAGE plpgsql;

-- ===================================================================
-- INDEXES FOR PERFORMANCE
-- ===================================================================

-- Queue message processing indexes
CREATE INDEX idx_queue_message_processing_queue_status ON queue_message_processing(queue_name, status, received_at);
CREATE INDEX idx_queue_message_processing_message_id ON queue_message_processing(message_id);
CREATE INDEX idx_queue_message_processing_interaction ON queue_message_processing(interaction_id) WHERE interaction_id IS NOT NULL;
CREATE INDEX idx_queue_message_processing_worker ON queue_message_processing(worker_id, status);
CREATE INDEX idx_queue_message_processing_timing ON queue_message_processing(received_at, completed_at) WHERE status = 'completed';

-- Consumer management indexes
CREATE INDEX idx_rabbitmq_consumers_queue_status ON rabbitmq_consumers(queue_name, status);
CREATE INDEX idx_rabbitmq_consumers_heartbeat ON rabbitmq_consumers(last_heartbeat) WHERE status = 'active';
CREATE INDEX idx_rabbitmq_consumers_worker ON rabbitmq_consumers(worker_id);

-- Metrics and monitoring indexes
CREATE INDEX idx_queue_metrics_queue_timestamp ON queue_metrics(queue_name, metric_timestamp DESC);
CREATE INDEX idx_queue_metrics_timestamp ON queue_metrics(metric_timestamp DESC);

-- Scaling events indexes
CREATE INDEX idx_consumer_scaling_events_queue_time ON consumer_scaling_events(queue_name, created_at DESC);
CREATE INDEX idx_consumer_scaling_events_executed ON consumer_scaling_events(executed, created_at DESC);

-- Dead letter indexes
CREATE INDEX idx_dead_letter_messages_queue_reason ON dead_letter_messages(original_queue_name, death_reason);
CREATE INDEX idx_dead_letter_messages_recovery ON dead_letter_messages(recovery_status, first_death_at);
CREATE INDEX idx_dead_letter_messages_interaction ON dead_letter_messages(interaction_id) WHERE interaction_id IS NOT NULL;

-- ===================================================================
-- INITIAL CONFIGURATION DATA
-- ===================================================================

-- Default exchanges
INSERT INTO rabbitmq_exchanges (exchange_name, exchange_type, description) VALUES
('ai.conversation.direct', 'direct', 'Direct exchange for AI conversation processing'),
('ai.conversation.topic', 'topic', 'Topic exchange for AI conversation routing'),
('ai.dlx', 'direct', 'Dead letter exchange for failed messages'),
('ai.retry', 'direct', 'Retry exchange for message reprocessing');

-- Default queues for AI conversation system
INSERT INTO rabbitmq_queues (
    queue_name, exchange_name, routing_key, description,
    max_priority, message_ttl_ms, max_length,
    prefetch_count, consumer_timeout_ms,
    dead_letter_exchange, dead_letter_routing_key,
    min_consumers, max_consumers,
    scale_up_threshold, scale_down_threshold,
    latency_threshold_ms, error_rate_threshold
) VALUES
-- High priority queue for urgent requests
('ai.conversation.high_priority', 'ai.conversation.direct', 'ai.high', 
 'High priority AI conversation processing', 
 10, 1800000, 1000, 5, 180000, 'ai.dlx', 'ai.high.failed', 2, 20, 50, 5, 15000, 0.03),

-- Standard processing queue
('ai.conversation.standard', 'ai.conversation.direct', 'ai.standard', 
 'Standard AI conversation processing', 
 5, 3600000, 5000, 10, 300000, 'ai.dlx', 'ai.standard.failed', 3, 50, 200, 20, 30000, 0.05),

-- Low priority queue for batch processing
('ai.conversation.low_priority', 'ai.conversation.direct', 'ai.low', 
 'Low priority AI conversation processing', 
 1, 7200000, 10000, 20, 600000, 'ai.dlx', 'ai.low.failed', 1, 10, 500, 50, 60000, 0.10),

-- Analytics processing queue
('ai.analytics.processing', 'ai.conversation.topic', 'analytics.*', 
 'AI conversation analytics processing', 
 3, 3600000, 2000, 15, 300000, 'ai.dlx', 'analytics.failed', 1, 5, 100, 10, 45000, 0.05),

-- Dead letter queue
('ai.conversation.dlq', 'ai.dlx', 'ai.*.failed', 
 'Dead letter queue for failed AI conversations', 
 0, NULL, 50000, 1, 300000, NULL, NULL, 1, 3, 1000, 0, 30000, 0.01);

COMMIT;