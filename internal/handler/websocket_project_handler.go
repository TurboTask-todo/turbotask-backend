package handler

import (
	"net/http"

	"macwrite-auth-api/internal/models"
	"macwrite-auth-api/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// WebSocketProjectHandler handles WebSocket-enabled project operations
type WebSocketProjectHandler struct {
	projectService *service.WebSocketProjectService
	wsHandler      *WebSocketHandler
}

// NewWebSocketProjectHandler creates a new WebSocket project handler
func NewWebSocketProjectHandler(projectService *service.WebSocketProjectService, wsHandler *WebSocketHandler) *WebSocketProjectHandler {
	return &WebSocketProjectHandler{
		projectService: projectService,
		wsHandler:      wsHandler,
	}
}

// CreateProject creates a new project with WebSocket notification
func (h *WebSocketProjectHandler) CreateProject(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	var req models.CreateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userUUID := userID.(uuid.UUID)
	sessionID := c.GetHeader("X-Session-ID")

	project, err := h.projectService.CreateProject(c.Request.Context(), userUUID, &req, getSessionIDPtr(sessionID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    project,
	})
}

// UpdateProject updates a project with WebSocket notification
func (h *WebSocketProjectHandler) UpdateProject(c *gin.Context) {
	projectID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project ID"})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	var req models.UpdateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userUUID := userID.(uuid.UUID)
	sessionID := c.GetHeader("X-Session-ID")

	project, err := h.projectService.UpdateProject(c.Request.Context(), projectID, userUUID, &req, getSessionIDPtr(sessionID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    project,
	})
}

// DeleteProject deletes a project with WebSocket notification
func (h *WebSocketProjectHandler) DeleteProject(c *gin.Context) {
	projectID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project ID"})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	userUUID := userID.(uuid.UUID)
	sessionID := c.GetHeader("X-Session-ID")

	err = h.projectService.DeleteProject(c.Request.Context(), projectID, userUUID, getSessionIDPtr(sessionID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Project deleted successfully",
	})
}

// ArchiveProject archives a project with WebSocket notification
func (h *WebSocketProjectHandler) ArchiveProject(c *gin.Context) {
	projectID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project ID"})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	userUUID := userID.(uuid.UUID)
	sessionID := c.GetHeader("X-Session-ID")

	err = h.projectService.ArchiveProject(c.Request.Context(), projectID, userUUID, getSessionIDPtr(sessionID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Project archived successfully",
	})
}
