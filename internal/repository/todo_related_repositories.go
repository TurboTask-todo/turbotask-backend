package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"quantumtask-auth-api/internal/database"
	"quantumtask-auth-api/internal/models"

	"github.com/google/uuid"
)

// ========================================
// SUBTASK REPOSITORY
// ========================================

// SubtaskRepository handles subtask-related database operations
type SubtaskRepository struct {
	db *database.DB
}

// NewSubtaskRepository creates a new subtask repository
func NewSubtaskRepository(db *database.DB) *SubtaskRepository {
	return &SubtaskRepository{db: db}
}

// Create creates a new subtask
func (r *SubtaskRepository) Create(ctx context.Context, subtask *models.Subtask) error {
	subtask.ID = uuid.New()
	subtask.ActualTime = 0
	subtask.IsArchived = false

	if subtask.Priority == "" {
		subtask.Priority = models.PriorityMedium
	}
	if subtask.Status == "" {
		subtask.Status = models.TaskStatusNotStarted
	}

	query := `
		INSERT INTO subtasks (
			id, todo_id, name, description, status, priority,
			estimated_time, actual_time, due_date, completed_at,
			sort_order, is_archived, created_at, updated_at
		) VALUES (
			:id, :todo_id, :name, :description, :status, :priority,
			:estimated_time, :actual_time, :due_date, :completed_at,
			:sort_order, :is_archived, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
		)`

	_, err := r.db.NamedExecContext(ctx, query, subtask)
	if err != nil {
		return fmt.Errorf("failed to create subtask: %w", err)
	}

	return nil
}

// GetByTodoID retrieves all subtasks for a todo
func (r *SubtaskRepository) GetByTodoID(ctx context.Context, todoID uuid.UUID, includeArchived bool) ([]*models.Subtask, error) {
	query := `
		SELECT * FROM subtasks 
		WHERE todo_id = $1`

	args := []interface{}{todoID}

	if !includeArchived {
		query += ` AND is_archived = false`
	}

	query += ` ORDER BY sort_order ASC, created_at ASC`

	var subtasks []*models.Subtask
	err := r.db.SelectContext(ctx, &subtasks, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get subtasks by todo ID: %w", err)
	}

	return subtasks, nil
}

// GetByID retrieves a subtask by ID
func (r *SubtaskRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Subtask, error) {
	var subtask models.Subtask
	query := `SELECT * FROM subtasks WHERE id = $1 AND is_archived = false`

	err := r.db.GetContext(ctx, &subtask, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get subtask by ID: %w", err)
	}

	return &subtask, nil
}

// Update updates a subtask
func (r *SubtaskRepository) Update(ctx context.Context, id uuid.UUID, updates map[string]interface{}) error {
	if len(updates) == 0 {
		return nil
	}

	setParts := make([]string, 0, len(updates))
	args := make([]interface{}, 0, len(updates)+1)
	argIndex := 1

	for field, value := range updates {
		setParts = append(setParts, fmt.Sprintf("%s = $%d", field, argIndex))
		args = append(args, value)
		argIndex++
	}

	setParts = append(setParts, fmt.Sprintf("updated_at = $%d", argIndex))
	args = append(args, time.Now())
	argIndex++

	args = append(args, id)

	query := fmt.Sprintf(`
		UPDATE subtasks 
		SET %s 
		WHERE id = $%d AND is_archived = false`,
		strings.Join(setParts, ", "), argIndex)

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update subtask: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("subtask not found or already archived")
	}

	return nil
}

// Delete deletes a subtask
func (r *SubtaskRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM subtasks WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete subtask: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("subtask not found")
	}

	return nil
}

// ReorderSubtasks updates the sort order of subtasks
func (r *SubtaskRepository) ReorderSubtasks(ctx context.Context, todoID uuid.UUID, subtaskIDs []uuid.UUID) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	for i, subtaskID := range subtaskIDs {
		query := `UPDATE subtasks SET sort_order = $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2 AND todo_id = $3`
		_, err := tx.ExecContext(ctx, query, i, subtaskID, todoID)
		if err != nil {
			return fmt.Errorf("failed to update subtask sort order: %w", err)
		}
	}

	return tx.Commit()
}

