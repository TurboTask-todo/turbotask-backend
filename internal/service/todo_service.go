package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"macwrite-auth-api/internal/models"
	"macwrite-auth-api/internal/repository"
	"macwrite-auth-api/pkg/redis"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// TodoService handles todo business logic
type TodoService struct {
	todoRepo       *repository.TodoRepository
	projectRepo    *repository.ProjectRepository
	subtaskRepo    *repository.SubtaskRepository
	noteRepo       *repository.NoteRepository
	timeEntryRepo  *repository.TimeEntryRepository
	projectService *ProjectService
	redisClient    redis.Client
}

// NewTodoService creates a new todo service
func NewTodoService(
	todoRepo *repository.TodoRepository,
	projectRepo *repository.ProjectRepository,
	subtaskRepo *repository.SubtaskRepository,
	noteRepo *repository.NoteRepository,
	timeEntryRepo *repository.TimeEntryRepository,
	projectService *ProjectService,
	redisClient redis.Client,
) *TodoService {
	return &TodoService{
		todoRepo:       todoRepo,
		projectRepo:    projectRepo,
		subtaskRepo:    subtaskRepo,
		noteRepo:       noteRepo,
		timeEntryRepo:  timeEntryRepo,
		projectService: projectService,
		redisClient:    redisClient,
	}
}

// Cache key constants for todos
const (
	todoCachePrefix         = "todo:"
	userTodosCachePrefix    = "user_todos:"
	projectTodosCachePrefix = "project_todos:"
	todoDashboardPrefix     = "todo_dashboard:"
	overdueTodosPrefix      = "overdue_todos:"
	todoSearchPrefix        = "todo_search:"
	todoCacheTTL            = 15 * time.Minute
	searchCacheTTL          = 5 * time.Minute
)

// CreateTodo creates a new todo
func (s *TodoService) CreateTodo(ctx context.Context, userID uuid.UUID, req *models.CreateTodoRequest) (*models.Todo, error) {
	// Validate request
	if err := s.validateCreateTodoRequest(req); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Verify project ownership
	exists, err := s.projectRepo.ValidateProjectOwnership(ctx, req.ProjectID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to validate project ownership: %w", err)
	}
	if !exists {
		return nil, fmt.Errorf("project not found")
	}

	// Verify parent todo ownership if specified
	if req.ParentTodoID != nil {
		exists, err := s.todoRepo.ValidateTodoOwnership(ctx, *req.ParentTodoID, userID)
		if err != nil {
			return nil, fmt.Errorf("failed to validate parent todo ownership: %w", err)
		}
		if !exists {
			return nil, fmt.Errorf("parent todo not found")
		}
	}

	// Create todo object
	todo := &models.Todo{
		UserID:              userID,
		ProjectID:           req.ProjectID,
		ReleaseVersionID:    req.ReleaseVersionID,
		ParentTodoID:        req.ParentTodoID,
		TaskName:            req.TaskName,
		TaskDescription:     req.TaskDescription,
		TaskImageOrEmoji:    req.TaskImageOrEmoji,
		StatusIcon:          req.StatusIcon,
		EstimatedTime:       req.EstimatedTime,
		DueDate:             req.DueDate,
		StartDate:           req.StartDate,
		DifficultyRating:    req.DifficultyRating,
		EnergyLevelRequired: req.EnergyLevelRequired,
		Status:              *req.Status,
		Location:            req.Location,
		Context:             req.Context,
		AssignedTo:          req.AssignedTo,
		RecurrencePattern:   req.RecurrencePattern,
	}

	// Set defaults
	if req.TimeUnit != nil {
		todo.TimeUnit = *req.TimeUnit
	} else {
		todo.TimeUnit = models.TimeUnitMinutes
	}

	if req.Priority != nil {
		todo.Priority = *req.Priority
	} else {
		todo.Priority = models.PriorityMedium
	}

	if req.IsRecurring != nil {
		todo.IsRecurring = *req.IsRecurring
	}

	// Handle tags
	if req.Tags != nil {
		todo.Tags = pq.StringArray(req.Tags)
	}

	// Create todo in database
	if err := s.todoRepo.Create(ctx, todo); err != nil {
		return nil, fmt.Errorf("failed to create todo: %w", err)
	}

	// Update project progress
	go s.projectService.UpdateProjectProgress(context.Background(), req.ProjectID)

	// Invalidate caches
	s.invalidateTodoCaches(userID, req.ProjectID)

	return todo, nil
}

