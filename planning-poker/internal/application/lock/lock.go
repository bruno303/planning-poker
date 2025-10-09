package lock

import "context"

type Lock interface {
	Lock(context.Context)
	Unlock(context.Context)
}

type LockManager interface {
	ExecuteWithLock(ctx context.Context, key string, fn func(context.Context) error) error
	WithLock(ctx context.Context, key string, fn func(context.Context) (any, error)) (any, error)
}
