package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"macwrite-auth-api/internal/database"
	"macwrite-auth-api/internal/models"

	"github.com/google/uuid"
)

// BreakHistoryRepository handles break history database operations
type BreakHistoryRepository struct {
	db *database.DB
}

// NewBreakHistoryRepository creates a new break history repository
func NewBreakHistoryRepository(db *database.DB) *BreakHistoryRepository {
	return &BreakHistoryRepository{db: db}
}

// EnsureTableExists creates the break_history table if it doesn't exist
func (r *BreakHistoryRepository) EnsureTableExists(ctx context.Context) error {
	createTableSQL := `
		CREATE TABLE IF NOT EXISTS break_history (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			todo_id UUID NOT NULL,
			project_id UUID NOT NULL,
			user_id UUID NOT NULL,
			start_time TIMESTAMP WITH TIME ZONE NOT NULL,
			end_time TIMESTAMP WITH TIME ZONE,
			duration INTEGER NOT NULL DEFAULT 0,
			break_type VARCHAR(50) NOT NULL DEFAULT 'manual',
			notes TEXT,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		);

		CREATE INDEX IF NOT EXISTS idx_break_history_user_id ON break_history(user_id);
		CREATE INDEX IF NOT EXISTS idx_break_history_todo_id ON break_history(todo_id);
		CREATE INDEX IF NOT EXISTS idx_break_history_project_id ON break_history(project_id);
		CREATE INDEX IF NOT EXISTS idx_break_history_start_time ON break_history(start_time);
		CREATE INDEX IF NOT EXISTS idx_break_history_active ON break_history(user_id, end_time) WHERE end_time IS NULL;
	`

	_, err := r.db.ExecContext(ctx, createTableSQL)
	if err != nil {
		return fmt.Errorf("failed to create break_history table: %w", err)
	}

	return nil
}

// Create creates a new break history record
func (r *BreakHistoryRepository) Create(ctx context.Context, breakHistory *models.BreakHistory) error {
	query := `
		INSERT INTO break_history (id, todo_id, project_id, user_id, start_time, end_time, duration, break_type, notes, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	_, err := r.db.ExecContext(ctx, query,
		breakHistory.ID,
		breakHistory.TodoID,
		breakHistory.ProjectID,
		breakHistory.UserID,
		breakHistory.StartTime,
		breakHistory.EndTime,
		breakHistory.Duration,
		breakHistory.BreakType,
		breakHistory.Notes,
		breakHistory.CreatedAt,
		breakHistory.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create break history: %w", err)
	}

	return nil
}

// GetByID retrieves a break history by ID
func (r *BreakHistoryRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.BreakHistory, error) {
	query := `
		SELECT id, todo_id, project_id, user_id, start_time, end_time, duration, break_type, notes, created_at, updated_at
		FROM break_history
		WHERE id = $1
	`

	var breakHistory models.BreakHistory
	err := r.db.GetContext(ctx, &breakHistory, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get break history by ID: %w", err)
	}

	return &breakHistory, nil
}

// GetByUserID retrieves break history for a user with pagination
func (r *BreakHistoryRepository) GetByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*models.BreakHistory, error) {
	query := `
		SELECT id, todo_id, project_id, user_id, start_time, end_time, duration, break_type, notes, created_at, updated_at
		FROM break_history
		WHERE user_id = $1
		ORDER BY start_time DESC
	`

	args := []interface{}{userID}
	if limit > 0 {
		query += " LIMIT $2"
		args = append(args, limit)
		if offset > 0 {
			query += " OFFSET $3"
			args = append(args, offset)
		}
	} else if offset > 0 {
		query += " OFFSET $2"
		args = append(args, offset)
	}

	var breakHistories []*models.BreakHistory
	err := r.db.SelectContext(ctx, &breakHistories, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get break history by user ID: %w", err)
	}

	return breakHistories, nil
}

// GetActiveBreakByUserID gets the currently active break for a user
func (r *BreakHistoryRepository) GetActiveBreakByUserID(ctx context.Context, userID uuid.UUID) (*models.BreakHistory, error) {
	query := `
		SELECT id, todo_id, project_id, user_id, start_time, end_time, duration, break_type, notes, created_at, updated_at
		FROM break_history
		WHERE user_id = $1 AND end_time IS NULL
		ORDER BY start_time DESC
		LIMIT 1
	`

	var breakHistory models.BreakHistory
	err := r.db.GetContext(ctx, &breakHistory, query, userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get active break: %w", err)
	}

	return &breakHistory, nil
}

// GetByTodoID retrieves break history for a specific todo
func (r *BreakHistoryRepository) GetByTodoID(ctx context.Context, todoID uuid.UUID, limit, offset int) ([]*models.BreakHistory, error) {
	query := `
		SELECT id, todo_id, project_id, user_id, start_time, end_time, duration, break_type, notes, created_at, updated_at
		FROM break_history
		WHERE todo_id = $1
		ORDER BY start_time DESC
	`

	args := []interface{}{todoID}
	if limit > 0 {
		query += " LIMIT $2"
		args = append(args, limit)
		if offset > 0 {
			query += " OFFSET $3"
			args = append(args, offset)
		}
	} else if offset > 0 {
		query += " OFFSET $2"
		args = append(args, offset)
	}

	var breakHistories []*models.BreakHistory
	err := r.db.SelectContext(ctx, &breakHistories, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get break history by todo ID: %w", err)
	}

	return breakHistories, nil
}

// GetByProjectID retrieves break history for a specific project
func (r *BreakHistoryRepository) GetByProjectID(ctx context.Context, projectID uuid.UUID, limit, offset int) ([]*models.BreakHistory, error) {
	query := `
		SELECT id, todo_id, project_id, user_id, start_time, end_time, duration, break_type, notes, created_at, updated_at
		FROM break_history
		WHERE project_id = $1
		ORDER BY start_time DESC
	`

	args := []interface{}{projectID}
	if limit > 0 {
		query += " LIMIT $2"
		args = append(args, limit)
		if offset > 0 {
			query += " OFFSET $3"
			args = append(args, offset)
		}
	} else if offset > 0 {
		query += " OFFSET $2"
		args = append(args, offset)
	}

	var breakHistories []*models.BreakHistory
	err := r.db.SelectContext(ctx, &breakHistories, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get break history by project ID: %w", err)
	}

	return breakHistories, nil
}

// Update updates a break history record
func (r *BreakHistoryRepository) Update(ctx context.Context, breakHistory *models.BreakHistory) error {
	breakHistory.UpdatedAt = time.Now()

	query := `
		UPDATE break_history
		SET end_time = $2, duration = $3, notes = $4, updated_at = $5
		WHERE id = $1
	`

	_, err := r.db.ExecContext(ctx, query,
		breakHistory.ID,
		breakHistory.EndTime,
		breakHistory.Duration,
		breakHistory.Notes,
		breakHistory.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to update break history: %w", err)
	}

	return nil
}

// Delete deletes a break history record
func (r *BreakHistoryRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM break_history WHERE id = $1`

	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete break history: %w", err)
	}

	return nil
}

