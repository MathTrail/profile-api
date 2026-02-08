package profile

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Repository defines the interface for profile data access.
type Repository interface {
	Create(ctx context.Context, profile *Profile) error
	GetByUserID(ctx context.Context, userID string) (*Profile, error)
	UpdateSkills(ctx context.Context, profileID uuid.UUID, skills []Skill) error
	UpdateProgress(ctx context.Context, profileID uuid.UUID, progress *Progress) error
	GetOrCreate(ctx context.Context, profile *Profile) (*Profile, bool, error)
}

type repositoryImpl struct {
	db *gorm.DB
}

// NewRepository creates a new profile repository backed by GORM.
func NewRepository(db *gorm.DB) Repository {
	return &repositoryImpl{db: db}
}

func (r *repositoryImpl) Create(ctx context.Context, profile *Profile) error {
	if err := r.db.WithContext(ctx).Create(profile).Error; err != nil {
		return fmt.Errorf("failed to create profile: %w", err)
	}
	return nil
}

func (r *repositoryImpl) GetByUserID(ctx context.Context, userID string) (*Profile, error) {
	var profile Profile
	err := r.db.WithContext(ctx).
		Preload("Skills").
		Preload("Progress").
		Where("user_id = ?", userID).
		First(&profile).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("profile not found for user %s: %w", userID, err)
		}
		return nil, fmt.Errorf("failed to get profile: %w", err)
	}
	return &profile, nil
}

func (r *repositoryImpl) UpdateSkills(ctx context.Context, profileID uuid.UUID, skills []Skill) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for i := range skills {
			skills[i].ProfileID = profileID
		}

		// Upsert: insert or update on (profile_id, skill_name) conflict
		if err := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "profile_id"}, {Name: "skill_name"}},
			DoUpdates: clause.AssignmentColumns([]string{"level", "proficiency", "updated_at"}),
		}).Create(&skills).Error; err != nil {
			return fmt.Errorf("failed to upsert skills: %w", err)
		}

		return nil
	})
}

func (r *repositoryImpl) UpdateProgress(ctx context.Context, profileID uuid.UUID, progress *Progress) error {
	progress.ProfileID = profileID

	result := r.db.WithContext(ctx).
		Where("profile_id = ?", profileID).
		Assign(Progress{
			TasksCompleted: progress.TasksCompleted,
			CurrentStreak:  progress.CurrentStreak,
			TotalPoints:    progress.TotalPoints,
		}).
		FirstOrCreate(progress)

	if result.Error != nil {
		return fmt.Errorf("failed to update progress: %w", result.Error)
	}
	return nil
}

func (r *repositoryImpl) GetOrCreate(ctx context.Context, profile *Profile) (*Profile, bool, error) {
	var existing Profile
	err := r.db.WithContext(ctx).
		Where("user_id = ?", profile.UserID).
		First(&existing).Error

	if err == nil {
		// Profile already exists, load associations
		r.db.WithContext(ctx).Preload("Skills").Preload("Progress").First(&existing)
		return &existing, false, nil
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, false, fmt.Errorf("failed to check existing profile: %w", err)
	}

	// Create new profile
	if err := r.db.WithContext(ctx).Create(profile).Error; err != nil {
		return nil, false, fmt.Errorf("failed to create profile: %w", err)
	}

	return profile, true, nil
}
