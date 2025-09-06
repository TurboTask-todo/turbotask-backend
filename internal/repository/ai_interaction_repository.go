package repository

import (
	"context"
	"time"

	"quantumtask-auth-api/internal/database"
)

// AIInteraction represents an AI API interaction record
type AIInteraction struct {
	ID              string                 `json:"id" db:"id"`
	UserID          string                 `json:"user_id" db:"user_id"`
	TodoID          *string                `json:"todo_id,omitempty" db:"todo_id"`
	InteractionType string                 `json:"interaction_type" db:"interaction_type"`
	RequestData     map[string]interface{} `json:"request_data" db:"request_data"`
	ResponseData    map[string]interface{} `json:"response_data,omitempty" db:"response_data"`
	TokensUsed      int                    `json:"tokens_used" db:"tokens_used"`
	DurationMs      int                    `json:"duration_ms" db:"duration_ms"`
	Success         bool                   `json:"success" db:"success"`
	ErrorMessage    string                 `json:"error_message,omitempty" db:"error_message"`
	CreatedAt       time.Time              `json:"created_at" db:"created_at"`
}

// AIInteractionRepositoryInterface defines the interface for AI interaction repository
type AIInteractionRepositoryInterface interface {
	Create(ctx context.Context, interaction *AIInteraction) (*AIInteraction, error)
	GetByID(ctx context.Context, id string) (*AIInteraction, error)
	GetByUserID(ctx context.Context, userID string) ([]*AIInteraction, error)
	GetByTodoID(ctx context.Context, todoID string) ([]*AIInteraction, error)
	GetByType(ctx context.Context, interactionType string) ([]*AIInteraction, error)
	GetUserMetrics(ctx context.Context, userID string, since time.Time) (*AIInteractionMetrics, error)
	Delete(ctx context.Context, id string) error
}

// AIInteractionMetrics represents AI usage metrics
type AIInteractionMetrics struct {
	TotalInteractions      int            `json:"total_interactions"`
	SuccessfulInteractions int            `json:"successful_interactions"`
	FailedInteractions     int            `json:"failed_interactions"`
	TotalTokensUsed        int            `json:"total_tokens_used"`
	AverageResponseTime    float64        `json:"average_response_time_ms"`
	InteractionsByType     map[string]int `json:"interactions_by_type"`
}

// AIInteractionRepository implements AIInteractionRepositoryInterface
type AIInteractionRepository struct {
	db *database.DB
}

// NewAIInteractionRepository creates a new AI interaction repository
func NewAIInteractionRepository(db *database.DB) AIInteractionRepositoryInterface {
	return &AIInteractionRepository{
		db: db,
	}
}

// Create creates a new AI interaction record
func (r *AIInteractionRepository) Create(ctx context.Context, interaction *AIInteraction) (*AIInteraction, error) {
	query := `
		INSERT INTO ai_interactions (
			user_id, todo_id, interaction_type, request_data, response_data,
			tokens_used, duration_ms, success, error_message, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, created_at
	`

	err := r.db.QueryRowContext(ctx, query,
		interaction.UserID,
		interaction.TodoID,
		interaction.InteractionType,
		interaction.RequestData,
		interaction.ResponseData,
		interaction.TokensUsed,
		interaction.DurationMs,
		interaction.Success,
		interaction.ErrorMessage,
		interaction.CreatedAt,
	).Scan(&interaction.ID, &interaction.CreatedAt)

	if err != nil {
		return nil, err
	}

	return interaction, nil
}

