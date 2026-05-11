// Package processor provides message processing logic for the Call2Action application.
package processor

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/its-the-vibe/Call2Action/internal/config"
	"github.com/its-the-vibe/Call2Action/internal/publisher"
)

// FileInfo contains metadata about the classified file.
type FileInfo struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Title    string `json:"title"`
	MIMEType string `json:"mimetype"`
	Size     int64  `json:"size"`
}

// Message is the incoming JSON payload from the queue.
type Message struct {
	FileInfo       FileInfo `json:"file_info"`
	OriginalPath   string   `json:"original_path"`
	NewPath        string   `json:"new_path"`
	ClassifierName string   `json:"classifier_name"`
	ClassifiedAt   string   `json:"classified_at"`
}

// Publisher is the interface used to publish Poppit payloads.
type Publisher interface {
	Publish(ctx context.Context, payload publisher.Payload) error
}

// Pusher is the interface used to push raw messages to a Redis list.
type Pusher interface {
	Push(ctx context.Context, key string, values ...string) error
}

// Processor handles incoming messages and dispatches Poppit commands.
type Processor struct {
	cfg       *config.Config
	publisher Publisher
	pusher    Pusher
	logger    *slog.Logger
}

// New creates a new Processor. pusher may be nil if no classifier uses rpush.
func New(cfg *config.Config, pub Publisher, pusher Pusher, logger *slog.Logger) *Processor {
	return &Processor{cfg: cfg, publisher: pub, pusher: pusher, logger: logger}
}

// Handle processes a raw JSON message string.
func (p *Processor) Handle(ctx context.Context, raw string) error {
	var msg Message
	if err := json.Unmarshal([]byte(raw), &msg); err != nil {
		return fmt.Errorf("unmarshal message: %w", err)
	}

	classifier, ok := p.cfg.Classifiers[msg.ClassifierName]
	if !ok {
		p.logger.Warn("no classifier config found", "classifier_name", msg.ClassifierName)
		return nil
	}

	hasCommands := len(classifier.Commands) > 0
	hasRPush := len(classifier.RPush) > 0 && classifier.RedisKey != ""

	if !hasCommands && !hasRPush {
		p.logger.Warn("classifier has no actions configured", "classifier_name", msg.ClassifierName)
		return nil
	}

	if hasCommands {
		commands := make([]string, 0, len(classifier.Commands))
		for _, tmpl := range classifier.Commands {
			commands = append(commands, substitute(tmpl, &msg))
		}

		poppitType := classifier.Type
		if poppitType == "" {
			poppitType = p.cfg.Poppit.Type
		}

		poppitDir := classifier.Dir
		if poppitDir == "" {
			poppitDir = p.cfg.Poppit.Dir
		}

		payload := publisher.Payload{
			Repo:     p.cfg.Poppit.Repo,
			Branch:   p.cfg.Poppit.Branch,
			Type:     poppitType,
			Dir:      poppitDir,
			Commands: commands,
			Metadata: map[string]string{"classifier_name": msg.ClassifierName},
		}

		if err := p.publisher.Publish(ctx, payload); err != nil {
			return fmt.Errorf("publish poppit payload: %w", err)
		}
	}

	if hasRPush {
		if p.pusher == nil {
			return fmt.Errorf("rpush is configured for classifier %q but no pusher is available", msg.ClassifierName)
		}
		messages := make([]string, 0, len(classifier.RPush))
		for _, tmpl := range classifier.RPush {
			messages = append(messages, substitute(tmpl, &msg))
		}
		if err := p.pusher.Push(ctx, classifier.RedisKey, messages...); err != nil {
			return fmt.Errorf("rpush to %q: %w", classifier.RedisKey, err)
		}
	}

	p.logger.Info("processed message",
		"classifier_name", msg.ClassifierName,
		"original_path", msg.OriginalPath,
		"new_path", msg.NewPath,
	)
	return nil
}

// substitute replaces template placeholders in a command string with values from msg.
// Supported placeholders: {file_id}, {file_name}, {file_title}, {file_mimetype},
// {file_size}, {original_path}, {new_path}, {classifier_name}, {classified_at}.
func substitute(tmpl string, msg *Message) string {
	r := strings.NewReplacer(
		"{file_id}", msg.FileInfo.ID,
		"{file_name}", msg.FileInfo.Name,
		"{file_title}", msg.FileInfo.Title,
		"{file_mimetype}", msg.FileInfo.MIMEType,
		"{file_size}", fmt.Sprintf("%d", msg.FileInfo.Size),
		"{original_path}", msg.OriginalPath,
		"{new_path}", msg.NewPath,
		"{classifier_name}", msg.ClassifierName,
		"{classified_at}", msg.ClassifiedAt,
	)
	return r.Replace(tmpl)
}
