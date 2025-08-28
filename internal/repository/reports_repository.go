package repository

import (
	"fmt"
	"strings"
	"time"

	"quantumtask-auth-api/internal/database"
	"quantumtask-auth-api/internal/models"

	"github.com/google/uuid"
	"github.com/pkg/errors"
)

type ReportsRepository interface {
	GetReportsOverview(userID uuid.UUID, filters *models.ReportFilterRequest) (*models.ReportsOverview, error)
	GetTaskMetrics(userID uuid.UUID, filters *models.ReportFilterRequest) (*models.TaskReportMetrics, error)
	GetProjectMetrics(userID uuid.UUID, filters *models.ReportFilterRequest) (*models.ProjectReportMetrics, error)
	GetTimeDistribution(userID uuid.UUID, filters *models.ReportFilterRequest) ([]*models.TimeDistribution, error)
	GetDailyProductivity(userID uuid.UUID, filters *models.ReportFilterRequest) ([]*models.DailyProductivity, error)
	GetHourlyProductivity(userID uuid.UUID, filters *models.ReportFilterRequest) ([]*models.HourlyProductivity, error)
	GetTasksByStatus(userID uuid.UUID, filters *models.ReportFilterRequest) ([]*models.TasksByStatus, error)
	GetTasksByPriority(userID uuid.UUID, filters *models.ReportFilterRequest) ([]*models.TasksByPriority, error)
	GetProductivityInsights(userID uuid.UUID, filters *models.ReportFilterRequest) (*models.ProductivityInsights, error)
	GetDrillDownReport(userID uuid.UUID, projectID uuid.UUID, filters *models.ReportFilterRequest) (*models.DrillDownReport, error)
	GetComparisonReport(userID uuid.UUID, filters *models.ReportFilterRequest) (*models.ComparisonReport, error)
	GetRecentTasks(userID uuid.UUID, limit int, filters *models.ReportFilterRequest) ([]*models.ReportTask, error)
	GetUpcomingDeadlines(userID uuid.UUID, limit int) ([]*models.Todo, error)
}

type reportsRepository struct {
	db *database.DB
}

func NewReportsRepository(db *database.DB) ReportsRepository {
	return &reportsRepository{db: db}
}

// Helper function to convert MySQL-style ? placeholders to PostgreSQL $n placeholders
func (r *reportsRepository) convertPlaceholders(query string) string {
	result := strings.Builder{}
	placeholderCount := 0
	
	for _, char := range query {
		if char == '?' {
			placeholderCount++
			result.WriteString(fmt.Sprintf("$%d", placeholderCount))
		} else {
			result.WriteRune(char)
		}
	}
	
	return result.String()
}

// Helper function to build complete WHERE clause with proper PostgreSQL placeholders
func (r *reportsRepository) buildWhereClause(userID uuid.UUID, filters *models.ReportFilterRequest) (string, []interface{}, error) {
	dateCondition, dateArgs, err := r.buildDateRangeCondition(filters)
	if err != nil {
		return "", nil, err
	}

	additionalCondition, additionalArgs := r.buildAdditionalFilters(filters)
	
	var whereClause strings.Builder
	whereClause.WriteString("user_id = $1")
	args := []interface{}{userID}
	argCount := 1
	
	if dateCondition != "" {
		whereClause.WriteString(" AND ")
		// Convert ? placeholders to numbered placeholders
		convertedDateCondition := dateCondition
		for i := 0; i < len(dateArgs); i++ {
			argCount++
			convertedDateCondition = strings.Replace(convertedDateCondition, "?", fmt.Sprintf("$%d", argCount), 1)
		}
		whereClause.WriteString(convertedDateCondition)
		args = append(args, dateArgs...)
	}
	
	if additionalCondition != "" {
		whereClause.WriteString(" AND ")
		// Convert ? placeholders to numbered placeholders
		convertedAdditionalCondition := additionalCondition
		for i := 0; i < len(additionalArgs); i++ {
			argCount++
			convertedAdditionalCondition = strings.Replace(convertedAdditionalCondition, "?", fmt.Sprintf("$%d", argCount), 1)
		}
		whereClause.WriteString(convertedAdditionalCondition)
		args = append(args, additionalArgs...)
	}
	
	return whereClause.String(), args, nil
}

