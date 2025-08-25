package websocket

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer
	maxMessageSize = 1024 * 1024 // 1MB
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// In production, implement proper origin checking
		return true
	},
}

// UpgradeConnection upgrades an HTTP connection to WebSocket
func UpgradeConnection(w http.ResponseWriter, r *http.Request) (*websocket.Conn, error) {
	return upgrader.Upgrade(w, r, nil)
}

// NewClient creates a new WebSocket client
func NewClient(hub *Hub, conn *websocket.Conn, userID uuid.UUID, userAgent, ipAddress string) *Client {
	client := &Client{
		ID:           uuid.New().String(),
		UserID:       userID,
		Connection:   conn,
		Send:         make(chan Message, 256),
		Hub:          hub,
		IsActive:     true,
		ConnectedAt:  time.Now(),
		LastActivity: time.Now(),
		UserAgent:    userAgent,
		IPAddress:    ipAddress,
		DeviceInfo:   parseDeviceInfo(userAgent),
	}

	return client
}

// parseDeviceInfo extracts device information from user agent
func parseDeviceInfo(userAgent string) DeviceInfo {
	info := DeviceInfo{
		Type:      "unknown",
		OS:        "unknown",
		Browser:   "unknown",
		Version:   "unknown",
		IsMobile:  false,
		IsTablet:  false,
		IsDesktop: false,
	}

	ua := strings.ToLower(userAgent)

	// Detect mobile devices
	if strings.Contains(ua, "mobile") || strings.Contains(ua, "android") || strings.Contains(ua, "iphone") {
		info.IsMobile = true
		info.Type = "mobile"
	} else if strings.Contains(ua, "tablet") || strings.Contains(ua, "ipad") {
		info.IsTablet = true
		info.Type = "tablet"
	} else {
		info.IsDesktop = true
		info.Type = "desktop"
	}

	// Detect OS
	if strings.Contains(ua, "windows") {
		info.OS = "Windows"
	} else if strings.Contains(ua, "mac") {
		info.OS = "macOS"
	} else if strings.Contains(ua, "linux") {
		info.OS = "Linux"
	} else if strings.Contains(ua, "android") {
		info.OS = "Android"
	} else if strings.Contains(ua, "ios") || strings.Contains(ua, "iphone") || strings.Contains(ua, "ipad") {
		info.OS = "iOS"
	}

	// Detect browser
	if strings.Contains(ua, "chrome") {
		info.Browser = "Chrome"
	} else if strings.Contains(ua, "firefox") {
		info.Browser = "Firefox"
	} else if strings.Contains(ua, "safari") {
		info.Browser = "Safari"
	} else if strings.Contains(ua, "edge") {
		info.Browser = "Edge"
	}

	return info
}

// readPump handles reading messages from the WebSocket connection
func (c *Client) readPump(ctx context.Context) {
	defer func() {
		c.Hub.Unregister <- c
		c.Connection.Close()
	}()

	c.Connection.SetReadLimit(maxMessageSize)
	c.Connection.SetReadDeadline(time.Now().Add(pongWait))
	c.Connection.SetPongHandler(func(string) error {
		c.Connection.SetReadDeadline(time.Now().Add(pongWait))
		c.LastActivity = time.Now()
		return nil
	})

	for {
		select {
		case <-ctx.Done():
			return
		default:
			_, messageBytes, err := c.Connection.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Printf("WebSocket error for client %s: %v", c.ID, err)
				}
				return
			}

			c.LastActivity = time.Now()

			// Parse the incoming message
			var incomingMsg Message
			if err := json.Unmarshal(messageBytes, &incomingMsg); err != nil {
				log.Printf("Failed to parse message from client %s: %v", c.ID, err)
				c.sendError("INVALID_MESSAGE", "Failed to parse message", "")
				continue
			}

			// Handle the message
			c.handleMessage(incomingMsg)
		}
	}
}

// writePump handles writing messages to the WebSocket connection
func (c *Client) writePump(ctx context.Context) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Connection.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Connection.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.Connection.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.Connection.WriteJSON(message); err != nil {
				log.Printf("Failed to write message to client %s: %v", c.ID, err)
				return
			}

		case <-ticker.C:
			c.Connection.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Connection.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}

		case <-ctx.Done():
			return
		}
	}
}

