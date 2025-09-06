package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"quantumtask-auth-api/internal/models"
	"quantumtask-auth-api/internal/repository"
	"quantumtask-auth-api/pkg/redis"

	"github.com/google/uuid"
)

// ========================================
// SUBTASK SERVICE
// ========================================

// SubtaskService handles subtask business logic
type SubtaskService struct {
	subtaskRepo *repository.SubtaskRepository
	todoRepo    *repository.TodoRepository
	projectRepo *repository.ProjectRepository
	redisClient redis.Client
}

// NewSubtaskService creates a new subtask service
func NewSubtaskService(
	subtaskRepo *repository.SubtaskRepository,
	todoRepo *repository.TodoRepository,
	projectRepo *repository.ProjectRepository,
	redisClient redis.Client,
) *SubtaskService {
	return &SubtaskService{
		subtaskRepo: subtaskRepo,
		todoRepo:    todoRepo,
		projectRepo: projectRepo,
		redisClient: redisClient,
	}
}

const (
	subtaskCachePrefix = "subtasks:"
	subtaskCacheTTL    = 20 * time.Minute
)

// CreateSubtask creates a new subtask
func (s *SubtaskService) CreateSubtask(ctx context.Context, userID uuid.UUID, subtask *models.Subtask) (*models.Subtask, error) {
	// Verify todo ownership
	exists, err := s.todoRepo.ValidateTodoOwnership(ctx, subtask.TodoID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to validate todo ownership: %w", err)
	}
	if !exists {
		return nil, fmt.Errorf("todo not found")
	}

	// Validate subtask
	if err := s.validateSubtask(subtask); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Set sort order if not provided
	if subtask.SortOrder == 0 {
		// Get count of existing subtasks to set sort order
		existing, err := s.subtaskRepo.GetByTodoID(ctx, subtask.TodoID, false)
		if err != nil {
			return nil, fmt.Errorf("failed to get existing subtasks: %w", err)
		}
		subtask.SortOrder = len(existing)
	}

	// Create subtask
	if err := s.subtaskRepo.Create(ctx, subtask); err != nil {
		return nil, fmt.Errorf("failed to create subtask: %w", err)
	}

	// Invalidate cache
	s.invalidateSubtaskCache(subtask.TodoID)

	return subtask, nil
}

// GetSubtasksByTodoID retrieves subtasks for a todo
func (s *SubtaskService) GetSubtasksByTodoID(ctx context.Context, todoID, userID uuid.UUID, includeArchived bool) ([]*models.Subtask, error) {
	// Verify todo ownership
	exists, err := s.todoRepo.ValidateTodoOwnership(ctx, todoID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to validate todo ownership: %w", err)
	}
	if !exists {
		return nil, fmt.Errorf("todo not found")
	}

	// Check cache
	cacheKey := fmt.Sprintf("%s%s:archived_%t", subtaskCachePrefix, todoID.String(), includeArchived)
	if cached, err := s.redisClient.Get(ctx, cacheKey); err == nil {
		var subtasks []*models.Subtask
		if err := json.Unmarshal(cached, &subtasks); err == nil {
			return subtasks, nil
		}
	}

	// Get from database
	subtasks, err := s.subtaskRepo.GetByTodoID(ctx, todoID, includeArchived)
	if err != nil {
		return nil, fmt.Errorf("failed to get subtasks: %w", err)
	}

	// Cache result
	if subtasksData, err := json.Marshal(subtasks); err == nil {
		s.redisClient.Set(ctx, cacheKey, subtasksData, subtaskCacheTTL)
	}

	return subtasks, nil
}

// GetSubtasksByProjectIDRequest represents the request for getting subtasks by project ID
type GetSubtasksByProjectIDRequest struct {
	ProjectID       uuid.UUID `json:"project_id" validate:"required"`
	IncludeArchived bool      `json:"include_archived"`
	Status          string    `json:"status,omitempty"`
	Priority        string    `json:"priority,omitempty"`
	Search          string    `json:"search,omitempty"`
	SortBy          string    `json:"sort_by,omitempty"`
	SortOrder       string    `json:"sort_order,omitempty"`
	Page            int       `json:"page" validate:"min=1"`
	Limit           int       `json:"limit" validate:"min=1,max=100"`
}

// GetSubtasksByProjectIDResponse represents the response for getting subtasks by project ID
type GetSubtasksByProjectIDResponse struct {
	Subtasks   []*repository.SubtaskWithTodo `json:"subtasks"`
	Pagination models.PaginationMeta         `json:"pagination"`
}