// SubtaskWithTodo represents a subtask with its parent todo information
type SubtaskWithTodo struct {
	*models.Subtask
	TodoTitle       string `json:"todo_title" db:"todo_title"`
	TodoDescription string `json:"todo_description" db:"todo_description"`
	TodoStatus      string `json:"todo_status" db:"todo_status"`
	TodoPriority    string `json:"todo_priority" db:"todo_priority"`
}

// SubtaskQueryOptions represents query options for fetching subtasks
type SubtaskQueryOptions struct {
	IncludeArchived bool
	Status          string
	Priority        string
	Search          string
	SortBy          string
	SortOrder       string
	Limit           int
	Offset          int
}

// GetByProjectIDWithPagination retrieves all subtasks for a project with pagination
func (r *SubtaskRepository) GetByProjectIDWithPagination(ctx context.Context, projectID uuid.UUID, options SubtaskQueryOptions) ([]*SubtaskWithTodo, int, error) {
	// Count total records first
	countQuery := `
		SELECT COUNT(DISTINCT s.id)
		FROM subtasks s
		INNER JOIN todos t ON s.todo_id = t.id
		WHERE t.project_id = $1`

	countArgs := []interface{}{projectID}

	if !options.IncludeArchived {
		countQuery += ` AND s.is_archived = false AND t.is_archived = false`
	}

	if options.Status != "" {
		countQuery += ` AND s.status = $` + fmt.Sprintf("%d", len(countArgs)+1)
		countArgs = append(countArgs, options.Status)
	}

	if options.Priority != "" {
		countQuery += ` AND s.priority = $` + fmt.Sprintf("%d", len(countArgs)+1)
		countArgs = append(countArgs, options.Priority)
	}

	if options.Search != "" {
		searchPattern := "%" + options.Search + "%"
		countQuery += ` AND (s.name ILIKE $` + fmt.Sprintf("%d", len(countArgs)+1) +
			` OR s.description ILIKE $` + fmt.Sprintf("%d", len(countArgs)+2) +
			` OR t.task_name ILIKE $` + fmt.Sprintf("%d", len(countArgs)+3) + `)`
		countArgs = append(countArgs, searchPattern, searchPattern, searchPattern)
	}

	var totalCount int
	err := r.db.GetContext(ctx, &totalCount, countQuery, countArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count subtasks: %w", err)
	}

	// Main query for fetching data
	query := `
		SELECT 
			s.id, s.todo_id, s.name, s.description, s.status, s.priority,
			s.estimated_time, s.actual_time, s.due_date, s.completed_at,
			s.sort_order, s.is_archived, s.created_at, s.updated_at,
			t.task_name as todo_title,
			COALESCE(t.task_description, '') as todo_description,
			t.status as todo_status,
			t.priority as todo_priority
		FROM subtasks s
		INNER JOIN todos t ON s.todo_id = t.id
		WHERE t.project_id = $1`

	args := []interface{}{projectID}

	if !options.IncludeArchived {
		query += ` AND s.is_archived = false AND t.is_archived = false`
	}

	if options.Status != "" {
		query += ` AND s.status = $` + fmt.Sprintf("%d", len(args)+1)
		args = append(args, options.Status)
	}

	if options.Priority != "" {
		query += ` AND s.priority = $` + fmt.Sprintf("%d", len(args)+1)
		args = append(args, options.Priority)
	}

	if options.Search != "" {
		searchPattern := "%" + options.Search + "%"
		query += ` AND (s.name ILIKE $` + fmt.Sprintf("%d", len(args)+1) +
			` OR s.description ILIKE $` + fmt.Sprintf("%d", len(args)+2) +
			` OR t.task_name ILIKE $` + fmt.Sprintf("%d", len(args)+3) + `)`
		args = append(args, searchPattern, searchPattern, searchPattern)
	}

	// Add sorting
	sortField := "s.created_at"
	sortOrder := "DESC"

	switch options.SortBy {
	case "name":
		sortField = "s.name"
	case "status":
		sortField = "s.status"
	case "priority":
		sortField = "s.priority"
	case "due_date":
		sortField = "s.due_date"
	case "todo_title":
		sortField = "t.task_name"
	case "sort_order":
		sortField = "s.sort_order"
		sortOrder = "ASC" // Default ascending for sort_order
	}

	if options.SortOrder != "" {
		sortOrder = strings.ToUpper(options.SortOrder)
	}

	query += fmt.Sprintf(` ORDER BY %s %s`, sortField, sortOrder)

	// Add pagination
	if options.Limit > 0 {
		query += ` LIMIT $` + fmt.Sprintf("%d", len(args)+1)
		args = append(args, options.Limit)

		if options.Offset > 0 {
			query += ` OFFSET $` + fmt.Sprintf("%d", len(args)+1)
			args = append(args, options.Offset)
		}
	}

	var subtasks []*SubtaskWithTodo
	err = r.db.SelectContext(ctx, &subtasks, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get subtasks by project ID: %w", err)
	}

	return subtasks, totalCount, nil
}

