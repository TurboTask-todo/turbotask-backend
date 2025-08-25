package service

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"time"

	"macwrite-auth-api/internal/config"
	"macwrite-auth-api/internal/entity"
	"macwrite-auth-api/internal/repository"
	"macwrite-auth-api/pkg/compression"
	"macwrite-auth-api/pkg/queue"
	"macwrite-auth-api/pkg/redis"

	"github.com/google/uuid"
)

// AIConversationService defines the interface for AI conversation business logic
type AIConversationService interface {
	// Conversation operations
	CreateConversation(ctx context.Context, userID uuid.UUID, req entity.CreateConversationRequest) (*entity.AIConversation, error)
	GetConversation(ctx context.Context, userID uuid.UUID, conversationID uuid.UUID) (*entity.AIConversation, error)
	GetConversations(ctx context.Context, query entity.GetConversationsQuery) ([]entity.AIConversation, int, error)
	UpdateConversation(ctx context.Context, userID uuid.UUID, conversationID uuid.UUID, req entity.UpdateConversationRequest) (*entity.AIConversation, error)
	DeleteConversation(ctx context.Context, userID uuid.UUID, conversationID uuid.UUID) error
	ArchiveConversation(ctx context.Context, userID uuid.UUID, conversationID uuid.UUID) error
	GetConversationWithInteractions(ctx context.Context, userID uuid.UUID, conversationID uuid.UUID) (*entity.ConversationWithInteractions, error)

	// Interaction operations
	CreateInteraction(ctx context.Context, userID uuid.UUID, req entity.CreateInteractionRequest, clientIP, userAgent, requestID string) (*entity.InteractionResponse, error)
	GetInteraction(ctx context.Context, userID uuid.UUID, interactionID uuid.UUID) (*entity.AIInteraction, error)
	ProcessInteractionAsync(ctx context.Context, interactionID uuid.UUID) error

	// Feedback operations
	CreateFeedback(ctx context.Context, userID uuid.UUID, req entity.CreateFeedbackRequest) (*entity.AIInteractionFeedback, error)
	GetFeedback(ctx context.Context, userID uuid.UUID, interactionID uuid.UUID) ([]entity.AIInteractionFeedback, error)

	// Analytics operations
	GetAnalytics(ctx context.Context, userID uuid.UUID, query entity.GetAnalyticsQuery) (*entity.AnalyticsResponse, error)
	GetUserSummary(ctx context.Context, userID uuid.UUID) (*entity.AnalyticsSummary, error)

	// Search operations
	SearchConversations(ctx context.Context, userID uuid.UUID, searchQuery string, limit, offset int) ([]entity.AIConversation, error)

	// Cache operations
	InvalidateUserCache(ctx context.Context, userID uuid.UUID) error
	WarmCache(ctx context.Context, userID uuid.UUID) error

	// Health and monitoring
	GetSystemHealth(ctx context.Context) (map[string]interface{}, error)
}

// aiConversationService implements AIConversationService
type aiConversationService struct {
	repo        repository.AIConversationRepository
	redisClient redis.Client
	queueClient queue.Client
	compressor  compression.Compressor
	config      *config.Config
}

// NewAIConversationService creates a new AI conversation service
func NewAIConversationService(
	repo repository.AIConversationRepository,
	redisClient redis.Client,
	queueClient queue.Client,
	compressor compression.Compressor,
	config *config.Config,
) AIConversationService {
	return &aiConversationService{
		repo:        repo,
		redisClient: redisClient,
		queueClient: queueClient,
		compressor:  compressor,
		config:      config,
	}
}

// CreateConversation creates a new conversation
func (s *aiConversationService) CreateConversation(ctx context.Context, userID uuid.UUID, req entity.CreateConversationRequest) (*entity.AIConversation, error) {
	// Check if conversation with session ID already exists
	existingConv, err := s.repo.GetConversationBySessionID(ctx, userID, req.SessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing conversation: %w", err)
	}
	if existingConv != nil {
		return existingConv, nil
	}

	// Create new conversation
	conversation := &entity.AIConversation{
		UserID:           userID,
		SessionID:        req.SessionID,
		Title:            &req.Title,
		ModelName:        req.ModelName,
		ModelVersion:     &req.ModelVersion,
		ConversationType: req.ConversationType,
		Priority:         req.Priority,
		Metadata:         req.Metadata,
		Tags:             req.Tags,
	}

	// Generate cache key
	cacheKey := s.generateConversationCacheKey(userID, req.SessionID)
	conversation.CacheKey = &cacheKey

	err = s.repo.CreateConversation(ctx, conversation)
	if err != nil {
		return nil, fmt.Errorf("failed to create conversation: %w", err)
	}

	// Cache the conversation
	go func() {
		cacheCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		s.cacheConversation(cacheCtx, conversation)
	}()

	// Invalidate user's conversation list cache
	go func() {
		cacheCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		s.invalidateUserConversationsCache(cacheCtx, userID)
	}()

	return conversation, nil
}

