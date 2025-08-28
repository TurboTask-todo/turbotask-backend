package handler

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"quantumtask-auth-api/internal/middleware"
	"quantumtask-auth-api/internal/models"
	"quantumtask-auth-api/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ProjectHandler handles project HTTP requests
type ProjectHandler struct {
	projectService *service.ProjectService
}

// NewProjectHandler creates a new project handler
func NewProjectHandler(projectService *service.ProjectService) *ProjectHandler {
	return &ProjectHandler{
		projectService: projectService,
	}
}

// CreateProject handles project creation
// @Summary Create a new project
// @Description Create a new project for the authenticated user
// @Tags Projects
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body models.CreateProjectRequest true "Project details"
// @Success 201 {object} models.APIResponse{data=models.Project} "Project created successfully"
// @Failure 400 {object} models.APIResponse "Invalid request"
// @Failure 401 {object} models.APIResponse "Unauthorized"
// @Failure 500 {object} models.APIResponse "Internal server error"
// @Router /projects [post]
func (h *ProjectHandler) CreateProject(c *gin.Context) {
	// Get user ID from context
	userID, _ := middleware.GetUserID(c)

	var req models.CreateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse(
			"INVALID_REQUEST",
			"Invalid request data",
			map[string]string{"details": err.Error()},
		))
		return
	}

	// Create project
	project, err := h.projectService.CreateProject(c.Request.Context(), userID, &req)
	if err != nil {
		statusCode := http.StatusInternalServerError
		errorCode := "PROJECT_CREATION_FAILED"

		if strings.Contains(err.Error(), "validation failed") {
			statusCode = http.StatusBadRequest
			errorCode = "VALIDATION_ERROR"
		}

		c.JSON(statusCode, models.NewErrorResponse(
			errorCode,
			err.Error(),
			nil,
		))
		return
	}

	c.JSON(http.StatusCreated, models.NewAPIResponse(
		"Project created successfully",
		project,
	))
}

// GetProject handles retrieving a single project
// @Summary Get project by ID
// @Description Get a project by its ID for the authenticated user
// @Tags Projects
// @Produce json
// @Security BearerAuth
// @Param id path string true "Project ID"
// @Param include_counts query boolean false "Include todo counts"
// @Success 200 {object} models.APIResponse{data=models.Project} "Project retrieved successfully"
// @Failure 400 {object} models.APIResponse "Invalid project ID"
// @Failure 401 {object} models.APIResponse "Unauthorized"
// @Failure 404 {object} models.APIResponse "Project not found"
// @Failure 500 {object} models.APIResponse "Internal server error"
// @Router /projects/{id} [get]
func (h *ProjectHandler) GetProject(c *gin.Context) {
	// Get user ID from context
	userID, _ := middleware.GetUserID(c)

	// Parse project ID
	projectIDStr := c.Param("id")
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse(
			"INVALID_PROJECT_ID",
			"Invalid project ID format",
			nil,
		))
		return
	}

	// Parse include_counts query parameter
	includeCounts := c.DefaultQuery("include_counts", "false") == "true"

	// Get project
	project, err := h.projectService.GetProject(c.Request.Context(), projectID, userID, includeCounts)
	if err != nil {
		statusCode := http.StatusInternalServerError
		errorCode := "PROJECT_RETRIEVAL_FAILED"

		if strings.Contains(err.Error(), "not found") {
			statusCode = http.StatusNotFound
			errorCode = "PROJECT_NOT_FOUND"
		}

		c.JSON(statusCode, models.NewErrorResponse(
			errorCode,
			err.Error(),
			nil,
		))
		return
	}

	c.JSON(http.StatusOK, models.NewAPIResponse(
		"Project retrieved successfully",
		project,
	))
}

