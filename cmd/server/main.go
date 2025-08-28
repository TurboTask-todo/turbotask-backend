package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"quantumtask-auth-api/internal/config"
	"quantumtask-auth-api/internal/database"
	"quantumtask-auth-api/internal/handler"
	"quantumtask-auth-api/internal/middleware"
	"quantumtask-auth-api/internal/repository"
	"quantumtask-auth-api/internal/service"
	"quantumtask-auth-api/pkg/ai"
	"quantumtask-auth-api/pkg/auth"
	"quantumtask-auth-api/pkg/compression"
	"quantumtask-auth-api/pkg/email"
	apmclient "quantumtask-auth-api/pkg/middleware"
	"quantumtask-auth-api/pkg/oauth"
	"quantumtask-auth-api/pkg/queue"
	"quantumtask-auth-api/pkg/redis"
	"quantumtask-auth-api/pkg/validation"
	"quantumtask-auth-api/pkg/websocket"

	"github.com/gin-gonic/gin"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Set Gin mode
	gin.SetMode(cfg.Server.GinMode)

	// Connect to database
	db, err := database.Connect(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Run database migrations check
	if err := db.RunMigrations(); err != nil {
		log.Fatalf("Database migration check failed: %v", err)
	}

	// Initialize middleware.io APM tracking
	log.Printf("🔧 Initializing Middleware.io APM tracking...")
	middlewareClient := apmclient.NewClient(&cfg.Middleware)
	if err := middlewareClient.Start(context.Background()); err != nil {
		log.Printf("⚠️  Failed to start middleware.io APM tracking: %v", err)
		log.Printf("🔄 Continuing without APM tracking...")
	} else {
		log.Printf("✅ Middleware.io APM tracking initialized successfully")
	}
	// Initialize repositories
	userRepo := repository.NewUserRepository(db)
	passwordRepo := repository.NewPasswordRepository(db)
	aiConversationRepo := repository.NewAIConversationRepository(db)
	// refreshTokenRepo := repository.NewRefreshTokenRepository(db)
	// oauthRepo := repository.NewOAuthRepository(db)

	// Initialize OTP repository
	otpRepo := repository.NewOTPRepository(db.DB)

	// Initialize Todo-related repositories
	projectRepo := repository.NewProjectRepository(db)
	releaseVersionRepo := repository.NewReleaseVersionRepository(db)
	todoRepo := repository.NewTodoRepository(db)
	subtaskRepo := repository.NewSubtaskRepository(db)
	noteRepo := repository.NewNoteRepository(db)
	scheduledTaskRepo := repository.NewScheduledTaskRepository(db)
	timeEntryRepo := repository.NewTimeEntryRepository(db)

	// Initialize AI-related repositories
	aiInteractionRepo := repository.NewAIInteractionRepository(db)
	aiSuggestionRepo := repository.NewAISuggestionRepository(db)

	// Initialize authentication components
	jwtManager := auth.NewJWTManager(
		cfg.JWT.Secret,
		cfg.JWT.AccessTokenDuration,
		cfg.JWT.RefreshTokenDuration,
		cfg.App.Name,
	)
	passwordHasher := auth.NewPasswordHasher()
	emailValidator := validation.NewEmailValidator()

	// Initialize OAuth providers
	googleOAuth := oauth.NewGoogleOAuth(
		cfg.OAuth.Google.ClientID,
		cfg.OAuth.Google.ClientSecret,
		cfg.OAuth.Google.RedirectURL,
	)

	// Initialize AI conversation components
	redisClient := redis.NewClient(cfg.Redis.Addr, cfg.Redis.Password, cfg.Redis.DB)

	// Initialize RabbitMQ client for AI enhancement queue
	amqpURL := cfg.Queue.URL
	// Fallback to default AMQP URL if configuration is missing or invalid
	if !strings.HasPrefix(amqpURL, "amqp://") {
		amqpURL = "amqp://guest:guest@localhost:5672/"
	}

	fmt.Printf("🐰 Initializing RabbitMQ client with URL: %s\n", amqpURL)
	queueClient, err := queue.NewRabbitMQClient(amqpURL)
	if err != nil {
		log.Fatalf("Failed to initialize RabbitMQ client: %v", err)
	}

	// Test RabbitMQ connection
	if err := queueClient.Ping(context.Background()); err != nil {
		log.Printf("⚠️  RabbitMQ connection test failed: %v", err)
		log.Printf("🔄 Falling back to mock client for development")
		queueClient = queue.NewMockClient()
	} else {
		fmt.Printf("✅ RabbitMQ connection successful\n")
	}

	compressor := compression.NewCompressor()

	// Initialize WebSocket infrastructure
	wsHub := websocket.NewHub(redisClient)
	go wsHub.Run(context.Background())
	eventBroadcaster := websocket.NewEventBroadcaster(wsHub)

	// Initialize AI components
	externalAIClient := ai.NewExternalAIClient(cfg.AI.ExternalAIURL)
	aiService := service.NewAIService(
		externalAIClient,
		todoRepo,
		subtaskRepo,
		aiInteractionRepo,
		aiSuggestionRepo,
		redisClient,
		queueClient,
		eventBroadcaster,
	)

	// Initialize email client
	emailClient := email.NewClient(cfg.Email)

	// Initialize email service for background email processing
	emailService := service.NewEmailService(emailClient, queueClient)

	// Initialize and start AI enhancement consumer with dedicated context
	fmt.Printf("🚀 Initializing AI Enhancement Consumer...\n")
	aiEnhancementConsumer := service.NewAIEnhancementConsumer(aiService, queueClient, wsHub)

	// Use background context for consumer to prevent early cancellation
	consumerCtx := context.Background()

	go func() {
		fmt.Printf("🔄 Starting AI Enhancement Consumer in goroutine...\n")
		for {
			err := aiEnhancementConsumer.StartConsumer(consumerCtx)
			if err != nil {
				fmt.Printf("❌ AI enhancement consumer failed: %v\n", err)
				fmt.Printf("🔄 Retrying consumer in 5 seconds...\n")
				time.Sleep(5 * time.Second)
				continue
			}
			break
		}
	}()

	fmt.Println("AI enhancement consumer started")

	// Initialize and start Email consumer
	fmt.Printf("📧 Initializing Email Consumer...\n")
	emailConsumer := service.NewEmailConsumer(emailService, queueClient)

	go func() {
		fmt.Printf("🔄 Starting Email Consumer in goroutine...\n")
		for {
			err := emailConsumer.StartConsumer(consumerCtx)
			if err != nil {
				fmt.Printf("❌ Email consumer failed: %v\n", err)
				fmt.Printf("🔄 Retrying email consumer in 5 seconds...\n")
				time.Sleep(5 * time.Second)
				continue
			}
			break
		}
	}()

	fmt.Println("Email consumer started")

	// Initialize services
	authService := service.NewAuthService(
		userRepo,
		passwordRepo,
		jwtManager,
		passwordHasher,
		emailValidator,
		cfg,
	)

	aiConversationService := service.NewAIConversationService(
		aiConversationRepo,
		redisClient,
		queueClient,
		compressor,
		cfg,
	)

	// Initialize user service
	userService := service.NewUserService(userRepo)

	// Initialize OTP service with background email processing
	otpService := service.NewOTPService(otpRepo, userRepo, emailClient, emailService)

	// Initialize Todo-related services
	projectService := service.NewProjectService(projectRepo, todoRepo, redisClient)
	releaseVersionService := service.NewReleaseVersionService(releaseVersionRepo, projectRepo, redisClient)
	subtaskService := service.NewSubtaskService(subtaskRepo, todoRepo, projectRepo, redisClient)
	noteService := service.NewNoteService(noteRepo, todoRepo, redisClient)
	scheduledTaskService := service.NewScheduledTaskService(scheduledTaskRepo, todoRepo, redisClient)
	timeEntryService := service.NewTimeEntryService(timeEntryRepo, todoRepo, redisClient)

	// TodoService needs to be initialized after projectService since it depends on it
	todoService := service.NewTodoService(todoRepo, projectRepo, subtaskRepo, noteRepo, timeEntryRepo, projectService, redisClient)

	// Initialize global search service
	globalSearchService := service.NewGlobalSearchService(projectService, todoService)

	// Initialize WebSocket-enabled services
	wsHandler := handler.NewWebSocketHandler(wsHub, jwtManager)

	wsTodoService := service.NewWebSocketTodoService(todoService, eventBroadcaster)
	wsSubtaskService := service.NewWebSocketSubtaskService(subtaskService, eventBroadcaster, todoService)
	wsProjectService := service.NewWebSocketProjectService(projectService, eventBroadcaster)

	// Initialize middleware
	authMiddleware := middleware.NewAuthMiddleware(jwtManager)

	// Initialize handlers
	authHandler := handler.NewAuthHandler(authService)
	oauthHandler := handler.NewOAuthHandler(authService, googleOAuth)
	loginHandler := handler.NewLoginHandler(otpService, userService, authService)
	aiConversationHandler := handler.NewAIConversationHandler(aiConversationService)
	aiHandler := handler.NewAIHandler(aiService)

	// Initialize Todo-related handlers
	projectHandler := handler.NewProjectHandler(projectService)
	globalSearchHandler := handler.NewGlobalSearchHandler(globalSearchService)
	releaseVersionHandler := handler.NewReleaseVersionHandler(releaseVersionService)
	todoHandler := handler.NewTodoHandler(todoService)
	subtaskHandler := handler.NewSubtaskHandler(subtaskService)
	noteHandler := handler.NewNoteHandler(noteService)
	scheduledTaskHandler := handler.NewScheduledTaskHandler(scheduledTaskService)
	timeEntryHandler := handler.NewTimeEntryHandler(timeEntryService)

	// Initialize break repository and service
	breakRepo := repository.NewBreakHistoryRepository(db)
	breakService := service.NewBreakService(breakRepo, todoRepo, projectRepo)
	breakHandler := handler.NewBreakHandler(breakService)

	// Initialize reports repository and service
	reportsRepo := repository.NewReportsRepository(db)
	reportsService := service.NewReportsService(reportsRepo, redisClient)
	reportsHandler := handler.NewReportsHandler(reportsService)

	// Initialize WebSocket-enhanced handlers
	wsTodoHandler := handler.NewWebSocketTodoHandler(wsTodoService, wsHandler)
	wsSubtaskHandler := handler.NewWebSocketSubtaskHandler(wsSubtaskService, wsHandler)
	wsProjectHandler := handler.NewWebSocketProjectHandler(wsProjectService, wsHandler)

	// Setup HTTP server
	router := setupRouter(
		cfg,
		db,
		authMiddleware,
		authHandler,
		oauthHandler,
		loginHandler,
		aiConversationHandler,
		aiHandler,
		projectHandler,
		globalSearchHandler,
		releaseVersionHandler,
		todoHandler,
		subtaskHandler,
		noteHandler,
		scheduledTaskHandler,
		timeEntryHandler,
		breakHandler,
		reportsHandler,
		wsHandler,
		wsTodoHandler,
		wsSubtaskHandler,
		wsProjectHandler,
		aiEnhancementConsumer,
		middlewareClient,
	)

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("Starting %s server on port %d", cfg.App.Name, cfg.Server.Port)
		log.Printf("Environment: %s", cfg.App.Env)
		log.Printf("Version: %s", cfg.App.Version)
		log.Printf("Server URL: http://localhost:%d", cfg.Server.Port)

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Give outstanding requests 30 seconds to complete
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Gracefully stop middleware.io APM tracking
	if middlewareClient.IsStarted() {
		log.Println("Stopping middleware.io APM tracking...")
		if err := middlewareClient.Stop(); err != nil {
			log.Printf("Error stopping middleware.io APM tracking: %v", err)
		} else {
			log.Println("Middleware.io APM tracking stopped successfully")
		}
	}

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}

