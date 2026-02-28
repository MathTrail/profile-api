package database

import (
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/MathTrail/profile-api/internal/logging"
)

// NewConnection opens a GORM PostgreSQL connection using an explicit DSN.
// The DSN must include all connection parameters including user and password.
// Credentials should be fetched from Vault before calling this function.
// SQL queries are logged through the zap-based GORM logger.
func NewConnection(dsn string, logger *zap.Logger) *gorm.DB {
	gormLogger := logging.NewGormLogger(logger)

	db, err := gorm.Open(postgres.New(postgres.Config{
		DSN:                  dsn,
		PreferSimpleProtocol: true, // disables implicit prepared statements; required for PgBouncer transaction mode
	}), &gorm.Config{
		Logger: gormLogger,
	})
	if err != nil {
		logger.Fatal("failed to connect to database", zap.Error(err))
	}

	sqlDB, err := db.DB()
	if err != nil {
		logger.Fatal("failed to get underlying sql.DB", zap.Error(err))
	}

	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(5)

	logger.Info("database connection established")

	return db
}
