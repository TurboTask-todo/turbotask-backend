package service

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"strings"
	"time"

	"macwrite-auth-api/internal/models"
	"macwrite-auth-api/internal/repository"
	"macwrite-auth-api/pkg/redis"

	"github.com/google/uuid"
)

// ProjectService handles project business logic
type ProjectService struct {
	projectRepo *repository.ProjectRepository
	todoRepo    *repository.TodoRepository
	redisClient redis.Client
}

// NewProjectService creates a new project service
func NewProjectService(
	projectRepo *repository.ProjectRepository,
	todoRepo *repository.TodoRepository,
	redisClient redis.Client,
) *ProjectService {
	return &ProjectService{
		projectRepo: projectRepo,
		todoRepo:    todoRepo,
		redisClient: redisClient,
	}
}

// Cache key constants
const (
	projectCachePrefix      = "project:"
	userProjectsCachePrefix = "user_projects:"
	projectStatsCachePrefix = "project_stats:"
	projectDashboardPrefix  = "project_dashboard:"
	cacheTTL                = 30 * time.Minute
	dashboardCacheTTL       = 5 * time.Minute
)

// CreateProject creates a new project
func (s *ProjectService) CreateProject(ctx context.Context, userID uuid.UUID, req *models.CreateProjectRequest) (*models.Project, error) {
	// Validate request
	if err := s.validateCreateProjectRequest(req); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	project := &models.Project{
		UserID:      userID,
		Title:       req.Title,
		Description: req.Description,
		ImageURL:    req.ImageURL,
		Category:    req.Category,
	}

	if req.ColorTheme != nil {
		project.ColorTheme = *req.ColorTheme
	}

	// Create project in database
	if err := s.projectRepo.Create(ctx, project); err != nil {
		return nil, fmt.Errorf("failed to create project: %w", err)
	}

	// Invalidate user projects cache
	s.invalidateUserProjectsCache(userID)

	return project, nil
}

// GetProject retrieves a project by ID
func (s *ProjectService) GetProject(ctx context.Context, projectID, userID uuid.UUID, includeCounts bool) (*models.Project, error) {
	// Check cache first
	cacheKey := fmt.Sprintf("%s%s", projectCachePrefix, projectID.String())
	if cached, err := s.redisClient.Get(ctx, cacheKey); err == nil {
		var project models.Project
		if err := json.Unmarshal(cached, &project); err == nil {
			// Verify ownership
			if project.UserID != userID {
				return nil, fmt.Errorf("project not found")
			}
			return &project, nil
		}
	}

	// Get from database
	project, err := s.projectRepo.GetByID(ctx, projectID, includeCounts)
	if err != nil {
		return nil, fmt.Errorf("failed to get project: %w", err)
	}
	if project == nil {
		return nil, fmt.Errorf("project not found")
	}

	// Verify ownership
	if project.UserID != userID {
		return nil, fmt.Errorf("project not found")
	}

	// Cache the result
	if projectData, err := json.Marshal(project); err == nil {
		s.redisClient.Set(ctx, cacheKey, projectData, cacheTTL)
	}

	return project, nil
}

// GetUserProjects retrieves all projects for a user
func (s *ProjectService) GetUserProjects(ctx context.Context, userID uuid.UUID, includeArchived, includeCounts bool) ([]*models.Project, error) {
	// Check cache first
	cacheKey := fmt.Sprintf("%s%s:archived_%t:counts_%t", userProjectsCachePrefix, userID.String(), includeArchived, includeCounts)
	if cached, err := s.redisClient.Get(ctx, cacheKey); err == nil {
		var projects []*models.Project
		if err := json.Unmarshal(cached, &projects); err == nil {
			return projects, nil
		}
	}

	// Get from database
	projects, err := s.projectRepo.GetByUserID(ctx, userID, includeArchived, includeCounts)
	if err != nil {
		return nil, fmt.Errorf("failed to get user projects: %w", err)
	}

	// Cache the result
	if projectsData, err := json.Marshal(projects); err == nil {
		s.redisClient.Set(ctx, cacheKey, projectsData, cacheTTL)
	}

	return projects, nil
}

