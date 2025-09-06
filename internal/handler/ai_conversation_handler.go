package handler

import (
	"net/http"
	"strconv"
	"time"

	"quantumtask-auth-api/internal/entity"
	"quantumtask-auth-api/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// AIConversationHandler handles AI conversation HTTP requests
type AIConversationHandler struct {
	aiService service.AIConversationService
}

// NewAIConversationHandler creates a new AI conversation handler
func NewAIConversationHandler(aiService service.AIConversationService) *AIConversationHandler {
	return &AIConversationHandler{
		aiService: aiService,
	}
}

// CreateConversation creates a new AI conversation
// @Summary Create a new AI conversation
// @Description Creates a new conversation session for AI interactions
// @Tags AI Conversations
// @Accept json
// @Produce json
// @Param request body entity.CreateConversationRequest true "Conversation creation request"
// @Success 201 {object} entity.AIConversation
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/ai/conversations [post]
func (h *AIConversationHandler) CreateConversation(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "UNAUTHORIZED",
				Message: "User not authenticated",
			},
			Timestamp: time.Now(),
		})
		return
	}

	var req entity.CreateConversationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INVALID_REQUEST",
				Message: "Invalid request body: " + err.Error(),
			},
			Timestamp: time.Now(),
		})
		return
	}

	conversation, err := h.aiService.CreateConversation(c.Request.Context(), userID.(uuid.UUID), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "CONVERSATION_CREATION_FAILED",
				Message: "Failed to create conversation: " + err.Error(),
			},
			Timestamp: time.Now(),
		})
		return
	}

	c.JSON(http.StatusCreated, SuccessResponse{
		Success:   true,
		Data:      conversation,
		Message:   "Conversation created successfully",
		Timestamp: time.Now(),
	})
}

// GetConversation gets a conversation by ID
// @Summary Get conversation by ID
// @Description Retrieves a specific conversation with all its details
// @Tags AI Conversations
// @Produce json
// @Param id path string true "Conversation ID"
// @Success 200 {object} entity.AIConversation
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/ai/conversations/{id} [get]
func (h *AIConversationHandler) GetConversation(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "UNAUTHORIZED",
				Message: "User not authenticated",
			},
			Timestamp: time.Now(),
		})
		return
	}

	conversationIDStr := c.Param("id")
	conversationID, err := uuid.Parse(conversationIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INVALID_CONVERSATION_ID",
				Message: "Invalid conversation ID format",
			},
			Timestamp: time.Now(),
		})
		return
	}

	conversation, err := h.aiService.GetConversation(c.Request.Context(), userID.(uuid.UUID), conversationID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "CONVERSATION_RETRIEVAL_FAILED",
				Message: "Failed to retrieve conversation: " + err.Error(),
			},
			Timestamp: time.Now(),
		})
		return
	}

	if conversation == nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "CONVERSATION_NOT_FOUND",
				Message: "Conversation not found",
			},
			Timestamp: time.Now(),
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success:   true,
		Data:      conversation,
		Message:   "Conversation retrieved successfully",
		Timestamp: time.Now(),
	})
}

