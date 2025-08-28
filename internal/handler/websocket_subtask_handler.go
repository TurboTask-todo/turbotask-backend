package handler

import (
	"net/http"

	"quantumtask-auth-api/internal/models"
	"quantumtask-auth-api/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// WebSocketSubtaskHandler handles WebSocket-enabled subtask operations
type WebSocketSubtaskHandler struct {
	subtaskService *service.WebSocketSubtaskService
	wsHandler      *WebSocketHandler
}

// NewWebSocketSubtaskHandler creates a new WebSocket subtask handler
func NewWebSocketSubtaskHandler(subtaskService *service.WebSocketSubtaskService, wsHandler *WebSocketHandler) *WebSocketSubtaskHandler {
	return &WebSocketSubtaskHandler{
		subtaskService: subtaskService,
		wsHandler:      wsHandler,
	}
}

// CreateSubtask creates a new subtask with WebSocket notification
func (h *WebSocketSubtaskHandler) CreateSubtask(c *gin.Context) {
	todoID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid todo ID"})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	var req models.CreateSubtaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	sessionID := c.GetHeader("X-Session-ID")
	userUUID := userID.(uuid.UUID)

	subtask, err := h.subtaskService.CreateSubtask(c.Request.Context(), todoID, userUUID, &req, getSessionIDPtr(sessionID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    subtask,
	})
}

// UpdateSubtask updates a subtask with WebSocket notification
func (h *WebSocketSubtaskHandler) UpdateSubtask(c *gin.Context) {
	subtaskID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid subtask ID"})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	var req models.UpdateSubtaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	sessionID := c.GetHeader("X-Session-ID")
	userUUID := userID.(uuid.UUID)

	subtask, err := h.subtaskService.UpdateSubtask(c.Request.Context(), subtaskID, userUUID, &req, getSessionIDPtr(sessionID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    subtask,
	})
}

// DeleteSubtask deletes a subtask with WebSocket notification
func (h *WebSocketSubtaskHandler) DeleteSubtask(c *gin.Context) {
	subtaskID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid subtask ID"})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	sessionID := c.GetHeader("X-Session-ID")
	userUUID := userID.(uuid.UUID)

	err = h.subtaskService.DeleteSubtask(c.Request.Context(), subtaskID, userUUID, getSessionIDPtr(sessionID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Subtask deleted successfully",
	})
}

// ReorderSubtasks reorders subtasks with WebSocket notification
func (h *WebSocketSubtaskHandler) ReorderSubtasks(c *gin.Context) {
	todoID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid todo ID"})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	var req struct {
		SubtaskIDs []string `json:"subtask_ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Convert string IDs to UUIDs
	var subtaskIDs []uuid.UUID
	for _, id := range req.SubtaskIDs {
		subtaskID, err := uuid.Parse(id)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid subtask ID: " + id})
			return
		}
		subtaskIDs = append(subtaskIDs, subtaskID)
	}

	sessionID := c.GetHeader("X-Session-ID")
	userUUID := userID.(uuid.UUID)

	err = h.subtaskService.ReorderSubtasks(c.Request.Context(), todoID, userUUID, subtaskIDs, getSessionIDPtr(sessionID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Subtasks reordered successfully",
	})
}
