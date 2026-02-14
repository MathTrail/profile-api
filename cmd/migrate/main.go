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
func main() {
	cfg := config.Load()
	logger := logging.NewLogger(cfg.LogLevel)
	defer logger.Sync()

	db := database.NewConnection(cfg, logger)
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
