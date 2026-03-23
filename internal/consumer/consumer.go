package consumer

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
)

// MessageHandler is a function that processes a raw message string.
type MessageHandler func(ctx context.Context, msg string) error

// Consumer reads messages from a Redis list using BLPOP and calls a handler for each message.
type Consumer struct {
	client    *redis.Client
	queueName string
	handler   MessageHandler
	logger    *slog.Logger
}

// New creates a new Consumer.
func New(client *redis.Client, queueName string, handler MessageHandler, logger *slog.Logger) *Consumer {
	return &Consumer{
		client:    client,
		queueName: queueName,
		handler:   handler,
		logger:    logger,
	}
}

// Run blocks and continuously pops messages from the queue until ctx is cancelled.
// It uses a 5-second BLPOP timeout so the loop can check for context cancellation.
func (c *Consumer) Run(ctx context.Context) error {
	c.logger.Info("starting consumer", "queue", c.queueName)
	for {
		select {
		case <-ctx.Done():
			c.logger.Info("consumer stopped")
			return nil
		default:
		}

		result, err := c.client.BLPop(ctx, 5*time.Second, c.queueName).Result()
		if err != nil {
			if err == redis.Nil {
				// timeout – no message, loop again
				continue
			}
			if ctx.Err() != nil {
				// context was cancelled while waiting
				c.logger.Info("consumer stopped")
				return nil
			}
			return fmt.Errorf("blpop %q: %w", c.queueName, err)
		}

		// BLPop returns [key, value]
		if len(result) < 2 {
			continue
		}
		msg := result[1]

		c.logger.Debug("received message", "queue", c.queueName)

		if err := c.handler(ctx, msg); err != nil {
			c.logger.Error("handler error", "err", err)
		}
	}
}
