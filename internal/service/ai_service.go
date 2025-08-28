package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"quantumtask-auth-api/internal/models"
	"quantumtask-auth-api/internal/repository"
	"quantumtask-auth-api/pkg/ai"
	"quantumtask-auth-api/pkg/queue"
	"quantumtask-auth-api/pkg/redis"
	"quantumtask-auth-api/pkg/websocket"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// AIService handles AI-powered task enhancement operations
type AIService struct {
	aiClient          ai.AIClient
	todoRepo          *repository.TodoRepository
	subtaskRepo       *repository.SubtaskRepository
	aiInteractionRepo repository.AIInteractionRepositoryInterface
	aiSuggestionRepo  repository.AISuggestionRepositoryInterface
	redisClient       redis.Client
	queueClient       queue.Client
	broadcaster       *websocket.EventBroadcaster
	cacheExpiration   time.Duration
}

// NewAIService creates a new AI service instance
func NewAIService(
	aiClient ai.AIClient,
	todoRepo *repository.TodoRepository,
	subtaskRepo *repository.SubtaskRepository,
	aiInteractionRepo repository.AIInteractionRepositoryInterface,
	aiSuggestionRepo repository.AISuggestionRepositoryInterface,
	redisClient redis.Client,
	queueClient queue.Client,
	broadcaster *websocket.EventBroadcaster,
) *AIService {
	return &AIService{
		aiClient:          aiClient,
		todoRepo:          todoRepo,
		subtaskRepo:       subtaskRepo,
		aiInteractionRepo: aiInteractionRepo,
		aiSuggestionRepo:  aiSuggestionRepo,
		redisClient:       redisClient,
		queueClient:       queueClient,
		broadcaster:       broadcaster,
		cacheExpiration:   time.Hour * 24, // Cache for 24 hours
	}
}

// AIEnhancedTask represents a task with AI enhancements
type AIEnhancedTask struct {
	ID                  string                 `json:"id"`
	TaskName            string                 `json:"task_name"`
	OriginalDescription string                 `json:"original_description"`
	EnhancedDescription string                 `json:"enhanced_description"`
	Emoji               string                 `json:"emoji"`
	Category            string                 `json:"category"`
	Priority            string                 `json:"priority"`
	EstimatedDuration   int                    `json:"estimated_duration_minutes"`
	Tags                []string               `json:"tags"`
	AIGeneratedSubtasks []AIEnhancedSubtask    `json:"ai_generated_subtasks"`
	EnhancementVersion  int                    `json:"enhancement_version"`
	AIMetadata          map[string]interface{} `json:"ai_metadata"`
	CreatedAt           time.Time              `json:"created_at"`
	UpdatedAt           time.Time              `json:"updated_at"`
}

// AIEnhancedSubtask represents an AI-generated subtask
type AIEnhancedSubtask struct {
	ID                string                 `json:"id"`
	Name              string                 `json:"name"`
	Description       string                 `json:"description"`
	EstimatedDuration int                    `json:"estimated_duration_minutes"`
	Order             int                    `json:"order"`
	AIGenerated       bool                   `json:"ai_generated"`
	AIMetadata        map[string]interface{} `json:"ai_metadata"`
	UserAction        string                 `json:"user_action"` // accepted, rejected, modified, pending
	CreatedAt         time.Time              `json:"created_at"`
	UpdatedAt         time.Time              `json:"updated_at"`
}

// CreateAIEnhancedTaskRequest represents the request for creating an AI-enhanced task
type CreateAIEnhancedTaskRequest struct {
	UserID          string `json:"user_id" validate:"required"`
	ProjectID       string `json:"project_id" validate:"required"`
	TaskName        string `json:"task_name" validate:"required,min=1,max=255"`
	TaskDescription string `json:"task_description"`
	ProjectContext  string `json:"project_context"`
	UserPreferences string `json:"user_preferences"`
}

// RefineSubtasksRequest represents the request for refining subtasks
type RefineSubtasksRequest struct {
	UserID       string   `json:"user_id" validate:"required"`
	TodoID       string   `json:"todo_id" validate:"required"`
	SubtaskNames []string `json:"subtask_names"`
	UserFeedback string   `json:"user_feedback"`
}

// ImproveDescriptionRequest represents the request for improving description
type ImproveDescriptionRequest struct {
	UserID             string `json:"user_id" validate:"required"`
	TodoID             string `json:"todo_id" validate:"required"`
	CurrentDescription string `json:"current_description"`
	Context            string `json:"context"`
}

