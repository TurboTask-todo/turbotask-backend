-- =====================================================
-- HIGH-PERFORMANCE TODO LIST APPLICATION SCHEMA
-- PostgreSQL Database Schema with UUID, Indexing & Performance Optimization
-- =====================================================

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm"; -- For text search performance

-- =====================================================
-- ENUMS AND CUSTOM TYPES
-- =====================================================

-- Priority levels for tasks
CREATE TYPE priority_level AS ENUM ('low', 'medium', 'high', 'urgent');

-- Task status
CREATE TYPE task_status AS ENUM ('not_started', 'in_progress', 'pending', 'completed', 'cancelled', 'on_hold');

ALTER TYPE task_status ADD VALUE 'backlog' AFTER 'on_hold';
ALTER TYPE task_status ADD VALUE 'done' AFTER 'backlog';

-- Project categories
CREATE TYPE project_category AS ENUM ('personal', 'work', 'education', 'health', 'finance', 'hobby', 'other');

-- Time unit for estimates
CREATE TYPE time_unit AS ENUM ('minutes', 'hours', 'days');

-- Notification frequency
CREATE TYPE notification_frequency AS ENUM ('none', 'once', 'daily', 'weekly', 'monthly');

-- =====================================================
-- PROJECTS TABLE
-- =====================================================

CREATE TABLE projects (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    image_url TEXT,
    category project_category DEFAULT 'personal',
    color_theme VARCHAR(7) DEFAULT '#3498db', -- Hex color for UI theming
    is_archived BOOLEAN DEFAULT false,
    is_favorite BOOLEAN DEFAULT false,
    progress_percentage DECIMAL(5,2) DEFAULT 0.00, -- Calculated field for progress tracking
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- =====================================================
-- RELEASE VERSIONS TABLE
-- =====================================================

CREATE TABLE release_versions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    version_name VARCHAR(100) NOT NULL,
    version_number VARCHAR(20) NOT NULL, -- e.g., "v1.0.0", "1.2.3"
    description TEXT,
    target_date DATE,
    release_date DATE,
    is_released BOOLEAN DEFAULT false,
    release_notes TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(project_id, version_number)
);

-- =====================================================
-- TODOS TABLE (Main Tasks)
-- =====================================================