// GetConversations gets all conversations for a user
// @Summary Get user conversations
// @Description Retrieves all conversations for the authenticated user with pagination and filtering
// @Tags AI Conversations
// @Produce json
// @Param status query string false "Filter by status" Enums(active, archived, deleted, suspended)
// @Param model query string false "Filter by model name"
// @Param from_date query string false "Filter from date (YYYY-MM-DD)"
// @Param to_date query string false "Filter to date (YYYY-MM-DD)"
// @Param search query string false "Search in conversation titles and content"
// @Param tags query []string false "Filter by tags"
// @Param limit query int false "Number of results per page" default(20)
// @Param offset query int false "Number of results to skip" default(0)
// @Param order_by query string false "Sort field" Enums(created_at, updated_at, total_cost, interaction_count)
// @Param order query string false "Sort order" Enums(asc, desc)
// @Success 200 {object} PaginatedResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/ai/conversations [get]
func (h *AIConversationHandler) GetConversations(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "UNAUTHORIZED",
				Message: "User not authenticated",
			},
			Timestamp: time.Now(),
		})
		return
	}

	var query entity.GetConversationsQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INVALID_QUERY_PARAMS",
				Message: "Invalid query parameters: " + err.Error(),
			},
			Timestamp: time.Now(),
		})
		return
	}

	query.UserID = userID.(uuid.UUID)

	// Set defaults
	if query.Limit == 0 {
		query.Limit = 20
	}
	if query.OrderBy == "" {
		query.OrderBy = "created_at"
	}
	if query.Order == "" {
		query.Order = "desc"
	}

	conversations, total, err := h.aiService.GetConversations(c.Request.Context(), query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "CONVERSATIONS_RETRIEVAL_FAILED",
				Message: "Failed to retrieve conversations: " + err.Error(),
			},
			Timestamp: time.Now(),
		})
		return
	}

	c.JSON(http.StatusOK, PaginatedResponse{
		Success: true,
		Data:    conversations,
		Pagination: PaginationInfo{
			Total:  total,
			Limit:  query.Limit,
			Offset: query.Offset,
			Pages:  (total + query.Limit - 1) / query.Limit,
		},
		Message:   "Conversations retrieved successfully",
		Timestamp: time.Now(),
	})
}

// GetConversationWithInteractions gets a conversation with all its interactions
// @Summary Get conversation with interactions
// @Description Retrieves a conversation with all its interactions and responses
// @Tags AI Conversations
// @Produce json
// @Param id path string true "Conversation ID"
// @Success 200 {object} entity.ConversationWithInteractions
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/ai/conversations/{id}/interactions [get]
func (h *AIConversationHandler) GetConversationWithInteractions(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "UNAUTHORIZED",
				Message: "User not authenticated",
			},
			Timestamp: time.Now(),
		})
		return
	}

	conversationIDStr := c.Param("id")
	conversationID, err := uuid.Parse(conversationIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INVALID_CONVERSATION_ID",
				Message: "Invalid conversation ID format",
			},
			Timestamp: time.Now(),
		})
		return
	}

	result, err := h.aiService.GetConversationWithInteractions(c.Request.Context(), userID.(uuid.UUID), conversationID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "CONVERSATION_RETRIEVAL_FAILED",
				Message: "Failed to retrieve conversation with interactions: " + err.Error(),
			},
			Timestamp: time.Now(),
		})
		return
	}

	if result == nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "CONVERSATION_NOT_FOUND",
				Message: "Conversation not found",
			},
			Timestamp: time.Now(),
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success:   true,
		Data:      result,
		Message:   "Conversation with interactions retrieved successfully",
		Timestamp: time.Now(),
	})
}

// UpdateConversation updates a conversation
// @Summary Update conversation
// @Description Updates conversation details like title, status, priority, metadata, and tags
// @Tags AI Conversations
// @Accept json
// @Produce json
// @Param id path string true "Conversation ID"
// @Param request body entity.UpdateConversationRequest true "Conversation update request"
// @Success 200 {object} entity.AIConversation
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/ai/conversations/{id} [put]
func (h *AIConversationHandler) UpdateConversation(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "UNAUTHORIZED",
				Message: "User not authenticated",
			},
			Timestamp: time.Now(),
		})
		return
	}

	conversationIDStr := c.Param("id")
	conversationID, err := uuid.Parse(conversationIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INVALID_CONVERSATION_ID",
				Message: "Invalid conversation ID format",
			},
			Timestamp: time.Now(),
		})
		return
	}

	var req entity.UpdateConversationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INVALID_REQUEST",
				Message: "Invalid request body: " + err.Error(),
			},
			Timestamp: time.Now(),
		})
		return
	}

	conversation, err := h.aiService.UpdateConversation(c.Request.Context(), userID.(uuid.UUID), conversationID, req)
	if err != nil {
		if err.Error() == "conversation not found" {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Success: false,
				Error: ErrorDetail{
					Code:    "CONVERSATION_NOT_FOUND",
					Message: "Conversation not found",
				},
				Timestamp: time.Now(),
			})
			return
		}

		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "CONVERSATION_UPDATE_FAILED",
				Message: "Failed to update conversation: " + err.Error(),
			},
			Timestamp: time.Now(),
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success:   true,
		Data:      conversation,
		Message:   "Conversation updated successfully",
		Timestamp: time.Now(),
	})
}

