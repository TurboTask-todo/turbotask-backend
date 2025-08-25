package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"macwrite-auth-api/pkg/queue"
	"macwrite-auth-api/pkg/websocket"

	"github.com/google/uuid"
)

// AIEnhancementConsumer handles processing of AI enhancement messages from RabbitMQ
type AIEnhancementConsumer struct {
	aiService   *AIService
	queueClient queue.Client
	wsHub       *websocket.Hub

	// Performance and scalability improvements
	workerPool     chan struct{} // Semaphore for limiting concurrent workers
	maxWorkers     int           // Maximum number of concurrent workers
	messageBuffer  chan []byte   // Buffer for incoming messages
	bufferSize     int           // Size of the message buffer
	processingTime time.Duration // Track processing time for monitoring

	// Metrics and monitoring
	metrics struct {
		sync.RWMutex
		processedCount      int64
		errorCount          int64
		totalProcessingTime time.Duration
		lastProcessedTime   time.Time
	}
}

// NewAIEnhancementConsumer creates a new AI enhancement consumer
func NewAIEnhancementConsumer(
	aiService *AIService,
	queueClient queue.Client,
	wsHub *websocket.Hub,
) *AIEnhancementConsumer {
	return &AIEnhancementConsumer{
		aiService:   aiService,
		queueClient: queueClient,
		wsHub:       wsHub,

		// Performance configuration
		maxWorkers:    10,                      // Process up to 10 messages concurrently
		workerPool:    make(chan struct{}, 10), // Semaphore for worker control
		messageBuffer: make(chan []byte, 100),  // Buffer 100 messages
		bufferSize:    100,
	}
}

// StartConsumer starts consuming AI enhancement messages from the queue
func (c *AIEnhancementConsumer) StartConsumer(ctx context.Context) error {
	const queueName = "ai_enhancement_list_v2"

	log.Printf("🚀 Starting AI Enhancement Consumer for queue: %s", queueName)
	log.Printf("⚡ Performance settings: maxWorkers=%d, bufferSize=%d", c.maxWorkers, c.bufferSize)

	// Create queue if it doesn't exist
	log.Printf("📥 Creating/declaring queue: %s", queueName)
	err := c.queueClient.CreateQueue(queueName, queue.QueueConfig{
		Durable:    true,
		AutoDelete: false,
		Exclusive:  false,
		MaxLength:  1000,
		TTL:        time.Hour * 24, // Messages expire after 24 hours
	})
	if err != nil {
		log.Printf("❌ Failed to create queue %s: %v", queueName, err)
		return fmt.Errorf("failed to create queue: %w", err)
	}
	log.Printf("✅ Queue %s created/declared successfully", queueName)

	// Start worker pool
	c.startWorkerPool(ctx)

	// Start consuming messages
	log.Printf("👂 Starting to consume messages from queue: %s", queueName)
	err = c.queueClient.Consume(ctx, queueName, c.handleEnhancementMessage)
	if err != nil {
		log.Printf("❌ Failed to start consuming from queue %s: %v", queueName, err)
		return fmt.Errorf("failed to start consuming: %w", err)
	}

	log.Printf("✅ AI Enhancement Consumer started successfully, waiting for messages on queue: %s", queueName)
	log.Printf("🔄 Consumer is now actively listening for AI enhancement requests...")

	// Keep consumer alive - don't block on context since we use background context
	// The consumer will stay active as long as the connection is maintained
	select {
	case <-ctx.Done():
		log.Printf("🛑 Consumer context cancelled, shutting down...")
		return ctx.Err()
	case <-time.After(1 * time.Hour):
		// Restart consumer every hour for health
		log.Printf("🔄 Restarting consumer for health maintenance...")
		return nil
	}
}

// startWorkerPool starts the worker pool for concurrent message processing
func (c *AIEnhancementConsumer) startWorkerPool(ctx context.Context) {
	// Start background workers
	for i := 0; i < c.maxWorkers; i++ {
		go c.worker(ctx, i)
	}
	log.Printf("👷 Started %d worker goroutines", c.maxWorkers)
}

// worker processes messages from the buffer
func (c *AIEnhancementConsumer) worker(ctx context.Context, workerID int) {
	for {
		select {
		case <-ctx.Done():
			log.Printf("🛑 Worker %d shutting down", workerID)
			return
		case message := <-c.messageBuffer:
			// Acquire worker slot
			select {
			case c.workerPool <- struct{}{}:
				// Process the message
				c.processMessage(ctx, message, workerID)
				// Release worker slot
				<-c.workerPool
			case <-ctx.Done():
				return
			}
		}
	}
}

