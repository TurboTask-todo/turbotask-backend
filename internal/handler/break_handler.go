package handler

import (
	"net/http"
	"strconv"

	"quantumtask-auth-api/internal/models"
	"quantumtask-auth-api/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// BreakHandler handles break-related HTTP requests
type BreakHandler struct {
	breakService service.BreakService
}

// NewBreakHandler creates a new break handler
func NewBreakHandler(breakService service.BreakService) *BreakHandler {
	return &BreakHandler{
		breakService: breakService,
	}
}

// StartBreak starts a new break session
// @Summary Start break
// @Description Start a new break session for the current user
// @Tags breaks
// @Accept json
// @Produce json
// @Param request body models.StartBreakRequest true "Start break request"
// @Success 201 {object} models.BreakHistory
// @Failure 400 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Failure 500 {object} models.APIResponse
// @Router /api/v1/breaks/start [post]
func (h *BreakHandler) StartBreak(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, models.NewErrorResponse(
			"UNAUTHORIZED",
			"User not authenticated",
			nil,
		))
		return
	}

	var req models.StartBreakRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse(
			"INVALID_REQUEST",
			"Invalid request body: "+err.Error(),
			nil,
		))
		return
	}

	breakSession, err := h.breakService.StartBreak(c.Request.Context(), userID.(uuid.UUID), &req)
	if err != nil {
		if err.Error() == "user already has an active break session" {
			c.JSON(http.StatusConflict, models.NewErrorResponse(
				"ACTIVE_BREAK_EXISTS",
				err.Error(),
				nil,
			))
			return
		}
		if err.Error() == "todo not found" || err.Error() == "project not found" {
			c.JSON(http.StatusNotFound, models.NewErrorResponse(
				"RESOURCE_NOT_FOUND",
				err.Error(),
				nil,
			))
			return
		}
		if err.Error() == "todo does not belong to user" || err.Error() == "project does not belong to user" {
			c.JSON(http.StatusForbidden, models.NewErrorResponse(
				"ACCESS_DENIED",
				err.Error(),
				nil,
			))
			return
		}
		c.JSON(http.StatusInternalServerError, models.NewErrorResponse(
			"INTERNAL_ERROR",
			"Failed to start break: "+err.Error(),
			nil,
		))
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"data": breakSession,
	})
}

// StopBreak stops the current active break session
// @Summary Stop break
// @Description Stop the current active break session
// @Tags breaks
// @Accept json
// @Produce json
// @Param request body models.StopBreakRequest true "Stop break request"
// @Success 200 {object} models.BreakHistory
// @Failure 400 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Failure 404 {object} models.APIResponse
// @Failure 500 {object} models.APIResponse
// @Router /api/v1/breaks/stop [post]
func (h *BreakHandler) StopBreak(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, models.NewErrorResponse(
			"UNAUTHORIZED",
			"User not authenticated",
			nil,
		))
		return
	}

	var req models.StopBreakRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse(
			"INVALID_REQUEST",
			"Invalid request body: "+err.Error(),
			nil,
		))
		return
	}

	breakSession, err := h.breakService.StopBreak(c.Request.Context(), userID.(uuid.UUID), &req)
	if err != nil {
		if err.Error() == "no active break session found" {
			c.JSON(http.StatusNotFound, models.NewErrorResponse(
				"NO_ACTIVE_BREAK",
				err.Error(),
				nil,
			))
			return
		}
		c.JSON(http.StatusInternalServerError, models.NewErrorResponse(
			"INTERNAL_ERROR",
			"Failed to stop break: "+err.Error(),
			nil,
		))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": breakSession,
	})
}

// GetActiveBreak gets the current active break session
// @Summary Get active break
// @Description Get the current active break session for the user
// @Tags breaks
// @Produce json
// @Success 200 {object} models.BreakHistory
// @Success 204 "No active break"
// @Failure 401 {object} models.APIResponse
// @Failure 500 {object} models.APIResponse
// @Router /api/v1/breaks/active [get]
func (h *BreakHandler) GetActiveBreak(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, models.NewErrorResponse(
			"UNAUTHORIZED",
			"User not authenticated",
			nil,
		))
		return
	}

	activeBreak, err := h.breakService.GetActiveBreak(c.Request.Context(), userID.(uuid.UUID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.NewErrorResponse(
			"INTERNAL_ERROR",
			"Failed to get active break: "+err.Error(),
			nil,
		))
		return
	}

	if activeBreak == nil {
		c.Status(http.StatusNoContent)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": activeBreak,
	})
}

