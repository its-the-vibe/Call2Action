package publisher

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/redis/go-redis/v9"
)

// Payload is the JSON structure pushed to the Poppit Redis list.
type Payload struct {
	Repo     string   `json:"repo"`
	Branch   string   `json:"branch"`
	Type     string   `json:"type"`
	Dir      string   `json:"dir"`
	Commands []string `json:"commands"`
}

// Publisher publishes Poppit payloads to a Redis list.
type Publisher struct {
	client *redis.Client
	list   string
	logger *slog.Logger
}

// New creates a new Publisher.
func New(client *redis.Client, list string, logger *slog.Logger) *Publisher {
	return &Publisher{client: client, list: list, logger: logger}
}

// Publish pushes a Poppit payload to the configured Redis list.
func (p *Publisher) Publish(ctx context.Context, payload Payload) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	if err := p.client.RPush(ctx, p.list, data).Err(); err != nil {
		return fmt.Errorf("rpush %q: %w", p.list, err)
	}

	p.logger.Info("published poppit payload",
		"list", p.list,
		"repo", payload.Repo,
		"commands", len(payload.Commands),
	)
	return nil
}
