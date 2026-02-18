package database

import (
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/MathTrail/profile-api/internal/config"
	"github.com/MathTrail/profile-api/internal/logging"
)

// NewConnection opens a GORM PostgreSQL connection using the provided config.
// SQL queries are logged through the zap-based GORM logger.
func NewConnection(cfg *config.Config, logger *zap.Logger) *gorm.DB {
	gormLogger := logging.NewGormLogger(logger)

	db, err := gorm.Open(postgres.New(postgres.Config{
		DSN:                  cfg.DSN(),
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

	logger.Info("database connection established",
		zap.String("host", cfg.DBHost),
		zap.String("port", cfg.DBPort),
		zap.String("dbname", cfg.DBName),
	)

	return db
}