// GetSubtasksByProjectID retrieves all subtasks for a project with filtering and pagination
func (s *SubtaskService) GetSubtasksByProjectID(ctx context.Context, userID uuid.UUID, req *GetSubtasksByProjectIDRequest) (*GetSubtasksByProjectIDResponse, error) {
	// Validate request
	if err := s.validateGetSubtasksByProjectIDRequest(req); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Check if project exists and user has access (simplified validation)
	project, err := s.projectRepo.GetByID(ctx, req.ProjectID, false)
	if err != nil {
		return nil, fmt.Errorf("failed to validate project access: %w", err)
	}
	if project == nil {
		return nil, fmt.Errorf("project not found")
	}

	// Set default pagination values
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.Limit <= 0 {
		req.Limit = 20
	}

	// Calculate offset
	offset := (req.Page - 1) * req.Limit

	// Build query options
	options := repository.SubtaskQueryOptions{
		IncludeArchived: req.IncludeArchived,
		Status:          req.Status,
		Priority:        req.Priority,
		Search:          req.Search,
		SortBy:          req.SortBy,
		SortOrder:       req.SortOrder,
		Limit:           req.Limit,
		Offset:          offset,
	}

	// Get subtasks from repository
	subtasks, totalCount, err := s.subtaskRepo.GetByProjectIDWithPagination(ctx, req.ProjectID, options)
	if err != nil {
		return nil, fmt.Errorf("failed to get subtasks: %w", err)
	}

	// Calculate pagination info
	totalPages := (totalCount + req.Limit - 1) / req.Limit
	hasNext := req.Page < totalPages
	hasPrev := req.Page > 1

	var nextPage, prevPage *int
	if hasNext {
		next := req.Page + 1
		nextPage = &next
	}
	if hasPrev {
		prev := req.Page - 1
		prevPage = &prev
	}

	pagination := models.PaginationMeta{
		Page:        req.Page,
		PageSize:    req.Limit,
		TotalCount:  totalCount,
		TotalPages:  totalPages,
		HasNextPage: hasNext,
		HasPrevPage: hasPrev,
		NextPage:    nextPage,
		PrevPage:    prevPage,
	}

	return &GetSubtasksByProjectIDResponse{
		Subtasks:   subtasks,
		Pagination: pagination,
	}, nil
}

// validateGetSubtasksByProjectIDRequest validates the request for getting subtasks by project ID
func (s *SubtaskService) validateGetSubtasksByProjectIDRequest(req *GetSubtasksByProjectIDRequest) error {
	if req.ProjectID == uuid.Nil {
		return fmt.Errorf("project_id is required")
	}

	// Validate status if provided
	if req.Status != "" && !models.IsValidTaskStatus(req.Status) {
		return fmt.Errorf("invalid status: %s", req.Status)
	}

	// Validate priority if provided
	if req.Priority != "" {
		priority := models.PriorityLevel(req.Priority)
		if !priority.IsValid() {
			return fmt.Errorf("invalid priority: %s", req.Priority)
		}
	}

	// Validate sort order if provided
	if req.SortOrder != "" {
		sortOrder := strings.ToUpper(req.SortOrder)
		if sortOrder != "ASC" && sortOrder != "DESC" {
			return fmt.Errorf("invalid sort_order: %s. Must be ASC or DESC", req.SortOrder)
		}
	}

	// Validate sort by if provided
	if req.SortBy != "" {
		validSortFields := []string{"name", "status", "priority", "due_date", "todo_title", "sort_order", "created_at"}
		isValid := false
		for _, field := range validSortFields {
			if req.SortBy == field {
				isValid = true
				break
			}
		}
		if !isValid {
			return fmt.Errorf("invalid sort_by: %s. Valid fields are: %v", req.SortBy, validSortFields)
		}
	}

	return nil
}

// UpdateSubtask updates a subtask
func (s *SubtaskService) UpdateSubtask(ctx context.Context, subtaskID, userID uuid.UUID, updates map[string]interface{}) (*models.Subtask, error) {
	// Get current subtask to verify ownership
	subtask, err := s.subtaskRepo.GetByID(ctx, subtaskID)
	if err != nil {
		return nil, fmt.Errorf("failed to get subtask: %w", err)
	}
	if subtask == nil {
		return nil, fmt.Errorf("subtask not found")
	}

	// Verify todo ownership
	exists, err := s.todoRepo.ValidateTodoOwnership(ctx, subtask.TodoID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to validate todo ownership: %w", err)
	}
	if !exists {
		return nil, fmt.Errorf("todo not found")
	}

	// Update subtask
	if err := s.subtaskRepo.Update(ctx, subtaskID, updates); err != nil {
		return nil, fmt.Errorf("failed to update subtask: %w", err)
	}

	// Invalidate cache
	s.invalidateSubtaskCache(subtask.TodoID)

	// Return updated subtask
	return s.subtaskRepo.GetByID(ctx, subtaskID)
}

