package models

import (
	"time"

	"github.com/google/uuid"
)

// ReportDateRange represents the date range options for reports
type ReportDateRange string

const (
	ReportDateRangeLastHour   ReportDateRange = "last_hour"
	ReportDateRangeDaily      ReportDateRange = "daily"
	ReportDateRangeWeekly     ReportDateRange = "weekly"
	ReportDateRangeMonthly    ReportDateRange = "monthly"
	ReportDateRangeCustom     ReportDateRange = "custom"
	ReportDateRangeLastWeek   ReportDateRange = "last_week"
	ReportDateRangeLastMonth  ReportDateRange = "last_month"
	ReportDateRangeLastYear   ReportDateRange = "last_year"
)

// ReportFilterRequest represents the filters for generating reports
type ReportFilterRequest struct {
	DateRange   ReportDateRange `json:"date_range" validate:"required"`
	StartDate   *time.Time      `json:"start_date,omitempty"`
	EndDate     *time.Time      `json:"end_date,omitempty"`
	ProjectID   *uuid.UUID      `json:"project_id,omitempty"`
	UserID      *uuid.UUID      `json:"user_id,omitempty"`
	Status      *TaskStatus     `json:"status,omitempty"`
	Category    *ProjectCategory `json:"category,omitempty"`
	Priority    *PriorityLevel  `json:"priority,omitempty"`
	Page        int             `json:"page" validate:"min=1"`
	Limit       int             `json:"limit" validate:"min=1,max=100"`
}

// TaskReportMetrics represents key task metrics
type TaskReportMetrics struct {
	TotalTasks            int     `json:"total_tasks" db:"total_tasks"`
	CompletedTasks        int     `json:"completed_tasks" db:"completed_tasks"`
	InProgressTasks       int     `json:"in_progress_tasks" db:"in_progress_tasks"`
	OverdueTasks          int     `json:"overdue_tasks" db:"overdue_tasks"`
	CompletionRate        float64 `json:"completion_rate" db:"completion_rate"`
	AverageCompletionTime float64 `json:"average_completion_time" db:"average_completion_time"` // in hours
	TasksPerDay           float64 `json:"tasks_per_day" db:"tasks_per_day"`
	HoursPerDay           float64 `json:"hours_per_day" db:"hours_per_day"`
	MinutesPerTask        float64 `json:"minutes_per_task" db:"minutes_per_task"`
	CurrentStreak         int     `json:"current_streak" db:"current_streak"`
	LongestStreak         int     `json:"longest_streak" db:"longest_streak"`
}

// ProjectReportMetrics represents key project metrics
type ProjectReportMetrics struct {
	TotalProjects       int      `json:"total_projects" db:"total_projects"`
	ActiveProjects      int      `json:"active_projects" db:"active_projects"`
	CompletedProjects   int      `json:"completed_projects" db:"completed_projects"`
	ArchivedProjects    int      `json:"archived_projects" db:"archived_projects"`
	AverageProgress     *float64 `json:"average_progress" db:"average_progress"`
	ProjectsWithOverdue int      `json:"projects_with_overdue" db:"projects_with_overdue"`
}

// TimeDistribution represents time spent by category
type TimeDistribution struct {
	Category      string  `json:"category" db:"category"`
	TotalMinutes  int     `json:"total_minutes" db:"total_minutes"`
	TotalHours    float64 `json:"total_hours" db:"total_hours"`
	Percentage    float64 `json:"percentage" db:"percentage"`
	TaskCount     int     `json:"task_count" db:"task_count"`
	Color         string  `json:"color" db:"color"`
}

// DailyProductivity represents productivity data for a specific day
type DailyProductivity struct {
	Date              time.Time `json:"date" db:"date"`
	TasksCompleted    int       `json:"tasks_completed" db:"tasks_completed"`
	HoursWorked       float64   `json:"hours_worked" db:"hours_worked"`
	BreaksTaken       int       `json:"breaks_taken" db:"breaks_taken"`
	BreakMinutes      int       `json:"break_minutes" db:"break_minutes"`
	FocusTime         int       `json:"focus_time" db:"focus_time"` // in minutes
	ProductivityScore float64   `json:"productivity_score" db:"productivity_score"` // 0-100
}