// GetConversation gets a conversation by ID with caching
func (s *aiConversationService) GetConversation(ctx context.Context, userID uuid.UUID, conversationID uuid.UUID) (*entity.AIConversation, error) {
	// Try cache first
	cacheKey := s.generateConversationIDCacheKey(conversationID)
	cachedConv, err := s.getCachedConversation(ctx, cacheKey)
	if err == nil && cachedConv != nil {
		// Verify user owns this conversation
		if cachedConv.UserID == userID {
			return cachedConv, nil
		}
	}

	// Get from database
	conversation, err := s.repo.GetConversationByID(ctx, conversationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get conversation: %w", err)
	}
	if conversation == nil {
		return nil, nil
	}

	// Verify user owns this conversation
	if conversation.UserID != userID {
		return nil, nil
	}

	// Cache the conversation
	go func() {
		cacheCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		s.cacheConversation(cacheCtx, conversation)
	}()

	return conversation, nil
}

// GetConversations gets conversations with caching
func (s *aiConversationService) GetConversations(ctx context.Context, query entity.GetConversationsQuery) ([]entity.AIConversation, int, error) {
	// Try cache for simple queries
	if s.isSimpleQuery(query) {
		cacheKey := s.generateUserConversationsCacheKey(query.UserID, query.Limit, query.Offset)
		cachedConvs, total, err := s.getCachedConversations(ctx, cacheKey)
		if err == nil && cachedConvs != nil {
			return cachedConvs, total, nil
		}
	}

	conversations, total, err := s.repo.GetConversations(ctx, query)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get conversations: %w", err)
	}

	// Cache simple queries
	if s.isSimpleQuery(query) {
		go func() {
			cacheCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			cacheKey := s.generateUserConversationsCacheKey(query.UserID, query.Limit, query.Offset)
			s.cacheConversations(cacheCtx, cacheKey, conversations, total)
		}()
	}

	return conversations, total, nil
}

// UpdateConversation updates a conversation
func (s *aiConversationService) UpdateConversation(ctx context.Context, userID uuid.UUID, conversationID uuid.UUID, req entity.UpdateConversationRequest) (*entity.AIConversation, error) {
	// Verify user owns this conversation
	conversation, err := s.GetConversation(ctx, userID, conversationID)
	if err != nil {
		return nil, err
	}
	if conversation == nil {
		return nil, fmt.Errorf("conversation not found")
	}

	err = s.repo.UpdateConversation(ctx, conversationID, req)
	if err != nil {
		return nil, fmt.Errorf("failed to update conversation: %w", err)
	}

	// Get updated conversation
	updatedConversation, err := s.repo.GetConversationByID(ctx, conversationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get updated conversation: %w", err)
	}

	// Invalidate caches
	go func() {
		cacheCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		s.invalidateConversationCache(cacheCtx, conversationID)
		s.invalidateUserConversationsCache(cacheCtx, userID)
	}()

	return updatedConversation, nil
}

// DeleteConversation deletes a conversation
func (s *aiConversationService) DeleteConversation(ctx context.Context, userID uuid.UUID, conversationID uuid.UUID) error {
	// Verify user owns this conversation
	conversation, err := s.GetConversation(ctx, userID, conversationID)
	if err != nil {
		return err
	}
	if conversation == nil {
		return fmt.Errorf("conversation not found")
	}

	err = s.repo.DeleteConversation(ctx, conversationID)
	if err != nil {
		return fmt.Errorf("failed to delete conversation: %w", err)
	}

	// Invalidate caches
	go func() {
		cacheCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		s.invalidateConversationCache(cacheCtx, conversationID)
		s.invalidateUserConversationsCache(cacheCtx, userID)
	}()

	return nil
}

// ArchiveConversation archives a conversation
func (s *aiConversationService) ArchiveConversation(ctx context.Context, userID uuid.UUID, conversationID uuid.UUID) error {
	// Verify user owns this conversation
	conversation, err := s.GetConversation(ctx, userID, conversationID)
	if err != nil {
		return err
	}
	if conversation == nil {
		return fmt.Errorf("conversation not found")
	}

	err = s.repo.ArchiveConversation(ctx, conversationID)
	if err != nil {
		return fmt.Errorf("failed to archive conversation: %w", err)
	}

	// Invalidate caches
	go func() {
		cacheCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		s.invalidateConversationCache(cacheCtx, conversationID)
		s.invalidateUserConversationsCache(cacheCtx, userID)
	}()

	return nil
}

