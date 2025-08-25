package entity

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// JSONMap represents a JSON object stored in database
type JSONMap map[string]interface{}

// Value implements the driver.Valuer interface for JSONMap
func (j JSONMap) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

// Scan implements the sql.Scanner interface for JSONMap
func (j *JSONMap) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}

	return json.Unmarshal(bytes, j)
}

// AIConversation represents a conversation session with AI
type AIConversation struct {
	ID                uuid.UUID      `json:"id" db:"id"`
	UserID            uuid.UUID      `json:"user_id" db:"user_id"`
	SessionID         string         `json:"session_id" db:"session_id"`
	Title             *string        `json:"title" db:"title"`
	ModelName         string         `json:"model_name" db:"model_name"`
	ModelVersion      *string        `json:"model_version" db:"model_version"`
	ConversationType  string         `json:"conversation_type" db:"conversation_type"`
	TotalInputTokens  int64          `json:"total_input_tokens" db:"total_input_tokens"`
	TotalOutputTokens int64          `json:"total_output_tokens" db:"total_output_tokens"`
	TotalCost         float64        `json:"total_cost" db:"total_cost"`
	InteractionCount  int            `json:"interaction_count" db:"interaction_count"`
	Status            string         `json:"status" db:"status"`
	Priority          int            `json:"priority" db:"priority"`
	CacheKey          *string        `json:"cache_key" db:"cache_key"`
	CacheTTL          int            `json:"cache_ttl" db:"cache_ttl"`
	LastAccessedAt    time.Time      `json:"last_accessed_at" db:"last_accessed_at"`
	Metadata          JSONMap        `json:"metadata" db:"metadata"`
	Tags              pq.StringArray `json:"tags" db:"tags"`
	CreatedAt         time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at" db:"updated_at"`
	ArchivedAt        *time.Time     `json:"archived_at" db:"archived_at"`
}

// AIInteraction represents a single AI interaction within a conversation
type AIInteraction struct {
	ID                    uuid.UUID      `json:"id" db:"id"`
	ConversationID        uuid.UUID      `json:"conversation_id" db:"conversation_id"`
	UserID                uuid.UUID      `json:"user_id" db:"user_id"`
	SequenceNumber        int            `json:"sequence_number" db:"sequence_number"`
	ParentInteractionID   *uuid.UUID     `json:"parent_interaction_id" db:"parent_interaction_id"`
	UserInput             *string        `json:"user_input" db:"user_input"`
	UserInputCompressed   []byte         `json:"-" db:"user_input_compressed"`
	InputCompressionType  string         `json:"input_compression_type" db:"input_compression_type"`
	InputTokenCount       int            `json:"input_token_count" db:"input_token_count"`
	InputCharacterCount   int            `json:"input_character_count" db:"input_character_count"`
	InputHash             *string        `json:"input_hash" db:"input_hash"`
	AIResponse            *string        `json:"ai_response" db:"ai_response"`
	AIResponseCompressed  []byte         `json:"-" db:"ai_response_compressed"`
	OutputCompressionType string         `json:"output_compression_type" db:"output_compression_type"`
	OutputTokenCount      int            `json:"output_token_count" db:"output_token_count"`
	OutputCharacterCount  int            `json:"output_character_count" db:"output_character_count"`
	OutputHash            *string        `json:"output_hash" db:"output_hash"`
	ModelName             string         `json:"model_name" db:"model_name"`
	ModelVersion          *string        `json:"model_version" db:"model_version"`
	Temperature           *float64       `json:"temperature" db:"temperature"`
	MaxTokens             *int           `json:"max_tokens" db:"max_tokens"`
	TopP                  *float64       `json:"top_p" db:"top_p"`
	FrequencyPenalty      *float64       `json:"frequency_penalty" db:"frequency_penalty"`
	PresencePenalty       *float64       `json:"presence_penalty" db:"presence_penalty"`
	StopSequences         pq.StringArray `json:"stop_sequences" db:"stop_sequences"`
	SystemPrompt          *string        `json:"system_prompt" db:"system_prompt"`
	ProcessingTimeMs      int            `json:"processing_time_ms" db:"processing_time_ms"`
	QueueTimeMs           int            `json:"queue_time_ms" db:"queue_time_ms"`
	TotalLatencyMs        int            `json:"total_latency_ms" db:"total_latency_ms"`
	RetryCount            int            `json:"retry_count" db:"retry_count"`
	InputCost             float64        `json:"input_cost" db:"input_cost"`
	OutputCost            float64        `json:"output_cost" db:"output_cost"`
	TotalCost             float64        `json:"total_cost" db:"total_cost"`
	QueueMessageID        *string        `json:"queue_message_id" db:"queue_message_id"`
	QueuePriority         int            `json:"queue_priority" db:"queue_priority"`
	CacheKey              *string        `json:"cache_key" db:"cache_key"`
	CacheHit              bool           `json:"cache_hit" db:"cache_hit"`
	Status                string         `json:"status" db:"status"`
	ErrorCode             *string        `json:"error_code" db:"error_code"`
	ErrorMessage          *string        `json:"error_message" db:"error_message"`
	ClientIP              *string        `json:"client_ip" db:"client_ip"`
	UserAgent             *string        `json:"user_agent" db:"user_agent"`
	RequestID             *string        `json:"request_id" db:"request_id"`
	TraceID               *string        `json:"trace_id" db:"trace_id"`
	RequestMetadata       JSONMap        `json:"request_metadata" db:"request_metadata"`
	ResponseMetadata      JSONMap        `json:"response_metadata" db:"response_metadata"`
	ProcessingMetadata    JSONMap        `json:"processing_metadata" db:"processing_metadata"`
	CreatedAt             time.Time      `json:"created_at" db:"created_at"`
	QueuedAt              *time.Time     `json:"queued_at" db:"queued_at"`
	StartedProcessingAt   *time.Time     `json:"started_processing_at" db:"started_processing_at"`
	CompletedAt           *time.Time     `json:"completed_at" db:"completed_at"`
	CachedAt              *time.Time     `json:"cached_at" db:"cached_at"`
}