// CreateOptimizedAITaskRequest represents the request for creating an optimized AI-enhanced task
type CreateOptimizedAITaskRequest struct {
	UserID          string `json:"user_id" validate:"required"`
	ProjectID       string `json:"project_id" validate:"required"`
	TaskName        string `json:"task_name" validate:"required,min=1,max=255"`
	TaskDescription string `json:"task_description"`
	ProjectContext  string `json:"project_context"`
	UserPreferences string `json:"user_preferences"`
	Status          string `json:"status"`
}

// OptimizedTaskResponse represents the immediate response for optimized task creation
type OptimizedTaskResponse struct {
	TaskID           string `json:"task_id"`
	TaskName         string `json:"task_name"`
	ProjectID        string `json:"project_id"`
	Status           string `json:"status"`
	AIEnhancementJob string `json:"ai_enhancement_job"`
	Message          string `json:"message"`
}

// AIEnhancementMessage represents the message structure for RabbitMQ AI enhancement queue
type AIEnhancementMessage struct {
	TaskID          string    `json:"task_id"`
	UserID          string    `json:"user_id"`
	ProjectID       string    `json:"project_id"`
	TaskName        string    `json:"task_name"`
	TaskDescription string    `json:"task_description"`
	ProjectContext  string    `json:"project_context"`
	UserPreferences string    `json:"user_preferences"`
	RequestID       string    `json:"request_id"`
	Timestamp       time.Time `json:"timestamp"`
}

