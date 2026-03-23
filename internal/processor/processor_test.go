package processor_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/its-the-vibe/Call2Action/internal/config"
	"github.com/its-the-vibe/Call2Action/internal/processor"
	"github.com/its-the-vibe/Call2Action/internal/publisher"
)

// mockPublisher captures published payloads for assertion.
type mockPublisher struct {
	payloads []publisher.Payload
	err      error
}

func (m *mockPublisher) Publish(_ context.Context, p publisher.Payload) error {
	if m.err != nil {
		return m.err
	}
	m.payloads = append(m.payloads, p)
	return nil
}

var discardLogger = slog.New(slog.NewTextHandler(io.Discard, nil))

func makeConfig() *config.Config {
	return &config.Config{
		Redis: config.RedisConfig{Addr: "localhost:6379"},
		Queue: config.QueueConfig{Name: "call2action:queue"},
		Poppit: config.PoppitConfig{
			List:   "poppit:notifications",
			Repo:   "its-the-vibe/Call2Action",
			Branch: "refs/heads/main",
			Type:   "call2action",
			Dir:    "/tmp",
		},
		Classifiers: map[string]config.ClassifierConfig{
			"documents": {Commands: []string{
				"process-doc --input {original_path} --output {new_path}",
			}},
			"images": {Commands: []string{
				"resize {original_path}",
				"convert {original_path} {new_path}",
			}},
			"empty": {},
		},
	}
}