// DeleteSubtask deletes a subtask
func (s *SubtaskService) DeleteSubtask(ctx context.Context, subtaskID, userID uuid.UUID) error {
	// Get current subtask to verify ownership
	subtask, err := s.subtaskRepo.GetByID(ctx, subtaskID)
	if err != nil {
		return fmt.Errorf("failed to get subtask: %w", err)
	}
	if subtask == nil {
		return fmt.Errorf("subtask not found")
	}

	// Verify todo ownership
	exists, err := s.todoRepo.ValidateTodoOwnership(ctx, subtask.TodoID, userID)
	if err != nil {
		return fmt.Errorf("failed to validate todo ownership: %w", err)
	}
	if !exists {
		return fmt.Errorf("todo not found")
	}

	// Delete subtask
	if err := s.subtaskRepo.Delete(ctx, subtaskID); err != nil {
		return fmt.Errorf("failed to delete subtask: %w", err)
	}

	// Invalidate cache
	s.invalidateSubtaskCache(subtask.TodoID)

	return nil
}

// ReorderSubtasks reorders subtasks for a todo
func (s *SubtaskService) ReorderSubtasks(ctx context.Context, todoID, userID uuid.UUID, subtaskIDs []uuid.UUID) error {
	// Verify todo ownership
	exists, err := s.todoRepo.ValidateTodoOwnership(ctx, todoID, userID)
	if err != nil {
		return fmt.Errorf("failed to validate todo ownership: %w", err)
	}
	if !exists {
		return fmt.Errorf("todo not found")
	}

	// Reorder subtasks
	if err := s.subtaskRepo.ReorderSubtasks(ctx, todoID, subtaskIDs); err != nil {
		return fmt.Errorf("failed to reorder subtasks: %w", err)
	}

	// Invalidate cache
	s.invalidateSubtaskCache(todoID)

	return nil
}

func (s *SubtaskService) validateSubtask(subtask *models.Subtask) error {
	if subtask.Name == "" {
		return fmt.Errorf("subtask name is required")
	}
	if len(subtask.Name) > 255 {
		return fmt.Errorf("subtask name must be less than 255 characters")
	}
	return nil
}

func (s *SubtaskService) invalidateSubtaskCache(todoID uuid.UUID) {
	ctx := context.Background()
	pattern := fmt.Sprintf("%s%s:*", subtaskCachePrefix, todoID.String())
	s.redisClient.DeletePattern(ctx, pattern)
}

// ========================================
// NOTE SERVICE
// ========================================

// NoteService handles note business logic
type NoteService struct {
	noteRepo    *repository.NoteRepository
	todoRepo    *repository.TodoRepository
	redisClient redis.Client
}

// NewNoteService creates a new note service
func NewNoteService(
	noteRepo *repository.NoteRepository,
	todoRepo *repository.TodoRepository,
	redisClient redis.Client,
) *NoteService {
	return &NoteService{
		noteRepo:    noteRepo,
		todoRepo:    todoRepo,
		redisClient: redisClient,
	}
}

const (
	noteCachePrefix = "notes:"
	noteCacheTTL    = 30 * time.Minute
)

// CreateNote creates a new note
func (s *NoteService) CreateNote(ctx context.Context, userID uuid.UUID, note *models.Note) (*models.Note, error) {
	// Verify todo ownership
	exists, err := s.todoRepo.ValidateTodoOwnership(ctx, note.TodoID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to validate todo ownership: %w", err)
	}
	if !exists {
		return nil, fmt.Errorf("todo not found")
	}

	// Validate note
	if err := s.validateNote(note); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Create note
	if err := s.noteRepo.Create(ctx, note); err != nil {
		return nil, fmt.Errorf("failed to create note: %w", err)
	}

	// Invalidate cache
	s.invalidateNoteCache(note.TodoID)

	return note, nil
}

// GetNotesByTodoID retrieves notes for a todo
func (s *NoteService) GetNotesByTodoID(ctx context.Context, todoID, userID uuid.UUID) ([]*models.Note, error) {
	// Verify todo ownership
	exists, err := s.todoRepo.ValidateTodoOwnership(ctx, todoID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to validate todo ownership: %w", err)
	}
	if !exists {
		return nil, fmt.Errorf("todo not found")
	}

	// Check cache
	cacheKey := fmt.Sprintf("%s%s", noteCachePrefix, todoID.String())
	if cached, err := s.redisClient.Get(ctx, cacheKey); err == nil {
		var notes []*models.Note
		if err := json.Unmarshal(cached, &notes); err == nil {
			return notes, nil
		}
	}

	// Get from database
	notes, err := s.noteRepo.GetByTodoID(ctx, todoID)
	if err != nil {
		return nil, fmt.Errorf("failed to get notes: %w", err)
	}

	// Cache result
	if notesData, err := json.Marshal(notes); err == nil {
		s.redisClient.Set(ctx, cacheKey, notesData, noteCacheTTL)
	}

	return notes, nil
}

