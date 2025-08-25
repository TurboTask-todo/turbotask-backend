package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"macwrite-auth-api/internal/middleware"
	"macwrite-auth-api/internal/models"
	"macwrite-auth-api/internal/repository"
	"macwrite-auth-api/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// TodoHandler handles todo HTTP requests
type TodoHandler struct {
	todoService *service.TodoService
}

// NewTodoHandler creates a new todo handler
func NewTodoHandler(todoService *service.TodoService) *TodoHandler {
	return &TodoHandler{
		todoService: todoService,
	}
}

// CreateTodo handles todo creation
// @Summary Create a new todo
// @Description Create a new todo for the authenticated user
// @Tags Todos
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body models.CreateTodoRequest true "Todo details"
// @Success 201 {object} models.APIResponse{data=models.Todo} "Todo created successfully"
// @Failure 400 {object} models.APIResponse "Invalid request"
// @Failure 401 {object} models.APIResponse "Unauthorized"
// @Failure 500 {object} models.APIResponse "Internal server error"
// @Router /todos [post]
func (h *TodoHandler) CreateTodo(c *gin.Context) {
	// Get user ID from context
	userID, _ := middleware.GetUserID(c)

	var req models.CreateTodoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse(
			"INVALID_REQUEST",
			"Invalid request data",
			map[string]string{"details": err.Error()},
		))
		return
	}

	// Create todo
	todo, err := h.todoService.CreateTodo(c.Request.Context(), userID, &req)
	if err != nil {
		statusCode := http.StatusInternalServerError
		errorCode := "TODO_CREATION_FAILED"

		if strings.Contains(err.Error(), "validation failed") {
			statusCode = http.StatusBadRequest
			errorCode = "VALIDATION_ERROR"
		} else if strings.Contains(err.Error(), "not found") {
			statusCode = http.StatusBadRequest
			errorCode = "PROJECT_NOT_FOUND"
		}

		c.JSON(statusCode, models.NewErrorResponse(
			errorCode,
			err.Error(),
			nil,
		))
		return
	}

	c.JSON(http.StatusCreated, models.NewAPIResponse(
		"Todo created successfully",
		todo,
	))
}

// GetTodo handles retrieving a single todo
// @Summary Get todo by ID
// @Description Get a todo by its ID for the authenticated user
// @Tags Todos
// @Produce json
// @Security BearerAuth
// @Param id path string true "Todo ID"
// @Param include_related query boolean false "Include related data (subtasks, comments, etc.)"
// @Success 200 {object} models.APIResponse{data=models.Todo} "Todo retrieved successfully"
// @Failure 400 {object} models.APIResponse "Invalid todo ID"
// @Failure 401 {object} models.APIResponse "Unauthorized"
// @Failure 404 {object} models.APIResponse "Todo not found"
// @Failure 500 {object} models.APIResponse "Internal server error"
// @Router /todos/{id} [get]
func (h *TodoHandler) GetTodo(c *gin.Context) {
	// Get user ID from context
	userID, _ := middleware.GetUserID(c)

	// Parse todo ID
	todoIDStr := c.Param("id")
	todoID, err := uuid.Parse(todoIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse(
			"INVALID_TODO_ID",
			"Invalid todo ID format",
			nil,
		))
		return
	}

	// Parse include_related query parameter
	includeRelated := c.DefaultQuery("include_related", "false") == "true"

	// Get todo
	todo, err := h.todoService.GetTodo(c.Request.Context(), todoID, userID, includeRelated)
	if err != nil {
		statusCode := http.StatusInternalServerError
		errorCode := "TODO_RETRIEVAL_FAILED"

		if strings.Contains(err.Error(), "not found") {
			statusCode = http.StatusNotFound
			errorCode = "TODO_NOT_FOUND"
		}

		c.JSON(statusCode, models.NewErrorResponse(
			errorCode,
			err.Error(),
			nil,
		))
		return
	}

	c.JSON(http.StatusOK, models.NewAPIResponse(
		"Todo retrieved successfully",
		todo,
	))
}

