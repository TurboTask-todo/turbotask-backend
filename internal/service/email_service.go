package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"quantumtask-auth-api/internal/models"
	"quantumtask-auth-api/pkg/email"
	"quantumtask-auth-api/pkg/queue"
)

const (
	EmailQueueName = "email_queue"
)

// EmailService handles background email processing
type EmailService struct {
	emailClient *email.Client
	queueClient queue.Client
}

// NewEmailService creates a new email service
func NewEmailService(emailClient *email.Client, queueClient queue.Client) *EmailService {
	return &EmailService{
		emailClient: emailClient,
		queueClient: queueClient,
	}
}

// QueueOTPEmail queues an OTP email for background processing
func (s *EmailService) QueueOTPEmail(ctx context.Context, email, code string, purpose models.OTPPurpose, expiresIn string) error {
	messageID := fmt.Sprintf("otp_%d_%s", time.Now().UnixNano(), email)

	emailMessage := models.EmailMessage{
		MessageID: messageID,
		To:        email,
		Subject:   getOTPSubject(purpose),
		Type:      models.EmailTypeOTP,
		Data: map[string]string{
			"code":       code,
			"email":      email,
			"purpose":    string(purpose),
			"expires_in": expiresIn,
		},
		Priority:   getEmailPriority(purpose),
		RetryCount: 0,
		MaxRetries: 3,
		QueuedAt:   time.Now(),
	}

	// Create email queue if it doesn't exist
	if err := s.queueClient.CreateQueue(EmailQueueName, queue.QueueConfig{
		Durable:    true,
		AutoDelete: false,
		Exclusive:  false,
		MaxLength:  10000,
		TTL:        24 * time.Hour, // 24 hours TTL
	}); err != nil {
		// Log error but don't fail - queue might already exist
		fmt.Printf("Warning: Could not create email queue: %v\n", err)
	}

	// Queue the email message with high priority for OTP emails
	return s.queueClient.Publish(ctx, EmailQueueName, emailMessage, emailMessage.Priority)
}

// ProcessEmailMessage processes a queued email message
func (s *EmailService) ProcessEmailMessage(ctx context.Context, body []byte) error {
	var emailMessage models.EmailMessage
	if err := json.Unmarshal(body, &emailMessage); err != nil {
		return fmt.Errorf("failed to unmarshal email message: %w", err)
	}

	fmt.Printf("📧 Processing email message: %s (type: %s, to: %s)\n",
		emailMessage.MessageID, emailMessage.Type, emailMessage.To)

	var err error
	switch emailMessage.Type {
	case models.EmailTypeOTP:
		err = s.processOTPEmail(emailMessage)
	default:
		err = fmt.Errorf("unknown email type: %s", emailMessage.Type)
	}

	if err != nil {
		emailMessage.RetryCount++
		fmt.Printf("❌ Failed to process email %s (attempt %d/%d): %v\n",
			emailMessage.MessageID, emailMessage.RetryCount, emailMessage.MaxRetries, err)

		// Retry if we haven't exceeded max retries
		if emailMessage.RetryCount < emailMessage.MaxRetries {
			// Exponential backoff: wait 2^retryCount seconds before retry
			delay := time.Duration(1<<emailMessage.RetryCount) * time.Second
			time.Sleep(delay)

			// Re-queue the message for retry
			return s.queueClient.Publish(ctx, EmailQueueName, emailMessage, emailMessage.Priority)
		}

		return fmt.Errorf("failed to process email %s after %d attempts: %w",
			emailMessage.MessageID, emailMessage.MaxRetries, err)
	}

	now := time.Now()
	emailMessage.ProcessedAt = &now
	fmt.Printf("✅ Successfully processed email: %s\n", emailMessage.MessageID)

	return nil
}

// processOTPEmail processes an OTP email
func (s *EmailService) processOTPEmail(emailMessage models.EmailMessage) error {
	data := emailMessage.Data
	purpose := models.OTPPurpose(data["purpose"])

	return s.emailClient.SendOTP(
		emailMessage.To,
		data["code"],
		purpose,
		data["expires_in"],
	)
}

// getOTPSubject returns the subject for OTP emails
func getOTPSubject(purpose models.OTPPurpose) string {
	switch purpose {
	case models.OTPPurposeLogin:
		return "🔐 Your MacWrite Login Code"
	case models.OTPPurposeRegister:
		return "🎉 Welcome to MacWrite - Verify Your Email"
	case models.OTPPurposeReset:
		return "🔄 Reset Your MacWrite Password"
	default:
		return "Your MacWrite Verification Code"
	}
}

// getEmailPriority returns priority for different email types
func getEmailPriority(purpose models.OTPPurpose) int {
	switch purpose {
	case models.OTPPurposeLogin:
		return 9 // High priority for login
	case models.OTPPurposeReset:
		return 8 // High priority for password reset
	case models.OTPPurposeRegister:
		return 7 // Medium-high priority for registration
	default:
		return 5 // Default priority
	}
}

// EmailConsumer handles email queue consumption
type EmailConsumer struct {
	emailService *EmailService
	queueClient  queue.Client
}

// NewEmailConsumer creates a new email consumer
func NewEmailConsumer(emailService *EmailService, queueClient queue.Client) *EmailConsumer {
	return &EmailConsumer{
		emailService: emailService,
		queueClient:  queueClient,
	}
}

// StartConsumer starts consuming email messages from the queue
func (c *EmailConsumer) StartConsumer(ctx context.Context) error {
	fmt.Printf("🚀 Starting email consumer...\n")

	// Create email queue if it doesn't exist
	if err := c.queueClient.CreateQueue(EmailQueueName, queue.QueueConfig{
		Durable:    true,
		AutoDelete: false,
		Exclusive:  false,
		MaxLength:  10000,
		TTL:        24 * time.Hour,
	}); err != nil {
		fmt.Printf("⚠️  Warning: Could not create email queue: %v\n", err)
	}

	// Start consuming messages
	return c.queueClient.Consume(ctx, EmailQueueName, c.emailService.ProcessEmailMessage)
}

// HealthCheck returns the health status of the email consumer
func (c *EmailConsumer) HealthCheck() map[string]interface{} {
	status := "healthy"

	// Test queue connection
	if err := c.queueClient.Ping(context.Background()); err != nil {
		status = "unhealthy"
	}

	return map[string]interface{}{
		"status":       status,
		"queue_name":   EmailQueueName,
		"service_type": "email_consumer",
		"last_checked": time.Now(),
	}
}

// GetMetrics returns metrics for the email consumer
func (c *EmailConsumer) GetMetrics() map[string]interface{} {
	return map[string]interface{}{
		"queue_name":    EmailQueueName,
		"consumer_type": "email_processor",
		"max_retries":   3,
		"last_updated":  time.Now(),
	}
}
