package handler

import (
	"log"
	"net/http"
	"strconv"
	"strings"

	"quantumtask-auth-api/internal/models"
	"quantumtask-auth-api/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// GlobalSearchHandler handles global search requests
type GlobalSearchHandler struct {
	globalSearchService *service.GlobalSearchService
}

// NewGlobalSearchHandler creates a new global search handler
func NewGlobalSearchHandler(globalSearchService *service.GlobalSearchService) *GlobalSearchHandler {
	return &GlobalSearchHandler{
		globalSearchService: globalSearchService,
	}
}

// GlobalSearch handles global search across projects and tasks
// @Summary Global search across projects and tasks
// @Description Performs a unified search across projects and tasks with advanced filtering and scoring
// @Tags search
// @Accept json
// @Produce json
// @Param q query string true "Search query"
// @Param types query string false "Search types (comma-separated: project,task)"
// @Param project_id query string false "Filter by project ID (UUID)"
// @Param limit query int false "Maximum number of results (default: 20, max: 100)"
// @Param offset query int false "Offset for pagination (default: 0)"
// @Param include_archived query bool false "Include archived items (default: false)"
// @Success 200 {object} models.GlobalSearchResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/search [get]
func (h *GlobalSearchHandler) GlobalSearch(c *gin.Context) {
	// Get user ID from context
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "Unauthorized",
			"message": "User ID not found in context",
		})
		return
	}

	userUUID, ok := userID.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "Unauthorized",
			"message": "Invalid user ID format",
		})
		return
	}

	// Parse and validate query parameters
	query := strings.TrimSpace(c.Query("q"))
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": "Search query is required",
		})
		return
	}

	// Sanitize query to prevent issues
	if len(query) > 500 {
		query = query[:500]
	}

	// Parse search types
	var searchTypes []models.SearchResultType
	if typesParam := c.Query("types"); typesParam != "" {
		typeStrs := strings.Split(typesParam, ",")
		for _, typeStr := range typeStrs {
			typeStr = strings.TrimSpace(typeStr)
			switch typeStr {
			case "project":
				searchTypes = append(searchTypes, models.SearchResultTypeProject)
			case "task":
				searchTypes = append(searchTypes, models.SearchResultTypeTask)
			default:
				c.JSON(http.StatusBadRequest, gin.H{
					"error":   "Bad Request",
					"message": "Invalid search type: " + typeStr + ". Allowed types: project, task",
				})
				return
			}
		}
	}

	// Parse project ID filter
	var projectID *uuid.UUID
	if projectIDParam := c.Query("project_id"); projectIDParam != "" {
		parsed, err := uuid.Parse(projectIDParam)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Bad Request",
				"message": "Invalid project ID format",
			})
			return
		}
		projectID = &parsed
	}

	// Parse pagination parameters
	limit := 20
	if limitParam := c.Query("limit"); limitParam != "" {
		if parsed, err := strconv.Atoi(limitParam); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	offset := 0
	if offsetParam := c.Query("offset"); offsetParam != "" {
		if parsed, err := strconv.Atoi(offsetParam); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	// Parse include archived
	includeArchived := false
	if archivedParam := c.Query("include_archived"); archivedParam != "" {
		includeArchived = archivedParam == "true" || archivedParam == "1"
	}

	// Create search request
	searchRequest := models.GlobalSearchRequest{
		Query:           query,
		Types:           searchTypes,
		ProjectID:       projectID,
		Limit:           limit,
		Offset:          offset,
		IncludeArchived: includeArchived,
	}

	// Perform search with enhanced error handling
	results, err := h.globalSearchService.GlobalSearch(c.Request.Context(), userUUID, searchRequest)
	if err != nil {
		// Log the error for debugging
		log.Printf("Global search error for user %s with query '%s': %v", userUUID, query, err)

		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Internal Server Error",
			"message": "Failed to perform global search",
			"query":   query,
		})
		return
	}

	// Ensure results is never nil
	if results == nil {
		results = &models.GlobalSearchResponse{
			Results:     []models.GlobalSearchResult{},
			Total:       0,
			Query:       query,
			Duration:    "0s",
			Suggestions: []string{},
			TypeCounts:  map[string]int64{},
		}
	}

	// Ensure results arrays are never nil
	if results.Results == nil {
		results.Results = []models.GlobalSearchResult{}
	}
	if results.Suggestions == nil {
		results.Suggestions = []string{}
	}
	if results.TypeCounts == nil {
		results.TypeCounts = map[string]int64{}
	}

	c.JSON(http.StatusOK, results)
}