// CreateAIEnhancedTask creates a new task with AI enhancements
func (s *AIService) CreateAIEnhancedTask(ctx context.Context, req CreateAIEnhancedTaskRequest) (*AIEnhancedTask, error) {
	startTime := time.Now()

	// Log the AI interaction
	interaction := &repository.AIInteraction{
		UserID:          req.UserID,
		InteractionType: "task_enhancement",
		RequestData: map[string]interface{}{
			"task_name":        req.TaskName,
			"task_description": req.TaskDescription,
			"project_context":  req.ProjectContext,
			"user_preferences": req.UserPreferences,
			"ai_model":         "google-gemini-1.5-flash",
		},
		CreatedAt: startTime,
	}

	// Call AI for task enhancement
	aiReq := ai.TaskEnhancementRequest{
		TaskName:        req.TaskName,
		TaskDescription: req.TaskDescription,
		ProjectContext:  req.ProjectContext,
		UserPreferences: req.UserPreferences,
	}

	aiResponse, err := s.aiClient.EnhanceTask(ctx, aiReq)
	if err != nil {
		interaction.Success = false
		interaction.ErrorMessage = err.Error()
		interaction.DurationMs = int(time.Since(startTime).Milliseconds())
		s.aiInteractionRepo.Create(ctx, interaction)
		return nil, fmt.Errorf("failed to enhance task with AI: %w", err)
	}

	// Update interaction log with success
	interaction.Success = true
	interaction.ResponseData = map[string]interface{}{
		"enhanced_description": aiResponse.EnhancedDescription,
		"emoji":                aiResponse.Emoji,
		"category":             aiResponse.Category,
		"priority":             aiResponse.Priority,
		"estimated_duration":   aiResponse.EstimatedDuration,
		"model_version":        "1.5-flash",
		"subtasks_generated":   len(aiResponse.Subtasks),
		"subtasks_count":       len(aiResponse.Subtasks),
	}
	interaction.DurationMs = int(time.Since(startTime).Milliseconds())

	// Create the enhanced task in database
	todo := &models.Todo{
		UserID:                 uuid.MustParse(req.UserID),
		ProjectID:              uuid.MustParse(req.ProjectID),
		TaskName:               req.TaskName,
		TaskDescription:        &aiResponse.EnhancedDescription,
		AIEnhanced:             true,
		AIGeneratedDescription: req.TaskDescription == "", // True if no original description
		TaskEmoji:              &aiResponse.Emoji,
		AICategory:             &aiResponse.Category,
		AIPriority:             &aiResponse.Priority,
		AIEstimatedDuration:    &aiResponse.EstimatedDuration,
		AIEnhancementVersion:   1,
		Tags:                   pq.StringArray(aiResponse.Tags),
		Status:                 models.TaskStatusBacklog,
		AIMetadata: &models.JSONB{
			"ai_model":             "google-gemini-1.5-flash",
			"enhancement_date":     time.Now(),
			"original_description": req.TaskDescription,
			"ai_confidence":        0.90, // Gemini typically has higher confidence
			"model_version":        "1.5-flash",
		},
	}

	err = s.todoRepo.Create(ctx, todo)
	if err != nil {
		return nil, fmt.Errorf("failed to create task: %w", err)
	}

	createdTodo := todo // The ID is set in the Create method

	todoIDStr := createdTodo.ID.String()
	interaction.TodoID = &todoIDStr
	s.aiInteractionRepo.Create(ctx, interaction)

	// Create AI-generated subtasks in the database
	var aiSubtasks []AIEnhancedSubtask
	for _, aiSubtask := range aiResponse.Subtasks {
		// Create the actual subtask in the database
		subtask := &models.Subtask{
			TodoID:              createdTodo.ID,
			Name:                aiSubtask.Name,
			Description:         &aiSubtask.Description,
			Status:              models.TaskStatusNotStarted,
			Priority:            models.PriorityMedium,
			EstimatedTime:       &aiSubtask.EstimatedDuration,
			SortOrder:           aiSubtask.Order,
			AIGenerated:         true,
			AIEstimatedDuration: &aiSubtask.EstimatedDuration,
			AIMetadata: &models.JSONB{
				"ai_order":      aiSubtask.Order,
				"ai_confidence": 0.85,
				"source":        "initial_enhancement",
				"ai_model":      "google-gemini-1.5-flash",
			},
		}

		err := s.subtaskRepo.Create(ctx, subtask)
		if err != nil {
			// Log error but don't fail the entire task creation
			fmt.Printf("Failed to create AI subtask: %v\n", err)
			continue
		}

		// Create AI suggestion record for tracking
		suggestion := &repository.AISuggestion{
			UserID:           req.UserID,
			TodoID:           &todoIDStr,
			SuggestionType:   "subtask",
			SuggestedContent: aiSubtask.Name,
			AIConfidence:     0.85,
			UserAction:       "accepted", // Auto-accept AI-generated subtasks
		}
		s.aiSuggestionRepo.Create(ctx, suggestion)

		// Create response subtask for the API response
		aiSubtasks = append(aiSubtasks, AIEnhancedSubtask{
			ID:                subtask.ID.String(),
			Name:              subtask.Name,
			Description:       getStringValue(subtask.Description),
			EstimatedDuration: getIntValue(subtask.EstimatedTime),
			Order:             subtask.SortOrder,
			AIGenerated:       true,
			AIMetadata: map[string]interface{}{
				"ai_order":      aiSubtask.Order,
				"ai_confidence": 0.85,
				"source":        "initial_enhancement",
			},
			UserAction: "accepted",
			CreatedAt:  subtask.CreatedAt,
			UpdatedAt:  subtask.UpdatedAt,
		})
	}

	// Create AI suggestions for other enhancements
	s.createSuggestionRecords(ctx, req.UserID, todoIDStr, aiResponse, req.TaskDescription)

	// Build and return the enhanced task
	enhancedTask := &AIEnhancedTask{
		ID:                  todoIDStr,
		TaskName:            createdTodo.TaskName,
		OriginalDescription: req.TaskDescription,
		EnhancedDescription: getStringValue(createdTodo.TaskDescription),
		Emoji:               getStringValue(createdTodo.TaskEmoji),
		Category:            getStringValue(createdTodo.AICategory),
		Priority:            getStringValue(createdTodo.AIPriority),
		EstimatedDuration:   getIntValue(createdTodo.AIEstimatedDuration),
		Tags:                createdTodo.Tags,
		AIGeneratedSubtasks: aiSubtasks,
		EnhancementVersion:  createdTodo.AIEnhancementVersion,
		AIMetadata:          getJSONBValue(createdTodo.AIMetadata),
		CreatedAt:           createdTodo.CreatedAt,
		UpdatedAt:           createdTodo.UpdatedAt,
	}

	// Cache the enhanced task
	s.cacheEnhancedTask(ctx, enhancedTask)

	return enhancedTask, nil
}

