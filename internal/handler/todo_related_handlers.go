package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"macwrite-auth-api/internal/middleware"
	"macwrite-auth-api/internal/models"
	"macwrite-auth-api/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ========================================
// SUBTASK HANDLER
// ========================================

// SubtaskHandler handles subtask HTTP requests
type SubtaskHandler struct {
	subtaskService *service.SubtaskService
}

// NewSubtaskHandler creates a new subtask handler
func NewSubtaskHandler(subtaskService *service.SubtaskService) *SubtaskHandler {
	return &SubtaskHandler{
		subtaskService: subtaskService,
	}
}

// CreateSubtask handles subtask creation
// @Summary Create a new subtask
// @Description Create a new subtask for a todo
// @Tags Subtasks
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param todo_id path string true "Todo ID"
// @Param request body models.Subtask true "Subtask details"
// @Success 201 {object} models.APIResponse{data=models.Subtask} "Subtask created successfully"
// @Failure 400 {object} models.APIResponse "Invalid request"
// @Failure 401 {object} models.APIResponse "Unauthorized"
// @Failure 404 {object} models.APIResponse "Todo not found"
// @Failure 500 {object} models.APIResponse "Internal server error"
// @Router /todos/{todo_id}/subtasks [post]
func (h *SubtaskHandler) CreateSubtask(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	todoIDStr := c.Param("id")
	todoID, err := uuid.Parse(todoIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse("INVALID_TODO_ID", "Invalid todo ID format", nil))
		return
	}

	var subtask models.Subtask
	if err := c.ShouldBindJSON(&subtask); err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse("INVALID_REQUEST", "Invalid request data", map[string]string{"details": err.Error()}))
		return
	}

	subtask.TodoID = todoID

	createdSubtask, err := h.subtaskService.CreateSubtask(c.Request.Context(), userID, &subtask)
	if err != nil {
		statusCode := http.StatusInternalServerError
		errorCode := "SUBTASK_CREATION_FAILED"

		if strings.Contains(err.Error(), "not found") {
			statusCode = http.StatusNotFound
			errorCode = "TODO_NOT_FOUND"
		} else if strings.Contains(err.Error(), "validation failed") {
			statusCode = http.StatusBadRequest
			errorCode = "VALIDATION_ERROR"
		}

		c.JSON(statusCode, models.NewErrorResponse(errorCode, err.Error(), nil))
		return
	}

	c.JSON(http.StatusCreated, models.NewAPIResponse("Subtask created successfully", createdSubtask))
}

// GetSubtasks handles retrieving subtasks for a todo
// @Summary Get subtasks for a todo
// @Description Get all subtasks for a specific todo
// @Tags Subtasks
// @Produce json
// @Security BearerAuth
// @Param todo_id path string true "Todo ID"
// @Param include_archived query boolean false "Include archived subtasks"
// @Success 200 {object} models.APIResponse{data=[]models.Subtask} "Subtasks retrieved successfully"
// @Failure 400 {object} models.APIResponse "Invalid todo ID"
// @Failure 401 {object} models.APIResponse "Unauthorized"
// @Failure 404 {object} models.APIResponse "Todo not found"
// @Failure 500 {object} models.APIResponse "Internal server error"
// @Router /todos/{todo_id}/subtasks [get]
func (h *SubtaskHandler) GetSubtasks(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	todoIDStr := c.Param("id")
	todoID, err := uuid.Parse(todoIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse("INVALID_TODO_ID", "Invalid todo ID format", nil))
		return
	}

	includeArchived := c.DefaultQuery("include_archived", "false") == "true"

	subtasks, err := h.subtaskService.GetSubtasksByTodoID(c.Request.Context(), todoID, userID, includeArchived)
	if err != nil {
		statusCode := http.StatusInternalServerError
		errorCode := "SUBTASK_RETRIEVAL_FAILED"

		if strings.Contains(err.Error(), "not found") {
			statusCode = http.StatusNotFound
			errorCode = "TODO_NOT_FOUND"
		}

		c.JSON(statusCode, models.NewErrorResponse(errorCode, err.Error(), nil))
		return
	}

	c.JSON(http.StatusOK, models.NewAPIResponse("Subtasks retrieved successfully", subtasks))
}