// DeleteConversation deletes a conversation
// @Summary Delete conversation
// @Description Soft deletes a conversation and all its interactions
// @Tags AI Conversations
// @Produce json
// @Param id path string true "Conversation ID"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/ai/conversations/{id} [delete]
func (h *AIConversationHandler) DeleteConversation(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "UNAUTHORIZED",
				Message: "User not authenticated",
			},
			Timestamp: time.Now(),
		})
		return
	}

	conversationIDStr := c.Param("id")
	conversationID, err := uuid.Parse(conversationIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INVALID_CONVERSATION_ID",
				Message: "Invalid conversation ID format",
			},
			Timestamp: time.Now(),
		})
		return
	}

	err = h.aiService.DeleteConversation(c.Request.Context(), userID.(uuid.UUID), conversationID)
	if err != nil {
		if err.Error() == "conversation not found" {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Success: false,
				Error: ErrorDetail{
					Code:    "CONVERSATION_NOT_FOUND",
					Message: "Conversation not found",
				},
				Timestamp: time.Now(),
			})
			return
		}

		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "CONVERSATION_DELETION_FAILED",
				Message: "Failed to delete conversation: " + err.Error(),
			},
			Timestamp: time.Now(),
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success:   true,
		Data:      nil,
		Message:   "Conversation deleted successfully",
		Timestamp: time.Now(),
	})
}

// ArchiveConversation archives a conversation
// @Summary Archive conversation
// @Description Archives a conversation to keep it but mark it as inactive
// @Tags AI Conversations
// @Produce json
// @Param id path string true "Conversation ID"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/ai/conversations/{id}/archive [post]
func (h *AIConversationHandler) ArchiveConversation(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "UNAUTHORIZED",
				Message: "User not authenticated",
			},
			Timestamp: time.Now(),
		})
		return
	}

	conversationIDStr := c.Param("id")
	conversationID, err := uuid.Parse(conversationIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INVALID_CONVERSATION_ID",
				Message: "Invalid conversation ID format",
			},
			Timestamp: time.Now(),
		})
		return
	}

	err = h.aiService.ArchiveConversation(c.Request.Context(), userID.(uuid.UUID), conversationID)
	if err != nil {
		if err.Error() == "conversation not found" {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Success: false,
				Error: ErrorDetail{
					Code:    "CONVERSATION_NOT_FOUND",
					Message: "Conversation not found",
				},
				Timestamp: time.Now(),
			})
			return
		}

		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "CONVERSATION_ARCHIVE_FAILED",
				Message: "Failed to archive conversation: " + err.Error(),
			},
			Timestamp: time.Now(),
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success:   true,
		Data:      nil,
		Message:   "Conversation archived successfully",
		Timestamp: time.Now(),
	})
}

// CreateInteraction creates a new AI interaction
// @Summary Create AI interaction
// @Description Creates a new interaction within a conversation and queues it for AI processing
// @Tags AI Interactions
// @Accept json
// @Produce json
// @Param request body entity.CreateInteractionRequest true "Interaction creation request"
// @Success 201 {object} entity.InteractionResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/ai/interactions [post]
func (h *AIConversationHandler) CreateInteraction(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "UNAUTHORIZED",
				Message: "User not authenticated",
			},
			Timestamp: time.Now(),
		})
		return
	}

	var req entity.CreateInteractionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INVALID_REQUEST",
				Message: "Invalid request body: " + err.Error(),
			},
			Timestamp: time.Now(),
		})
		return
	}

	// Get client context
	clientIP := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")
	requestID := c.GetHeader("X-Request-ID")
	if requestID == "" {
		requestID = uuid.New().String()
	}

	response, err := h.aiService.CreateInteraction(c.Request.Context(), userID.(uuid.UUID), req, clientIP, userAgent, requestID)
	if err != nil {
		if err.Error() == "conversation not found" {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Success: false,
				Error: ErrorDetail{
					Code:    "CONVERSATION_NOT_FOUND",
					Message: "Conversation not found",
				},
				Timestamp: time.Now(),
			})
			return
		}

		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INTERACTION_CREATION_FAILED",
				Message: "Failed to create interaction: " + err.Error(),
			},
			Timestamp: time.Now(),
		})
		return
	}

	c.JSON(http.StatusCreated, SuccessResponse{
		Success:   true,
		Data:      response,
		Message:   "Interaction created successfully",
		Timestamp: time.Now(),
	})
}

