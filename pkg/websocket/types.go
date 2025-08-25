package websocket

import (
	"macwrite-auth-api/pkg/redis"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// WebSocket event types
const (
	// Connection events
	EventTypeConnect    = "connect"
	EventTypeDisconnect = "disconnect"
	EventTypePong       = "pong"
	EventTypePing       = "ping"

	// Todo events
	EventTypeTodoCreated   = "todo_created"
	EventTypeTodoUpdated   = "todo_updated"
	EventTypeTodoDeleted   = "todo_deleted"
	EventTypeTodoCompleted = "todo_completed"
	EventTypeTodoPinned    = "todo_pinned"

	// Subtask events
	EventTypeSubtaskCreated   = "subtask_created"
	EventTypeSubtaskUpdated   = "subtask_updated"
	EventTypeSubtaskDeleted   = "subtask_deleted"
	EventTypeSubtaskCompleted = "subtask_completed"
	EventTypeSubtaskReordered = "subtask_reordered"

	// Project events
	EventTypeProjectCreated = "project_created"
	EventTypeProjectUpdated = "project_updated"
	EventTypeProjectDeleted = "project_deleted"

	// Focus session events
	EventTypeFocusSessionStarted = "focus_session_started"
	EventTypeFocusSessionEnded   = "focus_session_ended"
	EventTypeFocusSessionActive  = "focus_session_active"
	EventTypeFocusTimerUpdate    = "focus_timer_update"

	// User activity events
	EventTypeUserActivity      = "user_activity"
	EventTypeUserTyping        = "user_typing"
	EventTypeUserStoppedTyping = "user_stopped_typing"

	// System events
	EventTypeError        = "error"
	EventTypeSuccess      = "success"
	EventTypeNotification = "notification"
	EventTypeSystemStatus = "system_status"
)

// Message represents a WebSocket message
type Message struct {
	Type      string      `json:"type"`
	Payload   interface{} `json:"payload"`
	Timestamp time.Time   `json:"timestamp"`
	EventID   string      `json:"event_id"`
	UserID    *uuid.UUID  `json:"user_id,omitempty"`
	SessionID *string     `json:"session_id,omitempty"`
}

// Client represents a WebSocket client connection
type Client struct {
	ID           string          `json:"id"`
	UserID       uuid.UUID       `json:"user_id"`
	Connection   *websocket.Conn `json:"-"`
	Send         chan Message    `json:"-"`
	Hub          *Hub            `json:"-"`
	IsActive     bool            `json:"is_active"`
	ConnectedAt  time.Time       `json:"connected_at"`
	LastActivity time.Time       `json:"last_activity"`
	UserAgent    string          `json:"user_agent"`
	IPAddress    string          `json:"ip_address"`
	DeviceInfo   DeviceInfo      `json:"device_info"`
}

// DeviceInfo contains information about the client device
type DeviceInfo struct {
	Type      string `json:"type"`    // "desktop", "mobile", "tablet"
	OS        string `json:"os"`      // Operating system
	Browser   string `json:"browser"` // Browser name
	Version   string `json:"version"` // Browser version
	IsMobile  bool   `json:"is_mobile"`
	IsTablet  bool   `json:"is_tablet"`
	IsDesktop bool   `json:"is_desktop"`
}

// Hub maintains active clients and broadcasts messages
type Hub struct {
	// Registered clients by user ID
	Clients map[uuid.UUID]map[string]*Client `json:"clients"`

	// Broadcast messages to all clients of a user
	Broadcast chan BroadcastMessage `json:"-"`

	// Register new clients
	Register chan *Client `json:"-"`

	// Unregister clients
	Unregister chan *Client `json:"-"`

	// Focus sessions by user ID
	FocusSessions map[uuid.UUID]*FocusSession `json:"focus_sessions"`

	// User activity tracking
	UserActivity map[uuid.UUID]*UserActivity `json:"user_activity"`

	// Channel for focus timer updates
	FocusTimerUpdate chan FocusTimerMessage `json:"-"`

	// Redis client for pub/sub scaling
	redisClient redis.Client `json:"-"`

	// Mutex for thread-safe operations
	mutex *sync.RWMutex `json:"-"`

	// Server start time for uptime calculation
	startTime time.Time `json:"-"`
}

// BroadcastMessage represents a message to broadcast to specific users
type BroadcastMessage struct {
	UserIDs          []uuid.UUID `json:"user_ids"`
	Message          Message     `json:"message"`
	ExcludeSessionID *string     `json:"exclude_session_id,omitempty"`
}

// FocusSession represents an active focus session
type FocusSession struct {
	ID          string        `json:"id"`
	UserID      uuid.UUID     `json:"user_id"`
	TodoID      uuid.UUID     `json:"todo_id"`
	TodoTitle   string        `json:"todo_title"`
	StartTime   time.Time     `json:"start_time"`
	Duration    time.Duration `json:"duration"`
	IsActive    bool          `json:"is_active"`
	SessionType string        `json:"session_type"` // "focus", "break", "pomodoro"
	ClientID    string        `json:"client_id"`
	UpdatedAt   time.Time     `json:"updated_at"`
}

// FocusTimerMessage represents a focus timer update message
type FocusTimerMessage struct {
	SessionID string        `json:"session_id"`
	UserID    uuid.UUID     `json:"user_id"`
	Duration  time.Duration `json:"duration"`
	IsActive  bool          `json:"is_active"`
}

// UserActivity tracks user activity and presence
type UserActivity struct {
	UserID         uuid.UUID          `json:"user_id"`
	IsOnline       bool               `json:"is_online"`
	LastSeen       time.Time          `json:"last_seen"`
	ActiveSessions int                `json:"active_sessions"`
	CurrentFocus   *uuid.UUID         `json:"current_focus,omitempty"`
	TypingIn       map[uuid.UUID]bool `json:"typing_in,omitempty"` // todo_id -> is_typing
}

// Event payload structures

// TodoEventPayload represents todo-related events
type TodoEventPayload struct {
	Action    string                 `json:"action"`
	Todo      interface{}            `json:"todo"`
	ProjectID uuid.UUID              `json:"project_id"`
	Changes   map[string]interface{} `json:"changes,omitempty"`
}

// SubtaskEventPayload represents subtask-related events
type SubtaskEventPayload struct {
	Action  string                 `json:"action"`
	Subtask interface{}            `json:"subtask"`
	TodoID  uuid.UUID              `json:"todo_id"`
	Changes map[string]interface{} `json:"changes,omitempty"`
}

// ProjectEventPayload represents project-related events
type ProjectEventPayload struct {
	Action  string                 `json:"action"`
	Project interface{}            `json:"project"`
	Changes map[string]interface{} `json:"changes,omitempty"`
}

// FocusSessionEventPayload represents focus session events
type FocusSessionEventPayload struct {
	Action       string       `json:"action"`
	Session      FocusSession `json:"session"`
	Message      string       `json:"message,omitempty"`
	PreviousTask *uuid.UUID   `json:"previous_task,omitempty"`
}

// ErrorPayload represents error events
type ErrorPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// NotificationPayload represents notification events
type NotificationPayload struct {
	Title     string `json:"title"`
	Message   string `json:"message"`
	Type      string `json:"type"` // "info", "success", "warning", "error"
	Action    string `json:"action,omitempty"`
	ActionURL string `json:"action_url,omitempty"`
}

// UserActivityPayload represents user activity events
type UserActivityPayload struct {
	UserID   uuid.UUID          `json:"user_id"`
	Username string             `json:"username"`
	Activity string             `json:"activity"`
	TodoID   *uuid.UUID         `json:"todo_id,omitempty"`
	IsTyping bool               `json:"is_typing,omitempty"`
	TypingIn map[uuid.UUID]bool `json:"typing_in,omitempty"`
}

// ClientInfo represents client connection information
type ClientInfo struct {
	ID           string     `json:"id"`
	UserID       uuid.UUID  `json:"user_id"`
	ConnectedAt  time.Time  `json:"connected_at"`
	LastActivity time.Time  `json:"last_activity"`
	DeviceInfo   DeviceInfo `json:"device_info"`
	IsActive     bool       `json:"is_active"`
}

// SystemStatusPayload represents system status information
type SystemStatusPayload struct {
	ActiveConnections   int           `json:"active_connections"`
	ActiveUsers         int           `json:"active_users"`
	ActiveFocusSessions int           `json:"active_focus_sessions"`
	ServerTime          time.Time     `json:"server_time"`
	Uptime              time.Duration `json:"uptime"`
}
