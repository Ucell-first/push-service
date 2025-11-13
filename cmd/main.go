package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/Ucell/push-service/config"
	"github.com/Ucell/push-service/internal/kafka"
	"github.com/Ucell/push-service/internal/processor"
	"github.com/Ucell/push-service/internal/push"
)

func main() {
	log.Println("Starting Kafka Push Service...")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	log.Printf("Loaded config: Kafka brokers=%v, topics=%v, group=%s",
		cfg.Kafka.Brokers, cfg.Kafka.Topics, cfg.Kafka.GroupID)

	// Create push sender
	sender := push.NewSender(
		cfg.CNS.Endpoint,
		cfg.CNS.Region,
		cfg.CNS.AccessKeyID,
		cfg.CNS.SecretAccessKey,
	)

	// Create processor
	proc := processor.NewProcessor(sender, cfg.Retry)

	// Create Kafka consumer
	consumer, err := kafka.NewConsumer(
		cfg.Kafka.Brokers,
		cfg.Kafka.GroupID,
		cfg.Kafka.Topics,
	)
	if err != nil {
		log.Fatalf("Failed to create consumer: %v", err)
	}
	defer consumer.Close()

	// Setup context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown gracefully
	sigterm := make(chan os.Signal, 1)
	signal.Notify(sigterm, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigterm
		log.Println("Received shutdown signal, closing...")
		cancel()
	}()

	// Start consuming
	log.Println("Starting to consume messages...")
	if err := consumer.Start(ctx, proc.Process); err != nil {
		log.Fatalf("Failed to start consumer: %v", err)
	}

	log.Println("Service stopped")
}
