package service

import (
	"context"
	"fmt"
	"time"

	"macwrite-auth-api/internal/models"
	"macwrite-auth-api/internal/repository"

	"github.com/google/uuid"
)

// BreakService defines the interface for break-related business logic
type BreakService interface {
	StartBreak(ctx context.Context, userID uuid.UUID, req *models.StartBreakRequest) (*models.BreakHistory, error)
	StopBreak(ctx context.Context, userID uuid.UUID, req *models.StopBreakRequest) (*models.BreakHistory, error)
	GetActiveBreak(ctx context.Context, userID uuid.UUID) (*models.BreakHistory, error)
	GetBreakHistory(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*models.BreakHistory, error)
	GetBreaksByTodo(ctx context.Context, todoID uuid.UUID, limit, offset int) ([]*models.BreakHistory, error)
	GetBreaksByProject(ctx context.Context, projectID uuid.UUID, limit, offset int) ([]*models.BreakHistory, error)
	GetBreakStats(ctx context.Context, userID uuid.UUID) (*models.BreakStatsResponse, error)
	EnsureBreakTableExists(ctx context.Context) error
}

// breakService is the concrete implementation
type breakService struct {
	breakRepo   *repository.BreakHistoryRepository
	todoRepo    *repository.TodoRepository
	projectRepo *repository.ProjectRepository
}

// NewBreakService creates a new break service
func NewBreakService(
	breakRepo *repository.BreakHistoryRepository,
	todoRepo *repository.TodoRepository,
	projectRepo *repository.ProjectRepository,
) BreakService {
	return &breakService{
		breakRepo:   breakRepo,
		todoRepo:    todoRepo,
		projectRepo: projectRepo,
	}
}

// EnsureBreakTableExists ensures the break history table exists
func (s *breakService) EnsureBreakTableExists(ctx context.Context) error {
	return s.breakRepo.EnsureTableExists(ctx)
}

// StartBreak starts a new break session
func (s *breakService) StartBreak(ctx context.Context, userID uuid.UUID, req *models.StartBreakRequest) (*models.BreakHistory, error) {
	// Validate that the todo exists and belongs to the user
	todo, err := s.todoRepo.GetByID(ctx, req.TodoID, false) // includeRelated = false
	if err != nil {
		return nil, fmt.Errorf("failed to get todo: %w", err)
	}
	if todo == nil {
		return nil, fmt.Errorf("todo not found")
	}
	if todo.UserID != userID {
		return nil, fmt.Errorf("todo does not belong to user")
	}

	// Validate that the project exists and belongs to the user
	project, err := s.projectRepo.GetByID(ctx, req.ProjectID, false) // includeCounts = false
	if err != nil {
		return nil, fmt.Errorf("failed to get project: %w", err)
	}
	if project == nil {
		return nil, fmt.Errorf("project not found")
	}
	if project.UserID != userID {
		return nil, fmt.Errorf("project does not belong to user")
	}

	// Check if user already has an active break
	activeBreak, err := s.breakRepo.GetActiveBreakByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to check for active break: %w", err)
	}
	if activeBreak != nil {
		return nil, fmt.Errorf("user already has an active break session")
	}

	// Create new break session
	breakHistory := &models.BreakHistory{
		ID:        uuid.New(),
		TodoID:    req.TodoID,
		ProjectID: req.ProjectID,
		UserID:    userID,
		StartTime: time.Now(),
		BreakType: req.BreakType,
		Notes:     req.Notes,
		Duration:  0, // Will be calculated when stopped
	}

	if breakHistory.BreakType == "" {
		breakHistory.BreakType = "manual"
	}

	// Ensure table exists before creating
	if err := s.breakRepo.EnsureTableExists(ctx); err != nil {
		return nil, fmt.Errorf("failed to ensure break table exists: %w", err)
	}

	if err := s.breakRepo.Create(ctx, breakHistory); err != nil {
		return nil, fmt.Errorf("failed to create break session: %w", err)
	}

	// Load relationships for response
	createdBreak, err := s.breakRepo.GetByID(ctx, breakHistory.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get created break: %w", err)
	}

	return createdBreak, nil
}

// StopBreak stops the current active break session
func (s *breakService) StopBreak(ctx context.Context, userID uuid.UUID, req *models.StopBreakRequest) (*models.BreakHistory, error) {
	// Get the active break
	activeBreak, err := s.breakRepo.GetActiveBreakByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get active break: %w", err)
	}
	if activeBreak == nil {
		return nil, fmt.Errorf("no active break session found")
	}

	// Calculate duration and set end time
	now := time.Now()
	duration := int(now.Sub(activeBreak.StartTime).Seconds())

	activeBreak.EndTime = &now
	activeBreak.Duration = duration

	// Update notes if provided
	if req.Notes != "" {
		if activeBreak.Notes != "" {
			activeBreak.Notes += "\n" + req.Notes
		} else {
			activeBreak.Notes = req.Notes
		}
	}

	if err := s.breakRepo.Update(ctx, activeBreak); err != nil {
		return nil, fmt.Errorf("failed to update break session: %w", err)
	}

	// Return updated break with relationships
	updatedBreak, err := s.breakRepo.GetByID(ctx, activeBreak.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get updated break: %w", err)
	}

	return updatedBreak, nil
}

// GetActiveBreak gets the currently active break for a user
func (s *breakService) GetActiveBreak(ctx context.Context, userID uuid.UUID) (*models.BreakHistory, error) {
	return s.breakRepo.GetActiveBreakByUserID(ctx, userID)
}

// GetBreakHistory gets break history for a user with pagination
func (s *breakService) GetBreakHistory(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*models.BreakHistory, error) {
	return s.breakRepo.GetByUserID(ctx, userID, limit, offset)
}

// GetBreaksByTodo gets break history for a specific todo
func (s *breakService) GetBreaksByTodo(ctx context.Context, todoID uuid.UUID, limit, offset int) ([]*models.BreakHistory, error) {
	return s.breakRepo.GetByTodoID(ctx, todoID, limit, offset)
}

// GetBreaksByProject gets break history for a specific project
func (s *breakService) GetBreaksByProject(ctx context.Context, projectID uuid.UUID, limit, offset int) ([]*models.BreakHistory, error) {
	return s.breakRepo.GetByProjectID(ctx, projectID, limit, offset)
}

// GetBreakStats gets break statistics for a user
func (s *breakService) GetBreakStats(ctx context.Context, userID uuid.UUID) (*models.BreakStatsResponse, error) {
	return s.breakRepo.GetStats(ctx, userID)
}
