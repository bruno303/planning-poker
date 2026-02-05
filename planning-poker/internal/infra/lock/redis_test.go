package lock

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

func setupRedisLockManagerTest(t *testing.T) (*RedisLockManager, *redis.Client) {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   1, // Use DB 1 for tests to avoid conflicts
	})

	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Skip("Redis not available for testing")
	}

	// Clean up any existing locks
	keys, _ := client.Keys(ctx, lockKeyPrefix+"*").Result()
	if len(keys) > 0 {
		client.Del(ctx, keys...)
	}

	manager := NewRedisLockManager(client)
	return manager, client
}

func TestNewRedisLockManager(t *testing.T) {
	client := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	manager := NewRedisLockManager(client)

	if manager == nil {
		t.Fatal("expected manager to be non-nil")
	}
	if manager.client != client {
		t.Error("expected client to be set correctly")
	}
	if manager.lockTimeout != defaultLockTimeout {
		t.Errorf("expected lockTimeout %v, got %v", defaultLockTimeout, manager.lockTimeout)
	}
	if manager.retryDelay != defaultRetryDelay {
		t.Errorf("expected retryDelay %v, got %v", defaultRetryDelay, manager.retryDelay)
	}
	if manager.maxRetries != defaultMaxRetries {
		t.Errorf("expected maxRetries %d, got %d", defaultMaxRetries, manager.maxRetries)
	}
}

func TestRedisLockManager_WithLock_Success(t *testing.T) {
	manager, client := setupRedisLockManagerTest(t)
	defer client.Close()

	ctx := context.Background()
	key := "test-lock-1"
	executed := false

	result, err := manager.WithLock(ctx, key, func(ctx context.Context) (any, error) {
		executed = true
		return "success", nil
	})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !executed {
		t.Error("expected function to be executed")
	}
	if result != "success" {
		t.Errorf("expected result 'success', got %v", result)
	}

	// Verify lock was released
	lockKey := lockKeyPrefix + key
	exists, _ := client.Exists(ctx, lockKey).Result()
	if exists != 0 {
		t.Error("expected lock to be released after execution")
	}
}

func TestRedisLockManager_WithLock_FunctionError(t *testing.T) {
	manager, client := setupRedisLockManagerTest(t)
	defer client.Close()

	ctx := context.Background()
	key := "test-lock-2"
	expectedError := errors.New("function error")

	_, err := manager.WithLock(ctx, key, func(ctx context.Context) (any, error) {
		return nil, expectedError
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err != expectedError {
		t.Errorf("expected error %v, got %v", expectedError, err)
	}

	// Verify lock was released even after error
	lockKey := lockKeyPrefix + key
	exists, _ := client.Exists(ctx, lockKey).Result()
	if exists != 0 {
		t.Error("expected lock to be released after error")
	}
}

func TestRedisLockManager_ExecuteWithLock_Success(t *testing.T) {
	manager, client := setupRedisLockManagerTest(t)
	defer client.Close()

	ctx := context.Background()
	key := "test-lock-3"
	executed := false

	err := manager.ExecuteWithLock(ctx, key, func(ctx context.Context) error {
		executed = true
		return nil
	})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !executed {
		t.Error("expected function to be executed")
	}

	// Verify lock was released
	lockKey := lockKeyPrefix + key
	exists, _ := client.Exists(ctx, lockKey).Result()
	if exists != 0 {
		t.Error("expected lock to be released after execution")
	}
}

func TestRedisLockManager_ExecuteWithLock_FunctionError(t *testing.T) {
	manager, client := setupRedisLockManagerTest(t)
	defer client.Close()

	ctx := context.Background()
	key := "test-lock-4"
	expectedError := errors.New("execution error")

	err := manager.ExecuteWithLock(ctx, key, func(ctx context.Context) error {
		return expectedError
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err != expectedError {
		t.Errorf("expected error %v, got %v", expectedError, err)
	}

	// Verify lock was released even after error
	lockKey := lockKeyPrefix + key
	exists, _ := client.Exists(ctx, lockKey).Result()
	if exists != 0 {
		t.Error("expected lock to be released after error")
	}
}

func TestRedisLockManager_ConcurrentAccess(t *testing.T) {
	manager, client := setupRedisLockManagerTest(t)
	defer client.Close()

	ctx := context.Background()
	key := "test-lock-concurrent"
	
	var counter int
	var mu sync.Mutex
	var wg sync.WaitGroup

	numGoroutines := 10
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			_ = manager.ExecuteWithLock(ctx, key, func(ctx context.Context) error {
				// Critical section
				mu.Lock()
				temp := counter
				mu.Unlock()
				
				time.Sleep(10 * time.Millisecond) // Simulate work
				
				mu.Lock()
				counter = temp + 1
				mu.Unlock()
				
				return nil
			})
		}()
	}

	wg.Wait()

	if counter != numGoroutines {
		t.Errorf("expected counter to be %d, got %d - lock did not prevent race condition", numGoroutines, counter)
	}
}

func TestRedisLockManager_ContextCancellation(t *testing.T) {
	manager, client := setupRedisLockManagerTest(t)
	defer client.Close()

	// First, acquire a lock that we won't release
	ctx := context.Background()
	key := "test-lock-cancel"
	lockKey := lockKeyPrefix + key
	
	client.Set(ctx, lockKey, "held", 30*time.Second)

	// Now try to acquire with a context that will be cancelled
	ctx2, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Reduce retries for faster test
	manager.maxRetries = 5
	manager.retryDelay = 20 * time.Millisecond

	err := manager.ExecuteWithLock(ctx2, key, func(ctx context.Context) error {
		t.Error("function should not be executed when context is cancelled")
		return nil
	})

	if err == nil {
		t.Fatal("expected error due to context cancellation or timeout")
	}

	// Cleanup
	client.Del(ctx, lockKey)
}

func TestRedisLockManager_LockExpiration(t *testing.T) {
	manager, client := setupRedisLockManagerTest(t)
	defer client.Close()

	// Set a very short lock timeout for testing
	manager.lockTimeout = 100 * time.Millisecond

	ctx := context.Background()
	key := "test-lock-expiration"

	err := manager.ExecuteWithLock(ctx, key, func(ctx context.Context) error {
		// Sleep longer than lock timeout
		time.Sleep(200 * time.Millisecond)
		return nil
	})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify lock no longer exists (expired or released)
	lockKey := lockKeyPrefix + key
	exists, _ := client.Exists(ctx, lockKey).Result()
	if exists != 0 {
		t.Error("expected lock to not exist after expiration/release")
	}
}
