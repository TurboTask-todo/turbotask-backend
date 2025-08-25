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
	"github.com/lib/pq"
)

// TodoRepository handles todo-related database operations
type TodoRepository struct {
	db *database.DB
}

// NewTodoRepository creates a new todo repository
func NewTodoRepository(db *database.DB) *TodoRepository {
	return &TodoRepository{db: db}
}

// Create creates a new todo
func (r *TodoRepository) Create(ctx context.Context, todo *models.Todo) error {
	todo.ID = uuid.New()
	todo.ActualTime = 0
	todo.CompletionPercentage = 0.0
	todo.IsArchived = false
	todo.IsPinned = false
	todo.IsRecurring = false

	// Set default values
	if todo.Priority == "" {
		todo.Priority = models.PriorityMedium
	}
	if todo.Status == "" {
		todo.Status = models.TaskStatusNotStarted
	}
	if todo.TimeUnit == "" {
		todo.TimeUnit = models.TimeUnitMinutes
	}

	query := `
		INSERT INTO todos (
			id, user_id, project_id, release_version_id, parent_todo_id,
			task_name, task_description, task_image_or_emoji, status_icon, estimated_time,
			actual_time, time_unit, priority, status, completion_percentage,
			due_date, start_date, completed_at, tags, difficulty_rating,
			energy_level_required, location, context, assigned_to,
			is_recurring, recurrence_pattern, is_archived, is_pinned,
			created_at, updated_at
		) VALUES (
			:id, :user_id, :project_id, :release_version_id, :parent_todo_id,
			:task_name, :task_description, :task_image_or_emoji, :status_icon, :estimated_time,
			:actual_time, :time_unit, :priority, :status, :completion_percentage,
			:due_date, :start_date, :completed_at, :tags, :difficulty_rating,
			:energy_level_required, :location, :context, :assigned_to,
			:is_recurring, :recurrence_pattern, :is_archived, :is_pinned,
			CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
		)`

	_, err := r.db.NamedExecContext(ctx, query, todo)
	if err != nil {
		return fmt.Errorf("failed to create todo: %w", err)
	}

	return nil
}

// GetByID retrieves a todo by ID with related information
func (r *TodoRepository) GetByID(ctx context.Context, id uuid.UUID, includeRelated bool) (*models.Todo, error) {
	baseQuery := `
		SELECT t.id, t.user_id, t.project_id, t.release_version_id,
		       t.parent_todo_id, t.task_name, t.task_description,
		       t.task_image_or_emoji, t.status_icon, t.estimated_time, t.actual_time,
		       t.time_unit, t.priority, t.status, t.completion_percentage,
		       t.due_date, t.start_date, t.completed_at, t.tags,
		       t.difficulty_rating, t.energy_level_required, t.location,
		       t.context, t.assigned_to, t.is_recurring, t.recurrence_pattern,
		       t.is_archived, t.is_pinned, t.created_at, t.updated_at,
		       p.title as project_title,
		       COALESCE(u.username, u.first_name || ' ' || u.last_name) as assigned_to_name
		FROM todos t
		LEFT JOIN projects p ON t.project_id = p.id
		LEFT JOIN users u ON t.assigned_to = u.id
		WHERE t.id = $1 AND t.is_archived = false`

	var todo models.Todo
	var assignedToName sql.NullString
	var projectTitle sql.NullString

	err := r.db.QueryRowContext(ctx, baseQuery, id).Scan(
		&todo.ID, &todo.UserID, &todo.ProjectID, &todo.ReleaseVersionID,
		&todo.ParentTodoID, &todo.TaskName, &todo.TaskDescription,
		&todo.TaskImageOrEmoji, &todo.StatusIcon, &todo.EstimatedTime, &todo.ActualTime,
		&todo.TimeUnit, &todo.Priority, &todo.Status, &todo.CompletionPercentage,
		&todo.DueDate, &todo.StartDate, &todo.CompletedAt, &todo.Tags,
		&todo.DifficultyRating, &todo.EnergyLevelRequired, &todo.Location,
		&todo.Context, &todo.AssignedTo, &todo.IsRecurring, &todo.RecurrencePattern,
		&todo.IsArchived, &todo.IsPinned, &todo.CreatedAt, &todo.UpdatedAt,
		&projectTitle, &assignedToName,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get todo by ID: %w", err)
	}

	if projectTitle.Valid {
		todo.ProjectTitle = &projectTitle.String
	}
	if assignedToName.Valid {
		todo.AssignedToName = &assignedToName.String
	}

	if includeRelated {
		if err := r.loadRelatedCounts(ctx, &todo); err != nil {
			return nil, err
		}
	}

	return &todo, nil
}

