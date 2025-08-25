package handler

import (
	"context"
	"log"
	"net/http"

	"macwrite-auth-api/pkg/auth"
	"macwrite-auth-api/pkg/websocket"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// WebSocketHandler handles WebSocket connections
type WebSocketHandler struct {
	hub         *websocket.Hub
	jwtManager  *auth.JWTManager
	broadcaster *websocket.EventBroadcaster
}

// NewWebSocketHandler creates a new WebSocket handler
func NewWebSocketHandler(hub *websocket.Hub, jwtManager *auth.JWTManager) *WebSocketHandler {
	return &WebSocketHandler{
		hub:         hub,
		jwtManager:  jwtManager,
		broadcaster: websocket.NewEventBroadcaster(hub),
	}
}

// HandleWebSocketConnection handles WebSocket connection upgrades
func (h *WebSocketHandler) HandleWebSocketConnection(c *gin.Context) {
	// Get user ID from context (set by auth middleware)
	userIDInterface, exists := c.Get("user_id")
	if !exists {
		log.Println("WebSocket: User ID not found in context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	userID, ok := userIDInterface.(uuid.UUID)
	if !ok {
		log.Println("WebSocket: Invalid user ID format")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID"})
		return
	}

	// Upgrade HTTP connection to WebSocket
	conn, err := websocket.UpgradeConnection(c.Writer, c.Request)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to upgrade connection"})
		return
	}

	// Get client information
	userAgent := c.GetHeader("User-Agent")
	ipAddress := c.ClientIP()

	// Create new client
	client := websocket.NewClient(h.hub, conn, userID, userAgent, ipAddress)

	// Register client with hub
	h.hub.Register <- client

	// Start client goroutines
	ctx := context.Background()
	client.Start(ctx)

	log.Printf("WebSocket connection established for user %s from %s", userID.String(), ipAddress)
}

// HandleWebSocketAuth handles WebSocket authentication via query parameters
func (h *WebSocketHandler) HandleWebSocketAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Try to get token from query parameter first (for WebSocket upgrade)
		token := c.Query("token")

		// If not in query, try header
		if token == "" {
			authHeader := c.GetHeader("Authorization")
			if authHeader != "" && len(authHeader) > 7 && authHeader[:7] == "Bearer " {
				token = authHeader[7:]
			}
		}

		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication token required"})
			c.Abort()
			return
		}

		// Validate token
		claims, err := h.jwtManager.ValidateToken(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		// Set user information in context
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("email", claims.Email)

		c.Next()
	}
}

// GetConnectedUsers returns the list of connected users
func (h *WebSocketHandler) GetConnectedUsers(c *gin.Context) {
	users := h.hub.GetConnectedUsers()

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"connected_users": users,
			"count":           len(users),
		},
	})
}

// GetUserSessions returns active sessions for the current user
func (h *WebSocketHandler) GetUserSessions(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	userUUID := userID.(uuid.UUID)
	sessions := h.hub.GetUserSessions(userUUID)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"sessions": sessions,
			"count":    len(sessions),
		},
	})
}

// GetSystemStatus returns WebSocket system status
func (h *WebSocketHandler) GetSystemStatus(c *gin.Context) {
	status := h.hub.GetSystemStatus()

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    status,
	})
}

// GetActiveFocusSession returns the current user's active focus session
func (h *WebSocketHandler) GetActiveFocusSession(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	userUUID := userID.(uuid.UUID)
	focusManager := websocket.NewFocusManager(h.hub)
	session := focusManager.GetActiveFocusSession(userUUID)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"active_session": session,
		},
	})
}

// StartFocusSession starts a focus session via HTTP
func (h *WebSocketHandler) StartFocusSession(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	var req struct {
		TodoID      string `json:"todo_id" binding:"required"`
		TodoTitle   string `json:"todo_title"`
		SessionType string `json:"session_type"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	todoID, err := uuid.Parse(req.TodoID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid todo ID"})
		return
	}

	if req.SessionType == "" {
		req.SessionType = "focus"
	}

	userUUID := userID.(uuid.UUID)
	focusManager := websocket.NewFocusManager(h.hub)
	session, err := focusManager.StartFocusSession(userUUID, todoID, req.TodoTitle, "http-request", req.SessionType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"session": session,
			"message": "Focus session started successfully",
		},
	})
}

// EndFocusSession ends a focus session via HTTP
func (h *WebSocketHandler) EndFocusSession(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	userUUID := userID.(uuid.UUID)
	focusManager := websocket.NewFocusManager(h.hub)
	err := focusManager.EndFocusSession(userUUID, "manual_end_http")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"message": "Focus session ended successfully",
		},
	})
}

// BroadcastNotification sends a notification to all users (admin only)
func (h *WebSocketHandler) BroadcastNotification(c *gin.Context) {
	// This would typically require admin permissions
	var req struct {
		Title   string `json:"title" binding:"required"`
		Message string `json:"message" binding:"required"`
		Type    string `json:"type"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Type == "" {
		req.Type = "info"
	}

	ctx := context.Background()
	h.broadcaster.BroadcastSystemAlert(ctx, req.Title, req.Message, req.Type)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"message": "Notification broadcasted successfully",
		},
	})
}

// GetBroadcaster returns the event broadcaster for use by other handlers
func (h *WebSocketHandler) GetBroadcaster() *websocket.EventBroadcaster {
	return h.broadcaster
}

// GetHub returns the WebSocket hub for use by other handlers
func (h *WebSocketHandler) GetHub() *websocket.Hub {
	return h.hub
}