// ========================================
// NOTE REPOSITORY
// ========================================

// NoteRepository handles note-related database operations
type NoteRepository struct {
	db *database.DB
}

// NewNoteRepository creates a new note repository
func NewNoteRepository(db *database.DB) *NoteRepository {
	return &NoteRepository{db: db}
}

// Create creates a new note
func (r *NoteRepository) Create(ctx context.Context, note *models.Note) error {
	note.ID = uuid.New()
	note.IsPinned = false

	if note.NoteType == "" {
		note.NoteType = "general"
	}

	query := `
		INSERT INTO notes (
			id, todo_id, title, content, note_type, is_pinned, tags,
			created_at, updated_at
		) VALUES (
			:id, :todo_id, :title, :content, :note_type, :is_pinned, :tags,
			CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
		)`

	_, err := r.db.NamedExecContext(ctx, query, note)
	if err != nil {
		return fmt.Errorf("failed to create note: %w", err)
	}

	return nil
}

// GetByTodoID retrieves all notes for a todo
func (r *NoteRepository) GetByTodoID(ctx context.Context, todoID uuid.UUID) ([]*models.Note, error) {
	query := `
		SELECT * FROM notes 
		WHERE todo_id = $1 
		ORDER BY is_pinned DESC, created_at DESC`

	var notes []*models.Note
	err := r.db.SelectContext(ctx, &notes, query, todoID)
	if err != nil {
		return nil, fmt.Errorf("failed to get notes by todo ID: %w", err)
	}

	return notes, nil
}

// GetByID retrieves a note by ID
func (r *NoteRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Note, error) {
	var note models.Note
	query := `SELECT * FROM notes WHERE id = $1`

	err := r.db.GetContext(ctx, &note, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get note by ID: %w", err)
	}

	return &note, nil
}

// Update updates a note
func (r *NoteRepository) Update(ctx context.Context, id uuid.UUID, updates map[string]interface{}) error {
	if len(updates) == 0 {
		return nil
	}

	setParts := make([]string, 0, len(updates))
	args := make([]interface{}, 0, len(updates)+1)
	argIndex := 1

	for field, value := range updates {
		setParts = append(setParts, fmt.Sprintf("%s = $%d", field, argIndex))
		args = append(args, value)
		argIndex++
	}

	setParts = append(setParts, fmt.Sprintf("updated_at = $%d", argIndex))
	args = append(args, time.Now())
	argIndex++

	args = append(args, id)

	query := fmt.Sprintf(`
		UPDATE notes 
		SET %s 
		WHERE id = $%d`,
		strings.Join(setParts, ", "), argIndex)

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update note: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("note not found")
	}

	return nil
}

// Delete deletes a note
func (r *NoteRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM notes WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete note: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("note not found")
	}

	return nil
}