// GetUserTodos handles retrieving todos for a user with filtering
// @Summary Get user todos
// @Description Get todos for the authenticated user with optional filtering
// @Tags Todos
// @Produce json
// @Security BearerAuth
// @Param project_id query string false "Filter by project ID"
// @Param status query string false "Filter by status" Enums(not_started,in_progress,pending,completed,cancelled,on_hold)
// @Param priority query string false "Filter by priority" Enums(low,medium,high,urgent)
// @Param assigned_to query string false "Filter by assigned user ID"
// @Param due_before query string false "Filter by due date before (RFC3339 format)"
// @Param due_after query string false "Filter by due date after (RFC3339 format)"
// @Param tags query string false "Filter by tags (comma-separated)"
// @Param is_completed query boolean false "Filter by completion status"
// @Param is_pinned query boolean false "Filter by pinned status"
// @Param include_archived query boolean false "Include archived todos"
// @Param include_related query boolean false "Include related data"
// @Param limit query integer false "Limit number of results" default(20)
// @Param offset query integer false "Offset for pagination" default(0)
// @Param sort_by query string false "Sort field" Enums(created_at,updated_at,due_date,priority,status,task_name) default(created_at)
// @Param sort_order query string false "Sort order" Enums(asc,desc) default(desc)
// @Success 200 {object} models.APIResponse{data=map[string]interface{}} "Todos retrieved successfully"
// @Failure 400 {object} models.APIResponse "Invalid request"
// @Failure 401 {object} models.APIResponse "Unauthorized"
// @Failure 500 {object} models.APIResponse "Internal server error"
// @Router /todos [get]
func (h *TodoHandler) GetUserTodos(c *gin.Context) {
	// Get user ID from context
	userID, _ := middleware.GetUserID(c)

	// Parse filters from query parameters
	filters, err := h.parseTodoFilters(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse(
			"INVALID_FILTERS",
			err.Error(),
			nil,
		))
		return
	}

	// Get todos
	todos, total, err := h.todoService.GetUserTodos(c.Request.Context(), userID, filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.NewErrorResponse(
			"TODO_RETRIEVAL_FAILED",
			err.Error(),
			nil,
		))
		return
	}

	result := map[string]interface{}{
		"todos":  todos,
		"total":  total,
		"limit":  filters.Limit,
		"offset": filters.Offset,
	}

	c.JSON(http.StatusOK, models.NewAPIResponse(
		"Todos retrieved successfully",
		result,
	))
}

// GetProjectTodos handles retrieving todos for a specific project
// @Summary Get project todos
// @Description Get todos for a specific project
// @Tags Todos
// @Produce json
// @Security BearerAuth
// @Param project_id path string true "Project ID"
// @Param include_completed query boolean false "Include completed todos"
// @Success 200 {object} models.APIResponse{data=[]models.Todo} "Project todos retrieved successfully"
// @Failure 400 {object} models.APIResponse "Invalid project ID"
// @Failure 401 {object} models.APIResponse "Unauthorized"
// @Failure 404 {object} models.APIResponse "Project not found"
// @Failure 500 {object} models.APIResponse "Internal server error"
// @Router /projects/{project_id}/todos [get]
func (h *TodoHandler) GetProjectTodos(c *gin.Context) {
	// Get user ID from context
	userID, _ := middleware.GetUserID(c)

	// Parse project ID
	projectIDStr := c.Param("id")
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse(
			"INVALID_PROJECT_ID",
			"Invalid project ID format",
			nil,
		))
		return
	}

	// Parse include_completed query parameter
	includeCompleted := c.DefaultQuery("include_completed", "false") == "true"

	// Get project todos
	todos, err := h.todoService.GetProjectTodos(c.Request.Context(), projectID, userID, includeCompleted)
	if err != nil {
		statusCode := http.StatusInternalServerError
		errorCode := "TODO_RETRIEVAL_FAILED"

		if strings.Contains(err.Error(), "not found") {
			statusCode = http.StatusNotFound
			errorCode = "PROJECT_NOT_FOUND"
		}

		c.JSON(statusCode, models.NewErrorResponse(
			errorCode,
			err.Error(),
			nil,
		))
		return
	}

	c.JSON(http.StatusOK, models.NewAPIResponse(
		"Project todos retrieved successfully",
		todos,
	))
}