// GetUserProjects handles retrieving all projects for a user
// @Summary Get user projects
// @Description Get all projects for the authenticated user
// @Tags Projects
// @Produce json
// @Security BearerAuth
// @Param include_archived query boolean false "Include archived projects"
// @Param include_counts query boolean false "Include todo counts"
// @Success 200 {object} models.APIResponse{data=[]models.Project} "Projects retrieved successfully"
// @Failure 401 {object} models.APIResponse "Unauthorized"
// @Failure 500 {object} models.APIResponse "Internal server error"
// @Router /projects [get]
func (h *ProjectHandler) GetUserProjects(c *gin.Context) {
	// Get user ID from context
	userID, _ := middleware.GetUserID(c)

	// Parse query parameters
	includeArchived := c.DefaultQuery("include_archived", "false") == "true"
	includeCounts := c.DefaultQuery("include_counts", "false") == "true"

	// Get projects
	projects, err := h.projectService.GetUserProjects(c.Request.Context(), userID, includeArchived, includeCounts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.NewErrorResponse(
			"PROJECT_RETRIEVAL_FAILED",
			err.Error(),
			nil,
		))
		return
	}

	c.JSON(http.StatusOK, models.NewAPIResponse(
		"Projects retrieved successfully",
		projects,
	))
}

// GetProjectsPaginated retrieves projects for a user with pagination
// @Summary Get projects with pagination
// @Description Retrieve user's projects with advanced pagination, filtering, and sorting
// @Tags projects
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query integer false "Page number" default(1) minimum(1)
// @Param page_size query integer false "Number of items per page" default(20) minimum(1) maximum(100)
// @Param sort query string false "Sort field" Enums(title,created_at,updated_at,category,is_favorite,progress_percentage) default(updated_at)
// @Param order query string false "Sort order" Enums(asc,desc) default(desc)
// @Param include_archived query boolean false "Include archived projects" default(false)
// @Param include_counts query boolean false "Include todo counts" default(false)
// @Param category query string false "Filter by category"
// @Param is_favorite query boolean false "Filter by favorite status"
// @Param search query string false "Search in title and description"
// @Success 200 {object} models.PaginatedResponse{data=models.ProjectsResponse}
// @Failure 400 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Failure 500 {object} models.APIResponse
// @Router /api/v1/todo/projects/paginated [get]
func (h *ProjectHandler) GetProjectsPaginated(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	var req models.ProjectsQueryRequest

	// Bind query parameters
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid query parameters",
			"details": err.Error(),
		})
		return
	}

	// Set defaults
	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize < 1 {
		req.PageSize = 20
	}
	if req.PageSize > 100 {
		req.PageSize = 100
	}

	projects, totalCount, err := h.projectService.GetProjectsPaginated(c.Request.Context(), userID.(uuid.UUID), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Calculate pagination metadata
	totalPages := (totalCount + req.PageSize - 1) / req.PageSize
	hasNextPage := req.Page < totalPages
	hasPrevPage := req.Page > 1

	var nextPage, prevPage *int
	if hasNextPage {
		next := req.Page + 1
		nextPage = &next
	}
	if hasPrevPage {
		prev := req.Page - 1
		prevPage = &prev
	}

	pagination := models.PaginationMeta{
		Page:        req.Page,
		PageSize:    req.PageSize,
		TotalCount:  totalCount,
		TotalPages:  totalPages,
		HasNextPage: hasNextPage,
		HasPrevPage: hasPrevPage,
		NextPage:    nextPage,
		PrevPage:    prevPage,
	}

	response := models.PaginatedResponse{
		Success: true,
		Message: "Projects retrieved successfully",
		Data: models.ProjectsResponse{
			Projects:   projects,
			Pagination: pagination,
		},
		Pagination: pagination,
		Timestamp:  time.Now(),
	}

	c.JSON(http.StatusOK, response)
}

