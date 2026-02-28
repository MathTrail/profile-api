package kafka

import (
	"context"
	"encoding/json"
	"time"

	cloudevents "github.com/cloudevents/sdk-go/v2/event"
	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"

	"github.com/MathTrail/profile-api/internal/profile"
)

// UserRegisteredEvent is the payload from the user-registered topic.
// Sent by Ory Kratos webhook when a new user registers.
type UserRegisteredEvent struct {
	UserID    string    `json:"user_id"`
	Name      string    `json:"name"`
	Age       int       `json:"age"`
	Email     string    `json:"email"`
	Timestamp time.Time `json:"timestamp"`
}

// TaskSolvedEvent is the payload from the task-solved topic.
// Sent when a user completes a math trail task.
type TaskSolvedEvent struct {
	UserID       string         `json:"user_id"`
	TaskID       string         `json:"task_id"`
	SkillsGained map[string]int `json:"skills_gained"`
	PointsEarned int            `json:"points_earned"`
	Timestamp    time.Time      `json:"timestamp"`
}

// HandleUserRegistered returns a MessageHandler that creates a profile for each
// user-registered event. Messages may be raw JSON or wrapped in a CloudEvents envelope.
func HandleUserRegistered(svc profile.Service, logger *zap.Logger) MessageHandler {
	return func(ctx context.Context, msg kafka.Message) error {
		var event UserRegisteredEvent
		if err := unmarshalEvent(msg.Value, &event); err != nil {
			logger.Error("failed to parse user-registered event",
				zap.String("topic", msg.Topic),
				zap.Error(err),
			)
			return nil // non-retryable parse error — commit and move on
		}

		if event.UserID == "" {
			logger.Warn("user-registered event missing user_id, dropping",
				zap.String("topic", msg.Topic),
				zap.Int64("offset", msg.Offset),
			)
			return nil
		}

		logger.Info("processing user-registered event",
			zap.String("user_id", event.UserID),
			zap.String("name", event.Name),
		)

		newProfile := &profile.Profile{
			UserID: event.UserID,
			Name:   event.Name,
			Age:    event.Age,
		}

		if err := svc.CreateProfile(ctx, newProfile); err != nil {
			logger.Error("failed to create profile from user-registered event",
				zap.String("user_id", event.UserID),
				zap.Error(err),
			)
			return err // retryable — don't commit
		}

		logger.Info("profile created from user-registered event",
			zap.String("user_id", event.UserID),
			zap.String("profile_id", newProfile.ID.String()),
		)
		return nil
	}
}

// HandleTaskSolved returns a MessageHandler that updates skills and progress
// for the user who solved the task.
func HandleTaskSolved(svc profile.Service, logger *zap.Logger) MessageHandler {
	return func(ctx context.Context, msg kafka.Message) error {
		var event TaskSolvedEvent
		if err := unmarshalEvent(msg.Value, &event); err != nil {
			logger.Error("failed to parse task-solved event",
				zap.String("topic", msg.Topic),
				zap.Error(err),
			)
			return nil // non-retryable parse error — commit and move on
		}

		if event.UserID == "" || event.TaskID == "" {
			logger.Warn("task-solved event missing required fields, dropping",
				zap.String("topic", msg.Topic),
				zap.Int64("offset", msg.Offset),
				zap.String("user_id", event.UserID),
				zap.String("task_id", event.TaskID),
			)
			return nil
		}

		logger.Info("processing task-solved event",
			zap.String("user_id", event.UserID),
			zap.String("task_id", event.TaskID),
			zap.Int("points_earned", event.PointsEarned),
		)

		existingProfile, err := svc.GetProfile(ctx, event.UserID)
		if err != nil {
			logger.Error("failed to get profile for task-solved event",
				zap.String("user_id", event.UserID),
				zap.Error(err),
			)
			return err // retryable
		}

		if len(event.SkillsGained) > 0 {
			var skills []profile.Skill
			for skillName, level := range event.SkillsGained {
				skills = append(skills, profile.Skill{
					SkillName:   skillName,
					Level:       level,
					Proficiency: computeProficiency(level),
				})
			}
			if err := svc.UpdateSkills(ctx, event.UserID, existingProfile.ID, skills); err != nil {
				logger.Error("failed to update skills from task-solved event",
					zap.String("user_id", event.UserID),
					zap.Error(err),
				)
				return err // retryable
			}
		}

		progress := &profile.Progress{
			TasksCompleted: getTasksCompleted(existingProfile) + 1,
			CurrentStreak:  getCurrentStreak(existingProfile) + 1,
			TotalPoints:    getTotalPoints(existingProfile) + event.PointsEarned,
		}

		if err := svc.UpdateProgress(ctx, event.UserID, existingProfile.ID, progress); err != nil {
			logger.Error("failed to update progress from task-solved event",
				zap.String("user_id", event.UserID),
				zap.Error(err),
			)
			return err // retryable
		}

		logger.Info("profile updated from task-solved event",
			zap.String("user_id", event.UserID),
			zap.String("task_id", event.TaskID),
			zap.Int("new_total_points", progress.TotalPoints),
			zap.Int("tasks_completed", progress.TasksCompleted),
		)
		return nil
	}
}

// unmarshalEvent attempts to unmarshal raw bytes into target.
// It first tries a CloudEvents v1.0 envelope (standard wrapping), then falls
// back to treating the payload as raw JSON.
func unmarshalEvent[T any](data []byte, target *T) error {
	var ce cloudevents.Event
	if err := json.Unmarshal(data, &ce); err == nil && ce.DataContentType() == "application/json" {
		return ce.DataAs(target)
	}
	return json.Unmarshal(data, target)
}

func computeProficiency(level int) float64 {
	if level <= 0 {
		return 0.0
	}
	p := float64(level) * 0.1
	if p > 1.0 {
		return 1.0
	}
	return p
}

func getTasksCompleted(p *profile.Profile) int {
	if p.Progress != nil {
		return p.Progress.TasksCompleted
	}
	return 0
}

func getCurrentStreak(p *profile.Profile) int {
	if p.Progress != nil {
		return p.Progress.CurrentStreak
	}
	return 0
}

func getTotalPoints(p *profile.Profile) int {
	if p.Progress != nil {
		return p.Progress.TotalPoints
	}
	return 0
}