// SearchNotes searches notes by content or tags
func (r *NoteRepository) SearchNotes(ctx context.Context, todoID uuid.UUID, searchTerm string) ([]*models.Note, error) {
	searchPattern := "%" + strings.ToLower(searchTerm) + "%"

	query := `
		SELECT * FROM notes 
		WHERE todo_id = $1 
		AND (LOWER(content) LIKE $2 OR LOWER(title) LIKE $2
		     OR EXISTS(SELECT 1 FROM unnest(tags) tag WHERE LOWER(tag) LIKE $2))
		ORDER BY is_pinned DESC, updated_at DESC`

	var notes []*models.Note
	err := r.db.SelectContext(ctx, &notes, query, todoID, searchPattern)
	if err != nil {
		return nil, fmt.Errorf("failed to search notes: %w", err)
	}

	return notes, nil
}

// ========================================
// SCHEDULED TASK REPOSITORY
// ========================================

// ScheduledTaskRepository handles scheduled task-related database operations
type ScheduledTaskRepository struct {
	db *database.DB
}

// NewScheduledTaskRepository creates a new scheduled task repository
func NewScheduledTaskRepository(db *database.DB) *ScheduledTaskRepository {
	return &ScheduledTaskRepository{db: db}
}

// Create creates a new scheduled task
func (r *ScheduledTaskRepository) Create(ctx context.Context, task *models.ScheduledTask) error {
	task.ID = uuid.New()
	task.NotificationSent = false
	task.IsAllDay = false
	task.IsCancelled = false

	if task.ReminderMinutesBefore == 0 {
		task.ReminderMinutesBefore = 15
	}
	if task.NotificationFrequency == "" {
		task.NotificationFrequency = models.NotificationFrequencyOnce
	}

	// Set default values for recurring fields
	if task.RecurrencePattern == "" {
		task.RecurrencePattern = models.RecurrencePatternNone
	}
	if task.RecurrenceInterval == 0 {
		task.RecurrenceInterval = 1
	}

	query := `
		INSERT INTO scheduled_tasks (
			id, todo_id, user_id, scheduled_datetime, duration_minutes,
			notification_sent, notification_frequency, reminder_minutes_before,
			location, meeting_url, attendees, is_all_day, is_cancelled,
			is_recurring, recurrence_pattern, recurrence_interval, recurrence_end_date,
			parent_schedule_id, is_parent_schedule, recurrence_count, weekday_mask,
			created_at, updated_at
		) VALUES (
			:id, :todo_id, :user_id, :scheduled_datetime, :duration_minutes,
			:notification_sent, :notification_frequency, :reminder_minutes_before,
			:location, :meeting_url, :attendees, :is_all_day, :is_cancelled,
			:is_recurring, :recurrence_pattern, :recurrence_interval, :recurrence_end_date,
			:parent_schedule_id, :is_parent_schedule, :recurrence_count, :weekday_mask,
			CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
		)`

	_, err := r.db.NamedExecContext(ctx, query, task)
	if err != nil {
		return fmt.Errorf("failed to create scheduled task: %w", err)
	}

	return nil
}

