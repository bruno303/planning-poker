package http

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type (
	RedisClient interface {
		Ping(ctx context.Context) *redis.StatusCmd
	}

	RedisHealthChecker struct {
		client RedisClient
		name   string
	}
)

func NewRedisHealthChecker(client RedisClient, name string) *RedisHealthChecker {
	return &RedisHealthChecker{
		client: client,
		name:   name,
	}
}

func (c *RedisHealthChecker) Name() string {
	return c.name
}

func (c *RedisHealthChecker) Check(ctx context.Context) HealthStatus {
	start := time.Now()

	err := c.client.Ping(ctx).Err()
	duration := time.Since(start)

	if err != nil {
		return HealthStatus{
			Status:  HealthStatusFail,
			Message: fmt.Sprintf("Redis connection failed: %v", err),
			Details: map[string]any{
				"response_time_ms": duration.Milliseconds(),
				"error":            err.Error(),
			},
		}
	}

	status := HealthStatusPass
	message := "Redis connection successful"

	if duration > 100*time.Millisecond {
		status = HealthStatusWarn
		message = fmt.Sprintf("Redis connection successful but slow (%v)", duration)
	}

	return HealthStatus{
		Status:  status,
		Message: message,
		Details: map[string]any{
			"response_time_ms": duration.Milliseconds(),
		},
	}
}