// processMessage processes a single message with proper error handling and metrics
func (c *AIEnhancementConsumer) processMessage(ctx context.Context, message []byte, workerID int) {
	startTime := time.Now()

	log.Printf("👷 Worker %d processing message (size: %d bytes)", workerID, len(message))

	// Parse message
	var aiMessage AIEnhancementMessage
	if err := json.Unmarshal(message, &aiMessage); err != nil {
		log.Printf("❌ Worker %d: Failed to unmarshal message: %v", workerID, err)
		c.updateMetrics(false, time.Since(startTime))
		return
	}

	log.Printf("🔍 Worker %d: Processing AI enhancement for task: %s (request: %s)",
		workerID, aiMessage.TaskID, aiMessage.RequestID)

	// Process the enhancement
	err := c.aiService.ProcessAIEnhancement(ctx, aiMessage)
	duration := time.Since(startTime)

	if err != nil {
		log.Printf("❌ Worker %d: Failed to process AI enhancement for task %s: %v (took %v)",
			workerID, aiMessage.TaskID, err, duration)

		// Send error notification via WebSocket
		c.sendWebSocketNotification(aiMessage.UserID, map[string]interface{}{
			"type":      "ai_enhancement_error",
			"task_id":   aiMessage.TaskID,
			"error":     err.Error(),
			"duration":  duration.Milliseconds(),
			"worker_id": workerID,
		})

		c.updateMetrics(false, duration)
		return
	}

	log.Printf("✅ Worker %d: Successfully processed AI enhancement for task %s (took %v)",
		workerID, aiMessage.TaskID, duration)

	// Send success notification via WebSocket
	c.sendWebSocketNotification(aiMessage.UserID, map[string]interface{}{
		"type":       "ai_enhancement_completed",
		"task_id":    aiMessage.TaskID,
		"request_id": aiMessage.RequestID,
		"duration":   duration.Milliseconds(),
		"worker_id":  workerID,
		"message":    "AI enhancement completed successfully",
	})

	// Invalidate user projects cache
	c.aiService.invalidateTodoCaches(uuid.MustParse(aiMessage.UserID), uuid.MustParse(aiMessage.ProjectID))

	c.updateMetrics(true, duration)
	log.Printf("🎉 Worker %d: AI enhancement workflow completed successfully for task: %s", workerID, aiMessage.TaskID)
}

// handleEnhancementMessage handles incoming messages and buffers them for processing
func (c *AIEnhancementConsumer) handleEnhancementMessage(ctx context.Context, body []byte) error {
	log.Printf("📨 Received AI enhancement message (size: %d bytes)", len(body))

	// Buffer the message for processing by workers
	select {
	case c.messageBuffer <- body:
		log.Printf("📥 Message buffered for processing")
		return nil
	case <-ctx.Done():
		return ctx.Err()
	default:
		// Buffer is full, process immediately in this goroutine
		log.Printf("⚠️ Buffer full, processing message immediately")
		go c.processMessage(ctx, body, -1) // -1 indicates immediate processing
		return nil
	}
}

// updateMetrics updates the consumer metrics
func (c *AIEnhancementConsumer) updateMetrics(success bool, duration time.Duration) {
	c.metrics.Lock()
	defer c.metrics.Unlock()

	if success {
		c.metrics.processedCount++
	} else {
		c.metrics.errorCount++
	}

	c.metrics.totalProcessingTime += duration
	c.metrics.lastProcessedTime = time.Now()
}

// GetMetrics returns the current consumer metrics
func (c *AIEnhancementConsumer) GetMetrics() map[string]interface{} {
	c.metrics.RLock()
	defer c.metrics.RUnlock()

	var avgProcessingTime time.Duration
	if c.metrics.processedCount > 0 {
		avgProcessingTime = c.metrics.totalProcessingTime / time.Duration(c.metrics.processedCount)
	}

	return map[string]interface{}{
		"processed_count":        c.metrics.processedCount,
		"error_count":            c.metrics.errorCount,
		"success_rate":           float64(c.metrics.processedCount) / float64(c.metrics.processedCount+c.metrics.errorCount) * 100,
		"avg_processing_time_ms": avgProcessingTime.Milliseconds(),
		"last_processed_time":    c.metrics.lastProcessedTime,
		"active_workers":         len(c.workerPool),
		"max_workers":            c.maxWorkers,
		"buffer_utilization":     float64(len(c.messageBuffer)) / float64(c.bufferSize) * 100,
	}
}

// sendWebSocketNotification sends a notification to the user via WebSocket
func (c *AIEnhancementConsumer) sendWebSocketNotification(userID string, data map[string]interface{}) {
	if c.wsHub == nil {
		fmt.Printf("❌ WebSocket hub is nil, cannot send notification\n")
		return
	}

	// Parse userID string to UUID
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		fmt.Printf("❌ Invalid user ID for WebSocket notification: %s\n", userID)
		return
	}

	// Extract notification type from data
	notificationType, ok := data["type"].(string)
	if !ok {
		notificationType = "ai_enhancement_completed"
	}

	fmt.Printf("🔔 Sending WebSocket notification to user: %s (type: %s)\n", userID, notificationType)

	// Create the message with the notification type as the main type
	message := websocket.Message{
		Type:      notificationType, // Use the specific type like "ai_enhancement_completed"
		Payload:   data,
		Timestamp: time.Now(),
		EventID:   uuid.New().String(),
		UserID:    &userUUID,
	}

	c.wsHub.BroadcastToUser(userUUID, message, nil)
	fmt.Printf("✅ WebSocket notification sent successfully to user: %s\n", userID)
}

// HealthCheck returns the health status of the consumer
func (c *AIEnhancementConsumer) HealthCheck() map[string]interface{} {
	metrics := c.GetMetrics()

	// Determine health status
	healthStatus := "healthy"
	if metrics["error_count"].(int64) > 0 {
		healthStatus = "degraded"
	}
	if metrics["error_count"].(int64) > 10 {
		healthStatus = "unhealthy"
	}

	return map[string]interface{}{
		"status":    healthStatus,
		"metrics":   metrics,
		"timestamp": time.Now(),
		"consumer":  "ai_enhancement_consumer",
	}
}