// GetProjectsPaginated retrieves projects for a user with pagination
func (s *ProjectService) GetProjectsPaginated(ctx context.Context, userID uuid.UUID, req models.ProjectsQueryRequest) ([]*models.Project, int, error) {
	// Validate pagination parameters
	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize < 1 {
		req.PageSize = 20
	}
	if req.PageSize > 100 {
		req.PageSize = 100
	}

	// Generate cache key
	cacheKey := s.generatePaginatedCacheKey(userID, req)

	// Check cache first
	if cached, err := s.redisClient.Get(ctx, cacheKey); err == nil {
		type cachedData struct {
			Projects   []*models.Project `json:"projects"`
			TotalCount int               `json:"total_count"`
		}

		var data cachedData
		if err := json.Unmarshal(cached, &data); err == nil {
			return data.Projects, data.TotalCount, nil
		}
	}

	// Get from repository
	projects, totalCount, err := s.projectRepo.GetByUserIDPaginated(ctx, userID, req)
	if err != nil {
		return nil, 0, err
	}

	// Cache the result
	if projectsData, err := json.Marshal(map[string]interface{}{
		"projects":    projects,
		"total_count": totalCount,
	}); err == nil {
		s.redisClient.Set(ctx, cacheKey, projectsData, cacheTTL)
	}

	return projects, totalCount, nil
}

func (s *ProjectService) generatePaginatedCacheKey(userID uuid.UUID, req models.ProjectsQueryRequest) string {
	favStr := "nil"
	if req.IsFavorite != nil {
		favStr = fmt.Sprintf("%t", *req.IsFavorite)
	}

	return fmt.Sprintf("%s%s:page_%d:size_%d:archived_%t:counts_%t:cat_%s:fav_%s:search_%s:sort_%s:order_%s",
		projectCachePrefix,
		userID.String(),
		req.Page,
		req.PageSize,
		req.IncludeArchived,
		req.IncludeCounts,
		req.Category,
		favStr,
		req.Search,
		req.Sort,
		req.Order,
	)
}

