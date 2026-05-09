package config

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	KafkaBrokers []string
	KafkaTopic   string
	KafkaGroupID string

	RedisAddr     string
	RedisPassword string
	RedisDB       int

	PostgresDSN string

	S3Endpoint     string
	S3AccessKey    string
	S3SecretKey    string
	S3Bucket       string
	S3Region       string
	S3UsePathStyle bool

	CamundaBaseURL     string
	CamundaMessageName string

	ChannelBuffer int
}

func Load() *Config {
	return &Config{
		KafkaBrokers:       splitEnv("KAFKA_BROKERS", "localhost:9092"),
		KafkaTopic:         getEnv("KAFKA_TOPIC", "pdf.generate.requests"),
		KafkaGroupID:       getEnv("KAFKA_GROUP_ID", "document-convert-service"),
		RedisAddr:          getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword:      getEnv("REDIS_PASSWORD", ""),
		RedisDB:            getEnvInt("REDIS_DB", 0),
		PostgresDSN:        getEnv("POSTGRES_DSN", "postgres://postgres:postgres@localhost:5432/documents"),
		S3Endpoint:         getEnv("S3_ENDPOINT", "http://localhost:9000"),
		S3AccessKey:        getEnv("S3_ACCESS_KEY", "minioadmin"),
		S3SecretKey:        getEnv("S3_SECRET_KEY", "minioadmin"),
		S3Bucket:           getEnv("S3_BUCKET", "documents"),
		S3Region:           getEnv("S3_REGION", "us-east-1"),
		S3UsePathStyle:     getEnvBool("S3_USE_PATH_STYLE", true),
		CamundaBaseURL:     getEnv("CAMUNDA_BASE_URL", "http://localhost:8080"),
		CamundaMessageName: getEnv("CAMUNDA_MESSAGE_NAME", "pdf_generated"),
		ChannelBuffer:      getEnvInt("CHANNEL_BUFFER", 100),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func splitEnv(key, fallback string) []string {
	return strings.Split(getEnv(key, fallback), ",")
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	if v := os.Getenv(key); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return fallback
}