// GetConversationWithInteractions gets a conversation with all interactions
func (s *aiConversationService) GetConversationWithInteractions(ctx context.Context, userID uuid.UUID, conversationID uuid.UUID) (*entity.ConversationWithInteractions, error) {
	// Try cache first
	cacheKey := s.generateConversationWithInteractionsCacheKey(conversationID)
	cachedResult, err := s.getCachedConversationWithInteractions(ctx, cacheKey)
	if err == nil && cachedResult != nil {
		// Verify user owns this conversation
		if cachedResult.Conversation.UserID == userID {
			return cachedResult, nil
		}
	}

	// Get from database
	result, err := s.repo.GetConversationWithInteractions(ctx, conversationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get conversation with interactions: %w", err)
	}
	if result == nil {
		return nil, nil
	}

	// Verify user owns this conversation
	if result.Conversation.UserID != userID {
		return nil, nil
	}

	// Decompress interactions if needed
	for i := range result.Interactions {
		interaction := &result.Interactions[i]
		err = s.decompressInteraction(interaction)
		if err != nil {
			log.Printf("Failed to decompress interaction: %v", err)
		}
	}

	// Cache the result
	go func() {
		cacheCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		s.cacheConversationWithInteractions(cacheCtx, cacheKey, result)
	}()

	return result, nil
}

// CreateInteraction creates a new interaction and queues it for processing
func (s *aiConversationService) CreateInteraction(ctx context.Context, userID uuid.UUID, req entity.CreateInteractionRequest, clientIP, userAgent, requestID string) (*entity.InteractionResponse, error) {
	// Verify user owns the conversation
	conversation, err := s.GetConversation(ctx, userID, req.ConversationID)
	if err != nil {
		return nil, err
	}
	if conversation == nil {
		return nil, fmt.Errorf("conversation not found")
	}

	// Get next sequence number
	existingInteractions, err := s.repo.GetInteractionsByConversationID(ctx, req.ConversationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get existing interactions: %w", err)
	}
	sequenceNumber := len(existingInteractions) + 1

	// Generate hashes
	inputHash := s.generateInputHash(req.UserInput, req.ModelName, req.Temperature, req.MaxTokens)

	// Check cache for similar response
	var cacheHit bool
	var cachedResponse string
	if req.EnableCaching {
		cacheKey := s.generateResponseCacheKey(inputHash)
		cachedResponse, err = s.getCachedResponse(ctx, cacheKey)
		if err == nil && cachedResponse != "" {
			cacheHit = true
		}
	}

	// Create interaction
	interaction := &entity.AIInteraction{
		ConversationID:      req.ConversationID,
		UserID:              userID,
		SequenceNumber:      sequenceNumber,
		UserInput:           &req.UserInput,
		InputTokenCount:     s.estimateTokenCount(req.UserInput),
		InputCharacterCount: len(req.UserInput),
		InputHash:           &inputHash,
		ModelName:           req.ModelName,
		ModelVersion:        &req.ModelVersion,
		Temperature:         req.Temperature,
		MaxTokens:           req.MaxTokens,
		TopP:                req.TopP,
		FrequencyPenalty:    req.FrequencyPenalty,
		PresencePenalty:     req.PresencePenalty,
		SystemPrompt:        &req.SystemPrompt,
		QueuePriority:       req.Priority,
		CacheHit:            cacheHit,
		ClientIP:            &clientIP,
		UserAgent:           &userAgent,
		RequestID:           &requestID,
		RequestMetadata:     req.RequestMetadata,
	}

	// Handle compression if enabled
	if req.EnableCompression && len(req.UserInput) > 1024 {
		compressed, compressionType, err := s.compressor.Compress([]byte(req.UserInput))
		if err == nil && len(compressed) < len(req.UserInput) {
			interaction.UserInput = nil
			interaction.UserInputCompressed = compressed
			interaction.InputCompressionType = compressionType
		}
	}

	// If cache hit, set response immediately
	if cacheHit {
		interaction.AIResponse = &cachedResponse
		interaction.OutputTokenCount = s.estimateTokenCount(cachedResponse)
		interaction.OutputCharacterCount = len(cachedResponse)
		interaction.Status = "completed"
		interaction.ProcessingTimeMs = 0
		interaction.CompletedAt = &interaction.CreatedAt

		// Calculate cost (cached responses have no cost)
		interaction.InputCost = s.calculateInputCost(req.ModelName, interaction.InputTokenCount)
		interaction.OutputCost = 0 // No cost for cached responses
		interaction.TotalCost = interaction.InputCost
	}

	err = s.repo.CreateInteraction(ctx, interaction)
	if err != nil {
		return nil, fmt.Errorf("failed to create interaction: %w", err)
	}

	response := &entity.InteractionResponse{
		Interaction: *interaction,
	}

	// Queue for processing if not cached
	if !cacheHit {
		queueInfo, err := s.queueInteractionForProcessing(ctx, interaction)
		if err != nil {
			log.Printf("Failed to queue interaction for processing: %v", err)
		} else {
			response.QueueInfo = queueInfo
		}
	}

	// Set cache info
	if req.EnableCaching {
		response.CacheInfo = &entity.CacheInfo{
			CacheKey: s.generateResponseCacheKey(inputHash),
			CacheHit: cacheHit,
			TTL:      1800, // 30 minutes
		}
	}

	// Invalidate conversation cache
	go func() {
		cacheCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		s.invalidateConversationCache(cacheCtx, req.ConversationID)
		s.invalidateUserConversationsCache(cacheCtx, userID)
	}()

	return response, nil
}