// UpdateProject updates a project
func (s *ProjectService) UpdateProject(ctx context.Context, projectID, userID uuid.UUID, req *models.UpdateProjectRequest) (*models.Project, error) {
	// Verify ownership
	exists, err := s.projectRepo.ValidateProjectOwnership(ctx, projectID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to validate ownership: %w", err)
	}
	if !exists {
		return nil, fmt.Errorf("project not found")
	}

	// Validate request
	if err := s.validateUpdateProjectRequest(req); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Build updates map
	updates := make(map[string]interface{})
	if req.Title != nil {
		updates["title"] = *req.Title
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.ImageURL != nil {
		updates["image_url"] = *req.ImageURL
	}
	if req.Category != nil {
		updates["category"] = *req.Category
	}
	if req.ColorTheme != nil {
		updates["color_theme"] = *req.ColorTheme
	}
	if req.IsFavorite != nil {
		updates["is_favorite"] = *req.IsFavorite
	}

	// Update in database
	if err := s.projectRepo.Update(ctx, projectID, updates); err != nil {
		return nil, fmt.Errorf("failed to update project: %w", err)
	}

	// Invalidate caches
	s.invalidateProjectCaches(projectID, userID)

	// Return updated project
	return s.GetProject(ctx, projectID, userID, true)
}

// ArchiveProject archives a project
func (s *ProjectService) ArchiveProject(ctx context.Context, projectID, userID uuid.UUID) error {
	// Verify ownership
	exists, err := s.projectRepo.ValidateProjectOwnership(ctx, projectID, userID)
	if err != nil {
		return fmt.Errorf("failed to validate ownership: %w", err)
	}
	if !exists {
		return fmt.Errorf("project not found")
	}

	// Archive in database
	if err := s.projectRepo.Archive(ctx, projectID); err != nil {
		return fmt.Errorf("failed to archive project: %w", err)
	}

	// Invalidate caches
	s.invalidateProjectCaches(projectID, userID)

	return nil
}

// RestoreProject restores an archived project
func (s *ProjectService) RestoreProject(ctx context.Context, projectID, userID uuid.UUID) error {
	// Verify ownership (need to check even archived projects)
	// For this, we'll need to modify the validation to include archived projects
	project, err := s.projectRepo.GetByID(ctx, projectID, false)
	if err != nil {
		return fmt.Errorf("failed to get project: %w", err)
	}
	if project == nil || project.UserID != userID {
		return fmt.Errorf("project not found")
	}

	// Restore in database
	if err := s.projectRepo.Restore(ctx, projectID); err != nil {
		return fmt.Errorf("failed to restore project: %w", err)
	}

	// Invalidate caches
	s.invalidateProjectCaches(projectID, userID)

	return nil
}

// DeleteProject permanently deletes a project
func (s *ProjectService) DeleteProject(ctx context.Context, projectID, userID uuid.UUID) error {
	// Verify ownership
	exists, err := s.projectRepo.ValidateProjectOwnership(ctx, projectID, userID)
	if err != nil {
		return fmt.Errorf("failed to validate ownership: %w", err)
	}
	if !exists {
		return fmt.Errorf("project not found")
	}

	// Delete in database
	if err := s.projectRepo.Delete(ctx, projectID); err != nil {
		return fmt.Errorf("failed to delete project: %w", err)
	}

	// Invalidate caches
	s.invalidateProjectCaches(projectID, userID)

	return nil
}

// GetProjectStats retrieves project statistics
func (s *ProjectService) GetProjectStats(ctx context.Context, userID uuid.UUID, includeArchived bool) ([]*models.ProjectStats, error) {
	// Check cache first
	cacheKey := fmt.Sprintf("%s%s:archived_%t", projectStatsCachePrefix, userID.String(), includeArchived)
	if cached, err := s.redisClient.Get(ctx, cacheKey); err == nil {
		var stats []*models.ProjectStats
		if err := json.Unmarshal(cached, &stats); err == nil {
			return stats, nil
		}
	}

	// Get from database
	stats, err := s.projectRepo.GetProjectStats(ctx, userID, includeArchived)
	if err != nil {
		return nil, fmt.Errorf("failed to get project stats: %w", err)
	}

	// Cache the result
	if statsData, err := json.Marshal(stats); err == nil {
		s.redisClient.Set(ctx, cacheKey, statsData, cacheTTL)
	}

	return stats, nil
}

// GetProjectsByCategory retrieves projects by category
func (s *ProjectService) GetProjectsByCategory(ctx context.Context, userID uuid.UUID, category models.ProjectCategory) ([]*models.Project, error) {
	projects, err := s.projectRepo.GetProjectsByCategory(ctx, userID, category)
	if err != nil {
		return nil, fmt.Errorf("failed to get projects by category: %w", err)
	}

	return projects, nil
}

// GetFavoriteProjects retrieves favorite projects
func (s *ProjectService) GetFavoriteProjects(ctx context.Context, userID uuid.UUID) ([]*models.Project, error) {
	projects, err := s.projectRepo.GetFavoriteProjects(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get favorite projects: %w", err)
	}

	return projects, nil
}

// SearchProjects searches projects by text
func (s *ProjectService) SearchProjects(ctx context.Context, userID uuid.UUID, searchTerm string, limit, offset int) ([]*models.Project, int, error) {
	if searchTerm == "" {
		projects, err := s.GetUserProjects(ctx, userID, false, true)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to get user projects: %w", err)
		}
		return projects, 0, nil
	}

	projects, total, err := s.projectRepo.SearchProjects(ctx, userID, searchTerm, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to search projects: %w", err)
	}

	return projects, total, nil
}

