package redis

import (
	"context"
	"errors"
	"fmt"
	"time"

	redislib "github.com/redis/go-redis/v9"

	"planning-poker/internal/infra/bus"
)

const (
	commandKeyPrefix = "planning-poker:command:"
	pendingValue     = "pending"
)

type RedisClient interface {
	SetNX(ctx context.Context, key string, value any, expiration time.Duration) *redislib.BoolCmd
	Get(ctx context.Context, key string) *redislib.StringCmd
	Set(ctx context.Context, key string, value any, expiration time.Duration) *redislib.StatusCmd
}

// Ledger is the Redis-backed bus.CommandLedger shared across application
// instances. A pending marker reserves the command while it executes; the
// stored frame replaces it once the command completes.
type Ledger struct {
	client     RedisClient
	pendingTTL time.Duration
	resultTTL  time.Duration
}

var _ bus.CommandLedger = (*Ledger)(nil)

func New(client RedisClient, pendingTTL, resultTTL time.Duration) *Ledger {
	return &Ledger{
		client:     client,
		pendingTTL: pendingTTL,
		resultTTL:  resultTTL,
	}
}

func (l *Ledger) Claim(ctx context.Context, roomID, clientID, commandID string) ([]byte, bool, error) {
	key := commandKey(roomID, clientID, commandID)

	claimed, err := l.client.SetNX(ctx, key, pendingValue, l.pendingTTL).Result()
	if err != nil {
		return nil, false, fmt.Errorf("claim command outcome: %w", err)
	}
	if claimed {
		return nil, false, nil
	}

	value, err := l.client.Get(ctx, key).Bytes()
	if errors.Is(err, redislib.Nil) {
		// The pending reservation expired between SetNX and Get; retry once.
		claimed, setErr := l.client.SetNX(ctx, key, pendingValue, l.pendingTTL).Result()
		if setErr != nil {
			return nil, false, fmt.Errorf("claim command outcome: %w", setErr)
		}
		if claimed {
			return nil, false, nil
		}
		value, err = l.client.Get(ctx, key).Bytes()
	}
	if err != nil {
		return nil, false, fmt.Errorf("claim command outcome: %w", err)
	}
	if string(value) == pendingValue {
		return nil, false, bus.ErrCommandInProgress
	}

	return value, true, nil
}

func (l *Ledger) Complete(ctx context.Context, roomID, clientID, commandID string, outcome []byte) error {
	key := commandKey(roomID, clientID, commandID)

	if err := l.client.Set(ctx, key, outcome, l.resultTTL).Err(); err != nil {
		return fmt.Errorf("complete command outcome: %w", err)
	}

	return nil
}

func commandKey(roomID, clientID, commandID string) string {
	return commandKeyPrefix + roomID + ":" + clientID + ":" + commandID
}