// GetInteraction gets an interaction by ID
func (s *aiConversationService) GetInteraction(ctx context.Context, userID uuid.UUID, interactionID uuid.UUID) (*entity.AIInteraction, error) {
	interaction, err := s.repo.GetInteractionByID(ctx, interactionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get interaction: %w", err)
	}
	if interaction == nil {
		return nil, nil
	}

	// Verify user owns this interaction
	if interaction.UserID != userID {
		return nil, nil
	}

	// Decompress if needed
	err = s.decompressInteraction(interaction)
	if err != nil {
		log.Printf("Failed to decompress interaction: %v", err)
	}

	return interaction, nil
}

// ProcessInteractionAsync processes an interaction asynchronously
func (s *aiConversationService) ProcessInteractionAsync(ctx context.Context, interactionID uuid.UUID) error {
	interaction, err := s.repo.GetInteractionByID(ctx, interactionID)
	if err != nil {
		return fmt.Errorf("failed to get interaction: %w", err)
	}
	if interaction == nil {
		return fmt.Errorf("interaction not found")
	}

	// Update status to processing
	interaction.Status = "processing"
	startTime := time.Now()
	interaction.StartedProcessingAt = &startTime

	err = s.repo.UpdateInteraction(ctx, interactionID, interaction)
	if err != nil {
		return fmt.Errorf("failed to update interaction status: %w", err)
	}

	// Simulate AI processing (replace with actual AI service call)
	processingTime := time.Duration(1000+len(*interaction.UserInput)) * time.Millisecond
	time.Sleep(processingTime)

	// Generate mock response (replace with actual AI response)
	response := fmt.Sprintf("AI response to: %s", *interaction.UserInput)

	// Handle compression if needed
	if len(response) > 1024 {
		compressed, compressionType, err := s.compressor.Compress([]byte(response))
		if err == nil && len(compressed) < len(response) {
			interaction.AIResponse = nil
			interaction.AIResponseCompressed = compressed
			interaction.OutputCompressionType = compressionType
		} else {
			interaction.AIResponse = &response
		}
	} else {
		interaction.AIResponse = &response
	}

	// Update metrics
	completedTime := time.Now()
	interaction.CompletedAt = &completedTime
	interaction.ProcessingTimeMs = int(completedTime.Sub(startTime).Milliseconds())
	interaction.TotalLatencyMs = int(completedTime.Sub(interaction.CreatedAt).Milliseconds())
	interaction.OutputTokenCount = s.estimateTokenCount(response)
	interaction.OutputCharacterCount = len(response)
	interaction.Status = "completed"

	// Calculate costs
	interaction.InputCost = s.calculateInputCost(interaction.ModelName, interaction.InputTokenCount)
	interaction.OutputCost = s.calculateOutputCost(interaction.ModelName, interaction.OutputTokenCount)
	interaction.TotalCost = interaction.InputCost + interaction.OutputCost

	// Generate output hash for caching
	outputHash := s.generateOutputHash(response)
	interaction.OutputHash = &outputHash

	err = s.repo.UpdateInteraction(ctx, interactionID, interaction)
	if err != nil {
		return fmt.Errorf("failed to update interaction: %w", err)
	}

	// Cache the response
	if interaction.InputHash != nil {
		go func() {
			cacheCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			cacheKey := s.generateResponseCacheKey(*interaction.InputHash)
			s.cacheResponse(cacheCtx, cacheKey, response, 1800) // 30 minutes TTL
		}()
	}

	return nil
}

