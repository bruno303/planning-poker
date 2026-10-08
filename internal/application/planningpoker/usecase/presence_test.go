package usecase

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestPresenceGuardLockRespectsContextCancellation(t *testing.T) {
	guard := NewPresenceGuard()

	unlock, err := guard.Lock(context.Background())
	if err != nil {
		t.Fatalf("first lock: %v", err)
	}
	defer unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	if _, err := guard.Lock(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("got %v, want deadline exceeded", err)
	}
}

func TestPresenceGuardLockSerializesTransitions(t *testing.T) {
	guard := NewPresenceGuard()

	unlock, err := guard.Lock(context.Background())
	if err != nil {
		t.Fatalf("first lock: %v", err)
	}

	acquired := make(chan struct{})
	go func() {
		unlockNext, err := guard.Lock(context.Background())
		if err != nil {
			return
		}
		close(acquired)
		unlockNext()
	}()

	select {
	case <-acquired:
		t.Fatal("second lock acquired while the guard was held")
	case <-time.After(50 * time.Millisecond):
	}

	unlock()

	select {
	case <-acquired:
	case <-time.After(time.Second):
		t.Fatal("second lock did not acquire after the guard was released")
	}
}
