package unit

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/Ucell/push-service/config"
	"github.com/Ucell/push-service/internal/processor"
	"github.com/Ucell/push-service/internal/push"

	"github.com/IBM/sarama"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockSender is a mock implementation of push.Sender
type MockSender struct {
	mock.Mock
}

func (m *MockSender) Send(ctx context.Context, msg *push.PushMessage) (string, error) {
	args := m.Called(ctx, msg)
	return args.String(0), args.Error(1)
}

func TestProcessor_Process_Success(t *testing.T) {
	mockSender := new(MockSender)
	retryConfig := config.RetryConfig{
		MaxAttempts:  3,
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     1 * time.Second,
	}

	proc := processor.NewProcessor(mockSender, retryConfig)

	pushMsg := push.PushMessage{
		EndpointARN: "arn:aws:sns:ru-central1:123456:app/GCM/test-app",
		Title:       "Test Title",
		Body:        "Test Body",
	}

	msgBytes, _ := json.Marshal(pushMsg)
	kafkaMsg := &sarama.ConsumerMessage{
		Topic:     "test-topic",
		Partition: 0,
		Offset:    1,
		Value:     msgBytes,
	}

	mockSender.On("Send", mock.Anything, &pushMsg).Return("msg-123", nil)

	err := proc.Process(context.Background(), kafkaMsg)

	assert.NoError(t, err)
	mockSender.AssertExpectations(t)
}

func TestProcessor_Process_InvalidJSON(t *testing.T) {
	mockSender := new(MockSender)
	retryConfig := config.RetryConfig{
		MaxAttempts:  3,
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     1 * time.Second,
	}

	proc := processor.NewProcessor(mockSender, retryConfig)

	kafkaMsg := &sarama.ConsumerMessage{
		Topic:     "test-topic",
		Partition: 0,
		Offset:    1,
		Value:     []byte("invalid json"),
	}

	err := proc.Process(context.Background(), kafkaMsg)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid message format")
}

func TestProcessor_Process_MissingFields(t *testing.T) {
	mockSender := new(MockSender)
	retryConfig := config.RetryConfig{
		MaxAttempts:  3,
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     1 * time.Second,
	}

	proc := processor.NewProcessor(mockSender, retryConfig)

	pushMsg := push.PushMessage{
		Title: "Test Title",
		// Missing EndpointARN and Body
	}

	msgBytes, _ := json.Marshal(pushMsg)
	kafkaMsg := &sarama.ConsumerMessage{
		Topic:     "test-topic",
		Partition: 0,
		Offset:    1,
		Value:     msgBytes,
	}

	err := proc.Process(context.Background(), kafkaMsg)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "required")
}