// CreateFeedback creates feedback for an interaction
func (s *aiConversationService) CreateFeedback(ctx context.Context, userID uuid.UUID, req entity.CreateFeedbackRequest) (*entity.AIInteractionFeedback, error) {
	// Verify user owns the interaction
	interaction, err := s.GetInteraction(ctx, userID, req.InteractionID)
	if err != nil {
		return nil, err
	}
	if interaction == nil {
		return nil, fmt.Errorf("interaction not found")
	}

	feedback := &entity.AIInteractionFeedback{
		InteractionID:            req.InteractionID,
		UserID:                   userID,
		OverallRating:            req.OverallRating,
		AccuracyRating:           req.AccuracyRating,
		HelpfulnessRating:        req.HelpfulnessRating,
		RelevanceRating:          req.RelevanceRating,
		FeedbackType:             req.FeedbackType,
		FeedbackText:             req.FeedbackText,
		ImprovementSuggestions:   req.ImprovementSuggestions,
		Tags:                     req.Tags,
		SessionQuality:           req.SessionQuality,
		ResponseTimeSatisfaction: req.ResponseTimeSatisfaction,
	}

	err = s.repo.CreateFeedback(ctx, feedback)
	if err != nil {
		return nil, fmt.Errorf("failed to create feedback: %w", err)
	}

	return feedback, nil
}

// GetFeedback gets feedback for an interaction
func (s *aiConversationService) GetFeedback(ctx context.Context, userID uuid.UUID, interactionID uuid.UUID) ([]entity.AIInteractionFeedback, error) {
	// Verify user owns the interaction
	interaction, err := s.GetInteraction(ctx, userID, interactionID)
	if err != nil {
		return nil, err
	}
	if interaction == nil {
		return nil, fmt.Errorf("interaction not found")
	}

	feedback, err := s.repo.GetFeedbackByInteractionID(ctx, interactionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get feedback: %w", err)
	}

	return feedback, nil
}

// GetAnalytics gets analytics for a user
func (s *aiConversationService) GetAnalytics(ctx context.Context, userID uuid.UUID, query entity.GetAnalyticsQuery) (*entity.AnalyticsResponse, error) {
	// Try cache first
	cacheKey := s.generateAnalyticsCacheKey(userID, query.FromDate, query.ToDate)
	cachedAnalytics, err := s.getCachedAnalytics(ctx, cacheKey)
	if err == nil && cachedAnalytics != nil {
		return cachedAnalytics, nil
	}

	query.UserID = userID
	analytics, err := s.repo.GetUsageAnalytics(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get analytics: %w", err)
	}

	// Cache the analytics
	go func() {
		cacheCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		s.cacheAnalytics(cacheCtx, cacheKey, analytics, 3600) // 1 hour TTL
	}()

	return analytics, nil
}

// GetUserSummary gets user summary analytics
func (s *aiConversationService) GetUserSummary(ctx context.Context, userID uuid.UUID) (*entity.AnalyticsSummary, error) {
	// Try cache first
	cacheKey := s.generateUserSummaryCacheKey(userID)
	cachedSummary, err := s.getCachedUserSummary(ctx, cacheKey)
	if err == nil && cachedSummary != nil {
		return cachedSummary, nil
	}

	summary, err := s.repo.GetUserSummary(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user summary: %w", err)
	}

	// Cache the summary
	go func() {
		cacheCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		s.cacheUserSummary(cacheCtx, cacheKey, summary, 1800) // 30 minutes TTL
	}()

	return summary, nil
}

// SearchConversations searches conversations
func (s *aiConversationService) SearchConversations(ctx context.Context, userID uuid.UUID, searchQuery string, limit, offset int) ([]entity.AIConversation, error) {
	conversations, err := s.repo.SearchConversations(ctx, userID, searchQuery, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to search conversations: %w", err)
	}

	return conversations, nil
}

// InvalidateUserCache invalidates all cache entries for a user
func (s *aiConversationService) InvalidateUserCache(ctx context.Context, userID uuid.UUID) error {
	patterns := []string{
		fmt.Sprintf("ai:conversation:user:%s:*", userID),
		fmt.Sprintf("ai:conversations:user:%s:*", userID),
		fmt.Sprintf("ai:analytics:user:%s:*", userID),
		fmt.Sprintf("ai:summary:user:%s", userID),
	}

	for _, pattern := range patterns {
		err := s.redisClient.DeletePattern(ctx, pattern)
		if err != nil {
			log.Printf("Failed to delete cache pattern %s: %v", pattern, err)
		}
	}

	return nil
}

