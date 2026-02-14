package dapr

import (
	"fmt"
	"net/http"

	cloudevents "github.com/cloudevents/sdk-go/v2/event"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/MathTrail/profile-api/internal/profile"
)

// EventHandler handles Dapr pub/sub events and translates them into profile operations.
type EventHandler struct {
	profileService profile.Service
	logger         *zap.Logger
	pubSubName     string
	topicUserReg   string
	topicTaskSolve string
}

// NewEventHandler creates a new Dapr event handler.
func NewEventHandler(
	profileService profile.Service,
	logger *zap.Logger,
	pubSubName string,
	topicUserRegistered string,
	topicTaskSolved string,
) *EventHandler {
	return &EventHandler{
		profileService: profileService,
		logger:         logger,
		pubSubName:     pubSubName,
		topicUserReg:   topicUserRegistered,
		topicTaskSolve: topicTaskSolved,
	}
}

// RegisterRoutes registers Dapr pub/sub subscription and handler routes on the Gin engine.
func (eh *EventHandler) RegisterRoutes(router *gin.Engine) {
	// Dapr calls GET /dapr/subscribe to discover topic subscriptions
	router.GET("/dapr/subscribe", eh.handleSubscribe)

	// Topic handler endpoints
	router.POST("/events/user-registered", eh.handleUserRegistered)
	router.POST("/events/task-solved", eh.handleTaskSolved)
}

// handleSubscribe returns the list of pub/sub subscriptions for Dapr.
func (eh *EventHandler) handleSubscribe(c *gin.Context) {
	subscriptions := []Subscription{
		{
			PubSubName: eh.pubSubName,
			Topic:      eh.topicUserReg,
			Route:      "/events/user-registered",
		},
		{
			PubSubName: eh.pubSubName,
			Topic:      eh.topicTaskSolve,
			Route:      "/events/task-solved",
		},
	}
	c.JSON(http.StatusOK, subscriptions)
}

// handleUserRegistered processes CloudEvents from the user-registered topic.
// It creates a new profile for the registered user.
func (eh *EventHandler) handleUserRegistered(c *gin.Context) {
	var ce cloudevents.Event
	if err := c.ShouldBindJSON(&ce); err != nil {
		eh.logger.Error("failed to parse user-registered CloudEvent", zap.Error(err))
		c.JSON(http.StatusOK, TopicEventResponse{Status: StatusDrop})
		return
	}

	var event UserRegisteredEvent
	if err := ce.DataAs(&event); err != nil {
		eh.logger.Error("failed to extract user-registered event data from CloudEvent",
			zap.String("ce_id", ce.ID()),
			zap.String("ce_type", ce.Type()),
			zap.Error(err),
		)
		c.JSON(http.StatusOK, TopicEventResponse{Status: StatusDrop})
		return
	}

	if event.UserID == "" {
		eh.logger.Warn("user-registered event missing user_id, dropping",
			zap.String("ce_id", ce.ID()),
		)
		c.JSON(http.StatusOK, TopicEventResponse{Status: StatusDrop})
		return
	}

	eh.logger.Info("processing user-registered event",
		zap.String("ce_id", ce.ID()),
		zap.String("ce_source", ce.Source()),
		zap.String("user_id", event.UserID),
		zap.String("name", event.Name),
	)

	newProfile := &profile.Profile{
		UserID: event.UserID,
		Name:   event.Name,
		Age:    event.Age,
	}

	if err := eh.profileService.CreateProfile(c.Request.Context(), newProfile); err != nil {
		eh.logger.Error("failed to create profile from user-registered event",
			zap.String("user_id", event.UserID),
			zap.Error(err),
		)
		// Retry so Dapr redelivers the message
		c.JSON(http.StatusOK, TopicEventResponse{Status: StatusRetry})
		return
	}

	eh.logger.Info("profile created from user-registered event",
		zap.String("user_id", event.UserID),
		zap.String("profile_id", newProfile.ID.String()),
	)

	c.JSON(http.StatusOK, TopicEventResponse{Status: StatusSuccess})
}