// GetProjectDashboard retrieves project dashboard data
func (s *ProjectService) GetProjectDashboard(ctx context.Context, userID uuid.UUID) ([]*models.Project, error) {
	// Check cache first
	cacheKey := fmt.Sprintf("%s%s", projectDashboardPrefix, userID.String())
	if cached, err := s.redisClient.Get(ctx, cacheKey); err == nil {
		var projects []*models.Project
		if err := json.Unmarshal(cached, &projects); err == nil {
			return projects, nil
		}
	}

	// Get from database
	projects, err := s.projectRepo.GetProjectDashboard(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get project dashboard: %w", err)
	}

	// Cache the result with shorter TTL for dashboard
	if projectsData, err := json.Marshal(projects); err == nil {
		s.redisClient.Set(ctx, cacheKey, projectsData, dashboardCacheTTL)
	}

	return projects, nil
}

// UpdateProjectProgress updates project progress and invalidates cache
func (s *ProjectService) UpdateProjectProgress(ctx context.Context, projectID uuid.UUID) error {
	if err := s.projectRepo.UpdateProgress(ctx, projectID); err != nil {
		return fmt.Errorf("failed to update project progress: %w", err)
	}

	// Get project to get user ID for cache invalidation
	project, err := s.projectRepo.GetByID(ctx, projectID, false)
	if err != nil {
		return fmt.Errorf("failed to get project for cache invalidation: %w", err)
	}
	if project != nil {
		s.invalidateProjectCaches(projectID, project.UserID)
	}

	return nil
}

// Validation methods

func (s *ProjectService) validateCreateProjectRequest(req *models.CreateProjectRequest) error {
	if req.Title == "" {
		return fmt.Errorf("title is required")
	}
	if len(req.Title) > 255 {
		return fmt.Errorf("title must be less than 255 characters")
	}
	if !req.Category.IsValid() {
		return fmt.Errorf("invalid project category")
	}
	return nil
}

func (s *ProjectService) validateUpdateProjectRequest(req *models.UpdateProjectRequest) error {
	if req.Title != nil && *req.Title == "" {
		return fmt.Errorf("title cannot be empty")
	}
	if req.Title != nil && len(*req.Title) > 255 {
		return fmt.Errorf("title must be less than 255 characters")
	}
	if req.Category != nil && !req.Category.IsValid() {
		return fmt.Errorf("invalid project category")
	}
	return nil
}

// Cache invalidation methods

func (s *ProjectService) invalidateProjectCaches(projectID, userID uuid.UUID) {
	ctx := context.Background()

	// Invalidate specific project cache
	projectCacheKey := fmt.Sprintf("%s%s", projectCachePrefix, projectID.String())
	s.redisClient.Delete(ctx, projectCacheKey)

	// Invalidate user projects cache (all variations)
	userPattern := fmt.Sprintf("%s%s:*", userProjectsCachePrefix, userID.String())
	s.redisClient.DeletePattern(ctx, userPattern)

	// Invalidate project stats cache
	statsPattern := fmt.Sprintf("%s%s:*", projectStatsCachePrefix, userID.String())
	s.redisClient.DeletePattern(ctx, statsPattern)

	// Invalidate dashboard cache
	dashboardKey := fmt.Sprintf("%s%s", projectDashboardPrefix, userID.String())
	s.redisClient.Delete(ctx, dashboardKey)
}

func (s *ProjectService) invalidateUserProjectsCache(userID uuid.UUID) {
	ctx := context.Background()

	// Invalidate user projects cache (all variations)
	userPattern := fmt.Sprintf("%s%s:*", userProjectsCachePrefix, userID.String())
	s.redisClient.DeletePattern(ctx, userPattern)

	// Invalidate project stats cache
	statsPattern := fmt.Sprintf("%s%s:*", projectStatsCachePrefix, userID.String())
	s.redisClient.DeletePattern(ctx, statsPattern)

	// Invalidate dashboard cache
	dashboardKey := fmt.Sprintf("%s%s", projectDashboardPrefix, userID.String())
	s.redisClient.Delete(ctx, dashboardKey)
}

