package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	// Server
	ServerPort string

	// PostgreSQL connection (non-sensitive; credentials fetched from Dapr secret store)
	DBHost    string
	DBPort    string
	DBName    string
	DBSSLMode string

	// Redis connection (non-sensitive; password fetched from Dapr secret store)
	RedisAddr string
	RedisDB   int

	// Cache
	CacheTTLSeconds int

	// Dapr sidecar
	DaprHost     string
	DaprHTTPPort string // Dapr sidecar HTTP port (for secrets API and service invocation)

	// Dapr Secret Store config
	// DBSecretStore is the Dapr component name for the database credential store, e.g. "vault-db".
	DBSecretStore string
	// DBSecretKey is the secret path for dynamic DB credentials, e.g. "creds/profile-api-role".
	DBSecretKey string
	// KVSecretStore is the Dapr component name for the KV secret store, e.g. "vault".
	KVSecretStore string
	// KVSecretKey is the secret path for static secrets, e.g. "local/mathtrail-profile".
	KVSecretKey string

	// Logging
	LogLevel string

	// Pub/Sub Topics
	TopicUserRegistered string
	TopicTaskSolved     string
}

func Load() *Config {
	return &Config{
		ServerPort: getEnv("SERVER_PORT", "8080"),

		DBHost:    getEnv("PG_HOST", "postgres-pgbouncer"),
		DBPort:    getEnv("PG_PORT", "6432"),
		DBName:    getEnv("PG_DATABASE", "profile"),
		DBSSLMode: getEnv("PG_SSL_MODE", "disable"),

		RedisAddr: getEnv("REDIS_ADDR", "redis-master:6379"),
		RedisDB:   getEnvInt("REDIS_DB", 0),

		CacheTTLSeconds: getEnvInt("CACHE_TTL_SECONDS", 300),

		DaprHost:     getEnv("DAPR_HOST", "localhost"),
		DaprHTTPPort: getEnv("DAPR_HTTP_PORT", "3500"),

		DBSecretStore: getEnv("DB_SECRET_STORE", "vault-db"),
		DBSecretKey:   getEnv("DB_SECRET_KEY", "creds/profile-api-role"),
		KVSecretStore: getEnv("KV_SECRET_STORE", "vault"),
		KVSecretKey:   getEnv("KV_SECRET_KEY", "local/mathtrail-profile"),

		LogLevel: getEnv("LOG_LEVEL", "info"),

		TopicUserRegistered: getEnv("TOPIC_USER_REGISTERED", "user-registered"),
		TopicTaskSolved:     getEnv("TOPIC_TASK_SOLVED", "task-solved"),
	}
}

// DaprAddr returns the Dapr sidecar HTTP API address (host:port).
func (c *Config) DaprAddr() string {
	return fmt.Sprintf("%s:%s", c.DaprHost, c.DaprHTTPPort)
}

// PgDSNTemplate returns a libpq connection string without user/password.
// Credentials are appended at runtime after being fetched from Dapr secret store.
func (c *Config) PgDSNTemplate() string {
	return fmt.Sprintf(
		"host=%s port=%s dbname=%s sslmode=%s",
		c.DBHost, c.DBPort, c.DBName, c.DBSSLMode,
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
