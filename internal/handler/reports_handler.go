package handler

import (
	"net/http"
	"strconv"
	"time"

	"macwrite-auth-api/internal/models"
	"macwrite-auth-api/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pkg/errors"
)

type ReportsHandler struct {
	reportsService service.ReportsService
}

func NewReportsHandler(reportsService service.ReportsService) *ReportsHandler {
	return &ReportsHandler{
		reportsService: reportsService,
	}
}

// GetReportsOverview returns the complete reports dashboard
// @Summary Get reports overview
// @Description Get comprehensive reports overview with task and project metrics
// @Tags reports
// @Accept json
// @Produce json
// @Param date_range query string false "Date range filter" Enums(last_hour, daily, weekly, monthly, last_week, last_month, last_year, custom)
// @Param start_date query string false "Start date for custom range (YYYY-MM-DD)"
// @Param end_date query string false "End date for custom range (YYYY-MM-DD)"
// @Param project_id query string false "Filter by project ID"
// @Param status query string false "Filter by task status"
// @Param category query string false "Filter by project category"
// @Param priority query string false "Filter by task priority"
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(50)
// @Success 200 {object} models.ReportsOverview
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/reports/overview [get]
func (h *ReportsHandler) GetReportsOverview(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "UNAUTHORIZED",
				"message": "User not authenticated",
			},
		})
		return
	}

	filters, err := h.parseFilters(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INVALID_FILTERS",
				"message": err.Error(),
			},
		})
		return
	}

	overview, err := h.reportsService.GetReportsOverview(userID.(uuid.UUID), filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "Failed to get reports overview",
				"details": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    overview,
		"meta": gin.H{
			"filters": filters,
		},
	})
}

// GetTaskReport returns task-specific metrics
// @Summary Get task report
// @Description Get detailed task metrics and analytics
// @Tags reports
// @Accept json
// @Produce json
// @Success 200 {object} models.TaskReportMetrics
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/reports/tasks [get]
func (h *ReportsHandler) GetTaskReport(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "UNAUTHORIZED",
				"message": "User not authenticated",
			},
		})
		return
	}

	filters, err := h.parseFilters(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INVALID_FILTERS",
				"message": err.Error(),
			},
		})
		return
	}

	taskReport, err := h.reportsService.GetTaskReport(userID.(uuid.UUID), filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "Failed to get task report",
				"details": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    taskReport,
		"meta": gin.H{
			"filters": filters,
		},
	})
}

// GetProjectReport returns project-specific metrics
// @Summary Get project report
// @Description Get detailed project metrics and analytics
// @Tags reports
// @Accept json
// @Produce json
// @Success 200 {object} models.ProjectReportMetrics
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/reports/projects [get]
func (h *ReportsHandler) GetProjectReport(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "UNAUTHORIZED",
				"message": "User not authenticated",
			},
		})
		return
	}

	filters, err := h.parseFilters(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INVALID_FILTERS",
				"message": err.Error(),
			},
		})
		return
	}

	projectReport, err := h.reportsService.GetProjectReport(userID.(uuid.UUID), filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "Failed to get project report",
				"details": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    projectReport,
		"meta": gin.H{
			"filters": filters,
		},
	})
}

// GetAnalysisReport returns combined analysis and insights
// @Summary Get analysis report
// @Description Get comprehensive analysis with insights and trends
// @Tags reports
// @Accept json
// @Produce json
// @Success 200 {object} models.ReportsOverview
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/reports/analysis [get]
func (h *ReportsHandler) GetAnalysisReport(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "UNAUTHORIZED",
				"message": "User not authenticated",
			},
		})
		return
	}

	filters, err := h.parseFilters(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INVALID_FILTERS",
				"message": err.Error(),
			},
		})
		return
	}

	analysisReport, err := h.reportsService.GetAnalysisReport(userID.(uuid.UUID), filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "Failed to get analysis report",
				"details": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    analysisReport,
		"meta": gin.H{
			"filters":     filters,
			"report_type": "analysis",
		},
	})
}

// GetDrillDownReport returns detailed breakdown for a specific project
// @Summary Get drill-down report
// @Description Get detailed breakdown and analytics for a specific project
// @Tags reports
// @Accept json
// @Produce json
// @Param project_id path string true "Project ID"
// @Success 200 {object} models.DrillDownReport
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/reports/projects/{project_id}/drill-down [get]
func (h *ReportsHandler) GetDrillDownReport(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "UNAUTHORIZED",
				"message": "User not authenticated",
			},
		})
		return
	}

	projectIDStr := c.Param("project_id")
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INVALID_PROJECT_ID",
				"message": "Invalid project ID format",
			},
		})
		return
	}

	filters, err := h.parseFilters(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INVALID_FILTERS",
				"message": err.Error(),
			},
		})
		return
	}

	drillDownReport, err := h.reportsService.GetDrillDownReport(userID.(uuid.UUID), projectID, filters)
	if err != nil {
		if errors.Cause(err).Error() == "sql: no rows in result set" {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error": gin.H{
					"code":    "PROJECT_NOT_FOUND",
					"message": "Project not found or access denied",
				},
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "Failed to get drill-down report",
				"details": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    drillDownReport,
		"meta": gin.H{
			"project_id": projectID,
			"filters":    filters,
		},
	})
}

