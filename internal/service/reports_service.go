package service

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"quantumtask-auth-api/internal/models"
	"quantumtask-auth-api/internal/repository"
	"quantumtask-auth-api/pkg/redis"

	"github.com/google/uuid"
	"github.com/pkg/errors"
)

type ReportsService interface {
	GetReportsOverview(userID uuid.UUID, filters *models.ReportFilterRequest) (*models.ReportsOverview, error)
	GetTaskReport(userID uuid.UUID, filters *models.ReportFilterRequest) (*models.TaskReportMetrics, error)
	GetProjectReport(userID uuid.UUID, filters *models.ReportFilterRequest) (*models.ProjectReportMetrics, error)
	GetAnalysisReport(userID uuid.UUID, filters *models.ReportFilterRequest) (*models.ReportsOverview, error)
	GetDrillDownReport(userID uuid.UUID, projectID uuid.UUID, filters *models.ReportFilterRequest) (*models.DrillDownReport, error)
	GetComparisonReport(userID uuid.UUID, filters *models.ReportFilterRequest) (*models.ComparisonReport, error)
	ExportReport(userID uuid.UUID, exportReq *models.ExportRequest) ([]byte, string, error)
	ValidateFilters(filters *models.ReportFilterRequest) error
	GetCachedReports(userID uuid.UUID, cacheKey string) (*models.ReportsOverview, error)
	CacheReports(userID uuid.UUID, cacheKey string, data *models.ReportsOverview) error
	InvalidateUserReportsCache(userID uuid.UUID) error
}

type reportsService struct {
	reportsRepo repository.ReportsRepository
	redisClient redis.Client
}

func NewReportsService(
	reportsRepo repository.ReportsRepository,
	redisClient redis.Client,
) ReportsService {
	return &reportsService{
		reportsRepo: reportsRepo,
		redisClient: redisClient,
	}
}

func (s *reportsService) ValidateFilters(filters *models.ReportFilterRequest) error {
	if filters == nil {
		return errors.New("filters cannot be nil")
	}

	// Set defaults
	if filters.Page <= 0 {
		filters.Page = 1
	}
	if filters.Limit <= 0 {
		filters.Limit = 50
	}
	if filters.Limit > 100 {
		filters.Limit = 100
	}

	// Validate date range
	switch filters.DateRange {
	case models.ReportDateRangeCustom:
		if filters.StartDate == nil || filters.EndDate == nil {
			return errors.New("start_date and end_date are required for custom date range")
		}
		if filters.EndDate.Before(*filters.StartDate) {
			return errors.New("end_date must be after start_date")
		}
		// Limit custom range to 1 year
		if filters.EndDate.Sub(*filters.StartDate) > 365*24*time.Hour {
			return errors.New("custom date range cannot exceed 1 year")
		}
	case models.ReportDateRangeLastHour, models.ReportDateRangeDaily, 
		 models.ReportDateRangeWeekly, models.ReportDateRangeMonthly,
		 models.ReportDateRangeLastWeek, models.ReportDateRangeLastMonth,
		 models.ReportDateRangeLastYear:
		// Valid predefined ranges
	default:
		return errors.New("invalid date range")
	}

	return nil
}

func (s *reportsService) generateCacheKey(userID uuid.UUID, filters *models.ReportFilterRequest, reportType string) string {
	key := fmt.Sprintf("reports:%s:%s", userID.String(), reportType)
	
	// Add filter parameters to cache key
	var keyParts []string
	keyParts = append(keyParts, string(filters.DateRange))
	
	if filters.ProjectID != nil {
		keyParts = append(keyParts, "project:"+filters.ProjectID.String())
	}
	if filters.Status != nil {
		keyParts = append(keyParts, "status:"+string(*filters.Status))
	}
	if filters.Category != nil {
		keyParts = append(keyParts, "category:"+string(*filters.Category))
	}
	if filters.Priority != nil {
		keyParts = append(keyParts, "priority:"+string(*filters.Priority))
	}
	if filters.StartDate != nil {
		keyParts = append(keyParts, "start:"+filters.StartDate.Format("2006-01-02"))
	}
	if filters.EndDate != nil {
		keyParts = append(keyParts, "end:"+filters.EndDate.Format("2006-01-02"))
	}
	
	if len(keyParts) > 0 {
		key += ":" + strings.Join(keyParts, ":")
	}
	
	return key
}

