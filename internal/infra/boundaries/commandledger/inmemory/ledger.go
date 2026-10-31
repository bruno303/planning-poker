package inmemory

import (
	"context"
	"sync"
	"time"

	"planning-poker/internal/infra/bus"
)

type entry struct {
	pending   bool
	payload   []byte
	expiresAt time.Time
}

// Ledger is an in-memory bus.CommandLedger used when no shared store is
// available (tests and non-Redis environments).
type Ledger struct {
	mu         sync.Mutex
	entries    map[string]entry
	pendingTTL time.Duration
	resultTTL  time.Duration

	// now is a test seam so expiry can be exercised without sleeping.
	now func() time.Time
}

var _ bus.CommandLedger = (*Ledger)(nil)

func New(pendingTTL, resultTTL time.Duration) *Ledger {
	return &Ledger{
		entries:    make(map[string]entry),
		pendingTTL: pendingTTL,
		resultTTL:  resultTTL,
		now:        time.Now,
	}
}

func (l *Ledger) Claim(_ context.Context, roomID, clientID, commandID string) ([]byte, bool, error) {
	key := entryKey(roomID, clientID, commandID)

	l.mu.Lock()
	defer l.mu.Unlock()

	current, exists := l.entries[key]
	now := l.now()
	if exists && !now.Before(current.expiresAt) {
		delete(l.entries, key)
		exists = false
	}

	if exists {
		if current.pending {
			return nil, false, bus.ErrCommandInProgress
		}
		return current.payload, true, nil
	}

	l.entries[key] = entry{pending: true, expiresAt: now.Add(l.pendingTTL)}
	return nil, false, nil
}

func (l *Ledger) Complete(_ context.Context, roomID, clientID, commandID string, outcome []byte) error {
	key := entryKey(roomID, clientID, commandID)

	l.mu.Lock()
	defer l.mu.Unlock()

	l.entries[key] = entry{payload: outcome, expiresAt: l.now().Add(l.resultTTL)}
	return nil
}

func entryKey(roomID, clientID, commandID string) string {
	return roomID + ":" + clientID + ":" + commandID
}
