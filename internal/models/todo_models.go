package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// Priority levels for tasks
type PriorityLevel string

const (
	PriorityLow    PriorityLevel = "low"
	PriorityMedium PriorityLevel = "medium"
	PriorityHigh   PriorityLevel = "high"
	PriorityUrgent PriorityLevel = "urgent"
)

// Task status
type TaskStatus string

const (
	TaskStatusNotStarted TaskStatus = "not_started"
	TaskStatusInProgress TaskStatus = "in_progress"
	TaskStatusPending    TaskStatus = "pending"
	TaskStatusCompleted  TaskStatus = "completed"
	TaskStatusCancelled  TaskStatus = "cancelled"
	TaskStatusOnHold     TaskStatus = "on_hold"
	TaskStatusBacklog    TaskStatus = "backlog"
	TaskStatusDone       TaskStatus = "done"
	TaskStatusTodo       TaskStatus = "todo"
	TaskStatusBlocked    TaskStatus = "blocked"
	TaskStatusReview     TaskStatus = "review"
	TaskStatusTesting    TaskStatus = "testing"
)

// Project categories
type ProjectCategory string

const (
	ProjectCategoryPersonal  ProjectCategory = "personal"
	ProjectCategoryWork      ProjectCategory = "work"
	ProjectCategoryEducation ProjectCategory = "education"
	ProjectCategoryHealth    ProjectCategory = "health"
	ProjectCategoryFinance   ProjectCategory = "finance"
	ProjectCategoryHobby     ProjectCategory = "hobby"
	ProjectCategoryOther     ProjectCategory = "other"
)

// Time unit for estimates
type TimeUnit string

const (
	TimeUnitMinutes TimeUnit = "minutes"
	TimeUnitHours   TimeUnit = "hours"
	TimeUnitDays    TimeUnit = "days"
)

// Notification frequency
type NotificationFrequency string

const (
	NotificationFrequencyNone    NotificationFrequency = "none"
	NotificationFrequencyOnce    NotificationFrequency = "once"
	NotificationFrequencyDaily   NotificationFrequency = "daily"
	NotificationFrequencyWeekly  NotificationFrequency = "weekly"
	NotificationFrequencyMonthly NotificationFrequency = "monthly"
)

// Recurrence pattern for scheduled tasks
type RecurrencePattern string

const (
	RecurrencePatternNone     RecurrencePattern = "none"
	RecurrencePatternDaily    RecurrencePattern = "daily"
	RecurrencePatternWeekdays RecurrencePattern = "weekdays" // Monday to Friday
	RecurrencePatternWeekly   RecurrencePattern = "weekly"   // Same day every week
	RecurrencePatternMonthly  RecurrencePattern = "monthly"  // Same date every month
	RecurrencePatternCustom   RecurrencePattern = "custom"   // Custom pattern
)

// Project represents the projects table
type Project struct {
	ID                 uuid.UUID       `json:"id" db:"id"`
	UserID             uuid.UUID       `json:"user_id" db:"user_id"`
	Title              string          `json:"title" db:"title" validate:"required,min=1,max=255"`
	Description        *string         `json:"description,omitempty" db:"description"`
	ImageURL           *string         `json:"image_url,omitempty" db:"image_url"`
	Category           ProjectCategory `json:"category" db:"category"`
	ColorTheme         string          `json:"color_theme" db:"color_theme"`
	IsArchived         bool            `json:"is_archived" db:"is_archived"`
	IsFavorite         bool            `json:"is_favorite" db:"is_favorite"`
	ProgressPercentage float64         `json:"progress_percentage" db:"progress_percentage"`
	CreatedAt          time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt          time.Time       `json:"updated_at" db:"updated_at"`

	// Computed fields (not stored in DB)
	TodoCount      *int `json:"todo_count,omitempty" db:"-"`
	CompletedCount *int `json:"completed_count,omitempty" db:"-"`
}