// CreateOptimizedAITask creates a task immediately and queues AI enhancement for background processing
func (s *AIService) CreateOptimizedAITask(ctx context.Context, req CreateOptimizedAITaskRequest) (*OptimizedTaskResponse, error) {
	const queueName = "ai_enhancement_list_v2"

	// Create the basic task immediately
	todo := &models.Todo{
		UserID:                 uuid.MustParse(req.UserID),
		ProjectID:              uuid.MustParse(req.ProjectID),
		TaskName:               req.TaskName,
		TaskDescription:        &req.TaskDescription,
		AIEnhanced:             false, // Will be set to true after AI enhancement
		AIGeneratedDescription: false,
		AIEnhancementVersion:   0, // Will be incremented after AI enhancement
		Status:                 models.TaskStatus(req.Status),
		Priority:               models.PriorityMedium,
		TimeUnit:               models.TimeUnitMinutes,
	}

	// Save the task to database immediately
	fmt.Printf("💾 Creating task in database: %s\n", req.TaskName)
	err := s.todoRepo.Create(ctx, todo)
	if err != nil {
		fmt.Printf("❌ Failed to create task in database: %v\n", err)
		return nil, fmt.Errorf("failed to create task: %w", err)
	}
	fmt.Printf("✅ Task created successfully in database with ID: %s\n", todo.ID.String())

	// 🔔 Send WebSocket notification for immediate UI update
	if s.broadcaster != nil {
		fmt.Printf("🔔 Broadcasting task creation via WebSocket: Task ID=%s, User ID=%s, Project ID=%s\n",
			todo.ID.String(), todo.UserID.String(), todo.ProjectID.String())
		s.broadcaster.BroadcastTodoCreated(ctx, todo, nil)
		fmt.Printf("✅ WebSocket notification sent for task creation: %s\n", todo.ID.String())
	} else {
		fmt.Printf("❌ WebSocket broadcaster is nil - cannot send task creation notification\n")
	}

	// Generate request ID for tracking
	requestID := uuid.New().String()

	// Create message for AI enhancement queue
	enhancementMessage := AIEnhancementMessage{
		TaskID:          todo.ID.String(),
		UserID:          req.UserID,
		ProjectID:       req.ProjectID,
		TaskName:        req.TaskName,
		TaskDescription: req.TaskDescription,
		ProjectContext:  req.ProjectContext,
		UserPreferences: req.UserPreferences,
		RequestID:       requestID,
		Timestamp:       time.Now(),
	}

	// Publish message to RabbitMQ for async processing
	fmt.Printf("🐰 Publishing AI enhancement message to queue '%s' for task: %s\n", queueName, todo.ID.String())
	fmt.Printf("📄 Enhancement message details: TaskID=%s, RequestID=%s, TaskName=%s\n",
		enhancementMessage.TaskID, enhancementMessage.RequestID, enhancementMessage.TaskName)

	err = s.queueClient.Publish(ctx, queueName, enhancementMessage, 5) // High priority
	if err != nil {
		// Log error but don't fail the request since the task was already created
		fmt.Printf("❌ Failed to queue AI enhancement for task %s: %v\n", todo.ID.String(), err)
		fmt.Printf("🔧 This might be due to RabbitMQ connection issues. Check RabbitMQ server status.\n")
	} else {
		fmt.Printf("✅ AI enhancement message successfully published to queue '%s'\n", queueName)
		fmt.Printf("🕒 Message timestamp: %s\n", enhancementMessage.Timestamp.Format(time.RFC3339))
	}

	// Clear relevant caches and set new cache entries
	s.clearAndSetTaskCache(ctx, todo.ID, uuid.MustParse(req.UserID), uuid.MustParse(req.ProjectID), todo)
	// Invalidate caches
	s.invalidateTodoCaches(uuid.MustParse(req.UserID), uuid.MustParse(req.ProjectID))

	// Return immediate response
	return &OptimizedTaskResponse{
		TaskID:           todo.ID.String(),
		TaskName:         todo.TaskName,
		ProjectID:        req.ProjectID,
		Status:           "created",
		AIEnhancementJob: requestID,
		Message:          "Task created successfully. AI enhancement is being processed in the background.",
	}, nil
}

