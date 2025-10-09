package lock

import (
	"context"
	"planning-poker/internal/application/lock"
	"sync"

	"github.com/bruno303/go-toolkit/pkg/log"
)

type (
	InMemoryLockManager struct {
		locks map[string]*InMemoryLock
		log   log.Logger
	}
	InMemoryLock struct {
		lock sync.Mutex
		log  log.Logger
	}
)

var (
	_ lock.Lock        = (*InMemoryLock)(nil)
	_ lock.LockManager = (*InMemoryLockManager)(nil)
)

func NewInMemoryLockManager() *InMemoryLockManager {
	return &InMemoryLockManager{
		locks: make(map[string]*InMemoryLock),
		log:   log.NewLogger("inmemory.lockmanager"),
	}
}

func newInMemoryLock() *InMemoryLock {
	return &InMemoryLock{
		log: log.NewLogger("inmemory.lock"),
	}
}

func (m *InMemoryLockManager) getLock(ctx context.Context, key string) *InMemoryLock {
	if lock, exists := m.locks[key]; exists {
		m.log.Debug(ctx, "reusing lock for key '%s'", key)
		return lock
	}

	m.log.Debug(ctx, "creating new lock for key '%s'", key)
	lock := newInMemoryLock()
	m.locks[key] = lock
	return lock
}

func (l *InMemoryLock) Lock(ctx context.Context) {
	l.log.Debug(ctx, "acquiring lock")
	l.lock.Lock()
}

func (l *InMemoryLock) Unlock(ctx context.Context) {
	l.log.Debug(ctx, "releasing lock")
	l.lock.Unlock()
}

func (m *InMemoryLockManager) WithLock(
	ctx context.Context,
	key string,
	fn func(ctx context.Context) (any, error),
) (any, error) {
	lock := m.getLock(ctx, key)
	lock.Lock(ctx)
	defer lock.Unlock(ctx)

	return fn(ctx)
}

func (m *InMemoryLockManager) ExecuteWithLock(
	ctx context.Context,
	key string,
	fn func(context.Context) error,
) error {
	lock := m.getLock(ctx, key)
	lock.Lock(ctx)
	defer lock.Unlock(ctx)

	return fn(ctx)
}