// AIInteractionFeedback represents user feedback on AI interactions
type AIInteractionFeedback struct {
	ID                       uuid.UUID      `json:"id" db:"id"`
	InteractionID            uuid.UUID      `json:"interaction_id" db:"interaction_id"`
	UserID                   uuid.UUID      `json:"user_id" db:"user_id"`
	OverallRating            *int           `json:"overall_rating" db:"overall_rating"`
	AccuracyRating           *int           `json:"accuracy_rating" db:"accuracy_rating"`
	HelpfulnessRating        *int           `json:"helpfulness_rating" db:"helpfulness_rating"`
	RelevanceRating          *int           `json:"relevance_rating" db:"relevance_rating"`
	FeedbackType             *string        `json:"feedback_type" db:"feedback_type"`
	FeedbackText             *string        `json:"feedback_text" db:"feedback_text"`
	ImprovementSuggestions   *string        `json:"improvement_suggestions" db:"improvement_suggestions"`
	Tags                     pq.StringArray `json:"tags" db:"tags"`
	SessionQuality           *int           `json:"session_quality" db:"session_quality"`
	ResponseTimeSatisfaction *int           `json:"response_time_satisfaction" db:"response_time_satisfaction"`
	CreatedAt                time.Time      `json:"created_at" db:"created_at"`
}