// GetByUserID retrieves todos for a user with filtering options
func (r *TodoRepository) GetByUserID(ctx context.Context, userID uuid.UUID, filters TodoFilters) ([]*models.Todo, int, error) {
	whereClause, args := r.buildFilterConditions(userID, filters)

	// Count query
	countQuery := fmt.Sprintf(`
		SELECT COUNT(*) FROM todos t
		LEFT JOIN projects p ON t.project_id = p.id
		WHERE %s`, whereClause)

	var total int
	err := r.db.GetContext(ctx, &total, countQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count todos: %w", err)
	}

	// Main query with pagination
	orderClause := r.buildOrderClause(filters.SortBy, filters.SortOrder)

	query := fmt.Sprintf(`
		SELECT t.id, t.user_id, t.project_id, t.release_version_id,
		       t.parent_todo_id, t.task_name, t.task_description,
		       t.task_image_or_emoji, t.status_icon, t.estimated_time, t.actual_time,
		       t.time_unit, t.priority, t.status, t.completion_percentage,
		       t.due_date, t.start_date, t.completed_at, t.tags,
		       t.difficulty_rating, t.energy_level_required, t.location,
		       t.context, t.assigned_to, t.is_recurring, t.recurrence_pattern,
		       t.is_archived, t.is_pinned, t.created_at, t.updated_at,
		       p.title as project_title,
		       COALESCE(u.username, u.first_name || ' ' || u.last_name) as assigned_to_name
		FROM todos t
		LEFT JOIN projects p ON t.project_id = p.id
		LEFT JOIN users u ON t.assigned_to = u.id
		WHERE %s
		%s
		LIMIT $%d OFFSET $%d`,
		whereClause, orderClause, len(args)+1, len(args)+2)

	args = append(args, filters.Limit, filters.Offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get todos by user ID: %w", err)
	}
	defer rows.Close()

	var todos []*models.Todo
	for rows.Next() {
		var todo models.Todo
		var projectTitle, assignedToName sql.NullString

		err := rows.Scan(
			&todo.ID, &todo.UserID, &todo.ProjectID, &todo.ReleaseVersionID,
			&todo.ParentTodoID, &todo.TaskName, &todo.TaskDescription,
			&todo.TaskImageOrEmoji, &todo.StatusIcon, &todo.EstimatedTime, &todo.ActualTime,
			&todo.TimeUnit, &todo.Priority, &todo.Status, &todo.CompletionPercentage,
			&todo.DueDate, &todo.StartDate, &todo.CompletedAt, &todo.Tags,
			&todo.DifficultyRating, &todo.EnergyLevelRequired, &todo.Location,
			&todo.Context, &todo.AssignedTo, &todo.IsRecurring, &todo.RecurrencePattern,
			&todo.IsArchived, &todo.IsPinned, &todo.CreatedAt, &todo.UpdatedAt,
			&projectTitle, &assignedToName,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan todo row: %w", err)
		}

		if projectTitle.Valid {
			todo.ProjectTitle = &projectTitle.String
		}
		if assignedToName.Valid {
			todo.AssignedToName = &assignedToName.String
		}

		todos = append(todos, &todo)
	}

	if err = rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating todo rows: %w", err)
	}

	// Load related counts if needed
	if filters.IncludeRelated {
		for _, todo := range todos {
			if err := r.loadRelatedCounts(ctx, todo); err != nil {
				return nil, 0, err
			}
		}
	}

	return todos, total, nil
}