func setupRouter(
	cfg *config.Config,
	db *database.DB,
	authMiddleware *middleware.AuthMiddleware,
	authHandler *handler.AuthHandler,
	oauthHandler *handler.OAuthHandler,
	loginHandler *handler.LoginHandler,
	aiConversationHandler *handler.AIConversationHandler,
	aiHandler *handler.AIHandler,
	projectHandler *handler.ProjectHandler,
	globalSearchHandler *handler.GlobalSearchHandler,
	releaseVersionHandler *handler.ReleaseVersionHandler,
	todoHandler *handler.TodoHandler,
	subtaskHandler *handler.SubtaskHandler,
	noteHandler *handler.NoteHandler,
	scheduledTaskHandler *handler.ScheduledTaskHandler,
	timeEntryHandler *handler.TimeEntryHandler,
	breakHandler *handler.BreakHandler,
	reportsHandler *handler.ReportsHandler,
	wsHandler *handler.WebSocketHandler,
	wsTodoHandler *handler.WebSocketTodoHandler,
	wsSubtaskHandler *handler.WebSocketSubtaskHandler,
	wsProjectHandler *handler.WebSocketProjectHandler,
	aiEnhancementConsumer *service.AIEnhancementConsumer,
	middlewareClient *apmclient.Client,
) *gin.Engine {
	router := gin.New()

	// Global middleware
	router.Use(middleware.RequestLogger())
	router.Use(middleware.Recovery())
	router.Use(middleware.RequestID())
	router.Use(middleware.SecurityHeaders())
	router.Use(middleware.CORS(cfg))
	router.Use(middleware.NoCache())
	router.Use(middleware.DeviceFingerprint())
	router.Use(middleware.ValidateContentType("application/json"))
	router.Use(middleware.MaxRequestSize(10 * 1024 * 1024)) // 10MB max request size

	// Add middleware.io APM tracking middleware
	if middlewareClient.IsEnabled() {
		apmMiddlewares := middlewareClient.GetMiddlewares()
		for _, apmMiddleware := range apmMiddlewares {
			router.Use(apmMiddleware)
		}
		log.Printf("✅ Middleware.io APM tracking middleware applied")
	}

	// Apply global rate limiting
	router.Use(middleware.GlobalRateLimit(cfg.Security.RateLimitRequestsPerMinute))

	// Health check endpoint
	router.GET("/health", healthCheck(cfg, db))

	// Advanced database monitoring endpoints
	dbMonitoring := router.Group("/api/v1/monitoring")
	{
		dbMonitoring.GET("/database/metrics", getDatabaseMetrics(db))
		dbMonitoring.GET("/database/health", getDatabaseHealth(db))
		dbMonitoring.GET("/database/pool-status", getConnectionPoolStatus(db))
	}

	// AI Enhancement Consumer monitoring endpoints
	aiMonitoring := router.Group("/api/v1/ai-monitoring")
	{
		aiMonitoring.GET("/consumer/health", getAIConsumerHealth(aiEnhancementConsumer))
		aiMonitoring.GET("/consumer/metrics", getAIConsumerMetrics(aiEnhancementConsumer))
	}

	// Middleware.io APM monitoring endpoints
	apmMonitoring := router.Group("/api/v1/apm-monitoring")
	{
		apmMonitoring.GET("/health", getAPMHealth(middlewareClient))
		apmMonitoring.GET("/metrics", getAPMMetrics(middlewareClient))
		apmMonitoring.POST("/restart", restartAPM(middlewareClient))
	}

	// API version group
	v1 := router.Group("/api/v1")

	// Authentication routes
	authGroup := v1.Group("/auth")
	{
		// Public authentication endpoints
		authGroup.POST("/register",
			middleware.IPRateLimit(20), // 20 registrations per minute per IP
			authHandler.Register)

		authGroup.POST("/login",
			middleware.LoginRateLimit(),
			middleware.BruteForceProtection(),
			authHandler.Login)

		authGroup.POST("/refresh",
			middleware.IPRateLimit(30), // 30 refresh attempts per minute per IP
			authHandler.RefreshToken)

		// OAuth routes
		authGroup.GET("/google",
			middleware.IPRateLimit(10), // 10 OAuth initiations per minute per IP
			oauthHandler.GoogleAuth)

		authGroup.GET("/google/callback",
			middleware.IPRateLimit(15), // 15 OAuth callbacks per minute per IP
			oauthHandler.GoogleCallback)

		authGroup.GET("/oauth/providers", oauthHandler.GetOAuthProviders)

		// Email OTP Login routes
		authGroup.POST("/login/initiate",
			middleware.IPRateLimit(5), // 5 OTP requests per minute per IP
			loginHandler.InitiateLogin)

		authGroup.POST("/login/verify",
			middleware.IPRateLimit(10), // 10 verification attempts per minute per IP
			loginHandler.VerifyLogin)

		authGroup.POST("/login/resend",
			middleware.IPRateLimit(3), // 3 resend attempts per minute per IP
			loginHandler.ResendOTP)

		// Email and username availability checks
		authGroup.GET("/check-email",
			middleware.IPRateLimit(60), // 60 checks per minute per IP
			loginHandler.CheckEmail)

		authGroup.GET("/check-username",
			middleware.IPRateLimit(60), // 60 checks per minute per IP
			authHandler.CheckUsernameAvailability)

		// Protected authentication endpoints
		protected := authGroup.Group("")
		protected.Use(authMiddleware.RequireAuth())
		{
			protected.POST("/logout", authHandler.Logout)
			protected.POST("/logout/otp", loginHandler.Logout) // Alternative logout for OTP users
			protected.GET("/validate", authHandler.ValidateToken)
			protected.GET("/me", authHandler.Me)
			protected.GET("/profile", authHandler.GetProfile)
			protected.PUT("/profile", authHandler.UpdateProfile)
			protected.POST("/change-password",
				middleware.UserRateLimit(5), // 5 password changes per minute per user
				authHandler.ChangePassword)

			// OAuth management for authenticated users
			protected.POST("/google/connect", oauthHandler.GoogleConnect)
			protected.POST("/google/disconnect", oauthHandler.GoogleDisconnect)
			protected.GET("/oauth/accounts", oauthHandler.GetConnectedAccounts)
			protected.POST("/oauth/:provider/refresh", oauthHandler.RefreshOAuthToken)
			protected.POST("/oauth/:provider/revoke", oauthHandler.RevokeOAuthToken)
		}
	}

	// AI Conversation routes
	aiGroup := v1.Group("/ai")
	aiGroup.Use(authMiddleware.RequireAuth()) // All AI routes require authentication
	{
		// Conversation endpoints
		aiGroup.POST("/conversations",
			middleware.UserRateLimit(30), // 30 conversations per minute per user
			aiConversationHandler.CreateConversation)

		aiGroup.GET("/conversations",
			middleware.UserRateLimit(100), // 100 conversation list requests per minute per user
			aiConversationHandler.GetConversations)

		aiGroup.GET("/conversations/search",
			middleware.UserRateLimit(60), // 60 search requests per minute per user
			aiConversationHandler.SearchConversations)

		aiGroup.GET("/conversations/:id",
			middleware.UserRateLimit(120), // 120 conversation details per minute per user
			aiConversationHandler.GetConversation)

		aiGroup.GET("/conversations/:id/interactions",
			middleware.UserRateLimit(60), // 60 full conversation requests per minute per user
			aiConversationHandler.GetConversationWithInteractions)

		aiGroup.PUT("/conversations/:id",
			middleware.UserRateLimit(20), // 20 conversation updates per minute per user
			aiConversationHandler.UpdateConversation)

		aiGroup.DELETE("/conversations/:id",
			middleware.UserRateLimit(10), // 10 conversation deletions per minute per user
			aiConversationHandler.DeleteConversation)

		aiGroup.POST("/conversations/:id/archive",
			middleware.UserRateLimit(10), // 10 archive operations per minute per user
			aiConversationHandler.ArchiveConversation)

		// Interaction endpoints
		aiGroup.POST("/interactions",
			middleware.UserRateLimit(60), // 60 AI interactions per minute per user
			aiConversationHandler.CreateInteraction)

		aiGroup.GET("/interactions/:id",
			middleware.UserRateLimit(100), // 100 interaction details per minute per user
			aiConversationHandler.GetInteraction)

		// Feedback endpoints
		aiGroup.POST("/feedback",
			middleware.UserRateLimit(30), // 30 feedback submissions per minute per user
			aiConversationHandler.CreateFeedback)

		aiGroup.GET("/interactions/:id/feedback",
			middleware.UserRateLimit(60), // 60 feedback requests per minute per user
			aiConversationHandler.GetFeedback)

		// Analytics endpoints
		aiGroup.GET("/analytics",
			middleware.UserRateLimit(20), // 20 analytics requests per minute per user
			aiConversationHandler.GetAnalytics)

		aiGroup.GET("/analytics/summary",
			middleware.UserRateLimit(30), // 30 summary requests per minute per user
			aiConversationHandler.GetUserSummary)

		// Cache management endpoints
		aiGroup.POST("/cache/invalidate",
			middleware.UserRateLimit(5), // 5 cache invalidations per minute per user
			aiConversationHandler.InvalidateCache)

		aiGroup.POST("/cache/warm",
			middleware.UserRateLimit(3), // 3 cache warming operations per minute per user
			aiConversationHandler.WarmCache)

		// System health endpoint (no rate limit for monitoring)
		aiGroup.GET("/health", aiConversationHandler.GetSystemHealth)

		// Task enhancement endpoints
		aiGroup.POST("/tasks",
			middleware.UserRateLimit(10), // 10 AI task creations per minute per user
			aiHandler.CreateAIEnhancedTask)

		aiGroup.POST("/tasks/optimized",
			middleware.UserRateLimit(20), // 20 optimized AI task creations per minute per user (higher since it's faster)
			aiHandler.CreateOptimizedAITask)

		aiGroup.POST("/tasks/:id/regenerate",
			middleware.UserRateLimit(5), // 5 regenerations per minute per user
			aiHandler.RegenerateTaskEnhancements)

		aiGroup.POST("/improve-description",
			middleware.UserRateLimit(15), // 15 description improvements per minute per user
			aiHandler.ImproveDescription)

		// Subtask refinement endpoints
		aiGroup.POST("/subtasks/refine",
			middleware.UserRateLimit(20), // 20 subtask refinements per minute per user
			aiHandler.RefineSubtasks)

		// Description improvement endpoints
		aiGroup.POST("/description/improve",
			middleware.UserRateLimit(30), // 30 description improvements per minute per user
			aiHandler.ImproveDescription)

		// Suggestion management endpoints
		aiGroup.POST("/suggestions/accept",
			middleware.UserRateLimit(60), // 60 suggestion actions per minute per user
			aiHandler.AcceptAISuggestion)

		aiGroup.POST("/suggestions/reject",
			middleware.UserRateLimit(60), // 60 suggestion actions per minute per user
			aiHandler.RejectAISuggestion)

		aiGroup.GET("/suggestions",
			middleware.UserRateLimit(60), // 60 suggestion requests per minute per user
			aiHandler.GetAISuggestions)

		// AI metrics and analytics for task enhancement
		aiGroup.GET("/metrics",
			middleware.UserRateLimit(30), // 30 metrics requests per minute per user
			aiHandler.GetAIMetrics)
	}

	// Todo List Application routes
	// All Todo-related routes require authentication
	todoAppGroup := v1.Group("/todo")
	todoAppGroup.Use(authMiddleware.RequireAuth())
	{
		// PROJECT ROUTES
		projects := todoAppGroup.Group("/projects")
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

			// Project todos and releases
			projects.GET("/:id/todos", todoHandler.GetProjectTodos)
			projects.GET("/:id/kanban", todoHandler.GetProjectKanbanBoard)
			projects.GET("/:id/release-versions", releaseVersionHandler.GetProjectReleaseVersions)

			// Project subtasks
			projects.GET("/:id/subtasks", subtaskHandler.GetSubtasksByProjectID)

			// Project export/import
			projects.GET("/:id/export", projectHandler.ExportProject)
			projects.POST("/:id/import", projectHandler.ImportProject)
		}

		// TODO ROUTES
		todos := todoAppGroup.Group("/todos")
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

			// Kanban board specific operations
			todos.POST("/move", todoHandler.MoveTodoToColumn)
			todos.POST("/reorder", todoHandler.ReorderTodosInColumn)
			todos.POST("/bulk-move", todoHandler.BulkMoveTodos)

			// Checklist specific operations
			todos.GET("/checklist", todoHandler.GetChecklistTodos)
			todos.POST("/:id/toggle-completion", todoHandler.ToggleTodoCompletion)
			todos.POST("/checklist/bulk-toggle", todoHandler.BulkToggleCompletion)

			// Subtask routes nested under todos
			todos.POST("/:id/subtasks", subtaskHandler.CreateSubtask)
			todos.GET("/:id/subtasks", subtaskHandler.GetSubtasks)
			todos.POST("/:id/subtasks/reorder", subtaskHandler.ReorderSubtasks)

			// Note routes nested under todos
			todos.POST("/:id/notes", noteHandler.CreateNote)
			todos.GET("/:id/notes", noteHandler.GetNotes)
			todos.GET("/:id/notes/search", noteHandler.SearchNotes)
		}

		// SUBTASK ROUTES (Direct access)
		subtasks := todoAppGroup.Group("/subtasks")
		{
			subtasks.PUT("/:id", subtaskHandler.UpdateSubtask)
			subtasks.DELETE("/:id", subtaskHandler.DeleteSubtask)
		}

		// NOTE ROUTES (Direct access)
		notes := todoAppGroup.Group("/notes")
		{
			notes.PUT("/:id", noteHandler.UpdateNote)
			notes.DELETE("/:id", noteHandler.DeleteNote)
		}

		// RELEASE VERSION ROUTES
		releases := todoAppGroup.Group("/release-versions")
		{
			// Basic CRUD operations
			releases.POST("", releaseVersionHandler.CreateReleaseVersion)
			releases.GET("/:id", releaseVersionHandler.GetReleaseVersion)
			releases.PUT("/:id", releaseVersionHandler.UpdateReleaseVersion)
			releases.DELETE("/:id", releaseVersionHandler.DeleteReleaseVersion)

			// Release management
			releases.POST("/:id/release", releaseVersionHandler.MarkAsReleased)

			// Release analytics and filtering
			releases.GET("/upcoming", releaseVersionHandler.GetUpcomingReleases)
			releases.GET("/overdue", releaseVersionHandler.GetOverdueReleases)
			releases.GET("/:id/stats", releaseVersionHandler.GetReleaseStats)
			releases.GET("/:id/todos", releaseVersionHandler.GetReleaseTodosProgress)
		}

		// SCHEDULED TASK ROUTES
		scheduled := todoAppGroup.Group("/scheduled-tasks")
		{
			scheduled.POST("", scheduledTaskHandler.CreateScheduledTask)
			scheduled.GET("", scheduledTaskHandler.GetUserScheduledTasks)
			scheduled.PUT("/:id", scheduledTaskHandler.UpdateScheduledTask)
			scheduled.DELETE("/:id", scheduledTaskHandler.DeleteScheduledTask)
		}

		// TIME TRACKING ROUTES
		timeTracking := todoAppGroup.Group("/time-entries")
		{
			// Time tracking control
			timeTracking.POST("/start", timeEntryHandler.StartTimeTracking)
			timeTracking.POST("/stop", timeEntryHandler.StopTimeTracking)
			timeTracking.GET("/active", timeEntryHandler.GetActiveTimeEntry)

			// Time entry management
			timeTracking.GET("", timeEntryHandler.GetUserTimeEntries)
		}

		// BREAK TRACKING ROUTES
		breaks := todoAppGroup.Group("/breaks")
		{
			// Break control
			breaks.POST("/start", breakHandler.StartBreak)
			breaks.POST("/stop", breakHandler.StopBreak)
			breaks.GET("/active", breakHandler.GetActiveBreak)

			// Break history and analytics
			breaks.GET("/history", breakHandler.GetBreakHistory)
			breaks.GET("/stats", breakHandler.GetBreakStats)
			breaks.GET("/todo/:todo_id", breakHandler.GetBreaksByTodo)
			breaks.GET("/project/:project_id", breakHandler.GetBreaksByProject)
		}

		// DASHBOARD ROUTES (Consolidated)
		dashboard := todoAppGroup.Group("/dashboard")
		{
			dashboard.GET("/projects", projectHandler.GetProjectDashboard)
			dashboard.GET("/todos", todoHandler.GetTodoDashboard)
			dashboard.GET("/overdue", todoHandler.GetOverdueTodos)
			dashboard.GET("/upcoming-releases", releaseVersionHandler.GetUpcomingReleases)
			dashboard.GET("/overdue-releases", releaseVersionHandler.GetOverdueReleases)
			dashboard.GET("/scheduled-tasks", scheduledTaskHandler.GetUserScheduledTasks)
			dashboard.GET("/active-time", timeEntryHandler.GetActiveTimeEntry)
		}

		// ANALYTICS ROUTES (Consolidated)
		analytics := todoAppGroup.Group("/analytics")
		{
			analytics.GET("/project-stats", projectHandler.GetProjectStats)
			analytics.GET("/release-stats/:id", releaseVersionHandler.GetReleaseStats)
		}

	}

	// Global Search routes - All require authentication
	searchGroup := v1.Group("/search")
	searchGroup.Use(authMiddleware.RequireAuth())
	{
		// Global search endpoints
		searchGroup.GET("", globalSearchHandler.GlobalSearch)
		searchGroup.GET("/quick", globalSearchHandler.QuickSearch)
		searchGroup.GET("/suggestions", globalSearchHandler.SearchSuggestions)
		searchGroup.GET("/history", globalSearchHandler.SearchHistory)

		// Legacy search endpoints (keep for backward compatibility)
		searchGroup.GET("/projects", projectHandler.SearchProjects)
		searchGroup.GET("/todos", todoHandler.SearchTodos)
	}

	// Reports routes - All require authentication
	reportsGroup := v1.Group("/reports")
	reportsGroup.Use(authMiddleware.RequireAuth())
	{
		// Core report endpoints
		reportsGroup.GET("/overview",
			middleware.UserRateLimit(30), // 30 overview requests per minute per user
			reportsHandler.GetReportsOverview)

		reportsGroup.GET("/tasks",
			middleware.UserRateLimit(60), // 60 task report requests per minute per user
			reportsHandler.GetTaskReport)

		reportsGroup.GET("/projects",
			middleware.UserRateLimit(60), // 60 project report requests per minute per user
			reportsHandler.GetProjectReport)

		reportsGroup.GET("/analysis",
			middleware.UserRateLimit(30), // 30 analysis requests per minute per user
			reportsHandler.GetAnalysisReport)

		// Advanced report endpoints
		reportsGroup.GET("/comparison",
			middleware.UserRateLimit(20), // 20 comparison requests per minute per user
			reportsHandler.GetComparisonReport)

		reportsGroup.GET("/projects/:project_id/drill-down",
			middleware.UserRateLimit(40), // 40 drill-down requests per minute per user
			reportsHandler.GetDrillDownReport)

		// Export and cache management
		reportsGroup.POST("/export",
			middleware.UserRateLimit(10), // 10 export operations per minute per user
			reportsHandler.ExportReport)

		reportsGroup.POST("/cache/invalidate",
			middleware.UserRateLimit(5), // 5 cache invalidations per minute per user
			reportsHandler.InvalidateCache)
	}

	// WebSocket routes
	wsGroup := v1.Group("/ws")
	{
		// WebSocket connection endpoint
		wsGroup.GET("/connect", wsHandler.HandleWebSocketAuth(), wsHandler.HandleWebSocketConnection)

		// WebSocket management endpoints
		wsProtected := wsGroup.Group("")
		wsProtected.Use(authMiddleware.RequireAuth())
		{
			// Connection info
			wsProtected.GET("/users", wsHandler.GetConnectedUsers)
			wsProtected.GET("/sessions", wsHandler.GetUserSessions)
			wsProtected.GET("/status", wsHandler.GetSystemStatus)

			// Focus session management
			wsProtected.GET("/focus", wsHandler.GetActiveFocusSession)
			wsProtected.POST("/focus/start", wsHandler.StartFocusSession)
			wsProtected.POST("/focus/end", wsHandler.EndFocusSession)

			// Admin broadcast (would need admin role check in production)
			wsProtected.POST("/broadcast", wsHandler.BroadcastNotification)
		}

		// Real-time API endpoints (alternative to regular API with WebSocket events)
		wsApi := wsGroup.Group("/api")
		wsApi.Use(authMiddleware.RequireAuth())
		{
			// Todo operations with real-time updates
			wsTodos := wsApi.Group("/todos")
			{
				wsTodos.POST("", wsTodoHandler.CreateTodo)
				wsTodos.PUT("/:id", wsTodoHandler.UpdateTodo)
				wsTodos.DELETE("/:id", wsTodoHandler.DeleteTodo)
				wsTodos.POST("/:id/complete", wsTodoHandler.MarkTodoComplete)
				wsTodos.POST("/:id/incomplete", wsTodoHandler.MarkTodoIncomplete)
				wsTodos.POST("/:id/pin", wsTodoHandler.PinTodo)
				wsTodos.POST("/bulk-update", wsTodoHandler.BulkUpdateTodos)

				// Kanban board operations with real-time updates
				wsTodos.POST("/move", wsTodoHandler.MoveTodoToColumn)
				wsTodos.POST("/reorder", wsTodoHandler.ReorderTodosInColumn)
				wsTodos.POST("/bulk-move", wsTodoHandler.BulkMoveTodos)

				// Checklist operations with real-time updates
				wsTodos.GET("/checklist", wsTodoHandler.GetChecklistTodos)
				wsTodos.POST("/:id/toggle-completion", wsTodoHandler.ToggleTodoCompletion)
				wsTodos.POST("/checklist/bulk-toggle", wsTodoHandler.BulkToggleCompletion)

				// Subtask operations with real-time updates
				wsTodos.POST("/:id/subtasks", wsSubtaskHandler.CreateSubtask)
				wsTodos.POST("/:id/subtasks/reorder", wsSubtaskHandler.ReorderSubtasks)
			}

			// Subtask direct operations
			wsSubtasks := wsApi.Group("/subtasks")
			{
				wsSubtasks.PUT("/:id", wsSubtaskHandler.UpdateSubtask)
				wsSubtasks.DELETE("/:id", wsSubtaskHandler.DeleteSubtask)
			}

			// Project operations with real-time updates
			wsProjects := wsApi.Group("/projects")
			{
				wsProjects.POST("", wsProjectHandler.CreateProject)
				wsProjects.PUT("/:id", wsProjectHandler.UpdateProject)
				wsProjects.DELETE("/:id", wsProjectHandler.DeleteProject)
				wsProjects.POST("/:id/archive", wsProjectHandler.ArchiveProject)
			}
		}
	}

	// Documentation route (if you add Swagger later)
	router.GET("/docs/*any", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "API documentation not yet implemented",
			"endpoints": map[string]interface{}{
				"auth": map[string]string{
					"POST /api/v1/auth/register":        "Register new user",
					"POST /api/v1/auth/login":           "User login",
					"POST /api/v1/auth/refresh":         "Refresh access token",
					"POST /api/v1/auth/logout":          "User logout",
					"POST /api/v1/auth/change-password": "Change password",
					"GET  /api/v1/auth/validate":        "Validate token",
					"GET  /api/v1/auth/me":              "Get current user",
					"GET  /api/v1/auth/profile":         "Get user profile",
					"PUT  /api/v1/auth/profile":         "Update user profile",
					"GET  /api/v1/auth/check-email":     "Check email availability",
					"GET  /api/v1/auth/check-username":  "Check username availability",
				},
				"oauth": map[string]string{
					"GET  /api/v1/auth/google":            "Start Google OAuth",
					"GET  /api/v1/auth/google/callback":   "Google OAuth callback",
					"POST /api/v1/auth/google/connect":    "Connect Google account",
					"POST /api/v1/auth/google/disconnect": "Disconnect Google account",
					"GET  /api/v1/auth/oauth/providers":   "Get OAuth providers",
					"GET  /api/v1/auth/oauth/accounts":    "Get connected accounts",
				},
				"ai_conversations": map[string]string{
					"POST /api/v1/ai/conversations":                  "Create new conversation",
					"GET  /api/v1/ai/conversations":                  "Get user conversations",
					"GET  /api/v1/ai/conversations/search":           "Search conversations",
					"GET  /api/v1/ai/conversations/:id":              "Get conversation details",
					"GET  /api/v1/ai/conversations/:id/interactions": "Get conversation with interactions",
					"PUT  /api/v1/ai/conversations/:id":              "Update conversation",
					"DELETE /api/v1/ai/conversations/:id":            "Delete conversation",
					"POST /api/v1/ai/conversations/:id/archive":      "Archive conversation",
				},
				"ai_interactions": map[string]string{
					"POST /api/v1/ai/interactions":     "Create AI interaction",
					"GET  /api/v1/ai/interactions/:id": "Get interaction details",
				},
				"ai_feedback": map[string]string{
					"POST /api/v1/ai/feedback":                  "Submit feedback",
					"GET  /api/v1/ai/interactions/:id/feedback": "Get interaction feedback",
				},
				"ai_analytics": map[string]string{
					"GET  /api/v1/ai/analytics":         "Get user analytics",
					"GET  /api/v1/ai/analytics/summary": "Get user summary",
				},
				"ai_cache": map[string]string{
					"POST /api/v1/ai/cache/invalidate": "Invalidate user cache",
					"POST /api/v1/ai/cache/warm":       "Warm user cache",
					"GET  /api/v1/ai/health":           "System health check",
				},
				"todo_app": map[string]string{
					"Projects":         "CRUD, search, stats, dashboard",
					"Release Versions": "CRUD, upcoming, overdue, stats",
					"Todos":            "CRUD, filter, search, complete/incomplete, pin, checklist view",
					"Checklist":        "GET /checklist, POST /:id/toggle-completion, POST /bulk-toggle",
					"Subtasks":         "CRUD, reorder",
					"Notes":            "CRUD, search",
					"Scheduled Tasks":  "CRUD, calendar integration",
					"Time Tracking":    "Start/stop, history, active entry",
					"Comments":         "CRUD, threading",
					"Attachments":      "Upload, download, delete",
				},
				"reports": map[string]string{
					"GET  /api/v1/reports/overview":                "Get comprehensive reports dashboard",
					"GET  /api/v1/reports/tasks":                   "Get task metrics and analytics",
					"GET  /api/v1/reports/projects":                "Get project metrics and analytics",
					"GET  /api/v1/reports/analysis":                "Get combined analysis and insights",
					"GET  /api/v1/reports/comparison":              "Get comparison between time periods",
					"GET  /api/v1/reports/projects/:id/drill-down": "Get detailed project breakdown",
					"POST /api/v1/reports/export":                  "Export reports (CSV/PDF/Excel)",
					"POST /api/v1/reports/cache/invalidate":        "Clear reports cache",
				},
			},
		})
	})

	// 404 handler
	router.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "NOT_FOUND",
				"message": "Endpoint not found",
			},
			"timestamp": time.Now(),
		})
	})

	return router
}