// GetComparisonReport returns comparison between time periods
// @Summary Get comparison report
// @Description Get comparison analytics between current and previous periods
// @Tags reports
// @Accept json
// @Produce json
// @Success 200 {object} models.ComparisonReport
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/reports/comparison [get]
func (h *ReportsHandler) GetComparisonReport(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "UNAUTHORIZED",
				"message": "User not authenticated",
			},
		})
		return
	}

	filters, err := h.parseFilters(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INVALID_FILTERS",
				"message": err.Error(),
			},
		})
		return
	}

	comparisonReport, err := h.reportsService.GetComparisonReport(userID.(uuid.UUID), filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "Failed to get comparison report",
				"details": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    comparisonReport,
		"meta": gin.H{
			"filters": filters,
		},
	})
}

// ExportReport exports report data in various formats
// @Summary Export report
// @Description Export report data as CSV, PDF, or Excel
// @Tags reports
// @Accept json
// @Produce application/octet-stream
// @Param export_request body models.ExportRequest true "Export configuration"
// @Success 200 {file} file "Exported report file"
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/reports/export [post]
func (h *ReportsHandler) ExportReport(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "UNAUTHORIZED",
				"message": "User not authenticated",
			},
		})
		return
	}

	var exportReq models.ExportRequest
	if err := c.ShouldBindJSON(&exportReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INVALID_REQUEST",
				"message": "Invalid export request format",
				"details": err.Error(),
			},
		})
		return
	}

	data, filename, err := h.reportsService.ExportReport(userID.(uuid.UUID), &exportReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "EXPORT_FAILED",
				"message": "Failed to export report",
				"details": err.Error(),
			},
		})
		return
	}

	// Set appropriate headers for file download
	var contentType string
	switch exportReq.Format {
	case "csv":
		contentType = "text/csv"
	case "pdf":
		contentType = "application/pdf"
	case "excel":
		contentType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	default:
		contentType = "application/octet-stream"
	}

	c.Header("Content-Type", contentType)
	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Header("Content-Length", strconv.Itoa(len(data)))
	
	c.Data(http.StatusOK, contentType, data)
}

// InvalidateCache clears reports cache for the user
// @Summary Invalidate reports cache
// @Description Clear cached reports data for the authenticated user
// @Tags reports
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/reports/cache/invalidate [post]
func (h *ReportsHandler) InvalidateCache(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "UNAUTHORIZED",
				"message": "User not authenticated",
			},
		})
		return
	}

	err := h.reportsService.InvalidateUserReportsCache(userID.(uuid.UUID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "CACHE_ERROR",
				"message": "Failed to invalidate cache",
				"details": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Reports cache invalidated successfully",
	})
}

// Helper function to parse query parameters into filters
func (h *ReportsHandler) parseFilters(c *gin.Context) (*models.ReportFilterRequest, error) {
	filters := &models.ReportFilterRequest{
		DateRange: models.ReportDateRangeMonthly, // Default
		Page:      1,
		Limit:     50,
	}

	// Parse date range
	if dateRange := c.Query("date_range"); dateRange != "" {
		filters.DateRange = models.ReportDateRange(dateRange)
	}

	// Parse custom dates
	if startDate := c.Query("start_date"); startDate != "" {
		if parsedDate, err := parseDate(startDate); err == nil {
			filters.StartDate = &parsedDate
		} else {
			return nil, errors.Wrap(err, "invalid start_date format, use YYYY-MM-DD")
		}
	}

	if endDate := c.Query("end_date"); endDate != "" {
		if parsedDate, err := parseDate(endDate); err == nil {
			filters.EndDate = &parsedDate
		} else {
			return nil, errors.Wrap(err, "invalid end_date format, use YYYY-MM-DD")
		}
	}

	// Parse project ID
	if projectID := c.Query("project_id"); projectID != "" {
		if parsedID, err := uuid.Parse(projectID); err == nil {
			filters.ProjectID = &parsedID
		} else {
			return nil, errors.New("invalid project_id format")
		}
	}

	// Parse status
	if status := c.Query("status"); status != "" {
		taskStatus := models.TaskStatus(status)
		filters.Status = &taskStatus
	}

	// Parse category
	if category := c.Query("category"); category != "" {
		projectCategory := models.ProjectCategory(category)
		filters.Category = &projectCategory
	}

	// Parse priority
	if priority := c.Query("priority"); priority != "" {
		priorityLevel := models.PriorityLevel(priority)
		filters.Priority = &priorityLevel
	}

	// Parse pagination
	if page := c.Query("page"); page != "" {
		if parsedPage, err := strconv.Atoi(page); err == nil && parsedPage > 0 {
			filters.Page = parsedPage
		}
	}

	if limit := c.Query("limit"); limit != "" {
		if parsedLimit, err := strconv.Atoi(limit); err == nil && parsedLimit > 0 && parsedLimit <= 100 {
			filters.Limit = parsedLimit
		}
	}

	return filters, nil
}

// Helper function to parse date strings
func parseDate(dateStr string) (time.Time, error) {
	layouts := []string{
		"2006-01-02",
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05.000Z",
		"2006-01-02 15:04:05",
	}

	for _, layout := range layouts {
		if t, err := time.Parse(layout, dateStr); err == nil {
			return t, nil
		}
	}

	return time.Time{}, errors.New("unsupported date format")
}
