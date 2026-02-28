// Package kafka provides a generic Kafka consumer for the profile service.
package kafka

import (
	"context"

	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

// MessageHandler is called for each message received from Kafka.
// Returning nil commits the offset. Returning an error skips the commit
// (message will be redelivered on next consumer group rebalance or restart).
type MessageHandler func(ctx context.Context, msg kafka.Message) error

// Consumer wraps a kafka.Reader for a single topic with a consumer group.
type Consumer struct {
	reader  *kafka.Reader
	handler MessageHandler
	logger  *zap.Logger
}

// NewConsumer creates a Kafka consumer for the given topic and consumer group.
func NewConsumer(brokers []string, topic, groupID string, handler MessageHandler, logger *zap.Logger) *Consumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  brokers,
		Topic:    topic,
		GroupID:  groupID,
		MinBytes: 1,
		MaxBytes: 10e6, // 10 MB
	})
	return &Consumer{reader: reader, handler: handler, logger: logger}
}

// Run starts the consume loop. Blocks until ctx is cancelled.
func (c *Consumer) Run(ctx context.Context) error {
	for {
		msg, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return nil // clean shutdown
			}
			c.logger.Error("kafka fetch error", zap.Error(err))
			continue
		}

		if err := c.handler(ctx, msg); err != nil {
			c.logger.Error("message handler error — skipping commit",
				zap.String("topic", msg.Topic),
				zap.Int64("offset", msg.Offset),
				zap.Error(err),
			)
			continue
		}

		if err := c.reader.CommitMessages(ctx, msg); err != nil {
			c.logger.Error("kafka commit error", zap.Error(err))
		}
	}
}

// Close releases the reader resources.
func (c *Consumer) Close() error {
	return c.reader.Close()
}
