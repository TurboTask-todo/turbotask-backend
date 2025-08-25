package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// ExternalAIClient handles AI requests to external AI model API
type ExternalAIClient struct {
	baseURL    string
	httpClient *http.Client
	maxRetries int
	retryDelay time.Duration
}

// ExternalAIRequest represents the request structure for the external AI API
type ExternalAIRequest struct {
	Instruction string  `json:"instruction"`
	Text        string  `json:"text"`
	Model       string  `json:"model"`
	Temperature float64 `json:"temperature"`
	MaxTokens   int     `json:"max_tokens"`
	Stream      bool    `json:"stream"`
	CacheKey    string  `json:"cacheKey"`
	SkipCache   bool    `json:"skipCache"`
}

// ExternalAIResponse represents the response structure from the external AI API
type ExternalAIResponse struct {
	Response  string                 `json:"response"`
	ToolCalls []interface{}          `json:"tool_calls"`
	Usage     ExternalAIUsage        `json:"usage"`
	Metadata  map[string]interface{} `json:"metadata"`
}

// ExternalAIUsage represents token usage information
type ExternalAIUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// NewExternalAIClient creates a new external AI client
func NewExternalAIClient(baseURL string) *ExternalAIClient {
	return &ExternalAIClient{
		baseURL:    baseURL,
		maxRetries: 3,
		retryDelay: 2 * time.Second,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// NewExternalAIClientWithConfig creates a new external AI client with custom configuration
func NewExternalAIClientWithConfig(baseURL string, timeout time.Duration, maxRetries int) *ExternalAIClient {
	return &ExternalAIClient{
		baseURL:    baseURL,
		maxRetries: maxRetries,
		retryDelay: 2 * time.Second,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// EnhanceTask enhances a task with AI-generated description, tags, and subtasks
func (c *ExternalAIClient) EnhanceTask(ctx context.Context, req TaskEnhancementRequest) (*TaskEnhancementResponse, error) {
	// Create a comprehensive prompt for task enhancement
	prompt := c.buildTaskEnhancementPrompt(req)
	
	// Make the AI request
	aiReq := ExternalAIRequest{
		Instruction: "You are a professional task management assistant. Analyze the given task and provide: 1) An enhanced, detailed description, 2) Relevant tags (max 5), 3) Subtasks breakdown (max 6), 4) Priority level, 5) Estimated duration in minutes, 6) Task category, 7) An appropriate emoji. Format your response as JSON with keys: enhanced_description, tags (array), subtasks (array of objects with name, description, estimated_duration_minutes, order), priority, estimated_duration_minutes, category, emoji.",
		Text:        prompt,
		Model:       "@cf/meta/llama-3.3-70b-instruct-fp8-fast",
		Temperature: 0.7,
		MaxTokens:   1024,
		Stream:      false,
		CacheKey:    fmt.Sprintf("task_enhancement_%s", req.TaskName),
		SkipCache:   false,
	}

	response, err := c.makeAIRequest(ctx, aiReq)
	if err != nil {
		return nil, fmt.Errorf("failed to get AI response: %w", err)
	}

	// Parse the AI response to extract structured data
	return c.parseTaskEnhancementResponse(response.Response)
}

// RefineSubtasks refines existing subtasks using AI
func (c *ExternalAIClient) RefineSubtasks(ctx context.Context, taskName string, existingSubtasks []string, userFeedback string) ([]AIGeneratedSubtask, error) {
	prompt := fmt.Sprintf("Task: %s\nExisting subtasks: %s\nUser feedback: %s\n\nPlease refine and improve the subtasks based on the feedback. Provide 3-6 improved subtasks with descriptions and time estimates.",
		taskName, strings.Join(existingSubtasks, ", "), userFeedback)

	aiReq := ExternalAIRequest{
		Instruction: "You are a task management expert. Refine the given subtasks based on user feedback. Return JSON array with objects containing: name, description, estimated_duration_minutes, order.",
		Text:        prompt,
		Model:       "@cf/meta/llama-3.3-70b-instruct-fp8-fast",
		Temperature: 0.6,
		MaxTokens:   768,
		Stream:      false,
		CacheKey:    fmt.Sprintf("subtask_refinement_%s", taskName),
		SkipCache:   false,
	}

	response, err := c.makeAIRequest(ctx, aiReq)
	if err != nil {
		return nil, fmt.Errorf("failed to get AI response: %w", err)
	}

	return c.parseSubtaskRefinementResponse(response.Response)
}

// ImproveDescription improves a task description using AI
func (c *ExternalAIClient) ImproveDescription(ctx context.Context, taskName string, currentDescription string, priority string, tags []string) (string, error) {
	prompt := fmt.Sprintf("Task: %s\nCurrent description: %s\nPriority: %s\nTags: %s\n\nPlease improve this description to be more detailed, actionable, and professional.",
		taskName, currentDescription, priority, strings.Join(tags, ", "))

	aiReq := ExternalAIRequest{
		Instruction: "You are a professional task writer. Improve the given task description to be more detailed, actionable, and professional. Return only the improved description.",
		Text:        prompt,
		Model:       "@cf/meta/llama-3.3-70b-instruct-fp8-fast",
		Temperature: 0.5,
		MaxTokens:   512,
		Stream:      false,
		CacheKey:    fmt.Sprintf("description_improvement_%s", taskName),
		SkipCache:   false,
	}

	response, err := c.makeAIRequest(ctx, aiReq)
	if err != nil {
		return "", fmt.Errorf("failed to get AI response: %w", err)
	}

	return response.Response, nil
}

// buildTaskEnhancementPrompt creates a comprehensive prompt for task enhancement
func (c *ExternalAIClient) buildTaskEnhancementPrompt(req TaskEnhancementRequest) string {
	var prompt strings.Builder
	
	prompt.WriteString(fmt.Sprintf("Task Name: %s\n", req.TaskName))
	
	if req.TaskDescription != "" {
		prompt.WriteString(fmt.Sprintf("Current Description: %s\n", req.TaskDescription))
	}
	
	if req.ProjectContext != "" {
		prompt.WriteString(fmt.Sprintf("Project Context: %s\n", req.ProjectContext))
	}
	
	if req.UserPreferences != "" {
		prompt.WriteString(fmt.Sprintf("User Preferences: %s\n", req.UserPreferences))
	}
	
	prompt.WriteString("\nPlease enhance this task by providing:\n")
	prompt.WriteString("1. A detailed, actionable description\n")
	prompt.WriteString("2. 3-5 relevant tags\n")
	prompt.WriteString("3. 3-6 logical subtasks with descriptions and time estimates\n")
	prompt.WriteString("4. Priority level (low, medium, high, urgent)\n")
	prompt.WriteString("5. Estimated duration in minutes\n")
	prompt.WriteString("6. Task category (work, personal, learning, health, etc.)\n")
	prompt.WriteString("7. An appropriate emoji\n")
	
	return prompt.String()
}

// makeAIRequest makes the actual HTTP request to the AI API
func (c *ExternalAIClient) makeAIRequest(ctx context.Context, req ExternalAIRequest) (*ExternalAIResponse, error) {
	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	var lastErr error
	for attempt := 0; attempt < c.maxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(c.retryDelay * time.Duration(attempt))
		}

		httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL, bytes.NewBuffer(jsonData))
		if err != nil {
			lastErr = fmt.Errorf("failed to create request: %w", err)
			continue
		}

		httpReq.Header.Set("Content-Type", "application/json")

		resp, err := c.httpClient.Do(httpReq)
		if err != nil {
			lastErr = fmt.Errorf("request failed: %w", err)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("API returned status %d", resp.StatusCode)
			continue
		}

		var aiResp ExternalAIResponse
		if err := json.NewDecoder(resp.Body).Decode(&aiResp); err != nil {
			lastErr = fmt.Errorf("failed to decode response: %w", err)
			continue
		}

		return &aiResp, nil
	}

	return nil, fmt.Errorf("failed after %d attempts: %w", c.maxRetries, lastErr)
}

// parseTaskEnhancementResponse parses the AI response to extract structured task enhancement data
func (c *ExternalAIClient) parseTaskEnhancementResponse(response string) (*TaskEnhancementResponse, error) {
	// Try to extract JSON from the response
	jsonStart := strings.Index(response, "{")
	jsonEnd := strings.LastIndex(response, "}")
	
	if jsonStart == -1 || jsonEnd == -1 {
		// Fallback: create a basic response from the text
		return &TaskEnhancementResponse{
			EnhancedDescription: response,
			Emoji:               "📝",
			Subtasks:            []AIGeneratedSubtask{},
			Category:            "general",
			Priority:            "medium",
			EstimatedDuration:   30,
			Tags:                []string{"task", "enhanced"},
		}, nil
	}

	jsonStr := response[jsonStart : jsonEnd+1]
	
	var parsed struct {
		EnhancedDescription string   `json:"enhanced_description"`
		Tags                []string `json:"tags"`
		Subtasks            []struct {
			Name              string `json:"name"`
			Description       string `json:"description"`
			EstimatedDuration int    `json:"estimated_duration_minutes"`
			Order             int    `json:"order"`
		} `json:"subtasks"`
		Priority           string `json:"priority"`
		EstimatedDuration  int    `json:"estimated_duration_minutes"`
		Category           string `json:"category"`
		Emoji              string `json:"emoji"`
	}

	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		// Fallback: create a basic response
		return &TaskEnhancementResponse{
			EnhancedDescription: response,
			Emoji:               "📝",
			Subtasks:            []AIGeneratedSubtask{},
			Category:            "general",
			Priority:            "medium",
			EstimatedDuration:   30,
			Tags:                []string{"task", "enhanced"},
		}, nil
	}

	// Convert parsed subtasks to AIGeneratedSubtask
	subtasks := make([]AIGeneratedSubtask, len(parsed.Subtasks))
	for i, st := range parsed.Subtasks {
		subtasks[i] = AIGeneratedSubtask{
			Name:              st.Name,
			Description:       st.Description,
			EstimatedDuration: st.EstimatedDuration,
			Order:             st.Order,
		}
	}

	// Set defaults if values are missing
	if parsed.Emoji == "" {
		parsed.Emoji = "📝"
	}
	if parsed.Category == "" {
		parsed.Category = "general"
	}
	if parsed.Priority == "" {
		parsed.Priority = "medium"
	}
	if parsed.EstimatedDuration == 0 {
		parsed.EstimatedDuration = 30
	}
	if len(parsed.Tags) == 0 {
		parsed.Tags = []string{"task", "enhanced"}
	}

	return &TaskEnhancementResponse{
		EnhancedDescription: parsed.EnhancedDescription,
		Emoji:               parsed.Emoji,
		Subtasks:            subtasks,
		Category:            parsed.Category,
		Priority:            parsed.Priority,
		EstimatedDuration:   parsed.EstimatedDuration,
		Tags:                parsed.Tags,
	}, nil
}