// Helper methods

// GetProjectTitle retrieves just the project title (lightweight operation)
func (s *ProjectService) GetProjectTitle(ctx context.Context, projectID uuid.UUID) (string, error) {
	project, err := s.projectRepo.GetByID(ctx, projectID, false)
	if err != nil {
		return "", err
	}
	if project == nil {
		return "", fmt.Errorf("project not found")
	}
	return project.Title, nil
}

// ExportProject exports all tasks in a project as a ZIP file containing CSV data
func (s *ProjectService) ExportProject(ctx context.Context, userID, projectID uuid.UUID) ([]byte, string, error) {
	// Verify project ownership
	project, err := s.projectRepo.GetByID(ctx, projectID, false)
	if err != nil {
		return nil, "", err
	}
	if project == nil {
		return nil, "", fmt.Errorf("project not found")
	}
	if project.UserID != userID {
		return nil, "", fmt.Errorf("access denied: user does not own this project")
	}

	// Get all tasks for the project
	todos, err := s.todoRepo.GetByProjectID(ctx, projectID, true) // Include completed tasks
	if err != nil {
		return nil, "", fmt.Errorf("failed to retrieve project tasks: %w", err)
	}

	// Convert todos to CSV records
	csvRecords := make([]*models.CSVTaskRecord, 0, len(todos))
	for _, todo := range todos {
		record := models.CSVTaskRecordFromTodo(todo)
		csvRecords = append(csvRecords, record)
	}

	// Create export metadata
	metadata := &models.ExportMetadata{
		ProjectID:     projectID,
		ProjectTitle:  project.Title,
		ExportedAt:    time.Now(),
		ExportedBy:    userID,
		TotalTasks:    len(todos),
		ExportVersion: "1.0",
	}

	// Generate ZIP file
	zipData, err := s.createExportZIP(csvRecords, metadata)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create export ZIP: %w", err)
	}

	// Generate filename
	timestamp := time.Now().Format("20060102_150405")
	fileName := fmt.Sprintf("%s_export_%s.zip", project.Title, timestamp)

	return zipData, fileName, nil
}

// ImportProject imports tasks from a ZIP file into a project
func (s *ProjectService) ImportProject(ctx context.Context, userID, projectID uuid.UUID, file multipart.File, fileSize int64) (*models.ImportResult, error) {
	// Verify project ownership
	project, err := s.projectRepo.GetByID(ctx, projectID, false)
	if err != nil {
		return nil, err
	}
	if project == nil {
		return nil, fmt.Errorf("project not found")
	}
	if project.UserID != userID {
		return nil, fmt.Errorf("access denied: user does not own this project")
	}

	// Parse ZIP file and extract CSV records
	csvRecords, _, err := s.parseImportZIP(file, fileSize)
	if err != nil {
		return nil, fmt.Errorf("failed to parse import file: %w", err)
	}

	// Initialize import statistics
	stats := &models.ImportStatistics{
		TotalRecords:      len(csvRecords),
		CategoryBreakdown: make(map[string]int),
	}

	var errors []string
	var warnings []string

	// Process each CSV record
	for i, record := range csvRecords {
		// Convert CSV record to Todo
		todo, err := record.ToTodo(projectID, userID)
		if err != nil {
			stats.FailedImports++
			errors = append(errors, fmt.Sprintf("Row %d: Failed to convert record: %v", i+2, err))
			continue
		}

		// Check for duplicates (by task name)
		existing, err := s.todoRepo.GetByTaskNameAndProject(ctx, todo.TaskName, projectID)
		if err == nil && existing != nil {
			stats.DuplicatesFound++
			stats.SkippedRecords++
			warnings = append(warnings, fmt.Sprintf("Row %d: Task '%s' already exists, skipping", i+2, todo.TaskName))
			continue
		}

		// Create the todo
		err = s.todoRepo.Create(ctx, todo)
		if err != nil {
			stats.FailedImports++
			errors = append(errors, fmt.Sprintf("Row %d: Failed to create task '%s': %v", i+2, todo.TaskName, err))
			continue
		}

		stats.SuccessfulImports++

		// Update category breakdown
		if todo.Priority != "" {
			stats.CategoryBreakdown[string(todo.Priority)]++
		}
	}

	// Invalidate project caches
	s.invalidateProjectCaches(projectID, userID)

	// Create import result
	result := &models.ImportResult{
		Success:    stats.SuccessfulImports > 0,
		Statistics: *stats,
		Errors:     errors,
		Warnings:   warnings,
	}

	if stats.SuccessfulImports > 0 {
		result.Message = fmt.Sprintf("Successfully imported %d out of %d tasks", stats.SuccessfulImports, stats.TotalRecords)
	} else {
		result.Message = "No tasks were imported"
	}

	return result, nil
}