// GetByUserID retrieves scheduled tasks for a user within a date range
func (r *ScheduledTaskRepository) GetByUserID(ctx context.Context, userID uuid.UUID, fromDate, toDate *string) ([]*models.ScheduledTask, error) {
	query := `
		SELECT st.*, t.task_name as todo_title
		FROM scheduled_tasks st
		LEFT JOIN todos t ON st.todo_id = t.id
		WHERE st.user_id = $1 AND st.is_cancelled = false`

	args := []interface{}{userID}
	argIndex := 2

	if fromDate != nil {
		query += fmt.Sprintf(` AND st.scheduled_datetime >= $%d`, argIndex)
		args = append(args, *fromDate)
		argIndex++
	}

	if toDate != nil {
		query += fmt.Sprintf(` AND st.scheduled_datetime <= $%d`, argIndex)
		args = append(args, *toDate)
		argIndex++
	}

	query += ` ORDER BY st.scheduled_datetime ASC`

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get scheduled tasks by user ID: %w", err)
	}
	defer rows.Close()

	var tasks []*models.ScheduledTask
	for rows.Next() {
		var task models.ScheduledTask
		var todoTitle sql.NullString

		err := rows.Scan(
			&task.ID, &task.TodoID, &task.UserID, &task.ScheduledDatetime,
			&task.DurationMinutes, &task.NotificationSent, &task.NotificationFrequency,
			&task.ReminderMinutesBefore, &task.Location, &task.MeetingURL,
			&task.Attendees, &task.IsAllDay, &task.IsCancelled,
			&task.IsRecurring, &task.RecurrencePattern, &task.RecurrenceInterval,
			&task.RecurrenceEndDate, &task.ParentScheduleID, &task.IsParentSchedule,
			&task.RecurrenceCount, &task.WeekdayMask,
			&task.CreatedAt, &task.UpdatedAt, &todoTitle,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan scheduled task row: %w", err)
		}

		if todoTitle.Valid {
			task.TodoTitle = &todoTitle.String
		}

		tasks = append(tasks, &task)
	}

	return tasks, nil
}

// GetByID retrieves a scheduled task by ID
func (r *ScheduledTaskRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.ScheduledTask, error) {
	var task models.ScheduledTask
	query := `
		SELECT st.*, t.task_name as todo_title
		FROM scheduled_tasks st
		LEFT JOIN todos t ON st.todo_id = t.id
		WHERE st.id = $1`

	var todoTitle sql.NullString
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&task.ID, &task.TodoID, &task.UserID, &task.ScheduledDatetime,
		&task.DurationMinutes, &task.NotificationSent, &task.NotificationFrequency,
		&task.ReminderMinutesBefore, &task.Location, &task.MeetingURL,
		&task.Attendees, &task.IsAllDay, &task.IsCancelled,
		&task.CreatedAt, &task.UpdatedAt, &todoTitle,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get scheduled task by ID: %w", err)
	}

	if todoTitle.Valid {
		task.TodoTitle = &todoTitle.String
	}

	return &task, nil
}

// Update updates a scheduled task
func (r *ScheduledTaskRepository) Update(ctx context.Context, id uuid.UUID, updates map[string]interface{}) error {
	if len(updates) == 0 {
		return nil
	}

	setParts := make([]string, 0, len(updates))
	args := make([]interface{}, 0, len(updates)+1)
	argIndex := 1

	for field, value := range updates {
		setParts = append(setParts, fmt.Sprintf("%s = $%d", field, argIndex))
		args = append(args, value)
		argIndex++
	}

	setParts = append(setParts, fmt.Sprintf("updated_at = $%d", argIndex))
	args = append(args, time.Now())
	argIndex++

	args = append(args, id)

	query := fmt.Sprintf(`
		UPDATE scheduled_tasks 
		SET %s 
		WHERE id = $%d`,
		strings.Join(setParts, ", "), argIndex)

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update scheduled task: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("scheduled task not found")
	}

	return nil
}

// Delete deletes a scheduled task
func (r *ScheduledTaskRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM scheduled_tasks WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete scheduled task: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("scheduled task not found")
	}

	return nil
}

