package main

import (
	"fmt"
	"os"

	"go.uber.org/zap"

	"github.com/MathTrail/profile-api/internal/config"
	"github.com/MathTrail/profile-api/internal/database"
	"github.com/MathTrail/profile-api/internal/logging"
)

// migrate applies SQL migration files against the PostgreSQL database.
// Used by the Helm migration Job (mathtrail-service-lib contract).
// Credentials come from env vars DB_HOST, DB_USER, DB_PASSWORD, DB_PORT, DB_NAME, DB_SSL_MODE
// injected by the Helm migration Job (Bitnami postgres superuser secret).
func main() {
	cfg := config.Load()
	logger := logging.NewLogger(cfg.LogLevel)
	defer logger.Sync()

	dsn := fmt.Sprintf(
		"host=%s port=%s dbname=%s user=%s password=%s sslmode=%s",
		getEnv("DB_HOST", "postgres-postgresql"),
		getEnv("DB_PORT", "5432"),
		getEnv("DB_NAME", "profile"),
		getEnv("DB_USER", "postgres"),
		getEnv("DB_PASSWORD", ""),
		getEnv("DB_SSL_MODE", "disable"),
	)

	db := database.NewConnection(dsn, logger)
	sqlDB, err := db.DB()
	if err != nil {
		logger.Fatal("failed to get underlying sql.DB", zap.Error(err))
	}
	defer sqlDB.Close()

	migrationsDir := getEnv("MIGRATIONS_DIR", "/migrations")

	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		logger.Fatal("failed to read migrations directory", zap.String("dir", migrationsDir), zap.Error(err))
	}

	applied := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		filePath := fmt.Sprintf("%s/%s", migrationsDir, entry.Name())
		logger.Info("applying migration", zap.String("file", entry.Name()))

		content, err := os.ReadFile(filePath)
		if err != nil {
			logger.Fatal("failed to read migration file", zap.String("file", filePath), zap.Error(err))
		}

		if _, err := sqlDB.Exec(string(content)); err != nil {
			logger.Fatal("failed to apply migration", zap.String("file", entry.Name()), zap.Error(err))
		}

		applied++
		logger.Info("successfully applied", zap.String("file", entry.Name()))
	}

	logger.Info("migration complete", zap.Int("applied", applied))
}

func getEnv(key, defaultValue string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return defaultValue
}