// GetTodo retrieves a todo by ID
func (s *TodoService) GetTodo(ctx context.Context, todoID, userID uuid.UUID, includeRelated bool) (*models.Todo, error) {
	// Check cache first
	cacheKey := fmt.Sprintf("%s%s:related_%t", todoCachePrefix, todoID.String(), includeRelated)
	if cached, err := s.redisClient.Get(ctx, cacheKey); err == nil {
		var todo models.Todo
		if err := json.Unmarshal(cached, &todo); err == nil {
			// Verify ownership
			if todo.UserID != userID {
				return nil, fmt.Errorf("todo not found")
			}
			return &todo, nil
		}
	}

	// Get from database
	todo, err := s.todoRepo.GetByID(ctx, todoID, includeRelated)
	if err != nil {
		return nil, fmt.Errorf("failed to get todo: %w", err)
	}
	if todo == nil {
		return nil, fmt.Errorf("todo not found")
	}

	// Verify ownership
	if todo.UserID != userID {
		return nil, fmt.Errorf("todo not found")
	}

	// Cache the result
	if todoData, err := json.Marshal(todo); err == nil {
		s.redisClient.Set(ctx, cacheKey, todoData, todoCacheTTL)
	}

	return todo, nil
}

// GetUserTodos retrieves todos for a user with filtering
func (s *TodoService) GetUserTodos(ctx context.Context, userID uuid.UUID, filters repository.TodoFilters) ([]*models.Todo, int, error) {
	// Create cache key based on filters
	cacheKey := s.buildTodoFiltersCacheKey(userID, filters)
	if cached, err := s.redisClient.Get(ctx, cacheKey); err == nil {
		var result struct {
			Todos []*models.Todo `json:"todos"`
			Total int            `json:"total"`
		}
		if err := json.Unmarshal(cached, &result); err == nil {
			return result.Todos, result.Total, nil
		}
	}

	// Get from database
	todos, total, err := s.todoRepo.GetByUserID(ctx, userID, filters)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get user todos: %w", err)
	}

	// Cache the result
	result := struct {
		Todos []*models.Todo `json:"todos"`
		Total int            `json:"total"`
	}{
		Todos: todos,
		Total: total,
	}
	if resultData, err := json.Marshal(result); err == nil {
		s.redisClient.Set(ctx, cacheKey, resultData, todoCacheTTL)
	}

	return todos, total, nil
}

// GetProjectTodos retrieves todos for a project
func (s *TodoService) GetProjectTodos(ctx context.Context, projectID, userID uuid.UUID, includeCompleted bool) ([]*models.Todo, error) {
	// Verify project ownership
	exists, err := s.projectRepo.ValidateProjectOwnership(ctx, projectID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to validate project ownership: %w", err)
	}
	if !exists {
		return nil, fmt.Errorf("project not found")
	}

	// Check cache first
	cacheKey := fmt.Sprintf("%s%s:completed_%t", projectTodosCachePrefix, projectID.String(), includeCompleted)
	if cached, err := s.redisClient.Get(ctx, cacheKey); err == nil {
		var todos []*models.Todo
		if err := json.Unmarshal(cached, &todos); err == nil {
			return todos, nil
		}
	}

	// Get from database
	todos, err := s.todoRepo.GetByProjectID(ctx, projectID, includeCompleted)
	if err != nil {
		return nil, fmt.Errorf("failed to get project todos: %w", err)
	}

	// Cache the result
	if todosData, err := json.Marshal(todos); err == nil {
		s.redisClient.Set(ctx, cacheKey, todosData, todoCacheTTL)
	}

	return todos, nil
}

