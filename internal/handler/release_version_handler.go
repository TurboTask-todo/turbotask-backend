package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"quantumtask-auth-api/internal/middleware"
	"quantumtask-auth-api/internal/models"
	"quantumtask-auth-api/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ReleaseVersionHandler handles release version HTTP requests
type ReleaseVersionHandler struct {
	releaseService *service.ReleaseVersionService
}

// NewReleaseVersionHandler creates a new release version handler
func NewReleaseVersionHandler(releaseService *service.ReleaseVersionService) *ReleaseVersionHandler {
	return &ReleaseVersionHandler{
		releaseService: releaseService,
	}
}

// CreateReleaseVersion handles release version creation
// @Summary Create a new release version
// @Description Create a new release version for a project
// @Tags Release Versions
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body models.ReleaseVersion true "Release version details"
// @Success 201 {object} models.APIResponse{data=models.ReleaseVersion} "Release version created successfully"
// @Failure 400 {object} models.APIResponse "Invalid request"
// @Failure 401 {object} models.APIResponse "Unauthorized"
// @Failure 404 {object} models.APIResponse "Project not found"
// @Failure 409 {object} models.APIResponse "Version number already exists"
// @Failure 500 {object} models.APIResponse "Internal server error"
// @Router /release-versions [post]
func (h *ReleaseVersionHandler) CreateReleaseVersion(c *gin.Context) {
	// Get user ID from context
	userID, _ := middleware.GetUserID(c)

	var version models.ReleaseVersion
	if err := c.ShouldBindJSON(&version); err != nil {
		var details map[string]interface{}
		if strings.Contains(err.Error(), "parsing time") {
			details = map[string]interface{}{
				"details": "Invalid date format. Please use ISO 8601 format (YYYY-MM-DDTHH:MM:SSZ).",
			}
		} else {
			details = map[string]interface{}{
				"details": err.Error(),
			}
		}

		c.JSON(http.StatusBadRequest, models.NewErrorResponse(
			"INVALID_REQUEST",
			"Invalid request data",
			map[string]string{
				"details": fmt.Sprintf("%v", details["details"]),
			},
		))
		return
	}

	// Create release version
	createdVersion, err := h.releaseService.CreateReleaseVersion(c.Request.Context(), userID, &version)
	if err != nil {
		statusCode := http.StatusInternalServerError
		errorCode := "RELEASE_VERSION_CREATION_FAILED"

		if strings.Contains(err.Error(), "validation failed") {
			statusCode = http.StatusBadRequest
			errorCode = "VALIDATION_ERROR"
		} else if strings.Contains(err.Error(), "not found") {
			statusCode = http.StatusNotFound
			errorCode = "PROJECT_NOT_FOUND"
		} else if strings.Contains(err.Error(), "already exists") {
			statusCode = http.StatusConflict
			errorCode = "VERSION_NUMBER_EXISTS"
		}

		c.JSON(statusCode, models.NewErrorResponse(
			errorCode,
			err.Error(),
			nil,
		))
		return
	}

	c.JSON(http.StatusCreated, models.NewAPIResponse(
		"Release version created successfully",
		createdVersion,
	))
}