// UpdateNote updates a note
func (s *NoteService) UpdateNote(ctx context.Context, noteID, userID uuid.UUID, updates map[string]interface{}) (*models.Note, error) {
	// Get current note to verify ownership
	note, err := s.noteRepo.GetByID(ctx, noteID)
	if err != nil {
		return nil, fmt.Errorf("failed to get note: %w", err)
	}
	if note == nil {
		return nil, fmt.Errorf("note not found")
	}

	// Verify todo ownership
	exists, err := s.todoRepo.ValidateTodoOwnership(ctx, note.TodoID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to validate todo ownership: %w", err)
	}
	if !exists {
		return nil, fmt.Errorf("todo not found")
	}

	// Update note
	if err := s.noteRepo.Update(ctx, noteID, updates); err != nil {
		return nil, fmt.Errorf("failed to update note: %w", err)
	}

	// Invalidate cache
	s.invalidateNoteCache(note.TodoID)

	// Return updated note
	return s.noteRepo.GetByID(ctx, noteID)
}

// DeleteNote deletes a note
func (s *NoteService) DeleteNote(ctx context.Context, noteID, userID uuid.UUID) error {
	// Get current note to verify ownership
	note, err := s.noteRepo.GetByID(ctx, noteID)
	if err != nil {
		return fmt.Errorf("failed to get note: %w", err)
	}
	if note == nil {
		return fmt.Errorf("note not found")
	}

	// Verify todo ownership
	exists, err := s.todoRepo.ValidateTodoOwnership(ctx, note.TodoID, userID)
	if err != nil {
		return fmt.Errorf("failed to validate todo ownership: %w", err)
	}
	if !exists {
		return fmt.Errorf("todo not found")
	}

	// Delete note
	if err := s.noteRepo.Delete(ctx, noteID); err != nil {
		return fmt.Errorf("failed to delete note: %w", err)
	}

	// Invalidate cache
	s.invalidateNoteCache(note.TodoID)

	return nil
}

// SearchNotes searches notes by content
func (s *NoteService) SearchNotes(ctx context.Context, todoID, userID uuid.UUID, searchTerm string) ([]*models.Note, error) {
	// Verify todo ownership
	exists, err := s.todoRepo.ValidateTodoOwnership(ctx, todoID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to validate todo ownership: %w", err)
	}
	if !exists {
		return nil, fmt.Errorf("todo not found")
	}

	// Search notes
	notes, err := s.noteRepo.SearchNotes(ctx, todoID, searchTerm)
	if err != nil {
		return nil, fmt.Errorf("failed to search notes: %w", err)
	}

	return notes, nil
}

func (s *NoteService) validateNote(note *models.Note) error {
	if note.Content == "" {
		return fmt.Errorf("note content is required")
	}
	return nil
}

func (s *NoteService) invalidateNoteCache(todoID uuid.UUID) {
	ctx := context.Background()
	cacheKey := fmt.Sprintf("%s%s", noteCachePrefix, todoID.String())
	s.redisClient.Delete(ctx, cacheKey)
}

// ========================================
// SCHEDULED TASK SERVICE
// ========================================

// ScheduledTaskService handles scheduled task business logic
type ScheduledTaskService struct {
	scheduledRepo *repository.ScheduledTaskRepository
	todoRepo      *repository.TodoRepository
	redisClient   redis.Client
}

// NewScheduledTaskService creates a new scheduled task service
func NewScheduledTaskService(
	scheduledRepo *repository.ScheduledTaskRepository,
	todoRepo *repository.TodoRepository,
	redisClient redis.Client,
) *ScheduledTaskService {
	return &ScheduledTaskService{
		scheduledRepo: scheduledRepo,
		todoRepo:      todoRepo,
		redisClient:   redisClient,
	}
}

const (
	scheduledCachePrefix = "scheduled:"
	scheduledCacheTTL    = 10 * time.Minute
)

