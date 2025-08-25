package service

import (
	"context"

	"macwrite-auth-api/internal/models"
	"macwrite-auth-api/pkg/websocket"

	"github.com/google/uuid"
)

// WebSocketTodoService extends TodoService with real-time WebSocket events
type WebSocketTodoService struct {
	*TodoService
	broadcaster *websocket.EventBroadcaster
}

// NewWebSocketTodoService creates a new WebSocket-enabled todo service
func NewWebSocketTodoService(todoService *TodoService, broadcaster *websocket.EventBroadcaster) *WebSocketTodoService {
	return &WebSocketTodoService{
		TodoService: todoService,
		broadcaster: broadcaster,
	}
}

// CreateTodo creates a new todo and broadcasts the event
func (s *WebSocketTodoService) CreateTodo(ctx context.Context, userID uuid.UUID, req *models.CreateTodoRequest, sessionID *string) (*models.Todo, error) {
	// Create todo using the base service
	todo, err := s.TodoService.CreateTodo(ctx, userID, req)
	if err != nil {
		return nil, err
	}

	// Broadcast the creation event
	s.broadcaster.BroadcastTodoCreated(ctx, todo, sessionID)

	return todo, nil
}

// UpdateTodo updates a todo and broadcasts the event
func (s *WebSocketTodoService) UpdateTodo(ctx context.Context, todoID, userID uuid.UUID, req *models.UpdateTodoRequest, sessionID *string) (*models.Todo, error) {
	// Get current todo to track changes
	currentTodo, err := s.TodoService.GetTodo(ctx, todoID, userID, false)
	if err != nil {
		return nil, err
	}

	// Update todo using the base service
	updatedTodo, err := s.TodoService.UpdateTodo(ctx, todoID, userID, req)
	if err != nil {
		return nil, err
	}

	// Track changes for broadcast
	changes := make(map[string]interface{})

	if req.TaskName != nil && currentTodo.TaskName != *req.TaskName {
		changes["task_name"] = map[string]interface{}{
			"old": currentTodo.TaskName,
			"new": *req.TaskName,
		}
	}

	if req.Status != nil && currentTodo.Status != *req.Status {
		changes["status"] = map[string]interface{}{
			"old": currentTodo.Status,
			"new": *req.Status,
		}
	}

	if req.Priority != nil && currentTodo.Priority != *req.Priority {
		changes["priority"] = map[string]interface{}{
			"old": currentTodo.Priority,
			"new": *req.Priority,
		}
	}

	if req.DueDate != nil {
		if (currentTodo.DueDate == nil && req.DueDate != nil) ||
			(currentTodo.DueDate != nil && req.DueDate == nil) ||
			(currentTodo.DueDate != nil && req.DueDate != nil && !currentTodo.DueDate.Equal(*req.DueDate)) {
			changes["due_date"] = map[string]interface{}{
				"old": currentTodo.DueDate,
				"new": req.DueDate,
			}
		}
	}

	// Note: CompletionPercentage field doesn't exist in UpdateTodoRequest
	// This would need to be added to the models if completion percentage updates are needed

	// Broadcast the update event
	s.broadcaster.BroadcastTodoUpdated(ctx, updatedTodo, changes, sessionID)

	return updatedTodo, nil
}

// DeleteTodo deletes a todo and broadcasts the event
func (s *WebSocketTodoService) DeleteTodo(ctx context.Context, todoID, userID uuid.UUID, sessionID *string) error {
	// Get the todo before deletion to get project ID
	todo, err := s.TodoService.GetTodo(ctx, todoID, userID, false)
	if err != nil {
		return err
	}

	// Delete todo using the base service
	err = s.TodoService.DeleteTodo(ctx, todoID, userID)
	if err != nil {
		return err
	}

	// Broadcast the deletion event
	s.broadcaster.BroadcastTodoDeleted(ctx, todoID, userID, todo.ProjectID, sessionID)

	return nil
}

// MarkTodoComplete marks a todo as completed and broadcasts events
func (s *WebSocketTodoService) MarkTodoComplete(ctx context.Context, todoID, userID uuid.UUID, sessionID *string) error {
	// Mark complete using the base service
	err := s.TodoService.MarkTodoComplete(ctx, todoID, userID)
	if err != nil {
		return err
	}

	// Get the updated todo
	todo, err := s.TodoService.GetTodo(ctx, todoID, userID, false)
	if err != nil {
		return err
	}

	// Broadcast the completion event
	s.broadcaster.BroadcastTodoCompleted(ctx, todo, sessionID)

	return nil
}

// MarkTodoIncomplete marks a todo as incomplete and broadcasts the event
func (s *WebSocketTodoService) MarkTodoIncomplete(ctx context.Context, todoID, userID uuid.UUID, sessionID *string) error {
	// Mark incomplete using the base service
	err := s.TodoService.MarkTodoIncomplete(ctx, todoID, userID)
	if err != nil {
		return err
	}

	// Get the updated todo
	todo, err := s.TodoService.GetTodo(ctx, todoID, userID, false)
	if err != nil {
		return err
	}

	// Broadcast the update event
	changes := map[string]interface{}{
		"status": map[string]interface{}{
			"old": models.TaskStatusCompleted,
			"new": models.TaskStatusInProgress,
		},
	}
	s.broadcaster.BroadcastTodoUpdated(ctx, todo, changes, sessionID)

	return nil
}