// GetReleaseVersion handles retrieving a single release version
// @Summary Get release version by ID
// @Description Get a release version by its ID
// @Tags Release Versions
// @Produce json
// @Security BearerAuth
// @Param id path string true "Release Version ID"
// @Param include_counts query boolean false "Include todo counts"
// @Success 200 {object} models.APIResponse{data=models.ReleaseVersion} "Release version retrieved successfully"
// @Failure 400 {object} models.APIResponse "Invalid release version ID"
// @Failure 401 {object} models.APIResponse "Unauthorized"
// @Failure 404 {object} models.APIResponse "Release version not found"
// @Failure 500 {object} models.APIResponse "Internal server error"
// @Router /release-versions/{id} [get]
func (h *ReleaseVersionHandler) GetReleaseVersion(c *gin.Context) {
	// Get user ID from context
	userID, _ := middleware.GetUserID(c)

	// Parse release version ID
	releaseIDStr := c.Param("id")
	releaseID, err := uuid.Parse(releaseIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse(
			"INVALID_RELEASE_ID",
			"Invalid release version ID format",
			nil,
		))
		return
	}

	// Parse include_counts query parameter
	includeCounts := c.DefaultQuery("include_counts", "false") == "true"

	// Get release version
	version, err := h.releaseService.GetReleaseVersion(c.Request.Context(), releaseID, userID, includeCounts)
	if err != nil {
		statusCode := http.StatusInternalServerError
		errorCode := "RELEASE_VERSION_RETRIEVAL_FAILED"

		if strings.Contains(err.Error(), "not found") {
			statusCode = http.StatusNotFound
			errorCode = "RELEASE_VERSION_NOT_FOUND"
		}

		c.JSON(statusCode, models.NewErrorResponse(
			errorCode,
			err.Error(),
			nil,
		))
		return
	}

	c.JSON(http.StatusOK, models.NewAPIResponse(
		"Release version retrieved successfully",
		version,
	))
}

// GetProjectReleaseVersions handles retrieving release versions for a project
// @Summary Get project release versions
// @Description Get all release versions for a specific project
// @Tags Release Versions
// @Produce json
// @Security BearerAuth
// @Param project_id path string true "Project ID"
// @Param include_counts query boolean false "Include todo counts"
// @Success 200 {object} models.APIResponse{data=[]models.ReleaseVersion} "Release versions retrieved successfully"
// @Failure 400 {object} models.APIResponse "Invalid project ID"
// @Failure 401 {object} models.APIResponse "Unauthorized"
// @Failure 404 {object} models.APIResponse "Project not found"
// @Failure 500 {object} models.APIResponse "Internal server error"
// @Router /projects/{project_id}/release-versions [get]
func (h *ReleaseVersionHandler) GetProjectReleaseVersions(c *gin.Context) {
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

	fmt.Println("projectID", projectID)

	// Parse include_counts query parameter
	includeCounts := c.DefaultQuery("include_counts", "false") == "true"

	// Get project release versions
	versions, err := h.releaseService.GetProjectReleaseVersions(c.Request.Context(), projectID, userID, includeCounts)
	if err != nil {
		statusCode := http.StatusInternalServerError
		errorCode := "RELEASE_VERSION_RETRIEVAL_FAILED"

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
		"Release versions retrieved successfully",
		versions,
	))
}

// UpdateReleaseVersion handles release version updates
// @Summary Update release version
// @Description Update a release version
// @Tags Release Versions
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Release Version ID"
// @Param request body map[string]interface{} true "Release version update details"
// @Success 200 {object} models.APIResponse{data=models.ReleaseVersion} "Release version updated successfully"
// @Failure 400 {object} models.APIResponse "Invalid request"
// @Failure 401 {object} models.APIResponse "Unauthorized"
// @Failure 404 {object} models.APIResponse "Release version not found"
// @Failure 409 {object} models.APIResponse "Version number already exists"
// @Failure 500 {object} models.APIResponse "Internal server error"
// @Router /release-versions/{id} [put]
func (h *ReleaseVersionHandler) UpdateReleaseVersion(c *gin.Context) {
	// Get user ID from context
	userID, _ := middleware.GetUserID(c)

	// Parse release version ID
	releaseIDStr := c.Param("id")
	releaseID, err := uuid.Parse(releaseIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse(
			"INVALID_RELEASE_ID",
			"Invalid release version ID format",
			nil,
		))
		return
	}

	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse(
			"INVALID_REQUEST",
			"Invalid request data",
			map[string]string{"details": err.Error()},
		))
		return
	}

	// Update release version
	version, err := h.releaseService.UpdateReleaseVersion(c.Request.Context(), releaseID, userID, updates)
	if err != nil {
		statusCode := http.StatusInternalServerError
		errorCode := "RELEASE_VERSION_UPDATE_FAILED"

		if strings.Contains(err.Error(), "validation failed") {
			statusCode = http.StatusBadRequest
			errorCode = "VALIDATION_ERROR"
		} else if strings.Contains(err.Error(), "not found") {
			statusCode = http.StatusNotFound
			errorCode = "RELEASE_VERSION_NOT_FOUND"
		} else if strings.Contains(err.Error(), "already exists") {
			statusCode = http.StatusConflict
			errorCode = "VERSION_NUMBER_EXISTS"
		}

		c.JSON(statusCode, models.NewErrorResponse(
			errorCode,
			err.Error(),
			nil,
		))
		return
	}

	c.JSON(http.StatusOK, models.NewAPIResponse(
		"Release version updated successfully",
		version,
	))
}