func TestHandle_KnownClassifier(t *testing.T) {
	cfg := makeConfig()
	mock := &mockPublisher{}
	proc := processor.New(cfg, mock, discardLogger)

	msg := `{
"file_info": {"id":"F001","name":"report.pdf","title":"Q4","mimetype":"application/pdf","size":1024},
"original_path": "/downloads/report.pdf",
"new_path": "/classified/report.pdf",
"classifier_name": "documents",
"classified_at": "2025-01-15T12:34:56Z"
}`

	if err := proc.Handle(context.Background(), msg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(mock.payloads) != 1 {
		t.Fatalf("expected 1 payload, got %d", len(mock.payloads))
	}
	p := mock.payloads[0]
	if p.Repo != "its-the-vibe/Call2Action" {
		t.Errorf("repo = %q, want %q", p.Repo, "its-the-vibe/Call2Action")
	}
	if p.Branch != "refs/heads/main" {
		t.Errorf("branch = %q, want %q", p.Branch, "refs/heads/main")
	}
	if len(p.Commands) != 1 {
		t.Fatalf("expected 1 command, got %d", len(p.Commands))
	}
	want := "process-doc --input /downloads/report.pdf --output /classified/report.pdf"
	if p.Commands[0] != want {
		t.Errorf("command = %q, want %q", p.Commands[0], want)
	}
}

func TestHandle_UnknownClassifier(t *testing.T) {
	cfg := makeConfig()
	mock := &mockPublisher{}
	proc := processor.New(cfg, mock, discardLogger)

	msg := `{"classifier_name":"unknown","original_path":"/a","new_path":"/b"}`
	if err := proc.Handle(context.Background(), msg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(mock.payloads) != 0 {
		t.Errorf("expected no payloads for unknown classifier, got %d", len(mock.payloads))
	}
}

func TestHandle_InvalidJSON(t *testing.T) {
	cfg := makeConfig()
	mock := &mockPublisher{}
	proc := processor.New(cfg, mock, discardLogger)

	if err := proc.Handle(context.Background(), "not-json"); err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestHandle_MultipleCommands(t *testing.T) {
	cfg := makeConfig()
	mock := &mockPublisher{}
	proc := processor.New(cfg, mock, discardLogger)

	msg := `{
"classifier_name": "images",
"original_path": "/img/photo.jpg",
"new_path": "/out/photo.png"
}`
	if err := proc.Handle(context.Background(), msg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(mock.payloads) != 1 {
		t.Fatalf("expected 1 payload, got %d", len(mock.payloads))
	}
	if len(mock.payloads[0].Commands) != 2 {
		t.Errorf("expected 2 commands, got %d", len(mock.payloads[0].Commands))
	}
	if mock.payloads[0].Commands[0] != "resize /img/photo.jpg" {
		t.Errorf("command[0] = %q", mock.payloads[0].Commands[0])
	}
	if mock.payloads[0].Commands[1] != "convert /img/photo.jpg /out/photo.png" {
		t.Errorf("command[1] = %q", mock.payloads[0].Commands[1])
	}
}

func TestHandle_EmptyClassifierCommands(t *testing.T) {
	cfg := makeConfig()
	mock := &mockPublisher{}
	proc := processor.New(cfg, mock, discardLogger)

	msg := `{"classifier_name":"empty","original_path":"/a","new_path":"/b"}`
	if err := proc.Handle(context.Background(), msg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(mock.payloads) != 0 {
		t.Errorf("expected no payloads for empty classifier, got %d", len(mock.payloads))
	}
}

func TestHandle_PublisherError(t *testing.T) {
	cfg := makeConfig()
	mock := &mockPublisher{err: errors.New("redis unavailable")}
	proc := processor.New(cfg, mock, discardLogger)

	msg := `{"classifier_name":"documents","original_path":"/a","new_path":"/b"}`
	if err := proc.Handle(context.Background(), msg); err == nil {
		t.Fatal("expected error when publisher fails")
	}
}

func TestHandle_AllPlaceholders(t *testing.T) {
	cfg := makeConfig()
	cfg.Classifiers["full"] = config.ClassifierConfig{
		Commands: []string{
			"cmd {file_id} {file_name} {file_title} {file_mimetype} {file_size} {original_path} {new_path} {classifier_name} {classified_at}",
		},
	}
	mock := &mockPublisher{}
	proc := processor.New(cfg, mock, discardLogger)

	msg := `{
"file_info": {"id":"F001","name":"file.pdf","title":"Title","mimetype":"application/pdf","size":512},
"original_path": "/orig",
"new_path": "/new",
"classifier_name": "full",
"classified_at": "2025-01-01T00:00:00Z"
}`
	if err := proc.Handle(context.Background(), msg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "cmd F001 file.pdf Title application/pdf 512 /orig /new full 2025-01-01T00:00:00Z"
	if mock.payloads[0].Commands[0] != want {
		t.Errorf("command = %q, want %q", mock.payloads[0].Commands[0], want)
	}
}

func TestHandle_PerClassifierTypeAndDir(t *testing.T) {
	cfg := makeConfig()
	cfg.Classifiers["custom"] = config.ClassifierConfig{
		Type:     "custom-type",
		Dir:      "/custom/dir",
		Commands: []string{"run {original_path}"},
	}
	mock := &mockPublisher{}
	proc := processor.New(cfg, mock, discardLogger)

	msg := `{"classifier_name":"custom","original_path":"/a","new_path":"/b"}`
	if err := proc.Handle(context.Background(), msg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(mock.payloads) != 1 {
		t.Fatalf("expected 1 payload, got %d", len(mock.payloads))
	}
	p := mock.payloads[0]
	if p.Type != "custom-type" {
		t.Errorf("type = %q, want %q", p.Type, "custom-type")
	}
	if p.Dir != "/custom/dir" {
		t.Errorf("dir = %q, want %q", p.Dir, "/custom/dir")
	}
}

func TestHandle_GlobalTypeAndDirFallback(t *testing.T) {
	cfg := makeConfig()
	// documents classifier has no type/dir set – should fall back to global poppit values
	mock := &mockPublisher{}
	proc := processor.New(cfg, mock, discardLogger)

	msg := `{"classifier_name":"documents","original_path":"/a","new_path":"/b"}`
	if err := proc.Handle(context.Background(), msg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(mock.payloads) != 1 {
		t.Fatalf("expected 1 payload, got %d", len(mock.payloads))
	}
	p := mock.payloads[0]
	if p.Type != "call2action" {
		t.Errorf("type = %q, want %q (global fallback)", p.Type, "call2action")
	}
	if p.Dir != "/tmp" {
		t.Errorf("dir = %q, want %q (global fallback)", p.Dir, "/tmp")
	}
}

func TestHandle_MetadataContainsClassifierName(t *testing.T) {
	cfg := makeConfig()
	mock := &mockPublisher{}
	proc := processor.New(cfg, mock, discardLogger)

	msg := `{"classifier_name":"documents","original_path":"/a","new_path":"/b"}`
	if err := proc.Handle(context.Background(), msg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(mock.payloads) != 1 {
		t.Fatalf("expected 1 payload, got %d", len(mock.payloads))
	}
	p := mock.payloads[0]
	if p.Metadata == nil {
		t.Fatal("expected metadata to be set, got nil")
	}
	if got := p.Metadata["classifier_name"]; got != "documents" {
		t.Errorf("metadata[classifier_name] = %q, want %q", got, "documents")
	}
}