// handleMessage handles incoming messages from the client
func (c *Client) handleMessage(msg Message) {
	log.Printf("Received message from client %s: type=%s", c.ID, msg.Type)

	switch msg.Type {
	case EventTypePing:
		c.handlePing()
	case EventTypeFocusSessionStarted:
		c.handleStartFocusSession(msg)
	case EventTypeFocusSessionEnded:
		c.handleEndFocusSession(msg)
	case "switch_focus":
		c.handleSwitchFocus(msg)
	case "get_focus_status":
		c.handleGetFocusStatus()
	case "user_typing":
		c.handleUserTyping(msg)
	case "user_stopped_typing":
		c.handleUserStoppedTyping(msg)
	case "get_system_status":
		c.handleGetSystemStatus()
	default:
		log.Printf("Unknown message type from client %s: %s", c.ID, msg.Type)
		c.sendError("UNKNOWN_MESSAGE_TYPE", "Unknown message type", msg.Type)
	}
}

// handlePing handles ping messages
func (c *Client) handlePing() {
	response := Message{
		Type:      EventTypePong,
		Payload:   map[string]interface{}{"message": "pong"},
		Timestamp: time.Now(),
		EventID:   uuid.New().String(),
	}

	select {
	case c.Send <- response:
	default:
		log.Printf("Send channel full for client %s", c.ID)
	}
}

// handleStartFocusSession handles focus session start requests
func (c *Client) handleStartFocusSession(msg Message) {
	payload, ok := msg.Payload.(map[string]interface{})
	if !ok {
		c.sendError("INVALID_PAYLOAD", "Invalid payload for focus session start", "")
		return
	}

	todoIDStr, ok := payload["todo_id"].(string)
	if !ok {
		c.sendError("MISSING_TODO_ID", "Todo ID is required", "")
		return
	}

	todoID, err := uuid.Parse(todoIDStr)
	if err != nil {
		c.sendError("INVALID_TODO_ID", "Invalid Todo ID format", "")
		return
	}

	todoTitle, _ := payload["todo_title"].(string)
	sessionType, _ := payload["session_type"].(string)
	if sessionType == "" {
		sessionType = "focus"
	}

	focusManager := NewFocusManager(c.Hub)
	session, err := focusManager.StartFocusSession(c.UserID, todoID, todoTitle, c.ID, sessionType)
	if err != nil {
		c.sendError("FOCUS_SESSION_ERROR", "Failed to start focus session", err.Error())
		return
	}

	// Send success response
	response := Message{
		Type: EventTypeSuccess,
		Payload: map[string]interface{}{
			"message": "Focus session started successfully",
			"session": session,
		},
		Timestamp: time.Now(),
		EventID:   uuid.New().String(),
	}

	select {
	case c.Send <- response:
	default:
		log.Printf("Send channel full for client %s", c.ID)
	}
}

// handleEndFocusSession handles focus session end requests
func (c *Client) handleEndFocusSession(msg Message) {
	focusManager := NewFocusManager(c.Hub)
	err := focusManager.EndFocusSession(c.UserID, "manual_end")
	if err != nil {
		c.sendError("FOCUS_SESSION_ERROR", "Failed to end focus session", err.Error())
		return
	}

	// Send success response
	response := Message{
		Type: EventTypeSuccess,
		Payload: map[string]interface{}{
			"message": "Focus session ended successfully",
		},
		Timestamp: time.Now(),
		EventID:   uuid.New().String(),
	}

	select {
	case c.Send <- response:
	default:
		log.Printf("Send channel full for client %s", c.ID)
	}
}

