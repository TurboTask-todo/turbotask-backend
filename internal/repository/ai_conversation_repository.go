package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"quantumtask-auth-api/internal/database"
	"quantumtask-auth-api/internal/entity"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// AIConversationRepository defines the interface for AI conversation operations
type AIConversationRepository interface {
	// Conversation operations
	CreateConversation(ctx context.Context, conversation *entity.AIConversation) error
	GetConversationByID(ctx context.Context, id uuid.UUID) (*entity.AIConversation, error)
	GetConversationBySessionID(ctx context.Context, userID uuid.UUID, sessionID string) (*entity.AIConversation, error)
	GetConversations(ctx context.Context, query entity.GetConversationsQuery) ([]entity.AIConversation, int, error)
	UpdateConversation(ctx context.Context, id uuid.UUID, updates entity.UpdateConversationRequest) error
	DeleteConversation(ctx context.Context, id uuid.UUID) error
	ArchiveConversation(ctx context.Context, id uuid.UUID) error

	// Interaction operations
	CreateInteraction(ctx context.Context, interaction *entity.AIInteraction) error
	GetInteractionByID(ctx context.Context, id uuid.UUID) (*entity.AIInteraction, error)
	GetInteractionsByConversationID(ctx context.Context, conversationID uuid.UUID) ([]entity.AIInteraction, error)
	UpdateInteraction(ctx context.Context, id uuid.UUID, interaction *entity.AIInteraction) error
	GetConversationWithInteractions(ctx context.Context, id uuid.UUID) (*entity.ConversationWithInteractions, error)

	// Feedback operations
	CreateFeedback(ctx context.Context, feedback *entity.AIInteractionFeedback) error
	GetFeedbackByInteractionID(ctx context.Context, interactionID uuid.UUID) ([]entity.AIInteractionFeedback, error)
	GetFeedbackByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]entity.AIInteractionFeedback, error)

	// Analytics operations
	GetUsageAnalytics(ctx context.Context, query entity.GetAnalyticsQuery) (*entity.AnalyticsResponse, error)
	GetUserSummary(ctx context.Context, userID uuid.UUID) (*entity.AnalyticsSummary, error)
	UpdateUsageAnalytics(ctx context.Context, analytics *entity.AIUsageAnalytics) error

	// Cache operations
	CreateCacheEntry(ctx context.Context, entry *entity.CacheEntry) error
	GetCacheEntry(ctx context.Context, cacheKey string) (*entity.CacheEntry, error)
	UpdateCacheEntry(ctx context.Context, cacheKey string, entry *entity.CacheEntry) error
	DeleteExpiredCacheEntries(ctx context.Context) (int, error)

	// Queue operations
	CreateQueueMessage(ctx context.Context, message *entity.QueueMessage) error
	GetQueueMessage(ctx context.Context, messageID string) (*entity.QueueMessage, error)
	UpdateQueueMessage(ctx context.Context, messageID string, message *entity.QueueMessage) error
	GetPendingQueueMessages(ctx context.Context, queueName string, limit int) ([]entity.QueueMessage, error)

	// Search operations
	SearchConversations(ctx context.Context, userID uuid.UUID, searchQuery string, limit, offset int) ([]entity.AIConversation, error)
	GetPopularInteractions(ctx context.Context, userID uuid.UUID, limit int) ([]entity.AIInteraction, error)

	// Archive operations
	ArchiveOldInteractions(ctx context.Context, daysOld int) (int, error)
	GetArchivedInteractions(ctx context.Context, conversationID uuid.UUID) ([]entity.AIInteraction, error)
}

// aiConversationRepository implements AIConversationRepository
type aiConversationRepository struct {
	db *database.DB
}

// NewAIConversationRepository creates a new AI conversation repository
func NewAIConversationRepository(db *database.DB) AIConversationRepository {
	return &aiConversationRepository{
		db: db,
	}
}

