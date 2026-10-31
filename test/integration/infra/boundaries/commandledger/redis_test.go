package commandledgerredis_test

import (
	"context"
	"testing"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	commandledgerredis "planning-poker/internal/infra/boundaries/commandledger/redis"
	"planning-poker/internal/infra/bus"
	"planning-poker/test/integration"
)

const (
	testRoomID    = "room-1"
	testClientID  = "client-1"
	testCommandID = "6f1a0f9e-1a2b-4c3d-8e4f-5a6b7c8d9e0f"
)

func setupLedger(t *testing.T) (*redis.Client, *commandledgerredis.Ledger) {
	t.Helper()

	// Dedicated DB index: integration packages run in parallel, so every package
	// that flushes Redis must own a distinct index. Current ownership:
	// 1 = test/integration full server, 5 = hub, 6 = lock, 7 = command ledger.
	client := redis.NewClient(integration.RedisOptions(t, 7))
	client.FlushDB(context.Background())
	t.Cleanup(func() { _ = client.Close() })

	return client, commandledgerredis.New(client, bus.CommandPendingTTL, bus.CommandResultTTL)
}

func commandKey() string {
	return "planning-poker:command:" + testRoomID + ":" + testClientID + ":" + testCommandID
}

func TestIntegration_ClaimReservesCommand(t *testing.T) {
	client, ledger := setupLedger(t)
	ctx := context.Background()

	outcome, found, err := ledger.Claim(ctx, testRoomID, testClientID, testCommandID)
	require.NoError(t, err)
	assert.False(t, found)
	assert.Nil(t, outcome)

	_, found, err = ledger.Claim(ctx, testRoomID, testClientID, testCommandID)
	assert.False(t, found)
	assert.ErrorIs(t, err, bus.ErrCommandInProgress)

	exists, err := client.Exists(ctx, commandKey()).Result()
	require.NoError(t, err)
	assert.Equal(t, int64(1), exists, "expected the key %q to exist", commandKey())
	assert.Equal(t, "pending", client.Get(ctx, commandKey()).Val())

	ttl, err := client.TTL(ctx, commandKey()).Result()
	require.NoError(t, err)
	assert.Positive(t, ttl)
	assert.LessOrEqual(t, ttl, bus.CommandPendingTTL)
}

func TestIntegration_CompleteReplaysStoredOutcome(t *testing.T) {
	client, ledger := setupLedger(t)
	ctx := context.Background()
	payload := []byte(`{"type":"command-ack","commandId":"` + testCommandID + `","roomVersion":3}`)

	outcome, found, err := ledger.Claim(ctx, testRoomID, testClientID, testCommandID)
	require.NoError(t, err)
	assert.False(t, found)
	assert.Nil(t, outcome)

	require.NoError(t, ledger.Complete(ctx, testRoomID, testClientID, testCommandID, payload))

	outcome, found, err = ledger.Claim(ctx, testRoomID, testClientID, testCommandID)
	require.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, payload, outcome)

	ttl, err := client.TTL(ctx, commandKey()).Result()
	require.NoError(t, err)
	assert.Positive(t, ttl)
	assert.Greater(t, ttl, bus.CommandPendingTTL)
	assert.LessOrEqual(t, ttl, bus.CommandResultTTL)
}

func TestIntegration_ClaimAfterRemovedPendingReservation(t *testing.T) {
	client, ledger := setupLedger(t)
	ctx := context.Background()

	_, _, err := ledger.Claim(ctx, testRoomID, testClientID, testCommandID)
	require.NoError(t, err)

	require.NoError(t, client.Del(ctx, commandKey()).Err())

	outcome, found, err := ledger.Claim(ctx, testRoomID, testClientID, testCommandID)
	require.NoError(t, err)
	assert.False(t, found, "expected the expired reservation to be reclaimable")
	assert.Nil(t, outcome)

	ttl, err := client.TTL(ctx, commandKey()).Result()
	require.NoError(t, err)
	assert.Positive(t, ttl)
	assert.LessOrEqual(t, ttl, bus.CommandPendingTTL)
}