func (s *reportsService) GetCachedReports(userID uuid.UUID, cacheKey string) (*models.ReportsOverview, error) {
	if s.redisClient == nil {
		return nil, nil // No caching available
	}

	data, err := s.redisClient.Get(context.Background(), cacheKey)
	if err != nil {
		return nil, nil // Cache miss or error
	}

	var overview models.ReportsOverview
	if err := json.Unmarshal(data, &overview); err != nil {
		return nil, nil // Invalid cached data
	}

	return &overview, nil
}

func (s *reportsService) CacheReports(userID uuid.UUID, cacheKey string, data *models.ReportsOverview) error {
	if s.redisClient == nil {
		return nil // No caching available
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return errors.Wrap(err, "failed to marshal reports data")
	}

	// Cache for 15 minutes
	return s.redisClient.Set(context.Background(), cacheKey, jsonData, 15*time.Minute)
}

func (s *reportsService) InvalidateUserReportsCache(userID uuid.UUID) error {
	if s.redisClient == nil {
		return nil
	}

	pattern := fmt.Sprintf("reports:%s:*", userID.String())
	return s.redisClient.DeletePattern(context.Background(), pattern)
}

func (s *reportsService) GetReportsOverview(userID uuid.UUID, filters *models.ReportFilterRequest) (*models.ReportsOverview, error) {
	if err := s.ValidateFilters(filters); err != nil {
		return nil, errors.Wrap(err, "invalid filters")
	}

	// Try to get from cache first
	cacheKey := s.generateCacheKey(userID, filters, "overview")
	if cached, err := s.GetCachedReports(userID, cacheKey); err == nil && cached != nil {
		return cached, nil
	}

	// Get from database
	overview, err := s.reportsRepo.GetReportsOverview(userID, filters)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get reports overview")
	}

	// Cache the result
	if err := s.CacheReports(userID, cacheKey, overview); err != nil {
		// Log error but don't fail the request
		fmt.Printf("Warning: Failed to cache reports overview: %v\n", err)
	}

	return overview, nil
}

func (s *reportsService) GetTaskReport(userID uuid.UUID, filters *models.ReportFilterRequest) (*models.TaskReportMetrics, error) {
	if err := s.ValidateFilters(filters); err != nil {
		return nil, errors.Wrap(err, "invalid filters")
	}

	return s.reportsRepo.GetTaskMetrics(userID, filters)
}

func (s *reportsService) GetProjectReport(userID uuid.UUID, filters *models.ReportFilterRequest) (*models.ProjectReportMetrics, error) {
	if err := s.ValidateFilters(filters); err != nil {
		return nil, errors.Wrap(err, "invalid filters")
	}

	return s.reportsRepo.GetProjectMetrics(userID, filters)
}

func (s *reportsService) GetAnalysisReport(userID uuid.UUID, filters *models.ReportFilterRequest) (*models.ReportsOverview, error) {
	// Analysis report is the same as overview but with different caching
	if err := s.ValidateFilters(filters); err != nil {
		return nil, errors.Wrap(err, "invalid filters")
	}

	cacheKey := s.generateCacheKey(userID, filters, "analysis")
	if cached, err := s.GetCachedReports(userID, cacheKey); err == nil && cached != nil {
		return cached, nil
	}

	overview, err := s.reportsRepo.GetReportsOverview(userID, filters)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get analysis report")
	}

	if err := s.CacheReports(userID, cacheKey, overview); err != nil {
		fmt.Printf("Warning: Failed to cache analysis report: %v\n", err)
	}

	return overview, nil
}