// handleSwitchFocus handles focus task switching
func (c *Client) handleSwitchFocus(msg Message) {
	payload, ok := msg.Payload.(map[string]interface{})
	if !ok {
		c.sendError("INVALID_PAYLOAD", "Invalid payload for focus switch", "")
		return
	}

	todoIDStr, ok := payload["todo_id"].(string)
	if !ok {
		c.sendError("MISSING_TODO_ID", "Todo ID is required", "")
		return
	}

	todoID, err := uuid.Parse(todoIDStr)
	if err != nil {
		c.sendError("INVALID_TODO_ID", "Invalid Todo ID format", "")
		return
	}

	todoTitle, _ := payload["todo_title"].(string)

	focusManager := NewFocusManager(c.Hub)
	session, err := focusManager.SwitchFocusTask(c.UserID, todoID, todoTitle, c.ID)
	if err != nil {
		c.sendError("FOCUS_SWITCH_ERROR", "Failed to switch focus", err.Error())
		return
	}

	// Send success response
	response := Message{
		Type: EventTypeSuccess,
		Payload: map[string]interface{}{
			"message": "Focus switched successfully",
			"session": session,
		},
		Timestamp: time.Now(),
		EventID:   uuid.New().String(),
	}

	select {
	case c.Send <- response:
	default:
		log.Printf("Send channel full for client %s", c.ID)
	}
}

// handleGetFocusStatus handles focus status requests
func (c *Client) handleGetFocusStatus() {
	focusManager := NewFocusManager(c.Hub)
	session := focusManager.GetActiveFocusSession(c.UserID)

	response := Message{
		Type: "focus_status",
		Payload: map[string]interface{}{
			"active_session": session,
		},
		Timestamp: time.Now(),
		EventID:   uuid.New().String(),
	}

	select {
	case c.Send <- response:
	default:
		log.Printf("Send channel full for client %s", c.ID)
	}
}

// handleUserTyping handles user typing notifications
func (c *Client) handleUserTyping(msg Message) {
	payload, ok := msg.Payload.(map[string]interface{})
	if !ok {
		return
	}

	todoIDStr, ok := payload["todo_id"].(string)
	if !ok {
		return
	}

	todoID, err := uuid.Parse(todoIDStr)
	if err != nil {
		return
	}

	// Update user activity
	c.Hub.mutex.Lock()
	if activity, exists := c.Hub.UserActivity[c.UserID]; exists {
		if activity.TypingIn == nil {
			activity.TypingIn = make(map[uuid.UUID]bool)
		}
		activity.TypingIn[todoID] = true
	}
	c.Hub.mutex.Unlock()

	// Broadcast typing status
	c.Hub.broadcastUserActivity(c.UserID, "user_typing", &todoID)
}

// handleUserStoppedTyping handles user stopped typing notifications
func (c *Client) handleUserStoppedTyping(msg Message) {
	payload, ok := msg.Payload.(map[string]interface{})
	if !ok {
		return
	}

	todoIDStr, ok := payload["todo_id"].(string)
	if !ok {
		return
	}

	todoID, err := uuid.Parse(todoIDStr)
	if err != nil {
		return
	}

	// Update user activity
	c.Hub.mutex.Lock()
	if activity, exists := c.Hub.UserActivity[c.UserID]; exists {
		if activity.TypingIn != nil {
			activity.TypingIn[todoID] = false
		}
	}
	c.Hub.mutex.Unlock()

	// Broadcast typing status
	c.Hub.broadcastUserActivity(c.UserID, "user_stopped_typing", &todoID)
}

// handleGetSystemStatus handles system status requests
func (c *Client) handleGetSystemStatus() {
	status := c.Hub.GetSystemStatus()

	response := Message{
		Type:      EventTypeSystemStatus,
		Payload:   status,
		Timestamp: time.Now(),
		EventID:   uuid.New().String(),
	}

	select {
	case c.Send <- response:
	default:
		log.Printf("Send channel full for client %s", c.ID)
	}
}

// sendError sends an error message to the client
func (c *Client) sendError(code, message, details string) {
	errorMsg := Message{
		Type: EventTypeError,
		Payload: ErrorPayload{
			Code:    code,
			Message: message,
			Details: details,
		},
		Timestamp: time.Now(),
		EventID:   uuid.New().String(),
	}

	select {
	case c.Send <- errorMsg:
	default:
		log.Printf("Send channel full for client %s", c.ID)
	}
}

// Start starts the client's read and write pumps
func (c *Client) Start(ctx context.Context) {
	go c.writePump(ctx)
	go c.readPump(ctx)
}