// Helper function to build date range conditions
func (r *reportsRepository) buildDateRangeCondition(filters *models.ReportFilterRequest) (string, []interface{}, error) {
	var condition string
	var args []interface{}
	
	now := time.Now()
	var startDate, endDate time.Time

	switch filters.DateRange {
	case models.ReportDateRangeLastHour:
		startDate = now.Add(-time.Hour)
		endDate = now
	case models.ReportDateRangeDaily:
		startDate = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		endDate = startDate.Add(24 * time.Hour)
	case models.ReportDateRangeWeekly:
		weekday := now.Weekday()
		if weekday == 0 { weekday = 7 } // Make Sunday = 7 instead of 0
		startDate = now.AddDate(0, 0, -int(weekday-1)).Truncate(24 * time.Hour)
		endDate = startDate.Add(7 * 24 * time.Hour)
	case models.ReportDateRangeMonthly:
		startDate = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		endDate = startDate.AddDate(0, 1, 0)
	case models.ReportDateRangeLastWeek:
		weekday := now.Weekday()
		if weekday == 0 { weekday = 7 }
		endDate = now.AddDate(0, 0, -int(weekday-1)).Truncate(24 * time.Hour)
		startDate = endDate.Add(-7 * 24 * time.Hour)
	case models.ReportDateRangeLastMonth:
		endDate = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		startDate = endDate.AddDate(0, -1, 0)
	case models.ReportDateRangeLastYear:
		endDate = time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location())
		startDate = endDate.AddDate(-1, 0, 0)
	case models.ReportDateRangeCustom:
		if filters.StartDate == nil || filters.EndDate == nil {
			return "", nil, errors.New("start_date and end_date are required for custom range")
		}
		startDate = *filters.StartDate
		endDate = *filters.EndDate
	default:
		return "", nil, errors.New("invalid date range")
	}

	condition = "created_at >= ? AND created_at < ?"
	args = append(args, startDate, endDate)
	
	return condition, args, nil
}

// Helper function to build additional filters
func (r *reportsRepository) buildAdditionalFilters(filters *models.ReportFilterRequest) (string, []interface{}) {
	var conditions []string
	var args []interface{}

	if filters.ProjectID != nil {
		conditions = append(conditions, "project_id = ?")
		args = append(args, *filters.ProjectID)
	}

	if filters.Status != nil {
		conditions = append(conditions, "status = ?")
		args = append(args, *filters.Status)
	}

	if filters.Priority != nil {
		conditions = append(conditions, "priority = ?")
		args = append(args, *filters.Priority)
	}

	return strings.Join(conditions, " AND "), args
}

