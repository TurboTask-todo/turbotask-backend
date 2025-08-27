package service

import (
	"context"
	"log"
	"sort"
	"strings"
	"time"

	"macwrite-auth-api/internal/models"

	"github.com/google/uuid"
)

// GlobalSearchService handles unified search across projects and tasks
type GlobalSearchService struct {
	projectService *ProjectService
	todoService    *TodoService
}

// NewGlobalSearchService creates a new global search service
func NewGlobalSearchService(projectService *ProjectService, todoService *TodoService) *GlobalSearchService {
	return &GlobalSearchService{
		projectService: projectService,
		todoService:    todoService,
	}
}

// GlobalSearch performs a unified search across projects and tasks
func (s *GlobalSearchService) GlobalSearch(ctx context.Context, userID uuid.UUID, req models.GlobalSearchRequest) (*models.GlobalSearchResponse, error) {
	startTime := time.Now()

	// Set default values
	if req.Limit <= 0 {
		req.Limit = 20
	}
	if req.Limit > 100 {
		req.Limit = 100 // Max limit
	}

	// Determine which types to search
	searchProjects := len(req.Types) == 0 || contains(req.Types, models.SearchResultTypeProject)
	searchTasks := len(req.Types) == 0 || contains(req.Types, models.SearchResultTypeTask)

	var allResults []models.GlobalSearchResult
	var totalCount int64
	typeCounts := make(map[string]int64)

	// Search projects
	if searchProjects {
		projectResults, projectTotal, err := s.searchProjects(ctx, userID, req)
		if err != nil {
			log.Printf("Error searching projects: %v", err)
		} else {
			allResults = append(allResults, projectResults...)
			typeCounts["project"] = projectTotal
		}
	}

	// Search tasks
	if searchTasks {
		taskResults, taskTotal, err := s.searchTasks(ctx, userID, req)
		if err != nil {
			log.Printf("Error searching tasks: %v", err)
		} else {
			allResults = append(allResults, taskResults...)
			typeCounts["task"] = taskTotal
		}
	}

	// Sort results by relevance score
	sort.Slice(allResults, func(i, j int) bool {
		return allResults[i].Score > allResults[j].Score
	})

	// Apply pagination
	totalCount = int64(len(allResults))
	start := req.Offset
	end := req.Offset + req.Limit

	if start > len(allResults) {
		start = len(allResults)
	}
	if end > len(allResults) {
		end = len(allResults)
	}

	paginatedResults := allResults[start:end]

	// Generate search suggestions
	suggestions := s.generateSuggestions(req.Query, allResults)
	if suggestions == nil {
		suggestions = []string{}
	}

	duration := time.Since(startTime)

	// Ensure no nil values in response
	if paginatedResults == nil {
		paginatedResults = []models.GlobalSearchResult{}
	}

	return &models.GlobalSearchResponse{
		Results:     paginatedResults,
		Total:       totalCount,
		Query:       req.Query,
		Duration:    duration.String(),
		Suggestions: suggestions,
		TypeCounts:  typeCounts,
	}, nil
}

// QuickSearch performs a lightweight search for autocomplete
func (s *GlobalSearchService) QuickSearch(ctx context.Context, userID uuid.UUID, query string, limit int) (*models.QuickSearchResponse, error) {
	if limit <= 0 {
		limit = 10
	}
	if limit > 20 {
		limit = 20
	}

	req := models.GlobalSearchRequest{
		Query:           query,
		Limit:           limit * 2, // Get more results to filter
		IncludeArchived: false,
	}

	fullResults, err := s.GlobalSearch(ctx, userID, req)
	if err != nil {
		return nil, err
	}

	// Convert to quick search results
	var quickResults []models.QuickSearchResult
	for _, result := range fullResults.Results {
		if len(quickResults) >= limit {
			break
		}

		// Safe icon handling for quick search
		var icon *string
		if result.Icon != nil && *result.Icon != "" {
			icon = result.Icon
		}

		quickResults = append(quickResults, models.QuickSearchResult{
			ID:    result.ID,
			Type:  result.Type,
			Title: result.Title,
			Icon:  icon, // Will be null if not set
			Score: result.Score,
		})
	}

	// Ensure results is never nil
	if quickResults == nil {
		quickResults = []models.QuickSearchResult{}
	}

	return &models.QuickSearchResponse{
		Results: quickResults,
		Query:   query,
	}, nil
}

