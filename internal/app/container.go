package app

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/MathTrail/profile-api/internal/cache"
	"github.com/MathTrail/profile-api/internal/config"
	"github.com/MathTrail/profile-api/internal/database"
	kafkaconsumer "github.com/MathTrail/profile-api/internal/kafka"
	"github.com/MathTrail/profile-api/internal/logging"
	"github.com/MathTrail/profile-api/internal/profile"
)

// Container holds all application dependencies wired together.
type Container struct {
	DB                *gorm.DB
	RedisClient       *redis.Client
	Logger            *zap.Logger
	ProfileRepository profile.Repository
	ProfileService    profile.Service
	ProfileController *profile.Controller
	Consumers         []*kafkaconsumer.Consumer
}

// NewContainer initializes all dependencies and returns the DI container.
func NewContainer(cfg *config.Config) *Container {
	logger := logging.NewLogger(cfg.LogLevel)

	// Database; credentials injected by VSO via mathtrail-profile-db-secret K8s Secret.
	db := database.NewConnection(cfg.PgDSN(), logger)

	// Redis; password injected by ESO via mathtrail-profile-secrets K8s Secret.
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})
	redisCache := cache.NewProfileCache(rdb, time.Duration(cfg.CacheTTLSeconds)*time.Second, logger.Named("cache"))
	profileCache := &profileCacheAdapter{cache: redisCache}

	// Profile domain
	profileRepo := profile.NewRepository(db)
	profileService := profile.NewService(profileRepo, profileCache, logger.Named("profile"))
	profileController := profile.NewController(profileService)

	// Kafka consumers for event-driven updates
	brokers := strings.Split(cfg.KafkaBrokers, ",")
	consumers := []*kafkaconsumer.Consumer{
		kafkaconsumer.NewConsumer(
			brokers, cfg.TopicUserRegistered, cfg.KafkaConsumerGroup,
			kafkaconsumer.HandleUserRegistered(profileService, logger.Named("kafka.user-registered")),
			logger.Named("kafka.user-registered"),
		),
		kafkaconsumer.NewConsumer(
			brokers, cfg.TopicTaskSolved, cfg.KafkaConsumerGroup,
			kafkaconsumer.HandleTaskSolved(profileService, logger.Named("kafka.task-solved")),
			logger.Named("kafka.task-solved"),
		),
	}

	return &Container{
		DB:                db,
		RedisClient:       rdb,
		Logger:            logger,
		ProfileRepository: profileRepo,
		ProfileService:    profileService,
		ProfileController: profileController,
		Consumers:         consumers,
	}
}

// Close releases resources held by the container.
func (c *Container) Close() {
	for _, consumer := range c.Consumers {
		consumer.Close()
	}
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