// CreateScheduledTask creates a new scheduled task
func (s *ScheduledTaskService) CreateScheduledTask(ctx context.Context, userID uuid.UUID, task *models.ScheduledTask) (*models.ScheduledTask, error) {
	// Verify todo ownership
	exists, err := s.todoRepo.ValidateTodoOwnership(ctx, task.TodoID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to validate todo ownership: %w", err)
	}
	if !exists {
		return nil, fmt.Errorf("todo not found")
	}

	// Set user ID
	task.UserID = userID

	// Validate task
	if err := s.validateScheduledTask(task); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Create scheduled task
	if err := s.scheduledRepo.Create(ctx, task); err != nil {
		return nil, fmt.Errorf("failed to create scheduled task: %w", err)
	}

	// Invalidate cache
	s.invalidateScheduledCache(userID)

	return task, nil
}

// GetUserScheduledTasks retrieves scheduled tasks for a user
func (s *ScheduledTaskService) GetUserScheduledTasks(ctx context.Context, userID uuid.UUID, fromDate, toDate *string) ([]*models.ScheduledTask, error) {
	// Check cache
	cacheKey := fmt.Sprintf("%s%s:from_%v:to_%v", scheduledCachePrefix, userID.String(), fromDate, toDate)
	if cached, err := s.redisClient.Get(ctx, cacheKey); err == nil {
		var tasks []*models.ScheduledTask
		if err := json.Unmarshal(cached, &tasks); err == nil {
			return tasks, nil
		}
	}

	// Get from database
	tasks, err := s.scheduledRepo.GetByUserID(ctx, userID, fromDate, toDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get scheduled tasks: %w", err)
	}

	// Cache result
	if tasksData, err := json.Marshal(tasks); err == nil {
		s.redisClient.Set(ctx, cacheKey, tasksData, scheduledCacheTTL)
	}

	return tasks, nil
}

// UpdateScheduledTask updates a scheduled task
func (s *ScheduledTaskService) UpdateScheduledTask(ctx context.Context, taskID, userID uuid.UUID, updates map[string]interface{}) (*models.ScheduledTask, error) {
	// Get current task to verify ownership
	task, err := s.scheduledRepo.GetByID(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to get scheduled task: %w", err)
	}
	if task == nil {
		return nil, fmt.Errorf("scheduled task not found")
	}

	// Verify ownership
	if task.UserID != userID {
		return nil, fmt.Errorf("scheduled task not found")
	}

	// Update task
	if err := s.scheduledRepo.Update(ctx, taskID, updates); err != nil {
		return nil, fmt.Errorf("failed to update scheduled task: %w", err)
	}

	// Invalidate cache
	s.invalidateScheduledCache(userID)

	// Return updated task
	return s.scheduledRepo.GetByID(ctx, taskID)
}

// DeleteScheduledTask deletes a scheduled task
func (s *ScheduledTaskService) DeleteScheduledTask(ctx context.Context, taskID, userID uuid.UUID) error {
	// Get current task to verify ownership
	task, err := s.scheduledRepo.GetByID(ctx, taskID)
	if err != nil {
		return fmt.Errorf("failed to get scheduled task: %w", err)
	}
	if task == nil {
		return fmt.Errorf("scheduled task not found")
	}

	// Verify ownership
	if task.UserID != userID {
		return fmt.Errorf("scheduled task not found")
	}

	// Delete task
	if err := s.scheduledRepo.Delete(ctx, taskID); err != nil {
		return fmt.Errorf("failed to delete scheduled task: %w", err)
	}

	// Invalidate cache
	s.invalidateScheduledCache(userID)

	return nil
}

// GetPendingNotifications retrieves tasks that need notifications
func (s *ScheduledTaskService) GetPendingNotifications(ctx context.Context) ([]*models.ScheduledTask, error) {
	return s.scheduledRepo.GetPendingNotifications(ctx)
}

// MarkNotificationSent marks a notification as sent
func (s *ScheduledTaskService) MarkNotificationSent(ctx context.Context, taskID uuid.UUID) error {
	return s.scheduledRepo.MarkNotificationSent(ctx, taskID)
}

func (s *ScheduledTaskService) validateScheduledTask(task *models.ScheduledTask) error {
	if task.ScheduledDatetime.IsZero() {
		return fmt.Errorf("scheduled datetime is required")
	}
	if task.ScheduledDatetime.Before(time.Now()) {
		return fmt.Errorf("scheduled datetime cannot be in the past")
	}
	return nil
}

func (s *ScheduledTaskService) invalidateScheduledCache(userID uuid.UUID) {
	ctx := context.Background()
	pattern := fmt.Sprintf("%s%s:*", scheduledCachePrefix, userID.String())
	s.redisClient.DeletePattern(ctx, pattern)
}