func (s *reportsService) GetDrillDownReport(userID uuid.UUID, projectID uuid.UUID, filters *models.ReportFilterRequest) (*models.DrillDownReport, error) {
	if err := s.ValidateFilters(filters); err != nil {
		return nil, errors.Wrap(err, "invalid filters")
	}

	return s.reportsRepo.GetDrillDownReport(userID, projectID, filters)
}

func (s *reportsService) GetComparisonReport(userID uuid.UUID, filters *models.ReportFilterRequest) (*models.ComparisonReport, error) {
	if err := s.ValidateFilters(filters); err != nil {
		return nil, errors.Wrap(err, "invalid filters")
	}

	return s.reportsRepo.GetComparisonReport(userID, filters)
}

func (s *reportsService) ExportReport(userID uuid.UUID, exportReq *models.ExportRequest) ([]byte, string, error) {
	if exportReq == nil {
		return nil, "", errors.New("export request cannot be nil")
	}

	if exportReq.Filters == nil {
		exportReq.Filters = &models.ReportFilterRequest{
			DateRange: models.ReportDateRangeMonthly,
			Page:      1,
			Limit:     100,
		}
	}

	if err := s.ValidateFilters(exportReq.Filters); err != nil {
		return nil, "", errors.Wrap(err, "invalid export filters")
	}

	var data interface{}
	var err error

	// Get data based on report type
	switch exportReq.ReportType {
	case "task":
		data, err = s.GetTaskReport(userID, exportReq.Filters)
	case "project":
		data, err = s.GetProjectReport(userID, exportReq.Filters)
	case "analysis":
		data, err = s.GetAnalysisReport(userID, exportReq.Filters)
	default:
		return nil, "", errors.New("invalid report type")
	}

	if err != nil {
		return nil, "", errors.Wrap(err, "failed to get report data")
	}

	// Export data based on format
	switch exportReq.Format {
	case "csv":
		return s.exportToCSV(data, exportReq.ReportType)
	case "pdf":
		return s.exportToPDF(data, exportReq.ReportType)
	case "excel":
		return s.exportToExcel(data, exportReq.ReportType)
	default:
		return nil, "", errors.New("unsupported export format")
	}
}

