package lock_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"planning-poker/internal/infra/lock"
)

func TestInMemoryLockManagerSerializesCallbacksAndReleasesAfterError(t *testing.T) {
	manager := lock.NewInMemoryLockManager()
	const callbackCount = 8
	start := make(chan struct{})
	entered := make(chan struct{}, callbackCount)
	var wg sync.WaitGroup
	var mu sync.Mutex
	active, maxActive := 0, 0
	completed := 0
	errorsSeen := 0

	for i := 0; i < callbackCount; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			err := manager.ExecuteWithLock(context.Background(), "shared", func(context.Context) error {
				mu.Lock()
				active++
				if active > maxActive {
					maxActive = active
				}
				mu.Unlock()
				entered <- struct{}{}
				mu.Lock()
				active--
				completed++
				mu.Unlock()
				if i == 0 {
					return errors.New("callback failed")
				}
				return nil
			})
			if err != nil {
				mu.Lock()
				errorsSeen++
				mu.Unlock()
			}
		}(i)
	}
	close(start)
	wg.Wait()

	if len(entered) != callbackCount || completed != callbackCount {
		t.Fatalf("expected all callbacks to execute, entered=%d completed=%d", len(entered), completed)
	}
	if maxActive != 1 {
		t.Fatalf("expected callbacks for one key to be serialized, max active=%d", maxActive)
	}
	if errorsSeen != 1 {
		t.Fatalf("expected one callback error, got %d", errorsSeen)
	}

	if err := manager.ExecuteWithLock(context.Background(), "shared", func(context.Context) error { return nil }); err != nil {
		t.Fatalf("expected lock to be released after callback error: %v", err)
	}
}

func TestInMemoryLockManagerAllowsIndependentKeys(t *testing.T) {
	manager := lock.NewInMemoryLockManager()
	started := make(chan struct{}, 2)
	release := make(chan struct{})
	var wg sync.WaitGroup
	for _, key := range []string{"one", "two"} {
		wg.Add(1)
		go func(key string) {
			defer wg.Done()
			_ = manager.ExecuteWithLock(context.Background(), key, func(context.Context) error {
				started <- struct{}{}
				<-release
				return nil
			})
		}(key)
	}
	for i := 0; i < 2; i++ {
		<-started
	}
	close(release)
	wg.Wait()
}
