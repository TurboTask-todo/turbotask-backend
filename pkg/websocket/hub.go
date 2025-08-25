package websocket

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	"macwrite-auth-api/pkg/redis"

	"github.com/google/uuid"
)

// NewHub creates a new WebSocket hub
func NewHub(redisClient redis.Client) *Hub {
	return &Hub{
		Clients:          make(map[uuid.UUID]map[string]*Client),
		Broadcast:        make(chan BroadcastMessage, 1000),
		Register:         make(chan *Client, 100),
		Unregister:       make(chan *Client, 100),
		FocusSessions:    make(map[uuid.UUID]*FocusSession),
		UserActivity:     make(map[uuid.UUID]*UserActivity),
		FocusTimerUpdate: make(chan FocusTimerMessage, 100),
		redisClient:      redisClient,
		mutex:            &sync.RWMutex{},
	}
}

// Hub maintains the set of active clients and broadcasts messages to the clients
// Note: The Hub struct is defined in types.go, this is just a reference

// Run starts the hub and handles all WebSocket operations
func (h *Hub) Run(ctx context.Context) {
	h.startTime = time.Now()

	// Start focus timer routine
	go h.runFocusTimer(ctx)

	// Start Redis pub/sub listener for scaling
	go h.listenToRedis(ctx)

	// Start periodic cleanup
	go h.periodicCleanup(ctx)

	for {
		select {
		case client := <-h.Register:
			h.registerClient(client)

		case client := <-h.Unregister:
			h.unregisterClient(client)

		case message := <-h.Broadcast:
			h.broadcastMessage(message)

		case timerMsg := <-h.FocusTimerUpdate:
			h.handleFocusTimerUpdate(timerMsg)

		case <-ctx.Done():
			log.Println("WebSocket hub shutting down...")
			return
		}
	}
}

// registerClient registers a new client
func (h *Hub) registerClient(client *Client) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	if h.Clients[client.UserID] == nil {
		h.Clients[client.UserID] = make(map[string]*Client)
	}
	h.Clients[client.UserID][client.ID] = client

	// Update user activity
	if h.UserActivity[client.UserID] == nil {
		h.UserActivity[client.UserID] = &UserActivity{
			UserID:   client.UserID,
			TypingIn: make(map[uuid.UUID]bool),
		}
	}

	activity := h.UserActivity[client.UserID]
	activity.IsOnline = true
	activity.LastSeen = time.Now()
	activity.ActiveSessions = len(h.Clients[client.UserID])

	log.Printf("Client registered: %s for user %s (total sessions: %d)",
		client.ID, client.UserID.String(), activity.ActiveSessions)

	// Send welcome message
	welcomeMsg := Message{
		Type: EventTypeConnect,
		Payload: map[string]interface{}{
			"message":     "Connected to WebSocket server",
			"client_id":   client.ID,
			"user_id":     client.UserID,
			"server_time": time.Now(),
		},
		Timestamp: time.Now(),
		EventID:   uuid.New().String(),
	}

	select {
	case client.Send <- welcomeMsg:
	default:
		h.unregisterClientUnsafe(client)
	}

	// Broadcast user activity update to other sessions
	h.broadcastUserActivity(client.UserID, "user_connected", nil)

	// Publish to Redis for scaling
	h.publishToRedis("user_connected", map[string]interface{}{
		"user_id":     client.UserID,
		"client_id":   client.ID,
		"device_info": client.DeviceInfo,
	})
}

// unregisterClient unregisters a client
func (h *Hub) unregisterClient(client *Client) {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	h.unregisterClientUnsafe(client)
}

// unregisterClientUnsafe unregisters a client without locking (internal use)
func (h *Hub) unregisterClientUnsafe(client *Client) {
	if userClients, exists := h.Clients[client.UserID]; exists {
		if _, exists := userClients[client.ID]; exists {
			delete(userClients, client.ID)
			close(client.Send)

			// Clean up empty user map
			if len(userClients) == 0 {
				delete(h.Clients, client.UserID)

				// Update user activity to offline
				if activity, exists := h.UserActivity[client.UserID]; exists {
					activity.IsOnline = false
					activity.LastSeen = time.Now()
					activity.ActiveSessions = 0
				}
			} else {
				// Update active session count
				if activity, exists := h.UserActivity[client.UserID]; exists {
					activity.ActiveSessions = len(userClients)
				}
			}

			log.Printf("Client unregistered: %s for user %s", client.ID, client.UserID.String())

			// End any active focus session for this client
			h.endFocusSessionForClient(client.ID)

			// Broadcast user activity update
			h.broadcastUserActivity(client.UserID, "user_disconnected", nil)

			// Publish to Redis for scaling
			h.publishToRedis("user_disconnected", map[string]interface{}{
				"user_id":   client.UserID,
				"client_id": client.ID,
			})
		}
	}
}

// broadcastMessage broadcasts a message to specific users
func (h *Hub) broadcastMessage(broadcast BroadcastMessage) {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	for _, userID := range broadcast.UserIDs {
		if userClients, exists := h.Clients[userID]; exists {
			for clientID, client := range userClients {
				// Skip excluded session if specified
				if broadcast.ExcludeSessionID != nil && clientID == *broadcast.ExcludeSessionID {
					continue
				}

				select {
				case client.Send <- broadcast.Message:
				default:
					// Client's send channel is full, close it
					go func(c *Client) {
						h.Unregister <- c
					}(client)
				}
			}
		}
	}
}