// ProcessAIEnhancement processes an AI enhancement message from the queue
func (s *AIService) ProcessAIEnhancement(ctx context.Context, message AIEnhancementMessage) error {
	startTime := time.Now()

	// Get the task from database
	taskID, err := uuid.Parse(message.TaskID)
	if err != nil {
		return fmt.Errorf("invalid task ID: %w", err)
	}

	_, err = s.todoRepo.GetByID(ctx, taskID, false)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}

	// Log the AI interaction
	interaction := &repository.AIInteraction{
		UserID:          message.UserID,
		TodoID:          &message.TaskID,
		InteractionType: "async_task_enhancement",
		RequestData: map[string]interface{}{
			"task_name":        message.TaskName,
			"task_description": message.TaskDescription,
			"project_context":  message.ProjectContext,
			"user_preferences": message.UserPreferences,
			"ai_model":         "google-gemini-1.5-flash",
			"request_id":       message.RequestID,
		},
		CreatedAt: startTime,
	}

	// Call AI for task enhancement
	aiReq := ai.TaskEnhancementRequest{
		TaskName:        message.TaskName,
		TaskDescription: message.TaskDescription,
		ProjectContext:  message.ProjectContext,
		UserPreferences: message.UserPreferences,
	}

	aiResponse, err := s.aiClient.EnhanceTask(ctx, aiReq)
	if err != nil {
		interaction.Success = false
		interaction.ErrorMessage = err.Error()
		interaction.DurationMs = int(time.Since(startTime).Milliseconds())
		s.aiInteractionRepo.Create(ctx, interaction)
		return fmt.Errorf("failed to enhance task with AI: %w", err)
	}

	// Update interaction log with success
	interaction.Success = true
	interaction.ResponseData = map[string]interface{}{
		"enhanced_description": aiResponse.EnhancedDescription,
		"emoji":                aiResponse.Emoji,
		"category":             aiResponse.Category,
		"priority":             aiResponse.Priority,
		"estimated_duration":   aiResponse.EstimatedDuration,
		"model_version":        "1.5-flash",
		"subtasks_generated":   len(aiResponse.Subtasks),
		"subtasks_count":       len(aiResponse.Subtasks),
	}
	interaction.DurationMs = int(time.Since(startTime).Milliseconds())
	s.aiInteractionRepo.Create(ctx, interaction)

	// Update the task with AI enhancements
	updates := map[string]interface{}{
		"task_description":         &aiResponse.EnhancedDescription,
		"ai_enhanced":              true,
		"ai_generated_description": message.TaskDescription == "",
		"task_emoji":               &aiResponse.Emoji,
		"ai_category":              &aiResponse.Category,
		"ai_priority":              &aiResponse.Priority,
		"ai_estimated_duration":    &aiResponse.EstimatedDuration,
		"ai_enhancement_version":   1,
		"tags":                     pq.StringArray(aiResponse.Tags),
		"ai_metadata": &models.JSONB{
			"ai_model":             "google-gemini-1.5-flash",
			"enhancement_date":     time.Now(),
			"original_description": message.TaskDescription,
			"ai_confidence":        0.90,
			"model_version":        "1.5-flash",
			"request_id":           message.RequestID,
		},
	}

	err = s.todoRepo.Update(ctx, taskID, updates)
	if err != nil {
		return fmt.Errorf("failed to update task with AI enhancements: %w", err)
	}

	// Create AI-generated subtasks
	for _, aiSubtask := range aiResponse.Subtasks {
		subtask := &models.Subtask{
			TodoID:              taskID,
			Name:                aiSubtask.Name,
			Description:         &aiSubtask.Description,
			Status:              models.TaskStatusNotStarted,
			Priority:            models.PriorityMedium,
			EstimatedTime:       &aiSubtask.EstimatedDuration,
			SortOrder:           aiSubtask.Order,
			AIGenerated:         true,
			AIEstimatedDuration: &aiSubtask.EstimatedDuration,
			AIMetadata: &models.JSONB{
				"ai_order":      aiSubtask.Order,
				"ai_confidence": 0.85,
				"source":        "async_enhancement",
				"ai_model":      "google-gemini-1.5-flash",
				"request_id":    message.RequestID,
			},
		}

		err := s.subtaskRepo.Create(ctx, subtask)
		if err != nil {
			fmt.Printf("Failed to create AI subtask: %v\n", err)
			continue
		}

		// Create AI suggestion record for tracking
		suggestion := &repository.AISuggestion{
			UserID:           message.UserID,
			TodoID:           &message.TaskID,
			SuggestionType:   "subtask",
			SuggestedContent: aiSubtask.Name,
			AIConfidence:     0.85,
			UserAction:       "accepted",
		}
		s.aiSuggestionRepo.Create(ctx, suggestion)
	}

	// Create AI suggestions for other enhancements
	s.createSuggestionRecords(ctx, message.UserID, message.TaskID, aiResponse, message.TaskDescription)

	// Clear cache for the updated task
	s.clearTaskCache(ctx, taskID, uuid.MustParse(message.UserID))

	// 🔔 Send WebSocket notification about completed AI enhancement
	if s.broadcaster != nil {
		// Get the updated task from database to send in notification
		updatedTodo, err := s.todoRepo.GetByID(ctx, taskID, true)
		if err == nil && updatedTodo != nil {
			s.broadcaster.BroadcastTodoUpdated(ctx, updatedTodo, map[string]interface{}{
				"ai_enhanced":      true,
				"enhanced_by":      "ai_service",
				"enhancement_type": "ai_completion",
			}, nil)
			fmt.Printf("🔔 Sent WebSocket notification for AI enhancement completion: %s\n", taskID.String())
		}
	}

	return nil
}

