package handler

import (
	"macwrite-auth-api/internal/service"

	"github.com/gin-gonic/gin"
)

// TodoRoutes sets up all todo-related routes
func SetupTodoRoutes(
	router *gin.Engine,
	authMiddleware gin.HandlerFunc,
	rateLimitMiddleware gin.HandlerFunc,
	// Services
	projectService *service.ProjectService,
	todoService *service.TodoService,
	subtaskService *service.SubtaskService,
	noteService *service.NoteService,
	scheduledService *service.ScheduledTaskService,
	timeService *service.TimeEntryService,
	releaseService *service.ReleaseVersionService,
	globalSearchService *service.GlobalSearchService,
) {
	// Create handlers
	projectHandler := NewProjectHandler(projectService)
	todoHandler := NewTodoHandler(todoService)
	subtaskHandler := NewSubtaskHandler(subtaskService)
	noteHandler := NewNoteHandler(noteService)
	scheduledHandler := NewScheduledTaskHandler(scheduledService)
	timeHandler := NewTimeEntryHandler(timeService)
	releaseHandler := NewReleaseVersionHandler(releaseService)
	globalSearchHandler := NewGlobalSearchHandler(globalSearchService)

	// Create API v1 group
	api := router.Group("/api/v1")
	api.Use(rateLimitMiddleware)

	// Protected routes that require authentication
	protected := api.Group("")
	protected.Use(authMiddleware)

	// ========================================
	// PROJECT ROUTES
	// ========================================
	projects := protected.Group("/projects")
	{
		// Basic CRUD operations
		projects.POST("", projectHandler.CreateProject)
		projects.GET("", projectHandler.GetUserProjects)
		projects.GET("/:id", projectHandler.GetProject)
		projects.PUT("/:id", projectHandler.UpdateProject)
		projects.DELETE("/:id", projectHandler.DeleteProject)

		// Project management
		projects.POST("/:id/archive", projectHandler.ArchiveProject)
		projects.POST("/:id/restore", projectHandler.RestoreProject)

		// Project analytics and filtering
		projects.GET("/stats", projectHandler.GetProjectStats)
		projects.GET("/favorites", projectHandler.GetFavoriteProjects)
		projects.GET("/category/:category", projectHandler.GetProjectsByCategory)
		projects.GET("/search", projectHandler.SearchProjects)
		projects.GET("/dashboard", projectHandler.GetProjectDashboard)

		// Project todos
		projects.GET("/:project_id/todos", todoHandler.GetProjectTodos)

		// Project release versions
		projects.GET("/:project_id/release-versions", releaseHandler.GetProjectReleaseVersions)

		// Project export/import
		projects.GET("/:id/export", projectHandler.ExportProject)
		projects.POST("/:id/import", projectHandler.ImportProject)
	}

	// ========================================
	// TODO ROUTES
	// ========================================
	todos := protected.Group("/todos")
	{
		// Basic CRUD operations
		todos.POST("", todoHandler.CreateTodo)
		todos.GET("", todoHandler.GetUserTodos)
		todos.GET("/:id", todoHandler.GetTodo)
		todos.PUT("/:id", todoHandler.UpdateTodo)
		todos.DELETE("/:id", todoHandler.DeleteTodo)

		// Todo management
		todos.POST("/:id/archive", todoHandler.ArchiveTodo)
		todos.POST("/:id/complete", todoHandler.MarkTodoComplete)
		todos.POST("/:id/incomplete", todoHandler.MarkTodoIncomplete)
		todos.POST("/:id/pin", todoHandler.PinTodo)

		// Todo analytics and filtering
		todos.GET("/dashboard", todoHandler.GetTodoDashboard)
		todos.GET("/overdue", todoHandler.GetOverdueTodos)
		todos.GET("/search", todoHandler.SearchTodos)

		// Subtask routes nested under todos
		todos.POST("/:todo_id/subtasks", subtaskHandler.CreateSubtask)
		todos.GET("/:todo_id/subtasks", subtaskHandler.GetSubtasks)
		todos.POST("/:todo_id/subtasks/reorder", subtaskHandler.ReorderSubtasks)

		// Note routes nested under todos
		todos.POST("/:todo_id/notes", noteHandler.CreateNote)
		todos.GET("/:todo_id/notes", noteHandler.GetNotes)
		todos.GET("/:todo_id/notes/search", noteHandler.SearchNotes)
	}

	// ========================================
	// SUBTASK ROUTES (Direct access)
	// ========================================
	subtasks := protected.Group("/subtasks")
	{
		subtasks.PUT("/:id", subtaskHandler.UpdateSubtask)
		subtasks.DELETE("/:id", subtaskHandler.DeleteSubtask)
	}

	// ========================================
	// NOTE ROUTES (Direct access)
	// ========================================
	notes := protected.Group("/notes")
	{
		notes.PUT("/:id", noteHandler.UpdateNote)
		notes.DELETE("/:id", noteHandler.DeleteNote)
	}

	// ========================================
	// RELEASE VERSION ROUTES
	// ========================================
	releases := protected.Group("/release-versions")
	{
		// Basic CRUD operations
		releases.POST("", releaseHandler.CreateReleaseVersion)
		releases.GET("/:id", releaseHandler.GetReleaseVersion)
		releases.PUT("/:id", releaseHandler.UpdateReleaseVersion)
		releases.DELETE("/:id", releaseHandler.DeleteReleaseVersion)

		// Release management
		releases.POST("/:id/release", releaseHandler.MarkAsReleased)

		// Release analytics and filtering
		releases.GET("/upcoming", releaseHandler.GetUpcomingReleases)
		releases.GET("/overdue", releaseHandler.GetOverdueReleases)
		releases.GET("/:id/stats", releaseHandler.GetReleaseStats)
		releases.GET("/:id/todos", releaseHandler.GetReleaseTodosProgress)
	}

	// ========================================
	// SCHEDULED TASK ROUTES
	// ========================================
	scheduled := protected.Group("/scheduled-tasks")
	{
		scheduled.POST("", scheduledHandler.CreateScheduledTask)
		scheduled.GET("", scheduledHandler.GetUserScheduledTasks)
		scheduled.PUT("/:id", scheduledHandler.UpdateScheduledTask)
		scheduled.DELETE("/:id", scheduledHandler.DeleteScheduledTask)
	}

	// ========================================
	// TIME TRACKING ROUTES
	// ========================================
	timeTracking := protected.Group("/time-entries")
	{
		// Time tracking control
		timeTracking.POST("/start", timeHandler.StartTimeTracking)
		timeTracking.POST("/stop", timeHandler.StopTimeTracking)
		timeTracking.GET("/active", timeHandler.GetActiveTimeEntry)

		// Time entry management
		timeTracking.GET("", timeHandler.GetUserTimeEntries)
		// Note: Update and Delete endpoints can be added if needed
		// timeTracking.PUT("/:id", timeHandler.UpdateTimeEntry)
		// timeTracking.DELETE("/:id", timeHandler.DeleteTimeEntry)
	}

	// ========================================
	// DASHBOARD ROUTES (Consolidated)
	// ========================================
	dashboard := protected.Group("/dashboard")
	{
		dashboard.GET("/projects", projectHandler.GetProjectDashboard)
		dashboard.GET("/todos", todoHandler.GetTodoDashboard)
		dashboard.GET("/overdue", todoHandler.GetOverdueTodos)
		dashboard.GET("/upcoming-releases", releaseHandler.GetUpcomingReleases)
		dashboard.GET("/overdue-releases", releaseHandler.GetOverdueReleases)
		dashboard.GET("/scheduled-tasks", scheduledHandler.GetUserScheduledTasks)
		dashboard.GET("/active-time", timeHandler.GetActiveTimeEntry)
	}

	// ========================================
	// ANALYTICS ROUTES (Consolidated)
	// ========================================
	analytics := protected.Group("/analytics")
	{
		analytics.GET("/project-stats", projectHandler.GetProjectStats)
		analytics.GET("/release-stats/:id", releaseHandler.GetReleaseStats)
	}

	// ========================================
	// SEARCH ROUTES (Consolidated)
	// ========================================
	search := protected.Group("/search")
	{
		// Legacy search endpoints
		search.GET("/projects", projectHandler.SearchProjects)
		search.GET("/todos", todoHandler.SearchTodos)

		// New global search endpoints
		search.GET("", globalSearchHandler.GlobalSearch)                  // Global search across all types
		search.GET("/quick", globalSearchHandler.QuickSearch)             // Quick search for autocomplete
		search.GET("/suggestions", globalSearchHandler.SearchSuggestions) // Search suggestions
		search.GET("/history", globalSearchHandler.SearchHistory)         // Search history
	}
}