// DeleteReleaseVersion handles release version deletion
// @Summary Delete release version
// @Description Delete a release version
// @Tags Release Versions
// @Produce json
// @Security BearerAuth
// @Param id path string true "Release Version ID"
// @Success 200 {object} models.APIResponse "Release version deleted successfully"
// @Failure 400 {object} models.APIResponse "Invalid release version ID"
// @Failure 401 {object} models.APIResponse "Unauthorized"
// @Failure 404 {object} models.APIResponse "Release version not found"
// @Failure 409 {object} models.APIResponse "Cannot delete release version with assigned todos"
// @Failure 500 {object} models.APIResponse "Internal server error"
// @Router /release-versions/{id} [delete]
func (h *ReleaseVersionHandler) DeleteReleaseVersion(c *gin.Context) {
	// Get user ID from context
	userID, _ := middleware.GetUserID(c)

	// Parse release version ID
	releaseIDStr := c.Param("id")
	releaseID, err := uuid.Parse(releaseIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse(
			"INVALID_RELEASE_ID",
			"Invalid release version ID format",
			nil,
		))
		return
	}

	// Delete release version
	err = h.releaseService.DeleteReleaseVersion(c.Request.Context(), releaseID, userID)
	if err != nil {
		statusCode := http.StatusInternalServerError
		errorCode := "RELEASE_VERSION_DELETE_FAILED"

		if strings.Contains(err.Error(), "not found") {
			statusCode = http.StatusNotFound
			errorCode = "RELEASE_VERSION_NOT_FOUND"
		} else if strings.Contains(err.Error(), "assigned todos") {
			statusCode = http.StatusConflict
			errorCode = "RELEASE_VERSION_HAS_TODOS"
		}

		c.JSON(statusCode, models.NewErrorResponse(
			errorCode,
			err.Error(),
			nil,
		))
		return
	}

	c.JSON(http.StatusOK, models.NewAPIResponse(
		"Release version deleted successfully",
		nil,
	))
}

// MarkAsReleased handles marking a release version as released
// @Summary Mark release as released
// @Description Mark a release version as released
// @Tags Release Versions
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Release Version ID"
// @Param request body map[string]string true "Release notes" example({"release_notes": "Bug fixes and improvements"})
// @Success 200 {object} models.APIResponse "Release version marked as released successfully"
// @Failure 400 {object} models.APIResponse "Invalid request"
// @Failure 401 {object} models.APIResponse "Unauthorized"
// @Failure 404 {object} models.APIResponse "Release version not found"
// @Failure 409 {object} models.APIResponse "Release version already released"
// @Failure 500 {object} models.APIResponse "Internal server error"
// @Router /release-versions/{id}/release [post]
func (h *ReleaseVersionHandler) MarkAsReleased(c *gin.Context) {
	// Get user ID from context
	userID, _ := middleware.GetUserID(c)

	// Parse release version ID
	releaseIDStr := c.Param("id")
	releaseID, err := uuid.Parse(releaseIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse(
			"INVALID_RELEASE_ID",
			"Invalid release version ID format",
			nil,
		))
		return
	}

	var req map[string]string
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse(
			"INVALID_REQUEST",
			"Invalid request data",
			map[string]string{"details": err.Error()},
		))
		return
	}

	releaseNotes := req["release_notes"]

	// Mark as released
	err = h.releaseService.MarkAsReleased(c.Request.Context(), releaseID, userID, releaseNotes)
	if err != nil {
		statusCode := http.StatusInternalServerError
		errorCode := "RELEASE_MARK_FAILED"

		if strings.Contains(err.Error(), "not found") {
			statusCode = http.StatusNotFound
			errorCode = "RELEASE_VERSION_NOT_FOUND"
		} else if strings.Contains(err.Error(), "already released") {
			statusCode = http.StatusConflict
			errorCode = "RELEASE_ALREADY_RELEASED"
		}

		c.JSON(statusCode, models.NewErrorResponse(
			errorCode,
			err.Error(),
			nil,
		))
		return
	}

	c.JSON(http.StatusOK, models.NewAPIResponse(
		"Release version marked as released successfully",
		nil,
	))
}

