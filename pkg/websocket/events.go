package websocket

import (
	"context"
	"log"
	"time"

	"quantumtask-auth-api/internal/models"

	"github.com/google/uuid"
)

// EventBroadcaster handles broadcasting of real-time events
type EventBroadcaster struct {
	hub *Hub
}

// NewEventBroadcaster creates a new event broadcaster
func NewEventBroadcaster(hub *Hub) *EventBroadcaster {
	return &EventBroadcaster{
		hub: hub,
	}
}

// BroadcastTodoCreated broadcasts a todo created event
func (eb *EventBroadcaster) BroadcastTodoCreated(ctx context.Context, todo *models.Todo, excludeSessionID *string) {
	payload := TodoEventPayload{
		Action:    "created",
		Todo:      todo,
		ProjectID: todo.ProjectID,
	}

	message := Message{
		Type:      EventTypeTodoCreated,
		Payload:   payload,
		Timestamp: time.Now(),
		EventID:   uuid.New().String(),
		UserID:    &todo.UserID,
	}

	eb.hub.BroadcastToUser(todo.UserID, message, excludeSessionID)
	log.Printf("Broadcasted todo created event: %s for user %s", todo.ID.String(), todo.UserID.String())
}

// BroadcastTodoUpdated broadcasts a todo updated event
func (eb *EventBroadcaster) BroadcastTodoUpdated(ctx context.Context, todo *models.Todo, changes map[string]interface{}, excludeSessionID *string) {
	payload := TodoEventPayload{
		Action:    "updated",
		Todo:      todo,
		ProjectID: todo.ProjectID,
		Changes:   changes,
	}

	message := Message{
		Type:      EventTypeTodoUpdated,
		Payload:   payload,
		Timestamp: time.Now(),
		EventID:   uuid.New().String(),
		UserID:    &todo.UserID,
	}

	eb.hub.BroadcastToUser(todo.UserID, message, excludeSessionID)
	log.Printf("Broadcasted todo updated event: %s for user %s", todo.ID.String(), todo.UserID.String())
}

// BroadcastTodoDeleted broadcasts a todo deleted event
func (eb *EventBroadcaster) BroadcastTodoDeleted(ctx context.Context, todoID, userID, projectID uuid.UUID, excludeSessionID *string) {
	payload := TodoEventPayload{
		Action:    "deleted",
		Todo:      map[string]interface{}{"id": todoID},
		ProjectID: projectID,
	}

	message := Message{
		Type:      EventTypeTodoDeleted,
		Payload:   payload,
		Timestamp: time.Now(),
		EventID:   uuid.New().String(),
		UserID:    &userID,
	}

	eb.hub.BroadcastToUser(userID, message, excludeSessionID)
	log.Printf("Broadcasted todo deleted event: %s for user %s", todoID.String(), userID.String())
}

// BroadcastTodoCompleted broadcasts a todo completed event
func (eb *EventBroadcaster) BroadcastTodoCompleted(ctx context.Context, todo *models.Todo, excludeSessionID *string) {
	payload := TodoEventPayload{
		Action:    "completed",
		Todo:      todo,
		ProjectID: todo.ProjectID,
	}

	message := Message{
		Type:      EventTypeTodoCompleted,
		Payload:   payload,
		Timestamp: time.Now(),
		EventID:   uuid.New().String(),
		UserID:    &todo.UserID,
	}

	eb.hub.BroadcastToUser(todo.UserID, message, excludeSessionID)

	// Send congratulatory notification
	eb.BroadcastNotification(ctx, todo.UserID, NotificationPayload{
		Title:   "Task Completed! 🎉",
		Message: "Great job completing: " + todo.TaskName,
		Type:    "success",
	}, excludeSessionID)

	log.Printf("Broadcasted todo completed event: %s for user %s", todo.ID.String(), todo.UserID.String())
}

// BroadcastTodoPinned broadcasts a todo pinned/unpinned event
func (eb *EventBroadcaster) BroadcastTodoPinned(ctx context.Context, todo *models.Todo, pinned bool, excludeSessionID *string) {
	action := "unpinned"
	if pinned {
		action = "pinned"
	}

	payload := TodoEventPayload{
		Action:    action,
		Todo:      todo,
		ProjectID: todo.ProjectID,
	}

	message := Message{
		Type:      EventTypeTodoPinned,
		Payload:   payload,
		Timestamp: time.Now(),
		EventID:   uuid.New().String(),
		UserID:    &todo.UserID,
	}

	eb.hub.BroadcastToUser(todo.UserID, message, excludeSessionID)
	log.Printf("Broadcasted todo %s event: %s for user %s", action, todo.ID.String(), todo.UserID.String())
}

