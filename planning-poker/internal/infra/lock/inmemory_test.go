package lock

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestInMemoryLock_LockAndUnlock(t *testing.T) {
	lock := newInMemoryLock()
	ctx := context.Background()

	locked := make(chan struct{})
	go func() {
		lock.Lock(ctx)
		locked <- struct{}{}
		time.Sleep(50 * time.Millisecond)
		lock.Unlock(ctx)
	}()

	<-locked // Wait until the first goroutine acquires the lock

	locked2 := make(chan struct{})
	go func() {
		lock.Lock(ctx)
		locked2 <- struct{}{}
		lock.Unlock(ctx)
	}()
	select {
	case <-locked2:
		// Lock acquired after unlock, as expected
	case <-time.After(100 * time.Millisecond):
		t.Error("second lock was not acquired after unlock")
	}
}

func TestInMemoryLockManager_WithLock(t *testing.T) {
	manager := NewInMemoryLockManager()
	ctx := context.Background()
	key := "resource-1"
	var called bool

	result, err := manager.WithLock(ctx, key, func(ctx context.Context) (any, error) {
		called = true
		return "ok", nil
	})
	if err != nil {
		t.Fatalf("WithLock returned error: %v", err)
	}
	if !called {
		t.Error("WithLock did not call the function")
	}
	if result != "ok" {
		t.Errorf("WithLock returned wrong result: got %v, want %v", result, "ok")
	}
}

func TestInMemoryLockManager_ExecuteWithLock(t *testing.T) {
	manager := NewInMemoryLockManager()
	ctx := context.Background()
	key := "resource-2"
	var called bool

	err := manager.ExecuteWithLock(ctx, key, func(ctx context.Context) error {
		called = true
		return nil
	})
	if err != nil {
		t.Fatalf("ExecuteWithLock returned error: %v", err)
	}
	if !called {
		t.Error("ExecuteWithLock did not call the function")
	}
}

func TestInMemoryLockManager_ConcurrentAccess(t *testing.T) {
	manager := NewInMemoryLockManager()
	ctx := context.Background()
	key := "shared"
	counter := 0
	wg := sync.WaitGroup{}
	mu := sync.Mutex{}

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := manager.WithLock(ctx, key, func(ctx context.Context) (any, error) {
				mu.Lock()
				counter++
				mu.Unlock()
				time.Sleep(10 * time.Millisecond)
				return nil, nil
			})
			if err != nil {
				t.Errorf("WithLock returned error: %v", err)
			}
		}()
	}
	wg.Wait()
	if counter != 10 {
		t.Errorf("expected counter to be 10, got %d", counter)
	}
}