// GetInteraction gets an interaction by ID
// @Summary Get interaction by ID
// @Description Retrieves a specific AI interaction with all its details
// @Tags AI Interactions
// @Produce json
// @Param id path string true "Interaction ID"
// @Success 200 {object} entity.AIInteraction
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/ai/interactions/{id} [get]
func (h *AIConversationHandler) GetInteraction(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "UNAUTHORIZED",
				Message: "User not authenticated",
			},
			Timestamp: time.Now(),
		})
		return
	}

	interactionIDStr := c.Param("id")
	interactionID, err := uuid.Parse(interactionIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INVALID_INTERACTION_ID",
				Message: "Invalid interaction ID format",
			},
			Timestamp: time.Now(),
		})
		return
	}

	interaction, err := h.aiService.GetInteraction(c.Request.Context(), userID.(uuid.UUID), interactionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INTERACTION_RETRIEVAL_FAILED",
				Message: "Failed to retrieve interaction: " + err.Error(),
			},
			Timestamp: time.Now(),
		})
		return
	}

	if interaction == nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INTERACTION_NOT_FOUND",
				Message: "Interaction not found",
			},
			Timestamp: time.Now(),
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success:   true,
		Data:      interaction,
		Message:   "Interaction retrieved successfully",
		Timestamp: time.Now(),
	})
}

// CreateFeedback creates feedback for an interaction
// @Summary Create feedback
// @Description Creates feedback for an AI interaction to help improve future responses
// @Tags AI Feedback
// @Accept json
// @Produce json
// @Param request body entity.CreateFeedbackRequest true "Feedback creation request"
// @Success 201 {object} entity.AIInteractionFeedback
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/ai/feedback [post]
func (h *AIConversationHandler) CreateFeedback(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "UNAUTHORIZED",
				Message: "User not authenticated",
			},
			Timestamp: time.Now(),
		})
		return
	}

	var req entity.CreateFeedbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INVALID_REQUEST",
				Message: "Invalid request body: " + err.Error(),
			},
			Timestamp: time.Now(),
		})
		return
	}

	feedback, err := h.aiService.CreateFeedback(c.Request.Context(), userID.(uuid.UUID), req)
	if err != nil {
		if err.Error() == "interaction not found" {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Success: false,
				Error: ErrorDetail{
					Code:    "INTERACTION_NOT_FOUND",
					Message: "Interaction not found",
				},
				Timestamp: time.Now(),
			})
			return
		}

		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "FEEDBACK_CREATION_FAILED",
				Message: "Failed to create feedback: " + err.Error(),
			},
			Timestamp: time.Now(),
		})
		return
	}

	c.JSON(http.StatusCreated, SuccessResponse{
		Success:   true,
		Data:      feedback,
		Message:   "Feedback created successfully",
		Timestamp: time.Now(),
	})
}