// GetSubtasksByProjectID handles retrieving all subtasks for a project
// @Summary Get all subtasks for a project
// @Description Get all subtasks for a specific project with filtering, sorting, and pagination
// @Tags Subtasks
// @Produce json
// @Security BearerAuth
// @Param project_id path string true "Project ID"
// @Param include_archived query boolean false "Include archived subtasks"
// @Param status query string false "Filter by status"
// @Param priority query string false "Filter by priority"
// @Param search query string false "Search in subtask name, description, or todo title"
// @Param sort_by query string false "Sort field (name, status, priority, due_date, todo_title, sort_order, created_at)"
// @Param sort_order query string false "Sort order (ASC or DESC)"
// @Param page query int false "Page number (default: 1)"
// @Param limit query int false "Items per page (default: 20, max: 100)"
// @Success 200 {object} models.APIResponse{data=service.GetSubtasksByProjectIDResponse} "Subtasks retrieved successfully"
// @Failure 400 {object} models.APIResponse "Invalid request parameters"
// @Failure 401 {object} models.APIResponse "Unauthorized"
// @Failure 403 {object} models.APIResponse "Access denied"
// @Failure 404 {object} models.APIResponse "Project not found"
// @Failure 500 {object} models.APIResponse "Internal server error"
// @Router /projects/{project_id}/subtasks [get]
func (h *SubtaskHandler) GetSubtasksByProjectID(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	// Parse project ID from URL parameter
	projectIDStr := c.Param("id")
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse("INVALID_PROJECT_ID", "Invalid project ID format", nil))
		return
	}

	// Parse query parameters
	includeArchived := c.DefaultQuery("include_archived", "false") == "true"
	status := c.Query("status")
	priority := c.Query("priority")
	search := c.Query("search")
	sortBy := c.DefaultQuery("sort_by", "created_at")
	sortOrder := c.DefaultQuery("sort_order", "DESC")

	// Parse pagination parameters
	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil || page < 1 {
		page = 1
	}

	limit, err := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if err != nil || limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	// Build request object
	req := &service.GetSubtasksByProjectIDRequest{
		ProjectID:       projectID,
		IncludeArchived: includeArchived,
		Status:          status,
		Priority:        priority,
		Search:          search,
		SortBy:          sortBy,
		SortOrder:       sortOrder,
		Page:            page,
		Limit:           limit,
	}

	// Call service method
	response, err := h.subtaskService.GetSubtasksByProjectID(c.Request.Context(), userID, req)
	if err != nil {
		statusCode := http.StatusInternalServerError
		errorCode := "SUBTASK_RETRIEVAL_FAILED"

		if strings.Contains(err.Error(), "validation failed") {
			statusCode = http.StatusBadRequest
			errorCode = "VALIDATION_ERROR"
		} else if strings.Contains(err.Error(), "project not found") || strings.Contains(err.Error(), "access denied") {
			statusCode = http.StatusNotFound
			errorCode = "PROJECT_NOT_FOUND"
		}

		c.JSON(statusCode, models.NewErrorResponse(errorCode, err.Error(), nil))
		return
	}

	c.JSON(http.StatusOK, models.NewAPIResponse(
		"Project subtasks retrieved successfully",
		map[string]interface{}{
			"subtasks":   response.Subtasks,
			"pagination": response.Pagination,
			"filters": map[string]interface{}{
				"include_archived": includeArchived,
				"status":           status,
				"priority":         priority,
				"search":           search,
			},
			"sorting": map[string]interface{}{
				"sort_by":    sortBy,
				"sort_order": sortOrder,
			},
		},
	))
}

// UpdateSubtask handles subtask updates
// @Summary Update subtask
// @Description Update a subtask
// @Tags Subtasks
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Subtask ID"
// @Param request body map[string]interface{} true "Subtask updates"
// @Success 200 {object} models.APIResponse{data=models.Subtask} "Subtask updated successfully"
// @Failure 400 {object} models.APIResponse "Invalid request"
// @Failure 401 {object} models.APIResponse "Unauthorized"
// @Failure 404 {object} models.APIResponse "Subtask not found"
// @Failure 500 {object} models.APIResponse "Internal server error"
// @Router /subtasks/{id} [put]
func (h *SubtaskHandler) UpdateSubtask(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	subtaskIDStr := c.Param("id")
	subtaskID, err := uuid.Parse(subtaskIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse("INVALID_SUBTASK_ID", "Invalid subtask ID format", nil))
		return
	}

	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse("INVALID_REQUEST", "Invalid request data", map[string]string{"details": err.Error()}))
		return
	}

	subtask, err := h.subtaskService.UpdateSubtask(c.Request.Context(), subtaskID, userID, updates)
	if err != nil {
		statusCode := http.StatusInternalServerError
		errorCode := "SUBTASK_UPDATE_FAILED"

		if strings.Contains(err.Error(), "not found") {
			statusCode = http.StatusNotFound
			errorCode = "SUBTASK_NOT_FOUND"
		}

		c.JSON(statusCode, models.NewErrorResponse(errorCode, err.Error(), nil))
		return
	}

	c.JSON(http.StatusOK, models.NewAPIResponse("Subtask updated successfully", subtask))
}

