package main

import (
	"fmt"
	"log"
	"os"

	"github.com/mathtrail/mathtrail-profile/internal/config"
	"github.com/mathtrail/mathtrail-profile/internal/database"
)

// migrate applies SQL migration files against the PostgreSQL database.
// Used by the Helm migration Job (mathtrail-service-lib contract).
func main() {
	cfg := config.Load()

	db := database.NewConnection(cfg)
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("failed to get underlying sql.DB: %v", err)
	}
	defer sqlDB.Close()

	migrationsDir := getEnv("MIGRATIONS_DIR", "/migrations")

	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		log.Fatalf("failed to read migrations directory %s: %v", migrationsDir, err)
	}

	applied := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		filePath := fmt.Sprintf("%s/%s", migrationsDir, entry.Name())
		log.Printf("Applying migration: %s", entry.Name())

		content, err := os.ReadFile(filePath)
		if err != nil {
			log.Fatalf("failed to read migration file %s: %v", filePath, err)
		}

		if _, err := sqlDB.Exec(string(content)); err != nil {
			log.Fatalf("failed to apply migration %s: %v", entry.Name(), err)
		}

		applied++
		log.Printf("Successfully applied: %s", entry.Name())
	}

	log.Printf("Migration complete: %d file(s) applied", applied)
}

func getEnv(key, defaultValue string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return defaultValue
}