// CreateRecurringSchedule creates a parent recurring schedule and optionally generates initial child instances
func (s *ScheduledTaskService) CreateRecurringSchedule(ctx context.Context, userID uuid.UUID, task *models.ScheduledTask, generateInitialInstances bool) (*models.ScheduledTask, error) {
	// Verify todo ownership
	exists, err := s.todoRepo.ValidateTodoOwnership(ctx, task.TodoID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to validate todo ownership: %w", err)
	}
	if !exists {
		return nil, fmt.Errorf("todo not found")
	}

	// Set user ID and parent schedule properties
	task.UserID = userID
	task.IsRecurring = true
	task.IsParentSchedule = true
	task.ParentScheduleID = nil

	// Validate recurring task
	if err := s.validateRecurringTask(task); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Create parent schedule
	if err := s.scheduledRepo.Create(ctx, task); err != nil {
		return nil, fmt.Errorf("failed to create recurring schedule: %w", err)
	}

	// Generate initial child instances if requested
	if generateInitialInstances {
		if err := s.GenerateChildInstances(ctx, task, 4); err != nil { // Generate next 4 weeks worth
			// Log error but don't fail the parent creation
			fmt.Printf("Warning: failed to generate initial child instances: %v\n", err)
		}
	}

	// Invalidate cache
	s.invalidateScheduledCache(userID)

	return task, nil
}

// GenerateChildInstances generates child instances for a recurring schedule
func (s *ScheduledTaskService) GenerateChildInstances(ctx context.Context, parent *models.ScheduledTask, weeksAhead int) error {
	if !parent.IsRecurring || !parent.IsParentSchedule {
		return fmt.Errorf("task is not a recurring parent schedule")
	}

	// Get existing children to avoid duplicates
	existingChildren, err := s.scheduledRepo.GetChildrenByParentID(ctx, parent.ID)
	if err != nil {
		return fmt.Errorf("failed to get existing children: %w", err)
	}

	// Create a map of existing scheduled times
	existingTimes := make(map[string]bool)
	for _, child := range existingChildren {
		timeKey := child.ScheduledDatetime.Format("2006-01-02 15:04:05")
		existingTimes[timeKey] = true
	}

	// Generate new instances based on recurrence pattern
	instances := s.calculateRecurrenceInstances(parent, weeksAhead)

	var created int
	for _, instanceTime := range instances {
		timeKey := instanceTime.Format("2006-01-02 15:04:05")

		// Skip if instance already exists
		if existingTimes[timeKey] {
			continue
		}

		// Check if we've hit the recurrence count limit
		if parent.RecurrenceCount != nil && created >= *parent.RecurrenceCount {
			break
		}

		// Check if instance is past end date
		if parent.RecurrenceEndDate != nil && instanceTime.After(*parent.RecurrenceEndDate) {
			break
		}

		// Create child instance
		_, err := s.scheduledRepo.CreateChildInstance(ctx, parent, instanceTime)
		if err != nil {
			return fmt.Errorf("failed to create child instance for %v: %w", instanceTime, err)
		}
		created++
	}

	return nil
}

// calculateRecurrenceInstances calculates the next instances based on recurrence pattern
func (s *ScheduledTaskService) calculateRecurrenceInstances(parent *models.ScheduledTask, weeksAhead int) []time.Time {
	var instances []time.Time
	endTime := time.Now().AddDate(0, 0, weeksAhead*7)
	current := parent.ScheduledDatetime

	// Skip past instances
	for current.Before(time.Now()) {
		current = s.getNextOccurrence(current, parent)
	}

	for current.Before(endTime) && len(instances) < 50 { // Safety limit
		// Check if we've hit the recurrence count limit
		if parent.RecurrenceCount != nil && len(instances) >= *parent.RecurrenceCount {
			break
		}

		// Check if instance is past end date
		if parent.RecurrenceEndDate != nil && current.After(*parent.RecurrenceEndDate) {
			break
		}

		instances = append(instances, current)
		current = s.getNextOccurrence(current, parent)
	}

	return instances
}

// getNextOccurrence calculates the next occurrence based on recurrence pattern
func (s *ScheduledTaskService) getNextOccurrence(current time.Time, parent *models.ScheduledTask) time.Time {
	switch parent.RecurrencePattern {
	case models.RecurrencePatternDaily:
		return current.AddDate(0, 0, parent.RecurrenceInterval)

	case models.RecurrencePatternWeekdays:
		next := current.AddDate(0, 0, 1)
		for next.Weekday() == time.Saturday || next.Weekday() == time.Sunday {
			next = next.AddDate(0, 0, 1)
		}
		return next

	case models.RecurrencePatternWeekly:
		return current.AddDate(0, 0, 7*parent.RecurrenceInterval)

	case models.RecurrencePatternMonthly:
		return current.AddDate(0, parent.RecurrenceInterval, 0)

	case models.RecurrencePatternCustom:
		// For custom patterns, use weekday mask
		if parent.WeekdayMask != nil {
			return s.getNextWeekdayOccurrence(current, *parent.WeekdayMask)
		}
		// Fallback to weekly
		return current.AddDate(0, 0, 7)

	default:
		return current.AddDate(0, 0, 1)
	}
}

