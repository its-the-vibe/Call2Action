package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"

	"github.com/its-the-vibe/Call2Action/internal/config"
	"github.com/its-the-vibe/Call2Action/internal/consumer"
	"github.com/its-the-vibe/Call2Action/internal/processor"
	"github.com/its-the-vibe/Call2Action/internal/publisher"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	// Load .env file if present (non-fatal if missing)
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		logger.Warn("could not load .env file", "err", err)
	}

	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "config.yaml"
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		logger.Error("failed to load config", "err", err)
		os.Exit(1)
	}

	redisOpts := &redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       cfg.Redis.DB,
	}
	redisClient := redis.NewClient(redisOpts)
	defer redisClient.Close()

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	if err := redisClient.Ping(ctx).Err(); err != nil {
		logger.Error("failed to connect to Redis", "addr", cfg.Redis.Addr, "err", err)
		os.Exit(1)
	}
	logger.Info("connected to Redis", "addr", cfg.Redis.Addr)

	pub := publisher.New(redisClient, cfg.Poppit.List, logger)
	proc := processor.New(cfg, pub, logger)
	cons := consumer.New(redisClient, cfg.Queue.Name, proc.Handle, logger)

	if err := cons.Run(ctx); err != nil {
		logger.Error("consumer error", "err", err)
		os.Exit(1)
	}
}