// QuickSearch handles lightweight search for autocomplete
// @Summary Quick search for autocomplete
// @Description Performs a lightweight search optimized for autocomplete/typeahead functionality
// @Tags search
// @Accept json
// @Produce json
// @Param q query string true "Search query"
// @Param limit query int false "Maximum number of results (default: 10, max: 20)"
// @Success 200 {object} models.QuickSearchResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/search/quick [get]
func (h *GlobalSearchHandler) QuickSearch(c *gin.Context) {
	// Get user ID from context
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "Unauthorized",
			"message": "User ID not found in context",
		})
		return
	}

	userUUID, ok := userID.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "Unauthorized",
			"message": "Invalid user ID format",
		})
		return
	}

	// Parse query parameters
	query := strings.TrimSpace(c.Query("q"))
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Bad Request",
			"message": "Search query is required",
		})
		return
	}

	// Parse limit
	limit := 10
	if limitParam := c.Query("limit"); limitParam != "" {
		if parsed, err := strconv.Atoi(limitParam); err == nil && parsed > 0 && parsed <= 20 {
			limit = parsed
		}
	}

	// Perform quick search
	results, err := h.globalSearchService.QuickSearch(c.Request.Context(), userUUID, query, limit)
	if err != nil {
		// Log the error for debugging
		log.Printf("Quick search error for user %s with query '%s': %v", userUUID, query, err)

		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Internal Server Error",
			"message": "Failed to perform quick search",
			"query":   query,
		})
		return
	}

	// Ensure results is never nil
	if results == nil {
		results = &models.QuickSearchResponse{
			Results: []models.QuickSearchResult{},
			Query:   query,
		}
	}

	// Ensure results array is never nil
	if results.Results == nil {
		results.Results = []models.QuickSearchResult{}
	}

	c.JSON(http.StatusOK, results)
}

// SearchSuggestions handles search suggestions
// @Summary Get search suggestions
// @Description Returns search suggestions based on user's search history and popular searches
// @Tags search
// @Accept json
// @Produce json
// @Success 200 {object} []models.SearchSuggestion
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/search/suggestions [get]
func (h *GlobalSearchHandler) SearchSuggestions(c *gin.Context) {
	// Get user ID from context
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "Unauthorized",
			"message": "User ID not found in context",
		})
		return
	}

	_, ok := userID.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "Unauthorized",
			"message": "Invalid user ID format",
		})
		return
	}

	// For now, return some static suggestions
	// In production, this would pull from user's search history and popular searches
	suggestions := []models.SearchSuggestion{
		{Query: "urgent tasks", Score: 95.0, Type: "popular"},
		{Query: "overdue", Score: 90.0, Type: "popular"},
		{Query: "high priority", Score: 85.0, Type: "popular"},
		{Query: "completed today", Score: 80.0, Type: "recent"},
		{Query: "in progress", Score: 75.0, Type: "popular"},
	}

	c.JSON(http.StatusOK, suggestions)
}

// SearchHistory handles search history
// @Summary Get user's search history
// @Description Returns the user's recent search queries
// @Tags search
// @Accept json
// @Produce json
// @Param limit query int false "Maximum number of results (default: 20)"
// @Success 200 {object} []string
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/search/history [get]
func (h *GlobalSearchHandler) SearchHistory(c *gin.Context) {
	// Get user ID from context
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "Unauthorized",
			"message": "User ID not found in context",
		})
		return
	}

	userUUID, ok := userID.(uuid.UUID)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "Unauthorized",
			"message": "Invalid user ID format",
		})
		return
	}

	// Parse limit
	limit := 20
	if limitParam := c.Query("limit"); limitParam != "" {
		if parsed, err := strconv.Atoi(limitParam); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	// For now, return empty history
	// In production, this would pull from a search_history table
	_ = userUUID
	_ = limit

	history := []string{}
	c.JSON(http.StatusOK, history)
}