// getNextWeekdayOccurrence finds the next occurrence based on weekday mask
func (s *ScheduledTaskService) getNextWeekdayOccurrence(current time.Time, weekdayMask int) time.Time {
	next := current.AddDate(0, 0, 1)

	for i := 0; i < 7; i++ { // Check next 7 days
		weekday := int(next.Weekday())
		if weekday == 0 {
			weekday = 7 // Convert Sunday from 0 to 7
		}

		// Check if this weekday is enabled in the mask
		if weekdayMask&(1<<uint(weekday-1)) != 0 {
			return next
		}

		next = next.AddDate(0, 0, 1)
	}

	// Fallback to next week same day
	return current.AddDate(0, 0, 7)
}

// validateRecurringTask validates a recurring task
func (s *ScheduledTaskService) validateRecurringTask(task *models.ScheduledTask) error {
	if err := s.validateScheduledTask(task); err != nil {
		return err
	}

	if !task.IsRecurring {
		return fmt.Errorf("task must be marked as recurring")
	}

	if task.RecurrencePattern == models.RecurrencePatternNone {
		return fmt.Errorf("recurrence pattern is required for recurring tasks")
	}

	if task.RecurrenceInterval <= 0 {
		return fmt.Errorf("recurrence interval must be positive")
	}

	if task.RecurrenceEndDate != nil && task.RecurrenceEndDate.Before(task.ScheduledDatetime) {
		return fmt.Errorf("recurrence end date cannot be before start date")
	}

	if task.RecurrenceCount != nil && *task.RecurrenceCount <= 0 {
		return fmt.Errorf("recurrence count must be positive")
	}

	return nil
}

// ProcessRecurringSchedules processes all recurring schedules and generates needed child instances
func (s *ScheduledTaskService) ProcessRecurringSchedules(ctx context.Context) error {
	parents, err := s.scheduledRepo.GetRecurringParents(ctx)
	if err != nil {
		return fmt.Errorf("failed to get recurring parents: %w", err)
	}

	for _, parent := range parents {
		if err := s.GenerateChildInstances(ctx, parent, 4); err != nil {
			// Log error but continue processing other parents
			fmt.Printf("Warning: failed to generate instances for parent %s: %v\n", parent.ID, err)
		}
	}

	return nil
}

// ========================================
// TIME ENTRY SERVICE
// ========================================

// TimeEntryService handles time entry business logic
type TimeEntryService struct {
	timeRepo    *repository.TimeEntryRepository
	todoRepo    *repository.TodoRepository
	redisClient redis.Client
}

// NewTimeEntryService creates a new time entry service
func NewTimeEntryService(
	timeRepo *repository.TimeEntryRepository,
	todoRepo *repository.TodoRepository,
	redisClient redis.Client,
) *TimeEntryService {
	return &TimeEntryService{
		timeRepo:    timeRepo,
		todoRepo:    todoRepo,
		redisClient: redisClient,
	}
}

const (
	timeEntryCachePrefix = "time_entries:"
	timeEntryCacheTTL    = 20 * time.Minute
)

// StartTimeTracking starts time tracking for a todo
func (s *TimeEntryService) StartTimeTracking(ctx context.Context, userID, todoID uuid.UUID, description *string) (*models.TimeEntry, error) {
	// Verify todo ownership
	exists, err := s.todoRepo.ValidateTodoOwnership(ctx, todoID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to validate todo ownership: %w", err)
	}
	if !exists {
		return nil, fmt.Errorf("todo not found")
	}

	// Check if there's already an active time entry
	activeEntry, err := s.timeRepo.GetActiveTimeEntry(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to check active time entry: %w", err)
	}
	if activeEntry != nil {
		return nil, fmt.Errorf("there is already an active time entry")
	}

	// Create time entry
	entry := &models.TimeEntry{
		TodoID:      todoID,
		UserID:      userID,
		StartTime:   time.Now(),
		Description: description,
	}

	if err := s.timeRepo.Create(ctx, entry); err != nil {
		return nil, fmt.Errorf("failed to create time entry: %w", err)
	}

	// Invalidate cache
	s.invalidateTimeEntryCache(userID)

	return entry, nil
}

