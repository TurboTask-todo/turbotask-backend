package service

import (
	"context"

	"quantumtask-auth-api/internal/models"
	"quantumtask-auth-api/pkg/websocket"

	"github.com/google/uuid"
)

// WebSocketSubtaskService extends SubtaskService with real-time events
type WebSocketSubtaskService struct {
	*SubtaskService
	broadcaster *websocket.EventBroadcaster
	todoService *TodoService
}

// NewWebSocketSubtaskService creates a new WebSocket-enabled subtask service
func NewWebSocketSubtaskService(subtaskService *SubtaskService, broadcaster *websocket.EventBroadcaster, todoService *TodoService) *WebSocketSubtaskService {
	return &WebSocketSubtaskService{
		SubtaskService: subtaskService,
		broadcaster:    broadcaster,
		todoService:    todoService,
	}
}

// CreateSubtask creates a new subtask and broadcasts the event
func (s *WebSocketSubtaskService) CreateSubtask(ctx context.Context, todoID uuid.UUID, userID uuid.UUID, req *models.CreateSubtaskRequest, sessionID *string) (*models.Subtask, error) {
	// Create subtask object from request
	subtask := &models.Subtask{
		TodoID:        todoID,
		Name:          req.Name,
		Description:   req.Description,
		EstimatedTime: req.EstimatedTime,
		DueDate:       req.DueDate,
	}

	// Set defaults
	if req.Priority != nil {
		subtask.Priority = *req.Priority
	} else {
		subtask.Priority = models.PriorityMedium
	}

	if req.SortOrder != nil {
		subtask.SortOrder = *req.SortOrder
	}

	// Create subtask using the base service
	createdSubtask, err := s.SubtaskService.CreateSubtask(ctx, userID, subtask)
	if err != nil {
		return nil, err
	}

	// Broadcast the creation event
	s.broadcaster.BroadcastSubtaskCreated(ctx, createdSubtask, userID, sessionID)

	return createdSubtask, nil
}

// UpdateSubtask updates a subtask and broadcasts the event
func (s *WebSocketSubtaskService) UpdateSubtask(ctx context.Context, subtaskID uuid.UUID, userID uuid.UUID, req *models.UpdateSubtaskRequest, sessionID *string) (*models.Subtask, error) {
	// Get current subtask to track changes using repository directly
	currentSubtask, err := s.subtaskRepo.GetByID(ctx, subtaskID)
	if err != nil {
		return nil, err
	}

	// Build updates map from request
	updates := make(map[string]interface{})
	changes := make(map[string]interface{})

	if req.Name != nil {
		updates["name"] = *req.Name
		if currentSubtask.Name != *req.Name {
			changes["name"] = map[string]interface{}{
				"old": currentSubtask.Name,
				"new": *req.Name,
			}
		}
	}

	if req.Status != nil {
		updates["status"] = *req.Status
		if currentSubtask.Status != *req.Status {
			changes["status"] = map[string]interface{}{
				"old": currentSubtask.Status,
				"new": *req.Status,
			}
		}
	}

	if req.Priority != nil {
		updates["priority"] = *req.Priority
	}

	if req.EstimatedTime != nil {
		updates["estimated_time"] = *req.EstimatedTime
	}

	if req.DueDate != nil {
		updates["due_date"] = *req.DueDate
	}

	if req.Description != nil {
		updates["description"] = *req.Description
	}

	// Update subtask using the base service
	updatedSubtask, err := s.SubtaskService.UpdateSubtask(ctx, subtaskID, userID, updates)
	if err != nil {
		return nil, err
	}

	// Check if it was completed
	if req.Status != nil && *req.Status == models.TaskStatusCompleted {
		s.broadcaster.BroadcastSubtaskCompleted(ctx, updatedSubtask, userID, sessionID)
		return updatedSubtask, nil
	}

	// Broadcast the update event
	s.broadcaster.BroadcastSubtaskUpdated(ctx, updatedSubtask, userID, changes, sessionID)

	return updatedSubtask, nil
}

// DeleteSubtask deletes a subtask and broadcasts the event
func (s *WebSocketSubtaskService) DeleteSubtask(ctx context.Context, subtaskID uuid.UUID, userID uuid.UUID, sessionID *string) error {
	// Get the subtask before deletion
	subtask, err := s.subtaskRepo.GetByID(ctx, subtaskID)
	if err != nil {
		return err
	}

	// Delete subtask using the base service
	err = s.SubtaskService.DeleteSubtask(ctx, subtaskID, userID)
	if err != nil {
		return err
	}

	// Broadcast the deletion event
	s.broadcaster.BroadcastSubtaskDeleted(ctx, subtaskID, subtask.TodoID, userID, sessionID)

	return nil
}