// createExportZIP creates a ZIP file containing CSV data and metadata
func (s *ProjectService) createExportZIP(csvRecords []*models.CSVTaskRecord, metadata *models.ExportMetadata) ([]byte, error) {
	buf := new(bytes.Buffer)
	zipWriter := zip.NewWriter(buf)

	// Create project folder inside ZIP
	folderName := fmt.Sprintf("%s_export/", metadata.ProjectTitle)

	// Add CSV file
	csvFileName := folderName + "tasks.csv"
	csvFile, err := zipWriter.Create(csvFileName)
	if err != nil {
		return nil, fmt.Errorf("failed to create CSV file in ZIP: %w", err)
	}

	// Write CSV data
	csvWriter := csv.NewWriter(csvFile)

	// Write header
	header := []string{
		"Task Name", "Description", "Status", "Priority", "Due Date", "Start Date",
		"Completed At", "Tags", "Estimated Time", "Actual Time", "Assigned To",
		"Location", "Context", "Difficulty Rating", "Energy Level Required",
		"Completion Percentage", "Is Recurring", "Is Pinned", "Created At", "Updated At",
	}
	if err := csvWriter.Write(header); err != nil {
		return nil, fmt.Errorf("failed to write CSV header: %w", err)
	}

	// Write data rows
	for _, record := range csvRecords {
		row := []string{
			record.TaskName, record.TaskDescription, record.Status, record.Priority,
			record.DueDate, record.StartDate, record.CompletedAt, record.Tags,
			record.EstimatedTime, record.ActualTime, record.AssignedTo, record.Location,
			record.Context, record.DifficultyRating, record.EnergyLevelRequired,
			record.CompletionPercentage, record.IsRecurring, record.IsPinned,
			record.CreatedAt, record.UpdatedAt,
		}
		if err := csvWriter.Write(row); err != nil {
			return nil, fmt.Errorf("failed to write CSV row: %w", err)
		}
	}
	csvWriter.Flush()

	// Add metadata file
	metadataFile, err := zipWriter.Create(folderName + "export_metadata.json")
	if err != nil {
		return nil, fmt.Errorf("failed to create metadata file in ZIP: %w", err)
	}

	metadataJSON, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metadata: %w", err)
	}

	if _, err := metadataFile.Write(metadataJSON); err != nil {
		return nil, fmt.Errorf("failed to write metadata: %w", err)
	}

	// Close ZIP writer
	if err := zipWriter.Close(); err != nil {
		return nil, fmt.Errorf("failed to close ZIP writer: %w", err)
	}

	return buf.Bytes(), nil
}