// DeleteSubtask handles subtask deletion
// @Summary Delete subtask
// @Description Delete a subtask
// @Tags Subtasks
// @Produce json
// @Security BearerAuth
// @Param id path string true "Subtask ID"
// @Success 200 {object} models.APIResponse "Subtask deleted successfully"
// @Failure 400 {object} models.APIResponse "Invalid subtask ID"
// @Failure 401 {object} models.APIResponse "Unauthorized"
// @Failure 404 {object} models.APIResponse "Subtask not found"
// @Failure 500 {object} models.APIResponse "Internal server error"
// @Router /subtasks/{id} [delete]
func (h *SubtaskHandler) DeleteSubtask(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	subtaskIDStr := c.Param("id")
	subtaskID, err := uuid.Parse(subtaskIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse("INVALID_SUBTASK_ID", "Invalid subtask ID format", nil))
		return
	}

	err = h.subtaskService.DeleteSubtask(c.Request.Context(), subtaskID, userID)
	if err != nil {
		statusCode := http.StatusInternalServerError
		errorCode := "SUBTASK_DELETE_FAILED"

		if strings.Contains(err.Error(), "not found") {
			statusCode = http.StatusNotFound
			errorCode = "SUBTASK_NOT_FOUND"
		}

		c.JSON(statusCode, models.NewErrorResponse(errorCode, err.Error(), nil))
		return
	}

	c.JSON(http.StatusOK, models.NewAPIResponse("Subtask deleted successfully", nil))
}

// ReorderSubtasks handles subtask reordering
// @Summary Reorder subtasks
// @Description Reorder subtasks for a todo
// @Tags Subtasks
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param todo_id path string true "Todo ID"
// @Param request body map[string][]string true "Subtask IDs in new order" example({"subtask_ids": ["uuid1", "uuid2", "uuid3"]})
// @Success 200 {object} models.APIResponse "Subtasks reordered successfully"
// @Failure 400 {object} models.APIResponse "Invalid request"
// @Failure 401 {object} models.APIResponse "Unauthorized"
// @Failure 404 {object} models.APIResponse "Todo not found"
// @Failure 500 {object} models.APIResponse "Internal server error"
// @Router /todos/{todo_id}/subtasks/reorder [post]
func (h *SubtaskHandler) ReorderSubtasks(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	todoIDStr := c.Param("todo_id")
	todoID, err := uuid.Parse(todoIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse("INVALID_TODO_ID", "Invalid todo ID format", nil))
		return
	}

	var req map[string][]string
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse("INVALID_REQUEST", "Invalid request data", map[string]string{"details": err.Error()}))
		return
	}

	subtaskIDStrs, exists := req["subtask_ids"]
	if !exists {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse("MISSING_SUBTASK_IDS", "Subtask IDs are required", nil))
		return
	}

	var subtaskIDs []uuid.UUID
	for _, idStr := range subtaskIDStrs {
		id, err := uuid.Parse(idStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, models.NewErrorResponse("INVALID_SUBTASK_ID", fmt.Sprintf("Invalid subtask ID: %s", idStr), nil))
			return
		}
		subtaskIDs = append(subtaskIDs, id)
	}

	err = h.subtaskService.ReorderSubtasks(c.Request.Context(), todoID, userID, subtaskIDs)
	if err != nil {
		statusCode := http.StatusInternalServerError
		errorCode := "SUBTASK_REORDER_FAILED"

		if strings.Contains(err.Error(), "not found") {
			statusCode = http.StatusNotFound
			errorCode = "TODO_NOT_FOUND"
		}

		c.JSON(statusCode, models.NewErrorResponse(errorCode, err.Error(), nil))
		return
	}

	c.JSON(http.StatusOK, models.NewAPIResponse("Subtasks reordered successfully", nil))
}