// GetUpcomingReleases handles retrieving upcoming releases
// @Summary Get upcoming releases
// @Description Get upcoming releases for the authenticated user
// @Tags Release Versions
// @Produce json
// @Security BearerAuth
// @Param project_id query string false "Filter by project ID"
// @Param days_ahead query integer false "Days ahead to look" default(30)
// @Success 200 {object} models.APIResponse{data=[]models.ReleaseVersion} "Upcoming releases retrieved successfully"
// @Failure 400 {object} models.APIResponse "Invalid request"
// @Failure 401 {object} models.APIResponse "Unauthorized"
// @Failure 500 {object} models.APIResponse "Internal server error"
// @Router /release-versions/upcoming [get]
func (h *ReleaseVersionHandler) GetUpcomingReleases(c *gin.Context) {
	// Get user ID from context
	userID, _ := middleware.GetUserID(c)

	var projectID *uuid.UUID
	if projectIDStr := c.Query("project_id"); projectIDStr != "" {
		id, err := uuid.Parse(projectIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, models.NewErrorResponse(
				"INVALID_PROJECT_ID",
				"Invalid project ID format",
				nil,
			))
			return
		}
		projectID = &id
	}

	daysAheadStr := c.DefaultQuery("days_ahead", "30")
	daysAhead, err := strconv.Atoi(daysAheadStr)
	if err != nil || daysAhead <= 0 {
		daysAhead = 30
	}
	if daysAhead > 365 {
		daysAhead = 365 // Cap at 1 year
	}

	// Get upcoming releases
	releases, err := h.releaseService.GetUpcomingReleases(c.Request.Context(), userID, projectID, daysAhead)
	if err != nil {
		statusCode := http.StatusInternalServerError
		errorCode := "UPCOMING_RELEASES_RETRIEVAL_FAILED"

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
		"Upcoming releases retrieved successfully",
		releases,
	))
}

// GetOverdueReleases handles retrieving overdue releases
// @Summary Get overdue releases
// @Description Get overdue releases for the authenticated user
// @Tags Release Versions
// @Produce json
// @Security BearerAuth
// @Param project_id query string false "Filter by project ID"
// @Success 200 {object} models.APIResponse{data=[]models.ReleaseVersion} "Overdue releases retrieved successfully"
// @Failure 400 {object} models.APIResponse "Invalid request"
// @Failure 401 {object} models.APIResponse "Unauthorized"
// @Failure 500 {object} models.APIResponse "Internal server error"
// @Router /release-versions/overdue [get]
func (h *ReleaseVersionHandler) GetOverdueReleases(c *gin.Context) {
	// Get user ID from context
	userID, _ := middleware.GetUserID(c)

	var projectID *uuid.UUID
	if projectIDStr := c.Query("project_id"); projectIDStr != "" {
		id, err := uuid.Parse(projectIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, models.NewErrorResponse(
				"INVALID_PROJECT_ID",
				"Invalid project ID format",
				nil,
			))
			return
		}
		projectID = &id
	}

	// Get overdue releases
	releases, err := h.releaseService.GetOverdueReleases(c.Request.Context(), userID, projectID)
	if err != nil {
		statusCode := http.StatusInternalServerError
		errorCode := "OVERDUE_RELEASES_RETRIEVAL_FAILED"

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
		"Overdue releases retrieved successfully",
		releases,
	))
}