// ReorderSubtasks reorders subtasks and broadcasts the event
func (s *WebSocketSubtaskService) ReorderSubtasks(ctx context.Context, todoID uuid.UUID, userID uuid.UUID, subtaskIDs []uuid.UUID, sessionID *string) error {
	// Reorder using the base service
	err := s.SubtaskService.ReorderSubtasks(ctx, todoID, userID, subtaskIDs)
	if err != nil {
		return err
	}

	// Broadcast the reorder event
	s.broadcaster.BroadcastSubtaskReordered(ctx, todoID, userID, subtaskIDs, sessionID)

	return nil
}

// WebSocketProjectService extends ProjectService with real-time events
type WebSocketProjectService struct {
	*ProjectService
	broadcaster *websocket.EventBroadcaster
}

// NewWebSocketProjectService creates a new WebSocket-enabled project service
func NewWebSocketProjectService(projectService *ProjectService, broadcaster *websocket.EventBroadcaster) *WebSocketProjectService {
	return &WebSocketProjectService{
		ProjectService: projectService,
		broadcaster:    broadcaster,
	}
}

// CreateProject creates a new project and broadcasts the event
func (s *WebSocketProjectService) CreateProject(ctx context.Context, userID uuid.UUID, req *models.CreateProjectRequest, sessionID *string) (*models.Project, error) {
	// Create project using the base service
	project, err := s.ProjectService.CreateProject(ctx, userID, req)
	if err != nil {
		return nil, err
	}

	// Broadcast the creation event
	s.broadcaster.BroadcastProjectCreated(ctx, project, sessionID)

	return project, nil
}

// UpdateProject updates a project and broadcasts the event
func (s *WebSocketProjectService) UpdateProject(ctx context.Context, projectID, userID uuid.UUID, req *models.UpdateProjectRequest, sessionID *string) (*models.Project, error) {
	// Get current project to track changes
	currentProject, err := s.ProjectService.GetProject(ctx, projectID, userID, false)
	if err != nil {
		return nil, err
	}

	// Update project using the base service
	updatedProject, err := s.ProjectService.UpdateProject(ctx, projectID, userID, req)
	if err != nil {
		return nil, err
	}

	// Track changes for broadcast
	changes := make(map[string]interface{})

	if req.Title != nil && currentProject.Title != *req.Title {
		changes["title"] = map[string]interface{}{
			"old": currentProject.Title,
			"new": *req.Title,
		}
	}

	if req.Description != nil {
		if (currentProject.Description == nil && req.Description != nil) ||
			(currentProject.Description != nil && req.Description == nil) ||
			(currentProject.Description != nil && req.Description != nil && *currentProject.Description != *req.Description) {
			changes["description"] = map[string]interface{}{
				"old": currentProject.Description,
				"new": req.Description,
			}
		}
	}

	// Broadcast the update event
	s.broadcaster.BroadcastProjectUpdated(ctx, updatedProject, changes, sessionID)

	return updatedProject, nil
}

// DeleteProject deletes a project and broadcasts the event
func (s *WebSocketProjectService) DeleteProject(ctx context.Context, projectID, userID uuid.UUID, sessionID *string) error {
	// Delete project using the base service
	err := s.ProjectService.DeleteProject(ctx, projectID, userID)
	if err != nil {
		return err
	}

	// Broadcast the deletion event
	s.broadcaster.BroadcastProjectDeleted(ctx, projectID, userID, sessionID)

	return nil
}

// ArchiveProject archives a project and broadcasts the event
func (s *WebSocketProjectService) ArchiveProject(ctx context.Context, projectID, userID uuid.UUID, sessionID *string) error {
	// Archive using the base service
	err := s.ProjectService.ArchiveProject(ctx, projectID, userID)
	if err != nil {
		return err
	}

	// Get the updated project
	project, err := s.ProjectService.GetProject(ctx, projectID, userID, false)
	if err != nil {
		return nil // Don't fail if we can't broadcast
	}

	// Broadcast the update event
	changes := map[string]interface{}{
		"is_archived": map[string]interface{}{
			"old": false,
			"new": true,
		},
	}
	s.broadcaster.BroadcastProjectUpdated(ctx, project, changes, sessionID)

	return nil
}