// parseImportZIP parses a ZIP file and extracts CSV records and metadata
func (s *ProjectService) parseImportZIP(file multipart.File, fileSize int64) ([]*models.CSVTaskRecord, *models.ExportMetadata, error) {
	// Read file content
	fileContent, err := io.ReadAll(file)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read file content: %w", err)
	}

	// Create ZIP reader
	reader := bytes.NewReader(fileContent)
	zipReader, err := zip.NewReader(reader, fileSize)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create ZIP reader: %w", err)
	}

	var csvRecords []*models.CSVTaskRecord
	var metadata *models.ExportMetadata

	// Iterate through files in ZIP
	for _, zipFile := range zipReader.File {
		fileName := strings.ToLower(zipFile.Name)

		// Check if it's a CSV file
		if strings.HasSuffix(fileName, "tasks.csv") || strings.HasSuffix(fileName, ".csv") {
			records, err := s.parseCSVFromZipFile(zipFile)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to parse CSV file %s: %w", zipFile.Name, err)
			}
			csvRecords = records
		}

		// Check if it's a metadata file
		if strings.HasSuffix(fileName, "export_metadata.json") || strings.HasSuffix(fileName, "metadata.json") {
			meta, err := s.parseMetadataFromZipFile(zipFile)
			if err != nil {
				// Metadata is optional, so just log the error
				continue
			}
			metadata = meta
		}
	}

	if csvRecords == nil {
		return nil, nil, fmt.Errorf("no valid CSV file found in ZIP archive")
	}

	return csvRecords, metadata, nil
}

// parseCSVFromZipFile parses CSV data from a ZIP file entry
func (s *ProjectService) parseCSVFromZipFile(zipFile *zip.File) ([]*models.CSVTaskRecord, error) {
	rc, err := zipFile.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open CSV file: %w", err)
	}
	defer rc.Close()

	csvReader := csv.NewReader(rc)

	// Read all records
	records, err := csvReader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV records: %w", err)
	}

	if len(records) < 2 { // Header + at least one data row
		return nil, fmt.Errorf("CSV file must contain at least a header and one data row")
	}

	// Parse header to create field mapping
	header := records[0]
	fieldMap := make(map[string]int)
	for i, field := range header {
		fieldMap[strings.TrimSpace(field)] = i
	}

	// Validate required fields
	requiredFields := []string{"Task Name"}
	for _, field := range requiredFields {
		if _, exists := fieldMap[field]; !exists {
			return nil, fmt.Errorf("missing required field: %s", field)
		}
	}

	// Parse data rows
	var csvRecords []*models.CSVTaskRecord
	for i := 1; i < len(records); i++ {
		row := records[i]
		record := &models.CSVTaskRecord{}

		// Map fields safely
		if idx, exists := fieldMap["Task Name"]; exists && idx < len(row) {
			record.TaskName = strings.TrimSpace(row[idx])
		}
		if idx, exists := fieldMap["Description"]; exists && idx < len(row) {
			record.TaskDescription = strings.TrimSpace(row[idx])
		}
		if idx, exists := fieldMap["Status"]; exists && idx < len(row) {
			record.Status = strings.TrimSpace(row[idx])
		}
		if idx, exists := fieldMap["Priority"]; exists && idx < len(row) {
			record.Priority = strings.TrimSpace(row[idx])
		}
		// Map other fields...

		// Skip empty rows
		if record.TaskName == "" {
			continue
		}

		csvRecords = append(csvRecords, record)
	}

	return csvRecords, nil
}

// parseMetadataFromZipFile parses metadata from a ZIP file entry
func (s *ProjectService) parseMetadataFromZipFile(zipFile *zip.File) (*models.ExportMetadata, error) {
	rc, err := zipFile.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open metadata file: %w", err)
	}
	defer rc.Close()

	data, err := io.ReadAll(rc)
	if err != nil {
		return nil, fmt.Errorf("failed to read metadata file: %w", err)
	}

	var metadata models.ExportMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	return &metadata, nil
}