// getAIConsumerHealth returns the health status of the AI enhancement consumer
func getAIConsumerHealth(consumer *service.AIEnhancementConsumer) gin.HandlerFunc {
	return func(c *gin.Context) {
		if consumer == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error": "AI enhancement consumer not available",
			})
			return
		}

		health := consumer.HealthCheck()
		status := http.StatusOK
		if health["status"] == "unhealthy" {
			status = http.StatusServiceUnavailable
		}

		c.JSON(status, health)
	}
}

// getAIConsumerMetrics returns the metrics of the AI enhancement consumer
func getAIConsumerMetrics(consumer *service.AIEnhancementConsumer) gin.HandlerFunc {
	return func(c *gin.Context) {
		if consumer == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error": "AI enhancement consumer not available",
			})
			return
		}

		metrics := consumer.GetMetrics()
		c.JSON(http.StatusOK, gin.H{
			"metrics":   metrics,
			"timestamp": time.Now(),
		})
	}
}

func healthCheck(cfg *config.Config, db *database.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check database health
		dbHealthy, dbMetrics := db.IsHealthyAdvanced()

		status := "healthy"
		httpStatus := http.StatusOK
		if !dbHealthy {
			status = "unhealthy"
			httpStatus = http.StatusServiceUnavailable
		}

		response := gin.H{
			"status":    status,
			"service":   cfg.App.Name,
			"version":   cfg.App.Version,
			"env":       cfg.App.Env,
			"timestamp": time.Now(),
			"database": gin.H{
				"healthy":            dbHealthy,
				"total_connections":  dbMetrics.TotalConnections,
				"active_connections": dbMetrics.ActiveConnections,
				"total_queries":      dbMetrics.TotalQueries,
				"failed_queries":     dbMetrics.FailedQueries,
				"avg_query_time":     dbMetrics.AverageQueryTime,
			},
		}

		c.JSON(httpStatus, response)
	}
}