// UpdateTodo handles todo updates
// @Summary Update todo
// @Description Update a todo for the authenticated user
// @Tags Todos
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Todo ID"
// @Param request body models.UpdateTodoRequest true "Todo update details"
// @Success 200 {object} models.APIResponse{data=models.Todo} "Todo updated successfully"
// @Failure 400 {object} models.APIResponse "Invalid request"
// @Failure 401 {object} models.APIResponse "Unauthorized"
// @Failure 404 {object} models.APIResponse "Todo not found"
// @Failure 500 {object} models.APIResponse "Internal server error"
// @Router /todos/{id} [put]
func (h *TodoHandler) UpdateTodo(c *gin.Context) {
	// Get user ID from context
	userID, _ := middleware.GetUserID(c)

	// Parse todo ID
	todoIDStr := c.Param("id")
	todoID, err := uuid.Parse(todoIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse(
			"INVALID_TODO_ID",
			"Invalid todo ID format",
			nil,
		))
		return
	}

	var req models.UpdateTodoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse(
			"INVALID_REQUEST",
			"Invalid request data",
			map[string]string{"details": err.Error()},
		))
		return
	}

	// Update todo
	todo, err := h.todoService.UpdateTodo(c.Request.Context(), todoID, userID, &req)
	if err != nil {
		statusCode := http.StatusInternalServerError
		errorCode := "TODO_UPDATE_FAILED"

		if strings.Contains(err.Error(), "validation failed") {
			statusCode = http.StatusBadRequest
			errorCode = "VALIDATION_ERROR"
		} else if strings.Contains(err.Error(), "not found") {
			statusCode = http.StatusNotFound
			errorCode = "TODO_NOT_FOUND"
		}

		c.JSON(statusCode, models.NewErrorResponse(
			errorCode,
			err.Error(),
			nil,
		))
		return
	}

	c.JSON(http.StatusOK, models.NewAPIResponse(
		"Todo updated successfully",
		todo,
	))
}

// ArchiveTodo handles todo archiving
// @Summary Archive todo
// @Description Archive a todo for the authenticated user
// @Tags Todos
// @Produce json
// @Security BearerAuth
// @Param id path string true "Todo ID"
// @Success 200 {object} models.APIResponse "Todo archived successfully"
// @Failure 400 {object} models.APIResponse "Invalid todo ID"
// @Failure 401 {object} models.APIResponse "Unauthorized"
// @Failure 404 {object} models.APIResponse "Todo not found"
// @Failure 500 {object} models.APIResponse "Internal server error"
// @Router /todos/{id}/archive [post]
func (h *TodoHandler) ArchiveTodo(c *gin.Context) {
	// Get user ID from context
	userID, _ := middleware.GetUserID(c)

	// Parse todo ID
	todoIDStr := c.Param("id")
	todoID, err := uuid.Parse(todoIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse(
			"INVALID_TODO_ID",
			"Invalid todo ID format",
			nil,
		))
		return
	}

	// Archive todo
	err = h.todoService.ArchiveTodo(c.Request.Context(), todoID, userID)
	if err != nil {
		statusCode := http.StatusInternalServerError
		errorCode := "TODO_ARCHIVE_FAILED"

		if strings.Contains(err.Error(), "not found") {
			statusCode = http.StatusNotFound
			errorCode = "TODO_NOT_FOUND"
		}

		c.JSON(statusCode, models.NewErrorResponse(
			errorCode,
			err.Error(),
			nil,
		))
		return
	}

	c.JSON(http.StatusOK, models.NewAPIResponse(
		"Todo archived successfully",
		nil,
	))
}