// HourlyProductivity represents productivity data by hour of day
type HourlyProductivity struct {
	Hour              int     `json:"hour" db:"hour"`
	TasksCompleted    int     `json:"tasks_completed" db:"tasks_completed"`
	MinutesWorked     int     `json:"minutes_worked" db:"minutes_worked"`
	ProductivityScore float64 `json:"productivity_score" db:"productivity_score"`
}

// TasksByStatus represents task distribution by status
type TasksByStatus struct {
	Status     TaskStatus `json:"status" db:"status"`
	Count      int        `json:"count" db:"count"`
	Percentage float64    `json:"percentage" db:"percentage"`
}

// TasksByPriority represents task distribution by priority
type TasksByPriority struct {
	Priority   PriorityLevel `json:"priority" db:"priority"`
	Count      int           `json:"count" db:"count"`
	Percentage float64       `json:"percentage" db:"percentage"`
}

// ProductivityInsights represents key productivity insights
type ProductivityInsights struct {
	MostProductiveHour     int     `json:"most_productive_hour" db:"most_productive_hour"`
	MostProductiveDay      string  `json:"most_productive_day" db:"most_productive_day"`
	MostProductiveMonth    string  `json:"most_productive_month" db:"most_productive_month"`
	BestPerformingCategory string  `json:"best_performing_category" db:"best_performing_category"`
	AverageSessionLength   float64 `json:"average_session_length" db:"average_session_length"` // in minutes
	OptimalBreakFrequency  int     `json:"optimal_break_frequency" db:"optimal_break_frequency"` // minutes between breaks
}

// ComparisonReport represents comparison data between two time periods
type ComparisonReport struct {
	CurrentPeriod  *TaskReportMetrics `json:"current_period"`
	PreviousPeriod *TaskReportMetrics `json:"previous_period"`
	PercentageChange map[string]float64 `json:"percentage_change"`
}

// DrillDownReport represents detailed breakdown for a specific project
type DrillDownReport struct {
	Project           *Project               `json:"project"`
	TaskBreakdown     []*TasksByStatus       `json:"task_breakdown"`
	PriorityBreakdown []*TasksByPriority     `json:"priority_breakdown"`
	TimeDistribution  []*TimeDistribution    `json:"time_distribution"`
	RecentTasks       []*ReportTask          `json:"recent_tasks"`
	Milestones        []*ReleaseVersion      `json:"milestones"`
}

// ReportsOverview represents the complete reports dashboard
type ReportsOverview struct {
	TaskMetrics         *TaskReportMetrics      `json:"task_metrics"`
	ProjectMetrics      *ProjectReportMetrics   `json:"project_metrics"`
	TimeDistribution    []*TimeDistribution     `json:"time_distribution"`
	DailyProductivity   []*DailyProductivity    `json:"daily_productivity"`
	HourlyProductivity  []*HourlyProductivity   `json:"hourly_productivity"`
	TasksByStatus       []*TasksByStatus        `json:"tasks_by_status"`
	TasksByPriority     []*TasksByPriority      `json:"tasks_by_priority"`
	ProductivityInsights *ProductivityInsights  `json:"productivity_insights"`
	RecentTasks         []*ReportTask           `json:"recent_tasks"`
	UpcomingDeadlines   []*Todo                 `json:"upcoming_deadlines"`
}

// ExportRequest represents request for exporting reports
type ExportRequest struct {
	Format    string              `json:"format" validate:"required,oneof=pdf csv excel"`
	ReportType string             `json:"report_type" validate:"required,oneof=task project analysis"`
	Filters   *ReportFilterRequest `json:"filters"`
}

// ReportTask represents a simplified task for reports
type ReportTask struct {
	ID               uuid.UUID     `json:"id" db:"id"`
	TaskName         string        `json:"task_name" db:"task_name"`
	Status           TaskStatus    `json:"status" db:"status"`
	Priority         PriorityLevel `json:"priority" db:"priority"`
	ProjectTitle     string        `json:"project_title" db:"project_title"`
	CompletedAt      *time.Time    `json:"completed_at" db:"completed_at"`
	EstimatedTime    *int          `json:"estimated_time" db:"estimated_time"`
	ActualTime       int           `json:"actual_time" db:"actual_time"`
	DifficultyRating *int          `json:"difficulty_rating" db:"difficulty_rating"`
	CreatedAt        time.Time     `json:"created_at" db:"created_at"`
	DueDate          *time.Time    `json:"due_date" db:"due_date"`
	IsOverdue        bool          `json:"is_overdue" db:"is_overdue"`
}
