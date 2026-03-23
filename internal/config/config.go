package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config is the top-level configuration structure.
type Config struct {
	Redis       RedisConfig                 `yaml:"redis"`
	Queue       QueueConfig                 `yaml:"queue"`
	Poppit      PoppitConfig                `yaml:"poppit"`
	Classifiers map[string]ClassifierConfig `yaml:"classifiers"`
}

// RedisConfig holds Redis connection settings.
// The password is read from the REDIS_PASSWORD environment variable.
type RedisConfig struct {
	Addr string `yaml:"addr"`
	DB   int    `yaml:"db"`
}

// QueueConfig holds the name of the incoming Redis list to consume messages from.
type QueueConfig struct {
	Name string `yaml:"name"`
}

// PoppitConfig holds the settings for publishing to Poppit.
type PoppitConfig struct {
	List   string `yaml:"list"`
	Repo   string `yaml:"repo"`
	Branch string `yaml:"branch"`
	Type   string `yaml:"type"`
	Dir    string `yaml:"dir"`
}

// ClassifierConfig holds the list of command templates for a classifier.
type ClassifierConfig struct {
	Commands []string `yaml:"commands"`
}

// Load reads a YAML configuration file and returns a Config.
func Load(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open config file: %w", err)
	}
	defer f.Close()

	var cfg Config
	if err := yaml.NewDecoder(f).Decode(&cfg); err != nil {
		return nil, fmt.Errorf("decode config file: %w", err)
	}

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &cfg, nil
}

func (c *Config) validate() error {
	if c.Redis.Addr == "" {
		return fmt.Errorf("redis.addr is required")
	}
	if c.Queue.Name == "" {
		return fmt.Errorf("queue.name is required")
	}
	if c.Poppit.List == "" {
		return fmt.Errorf("poppit.list is required")
	}
	return nil
}
