package profile

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Sentinel errors for the service layer.
var (
	ErrProfileNotFound = errors.New("profile not found")
	ErrInvalidUserID   = errors.New("invalid user ID")
)

// Cache defines the interface for profile caching.
type Cache interface {
	Get(ctx context.Context, userID string) (*Profile, error)
	Set(ctx context.Context, userID string, profile *Profile) error
	Invalidate(ctx context.Context, userID string) error
}

// Service defines the interface for profile business logic.
type Service interface {
	GetProfile(ctx context.Context, userID string) (*Profile, error)
	CreateProfile(ctx context.Context, profile *Profile) error
	UpdateSkills(ctx context.Context, userID string, profileID uuid.UUID, skills []Skill) error
	UpdateProgress(ctx context.Context, userID string, profileID uuid.UUID, progress *Progress) error
}

type serviceImpl struct {
	repo   Repository
	cache  Cache
	logger *zap.Logger
}

// NewService creates a new profile service with cache-aside pattern.
func NewService(repo Repository, cache Cache) Service {
	return &serviceImpl{repo: repo, cache: cache}
}

func (s *serviceImpl) GetProfile(ctx context.Context, userID string) (*Profile, error) {
	if userID == "" {
		return nil, ErrInvalidUserID
	}

	// 1. Try cache
	cached, err := s.cache.Get(ctx, userID)
	if err != nil && s.logger != nil {
		s.logger.Warn("cache read failed, falling back to DB", zap.Error(err))
	}
	if cached != nil {
		return cached, nil
	}

	// 2. Query DB
	profile, err := s.repo.GetByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("%w: %s", ErrProfileNotFound, userID)
		}
		return nil, fmt.Errorf("failed to get profile: %w", err)
	}

	// 3. Populate cache (best-effort)
	if cacheErr := s.cache.Set(ctx, userID, profile); cacheErr != nil && s.logger != nil {
		s.logger.Warn("cache write failed", zap.Error(cacheErr))
	}

	return profile, nil
}

func (s *serviceImpl) CreateProfile(ctx context.Context, profile *Profile) error {
	if err := s.repo.Create(ctx, profile); err != nil {
		return err
	}
	// Invalidate cache so next read picks up fresh data
	_ = s.cache.Invalidate(ctx, profile.UserID)
	return nil
}

func (s *serviceImpl) UpdateSkills(ctx context.Context, userID string, profileID uuid.UUID, skills []Skill) error {
	if err := s.repo.UpdateSkills(ctx, profileID, skills); err != nil {
		return err
	}
	return s.cache.Invalidate(ctx, userID)
}

func (s *serviceImpl) UpdateProgress(ctx context.Context, userID string, profileID uuid.UUID, progress *Progress) error {
	if err := s.repo.UpdateProgress(ctx, profileID, progress); err != nil {
		return err
	}
	return s.cache.Invalidate(ctx, userID)
}
