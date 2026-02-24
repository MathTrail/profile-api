package app

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/MathTrail/profile-api/internal/cache"
	"github.com/MathTrail/profile-api/internal/config"
	"github.com/MathTrail/profile-api/internal/dapr"
	"github.com/MathTrail/profile-api/internal/database"
	"github.com/MathTrail/profile-api/internal/logging"
	"github.com/MathTrail/profile-api/internal/profile"
	"github.com/MathTrail/profile-api/internal/secrets"
)

// Container holds all application dependencies wired together.
type Container struct {
	DB                *gorm.DB
	RedisClient       *redis.Client
	Logger            *zap.Logger
	ProfileRepository profile.Repository
	ProfileService    profile.Service
	ProfileController *profile.Controller
	EventHandler      *dapr.EventHandler
}

// NewContainer initializes all dependencies and returns the DI container.
func NewContainer(cfg *config.Config) *Container {
	// Logger
	logger := logging.NewLogger(cfg.LogLevel)

	ctx := context.Background()
	daprAddr := cfg.DaprAddr()

	// Fetch dynamic DB credentials from Dapr secret store.
	// Retries up to 10 times with linear backoff to handle sidecar startup race.
	dbCreds, err := secrets.GetDaprSecretWithRetry(ctx, daprAddr, cfg.DBSecretStore, cfg.DBSecretKey, 10)
	if err != nil {
		logger.Fatal("failed to fetch DB credentials from dapr secret store", zap.Error(err))
	}
	dbDSN := fmt.Sprintf("%s user=%s password=%s", cfg.PgDSNTemplate(), dbCreds["username"], dbCreds["password"])

	// Fetch static secrets (Redis password) from Dapr KV secret store.
	kvSecrets, err := secrets.GetDaprSecretWithRetry(ctx, daprAddr, cfg.KVSecretStore, cfg.KVSecretKey, 10)
	if err != nil {
		logger.Fatal("failed to fetch KV secrets from dapr secret store", zap.Error(err))
	}

	// Database
	db := database.NewConnection(dbDSN, logger)

	// Redis
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: kvSecrets["redis-password"],
		DB:       cfg.RedisDB,
	})
	redisCache := cache.NewProfileCache(rdb, time.Duration(cfg.CacheTTLSeconds)*time.Second, logger.Named("cache"))
	profileCache := &profileCacheAdapter{cache: redisCache}

	// Profile domain
	profileRepo := profile.NewRepository(db)
	profileService := profile.NewService(profileRepo, profileCache, logger.Named("profile"))
	profileController := profile.NewController(profileService)

	// Dapr event handler
	eventHandler := dapr.NewEventHandler(
		profileService,
		logger,
		"pubsub",
		cfg.TopicUserRegistered,
		cfg.TopicTaskSolved,
	)

	return &Container{
		DB:                db,
		RedisClient:       rdb,
		Logger:            logger,
		ProfileRepository: profileRepo,
		ProfileService:    profileService,
		ProfileController: profileController,
		EventHandler:      eventHandler,
	}
}

// Close releases resources held by the container.
func (c *Container) Close() {
	sqlDB, _ := c.DB.DB()
	sqlDB.Close()
	c.RedisClient.Close()
	_ = c.Logger.Sync()
}

// Ready checks that all downstream dependencies (DB, Redis) are reachable.
// Used by the /health/ready probe.
func (c *Container) Ready(ctx context.Context) error {
	sqlDB, err := c.DB.DB()
	if err != nil {
		return fmt.Errorf("database not available: %w", err)
	}
	if err := sqlDB.PingContext(ctx); err != nil {
		return fmt.Errorf("database ping failed: %w", err)
	}
	if err := c.RedisClient.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("redis ping failed: %w", err)
	}
	return nil
}

// profileCacheAdapter bridges cache.ProfileCache (no profile import) with profile.Cache interface.
type profileCacheAdapter struct {
	cache *cache.ProfileCache
}

func (a *profileCacheAdapter) Get(ctx context.Context, userID string) (*profile.Profile, error) {
	data, err := a.cache.Get(ctx, userID)
	if err != nil {
		return nil, err
	}
	if data == nil {
		return nil, nil
	}
	var p profile.Profile
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

func (a *profileCacheAdapter) Set(ctx context.Context, userID string, p *profile.Profile) error {
	data, err := json.Marshal(p)
	if err != nil {
		return err
	}
	return a.cache.Set(ctx, userID, data)
}

func (a *profileCacheAdapter) Invalidate(ctx context.Context, userID string) error {
	return a.cache.Invalidate(ctx, userID)
}
