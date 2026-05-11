// Package main is the entry point for the Call2Action application.
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
	os.Exit(run())
}

func run() int {
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
		return 1
	}

	redisOpts := &redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       cfg.Redis.DB,
	}
	redisClient := redis.NewClient(redisOpts)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)

	if err := redisClient.Ping(ctx).Err(); err != nil {
		logger.Error("failed to connect to Redis", "addr", cfg.Redis.Addr, "err", err)
		cancel()
		_ = redisClient.Close()
		return 1
	}

	defer cancel()
	defer func() {
		if err := redisClient.Close(); err != nil {
			logger.Error("failed to close redis client", "err", err)
		}
	}()

	logger.Info("connected to Redis", "addr", cfg.Redis.Addr)

	pub := publisher.New(redisClient, cfg.Poppit.List, logger)
	pusher := publisher.NewRedisPusher(redisClient, logger)
	proc := processor.New(cfg, pub, pusher, logger)
	cons := consumer.New(redisClient, cfg.Queue.Name, proc.Handle, logger)

	if err := cons.Run(ctx); err != nil {
		logger.Error("consumer error", "err", err)
		return 1
	}
	return 0
}
