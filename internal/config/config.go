package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	// Server
	ServerPort string

	// Database
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	DBSSLMode  string

	// Redis
	RedisAddr     string
	RedisPassword string
	RedisDB       int

	// Cache
	CacheTTLSeconds int

	// Dapr
	DaprHost string
	DaprPort string

	// Logging
	LogLevel string

	// Pub/Sub Topics
	TopicUserRegistered string
	TopicTaskSolved     string
}

func Load() *Config {
	return &Config{
		ServerPort: getEnv("SERVER_PORT", "8080"),

		DBHost:     getEnv("DB_HOST", "postgres-postgresql"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     getEnv("DB_USER", "postgres"),
		DBPassword: getEnv("DB_PASSWORD", "postgres"),
		DBName:     getEnv("DB_NAME", "profile"),
		DBSSLMode:  getEnv("DB_SSL_MODE", "disable"),

		RedisAddr:     getEnv("REDIS_ADDR", "redis-master:6379"),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
		RedisDB:       getEnvInt("REDIS_DB", 0),

		CacheTTLSeconds: getEnvInt("CACHE_TTL_SECONDS", 300),

		DaprHost: getEnv("DAPR_HOST", "localhost"),
		DaprPort: getEnv("DAPR_PORT", "3500"),

		LogLevel: getEnv("LOG_LEVEL", "info"),

		TopicUserRegistered: getEnv("TOPIC_USER_REGISTERED", "user-registered"),
		TopicTaskSolved:     getEnv("TOPIC_TASK_SOLVED", "task-solved"),
	}
}

func (c *Config) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.DBHost, c.DBPort, c.DBUser, c.DBPassword, c.DBName, c.DBSSLMode,
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
