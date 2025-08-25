-- Migration: Add recurring task support to scheduled_tasks table
-- Date: 2024-01-15

-- Create enum type for recurrence pattern
CREATE TYPE recurrence_pattern AS ENUM (
    'none',
    'daily',
    'weekdays',
    'weekly',
    'monthly',
    'custom'
);

-- Add new columns for recurring task support
ALTER TABLE scheduled_tasks 
ADD COLUMN is_recurring BOOLEAN DEFAULT false,
ADD COLUMN recurrence_pattern recurrence_pattern DEFAULT 'none',
ADD COLUMN recurrence_interval INTEGER DEFAULT 1,
ADD COLUMN recurrence_end_date TIMESTAMP WITH TIME ZONE,
ADD COLUMN parent_schedule_id UUID REFERENCES scheduled_tasks(id) ON DELETE CASCADE,
ADD COLUMN is_parent_schedule BOOLEAN DEFAULT false,
ADD COLUMN recurrence_count INTEGER,
ADD COLUMN weekday_mask INTEGER;

-- Add indexes for better performance
CREATE INDEX idx_scheduled_tasks_recurring ON scheduled_tasks(is_recurring);
CREATE INDEX idx_scheduled_tasks_parent ON scheduled_tasks(parent_schedule_id);
CREATE INDEX idx_scheduled_tasks_pattern ON scheduled_tasks(recurrence_pattern);
CREATE INDEX idx_scheduled_tasks_next_occurrence ON scheduled_tasks(scheduled_datetime, is_recurring, is_cancelled);

-- Add constraint to ensure parent schedule logic
ALTER TABLE scheduled_tasks 
ADD CONSTRAINT chk_parent_schedule_logic 
CHECK (
    (is_parent_schedule = true AND parent_schedule_id IS NULL) OR
    (is_parent_schedule = false)
);

-- Add constraint for recurrence interval
ALTER TABLE scheduled_tasks 
ADD CONSTRAINT chk_recurrence_interval_positive 
CHECK (recurrence_interval > 0);

-- Add constraint for weekday mask (1-127 for 7 days)
ALTER TABLE scheduled_tasks 
ADD CONSTRAINT chk_weekday_mask_valid 
CHECK (weekday_mask IS NULL OR (weekday_mask >= 1 AND weekday_mask <= 127));