// AIUsageAnalytics represents usage analytics aggregated by time periods
type AIUsageAnalytics struct {
	ID                  uuid.UUID `json:"id" db:"id"`
	UserID              uuid.UUID `json:"user_id" db:"user_id"`
	Date                time.Time `json:"date" db:"date"`
	Hour                int       `json:"hour" db:"hour"`
	ModelName           string    `json:"model_name" db:"model_name"`
	InteractionCount    int       `json:"interaction_count" db:"interaction_count"`
	TotalInputTokens    int64     `json:"total_input_tokens" db:"total_input_tokens"`
	TotalOutputTokens   int64     `json:"total_output_tokens" db:"total_output_tokens"`
	TotalCost           float64   `json:"total_cost" db:"total_cost"`
	AvgProcessingTimeMs int       `json:"avg_processing_time_ms" db:"avg_processing_time_ms"`
	AvgQueueTimeMs      int       `json:"avg_queue_time_ms" db:"avg_queue_time_ms"`
	SuccessRate         float64   `json:"success_rate" db:"success_rate"`
	CacheHitRate        float64   `json:"cache_hit_rate" db:"cache_hit_rate"`
	ConversationTypes   JSONMap   `json:"conversation_types" db:"conversation_types"`
	PeakUsageMinute     *int      `json:"peak_usage_minute" db:"peak_usage_minute"`
	CreatedAt           time.Time `json:"created_at" db:"created_at"`
	UpdatedAt           time.Time `json:"updated_at" db:"updated_at"`
}

// CacheEntry represents a cache entry for Redis integration
type CacheEntry struct {
	ID             uuid.UUID      `json:"id" db:"id"`
	CacheKey       string         `json:"cache_key" db:"cache_key"`
	ContentHash    string         `json:"content_hash" db:"content_hash"`
	ContentType    string         `json:"content_type" db:"content_type"`
	SizeBytes      int64          `json:"size_bytes" db:"size_bytes"`
	HitCount       int64          `json:"hit_count" db:"hit_count"`
	MissCount      int64          `json:"miss_count" db:"miss_count"`
	TTLSeconds     int            `json:"ttl_seconds" db:"ttl_seconds"`
	Tags           pq.StringArray `json:"tags" db:"tags"`
	CreatedAt      time.Time      `json:"created_at" db:"created_at"`
	LastAccessedAt time.Time      `json:"last_accessed_at" db:"last_accessed_at"`
	ExpiresAt      *time.Time     `json:"expires_at" db:"expires_at"`
}

// QueueMessage represents a RabbitMQ message
type QueueMessage struct {
	ID              uuid.UUID  `json:"id" db:"id"`
	MessageID       string     `json:"message_id" db:"message_id"`
	QueueName       string     `json:"queue_name" db:"queue_name"`
	RoutingKey      *string    `json:"routing_key" db:"routing_key"`
	ExchangeName    *string    `json:"exchange_name" db:"exchange_name"`
	Payload         JSONMap    `json:"payload" db:"payload"`
	ContentType     string     `json:"content_type" db:"content_type"`
	ContentEncoding string     `json:"content_encoding" db:"content_encoding"`
	Priority        int        `json:"priority" db:"priority"`
	DelaySeconds    int        `json:"delay_seconds" db:"delay_seconds"`
	MaxRetries      int        `json:"max_retries" db:"max_retries"`
	RetryCount      int        `json:"retry_count" db:"retry_count"`
	Status          string     `json:"status" db:"status"`
	WorkerID        *string    `json:"worker_id" db:"worker_id"`
	ErrorMessage    *string    `json:"error_message" db:"error_message"`
	InteractionID   *uuid.UUID `json:"interaction_id" db:"interaction_id"`
	UserID          *uuid.UUID `json:"user_id" db:"user_id"`
	CreatedAt       time.Time  `json:"created_at" db:"created_at"`
	ScheduledAt     time.Time  `json:"scheduled_at" db:"scheduled_at"`
	StartedAt       *time.Time `json:"started_at" db:"started_at"`
	CompletedAt     *time.Time `json:"completed_at" db:"completed_at"`
	ExpiresAt       time.Time  `json:"expires_at" db:"expires_at"`
}

// CreateConversationRequest represents the request to create a new conversation
type CreateConversationRequest struct {
	SessionID        string                 `json:"session_id" binding:"required,min=8"`
	Title            string                 `json:"title" binding:"omitempty,max=500"`
	ModelName        string                 `json:"model_name" binding:"required,min=3"`
	ModelVersion     string                 `json:"model_version" binding:"omitempty"`
	ConversationType string                 `json:"conversation_type" binding:"omitempty,oneof=chat completion embedding"`
	Priority         int                    `json:"priority" binding:"omitempty,min=1,max=10"`
	Metadata         map[string]interface{} `json:"metadata" binding:"omitempty"`
	Tags             []string               `json:"tags" binding:"omitempty"`
}

