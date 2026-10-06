package usecase

import "context"

// localPresenceTransitions serializes presence snapshots, bus changes, and metric
// deltas on this process. Room locks alone cannot protect a switch's source room.
// Waiting is cancellable because a transition may perform network operations.
var localPresenceTransitions = make(chan struct{}, 1)

func lockLocalPresence(ctx context.Context) (func(), error) {
	select {
	case localPresenceTransitions <- struct{}{}:
		return func() { <-localPresenceTransitions }, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}