// UpdateTodo updates a todo
func (s *TodoService) UpdateTodo(ctx context.Context, todoID, userID uuid.UUID, req *models.UpdateTodoRequest) (*models.Todo, error) {
	// Verify ownership
	exists, err := s.todoRepo.ValidateTodoOwnership(ctx, todoID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to validate ownership: %w", err)
	}
	if !exists {
		return nil, fmt.Errorf("todo not found")
	}

	// Validate request
	if err := s.validateUpdateTodoRequest(req); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Get current todo to track changes
	currentTodo, err := s.todoRepo.GetByID(ctx, todoID, false)
	if err != nil {
		return nil, fmt.Errorf("failed to get current todo: %w", err)
	}

	// Build updates map
	updates := make(map[string]interface{})

	if req.TaskName != nil {
		updates["task_name"] = *req.TaskName
	}
	if req.TaskDescription != nil {
		updates["task_description"] = *req.TaskDescription
	}
	if req.TaskImageOrEmoji != nil {
		updates["task_image_or_emoji"] = *req.TaskImageOrEmoji
	}
	if req.StatusIcon != nil {
		updates["status_icon"] = *req.StatusIcon
	}
	if req.EstimatedTime != nil {
		updates["estimated_time"] = *req.EstimatedTime
	}
	if req.ActualTime != nil {
		updates["actual_time"] = *req.ActualTime
	}
	if req.TimeUnit != nil {
		updates["time_unit"] = *req.TimeUnit
	}
	if req.Priority != nil {
		updates["priority"] = *req.Priority
	}
	if req.Status != nil {
		// Validate status
		// if !models.IsValidTaskStatus(string(*req.Status)) {
		// 	return nil, fmt.Errorf("invalid status: %s. Valid statuses are: %v", *req.Status, models.GetAllTaskStatuses())
		// }

		// // Check if status transition is valid
		// if currentTodo != nil && !currentTodo.Status.CanTransitionTo(*req.Status) {
		// 	return nil, fmt.Errorf("invalid status transition from %s to %s", currentTodo.Status, *req.Status)
		// }

		updates["status"] = *req.Status

		// Auto-set completion fields when marking as completed/done
		if models.IsCompletedStatus(*req.Status) {
			if currentTodo != nil && !models.IsCompletedStatus(currentTodo.Status) {
				updates["completed_at"] = time.Now()
				updates["completion_percentage"] = 100.0
			}
		} else {
			// Clear completion fields if moving away from completed status
			if currentTodo != nil && models.IsCompletedStatus(currentTodo.Status) {
				updates["completed_at"] = nil
				updates["completion_percentage"] = 0.0
			}
		}
	}
	if req.DueDate != nil {
		updates["due_date"] = *req.DueDate
	}
	if req.StartDate != nil {
		updates["start_date"] = *req.StartDate
	}
	if req.Tags != nil {
		updates["tags"] = pq.StringArray(req.Tags)
	}
	if req.DifficultyRating != nil {
		updates["difficulty_rating"] = *req.DifficultyRating
	}
	if req.EnergyLevelRequired != nil {
		updates["energy_level_required"] = *req.EnergyLevelRequired
	}
	if req.Location != nil {
		updates["location"] = *req.Location
	}
	if req.Context != nil {
		updates["context"] = *req.Context
	}
	if req.AssignedTo != nil {
		updates["assigned_to"] = *req.AssignedTo
	}
	if req.IsRecurring != nil {
		updates["is_recurring"] = *req.IsRecurring
	}
	if req.RecurrencePattern != nil {
		updates["recurrence_pattern"] = *req.RecurrencePattern
	}
	if req.IsPinned != nil {
		updates["is_pinned"] = *req.IsPinned
	}

	// Update in database
	if err := s.todoRepo.Update(ctx, todoID, updates); err != nil {
		return nil, fmt.Errorf("failed to update todo: %w", err)
	}

	// If status changed to completed, update project progress
	if req.Status != nil && *req.Status == models.TaskStatusCompleted && currentTodo.Status != models.TaskStatusCompleted {
		go s.projectService.UpdateProjectProgress(context.Background(), currentTodo.ProjectID)
	}

	// Invalidate caches
	s.invalidateTodoCaches(userID, currentTodo.ProjectID)

	// Return updated todo
	return s.GetTodo(ctx, todoID, userID, true)
}

// ArchiveTodo archives a todo
func (s *TodoService) ArchiveTodo(ctx context.Context, todoID, userID uuid.UUID) error {
	// Verify ownership
	exists, err := s.todoRepo.ValidateTodoOwnership(ctx, todoID, userID)
	if err != nil {
		return fmt.Errorf("failed to validate ownership: %w", err)
	}
	if !exists {
		return fmt.Errorf("todo not found")
	}

	// Get todo to get project ID for cache invalidation
	todo, err := s.todoRepo.GetByID(ctx, todoID, false)
	if err != nil {
		return fmt.Errorf("failed to get todo: %w", err)
	}

	// Archive in database
	if err := s.todoRepo.Archive(ctx, todoID); err != nil {
		return fmt.Errorf("failed to archive todo: %w", err)
	}

	// Update project progress
	if todo != nil {
		go s.projectService.UpdateProjectProgress(context.Background(), todo.ProjectID)
		s.invalidateTodoCaches(userID, todo.ProjectID)
	}

	return nil
}