// BroadcastSubtaskCreated broadcasts a subtask created event
func (eb *EventBroadcaster) BroadcastSubtaskCreated(ctx context.Context, subtask *models.Subtask, todoUserID uuid.UUID, excludeSessionID *string) {
	payload := SubtaskEventPayload{
		Action:  "created",
		Subtask: subtask,
		TodoID:  subtask.TodoID,
	}

	message := Message{
		Type:      EventTypeSubtaskCreated,
		Payload:   payload,
		Timestamp: time.Now(),
		EventID:   uuid.New().String(),
		UserID:    &todoUserID,
	}

	eb.hub.BroadcastToUser(todoUserID, message, excludeSessionID)
	log.Printf("Broadcasted subtask created event: %s for todo %s", subtask.ID.String(), subtask.TodoID.String())
}

// BroadcastSubtaskUpdated broadcasts a subtask updated event
func (eb *EventBroadcaster) BroadcastSubtaskUpdated(ctx context.Context, subtask *models.Subtask, todoUserID uuid.UUID, changes map[string]interface{}, excludeSessionID *string) {
	payload := SubtaskEventPayload{
		Action:  "updated",
		Subtask: subtask,
		TodoID:  subtask.TodoID,
		Changes: changes,
	}

	message := Message{
		Type:      EventTypeSubtaskUpdated,
		Payload:   payload,
		Timestamp: time.Now(),
		EventID:   uuid.New().String(),
		UserID:    &todoUserID,
	}

	eb.hub.BroadcastToUser(todoUserID, message, excludeSessionID)
	log.Printf("Broadcasted subtask updated event: %s for todo %s", subtask.ID.String(), subtask.TodoID.String())
}

// BroadcastSubtaskDeleted broadcasts a subtask deleted event
func (eb *EventBroadcaster) BroadcastSubtaskDeleted(ctx context.Context, subtaskID, todoID, todoUserID uuid.UUID, excludeSessionID *string) {
	payload := SubtaskEventPayload{
		Action:  "deleted",
		Subtask: map[string]interface{}{"id": subtaskID},
		TodoID:  todoID,
	}

	message := Message{
		Type:      EventTypeSubtaskDeleted,
		Payload:   payload,
		Timestamp: time.Now(),
		EventID:   uuid.New().String(),
		UserID:    &todoUserID,
	}

	eb.hub.BroadcastToUser(todoUserID, message, excludeSessionID)
	log.Printf("Broadcasted subtask deleted event: %s for todo %s", subtaskID.String(), todoID.String())
}

// BroadcastSubtaskCompleted broadcasts a subtask completed event
func (eb *EventBroadcaster) BroadcastSubtaskCompleted(ctx context.Context, subtask *models.Subtask, todoUserID uuid.UUID, excludeSessionID *string) {
	payload := SubtaskEventPayload{
		Action:  "completed",
		Subtask: subtask,
		TodoID:  subtask.TodoID,
	}

	message := Message{
		Type:      EventTypeSubtaskCompleted,
		Payload:   payload,
		Timestamp: time.Now(),
		EventID:   uuid.New().String(),
		UserID:    &todoUserID,
	}

	eb.hub.BroadcastToUser(todoUserID, message, excludeSessionID)
	log.Printf("Broadcasted subtask completed event: %s for todo %s", subtask.ID.String(), subtask.TodoID.String())
}

// BroadcastSubtaskReordered broadcasts a subtask reorder event
func (eb *EventBroadcaster) BroadcastSubtaskReordered(ctx context.Context, todoID, todoUserID uuid.UUID, subtaskIDs []uuid.UUID, excludeSessionID *string) {
	payload := SubtaskEventPayload{
		Action: "reordered",
		Subtask: map[string]interface{}{
			"todo_id":     todoID,
			"subtask_ids": subtaskIDs,
		},
		TodoID: todoID,
	}

	message := Message{
		Type:      EventTypeSubtaskReordered,
		Payload:   payload,
		Timestamp: time.Now(),
		EventID:   uuid.New().String(),
		UserID:    &todoUserID,
	}

	eb.hub.BroadcastToUser(todoUserID, message, excludeSessionID)
	log.Printf("Broadcasted subtask reordered event for todo %s", todoID.String())
}

// BroadcastProjectCreated broadcasts a project created event
func (eb *EventBroadcaster) BroadcastProjectCreated(ctx context.Context, project *models.Project, excludeSessionID *string) {
	payload := ProjectEventPayload{
		Action:  "created",
		Project: project,
	}

	message := Message{
		Type:      EventTypeProjectCreated,
		Payload:   payload,
		Timestamp: time.Now(),
		EventID:   uuid.New().String(),
		UserID:    &project.UserID,
	}

	eb.hub.BroadcastToUser(project.UserID, message, excludeSessionID)
	log.Printf("Broadcasted project created event: %s for user %s", project.ID.String(), project.UserID.String())
}