// DeleteTodo handles todo deletion
// @Summary Delete todo
// @Description Permanently delete a todo for the authenticated user
// @Tags Todos
// @Produce json
// @Security BearerAuth
// @Param id path string true "Todo ID"
// @Success 200 {object} models.APIResponse "Todo deleted successfully"
// @Failure 400 {object} models.APIResponse "Invalid todo ID"
// @Failure 401 {object} models.APIResponse "Unauthorized"
// @Failure 404 {object} models.APIResponse "Todo not found"
// @Failure 500 {object} models.APIResponse "Internal server error"
// @Router /todos/{id} [delete]
func (h *TodoHandler) DeleteTodo(c *gin.Context) {
	// Get user ID from context
	userID, _ := middleware.GetUserID(c)

	// Parse todo ID
	todoIDStr := c.Param("id")
	todoID, err := uuid.Parse(todoIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse(
			"INVALID_TODO_ID",
			"Invalid todo ID format",
			nil,
		))
		return
	}

	// Delete todo
	err = h.todoService.DeleteTodo(c.Request.Context(), todoID, userID)
	if err != nil {
		statusCode := http.StatusInternalServerError
		errorCode := "TODO_DELETE_FAILED"

		if strings.Contains(err.Error(), "not found") {
			statusCode = http.StatusNotFound
			errorCode = "TODO_NOT_FOUND"
		}

		c.JSON(statusCode, models.NewErrorResponse(
			errorCode,
			err.Error(),
			nil,
		))
		return
	}

	c.JSON(http.StatusOK, models.NewAPIResponse(
		"Todo deleted successfully",
		nil,
	))
}

// GetTodoDashboard handles retrieving todo dashboard data
// @Summary Get todo dashboard
// @Description Get todo dashboard data for the authenticated user
// @Tags Todos
// @Produce json
// @Security BearerAuth
// @Param limit query integer false "Limit number of results" default(10)
// @Success 200 {object} models.APIResponse{data=[]models.TodoDashboard} "Todo dashboard retrieved successfully"
// @Failure 401 {object} models.APIResponse "Unauthorized"
// @Failure 500 {object} models.APIResponse "Internal server error"
// @Router /todos/dashboard [get]
func (h *TodoHandler) GetTodoDashboard(c *gin.Context) {
	// Get user ID from context
	userID, _ := middleware.GetUserID(c)

	// Parse limit query parameter
	limitStr := c.DefaultQuery("limit", "10")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 10
	}
	if limit > 50 {
		limit = 50 // Cap at 50
	}

	// Get todo dashboard
	dashboard, err := h.todoService.GetTodoDashboard(c.Request.Context(), userID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.NewErrorResponse(
			"DASHBOARD_RETRIEVAL_FAILED",
			err.Error(),
			nil,
		))
		return
	}

	c.JSON(http.StatusOK, models.NewAPIResponse(
		"Todo dashboard retrieved successfully",
		dashboard,
	))
}

// GetOverdueTodos handles retrieving overdue todos
// @Summary Get overdue todos
// @Description Get overdue todos for the authenticated user
// @Tags Todos
// @Produce json
// @Security BearerAuth
// @Success 200 {object} models.APIResponse{data=[]models.Todo} "Overdue todos retrieved successfully"
// @Failure 401 {object} models.APIResponse "Unauthorized"
// @Failure 500 {object} models.APIResponse "Internal server error"
// @Router /todos/overdue [get]
func (h *TodoHandler) GetOverdueTodos(c *gin.Context) {
	// Get user ID from context
	userID, _ := middleware.GetUserID(c)

	// Get overdue todos
	todos, err := h.todoService.GetOverdueTodos(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.NewErrorResponse(
			"OVERDUE_TODOS_RETRIEVAL_FAILED",
			err.Error(),
			nil,
		))
		return
	}

	c.JSON(http.StatusOK, models.NewAPIResponse(
		"Overdue todos retrieved successfully",
		todos,
	))
}

