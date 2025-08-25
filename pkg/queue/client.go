package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/streadway/amqp"
)

// Client defines the interface for queue operations
type Client interface {
	Publish(ctx context.Context, queueName string, payload interface{}, priority int) error
	Consume(ctx context.Context, queueName string, handler MessageHandler) error
	CreateQueue(queueName string, config QueueConfig) error
	Ping(ctx context.Context) error
	Close() error
}

// MessageHandler defines the function signature for message handlers
type MessageHandler func(ctx context.Context, body []byte) error

// QueueConfig defines queue configuration
type QueueConfig struct {
	Durable    bool
	AutoDelete bool
	Exclusive  bool
	MaxLength  int
	TTL        time.Duration
	Priority   int
}

// rabbitMQClient implements the Client interface
type rabbitMQClient struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	url     string
}

// NewRabbitMQClient creates a new RabbitMQ client
func NewRabbitMQClient(url string) (Client, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	return &rabbitMQClient{
		conn:    conn,
		channel: ch,
		url:     url,
	}, nil
}

// ensureConnection ensures the connection and channel are open
func (c *rabbitMQClient) ensureConnection() error {
	if c.conn == nil || c.conn.IsClosed() {
		fmt.Printf("🔄 Reconnecting to RabbitMQ...\n")
		conn, err := amqp.Dial(c.url)
		if err != nil {
			return fmt.Errorf("failed to reconnect to RabbitMQ: %w", err)
		}
		c.conn = conn
		c.channel = nil // Force channel recreation
	}

	if c.channel == nil {
		fmt.Printf("🔄 Creating new RabbitMQ channel...\n")
		ch, err := c.conn.Channel()
		if err != nil {
			return fmt.Errorf("failed to create channel: %w", err)
		}
		c.channel = ch
	}

	return nil
}

// Publish publishes a message to a queue
func (c *rabbitMQClient) Publish(ctx context.Context, queueName string, payload interface{}, priority int) error {
	fmt.Printf("📤 Publishing message to queue: %s (priority: %d)\n", queueName, priority)

	// Ensure connection is open
	if err := c.ensureConnection(); err != nil {
		fmt.Printf("❌ Failed to ensure connection: %v\n", err)
		return fmt.Errorf("failed to ensure connection: %w", err)
	}

	body, err := json.Marshal(payload)
	if err != nil {
		fmt.Printf("❌ Failed to marshal payload: %v\n", err)
		return fmt.Errorf("failed to marshal payload: %w", err)
	}
	fmt.Printf("📄 Message body size: %d bytes\n", len(body))

	// Ensure queue exists with consistent parameters (must match CreateQueue)
	fmt.Printf("📥 Declaring queue: %s\n", queueName)
	_, err = c.channel.QueueDeclare(
		queueName,
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		amqp.Table{
			"x-max-priority": 10,
			"x-message-ttl":  int64(24 * 60 * 60 * 1000), // 24 hours in milliseconds
			"x-max-length":   1000,                       // Max queue length (must match CreateQueue)
		},
	)
	if err != nil {
		fmt.Printf("❌ Failed to declare queue %s: %v\n", queueName, err)
		return fmt.Errorf("failed to declare queue: %w", err)
	}
	fmt.Printf("✅ Queue %s declared successfully with TTL\n", queueName)

	messageID := generateMessageID()
	fmt.Printf("🆔 Generated message ID: %s\n", messageID)

	err = c.channel.Publish(
		"",        // exchange
		queueName, // routing key
		false,     // mandatory
		false,     // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
			Priority:     uint8(priority),
			Timestamp:    time.Now(),
			MessageId:    messageID,
			DeliveryMode: amqp.Persistent,
		},
	)
	if err != nil {
		fmt.Printf("❌ Failed to publish message to queue %s: %v\n", queueName, err)
		return fmt.Errorf("failed to publish message: %w", err)
	}

	fmt.Printf("✅ Message published successfully to queue: %s (ID: %s)\n", queueName, messageID)
	return nil
}

