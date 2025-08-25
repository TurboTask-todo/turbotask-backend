package handler

import (
	"net/http"
	"strconv"
	"time"

	"macwrite-auth-api/internal/middleware"
	"macwrite-auth-api/internal/service"

	"github.com/gin-gonic/gin"
)

// AIHandler handles AI-related HTTP requests
type AIHandler struct {
	aiService *service.AIService
}

// NewAIHandler creates a new AI handler
func NewAIHandler(aiService *service.AIService) *AIHandler {
	return &AIHandler{
		aiService: aiService,
	}
}

// Response utilities
func errorResponse(c *gin.Context, status int, code, message string, err error) {
	c.JSON(status, gin.H{
		"success": false,
		"error": gin.H{
			"code":    code,
			"message": message,
		},
		"timestamp": time.Now(),
	})
}

func successResponse(c *gin.Context, status int, message string, data interface{}) {
	c.JSON(status, gin.H{
		"success":   true,
		"message":   message,
		"data":      data,
		"timestamp": time.Now(),
	})
}

func getUserIDFromContext(c *gin.Context) string {
	userID, exists := c.Get("userID")
	if !exists {
		return ""
	}
	return userID.(string)
}

// CreateAIEnhancedTaskRequest represents the HTTP request for creating AI-enhanced tasks
type CreateAIEnhancedTaskRequest struct {
	TaskName        string `json:"task_name" binding:"required,min=1,max=255"`
	TaskDescription string `json:"task_description"`
	ProjectID       string `json:"project_id" binding:"required"`
	ProjectContext  string `json:"project_context"`
	UserPreferences string `json:"user_preferences"`
}

// CreateOptimizedAITaskRequest represents the HTTP request for creating optimized AI-enhanced tasks
type CreateOptimizedAITaskRequest struct {
	TaskName        string `json:"task_name" binding:"required,min=1,max=255"`
	TaskDescription string `json:"task_description"`
	ProjectID       string `json:"project_id" binding:"required"`
	ProjectContext  string `json:"project_context"`
	UserPreferences string `json:"user_preferences"`
	Status          string `json:"status"`
}

// RefineSubtasksRequest represents the HTTP request for refining subtasks
type RefineSubtasksRequest struct {
	TodoID       string   `json:"todo_id" binding:"required"`
	SubtaskNames []string `json:"subtask_names"`
	UserFeedback string   `json:"user_feedback"`
}

// ImproveDescriptionRequest represents the HTTP request for improving descriptions
type ImproveDescriptionRequest struct {
	TaskName           string   `json:"task_name" binding:"required"`
	CurrentDescription string   `json:"current_description"`
	Priority           string   `json:"priority"`
	Tags               []string `json:"tags"`
}

// AcceptSuggestionRequest represents the HTTP request for accepting suggestions
type AcceptSuggestionRequest struct {
	SuggestionID string `json:"suggestion_id" binding:"required"`
}

// RejectSuggestionRequest represents the HTTP request for rejecting suggestions
type RejectSuggestionRequest struct {
	SuggestionID string `json:"suggestion_id" binding:"required"`
	Feedback     string `json:"feedback"`
}

// CreateAIEnhancedTask creates a new task with AI enhancements
// @Summary Create AI-enhanced task
// @Description Creates a new task with AI-generated description, subtasks, and metadata
// @Tags AI
// @Accept json
// @Produce json
// @Param request body CreateAIEnhancedTaskRequest true "Task creation request"
// @Success 201 {object} service.AIEnhancedTask
// @Failure 400 {object} errorResponse
// @Failure 401 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Router /api/v1/ai/tasks [post]
func (h *AIHandler) CreateAIEnhancedTask(c *gin.Context) {
	// Get user ID from context
	userID, _ := middleware.GetUserID(c)

	var req CreateAIEnhancedTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errorResponse(c, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request format", err)
		return
	}

	serviceReq := service.CreateAIEnhancedTaskRequest{
		UserID:          userID.String(),
		ProjectID:       req.ProjectID,
		TaskName:        req.TaskName,
		TaskDescription: req.TaskDescription,
		ProjectContext:  req.ProjectContext,
		UserPreferences: req.UserPreferences,
	}

	enhancedTask, err := h.aiService.CreateAIEnhancedTask(c.Request.Context(), serviceReq)
	if err != nil {
		errorResponse(c, http.StatusInternalServerError, "AI_ENHANCEMENT_FAILED", "Failed to create AI-enhanced task", err)
		return
	}

	successResponse(c, http.StatusCreated, "AI-enhanced task created successfully", enhancedTask)
}

