-- =====================================================
-- TODO APPLICATION DATABASE SETUP SCRIPT
-- Complete setup script for production deployment
-- =====================================================

-- Check PostgreSQL version (requires 13+)
DO $$
BEGIN
    IF current_setting('server_version_num')::integer < 130000 THEN
        RAISE EXCEPTION 'PostgreSQL version 13 or higher is required. Current version: %', 
                        current_setting('server_version');
    END IF;
END
$$;

-- =====================================================
-- DATABASE CONFIGURATION
-- =====================================================

-- Set timezone to UTC for consistency
SET timezone = 'UTC';

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";
CREATE EXTENSION IF NOT EXISTS "pg_stat_statements"; -- For query performance monitoring

-- Create custom schemas for organization (optional)
-- CREATE SCHEMA IF NOT EXISTS todo_app;
-- SET search_path TO todo_app, public;

-- =====================================================
-- PERFORMANCE SETTINGS (Apply to postgresql.conf)
-- =====================================================

/*
Recommended postgresql.conf settings for production:

# Memory Settings
shared_buffers = 256MB                      # 25% of available RAM
effective_cache_size = 1GB                  # 75% of available RAM
work_mem = 4MB                              # Per query working memory
maintenance_work_mem = 64MB                 # For maintenance operations

# Storage Settings
random_page_cost = 1.1                      # For SSD storage
effective_io_concurrency = 200              # For SSD storage
checkpoint_completion_target = 0.9          # Spread checkpoints

# WAL Settings (for replication/backup)
wal_level = replica
archive_mode = on
archive_command = 'cp %p /path/to/archive/%f'
max_wal_senders = 3
wal_keep_segments = 32

# Connection Settings
max_connections = 200                       # Adjust based on needs
*/

-- =====================================================
-- MONITORING AND STATISTICS
-- =====================================================

-- Enable query statistics collection
SELECT pg_stat_statements_reset();

-- Create monitoring views
CREATE OR REPLACE VIEW performance_overview AS
SELECT
    'Database Size' as metric,
    pg_size_pretty(pg_database_size(current_database())) as value,
    'Total database size including indexes' as description
UNION ALL
SELECT
    'Active Connections',
    COUNT(*)::text,
    'Current active database connections'
FROM pg_stat_activity
WHERE state = 'active'
UNION ALL
SELECT
    'Table Count',
    COUNT(*)::text,
    'Total number of tables'
FROM information_schema.tables
WHERE table_schema = 'public'
UNION ALL
SELECT
    'Index Count',
    COUNT(*)::text,
    'Total number of indexes'
FROM pg_indexes
WHERE schemaname = 'public';

-- Table size monitoring
CREATE OR REPLACE VIEW table_sizes AS
SELECT
    schemaname,
    tablename,
    pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) as total_size,
    pg_size_pretty(pg_relation_size(schemaname||'.'||tablename)) as table_size,
    pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename) - pg_relation_size(schemaname||'.'||tablename)) as index_size,
    pg_total_relation_size(schemaname||'.'||tablename) as total_bytes
FROM pg_tables
WHERE schemaname = 'public'
ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC;

-- Index usage monitoring
CREATE OR REPLACE VIEW index_usage AS
SELECT
    schemaname,
    tablename,
    indexname,
    idx_tup_read,
    idx_tup_fetch,
    idx_scan,
    CASE 
        WHEN idx_scan = 0 THEN 'Never Used'
        WHEN idx_scan < 10 THEN 'Rarely Used'
        WHEN idx_scan < 100 THEN 'Sometimes Used'
        ELSE 'Frequently Used'
    END as usage_category
FROM pg_stat_user_indexes
ORDER BY idx_scan DESC;

-- Slow query monitoring (requires pg_stat_statements)
CREATE OR REPLACE VIEW slow_queries AS
SELECT
    query,
    calls,
    total_time,
    mean_time,
    rows,
    100.0 * shared_blks_hit / nullif(shared_blks_hit + shared_blks_read, 0) AS hit_percent