// CreateInteractionRequest represents the request to create a new interaction
type CreateInteractionRequest struct {
	ConversationID    uuid.UUID              `json:"conversation_id" binding:"required"`
	UserInput         string                 `json:"user_input" binding:"required,min=1"`
	ModelName         string                 `json:"model_name" binding:"required,min=3"`
	ModelVersion      string                 `json:"model_version" binding:"omitempty"`
	Temperature       *float64               `json:"temperature" binding:"omitempty,min=0,max=2"`
	MaxTokens         *int                   `json:"max_tokens" binding:"omitempty,min=1"`
	TopP              *float64               `json:"top_p" binding:"omitempty,min=0,max=1"`
	FrequencyPenalty  *float64               `json:"frequency_penalty" binding:"omitempty,min=-2,max=2"`
	PresencePenalty   *float64               `json:"presence_penalty" binding:"omitempty,min=-2,max=2"`
	StopSequences     []string               `json:"stop_sequences" binding:"omitempty"`
	SystemPrompt      string                 `json:"system_prompt" binding:"omitempty"`
	Priority          int                    `json:"priority" binding:"omitempty,min=1,max=10"`
	EnableCompression bool                   `json:"enable_compression"`
	EnableCaching     bool                   `json:"enable_caching"`
	RequestMetadata   map[string]interface{} `json:"request_metadata" binding:"omitempty"`
}

// UpdateConversationRequest represents the request to update a conversation
type UpdateConversationRequest struct {
	Title    *string                `json:"title" binding:"omitempty,max=500"`
	Status   *string                `json:"status" binding:"omitempty,oneof=active archived deleted suspended"`
	Priority *int                   `json:"priority" binding:"omitempty,min=1,max=10"`
	Metadata map[string]interface{} `json:"metadata" binding:"omitempty"`
	Tags     []string               `json:"tags" binding:"omitempty"`
}

// CreateFeedbackRequest represents the request to create feedback
type CreateFeedbackRequest struct {
	InteractionID            uuid.UUID `json:"interaction_id" binding:"required"`
	OverallRating            *int      `json:"overall_rating" binding:"omitempty,min=1,max=5"`
	AccuracyRating           *int      `json:"accuracy_rating" binding:"omitempty,min=1,max=5"`
	HelpfulnessRating        *int      `json:"helpfulness_rating" binding:"omitempty,min=1,max=5"`
	RelevanceRating          *int      `json:"relevance_rating" binding:"omitempty,min=1,max=5"`
	FeedbackType             *string   `json:"feedback_type" binding:"omitempty"`
	FeedbackText             *string   `json:"feedback_text" binding:"omitempty,max=2000"`
	ImprovementSuggestions   *string   `json:"improvement_suggestions" binding:"omitempty,max=2000"`
	Tags                     []string  `json:"tags" binding:"omitempty"`
	SessionQuality           *int      `json:"session_quality" binding:"omitempty,min=1,max=5"`
	ResponseTimeSatisfaction *int      `json:"response_time_satisfaction" binding:"omitempty,min=1,max=5"`
}

// GetConversationsQuery represents query parameters for getting conversations
type GetConversationsQuery struct {
	UserID   uuid.UUID `form:"user_id"`
	Status   string    `form:"status" binding:"omitempty,oneof=active archived deleted suspended"`
	Model    string    `form:"model"`
	FromDate string    `form:"from_date"`
	ToDate   string    `form:"to_date"`
	Search   string    `form:"search"`
	Tags     []string  `form:"tags"`
	Limit    int       `form:"limit" binding:"omitempty,min=1,max=100"`
	Offset   int       `form:"offset" binding:"omitempty,min=0"`
	OrderBy  string    `form:"order_by" binding:"omitempty,oneof=created_at updated_at total_cost interaction_count"`
	Order    string    `form:"order" binding:"omitempty,oneof=asc desc"`
}