// WarmCache warms cache for a user
func (s *aiConversationService) WarmCache(ctx context.Context, userID uuid.UUID) error {
	// Warm recent conversations
	query := entity.GetConversationsQuery{
		UserID: userID,
		Status: "active",
		Limit:  10,
		Offset: 0,
	}

	conversations, _, err := s.repo.GetConversations(ctx, query)
	if err != nil {
		return err
	}

	// Cache each conversation
	for _, conv := range conversations {
		go func(c entity.AIConversation) {
			cacheCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			s.cacheConversation(cacheCtx, &c)
		}(conv)
	}

	// Warm user summary
	go func() {
		cacheCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		s.GetUserSummary(cacheCtx, userID)
	}()

	return nil
}

// GetSystemHealth gets system health information
func (s *aiConversationService) GetSystemHealth(ctx context.Context) (map[string]interface{}, error) {
	health := make(map[string]interface{})

	// Check Redis
	redisStatus := "healthy"
	if err := s.redisClient.Ping(ctx); err != nil {
		redisStatus = "unhealthy"
	}
	health["redis"] = redisStatus

	// Check Queue
	queueStatus := "healthy"
	if err := s.queueClient.Ping(ctx); err != nil {
		queueStatus = "unhealthy"
	}
	health["queue"] = queueStatus

	// Check database by getting a simple count
	dbStatus := "healthy"
	// You might want to add a simple query to check DB health
	health["database"] = dbStatus

	health["timestamp"] = time.Now()
	health["version"] = s.config.App.Version

	return health, nil
}

// Helper methods for caching, compression, and queue operations...

// Cache helper methods
func (s *aiConversationService) generateConversationCacheKey(userID uuid.UUID, sessionID string) string {
	return fmt.Sprintf("ai:conversation:user:%s:session:%s", userID, sessionID)
}

func (s *aiConversationService) generateConversationIDCacheKey(conversationID uuid.UUID) string {
	return fmt.Sprintf("ai:conversation:id:%s", conversationID)
}

func (s *aiConversationService) generateUserConversationsCacheKey(userID uuid.UUID, limit, offset int) string {
	return fmt.Sprintf("ai:conversations:user:%s:limit:%d:offset:%d", userID, limit, offset)
}

func (s *aiConversationService) generateConversationWithInteractionsCacheKey(conversationID uuid.UUID) string {
	return fmt.Sprintf("ai:conversation:full:%s", conversationID)
}

func (s *aiConversationService) generateResponseCacheKey(inputHash string) string {
	return fmt.Sprintf("ai:response:hash:%s", inputHash)
}

func (s *aiConversationService) generateAnalyticsCacheKey(userID uuid.UUID, fromDate, toDate string) string {
	return fmt.Sprintf("ai:analytics:user:%s:from:%s:to:%s", userID, fromDate, toDate)
}

func (s *aiConversationService) generateUserSummaryCacheKey(userID uuid.UUID) string {
	return fmt.Sprintf("ai:summary:user:%s", userID)
}

