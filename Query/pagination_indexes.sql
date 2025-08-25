-- Optimized Database Indexes for Pagination Performance
-- Run this script to add indexes that will significantly improve pagination query performance

-- ============================================================================
-- PROJECT PAGINATION INDEXES
-- ============================================================================

-- Composite index for user-based queries with common filters
-- This index will speed up the most common pagination queries
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_projects_user_pagination
ON projects (user_id, is_archived, is_favorite, updated_at DESC);

-- Index for category-based filtering with pagination
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_projects_category_pagination
ON projects (user_id, category, is_archived, updated_at DESC);

-- Index for title-based sorting (used when sort=title)
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_projects_title_sort
ON projects (user_id, is_archived, title);

-- Index for created_at sorting (used when sort=created_at)
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_projects_created_sort
ON projects (user_id, is_archived, created_at DESC);

-- Index for progress percentage sorting (used when sort=progress_percentage)
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_projects_progress_sort
ON projects (user_id, is_archived, progress_percentage DESC);

-- Full-text search indexes for title and description
-- These will speed up LIKE queries for search functionality
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_projects_title_search
ON projects USING gin (to_tsvector('english', COALESCE(title, '')));

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_projects_description_search
ON projects USING gin (to_tsvector('english', COALESCE(description, '')));

-- Combined full-text search index for both title and description
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_projects_fulltext_search
ON projects USING gin (to_tsvector('english', 
    COALESCE(title, '') || ' ' || COALESCE(description, '')
));

-- ============================================================================
-- TODO COUNT OPTIMIZATION INDEXES
-- ============================================================================

-- Index for efficiently counting todos per project
-- This speeds up the todo count queries when include_counts=true
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_todos_project_count
ON todos (project_id, is_archived, status);

-- Index for completed todos count
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_todos_completed_count
ON todos (project_id, status) 
WHERE status = 'completed' AND is_archived = false;

-- ============================================================================
-- PERFORMANCE MONITORING QUERIES
-- ============================================================================

-- Use these queries to monitor index usage and performance

-- 1. Check index usage statistics
-- SELECT 
--     schemaname,
--     tablename,
--     indexname,
--     idx_scan,
--     idx_tup_read,
--     idx_tup_fetch
-- FROM pg_stat_user_indexes 
-- WHERE tablename IN ('projects', 'todos')
-- ORDER BY idx_scan DESC;

-- 2. Check index sizes
-- SELECT 
--     schemaname,
--     tablename,
--     indexname,
--     pg_size_pretty(pg_relation_size(indexrelid)) as index_size
-- FROM pg_stat_user_indexes 
-- WHERE tablename IN ('projects', 'todos')
-- ORDER BY pg_relation_size(indexrelid) DESC;

-- 3. Monitor slow queries (add to postgresql.conf: log_min_duration_statement = 1000)
-- This will log any query taking longer than 1 second

-- ============================================================================
-- QUERY PERFORMANCE EXPLANATIONS
-- ============================================================================

-- Example queries and their optimizations:

-- 1. Basic pagination query:
-- SELECT * FROM projects 
-- WHERE user_id = $1 AND is_archived = false 
-- ORDER BY is_favorite DESC, updated_at DESC 
-- LIMIT 20 OFFSET 0;
-- 
-- Optimized by: idx_projects_user_pagination

-- 2. Category filter with pagination:
-- SELECT * FROM projects 
-- WHERE user_id = $1 AND is_archived = false AND category = $2
-- ORDER BY updated_at DESC 
-- LIMIT 20 OFFSET 0;
-- 
-- Optimized by: idx_projects_category_pagination

-- 3. Search query:
-- SELECT * FROM projects 
-- WHERE user_id = $1 AND is_archived = false 
-- AND (LOWER(title) LIKE LOWER($2) OR LOWER(description) LIKE LOWER($2))
-- ORDER BY updated_at DESC 
-- LIMIT 20 OFFSET 0;
-- 
-- Optimized by: idx_projects_fulltext_search (for better performance, consider upgrading to full-text search)

-- 4. Todo count query:
-- SELECT project_id, COUNT(*) as total_count,
--        COUNT(CASE WHEN status = 'completed' THEN 1 END) as completed_count
-- FROM todos 
-- WHERE project_id = ANY($1) AND is_archived = false
-- GROUP BY project_id;
-- 
-- Optimized by: idx_todos_project_count

-- ============================================================================
-- MAINTENANCE COMMANDS
-- ============================================================================

-- Check for unused indexes (run periodically)
-- SELECT 
--     schemaname,
--     tablename,
--     indexname,
--     idx_scan
-- FROM pg_stat_user_indexes 
-- WHERE tablename IN ('projects', 'todos') AND idx_scan = 0;

-- Analyze tables after creating indexes
ANALYZE projects;
ANALYZE todos;

-- ============================================================================
-- ADVANCED OPTIMIZATION TIPS
-- ============================================================================

-- 1. Consider partitioning for very large datasets (millions of projects)
-- 2. Use pg_stat_statements extension to monitor query performance
-- 3. Consider materialized views for complex aggregations
-- 4. Monitor query plans with EXPLAIN ANALYZE for specific slow queries

-- Example monitoring query for pagination performance:
-- EXPLAIN (ANALYZE, BUFFERS) 
-- SELECT * FROM projects 
-- WHERE user_id = 'user-uuid' AND is_archived = false 
-- ORDER BY is_favorite DESC, updated_at DESC 
-- LIMIT 20 OFFSET 0;

-- ============================================================================
-- ROLLBACK COMMANDS (if needed)
-- ============================================================================

-- DROP INDEX CONCURRENTLY IF EXISTS idx_projects_user_pagination;
-- DROP INDEX CONCURRENTLY IF EXISTS idx_projects_category_pagination;
-- DROP INDEX CONCURRENTLY IF EXISTS idx_projects_title_sort;
-- DROP INDEX CONCURRENTLY IF EXISTS idx_projects_created_sort;
-- DROP INDEX CONCURRENTLY IF EXISTS idx_projects_progress_sort;
-- DROP INDEX CONCURRENTLY IF EXISTS idx_projects_title_search;
-- DROP INDEX CONCURRENTLY IF EXISTS idx_projects_description_search;
-- DROP INDEX CONCURRENTLY IF EXISTS idx_projects_fulltext_search;
-- DROP INDEX CONCURRENTLY IF EXISTS idx_todos_project_count;
-- DROP INDEX CONCURRENTLY IF EXISTS idx_todos_completed_count;

-- ============================================================================
-- NOTES
-- ============================================================================

-- 1. CONCURRENTLY option allows index creation without locking the table
-- 2. These indexes will use additional storage space but significantly improve query performance
-- 3. Monitor index usage and drop unused indexes to save space
-- 4. Consider running VACUUM and ANALYZE after creating indexes
-- 5. Test these indexes on a staging environment first

-- For very high-traffic applications, consider:
-- - Connection pooling (PgBouncer)
-- - Read replicas for read-heavy workloads
-- - Query result caching with Redis
-- - Database monitoring with tools like pg_stat_statements