// PinTodo pins/unpins a todo and broadcasts the event
func (s *WebSocketTodoService) PinTodo(ctx context.Context, todoID, userID uuid.UUID, pinned bool, sessionID *string) error {
	// Pin/unpin using the base service
	err := s.TodoService.PinTodo(ctx, todoID, userID, pinned)
	if err != nil {
		return err
	}

	// Get the updated todo
	todo, err := s.TodoService.GetTodo(ctx, todoID, userID, false)
	if err != nil {
		return err
	}

	// Broadcast the pin event
	s.broadcaster.BroadcastTodoPinned(ctx, todo, pinned, sessionID)

	return nil
}

// ArchiveTodo archives a todo and broadcasts the event
func (s *WebSocketTodoService) ArchiveTodo(ctx context.Context, todoID, userID uuid.UUID, sessionID *string) error {
	// Archive using the base service
	err := s.TodoService.ArchiveTodo(ctx, todoID, userID)
	if err != nil {
		return err
	}

	// Get the updated todo
	todo, err := s.TodoService.GetTodo(ctx, todoID, userID, false)
	if err != nil {
		return err
	}

	// Broadcast the update event
	changes := map[string]interface{}{
		"is_archived": map[string]interface{}{
			"old": false,
			"new": true,
		},
	}
	s.broadcaster.BroadcastTodoUpdated(ctx, todo, changes, sessionID)

	return nil
}

// BulkUpdateTodos updates multiple todos and broadcasts bulk event
func (s *WebSocketTodoService) BulkUpdateTodos(ctx context.Context, userID uuid.UUID, updates []struct {
	TodoID  uuid.UUID
	Updates *models.UpdateTodoRequest
}, sessionID *string) error {
	var eventPayloads []websocket.TodoEventPayload

	for _, update := range updates {
		// Update individual todo
		updatedTodo, err := s.TodoService.UpdateTodo(ctx, update.TodoID, userID, update.Updates)
		if err != nil {
			continue // Skip failed updates
		}

		// Create event payload
		payload := websocket.TodoEventPayload{
			Action:    "updated",
			Todo:      updatedTodo,
			ProjectID: updatedTodo.ProjectID,
		}
		eventPayloads = append(eventPayloads, payload)
	}

	// Broadcast bulk update
	if len(eventPayloads) > 0 {
		s.broadcaster.BroadcastBulkTodoUpdates(ctx, eventPayloads, userID, sessionID)
	}

	return nil
}

// Kanban Board WebSocket Methods

// MoveTodoToColumn moves a todo to a different column and broadcasts the event
func (s *WebSocketTodoService) MoveTodoToColumn(ctx context.Context, userID uuid.UUID, req *models.MoveTodoRequest, sessionID *string) (*models.TodoMoveResponse, error) {
	// Move todo using the base service
	response, err := s.TodoService.MoveTodoToColumn(ctx, userID, req)
	if err != nil {
		return nil, err
	}

	// Get the updated todo for broadcasting
	todo, err := s.TodoService.GetTodo(ctx, req.TodoID, userID, false)
	if err != nil {
		return response, nil // Return response even if broadcast fails
	}

	// Broadcast the kanban move event as a todo update
	changes := map[string]interface{}{
		"status": map[string]interface{}{
			"old": response.OldStatus,
			"new": response.NewStatus,
		},
		"position": response.NewPosition,
	}
	s.broadcaster.BroadcastTodoUpdated(ctx, todo, changes, sessionID)

	return response, nil
}

// ReorderTodosInColumn reorders todos within a column and broadcasts the event
func (s *WebSocketTodoService) ReorderTodosInColumn(ctx context.Context, userID uuid.UUID, req *models.ReorderTodosRequest, sessionID *string) (*models.ReorderResponse, error) {
	// Reorder todos using the base service
	response, err := s.TodoService.ReorderTodosInColumn(ctx, userID, req)
	if err != nil {
		return nil, err
	}

	// Broadcast reorder event as a general update for each todo
	for _, todoID := range req.TodoIDs {
		todo, err := s.TodoService.GetTodo(ctx, todoID, userID, false)
		if err != nil {
			continue // Skip if we can't get the todo
		}
		changes := map[string]interface{}{
			"reordered": true,
			"status":    req.Status,
		}
		s.broadcaster.BroadcastTodoUpdated(ctx, todo, changes, sessionID)
	}

	return response, nil
}

// BulkMoveTodos moves multiple todos and broadcasts the events
func (s *WebSocketTodoService) BulkMoveTodos(ctx context.Context, userID uuid.UUID, req *models.BulkMoveTodosRequest, sessionID *string) (*models.BulkMoveResponse, error) {
	// Perform bulk move using the base service
	response, err := s.TodoService.BulkMoveTodos(ctx, userID, req)
	if err != nil {
		return nil, err
	}

	// Broadcast events for successful moves
	for _, movedTodo := range response.MovedTodos {
		// Get the updated todo
		todo, err := s.TodoService.GetTodo(ctx, movedTodo.TodoID, userID, false)
		if err != nil {
			continue // Skip if we can't get the todo
		}

		// Broadcast as individual todo updates
		changes := map[string]interface{}{
			"status": map[string]interface{}{
				"old": movedTodo.OldStatus,
				"new": movedTodo.NewStatus,
			},
			"position": movedTodo.NewPosition,
		}
		s.broadcaster.BroadcastTodoUpdated(ctx, todo, changes, sessionID)
	}

	return response, nil
}

// GetSessionIDFromContext extracts session ID from request context
func GetSessionIDFromContext(ctx context.Context) *string {
	if sessionID, ok := ctx.Value("session_id").(string); ok {
		return &sessionID
	}
	return nil
}
