package integration

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/Ucell/push-service/config"
	"github.com/Ucell/push-service/internal/kafka"
	"github.com/Ucell/push-service/internal/processor"
	"github.com/Ucell/push-service/internal/push"

	"github.com/IBM/sarama"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Integration test requires running Kafka instance
// Run: docker-compose up -d kafka
func TestKafkaIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	brokers := []string{"localhost:9092"}
	topic := "test-push-notifications"
	groupID := "test-group"

	// Wait for Kafka to be ready
	time.Sleep(5 * time.Second)

	// Create test topic
	err := createTopic(brokers, topic)
	require.NoError(t, err)

	// Create mock sender
	mockSender := &TestSender{
		messages: make([]push.PushMessage, 0),
	}

	retryConfig := config.RetryConfig{
		MaxAttempts:  2,
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     500 * time.Millisecond,
	}

	proc := processor.NewProcessor(mockSender, retryConfig)

	// Create consumer
	consumer, err := kafka.NewConsumer(brokers, groupID, []string{topic})
	require.NoError(t, err)
	defer consumer.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Start consumer in background
	go func() {
		_ = consumer.Start(ctx, proc.Process)
	}()

	// Wait for consumer to be ready
	time.Sleep(2 * time.Second)

	// Produce test message
	testMsg := push.PushMessage{
		EndpointARN: "arn:aws:sns:ru-central1:test:endpoint/GCM/test/123",
		Title:       "Integration Test",
		Body:        "Test message body",
		Data: map[string]interface{}{
			"test_key": "test_value",
		},
	}

	err = produceMessage(brokers, topic, testMsg)
	require.NoError(t, err)

	// Wait for message to be processed
	time.Sleep(3 * time.Second)

	// Verify message was received and processed
	assert.Len(t, mockSender.messages, 1)
	assert.Equal(t, testMsg.EndpointARN, mockSender.messages[0].EndpointARN)
	assert.Equal(t, testMsg.Title, mockSender.messages[0].Title)
	assert.Equal(t, testMsg.Body, mockSender.messages[0].Body)
}

// TestSender is a mock sender for integration tests
type TestSender struct {
	messages []push.PushMessage
}

func (ts *TestSender) Send(ctx context.Context, msg *push.PushMessage) (string, error) {
	ts.messages = append(ts.messages, *msg)
	return "test-message-id", nil
}

// createTopic creates a Kafka topic for testing
func createTopic(brokers []string, topic string) error {
	config := sarama.NewConfig()
	config.Version = sarama.V2_8_0_0

	admin, err := sarama.NewClusterAdmin(brokers, config)
	if err != nil {
		return err
	}
	defer admin.Close()

	topicDetail := &sarama.TopicDetail{
		NumPartitions:     1,
		ReplicationFactor: 1,
	}

	return admin.CreateTopic(topic, topicDetail, false)
}

// produceMessage sends a test message to Kafka
func produceMessage(brokers []string, topic string, msg push.PushMessage) error {
	config := sarama.NewConfig()
	config.Producer.Return.Successes = true

	producer, err := sarama.NewSyncProducer(brokers, config)
	if err != nil {
		return err
	}
	defer producer.Close()

	msgBytes, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	message := &sarama.ProducerMessage{
		Topic: topic,
		Value: sarama.ByteEncoder(msgBytes),
	}

	_, _, err = producer.SendMessage(message)
	return err
}
