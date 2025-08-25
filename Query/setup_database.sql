-- ===================================================================
-- AI CONVERSATION SYSTEM - SETUP SCRIPT
-- ===================================================================
-- Run this script to set up the complete AI conversation system
-- Execute in order: 1) This file 2) ai_conversation_schema.sql 
-- 3) redis_cache_config.sql 4) rabbitmq_queue_schema.sql
-- ===================================================================

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_stat_statements";
CREATE EXTENSION IF NOT EXISTS "btree_gin";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";

-- Create users table (if doesn't exist)
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    username VARCHAR(100) UNIQUE,
    tier VARCHAR(20) DEFAULT 'standard' CHECK (tier IN ('free', 'standard', 'premium', 'enterprise')),
    daily_budget DECIMAL(10,4) DEFAULT 10.0000,
    monthly_budget DECIMAL(10,4) DEFAULT 300.0000,
    api_key_hash VARCHAR(255),
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create application user for database connections
DO $$
BEGIN
    IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'ai_app_user') THEN
        CREATE ROLE ai_app_user WITH LOGIN PASSWORD 'your_secure_password_here';
    END IF;
END
$$;

-- Create read-only user for analytics
DO $$
BEGIN
    IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'ai_read_user') THEN
        CREATE ROLE ai_read_user WITH LOGIN PASSWORD 'your_readonly_password_here';
    END IF;
END
$$;

-- Performance configuration
ALTER SYSTEM SET shared_preload_libraries = 'pg_stat_statements';
ALTER SYSTEM SET track_activity_query_size = 2048;
ALTER SYSTEM SET track_io_timing = on;
ALTER SYSTEM SET log_min_duration_statement = 1000; -- Log slow queries
ALTER SYSTEM SET log_line_prefix = '%t [%p]: [%l-1] user=%u,db=%d,app=%a,client=%h ';

-- Memory settings (adjust based on your server)
ALTER SYSTEM SET shared_buffers = '256MB';
ALTER SYSTEM SET effective_cache_size = '1GB';
ALTER SYSTEM SET maintenance_work_mem = '64MB';
ALTER SYSTEM SET checkpoint_completion_target = 0.9;
ALTER SYSTEM SET wal_buffers = '16MB';

-- Connection settings
ALTER SYSTEM SET max_connections = 200;
ALTER SYSTEM SET max_prepared_transactions = 100;

-- Create schemas for organization
CREATE SCHEMA IF NOT EXISTS ai_core;
CREATE SCHEMA IF NOT EXISTS ai_cache;
CREATE SCHEMA IF NOT EXISTS ai_queue;
CREATE SCHEMA IF NOT EXISTS ai_analytics;

-- Grant permissions
GRANT USAGE ON SCHEMA public TO ai_app_user;
GRANT USAGE ON SCHEMA ai_core TO ai_app_user;
GRANT USAGE ON SCHEMA ai_cache TO ai_app_user;
GRANT USAGE ON SCHEMA ai_queue TO ai_app_user;
GRANT USAGE ON SCHEMA ai_analytics TO ai_app_user;

GRANT USAGE ON SCHEMA public TO ai_read_user;
GRANT USAGE ON SCHEMA ai_core TO ai_read_user;
GRANT USAGE ON SCHEMA ai_cache TO ai_read_user;
GRANT USAGE ON SCHEMA ai_queue TO ai_read_user;
GRANT USAGE ON SCHEMA ai_analytics TO ai_read_user;

-- Enable row-level security
ALTER TABLE users ENABLE ROW LEVEL SECURITY;

-- Create RLS policy for users (users can only see their own data)
CREATE POLICY user_isolation_policy ON users
    FOR ALL TO ai_app_user
    USING (id = current_setting('app.current_user_id', true)::uuid);

-- Function to set current user context
CREATE OR REPLACE FUNCTION set_current_user_id(user_uuid UUID)
RETURNS void AS $$
BEGIN
    PERFORM set_config('app.current_user_id', user_uuid::text, false);
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

-- Function to get current user context
CREATE OR REPLACE FUNCTION get_current_user_id()
RETURNS UUID AS $$
BEGIN
    RETURN current_setting('app.current_user_id', true)::uuid;
EXCEPTION WHEN OTHERS THEN
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

