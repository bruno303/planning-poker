package usecase

import "context"

// PresenceGuard serializes local presence transitions. A room lock protects a
// single room, but a room switch also changes the local count of the source
// room, which no single room lock covers. The guard is process-wide, so callers
// hold it only around the in-memory presence bookkeeping and never around
// network reads or socket writes.
type PresenceGuard interface {
	Lock(ctx context.Context) (func(), error)
}

type localPresenceGuard struct {
	transitions chan struct{}
}

func NewPresenceGuard() PresenceGuard {
	return &localPresenceGuard{transitions: make(chan struct{}, 1)}
}

func (g *localPresenceGuard) Lock(ctx context.Context) (func(), error) {
	select {
	case g.transitions <- struct{}{}:
		return func() { <-g.transitions }, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}