// ReleaseVersion represents the release_versions table
type ReleaseVersion struct {
	ID            uuid.UUID  `json:"id" db:"id"`
	ProjectID     uuid.UUID  `json:"project_id" db:"project_id"`
	VersionName   string     `json:"version_name" db:"version_name" validate:"required,min=1,max=100"`
	VersionNumber string     `json:"version_number" db:"version_number" validate:"required,min=1,max=20"`
	Description   *string    `json:"description,omitempty" db:"description"`
	TargetDate    *time.Time `json:"target_date,omitempty" db:"target_date"`
	ReleaseDate   *time.Time `json:"release_date,omitempty" db:"release_date"`
	IsReleased    bool       `json:"is_released" db:"is_released"`
	ReleaseNotes  *string    `json:"release_notes,omitempty" db:"release_notes"`
	CreatedAt     time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at" db:"updated_at"`

	// Computed fields
	TodoCount      *int `json:"todo_count,omitempty" db:"-"`
	CompletedCount *int `json:"completed_count,omitempty" db:"-"`
}

// Todo represents the todos table
type Todo struct {
	ID                   uuid.UUID      `json:"id" db:"id"`
	UserID               uuid.UUID      `json:"user_id" db:"user_id"`
	ProjectID            uuid.UUID      `json:"project_id" db:"project_id"`
	ReleaseVersionID     *uuid.UUID     `json:"release_version_id,omitempty" db:"release_version_id"`
	ParentTodoID         *uuid.UUID     `json:"parent_todo_id,omitempty" db:"parent_todo_id"`
	TaskName             string         `json:"task_name" db:"task_name" validate:"required,min=1,max=255"`
	TaskDescription      *string        `json:"task_description,omitempty" db:"task_description"`
	TaskImageOrEmoji     *string        `json:"task_image_or_emoji,omitempty" db:"task_image_or_emoji"`
	StatusIcon           *string        `json:"status_icon,omitempty" db:"status_icon"`
	EstimatedTime        *int           `json:"estimated_time,omitempty" db:"estimated_time"`
	ActualTime           int            `json:"actual_time" db:"actual_time"`
	TimeUnit             TimeUnit       `json:"time_unit" db:"time_unit"`
	Priority             PriorityLevel  `json:"priority" db:"priority"`
	Status               TaskStatus     `json:"status" db:"status"`
	CompletionPercentage float64        `json:"completion_percentage" db:"completion_percentage"`
	DueDate              *time.Time     `json:"due_date,omitempty" db:"due_date"`
	StartDate            *time.Time     `json:"start_date,omitempty" db:"start_date"`
	CompletedAt          *time.Time     `json:"completed_at,omitempty" db:"completed_at"`
	Tags                 pq.StringArray `json:"tags" db:"tags"`
	DifficultyRating     *int           `json:"difficulty_rating,omitempty" db:"difficulty_rating" validate:"omitempty,min=1,max=10"`
	EnergyLevelRequired  *int           `json:"energy_level_required,omitempty" db:"energy_level_required" validate:"omitempty,min=1,max=5"`
	Location             *string        `json:"location,omitempty" db:"location"`
	Context              *string        `json:"context,omitempty" db:"context"`
	AssignedTo           *uuid.UUID     `json:"assigned_to,omitempty" db:"assigned_to"`
	IsRecurring          bool           `json:"is_recurring" db:"is_recurring"`
	RecurrencePattern    *JSONB         `json:"recurrence_pattern,omitempty" db:"recurrence_pattern"`
	IsArchived           bool           `json:"is_archived" db:"is_archived"`
	IsPinned             bool           `json:"is_pinned" db:"is_pinned"`

	// AI Enhancement fields
	AIEnhanced             bool    `json:"ai_enhanced" db:"ai_enhanced"`
	AIGeneratedDescription bool    `json:"ai_generated_description" db:"ai_generated_description"`
	TaskEmoji              *string `json:"task_emoji,omitempty" db:"task_emoji"`
	AICategory             *string `json:"ai_category,omitempty" db:"ai_category"`
	AIPriority             *string `json:"ai_priority,omitempty" db:"ai_priority"`
	AIEstimatedDuration    *int    `json:"ai_estimated_duration,omitempty" db:"ai_estimated_duration"`
	AIEnhancementVersion   int     `json:"ai_enhancement_version" db:"ai_enhancement_version"`
	AIMetadata             *JSONB  `json:"ai_metadata,omitempty" db:"ai_metadata"`

	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`

	// Computed/Related fields
	ProjectTitle        *string `json:"project_title,omitempty" db:"-"`
	AssignedToName      *string `json:"assigned_to_name,omitempty" db:"-"`
	SubtaskCount        *int    `json:"subtask_count,omitempty" db:"-"`
	CompletedSubs       *int    `json:"completed_subtasks,omitempty" db:"-"`
	CommentCount        *int    `json:"comment_count,omitempty" db:"-"`
	AttachmentCount     *int    `json:"attachment_count,omitempty" db:"-"`
	AISubtaskCount      *int    `json:"ai_subtask_count,omitempty" db:"-"`
	AcceptedSuggestions *int    `json:"accepted_suggestions,omitempty" db:"-"`
}

// Subtask represents the subtasks table
type Subtask struct {
	ID            uuid.UUID     `json:"id" db:"id"`
	TodoID        uuid.UUID     `json:"todo_id" db:"todo_id"`
	UserID        uuid.UUID     `json:"user_id" db:"user_id"`
	Title         string        `json:"title" db:"title" validate:"required,min=1,max=255"`
	Name          string        `json:"name" db:"name" validate:"required,min=1,max=255"`
	Description   *string       `json:"description,omitempty" db:"description"`
	Status        TaskStatus    `json:"status" db:"status"`
	Priority      PriorityLevel `json:"priority" db:"priority"`
	EstimatedTime *int          `json:"estimated_time,omitempty" db:"estimated_time"`
	ActualTime    int           `json:"actual_time" db:"actual_time"`
	DueDate       *time.Time    `json:"due_date,omitempty" db:"due_date"`
	CompletedAt   *time.Time    `json:"completed_at,omitempty" db:"completed_at"`
	Order         int           `json:"order" db:"sort_order"`
	SortOrder     int           `json:"sort_order" db:"sort_order"`
	IsArchived    bool          `json:"is_archived" db:"is_archived"`

	// AI Enhancement fields
	AIGenerated         bool   `json:"ai_generated" db:"ai_generated"`
	AIEstimatedDuration *int   `json:"ai_estimated_duration,omitempty" db:"ai_estimated_duration"`
	AIMetadata          *JSONB `json:"ai_metadata,omitempty" db:"ai_metadata"`

	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// Note represents the notes table
type Note struct {
	ID        uuid.UUID      `json:"id" db:"id"`
	TodoID    uuid.UUID      `json:"todo_id" db:"todo_id"`
	Title     *string        `json:"title,omitempty" db:"title"`
	Content   string         `json:"content" db:"content" validate:"required"`
	NoteType  string         `json:"note_type" db:"note_type"`
	IsPinned  bool           `json:"is_pinned" db:"is_pinned"`
	Tags      pq.StringArray `json:"tags" db:"tags"`
	CreatedAt time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt time.Time      `json:"updated_at" db:"updated_at"`
}

// ScheduledTask represents the scheduled_tasks table
type ScheduledTask struct {
	ID                    uuid.UUID             `json:"id" db:"id"`
	TodoID                uuid.UUID             `json:"todo_id" db:"todo_id"`
	UserID                uuid.UUID             `json:"user_id" db:"user_id"`
	ScheduledDatetime     time.Time             `json:"scheduled_datetime" db:"scheduled_datetime" validate:"required"`
	DurationMinutes       *int                  `json:"duration_minutes,omitempty" db:"duration_minutes"`
	NotificationSent      bool                  `json:"notification_sent" db:"notification_sent"`
	NotificationFrequency NotificationFrequency `json:"notification_frequency" db:"notification_frequency"`
	ReminderMinutesBefore int                   `json:"reminder_minutes_before" db:"reminder_minutes_before"`
	Location              *string               `json:"location,omitempty" db:"location"`
	MeetingURL            *string               `json:"meeting_url,omitempty" db:"meeting_url"`
	Attendees             pq.StringArray        `json:"attendees" db:"attendees"`
	IsAllDay              bool                  `json:"is_all_day" db:"is_all_day"`
	IsCancelled           bool                  `json:"is_cancelled" db:"is_cancelled"`

	// Recurring task fields
	IsRecurring        bool              `json:"is_recurring" db:"is_recurring"`
	RecurrencePattern  RecurrencePattern `json:"recurrence_pattern" db:"recurrence_pattern"`
	RecurrenceInterval int               `json:"recurrence_interval" db:"recurrence_interval"` // Every N days/weeks/months
	RecurrenceEndDate  *time.Time        `json:"recurrence_end_date,omitempty" db:"recurrence_end_date"`
	ParentScheduleID   *uuid.UUID        `json:"parent_schedule_id,omitempty" db:"parent_schedule_id"`
	IsParentSchedule   bool              `json:"is_parent_schedule" db:"is_parent_schedule"`
	RecurrenceCount    *int              `json:"recurrence_count,omitempty" db:"recurrence_count"` // Max occurrences
	WeekdayMask        *int              `json:"weekday_mask,omitempty" db:"weekday_mask"`         // Bitmask for weekdays (1=Sunday, 2=Monday, etc.)

	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`

	// Related fields
	TodoTitle *string `json:"todo_title,omitempty" db:"-"`
}