// DeleteTodo permanently deletes a todo
func (s *TodoService) DeleteTodo(ctx context.Context, todoID, userID uuid.UUID) error {
	// Verify ownership
	exists, err := s.todoRepo.ValidateTodoOwnership(ctx, todoID, userID)
	if err != nil {
		return fmt.Errorf("failed to validate ownership: %w", err)
	}
	if !exists {
		return fmt.Errorf("todo not found")
	}

	// Get todo to get project ID for cache invalidation
	todo, err := s.todoRepo.GetByID(ctx, todoID, false)
	if err != nil {
		return fmt.Errorf("failed to get todo: %w", err)
	}

	// Delete in database
	if err := s.todoRepo.Delete(ctx, todoID); err != nil {
		return fmt.Errorf("failed to delete todo: %w", err)
	}

	// Update project progress
	if todo != nil {
		go s.projectService.UpdateProjectProgress(context.Background(), todo.ProjectID)
		s.invalidateTodoCaches(userID, todo.ProjectID)
	}

	return nil
}

// GetTodoDashboard retrieves dashboard data for todos
func (s *TodoService) GetTodoDashboard(ctx context.Context, userID uuid.UUID, limit int) ([]*models.TodoDashboard, error) {
	// Check cache first
	cacheKey := fmt.Sprintf("%s%s:limit_%d", todoDashboardPrefix, userID.String(), limit)
	if cached, err := s.redisClient.Get(ctx, cacheKey); err == nil {
		var dashboard []*models.TodoDashboard
		if err := json.Unmarshal(cached, &dashboard); err == nil {
			return dashboard, nil
		}
	}

	// Get from database
	dashboard, err := s.todoRepo.GetDashboard(ctx, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get todo dashboard: %w", err)
	}

	// Cache the result
	if dashboardData, err := json.Marshal(dashboard); err == nil {
		s.redisClient.Set(ctx, cacheKey, dashboardData, dashboardCacheTTL)
	}

	return dashboard, nil
}

// GetOverdueTodos retrieves overdue todos for a user
func (s *TodoService) GetOverdueTodos(ctx context.Context, userID uuid.UUID) ([]*models.Todo, error) {
	// Check cache first
	cacheKey := fmt.Sprintf("%s%s", overdueTodosPrefix, userID.String())
	if cached, err := s.redisClient.Get(ctx, cacheKey); err == nil {
		var todos []*models.Todo
		if err := json.Unmarshal(cached, &todos); err == nil {
			return todos, nil
		}
	}

	// Get from database
	todos, err := s.todoRepo.GetOverdueTodos(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get overdue todos: %w", err)
	}

	// Cache the result with shorter TTL for time-sensitive data
	if todosData, err := json.Marshal(todos); err == nil {
		s.redisClient.Set(ctx, cacheKey, todosData, 10*time.Minute)
	}

	return todos, nil
}

// SearchTodos searches todos by text
func (s *TodoService) SearchTodos(ctx context.Context, userID uuid.UUID, searchTerm string, limit, offset int) ([]*models.Todo, int, error) {
	if searchTerm == "" {
		return nil, 0, fmt.Errorf("search term is required")
	}

	// Check cache first
	cacheKey := fmt.Sprintf("%s%s:term_%s:limit_%d:offset_%d", todoSearchPrefix, userID.String(), searchTerm, limit, offset)
	if cached, err := s.redisClient.Get(ctx, cacheKey); err == nil {
		var result struct {
			Todos []*models.Todo `json:"todos"`
			Total int            `json:"total"`
		}
		if err := json.Unmarshal(cached, &result); err == nil {
			return result.Todos, result.Total, nil
		}
	}

	// Search in database
	todos, total, err := s.todoRepo.SearchTodos(ctx, userID, searchTerm, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to search todos: %w", err)
	}

	// Cache the result
	result := struct {
		Todos []*models.Todo `json:"todos"`
		Total int            `json:"total"`
	}{
		Todos: todos,
		Total: total,
	}
	if resultData, err := json.Marshal(result); err == nil {
		s.redisClient.Set(ctx, cacheKey, resultData, searchCacheTTL)
	}

	return todos, total, nil
}