// UpdateProject handles project updates
// @Summary Update project
// @Description Update a project for the authenticated user
// @Tags Projects
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Project ID"
// @Param request body models.UpdateProjectRequest true "Project update details"
// @Success 200 {object} models.APIResponse{data=models.Project} "Project updated successfully"
// @Failure 400 {object} models.APIResponse "Invalid request"
// @Failure 401 {object} models.APIResponse "Unauthorized"
// @Failure 404 {object} models.APIResponse "Project not found"
// @Failure 500 {object} models.APIResponse "Internal server error"
// @Router /projects/{id} [put]
func (h *ProjectHandler) UpdateProject(c *gin.Context) {
	// Get user ID from context
	userID, _ := middleware.GetUserID(c)

	// Parse project ID
	projectIDStr := c.Param("id")
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse(
			"INVALID_PROJECT_ID",
			"Invalid project ID format",
			nil,
		))
		return
	}

	var req models.UpdateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse(
			"INVALID_REQUEST",
			"Invalid request data",
			map[string]string{"details": err.Error()},
		))
		return
	}

	// Update project
	project, err := h.projectService.UpdateProject(c.Request.Context(), projectID, userID, &req)
	if err != nil {
		statusCode := http.StatusInternalServerError
		errorCode := "PROJECT_UPDATE_FAILED"

		if strings.Contains(err.Error(), "validation failed") {
			statusCode = http.StatusBadRequest
			errorCode = "VALIDATION_ERROR"
		} else if strings.Contains(err.Error(), "not found") {
			statusCode = http.StatusNotFound
			errorCode = "PROJECT_NOT_FOUND"
		}

		c.JSON(statusCode, models.NewErrorResponse(
			errorCode,
			err.Error(),
			nil,
		))
		return
	}

	c.JSON(http.StatusOK, models.NewAPIResponse(
		"Project updated successfully",
		project,
	))
}

// ArchiveProject handles project archiving
// @Summary Archive project
// @Description Archive a project for the authenticated user
// @Tags Projects
// @Produce json
// @Security BearerAuth
// @Param id path string true "Project ID"
// @Success 200 {object} models.APIResponse "Project archived successfully"
// @Failure 400 {object} models.APIResponse "Invalid project ID"
// @Failure 401 {object} models.APIResponse "Unauthorized"
// @Failure 404 {object} models.APIResponse "Project not found"
// @Failure 500 {object} models.APIResponse "Internal server error"
// @Router /projects/{id}/archive [post]
func (h *ProjectHandler) ArchiveProject(c *gin.Context) {
	// Get user ID from context
	userID, _ := middleware.GetUserID(c)

	// Parse project ID
	projectIDStr := c.Param("id")
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse(
			"INVALID_PROJECT_ID",
			"Invalid project ID format",
			nil,
		))
		return
	}

	// Archive project
	err = h.projectService.ArchiveProject(c.Request.Context(), projectID, userID)
	if err != nil {
		statusCode := http.StatusInternalServerError
		errorCode := "PROJECT_ARCHIVE_FAILED"

		if strings.Contains(err.Error(), "not found") {
			statusCode = http.StatusNotFound
			errorCode = "PROJECT_NOT_FOUND"
		}

		c.JSON(statusCode, models.NewErrorResponse(
			errorCode,
			err.Error(),
			nil,
		))
		return
	}

	c.JSON(http.StatusOK, models.NewAPIResponse(
		"Project archived successfully",
		nil,
	))
}

// RestoreProject handles project restoration
// @Summary Restore project
// @Description Restore an archived project for the authenticated user
// @Tags Projects
// @Produce json
// @Security BearerAuth
// @Param id path string true "Project ID"
// @Success 200 {object} models.APIResponse "Project restored successfully"
// @Failure 400 {object} models.APIResponse "Invalid project ID"
// @Failure 401 {object} models.APIResponse "Unauthorized"
// @Failure 404 {object} models.APIResponse "Project not found"
// @Failure 500 {object} models.APIResponse "Internal server error"
// @Router /projects/{id}/restore [post]
func (h *ProjectHandler) RestoreProject(c *gin.Context) {
	// Get user ID from context
	userID, _ := middleware.GetUserID(c)

	// Parse project ID
	projectIDStr := c.Param("id")
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse(
			"INVALID_PROJECT_ID",
			"Invalid project ID format",
			nil,
		))
		return
	}

	// Restore project
	err = h.projectService.RestoreProject(c.Request.Context(), projectID, userID)
	if err != nil {
		statusCode := http.StatusInternalServerError
		errorCode := "PROJECT_RESTORE_FAILED"

		if strings.Contains(err.Error(), "not found") {
			statusCode = http.StatusNotFound
			errorCode = "PROJECT_NOT_FOUND"
		}

		c.JSON(statusCode, models.NewErrorResponse(
			errorCode,
			err.Error(),
			nil,
		))
		return
	}

	c.JSON(http.StatusOK, models.NewAPIResponse(
		"Project restored successfully",
		nil,
	))
}

