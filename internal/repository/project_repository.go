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

// ProjectRepository handles project-related database operations
type ProjectRepository struct {
	db *database.DB
}

// NewProjectRepository creates a new project repository
func NewProjectRepository(db *database.DB) *ProjectRepository {
	return &ProjectRepository{db: db}
}

// Create creates a new project
func (r *ProjectRepository) Create(ctx context.Context, project *models.Project) error {
	project.ID = uuid.New()
	project.ProgressPercentage = 0.0
	project.IsArchived = false
	project.IsFavorite = false

	if project.ColorTheme == "" {
		project.ColorTheme = "#3498db" // Default blue
	}

	query := `
		INSERT INTO projects (
			id, user_id, title, description, image_url, category, 
			color_theme, is_archived, is_favorite, progress_percentage,
			created_at, updated_at
		) VALUES (
			:id, :user_id, :title, :description, :image_url, :category,
			:color_theme, :is_archived, :is_favorite, :progress_percentage,
			CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
		)`

	_, err := r.db.NamedExecContext(ctx, query, project)
	if err != nil {
		return fmt.Errorf("failed to create project: %w", err)
	}

	return nil
}

// GetByID retrieves a project by ID with optional todo counts
func (r *ProjectRepository) GetByID(ctx context.Context, id uuid.UUID, includeCounts bool) (*models.Project, error) {
	var project models.Project

	baseQuery := `
		SELECT p.* FROM projects p 
		WHERE p.id = $1 AND p.is_archived = false`

	err := r.db.GetContext(ctx, &project, baseQuery, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get project by ID: %w", err)
	}

	if includeCounts {
		if err := r.loadTodoCounts(ctx, &project); err != nil {
			return nil, err
		}
	}

	return &project, nil
}

// GetByUserID retrieves all projects for a user (legacy method - kept for backward compatibility)
func (r *ProjectRepository) GetByUserID(ctx context.Context, userID uuid.UUID, includeArchived bool, includeCounts bool) ([]*models.Project, error) {
	// Use the new paginated method with default values
	projects, _, err := r.GetByUserIDPaginated(ctx, userID, models.ProjectsQueryRequest{
		PaginationRequest: models.PaginationRequest{
			Page:     1,
			PageSize: 1000, // Large default for backward compatibility
			Sort:     "updated_at",
			Order:    "desc",
		},
		IncludeArchived: includeArchived,
		IncludeCounts:   includeCounts,
	})
	return projects, err
}

// GetByUserIDPaginated retrieves projects for a user with pagination and advanced filtering
func (r *ProjectRepository) GetByUserIDPaginated(ctx context.Context, userID uuid.UUID, req models.ProjectsQueryRequest) ([]*models.Project, int, error) {
	// Set defaults for pagination
	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize < 1 || req.PageSize > 100 {
		req.PageSize = 20
	}
	if req.Sort == "" {
		req.Sort = "updated_at"
	}
	if req.Order == "" {
		req.Order = "desc"
	}

	// Calculate offset
	offset := (req.Page - 1) * req.PageSize

	// Build WHERE clause conditions
	var conditions []string
	var args []interface{}
	argIndex := 1

	// Base condition: user_id
	conditions = append(conditions, fmt.Sprintf("p.user_id = $%d", argIndex))
	args = append(args, userID)
	argIndex++

	// Include archived filter
	if !req.IncludeArchived {
		conditions = append(conditions, fmt.Sprintf("p.is_archived = $%d", argIndex))
		args = append(args, false)
		argIndex++
	}

	// Category filter
	if req.Category != "" {
		conditions = append(conditions, fmt.Sprintf("p.category = $%d", argIndex))
		args = append(args, req.Category)
		argIndex++
	}

	// Favorite filter
	if req.IsFavorite != nil {
		conditions = append(conditions, fmt.Sprintf("p.is_favorite = $%d", argIndex))
		args = append(args, *req.IsFavorite)
		argIndex++
	}

	// Search filter (title, description)
	if req.Search != "" {
		searchPattern := "%" + req.Search + "%"
		conditions = append(conditions, fmt.Sprintf("(LOWER(p.title) LIKE LOWER($%d) OR LOWER(p.description) LIKE LOWER($%d))", argIndex, argIndex))
		args = append(args, searchPattern)
		argIndex++
	}

	// Build the WHERE clause
	whereClause := "WHERE " + strings.Join(conditions, " AND ")

	// Validate sort field to prevent SQL injection
	validSortFields := map[string]string{
		"title":               "p.title",
		"created_at":          "p.created_at",
		"updated_at":          "p.updated_at",
		"category":            "p.category",
		"is_favorite":         "p.is_favorite",
		"progress_percentage": "p.progress_percentage",
	}

	sortField, exists := validSortFields[req.Sort]
	if !exists {
		sortField = "p.updated_at"
	}

	// Validate order direction
	if req.Order != "asc" && req.Order != "desc" {
		req.Order = "desc"
	}

	// Count query for total records
	countQuery := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM projects p
		%s`, whereClause)

	var totalCount int
	err := r.db.GetContext(ctx, &totalCount, countQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count projects: %w", err)
	}

	// Main query with optimized structure
	query := fmt.Sprintf(`
		SELECT 
			p.id,
			p.user_id,
			p.title,
			p.description,
			p.color_theme,
			p.image_url,
			p.is_archived,
			p.is_favorite,
			p.progress_percentage,
			p.created_at,
			p.updated_at
		FROM projects p
		%s
		ORDER BY p.is_favorite DESC, %s %s
		LIMIT $%d OFFSET $%d`,
		whereClause,
		sortField,
		strings.ToUpper(req.Order),
		argIndex,
		argIndex+1)

	// Add LIMIT and OFFSET parameters
	args = append(args, req.PageSize, offset)

	var projects []*models.Project
	err = r.db.SelectContext(ctx, &projects, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get paginated projects: %w", err)
	}

	// Load todo counts if requested
	if req.IncludeCounts {
		// Load todo counts individually for now (can be optimized to batch later)
		for _, project := range projects {
			if err := r.loadTodoCounts(ctx, project); err != nil {
				return nil, 0, fmt.Errorf("failed to load todo counts: %w", err)
			}
		}
	}

	return projects, totalCount, nil
}

// Update updates a project
func (r *ProjectRepository) Update(ctx context.Context, id uuid.UUID, updates map[string]interface{}) error {
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
		UPDATE projects 
		SET %s 
		WHERE id = $%d AND is_archived = false`,
		strings.Join(setParts, ", "), argIndex)

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update project: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("project not found or already archived")
	}

	return nil
}