// GetPendingNotifications retrieves scheduled tasks that need notifications
func (r *ScheduledTaskRepository) GetPendingNotifications(ctx context.Context) ([]*models.ScheduledTask, error) {
	query := `
		SELECT st.*, t.task_name as todo_title
		FROM scheduled_tasks st
		LEFT JOIN todos t ON st.todo_id = t.id
		WHERE st.notification_sent = false 
		AND st.is_cancelled = false
		AND st.scheduled_datetime <= CURRENT_TIMESTAMP + (st.reminder_minutes_before || ' minutes')::INTERVAL
		ORDER BY st.scheduled_datetime ASC`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get pending notifications: %w", err)
	}
	defer rows.Close()

	var tasks []*models.ScheduledTask
	for rows.Next() {
		var task models.ScheduledTask
		var todoTitle sql.NullString

		err := rows.Scan(
			&task.ID, &task.TodoID, &task.UserID, &task.ScheduledDatetime,
			&task.DurationMinutes, &task.NotificationSent, &task.NotificationFrequency,
			&task.ReminderMinutesBefore, &task.Location, &task.MeetingURL,
			&task.Attendees, &task.IsAllDay, &task.IsCancelled,
			&task.CreatedAt, &task.UpdatedAt, &todoTitle,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan pending notification row: %w", err)
		}

		if todoTitle.Valid {
			task.TodoTitle = &todoTitle.String
		}

		tasks = append(tasks, &task)
	}

	return tasks, nil
}