// GetFeedback gets feedback for an interaction
// @Summary Get interaction feedback
// @Description Retrieves all feedback for a specific interaction
// @Tags AI Feedback
// @Produce json
// @Param id path string true "Interaction ID"
// @Success 200 {object} []entity.AIInteractionFeedback
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/ai/interactions/{id}/feedback [get]
func (h *AIConversationHandler) GetFeedback(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "UNAUTHORIZED",
				Message: "User not authenticated",
			},
			Timestamp: time.Now(),
		})
		return
	}

	interactionIDStr := c.Param("id")
	interactionID, err := uuid.Parse(interactionIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INVALID_INTERACTION_ID",
				Message: "Invalid interaction ID format",
			},
			Timestamp: time.Now(),
		})
		return
	}

	feedback, err := h.aiService.GetFeedback(c.Request.Context(), userID.(uuid.UUID), interactionID)
	if err != nil {
		if err.Error() == "interaction not found" {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Success: false,
				Error: ErrorDetail{
					Code:    "INTERACTION_NOT_FOUND",
					Message: "Interaction not found",
				},
				Timestamp: time.Now(),
			})
			return
		}

		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "FEEDBACK_RETRIEVAL_FAILED",
				Message: "Failed to retrieve feedback: " + err.Error(),
			},
			Timestamp: time.Now(),
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success:   true,
		Data:      feedback,
		Message:   "Feedback retrieved successfully",
		Timestamp: time.Now(),
	})
}

// GetAnalytics gets analytics for the user
// @Summary Get user analytics
// @Description Retrieves analytics data for the user's AI interactions over a specified time period
// @Tags AI Analytics
// @Produce json
// @Param from_date query string true "Start date (YYYY-MM-DD)"
// @Param to_date query string true "End date (YYYY-MM-DD)"
// @Param model query string false "Filter by model name"
// @Param group_by query string false "Group by period" Enums(day, hour, model)
// @Param metrics query []string false "Specific metrics to include"
// @Success 200 {object} entity.AnalyticsResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/ai/analytics [get]
func (h *AIConversationHandler) GetAnalytics(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "UNAUTHORIZED",
				Message: "User not authenticated",
			},
			Timestamp: time.Now(),
		})
		return
	}

	var query entity.GetAnalyticsQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "INVALID_QUERY_PARAMS",
				Message: "Invalid query parameters: " + err.Error(),
			},
			Timestamp: time.Now(),
		})
		return
	}

	if query.FromDate == "" || query.ToDate == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "MISSING_DATE_PARAMS",
				Message: "from_date and to_date are required",
			},
			Timestamp: time.Now(),
		})
		return
	}

	analytics, err := h.aiService.GetAnalytics(c.Request.Context(), userID.(uuid.UUID), query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "ANALYTICS_RETRIEVAL_FAILED",
				Message: "Failed to retrieve analytics: " + err.Error(),
			},
			Timestamp: time.Now(),
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success:   true,
		Data:      analytics,
		Message:   "Analytics retrieved successfully",
		Timestamp: time.Now(),
	})
}

// GetUserSummary gets user summary analytics
// @Summary Get user summary
// @Description Retrieves a summary of the user's AI interaction statistics
// @Tags AI Analytics
// @Produce json
// @Success 200 {object} entity.AnalyticsSummary
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/ai/analytics/summary [get]
func (h *AIConversationHandler) GetUserSummary(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "UNAUTHORIZED",
				Message: "User not authenticated",
			},
			Timestamp: time.Now(),
		})
		return
	}

	summary, err := h.aiService.GetUserSummary(c.Request.Context(), userID.(uuid.UUID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "SUMMARY_RETRIEVAL_FAILED",
				Message: "Failed to retrieve user summary: " + err.Error(),
			},
			Timestamp: time.Now(),
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success:   true,
		Data:      summary,
		Message:   "User summary retrieved successfully",
		Timestamp: time.Now(),
	})
}