func (r *reportsRepository) GetTaskMetrics(userID uuid.UUID, filters *models.ReportFilterRequest) (*models.TaskReportMetrics, error) {
	whereClause, args, err := r.buildWhereClause(userID, filters)
	if err != nil {
		return nil, err
	}

	query := fmt.Sprintf(`
		WITH task_stats AS (
			SELECT 
				COUNT(*) as total_tasks,
				COUNT(CASE WHEN status = 'completed' OR status = 'done' THEN 1 END) as completed_tasks,
				COUNT(CASE WHEN status = 'in_progress' THEN 1 END) as in_progress_tasks,
				COUNT(CASE WHEN due_date < NOW() AND status NOT IN ('completed', 'done', 'cancelled') THEN 1 END) as overdue_tasks,
				AVG(CASE WHEN completed_at IS NOT NULL AND created_at IS NOT NULL 
					THEN EXTRACT(EPOCH FROM (completed_at - created_at))/3600.0 END) as avg_completion_time,
				AVG(actual_time) as avg_actual_time
			FROM todos 
			WHERE %s
		),
		daily_stats AS (
			SELECT 
				DATE(created_at) as task_date,
				COUNT(*) as daily_tasks,
				SUM(actual_time) as daily_minutes
			FROM todos 
			WHERE %s AND status IN ('completed', 'done')
			GROUP BY DATE(created_at)
		),
		streak_data AS (
			SELECT 
				task_date,
				daily_tasks,
				ROW_NUMBER() OVER (ORDER BY task_date) - 
				ROW_NUMBER() OVER (PARTITION BY CASE WHEN daily_tasks > 0 THEN 1 ELSE 0 END ORDER BY task_date) as streak_group
			FROM daily_stats
		),
		streaks AS (
			SELECT 
				COUNT(*) as streak_length,
				MAX(task_date) as streak_end
			FROM streak_data
			WHERE daily_tasks > 0
			GROUP BY streak_group
		)
		SELECT 
			ts.total_tasks,
			ts.completed_tasks,
			ts.in_progress_tasks,
			ts.overdue_tasks,
			COALESCE(CASE WHEN ts.total_tasks > 0 THEN ts.completed_tasks::float / ts.total_tasks * 100 ELSE 0 END, 0) as completion_rate,
			COALESCE(ts.avg_completion_time, 0) as average_completion_time,
			COALESCE((SELECT AVG(daily_tasks) FROM daily_stats), 0) as tasks_per_day,
			COALESCE((SELECT AVG(daily_minutes/60.0) FROM daily_stats), 0) as hours_per_day,
			COALESCE(ts.avg_actual_time, 0) as minutes_per_task,
			COALESCE((SELECT streak_length FROM streaks WHERE streak_end = (SELECT MAX(task_date) FROM daily_stats) ORDER BY streak_length DESC LIMIT 1), 0) as current_streak,
			COALESCE((SELECT MAX(streak_length) FROM streaks), 0) as longest_streak
		FROM task_stats ts
	`, whereClause, whereClause)

	var metrics models.TaskReportMetrics
	err = r.db.Get(&metrics, query, args...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get task metrics")
	}

	return &metrics, nil
}

func (r *reportsRepository) GetProjectMetrics(userID uuid.UUID, filters *models.ReportFilterRequest) (*models.ProjectReportMetrics, error) {
	dateCondition, dateArgs, err := r.buildDateRangeCondition(filters)
	if err != nil {
		return nil, err
	}

	var whereClause strings.Builder
	whereClause.WriteString("user_id = $1")
	args := []interface{}{userID}
	argCount := 1
	
	if dateCondition != "" {
		whereClause.WriteString(" AND ")
		// Convert ? placeholders to numbered placeholders
		convertedDateCondition := dateCondition
		for i := 0; i < len(dateArgs); i++ {
			argCount++
			convertedDateCondition = strings.Replace(convertedDateCondition, "?", fmt.Sprintf("$%d", argCount), 1)
		}
		whereClause.WriteString(convertedDateCondition)
		args = append(args, dateArgs...)
	}

	if filters.Category != nil {
		argCount++
		whereClause.WriteString(fmt.Sprintf(" AND category = $%d", argCount))
		args = append(args, *filters.Category)
	}

	query := fmt.Sprintf(`
		WITH project_stats AS (
			SELECT 
				p.id,
				p.is_archived,
				p.progress_percentage,
				CASE WHEN EXISTS(
					SELECT 1 FROM todos t 
					WHERE t.project_id = p.id 
					AND t.due_date < NOW() 
					AND t.status NOT IN ('completed', 'done', 'cancelled')
				) THEN 1 ELSE 0 END as has_overdue
			FROM projects p
			WHERE %s
		)
		SELECT 
			COUNT(*) as total_projects,
			COUNT(CASE WHEN is_archived = false THEN 1 END) as active_projects,
			COUNT(CASE WHEN progress_percentage >= 100 THEN 1 END) as completed_projects,
			COUNT(CASE WHEN is_archived = true THEN 1 END) as archived_projects,
			COALESCE(AVG(progress_percentage), 0) as average_progress,
			COUNT(CASE WHEN has_overdue = 1 THEN 1 END) as projects_with_overdue
		FROM project_stats
	`, whereClause.String())

	var metrics models.ProjectReportMetrics
	err = r.db.Get(&metrics, query, args...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get project metrics")
	}

	return &metrics, nil
}