// SearchTodos handles todo search
// @Summary Search todos
// @Description Search todos by text for the authenticated user
// @Tags Todos
// @Produce json
// @Security BearerAuth
// @Param q query string true "Search query"
// @Param limit query integer false "Limit number of results" default(20)
// @Param offset query integer false "Offset for pagination" default(0)
// @Success 200 {object} models.APIResponse{data=map[string]interface{}} "Todos search results"
// @Failure 400 {object} models.APIResponse "Invalid request"
// @Failure 401 {object} models.APIResponse "Unauthorized"
// @Failure 500 {object} models.APIResponse "Internal server error"
// @Router /todos/search [get]
func (h *TodoHandler) SearchTodos(c *gin.Context) {
	// Get user ID from context
	userID, _ := middleware.GetUserID(c)

	// Parse query parameters
	searchTerm := c.Query("q")
	if searchTerm == "" {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse(
			"MISSING_SEARCH_TERM",
			"Search query is required",
			nil,
		))
		return
	}

	limitStr := c.DefaultQuery("limit", "20")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100 // Cap at 100
	}

	offsetStr := c.DefaultQuery("offset", "0")
	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = 0
	}

	// Search todos
	todos, total, err := h.todoService.SearchTodos(c.Request.Context(), userID, searchTerm, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.NewErrorResponse(
			"SEARCH_FAILED",
			err.Error(),
			nil,
		))
		return
	}

	result := map[string]interface{}{
		"todos":  todos,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	}

	c.JSON(http.StatusOK, models.NewAPIResponse(
		"Todos search completed successfully",
		result,
	))
}

// MarkTodoComplete handles marking a todo as complete
// @Summary Mark todo as complete
// @Description Mark a todo as completed for the authenticated user
// @Tags Todos
// @Produce json
// @Security BearerAuth
// @Param id path string true "Todo ID"
// @Success 200 {object} models.APIResponse "Todo marked as complete successfully"
// @Failure 400 {object} models.APIResponse "Invalid todo ID"
// @Failure 401 {object} models.APIResponse "Unauthorized"
// @Failure 404 {object} models.APIResponse "Todo not found"
// @Failure 500 {object} models.APIResponse "Internal server error"
// @Router /todos/{id}/complete [post]
func (h *TodoHandler) MarkTodoComplete(c *gin.Context) {
	// Get user ID from context
	userID, _ := middleware.GetUserID(c)

	// Parse todo ID
	todoIDStr := c.Param("id")
	todoID, err := uuid.Parse(todoIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse(
			"INVALID_TODO_ID",
			"Invalid todo ID format",
			nil,
		))
		return
	}

	// Mark todo as complete
	err = h.todoService.MarkTodoComplete(c.Request.Context(), todoID, userID)
	if err != nil {
		statusCode := http.StatusInternalServerError
		errorCode := "TODO_COMPLETE_FAILED"

		if strings.Contains(err.Error(), "not found") {
			statusCode = http.StatusNotFound
			errorCode = "TODO_NOT_FOUND"
		}

		c.JSON(statusCode, models.NewErrorResponse(
			errorCode,
			err.Error(),
			nil,
		))
		return
	}

	c.JSON(http.StatusOK, models.NewAPIResponse(
		"Todo marked as complete successfully",
		nil,
	))
}

// MarkTodoIncomplete handles marking a todo as incomplete
// @Summary Mark todo as incomplete
// @Description Mark a todo as incomplete for the authenticated user
// @Tags Todos
// @Produce json
// @Security BearerAuth
// @Param id path string true "Todo ID"
// @Success 200 {object} models.APIResponse "Todo marked as incomplete successfully"
// @Failure 400 {object} models.APIResponse "Invalid todo ID"
// @Failure 401 {object} models.APIResponse "Unauthorized"
// @Failure 404 {object} models.APIResponse "Todo not found"
// @Failure 500 {object} models.APIResponse "Internal server error"
// @Router /todos/{id}/incomplete [post]
func (h *TodoHandler) MarkTodoIncomplete(c *gin.Context) {
	// Get user ID from context
	userID, _ := middleware.GetUserID(c)

	// Parse todo ID
	todoIDStr := c.Param("id")
	todoID, err := uuid.Parse(todoIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse(
			"INVALID_TODO_ID",
			"Invalid todo ID format",
			nil,
		))
		return
	}

	// Mark todo as incomplete
	err = h.todoService.MarkTodoIncomplete(c.Request.Context(), todoID, userID)
	if err != nil {
		statusCode := http.StatusInternalServerError
		errorCode := "TODO_INCOMPLETE_FAILED"

		if strings.Contains(err.Error(), "not found") {
			statusCode = http.StatusNotFound
			errorCode = "TODO_NOT_FOUND"
		}

		c.JSON(statusCode, models.NewErrorResponse(
			errorCode,
			err.Error(),
			nil,
		))
		return
	}

	c.JSON(http.StatusOK, models.NewAPIResponse(
		"Todo marked as incomplete successfully",
		nil,
	))
}

