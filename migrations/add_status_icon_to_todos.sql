-- Migration: Add status_icon column to todos table
-- Description: Adds a visual status indicator field for todos (can store emoji, icon class, or image URL)

-- Add status_icon column to todos table
ALTER TABLE todos 
ADD COLUMN status_icon TEXT;

-- Add comment for documentation
COMMENT ON COLUMN todos.status_icon IS 'Visual status indicator - can store emoji, CSS icon class, or image URL';

-- Index for filtering by status icon (optional, for performance if needed)
-- CREATE INDEX idx_todos_status_icon ON todos(status_icon) WHERE status_icon IS NOT NULL;
