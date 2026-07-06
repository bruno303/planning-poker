package lock_test

import (
	"context"
	"errors"
	"fmt"
	"planning-poker/internal/infra/lock"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

func setupRedisLockManagerTest(t *testing.T) (*lock.RedisLockManager, *redis.Client) {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   6,
	})

	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Fatal("Redis not available for testing")
	}
	client.FlushDB(ctx)

	manager := lock.NewRedisLockManager(client)
	return manager, client
}

func TestRedisLockManager_WithLock_Success(t *testing.T) {
	manager, client := setupRedisLockManagerTest(t)
	defer client.Close()

	ctx := context.Background()
	key := "test-lock-1" + uuid.NewString()
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
	exists, _ := client.Exists(ctx, key).Result()
	if exists != 0 {
		t.Error("expected lock to be released after execution")
	}
}

func TestRedisLockManager_WithLock_FunctionError(t *testing.T) {
	manager, client := setupRedisLockManagerTest(t)
	defer client.Close()

	ctx := context.Background()
	key := "test-lock-2" + uuid.NewString()
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
	exists, _ := client.Exists(ctx, key).Result()
	if exists != 0 {
		t.Error("expected lock to be released after error")
	}
}

func TestRedisLockManager_ExecuteWithLock_Success(t *testing.T) {
	manager, client := setupRedisLockManagerTest(t)
	defer client.Close()

	ctx := context.Background()
	key := "test-lock-3" + uuid.NewString()
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
	exists, _ := client.Exists(ctx, key).Result()
	if exists != 0 {
		t.Error("expected lock to be released after execution")
	}
}

func TestRedisLockManager_ExecuteWithLock_FunctionError(t *testing.T) {
	manager, client := setupRedisLockManagerTest(t)
	defer client.Close()

	ctx := context.Background()
	key := "test-lock-4" + uuid.NewString()
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
	exists, _ := client.Exists(ctx, key).Result()
	if exists != 0 {
		t.Error("expected lock to be released after error")
	}
}

func TestRedisLockManager_ConcurrentAccess(t *testing.T) {
	manager, client := setupRedisLockManagerTest(t)
	defer client.Close()

	ctx := context.Background()
	key := "test-lock-concurrent" + uuid.NewString()

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
	key := "test-lock-cancel" + uuid.NewString()
	keyWithPrefix := fmt.Sprintf("planning-poker:lock:%s", key)

	client.Set(ctx, keyWithPrefix, "held", 30*time.Second)

	// Now try to acquire with a context that will be cancelled
	ctx2, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Reduce retries for faster test
	manager.SetRetry(6, 20*time.Millisecond)

	err := manager.ExecuteWithLock(ctx2, key, func(ctx context.Context) error {
		t.Error("function should not be executed when context is cancelled")
		return nil
	})

	if err == nil {
		t.Fatal("expected error due to context cancellation or timeout")
	}

	// Cleanup
	client.Del(ctx, keyWithPrefix)
}

func TestRedisLockManager_LockExpiration(t *testing.T) {
	manager, client := setupRedisLockManagerTest(t)
	defer client.Close()

	// Set a very short lock timeout for testing
	manager.SetLockTimeout(100 * time.Millisecond)

	ctx := context.Background()
	key := "test-lock-expiration" + uuid.NewString()

	err := manager.ExecuteWithLock(ctx, key, func(ctx context.Context) error {
		// Sleep longer than lock timeout
		time.Sleep(200 * time.Millisecond)
		return nil
	})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify lock no longer exists (expired or released)
	exists, _ := client.Exists(ctx, key).Result()
	if exists != 0 {
		t.Error("expected lock to not exist after expiration/release")
	}
}