// GetByProjectID retrieves todos for a project
func (r *TodoRepository) GetByProjectID(ctx context.Context, projectID uuid.UUID, includeCompleted bool) ([]*models.Todo, error) {
	query := `
		SELECT t.id, t.user_id, t.project_id, t.release_version_id,
		       t.parent_todo_id, t.task_name, t.task_description,
		       t.task_image_or_emoji, t.status_icon, t.estimated_time, t.actual_time,
		       t.time_unit, t.priority, t.status, t.completion_percentage,
		       t.due_date, t.start_date, t.completed_at, t.tags,
		       t.difficulty_rating, t.energy_level_required, t.location,
		       t.context, t.assigned_to, t.is_recurring, t.recurrence_pattern,
		       t.is_archived, t.is_pinned, t.created_at, t.updated_at,
		       p.title as project_title,
		       COALESCE(u.username, u.first_name || ' ' || u.last_name) as assigned_to_name
		FROM todos t
		LEFT JOIN projects p ON t.project_id = p.id
		LEFT JOIN users u ON t.assigned_to = u.id
		WHERE t.project_id = $1 AND t.is_archived = false`

	args := []interface{}{projectID}

	if !includeCompleted {
		query += ` AND t.status != 'completed'`
	}

	query += ` ORDER BY t.is_pinned DESC, t.priority DESC, t.due_date ASC NULLS LAST`

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get todos by project ID: %w", err)
	}
	defer rows.Close()

	var todos []*models.Todo
	for rows.Next() {
		var todo models.Todo
		var projectTitle, assignedToName sql.NullString

		err := rows.Scan(
			&todo.ID, &todo.UserID, &todo.ProjectID, &todo.ReleaseVersionID,
			&todo.ParentTodoID, &todo.TaskName, &todo.TaskDescription,
			&todo.TaskImageOrEmoji, &todo.StatusIcon, &todo.EstimatedTime, &todo.ActualTime,
			&todo.TimeUnit, &todo.Priority, &todo.Status, &todo.CompletionPercentage,
			&todo.DueDate, &todo.StartDate, &todo.CompletedAt, &todo.Tags,
			&todo.DifficultyRating, &todo.EnergyLevelRequired, &todo.Location,
			&todo.Context, &todo.AssignedTo, &todo.IsRecurring, &todo.RecurrencePattern,
			&todo.IsArchived, &todo.IsPinned, &todo.CreatedAt, &todo.UpdatedAt,
			&projectTitle, &assignedToName,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan todo row: %w", err)
		}

		if projectTitle.Valid {
			todo.ProjectTitle = &projectTitle.String
		}
		if assignedToName.Valid {
			todo.AssignedToName = &assignedToName.String
		}

		todos = append(todos, &todo)
	}

	return todos, nil
}

// Update updates a todo
func (r *TodoRepository) Update(ctx context.Context, id uuid.UUID, updates map[string]interface{}) error {
	if len(updates) == 0 {
		return nil
	}

	// Handle status change to completed
	if status, exists := updates["status"]; exists && status == models.TaskStatusCompleted {
		updates["completed_at"] = time.Now()
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
		UPDATE todos 
		SET %s 
		WHERE id = $%d AND is_archived = false`,
		strings.Join(setParts, ", "), argIndex)

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update todo: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("todo not found or already archived")
	}

	return nil
}

// Archive archives a todo
func (r *TodoRepository) Archive(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE todos 
		SET is_archived = true, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1 AND is_archived = false`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to archive todo: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("todo not found or already archived")
	}

	return nil
}

// Delete permanently deletes a todo
func (r *TodoRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM todos WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete todo: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("todo not found")
	}

	return nil
}

// GetDashboard retrieves todos for dashboard display
func (r *TodoRepository) GetDashboard(ctx context.Context, userID uuid.UUID, limit int) ([]*models.TodoDashboard, error) {
	query := `
		SELECT 
			t.id, t.task_name, t.status, t.priority, t.due_date, t.completion_percentage,
			p.title as project_title, p.color_theme as project_color,
			rv.version_name as release_version,
			COALESCE(u.username, u.first_name || ' ' || u.last_name) as assigned_username,
			(SELECT COUNT(*) FROM subtasks st WHERE st.todo_id = t.id AND st.is_archived = false) as subtask_count,
			(SELECT COUNT(*) FROM subtasks st WHERE st.todo_id = t.id AND st.status = 'completed') as completed_subtasks,
			(SELECT COUNT(*) FROM comments c WHERE c.todo_id = t.id) as comment_count,
			(SELECT COUNT(*) FROM attachments a WHERE a.todo_id = t.id) as attachment_count
		FROM todos t
		JOIN projects p ON t.project_id = p.id
		LEFT JOIN release_versions rv ON t.release_version_id = rv.id
		LEFT JOIN users u ON t.assigned_to = u.id
		WHERE t.user_id = $1 AND t.is_archived = false
		ORDER BY 
			CASE t.priority 
				WHEN 'urgent' THEN 1 
				WHEN 'high' THEN 2 
				WHEN 'medium' THEN 3 
				ELSE 4 
			END,
			t.due_date ASC NULLS LAST
		LIMIT $2`

	var dashboard []*models.TodoDashboard
	err := r.db.SelectContext(ctx, &dashboard, query, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get todo dashboard: %w", err)
	}

	return dashboard, nil
}