FROM pg_stat_statements
WHERE calls > 1
ORDER BY mean_time DESC
LIMIT 20;

-- =====================================================
-- MAINTENANCE PROCEDURES
-- =====================================================

-- Comprehensive maintenance function
CREATE OR REPLACE FUNCTION perform_maintenance(
    archive_days INTEGER DEFAULT 90,
    log_retention_days INTEGER DEFAULT 365,
    vacuum_analyze BOOLEAN DEFAULT true
)
RETURNS TEXT AS $$
DECLARE
    result TEXT := '';
    archived_todos INTEGER;
    cleaned_logs INTEGER;
BEGIN
    -- Archive old completed todos
    SELECT archive_old_completed_todos(archive_days) INTO archived_todos;
    result := result || format('Archived %s old completed todos. ', archived_todos);
    
    -- Clean old activity logs
    SELECT cleanup_old_activity_logs(log_retention_days) INTO cleaned_logs;
    result := result || format('Cleaned %s old activity log entries. ', cleaned_logs);
    
    -- Vacuum and analyze if requested
    IF vacuum_analyze THEN
        VACUUM ANALYZE;
        result := result || 'Performed VACUUM ANALYZE. ';
    END IF;
    
    -- Update statistics
    ANALYZE;
    result := result || 'Updated table statistics. ';
    
    RETURN result || format('Maintenance completed at %s.', CURRENT_TIMESTAMP);
END;
$$ LANGUAGE plpgsql;

-- =====================================================
-- BACKUP AND RECOVERY HELPERS
-- =====================================================

