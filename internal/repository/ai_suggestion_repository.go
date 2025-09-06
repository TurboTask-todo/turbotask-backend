package repository

import (
	"context"
	"time"

	"quantumtask-auth-api/internal/database"
)

// AISuggestion represents an AI suggestion record
type AISuggestion struct {
	ID                string    `json:"id" db:"id"`
	UserID            string    `json:"user_id" db:"user_id"`
	TodoID            *string   `json:"todo_id,omitempty" db:"todo_id"`
	SubtaskID         *string   `json:"subtask_id,omitempty" db:"subtask_id"`
	SuggestionType    string    `json:"suggestion_type" db:"suggestion_type"`
	OriginalContent   string    `json:"original_content" db:"original_content"`
	SuggestedContent  string    `json:"suggested_content" db:"suggested_content"`
	AIConfidence      float64   `json:"ai_confidence" db:"ai_confidence"`
	UserAction        string    `json:"user_action" db:"user_action"`
	UserFeedback      string    `json:"user_feedback" db:"user_feedback"`
	RegenerationCount int       `json:"regeneration_count" db:"regeneration_count"`
	CreatedAt         time.Time `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time `json:"updated_at" db:"updated_at"`
}

// AISuggestionRepositoryInterface defines the interface for AI suggestion repository
type AISuggestionRepositoryInterface interface {
	Create(ctx context.Context, suggestion *AISuggestion) (*AISuggestion, error)
	GetByID(ctx context.Context, id string) (*AISuggestion, error)
	GetByUserID(ctx context.Context, userID string) ([]*AISuggestion, error)
	GetByTodoID(ctx context.Context, todoID string) ([]*AISuggestion, error)
	GetBySubtaskID(ctx context.Context, subtaskID string) ([]*AISuggestion, error)
	GetByType(ctx context.Context, suggestionType string) ([]*AISuggestion, error)
	GetPendingSuggestions(ctx context.Context, userID string) ([]*AISuggestion, error)
	Update(ctx context.Context, suggestion *AISuggestion) (*AISuggestion, error)
	Delete(ctx context.Context, id string) error
	GetUserSuggestionStats(ctx context.Context, userID string) (*AISuggestionStats, error)
}

// AISuggestionStats represents AI suggestion statistics
type AISuggestionStats struct {
	TotalSuggestions    int            `json:"total_suggestions"`
	AcceptedSuggestions int            `json:"accepted_suggestions"`
	RejectedSuggestions int            `json:"rejected_suggestions"`
	PendingSuggestions  int            `json:"pending_suggestions"`
	AcceptanceRate      float64        `json:"acceptance_rate"`
	SuggestionsByType   map[string]int `json:"suggestions_by_type"`
	AvgConfidence       float64        `json:"average_confidence"`
}

// AISuggestionRepository implements AISuggestionRepositoryInterface
type AISuggestionRepository struct {
	db *database.DB
}

// NewAISuggestionRepository creates a new AI suggestion repository
func NewAISuggestionRepository(db *database.DB) AISuggestionRepositoryInterface {
	return &AISuggestionRepository{
		db: db,
	}
}

// Create creates a new AI suggestion record
func (r *AISuggestionRepository) Create(ctx context.Context, suggestion *AISuggestion) (*AISuggestion, error) {
	query := `
		INSERT INTO ai_suggestions (
			user_id, todo_id, subtask_id, suggestion_type, original_content,
			suggested_content, ai_confidence, user_action, user_feedback,
			regeneration_count, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id, created_at, updated_at
	`

	now := time.Now()
	if suggestion.CreatedAt.IsZero() {
		suggestion.CreatedAt = now
	}
	if suggestion.UpdatedAt.IsZero() {
		suggestion.UpdatedAt = now
	}

	err := r.db.QueryRowContext(
		ctx, query,
		suggestion.UserID,
		suggestion.TodoID,
		suggestion.SubtaskID,
		suggestion.SuggestionType,
		suggestion.OriginalContent,
		suggestion.SuggestedContent,
		suggestion.AIConfidence,
		suggestion.UserAction,
		suggestion.UserFeedback,
		suggestion.RegenerationCount,
		suggestion.CreatedAt,
		suggestion.UpdatedAt,
	).Scan(&suggestion.ID, &suggestion.CreatedAt, &suggestion.UpdatedAt)

	if err != nil {
		return nil, err
	}

	return suggestion, nil
}

// GetByID retrieves an AI suggestion by ID
func (r *AISuggestionRepository) GetByID(ctx context.Context, id string) (*AISuggestion, error) {
	query := `
		SELECT id, user_id, todo_id, subtask_id, suggestion_type, original_content,
		       suggested_content, ai_confidence, user_action, user_feedback,
		       regeneration_count, created_at, updated_at
		FROM ai_suggestions
		WHERE id = $1
	`

	suggestion := &AISuggestion{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&suggestion.ID,
		&suggestion.UserID,
		&suggestion.TodoID,
		&suggestion.SubtaskID,
		&suggestion.SuggestionType,
		&suggestion.OriginalContent,
		&suggestion.SuggestedContent,
		&suggestion.AIConfidence,
		&suggestion.UserAction,
		&suggestion.UserFeedback,
		&suggestion.RegenerationCount,
		&suggestion.CreatedAt,
		&suggestion.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return suggestion, nil
}

// GetByUserID retrieves all AI suggestions for a user
func (r *AISuggestionRepository) GetByUserID(ctx context.Context, userID string) ([]*AISuggestion, error) {
	query := `
		SELECT id, user_id, todo_id, subtask_id, suggestion_type, original_content,
		       suggested_content, ai_confidence, user_action, user_feedback,
		       regeneration_count, created_at, updated_at
		FROM ai_suggestions
		WHERE user_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var suggestions []*AISuggestion
	for rows.Next() {
		suggestion := &AISuggestion{}
		err := rows.Scan(
			&suggestion.ID,
			&suggestion.UserID,
			&suggestion.TodoID,
			&suggestion.SubtaskID,
			&suggestion.SuggestionType,
			&suggestion.OriginalContent,
			&suggestion.SuggestedContent,
			&suggestion.AIConfidence,
			&suggestion.UserAction,
			&suggestion.UserFeedback,
			&suggestion.RegenerationCount,
			&suggestion.CreatedAt,
			&suggestion.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		suggestions = append(suggestions, suggestion)
	}

	return suggestions, nil
}

// GetByTodoID retrieves all AI suggestions for a specific todo
func (r *AISuggestionRepository) GetByTodoID(ctx context.Context, todoID string) ([]*AISuggestion, error) {
	query := `
		SELECT id, user_id, todo_id, subtask_id, suggestion_type, original_content,
		       suggested_content, ai_confidence, user_action, user_feedback,
		       regeneration_count, created_at, updated_at
		FROM ai_suggestions
		WHERE todo_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, todoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var suggestions []*AISuggestion
	for rows.Next() {
		suggestion := &AISuggestion{}
		err := rows.Scan(
			&suggestion.ID,
			&suggestion.UserID,
			&suggestion.TodoID,
			&suggestion.SubtaskID,
			&suggestion.SuggestionType,
			&suggestion.OriginalContent,
			&suggestion.SuggestedContent,
			&suggestion.AIConfidence,
			&suggestion.UserAction,
			&suggestion.UserFeedback,
			&suggestion.RegenerationCount,
			&suggestion.CreatedAt,
			&suggestion.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		suggestions = append(suggestions, suggestion)
	}

	return suggestions, nil
}

// GetBySubtaskID retrieves all AI suggestions for a specific subtask
func (r *AISuggestionRepository) GetBySubtaskID(ctx context.Context, subtaskID string) ([]*AISuggestion, error) {
	query := `
		SELECT id, user_id, todo_id, subtask_id, suggestion_type, original_content,
		       suggested_content, ai_confidence, user_action, user_feedback,
		       regeneration_count, created_at, updated_at
		FROM ai_suggestions
		WHERE subtask_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, subtaskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var suggestions []*AISuggestion
	for rows.Next() {
		suggestion := &AISuggestion{}
		err := rows.Scan(
			&suggestion.ID,
			&suggestion.UserID,
			&suggestion.TodoID,
			&suggestion.SubtaskID,
			&suggestion.SuggestionType,
			&suggestion.OriginalContent,
			&suggestion.SuggestedContent,
			&suggestion.AIConfidence,
			&suggestion.UserAction,
			&suggestion.UserFeedback,
			&suggestion.RegenerationCount,
			&suggestion.CreatedAt,
			&suggestion.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		suggestions = append(suggestions, suggestion)
	}

	return suggestions, nil
}

// GetByType retrieves all AI suggestions of a specific type
func (r *AISuggestionRepository) GetByType(ctx context.Context, suggestionType string) ([]*AISuggestion, error) {
	query := `
		SELECT id, user_id, todo_id, subtask_id, suggestion_type, original_content,
		       suggested_content, ai_confidence, user_action, user_feedback,
		       regeneration_count, created_at, updated_at
		FROM ai_suggestions
		WHERE suggestion_type = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, suggestionType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var suggestions []*AISuggestion
	for rows.Next() {
		suggestion := &AISuggestion{}
		err := rows.Scan(
			&suggestion.ID,
			&suggestion.UserID,
			&suggestion.TodoID,
			&suggestion.SubtaskID,
			&suggestion.SuggestionType,
			&suggestion.OriginalContent,
			&suggestion.SuggestedContent,
			&suggestion.AIConfidence,
			&suggestion.UserAction,
			&suggestion.UserFeedback,
			&suggestion.RegenerationCount,
			&suggestion.CreatedAt,
			&suggestion.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		suggestions = append(suggestions, suggestion)
	}

	return suggestions, nil
}

// GetPendingSuggestions retrieves all pending AI suggestions for a user
func (r *AISuggestionRepository) GetPendingSuggestions(ctx context.Context, userID string) ([]*AISuggestion, error) {
	query := `
		SELECT id, user_id, todo_id, subtask_id, suggestion_type, original_content,
		       suggested_content, ai_confidence, user_action, user_feedback,
		       regeneration_count, created_at, updated_at
		FROM ai_suggestions
		WHERE user_id = $1 AND user_action = 'pending'
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var suggestions []*AISuggestion
	for rows.Next() {
		suggestion := &AISuggestion{}
		err := rows.Scan(
			&suggestion.ID,
			&suggestion.UserID,
			&suggestion.TodoID,
			&suggestion.SubtaskID,
			&suggestion.SuggestionType,
			&suggestion.OriginalContent,
			&suggestion.SuggestedContent,
			&suggestion.AIConfidence,
			&suggestion.UserAction,
			&suggestion.UserFeedback,
			&suggestion.RegenerationCount,
			&suggestion.CreatedAt,
			&suggestion.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		suggestions = append(suggestions, suggestion)
	}

	return suggestions, nil
}

// Update updates an AI suggestion record
func (r *AISuggestionRepository) Update(ctx context.Context, suggestion *AISuggestion) (*AISuggestion, error) {
	query := `
		UPDATE ai_suggestions
		SET user_action = $1, user_feedback = $2, regeneration_count = $3, updated_at = $4
		WHERE id = $5
		RETURNING updated_at
	`

	suggestion.UpdatedAt = time.Now()

	err := r.db.QueryRowContext(
		ctx, query,
		suggestion.UserAction,
		suggestion.UserFeedback,
		suggestion.RegenerationCount,
		suggestion.UpdatedAt,
		suggestion.ID,
	).Scan(&suggestion.UpdatedAt)

	if err != nil {
		return nil, err
	}

	return suggestion, nil
}

// Delete removes an AI suggestion record
func (r *AISuggestionRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM ai_suggestions WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

// GetUserSuggestionStats calculates AI suggestion statistics for a user
func (r *AISuggestionRepository) GetUserSuggestionStats(ctx context.Context, userID string) (*AISuggestionStats, error) {
	query := `
		SELECT 
			COUNT(*) as total_suggestions,
			COUNT(CASE WHEN user_action = 'accepted' THEN 1 END) as accepted_suggestions,
			COUNT(CASE WHEN user_action = 'rejected' THEN 1 END) as rejected_suggestions,
			COUNT(CASE WHEN user_action = 'pending' THEN 1 END) as pending_suggestions,
			COALESCE(AVG(ai_confidence), 0) as avg_confidence,
			suggestion_type,
			COUNT(*) as type_count
		FROM ai_suggestions
		WHERE user_id = $1
		GROUP BY suggestion_type
	`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	stats := &AISuggestionStats{
		SuggestionsByType: make(map[string]int),
	}

	for rows.Next() {
		var suggestionType string
		var typeCount int
		err := rows.Scan(
			&stats.TotalSuggestions,
			&stats.AcceptedSuggestions,
			&stats.RejectedSuggestions,
			&stats.PendingSuggestions,
			&stats.AvgConfidence,
			&suggestionType,
			&typeCount,
		)
		if err != nil {
			return nil, err
		}
		stats.SuggestionsByType[suggestionType] = typeCount
	}

	// Calculate acceptance rate
	if stats.TotalSuggestions > 0 {
		stats.AcceptanceRate = float64(stats.AcceptedSuggestions) / float64(stats.TotalSuggestions) * 100
	}

	return stats, nil
}
