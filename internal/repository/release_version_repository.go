package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"macwrite-auth-api/internal/database"
	"macwrite-auth-api/internal/models"

	"github.com/google/uuid"
)

// ReleaseVersionRepository handles release version-related database operations
type ReleaseVersionRepository struct {
	db *database.DB
}

// NewReleaseVersionRepository creates a new release version repository
func NewReleaseVersionRepository(db *database.DB) *ReleaseVersionRepository {
	return &ReleaseVersionRepository{db: db}
}

// Create creates a new release version
func (r *ReleaseVersionRepository) Create(ctx context.Context, version *models.ReleaseVersion) error {
	version.ID = uuid.New()
	version.IsReleased = false

	query := `
		INSERT INTO release_versions (
			id, project_id, version_name, version_number, description,
			target_date, release_date, is_released, release_notes,
			created_at, updated_at
		) VALUES (
			:id, :project_id, :version_name, :version_number, :description,
			:target_date, :release_date, :is_released, :release_notes,
			CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
		)`

	_, err := r.db.NamedExecContext(ctx, query, version)
	if err != nil {
		if database.IsUniqueViolation(err) {
			return fmt.Errorf("version number already exists for this project")
		}
		return fmt.Errorf("failed to create release version: %w", err)
	}

	return nil
}

// GetByID retrieves a release version by ID with optional todo counts
func (r *ReleaseVersionRepository) GetByID(ctx context.Context, id uuid.UUID, includeCounts bool) (*models.ReleaseVersion, error) {
	var version models.ReleaseVersion
	query := `SELECT * FROM release_versions WHERE id = $1`

	err := r.db.GetContext(ctx, &version, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get release version by ID: %w", err)
	}

	if includeCounts {
		if err := r.loadTodoCounts(ctx, &version); err != nil {
			return nil, err
		}
	}

	return &version, nil
}

// GetByProjectID retrieves all release versions for a project
func (r *ReleaseVersionRepository) GetByProjectID(ctx context.Context, projectID uuid.UUID, includeCounts bool) ([]*models.ReleaseVersion, error) {
	query := `
		SELECT * FROM release_versions 
		WHERE project_id = $1 
		ORDER BY is_released ASC, target_date ASC NULLS LAST, created_at DESC`

	var versions []*models.ReleaseVersion
	err := r.db.SelectContext(ctx, &versions, query, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to get release versions by project ID: %w", err)
	}

	if includeCounts {
		for _, version := range versions {
			if err := r.loadTodoCounts(ctx, version); err != nil {
				return nil, err
			}
		}
	}

	return versions, nil
}

// GetByVersionNumber retrieves a release version by project ID and version number
func (r *ReleaseVersionRepository) GetByVersionNumber(ctx context.Context, projectID uuid.UUID, versionNumber string) (*models.ReleaseVersion, error) {
	var version models.ReleaseVersion
	query := `
		SELECT * FROM release_versions 
		WHERE project_id = $1 AND version_number = $2`

	err := r.db.GetContext(ctx, &version, query, projectID, versionNumber)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get release version by version number: %w", err)
	}

	return &version, nil
}

// Update updates a release version
func (r *ReleaseVersionRepository) Update(ctx context.Context, id uuid.UUID, updates map[string]interface{}) error {
	if len(updates) == 0 {
		return nil
	}

	// Build dynamic update query
	setParts := make([]string, 0, len(updates))
	args := make([]interface{}, 0, len(updates)+1)
	argIndex := 1

	for field, value := range updates {
		setParts = append(setParts, fmt.Sprintf("%s = $%d", field, argIndex))
		args = append(args, value)
		argIndex++
	}

	// Always update the updated_at field
	setParts = append(setParts, fmt.Sprintf("updated_at = $%d", argIndex))
	args = append(args, time.Now())
	argIndex++

	// Add the ID for WHERE clause
	args = append(args, id)

	query := fmt.Sprintf(`
		UPDATE release_versions 
		SET %s 
		WHERE id = $%d`,
		strings.Join(setParts, ", "), argIndex)

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		if database.IsUniqueViolation(err) {
			return fmt.Errorf("version number already exists for this project")
		}
		return fmt.Errorf("failed to update release version: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("release version not found")
	}

	return nil
}

// Delete deletes a release version
func (r *ReleaseVersionRepository) Delete(ctx context.Context, id uuid.UUID) error {
	// Check if there are any todos assigned to this release version
	var todoCount int
	countQuery := `SELECT COUNT(*) FROM todos WHERE release_version_id = $1`
	err := r.db.GetContext(ctx, &todoCount, countQuery, id)
	if err != nil {
		return fmt.Errorf("failed to check todo count: %w", err)
	}

	if todoCount > 0 {
		return fmt.Errorf("cannot delete release version with assigned todos")
	}

	query := `DELETE FROM release_versions WHERE id = $1`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete release version: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("release version not found")
	}

	return nil
}