// ========================================
// NOTE HANDLER
// ========================================

// NoteHandler handles note HTTP requests
type NoteHandler struct {
	noteService *service.NoteService
}

// NewNoteHandler creates a new note handler
func NewNoteHandler(noteService *service.NoteService) *NoteHandler {
	return &NoteHandler{
		noteService: noteService,
	}
}

// CreateNote handles note creation
// @Summary Create a new note
// @Description Create a new note for a todo
// @Tags Notes
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param todo_id path string true "Todo ID"
// @Param request body models.Note true "Note details"
// @Success 201 {object} models.APIResponse{data=models.Note} "Note created successfully"
// @Failure 400 {object} models.APIResponse "Invalid request"
// @Failure 401 {object} models.APIResponse "Unauthorized"
// @Failure 404 {object} models.APIResponse "Todo not found"
// @Failure 500 {object} models.APIResponse "Internal server error"
// @Router /todos/{todo_id}/notes [post]
func (h *NoteHandler) CreateNote(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	todoIDStr := c.Param("id")
	todoID, err := uuid.Parse(todoIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse("INVALID_TODO_ID", "Invalid todo ID format", nil))
		return
	}

	var note models.Note
	if err := c.ShouldBindJSON(&note); err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse("INVALID_REQUEST", "Invalid request data", map[string]string{"details": err.Error()}))
		return
	}

	note.TodoID = todoID

	createdNote, err := h.noteService.CreateNote(c.Request.Context(), userID, &note)
	if err != nil {
		statusCode := http.StatusInternalServerError
		errorCode := "NOTE_CREATION_FAILED"

		if strings.Contains(err.Error(), "not found") {
			statusCode = http.StatusNotFound
			errorCode = "TODO_NOT_FOUND"
		} else if strings.Contains(err.Error(), "validation failed") {
			statusCode = http.StatusBadRequest
			errorCode = "VALIDATION_ERROR"
		}

		c.JSON(statusCode, models.NewErrorResponse(errorCode, err.Error(), nil))
		return
	}

	c.JSON(http.StatusCreated, models.NewAPIResponse("Note created successfully", createdNote))
}

// GetNotes handles retrieving notes for a todo
// @Summary Get notes for a todo
// @Description Get all notes for a specific todo
// @Tags Notes
// @Produce json
// @Security BearerAuth
// @Param todo_id path string true "Todo ID"
// @Success 200 {object} models.APIResponse{data=[]models.Note} "Notes retrieved successfully"
// @Failure 400 {object} models.APIResponse "Invalid todo ID"
// @Failure 401 {object} models.APIResponse "Unauthorized"
// @Failure 404 {object} models.APIResponse "Todo not found"
// @Failure 500 {object} models.APIResponse "Internal server error"
// @Router /todos/{todo_id}/notes [get]
func (h *NoteHandler) GetNotes(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	todoIDStr := c.Param("id")
	todoID, err := uuid.Parse(todoIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse("INVALID_TODO_ID", "Invalid todo ID format", nil))
		return
	}

	notes, err := h.noteService.GetNotesByTodoID(c.Request.Context(), todoID, userID)
	if err != nil {
		statusCode := http.StatusInternalServerError
		errorCode := "NOTE_RETRIEVAL_FAILED"

		if strings.Contains(err.Error(), "not found") {
			statusCode = http.StatusNotFound
			errorCode = "TODO_NOT_FOUND"
		}

		c.JSON(statusCode, models.NewErrorResponse(errorCode, err.Error(), nil))
		return
	}

	c.JSON(http.StatusOK, models.NewAPIResponse("Notes retrieved successfully", notes))
}