// BroadcastProjectUpdated broadcasts a project updated event
func (eb *EventBroadcaster) BroadcastProjectUpdated(ctx context.Context, project *models.Project, changes map[string]interface{}, excludeSessionID *string) {
	payload := ProjectEventPayload{
		Action:  "updated",
		Project: project,
		Changes: changes,
	}

	message := Message{
		Type:      EventTypeProjectUpdated,
		Payload:   payload,
		Timestamp: time.Now(),
		EventID:   uuid.New().String(),
		UserID:    &project.UserID,
	}

	eb.hub.BroadcastToUser(project.UserID, message, excludeSessionID)
	log.Printf("Broadcasted project updated event: %s for user %s", project.ID.String(), project.UserID.String())
}

// BroadcastProjectDeleted broadcasts a project deleted event
func (eb *EventBroadcaster) BroadcastProjectDeleted(ctx context.Context, projectID, userID uuid.UUID, excludeSessionID *string) {
	payload := ProjectEventPayload{
		Action:  "deleted",
		Project: map[string]interface{}{"id": projectID},
	}

	message := Message{
		Type:      EventTypeProjectDeleted,
		Payload:   payload,
		Timestamp: time.Now(),
		EventID:   uuid.New().String(),
		UserID:    &userID,
	}

	eb.hub.BroadcastToUser(userID, message, excludeSessionID)
	log.Printf("Broadcasted project deleted event: %s for user %s", projectID.String(), userID.String())
}

// BroadcastNotification broadcasts a notification to a user
func (eb *EventBroadcaster) BroadcastNotification(ctx context.Context, userID uuid.UUID, notification NotificationPayload, excludeSessionID *string) {
	message := Message{
		Type:      EventTypeNotification,
		Payload:   notification,
		Timestamp: time.Now(),
		EventID:   uuid.New().String(),
		UserID:    &userID,
	}

	eb.hub.BroadcastToUser(userID, message, excludeSessionID)
	log.Printf("Broadcasted notification to user %s: %s", userID.String(), notification.Title)
}

// BroadcastFocusSessionActive broadcasts that a user has an active focus session
func (eb *EventBroadcaster) BroadcastFocusSessionActive(ctx context.Context, userID uuid.UUID, session *FocusSession, excludeSessionID *string) {
	payload := FocusSessionEventPayload{
		Action:  "active",
		Session: *session,
		Message: "User is currently focusing",
	}

	message := Message{
		Type:      EventTypeFocusSessionActive,
		Payload:   payload,
		Timestamp: time.Now(),
		EventID:   uuid.New().String(),
		UserID:    &userID,
	}

	// Broadcast to all connected users so they can see who's focusing
	eb.hub.BroadcastToUsers(eb.hub.GetConnectedUsers(), message)
	log.Printf("Broadcasted focus session active for user %s", userID.String())
}

// BroadcastBulkTodoUpdates broadcasts multiple todo updates at once
func (eb *EventBroadcaster) BroadcastBulkTodoUpdates(ctx context.Context, updates []TodoEventPayload, userID uuid.UUID, excludeSessionID *string) {
	message := Message{
		Type: "bulk_todo_updates",
		Payload: map[string]interface{}{
			"updates": updates,
		},
		Timestamp: time.Now(),
		EventID:   uuid.New().String(),
		UserID:    &userID,
	}

	eb.hub.BroadcastToUser(userID, message, excludeSessionID)
	log.Printf("Broadcasted bulk todo updates for user %s (%d updates)", userID.String(), len(updates))
}

// BroadcastUserPresence broadcasts user presence information
func (eb *EventBroadcaster) BroadcastUserPresence(ctx context.Context, userID uuid.UUID, isOnline bool, excludeSessionID *string) {
	activity := "user_offline"
	if isOnline {
		activity = "user_online"
	}

	payload := UserActivityPayload{
		UserID:   userID,
		Activity: activity,
	}

	message := Message{
		Type:      EventTypeUserActivity,
		Payload:   payload,
		Timestamp: time.Now(),
		EventID:   uuid.New().String(),
		UserID:    &userID,
	}

	// Broadcast to all connected users
	eb.hub.BroadcastToUsers(eb.hub.GetConnectedUsers(), message)
	log.Printf("Broadcasted user presence for user %s: %s", userID.String(), activity)
}

// BroadcastSystemAlert broadcasts a system-wide alert
func (eb *EventBroadcaster) BroadcastSystemAlert(ctx context.Context, title, message, alertType string) {
	notification := NotificationPayload{
		Title:   title,
		Message: message,
		Type:    alertType,
	}

	msgObj := Message{
		Type:      EventTypeNotification,
		Payload:   notification,
		Timestamp: time.Now(),
		EventID:   uuid.New().String(),
	}

	// Broadcast to all connected users
	eb.hub.BroadcastToUsers(eb.hub.GetConnectedUsers(), msgObj)
	log.Printf("Broadcasted system alert: %s", title)
}