// MarkNotificationSent marks a scheduled task notification as sent
func (r *ScheduledTaskRepository) MarkNotificationSent(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE scheduled_tasks 
		SET notification_sent = true, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1`

	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to mark notification as sent: %w", err)
	}

	return nil
}

// GetRecurringParents retrieves all parent recurring schedules that need child instances generated
func (r *ScheduledTaskRepository) GetRecurringParents(ctx context.Context) ([]*models.ScheduledTask, error) {
	query := `
		SELECT st.*, t.task_name as todo_title
		FROM scheduled_tasks st
		LEFT JOIN todos t ON st.todo_id = t.id
		WHERE st.is_parent_schedule = true 
		AND st.is_recurring = true 
		AND st.is_cancelled = false
		AND (st.recurrence_end_date IS NULL OR st.recurrence_end_date >= CURRENT_TIMESTAMP)
		ORDER BY st.scheduled_datetime ASC`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get recurring parent schedules: %w", err)
	}
	defer rows.Close()

	var tasks []*models.ScheduledTask
	for rows.Next() {
		var task models.ScheduledTask
		var todoTitle sql.NullString

		err := rows.Scan(
			&task.ID, &task.TodoID, &task.UserID, &task.ScheduledDatetime,
			&task.DurationMinutes, &task.NotificationSent, &task.NotificationFrequency,
			&task.ReminderMinutesBefore, &task.Location, &task.MeetingURL,
			&task.Attendees, &task.IsAllDay, &task.IsCancelled,
			&task.IsRecurring, &task.RecurrencePattern, &task.RecurrenceInterval,
			&task.RecurrenceEndDate, &task.ParentScheduleID, &task.IsParentSchedule,
			&task.RecurrenceCount, &task.WeekdayMask,
			&task.CreatedAt, &task.UpdatedAt, &todoTitle,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan recurring parent row: %w", err)
		}

		if todoTitle.Valid {
			task.TodoTitle = &todoTitle.String
		}

		tasks = append(tasks, &task)
	}

	return tasks, nil
}

// GetChildrenByParentID retrieves all child instances for a parent recurring schedule
func (r *ScheduledTaskRepository) GetChildrenByParentID(ctx context.Context, parentID uuid.UUID) ([]*models.ScheduledTask, error) {
	query := `
		SELECT st.*, t.task_name as todo_title
		FROM scheduled_tasks st
		LEFT JOIN todos t ON st.todo_id = t.id
		WHERE st.parent_schedule_id = $1
		AND st.is_cancelled = false
		ORDER BY st.scheduled_datetime ASC`

	rows, err := r.db.QueryContext(ctx, query, parentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get child schedules: %w", err)
	}
	defer rows.Close()

	var tasks []*models.ScheduledTask
	for rows.Next() {
		var task models.ScheduledTask
		var todoTitle sql.NullString

		err := rows.Scan(
			&task.ID, &task.TodoID, &task.UserID, &task.ScheduledDatetime,
			&task.DurationMinutes, &task.NotificationSent, &task.NotificationFrequency,
			&task.ReminderMinutesBefore, &task.Location, &task.MeetingURL,
			&task.Attendees, &task.IsAllDay, &task.IsCancelled,
			&task.IsRecurring, &task.RecurrencePattern, &task.RecurrenceInterval,
			&task.RecurrenceEndDate, &task.ParentScheduleID, &task.IsParentSchedule,
			&task.RecurrenceCount, &task.WeekdayMask,
			&task.CreatedAt, &task.UpdatedAt, &todoTitle,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan child schedule row: %w", err)
		}

		if todoTitle.Valid {
			task.TodoTitle = &todoTitle.String
		}

		tasks = append(tasks, &task)
	}

	return tasks, nil
}

// CreateChildInstance creates a child instance from a parent recurring schedule
func (r *ScheduledTaskRepository) CreateChildInstance(ctx context.Context, parent *models.ScheduledTask, scheduledTime time.Time) (*models.ScheduledTask, error) {
	child := &models.ScheduledTask{
		ID:                    uuid.New(),
		TodoID:                parent.TodoID,
		UserID:                parent.UserID,
		ScheduledDatetime:     scheduledTime,
		DurationMinutes:       parent.DurationMinutes,
		NotificationSent:      false,
		NotificationFrequency: parent.NotificationFrequency,
		ReminderMinutesBefore: parent.ReminderMinutesBefore,
		Location:              parent.Location,
		MeetingURL:            parent.MeetingURL,
		Attendees:             parent.Attendees,
		IsAllDay:              parent.IsAllDay,
		IsCancelled:           false,
		IsRecurring:           false, // Child instances are not recurring themselves
		RecurrencePattern:     models.RecurrencePatternNone,
		RecurrenceInterval:    1,
		ParentScheduleID:      &parent.ID,
		IsParentSchedule:      false,
		CreatedAt:             time.Now(),
		UpdatedAt:             time.Now(),
	}

	err := r.Create(ctx, child)
	if err != nil {
		return nil, fmt.Errorf("failed to create child instance: %w", err)
	}

	return child, nil
}

// ========================================
// TIME ENTRY REPOSITORY
// ========================================

// TimeEntryRepository handles time entry-related database operations
type TimeEntryRepository struct {
	db *database.DB
}

// NewTimeEntryRepository creates a new time entry repository
func NewTimeEntryRepository(db *database.DB) *TimeEntryRepository {
	return &TimeEntryRepository{db: db}
}

// Create creates a new time entry
func (r *TimeEntryRepository) Create(ctx context.Context, entry *models.TimeEntry) error {
	entry.ID = uuid.New()
	entry.IsBillable = false

	query := `
		INSERT INTO time_entries (
			id, todo_id, user_id, start_time, end_time, duration_minutes,
			description, is_billable, hourly_rate, created_at, updated_at
		) VALUES (
			:id, :todo_id, :user_id, :start_time, :end_time, :duration_minutes,
			:description, :is_billable, :hourly_rate, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
		)`

	_, err := r.db.NamedExecContext(ctx, query, entry)
	if err != nil {
		return fmt.Errorf("failed to create time entry: %w", err)
	}

	return nil
}

// GetByUserID retrieves time entries for a user
func (r *TimeEntryRepository) GetByUserID(ctx context.Context, userID uuid.UUID, fromDate, toDate *string, limit, offset int) ([]*models.TimeEntry, int, error) {
	whereClause := "te.user_id = $1"
	args := []interface{}{userID}
	argIndex := 2

	if fromDate != nil {
		whereClause += fmt.Sprintf(" AND te.start_time >= $%d", argIndex)
		args = append(args, *fromDate)
		argIndex++
	}

	if toDate != nil {
		whereClause += fmt.Sprintf(" AND te.start_time <= $%d", argIndex)
		args = append(args, *toDate)
		argIndex++
	}

	// Count query
	countQuery := fmt.Sprintf(`
		SELECT COUNT(*) FROM time_entries te 
		WHERE %s`, whereClause)

	var total int
	err := r.db.GetContext(ctx, &total, countQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count time entries: %w", err)
	}

	// Main query
	query := fmt.Sprintf(`
		SELECT te.*, t.task_name as todo_title
		FROM time_entries te
		LEFT JOIN todos t ON te.todo_id = t.id
		WHERE %s
		ORDER BY te.start_time DESC
		LIMIT $%d OFFSET $%d`, whereClause, argIndex, argIndex+1)

	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get time entries by user ID: %w", err)
	}
	defer rows.Close()

	var entries []*models.TimeEntry
	for rows.Next() {
		var entry models.TimeEntry
		var todoTitle sql.NullString

		err := rows.Scan(
			&entry.ID, &entry.TodoID, &entry.UserID, &entry.StartTime,
			&entry.EndTime, &entry.DurationMinutes, &entry.Description,
			&entry.IsBillable, &entry.HourlyRate, &entry.CreatedAt,
			&entry.UpdatedAt, &todoTitle,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan time entry row: %w", err)
		}

		if todoTitle.Valid {
			entry.TodoTitle = &todoTitle.String
		}

		entries = append(entries, &entry)
	}

	return entries, total, nil
}

// GetByID retrieves a time entry by ID
func (r *TimeEntryRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.TimeEntry, error) {
	var entry models.TimeEntry
	query := `
		SELECT te.*, t.task_name as todo_title
		FROM time_entries te
		LEFT JOIN todos t ON te.todo_id = t.id
		WHERE te.id = $1`

	var todoTitle sql.NullString
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&entry.ID, &entry.TodoID, &entry.UserID, &entry.StartTime,
		&entry.EndTime, &entry.DurationMinutes, &entry.Description,
		&entry.IsBillable, &entry.HourlyRate, &entry.CreatedAt,
		&entry.UpdatedAt, &todoTitle,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get time entry by ID: %w", err)
	}

	if todoTitle.Valid {
		entry.TodoTitle = &todoTitle.String
	}

	return &entry, nil
}

// Update updates a time entry
func (r *TimeEntryRepository) Update(ctx context.Context, id uuid.UUID, updates map[string]interface{}) error {
	if len(updates) == 0 {
		return nil
	}

	setParts := make([]string, 0, len(updates))
	args := make([]interface{}, 0, len(updates)+1)
	argIndex := 1

	for field, value := range updates {
		setParts = append(setParts, fmt.Sprintf("%s = $%d", field, argIndex))
		args = append(args, value)
		argIndex++
	}

	setParts = append(setParts, fmt.Sprintf("updated_at = $%d", argIndex))
	args = append(args, time.Now())
	argIndex++

	args = append(args, id)

	query := fmt.Sprintf(`
		UPDATE time_entries 
		SET %s 
		WHERE id = $%d`,
		strings.Join(setParts, ", "), argIndex)

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update time entry: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("time entry not found")
	}

	return nil
}

// Delete deletes a time entry
func (r *TimeEntryRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM time_entries WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete time entry: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("time entry not found")
	}

	return nil
}

// GetActiveTimeEntry retrieves the currently active time entry for a user
func (r *TimeEntryRepository) GetActiveTimeEntry(ctx context.Context, userID uuid.UUID) (*models.TimeEntry, error) {
	var entry models.TimeEntry
	query := `
		SELECT te.*, t.task_name as todo_title
		FROM time_entries te
		LEFT JOIN todos t ON te.todo_id = t.id
		WHERE te.user_id = $1 AND te.end_time IS NULL
		ORDER BY te.start_time DESC
		LIMIT 1`

	var todoTitle sql.NullString
	err := r.db.QueryRowContext(ctx, query, userID).Scan(
		&entry.ID, &entry.TodoID, &entry.UserID, &entry.StartTime,
		&entry.EndTime, &entry.DurationMinutes, &entry.Description,
		&entry.IsBillable, &entry.HourlyRate, &entry.CreatedAt,
		&entry.UpdatedAt, &todoTitle,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get active time entry: %w", err)
	}

	if todoTitle.Valid {
		entry.TodoTitle = &todoTitle.String
	}

	return &entry, nil
}