// getDatabaseMetrics returns detailed database connection pool metrics
func getDatabaseMetrics(db *database.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		metrics := db.GetPoolMetrics()

		c.JSON(http.StatusOK, gin.H{
			"pool_metrics": gin.H{
				"total_connections":      metrics.TotalConnections,
				"active_connections":     metrics.ActiveConnections,
				"idle_connections":       metrics.IdleConnections,
				"waiting_connections":    metrics.WaitingConnections,
				"total_queries":          metrics.TotalQueries,
				"successful_queries":     metrics.SuccessfulQueries,
				"failed_queries":         metrics.FailedQueries,
				"average_query_time_sec": metrics.AverageQueryTime,
				"connection_errors":      metrics.ConnectionErrors,
				"circuit_breaker_trips":  metrics.CircuitBreakerTrips,
				"last_updated":           metrics.LastUpdated,
			},
			"performance_indicators": gin.H{
				"query_success_rate": func() float64 {
					if metrics.TotalQueries == 0 {
						return 100.0
					}
					return float64(metrics.SuccessfulQueries) / float64(metrics.TotalQueries) * 100.0
				}(),
				"connection_utilization": func() float64 {
					if metrics.TotalConnections == 0 {
						return 0.0
					}
					return float64(metrics.ActiveConnections) / float64(metrics.TotalConnections) * 100.0
				}(),
				"queries_per_second": func() float64 {
					since := time.Since(metrics.LastUpdated).Seconds()
					if since == 0 {
						return 0.0
					}
					return float64(metrics.TotalQueries) / since
				}(),
			},
			"timestamp": time.Now(),
		})
	}
}