// handleTaskSolved processes CloudEvents from the task-solved topic.
// It updates skills and progress for the user who solved the task.
func (eh *EventHandler) handleTaskSolved(c *gin.Context) {
	var ce cloudevents.Event
	if err := c.ShouldBindJSON(&ce); err != nil {
		eh.logger.Error("failed to parse task-solved CloudEvent", zap.Error(err))
		c.JSON(http.StatusOK, TopicEventResponse{Status: StatusDrop})
		return
	}

	var event TaskSolvedEvent
	if err := ce.DataAs(&event); err != nil {
		eh.logger.Error("failed to extract task-solved event data from CloudEvent",
			zap.String("ce_id", ce.ID()),
			zap.String("ce_type", ce.Type()),
			zap.Error(err),
		)
		c.JSON(http.StatusOK, TopicEventResponse{Status: StatusDrop})
		return
	}

	if event.UserID == "" || event.TaskID == "" {
		eh.logger.Warn("task-solved event missing required fields, dropping",
			zap.String("ce_id", ce.ID()),
			zap.String("user_id", event.UserID),
			zap.String("task_id", event.TaskID),
		)
		c.JSON(http.StatusOK, TopicEventResponse{Status: StatusDrop})
		return
	}

	eh.logger.Info("processing task-solved event",
		zap.String("ce_id", ce.ID()),
		zap.String("ce_source", ce.Source()),
		zap.String("user_id", event.UserID),
		zap.String("task_id", event.TaskID),
		zap.Int("points_earned", event.PointsEarned),
	)

	ctx := c.Request.Context()

	// Look up existing profile
	existingProfile, err := eh.profileService.GetProfile(ctx, event.UserID)
	if err != nil {
		eh.logger.Error("failed to get profile for task-solved event",
			zap.String("user_id", event.UserID),
			zap.Error(err),
		)
		c.JSON(http.StatusOK, TopicEventResponse{Status: StatusRetry})
		return
	}

	// Build skills from the event
	if len(event.SkillsGained) > 0 {
		var skills []profile.Skill
		for skillName, level := range event.SkillsGained {
			skills = append(skills, profile.Skill{
				SkillName:   skillName,
				Level:       level,
				Proficiency: computeProficiency(level),
			})
		}

		if err := eh.profileService.UpdateSkills(ctx, event.UserID, existingProfile.ID, skills); err != nil {
			eh.logger.Error("failed to update skills from task-solved event",
				zap.String("user_id", event.UserID),
				zap.Error(err),
			)
			c.JSON(http.StatusOK, TopicEventResponse{Status: StatusRetry})
			return
		}
	}

	// Update progress
	progress := &profile.Progress{
		TasksCompleted: getProgressTasksCompleted(existingProfile) + 1,
		CurrentStreak:  getProgressCurrentStreak(existingProfile) + 1,
		TotalPoints:    getProgressTotalPoints(existingProfile) + event.PointsEarned,
	}

	if err := eh.profileService.UpdateProgress(ctx, event.UserID, existingProfile.ID, progress); err != nil {
		eh.logger.Error("failed to update progress from task-solved event",
			zap.String("user_id", event.UserID),
			zap.Error(err),
		)
		c.JSON(http.StatusOK, TopicEventResponse{Status: StatusRetry})
		return
	}

	eh.logger.Info("profile updated from task-solved event",
		zap.String("user_id", event.UserID),
		zap.String("task_id", event.TaskID),
		zap.Int("new_total_points", progress.TotalPoints),
		zap.Int("tasks_completed", progress.TasksCompleted),
	)

	c.JSON(http.StatusOK, TopicEventResponse{Status: StatusSuccess})
}

// computeProficiency calculates a proficiency score (0.0–1.0) from a skill level.
func computeProficiency(level int) float64 {
	if level <= 0 {
		return 0.0
	}
	// Simple formula: proficiency = min(level * 0.1, 1.0)
	p := float64(level) * 0.1
	if p > 1.0 {
		return 1.0
	}
	return p
}

// Helper functions to safely extract progress fields from a profile.

func getProgressTasksCompleted(p *profile.Profile) int {
	if p.Progress != nil {
		return p.Progress.TasksCompleted
	}
	return 0
}

func getProgressCurrentStreak(p *profile.Profile) int {
	if p.Progress != nil {
		return p.Progress.CurrentStreak
	}
	return 0
}

func getProgressTotalPoints(p *profile.Profile) int {
	if p.Progress != nil {
		return p.Progress.TotalPoints
	}
	return 0
}

// Ensure fmt is used (for future error formatting).
var _ = fmt.Sprintf