// TimeEntry represents the time_entries table
type TimeEntry struct {
	ID              uuid.UUID  `json:"id" db:"id"`
	TodoID          uuid.UUID  `json:"todo_id" db:"todo_id"`
	UserID          uuid.UUID  `json:"user_id" db:"user_id"`
	StartTime       time.Time  `json:"start_time" db:"start_time" validate:"required"`
	EndTime         *time.Time `json:"end_time,omitempty" db:"end_time"`
	DurationMinutes *int       `json:"duration_minutes,omitempty" db:"duration_minutes"`
	Description     *string    `json:"description,omitempty" db:"description"`
	IsBillable      bool       `json:"is_billable" db:"is_billable"`
	HourlyRate      *float64   `json:"hourly_rate,omitempty" db:"hourly_rate"`
	CreatedAt       time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at" db:"updated_at"`

	// Related fields
	TodoTitle *string `json:"todo_title,omitempty" db:"-"`
}

// Comment represents the comments table
type Comment struct {
	ID              uuid.UUID  `json:"id" db:"id"`
	TodoID          uuid.UUID  `json:"todo_id" db:"todo_id"`
	UserID          uuid.UUID  `json:"user_id" db:"user_id"`
	ParentCommentID *uuid.UUID `json:"parent_comment_id,omitempty" db:"parent_comment_id"`
	Content         string     `json:"content" db:"content" validate:"required"`
	IsEdited        bool       `json:"is_edited" db:"is_edited"`
	EditedAt        *time.Time `json:"edited_at,omitempty" db:"edited_at"`
	CreatedAt       time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at" db:"updated_at"`

	// Related fields
	AuthorName   *string    `json:"author_name,omitempty" db:"-"`
	AuthorAvatar *string    `json:"author_avatar,omitempty" db:"-"`
	ReplyCount   *int       `json:"reply_count,omitempty" db:"-"`
	Replies      []*Comment `json:"replies,omitempty" db:"-"`
}

