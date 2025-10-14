package lock

import "context"

type LockManager interface {
	ExecuteWithLock(ctx context.Context, key string, fn func(context.Context) error) error
	WithLock(ctx context.Context, key string, fn func(context.Context) (any, error)) (any, error)
}