// DeleteProject handles project deletion
// @Summary Delete project
// @Description Permanently delete a project for the authenticated user
// @Tags Projects
// @Produce json
// @Security BearerAuth
// @Param id path string true "Project ID"
// @Success 200 {object} models.APIResponse "Project deleted successfully"
// @Failure 400 {object} models.APIResponse "Invalid project ID"
// @Failure 401 {object} models.APIResponse "Unauthorized"
// @Failure 404 {object} models.APIResponse "Project not found"
// @Failure 500 {object} models.APIResponse "Internal server error"
// @Router /projects/{id} [delete]
func (h *ProjectHandler) DeleteProject(c *gin.Context) {
	// Get user ID from context
	userID, _ := middleware.GetUserID(c)

	// Parse project ID
	projectIDStr := c.Param("id")
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse(
			"INVALID_PROJECT_ID",
			"Invalid project ID format",
			nil,
		))
		return
	}

	// Delete project
	err = h.projectService.DeleteProject(c.Request.Context(), projectID, userID)
	if err != nil {
		statusCode := http.StatusInternalServerError
		errorCode := "PROJECT_DELETE_FAILED"

		if strings.Contains(err.Error(), "not found") {
			statusCode = http.StatusNotFound
			errorCode = "PROJECT_NOT_FOUND"
		}

		c.JSON(statusCode, models.NewErrorResponse(
			errorCode,
			err.Error(),
			nil,
		))
		return
	}

	c.JSON(http.StatusOK, models.NewAPIResponse(
		"Project deleted successfully",
		nil,
	))
}

// GetProjectStats handles retrieving project statistics
// @Summary Get project statistics
// @Description Get comprehensive statistics for user projects
// @Tags Projects
// @Produce json
// @Security BearerAuth
// @Param include_archived query boolean false "Include archived projects"
// @Success 200 {object} models.APIResponse{data=[]models.ProjectStats} "Project statistics retrieved successfully"
// @Failure 401 {object} models.APIResponse "Unauthorized"
// @Failure 500 {object} models.APIResponse "Internal server error"
// @Router /projects/stats [get]
func (h *ProjectHandler) GetProjectStats(c *gin.Context) {
	// Get user ID from context
	userID, _ := middleware.GetUserID(c)

	// Parse query parameters
	includeArchived := c.DefaultQuery("include_archived", "false") == "true"

	// Get project stats
	stats, err := h.projectService.GetProjectStats(c.Request.Context(), userID, includeArchived)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.NewErrorResponse(
			"STATS_RETRIEVAL_FAILED",
			err.Error(),
			nil,
		))
		return
	}

	c.JSON(http.StatusOK, models.NewAPIResponse(
		"Project statistics retrieved successfully",
		stats,
	))
}

// GetProjectsByCategory handles retrieving projects by category
// @Summary Get projects by category
// @Description Get projects filtered by category for the authenticated user
// @Tags Projects
// @Produce json
// @Security BearerAuth
// @Param category path string true "Project category" Enums(personal,work,education,health,finance,hobby,other)
// @Success 200 {object} models.APIResponse{data=[]models.Project} "Projects retrieved successfully"
// @Failure 400 {object} models.APIResponse "Invalid category"
// @Failure 401 {object} models.APIResponse "Unauthorized"
// @Failure 500 {object} models.APIResponse "Internal server error"
// @Router /projects/category/{category} [get]
func (h *ProjectHandler) GetProjectsByCategory(c *gin.Context) {
	// Get user ID from context
	userID, _ := middleware.GetUserID(c)

	// Parse category
	categoryStr := c.Param("category")
	category := models.ProjectCategory(categoryStr)
	if !category.IsValid() {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse(
			"INVALID_CATEGORY",
			"Invalid project category",
			nil,
		))
		return
	}

	// Get projects by category
	projects, err := h.projectService.GetProjectsByCategory(c.Request.Context(), userID, category)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.NewErrorResponse(
			"PROJECT_RETRIEVAL_FAILED",
			err.Error(),
			nil,
		))
		return
	}

	c.JSON(http.StatusOK, models.NewAPIResponse(
		"Projects retrieved successfully",
		projects,
	))
}

