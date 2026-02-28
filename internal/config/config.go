package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	// Server
	ServerPort string

	// PostgreSQL connection; credentials injected by VSO via K8s Secret (mathtrail-profile-db-secret)
	DBHost     string
	DBPort     string
	DBName     string
	DBSSLMode  string
	DBUser     string
	DBPassword string

	// Redis connection; password injected by ESO via K8s Secret (mathtrail-profile-secrets)
	RedisAddr     string
	RedisDB       int
	RedisPassword string

	// Cache
	CacheTTLSeconds int

	// Logging
	LogLevel string

	// Kafka
	KafkaBrokers       string // comma-separated list, e.g. "kafka-kafka-bootstrap:9092"
	KafkaConsumerGroup string

	// Pub/Sub Topics
	TopicUserRegistered string
	TopicTaskSolved     string
}

func Load() *Config {
	return &Config{
		ServerPort: getEnv("SERVER_PORT", "8080"),

		DBHost:     getEnv("PG_HOST", "postgres-pgbouncer"),
		DBPort:     getEnv("PG_PORT", "6432"),
		DBName:     getEnv("PG_DATABASE", "profile"),
		DBSSLMode:  getEnv("PG_SSL_MODE", "disable"),
		DBUser:     getEnv("PG_USER", ""),
		DBPassword: getEnv("PG_PASSWORD", ""),

		RedisAddr:     getEnv("REDIS_ADDR", "redis-master:6379"),
		RedisDB:       getEnvInt("REDIS_DB", 0),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),

		CacheTTLSeconds: getEnvInt("CACHE_TTL_SECONDS", 300),

		LogLevel: getEnv("LOG_LEVEL", "info"),

		KafkaBrokers:       getEnv("KAFKA_BROKERS", "kafka-kafka-bootstrap:9092"),
		KafkaConsumerGroup: getEnv("KAFKA_CONSUMER_GROUP", "profile-service"),

		TopicUserRegistered: getEnv("TOPIC_USER_REGISTERED", "user-registered"),
		TopicTaskSolved:     getEnv("TOPIC_TASK_SOLVED", "task-solved"),
	}
}

// PgDSN returns a full libpq connection string using credentials from env.
func (c *Config) PgDSN() string {
	return fmt.Sprintf(
		"host=%s port=%s dbname=%s sslmode=%s user=%s password=%s",
		c.DBHost, c.DBPort, c.DBName, c.DBSSLMode, c.DBUser, c.DBPassword,
	)
}

func getEnv(key, defaultValue string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value, ok := os.LookupEnv(key); ok {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}