// GetAnalyticsQuery represents query parameters for analytics
type GetAnalyticsQuery struct {
	UserID   uuid.UUID `form:"user_id"`
	FromDate string    `form:"from_date" binding:"required"`
	ToDate   string    `form:"to_date" binding:"required"`
	Model    string    `form:"model"`
	GroupBy  string    `form:"group_by" binding:"omitempty,oneof=day hour model"`
	Metrics  []string  `form:"metrics"`
}

// ConversationWithInteractions represents a conversation with its interactions
type ConversationWithInteractions struct {
	Conversation AIConversation  `json:"conversation"`
	Interactions []AIInteraction `json:"interactions"`
}

// InteractionResponse represents the response after creating an interaction
type InteractionResponse struct {
	Interaction AIInteraction `json:"interaction"`
	QueueInfo   *QueueInfo    `json:"queue_info,omitempty"`
	CacheInfo   *CacheInfo    `json:"cache_info,omitempty"`
}

// QueueInfo represents queue-related information
type QueueInfo struct {
	MessageID     string `json:"message_id"`
	QueueName     string `json:"queue_name"`
	Priority      int    `json:"priority"`
	EstimatedWait string `json:"estimated_wait"`
}

// CacheInfo represents cache-related information
type CacheInfo struct {
	CacheKey string `json:"cache_key"`
	CacheHit bool   `json:"cache_hit"`
	TTL      int    `json:"ttl"`
}

// AnalyticsResponse represents analytics data
type AnalyticsResponse struct {
	UserID     uuid.UUID            `json:"user_id"`
	DateRange  DateRange            `json:"date_range"`
	Summary    AnalyticsSummary     `json:"summary"`
	TimeSeries []AnalyticsDataPoint `json:"time_series"`
	Models     []ModelAnalytics     `json:"models"`
	Patterns   UsagePatterns        `json:"patterns"`
}

// DateRange represents a date range
type DateRange struct {
	From time.Time `json:"from"`
	To   time.Time `json:"to"`
}

// AnalyticsSummary represents summary analytics
type AnalyticsSummary struct {
	TotalInteractions   int     `json:"total_interactions"`
	TotalCost           float64 `json:"total_cost"`
	TotalInputTokens    int64   `json:"total_input_tokens"`
	TotalOutputTokens   int64   `json:"total_output_tokens"`
	AvgProcessingTime   int     `json:"avg_processing_time_ms"`
	SuccessRate         float64 `json:"success_rate"`
	CacheHitRate        float64 `json:"cache_hit_rate"`
	UniqueConversations int     `json:"unique_conversations"`
}

// AnalyticsDataPoint represents a single analytics data point
type AnalyticsDataPoint struct {
	Timestamp      time.Time `json:"timestamp"`
	Interactions   int       `json:"interactions"`
	Cost           float64   `json:"cost"`
	InputTokens    int64     `json:"input_tokens"`
	OutputTokens   int64     `json:"output_tokens"`
	ProcessingTime int       `json:"avg_processing_time_ms"`
	SuccessRate    float64   `json:"success_rate"`
	CacheHitRate   float64   `json:"cache_hit_rate"`
}

// ModelAnalytics represents analytics per model
type ModelAnalytics struct {
	ModelName       string  `json:"model_name"`
	Interactions    int     `json:"interactions"`
	Cost            float64 `json:"cost"`
	AvgCostPerToken float64 `json:"avg_cost_per_token"`
	SuccessRate     float64 `json:"success_rate"`
	AvgResponseTime int     `json:"avg_response_time_ms"`
}

// UsagePatterns represents usage patterns
type UsagePatterns struct {
	PeakHours         []int                  `json:"peak_hours"`
	ConversationTypes map[string]int         `json:"conversation_types"`
	MostUsedModels    []string               `json:"most_used_models"`
	AvgSessionLength  float64                `json:"avg_session_length"`
	PreferredSettings map[string]interface{} `json:"preferred_settings"`
}