// CreateOptimizedAITask creates a new task immediately and queues AI enhancement asynchronously
// @Summary Create optimized AI-enhanced task
// @Description Creates a new task immediately and queues AI enhancement for background processing
// @Tags AI
// @Accept json
// @Produce json
// @Param request body CreateOptimizedAITaskRequest true "Task creation request"
// @Success 201 {object} map[string]interface{}
// @Failure 400 {object} errorResponse
// @Failure 401 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Router /api/v1/ai/tasks/optimized [post]
func (h *AIHandler) CreateOptimizedAITask(c *gin.Context) {
	// Get user ID from context
	userID, _ := middleware.GetUserID(c)

	var req CreateOptimizedAITaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errorResponse(c, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request format", err)
		return
	}

	serviceReq := service.CreateOptimizedAITaskRequest{
		UserID:          userID.String(),
		ProjectID:       req.ProjectID,
		TaskName:        req.TaskName,
		TaskDescription: req.TaskDescription,
		ProjectContext:  req.ProjectContext,
		UserPreferences: req.UserPreferences,
		Status:          req.Status,
	}

	taskResponse, err := h.aiService.CreateOptimizedAITask(c.Request.Context(), serviceReq)
	if err != nil {
		errorResponse(c, http.StatusInternalServerError, "TASK_CREATION_FAILED", "Failed to create task", err)
		return
	}

	successResponse(c, http.StatusCreated, "Task created successfully, AI enhancement queued", taskResponse)
}

// RefineSubtasks refines existing subtasks using AI
// @Summary Refine subtasks with AI
// @Description Uses AI to refine and improve existing subtasks based on user feedback
// @Tags AI
// @Accept json
// @Produce json
// @Param request body RefineSubtasksRequest true "Subtask refinement request"
// @Success 200 {array} service.AIEnhancedSubtask
// @Failure 400 {object} errorResponse
// @Failure 401 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Router /api/v1/ai/subtasks/refine [post]
func (h *AIHandler) RefineSubtasks(c *gin.Context) {
	userID := getUserIDFromContext(c)
	if userID == "" {
		errorResponse(c, http.StatusUnauthorized, "AUTH_REQUIRED", "Authentication required", nil)
		return
	}

	var req RefineSubtasksRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errorResponse(c, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request format", err)
		return
	}

	serviceReq := service.RefineSubtasksRequest{
		UserID:       userID,
		TodoID:       req.TodoID,
		SubtaskNames: req.SubtaskNames,
		UserFeedback: req.UserFeedback,
	}

	refinedSubtasks, err := h.aiService.RefineSubtasks(c.Request.Context(), serviceReq)
	if err != nil {
		errorResponse(c, http.StatusInternalServerError, "SUBTASK_REFINEMENT_FAILED", "Failed to refine subtasks", err)
		return
	}

	successResponse(c, http.StatusOK, "Subtasks refined successfully", refinedSubtasks)
}

// ImproveDescription improves task description using AI
// @Summary Improve task description with AI
// @Description Uses AI to improve and enhance task descriptions
// @Tags AI
// @Accept json
// @Produce json
// @Param request body ImproveDescriptionRequest true "Description improvement request"
// @Success 200 {object} map[string]string
// @Failure 400 {object} errorResponse
// @Failure 401 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Router /api/v1/ai/description/improve [post]
func (h *AIHandler) ImproveDescription(c *gin.Context) {
	userID := getUserIDFromContext(c)
	if userID == "" {
		errorResponse(c, http.StatusUnauthorized, "AUTH_REQUIRED", "Authentication required", nil)
		return
	}

	var req ImproveDescriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errorResponse(c, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request format", err)
		return
	}

	// Call AI service to improve description
	enhancedDescription, err := h.aiService.ImproveDescription(c.Request.Context(), req.TaskName, req.CurrentDescription, req.Priority, req.Tags)
	if err != nil {
		errorResponse(c, http.StatusInternalServerError, "AI_SERVICE_ERROR", "Failed to improve description", err)
		return
	}

	// Create response
	response := ImproveDescriptionResponse{
		EnhancedDescription: enhancedDescription,
		Suggestions: []string{
			"Consider adding specific deadlines",
			"Include measurable success criteria",
			"Add relevant context or background",
		},
		Confidence: 0.85,
	}

	successResponse(c, http.StatusOK, "Description improved successfully", response)
}

// AcceptAISuggestion accepts an AI suggestion
// @Summary Accept AI suggestion
// @Description Marks an AI suggestion as accepted by the user
// @Tags AI
// @Accept json
// @Produce json
// @Param request body AcceptSuggestionRequest true "Accept suggestion request"
// @Success 200 {object} successResponse
// @Failure 400 {object} errorResponse
// @Failure 401 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Router /api/v1/ai/suggestions/accept [post]
func (h *AIHandler) AcceptAISuggestion(c *gin.Context) {
	userID := getUserIDFromContext(c)
	if userID == "" {
		errorResponse(c, http.StatusUnauthorized, "AUTH_REQUIRED", "Authentication required", nil)
		return
	}

	var req AcceptSuggestionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errorResponse(c, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request format", err)
		return
	}

	err := h.aiService.AcceptAISuggestion(c.Request.Context(), userID, req.SuggestionID)
	if err != nil {
		errorResponse(c, http.StatusInternalServerError, "SUGGESTION_ACCEPT_FAILED", "Failed to accept suggestion", err)
		return
	}

	successResponse(c, http.StatusOK, "AI suggestion accepted successfully", nil)
}