func (s *aiConversationService) generateInputHash(input, model string, temperature *float64, maxTokens *int) string {
	h := sha256.New()
	h.Write([]byte(input))
	h.Write([]byte(model))
	if temperature != nil {
		h.Write([]byte(fmt.Sprintf("%.2f", *temperature)))
	}
	if maxTokens != nil {
		h.Write([]byte(strconv.Itoa(*maxTokens)))
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}

func (s *aiConversationService) generateOutputHash(output string) string {
	h := sha256.New()
	h.Write([]byte(output))
	return fmt.Sprintf("%x", h.Sum(nil))
}

func (s *aiConversationService) estimateTokenCount(text string) int {
	// Simple estimation: 1 token ≈ 4 characters
	return len(text) / 4
}

func (s *aiConversationService) calculateInputCost(model string, tokens int) float64 {
	// Mock cost calculation - replace with actual pricing
	costPerToken := 0.0001
	switch model {
	case "gpt-4":
		costPerToken = 0.00003
	case "gpt-3.5-turbo":
		costPerToken = 0.0000015
	}
	return float64(tokens) * costPerToken
}

func (s *aiConversationService) calculateOutputCost(model string, tokens int) float64 {
	// Mock cost calculation - replace with actual pricing
	costPerToken := 0.0002
	switch model {
	case "gpt-4":
		costPerToken = 0.00006
	case "gpt-3.5-turbo":
		costPerToken = 0.000002
	}
	return float64(tokens) * costPerToken
}

func (s *aiConversationService) isSimpleQuery(query entity.GetConversationsQuery) bool {
	return query.Search == "" && len(query.Tags) == 0 &&
		query.FromDate == "" && query.ToDate == "" &&
		query.Model == "" && query.Status == ""
}

func (s *aiConversationService) decompressInteraction(interaction *entity.AIInteraction) error {
	// Decompress user input if compressed
	if interaction.UserInput == nil && len(interaction.UserInputCompressed) > 0 {
		decompressed, err := s.compressor.Decompress(interaction.UserInputCompressed, interaction.InputCompressionType)
		if err != nil {
			return err
		}
		input := string(decompressed)
		interaction.UserInput = &input
	}

	// Decompress AI response if compressed
	if interaction.AIResponse == nil && len(interaction.AIResponseCompressed) > 0 {
		decompressed, err := s.compressor.Decompress(interaction.AIResponseCompressed, interaction.OutputCompressionType)
		if err != nil {
			return err
		}
		response := string(decompressed)
		interaction.AIResponse = &response
	}

	return nil
}

// Queue helper methods
func (s *aiConversationService) queueInteractionForProcessing(ctx context.Context, interaction *entity.AIInteraction) (*entity.QueueInfo, error) {
	queueName := s.getQueueNameByPriority(interaction.QueuePriority)

	payload := map[string]interface{}{
		"interaction_id": interaction.ID,
		"user_id":        interaction.UserID,
		"priority":       interaction.QueuePriority,
		"model_name":     interaction.ModelName,
		"created_at":     interaction.CreatedAt,
	}

	messageID := uuid.New().String()
	message := &entity.QueueMessage{
		MessageID:     messageID,
		QueueName:     queueName,
		RoutingKey:    &queueName,
		ExchangeName:  &s.config.Queue.ExchangeName,
		Payload:       payload,
		Priority:      interaction.QueuePriority,
		MaxRetries:    3,
		InteractionID: &interaction.ID,
		UserID:        &interaction.UserID,
		Status:        "pending",
	}

	err := s.repo.CreateQueueMessage(ctx, message)
	if err != nil {
		return nil, err
	}

	err = s.queueClient.Publish(ctx, queueName, payload, interaction.QueuePriority)
	if err != nil {
		return nil, err
	}

	// Update interaction with queue message ID
	interaction.QueueMessageID = &messageID
	interaction.Status = "queued"
	queuedTime := time.Now()
	interaction.QueuedAt = &queuedTime

	err = s.repo.UpdateInteraction(ctx, interaction.ID, interaction)
	if err != nil {
		log.Printf("Failed to update interaction with queue info: %v", err)
	}

	return &entity.QueueInfo{
		MessageID:     messageID,
		QueueName:     queueName,
		Priority:      interaction.QueuePriority,
		EstimatedWait: s.estimateWaitTime(queueName),
	}, nil
}

func (s *aiConversationService) getQueueNameByPriority(priority int) string {
	if priority >= 8 {
		return "ai.conversation.high_priority"
	} else if priority >= 4 {
		return "ai.conversation.standard"
	} else {
		return "ai.conversation.low_priority"
	}
}

func (s *aiConversationService) estimateWaitTime(queueName string) string {
	// Mock estimation - replace with actual queue depth calculation
	switch queueName {
	case "ai.conversation.high_priority":
		return "< 30s"
	case "ai.conversation.standard":
		return "1-2 min"
	default:
		return "5-10 min"
	}
}

// Cache implementation methods (these would use Redis client)
func (s *aiConversationService) cacheConversation(ctx context.Context, conversation *entity.AIConversation) {
	data, err := json.Marshal(conversation)
	if err != nil {
		log.Printf("Failed to marshal conversation for cache: %v", err)
		return
	}

	cacheKey := s.generateConversationIDCacheKey(conversation.ID)
	err = s.redisClient.Set(ctx, cacheKey, data, time.Duration(conversation.CacheTTL)*time.Second)
	if err != nil {
		log.Printf("Failed to cache conversation: %v", err)
	}
}

func (s *aiConversationService) getCachedConversation(ctx context.Context, cacheKey string) (*entity.AIConversation, error) {
	data, err := s.redisClient.Get(ctx, cacheKey)
	if err != nil {
		return nil, err
	}

	var conversation entity.AIConversation
	err = json.Unmarshal(data, &conversation)
	if err != nil {
		return nil, err
	}

	return &conversation, nil
}

func (s *aiConversationService) cacheConversations(ctx context.Context, cacheKey string, conversations []entity.AIConversation, total int) {
	data := map[string]interface{}{
		"conversations": conversations,
		"total":         total,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		log.Printf("Failed to marshal conversations for cache: %v", err)
		return
	}

	err = s.redisClient.Set(ctx, cacheKey, jsonData, 1800*time.Second) // 30 minutes
	if err != nil {
		log.Printf("Failed to cache conversations: %v", err)
	}
}

func (s *aiConversationService) getCachedConversations(ctx context.Context, cacheKey string) ([]entity.AIConversation, int, error) {
	data, err := s.redisClient.Get(ctx, cacheKey)
	if err != nil {
		return nil, 0, err
	}

	var result struct {
		Conversations []entity.AIConversation `json:"conversations"`
		Total         int                     `json:"total"`
	}

	err = json.Unmarshal(data, &result)
	if err != nil {
		return nil, 0, err
	}

	return result.Conversations, result.Total, nil
}

func (s *aiConversationService) cacheConversationWithInteractions(ctx context.Context, cacheKey string, data *entity.ConversationWithInteractions) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		log.Printf("Failed to marshal conversation with interactions for cache: %v", err)
		return
	}

	err = s.redisClient.Set(ctx, cacheKey, jsonData, 1800*time.Second) // 30 minutes
	if err != nil {
		log.Printf("Failed to cache conversation with interactions: %v", err)
	}
}