// PinTodo handles pinning/unpinning a todo
// @Summary Pin/unpin todo
// @Description Pin or unpin a todo for the authenticated user
// @Tags Todos
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Todo ID"
// @Param request body map[string]bool true "Pin status" example({"pinned": true})
// @Success 200 {object} models.APIResponse "Todo pin status updated successfully"
// @Failure 400 {object} models.APIResponse "Invalid request"
// @Failure 401 {object} models.APIResponse "Unauthorized"
// @Failure 404 {object} models.APIResponse "Todo not found"
// @Failure 500 {object} models.APIResponse "Internal server error"
// @Router /todos/{id}/pin [post]
func (h *TodoHandler) PinTodo(c *gin.Context) {
	// Get user ID from context
	userID, _ := middleware.GetUserID(c)

	// Parse todo ID
	todoIDStr := c.Param("id")
	todoID, err := uuid.Parse(todoIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse(
			"INVALID_TODO_ID",
			"Invalid todo ID format",
			nil,
		))
		return
	}

	var req map[string]bool
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse(
			"INVALID_REQUEST",
			"Invalid request data",
			map[string]string{"details": err.Error()},
		))
		return
	}

	pinned := req["pinned"]

	// Pin/unpin todo
	err = h.todoService.PinTodo(c.Request.Context(), todoID, userID, pinned)
	if err != nil {
		statusCode := http.StatusInternalServerError
		errorCode := "TODO_PIN_FAILED"

		if strings.Contains(err.Error(), "not found") {
			statusCode = http.StatusNotFound
			errorCode = "TODO_NOT_FOUND"
		}

		c.JSON(statusCode, models.NewErrorResponse(
			errorCode,
			err.Error(),
			nil,
		))
		return
	}

	message := "Todo unpinned successfully"
	if pinned {
		message = "Todo pinned successfully"
	}

	c.JSON(http.StatusOK, models.NewAPIResponse(
		message,
		nil,
	))
}