// GetBreakHistory gets break history for the user
// @Summary Get break history
// @Description Get break history for the current user with pagination
// @Tags breaks
// @Produce json
// @Param limit query int false "Limit" default(20)
// @Param offset query int false "Offset" default(0)
// @Success 200 {array} models.BreakHistory
// @Failure 401 {object} models.APIResponse
// @Failure 500 {object} models.APIResponse
// @Router /api/v1/breaks/history [get]
func (h *BreakHandler) GetBreakHistory(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, models.NewErrorResponse(
			"UNAUTHORIZED",
			"User not authenticated",
			nil,
		))
		return
	}

	// Parse pagination parameters
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	if limit > 100 {
		limit = 100 // Cap at 100
	}

	breakHistory, err := h.breakService.GetBreakHistory(c.Request.Context(), userID.(uuid.UUID), limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.NewErrorResponse(
			"INTERNAL_ERROR",
			"Failed to get break history: "+err.Error(),
			nil,
		))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": breakHistory,
	})
}

// GetBreaksByTodo gets break history for a specific todo
// @Summary Get breaks by todo
// @Description Get break history for a specific todo
// @Tags breaks
// @Produce json
// @Param todo_id path string true "Todo ID"
// @Param limit query int false "Limit" default(20)
// @Param offset query int false "Offset" default(0)
// @Success 200 {array} models.BreakHistory
// @Failure 400 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Failure 500 {object} models.APIResponse
// @Router /api/v1/breaks/todo/{todo_id} [get]
func (h *BreakHandler) GetBreaksByTodo(c *gin.Context) {
	_, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, models.NewErrorResponse(
			"UNAUTHORIZED",
			"User not authenticated",
			nil,
		))
		return
	}

	todoIDStr := c.Param("todo_id")
	todoID, err := uuid.Parse(todoIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse(
			"INVALID_TODO_ID",
			"Invalid todo ID format",
			nil,
		))
		return
	}

	// Parse pagination parameters
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	if limit > 100 {
		limit = 100 // Cap at 100
	}

	breakHistory, err := h.breakService.GetBreaksByTodo(c.Request.Context(), todoID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.NewErrorResponse(
			"INTERNAL_ERROR",
			"Failed to get breaks by todo: "+err.Error(),
			nil,
		))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": breakHistory,
	})
}

// GetBreaksByProject gets break history for a specific project
// @Summary Get breaks by project
// @Description Get break history for a specific project
// @Tags breaks
// @Produce json
// @Param project_id path string true "Project ID"
// @Param limit query int false "Limit" default(20)
// @Param offset query int false "Offset" default(0)
// @Success 200 {array} models.BreakHistory
// @Failure 400 {object} models.APIResponse
// @Failure 401 {object} models.APIResponse
// @Failure 500 {object} models.APIResponse
// @Router /api/v1/breaks/project/{project_id} [get]
func (h *BreakHandler) GetBreaksByProject(c *gin.Context) {
	_, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, models.NewErrorResponse(
			"UNAUTHORIZED",
			"User not authenticated",
			nil,
		))
		return
	}

	projectIDStr := c.Param("project_id")
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse(
			"INVALID_PROJECT_ID",
			"Invalid project ID format",
			nil,
		))
		return
	}

	// Parse pagination parameters
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	if limit > 100 {
		limit = 100 // Cap at 100
	}

	breakHistory, err := h.breakService.GetBreaksByProject(c.Request.Context(), projectID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.NewErrorResponse(
			"INTERNAL_ERROR",
			"Failed to get breaks by project: "+err.Error(),
			nil,
		))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": breakHistory,
	})
}

// GetBreakStats gets break statistics for the user
// @Summary Get break statistics
// @Description Get break statistics for the current user
// @Tags breaks
// @Produce json
// @Success 200 {object} models.BreakStatsResponse
// @Failure 401 {object} models.APIResponse
// @Failure 500 {object} models.APIResponse
// @Router /api/v1/breaks/stats [get]
func (h *BreakHandler) GetBreakStats(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, models.NewErrorResponse(
			"UNAUTHORIZED",
			"User not authenticated",
			nil,
		))
		return
	}

	stats, err := h.breakService.GetBreakStats(c.Request.Context(), userID.(uuid.UUID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.NewErrorResponse(
			"INTERNAL_ERROR",
			"Failed to get break stats: "+err.Error(),
			nil,
		))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": stats,
	})
}