// UpdateNote handles note updates
// @Summary Update note
// @Description Update a note
// @Tags Notes
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Note ID"
// @Param request body map[string]interface{} true "Note updates"
// @Success 200 {object} models.APIResponse{data=models.Note} "Note updated successfully"
// @Failure 400 {object} models.APIResponse "Invalid request"
// @Failure 401 {object} models.APIResponse "Unauthorized"
// @Failure 404 {object} models.APIResponse "Note not found"
// @Failure 500 {object} models.APIResponse "Internal server error"
// @Router /notes/{id} [put]
func (h *NoteHandler) UpdateNote(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	noteIDStr := c.Param("id")
	noteID, err := uuid.Parse(noteIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse("INVALID_NOTE_ID", "Invalid note ID format", nil))
		return
	}

	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse("INVALID_REQUEST", "Invalid request data", map[string]string{"details": err.Error()}))
		return
	}

	note, err := h.noteService.UpdateNote(c.Request.Context(), noteID, userID, updates)
	if err != nil {
		statusCode := http.StatusInternalServerError
		errorCode := "NOTE_UPDATE_FAILED"

		if strings.Contains(err.Error(), "not found") {
			statusCode = http.StatusNotFound
			errorCode = "NOTE_NOT_FOUND"
		}

		c.JSON(statusCode, models.NewErrorResponse(errorCode, err.Error(), nil))
		return
	}

	c.JSON(http.StatusOK, models.NewAPIResponse("Note updated successfully", note))
}

// DeleteNote handles note deletion
// @Summary Delete note
// @Description Delete a note
// @Tags Notes
// @Produce json
// @Security BearerAuth
// @Param id path string true "Note ID"
// @Success 200 {object} models.APIResponse "Note deleted successfully"
// @Failure 400 {object} models.APIResponse "Invalid note ID"
// @Failure 401 {object} models.APIResponse "Unauthorized"
// @Failure 404 {object} models.APIResponse "Note not found"
// @Failure 500 {object} models.APIResponse "Internal server error"
// @Router /notes/{id} [delete]
func (h *NoteHandler) DeleteNote(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	noteIDStr := c.Param("id")
	noteID, err := uuid.Parse(noteIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse("INVALID_NOTE_ID", "Invalid note ID format", nil))
		return
	}

	err = h.noteService.DeleteNote(c.Request.Context(), noteID, userID)
	if err != nil {
		statusCode := http.StatusInternalServerError
		errorCode := "NOTE_DELETE_FAILED"

		if strings.Contains(err.Error(), "not found") {
			statusCode = http.StatusNotFound
			errorCode = "NOTE_NOT_FOUND"
		}

		c.JSON(statusCode, models.NewErrorResponse(errorCode, err.Error(), nil))
		return
	}

	c.JSON(http.StatusOK, models.NewAPIResponse("Note deleted successfully", nil))
}

// SearchNotes handles note search
// @Summary Search notes
// @Description Search notes by content for a todo
// @Tags Notes
// @Produce json
// @Security BearerAuth
// @Param todo_id path string true "Todo ID"
// @Param q query string true "Search query"
// @Success 200 {object} models.APIResponse{data=[]models.Note} "Notes search completed successfully"
// @Failure 400 {object} models.APIResponse "Invalid request"
// @Failure 401 {object} models.APIResponse "Unauthorized"
// @Failure 404 {object} models.APIResponse "Todo not found"
// @Failure 500 {object} models.APIResponse "Internal server error"
// @Router /todos/{todo_id}/notes/search [get]
func (h *NoteHandler) SearchNotes(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	todoIDStr := c.Param("todo_id")
	todoID, err := uuid.Parse(todoIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse("INVALID_TODO_ID", "Invalid todo ID format", nil))
		return
	}

	searchTerm := c.Query("q")
	if searchTerm == "" {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse("MISSING_SEARCH_TERM", "Search query is required", nil))
		return
	}

	notes, err := h.noteService.SearchNotes(c.Request.Context(), todoID, userID, searchTerm)
	if err != nil {
		statusCode := http.StatusInternalServerError
		errorCode := "SEARCH_FAILED"

		if strings.Contains(err.Error(), "not found") {
			statusCode = http.StatusNotFound
			errorCode = "TODO_NOT_FOUND"
		}

		c.JSON(statusCode, models.NewErrorResponse(errorCode, err.Error(), nil))
		return
	}

	c.JSON(http.StatusOK, models.NewAPIResponse("Notes search completed successfully", notes))
}

// ========================================
// SCHEDULED TASK HANDLER
// ========================================

// ScheduledTaskHandler handles scheduled task HTTP requests
type ScheduledTaskHandler struct {
	scheduledService *service.ScheduledTaskService
}