// searchProjects searches within projects
func (s *GlobalSearchService) searchProjects(ctx context.Context, userID uuid.UUID, req models.GlobalSearchRequest) ([]models.GlobalSearchResult, int64, error) {
	// Use existing project search
	projects, total, err := s.projectService.SearchProjects(ctx, userID, req.Query, req.Limit*2, 0)
	if err != nil {
		return nil, 0, err
	}

	var results []models.GlobalSearchResult
	for _, project := range projects {
		// Skip archived projects unless requested
		if project.IsArchived && !req.IncludeArchived {
			continue
		}

		// Filter by project ID if specified
		if req.ProjectID != nil && project.ID != *req.ProjectID {
			continue
		}

		// Calculate relevance score
		score := s.calculateProjectScore(*project, req.Query)

		// Generate snippet
		snippet := s.generateProjectSnippet(*project, req.Query)

		// Safe description handling
		var description *string
		if project.Description != nil && *project.Description != "" {
			description = project.Description
		}

		// Safe image URL handling
		var imageURL *string
		if project.ImageURL != nil && *project.ImageURL != "" {
			imageURL = project.ImageURL
		}

		// Safe color theme handling
		var colorTheme *string
		if project.ColorTheme != "" {
			colorTheme = &project.ColorTheme
		}

		result := models.GlobalSearchResult{
			ID:          project.ID,
			Type:        models.SearchResultTypeProject,
			Title:       project.Title,
			Description: description,
			Snippet:     snippet,
			ProjectID:   nil, // Projects don't have parent project
			ProjectName: nil, // Projects don't have parent project name
			ImageURL:    imageURL,
			Icon:        nil, // Projects don't have emoji icons
			Status:      nil, // Projects don't have status
			Priority:    nil, // Projects don't have priority
			Progress:    &project.ProgressPercentage,
			IsArchived:  project.IsArchived,
			IsFavorite:  &project.IsFavorite,
			IsCompleted: nil, // Projects don't have completion status
			DueDate:     nil, // Projects don't have due dates
			CompletedAt: nil, // Projects don't have completion dates
			CreatedAt:   project.CreatedAt,
			UpdatedAt:   project.UpdatedAt,
			ColorTheme:  colorTheme,
			Tags:        []string{}, // Projects don't have tags
			Score:       score,
			MatchType:   s.getProjectMatchType(*project, req.Query),
		}

		results = append(results, result)
	}

	return results, int64(total), nil
}