// parseSubtaskRefinementResponse parses the AI response for subtask refinement
func (c *ExternalAIClient) parseSubtaskRefinementResponse(response string) ([]AIGeneratedSubtask, error) {
	// Try to extract JSON array from the response
	jsonStart := strings.Index(response, "[")
	jsonEnd := strings.LastIndex(response, "]")
	
	if jsonStart == -1 || jsonEnd == -1 {
		// Fallback: create basic subtasks
		return []AIGeneratedSubtask{
			{
				Name:              "Refined subtask",
				Description:       response,
				EstimatedDuration: 30,
				Order:             1,
			},
		}, nil
	}

	jsonStr := response[jsonStart : jsonEnd+1]
	
	var parsed []struct {
		Name              string `json:"name"`
		Description       string `json:"description"`
		EstimatedDuration int    `json:"estimated_duration_minutes"`
		Order             int    `json:"order"`
	}

	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		// Fallback: create basic subtasks
		return []AIGeneratedSubtask{
			{
				Name:              "Refined subtask",
				Description:       response,
				EstimatedDuration: 30,
				Order:             1,
			},
		}, nil
	}

	// Convert to AIGeneratedSubtask
	subtasks := make([]AIGeneratedSubtask, len(parsed))
	for i, st := range parsed {
		if st.EstimatedDuration == 0 {
			st.EstimatedDuration = 30
		}
		if st.Order == 0 {
			st.Order = i + 1
		}
		
		subtasks[i] = AIGeneratedSubtask{
			Name:              st.Name,
			Description:       st.Description,
			EstimatedDuration: st.EstimatedDuration,
			Order:             st.Order,
		}
	}

	return subtasks, nil
}