// GetOverdueTodos retrieves overdue todos for a user
func (r *TodoRepository) GetOverdueTodos(ctx context.Context, userID uuid.UUID) ([]*models.Todo, error) {
	query := `
		SELECT t.*, p.title as project_title
		FROM todos t
		LEFT JOIN projects p ON t.project_id = p.id
		WHERE t.user_id = $1 
		AND t.due_date < CURRENT_TIMESTAMP 
		AND t.status NOT IN ('completed', 'cancelled')
		AND t.is_archived = false
		ORDER BY t.due_date ASC`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get overdue todos: %w", err)
	}
	defer rows.Close()

	var todos []*models.Todo
	for rows.Next() {
		var todo models.Todo
		var projectTitle sql.NullString

		err := rows.Scan(
			&todo.ID, &todo.UserID, &todo.ProjectID, &todo.ReleaseVersionID,
			&todo.ParentTodoID, &todo.TaskName, &todo.TaskDescription,
			&todo.TaskImageOrEmoji, &todo.StatusIcon, &todo.EstimatedTime, &todo.ActualTime,
			&todo.TimeUnit, &todo.Priority, &todo.Status, &todo.CompletionPercentage,
			&todo.DueDate, &todo.StartDate, &todo.CompletedAt, &todo.Tags,
			&todo.DifficultyRating, &todo.EnergyLevelRequired, &todo.Location,
			&todo.Context, &todo.AssignedTo, &todo.IsRecurring, &todo.RecurrencePattern,
			&todo.IsArchived, &todo.IsPinned, &todo.CreatedAt, &todo.UpdatedAt,
			&projectTitle,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan overdue todo row: %w", err)
		}

		if projectTitle.Valid {
			todo.ProjectTitle = &projectTitle.String
		}

		todos = append(todos, &todo)
	}

	return todos, nil
}

// SearchTodos searches todos by text
func (r *TodoRepository) SearchTodos(ctx context.Context, userID uuid.UUID, searchTerm string, limit int, offset int) ([]*models.Todo, int, error) {
	searchPattern := "%" + strings.ToLower(searchTerm) + "%"

	// Count query
	countQuery := `
		SELECT COUNT(*) FROM todos t 
		WHERE t.user_id = $1 AND t.is_archived = false
		AND (LOWER(t.task_name) LIKE $2 OR LOWER(t.task_description) LIKE $2 
		     OR EXISTS(SELECT 1 FROM unnest(t.tags) tag WHERE LOWER(tag) LIKE $2))`

	var total int
	err := r.db.GetContext(ctx, &total, countQuery, userID, searchPattern)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count search results: %w", err)
	}

	// Main search query
	query := `
		SELECT t.*, p.title as project_title
		FROM todos t
		LEFT JOIN projects p ON t.project_id = p.id
		WHERE t.user_id = $1 AND t.is_archived = false
		AND (LOWER(t.task_name) LIKE $2 OR LOWER(t.task_description) LIKE $2 
		     OR EXISTS(SELECT 1 FROM unnest(t.tags) tag WHERE LOWER(tag) LIKE $2))
		ORDER BY 
			CASE WHEN LOWER(t.task_name) LIKE $2 THEN 1 ELSE 2 END,
			t.updated_at DESC
		LIMIT $3 OFFSET $4`

	rows, err := r.db.QueryContext(ctx, query, userID, searchPattern, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to search todos: %w", err)
	}
	defer rows.Close()

	var todos []*models.Todo
	for rows.Next() {
		var todo models.Todo
		var projectTitle sql.NullString

		err := rows.Scan(
			&todo.ID, &todo.UserID, &todo.ProjectID, &todo.ReleaseVersionID,
			&todo.ParentTodoID, &todo.TaskName, &todo.TaskDescription,
			&todo.TaskImageOrEmoji, &todo.StatusIcon, &todo.EstimatedTime, &todo.ActualTime,
			&todo.TimeUnit, &todo.Priority, &todo.Status, &todo.CompletionPercentage,
			&todo.DueDate, &todo.StartDate, &todo.CompletedAt, &todo.Tags,
			&todo.DifficultyRating, &todo.EnergyLevelRequired, &todo.Location,
			&todo.Context, &todo.AssignedTo, &todo.IsRecurring, &todo.RecurrencePattern,
			&todo.IsArchived, &todo.IsPinned, &todo.CreatedAt, &todo.UpdatedAt,
			&projectTitle,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan search result row: %w", err)
		}

		if projectTitle.Valid {
			todo.ProjectTitle = &projectTitle.String
		}

		todos = append(todos, &todo)
	}

	return todos, total, nil
}