// RejectAISuggestion rejects an AI suggestion with optional feedback
// @Summary Reject AI suggestion
// @Description Marks an AI suggestion as rejected by the user with optional feedback
// @Tags AI
// @Accept json
// @Produce json
// @Param request body RejectSuggestionRequest true "Reject suggestion request"
// @Success 200 {object} successResponse
// @Failure 400 {object} errorResponse
// @Failure 401 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Router /api/v1/ai/suggestions/reject [post]
func (h *AIHandler) RejectAISuggestion(c *gin.Context) {
	userID := getUserIDFromContext(c)
	if userID == "" {
		errorResponse(c, http.StatusUnauthorized, "AUTH_REQUIRED", "Authentication required", nil)
		return
	}

	var req RejectSuggestionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errorResponse(c, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request format", err)
		return
	}

	err := h.aiService.RejectAISuggestion(c.Request.Context(), userID, req.SuggestionID, req.Feedback)
	if err != nil {
		errorResponse(c, http.StatusInternalServerError, "SUGGESTION_REJECT_FAILED", "Failed to reject suggestion", err)
		return
	}

	successResponse(c, http.StatusOK, "AI suggestion rejected successfully", nil)
}

// GetAIMetrics returns AI usage metrics for the authenticated user
// @Summary Get AI usage metrics
// @Description Returns comprehensive AI usage metrics and statistics for the user
// @Tags AI
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Router /api/v1/ai/metrics [get]
func (h *AIHandler) GetAIMetrics(c *gin.Context) {
	userID := getUserIDFromContext(c)
	if userID == "" {
		errorResponse(c, http.StatusUnauthorized, "AUTH_REQUIRED", "Authentication required", nil)
		return
	}

	metrics, err := h.aiService.GetAIMetrics(c.Request.Context(), userID)
	if err != nil {
		errorResponse(c, http.StatusInternalServerError, "METRICS_FETCH_FAILED", "Failed to fetch AI metrics", err)
		return
	}

	successResponse(c, http.StatusOK, "AI metrics retrieved successfully", metrics)
}

// GetAISuggestions returns pending AI suggestions for the user
// @Summary Get AI suggestions
// @Description Returns all pending AI suggestions for the authenticated user
// @Tags AI
// @Produce json
// @Param limit query int false "Limit number of suggestions" default(20)
// @Param offset query int false "Offset for pagination" default(0)
// @Success 200 {array} repository.AISuggestion
// @Failure 401 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Router /api/v1/ai/suggestions [get]
func (h *AIHandler) GetAISuggestions(c *gin.Context) {
	userID := getUserIDFromContext(c)
	if userID == "" {
		errorResponse(c, http.StatusUnauthorized, "AUTH_REQUIRED", "Authentication required", nil)
		return
	}

	// Parse pagination parameters
	limitStr := c.DefaultQuery("limit", "20")
	offsetStr := c.DefaultQuery("offset", "0")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 || limit > 100 {
		limit = 20
	}

	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = 0
	}

	// This would need to be implemented in the service layer
	// suggestions, err := h.aiService.GetPendingSuggestions(c.Request.Context(), userID, limit, offset)
	// For now, return empty response
	suggestions := []interface{}{}

	successResponse(c, http.StatusOK, "AI suggestions retrieved successfully", map[string]interface{}{
		"suggestions": suggestions,
		"pagination": map[string]int{
			"limit":  limit,
			"offset": offset,
			"total":  0,
		},
	})
}

// RegenerateTaskEnhancements regenerates AI enhancements for an existing task
// @Summary Regenerate task AI enhancements
// @Description Regenerates AI enhancements (description, subtasks, metadata) for an existing task
// @Tags AI
// @Accept json
// @Produce json
// @Param id path string true "Task ID"
// @Success 200 {object} service.AIEnhancedTask
// @Failure 400 {object} errorResponse
// @Failure 401 {object} errorResponse
// @Failure 404 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Router /api/v1/ai/tasks/{id}/regenerate [post]
func (h *AIHandler) RegenerateTaskEnhancements(c *gin.Context) {
	userID := getUserIDFromContext(c)
	if userID == "" {
		errorResponse(c, http.StatusUnauthorized, "AUTH_REQUIRED", "Authentication required", nil)
		return
	}

	taskID := c.Param("id")
	if taskID == "" {
		errorResponse(c, http.StatusBadRequest, "INVALID_TASK_ID", "Task ID is required", nil)
		return
	}

	// This would need to be implemented in the service layer
	// enhancedTask, err := h.aiService.RegenerateTaskEnhancements(c.Request.Context(), userID, taskID)

	errorResponse(c, http.StatusNotImplemented, "NOT_IMPLEMENTED", "Regeneration feature is under development", nil)
}

// ImproveDescriptionResponse represents the AI-improved description response
type ImproveDescriptionResponse struct {
	EnhancedDescription string   `json:"enhanced_description"`
	Suggestions         []string `json:"suggestions"`
	Confidence          float64  `json:"confidence"`
}
