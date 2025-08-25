package models

import (
	"time"

	"github.com/google/uuid"
)

// ImportResult represents the result of a project import operation
type ImportResult struct {
	Success    bool             `json:"success"`
	Message    string           `json:"message"`
	Statistics ImportStatistics `json:"statistics,omitempty"`
	Errors     []string         `json:"errors,omitempty"`
	Warnings   []string         `json:"warnings,omitempty"`
}

// ImportStatistics contains statistics about the import operation
type ImportStatistics struct {
	TotalRecords      int            `json:"total_records"`
	SuccessfulImports int            `json:"successful_imports"`
	FailedImports     int            `json:"failed_imports"`
	SkippedRecords    int            `json:"skipped_records"`
	DuplicatesFound   int            `json:"duplicates_found"`
	CategoryBreakdown map[string]int `json:"category_breakdown,omitempty"`
}

// CSVTaskRecord represents a task record in CSV format for export/import
type CSVTaskRecord struct {
	TaskName             string `csv:"Task Name" json:"task_name"`
	TaskDescription      string `csv:"Description" json:"task_description,omitempty"`
	Status               string `csv:"Status" json:"status"`
	Priority             string `csv:"Priority" json:"priority"`
	DueDate              string `csv:"Due Date" json:"due_date,omitempty"`
	StartDate            string `csv:"Start Date" json:"start_date,omitempty"`
	CompletedAt          string `csv:"Completed At" json:"completed_at,omitempty"`
	Tags                 string `csv:"Tags" json:"tags,omitempty"`
	EstimatedTime        string `csv:"Estimated Time" json:"estimated_time,omitempty"`
	ActualTime           string `csv:"Actual Time" json:"actual_time,omitempty"`
	AssignedTo           string `csv:"Assigned To" json:"assigned_to,omitempty"`
	Location             string `csv:"Location" json:"location,omitempty"`
	Context              string `csv:"Context" json:"context,omitempty"`
	DifficultyRating     string `csv:"Difficulty Rating" json:"difficulty_rating,omitempty"`
	EnergyLevelRequired  string `csv:"Energy Level Required" json:"energy_level_required,omitempty"`
	CompletionPercentage string `csv:"Completion Percentage" json:"completion_percentage,omitempty"`
	IsRecurring          string `csv:"Is Recurring" json:"is_recurring,omitempty"`
	IsPinned             string `csv:"Is Pinned" json:"is_pinned,omitempty"`
	CreatedAt            string `csv:"Created At" json:"created_at"`
	UpdatedAt            string `csv:"Updated At" json:"updated_at"`
}

// ToTodo converts a CSVTaskRecord to a Todo model for database insertion
func (c *CSVTaskRecord) ToTodo(projectID, userID uuid.UUID) (*Todo, error) {
	todo := &Todo{
		ID:              uuid.New(),
		UserID:          userID,
		ProjectID:       projectID,
		TaskName:        c.TaskName,
		TaskDescription: &c.TaskDescription,
		Status:          TaskStatus(c.Status),
		Priority:        PriorityLevel(c.Priority),
		Location:        &c.Location,
		Context:         &c.Context,
		IsRecurring:     c.IsRecurring == "true",
		IsPinned:        c.IsPinned == "true",
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	// Parse optional fields
	if c.DueDate != "" {
		if dueDate, err := time.Parse("2006-01-02 15:04:05", c.DueDate); err == nil {
			todo.DueDate = &dueDate
		}
	}

	if c.StartDate != "" {
		if startDate, err := time.Parse("2006-01-02 15:04:05", c.StartDate); err == nil {
			todo.StartDate = &startDate
		}
	}

	if c.CompletedAt != "" {
		if completedAt, err := time.Parse("2006-01-02 15:04:05", c.CompletedAt); err == nil {
			todo.CompletedAt = &completedAt
		}
	}

	return todo, nil
}

// FromTodo creates a CSVTaskRecord from a Todo model for export
func CSVTaskRecordFromTodo(todo *Todo) *CSVTaskRecord {
	record := &CSVTaskRecord{
		TaskName:    todo.TaskName,
		Status:      string(todo.Status),
		Priority:    string(todo.Priority),
		IsRecurring: "false",
		IsPinned:    "false",
		CreatedAt:   todo.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:   todo.UpdatedAt.Format("2006-01-02 15:04:05"),
	}

	// Handle optional string fields
	if todo.TaskDescription != nil {
		record.TaskDescription = *todo.TaskDescription
	}
	if todo.Location != nil {
		record.Location = *todo.Location
	}
	if todo.Context != nil {
		record.Context = *todo.Context
	}

	// Handle boolean fields
	if todo.IsRecurring {
		record.IsRecurring = "true"
	}
	if todo.IsPinned {
		record.IsPinned = "true"
	}

	// Handle optional time fields
	if todo.DueDate != nil {
		record.DueDate = todo.DueDate.Format("2006-01-02 15:04:05")
	}
	if todo.StartDate != nil {
		record.StartDate = todo.StartDate.Format("2006-01-02 15:04:05")
	}
	if todo.CompletedAt != nil {
		record.CompletedAt = todo.CompletedAt.Format("2006-01-02 15:04:05")
	}

	return record
}

// ExportMetadata contains metadata about an export operation
type ExportMetadata struct {
	ProjectID     uuid.UUID `json:"project_id"`
	ProjectTitle  string    `json:"project_title"`
	ExportedAt    time.Time `json:"exported_at"`
	ExportedBy    uuid.UUID `json:"exported_by"`
	TotalTasks    int       `json:"total_tasks"`
	ExportVersion string    `json:"export_version"`
}