// BroadcastToUser broadcasts a message to all sessions of a specific user
func (h *Hub) BroadcastToUser(userID uuid.UUID, message Message, excludeSessionID *string) {
	broadcast := BroadcastMessage{
		UserIDs:          []uuid.UUID{userID},
		Message:          message,
		ExcludeSessionID: excludeSessionID,
	}

	select {
	case h.Broadcast <- broadcast:
	default:
		log.Printf("Broadcast channel full, dropping message for user %s", userID.String())
	}
}

// BroadcastToUsers broadcasts a message to multiple users
func (h *Hub) BroadcastToUsers(userIDs []uuid.UUID, message Message) {
	broadcast := BroadcastMessage{
		UserIDs: userIDs,
		Message: message,
	}

	select {
	case h.Broadcast <- broadcast:
	default:
		log.Println("Broadcast channel full, dropping message")
	}
}

// GetConnectedUsers returns a list of currently connected users
func (h *Hub) GetConnectedUsers() []uuid.UUID {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	users := make([]uuid.UUID, 0, len(h.Clients))
	for userID := range h.Clients {
		users = append(users, userID)
	}
	return users
}

// GetUserSessions returns all active sessions for a user
func (h *Hub) GetUserSessions(userID uuid.UUID) []*ClientInfo {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	var sessions []*ClientInfo
	if userClients, exists := h.Clients[userID]; exists {
		for _, client := range userClients {
			sessions = append(sessions, &ClientInfo{
				ID:           client.ID,
				UserID:       client.UserID,
				ConnectedAt:  client.ConnectedAt,
				LastActivity: client.LastActivity,
				DeviceInfo:   client.DeviceInfo,
				IsActive:     client.IsActive,
			})
		}
	}
	return sessions
}

// GetSystemStatus returns current system status
func (h *Hub) GetSystemStatus() SystemStatusPayload {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	totalConnections := 0
	activeFocusSessions := 0

	for _, userClients := range h.Clients {
		totalConnections += len(userClients)
	}

	for _, session := range h.FocusSessions {
		if session.IsActive {
			activeFocusSessions++
		}
	}

	return SystemStatusPayload{
		ActiveConnections:   totalConnections,
		ActiveUsers:         len(h.Clients),
		ActiveFocusSessions: activeFocusSessions,
		ServerTime:          time.Now(),
		Uptime:              time.Since(h.startTime),
	}
}

// broadcastUserActivity broadcasts user activity updates
func (h *Hub) broadcastUserActivity(userID uuid.UUID, activity string, todoID *uuid.UUID) {
	activity_data := h.UserActivity[userID]
	if activity_data == nil {
		return
	}

	payload := UserActivityPayload{
		UserID:   userID,
		Activity: activity,
		TodoID:   todoID,
		TypingIn: activity_data.TypingIn,
	}

	message := Message{
		Type:      EventTypeUserActivity,
		Payload:   payload,
		Timestamp: time.Now(),
		EventID:   uuid.New().String(),
		UserID:    &userID,
	}

	// Broadcast to all users (they can filter on client side)
	h.BroadcastToUsers(h.GetConnectedUsers(), message)
}

// publishToRedis publishes events to Redis for horizontal scaling
func (h *Hub) publishToRedis(eventType string, data interface{}) {
	if h.redisClient == nil {
		return
	}

	payload := map[string]interface{}{
		"event_type": eventType,
		"data":       data,
		"timestamp":  time.Now(),
		"server_id":  "websocket-server", // Could be configurable
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Failed to marshal Redis payload: %v", err)
		return
	}

	// Note: Redis Publish method would need to be implemented in the redis.Client interface
	// For now, we'll skip Redis publishing until the interface is extended
	log.Printf("Would publish to Redis: %s", string(payloadBytes))
}

// listenToRedis listens for events from other server instances
func (h *Hub) listenToRedis(ctx context.Context) {
	if h.redisClient == nil {
		return
	}

	// Note: Redis Subscribe method would need to be implemented in the redis.Client interface
	// For now, we'll use a placeholder implementation
	log.Println("Redis pub/sub listener would start here")

	// Placeholder for Redis subscription logic
	// When Redis interface is extended, implement:
	// pubsub := h.redisClient.Subscribe(ctx, "websocket:events")
	// ... rest of the subscription logic
}

// periodicCleanup runs periodic cleanup tasks
func (h *Hub) periodicCleanup(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			h.cleanupInactiveClients()
			h.cleanupExpiredFocusSessions()
		case <-ctx.Done():
			return
		}
	}
}

// cleanupInactiveClients removes inactive clients
func (h *Hub) cleanupInactiveClients() {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	threshold := time.Now().Add(-10 * time.Minute)

	for _, userClients := range h.Clients {
		for clientID, client := range userClients {
			if client.LastActivity.Before(threshold) {
				log.Printf("Cleaning up inactive client: %s", clientID)
				h.unregisterClientUnsafe(client)
			}
		}
	}
}

// cleanupExpiredFocusSessions removes expired focus sessions
func (h *Hub) cleanupExpiredFocusSessions() {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	threshold := time.Now().Add(-2 * time.Hour)

	for sessionUserID, session := range h.FocusSessions {
		if !session.IsActive && session.UpdatedAt.Before(threshold) {
			log.Printf("Cleaning up expired focus session: %s", session.ID)
			delete(h.FocusSessions, sessionUserID)
		}
	}
}