// Helper function to parse todo filters from query parameters
func (h *TodoHandler) parseTodoFilters(c *gin.Context) (repository.TodoFilters, error) {
	filters := repository.TodoFilters{
		Limit:     20,
		Offset:    0,
		SortBy:    "created_at",
		SortOrder: "desc",
	}

	// Parse project_id
	if projectIDStr := c.Query("project_id"); projectIDStr != "" {
		projectID, err := uuid.Parse(projectIDStr)
		if err != nil {
			return filters, fmt.Errorf("invalid project_id format")
		}
		filters.ProjectID = &projectID
	}

	// Parse status
	if statusStr := c.Query("status"); statusStr != "" {
		status := models.TaskStatus(statusStr)
		if !status.IsValid() {
			return filters, fmt.Errorf("invalid status value")
		}
		filters.Status = &status
	}

	// Parse priority
	if priorityStr := c.Query("priority"); priorityStr != "" {
		priority := models.PriorityLevel(priorityStr)
		if !priority.IsValid() {
			return filters, fmt.Errorf("invalid priority value")
		}
		filters.Priority = &priority
	}

	// Parse assigned_to
	if assignedToStr := c.Query("assigned_to"); assignedToStr != "" {
		assignedTo, err := uuid.Parse(assignedToStr)
		if err != nil {
			return filters, fmt.Errorf("invalid assigned_to format")
		}
		filters.AssignedTo = &assignedTo
	}

	// Parse due_before
	if dueBefore := c.Query("due_before"); dueBefore != "" {
		// Here you would parse the date string (RFC3339 format)
		// For simplicity, we'll skip the actual parsing
		// In production, use time.Parse with proper format
	}

	// Parse due_after
	if dueAfter := c.Query("due_after"); dueAfter != "" {
		// Similar to due_before
	}

	// Parse tags
	if tagsStr := c.Query("tags"); tagsStr != "" {
		tags := strings.Split(tagsStr, ",")
		filters.Tags = tags
	}

	// Parse boolean filters
	if isCompletedStr := c.Query("is_completed"); isCompletedStr != "" {
		isCompleted := isCompletedStr == "true"
		filters.IsCompleted = &isCompleted
	}

	if isPinnedStr := c.Query("is_pinned"); isPinnedStr != "" {
		isPinned := isPinnedStr == "true"
		filters.IsPinned = &isPinned
	}

	// Parse include_archived
	filters.IncludeArchived = c.DefaultQuery("include_archived", "false") == "true"

	// Parse include_related
	filters.IncludeRelated = c.DefaultQuery("include_related", "false") == "true"

	// Parse limit
	if limitStr := c.Query("limit"); limitStr != "" {
		limit, err := strconv.Atoi(limitStr)
		if err != nil || limit <= 0 {
			return filters, fmt.Errorf("invalid limit value")
		}
		if limit > 100 {
			limit = 100 // Cap at 100
		}
		filters.Limit = limit
	}

	// Parse offset
	if offsetStr := c.Query("offset"); offsetStr != "" {
		offset, err := strconv.Atoi(offsetStr)
		if err != nil || offset < 0 {
			return filters, fmt.Errorf("invalid offset value")
		}
		filters.Offset = offset
	}

	// Parse sort_by
	if sortBy := c.Query("sort_by"); sortBy != "" {
		validSortFields := []string{"created_at", "updated_at", "due_date", "priority", "status", "task_name"}
		isValid := false
		for _, field := range validSortFields {
			if sortBy == field {
				isValid = true
				break
			}
		}
		if !isValid {
			return filters, fmt.Errorf("invalid sort_by field")
		}
		filters.SortBy = sortBy
	}

	// Parse sort_order
	if sortOrder := c.Query("sort_order"); sortOrder != "" {
		if sortOrder != "asc" && sortOrder != "desc" {
			return filters, fmt.Errorf("invalid sort_order value")
		}
		filters.SortOrder = sortOrder
	}

	return filters, nil
}

// Kanban Board Methods

// GetProjectKanbanBoard handles retrieving kanban board for a project
// @Summary Get project kanban board
// @Description Get todos organized in kanban columns for a specific project
// @Tags Kanban
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Project ID"
// @Success 200 {object} models.APIResponse{data=models.KanbanBoardResponse} "Kanban board retrieved successfully"
// @Failure 400 {object} models.APIResponse "Invalid project ID"
// @Failure 401 {object} models.APIResponse "Unauthorized"
// @Failure 404 {object} models.APIResponse "Project not found"
// @Failure 500 {object} models.APIResponse "Internal server error"
// @Router /projects/{id}/kanban [get]
func (h *TodoHandler) GetProjectKanbanBoard(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	// Parse project ID
	projectIDStr := c.Param("id")
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse(
			"INVALID_PROJECT_ID",
			"Invalid project ID format",
			nil,
		))
		return
	}

	// Get kanban board data
	board, err := h.todoService.GetProjectKanbanBoard(c.Request.Context(), projectID, userID)
	if err != nil {
		if strings.Contains(err.Error(), "project not found") {
			c.JSON(http.StatusNotFound, models.NewErrorResponse(
				"PROJECT_NOT_FOUND",
				"Project not found",
				nil,
			))
			return
		}
		c.JSON(http.StatusInternalServerError, models.NewErrorResponse(
			"INTERNAL_ERROR",
			fmt.Sprintf("Failed to retrieve kanban board: %v", err),
			nil,
		))
		return
	}

	c.JSON(http.StatusOK, models.NewAPIResponse(
		"Kanban board retrieved successfully",
		board,
	))
}