// ValidateTodoOwnership verifies that a user owns a todo
func (r *TodoRepository) ValidateTodoOwnership(ctx context.Context, todoID, userID uuid.UUID) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM todos WHERE id = $1 AND user_id = $2 AND is_archived = false)`

	var exists bool
	err := r.db.GetContext(ctx, &exists, query, todoID, userID)
	if err != nil {
		return false, fmt.Errorf("failed to validate todo ownership: %w", err)
	}

	return exists, nil
}

// GetByTaskNameAndProject retrieves a todo by task name and project ID
func (r *TodoRepository) GetByTaskNameAndProject(ctx context.Context, taskName string, projectID uuid.UUID) (*models.Todo, error) {
	query := `
		SELECT t.id, t.user_id, t.project_id, t.release_version_id,
		       t.parent_todo_id, t.task_name, t.task_description,
		       t.task_image_or_emoji, t.status_icon, t.estimated_time, t.actual_time,
		       t.time_unit, t.priority, t.status, t.completion_percentage,
		       t.due_date, t.start_date, t.completed_at, t.tags,
		       t.difficulty_rating, t.energy_level_required, t.location,
		       t.context, t.assigned_to, t.is_recurring, t.recurrence_pattern,
		       t.is_archived, t.is_pinned, t.created_at, t.updated_at,
		       p.title as project_title,
		       COALESCE(u.username, u.first_name || ' ' || u.last_name) as assigned_to_name
		FROM todos t
		LEFT JOIN projects p ON t.project_id = p.id
		LEFT JOIN users u ON t.assigned_to = u.id
		WHERE t.task_name = $1 AND t.project_id = $2 AND t.is_archived = false`

	var todo models.Todo
	var assignedToName sql.NullString
	var projectTitle sql.NullString

	err := r.db.QueryRowContext(ctx, query, taskName, projectID).Scan(
		&todo.ID, &todo.UserID, &todo.ProjectID, &todo.ReleaseVersionID,
		&todo.ParentTodoID, &todo.TaskName, &todo.TaskDescription,
		&todo.TaskImageOrEmoji, &todo.StatusIcon, &todo.EstimatedTime, &todo.ActualTime,
		&todo.TimeUnit, &todo.Priority, &todo.Status, &todo.CompletionPercentage,
		&todo.DueDate, &todo.StartDate, &todo.CompletedAt, &todo.Tags,
		&todo.DifficultyRating, &todo.EnergyLevelRequired, &todo.Location,
		&todo.Context, &todo.AssignedTo, &todo.IsRecurring, &todo.RecurrencePattern,
		&todo.IsArchived, &todo.IsPinned, &todo.CreatedAt, &todo.UpdatedAt,
		&projectTitle, &assignedToName,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get todo by task name and project: %w", err)
	}

	if projectTitle.Valid {
		todo.ProjectTitle = &projectTitle.String
	}
	if assignedToName.Valid {
		todo.AssignedToName = &assignedToName.String
	}

	return &todo, nil
}

// Helper structures and methods

// TodoFilters represents filtering options for todos
type TodoFilters struct {
	ProjectID       *uuid.UUID            `json:"project_id,omitempty"`
	Status          *models.TaskStatus    `json:"status,omitempty"`
	Priority        *models.PriorityLevel `json:"priority,omitempty"`
	AssignedTo      *uuid.UUID            `json:"assigned_to,omitempty"`
	DueBefore       *time.Time            `json:"due_before,omitempty"`
	DueAfter        *time.Time            `json:"due_after,omitempty"`
	Tags            []string              `json:"tags,omitempty"`
	IsCompleted     *bool                 `json:"is_completed,omitempty"`
	IsPinned        *bool                 `json:"is_pinned,omitempty"`
	IncludeArchived bool                  `json:"include_archived"`
	IncludeRelated  bool                  `json:"include_related"`
	Limit           int                   `json:"limit"`
	Offset          int                   `json:"offset"`
	SortBy          string                `json:"sort_by"`
	SortOrder       string                `json:"sort_order"`
}

// buildFilterConditions builds WHERE clause and arguments for todo filtering
func (r *TodoRepository) buildFilterConditions(userID uuid.UUID, filters TodoFilters) (string, []interface{}) {
	conditions := []string{"t.user_id = $1"}
	args := []interface{}{userID}
	argIndex := 2

	if !filters.IncludeArchived {
		conditions = append(conditions, "t.is_archived = false")
	}

	if filters.ProjectID != nil {
		conditions = append(conditions, fmt.Sprintf("t.project_id = $%d", argIndex))
		args = append(args, *filters.ProjectID)
		argIndex++
	}

	if filters.Status != nil {
		conditions = append(conditions, fmt.Sprintf("t.status = $%d", argIndex))
		args = append(args, *filters.Status)
		argIndex++
	}

	if filters.Priority != nil {
		conditions = append(conditions, fmt.Sprintf("t.priority = $%d", argIndex))
		args = append(args, *filters.Priority)
		argIndex++
	}

	if filters.AssignedTo != nil {
		conditions = append(conditions, fmt.Sprintf("t.assigned_to = $%d", argIndex))
		args = append(args, *filters.AssignedTo)
		argIndex++
	}

	if filters.DueBefore != nil {
		conditions = append(conditions, fmt.Sprintf("t.due_date < $%d", argIndex))
		args = append(args, *filters.DueBefore)
		argIndex++
	}

	if filters.DueAfter != nil {
		conditions = append(conditions, fmt.Sprintf("t.due_date > $%d", argIndex))
		args = append(args, *filters.DueAfter)
		argIndex++
	}

	if len(filters.Tags) > 0 {
		conditions = append(conditions, fmt.Sprintf("t.tags && $%d", argIndex))
		args = append(args, pq.Array(filters.Tags))
		argIndex++
	}

	if filters.IsCompleted != nil {
		if *filters.IsCompleted {
			conditions = append(conditions, "t.status = 'completed'")
		} else {
			conditions = append(conditions, "t.status != 'completed'")
		}
	}

	if filters.IsPinned != nil {
		conditions = append(conditions, fmt.Sprintf("t.is_pinned = $%d", argIndex))
		args = append(args, *filters.IsPinned)
		argIndex++
	}

	return strings.Join(conditions, " AND "), args
}

// buildOrderClause builds ORDER BY clause
func (r *TodoRepository) buildOrderClause(sortBy, sortOrder string) string {
	validSortFields := map[string]string{
		"created_at": "t.created_at",
		"updated_at": "t.updated_at",
		"due_date":   "t.due_date",
		"priority":   "t.priority",
		"status":     "t.status",
		"task_name":  "t.task_name",
	}

	field, exists := validSortFields[sortBy]
	if !exists {
		field = "t.created_at"
	}

	order := "DESC"
	if sortOrder == "asc" {
		order = "ASC"
	}

	// Special handling for priority and due_date
	if sortBy == "priority" {
		return `ORDER BY 
			CASE t.priority 
				WHEN 'urgent' THEN 1 
				WHEN 'high' THEN 2 
				WHEN 'medium' THEN 3 
				ELSE 4 
			END ` + order
	}

	if sortBy == "due_date" {
		return fmt.Sprintf("ORDER BY %s %s NULLS LAST", field, order)
	}

	return fmt.Sprintf("ORDER BY %s %s", field, order)
}

// loadRelatedCounts loads related counts for a todo
func (r *TodoRepository) loadRelatedCounts(ctx context.Context, todo *models.Todo) error {
	query := `
		SELECT 
			(SELECT COUNT(*) FROM subtasks WHERE todo_id = $1 AND is_archived = false) as subtask_count,
			(SELECT COUNT(*) FROM subtasks WHERE todo_id = $1 AND status = 'completed') as completed_subtasks,
			(SELECT COUNT(*) FROM comments WHERE todo_id = $1) as comment_count,
			(SELECT COUNT(*) FROM attachments WHERE todo_id = $1) as attachment_count`

	var counts struct {
		SubtaskCount    int `db:"subtask_count"`
		CompletedSubs   int `db:"completed_subtasks"`
		CommentCount    int `db:"comment_count"`
		AttachmentCount int `db:"attachment_count"`
	}

	err := r.db.GetContext(ctx, &counts, query, todo.ID)
	if err != nil {
		return fmt.Errorf("failed to load related counts: %w", err)
	}

	todo.SubtaskCount = &counts.SubtaskCount
	todo.CompletedSubs = &counts.CompletedSubs
	todo.CommentCount = &counts.CommentCount
	todo.AttachmentCount = &counts.AttachmentCount

	return nil
}