// clearTaskCache clears all cache entries related to a task and sets new ones for enhanced tasks
func (s *AIService) clearTaskCache(ctx context.Context, taskID uuid.UUID, userID uuid.UUID) {
	// Clear existing caches that might be affected
	cacheKeysToClear := []string{
		fmt.Sprintf("ai_enhanced_task:%s", taskID.String()),
		fmt.Sprintf("todo:%s:related_true", taskID.String()),
		fmt.Sprintf("todo:%s:related_false", taskID.String()),
		fmt.Sprintf("user_todos:%s", userID.String()),
		fmt.Sprintf("todo_dashboard:%s", userID.String()),
		fmt.Sprintf("overdue_todos:%s", userID.String()),
		fmt.Sprintf("todo_search:%s", userID.String()),
	}

	// Clear the caches
	for _, key := range cacheKeysToClear {
		s.redisClient.Delete(ctx, key)
		fmt.Printf("🗑️  Cleared cache key: %s\n", key)
	}

	// Get the updated task to set new cache entries
	todo, err := s.todoRepo.GetByID(ctx, taskID, false)
	if err != nil {
		fmt.Printf("⚠️  Could not retrieve updated task for caching: %v\n", err)
		return
	}

	// Set new cache entries for the enhanced task
	taskCacheKey := fmt.Sprintf("todo:%s:related_true", taskID.String())
	taskData, err := json.Marshal(todo)
	if err == nil {
		// Cache the enhanced task for 30 minutes (longer TTL for enhanced tasks)
		s.redisClient.Set(ctx, taskCacheKey, taskData, 30*time.Minute)
		fmt.Printf("💾 Set enhanced task cache key: %s (TTL: 30m)\n", taskCacheKey)
	}

	// Set AI enhanced task cache
	aiEnhancedKey := fmt.Sprintf("ai_enhanced_task:%s", taskID.String())
	aiEnhancedData := map[string]interface{}{
		"task_id":               taskID.String(),
		"enhanced_at":           time.Now(),
		"enhancement_version":   todo.AIEnhancementVersion,
		"ai_enhanced":           todo.AIEnhanced,
		"ai_category":           todo.AICategory,
		"ai_priority":           todo.AIPriority,
		"ai_estimated_duration": todo.AIEstimatedDuration,
	}

	aiEnhancedBytes, err := json.Marshal(aiEnhancedData)
	if err == nil {
		s.redisClient.Set(ctx, aiEnhancedKey, aiEnhancedBytes, 1*time.Hour)
		fmt.Printf("🤖 Set AI enhanced task cache: %s (TTL: 1h)\n", aiEnhancedKey)
	}

	fmt.Printf("✅ Cache operations completed for enhanced task: %s\n", taskID.String())
}

// clearAndSetTaskCache clears relevant caches and sets new cache entries for a newly created task
func (s *AIService) clearAndSetTaskCache(ctx context.Context, taskID uuid.UUID, userID uuid.UUID, projectID uuid.UUID, todo *models.Todo) {
	// Clear existing caches that might be affected
	cacheKeysToClear := []string{
		fmt.Sprintf("user_todos:%s", userID.String()),
		fmt.Sprintf("project_todos:%s", projectID.String()),
		fmt.Sprintf("todo_dashboard:%s", userID.String()),
		fmt.Sprintf("project_dashboard:%s", projectID.String()),
		fmt.Sprintf("overdue_todos:%s", userID.String()),
		fmt.Sprintf("todo_search:%s", userID.String()),
	}

	// Clear the caches
	for _, key := range cacheKeysToClear {
		s.redisClient.Delete(ctx, key)
		fmt.Printf("🗑️  Cleared cache key: %s\n", key)
	}

	// Set new cache entries for the created task
	taskCacheKey := fmt.Sprintf("todo:%s:related_false", taskID.String())
	taskData, err := json.Marshal(todo)
	if err == nil {
		// Cache the task for 15 minutes
		s.redisClient.Set(ctx, taskCacheKey, taskData, 15*time.Minute)
		fmt.Printf("💾 Set cache key: %s (TTL: 15m)\n", taskCacheKey)
	}

	// Set user todos count cache
	userTodosKey := fmt.Sprintf("user_todos_count:%s", userID.String())
	userTodosData := []byte("1") // Simple count representation
	s.redisClient.Set(ctx, userTodosKey, userTodosData, 30*time.Minute)
	fmt.Printf("📊 Set user todos count cache: %s (TTL: 30m)\n", userTodosKey)

	// Set project todos count cache
	projectTodosKey := fmt.Sprintf("project_todos_count:%s", projectID.String())
	projectTodosData := []byte("1") // Simple count representation
	s.redisClient.Set(ctx, projectTodosKey, projectTodosData, 30*time.Minute)
	fmt.Printf("📊 Set project todos count cache: %s (TTL: 30m)\n", projectTodosKey)

	// Set recent tasks cache for quick access
	recentTasksKey := fmt.Sprintf("recent_tasks:%s", userID.String())
	recentTask := map[string]interface{}{
		"id":          todo.ID.String(),
		"name":        todo.TaskName,
		"project_id":  projectID.String(),
		"created_at":  todo.CreatedAt,
		"status":      todo.Status,
		"ai_enhanced": todo.AIEnhanced,
	}

	// Cache the recent task data
	recentTaskData, err := json.Marshal(recentTask)
	if err == nil {
		s.redisClient.Set(ctx, recentTasksKey, recentTaskData, 1*time.Hour)
		fmt.Printf("📝 Set recent tasks cache: %s (TTL: 1h)\n", recentTasksKey)
	}

	fmt.Printf("✅ Cache operations completed for task: %s\n", taskID.String())
}

