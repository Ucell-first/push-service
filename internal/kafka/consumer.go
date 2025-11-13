package kafka

import (
	"context"
	"log"
	"sync"

	"github.com/IBM/sarama"
)

type Consumer struct {
	client        sarama.ConsumerGroup
	topics        []string
	messageBuffer chan *sarama.ConsumerMessage
	handler       MessageHandler
}

type MessageHandler func(ctx context.Context, msg *sarama.ConsumerMessage) error

// NewConsumer creates a new Kafka consumer
func NewConsumer(brokers []string, groupID string, topics []string) (*Consumer, error) {
	config := sarama.NewConfig()
	config.Version = sarama.V2_8_0_0
	config.Consumer.Group.Rebalance.GroupStrategies = []sarama.BalanceStrategy{
		sarama.NewBalanceStrategyRoundRobin(),
	}
	config.Consumer.Offsets.Initial = sarama.OffsetNewest
	config.Consumer.Return.Errors = true

	client, err := sarama.NewConsumerGroup(brokers, groupID, config)
	if err != nil {
		return nil, err
	}

	return &Consumer{
		client:        client,
		topics:        topics,
		messageBuffer: make(chan *sarama.ConsumerMessage, 100),
	}, nil
}

// Start begins consuming messages from Kafka
func (c *Consumer) Start(ctx context.Context, handler MessageHandler) error {
	c.handler = handler

	wg := &sync.WaitGroup{}
	wg.Add(1)

	go func() {
		defer wg.Done()
		for {
			// Consume should be called inside an infinite loop
			if err := c.client.Consume(ctx, c.topics, c); err != nil {
				log.Printf("Error from consumer: %v", err)
			}

			// Check if context was cancelled
			if ctx.Err() != nil {
				return
			}
		}
	}()

	// Handle errors
	go func() {
		for err := range c.client.Errors() {
			log.Printf("Kafka error: %v", err)
		}
	}()

	wg.Wait()
	return nil
}

// Close closes the consumer
func (c *Consumer) Close() error {
	return c.client.Close()
}

// Setup is run at the beginning of a new session, before ConsumeClaim
func (c *Consumer) Setup(sarama.ConsumerGroupSession) error {
	return nil
}

// Cleanup is run at the end of a session, once all ConsumeClaim goroutines have exited
func (c *Consumer) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

// ConsumeClaim must start a consumer loop of ConsumerGroupClaim's Messages()
func (c *Consumer) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for message := range claim.Messages() {
		// Process message with handler
		if err := c.handler(session.Context(), message); err != nil {
			log.Printf("Error processing message: %v", err)
			// Don't mark as consumed if handler returns error
			continue
		}

		// Mark message as consumed only after successful processing
		session.MarkMessage(message, "")
	}

	return nil
}
