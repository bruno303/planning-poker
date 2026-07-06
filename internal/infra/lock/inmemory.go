package lock

import (
	"context"
	"planning-poker/internal/application/lock"
	"sync"

	"github.com/bruno303/go-toolkit/pkg/log"
	"github.com/bruno303/go-toolkit/pkg/trace"
)

type (
	InMemoryLockManager struct {
		locks sync.Map
		log   log.Logger
	}
	inMemoryLock struct {
		lock sync.Mutex
		log  log.Logger
	}
)

var (
	_ lock.LockManager = (*InMemoryLockManager)(nil)
)

func NewInMemoryLockManager() *InMemoryLockManager {
	return &InMemoryLockManager{
		locks: sync.Map{},
		log:   log.NewLogger("inmemory.lockmanager"),
	}
}

func newInMemoryLock() *inMemoryLock {
	return &inMemoryLock{
		log: log.NewLogger("inmemory.lock"),
	}
}

func (m *InMemoryLockManager) getLock(ctx context.Context, key string) *inMemoryLock {
	lock, _ := trace.Trace(ctx, trace.NameConfig("InMemoryLockManager", "getLock"), func(ctx context.Context) (any, error) {
		lock := newInMemoryLock()
		actual, _ := m.locks.LoadOrStore(key, lock)
		return actual, nil
	})

	return lock.(*inMemoryLock)
}

func (l *inMemoryLock) Lock(ctx context.Context) {
	l.log.Debug(ctx, "acquiring lock")
	l.lock.Lock()
}

func (l *inMemoryLock) Unlock(ctx context.Context) {
	l.log.Debug(ctx, "releasing lock")
	l.lock.Unlock()
}

func (m *InMemoryLockManager) WithLock(
	ctx context.Context,
	key string,
	fn func(ctx context.Context) (any, error),
) (any, error) {
	return trace.Trace(ctx, trace.NameConfig("InMemoryLockManager", "WithLock"), func(ctx context.Context) (any, error) {
		lock := m.getLock(ctx, key)
		lock.Lock(ctx)
		defer lock.Unlock(ctx)

		return fn(ctx)
	})
}

func (m *InMemoryLockManager) ExecuteWithLock(
	ctx context.Context,
	key string,
	fn func(context.Context) error,
) error {
	_, err := trace.Trace(ctx, trace.NameConfig("InMemoryLockManager", "ExecuteWithLock"), func(ctx context.Context) (any, error) {
		lock := m.getLock(ctx, key)
		lock.Lock(ctx)
		defer lock.Unlock(ctx)

		return nil, fn(ctx)
	})

	return err
}