CREATE TABLE todos (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    release_version_id UUID REFERENCES release_versions(id) ON DELETE SET NULL,
    parent_todo_id UUID REFERENCES todos(id) ON DELETE CASCADE, -- For hierarchical tasks
    
    -- Core task fields
    task_name VARCHAR(255) NOT NULL,
    task_description TEXT,
    task_image_or_emoji TEXT, -- URL or Unicode emoji
    
    -- Time tracking
    estimated_time INTEGER, -- in minutes
    actual_time INTEGER DEFAULT 0, -- in minutes
    time_unit time_unit DEFAULT 'minutes',
    
    -- Priority and status
    priority priority_level DEFAULT 'medium',
    status task_status DEFAULT 'not_started',
    
    -- Progress tracking
    completion_percentage DECIMAL(5,2) DEFAULT 0.00,
    
    -- Scheduling and deadlines
    due_date TIMESTAMP WITH TIME ZONE,
    start_date TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    
    -- Additional optional fields for enhanced functionality
    tags TEXT[], -- Array of tags for categorization
    difficulty_rating INTEGER CHECK (difficulty_rating >= 1 AND difficulty_rating <= 10),
    energy_level_required INTEGER CHECK (energy_level_required >= 1 AND energy_level_required <= 5),
    location VARCHAR(255), -- Where the task should be performed
    context VARCHAR(100), -- Context like @calls, @computer, @errands
    
    -- Collaboration
    assigned_to UUID REFERENCES users(id) ON DELETE SET NULL,
    
    -- Metadata
    is_recurring BOOLEAN DEFAULT false,
    recurrence_pattern JSONB, -- Store recurrence rules in JSON format
    is_archived BOOLEAN DEFAULT false,
    is_pinned BOOLEAN DEFAULT false,
    
    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- =====================================================
-- SUBTASKS TABLE
-- =====================================================

CREATE TABLE subtasks (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    todo_id UUID NOT NULL REFERENCES todos(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    status task_status DEFAULT 'not_started',
    priority priority_level DEFAULT 'medium',
    estimated_time INTEGER, -- in minutes
    actual_time INTEGER DEFAULT 0,
    due_date TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    sort_order INTEGER DEFAULT 0,
    is_archived BOOLEAN DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- =====================================================
-- NOTES TABLE
-- =====================================================

CREATE TABLE notes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    todo_id UUID NOT NULL REFERENCES todos(id) ON DELETE CASCADE,
    title VARCHAR(255),
    content TEXT NOT NULL,
    note_type VARCHAR(50) DEFAULT 'general', -- general, meeting, idea, reminder
    is_pinned BOOLEAN DEFAULT false,
    tags TEXT[],
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- =====================================================
-- SCHEDULED TASKS TABLE
-- =====================================================

CREATE TABLE scheduled_tasks (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    todo_id UUID NOT NULL REFERENCES todos(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    scheduled_datetime TIMESTAMP WITH TIME ZONE NOT NULL,
    duration_minutes INTEGER,
    notification_sent BOOLEAN DEFAULT false,
    notification_frequency notification_frequency DEFAULT 'once',
    reminder_minutes_before INTEGER DEFAULT 15, -- Remind X minutes before
    location VARCHAR(255),
    meeting_url TEXT,
    attendees TEXT[], -- Array of email addresses
    is_all_day BOOLEAN DEFAULT false,
    is_cancelled BOOLEAN DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- =====================================================
-- TIME TRACKING TABLE
-- =====================================================

CREATE TABLE time_entries (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    todo_id UUID NOT NULL REFERENCES todos(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    start_time TIMESTAMP WITH TIME ZONE NOT NULL,
    end_time TIMESTAMP WITH TIME ZONE,
    duration_minutes INTEGER,
    description TEXT,
    is_billable BOOLEAN DEFAULT false,
    hourly_rate DECIMAL(10,2),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- =====================================================
-- COMMENTS TABLE (for collaboration)
-- =====================================================

CREATE TABLE comments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    todo_id UUID NOT NULL REFERENCES todos(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    parent_comment_id UUID REFERENCES comments(id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    is_edited BOOLEAN DEFAULT false,
    edited_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- =====================================================
-- ATTACHMENTS TABLE
-- =====================================================

CREATE TABLE attachments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    todo_id UUID REFERENCES todos(id) ON DELETE CASCADE,
    project_id UUID REFERENCES projects(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    filename VARCHAR(255) NOT NULL,
    original_filename VARCHAR(255) NOT NULL,
    file_path TEXT NOT NULL,
    file_size BIGINT NOT NULL,
    mime_type VARCHAR(100) NOT NULL,
    checksum VARCHAR(64), -- For file integrity
    is_image BOOLEAN DEFAULT false,
    thumbnail_path TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    -- Ensure attachment belongs to either todo or project, not both
    CONSTRAINT attachment_belongs_to_one CHECK (
        (todo_id IS NOT NULL AND project_id IS NULL) OR 
        (todo_id IS NULL AND project_id IS NOT NULL)
    )
);

-- =====================================================
-- ACTIVITY LOG TABLE (for audit trail)
-- =====================================================

CREATE TABLE activity_log (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    entity_type VARCHAR(50) NOT NULL, -- 'todo', 'project', 'subtask', etc.
    entity_id UUID NOT NULL,
    action VARCHAR(50) NOT NULL, -- 'created', 'updated', 'deleted', 'completed'
    old_values JSONB,
    new_values JSONB,
    ip_address INET,
    user_agent TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- =====================================================
-- INDEXES FOR PERFORMANCE OPTIMIZATION
-- =====================================================

-- Users table indexes
-- Projects table indexes
CREATE INDEX idx_projects_user_id ON projects(user_id);
CREATE INDEX idx_projects_category ON projects(category);
CREATE INDEX idx_projects_archived ON projects(is_archived) WHERE is_archived = false;
CREATE INDEX idx_projects_favorite ON projects(is_favorite) WHERE is_favorite = true;
CREATE INDEX idx_projects_title_gin ON projects USING gin(title gin_trgm_ops);

-- Release versions indexes
CREATE INDEX idx_release_versions_project_id ON release_versions(project_id);
CREATE INDEX idx_release_versions_released ON release_versions(is_released);
CREATE INDEX idx_release_versions_target_date ON release_versions(target_date);

-- Todos table indexes (most critical for performance)
CREATE INDEX idx_todos_user_id ON todos(user_id);
CREATE INDEX idx_todos_project_id ON todos(project_id);
CREATE INDEX idx_todos_release_version_id ON todos(release_version_id);
CREATE INDEX idx_todos_parent_todo_id ON todos(parent_todo_id);
CREATE INDEX idx_todos_status ON todos(status);
CREATE INDEX idx_todos_priority ON todos(priority);
CREATE INDEX idx_todos_due_date ON todos(due_date) WHERE due_date IS NOT NULL;
CREATE INDEX idx_todos_completed_at ON todos(completed_at) WHERE completed_at IS NOT NULL;
CREATE INDEX idx_todos_archived ON todos(is_archived) WHERE is_archived = false;
CREATE INDEX idx_todos_pinned ON todos(is_pinned) WHERE is_pinned = true;
CREATE INDEX idx_todos_assigned_to ON todos(assigned_to) WHERE assigned_to IS NOT NULL;
CREATE INDEX idx_todos_tags_gin ON todos USING gin(tags);
CREATE INDEX idx_todos_task_name_gin ON todos USING gin(task_name gin_trgm_ops);
CREATE INDEX idx_todos_created_at ON todos(created_at);

-- Composite indexes for common queries
CREATE INDEX idx_todos_user_project_status ON todos(user_id, project_id, status);
CREATE INDEX idx_todos_user_due_date ON todos(user_id, due_date) WHERE due_date IS NOT NULL;
CREATE INDEX idx_todos_project_priority ON todos(project_id, priority);

-- Subtasks indexes
CREATE INDEX idx_subtasks_todo_id ON subtasks(todo_id);
CREATE INDEX idx_subtasks_status ON subtasks(status);
CREATE INDEX idx_subtasks_sort_order ON subtasks(todo_id, sort_order);

-- Notes indexes
CREATE INDEX idx_notes_todo_id ON notes(todo_id);
CREATE INDEX idx_notes_content_gin ON notes USING gin(content gin_trgm_ops);
CREATE INDEX idx_notes_tags_gin ON notes USING gin(tags);

-- Scheduled tasks indexes
CREATE INDEX idx_scheduled_tasks_todo_id ON scheduled_tasks(todo_id);
CREATE INDEX idx_scheduled_tasks_user_id ON scheduled_tasks(user_id);
CREATE INDEX idx_scheduled_tasks_datetime ON scheduled_tasks(scheduled_datetime);
CREATE INDEX idx_scheduled_tasks_notification ON scheduled_tasks(notification_sent) WHERE notification_sent = false;

-- Time entries indexes
CREATE INDEX idx_time_entries_todo_id ON time_entries(todo_id);
CREATE INDEX idx_time_entries_user_id ON time_entries(user_id);
CREATE INDEX idx_time_entries_start_time ON time_entries(start_time);
CREATE INDEX idx_time_entries_billable ON time_entries(is_billable) WHERE is_billable = true;

-- Comments indexes
CREATE INDEX idx_comments_todo_id ON comments(todo_id);
CREATE INDEX idx_comments_user_id ON comments(user_id);
CREATE INDEX idx_comments_parent_id ON comments(parent_comment_id);

-- Attachments indexes
CREATE INDEX idx_attachments_todo_id ON attachments(todo_id) WHERE todo_id IS NOT NULL;
CREATE INDEX idx_attachments_project_id ON attachments(project_id) WHERE project_id IS NOT NULL;
CREATE INDEX idx_attachments_user_id ON attachments(user_id);

-- Activity log indexes
CREATE INDEX idx_activity_log_user_id ON activity_log(user_id);
CREATE INDEX idx_activity_log_entity ON activity_log(entity_type, entity_id);
CREATE INDEX idx_activity_log_created_at ON activity_log(created_at);

-- =====================================================
-- TRIGGERS FOR AUTOMATIC TIMESTAMP UPDATES
-- =====================================================

-- Function to update timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Apply triggers to all tables with updated_at
CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_projects_updated_at BEFORE UPDATE ON projects
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_release_versions_updated_at BEFORE UPDATE ON release_versions
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_todos_updated_at BEFORE UPDATE ON todos
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_subtasks_updated_at BEFORE UPDATE ON subtasks
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_notes_updated_at BEFORE UPDATE ON notes
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_scheduled_tasks_updated_at BEFORE UPDATE ON scheduled_tasks
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_time_entries_updated_at BEFORE UPDATE ON time_entries
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_comments_updated_at BEFORE UPDATE ON comments
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- =====================================================
-- FUNCTIONS FOR PROGRESS CALCULATION
-- =====================================================

-- Function to calculate project progress
CREATE OR REPLACE FUNCTION calculate_project_progress(p_project_id UUID)
RETURNS DECIMAL(5,2) AS $$
DECLARE
    total_todos INTEGER;
    completed_todos INTEGER;
    progress DECIMAL(5,2);
BEGIN
    SELECT COUNT(*) INTO total_todos 
    FROM todos 
    WHERE project_id = p_project_id AND is_archived = false;
    
    SELECT COUNT(*) INTO completed_todos 
    FROM todos 
    WHERE project_id = p_project_id AND status = 'completed' AND is_archived = false;
    
    IF total_todos = 0 THEN
        RETURN 0.00;
    END IF;
    
    progress := (completed_todos::DECIMAL / total_todos::DECIMAL) * 100;
    RETURN ROUND(progress, 2);
END;
$$ LANGUAGE plpgsql;

-- Function to calculate todo completion percentage based on subtasks
CREATE OR REPLACE FUNCTION calculate_todo_progress(p_todo_id UUID)
RETURNS DECIMAL(5,2) AS $$
DECLARE
    total_subtasks INTEGER;
    completed_subtasks INTEGER;
    progress DECIMAL(5,2);
BEGIN
    SELECT COUNT(*) INTO total_subtasks 
    FROM subtasks 
    WHERE todo_id = p_todo_id AND is_archived = false;
    
    IF total_subtasks = 0 THEN
        -- If no subtasks, return 100% if todo is completed, 0% otherwise
        SELECT CASE WHEN status = 'completed' THEN 100.00 ELSE 0.00 END INTO progress
        FROM todos WHERE id = p_todo_id;
        RETURN progress;
    END IF;
    
    SELECT COUNT(*) INTO completed_subtasks 
    FROM subtasks 
    WHERE todo_id = p_todo_id AND status = 'completed' AND is_archived = false;
    
    progress := (completed_subtasks::DECIMAL / total_subtasks::DECIMAL) * 100;
    RETURN ROUND(progress, 2);
END;
$$ LANGUAGE plpgsql;

-- =====================================================
-- TRIGGERS FOR PROGRESS UPDATES
-- =====================================================

-- Function to update project progress when todo status changes
CREATE OR REPLACE FUNCTION update_project_progress()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE projects 
    SET progress_percentage = calculate_project_progress(NEW.project_id)
    WHERE id = NEW.project_id;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Function to update todo progress when subtask status changes
CREATE OR REPLACE FUNCTION update_todo_progress()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE todos 
    SET completion_percentage = calculate_todo_progress(NEW.todo_id)
    WHERE id = NEW.todo_id;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Apply progress update triggers
CREATE TRIGGER trigger_update_project_progress 
    AFTER UPDATE OF status ON todos
    FOR EACH ROW EXECUTE FUNCTION update_project_progress();

CREATE TRIGGER trigger_update_todo_progress 
    AFTER UPDATE OF status ON subtasks
    FOR EACH ROW EXECUTE FUNCTION update_todo_progress();


-- =====================================================
-- VIEWS FOR COMMON QUERIES
-- =====================================================

-- View for todo dashboard with aggregated information
CREATE VIEW todo_dashboard AS
SELECT 
    t.id,
    t.task_name,
    t.status,
    t.priority,
    t.due_date,
    t.completion_percentage,
    p.title as project_title,
    p.color_theme as project_color,
    rv.version_name as release_version,
    u.username as assigned_username,
    (SELECT COUNT(*) FROM subtasks st WHERE st.todo_id = t.id AND st.is_archived = false) as subtask_count,
    (SELECT COUNT(*) FROM subtasks st WHERE st.todo_id = t.id AND st.status = 'completed') as completed_subtasks,
    (SELECT COUNT(*) FROM comments c WHERE c.todo_id = t.id) as comment_count,
    (SELECT COUNT(*) FROM attachments a WHERE a.todo_id = t.id) as attachment_count
FROM todos t
JOIN projects p ON t.project_id = p.id
LEFT JOIN release_versions rv ON t.release_version_id = rv.id
LEFT JOIN users u ON t.assigned_to = u.id
WHERE t.is_archived = false;

-- View for project statistics
CREATE VIEW project_stats AS
SELECT 
    p.id,
    p.title,
    p.category,
    p.progress_percentage,
    COUNT(t.id) as total_todos,
    COUNT(CASE WHEN t.status = 'completed' THEN 1 END) as completed_todos,
    COUNT(CASE WHEN t.status = 'in_progress' THEN 1 END) as in_progress_todos,
    COUNT(CASE WHEN t.due_date < CURRENT_TIMESTAMP AND t.status != 'completed' THEN 1 END) as overdue_todos,
    AVG(t.actual_time) as avg_task_duration,
    SUM(t.estimated_time) as total_estimated_time,
    SUM(t.actual_time) as total_actual_time
FROM projects p
LEFT JOIN todos t ON p.id = t.project_id AND t.is_archived = false
WHERE p.is_archived = false
GROUP BY p.id, p.title, p.category, p.progress_percentage;

-- =====================================================
-- PERFORMANCE ANALYSIS QUERIES
-- =====================================================

-- Query to find slow queries (run EXPLAIN ANALYZE on these)
/*
-- Most common todo queries:
SELECT * FROM todos WHERE user_id = ? AND status = 'in_progress';
SELECT * FROM todos WHERE project_id = ? ORDER BY priority DESC, due_date ASC;
SELECT * FROM todos WHERE due_date BETWEEN ? AND ? AND user_id = ?;
SELECT * FROM subtasks WHERE todo_id = ? ORDER BY sort_order;

-- Most common project queries:
SELECT * FROM projects WHERE user_id = ? AND is_archived = false;
SELECT * FROM project_stats WHERE id = ?;

-- Most common scheduled task queries:
SELECT * FROM scheduled_tasks WHERE user_id = ? AND scheduled_datetime >= CURRENT_TIMESTAMP;
SELECT * FROM scheduled_tasks WHERE notification_sent = false AND scheduled_datetime <= CURRENT_TIMESTAMP + INTERVAL '15 minutes';
*/

-- =====================================================
-- CLEANUP AND MAINTENANCE
-- =====================================================

-- Function to archive old completed todos
CREATE OR REPLACE FUNCTION archive_old_completed_todos(days_old INTEGER DEFAULT 90)
RETURNS INTEGER AS $$
DECLARE
    affected_rows INTEGER;
BEGIN
    UPDATE todos 
    SET is_archived = true 
    WHERE status = 'completed' 
    AND completed_at < CURRENT_TIMESTAMP - (days_old || ' days')::INTERVAL
    AND is_archived = false;
    
    GET DIAGNOSTICS affected_rows = ROW_COUNT;
    RETURN affected_rows;
END;
$$ LANGUAGE plpgsql;

-- Function to clean up old activity logs
CREATE OR REPLACE FUNCTION cleanup_old_activity_logs(days_old INTEGER DEFAULT 365)
RETURNS INTEGER AS $$
DECLARE
    affected_rows INTEGER;
BEGIN
    DELETE FROM activity_log 
    WHERE created_at < CURRENT_TIMESTAMP - (days_old || ' days')::INTERVAL;
    
    GET DIAGNOSTICS affected_rows = ROW_COUNT;
    RETURN affected_rows;
END;
$$ LANGUAGE plpgsql;

-- =====================================================
-- COMMENTS AND DOCUMENTATION
-- =====================================================

COMMENT ON TABLE users IS 'Application users with authentication and profile information';
COMMENT ON TABLE projects IS 'User projects that contain todos and organize work';
COMMENT ON TABLE release_versions IS 'Version tracking for projects with release management';
COMMENT ON TABLE todos IS 'Main tasks/todos with comprehensive tracking and metadata';
COMMENT ON TABLE subtasks IS 'Sub-items of todos for breaking down complex tasks';
COMMENT ON TABLE notes IS 'Rich text notes attached to todos for additional context';
COMMENT ON TABLE scheduled_tasks IS 'Calendar integration and scheduled reminders for todos';
COMMENT ON TABLE time_entries IS 'Time tracking entries for todos with billing capabilities';
COMMENT ON TABLE comments IS 'Collaboration comments on todos with threading support';
COMMENT ON TABLE attachments IS 'File attachments for todos and projects';
COMMENT ON TABLE activity_log IS 'Audit trail for all user actions and system changes';

-- Performance notes:
-- 1. UUID primary keys provide excellent distribution for sharding
-- 2. GIN indexes on arrays and text fields enable fast search
-- 3. Partial indexes reduce index size for common filtered queries
-- 4. Composite indexes support multi-column WHERE clauses
-- 5. pg_trgm extension enables fuzzy text search on names and content
-- 6. JSONB fields allow flexible storage while maintaining query performance
-- 7. Triggers automatically maintain calculated fields and timestamps
-- 8. Views provide pre-optimized queries for dashboard operations

-- Scalability considerations:
-- 1. Ready for horizontal partitioning by user_id
-- 2. Connection pooling recommended for high concurrency
-- 3. Read replicas can handle reporting and analytics workloads
-- 4. Archive strategy implemented for data lifecycle management
-- 5. Indexes designed to support both OLTP and light analytics workloads