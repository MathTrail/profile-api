package profile

import (
	"time"

	"github.com/google/uuid"
)

type Profile struct {
	ID        uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
	UserID    string    `gorm:"uniqueIndex;not null" json:"user_id"`
	Name      string    `gorm:"not null" json:"name"`
	Age       int       `json:"age"`
	Skills    []Skill   `gorm:"foreignKey:ProfileID" json:"skills"`
	Progress  *Progress `gorm:"foreignKey:ProfileID" json:"progress"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Skill struct {
	ID          uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
	ProfileID   uuid.UUID `gorm:"type:uuid;index;not null;uniqueIndex:idx_profile_skill" json:"profile_id"`
	SkillName   string    `gorm:"not null;uniqueIndex:idx_profile_skill" json:"skill_name"`
	Level       int       `gorm:"default:1;not null" json:"level"`
	Proficiency float64   `json:"proficiency"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Progress struct {
	ID             uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
	ProfileID      uuid.UUID `gorm:"type:uuid;uniqueIndex;not null" json:"profile_id"`
	TasksCompleted int       `gorm:"default:0;not null" json:"tasks_completed"`
	CurrentStreak  int       `gorm:"default:0;not null" json:"current_streak"`
	TotalPoints    int       `gorm:"default:0;not null" json:"total_points"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}
