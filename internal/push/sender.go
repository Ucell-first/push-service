package push

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	"github.com/aws/aws-sdk-go-v2/service/sns/types"
)

type Sender struct {
	client *sns.Client
}

// PushMessage represents the structure of push notification
type PushMessage struct {
	EndpointARN string                 `json:"endpoint_arn"`
	Title       string                 `json:"title"`
	Body        string                 `json:"body"`
	Data        map[string]interface{} `json:"data,omitempty"`
}

// NewSender creates a new push notification sender
func NewSender(endpoint, region, accessKeyID, secretAccessKey string) *Sender {
	credProvider := aws.CredentialsProviderFunc(func(ctx context.Context) (aws.Credentials, error) {
		return aws.Credentials{
			AccessKeyID:     accessKeyID,
			SecretAccessKey: secretAccessKey,
		}, nil
	})

	config := aws.Config{
		BaseEndpoint: &endpoint,
		Region:       region,
		Credentials:  credProvider,
	}

	client := sns.NewFromConfig(config)

	return &Sender{client: client}
}

// Send sends push notification to the specified endpoint
func (s *Sender) Send(ctx context.Context, msg *PushMessage) (string, error) {
	// Build FCM (GCM) message format
	gcmPayload := map[string]interface{}{
		"notification": map[string]string{
			"title": msg.Title,
			"body":  msg.Body,
		},
	}

	// Add custom data if provided
	if len(msg.Data) > 0 {
		gcmPayload["data"] = msg.Data
	}

	gcmJSON, err := json.Marshal(gcmPayload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal GCM payload: %w", err)
	}

	// Build SNS message with platform-specific payloads
	messagePayload := map[string]string{
		"default": msg.Body,
		"GCM":     string(gcmJSON),
	}

	messageJSON, err := json.Marshal(messagePayload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal message: %w", err)
	}

	messageStr := string(messageJSON)
	messageStructure := "json"

	input := &sns.PublishInput{
		TargetArn:        &msg.EndpointARN,
		Message:          &messageStr,
		MessageStructure: &messageStructure,
	}

	output, err := s.client.Publish(ctx, input)
	if err != nil {
		return "", fmt.Errorf("failed to publish message: %w", err)
	}

	log.Printf("Push sent successfully. MessageId: %s", *output.MessageId)
	return *output.MessageId, nil
}

// CreateEndpoint creates a new endpoint for push notifications
func (s *Sender) CreateEndpoint(ctx context.Context, platformARN, deviceToken string) (string, error) {
	input := &sns.CreatePlatformEndpointInput{
		PlatformApplicationArn: &platformARN,
		Token:                  &deviceToken,
	}

	output, err := s.client.CreatePlatformEndpoint(ctx, input)
	if err != nil {
		return "", fmt.Errorf("failed to create endpoint: %w", err)
	}

	return *output.EndpointArn, nil
}

// IsRetryableError checks if the error is retryable (5xx errors)
func IsRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// Check for AWS service errors
	//var apiErr *types.InternalErrorException
	if ok := err.(*types.InternalErrorException); ok != nil {
		return true
	}

	// Check for other 5xx errors by error message
	errMsg := err.Error()
	return contains(errMsg, "InternalError") ||
		contains(errMsg, "ServiceUnavailable") ||
		contains(errMsg, "Throttling")
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr))
}
