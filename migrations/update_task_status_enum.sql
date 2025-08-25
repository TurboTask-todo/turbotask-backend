-- =====================================================
-- UPDATE TASK STATUS ENUM WITH MISSING VALUES
-- =====================================================

-- Add missing enum values to task_status
-- Note: PostgreSQL requires adding enum values one at a time

-- Add 'done' status (commonly used in many todo apps)
ALTER TYPE task_status ADD VALUE IF NOT EXISTS 'done';

-- Add 'todo' status (initial status for new tasks)
ALTER TYPE task_status ADD VALUE IF NOT EXISTS 'todo';

-- Add 'backlog' status (for tasks in backlog)
ALTER TYPE task_status ADD VALUE IF NOT EXISTS 'backlog';

-- Add 'blocked' status (for tasks that are blocked)
ALTER TYPE task_status ADD VALUE IF NOT EXISTS 'blocked';

-- Add 'review' status (for tasks under review)
ALTER TYPE task_status ADD VALUE IF NOT EXISTS 'review';

-- Add 'testing' status (for tasks being tested)
ALTER TYPE task_status ADD VALUE IF NOT EXISTS 'testing';

-- Verify the updated enum values
-- SELECT enum_range(NULL::task_status);

-- Update any existing todos that might have invalid status values
-- Set default status for any NULL values
UPDATE todos SET status = 'todo' WHERE status IS NULL;

-- Create index on status for better query performance
CREATE INDEX IF NOT EXISTS idx_todos_status ON todos(status);
CREATE INDEX IF NOT EXISTS idx_todos_status_user ON todos(user_id, status);
