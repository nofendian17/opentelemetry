package redis

import (
	"context"
	"fmt"
	"time"

	"go-app/internal/infrastructure/config"
	"go-app/internal/infrastructure/telemetry"

	"github.com/go-redis/redis/extra/redisotel/v8"
	"github.com/go-redis/redis/v8"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// Client wraps redis.Client with additional functionality
type Client struct {
	*redis.Client
	tracer trace.Tracer
}

// NewClient creates a new Redis client with best practices configuration
func NewClient(ctx context.Context, cfg config.RedisConfig, tel *telemetry.Telemetry) (*Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:            cfg.Addr,
		Password:        cfg.Password,
		DB:              cfg.DB,
		MaxRetries:      cfg.MaxRetries,
		MinRetryBackoff: 8 * time.Millisecond,
		MaxRetryBackoff: 512 * time.Millisecond,
		DialTimeout:     time.Duration(cfg.DialTimeout) * time.Second,
		ReadTimeout:     time.Duration(cfg.ReadTimeout) * time.Second,
		WriteTimeout:    time.Duration(cfg.WriteTimeout) * time.Second,
		PoolSize:        cfg.PoolSize,
		MinIdleConns:    cfg.MinIdleConns,
		MaxConnAge:      time.Duration(cfg.MaxConnAge) * time.Minute,
		PoolTimeout:     time.Duration(cfg.PoolTimeout) * time.Second,
		IdleTimeout:     time.Duration(cfg.IdleTimeout) * time.Minute,
	})

	// Add OpenTelemetry tracing
	rdb.AddHook(redisotel.NewTracingHook())

	// Test connection
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	telemetry.Log(ctx, telemetry.LevelInfo, "Successfully connected to Redis", nil,
		attribute.String("redis.addr", cfg.Addr),
		attribute.Int("redis.db", cfg.DB),
		attribute.Int("redis.pool_size", cfg.PoolSize),
	)

	return &Client{
		Client: rdb,
		tracer: tel.Tracer,
	}, nil
}

// HealthCheck performs a health check on the Redis connection
func (c *Client) HealthCheck(ctx context.Context) error {
	ctx, span := c.tracer.Start(ctx, "redis.health_check")
	defer span.End()

	if err := c.Ping(ctx).Err(); err != nil {
		span.SetAttributes(attribute.Bool("redis.healthy", false))
		return fmt.Errorf("redis health check failed: %w", err)
	}

	span.SetAttributes(attribute.Bool("redis.healthy", true))
	return nil
}

// GetPoolStats returns connection pool statistics
func (c *Client) GetPoolStats() *redis.PoolStats {
	return c.PoolStats()
}

// Close closes the Redis client
func (c *Client) Close() error {
	return c.Client.Close()
}

// Set with tracing and error handling
func (c *Client) SetWithTracing(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	ctx, span := c.tracer.Start(ctx, "redis.set")
	defer span.End()

	span.SetAttributes(
		attribute.String("redis.key", key),
		attribute.String("redis.operation", "set"),
		attribute.String("redis.expiration", expiration.String()),
	)

	err := c.Set(ctx, key, value, expiration).Err()
	if err != nil {
		span.SetAttributes(attribute.Bool("redis.error", true))
	}

	return err
}

// Get with tracing and error handling
func (c *Client) GetWithTracing(ctx context.Context, key string) (string, error) {
	ctx, span := c.tracer.Start(ctx, "redis.get")
	defer span.End()

	span.SetAttributes(
		attribute.String("redis.key", key),
		attribute.String("redis.operation", "get"),
	)

	result, err := c.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			span.SetAttributes(attribute.Bool("redis.key_not_found", true))
		} else {
			span.SetAttributes(attribute.Bool("redis.error", true))
		}
	}

	return result, err
}

// Del with tracing and error handling
func (c *Client) DelWithTracing(ctx context.Context, keys ...string) error {
	ctx, span := c.tracer.Start(ctx, "redis.del")
	defer span.End()

	span.SetAttributes(
		attribute.StringSlice("redis.keys", keys),
		attribute.String("redis.operation", "del"),
		attribute.Int("redis.key_count", len(keys)),
	)

	err := c.Del(ctx, keys...).Err()
	if err != nil {
		span.SetAttributes(attribute.Bool("redis.error", true))
	}

	return err
}