// GetByID retrieves an AI interaction by ID
func (r *AIInteractionRepository) GetByID(ctx context.Context, id string) (*AIInteraction, error) {
	query := `
		SELECT id, user_id, todo_id, interaction_type, request_data, response_data,
		       tokens_used, duration_ms, success, error_message, created_at
		FROM ai_interactions
		WHERE id = $1
	`

	interaction := &AIInteraction{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&interaction.ID,
		&interaction.UserID,
		&interaction.TodoID,
		&interaction.InteractionType,
		&interaction.RequestData,
		&interaction.ResponseData,
		&interaction.TokensUsed,
		&interaction.DurationMs,
		&interaction.Success,
		&interaction.ErrorMessage,
		&interaction.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	return interaction, nil
}

// GetByUserID retrieves all AI interactions for a user
func (r *AIInteractionRepository) GetByUserID(ctx context.Context, userID string) ([]*AIInteraction, error) {
	query := `
		SELECT id, user_id, todo_id, interaction_type, request_data, response_data,
		       tokens_used, duration_ms, success, error_message, created_at
		FROM ai_interactions
		WHERE user_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var interactions []*AIInteraction
	for rows.Next() {
		interaction := &AIInteraction{}
		err := rows.Scan(
			&interaction.ID,
			&interaction.UserID,
			&interaction.TodoID,
			&interaction.InteractionType,
			&interaction.RequestData,
			&interaction.ResponseData,
			&interaction.TokensUsed,
			&interaction.DurationMs,
			&interaction.Success,
			&interaction.ErrorMessage,
			&interaction.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		interactions = append(interactions, interaction)
	}

	return interactions, nil
}

// GetByTodoID retrieves all AI interactions for a specific todo
func (r *AIInteractionRepository) GetByTodoID(ctx context.Context, todoID string) ([]*AIInteraction, error) {
	query := `
		SELECT id, user_id, todo_id, interaction_type, request_data, response_data,
		       tokens_used, duration_ms, success, error_message, created_at
		FROM ai_interactions
		WHERE todo_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, todoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var interactions []*AIInteraction
	for rows.Next() {
		interaction := &AIInteraction{}
		err := rows.Scan(
			&interaction.ID,
			&interaction.UserID,
			&interaction.TodoID,
			&interaction.InteractionType,
			&interaction.RequestData,
			&interaction.ResponseData,
			&interaction.TokensUsed,
			&interaction.DurationMs,
			&interaction.Success,
			&interaction.ErrorMessage,
			&interaction.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		interactions = append(interactions, interaction)
	}

	return interactions, nil
}

// GetByType retrieves all AI interactions of a specific type
func (r *AIInteractionRepository) GetByType(ctx context.Context, interactionType string) ([]*AIInteraction, error) {
	query := `
		SELECT id, user_id, todo_id, interaction_type, request_data, response_data,
		       tokens_used, duration_ms, success, error_message, created_at
		FROM ai_interactions
		WHERE interaction_type = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, interactionType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var interactions []*AIInteraction
	for rows.Next() {
		interaction := &AIInteraction{}
		err := rows.Scan(
			&interaction.ID,
			&interaction.UserID,
			&interaction.TodoID,
			&interaction.InteractionType,
			&interaction.RequestData,
			&interaction.ResponseData,
			&interaction.TokensUsed,
			&interaction.DurationMs,
			&interaction.Success,
			&interaction.ErrorMessage,
			&interaction.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		interactions = append(interactions, interaction)
	}

	return interactions, nil
}

// GetUserMetrics calculates AI usage metrics for a user
func (r *AIInteractionRepository) GetUserMetrics(ctx context.Context, userID string, since time.Time) (*AIInteractionMetrics, error) {
	query := `
		SELECT 
			COUNT(*) as total_interactions,
			COUNT(CASE WHEN success = true THEN 1 END) as successful_interactions,
			COUNT(CASE WHEN success = false THEN 1 END) as failed_interactions,
			COALESCE(SUM(tokens_used), 0) as total_tokens_used,
			COALESCE(AVG(duration_ms), 0) as average_response_time,
			interaction_type,
			COUNT(*) as type_count
		FROM ai_interactions
		WHERE user_id = $1 AND created_at >= $2
		GROUP BY interaction_type
	`

	rows, err := r.db.QueryContext(ctx, query, userID, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	metrics := &AIInteractionMetrics{
		InteractionsByType: make(map[string]int),
	}

	for rows.Next() {
		var interactionType string
		var typeCount int
		err := rows.Scan(
			&metrics.TotalInteractions,
			&metrics.SuccessfulInteractions,
			&metrics.FailedInteractions,
			&metrics.TotalTokensUsed,
			&metrics.AverageResponseTime,
			&interactionType,
			&typeCount,
		)
		if err != nil {
			return nil, err
		}
		metrics.InteractionsByType[interactionType] = typeCount
	}

	return metrics, nil
}

// Delete removes an AI interaction record
func (r *AIInteractionRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM ai_interactions WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}