// NewScheduledTaskHandler creates a new scheduled task handler
func NewScheduledTaskHandler(scheduledService *service.ScheduledTaskService) *ScheduledTaskHandler {
	return &ScheduledTaskHandler{
		scheduledService: scheduledService,
	}
}

// CreateScheduledTask handles scheduled task creation
// @Summary Create a scheduled task
// @Description Create a new scheduled task for a todo
// @Tags Scheduled Tasks
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body models.ScheduledTask true "Scheduled task details"
// @Success 201 {object} models.APIResponse{data=models.ScheduledTask} "Scheduled task created successfully"
// @Failure 400 {object} models.APIResponse "Invalid request"
// @Failure 401 {object} models.APIResponse "Unauthorized"
// @Failure 404 {object} models.APIResponse "Todo not found"
// @Failure 500 {object} models.APIResponse "Internal server error"
// @Router /scheduled-tasks [post]
func (h *ScheduledTaskHandler) CreateScheduledTask(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	var task models.ScheduledTask
	if err := c.ShouldBindJSON(&task); err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse("INVALID_REQUEST", "Invalid request data", map[string]string{"details": err.Error()}))
		return
	}

	createdTask, err := h.scheduledService.CreateScheduledTask(c.Request.Context(), userID, &task)
	if err != nil {
		statusCode := http.StatusInternalServerError
		errorCode := "SCHEDULED_TASK_CREATION_FAILED"

		if strings.Contains(err.Error(), "not found") {
			statusCode = http.StatusNotFound
			errorCode = "TODO_NOT_FOUND"
		} else if strings.Contains(err.Error(), "validation failed") {
			statusCode = http.StatusBadRequest
			errorCode = "VALIDATION_ERROR"
		}

		c.JSON(statusCode, models.NewErrorResponse(errorCode, err.Error(), nil))
		return
	}

	c.JSON(http.StatusCreated, models.NewAPIResponse("Scheduled task created successfully", createdTask))
}

// GetUserScheduledTasks handles retrieving scheduled tasks for a user
// @Summary Get user scheduled tasks
// @Description Get scheduled tasks for the authenticated user
// @Tags Scheduled Tasks
// @Produce json
// @Security BearerAuth
// @Param from_date query string false "Filter from date (RFC3339 format)"
// @Param to_date query string false "Filter to date (RFC3339 format)"
// @Success 200 {object} models.APIResponse{data=[]models.ScheduledTask} "Scheduled tasks retrieved successfully"
// @Failure 401 {object} models.APIResponse "Unauthorized"
// @Failure 500 {object} models.APIResponse "Internal server error"
// @Router /scheduled-tasks [get]
func (h *ScheduledTaskHandler) GetUserScheduledTasks(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	var fromDate, toDate *string
	if fd := c.Query("from_date"); fd != "" {
		fromDate = &fd
	}
	if td := c.Query("to_date"); td != "" {
		toDate = &td
	}

	tasks, err := h.scheduledService.GetUserScheduledTasks(c.Request.Context(), userID, fromDate, toDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.NewErrorResponse("SCHEDULED_TASK_RETRIEVAL_FAILED", err.Error(), nil))
		return
	}

	c.JSON(http.StatusOK, models.NewAPIResponse("Scheduled tasks retrieved successfully", tasks))
}

// UpdateScheduledTask handles scheduled task updates
// @Summary Update scheduled task
// @Description Update a scheduled task
// @Tags Scheduled Tasks
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Scheduled Task ID"
// @Param request body map[string]interface{} true "Scheduled task updates"
// @Success 200 {object} models.APIResponse{data=models.ScheduledTask} "Scheduled task updated successfully"
// @Failure 400 {object} models.APIResponse "Invalid request"
// @Failure 401 {object} models.APIResponse "Unauthorized"
// @Failure 404 {object} models.APIResponse "Scheduled task not found"
// @Failure 500 {object} models.APIResponse "Internal server error"
// @Router /scheduled-tasks/{id} [put]
func (h *ScheduledTaskHandler) UpdateScheduledTask(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	taskIDStr := c.Param("id")
	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse("INVALID_TASK_ID", "Invalid scheduled task ID format", nil))
		return
	}

	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse("INVALID_REQUEST", "Invalid request data", map[string]string{"details": err.Error()}))
		return
	}

	task, err := h.scheduledService.UpdateScheduledTask(c.Request.Context(), taskID, userID, updates)
	if err != nil {
		statusCode := http.StatusInternalServerError
		errorCode := "SCHEDULED_TASK_UPDATE_FAILED"

		if strings.Contains(err.Error(), "not found") {
			statusCode = http.StatusNotFound
			errorCode = "SCHEDULED_TASK_NOT_FOUND"
		}

		c.JSON(statusCode, models.NewErrorResponse(errorCode, err.Error(), nil))
		return
	}

	c.JSON(http.StatusOK, models.NewAPIResponse("Scheduled task updated successfully", task))
}

