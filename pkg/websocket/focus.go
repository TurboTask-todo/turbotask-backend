package websocket

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
)

// FocusManager handles all focus session operations
type FocusManager struct {
	hub         *Hub
	mutex       sync.RWMutex
	timerTicker *time.Ticker
	stopTimer   chan bool
}

// NewFocusManager creates a new focus manager
func NewFocusManager(hub *Hub) *FocusManager {
	return &FocusManager{
		hub:       hub,
		stopTimer: make(chan bool),
	}
}

// StartFocusSession starts a new focus session for a user
func (fm *FocusManager) StartFocusSession(userID uuid.UUID, todoID uuid.UUID, todoTitle string, clientID string, sessionType string) (*FocusSession, error) {
	fm.mutex.Lock()
	defer fm.mutex.Unlock()

	// End any existing focus session for this user
	if existingSession, exists := fm.hub.FocusSessions[userID]; exists && existingSession.IsActive {
		fm.endFocusSessionUnsafe(userID, "new_session_started")
	}

	// Create new focus session
	session := &FocusSession{
		ID:          uuid.New().String(),
		UserID:      userID,
		TodoID:      todoID,
		TodoTitle:   todoTitle,
		StartTime:   time.Now(),
		Duration:    0,
		IsActive:    true,
		SessionType: sessionType,
		ClientID:    clientID,
		UpdatedAt:   time.Now(),
	}

	fm.hub.FocusSessions[userID] = session

	// Update user activity
	if activity, exists := fm.hub.UserActivity[userID]; exists {
		activity.CurrentFocus = &todoID
	}

	// Broadcast focus session started event
	payload := FocusSessionEventPayload{
		Action:  "started",
		Session: *session,
		Message: "Focus session started",
	}

	message := Message{
		Type:      EventTypeFocusSessionStarted,
		Payload:   payload,
		Timestamp: time.Now(),
		EventID:   uuid.New().String(),
		UserID:    &userID,
	}

	// Broadcast to all user sessions
	fm.hub.BroadcastToUser(userID, message, nil)

	// Publish to Redis for cross-server sync
	fm.hub.publishToRedis("focus_session_started", map[string]interface{}{
		"user_id":      userID,
		"todo_id":      todoID,
		"session_id":   session.ID,
		"session_type": sessionType,
		"client_id":    clientID,
	})

	log.Printf("Focus session started: %s for user %s on todo %s", session.ID, userID.String(), todoID.String())

	return session, nil
}

// EndFocusSession ends a focus session for a user
func (fm *FocusManager) EndFocusSession(userID uuid.UUID, reason string) error {
	fm.mutex.Lock()
	defer fm.mutex.Unlock()

	return fm.endFocusSessionUnsafe(userID, reason)
}

// endFocusSessionUnsafe ends a focus session without locking (internal use)
func (fm *FocusManager) endFocusSessionUnsafe(userID uuid.UUID, reason string) error {
	session, exists := fm.hub.FocusSessions[userID]
	if !exists || !session.IsActive {
		return nil // No active session to end
	}

	// Calculate final duration
	session.Duration = time.Since(session.StartTime)
	session.IsActive = false
	session.UpdatedAt = time.Now()

	// Update user activity
	if activity, exists := fm.hub.UserActivity[userID]; exists {
		activity.CurrentFocus = nil
	}

	// Broadcast focus session ended event
	payload := FocusSessionEventPayload{
		Action:  "ended",
		Session: *session,
		Message: "Focus session ended: " + reason,
	}

	message := Message{
		Type:      EventTypeFocusSessionEnded,
		Payload:   payload,
		Timestamp: time.Now(),
		EventID:   uuid.New().String(),
		UserID:    &userID,
	}

	// Broadcast to all user sessions
	fm.hub.BroadcastToUser(userID, message, nil)

	// Publish to Redis for cross-server sync
	fm.hub.publishToRedis("focus_session_ended", map[string]interface{}{
		"user_id":    userID,
		"session_id": session.ID,
		"duration":   session.Duration.Seconds(),
		"reason":     reason,
	})

	log.Printf("Focus session ended: %s for user %s (duration: %v, reason: %s)",
		session.ID, userID.String(), session.Duration, reason)

	return nil
}

// GetActiveFocusSession returns the active focus session for a user
func (fm *FocusManager) GetActiveFocusSession(userID uuid.UUID) *FocusSession {
	fm.mutex.RLock()
	defer fm.mutex.RUnlock()

	if session, exists := fm.hub.FocusSessions[userID]; exists && session.IsActive {
		// Update duration
		session.Duration = time.Since(session.StartTime)
		return session
	}
	return nil
}

// GetAllActiveFocusSessions returns all active focus sessions
func (fm *FocusManager) GetAllActiveFocusSessions() map[uuid.UUID]*FocusSession {
	fm.mutex.RLock()
	defer fm.mutex.RUnlock()

	activeSessions := make(map[uuid.UUID]*FocusSession)
	for userID, session := range fm.hub.FocusSessions {
		if session.IsActive {
			// Update duration
			session.Duration = time.Since(session.StartTime)
			activeSessions[userID] = session
		}
	}
	return activeSessions
}

// runFocusTimer runs the focus timer updates
func (h *Hub) runFocusTimer(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute) // Update every minute
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			h.sendFocusTimerUpdates()
		case timerMsg := <-h.FocusTimerUpdate:
			h.handleFocusTimerUpdate(timerMsg)
		case <-ctx.Done():
			return
		}
	}
}

