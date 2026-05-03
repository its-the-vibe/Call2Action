package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/its-the-vibe/Call2Action/internal/config"
)

const validYAML = `
redis:
  addr: "localhost:6379"
  db: 0
queue:
  name: "call2action:queue"
poppit:
  list: "poppit:notifications"
  repo: "its-the-vibe/Call2Action"
  branch: "refs/heads/main"
  type: "call2action"
  dir: "/tmp"
classifiers:
  documents:
    commands:
      - "process-doc --input {original_path} --output {new_path}"
  images:
    commands:
      - "process-img --file {original_path}"
`

func writeTemp(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "config-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString(content); err != nil {
		t.Fatal(err)
	}
	f.Close()
	return f.Name()
}

func TestLoad_Valid(t *testing.T) {
	path := writeTemp(t, validYAML)
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Redis.Addr != "localhost:6379" {
		t.Errorf("redis.addr = %q, want %q", cfg.Redis.Addr, "localhost:6379")
	}
	if cfg.Queue.Name != "call2action:queue" {
		t.Errorf("queue.name = %q, want %q", cfg.Queue.Name, "call2action:queue")
	}
	if cfg.Poppit.List != "poppit:notifications" {
		t.Errorf("poppit.list = %q, want %q", cfg.Poppit.List, "poppit:notifications")
	}
	if _, ok := cfg.Classifiers["documents"]; !ok {
		t.Error("expected classifier 'documents'")
	}
}

func TestLoad_MissingFile(t *testing.T) {
	_, err := config.Load(filepath.Join(t.TempDir(), "nonexistent.yaml"))
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestLoad_MissingRedisAddr(t *testing.T) {
	yaml := `
redis:
  db: 0
queue:
  name: "call2action:queue"
poppit:
  list: "poppit:notifications"
`
	path := writeTemp(t, yaml)
	_, err := config.Load(path)
	if err == nil {
		t.Fatal("expected validation error for missing redis.addr")
	}
}

func TestLoad_MissingQueueName(t *testing.T) {
	yaml := `
redis:
  addr: "localhost:6379"
poppit:
  list: "poppit:notifications"
`
	path := writeTemp(t, yaml)
	_, err := config.Load(path)
	if err == nil {
		t.Fatal("expected validation error for missing queue.name")
	}
}

func TestLoad_MissingPoppitList(t *testing.T) {
	yaml := `
redis:
  addr: "localhost:6379"
queue:
  name: "call2action:queue"
`
	path := writeTemp(t, yaml)
	_, err := config.Load(path)
	if err == nil {
		t.Fatal("expected validation error for missing poppit.list")
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	path := writeTemp(t, "not: valid: yaml: {[}")
	_, err := config.Load(path)
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}

func TestLoad_ClassifierTypeAndDir(t *testing.T) {
	yml := `
redis:
  addr: "localhost:6379"
queue:
  name: "call2action:queue"
poppit:
  list: "poppit:notifications"
classifiers:
  custom:
    type: "custom-type"
    dir: "/custom/dir"
    commands:
      - "run {original_path}"
`
	path := writeTemp(t, yml)
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	c, ok := cfg.Classifiers["custom"]
	if !ok {
		t.Fatal("expected classifier 'custom'")
	}
	if c.Type != "custom-type" {
		t.Errorf("type = %q, want %q", c.Type, "custom-type")
	}
	if c.Dir != "/custom/dir" {
		t.Errorf("dir = %q, want %q", c.Dir, "/custom/dir")
	}
	if len(c.Commands) != 1 || c.Commands[0] != "run {original_path}" {
		t.Errorf("commands = %v, want [run {original_path}]", c.Commands)
	}
}

func TestLoad_RedisDB(t *testing.T) {
	yml := `
redis:
  addr: "localhost:6379"
  db: 3
queue:
  name: "call2action:queue"
poppit:
  list: "poppit:notifications"
`
	path := writeTemp(t, yml)
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Redis.DB != 3 {
		t.Errorf("redis.db = %d, want 3", cfg.Redis.DB)
	}
}