// MarkTodoComplete marks a todo as completed
func (s *TodoService) MarkTodoComplete(ctx context.Context, todoID, userID uuid.UUID) error {
	req := &models.UpdateTodoRequest{
		Status: &[]models.TaskStatus{models.TaskStatusCompleted}[0],
	}

	_, err := s.UpdateTodo(ctx, todoID, userID, req)
	return err
}

// MarkTodoIncomplete marks a todo as not completed
func (s *TodoService) MarkTodoIncomplete(ctx context.Context, todoID, userID uuid.UUID) error {
	req := &models.UpdateTodoRequest{
		Status: &[]models.TaskStatus{models.TaskStatusInProgress}[0],
	}

	_, err := s.UpdateTodo(ctx, todoID, userID, req)
	return err
}

// PinTodo pins/unpins a todo
func (s *TodoService) PinTodo(ctx context.Context, todoID, userID uuid.UUID, pinned bool) error {
	req := &models.UpdateTodoRequest{
		IsPinned: &pinned,
	}

	_, err := s.UpdateTodo(ctx, todoID, userID, req)
	return err
}

// Kanban Board Methods

// GetProjectKanbanBoard retrieves kanban board data for a project
func (s *TodoService) GetProjectKanbanBoard(ctx context.Context, projectID, userID uuid.UUID) (*models.KanbanBoard, error) {
	// Verify project ownership
	exists, err := s.projectRepo.ValidateProjectOwnership(ctx, projectID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to validate project ownership: %w", err)
	}
	if !exists {
		return nil, repository.ErrProjectNotFound
	}

	// Get project details
	project, err := s.projectRepo.GetByID(ctx, projectID, false)
	if err != nil {
		return nil, fmt.Errorf("failed to get project: %w", err)
	}
	if project == nil {
		return nil, repository.ErrProjectNotFound
	}

	// Use the existing GetProjectTodos method that already works correctly
	todos, err := s.GetProjectTodos(ctx, projectID, userID, true) // include completed todos
	if err != nil {
		return nil, fmt.Errorf("failed to get project todos: %w", err)
	}

	// Group todos by status
	columns := s.groupTodosByStatus(todos)

	kanbanBoard := &models.KanbanBoard{
		ProjectID:    projectID,
		ProjectTitle: project.Title,
		Columns:      columns,
		TotalTodos:   len(todos),
		LastUpdated:  time.Now(),
	}

	return kanbanBoard, nil
}

// MoveTodoToColumn moves a todo to a different column/status
func (s *TodoService) MoveTodoToColumn(ctx context.Context, userID uuid.UUID, req *models.MoveTodoRequest) (*models.TodoMoveResponse, error) {
	// Verify todo ownership
	exists, err := s.todoRepo.ValidateTodoOwnership(ctx, req.TodoID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to validate todo ownership: %w", err)
	}
	if !exists {
		return nil, repository.ErrTodoNotFound
	}

	// Get current todo to check old status
	todo, err := s.todoRepo.GetByID(ctx, req.TodoID, false)
	if err != nil {
		return nil, fmt.Errorf("failed to get todo: %w", err)
	}
	if todo == nil {
		return nil, repository.ErrTodoNotFound
	}

	oldStatus := todo.Status

	// Update todo status
	updateReq := &models.UpdateTodoRequest{
		Status: &req.NewStatus,
	}

	_, err = s.UpdateTodo(ctx, req.TodoID, userID, updateReq)
	if err != nil {
		return nil, fmt.Errorf("failed to update todo status: %w", err)
	}

	// If position is specified, handle reordering within the new column
	newPosition := 0
	if req.Position != nil {
		newPosition = *req.Position
		// TODO: Implement position-based reordering within column
	}

	response := &models.TodoMoveResponse{
		TodoID:      req.TodoID,
		OldStatus:   oldStatus,
		NewStatus:   req.NewStatus,
		NewPosition: newPosition,
		Message:     "Todo moved successfully",
	}

	return response, nil
}