func (s *aiConversationService) getCachedConversationWithInteractions(ctx context.Context, cacheKey string) (*entity.ConversationWithInteractions, error) {
	data, err := s.redisClient.Get(ctx, cacheKey)
	if err != nil {
		return nil, err
	}

	var result entity.ConversationWithInteractions
	err = json.Unmarshal(data, &result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

func (s *aiConversationService) cacheResponse(ctx context.Context, cacheKey, response string, ttl int) {
	err := s.redisClient.Set(ctx, cacheKey, []byte(response), time.Duration(ttl)*time.Second)
	if err != nil {
		log.Printf("Failed to cache response: %v", err)
	}
}

func (s *aiConversationService) getCachedResponse(ctx context.Context, cacheKey string) (string, error) {
	data, err := s.redisClient.Get(ctx, cacheKey)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func (s *aiConversationService) cacheAnalytics(ctx context.Context, cacheKey string, analytics *entity.AnalyticsResponse, ttl int) {
	data, err := json.Marshal(analytics)
	if err != nil {
		log.Printf("Failed to marshal analytics for cache: %v", err)
		return
	}

	err = s.redisClient.Set(ctx, cacheKey, data, time.Duration(ttl)*time.Second)
	if err != nil {
		log.Printf("Failed to cache analytics: %v", err)
	}
}

func (s *aiConversationService) getCachedAnalytics(ctx context.Context, cacheKey string) (*entity.AnalyticsResponse, error) {
	data, err := s.redisClient.Get(ctx, cacheKey)
	if err != nil {
		return nil, err
	}

	var analytics entity.AnalyticsResponse
	err = json.Unmarshal(data, &analytics)
	if err != nil {
		return nil, err
	}

	return &analytics, nil
}

func (s *aiConversationService) cacheUserSummary(ctx context.Context, cacheKey string, summary *entity.AnalyticsSummary, ttl int) {
	data, err := json.Marshal(summary)
	if err != nil {
		log.Printf("Failed to marshal user summary for cache: %v", err)
		return
	}

	err = s.redisClient.Set(ctx, cacheKey, data, time.Duration(ttl)*time.Second)
	if err != nil {
		log.Printf("Failed to cache user summary: %v", err)
	}
}

func (s *aiConversationService) getCachedUserSummary(ctx context.Context, cacheKey string) (*entity.AnalyticsSummary, error) {
	data, err := s.redisClient.Get(ctx, cacheKey)
	if err != nil {
		return nil, err
	}

	var summary entity.AnalyticsSummary
	err = json.Unmarshal(data, &summary)
	if err != nil {
		return nil, err
	}

	return &summary, nil
}

func (s *aiConversationService) invalidateConversationCache(ctx context.Context, conversationID uuid.UUID) {
	cacheKeys := []string{
		s.generateConversationIDCacheKey(conversationID),
		s.generateConversationWithInteractionsCacheKey(conversationID),
	}

	for _, key := range cacheKeys {
		err := s.redisClient.Delete(ctx, key)
		if err != nil {
			log.Printf("Failed to invalidate cache key %s: %v", key, err)
		}
	}
}

func (s *aiConversationService) invalidateUserConversationsCache(ctx context.Context, userID uuid.UUID) {
	pattern := fmt.Sprintf("ai:conversations:user:%s:*", userID)
	err := s.redisClient.DeletePattern(ctx, pattern)
	if err != nil {
		log.Printf("Failed to invalidate user conversations cache pattern %s: %v", pattern, err)
	}
}
