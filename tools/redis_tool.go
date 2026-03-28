package tools

import (
	"context"
	"time"

	"github.com/eflowcr/eSTOCK_backend/configuration"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

// InitRedis connects to Redis using the REDIS_URL env var.
// Returns nil if REDIS_URL is not set (in-memory fallback will be used).
// Logs a warning if connection ping fails so the app still starts.
func InitRedis(cfg configuration.Config) *redis.Client {
	if cfg.RedisURL == "" {
		log.Info().Msg("REDIS_URL not set — using in-memory cache for stock alerts")
		return nil
	}

	opt, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		log.Warn().Err(err).Str("redis_url", cfg.RedisURL).Msg("invalid REDIS_URL — falling back to in-memory cache")
		return nil
	}

	client := redis.NewClient(opt)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		log.Warn().Err(err).Msg("Redis ping failed — falling back to in-memory cache")
		client.Close()
		return nil
	}

	log.Info().Str("addr", opt.Addr).Msg("Redis connected")
	return client
}