// GetReleaseStats handles retrieving release statistics
// @Summary Get release statistics
// @Description Get comprehensive statistics for a release version
// @Tags Release Versions
// @Produce json
// @Security BearerAuth
// @Param id path string true "Release Version ID"
// @Success 200 {object} models.APIResponse{data=map[string]interface{}} "Release statistics retrieved successfully"
// @Failure 400 {object} models.APIResponse "Invalid release version ID"
// @Failure 401 {object} models.APIResponse "Unauthorized"
// @Failure 404 {object} models.APIResponse "Release version not found"
// @Failure 500 {object} models.APIResponse "Internal server error"
// @Router /release-versions/{id}/stats [get]
func (h *ReleaseVersionHandler) GetReleaseStats(c *gin.Context) {
	// Get user ID from context
	userID, _ := middleware.GetUserID(c)

	// Parse release version ID
	releaseIDStr := c.Param("id")
	releaseID, err := uuid.Parse(releaseIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse(
			"INVALID_RELEASE_ID",
			"Invalid release version ID format",
			nil,
		))
		return
	}

	// Get release stats
	stats, err := h.releaseService.GetReleaseStats(c.Request.Context(), releaseID, userID)
	if err != nil {
		statusCode := http.StatusInternalServerError
		errorCode := "RELEASE_STATS_RETRIEVAL_FAILED"

		if strings.Contains(err.Error(), "not found") {
			statusCode = http.StatusNotFound
			errorCode = "RELEASE_VERSION_NOT_FOUND"
		}

		c.JSON(statusCode, models.NewErrorResponse(
			errorCode,
			err.Error(),
			nil,
		))
		return
	}

	c.JSON(http.StatusOK, models.NewAPIResponse(
		"Release statistics retrieved successfully",
		stats,
	))
}

// GetReleaseTodosProgress handles retrieving todos with progress for a release
// @Summary Get release todos progress
// @Description Get todos with progress information for a release version
// @Tags Release Versions
// @Produce json
// @Security BearerAuth
// @Param id path string true "Release Version ID"
// @Success 200 {object} models.APIResponse{data=[]models.Todo} "Release todos progress retrieved successfully"
// @Failure 400 {object} models.APIResponse "Invalid release version ID"
// @Failure 401 {object} models.APIResponse "Unauthorized"
// @Failure 404 {object} models.APIResponse "Release version not found"
// @Failure 500 {object} models.APIResponse "Internal server error"
// @Router /release-versions/{id}/todos [get]
func (h *ReleaseVersionHandler) GetReleaseTodosProgress(c *gin.Context) {
	// Get user ID from context
	userID, _ := middleware.GetUserID(c)

	// Parse release version ID
	releaseIDStr := c.Param("id")
	releaseID, err := uuid.Parse(releaseIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse(
			"INVALID_RELEASE_ID",
			"Invalid release version ID format",
			nil,
		))
		return
	}

	// Get release todos progress
	todos, err := h.releaseService.GetReleaseTodosProgress(c.Request.Context(), releaseID, userID)
	if err != nil {
		statusCode := http.StatusInternalServerError
		errorCode := "RELEASE_TODOS_RETRIEVAL_FAILED"

		if strings.Contains(err.Error(), "not found") {
			statusCode = http.StatusNotFound
			errorCode = "RELEASE_VERSION_NOT_FOUND"
		}

		c.JSON(statusCode, models.NewErrorResponse(
			errorCode,
			err.Error(),
			nil,
		))
		return
	}

	c.JSON(http.StatusOK, models.NewAPIResponse(
		"Release todos progress retrieved successfully",
		todos,
	))
}