// Consume consumes messages from a queue
func (c *rabbitMQClient) Consume(ctx context.Context, queueName string, handler MessageHandler) error {
	// Ensure connection is open
	if err := c.ensureConnection(); err != nil {
		fmt.Printf("❌ Failed to ensure connection for consumption: %v\n", err)
		return fmt.Errorf("failed to ensure connection: %w", err)
	}

	// Ensure queue exists with consistent parameters (must match CreateQueue)
	fmt.Printf("📥 Declaring queue for consumption: %s\n", queueName)
	_, err := c.channel.QueueDeclare(
		queueName,
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		amqp.Table{
			"x-max-priority": 10,
			"x-message-ttl":  int64(24 * 60 * 60 * 1000), // 24 hours in milliseconds
			"x-max-length":   1000,                       // Max queue length (must match CreateQueue)
		},
	)
	if err != nil {
		fmt.Printf("❌ Failed to declare queue %s for consumption: %v\n", queueName, err)
		return fmt.Errorf("failed to declare queue: %w", err)
	}
	fmt.Printf("✅ Queue %s declared successfully for consumption with TTL\n", queueName)

	// Set QoS
	err = c.channel.Qos(
		10,    // prefetch count
		0,     // prefetch size
		false, // global
	)
	if err != nil {
		return fmt.Errorf("failed to set QoS: %w", err)
	}

	msgs, err := c.channel.Consume(
		queueName,
		"",    // consumer
		false, // auto-ack
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,   // args
	)
	if err != nil {
		return fmt.Errorf("failed to register consumer: %w", err)
	}

	go func() {
		fmt.Printf("🔄 Consumer goroutine started, waiting for messages...\n")
		defer func() {
			if r := recover(); r != nil {
				fmt.Printf("🚨 Consumer goroutine panic: %v\n", r)
			}
			fmt.Printf("🛑 Consumer goroutine ended\n")
		}()

		for {
			select {
			case d, ok := <-msgs:
				if !ok {
					fmt.Printf("🛑 Message channel closed, consumer ending\n")
					return
				}

				fmt.Printf("📥 Received message: %s\n", string(d.Body))

				err := handler(ctx, d.Body)
				if err != nil {
					fmt.Printf("❌ Message processing failed, rejecting and requeuing: %v\n", err)
					// Reject and requeue the message
					if ackErr := d.Nack(false, true); ackErr != nil {
						fmt.Printf("❌ Failed to nack message: %v\n", ackErr)
					}
				} else {
					fmt.Printf("✅ Message processed successfully, acknowledging\n")
					// Acknowledge the message
					if ackErr := d.Ack(false); ackErr != nil {
						fmt.Printf("❌ Failed to ack message: %v\n", ackErr)
					}
				}
			case <-ctx.Done():
				fmt.Printf("🛑 Consumer context cancelled\n")
				return
			}
		}
	}()

	fmt.Printf("✅ Consumer goroutine launched successfully\n")
	return nil
}

// CreateQueue creates a queue with specific configuration
func (c *rabbitMQClient) CreateQueue(queueName string, config QueueConfig) error {
	// Ensure connection is open
	if err := c.ensureConnection(); err != nil {
		fmt.Printf("❌ Failed to ensure connection for queue creation: %v\n", err)
		return fmt.Errorf("failed to ensure connection: %w", err)
	}

	args := amqp.Table{
		"x-max-priority": 10,
	}

	if config.MaxLength > 0 {
		args["x-max-length"] = config.MaxLength
	}

	if config.TTL > 0 {
		args["x-message-ttl"] = int64(config.TTL / time.Millisecond)
	}

	_, err := c.channel.QueueDeclare(
		queueName,
		config.Durable,
		config.AutoDelete,
		config.Exclusive,
		false, // no-wait
		args,
	)

	return err
}

// Ping checks if RabbitMQ is available
func (c *rabbitMQClient) Ping(ctx context.Context) error {
	return c.ensureConnection()
}

// Close closes the RabbitMQ connection
func (c *rabbitMQClient) Close() error {
	if c.channel != nil {
		c.channel.Close()
	}
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// generateMessageID generates a unique message ID
func generateMessageID() string {
	return fmt.Sprintf("msg_%d", time.Now().UnixNano())
}

// MockClient is a mock implementation for testing
type MockClient struct{}

// NewMockClient creates a new mock client
func NewMockClient() Client {
	return &MockClient{}
}

func (m *MockClient) Publish(ctx context.Context, queueName string, payload interface{}, priority int) error {
	return nil
}

func (m *MockClient) Consume(ctx context.Context, queueName string, handler MessageHandler) error {
	return nil
}

func (m *MockClient) CreateQueue(queueName string, config QueueConfig) error {
	return nil
}

func (m *MockClient) Ping(ctx context.Context) error {
	return nil
}

func (m *MockClient) Close() error {
	return nil
}
