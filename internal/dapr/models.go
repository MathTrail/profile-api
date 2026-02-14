package dapr

import (
	"time"

	cloudevents "github.com/cloudevents/sdk-go/v2/event"
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

// NewCloudEvent creates a CloudEvents v1.0 envelope with the given type, source, and data.
func NewCloudEvent(eventType, source string, data interface{}) (*cloudevents.Event, error) {
	e := cloudevents.New()
	e.SetType(eventType)
	e.SetSource(source)
	if err := e.SetData(cloudevents.ApplicationJSON, data); err != nil {
		return nil, err
	}
	return &e, nil
}

// Subscription represents a Dapr pub/sub subscription entry
// returned by GET /dapr/subscribe.
type Subscription struct {
	PubSubName string `json:"pubsubname"`
	Topic      string `json:"topic"`
	Route      string `json:"route"`
}

// TopicEventResponse is the response Dapr expects from a topic handler.
// Status can be "SUCCESS", "RETRY", or "DROP".
type TopicEventResponse struct {
	Status string `json:"status"`
}

const (
	StatusSuccess = "SUCCESS"
	StatusRetry   = "RETRY"
	StatusDrop    = "DROP"
)