// Archive archives a project (soft delete)
func (r *ProjectRepository) Archive(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE projects 
		SET is_archived = true, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1 AND is_archived = false`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to archive project: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("project not found or already archived")
	}

	return nil
}

// Restore restores an archived project
func (r *ProjectRepository) Restore(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE projects 
		SET is_archived = false, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1 AND is_archived = true`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to restore project: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("project not found or not archived")
	}

	return nil
}

// Delete permanently deletes a project and all related data
func (r *ProjectRepository) Delete(ctx context.Context, id uuid.UUID) error {
	// Start transaction
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Delete project (cascade will handle related data)
	query := `DELETE FROM projects WHERE id = $1`
	result, err := tx.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete project: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("project not found")
	}

	return tx.Commit()
}

// GetProjectStats retrieves comprehensive project statistics
func (r *ProjectRepository) GetProjectStats(ctx context.Context, userID uuid.UUID, includeArchived bool) ([]*models.ProjectStats, error) {
	query := `
		SELECT 
			p.id,
			p.title,
			p.category,
			p.progress_percentage,
			COUNT(t.id) as total_todos,
			COUNT(CASE WHEN t.status = 'completed' THEN 1 END) as completed_todos,
			COUNT(CASE WHEN t.status = 'in_progress' THEN 1 END) as in_progress_todos,
			COUNT(CASE WHEN t.due_date < CURRENT_TIMESTAMP AND t.status != 'completed' THEN 1 END) as overdue_todos,
			AVG(t.actual_time) as avg_task_duration,
			SUM(t.estimated_time) as total_estimated_time,
			SUM(t.actual_time) as total_actual_time
		FROM projects p
		LEFT JOIN todos t ON p.id = t.project_id AND t.is_archived = false
		WHERE p.user_id = $1`

	args := []interface{}{userID}

	if !includeArchived {
		query += ` AND p.is_archived = false`
	}

	query += ` GROUP BY p.id, p.title, p.category, p.progress_percentage ORDER BY p.updated_at DESC`

	var stats []*models.ProjectStats
	err := r.db.SelectContext(ctx, &stats, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get project stats: %w", err)
	}

	return stats, nil
}

// GetProjectsByCategory retrieves projects grouped by category
func (r *ProjectRepository) GetProjectsByCategory(ctx context.Context, userID uuid.UUID, category models.ProjectCategory) ([]*models.Project, error) {
	query := `
		SELECT p.* FROM projects p 
		WHERE p.user_id = $1 AND p.category = $2 AND p.is_archived = false
		ORDER BY p.is_favorite DESC, p.updated_at DESC`

	var projects []*models.Project
	err := r.db.SelectContext(ctx, &projects, query, userID, category)
	if err != nil {
		return nil, fmt.Errorf("failed to get projects by category: %w", err)
	}

	return projects, nil
}

