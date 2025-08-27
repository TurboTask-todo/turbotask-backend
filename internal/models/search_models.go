package models

import (
	"time"

	"github.com/google/uuid"
)

// SearchResultType represents the type of search result
type SearchResultType string

const (
	SearchResultTypeProject SearchResultType = "project"
	SearchResultTypeTask    SearchResultType = "task"
)

// GlobalSearchRequest represents a global search request
type GlobalSearchRequest struct {
	Query           string             `json:"query" validate:"required,min=1"`
	Types           []SearchResultType `json:"types,omitempty"`            // Filter by types
	ProjectID       *uuid.UUID         `json:"project_id,omitempty"`       // Filter by project
	Limit           int                `json:"limit,omitempty"`            // Default: 20
	Offset          int                `json:"offset,omitempty"`           // Default: 0
	IncludeArchived bool               `json:"include_archived,omitempty"` // Default: false
}

// GlobalSearchResult represents a unified search result
type GlobalSearchResult struct {
	ID          uuid.UUID        `json:"id"`
	Type        SearchResultType `json:"type"`
	Title       string           `json:"title"`
	Description *string          `json:"description"`
	Snippet     *string          `json:"snippet"`      // Highlighted search snippet
	ProjectID   *uuid.UUID       `json:"project_id"`   // For tasks
	ProjectName *string          `json:"project_name"` // For tasks
	ImageURL    *string          `json:"image_url"`
	Icon        *string          `json:"icon"` // Emoji or icon identifier

	// Status information
	Status      *string  `json:"status"`   // For tasks
	Priority    *string  `json:"priority"` // For tasks
	Progress    *float64 `json:"progress"` // Progress percentage
	IsArchived  bool     `json:"is_archived"`
	IsFavorite  *bool    `json:"is_favorite"`  // For projects
	IsCompleted *bool    `json:"is_completed"` // For tasks

	// Dates
	DueDate     *time.Time `json:"due_date"`
	CompletedAt *time.Time `json:"completed_at"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`

	// Additional metadata for navigation
	ColorTheme *string  `json:"color_theme"` // For projects
	Tags       []string `json:"tags"`        // For tasks

	// Relevance scoring
	Score     float64 `json:"score"`      // Search relevance score
	MatchType string  `json:"match_type"` // title, description, tags, etc.
}

// GlobalSearchResponse represents the API response for global search
type GlobalSearchResponse struct {
	Results     []GlobalSearchResult `json:"results"`
	Total       int64                `json:"total"`
	Query       string               `json:"query"`
	Duration    string               `json:"duration"`    // Search duration
	Suggestions []string             `json:"suggestions"` // Search suggestions
	TypeCounts  map[string]int64     `json:"type_counts"` // Count by type
}

// SearchSuggestion represents a search suggestion
type SearchSuggestion struct {
	Query string  `json:"query"`
	Score float64 `json:"score"`
	Type  string  `json:"type"` // recent, popular, similar
}

// QuickSearchResult represents a lightweight search result for autocomplete
type QuickSearchResult struct {
	ID    uuid.UUID        `json:"id"`
	Type  SearchResultType `json:"type"`
	Title string           `json:"title"`
	Icon  *string          `json:"icon"`
	Score float64          `json:"score"`
}

// QuickSearchResponse represents the API response for quick search
type QuickSearchResponse struct {
	Results []QuickSearchResult `json:"results"`
	Query   string              `json:"query"`
}