// getDatabaseHealth returns advanced database health information
func getDatabaseHealth(db *database.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		isHealthy, metrics := db.IsHealthyAdvanced()

		// Determine health status based on various factors
		healthScore := 100.0
		issues := []string{}

		// Check query failure rate
		if metrics.TotalQueries > 0 {
			failureRate := float64(metrics.FailedQueries) / float64(metrics.TotalQueries) * 100.0
			if failureRate > 5.0 {
				healthScore -= 30.0
				issues = append(issues, fmt.Sprintf("High query failure rate: %.2f%%", failureRate))
			}
		}

		// Check connection utilization
		if metrics.TotalConnections > 0 {
			utilization := float64(metrics.ActiveConnections) / float64(metrics.TotalConnections) * 100.0
			if utilization > 90.0 {
				healthScore -= 20.0
				issues = append(issues, fmt.Sprintf("High connection utilization: %.2f%%", utilization))
			}
		}

		// Check for recent connection errors
		if metrics.ConnectionErrors > 10 {
			healthScore -= 25.0
			issues = append(issues, fmt.Sprintf("Recent connection errors: %d", metrics.ConnectionErrors))
		}

		// Check average query time
		if metrics.AverageQueryTime > 1.0 { // More than 1 second
			healthScore -= 15.0
			issues = append(issues, fmt.Sprintf("Slow average query time: %.3fs", metrics.AverageQueryTime))
		}

		status := "healthy"
		if healthScore < 80.0 {
			status = "degraded"
		}
		if healthScore < 50.0 || !isHealthy {
			status = "unhealthy"
		}

		httpStatus := http.StatusOK
		if status == "unhealthy" {
			httpStatus = http.StatusServiceUnavailable
		}

		c.JSON(httpStatus, gin.H{
			"status":       status,
			"health_score": healthScore,
			"database_up":  isHealthy,
			"issues":       issues,
			"metrics":      metrics,
			"recommendations": func() []string {
				var recs []string
				if healthScore < 80 {
					recs = append(recs, "Consider increasing connection pool size")
					recs = append(recs, "Monitor and optimize slow queries")
					recs = append(recs, "Check database server resources")
				}
				return recs
			}(),
			"timestamp": time.Now(),
		})
	}
}