func (s *reportsService) exportToCSV(data interface{}, reportType string) ([]byte, string, error) {
	var records [][]string
	filename := fmt.Sprintf("%s_report_%s.csv", reportType, time.Now().Format("2006-01-02"))

	switch reportType {
	case "task":
		taskData, ok := data.(*models.TaskReportMetrics)
		if !ok {
			return nil, "", errors.New("invalid task data for CSV export")
		}
		
		records = append(records, []string{"Metric", "Value"})
		records = append(records, []string{"Total Tasks", strconv.Itoa(taskData.TotalTasks)})
		records = append(records, []string{"Completed Tasks", strconv.Itoa(taskData.CompletedTasks)})
		records = append(records, []string{"In Progress Tasks", strconv.Itoa(taskData.InProgressTasks)})
		records = append(records, []string{"Overdue Tasks", strconv.Itoa(taskData.OverdueTasks)})
		records = append(records, []string{"Completion Rate (%)", fmt.Sprintf("%.2f", taskData.CompletionRate)})
		records = append(records, []string{"Average Completion Time (hours)", fmt.Sprintf("%.2f", taskData.AverageCompletionTime)})
		records = append(records, []string{"Tasks Per Day", fmt.Sprintf("%.2f", taskData.TasksPerDay)})
		records = append(records, []string{"Hours Per Day", fmt.Sprintf("%.2f", taskData.HoursPerDay)})
		records = append(records, []string{"Minutes Per Task", fmt.Sprintf("%.2f", taskData.MinutesPerTask)})
		records = append(records, []string{"Current Streak", strconv.Itoa(taskData.CurrentStreak)})
		records = append(records, []string{"Longest Streak", strconv.Itoa(taskData.LongestStreak)})

	case "project":
		projectData, ok := data.(*models.ProjectReportMetrics)
		if !ok {
			return nil, "", errors.New("invalid project data for CSV export")
		}
		
		records = append(records, []string{"Metric", "Value"})
		records = append(records, []string{"Total Projects", strconv.Itoa(projectData.TotalProjects)})
		records = append(records, []string{"Active Projects", strconv.Itoa(projectData.ActiveProjects)})
		records = append(records, []string{"Completed Projects", strconv.Itoa(projectData.CompletedProjects)})
		records = append(records, []string{"Archived Projects", strconv.Itoa(projectData.ArchivedProjects)})
		records = append(records, []string{"Average Progress (%)", fmt.Sprintf("%.2f", projectData.AverageProgress)})
		records = append(records, []string{"Projects with Overdue Tasks", strconv.Itoa(projectData.ProjectsWithOverdue)})

	case "analysis":
		analysisData, ok := data.(*models.ReportsOverview)
		if !ok {
			return nil, "", errors.New("invalid analysis data for CSV export")
		}
		
		// Export summary metrics
		records = append(records, []string{"Analysis Report"})
		records = append(records, []string{""})
		records = append(records, []string{"Task Metrics"})
		records = append(records, []string{"Total Tasks", strconv.Itoa(analysisData.TaskMetrics.TotalTasks)})
		records = append(records, []string{"Completed Tasks", strconv.Itoa(analysisData.TaskMetrics.CompletedTasks)})
		records = append(records, []string{"Completion Rate (%)", fmt.Sprintf("%.2f", analysisData.TaskMetrics.CompletionRate)})
		records = append(records, []string{""})
		records = append(records, []string{"Project Metrics"})
		records = append(records, []string{"Total Projects", strconv.Itoa(analysisData.ProjectMetrics.TotalProjects)})
		records = append(records, []string{"Active Projects", strconv.Itoa(analysisData.ProjectMetrics.ActiveProjects)})
		records = append(records, []string{"Average Progress (%)", fmt.Sprintf("%.2f", analysisData.ProjectMetrics.AverageProgress)})
		
		// Add time distribution
		records = append(records, []string{""})
		records = append(records, []string{"Time Distribution by Category"})
		records = append(records, []string{"Category", "Hours", "Percentage", "Tasks"})
		for _, dist := range analysisData.TimeDistribution {
			records = append(records, []string{
				dist.Category,
				fmt.Sprintf("%.2f", dist.TotalHours),
				fmt.Sprintf("%.2f", dist.Percentage),
				strconv.Itoa(dist.TaskCount),
			})
		}
	}

	// Convert records to CSV bytes
	var csvBuffer strings.Builder
	writer := csv.NewWriter(&csvBuffer)
	
	for _, record := range records {
		if err := writer.Write(record); err != nil {
			return nil, "", errors.Wrap(err, "failed to write CSV record")
		}
	}
	
	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, "", errors.Wrap(err, "failed to flush CSV writer")
	}

	return []byte(csvBuffer.String()), filename, nil
}

func (s *reportsService) exportToPDF(data interface{}, reportType string) ([]byte, string, error) {
	// For now, return a simple text-based PDF representation
	// In a real implementation, you would use a PDF library like gofpdf
	filename := fmt.Sprintf("%s_report_%s.pdf", reportType, time.Now().Format("2006-01-02"))
	
	pdfContent := fmt.Sprintf("PDF Report - %s\nGenerated on: %s\n\nNote: PDF generation not implemented yet. Use CSV export instead.", 
		reportType, time.Now().Format("2006-01-02 15:04:05"))
	
	return []byte(pdfContent), filename, nil
}

func (s *reportsService) exportToExcel(data interface{}, reportType string) ([]byte, string, error) {
	// For now, return a simple Excel representation
	// In a real implementation, you would use a library like excelize
	filename := fmt.Sprintf("%s_report_%s.xlsx", reportType, time.Now().Format("2006-01-02"))
	
	excelContent := fmt.Sprintf("Excel Report - %s\nGenerated on: %s\n\nNote: Excel generation not implemented yet. Use CSV export instead.", 
		reportType, time.Now().Format("2006-01-02 15:04:05"))
	
	return []byte(excelContent), filename, nil
}
