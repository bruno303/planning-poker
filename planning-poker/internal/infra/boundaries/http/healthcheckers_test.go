package http

import (
	"context"
	"testing"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

func TestRedisHealthChecker_Check_Success(t *testing.T) {
	client := &MockRedisClient{
		pingResult: nil,
	}

	checker := NewRedisHealthChecker(client, "redis")
	ctx := context.Background()

	status := checker.Check(ctx)

	assert.Equal(t, "redis", checker.Name())
	assert.Equal(t, HealthStatusPass, status.Status)
	assert.Contains(t, status.Message, "Redis connection successful")
	assert.Contains(t, status.Details, "response_time_ms")
}

func TestRedisHealthChecker_Check_Failure(t *testing.T) {
	client := &MockRedisClient{
		pingResult: assert.AnError,
	}

	checker := NewRedisHealthChecker(client, "redis")
	ctx := context.Background()

	status := checker.Check(ctx)

	assert.Equal(t, "redis", checker.Name())
	assert.Equal(t, HealthStatusFail, status.Status)
	assert.Contains(t, status.Message, "Redis connection failed")
	assert.Contains(t, status.Details["error"], assert.AnError.Error())
}

// Mock Redis client for testing
type MockRedisClient struct {
	pingResult error
}

func (m *MockRedisClient) Ping(ctx context.Context) *redis.StatusCmd {
	cmd := redis.NewStatusCmd(ctx)
	if m.pingResult != nil {
		cmd.SetErr(m.pingResult)
	} else {
		cmd.SetVal("PONG")
	}
	return cmd
}