// Attachment represents the attachments table
type Attachment struct {
	ID               uuid.UUID  `json:"id" db:"id"`
	TodoID           *uuid.UUID `json:"todo_id,omitempty" db:"todo_id"`
	ProjectID        *uuid.UUID `json:"project_id,omitempty" db:"project_id"`
	UserID           uuid.UUID  `json:"user_id" db:"user_id"`
	Filename         string     `json:"filename" db:"filename"`
	OriginalFilename string     `json:"original_filename" db:"original_filename"`
	FilePath         string     `json:"file_path" db:"file_path"`
	FileSize         int64      `json:"file_size" db:"file_size"`
	MimeType         string     `json:"mime_type" db:"mime_type"`
	Checksum         *string    `json:"checksum,omitempty" db:"checksum"`
	IsImage          bool       `json:"is_image" db:"is_image"`
	ThumbnailPath    *string    `json:"thumbnail_path,omitempty" db:"thumbnail_path"`
	CreatedAt        time.Time  `json:"created_at" db:"created_at"`
}

// ActivityLog represents the activity_log table
type ActivityLog struct {
	ID         uuid.UUID `json:"id" db:"id"`
	UserID     uuid.UUID `json:"user_id" db:"user_id"`
	EntityType string    `json:"entity_type" db:"entity_type"`
	EntityID   uuid.UUID `json:"entity_id" db:"entity_id"`
	Action     string    `json:"action" db:"action"`
	OldValues  *JSONB    `json:"old_values,omitempty" db:"old_values"`
	NewValues  *JSONB    `json:"new_values,omitempty" db:"new_values"`
	IPAddress  *string   `json:"ip_address,omitempty" db:"ip_address"`
	UserAgent  *string   `json:"user_agent,omitempty" db:"user_agent"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
}

// Dashboard views and aggregated data structures

// TodoDashboard represents a comprehensive todo view with related information
type TodoDashboard struct {
	ID                   uuid.UUID     `json:"id" db:"id"`
	TaskName             string        `json:"task_name" db:"task_name"`
	Status               TaskStatus    `json:"status" db:"status"`
	Priority             PriorityLevel `json:"priority" db:"priority"`
	DueDate              *time.Time    `json:"due_date" db:"due_date"`
	CompletionPercentage float64       `json:"completion_percentage" db:"completion_percentage"`
	ProjectTitle         string        `json:"project_title" db:"project_title"`
	ProjectColor         string        `json:"project_color" db:"project_color"`
	ReleaseVersion       *string       `json:"release_version" db:"release_version"`
	AssignedUsername     *string       `json:"assigned_username" db:"assigned_username"`
	SubtaskCount         int           `json:"subtask_count" db:"subtask_count"`
	CompletedSubtasks    int           `json:"completed_subtasks" db:"completed_subtasks"`
	CommentCount         int           `json:"comment_count" db:"comment_count"`
	AttachmentCount      int           `json:"attachment_count" db:"attachment_count"`
}

// ProjectStats represents project statistics
type ProjectStats struct {
	ID                 uuid.UUID       `json:"id" db:"id"`
	Title              string          `json:"title" db:"title"`
	Category           ProjectCategory `json:"category" db:"category"`
	ProgressPercentage float64         `json:"progress_percentage" db:"progress_percentage"`
	TotalTodos         int             `json:"total_todos" db:"total_todos"`
	CompletedTodos     int             `json:"completed_todos" db:"completed_todos"`
	InProgressTodos    int             `json:"in_progress_todos" db:"in_progress_todos"`
	OverdueTodos       int             `json:"overdue_todos" db:"overdue_todos"`
	AvgTaskDuration    *float64        `json:"avg_task_duration" db:"avg_task_duration"`
	TotalEstimatedTime *int            `json:"total_estimated_time" db:"total_estimated_time"`
	TotalActualTime    *int            `json:"total_actual_time" db:"total_actual_time"`
}

// Request/Response DTOs

// CreateProjectRequest represents the request body for creating a project
type CreateProjectRequest struct {
	Title       string          `json:"title" validate:"required,min=1,max=255"`
	Description *string         `json:"description,omitempty"`
	ImageURL    *string         `json:"image_url,omitempty"`
	Category    ProjectCategory `json:"category" validate:"required"`
	ColorTheme  *string         `json:"color_theme,omitempty"`
}

// UpdateProjectRequest represents the request body for updating a project
type UpdateProjectRequest struct {
	Title       *string          `json:"title,omitempty" validate:"omitempty,min=1,max=255"`
	Description *string          `json:"description,omitempty"`
	ImageURL    *string          `json:"image_url,omitempty"`
	Category    *ProjectCategory `json:"category,omitempty"`
	ColorTheme  *string          `json:"color_theme,omitempty"`
	IsFavorite  *bool            `json:"is_favorite,omitempty"`
}

// CreateTodoRequest represents the request body for creating a todo
type CreateTodoRequest struct {
	ProjectID           uuid.UUID      `json:"project_id" validate:"required"`
	ReleaseVersionID    *uuid.UUID     `json:"release_version_id,omitempty"`
	ParentTodoID        *uuid.UUID     `json:"parent_todo_id,omitempty"`
	TaskName            string         `json:"task_name" validate:"required,min=1,max=255"`
	TaskDescription     *string        `json:"task_description,omitempty"`
	TaskImageOrEmoji    *string        `json:"task_image_or_emoji,omitempty"`
	Status              *TaskStatus    `json:"status,omitempty"`
	StatusIcon          *string        `json:"status_icon,omitempty"`
	EstimatedTime       *int           `json:"estimated_time,omitempty"`
	TimeUnit            *TimeUnit      `json:"time_unit,omitempty"`
	Priority            *PriorityLevel `json:"priority,omitempty"`
	DueDate             *time.Time     `json:"due_date,omitempty"`
	StartDate           *time.Time     `json:"start_date,omitempty"`
	Tags                []string       `json:"tags,omitempty"`
	DifficultyRating    *int           `json:"difficulty_rating,omitempty" validate:"omitempty,min=1,max=10"`
	EnergyLevelRequired *int           `json:"energy_level_required,omitempty" validate:"omitempty,min=1,max=5"`
	Location            *string        `json:"location,omitempty"`
	Context             *string        `json:"context,omitempty"`
	AssignedTo          *uuid.UUID     `json:"assigned_to,omitempty"`
	IsRecurring         *bool          `json:"is_recurring,omitempty"`
	RecurrencePattern   *JSONB         `json:"recurrence_pattern,omitempty"`
}

// UpdateTodoRequest represents the request body for updating a todo
type UpdateTodoRequest struct {
	TaskName            *string        `json:"task_name,omitempty" validate:"omitempty,min=1,max=255"`
	TaskDescription     *string        `json:"task_description,omitempty"`
	TaskImageOrEmoji    *string        `json:"task_image_or_emoji,omitempty"`
	StatusIcon          *string        `json:"status_icon,omitempty"`
	EstimatedTime       *int           `json:"estimated_time,omitempty"`
	ActualTime          *int           `json:"actual_time,omitempty"`
	TimeUnit            *TimeUnit      `json:"time_unit,omitempty"`
	Priority            *PriorityLevel `json:"priority,omitempty"`
	Status              *TaskStatus    `json:"status,omitempty"`
	DueDate             *time.Time     `json:"due_date,omitempty"`
	StartDate           *time.Time     `json:"start_date,omitempty"`
	Tags                []string       `json:"tags,omitempty"`
	DifficultyRating    *int           `json:"difficulty_rating,omitempty" validate:"omitempty,min=1,max=10"`
	EnergyLevelRequired *int           `json:"energy_level_required,omitempty" validate:"omitempty,min=1,max=5"`
	Location            *string        `json:"location,omitempty"`
	Context             *string        `json:"context,omitempty"`
	AssignedTo          *uuid.UUID     `json:"assigned_to,omitempty"`
	IsRecurring         *bool          `json:"is_recurring,omitempty"`
	RecurrencePattern   *JSONB         `json:"recurrence_pattern,omitempty"`
	IsPinned            *bool          `json:"is_pinned,omitempty"`
}

// CreateSubtaskRequest represents the request body for creating a subtask
type CreateSubtaskRequest struct {
	Name          string         `json:"name" validate:"required,min=1,max=255"`
	Description   *string        `json:"description,omitempty"`
	Priority      *PriorityLevel `json:"priority,omitempty"`
	EstimatedTime *int           `json:"estimated_time,omitempty"`
	DueDate       *time.Time     `json:"due_date,omitempty"`
	SortOrder     *int           `json:"sort_order,omitempty"`
}

// UpdateSubtaskRequest represents the request body for updating a subtask
type UpdateSubtaskRequest struct {
	Name          *string        `json:"name,omitempty" validate:"omitempty,min=1,max=255"`
	Description   *string        `json:"description,omitempty"`
	Status        *TaskStatus    `json:"status,omitempty"`
	Priority      *PriorityLevel `json:"priority,omitempty"`
	EstimatedTime *int           `json:"estimated_time,omitempty"`
	DueDate       *time.Time     `json:"due_date,omitempty"`
	SortOrder     *int           `json:"sort_order,omitempty"`
}

// Helper methods for enum validation

func (p PriorityLevel) IsValid() bool {
	return p == PriorityLow || p == PriorityMedium || p == PriorityHigh || p == PriorityUrgent
}

func (s TaskStatus) IsValid() bool {
	return s == TaskStatusNotStarted || s == TaskStatusInProgress || s == TaskStatusPending ||
		s == TaskStatusCompleted || s == TaskStatusCancelled || s == TaskStatusOnHold || s == TaskStatusBacklog || s == TaskStatusDone || s == TaskStatusTodo
}

func (c ProjectCategory) IsValid() bool {
	return c == ProjectCategoryPersonal || c == ProjectCategoryWork || c == ProjectCategoryEducation ||
		c == ProjectCategoryHealth || c == ProjectCategoryFinance || c == ProjectCategoryHobby || c == ProjectCategoryOther
}

func (t TimeUnit) IsValid() bool {
	return t == TimeUnitMinutes || t == TimeUnitHours || t == TimeUnitDays
}

// Scan implements the sql.Scanner interface for enum types
func (p *PriorityLevel) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	switch v := value.(type) {
	case string:
		*p = PriorityLevel(v)
		return nil
	case []byte:
		*p = PriorityLevel(string(v))
		return nil
	default:
		return fmt.Errorf("cannot scan %T into PriorityLevel", value)
	}
}

func (p PriorityLevel) Value() (driver.Value, error) {
	return string(p), nil
}

func (s *TaskStatus) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	switch v := value.(type) {
	case string:
		*s = TaskStatus(v)
		return nil
	case []byte:
		*s = TaskStatus(string(v))
		return nil
	default:
		return fmt.Errorf("cannot scan %T into TaskStatus", value)
	}
}

func (s TaskStatus) Value() (driver.Value, error) {
	return string(s), nil
}

func (c *ProjectCategory) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	switch v := value.(type) {
	case string:
		*c = ProjectCategory(v)
		return nil
	case []byte:
		*c = ProjectCategory(string(v))
		return nil
	default:
		return fmt.Errorf("cannot scan %T into ProjectCategory", value)
	}
}

func (c ProjectCategory) Value() (driver.Value, error) {
	return string(c), nil
}

func (t *TimeUnit) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	switch v := value.(type) {
	case string:
		*t = TimeUnit(v)
		return nil
	case []byte:
		*t = TimeUnit(string(v))
		return nil
	default:
		return fmt.Errorf("cannot scan %T into TimeUnit", value)
	}
}

func (t TimeUnit) Value() (driver.Value, error) {
	return string(t), nil
}

// JSONB represents a JSON field for database storage
type JSONB map[string]interface{}

// Scan implements the sql.Scanner interface for JSONB
func (j *JSONB) Scan(value interface{}) error {
	if value == nil {
		return nil
	}

	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, j)
	case string:
		return json.Unmarshal([]byte(v), j)
	default:
		return fmt.Errorf("cannot scan %T into JSONB", value)
	}
}

// Value implements the driver.Valuer interface for JSONB
func (j JSONB) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

// IsValidTaskStatus validates if a status is valid
func IsValidTaskStatus(status string) bool {
	switch TaskStatus(status) {
	case TaskStatusNotStarted, TaskStatusInProgress, TaskStatusPending,
		TaskStatusCompleted, TaskStatusCancelled, TaskStatusOnHold,
		TaskStatusBacklog, TaskStatusDone, TaskStatusTodo,
		TaskStatusBlocked, TaskStatusReview, TaskStatusTesting:
		return true
	default:
		return false
	}
}

// GetAllTaskStatuses returns all valid task statuses
func GetAllTaskStatuses() []TaskStatus {
	return []TaskStatus{
		TaskStatusTodo,
		TaskStatusNotStarted,
		TaskStatusInProgress,
		TaskStatusTesting,
		TaskStatusReview,
		TaskStatusPending,
		TaskStatusCompleted,
		TaskStatusDone,
		TaskStatusOnHold,
		TaskStatusBacklog,
		TaskStatusBlocked,
		TaskStatusCancelled,
	}
}

// GetTaskStatusDisplay returns user-friendly display text for status
func GetTaskStatusDisplay(status TaskStatus) string {
	switch status {
	case TaskStatusNotStarted:
		return "Not Started"
	case TaskStatusInProgress:
		return "In Progress"
	case TaskStatusPending:
		return "Pending"
	case TaskStatusCompleted:
		return "Completed"
	case TaskStatusCancelled:
		return "Cancelled"
	case TaskStatusOnHold:
		return "On Hold"
	case TaskStatusBacklog:
		return "Backlog"
	case TaskStatusDone:
		return "Done"
	case TaskStatusTodo:
		return "Todo"
	case TaskStatusBlocked:
		return "Blocked"
	case TaskStatusReview:
		return "Under Review"
	case TaskStatusTesting:
		return "Testing"
	default:
		return string(status)
	}
}

// IsCompletedStatus checks if the status represents a completed state
func IsCompletedStatus(status TaskStatus) bool {
	return status == TaskStatusCompleted || status == TaskStatusDone
}

// CanTransitionTo checks if status transition is valid
func (t TaskStatus) CanTransitionTo(newStatus TaskStatus) bool {
	// Define valid status transitions
	validTransitions := map[TaskStatus][]TaskStatus{
		TaskStatusTodo:       {TaskStatusNotStarted, TaskStatusInProgress, TaskStatusBacklog, TaskStatusCancelled},
		TaskStatusNotStarted: {TaskStatusInProgress, TaskStatusOnHold, TaskStatusCancelled, TaskStatusBacklog},
		TaskStatusInProgress: {TaskStatusTesting, TaskStatusReview, TaskStatusCompleted, TaskStatusDone, TaskStatusOnHold, TaskStatusBlocked, TaskStatusCancelled},
		TaskStatusTesting:    {TaskStatusInProgress, TaskStatusReview, TaskStatusCompleted, TaskStatusDone, TaskStatusBlocked},
		TaskStatusReview:     {TaskStatusInProgress, TaskStatusTesting, TaskStatusCompleted, TaskStatusDone, TaskStatusBlocked},
		TaskStatusPending:    {TaskStatusInProgress, TaskStatusOnHold, TaskStatusCancelled},
		TaskStatusBlocked:    {TaskStatusInProgress, TaskStatusOnHold, TaskStatusCancelled},
		TaskStatusOnHold:     {TaskStatusInProgress, TaskStatusCancelled, TaskStatusBacklog},
		TaskStatusBacklog:    {TaskStatusTodo, TaskStatusNotStarted, TaskStatusInProgress, TaskStatusCancelled},
		TaskStatusCompleted:  {TaskStatusInProgress, TaskStatusReview},                  // Allow reopening
		TaskStatusDone:       {TaskStatusInProgress, TaskStatusReview},                  // Allow reopening
		TaskStatusCancelled:  {TaskStatusTodo, TaskStatusNotStarted, TaskStatusBacklog}, // Allow reactivation
	}

	allowedStatuses, exists := validTransitions[t]
	if !exists {
		return true // If no restrictions defined, allow transition
	}

	for _, allowed := range allowedStatuses {
		if allowed == newStatus {
			return true
		}
	}
	return false
}

func (n *NotificationFrequency) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	switch v := value.(type) {
	case string:
		*n = NotificationFrequency(v)
		return nil
	case []byte:
		*n = NotificationFrequency(string(v))
		return nil
	default:
		return fmt.Errorf("cannot scan %T into NotificationFrequency", value)
	}
}

func (n NotificationFrequency) Value() (driver.Value, error) {
	return string(n), nil
}

// Scan implements the sql.Scanner interface for RecurrencePattern
func (r *RecurrencePattern) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	switch v := value.(type) {
	case string:
		*r = RecurrencePattern(v)
		return nil
	case []byte:
		*r = RecurrencePattern(string(v))
		return nil
	default:
		return fmt.Errorf("cannot scan %T into RecurrencePattern", value)
	}
}

// Value implements the driver.Valuer interface for RecurrencePattern
func (r RecurrencePattern) Value() (driver.Value, error) {
	return string(r), nil
}