// MoveTodoToColumn handles moving a todo to a different column/status
// @Summary Move todo to column
// @Description Move a todo to a different kanban column (status)
// @Tags Kanban
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body models.MoveTodoRequest true "Move todo request"
// @Success 200 {object} models.APIResponse{data=models.TodoMoveResponse} "Todo moved successfully"
// @Failure 400 {object} models.APIResponse "Invalid request"
// @Failure 401 {object} models.APIResponse "Unauthorized"
// @Failure 404 {object} models.APIResponse "Todo not found"
// @Failure 500 {object} models.APIResponse "Internal server error"
// @Router /todos/move [post]
func (h *TodoHandler) MoveTodoToColumn(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	var req models.MoveTodoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse(
			"INVALID_REQUEST",
			"Invalid request format",
			nil,
		))
		return
	}

	// Move todo
	response, err := h.todoService.MoveTodoToColumn(c.Request.Context(), userID, &req)
	if err != nil {
		if err == repository.ErrTodoNotFound {
			c.JSON(http.StatusNotFound, models.NewErrorResponse(
				"TODO_NOT_FOUND",
				"Todo not found",
				nil,
			))
			return
		}
		c.JSON(http.StatusInternalServerError, models.NewErrorResponse(
			"INTERNAL_ERROR",
			"Failed to move todo",
			nil,
		))
		return
	}

	c.JSON(http.StatusOK, models.NewAPIResponse(
		"Todo moved successfully",
		response,
	))
}

// ReorderTodosInColumn handles reordering todos within the same column
// @Summary Reorder todos in column
// @Description Reorder todos within the same kanban column
// @Tags Kanban
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body models.ReorderTodosRequest true "Reorder todos request"
// @Success 200 {object} models.APIResponse{data=models.ReorderResponse} "Todos reordered successfully"
// @Failure 400 {object} models.APIResponse "Invalid request"
// @Failure 401 {object} models.APIResponse "Unauthorized"
// @Failure 500 {object} models.APIResponse "Internal server error"
// @Router /todos/reorder [post]
func (h *TodoHandler) ReorderTodosInColumn(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	var req models.ReorderTodosRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse(
			"INVALID_REQUEST",
			"Invalid request format",
			nil,
		))
		return
	}

	// Reorder todos
	response, err := h.todoService.ReorderTodosInColumn(c.Request.Context(), userID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.NewErrorResponse(
			"INTERNAL_ERROR",
			"Failed to reorder todos",
			nil,
		))
		return
	}

	c.JSON(http.StatusOK, models.NewAPIResponse(
		"Todos reordered successfully",
		response,
	))
}

// BulkMoveTodos handles moving multiple todos at once
// @Summary Bulk move todos
// @Description Move multiple todos to different columns in a single operation
// @Tags Kanban
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body models.BulkMoveTodosRequest true "Bulk move request"
// @Success 200 {object} models.APIResponse{data=models.BulkMoveResponse} "Todos moved successfully"
// @Failure 400 {object} models.APIResponse "Invalid request"
// @Failure 401 {object} models.APIResponse "Unauthorized"
// @Failure 500 {object} models.APIResponse "Internal server error"
// @Router /todos/bulk-move [post]
func (h *TodoHandler) BulkMoveTodos(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	var req models.BulkMoveTodosRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse(
			"INVALID_REQUEST",
			"Invalid request format",
			nil,
		))
		return
	}

	// Bulk move todos
	response, err := h.todoService.BulkMoveTodos(c.Request.Context(), userID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.NewErrorResponse(
			"INTERNAL_ERROR",
			"Failed to bulk move todos",
			nil,
		))
		return
	}

	c.JSON(http.StatusOK, models.NewAPIResponse(
		"Bulk move completed",
		response,
	))
}
