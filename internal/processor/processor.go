package processor

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"time"

	"github.com/Ucell/push-service/config"
	"github.com/Ucell/push-service/internal/push"

	"github.com/IBM/sarama"
)

// PushSender interface for dependency injection
type PushSender interface {
	Send(ctx context.Context, msg *push.PushMessage) (string, error)
}

type Processor struct {
	sender      PushSender
	retryConfig config.RetryConfig
}

// NewProcessor creates a new message processor
func NewProcessor(sender PushSender, retryConfig config.RetryConfig) *Processor {
	return &Processor{
		sender:      sender,
		retryConfig: retryConfig,
	}
}

// Process handles incoming Kafka message and sends push notification
func (p *Processor) Process(ctx context.Context, msg *sarama.ConsumerMessage) error {
	log.Printf("Processing message from topic=%s partition=%d offset=%d",
		msg.Topic, msg.Partition, msg.Offset)

	// Parse message
	var pushMsg push.PushMessage
	if err := json.Unmarshal(msg.Value, &pushMsg); err != nil {
		log.Printf("Failed to unmarshal message: %v", err)
		return fmt.Errorf("invalid message format: %w", err)
	}

	// Validate message
	if pushMsg.EndpointARN == "" {
		return fmt.Errorf("endpoint_arn is required")
	}
	if pushMsg.Body == "" {
		return fmt.Errorf("body is required")
	}

	// Send push with retry
	return p.sendWithRetry(ctx, &pushMsg)
}

// sendWithRetry attempts to send push notification with exponential backoff
func (p *Processor) sendWithRetry(ctx context.Context, msg *push.PushMessage) error {
	var lastErr error

	for attempt := 0; attempt < p.retryConfig.MaxAttempts; attempt++ {
		if attempt > 0 {
			// Calculate backoff delay with exponential increase
			delay := p.calculateBackoff(attempt)
			log.Printf("Retrying after %v (attempt %d/%d)",
				delay, attempt+1, p.retryConfig.MaxAttempts)

			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		// Attempt to send
		messageID, err := p.sender.Send(ctx, msg)
		if err == nil {
			log.Printf("Push sent successfully on attempt %d. MessageID: %s",
				attempt+1, messageID)
			return nil
		}

		lastErr = err
		log.Printf("Failed to send push (attempt %d/%d): %v",
			attempt+1, p.retryConfig.MaxAttempts, err)

		// Check if error is retryable
		if !push.IsRetryableError(err) {
			log.Printf("Error is not retryable, giving up")
			return err
		}
	}

	return fmt.Errorf("failed after %d attempts: %w", p.retryConfig.MaxAttempts, lastErr)
}

// calculateBackoff calculates exponential backoff delay
func (p *Processor) calculateBackoff(attempt int) time.Duration {
	// Exponential backoff: initialDelay * 2^attempt
	delay := time.Duration(float64(p.retryConfig.InitialDelay) * math.Pow(2, float64(attempt)))

	// Cap at max delay
	if delay > p.retryConfig.MaxDelay {
		delay = p.retryConfig.MaxDelay
	}

	return delay
}