// MarkAsReleased marks a release version as released
func (r *ReleaseVersionRepository) MarkAsReleased(ctx context.Context, id uuid.UUID, releaseNotes string) error {
	query := `
		UPDATE release_versions 
		SET is_released = true, 
			release_date = CURRENT_DATE,
			release_notes = $2,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = $1 AND is_released = false`

	result, err := r.db.ExecContext(ctx, query, id, releaseNotes)
	if err != nil {
		return fmt.Errorf("failed to mark release version as released: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("release version not found or already released")
	}

	return nil
}

// GetUpcomingReleases retrieves upcoming releases (not yet released) within a date range
func (r *ReleaseVersionRepository) GetUpcomingReleases(ctx context.Context, projectID *uuid.UUID, daysAhead int) ([]*models.ReleaseVersion, error) {
	query := `
		SELECT rv.* FROM release_versions rv
		WHERE rv.is_released = false 
		AND rv.target_date IS NOT NULL
		AND rv.target_date <= CURRENT_DATE + INTERVAL '%d days'`

	args := []interface{}{}

	if projectID != nil {
		query += ` AND rv.project_id = $1`
		args = append(args, *projectID)
	}

	query += ` ORDER BY rv.target_date ASC`
	query = fmt.Sprintf(query, daysAhead)

	var versions []*models.ReleaseVersion
	err := r.db.SelectContext(ctx, &versions, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get upcoming releases: %w", err)
	}

	return versions, nil
}

// GetOverdueReleases retrieves releases that are past their target date but not released
func (r *ReleaseVersionRepository) GetOverdueReleases(ctx context.Context, projectID *uuid.UUID) ([]*models.ReleaseVersion, error) {
	query := `
		SELECT rv.* FROM release_versions rv
		WHERE rv.is_released = false 
		AND rv.target_date IS NOT NULL
		AND rv.target_date < CURRENT_DATE`

	args := []interface{}{}

	if projectID != nil {
		query += ` AND rv.project_id = $1`
		args = append(args, *projectID)
	}

	query += ` ORDER BY rv.target_date ASC`

	var versions []*models.ReleaseVersion
	err := r.db.SelectContext(ctx, &versions, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get overdue releases: %w", err)
	}

	return versions, nil
}

// GetReleaseStats retrieves comprehensive statistics for a release version
func (r *ReleaseVersionRepository) GetReleaseStats(ctx context.Context, id uuid.UUID) (map[string]interface{}, error) {
	query := `
		SELECT 
			COUNT(*) as total_todos,
			COUNT(CASE WHEN t.status = 'completed' THEN 1 END) as completed_todos,
			COUNT(CASE WHEN t.status = 'in_progress' THEN 1 END) as in_progress_todos,
			COUNT(CASE WHEN t.status = 'not_started' THEN 1 END) as not_started_todos,
			COUNT(CASE WHEN t.due_date < CURRENT_TIMESTAMP AND t.status != 'completed' THEN 1 END) as overdue_todos,
			SUM(t.estimated_time) as total_estimated_time,
			SUM(t.actual_time) as total_actual_time,
			AVG(t.actual_time) as avg_task_duration,
			MIN(t.created_at) as first_todo_created,
			MAX(t.completed_at) as last_todo_completed
		FROM todos t
		WHERE t.release_version_id = $1 AND t.is_archived = false`

	row := r.db.QueryRowContext(ctx, query, id)

	var stats struct {
		TotalTodos         int             `db:"total_todos"`
		CompletedTodos     int             `db:"completed_todos"`
		InProgressTodos    int             `db:"in_progress_todos"`
		NotStartedTodos    int             `db:"not_started_todos"`
		OverdueTodos       int             `db:"overdue_todos"`
		TotalEstimatedTime sql.NullInt64   `db:"total_estimated_time"`
		TotalActualTime    sql.NullInt64   `db:"total_actual_time"`
		AvgTaskDuration    sql.NullFloat64 `db:"avg_task_duration"`
		FirstTodoCreated   sql.NullTime    `db:"first_todo_created"`
		LastTodoCompleted  sql.NullTime    `db:"last_todo_completed"`
	}

	err := row.Scan(
		&stats.TotalTodos, &stats.CompletedTodos, &stats.InProgressTodos,
		&stats.NotStartedTodos, &stats.OverdueTodos, &stats.TotalEstimatedTime,
		&stats.TotalActualTime, &stats.AvgTaskDuration, &stats.FirstTodoCreated,
		&stats.LastTodoCompleted,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get release stats: %w", err)
	}

	result := map[string]interface{}{
		"total_todos":       stats.TotalTodos,
		"completed_todos":   stats.CompletedTodos,
		"in_progress_todos": stats.InProgressTodos,
		"not_started_todos": stats.NotStartedTodos,
		"overdue_todos":     stats.OverdueTodos,
	}

	if stats.TotalTodos > 0 {
		completionPercentage := float64(stats.CompletedTodos) / float64(stats.TotalTodos) * 100
		result["completion_percentage"] = completionPercentage
	} else {
		result["completion_percentage"] = 0.0
	}

	if stats.TotalEstimatedTime.Valid {
		result["total_estimated_time"] = stats.TotalEstimatedTime.Int64
	}
	if stats.TotalActualTime.Valid {
		result["total_actual_time"] = stats.TotalActualTime.Int64
	}
	if stats.AvgTaskDuration.Valid {
		result["avg_task_duration"] = stats.AvgTaskDuration.Float64
	}
	if stats.FirstTodoCreated.Valid {
		result["first_todo_created"] = stats.FirstTodoCreated.Time
	}
	if stats.LastTodoCompleted.Valid {
		result["last_todo_completed"] = stats.LastTodoCompleted.Time
	}

	return result, nil
}

// ValidateReleaseOwnership verifies that a user owns a release version through project ownership
func (r *ReleaseVersionRepository) ValidateReleaseOwnership(ctx context.Context, releaseID, userID uuid.UUID) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM release_versions rv
			JOIN projects p ON rv.project_id = p.id
			WHERE rv.id = $1 AND p.user_id = $2 AND p.is_archived = false
		)`

	var exists bool
	err := r.db.GetContext(ctx, &exists, query, releaseID, userID)
	if err != nil {
		return false, fmt.Errorf("failed to validate release ownership: %w", err)
	}

	return exists, nil
}

// GetReleasesTodosProgress retrieves todos for a release version with progress information
func (r *ReleaseVersionRepository) GetReleaseTodosProgress(ctx context.Context, id uuid.UUID) ([]*models.Todo, error) {
	query := `
		SELECT 
			t.id, t.task_name, t.status, t.priority, t.completion_percentage,
			t.estimated_time, t.actual_time, t.due_date, t.created_at, t.updated_at,
			(SELECT COUNT(*) FROM subtasks st WHERE st.todo_id = t.id AND st.is_archived = false) as subtask_count,
			(SELECT COUNT(*) FROM subtasks st WHERE st.todo_id = t.id AND st.status = 'completed') as completed_subtasks
		FROM todos t
		WHERE t.release_version_id = $1 AND t.is_archived = false
		ORDER BY 
			CASE t.status 
				WHEN 'completed' THEN 3
				WHEN 'in_progress' THEN 1
				WHEN 'not_started' THEN 2
				ELSE 4
			END,
			CASE t.priority 
				WHEN 'urgent' THEN 1 
				WHEN 'high' THEN 2 
				WHEN 'medium' THEN 3 
				ELSE 4 
			END,
			t.due_date ASC NULLS LAST`

	rows, err := r.db.QueryContext(ctx, query, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get release todos progress: %w", err)
	}
	defer rows.Close()

	var todos []*models.Todo
	for rows.Next() {
		var todo models.Todo
		var subtaskCount, completedSubtasks sql.NullInt64

		err := rows.Scan(
			&todo.ID, &todo.TaskName, &todo.Status, &todo.Priority,
			&todo.CompletionPercentage, &todo.EstimatedTime, &todo.ActualTime,
			&todo.DueDate, &todo.CreatedAt, &todo.UpdatedAt,
			&subtaskCount, &completedSubtasks,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan todo row: %w", err)
		}

		if subtaskCount.Valid {
			count := int(subtaskCount.Int64)
			todo.SubtaskCount = &count
		}
		if completedSubtasks.Valid {
			count := int(completedSubtasks.Int64)
			todo.CompletedSubs = &count
		}

		todos = append(todos, &todo)
	}

	return todos, nil
}

// Helper method to load todo counts for a release version
func (r *ReleaseVersionRepository) loadTodoCounts(ctx context.Context, version *models.ReleaseVersion) error {
	query := `
		SELECT 
			COUNT(*) as total_todos,
			COUNT(CASE WHEN status = 'completed' THEN 1 END) as completed_todos
		FROM todos 
		WHERE release_version_id = $1 AND is_archived = false`

	var counts struct {
		TotalTodos     int `db:"total_todos"`
		CompletedTodos int `db:"completed_todos"`
	}

	err := r.db.GetContext(ctx, &counts, query, version.ID)
	if err != nil {
		return fmt.Errorf("failed to load todo counts: %w", err)
	}

	version.TodoCount = &counts.TotalTodos
	version.CompletedCount = &counts.CompletedTodos

	return nil
}