// searchTasks searches within tasks
func (s *GlobalSearchService) searchTasks(ctx context.Context, userID uuid.UUID, req models.GlobalSearchRequest) ([]models.GlobalSearchResult, int64, error) {
	// Use existing todo search
	todos, total, err := s.todoService.SearchTodos(ctx, userID, req.Query, req.Limit*2, 0)
	if err != nil {
		return nil, 0, err
	}

	var results []models.GlobalSearchResult
	for _, todo := range todos {
		// Skip archived tasks unless requested
		if todo.IsArchived && !req.IncludeArchived {
			continue
		}

		// Filter by project ID if specified
		if req.ProjectID != nil && todo.ProjectID != *req.ProjectID {
			continue
		}

		// Calculate relevance score
		score := s.calculateTaskScore(*todo, req.Query)

		// Generate snippet
		snippet := s.generateTaskSnippet(*todo, req.Query)

		// Get project name with better error handling
		var projectName *string
		if project, err := s.projectService.GetProject(ctx, userID, todo.ProjectID, false); err == nil && project != nil {
			projectName = &project.Title
		}

		// Convert status and priority to strings - ensure they're never empty
		status := string(todo.Status)
		if status == "" {
			status = string(models.TaskStatusNotStarted)
		}
		priority := string(todo.Priority)
		if priority == "" {
			priority = string(models.PriorityMedium)
		}

		// Check if completed
		isCompleted := todo.Status == models.TaskStatusCompleted || todo.Status == models.TaskStatusDone

		// Ensure tags is never nil
		tags := []string(todo.Tags)
		if tags == nil {
			tags = []string{}
		}

		// Safe icon handling - ensure consistent string pointer
		var icon *string
		if todo.TaskImageOrEmoji != nil && *todo.TaskImageOrEmoji != "" {
			icon = todo.TaskImageOrEmoji
		}

		// Safe description handling
		var description *string
		if todo.TaskDescription != nil && *todo.TaskDescription != "" {
			description = todo.TaskDescription
		}

		result := models.GlobalSearchResult{
			ID:          todo.ID,
			Type:        models.SearchResultTypeTask,
			Title:       todo.TaskName,
			Description: description,
			Snippet:     snippet,
			ProjectID:   &todo.ProjectID,
			ProjectName: projectName,
			Icon:        icon,
			Status:      &status,
			Priority:    &priority,
			Progress:    &todo.CompletionPercentage,
			IsArchived:  todo.IsArchived,
			IsCompleted: &isCompleted,
			DueDate:     todo.DueDate,
			CompletedAt: todo.CompletedAt,
			CreatedAt:   todo.CreatedAt,
			UpdatedAt:   todo.UpdatedAt,
			Tags:        tags,
			Score:       score,
			MatchType:   s.getTaskMatchType(*todo, req.Query),
		}

		results = append(results, result)
	}

	return results, int64(total), nil
}

// calculateProjectScore calculates relevance score for a project
func (s *GlobalSearchService) calculateProjectScore(project models.Project, query string) float64 {
	query = strings.ToLower(query)
	title := strings.ToLower(project.Title)
	description := ""
	if project.Description != nil {
		description = strings.ToLower(*project.Description)
	}

	score := 0.0

	// Exact title match gets highest score
	if title == query {
		score += 100.0
	} else if strings.HasPrefix(title, query) {
		score += 80.0
	} else if strings.Contains(title, query) {
		score += 60.0
	}

	// Description matches
	if strings.Contains(description, query) {
		score += 30.0
	}

	// Boost for favorites
	if project.IsFavorite {
		score += 10.0
	}

	// Reduce score for archived
	if project.IsArchived {
		score -= 20.0
	}

	// Recent activity boost
	daysSinceUpdate := time.Since(project.UpdatedAt).Hours() / 24
	if daysSinceUpdate < 7 {
		score += 5.0
	}

	return score
}

// calculateTaskScore calculates relevance score for a task
func (s *GlobalSearchService) calculateTaskScore(todo models.Todo, query string) float64 {
	query = strings.ToLower(query)
	title := strings.ToLower(todo.TaskName)
	description := ""
	if todo.TaskDescription != nil {
		description = strings.ToLower(*todo.TaskDescription)
	}

	score := 0.0

	// Exact title match gets highest score
	if title == query {
		score += 100.0
	} else if strings.HasPrefix(title, query) {
		score += 80.0
	} else if strings.Contains(title, query) {
		score += 60.0
	}

	// Description matches
	if strings.Contains(description, query) {
		score += 30.0
	}

	// Tag matches
	for _, tag := range todo.Tags {
		if strings.Contains(strings.ToLower(tag), query) {
			score += 25.0
		}
	}

	// Priority boost
	switch todo.Priority {
	case models.PriorityUrgent:
		score += 15.0
	case models.PriorityHigh:
		score += 10.0
	}

	// Reduce score for completed tasks
	if todo.Status == models.TaskStatusCompleted || todo.Status == models.TaskStatusDone {
		score -= 15.0
	}

	// Reduce score for archived
	if todo.IsArchived {
		score -= 20.0
	}

	// Due date proximity boost
	if todo.DueDate != nil {
		daysUntilDue := time.Until(*todo.DueDate).Hours() / 24
		if daysUntilDue <= 7 && daysUntilDue >= 0 {
			score += 8.0
		}
	}

	return score
}