// DeleteScheduledTask handles scheduled task deletion
// @Summary Delete scheduled task
// @Description Delete a scheduled task
// @Tags Scheduled Tasks
// @Produce json
// @Security BearerAuth
// @Param id path string true "Scheduled Task ID"
// @Success 200 {object} models.APIResponse "Scheduled task deleted successfully"
// @Failure 400 {object} models.APIResponse "Invalid scheduled task ID"
// @Failure 401 {object} models.APIResponse "Unauthorized"
// @Failure 404 {object} models.APIResponse "Scheduled task not found"
// @Failure 500 {object} models.APIResponse "Internal server error"
// @Router /scheduled-tasks/{id} [delete]
func (h *ScheduledTaskHandler) DeleteScheduledTask(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	taskIDStr := c.Param("id")
	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse("INVALID_TASK_ID", "Invalid scheduled task ID format", nil))
		return
	}

	err = h.scheduledService.DeleteScheduledTask(c.Request.Context(), taskID, userID)
	if err != nil {
		statusCode := http.StatusInternalServerError
		errorCode := "SCHEDULED_TASK_DELETE_FAILED"

		if strings.Contains(err.Error(), "not found") {
			statusCode = http.StatusNotFound
			errorCode = "SCHEDULED_TASK_NOT_FOUND"
		}

		c.JSON(statusCode, models.NewErrorResponse(errorCode, err.Error(), nil))
		return
	}

	c.JSON(http.StatusOK, models.NewAPIResponse("Scheduled task deleted successfully", nil))
}

// ========================================
// TIME ENTRY HANDLER
// ========================================

// TimeEntryHandler handles time entry HTTP requests
type TimeEntryHandler struct {
	timeService *service.TimeEntryService
}

// NewTimeEntryHandler creates a new time entry handler
func NewTimeEntryHandler(timeService *service.TimeEntryService) *TimeEntryHandler {
	return &TimeEntryHandler{
		timeService: timeService,
	}
}

// StartTimeTracking handles starting time tracking
// @Summary Start time tracking
// @Description Start time tracking for a todo
// @Tags Time Tracking
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body map[string]interface{} true "Time tracking details" example({"todo_id": "uuid", "description": "Working on feature"})
// @Success 201 {object} models.APIResponse{data=models.TimeEntry} "Time tracking started successfully"
// @Failure 400 {object} models.APIResponse "Invalid request"
// @Failure 401 {object} models.APIResponse "Unauthorized"
// @Failure 404 {object} models.APIResponse "Todo not found"
// @Failure 409 {object} models.APIResponse "Active time entry already exists"
// @Failure 500 {object} models.APIResponse "Internal server error"
// @Router /time-entries/start [post]
func (h *TimeEntryHandler) StartTimeTracking(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse("INVALID_REQUEST", "Invalid request data", map[string]string{"details": err.Error()}))
		return
	}

	todoIDStr, exists := req["todo_id"]
	if !exists {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse("MISSING_TODO_ID", "Todo ID is required", nil))
		return
	}

	todoID, err := uuid.Parse(todoIDStr.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse("INVALID_TODO_ID", "Invalid todo ID format", nil))
		return
	}

	var description *string
	if desc, exists := req["description"]; exists && desc != nil {
		descStr := desc.(string)
		description = &descStr
	}

	entry, err := h.timeService.StartTimeTracking(c.Request.Context(), userID, todoID, description)
	if err != nil {
		statusCode := http.StatusInternalServerError
		errorCode := "TIME_TRACKING_START_FAILED"

		if strings.Contains(err.Error(), "not found") {
			statusCode = http.StatusNotFound
			errorCode = "TODO_NOT_FOUND"
		} else if strings.Contains(err.Error(), "already an active") {
			statusCode = http.StatusConflict
			errorCode = "ACTIVE_TIME_ENTRY_EXISTS"
		}

		c.JSON(statusCode, models.NewErrorResponse(errorCode, err.Error(), nil))
		return
	}

	c.JSON(http.StatusCreated, models.NewAPIResponse("Time tracking started successfully", entry))
}