// GetFavoriteProjects handles retrieving favorite projects
// @Summary Get favorite projects
// @Description Get favorite projects for the authenticated user
// @Tags Projects
// @Produce json
// @Security BearerAuth
// @Success 200 {object} models.APIResponse{data=[]models.Project} "Favorite projects retrieved successfully"
// @Failure 401 {object} models.APIResponse "Unauthorized"
// @Failure 500 {object} models.APIResponse "Internal server error"
// @Router /projects/favorites [get]
func (h *ProjectHandler) GetFavoriteProjects(c *gin.Context) {
	// Get user ID from context
	userID, _ := middleware.GetUserID(c)

	// Get favorite projects
	projects, err := h.projectService.GetFavoriteProjects(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.NewErrorResponse(
			"PROJECT_RETRIEVAL_FAILED",
			err.Error(),
			nil,
		))
		return
	}

	c.JSON(http.StatusOK, models.NewAPIResponse(
		"Favorite projects retrieved successfully",
		projects,
	))
}

// SearchProjects handles project search
// @Summary Search projects
// @Description Search projects by title or description for the authenticated user
// @Tags Projects
// @Produce json
// @Security BearerAuth
// @Param q query string true "Search query"
// @Param limit query integer false "Limit number of results" default(20)
// @Param offset query integer false "Offset for pagination" default(0)
// @Success 200 {object} models.APIResponse{data=map[string]interface{}} "Projects search results"
// @Failure 400 {object} models.APIResponse "Invalid request"
// @Failure 401 {object} models.APIResponse "Unauthorized"
// @Failure 500 {object} models.APIResponse "Internal server error"
// @Router /projects/search [get]
func (h *ProjectHandler) SearchProjects(c *gin.Context) {
	// Get user ID from context
	userID, _ := middleware.GetUserID(c)

	// Parse query parameters
	searchTerm := c.Query("q")
	if searchTerm == "" {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse(
			"MISSING_SEARCH_TERM",
			"Search query is required",
			nil,
		))
		return
	}

	limitStr := c.DefaultQuery("limit", "20")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100 // Cap at 100
	}

	offsetStr := c.DefaultQuery("offset", "0")
	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = 0
	}

	// Search projects
	projects, total, err := h.projectService.SearchProjects(c.Request.Context(), userID, searchTerm, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.NewErrorResponse(
			"SEARCH_FAILED",
			err.Error(),
			nil,
		))
		return
	}

	result := map[string]interface{}{
		"projects": projects,
		"total":    total,
		"limit":    limit,
		"offset":   offset,
	}

	c.JSON(http.StatusOK, models.NewAPIResponse(
		"Projects search completed successfully",
		result,
	))
}

// GetProjectDashboard handles retrieving project dashboard data
// @Summary Get project dashboard
// @Description Get project dashboard data for the authenticated user
// @Tags Projects
// @Produce json
// @Security BearerAuth
// @Success 200 {object} models.APIResponse{data=[]models.Project} "Project dashboard retrieved successfully"
// @Failure 401 {object} models.APIResponse "Unauthorized"
// @Failure 500 {object} models.APIResponse "Internal server error"
// @Router /projects/dashboard [get]
func (h *ProjectHandler) GetProjectDashboard(c *gin.Context) {
	// Get user ID from context
	userID, _ := middleware.GetUserID(c)

	// Get project dashboard
	projects, err := h.projectService.GetProjectDashboard(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.NewErrorResponse(
			"DASHBOARD_RETRIEVAL_FAILED",
			err.Error(),
			nil,
		))
		return
	}

	c.JSON(http.StatusOK, models.NewAPIResponse(
		"Project dashboard retrieved successfully",
		projects,
	))
}