// GetFavoriteProjects retrieves favorite projects for a user
func (r *ProjectRepository) GetFavoriteProjects(ctx context.Context, userID uuid.UUID) ([]*models.Project, error) {
	query := `
		SELECT p.* FROM projects p 
		WHERE p.user_id = $1 AND p.is_favorite = true AND p.is_archived = false
		ORDER BY p.updated_at DESC`

	var projects []*models.Project
	err := r.db.SelectContext(ctx, &projects, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get favorite projects: %w", err)
	}

	return projects, nil
}

// UpdateProgress updates project progress percentage
func (r *ProjectRepository) UpdateProgress(ctx context.Context, projectID uuid.UUID) error {
	query := `
		UPDATE projects 
		SET progress_percentage = calculate_project_progress($1),
			updated_at = CURRENT_TIMESTAMP
		WHERE id = $1`

	_, err := r.db.ExecContext(ctx, query, projectID)
	if err != nil {
		return fmt.Errorf("failed to update project progress: %w", err)
	}

	return nil
}

// SearchProjects searches projects by title or description
func (r *ProjectRepository) SearchProjects(ctx context.Context, userID uuid.UUID, searchTerm string, limit int, offset int) ([]*models.Project, int, error) {
	searchPattern := "%" + strings.ToLower(searchTerm) + "%"

	// Count total results
	countQuery := `
		SELECT COUNT(*) FROM projects p 
		WHERE p.user_id = $1 AND p.is_archived = false
		AND (LOWER(p.title) LIKE $2 OR LOWER(p.description) LIKE $2)`

	var total int
	err := r.db.GetContext(ctx, &total, countQuery, userID, searchPattern)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count search results: %w", err)
	}

	// Get paginated results
	query := `
		SELECT p.* FROM projects p 
		WHERE p.user_id = $1 AND p.is_archived = false
		AND (LOWER(p.title) LIKE $2 OR LOWER(p.description) LIKE $2)
		ORDER BY p.is_favorite DESC, p.updated_at DESC
		LIMIT $3 OFFSET $4`

	var projects []*models.Project
	err = r.db.SelectContext(ctx, &projects, query, userID, searchPattern, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to search projects: %w", err)
	}

	return projects, total, nil
}

// Helper method to load todo counts for a project
func (r *ProjectRepository) loadTodoCounts(ctx context.Context, project *models.Project) error {
	query := `
		SELECT 
			COUNT(*) as total_todos,
			COUNT(CASE WHEN status = 'completed' THEN 1 END) as completed_todos
		FROM todos 
		WHERE project_id = $1 AND is_archived = false`

	var counts struct {
		TotalTodos     int `db:"total_todos"`
		CompletedTodos int `db:"completed_todos"`
	}

	err := r.db.GetContext(ctx, &counts, query, project.ID)
	if err != nil {
		return fmt.Errorf("failed to load todo counts: %w", err)
	}

	project.TodoCount = &counts.TotalTodos
	project.CompletedCount = &counts.CompletedTodos

	return nil
}

// ValidateProjectOwnership verifies that a user owns a project
func (r *ProjectRepository) ValidateProjectOwnership(ctx context.Context, projectID, userID uuid.UUID) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM projects WHERE id = $1 AND user_id = $2 AND is_archived = false)`

	var exists bool
	err := r.db.GetContext(ctx, &exists, query, projectID, userID)
	if err != nil {
		return false, fmt.Errorf("failed to validate project ownership: %w", err)
	}

	return exists, nil
}

// GetProjectDashboard retrieves project information for dashboard display
func (r *ProjectRepository) GetProjectDashboard(ctx context.Context, userID uuid.UUID) ([]*models.Project, error) {
	query := `
		SELECT 
			p.*,
			COUNT(t.id) as todo_count,
			COUNT(CASE WHEN t.status = 'completed' THEN 1 END) as completed_count
		FROM projects p
		LEFT JOIN todos t ON p.id = t.project_id AND t.is_archived = false
		WHERE p.user_id = $1 AND p.is_archived = false
		GROUP BY p.id
		ORDER BY p.is_favorite DESC, p.updated_at DESC`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get project dashboard: %w", err)
	}
	defer rows.Close()

	var projects []*models.Project
	for rows.Next() {
		var project models.Project
		var todoCount, completedCount sql.NullInt64

		err := rows.Scan(
			&project.ID, &project.UserID, &project.Title, &project.Description,
			&project.ImageURL, &project.Category, &project.ColorTheme,
			&project.IsArchived, &project.IsFavorite, &project.ProgressPercentage,
			&project.CreatedAt, &project.UpdatedAt, &todoCount, &completedCount,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan project row: %w", err)
		}

		if todoCount.Valid {
			count := int(todoCount.Int64)
			project.TodoCount = &count
		}
		if completedCount.Valid {
			count := int(completedCount.Int64)
			project.CompletedCount = &count
		}

		projects = append(projects, &project)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating project rows: %w", err)
	}

	return projects, nil
}
