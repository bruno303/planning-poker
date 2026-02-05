package lock

import (
	"context"
	"fmt"
	"planning-poker/internal/application/lock"
	"time"

	"github.com/bruno303/go-toolkit/pkg/log"
	"github.com/bruno303/go-toolkit/pkg/trace"
	"github.com/redis/go-redis/v9"
)

type RedisLockManager struct {
	client      *redis.Client
	lockTimeout time.Duration
	retryDelay  time.Duration
	maxRetries  int
	logger      log.Logger
}

var _ lock.LockManager = (*RedisLockManager)(nil)

const (
	defaultLockTimeout = 10 * time.Second
	defaultRetryDelay  = 50 * time.Millisecond
	defaultMaxRetries  = 100
	lockKeyPrefix      = "planning-poker:lock:"
)

func NewRedisLockManager(client *redis.Client) *RedisLockManager {
	return &RedisLockManager{
		client:      client,
		lockTimeout: defaultLockTimeout,
		retryDelay:  defaultRetryDelay,
		maxRetries:  defaultMaxRetries,
		logger:      log.NewLogger("redis.lockmanager"),
	}
}

func (m *RedisLockManager) acquireLock(ctx context.Context, key string) (string, error) {
	lockKey := lockKeyPrefix + key
	lockValue := fmt.Sprintf("%d", time.Now().UnixNano())

	for i := 0; i < m.maxRetries; i++ {
		acquired, err := m.client.SetNX(ctx, lockKey, lockValue, m.lockTimeout).Result()
		if err != nil {
			return "", fmt.Errorf("failed to acquire lock: %w", err)
		}

		if acquired {
			m.logger.Debug(ctx, "Lock acquired for key '%s'", key)
			return lockValue, nil
		}

		m.logger.Debug(ctx, "Lock for key '%s' is held by another process, retrying... (attempt %d/%d)", key, i+1, m.maxRetries)
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(m.retryDelay):
			// Continue to next iteration
		}
	}

	return "", fmt.Errorf("failed to acquire lock for key '%s' after %d retries", key, m.maxRetries)
}

func (m *RedisLockManager) releaseLock(ctx context.Context, key string, lockValue string) error {
	lockKey := lockKeyPrefix + key

	script := `
		if redis.call("get", KEYS[1]) == ARGV[1] then
			return redis.call("del", KEYS[1])
		else
			return 0
		end
	`

	result, err := m.client.Eval(ctx, script, []string{lockKey}, lockValue).Result()
	if err != nil {
		return fmt.Errorf("failed to release lock: %w", err)
	}

	if result.(int64) == 1 {
		m.logger.Debug(ctx, "Lock released for key '%s'", key)
	} else {
		m.logger.Warn(ctx, "Lock for key '%s' was already released or expired", key)
	}

	return nil
}

func (m *RedisLockManager) WithLock(
	ctx context.Context,
	key string,
	fn func(ctx context.Context) (any, error),
) (any, error) {
	return trace.Trace(ctx, trace.NameConfig("RedisLockManager", "WithLock"), func(ctx context.Context) (any, error) {
		lockValue, err := m.acquireLock(ctx, key)
		if err != nil {
			return nil, err
		}

		defer func() {
			if releaseErr := m.releaseLock(ctx, key, lockValue); releaseErr != nil {
				m.logger.Error(ctx, fmt.Sprintf("Failed to release lock for key '%s'", key), releaseErr)
			}
		}()

		return fn(ctx)
	})
}

func (m *RedisLockManager) ExecuteWithLock(
	ctx context.Context,
	key string,
	fn func(context.Context) error,
) error {
	_, err := trace.Trace(ctx, trace.NameConfig("RedisLockManager", "ExecuteWithLock"), func(ctx context.Context) (any, error) {
		lockValue, err := m.acquireLock(ctx, key)
		if err != nil {
			return nil, err
		}

		defer func() {
			if releaseErr := m.releaseLock(ctx, key, lockValue); releaseErr != nil {
				m.logger.Error(ctx, fmt.Sprintf("Failed to release lock for key '%s'", key), releaseErr)
			}
		}()

		return nil, fn(ctx)
	})

	return err
}