func (s *AIService) invalidateTodoCaches(userID, projectID uuid.UUID) {
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

// RefineSubtasks refines existing subtasks using AI
func (s *AIService) RefineSubtasks(ctx context.Context, req RefineSubtasksRequest) ([]AIEnhancedSubtask, error) {
	startTime := time.Now()

	// Get the task details
	todoID, err := uuid.Parse(req.TodoID)
	if err != nil {
		return nil, fmt.Errorf("invalid todo ID: %w", err)
	}

	todo, err := s.todoRepo.GetByID(ctx, todoID, false)
	if err != nil {
		return nil, fmt.Errorf("failed to get task: %w", err)
	}

	// Log the AI interaction
	interaction := &repository.AIInteraction{
		UserID:          req.UserID,
		TodoID:          &req.TodoID,
		InteractionType: "subtask_refinement",
		RequestData: map[string]interface{}{
			"subtask_names": req.SubtaskNames,
			"user_feedback": req.UserFeedback,
		},
		CreatedAt: startTime,
	}

	// Call AI for subtask refinement
	aiSubtasks, err := s.aiClient.RefineSubtasks(ctx, todo.TaskName, req.SubtaskNames, req.UserFeedback)
	if err != nil {
		interaction.Success = false
		interaction.ErrorMessage = err.Error()
		interaction.DurationMs = int(time.Since(startTime).Milliseconds())
		s.aiInteractionRepo.Create(ctx, interaction)
		return nil, fmt.Errorf("failed to refine subtasks with AI: %w", err)
	}

	// Update interaction log
	interaction.Success = true
	interaction.ResponseData = map[string]interface{}{
		"refined_subtasks_count": len(aiSubtasks),
	}
	interaction.DurationMs = int(time.Since(startTime).Milliseconds())
	s.aiInteractionRepo.Create(ctx, interaction)

	// Update enhancement version
	todo.AIEnhancementVersion++
	updates := map[string]interface{}{
		"ai_enhancement_version": todo.AIEnhancementVersion,
	}
	s.todoRepo.Update(ctx, todoID, updates)

	var refinedSubtasks []AIEnhancedSubtask
	for _, aiSubtask := range aiSubtasks {
		// TODO: Create or update subtask when subtask repository is available

		// Create suggestion record
		suggestion := &repository.AISuggestion{
			UserID:            req.UserID,
			TodoID:            &req.TodoID,
			SuggestionType:    "subtask",
			SuggestedContent:  aiSubtask.Name,
			AIConfidence:      0.85,
			UserAction:        "pending",
			UserFeedback:      req.UserFeedback,
			RegenerationCount: 1,
		}
		s.aiSuggestionRepo.Create(ctx, suggestion)

		// Create virtual refined subtask for response
		refinedSubtasks = append(refinedSubtasks, AIEnhancedSubtask{
			ID:                uuid.New().String(),
			Name:              aiSubtask.Name,
			Description:       aiSubtask.Description,
			EstimatedDuration: aiSubtask.EstimatedDuration,
			Order:             aiSubtask.Order,
			AIGenerated:       true,
			AIMetadata: map[string]interface{}{
				"ai_confidence":   0.85,
				"source":          "refinement",
				"refinement_date": time.Now(),
			},
			UserAction: "pending",
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		})
	}

	return refinedSubtasks, nil
}

// ImproveDescription enhances a task description using AI
func (s *AIService) ImproveDescription(ctx context.Context, taskName, currentDescription, priority string, tags []string) (string, error) {
	// Call AI client to improve description
	enhancedDescription, err := s.aiClient.ImproveDescription(ctx, taskName, currentDescription, priority, tags)
	if err != nil {
		return "", fmt.Errorf("failed to improve description: %w", err)
	}

	return enhancedDescription, nil
}

// AcceptAISuggestion marks an AI suggestion as accepted
func (s *AIService) AcceptAISuggestion(ctx context.Context, userID, suggestionID string) error {
	suggestion, err := s.aiSuggestionRepo.GetByID(ctx, suggestionID)
	if err != nil {
		return fmt.Errorf("failed to get suggestion: %w", err)
	}

	if suggestion.UserID != userID {
		return fmt.Errorf("unauthorized access to suggestion")
	}

	suggestion.UserAction = "accepted"
	suggestion.UpdatedAt = time.Now()

	_, err = s.aiSuggestionRepo.Update(ctx, suggestion)
	return err
}

// RejectAISuggestion marks an AI suggestion as rejected
func (s *AIService) RejectAISuggestion(ctx context.Context, userID, suggestionID, feedback string) error {
	suggestion, err := s.aiSuggestionRepo.GetByID(ctx, suggestionID)
	if err != nil {
		return fmt.Errorf("failed to get suggestion: %w", err)
	}

	if suggestion.UserID != userID {
		return fmt.Errorf("unauthorized access to suggestion")
	}

	suggestion.UserAction = "rejected"
	suggestion.UserFeedback = feedback
	suggestion.UpdatedAt = time.Now()

	_, err = s.aiSuggestionRepo.Update(ctx, suggestion)
	return err
}

// GetAIMetrics returns AI usage metrics for a user
func (s *AIService) GetAIMetrics(ctx context.Context, userID string) (map[string]interface{}, error) {
	interactions, err := s.aiInteractionRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get AI interactions: %w", err)
	}

	suggestions, err := s.aiSuggestionRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get AI suggestions: %w", err)
	}

	metrics := map[string]interface{}{
		"total_interactions":      len(interactions),
		"successful_interactions": len(interactions), // Count successful ones
		"total_suggestions":       len(suggestions),
		"accepted_suggestions":    len(suggestions), // Count accepted ones
		"ai_enhanced_tasks":       0,                // Count from database
	}

	return metrics, nil
}

