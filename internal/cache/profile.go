package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// ProfileCache provides Redis-based caching for profile data.
// It implements profile.Cache by working with interface{} values
// to avoid circular imports.
type ProfileCache struct {
	client *redis.Client
	ttl    time.Duration
	logger *zap.Logger
}

// NewProfileCache creates a new ProfileCache instance.
func NewProfileCache(client *redis.Client, ttl time.Duration, logger *zap.Logger) *ProfileCache {
	return &ProfileCache{client: client, ttl: ttl, logger: logger}
}

func (c *ProfileCache) cacheKey(userID string) string {
	return fmt.Sprintf("profile:%s", userID)
}

// Get retrieves a cached profile by unmarshalling JSON; returns nil on cache miss.
func (c *ProfileCache) Get(ctx context.Context, userID string) (json.RawMessage, error) {
	data, err := c.client.Get(ctx, c.cacheKey(userID)).Bytes()
	if err == redis.Nil {
		return nil, nil // cache miss
	}
	if err != nil {
		return nil, err
	}
	return data, nil
}

// Set stores a profile as JSON in cache with TTL.
func (c *ProfileCache) Set(ctx context.Context, userID string, data []byte) error {
	return c.client.Set(ctx, c.cacheKey(userID), data, c.ttl).Err()
}

// Invalidate removes a profile from cache.
func (c *ProfileCache) Invalidate(ctx context.Context, userID string) error {
	return c.client.Del(ctx, c.cacheKey(userID)).Err()
}