// Helper function to setup all Todo services and routes
func SetupTodoAPI(
	router *gin.Engine,
	authMiddleware gin.HandlerFunc,
	rateLimitMiddleware gin.HandlerFunc,
	services *TodoServices,
) {
	SetupTodoRoutes(
		router,
		authMiddleware,
		rateLimitMiddleware,
		services.ProjectService,
		services.TodoService,
		services.SubtaskService,
		services.NoteService,
		services.ScheduledService,
		services.TimeService,
		services.ReleaseService,
		services.GlobalSearchService,
	)
}

// TodoServices groups all todo-related services
type TodoServices struct {
	ProjectService      *service.ProjectService
	TodoService         *service.TodoService
	SubtaskService      *service.SubtaskService
	NoteService         *service.NoteService
	ScheduledService    *service.ScheduledTaskService
	TimeService         *service.TimeEntryService
	ReleaseService      *service.ReleaseVersionService
	GlobalSearchService *service.GlobalSearchService
}

// RouteInfo represents API route information for documentation
type RouteInfo struct {
	Method      string            `json:"method"`
	Path        string            `json:"path"`
	Description string            `json:"description"`
	Auth        bool              `json:"auth_required"`
	Parameters  map[string]string `json:"parameters,omitempty"`
}

// GetTodoAPIRoutes returns all todo API routes for documentation
func GetTodoAPIRoutes() []RouteInfo {
	return []RouteInfo{
		// Projects
		{Method: "POST", Path: "/api/v1/projects", Description: "Create a new project", Auth: true},
		{Method: "GET", Path: "/api/v1/projects", Description: "Get user projects", Auth: true},
		{Method: "GET", Path: "/api/v1/projects/:id", Description: "Get project by ID", Auth: true},
		{Method: "PUT", Path: "/api/v1/projects/:id", Description: "Update project", Auth: true},
		{Method: "DELETE", Path: "/api/v1/projects/:id", Description: "Delete project", Auth: true},
		{Method: "POST", Path: "/api/v1/projects/:id/archive", Description: "Archive project", Auth: true},
		{Method: "POST", Path: "/api/v1/projects/:id/restore", Description: "Restore archived project", Auth: true},
		{Method: "GET", Path: "/api/v1/projects/stats", Description: "Get project statistics", Auth: true},
		{Method: "GET", Path: "/api/v1/projects/favorites", Description: "Get favorite projects", Auth: true},
		{Method: "GET", Path: "/api/v1/projects/category/:category", Description: "Get projects by category", Auth: true},
		{Method: "GET", Path: "/api/v1/projects/search", Description: "Search projects", Auth: true},
		{Method: "GET", Path: "/api/v1/projects/dashboard", Description: "Get project dashboard", Auth: true},

		// Todos
		{Method: "POST", Path: "/api/v1/todos", Description: "Create a new todo", Auth: true},
		{Method: "GET", Path: "/api/v1/todos", Description: "Get user todos with filtering", Auth: true},
		{Method: "GET", Path: "/api/v1/todos/:id", Description: "Get todo by ID", Auth: true},
		{Method: "PUT", Path: "/api/v1/todos/:id", Description: "Update todo", Auth: true},
		{Method: "DELETE", Path: "/api/v1/todos/:id", Description: "Delete todo", Auth: true},
		{Method: "POST", Path: "/api/v1/todos/:id/archive", Description: "Archive todo", Auth: true},
		{Method: "POST", Path: "/api/v1/todos/:id/complete", Description: "Mark todo as complete", Auth: true},
		{Method: "POST", Path: "/api/v1/todos/:id/incomplete", Description: "Mark todo as incomplete", Auth: true},
		{Method: "POST", Path: "/api/v1/todos/:id/pin", Description: "Pin/unpin todo", Auth: true},
		{Method: "GET", Path: "/api/v1/todos/dashboard", Description: "Get todo dashboard", Auth: true},
		{Method: "GET", Path: "/api/v1/todos/overdue", Description: "Get overdue todos", Auth: true},
		{Method: "GET", Path: "/api/v1/todos/search", Description: "Search todos", Auth: true},

		// Project Todos
		{Method: "GET", Path: "/api/v1/projects/:project_id/todos", Description: "Get todos for a project", Auth: true},

		// Subtasks
		{Method: "POST", Path: "/api/v1/todos/:todo_id/subtasks", Description: "Create subtask", Auth: true},
		{Method: "GET", Path: "/api/v1/todos/:todo_id/subtasks", Description: "Get subtasks for todo", Auth: true},
		{Method: "POST", Path: "/api/v1/todos/:todo_id/subtasks/reorder", Description: "Reorder subtasks", Auth: true},
		{Method: "PUT", Path: "/api/v1/subtasks/:id", Description: "Update subtask", Auth: true},
		{Method: "DELETE", Path: "/api/v1/subtasks/:id", Description: "Delete subtask", Auth: true},

		// Notes
		{Method: "POST", Path: "/api/v1/todos/:todo_id/notes", Description: "Create note", Auth: true},
		{Method: "GET", Path: "/api/v1/todos/:todo_id/notes", Description: "Get notes for todo", Auth: true},
		{Method: "GET", Path: "/api/v1/todos/:todo_id/notes/search", Description: "Search notes", Auth: true},
		{Method: "PUT", Path: "/api/v1/notes/:id", Description: "Update note", Auth: true},
		{Method: "DELETE", Path: "/api/v1/notes/:id", Description: "Delete note", Auth: true},

		// Release Versions
		{Method: "POST", Path: "/api/v1/release-versions", Description: "Create release version", Auth: true},
		{Method: "GET", Path: "/api/v1/release-versions/:id", Description: "Get release version", Auth: true},
		{Method: "PUT", Path: "/api/v1/release-versions/:id", Description: "Update release version", Auth: true},
		{Method: "DELETE", Path: "/api/v1/release-versions/:id", Description: "Delete release version", Auth: true},
		{Method: "POST", Path: "/api/v1/release-versions/:id/release", Description: "Mark as released", Auth: true},
		{Method: "GET", Path: "/api/v1/release-versions/upcoming", Description: "Get upcoming releases", Auth: true},
		{Method: "GET", Path: "/api/v1/release-versions/overdue", Description: "Get overdue releases", Auth: true},
		{Method: "GET", Path: "/api/v1/release-versions/:id/stats", Description: "Get release statistics", Auth: true},
		{Method: "GET", Path: "/api/v1/release-versions/:id/todos", Description: "Get release todos progress", Auth: true},
		{Method: "GET", Path: "/api/v1/projects/:project_id/release-versions", Description: "Get project release versions", Auth: true},

		// Scheduled Tasks
		{Method: "POST", Path: "/api/v1/scheduled-tasks", Description: "Create scheduled task", Auth: true},
		{Method: "GET", Path: "/api/v1/scheduled-tasks", Description: "Get user scheduled tasks", Auth: true},
		{Method: "PUT", Path: "/api/v1/scheduled-tasks/:id", Description: "Update scheduled task", Auth: true},
		{Method: "DELETE", Path: "/api/v1/scheduled-tasks/:id", Description: "Delete scheduled task", Auth: true},

		// Time Tracking
		{Method: "POST", Path: "/api/v1/time-entries/start", Description: "Start time tracking", Auth: true},
		{Method: "POST", Path: "/api/v1/time-entries/stop", Description: "Stop time tracking", Auth: true},
		{Method: "GET", Path: "/api/v1/time-entries/active", Description: "Get active time entry", Auth: true},
		{Method: "GET", Path: "/api/v1/time-entries", Description: "Get user time entries", Auth: true},

		// Dashboard
		{Method: "GET", Path: "/api/v1/dashboard/projects", Description: "Get project dashboard", Auth: true},
		{Method: "GET", Path: "/api/v1/dashboard/todos", Description: "Get todo dashboard", Auth: true},
		{Method: "GET", Path: "/api/v1/dashboard/overdue", Description: "Get overdue todos", Auth: true},
		{Method: "GET", Path: "/api/v1/dashboard/upcoming-releases", Description: "Get upcoming releases", Auth: true},
		{Method: "GET", Path: "/api/v1/dashboard/overdue-releases", Description: "Get overdue releases", Auth: true},
		{Method: "GET", Path: "/api/v1/dashboard/scheduled-tasks", Description: "Get scheduled tasks", Auth: true},
		{Method: "GET", Path: "/api/v1/dashboard/active-time", Description: "Get active time entry", Auth: true},

		// Analytics
		{Method: "GET", Path: "/api/v1/analytics/project-stats", Description: "Get project statistics", Auth: true},
		{Method: "GET", Path: "/api/v1/analytics/release-stats/:id", Description: "Get release statistics", Auth: true},

		// Search
		{Method: "GET", Path: "/api/v1/search", Description: "Global search across projects and tasks", Auth: true},
		{Method: "GET", Path: "/api/v1/search/quick", Description: "Quick search for autocomplete", Auth: true},
		{Method: "GET", Path: "/api/v1/search/suggestions", Description: "Get search suggestions", Auth: true},
		{Method: "GET", Path: "/api/v1/search/history", Description: "Get search history", Auth: true},
		{Method: "GET", Path: "/api/v1/search/projects", Description: "Search projects (legacy)", Auth: true},
		{Method: "GET", Path: "/api/v1/search/todos", Description: "Search todos (legacy)", Auth: true},
	}
}