// SearchConversations searches conversations
// @Summary Search conversations
// @Description Searches through conversations using full-text search
// @Tags AI Conversations
// @Produce json
// @Param query query string true "Search query"
// @Param limit query int false "Number of results" default(20)
// @Param offset query int false "Number of results to skip" default(0)
// @Success 200 {object} PaginatedResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/ai/conversations/search [get]
func (h *AIConversationHandler) SearchConversations(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "UNAUTHORIZED",
				Message: "User not authenticated",
			},
			Timestamp: time.Now(),
		})
		return
	}

	searchQuery := c.Query("query")
	if searchQuery == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "MISSING_SEARCH_QUERY",
				Message: "Search query is required",
			},
			Timestamp: time.Now(),
		})
		return
	}

	limit := 20
	if limitStr := c.Query("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	offset := 0
	if offsetStr := c.Query("offset"); offsetStr != "" {
		if parsedOffset, err := strconv.Atoi(offsetStr); err == nil && parsedOffset >= 0 {
			offset = parsedOffset
		}
	}

	conversations, err := h.aiService.SearchConversations(c.Request.Context(), userID.(uuid.UUID), searchQuery, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "SEARCH_FAILED",
				Message: "Failed to search conversations: " + err.Error(),
			},
			Timestamp: time.Now(),
		})
		return
	}

	c.JSON(http.StatusOK, PaginatedResponse{
		Success: true,
		Data:    conversations,
		Pagination: PaginationInfo{
			Total:  len(conversations), // Note: Search doesn't return total count
			Limit:  limit,
			Offset: offset,
			Pages:  1,
		},
		Message:   "Search completed successfully",
		Timestamp: time.Now(),
	})
}

// InvalidateCache invalidates user cache
// @Summary Invalidate user cache
// @Description Invalidates all cache entries for the current user
// @Tags AI Cache
// @Produce json
// @Success 200 {object} SuccessResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/ai/cache/invalidate [post]
func (h *AIConversationHandler) InvalidateCache(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "UNAUTHORIZED",
				Message: "User not authenticated",
			},
			Timestamp: time.Now(),
		})
		return
	}

	err := h.aiService.InvalidateUserCache(c.Request.Context(), userID.(uuid.UUID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "CACHE_INVALIDATION_FAILED",
				Message: "Failed to invalidate cache: " + err.Error(),
			},
			Timestamp: time.Now(),
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success:   true,
		Data:      nil,
		Message:   "Cache invalidated successfully",
		Timestamp: time.Now(),
	})
}

// WarmCache warms user cache
// @Summary Warm user cache
// @Description Pre-loads frequently accessed data into cache for better performance
// @Tags AI Cache
// @Produce json
// @Success 200 {object} SuccessResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/ai/cache/warm [post]
func (h *AIConversationHandler) WarmCache(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "UNAUTHORIZED",
				Message: "User not authenticated",
			},
			Timestamp: time.Now(),
		})
		return
	}

	err := h.aiService.WarmCache(c.Request.Context(), userID.(uuid.UUID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "CACHE_WARMING_FAILED",
				Message: "Failed to warm cache: " + err.Error(),
			},
			Timestamp: time.Now(),
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success:   true,
		Data:      nil,
		Message:   "Cache warmed successfully",
		Timestamp: time.Now(),
	})
}

// GetSystemHealth gets system health information
// @Summary Get system health
// @Description Returns the health status of AI conversation system components
// @Tags AI System
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/ai/health [get]
func (h *AIConversationHandler) GetSystemHealth(c *gin.Context) {
	health, err := h.aiService.GetSystemHealth(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Success: false,
			Error: ErrorDetail{
				Code:    "HEALTH_CHECK_FAILED",
				Message: "Failed to get system health: " + err.Error(),
			},
			Timestamp: time.Now(),
		})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{
		Success:   true,
		Data:      health,
		Message:   "System health retrieved successfully",
		Timestamp: time.Now(),
	})
}

// Response structures used in handlers (add to existing response types)
type SuccessResponse struct {
	Success   bool        `json:"success"`
	Data      interface{} `json:"data,omitempty"`
	Message   string      `json:"message,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
}

type ErrorResponse struct {
	Success   bool        `json:"success"`
	Error     ErrorDetail `json:"error"`
	Timestamp time.Time   `json:"timestamp"`
}

type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type PaginatedResponse struct {
	Success    bool           `json:"success"`
	Data       interface{}    `json:"data"`
	Pagination PaginationInfo `json:"pagination"`
	Message    string         `json:"message,omitempty"`
	Timestamp  time.Time      `json:"timestamp"`
}

type PaginationInfo struct {
	Total  int `json:"total"`
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
	Pages  int `json:"pages"`
}