// Temporary simplified implementations for testing
func (r *reportsRepository) GetTimeDistribution(userID uuid.UUID, filters *models.ReportFilterRequest) ([]*models.TimeDistribution, error) {
	return []*models.TimeDistribution{}, nil
}

func (r *reportsRepository) GetDailyProductivity(userID uuid.UUID, filters *models.ReportFilterRequest) ([]*models.DailyProductivity, error) {
	return []*models.DailyProductivity{}, nil
}

func (r *reportsRepository) GetHourlyProductivity(userID uuid.UUID, filters *models.ReportFilterRequest) ([]*models.HourlyProductivity, error) {
	return []*models.HourlyProductivity{}, nil
}

func (r *reportsRepository) GetTasksByStatus(userID uuid.UUID, filters *models.ReportFilterRequest) ([]*models.TasksByStatus, error) {
	return []*models.TasksByStatus{}, nil
}

func (r *reportsRepository) GetTasksByPriority(userID uuid.UUID, filters *models.ReportFilterRequest) ([]*models.TasksByPriority, error) {
	return []*models.TasksByPriority{}, nil
}

func (r *reportsRepository) GetProductivityInsights(userID uuid.UUID, filters *models.ReportFilterRequest) (*models.ProductivityInsights, error) {
	return &models.ProductivityInsights{}, nil
}

func (r *reportsRepository) GetRecentTasks(userID uuid.UUID, limit int, filters *models.ReportFilterRequest) ([]*models.ReportTask, error) {
	return []*models.ReportTask{}, nil
}

func (r *reportsRepository) GetUpcomingDeadlines(userID uuid.UUID, limit int) ([]*models.Todo, error) {
	return []*models.Todo{}, nil
}

func (r *reportsRepository) GetReportsOverview(userID uuid.UUID, filters *models.ReportFilterRequest) (*models.ReportsOverview, error) {
	// Get all report components
	taskMetrics, err := r.GetTaskMetrics(userID, filters)
	if err != nil {
		return nil, err
	}

	projectMetrics, err := r.GetProjectMetrics(userID, filters)
	if err != nil {
		return nil, err
	}

	timeDistribution, err := r.GetTimeDistribution(userID, filters)
	if err != nil {
		return nil, err
	}

	dailyProductivity, err := r.GetDailyProductivity(userID, filters)
	if err != nil {
		return nil, err
	}

	hourlyProductivity, err := r.GetHourlyProductivity(userID, filters)
	if err != nil {
		return nil, err
	}

	tasksByStatus, err := r.GetTasksByStatus(userID, filters)
	if err != nil {
		return nil, err
	}

	tasksByPriority, err := r.GetTasksByPriority(userID, filters)
	if err != nil {
		return nil, err
	}

	productivityInsights, err := r.GetProductivityInsights(userID, filters)
	if err != nil {
		return nil, err
	}

	recentTasks, err := r.GetRecentTasks(userID, 10, filters)
	if err != nil {
		return nil, err
	}

	upcomingDeadlines, err := r.GetUpcomingDeadlines(userID, 10)
	if err != nil {
		return nil, err
	}

	return &models.ReportsOverview{
		TaskMetrics:         taskMetrics,
		ProjectMetrics:      projectMetrics,
		TimeDistribution:    timeDistribution,
		DailyProductivity:   dailyProductivity,
		HourlyProductivity:  hourlyProductivity,
		TasksByStatus:       tasksByStatus,
		TasksByPriority:     tasksByPriority,
		ProductivityInsights: productivityInsights,
		RecentTasks:         recentTasks,
		UpcomingDeadlines:   upcomingDeadlines,
	}, nil
}

func (r *reportsRepository) GetDrillDownReport(userID uuid.UUID, projectID uuid.UUID, filters *models.ReportFilterRequest) (*models.DrillDownReport, error) {
	return &models.DrillDownReport{}, nil
}

func (r *reportsRepository) GetComparisonReport(userID uuid.UUID, filters *models.ReportFilterRequest) (*models.ComparisonReport, error) {
	return &models.ComparisonReport{}, nil
}