// StopTimeTracking handles stopping time tracking
// @Summary Stop time tracking
// @Description Stop the currently active time tracking session
// @Tags Time Tracking
// @Produce json
// @Security BearerAuth
// @Success 200 {object} models.APIResponse{data=models.TimeEntry} "Time tracking stopped successfully"
// @Failure 401 {object} models.APIResponse "Unauthorized"
// @Failure 404 {object} models.APIResponse "No active time entry found"
// @Failure 500 {object} models.APIResponse "Internal server error"
// @Router /time-entries/stop [post]
func (h *TimeEntryHandler) StopTimeTracking(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	entry, err := h.timeService.StopTimeTracking(c.Request.Context(), userID)
	if err != nil {
		statusCode := http.StatusInternalServerError
		errorCode := "TIME_TRACKING_STOP_FAILED"

		if strings.Contains(err.Error(), "no active") {
			statusCode = http.StatusNotFound
			errorCode = "NO_ACTIVE_TIME_ENTRY"
		}

		c.JSON(statusCode, models.NewErrorResponse(errorCode, err.Error(), nil))
		return
	}

	c.JSON(http.StatusOK, models.NewAPIResponse("Time tracking stopped successfully", entry))
}

// GetUserTimeEntries handles retrieving time entries for a user
// @Summary Get user time entries
// @Description Get time entries for the authenticated user
// @Tags Time Tracking
// @Produce json
// @Security BearerAuth
// @Param from_date query string false "Filter from date (RFC3339 format)"
// @Param to_date query string false "Filter to date (RFC3339 format)"
// @Param limit query integer false "Limit number of results" default(20)
// @Param offset query integer false "Offset for pagination" default(0)
// @Success 200 {object} models.APIResponse{data=map[string]interface{}} "Time entries retrieved successfully"
// @Failure 401 {object} models.APIResponse "Unauthorized"
// @Failure 500 {object} models.APIResponse "Internal server error"
// @Router /time-entries [get]
func (h *TimeEntryHandler) GetUserTimeEntries(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	var fromDate, toDate *string
	if fd := c.Query("from_date"); fd != "" {
		fromDate = &fd
	}
	if td := c.Query("to_date"); td != "" {
		toDate = &td
	}

	limitStr := c.DefaultQuery("limit", "20")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	offsetStr := c.DefaultQuery("offset", "0")
	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = 0
	}

	entries, total, err := h.timeService.GetUserTimeEntries(c.Request.Context(), userID, fromDate, toDate, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.NewErrorResponse("TIME_ENTRY_RETRIEVAL_FAILED", err.Error(), nil))
		return
	}

	result := map[string]interface{}{
		"entries": entries,
		"total":   total,
		"limit":   limit,
		"offset":  offset,
	}

	c.JSON(http.StatusOK, models.NewAPIResponse("Time entries retrieved successfully", result))
}

// GetActiveTimeEntry handles retrieving the active time entry
// @Summary Get active time entry
// @Description Get the currently active time tracking session
// @Tags Time Tracking
// @Produce json
// @Security BearerAuth
// @Success 200 {object} models.APIResponse{data=models.TimeEntry} "Active time entry retrieved successfully"
// @Failure 401 {object} models.APIResponse "Unauthorized"
// @Failure 404 {object} models.APIResponse "No active time entry found"
// @Failure 500 {object} models.APIResponse "Internal server error"
// @Router /time-entries/active [get]
func (h *TimeEntryHandler) GetActiveTimeEntry(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	entry, err := h.timeService.GetActiveTimeEntry(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.NewErrorResponse("ACTIVE_TIME_ENTRY_RETRIEVAL_FAILED", err.Error(), nil))
		return
	}

	if entry == nil {
		c.JSON(http.StatusNotFound, models.NewErrorResponse("NO_ACTIVE_TIME_ENTRY", "No active time entry found", nil))
		return
	}

	c.JSON(http.StatusOK, models.NewAPIResponse("Active time entry retrieved successfully", entry))
}