// ExportProject handles exporting project data as ZIP file
// @Summary Export project data
// @Description Export project task data as a ZIP file containing CSV
// @Tags Projects
// @Produce application/zip
// @Security BearerAuth
// @Param id path string true "Project ID"
// @Success 200 {file} file "ZIP file containing project data"
// @Failure 400 {object} models.APIResponse "Invalid project ID"
// @Failure 401 {object} models.APIResponse "Unauthorized"
// @Failure 403 {object} models.APIResponse "Access denied"
// @Failure 404 {object} models.APIResponse "Project not found"
// @Failure 500 {object} models.APIResponse "Internal server error"
// @Router /projects/{id}/export [get]
func (h *ProjectHandler) ExportProject(c *gin.Context) {
	// Get user ID from context
	userID, _ := middleware.GetUserID(c)

	// Parse project ID
	projectIDStr := c.Param("id")
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse(
			"INVALID_PROJECT_ID",
			"Invalid project ID format",
			nil,
		))
		return
	}

	// Export project
	zipData, fileName, err := h.projectService.ExportProject(c.Request.Context(), userID, projectID)
	if err != nil {
		statusCode := http.StatusInternalServerError
		errorCode := "EXPORT_FAILED"

		if strings.Contains(err.Error(), "not found") {
			statusCode = http.StatusNotFound
			errorCode = "PROJECT_NOT_FOUND"
		} else if strings.Contains(err.Error(), "access denied") {
			statusCode = http.StatusForbidden
			errorCode = "ACCESS_DENIED"
		}

		c.JSON(statusCode, models.NewErrorResponse(
			errorCode,
			err.Error(),
			nil,
		))
		return
	}

	// Set headers for file download
	c.Header("Content-Type", "application/zip")
	c.Header("Content-Disposition", "attachment; filename="+fileName)
	c.Header("Content-Length", strconv.Itoa(len(zipData)))

	// Send the ZIP file
	c.Data(http.StatusOK, "application/zip", zipData)
}

// ImportProject handles importing project data from ZIP file
// @Summary Import project data
// @Description Import project task data from a ZIP file containing CSV
// @Tags Projects
// @Accept multipart/form-data
// @Produce json
// @Security BearerAuth
// @Param id path string true "Project ID"
// @Param file formData file true "ZIP file containing project data"
// @Success 200 {object} models.APIResponse{data=models.ImportResult} "Import completed successfully"
// @Failure 400 {object} models.APIResponse "Invalid request or file format"
// @Failure 401 {object} models.APIResponse "Unauthorized"
// @Failure 403 {object} models.APIResponse "Access denied"
// @Failure 404 {object} models.APIResponse "Project not found"
// @Failure 500 {object} models.APIResponse "Internal server error"
// @Router /projects/{id}/import [post]
func (h *ProjectHandler) ImportProject(c *gin.Context) {
	// Get user ID from context
	userID, _ := middleware.GetUserID(c)

	// Parse project ID
	projectIDStr := c.Param("id")
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse(
			"INVALID_PROJECT_ID",
			"Invalid project ID format",
			nil,
		))
		return
	}

	// Get uploaded file
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse(
			"INVALID_FILE",
			"No file provided or invalid file format",
			nil,
		))
		return
	}
	defer file.Close()

	// Validate file extension
	if !strings.HasSuffix(strings.ToLower(header.Filename), ".zip") {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse(
			"INVALID_FILE_FORMAT",
			"File must be a ZIP archive",
			nil,
		))
		return
	}

	// Import project
	result, err := h.projectService.ImportProject(c.Request.Context(), userID, projectID, file, header.Size)
	if err != nil {
		statusCode := http.StatusInternalServerError
		errorCode := "IMPORT_FAILED"

		if strings.Contains(err.Error(), "not found") {
			statusCode = http.StatusNotFound
			errorCode = "PROJECT_NOT_FOUND"
		} else if strings.Contains(err.Error(), "access denied") {
			statusCode = http.StatusForbidden
			errorCode = "ACCESS_DENIED"
		} else if strings.Contains(err.Error(), "invalid format") {
			statusCode = http.StatusBadRequest
			errorCode = "INVALID_FILE_FORMAT"
		}

		c.JSON(statusCode, models.NewErrorResponse(
			errorCode,
			err.Error(),
			nil,
		))
		return
	}

	c.JSON(http.StatusOK, models.NewAPIResponse(
		"Project import completed successfully",
		result,
	))
}
