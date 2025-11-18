# push-service

Go service for consuming messages from Kafka and sending push notifications via Yandex Cloud Notification Service (CNS).

## Features

- ✅ Consumes messages from Kafka topics
- ✅ Sends push notifications via Yandex CNS (FCM/APNs/HMS/RuStore)
- ✅ Automatic retry with exponential backoff for 5xx errors
- ✅ Offset commit only after successful delivery
- ✅ Graceful shutdown
- ✅ Unit and integration tests
- ✅ Docker support

## Prerequisites

### 1. Yandex Cloud Setup

You need:
- **Service Account** with `editor` role
- **Static Access Key** (Access Key ID + Secret Key)
- **FCM/APNs Channel** created in CNS
- **Platform ARN** from your notification channel

### How to get Static Access Key:

1. Go to [Yandex Cloud Console](https://console.yandex.cloud/)
2. Select your folder
3. Navigate to **Service Accounts**
4. Create or select a service account
5. Go to **Access Keys** → **Create Static Access Key**
6. Save **Access Key ID** and **Secret Key**

### How to create FCM Channel:

1. Go to **Cloud Notification Service** in console
2. Click **Create notification channel**
3. Select **Google Android (FCM)**
4. Choose **HTTP v1 API**
5. Upload your Firebase **service account JSON key**
6. Save the channel
7. Copy the **Platform ARN** (format: `arn:aws:sns:ru-central1:xxx:app/GCM/your-channel`)

## Installation

### Local Development

1. Clone the repository
2. Copy `.env.example` to `.env` and fill in your credentials:
```bash
cp .env.example .env
```

3. Edit `.env`:
```env
KAFKA_BROKERS=localhost:9092
KAFKA_TOPICS=push-notifications
KAFKA_GROUP_ID=push-service-group

CNS_ENDPOINT=https://notifications.yandexcloud.net/
CNS_REGION=ru-central1
CNS_ACCESS_KEY_ID=your_actual_key_id
CNS_SECRET_ACCESS_KEY=your_actual_secret_key
CNS_PLATFORM_ARN=arn:aws:sns:ru-central1:xxx:app/GCM/your-channel

MAX_RETRY_ATTEMPTS=3
RETRY_INITIAL_DELAY=1s
RETRY_MAX_DELAY=30s
```

4. Install dependencies:
```bash
go mod download
```

5. Run the service:
```bash
go run cmd/main.go
```

### Docker Compose (with Kafka)
```bash
docker-compose up --build
```

This will start:
- Zookeeper
- Kafka
- Push service

## Message Format

Send messages to Kafka in this JSON format:
```json
{
  "endpoint_arn": "arn:aws:sns:ru-central1:xxx:endpoint/GCM/your-channel/uuid",
  "title": "Notification Title",
  "body": "Notification body text",
  "data": {
    "key1": "value1",
    "key2": "value2"
  }
}
```

**Required fields:**
- `endpoint_arn`: Device endpoint ARN from CNS
- `body`: Notification text

**Optional fields:**
- `title`: Notification title
- `data`: Custom data payload

## Testing

### Unit Tests
```bash
go test ./tests/unit/...
```

### Integration Tests
```bash
# Start Kafka first
docker-compose up -d kafka

# Run integration tests
go test ./tests/integration/...
```

## Creating Device Endpoints

Before sending push notifications, you need to create endpoints for each device:
```go
sender := push.NewSender(endpoint, region, keyID, secretKey)
endpointARN, err := sender.CreateEndpoint(ctx, platformARN, deviceToken)
```

Where:
- `platformARN`: Your CNS channel ARN
- `deviceToken`: FCM registration token from the mobile device

## Sending Test Message to Kafka
```bash
# Using kafka-console-producer
docker exec -it <kafka-container> kafka-console-producer --broker-list localhost:9092 --topic push-notifications

# Then paste this JSON:
{"endpoint_arn":"arn:aws:sns:ru-central1:xxx:endpoint/GCM/xxx/xxx","title":"Test","body":"Hello from Kafka"}
```

## Environment Variables

| Variable | Description | Required | Default |
|----------|-------------|----------|---------|
| KAFKA_BROKERS | Comma-separated Kafka broker addresses | Yes | localhost:9092 |
| KAFKA_TOPICS | Comma-separated topic names | Yes | push-notifications |
| KAFKA_GROUP_ID | Consumer group ID | Yes | push-service-group |
| CNS_ENDPOINT | Yandex CNS API endpoint | Yes | https://notifications.yandexcloud.net/ |
| CNS_REGION | Yandex Cloud region | Yes | ru-central1 |
| CNS_ACCESS_KEY_ID | Static access key ID | Yes | - |
| CNS_SECRET_ACCESS_KEY | Static secret key | Yes | - |
| CNS_PLATFORM_ARN | Platform application ARN | Yes | - |
| MAX_RETRY_ATTEMPTS | Max retry attempts for 5xx errors | No | 3 |
| RETRY_INITIAL_DELAY | Initial retry delay | No | 1s |
| RETRY_MAX_DELAY | Maximum retry delay | No | 30s |

## Troubleshooting

### "Failed to create endpoint: InvalidParameter"
- Check that your Firebase JSON key is correct
- Verify the device token is valid

### "Failed to publish message: EndpointDisabled"
- The device token is no longer valid
- Delete and recreate the endpoint

### "Context deadline exceeded"
- Check network connectivity to Yandex Cloud
- Verify CNS_ENDPOINT is correct

## License

MIT