-- Function to get backup information
CREATE OR REPLACE FUNCTION backup_info()
RETURNS TABLE(
    metric TEXT,
    value TEXT,
    description TEXT
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        'Last Checkpoint'::TEXT,
        checkpoint_time::TEXT,
        'Time of last checkpoint'::TEXT
    FROM pg_control_checkpoint()
    UNION ALL
    SELECT
        'WAL Location',
        pg_current_wal_lsn()::TEXT,
        'Current WAL write position'
    UNION ALL
    SELECT
        'Database Size',
        pg_size_pretty(pg_database_size(current_database())),
        'Total size for backup planning'
    UNION ALL
    SELECT
        'Active Transactions',
        COUNT(*)::TEXT,
        'Transactions that would block backup'
    FROM pg_stat_activity
    WHERE state IN ('active', 'idle in transaction');
END;
$$ LANGUAGE plpgsql;

-- =====================================================
-- HEALTH CHECK FUNCTIONS
-- =====================================================

-- Comprehensive health check
CREATE OR REPLACE FUNCTION health_check()
RETURNS TABLE(
    check_name TEXT,
    status TEXT,
    details TEXT,
    recommendation TEXT
) AS $$
BEGIN
    -- Check for orphaned records
    RETURN QUERY
    WITH orphan_checks AS (
        SELECT 'Orphaned Todos' as check_name,
               CASE WHEN COUNT(*) = 0 THEN 'PASS' ELSE 'FAIL' END as status,
               'Found ' || COUNT(*) || ' todos without valid projects' as details,
               CASE WHEN COUNT(*) > 0 THEN 'Investigate and fix data integrity' ELSE 'No action needed' END as recommendation
        FROM todos t
        LEFT JOIN projects p ON t.project_id = p.id
        WHERE p.id IS NULL
        
        UNION ALL
        
        SELECT 'Orphaned Subtasks',
               CASE WHEN COUNT(*) = 0 THEN 'PASS' ELSE 'FAIL' END,
               'Found ' || COUNT(*) || ' subtasks without valid todos',
               CASE WHEN COUNT(*) > 0 THEN 'Investigate and fix data integrity' ELSE 'No action needed' END
        FROM subtasks st
        LEFT JOIN todos t ON st.todo_id = t.id
        WHERE t.id IS NULL
        
        UNION ALL
        
        SELECT 'Large Tables',
               CASE WHEN MAX(total_bytes) > 1073741824 THEN 'WARNING' ELSE 'PASS' END,
               'Largest table: ' || pg_size_pretty(MAX(total_bytes)),
               CASE WHEN MAX(total_bytes) > 1073741824 THEN 'Consider partitioning or archiving' ELSE 'Table sizes acceptable' END
        FROM (SELECT pg_total_relation_size('public.' || tablename) as total_bytes FROM pg_tables WHERE schemaname = 'public') sizes
        
        UNION ALL
        
        SELECT 'Unused Indexes',
               CASE WHEN COUNT(*) > 0 THEN 'WARNING' ELSE 'PASS' END,
               'Found ' || COUNT(*) || ' indexes with zero scans',
               CASE WHEN COUNT(*) > 0 THEN 'Review and consider dropping unused indexes' ELSE 'All indexes are being used' END
        FROM pg_stat_user_indexes
        WHERE idx_scan = 0
        
        UNION ALL
        
        SELECT 'Connection Count',
               CASE WHEN COUNT(*) > 150 THEN 'WARNING' ELSE 'PASS' END,
               'Current connections: ' || COUNT(*),
               CASE WHEN COUNT(*) > 150 THEN 'Consider connection pooling' ELSE 'Connection count normal' END
        FROM pg_stat_activity
        WHERE state = 'active'
    )
    SELECT * FROM orphan_checks;
END;
$$ LANGUAGE plpgsql;

-- =====================================================
-- DEPLOYMENT VALIDATION
-- =====================================================

-- Validate deployment
CREATE OR REPLACE FUNCTION validate_deployment()
RETURNS TEXT AS $$
DECLARE
    table_count INTEGER;
    index_count INTEGER;
    function_count INTEGER;
    trigger_count INTEGER;
    constraint_count INTEGER;
    result TEXT := '';
BEGIN
    -- Count database objects
    SELECT COUNT(*) INTO table_count FROM pg_tables WHERE schemaname = 'public';
    SELECT COUNT(*) INTO index_count FROM pg_indexes WHERE schemaname = 'public';
    SELECT COUNT(*) INTO function_count FROM pg_proc p JOIN pg_namespace n ON p.pronamespace = n.oid WHERE n.nspname = 'public';
    SELECT COUNT(*) INTO trigger_count FROM pg_trigger WHERE NOT tgisinternal;
    SELECT COUNT(*) INTO constraint_count FROM pg_constraint;
    
    result := format('Deployment Validation Results:%s', chr(10));
    result := result || format('- Tables: %s (expected: 11)%s', table_count, chr(10));
    result := result || format('- Indexes: %s (expected: 30+)%s', index_count, chr(10));
    result := result || format('- Functions: %s (expected: 8+)%s', function_count, chr(10));
    result := result || format('- Triggers: %s (expected: 10+)%s', trigger_count, chr(10));
    result := result || format('- Constraints: %s%s', constraint_count, chr(10));
    
    -- Validate critical tables exist
    IF NOT EXISTS (SELECT 1 FROM pg_tables WHERE tablename = 'users' AND schemaname = 'public') THEN
        result := result || 'ERROR: users table missing!' || chr(10);
    END IF;
    
    IF NOT EXISTS (SELECT 1 FROM pg_tables WHERE tablename = 'todos' AND schemaname = 'public') THEN
        result := result || 'ERROR: todos table missing!' || chr(10);
    END IF;
    
    IF NOT EXISTS (SELECT 1 FROM pg_tables WHERE tablename = 'projects' AND schemaname = 'public') THEN
        result := result || 'ERROR: projects table missing!' || chr(10);
    END IF;
    
    -- Validate critical indexes exist
    IF NOT EXISTS (SELECT 1 FROM pg_indexes WHERE indexname = 'idx_todos_user_id' AND schemaname = 'public') THEN
        result := result || 'WARNING: Critical index idx_todos_user_id missing!' || chr(10);
    END IF;
    
    -- Test sample data insertion (and rollback)
    BEGIN
        INSERT INTO users (email, username, password_hash) VALUES ('test@example.com', 'testuser', 'hashedpassword');
        result := result || 'Sample data insertion: SUCCESS' || chr(10);
        ROLLBACK;
    EXCEPTION WHEN OTHERS THEN
        result := result || 'Sample data insertion: FAILED - ' || SQLERRM || chr(10);
        ROLLBACK;
    END;
    
    result := result || format('Validation completed at: %s', CURRENT_TIMESTAMP);
    
    RETURN result;
END;
$$ LANGUAGE plpgsql;

-- =====================================================
-- INITIAL SETUP COMPLETION
-- =====================================================

-- Load the main schema
\i todo_app_schema.sql

-- Run initial validation
SELECT validate_deployment();

-- Display setup summary
SELECT 
    'Todo Application Database Setup Complete' as message,
    CURRENT_TIMESTAMP as completed_at,
    current_database() as database_name,
    current_user as setup_user,
    version() as postgresql_version;

-- Show table summary
SELECT 
    'Created Tables:' as summary,
    STRING_AGG(tablename, ', ' ORDER BY tablename) as tables
FROM pg_tables 
WHERE schemaname = 'public';

-- Show index summary
SELECT 
    'Created Indexes:' as summary,
    COUNT(*) as index_count
FROM pg_indexes 
WHERE schemaname = 'public';

-- =====================================================
-- POST-SETUP RECOMMENDATIONS
-- =====================================================

DO $$
BEGIN
    RAISE NOTICE '
=====================================
TODO APPLICATION SETUP COMPLETE
=====================================

Next Steps:
1. Configure postgresql.conf with recommended settings (see comments above)
2. Set up regular backups using pg_dump or pg_basebackup
3. Configure connection pooling (PgBouncer recommended)
4. Set up monitoring alerts for disk space and performance
5. Schedule regular maintenance: SELECT perform_maintenance();
6. Test application connectivity and basic CRUD operations

Performance Monitoring:
- SELECT * FROM performance_overview;
- SELECT * FROM table_sizes;
- SELECT * FROM index_usage;
- SELECT * FROM slow_queries;

Health Checks:
- SELECT * FROM health_check();
- SELECT backup_info();

For production deployment:
1. Change default passwords
2. Configure SSL/TLS
3. Set up replication if needed
4. Configure log rotation
5. Set up monitoring (Prometheus/Grafana recommended)

Documentation: See TODO_APP_SCHEMA_DOCUMENTATION.md
Schema File: todo_app_schema.sql
=====================================
';
END
$$;

-- =====================================================
-- OPTIONAL: CREATE APPLICATION USER
-- =====================================================

-- Uncomment and modify for production use
/*
-- Create application database user
CREATE USER todo_app_user WITH PASSWORD 'your_secure_password_here';

-- Grant necessary permissions
GRANT CONNECT ON DATABASE todo_app_db TO todo_app_user;
GRANT USAGE ON SCHEMA public TO todo_app_user;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO todo_app_user;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO todo_app_user;
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA public TO todo_app_user;

-- Grant permissions on future objects
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO todo_app_user;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT USAGE, SELECT ON SEQUENCES TO todo_app_user;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT EXECUTE ON FUNCTIONS TO todo_app_user;
*/

-- =====================================================
-- SECURITY HARDENING (PRODUCTION)
-- =====================================================

-- Revoke public schema permissions (uncomment for production)
-- REVOKE CREATE ON SCHEMA public FROM PUBLIC;
-- REVOKE ALL ON DATABASE todo_app_db FROM PUBLIC;

-- Create read-only user for analytics/reporting
-- CREATE USER todo_readonly WITH PASSWORD 'readonly_password';
-- GRANT CONNECT ON DATABASE todo_app_db TO todo_readonly;
-- GRANT USAGE ON SCHEMA public TO todo_readonly;
-- GRANT SELECT ON ALL TABLES IN SCHEMA public TO todo_readonly;

-- =====================================================
-- SETUP COMPLETE
-- =====================================================