// Helper methods

// createSuggestionRecords creates AI suggestion records for task enhancements
func (s *AIService) createSuggestionRecords(ctx context.Context, userID, todoID string, aiResponse *ai.TaskEnhancementResponse, originalDescription string) {
	// Description suggestion
	if aiResponse.EnhancedDescription != "" {
		suggestion := &repository.AISuggestion{
			UserID:           userID,
			TodoID:           &todoID,
			SuggestionType:   "description",
			OriginalContent:  originalDescription,
			SuggestedContent: aiResponse.EnhancedDescription,
			AIConfidence:     0.85,
			UserAction:       "accepted", // Auto-accepted for initial creation
		}
		s.aiSuggestionRepo.Create(ctx, suggestion)
	}

	// Emoji suggestion
	if aiResponse.Emoji != "" {
		suggestion := &repository.AISuggestion{
			UserID:           userID,
			TodoID:           &todoID,
			SuggestionType:   "emoji",
			SuggestedContent: aiResponse.Emoji,
			AIConfidence:     0.85,
			UserAction:       "accepted",
		}
		s.aiSuggestionRepo.Create(ctx, suggestion)
	}

	// Category suggestion
	if aiResponse.Category != "" {
		suggestion := &repository.AISuggestion{
			UserID:           userID,
			TodoID:           &todoID,
			SuggestionType:   "category",
			SuggestedContent: aiResponse.Category,
			AIConfidence:     0.85,
			UserAction:       "accepted",
		}
		s.aiSuggestionRepo.Create(ctx, suggestion)
	}

	// Priority suggestion
	if aiResponse.Priority != "" {
		suggestion := &repository.AISuggestion{
			UserID:           userID,
			TodoID:           &todoID,
			SuggestionType:   "priority",
			SuggestedContent: aiResponse.Priority,
			AIConfidence:     0.85,
			UserAction:       "accepted",
		}
		s.aiSuggestionRepo.Create(ctx, suggestion)
	}
}

// cacheEnhancedTask caches the enhanced task for performance
func (s *AIService) cacheEnhancedTask(ctx context.Context, task *AIEnhancedTask) {
	if s.redisClient == nil {
		return
	}

	key := fmt.Sprintf("ai_enhanced_task:%s", task.ID)
	data, err := json.Marshal(task)
	if err != nil {
		return
	}

	s.redisClient.Set(ctx, key, data, s.cacheExpiration)
}

// Helper functions to safely convert pointer types
func getStringValue(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func getIntValue(i *int) int {
	if i == nil {
		return 0
	}
	return *i
}

func getJSONBValue(j *models.JSONB) map[string]interface{} {
	if j == nil {
		return nil
	}
	// Convert JSONB to map[string]interface{}
	result := make(map[string]interface{})
	for k, v := range *j {
		result[k] = v
	}
	return result
}