-- Create monitoring views
CREATE OR REPLACE VIEW v_system_health AS
SELECT 
    'database' as component,
    CASE 
        WHEN pg_is_in_recovery() THEN 'replica'
        ELSE 'primary'
    END as role,
    CASE 
        WHEN (SELECT count(*) FROM pg_stat_activity WHERE state = 'active') < 50 THEN 'healthy'
        WHEN (SELECT count(*) FROM pg_stat_activity WHERE state = 'active') < 100 THEN 'warning'
        ELSE 'critical'
    END as status,
    (SELECT count(*) FROM pg_stat_activity WHERE state = 'active') as active_connections,
    (SELECT setting::int FROM pg_settings WHERE name = 'max_connections') as max_connections,
    pg_size_pretty(pg_database_size(current_database())) as database_size,
    NOW() as checked_at;

-- Insert sample data (optional, for testing)
INSERT INTO users (email, username, tier, daily_budget) VALUES
('admin@example.com', 'admin', 'enterprise', 1000.0000),
('user1@example.com', 'user1', 'premium', 100.0000),
('user2@example.com', 'user2', 'standard', 10.0000)
ON CONFLICT (email) DO NOTHING;

-- Create maintenance procedures
CREATE OR REPLACE FUNCTION maintenance_cleanup()
RETURNS TABLE(
    task TEXT,
    records_affected BIGINT,
    space_freed TEXT
) AS $$
DECLARE
    cleanup_result RECORD;
BEGIN
    -- Clean up expired cache entries
    WITH cache_cleanup AS (
        DELETE FROM cache_entries 
        WHERE expires_at IS NOT NULL AND expires_at < NOW()
        RETURNING size_bytes
    )
    SELECT 'cache_cleanup' as task,
           COUNT(*) as records_affected,
           pg_size_pretty(SUM(size_bytes)) as space_freed
    INTO cleanup_result
    FROM cache_cleanup;
    
    RETURN QUERY SELECT cleanup_result.task, cleanup_result.records_affected, cleanup_result.space_freed;
    
    -- Vacuum analyze for performance
    VACUUM ANALYZE;
    
    RETURN QUERY SELECT 'vacuum_analyze'::TEXT, 0::BIGINT, '0 bytes'::TEXT;
    
    -- Update statistics
    ANALYZE;
    
    RETURN QUERY SELECT 'update_statistics'::TEXT, 0::BIGINT, '0 bytes'::TEXT;
END;
$$ LANGUAGE plpgsql;

-- Create backup verification function
CREATE OR REPLACE FUNCTION verify_backup_integrity()
RETURNS TABLE(
    table_name TEXT,
    record_count BIGINT,
    last_modified TIMESTAMP WITH TIME ZONE
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        t.table_name::TEXT,
        t.record_count,
        t.last_modified
    FROM (
        SELECT 'ai_conversations' as table_name,
               COUNT(*) as record_count,
               MAX(updated_at) as last_modified
        FROM ai_conversations
        
        UNION ALL
        
        SELECT 'ai_interactions' as table_name,
               COUNT(*) as record_count,
               MAX(created_at) as last_modified
        FROM ai_interactions
        
        UNION ALL
        
        SELECT 'cache_entries' as table_name,
               COUNT(*) as record_count,
               MAX(last_accessed_at) as last_modified
        FROM cache_entries
        
        UNION ALL
        
        SELECT 'queue_messages' as table_name,
               COUNT(*) as record_count,
               MAX(created_at) as last_modified
        FROM queue_messages
    ) t;
END;
$$ LANGUAGE plpgsql;

-- Grant execution permissions
GRANT EXECUTE ON FUNCTION set_current_user_id(UUID) TO ai_app_user;
GRANT EXECUTE ON FUNCTION get_current_user_id() TO ai_app_user;
GRANT EXECUTE ON FUNCTION maintenance_cleanup() TO ai_app_user;
GRANT EXECUTE ON FUNCTION verify_backup_integrity() TO ai_read_user;

-- Log setup completion
DO $$
BEGIN
    RAISE NOTICE 'AI Conversation System setup completed successfully!';
    RAISE NOTICE 'Next steps:';
    RAISE NOTICE '1. Run ai_conversation_schema.sql';
    RAISE NOTICE '2. Run redis_cache_config.sql';
    RAISE NOTICE '3. Run rabbitmq_queue_schema.sql';
    RAISE NOTICE '4. Configure application connection strings';
    RAISE NOTICE '5. Set up Redis and RabbitMQ services';
END $$;

COMMIT;