.PHONY: build run test test-unit test-integration docker-build docker-up docker-down clean

# Build the application
build:
	go build -o bin/kafka-push-service ./cmd/main.go

# Run the application locally
run:
	go run ./cmd/main.go

# Run all tests
test: test-unit test-integration

# Run unit tests
test-unit:
	go test -v ./tests/unit/...

# Run integration tests
test-integration:
	docker-compose up -d kafka
	sleep 10
	go test -v ./tests/integration/...
	docker-compose down

# Build Docker image
docker-build:
	docker build -t kafka-push-service:latest .

# Start all services with Docker Compose
docker-up:
	docker-compose up --build -d

# Stop all services
docker-down:
	docker-compose down

# View logs
logs:
	docker-compose logs -f kafka-push-service

# Clean build artifacts
clean:
	rm -rf bin/
	go clean

# Install dependencies
deps:
	go mod download
	go mod tidy

# Format code
fmt:
	go fmt ./...

# Lint code (requires golangci-lint)
lint:
	golangci-lint run

# Create Kafka test topic
create-topic:
	docker exec -it $$(docker ps -qf "name=kafka") \
		kafka-topics --create \
		--bootstrap-server localhost:9092 \
		--topic push-notifications \
		--partitions 3 \
		--replication-factor 1

# Send test message to Kafka
test-message:
	@echo '{"endpoint_arn":"arn:aws:sns:ru-central1:xxx:endpoint/GCM/test/123","title":"Test","body":"Hello from Kafka!"}' | \
	docker exec -i $$(docker ps -qf "name=kafka") \
		kafka-console-producer \
		--broker-list localhost:9092 \
		--topic push-notifications