// GetStats calculates break statistics for a user
func (r *BreakHistoryRepository) GetStats(ctx context.Context, userID uuid.UUID) (*models.BreakStatsResponse, error) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	weekStart := today.AddDate(0, 0, -int(today.Weekday()))

	stats := &models.BreakStatsResponse{}

	// Get total stats
	totalQuery := `
		SELECT 
			COUNT(*) as total_breaks,
			COALESCE(SUM(duration), 0) as total_duration
		FROM break_history 
		WHERE user_id = $1 AND end_time IS NOT NULL
	`

	var totalBreaks int64
	var totalDuration int64
	err := r.db.QueryRowContext(ctx, totalQuery, userID).Scan(&totalBreaks, &totalDuration)
	if err != nil {
		return nil, fmt.Errorf("failed to get total stats: %w", err)
	}

	stats.TotalBreaks = int(totalBreaks)
	stats.TotalDuration = int(totalDuration)

	if stats.TotalBreaks > 0 {
		stats.AverageDuration = float64(stats.TotalDuration) / float64(stats.TotalBreaks)
	}

	// Get today's stats
	todayQuery := `
		SELECT 
			COUNT(*) as today_breaks,
			COALESCE(SUM(duration), 0) as today_duration
		FROM break_history 
		WHERE user_id = $1 AND start_time >= $2 AND end_time IS NOT NULL
	`

	var todayBreaks, todayDuration int64
	err = r.db.QueryRowContext(ctx, todayQuery, userID, today).Scan(&todayBreaks, &todayDuration)
	if err != nil {
		return nil, fmt.Errorf("failed to get today stats: %w", err)
	}

	stats.TodayBreaks = int(todayBreaks)
	stats.TodayDuration = int(todayDuration)

	// Get this week's stats
	weekQuery := `
		SELECT 
			COUNT(*) as week_breaks,
			COALESCE(SUM(duration), 0) as week_duration
		FROM break_history 
		WHERE user_id = $1 AND start_time >= $2 AND end_time IS NOT NULL
	`

	var weekBreaks, weekDuration int64
	err = r.db.QueryRowContext(ctx, weekQuery, userID, weekStart).Scan(&weekBreaks, &weekDuration)
	if err != nil {
		return nil, fmt.Errorf("failed to get week stats: %w", err)
	}

	stats.ThisWeekBreaks = int(weekBreaks)
	stats.ThisWeekDuration = int(weekDuration)

	// Get current active break
	activeBreak, err := r.GetActiveBreakByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get active break: %w", err)
	}
	stats.CurrentActiveBreak = activeBreak

	return stats, nil
}