// CreateConversation creates a new conversation
func (r *aiConversationRepository) CreateConversation(ctx context.Context, conversation *entity.AIConversation) error {
	query := `
		INSERT INTO ai_conversations (
			id, user_id, session_id, title, model_name, model_version, conversation_type,
			priority, cache_key, cache_ttl, metadata, tags, created_at, updated_at
		) VALUES (
			:id, :user_id, :session_id, :title, :model_name, :model_version, :conversation_type,
			:priority, :cache_key, :cache_ttl, :metadata, :tags, :created_at, :updated_at
		)`

	// Set default values
	conversation.ID = uuid.New()
	conversation.CreatedAt = time.Now()
	conversation.UpdatedAt = time.Now()
	conversation.LastAccessedAt = time.Now()

	if conversation.Status == "" {
		conversation.Status = "active"
	}
	if conversation.ConversationType == "" {
		conversation.ConversationType = "chat"
	}
	if conversation.Priority == 0 {
		conversation.Priority = 5
	}
	if conversation.CacheTTL == 0 {
		conversation.CacheTTL = 3600
	}

	_, err := r.db.NamedExecContext(ctx, query, conversation)
	return err
}

// GetConversationByID gets a conversation by ID
func (r *aiConversationRepository) GetConversationByID(ctx context.Context, id uuid.UUID) (*entity.AIConversation, error) {
	var conversation entity.AIConversation
	query := `
		SELECT * FROM ai_conversations 
		WHERE id = $1 AND status != 'deleted'`

	err := r.db.GetContext(ctx, &conversation, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &conversation, nil
}

// GetConversationBySessionID gets a conversation by user ID and session ID
func (r *aiConversationRepository) GetConversationBySessionID(ctx context.Context, userID uuid.UUID, sessionID string) (*entity.AIConversation, error) {
	var conversation entity.AIConversation
	query := `
		SELECT * FROM ai_conversations 
		WHERE user_id = $1 AND session_id = $2 AND status != 'deleted'
		ORDER BY created_at DESC
		LIMIT 1`

	err := r.db.GetContext(ctx, &conversation, query, userID, sessionID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &conversation, nil
}

// GetConversations gets conversations with filtering and pagination
func (r *aiConversationRepository) GetConversations(ctx context.Context, query entity.GetConversationsQuery) ([]entity.AIConversation, int, error) {
	whereClause := []string{"status != 'deleted'"}
	args := []interface{}{}
	argIndex := 1

	// Add filters
	if query.UserID != uuid.Nil {
		whereClause = append(whereClause, fmt.Sprintf("user_id = $%d", argIndex))
		args = append(args, query.UserID)
		argIndex++
	}

	if query.Status != "" {
		whereClause = append(whereClause, fmt.Sprintf("status = $%d", argIndex))
		args = append(args, query.Status)
		argIndex++
	}

	if query.Model != "" {
		whereClause = append(whereClause, fmt.Sprintf("model_name = $%d", argIndex))
		args = append(args, query.Model)
		argIndex++
	}

	if query.FromDate != "" {
		whereClause = append(whereClause, fmt.Sprintf("created_at >= $%d", argIndex))
		args = append(args, query.FromDate)
		argIndex++
	}

	if query.ToDate != "" {
		whereClause = append(whereClause, fmt.Sprintf("created_at <= $%d", argIndex))
		args = append(args, query.ToDate)
		argIndex++
	}

	if len(query.Tags) > 0 {
		whereClause = append(whereClause, fmt.Sprintf("tags && $%d", argIndex))
		args = append(args, pq.StringArray(query.Tags))
		argIndex++
	}

	// Handle search
	if query.Search != "" {
		whereClause = append(whereClause, fmt.Sprintf("(title ILIKE $%d OR metadata::text ILIKE $%d)", argIndex, argIndex))
		args = append(args, "%"+query.Search+"%")
		argIndex++
	}

	whereSQL := strings.Join(whereClause, " AND ")

	// Count total records
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM ai_conversations WHERE %s", whereSQL)
	var total int
	err := r.db.GetContext(ctx, &total, countQuery, args...)
	if err != nil {
		return nil, 0, err
	}

	// Build order clause
	orderBy := "created_at"
	if query.OrderBy != "" {
		orderBy = query.OrderBy
	}
	order := "DESC"
	if query.Order == "asc" {
		order = "ASC"
	}

	// Set defaults for pagination
	if query.Limit == 0 {
		query.Limit = 20
	}

	// Get records with pagination
	dataQuery := fmt.Sprintf(`
		SELECT c.*, 
		       i.created_at as last_interaction_at,
		       LEFT(COALESCE(i.ai_response, ''), 200) as last_response_preview
		FROM ai_conversations c
		LEFT JOIN LATERAL (
		    SELECT created_at, ai_response
		    FROM ai_interactions 
		    WHERE conversation_id = c.id 
		    ORDER BY sequence_number DESC 
		    LIMIT 1
		) i ON true
		WHERE %s
		ORDER BY c.%s %s
		LIMIT $%d OFFSET $%d`,
		whereSQL, orderBy, order, argIndex, argIndex+1)

	args = append(args, query.Limit, query.Offset)

	var conversations []entity.AIConversation
	err = r.db.SelectContext(ctx, &conversations, dataQuery, args...)
	if err != nil {
		return nil, 0, err
	}

	return conversations, total, nil
}

// UpdateConversation updates a conversation
func (r *aiConversationRepository) UpdateConversation(ctx context.Context, id uuid.UUID, updates entity.UpdateConversationRequest) error {
	setParts := []string{"updated_at = NOW()"}
	args := []interface{}{}
	argIndex := 1

	if updates.Title != nil {
		setParts = append(setParts, fmt.Sprintf("title = $%d", argIndex))
		args = append(args, *updates.Title)
		argIndex++
	}

	if updates.Status != nil {
		setParts = append(setParts, fmt.Sprintf("status = $%d", argIndex))
		args = append(args, *updates.Status)
		argIndex++

		if *updates.Status == "archived" {
			setParts = append(setParts, "archived_at = NOW()")
		}
	}

	if updates.Priority != nil {
		setParts = append(setParts, fmt.Sprintf("priority = $%d", argIndex))
		args = append(args, *updates.Priority)
		argIndex++
	}

	if updates.Metadata != nil {
		setParts = append(setParts, fmt.Sprintf("metadata = $%d", argIndex))
		args = append(args, updates.Metadata)
		argIndex++
	}

	if updates.Tags != nil {
		setParts = append(setParts, fmt.Sprintf("tags = $%d", argIndex))
		args = append(args, pq.StringArray(updates.Tags))
		argIndex++
	}

	if len(setParts) == 1 { // Only updated_at
		return nil
	}

	query := fmt.Sprintf(`
		UPDATE ai_conversations 
		SET %s 
		WHERE id = $%d AND status != 'deleted'`,
		strings.Join(setParts, ", "), argIndex)

	args = append(args, id)

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}

// DeleteConversation soft deletes a conversation
func (r *aiConversationRepository) DeleteConversation(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE ai_conversations 
		SET status = 'deleted', updated_at = NOW() 
		WHERE id = $1 AND status != 'deleted'`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}

// ArchiveConversation archives a conversation
func (r *aiConversationRepository) ArchiveConversation(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE ai_conversations 
		SET status = 'archived', archived_at = NOW(), updated_at = NOW() 
		WHERE id = $1 AND status = 'active'`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}

// CreateInteraction creates a new interaction
func (r *aiConversationRepository) CreateInteraction(ctx context.Context, interaction *entity.AIInteraction) error {
	// Set default values
	interaction.ID = uuid.New()
	interaction.CreatedAt = time.Now()

	if interaction.Status == "" {
		interaction.Status = "pending"
	}
	if interaction.QueuePriority == 0 {
		interaction.QueuePriority = 5
	}

	query := `
		INSERT INTO ai_interactions (
			id, conversation_id, user_id, sequence_number, parent_interaction_id,
			user_input, user_input_compressed, input_compression_type, input_token_count, input_character_count, input_hash,
			ai_response, ai_response_compressed, output_compression_type, output_token_count, output_character_count, output_hash,
			model_name, model_version, temperature, max_tokens, top_p, frequency_penalty, presence_penalty, stop_sequences, system_prompt,
			processing_time_ms, queue_time_ms, total_latency_ms, retry_count,
			input_cost, output_cost, total_cost,
			queue_message_id, queue_priority, cache_key, cache_hit,
			status, error_code, error_message,
			client_ip, user_agent, request_id, trace_id,
			request_metadata, response_metadata, processing_metadata,
			created_at, queued_at, started_processing_at, completed_at, cached_at
		) VALUES (
			:id, :conversation_id, :user_id, :sequence_number, :parent_interaction_id,
			:user_input, :user_input_compressed, :input_compression_type, :input_token_count, :input_character_count, :input_hash,
			:ai_response, :ai_response_compressed, :output_compression_type, :output_token_count, :output_character_count, :output_hash,
			:model_name, :model_version, :temperature, :max_tokens, :top_p, :frequency_penalty, :presence_penalty, :stop_sequences, :system_prompt,
			:processing_time_ms, :queue_time_ms, :total_latency_ms, :retry_count,
			:input_cost, :output_cost, :total_cost,
			:queue_message_id, :queue_priority, :cache_key, :cache_hit,
			:status, :error_code, :error_message,
			:client_ip, :user_agent, :request_id, :trace_id,
			:request_metadata, :response_metadata, :processing_metadata,
			:created_at, :queued_at, :started_processing_at, :completed_at, :cached_at
		)`

	_, err := r.db.NamedExecContext(ctx, query, interaction)
	return err
}

// GetInteractionByID gets an interaction by ID
func (r *aiConversationRepository) GetInteractionByID(ctx context.Context, id uuid.UUID) (*entity.AIInteraction, error) {
	var interaction entity.AIInteraction
	query := `
		SELECT * FROM ai_interactions 
		WHERE id = $1`

	err := r.db.GetContext(ctx, &interaction, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &interaction, nil
}

// GetInteractionsByConversationID gets all interactions for a conversation
func (r *aiConversationRepository) GetInteractionsByConversationID(ctx context.Context, conversationID uuid.UUID) ([]entity.AIInteraction, error) {
	var interactions []entity.AIInteraction
	query := `
		SELECT * FROM ai_interactions 
		WHERE conversation_id = $1 
		ORDER BY sequence_number ASC`

	err := r.db.SelectContext(ctx, &interactions, query, conversationID)
	if err != nil {
		return nil, err
	}

	return interactions, nil
}

// UpdateInteraction updates an interaction
func (r *aiConversationRepository) UpdateInteraction(ctx context.Context, id uuid.UUID, interaction *entity.AIInteraction) error {
	query := `
		UPDATE ai_interactions SET
			ai_response = :ai_response,
			ai_response_compressed = :ai_response_compressed,
			output_compression_type = :output_compression_type,
			output_token_count = :output_token_count,
			output_character_count = :output_character_count,
			output_hash = :output_hash,
			processing_time_ms = :processing_time_ms,
			queue_time_ms = :queue_time_ms,
			total_latency_ms = :total_latency_ms,
			retry_count = :retry_count,
			input_cost = :input_cost,
			output_cost = :output_cost,
			total_cost = :total_cost,
			cache_hit = :cache_hit,
			status = :status,
			error_code = :error_code,
			error_message = :error_message,
			response_metadata = :response_metadata,
			processing_metadata = :processing_metadata,
			started_processing_at = :started_processing_at,
			completed_at = :completed_at,
			cached_at = :cached_at
		WHERE id = :id`

	interaction.ID = id
	_, err := r.db.NamedExecContext(ctx, query, interaction)
	return err
}

// GetConversationWithInteractions gets a conversation with all its interactions
func (r *aiConversationRepository) GetConversationWithInteractions(ctx context.Context, id uuid.UUID) (*entity.ConversationWithInteractions, error) {
	// Get conversation
	conversation, err := r.GetConversationByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if conversation == nil {
		return nil, nil
	}

	// Get interactions
	interactions, err := r.GetInteractionsByConversationID(ctx, id)
	if err != nil {
		return nil, err
	}

	return &entity.ConversationWithInteractions{
		Conversation: *conversation,
		Interactions: interactions,
	}, nil
}

// CreateFeedback creates new feedback
func (r *aiConversationRepository) CreateFeedback(ctx context.Context, feedback *entity.AIInteractionFeedback) error {
	feedback.ID = uuid.New()
	feedback.CreatedAt = time.Now()

	query := `
		INSERT INTO ai_interaction_feedback (
			id, interaction_id, user_id, overall_rating, accuracy_rating, helpfulness_rating, relevance_rating,
			feedback_type, feedback_text, improvement_suggestions, tags, session_quality, response_time_satisfaction, created_at
		) VALUES (
			:id, :interaction_id, :user_id, :overall_rating, :accuracy_rating, :helpfulness_rating, :relevance_rating,
			:feedback_type, :feedback_text, :improvement_suggestions, :tags, :session_quality, :response_time_satisfaction, :created_at
		)`

	_, err := r.db.NamedExecContext(ctx, query, feedback)
	return err
}

// GetFeedbackByInteractionID gets feedback for an interaction
func (r *aiConversationRepository) GetFeedbackByInteractionID(ctx context.Context, interactionID uuid.UUID) ([]entity.AIInteractionFeedback, error) {
	var feedback []entity.AIInteractionFeedback
	query := `
		SELECT * FROM ai_interaction_feedback 
		WHERE interaction_id = $1 
		ORDER BY created_at DESC`

	err := r.db.SelectContext(ctx, &feedback, query, interactionID)
	if err != nil {
		return nil, err
	}

	return feedback, nil
}

// GetFeedbackByUserID gets feedback for a user
func (r *aiConversationRepository) GetFeedbackByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]entity.AIInteractionFeedback, error) {
	var feedback []entity.AIInteractionFeedback
	query := `
		SELECT * FROM ai_interaction_feedback 
		WHERE user_id = $1 
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`

	err := r.db.SelectContext(ctx, &feedback, query, userID, limit, offset)
	if err != nil {
		return nil, err
	}

	return feedback, nil
}

// Additional repository methods will be implemented in the next part...

// SearchConversations searches conversations using full-text search
func (r *aiConversationRepository) SearchConversations(ctx context.Context, userID uuid.UUID, searchQuery string, limit, offset int) ([]entity.AIConversation, error) {
	var conversations []entity.AIConversation
	query := `
		SELECT c.*, ts_rank(search_vector.tsv, plainto_tsquery($2)) as relevance_score
		FROM ai_conversations c
		JOIN LATERAL (
		    SELECT to_tsvector('english', 
		        COALESCE(c.title, '') || ' ' ||
		        string_agg(COALESCE(i.user_input, ''), ' ') || ' ' ||
		        string_agg(COALESCE(i.ai_response, ''), ' ')
		    ) as tsv
		    FROM ai_interactions i
		    WHERE i.conversation_id = c.id
		    AND i.status = 'completed'
		) search_vector ON true
		WHERE c.user_id = $1
		AND c.status = 'active'
		AND search_vector.tsv @@ plainto_tsquery($2)
		ORDER BY relevance_score DESC, c.updated_at DESC
		LIMIT $3 OFFSET $4`

	err := r.db.SelectContext(ctx, &conversations, query, userID, searchQuery, limit, offset)
	if err != nil {
		return nil, err
	}

	return conversations, nil
}

// GetUsageAnalytics gets usage analytics
func (r *aiConversationRepository) GetUsageAnalytics(ctx context.Context, query entity.GetAnalyticsQuery) (*entity.AnalyticsResponse, error) {
	// Parse date range
	fromDate, err := time.Parse("2006-01-02", query.FromDate)
	if err != nil {
		return nil, fmt.Errorf("invalid from_date format: %w", err)
	}
	toDate, err := time.Parse("2006-01-02", query.ToDate)
	if err != nil {
		return nil, fmt.Errorf("invalid to_date format: %w", err)
	}

	response := &entity.AnalyticsResponse{
		UserID: query.UserID,
		DateRange: entity.DateRange{
			From: fromDate,
			To:   toDate,
		},
	}

	// Get summary
	summaryQuery := `
		SELECT 
		    COUNT(*) as total_interactions,
		    SUM(total_cost) as total_cost,
		    SUM(input_token_count) as total_input_tokens,
		    SUM(output_token_count) as total_output_tokens,
		    ROUND(AVG(processing_time_ms)) as avg_processing_time,
		    ROUND(COUNT(*) FILTER (WHERE status = 'completed') * 100.0 / COUNT(*), 2) as success_rate,
		    ROUND(COUNT(*) FILTER (WHERE cache_hit = true) * 100.0 / COUNT(*), 2) as cache_hit_rate,
		    COUNT(DISTINCT conversation_id) as unique_conversations
		FROM ai_interactions
		WHERE user_id = $1 
		AND created_at >= $2 
		AND created_at <= $3`

	var summary entity.AnalyticsSummary
	err = r.db.GetContext(ctx, &summary, summaryQuery, query.UserID, fromDate, toDate)
	if err != nil {
		return nil, err
	}
	response.Summary = summary

	// Get time series data
	timeSeriesQuery := `
		SELECT 
		    DATE(created_at) as timestamp,
		    COUNT(*) as interactions,
		    SUM(total_cost) as cost,
		    SUM(input_token_count) as input_tokens,
		    SUM(output_token_count) as output_tokens,
		    ROUND(AVG(processing_time_ms)) as avg_processing_time_ms,
		    ROUND(COUNT(*) FILTER (WHERE status = 'completed') * 100.0 / COUNT(*), 2) as success_rate,
		    ROUND(COUNT(*) FILTER (WHERE cache_hit = true) * 100.0 / COUNT(*), 2) as cache_hit_rate
		FROM ai_interactions
		WHERE user_id = $1 
		AND created_at >= $2 
		AND created_at <= $3
		GROUP BY DATE(created_at)
		ORDER BY timestamp`

	err = r.db.SelectContext(ctx, &response.TimeSeries, timeSeriesQuery, query.UserID, fromDate, toDate)
	if err != nil {
		return nil, err
	}

	return response, nil
}

// GetUserSummary gets user summary analytics
func (r *aiConversationRepository) GetUserSummary(ctx context.Context, userID uuid.UUID) (*entity.AnalyticsSummary, error) {
	var summary entity.AnalyticsSummary
	query := `
		SELECT 
		    COUNT(*) as total_interactions,
		    SUM(total_cost) as total_cost,
		    SUM(input_token_count) as total_input_tokens,
		    SUM(output_token_count) as total_output_tokens,
		    ROUND(AVG(processing_time_ms)) as avg_processing_time,
		    ROUND(COUNT(*) FILTER (WHERE status = 'completed') * 100.0 / COUNT(*), 2) as success_rate,
		    ROUND(COUNT(*) FILTER (WHERE cache_hit = true) * 100.0 / COUNT(*), 2) as cache_hit_rate,
		    COUNT(DISTINCT conversation_id) as unique_conversations
		FROM ai_interactions
		WHERE user_id = $1`

	err := r.db.GetContext(ctx, &summary, query, userID)
	if err != nil {
		return nil, err
	}

	return &summary, nil
}

// UpdateUsageAnalytics updates usage analytics
func (r *aiConversationRepository) UpdateUsageAnalytics(ctx context.Context, analytics *entity.AIUsageAnalytics) error {
	analytics.ID = uuid.New()
	analytics.CreatedAt = time.Now()
	analytics.UpdatedAt = time.Now()

	query := `
		INSERT INTO ai_usage_analytics (
			id, user_id, date, hour, model_name, interaction_count, total_input_tokens, total_output_tokens, total_cost,
			avg_processing_time_ms, avg_queue_time_ms, success_rate, cache_hit_rate, conversation_types, peak_usage_minute,
			created_at, updated_at
		) VALUES (
			:id, :user_id, :date, :hour, :model_name, :interaction_count, :total_input_tokens, :total_output_tokens, :total_cost,
			:avg_processing_time_ms, :avg_queue_time_ms, :success_rate, :cache_hit_rate, :conversation_types, :peak_usage_minute,
			:created_at, :updated_at
		) ON CONFLICT (user_id, date, hour, model_name) 
		DO UPDATE SET
			interaction_count = EXCLUDED.interaction_count,
			total_input_tokens = EXCLUDED.total_input_tokens,
			total_output_tokens = EXCLUDED.total_output_tokens,
			total_cost = EXCLUDED.total_cost,
			avg_processing_time_ms = EXCLUDED.avg_processing_time_ms,
			avg_queue_time_ms = EXCLUDED.avg_queue_time_ms,
			success_rate = EXCLUDED.success_rate,
			cache_hit_rate = EXCLUDED.cache_hit_rate,
			conversation_types = EXCLUDED.conversation_types,
			peak_usage_minute = EXCLUDED.peak_usage_minute,
			updated_at = NOW()`

	_, err := r.db.NamedExecContext(ctx, query, analytics)
	return err
}

// Cache and Queue operations...

// CreateCacheEntry creates a cache entry
func (r *aiConversationRepository) CreateCacheEntry(ctx context.Context, entry *entity.CacheEntry) error {
	entry.ID = uuid.New()
	entry.CreatedAt = time.Now()
	entry.LastAccessedAt = time.Now()

	query := `
		INSERT INTO cache_entries (
			id, cache_key, content_hash, content_type, size_bytes, hit_count, miss_count, ttl_seconds, tags,
			created_at, last_accessed_at, expires_at
		) VALUES (
			:id, :cache_key, :content_hash, :content_type, :size_bytes, :hit_count, :miss_count, :ttl_seconds, :tags,
			:created_at, :last_accessed_at, :expires_at
		) ON CONFLICT (cache_key) DO UPDATE SET
			hit_count = cache_entries.hit_count + 1,
			last_accessed_at = NOW()`

	_, err := r.db.NamedExecContext(ctx, query, entry)
	return err
}

// GetCacheEntry gets a cache entry
func (r *aiConversationRepository) GetCacheEntry(ctx context.Context, cacheKey string) (*entity.CacheEntry, error) {
	var entry entity.CacheEntry
	query := `
		SELECT * FROM cache_entries 
		WHERE cache_key = $1 
		AND (expires_at IS NULL OR expires_at > NOW())`

	err := r.db.GetContext(ctx, &entry, query, cacheKey)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &entry, nil
}

// UpdateCacheEntry updates a cache entry
func (r *aiConversationRepository) UpdateCacheEntry(ctx context.Context, cacheKey string, entry *entity.CacheEntry) error {
	query := `
		UPDATE cache_entries SET
			hit_count = hit_count + 1,
			last_accessed_at = NOW()
		WHERE cache_key = $1`

	_, err := r.db.ExecContext(ctx, query, cacheKey)
	return err
}

// DeleteExpiredCacheEntries deletes expired cache entries
func (r *aiConversationRepository) DeleteExpiredCacheEntries(ctx context.Context) (int, error) {
	query := `
		DELETE FROM cache_entries 
		WHERE expires_at IS NOT NULL AND expires_at < NOW()`

	result, err := r.db.ExecContext(ctx, query)
	if err != nil {
		return 0, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}

	return int(rowsAffected), nil
}

// CreateQueueMessage creates a queue message
func (r *aiConversationRepository) CreateQueueMessage(ctx context.Context, message *entity.QueueMessage) error {
	message.ID = uuid.New()
	message.CreatedAt = time.Now()
	message.ScheduledAt = time.Now()
	message.ExpiresAt = time.Now().Add(24 * time.Hour)

	query := `
		INSERT INTO queue_messages (
			id, message_id, queue_name, routing_key, exchange_name, payload, content_type, content_encoding,
			priority, delay_seconds, max_retries, retry_count, status, worker_id, error_message,
			interaction_id, user_id, created_at, scheduled_at, started_at, completed_at, expires_at
		) VALUES (
			:id, :message_id, :queue_name, :routing_key, :exchange_name, :payload, :content_type, :content_encoding,
			:priority, :delay_seconds, :max_retries, :retry_count, :status, :worker_id, :error_message,
			:interaction_id, :user_id, :created_at, :scheduled_at, :started_at, :completed_at, :expires_at
		)`

	_, err := r.db.NamedExecContext(ctx, query, message)
	return err
}

// GetQueueMessage gets a queue message
func (r *aiConversationRepository) GetQueueMessage(ctx context.Context, messageID string) (*entity.QueueMessage, error) {
	var message entity.QueueMessage
	query := `
		SELECT * FROM queue_messages 
		WHERE message_id = $1`

	err := r.db.GetContext(ctx, &message, query, messageID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &message, nil
}

// UpdateQueueMessage updates a queue message
func (r *aiConversationRepository) UpdateQueueMessage(ctx context.Context, messageID string, message *entity.QueueMessage) error {
	query := `
		UPDATE queue_messages SET
			status = :status,
			worker_id = :worker_id,
			error_message = :error_message,
			retry_count = :retry_count,
			started_at = :started_at,
			completed_at = :completed_at
		WHERE message_id = :message_id`

	message.MessageID = messageID
	_, err := r.db.NamedExecContext(ctx, query, message)
	return err
}

// GetPendingQueueMessages gets pending queue messages
func (r *aiConversationRepository) GetPendingQueueMessages(ctx context.Context, queueName string, limit int) ([]entity.QueueMessage, error) {
	var messages []entity.QueueMessage
	query := `
		SELECT * FROM queue_messages 
		WHERE queue_name = $1 
		AND status = 'pending'
		AND scheduled_at <= NOW()
		ORDER BY priority DESC, created_at ASC
		LIMIT $2`

	err := r.db.SelectContext(ctx, &messages, query, queueName, limit)
	if err != nil {
		return nil, err
	}

	return messages, nil
}

// GetPopularInteractions gets popular interactions for a user
func (r *aiConversationRepository) GetPopularInteractions(ctx context.Context, userID uuid.UUID, limit int) ([]entity.AIInteraction, error) {
	var interactions []entity.AIInteraction
	query := `
		SELECT i.*, COUNT(f.id) as feedback_count
		FROM ai_interactions i
		LEFT JOIN ai_interaction_feedback f ON i.id = f.interaction_id
		WHERE i.user_id = $1 
		AND i.status = 'completed'
		GROUP BY i.id
		ORDER BY feedback_count DESC, i.created_at DESC
		LIMIT $2`

	err := r.db.SelectContext(ctx, &interactions, query, userID, limit)
	if err != nil {
		return nil, err
	}

	return interactions, nil
}

// ArchiveOldInteractions archives old interactions
func (r *aiConversationRepository) ArchiveOldInteractions(ctx context.Context, daysOld int) (int, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	// Insert into archive
	insertQuery := `
		INSERT INTO ai_interactions_archive 
		SELECT * FROM ai_interactions 
		WHERE created_at < NOW() - INTERVAL '%d days'
		AND status IN ('completed', 'failed', 'cancelled')`

	insertQuery = fmt.Sprintf(insertQuery, daysOld)
	result, err := tx.ExecContext(ctx, insertQuery)
	if err != nil {
		return 0, err
	}

	rowsArchived, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}

	// Delete from main table
	deleteQuery := `
		DELETE FROM ai_interactions 
		WHERE created_at < NOW() - INTERVAL '%d days'
		AND status IN ('completed', 'failed', 'cancelled')`

	deleteQuery = fmt.Sprintf(deleteQuery, daysOld)
	_, err = tx.ExecContext(ctx, deleteQuery)
	if err != nil {
		return 0, err
	}

	err = tx.Commit()
	if err != nil {
		return 0, err
	}

	return int(rowsArchived), nil
}

// GetArchivedInteractions gets archived interactions
func (r *aiConversationRepository) GetArchivedInteractions(ctx context.Context, conversationID uuid.UUID) ([]entity.AIInteraction, error) {
	var interactions []entity.AIInteraction
	query := `
		SELECT * FROM ai_interactions_archive 
		WHERE conversation_id = $1 
		ORDER BY sequence_number ASC`

	err := r.db.SelectContext(ctx, &interactions, query, conversationID)
	if err != nil {
		return nil, err
	}

	return interactions, nil
}