// Helper functions
func (s *GlobalSearchService) generateProjectSnippet(project models.Project, query string) *string {
	if project.Description == nil {
		return nil
	}

	description := *project.Description
	query = strings.ToLower(query)
	descLower := strings.ToLower(description)

	// Find query in description
	index := strings.Index(descLower, query)
	if index == -1 {
		// Return first 100 characters if no match
		if len(description) > 100 {
			snippet := description[:100] + "..."
			return &snippet
		}
		return &description
	}

	// Extract snippet around the match
	start := max(0, index-50)
	end := min(len(description), index+len(query)+50)

	snippet := description[start:end]
	if start > 0 {
		snippet = "..." + snippet
	}
	if end < len(description) {
		snippet = snippet + "..."
	}

	return &snippet
}

func (s *GlobalSearchService) generateTaskSnippet(todo models.Todo, query string) *string {
	if todo.TaskDescription == nil {
		return nil
	}

	description := *todo.TaskDescription
	query = strings.ToLower(query)
	descLower := strings.ToLower(description)

	// Find query in description
	index := strings.Index(descLower, query)
	if index == -1 {
		// Return first 100 characters if no match
		if len(description) > 100 {
			snippet := description[:100] + "..."
			return &snippet
		}
		return &description
	}

	// Extract snippet around the match
	start := max(0, index-50)
	end := min(len(description), index+len(query)+50)

	snippet := description[start:end]
	if start > 0 {
		snippet = "..." + snippet
	}
	if end < len(description) {
		snippet = snippet + "..."
	}

	return &snippet
}

func (s *GlobalSearchService) getProjectMatchType(project models.Project, query string) string {
	query = strings.ToLower(query)
	title := strings.ToLower(project.Title)

	if strings.Contains(title, query) {
		return "title"
	}

	if project.Description != nil && strings.Contains(strings.ToLower(*project.Description), query) {
		return "description"
	}

	return "other"
}

func (s *GlobalSearchService) getTaskMatchType(todo models.Todo, query string) string {
	query = strings.ToLower(query)
	title := strings.ToLower(todo.TaskName)

	if strings.Contains(title, query) {
		return "title"
	}

	if todo.TaskDescription != nil && strings.Contains(strings.ToLower(*todo.TaskDescription), query) {
		return "description"
	}

	for _, tag := range todo.Tags {
		if strings.Contains(strings.ToLower(tag), query) {
			return "tags"
		}
	}

	return "other"
}

func (s *GlobalSearchService) generateSuggestions(query string, results []models.GlobalSearchResult) []string {
	var suggestions []string
	queryLower := strings.ToLower(query)

	// Extract unique words from high-scoring results
	wordFreq := make(map[string]int)

	for _, result := range results {
		if result.Score > 50 { // Only consider high-relevance results
			words := strings.Fields(strings.ToLower(result.Title))
			for _, word := range words {
				if len(word) > 2 && !strings.Contains(queryLower, word) {
					wordFreq[word]++
				}
			}
		}
	}

	// Sort by frequency
	type wordCount struct {
		word  string
		count int
	}

	var wordCounts []wordCount
	for word, count := range wordFreq {
		wordCounts = append(wordCounts, wordCount{word, count})
	}

	sort.Slice(wordCounts, func(i, j int) bool {
		return wordCounts[i].count > wordCounts[j].count
	})

	// Generate suggestions
	for i, wc := range wordCounts {
		if i >= 5 { // Limit to 5 suggestions
			break
		}
		suggestions = append(suggestions, query+" "+wc.word)
	}

	return suggestions
}

// Utility functions
func contains(slice []models.SearchResultType, item models.SearchResultType) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