// sendFocusTimerUpdates sends timer updates for all active focus sessions
func (h *Hub) sendFocusTimerUpdates() {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	for userID, session := range h.FocusSessions {
		if !session.IsActive {
			continue
		}

		// Update duration
		session.Duration = time.Since(session.StartTime)
		session.UpdatedAt = time.Now()

		// Send timer update
		payload := FocusSessionEventPayload{
			Action:  "timer_update",
			Session: *session,
		}

		message := Message{
			Type:      EventTypeFocusTimerUpdate,
			Payload:   payload,
			Timestamp: time.Now(),
			EventID:   uuid.New().String(),
			UserID:    &userID,
		}

		h.BroadcastToUser(userID, message, nil)
	}
}

// handleFocusTimerUpdate handles focus timer update messages
func (h *Hub) handleFocusTimerUpdate(timerMsg FocusTimerMessage) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	if session, exists := h.FocusSessions[timerMsg.UserID]; exists {
		session.Duration = timerMsg.Duration
		session.IsActive = timerMsg.IsActive
		session.UpdatedAt = time.Now()

		if !timerMsg.IsActive {
			// Update user activity
			if activity, exists := h.UserActivity[timerMsg.UserID]; exists {
				activity.CurrentFocus = nil
			}
		}
	}
}

// endFocusSessionForClient ends focus session for a specific client
func (h *Hub) endFocusSessionForClient(clientID string) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	for userID, session := range h.FocusSessions {
		if session.ClientID == clientID && session.IsActive {
			session.IsActive = false
			session.Duration = time.Since(session.StartTime)
			session.UpdatedAt = time.Now()

			// Update user activity
			if activity, exists := h.UserActivity[userID]; exists {
				activity.CurrentFocus = nil
			}

			// Broadcast session ended
			payload := FocusSessionEventPayload{
				Action:  "ended",
				Session: *session,
				Message: "Focus session ended: client disconnected",
			}

			message := Message{
				Type:      EventTypeFocusSessionEnded,
				Payload:   payload,
				Timestamp: time.Now(),
				EventID:   uuid.New().String(),
				UserID:    &userID,
			}

			h.BroadcastToUser(userID, message, &clientID)

			log.Printf("Focus session ended due to client disconnect: %s", session.ID)
			break
		}
	}
}

// PauseFocusSession pauses an active focus session
func (fm *FocusManager) PauseFocusSession(userID uuid.UUID) error {
	fm.mutex.Lock()
	defer fm.mutex.Unlock()

	session, exists := fm.hub.FocusSessions[userID]
	if !exists || !session.IsActive {
		return nil
	}

	// For now, we'll end the session on pause
	// In a more advanced implementation, you could track pause/resume states
	return fm.endFocusSessionUnsafe(userID, "paused")
}

// SwitchFocusTask switches focus to a different todo
func (fm *FocusManager) SwitchFocusTask(userID uuid.UUID, newTodoID uuid.UUID, newTodoTitle string, clientID string) (*FocusSession, error) {
	fm.mutex.Lock()
	defer fm.mutex.Unlock()

	var previousTaskID *uuid.UUID

	// End existing session if active
	if existingSession, exists := fm.hub.FocusSessions[userID]; exists && existingSession.IsActive {
		previousTaskID = &existingSession.TodoID
		fm.endFocusSessionUnsafe(userID, "switched_task")
	}

	// Start new session
	session := &FocusSession{
		ID:          uuid.New().String(),
		UserID:      userID,
		TodoID:      newTodoID,
		TodoTitle:   newTodoTitle,
		StartTime:   time.Now(),
		Duration:    0,
		IsActive:    true,
		SessionType: "focus",
		ClientID:    clientID,
		UpdatedAt:   time.Now(),
	}

	fm.hub.FocusSessions[userID] = session

	// Update user activity
	if activity, exists := fm.hub.UserActivity[userID]; exists {
		activity.CurrentFocus = &newTodoID
	}

	// Broadcast focus session started event with previous task info
	payload := FocusSessionEventPayload{
		Action:       "switched",
		Session:      *session,
		Message:      "Focus switched to new task",
		PreviousTask: previousTaskID,
	}

	message := Message{
		Type:      EventTypeFocusSessionStarted,
		Payload:   payload,
		Timestamp: time.Now(),
		EventID:   uuid.New().String(),
		UserID:    &userID,
	}

	fm.hub.BroadcastToUser(userID, message, nil)

	log.Printf("Focus session switched: %s for user %s from todo %v to todo %s",
		session.ID, userID.String(), previousTaskID, newTodoID.String())

	return session, nil
}

// GetFocusSessionStats returns statistics for a user's focus sessions
func (fm *FocusManager) GetFocusSessionStats(userID uuid.UUID) map[string]interface{} {
	fm.mutex.RLock()
	defer fm.mutex.RUnlock()

	stats := map[string]interface{}{
		"current_session": nil,
		"total_sessions":  0,
		"total_time":      0,
		"today_time":      0,
		"week_time":       0,
	}

	// Get current session
	if session, exists := fm.hub.FocusSessions[userID]; exists && session.IsActive {
		session.Duration = time.Since(session.StartTime)
		stats["current_session"] = session
	}

	// In a real implementation, you would query historical data from database
	// For now, we'll just return the basic stats structure

	return stats
}