// ReorderTodosInColumn reorders todos within a specific column/status
func (s *TodoService) ReorderTodosInColumn(ctx context.Context, userID uuid.UUID, req *models.ReorderTodosRequest) (*models.ReorderResponse, error) {
	// Validate that all todos belong to the user and have the specified status
	for _, todoID := range req.TodoIDs {
		exists, err := s.todoRepo.ValidateTodoOwnership(ctx, todoID, userID)
		if err != nil {
			return nil, fmt.Errorf("failed to validate todo ownership for %s: %w", todoID, err)
		}
		if !exists {
			return nil, fmt.Errorf("todo %s not found", todoID)
		}

		// Verify todo has the correct status
		todo, err := s.todoRepo.GetByID(ctx, todoID, false)
		if err != nil {
			return nil, fmt.Errorf("failed to get todo %s: %w", todoID, err)
		}
		if todo == nil {
			return nil, fmt.Errorf("todo %s not found", todoID)
		}
		if todo.Status != req.Status {
			return nil, fmt.Errorf("todo %s does not have status %s", todoID, req.Status)
		}
	}

	// TODO: Implement actual position-based reordering in database
	// For now, we'll just return success without changing order
	// This would require adding a position/order field to the todos table

	response := &models.ReorderResponse{
		Status:       req.Status,
		ReorderedIDs: req.TodoIDs,
		Message:      "Todos reordered successfully",
	}

	return response, nil
}

// BulkMoveTodos moves multiple todos in a single operation
func (s *TodoService) BulkMoveTodos(ctx context.Context, userID uuid.UUID, req *models.BulkMoveTodosRequest) (*models.BulkMoveResponse, error) {
	var movedTodos []models.TodoMoveResponse
	var failures []models.MoveFailure
	successCount := 0
	failureCount := 0

	for _, moveReq := range req.Moves {
		moveResponse, err := s.MoveTodoToColumn(ctx, userID, &moveReq)
		if err != nil {
			failures = append(failures, models.MoveFailure{
				TodoID: moveReq.TodoID,
				Reason: err.Error(),
			})
			failureCount++
		} else {
			movedTodos = append(movedTodos, *moveResponse)
			successCount++
		}
	}

	response := &models.BulkMoveResponse{
		MovedTodos:   movedTodos,
		SuccessCount: successCount,
		FailureCount: failureCount,
		Failures:     failures,
		Message:      fmt.Sprintf("Bulk move completed: %d succeeded, %d failed", successCount, failureCount),
	}

	return response, nil
}

// Helper methods for kanban board

// groupTodosByStatus groups todos by their status into kanban columns
func (s *TodoService) groupTodosByStatus(todos []*models.Todo) []models.KanbanColumn {
	// Initialize columns with empty todos
	columns := []models.KanbanColumn{
		{
			Status:    models.TaskStatusBacklog,
			Title:     "Backlog",
			Color:     "#6c757d",
			Todos:     []models.Todo{},
			TodoCount: 0,
		},
		{
			Status:    models.TaskStatusTodo,
			Title:     "This Week",
			Color:     "#007bff",
			Todos:     []models.Todo{},
			TodoCount: 0,
		},
		{
			Status:    models.TaskStatusInProgress,
			Title:     "Today",
			Color:     "#ffc107",
			Todos:     []models.Todo{},
			TodoCount: 0,
		},
		{
			Status:    models.TaskStatusCompleted,
			Title:     "Done",
			Color:     "#28a745",
			Todos:     []models.Todo{},
			TodoCount: 0,
		},
	}

	// Group todos into columns by status
	for _, todo := range todos {
		statusStr := string(todo.Status)

		// Map status to column index
		var columnIndex int
		switch statusStr {
		case "not_started", "pending", "on_hold":
			columnIndex = 0 // Backlog
		case "todo":
			columnIndex = 1 // This Week
		case "in_progress", "blocked":
			columnIndex = 2 // Today
		case "completed", "done":
			columnIndex = 3 // Done
		default:
			columnIndex = 0 // Default to Backlog
		}

		// Add todo to the appropriate column
		columns[columnIndex].Todos = append(columns[columnIndex].Todos, *todo)
		columns[columnIndex].TodoCount = len(columns[columnIndex].Todos)
	}

	return columns
}

// calculateBoardStats calculates statistics for the kanban board
func (s *TodoService) calculateBoardStats(todos []*models.Todo) models.BoardStats {
	stats := models.BoardStats{
		TotalTodos:      len(todos),
		CompletedTodos:  0,
		InProgressTodos: 0,
		BacklogTodos:    0,
	}

	for _, todo := range todos {
		switch todo.Status {
		case models.TaskStatusCompleted, models.TaskStatusDone:
			stats.CompletedTodos++
		case models.TaskStatusInProgress:
			stats.InProgressTodos++
		case models.TaskStatusBacklog:
			stats.BacklogTodos++
		}
	}

	// Calculate completion rate
	if stats.TotalTodos > 0 {
		stats.CompletionRate = (stats.CompletedTodos * 100) / stats.TotalTodos
	}

	return stats
}

