package redishub_test

import (
	"context"
	"sync"
	"testing"

	toolkitmetric "github.com/bruno303/go-toolkit/pkg/metric"
	"github.com/stretchr/testify/assert"

	"planning-poker/internal/application/planningpoker/metric"
	"planning-poker/internal/application/planningpoker/usecase"
	"planning-poker/internal/domain"
	redishub "planning-poker/internal/infra/boundaries/hub/redis"
	infralock "planning-poker/internal/infra/lock"
)

// recordingMeter captures the delta counters the use cases emit.
type recordingMeter struct {
	mu    sync.Mutex
	calls map[string]float64
}

func newRecordingMeter() *recordingMeter {
	return &recordingMeter{calls: map[string]float64{}}
}

func (m *recordingMeter) AddCounter(_ context.Context, name, _, _ string, value float64, _ ...toolkitmetric.Attribute) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls[name] += value
	return nil
}

func (m *recordingMeter) AddGauge(_ context.Context, name, _, _ string, value float64, _ ...toolkitmetric.Attribute) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls[name] += value
	return nil
}

func (m *recordingMeter) value(name string) float64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.calls[name]
}

// TestIntegration_ActiveMetricsFollowLocalPresenceOnRedisHub runs the real join
// and leave use cases against the production RedisHub to confirm the
// process-local active counters track the hub's local presence, including a
// client that switches rooms while the previous socket is still open.
func TestIntegration_ActiveMetricsFollowLocalPresenceOnRedisHub(t *testing.T) {
	client := setupRedisClient(t)
	defer client.Close()

	hub, err := redishub.NewRedisHub(context.Background(), client)
	assert.NoError(t, err)
	defer func() { _ = hub.Close() }()

	recorder := newRecordingMeter()
	pokerMetric := metric.NewPlanningPokerMetricWithMeter(recorder)
	lockManager := infralock.NewInMemoryLockManager()
	ctx := context.Background()

	join := usecase.NewJoinRoomUseCase(hub, lockManager, pokerMetric)
	leave := usecase.NewLeaveRoomUseCase(hub, lockManager, pokerMetric)

	roomA, err := hub.NewRoomWithID(ctx, "metrics-room-a")
	assert.NoError(t, err)
	roomB, err := hub.NewRoomWithID(ctx, "metrics-room-b")
	assert.NoError(t, err)
	assert.NotEmpty(t, roomA.ID)
	assert.NotEmpty(t, roomB.ID)

	_, err = join.Execute(ctx, usecase.JoinRoomCommand{RoomID: roomA.ID, SenderID: "metrics-client", Bus: &stubBus{roomID: roomA.ID}})
	assert.NoError(t, err)

	// The client opens the second room while the first socket is still open.
	_, err = join.Execute(ctx, usecase.JoinRoomCommand{RoomID: roomB.ID, SenderID: "metrics-client", Bus: &stubBus{roomID: roomB.ID}})
	assert.NoError(t, err)

	// The late close of the first socket must not drift the counters: the client
	// is still connected through the second room.
	err = leave.Execute(ctx, usecase.LeaveRoomCommand{RoomID: roomA.ID, SenderID: "metrics-client"})
	assert.NoError(t, err)
	assert.Equal(t, float64(1), recorder.value(metric.PlanningPokerActiveUsersMetric))
	assert.Equal(t, float64(1), recorder.value(metric.PlanningPokerActiveRoomsMetric))

	err = leave.Execute(ctx, usecase.LeaveRoomCommand{RoomID: roomB.ID, SenderID: "metrics-client"})
	assert.NoError(t, err)
	assert.Zero(t, recorder.value(metric.PlanningPokerActiveUsersMetric))
	assert.Zero(t, recorder.value(metric.PlanningPokerActiveRoomsMetric))
}

// stubBus is a no-op domain.Bus bound to a room.
type stubBus struct {
	roomID string
}

func (b *stubBus) RoomID() string                  { return b.roomID }
func (b *stubBus) Detach()                         {}
func (b *stubBus) Close() error                    { return nil }
func (b *stubBus) Listen(context.Context)          {}
func (b *stubBus) Send(context.Context, any) error { return nil }

var _ domain.Bus = (*stubBus)(nil)
