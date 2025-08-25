package handler

import (
	"net/http"

	"macwrite-auth-api/internal/models"
	"macwrite-auth-api/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// WebSocketTodoHandler handles WebSocket-enabled todo operations
type WebSocketTodoHandler struct {
	todoService *service.WebSocketTodoService
	wsHandler   *WebSocketHandler
}

// NewWebSocketTodoHandler creates a new WebSocket todo handler
func NewWebSocketTodoHandler(todoService *service.WebSocketTodoService, wsHandler *WebSocketHandler) *WebSocketTodoHandler {
	return &WebSocketTodoHandler{
		todoService: todoService,
		wsHandler:   wsHandler,
	}
}

// CreateTodo creates a new todo with WebSocket notification
func (h *WebSocketTodoHandler) CreateTodo(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	var req models.CreateTodoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userUUID := userID.(uuid.UUID)
	sessionID := c.GetHeader("X-Session-ID") // Optional session ID to exclude from broadcast

	todo, err := h.todoService.CreateTodo(c.Request.Context(), userUUID, &req, getSessionIDPtr(sessionID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    todo,
	})
}

// UpdateTodo updates a todo with WebSocket notification
func (h *WebSocketTodoHandler) UpdateTodo(c *gin.Context) {
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

	var req models.UpdateTodoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userUUID := userID.(uuid.UUID)
	sessionID := c.GetHeader("X-Session-ID")

	todo, err := h.todoService.UpdateTodo(c.Request.Context(), todoID, userUUID, &req, getSessionIDPtr(sessionID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    todo,
	})
}

// DeleteTodo deletes a todo with WebSocket notification
func (h *WebSocketTodoHandler) DeleteTodo(c *gin.Context) {
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

	userUUID := userID.(uuid.UUID)
	sessionID := c.GetHeader("X-Session-ID")

	err = h.todoService.DeleteTodo(c.Request.Context(), todoID, userUUID, getSessionIDPtr(sessionID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Todo deleted successfully",
	})
}

// MarkTodoComplete marks a todo as complete with WebSocket notification
func (h *WebSocketTodoHandler) MarkTodoComplete(c *gin.Context) {
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

	userUUID := userID.(uuid.UUID)
	sessionID := c.GetHeader("X-Session-ID")

	err = h.todoService.MarkTodoComplete(c.Request.Context(), todoID, userUUID, getSessionIDPtr(sessionID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Todo marked as complete",
	})
}

// MarkTodoIncomplete marks a todo as incomplete with WebSocket notification
func (h *WebSocketTodoHandler) MarkTodoIncomplete(c *gin.Context) {
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

	userUUID := userID.(uuid.UUID)
	sessionID := c.GetHeader("X-Session-ID")

	err = h.todoService.MarkTodoIncomplete(c.Request.Context(), todoID, userUUID, getSessionIDPtr(sessionID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Todo marked as incomplete",
	})
}

// PinTodo pins/unpins a todo with WebSocket notification
func (h *WebSocketTodoHandler) PinTodo(c *gin.Context) {
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
		Pinned bool `json:"pinned"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userUUID := userID.(uuid.UUID)
	sessionID := c.GetHeader("X-Session-ID")

	err = h.todoService.PinTodo(c.Request.Context(), todoID, userUUID, req.Pinned, getSessionIDPtr(sessionID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	action := "unpinned"
	if req.Pinned {
		action = "pinned"
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Todo " + action + " successfully",
	})
}

// BulkUpdateTodos updates multiple todos with WebSocket notifications
func (h *WebSocketTodoHandler) BulkUpdateTodos(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	var req struct {
		Updates []struct {
			TodoID  string                   `json:"todo_id" binding:"required"`
			Updates models.UpdateTodoRequest `json:"updates" binding:"required"`
		} `json:"updates" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userUUID := userID.(uuid.UUID)
	sessionID := c.GetHeader("X-Session-ID")

	// Convert to service format
	var updates []struct {
		TodoID  uuid.UUID
		Updates *models.UpdateTodoRequest
	}

	for _, update := range req.Updates {
		todoID, err := uuid.Parse(update.TodoID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid todo ID: " + update.TodoID})
			return
		}
		updates = append(updates, struct {
			TodoID  uuid.UUID
			Updates *models.UpdateTodoRequest
		}{
			TodoID:  todoID,
			Updates: &update.Updates,
		})
	}

	err := h.todoService.BulkUpdateTodos(c.Request.Context(), userUUID, updates, getSessionIDPtr(sessionID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Todos updated successfully",
	})
}

// WebSocket Kanban Board Methods

// MoveTodoToColumn moves a todo to a different column with WebSocket notification
func (h *WebSocketTodoHandler) MoveTodoToColumn(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	var req models.MoveTodoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userUUID := userID.(uuid.UUID)
	sessionID := c.GetHeader("X-Session-ID")

	response, err := h.todoService.MoveTodoToColumn(c.Request.Context(), userUUID, &req, getSessionIDPtr(sessionID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Todo moved successfully",
		"data":    response,
	})
}

// ReorderTodosInColumn reorders todos within a column with WebSocket notification
func (h *WebSocketTodoHandler) ReorderTodosInColumn(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	var req models.ReorderTodosRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userUUID := userID.(uuid.UUID)
	sessionID := c.GetHeader("X-Session-ID")

	response, err := h.todoService.ReorderTodosInColumn(c.Request.Context(), userUUID, &req, getSessionIDPtr(sessionID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Todos reordered successfully",
		"data":    response,
	})
}

// BulkMoveTodos moves multiple todos with WebSocket notification
func (h *WebSocketTodoHandler) BulkMoveTodos(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	var req models.BulkMoveTodosRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userUUID := userID.(uuid.UUID)
	sessionID := c.GetHeader("X-Session-ID")

	response, err := h.todoService.BulkMoveTodos(c.Request.Context(), userUUID, &req, getSessionIDPtr(sessionID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Bulk move completed successfully",
		"data":    response,
	})
}

// Helper function to convert string to pointer
func getSessionIDPtr(sessionID string) *string {
	if sessionID == "" {
		return nil
	}
	return &sessionID
}