// StopTimeTracking stops the active time tracking
func (s *TimeEntryService) StopTimeTracking(ctx context.Context, userID uuid.UUID) (*models.TimeEntry, error) {
	// Get active time entry
	activeEntry, err := s.timeRepo.GetActiveTimeEntry(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get active time entry: %w", err)
	}
	if activeEntry == nil {
		return nil, fmt.Errorf("no active time entry found")
	}

	// Calculate duration
	endTime := time.Now()
	duration := int(endTime.Sub(activeEntry.StartTime).Minutes())

	// Update time entry
	updates := map[string]interface{}{
		"end_time":         endTime,
		"duration_minutes": duration,
	}

	if err := s.timeRepo.Update(ctx, activeEntry.ID, updates); err != nil {
		return nil, fmt.Errorf("failed to update time entry: %w", err)
	}

	// Invalidate cache
	s.invalidateTimeEntryCache(userID)

	// Return updated entry
	return s.timeRepo.GetByID(ctx, activeEntry.ID)
}

// GetUserTimeEntries retrieves time entries for a user
func (s *TimeEntryService) GetUserTimeEntries(ctx context.Context, userID uuid.UUID, fromDate, toDate *string, limit, offset int) ([]*models.TimeEntry, int, error) {
	// Check cache
	cacheKey := fmt.Sprintf("%s%s:from_%v:to_%v:limit_%d:offset_%d", timeEntryCachePrefix, userID.String(), fromDate, toDate, limit, offset)
	if cached, err := s.redisClient.Get(ctx, cacheKey); err == nil {
		var result struct {
			Entries []*models.TimeEntry `json:"entries"`
			Total   int                 `json:"total"`
		}
		if err := json.Unmarshal(cached, &result); err == nil {
			return result.Entries, result.Total, nil
		}
	}

	// Get from database
	entries, total, err := s.timeRepo.GetByUserID(ctx, userID, fromDate, toDate, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get time entries: %w", err)
	}

	// Cache result
	result := struct {
		Entries []*models.TimeEntry `json:"entries"`
		Total   int                 `json:"total"`
	}{
		Entries: entries,
		Total:   total,
	}
	if resultData, err := json.Marshal(result); err == nil {
		s.redisClient.Set(ctx, cacheKey, resultData, timeEntryCacheTTL)
	}

	return entries, total, nil
}

// UpdateTimeEntry updates a time entry
func (s *TimeEntryService) UpdateTimeEntry(ctx context.Context, entryID, userID uuid.UUID, updates map[string]interface{}) (*models.TimeEntry, error) {
	// Get current entry to verify ownership
	entry, err := s.timeRepo.GetByID(ctx, entryID)
	if err != nil {
		return nil, fmt.Errorf("failed to get time entry: %w", err)
	}
	if entry == nil {
		return nil, fmt.Errorf("time entry not found")
	}

	// Verify ownership
	if entry.UserID != userID {
		return nil, fmt.Errorf("time entry not found")
	}

	// Update entry
	if err := s.timeRepo.Update(ctx, entryID, updates); err != nil {
		return nil, fmt.Errorf("failed to update time entry: %w", err)
	}

	// Invalidate cache
	s.invalidateTimeEntryCache(userID)

	// Return updated entry
	return s.timeRepo.GetByID(ctx, entryID)
}

// DeleteTimeEntry deletes a time entry
func (s *TimeEntryService) DeleteTimeEntry(ctx context.Context, entryID, userID uuid.UUID) error {
	// Get current entry to verify ownership
	entry, err := s.timeRepo.GetByID(ctx, entryID)
	if err != nil {
		return fmt.Errorf("failed to get time entry: %w", err)
	}
	if entry == nil {
		return fmt.Errorf("time entry not found")
	}

	// Verify ownership
	if entry.UserID != userID {
		return fmt.Errorf("time entry not found")
	}

	// Delete entry
	if err := s.timeRepo.Delete(ctx, entryID); err != nil {
		return fmt.Errorf("failed to delete time entry: %w", err)
	}

	// Invalidate cache
	s.invalidateTimeEntryCache(userID)

	return nil
}

// GetActiveTimeEntry gets the current active time tracking session
func (s *TimeEntryService) GetActiveTimeEntry(ctx context.Context, userID uuid.UUID) (*models.TimeEntry, error) {
	return s.timeRepo.GetActiveTimeEntry(ctx, userID)
}

func (s *TimeEntryService) invalidateTimeEntryCache(userID uuid.UUID) {
	ctx := context.Background()
	pattern := fmt.Sprintf("%s%s:*", timeEntryCachePrefix, userID.String())
	s.redisClient.DeletePattern(ctx, pattern)
}