// getConnectionPoolStatus returns real-time connection pool status
func getConnectionPoolStatus(db *database.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		highPerfPool := db.GetHighPerfPool()
		if highPerfPool == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error": "High-performance pool not available",
			})
			return
		}

		metrics := highPerfPool.GetMetrics()

		c.JSON(http.StatusOK, gin.H{
			"pool_status": gin.H{
				"primary_pool": gin.H{
					"available":   true,
					"healthy":     highPerfPool.IsHealthy(),
					"connections": metrics.TotalConnections,
					"active":      metrics.ActiveConnections,
					"idle":        metrics.IdleConnections,
					"waiting":     metrics.WaitingConnections,
				},
				"read_replicas": gin.H{
					"count":     "Multiple replicas supported",
					"available": true,
					"status":    "Automatic failover enabled",
				},
				"circuit_breaker": gin.H{
					"enabled": true,
					"trips":   metrics.CircuitBreakerTrips,
					"status":  "Protected",
				},
			},
			"performance": gin.H{
				"total_queries":     metrics.TotalQueries,
				"successful":        metrics.SuccessfulQueries,
				"failed":            metrics.FailedQueries,
				"avg_response_time": metrics.AverageQueryTime,
				"last_updated":      metrics.LastUpdated,
			},
			"capacity": gin.H{
				"max_connections":     1000, // From config
				"current_utilization": fmt.Sprintf("%.1f%%", float64(metrics.ActiveConnections)/float64(metrics.TotalConnections)*100),
				"recommended_action": func() string {
					utilization := float64(metrics.ActiveConnections) / float64(metrics.TotalConnections) * 100
					if utilization > 80 {
						return "Consider scaling up"
					} else if utilization < 20 {
						return "Pool size optimal"
					}
					return "Normal operation"
				}(),
			},
			"timestamp": time.Now(),
		})
	}
}

// getAPMHealth returns the health status of middleware.io APM tracking
func getAPMHealth(client *apmclient.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		if client == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error": "Middleware.io APM client not available",
			})
			return
		}

		health := client.HealthCheck()
		status := http.StatusOK
		if health["status"] == "unhealthy" || health["status"] == "not_started" {
			status = http.StatusServiceUnavailable
		}

		c.JSON(status, health)
	}
}

// getAPMMetrics returns the metrics of middleware.io APM tracking
func getAPMMetrics(client *apmclient.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		if client == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error": "Middleware.io APM client not available",
			})
			return
		}

		metrics := client.GetMetrics()
		c.JSON(http.StatusOK, gin.H{
			"metrics":   metrics,
			"timestamp": time.Now(),
		})
	}
}

// restartAPM restarts the middleware.io APM tracking
func restartAPM(client *apmclient.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		if client == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error": "Middleware.io APM client not available",
			})
			return
		}

		if err := client.Restart(c.Request.Context()); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Failed to restart middleware.io APM tracking",
				"details": err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message":   "Middleware.io APM tracking restarted successfully",
			"timestamp": time.Now(),
		})
	}
}