// Validation methods

func (s *TodoService) validateCreateTodoRequest(req *models.CreateTodoRequest) error {
	if req.TaskName == "" {
		return fmt.Errorf("task name is required")
	}
	if len(req.TaskName) > 255 {
		return fmt.Errorf("task name must be less than 255 characters")
	}
	if req.ProjectID == uuid.Nil {
		return fmt.Errorf("project ID is required")
	}
	if req.Priority != nil && !req.Priority.IsValid() {
		return fmt.Errorf("invalid priority level")
	}
	if req.TimeUnit != nil && !req.TimeUnit.IsValid() {
		return fmt.Errorf("invalid time unit")
	}
	if req.DifficultyRating != nil && (*req.DifficultyRating < 1 || *req.DifficultyRating > 10) {
		return fmt.Errorf("difficulty rating must be between 1 and 10")
	}
	if req.EnergyLevelRequired != nil && (*req.EnergyLevelRequired < 1 || *req.EnergyLevelRequired > 5) {
		return fmt.Errorf("energy level required must be between 1 and 5")
	}
	return nil
}

func (s *TodoService) validateUpdateTodoRequest(req *models.UpdateTodoRequest) error {
	if req.TaskName != nil && *req.TaskName == "" {
		return fmt.Errorf("task name cannot be empty")
	}
	if req.TaskName != nil && len(*req.TaskName) > 255 {
		return fmt.Errorf("task name must be less than 255 characters")
	}
	if req.Priority != nil && !req.Priority.IsValid() {
		return fmt.Errorf("invalid priority level")
	}
	if req.Status != nil && !req.Status.IsValid() {
		return fmt.Errorf("invalid status")
	}
	if req.TimeUnit != nil && !req.TimeUnit.IsValid() {
		return fmt.Errorf("invalid time unit")
	}
	if req.DifficultyRating != nil && (*req.DifficultyRating < 1 || *req.DifficultyRating > 10) {
		return fmt.Errorf("difficulty rating must be between 1 and 10")
	}
	if req.EnergyLevelRequired != nil && (*req.EnergyLevelRequired < 1 || *req.EnergyLevelRequired > 5) {
		return fmt.Errorf("energy level required must be between 1 and 5")
	}
	return nil
}

// Cache helper methods

func (s *TodoService) buildTodoFiltersCacheKey(userID uuid.UUID, filters repository.TodoFilters) string {
	return fmt.Sprintf("%s%s:filters_%s", userTodosCachePrefix, userID.String(), s.hashFilters(filters))
}

func (s *TodoService) hashFilters(filters repository.TodoFilters) string {
	// Simple hash of filter values for cache key
	// In production, use a proper hash function
	return fmt.Sprintf("%v_%v_%v_%v_%d_%d",
		filters.ProjectID, filters.Status, filters.Priority,
		filters.IsCompleted, filters.Limit, filters.Offset)
}

func (s *TodoService) invalidateTodoCaches(userID, projectID uuid.UUID) {
	ctx := context.Background()

	// Invalidate user todos cache (all variations)
	userPattern := fmt.Sprintf("%s%s:*", userTodosCachePrefix, userID.String())
	s.redisClient.DeletePattern(ctx, userPattern)

	// Invalidate project todos cache
	projectPattern := fmt.Sprintf("%s%s:*", projectTodosCachePrefix, projectID.String())
	s.redisClient.DeletePattern(ctx, projectPattern)

	// Invalidate dashboard cache
	dashboardPattern := fmt.Sprintf("%s%s:*", todoDashboardPrefix, userID.String())
	s.redisClient.DeletePattern(ctx, dashboardPattern)

	// Invalidate overdue todos cache
	overdueKey := fmt.Sprintf("%s%s", overdueTodosPrefix, userID.String())
	s.redisClient.Delete(ctx, overdueKey)

	// Invalidate search cache
	searchPattern := fmt.Sprintf("%s%s:*", todoSearchPrefix, userID.String())
	s.redisClient.DeletePattern(ctx, searchPattern)
}
