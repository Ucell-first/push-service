package config

import (
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Kafka KafkaConfig
	CNS   CNSConfig
	Retry RetryConfig
}

type KafkaConfig struct {
	Brokers []string
	Topics  []string
	GroupID string
}

type CNSConfig struct {
	Endpoint        string
	Region          string
	AccessKeyID     string
	SecretAccessKey string
	PlatformARN     string
}

type RetryConfig struct {
	MaxAttempts  int
	InitialDelay time.Duration
	MaxDelay     time.Duration
}

// Load reads configuration from environment variables
func Load() (*Config, error) {
	// Load .env file if exists (ignore error in production)
	_ = godotenv.Load()

	cfg := &Config{
		Kafka: KafkaConfig{
			Brokers: strings.Split(getEnv("KAFKA_BROKERS", "localhost:9092"), ","),
			Topics:  strings.Split(getEnv("KAFKA_TOPICS", "push-notifications"), ","),
			GroupID: getEnv("KAFKA_GROUP_ID", "push-service-group"),
		},
		CNS: CNSConfig{
			Endpoint:        getEnv("CNS_ENDPOINT", "https://notifications.yandexcloud.net/"),
			Region:          getEnv("CNS_REGION", "ru-central1"),
			AccessKeyID:     getEnv("CNS_ACCESS_KEY_ID", ""),
			SecretAccessKey: getEnv("CNS_SECRET_ACCESS_KEY", ""),
			PlatformARN:     getEnv("CNS_PLATFORM_ARN", ""),
		},
		Retry: RetryConfig{
			MaxAttempts:  getEnvAsInt("MAX_RETRY_ATTEMPTS", 3),
			InitialDelay: getEnvAsDuration("RETRY_INITIAL_DELAY", time.Second),
			MaxDelay:     getEnvAsDuration("RETRY_MAX_DELAY", 30*time.Second),
		},
	}

	// Validate required fields
	if cfg.CNS.AccessKeyID == "" || cfg.CNS.SecretAccessKey == "" {
		log.Fatal("CNS_ACCESS_KEY_ID and CNS_SECRET_ACCESS_KEY are required")
	}

	return cfg, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	valueStr := getEnv(key, "")
	if value, err := strconv.Atoi(valueStr); err == nil {
		return value
	}
	return defaultValue
}

func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
	valueStr := getEnv(key, "")
	if value, err := time.ParseDuration(valueStr); err == nil {
		return value
	}
	return defaultValue
}
