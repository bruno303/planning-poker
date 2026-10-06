package redishub_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	toolkitmetric "github.com/bruno303/go-toolkit/pkg/metric"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"planning-poker/internal/application/planningpoker/metric"
	"planning-poker/internal/application/planningpoker/usecase"
	"planning-poker/internal/domain"
	redishub "planning-poker/internal/infra/boundaries/hub/redis"
	infralock "planning-poker/internal/infra/lock"
)

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

type stubBus struct {
	roomID string
}

func (b *stubBus) RoomID() string                  { return b.roomID }
func (b *stubBus) Detach()                         {}
func (b *stubBus) Close() error                    { return nil }
func (b *stubBus) Listen(context.Context)          {}
func (b *stubBus) Send(context.Context, any) error { return nil }

var _ domain.Bus = (*stubBus)(nil)

type failingPresenceClient struct {
	redishub.RedisClient
	failSubscription string
	failLoad         bool
}

func (c *failingPresenceClient) Subscribe(ctx context.Context, channels ...string) *redis.PubSub {
	sub := c.RedisClient.Subscribe(ctx, channels...)
	if len(channels) == 1 && channels[0] == c.failSubscription {
		_ = sub.Close()
	}
	return sub
}

func (c *failingPresenceClient) Get(ctx context.Context, key string) *redis.StringCmd {
	if c.failLoad {
		cmd := redis.NewStringCmd(ctx)
		cmd.SetErr(errors.New("injected load failure"))
		return cmd
	}
	return c.RedisClient.Get(ctx, key)
}

func TestIntegration_FailedPresenceTransitionsKeepMetricsBalanced(t *testing.T) {
	for _, failure := range []string{"subscription", "leave persistence"} {
		t.Run(failure, func(t *testing.T) {
			client := setupRedisClient(t)
			defer client.Close()
			boundary := &failingPresenceClient{RedisClient: client}
			ctx := context.Background()
			hub, err := redishub.NewRedisHub(ctx, boundary)
			require.NoError(t, err)
			defer func() { _ = hub.Close() }()
			recorder := newRecordingMeter()
			pokerMetric := metric.NewPlanningPokerMetricWithMeter(recorder)
			manager := infralock.NewInMemoryLockManager()
			join := usecase.NewJoinRoomUseCase(hub, manager, pokerMetric)
			leave := usecase.NewLeaveRoomUseCase(hub, manager, pokerMetric)
			roomA, err := hub.NewRoomWithID(ctx, "failure-room-a")
			require.NoError(t, err)
			_, err = join.Execute(ctx, usecase.JoinRoomCommand{RoomID: roomA.ID, SenderID: "failure-client", Bus: &stubBus{roomID: roomA.ID}})
			require.NoError(t, err)
			if failure == "subscription" {
				roomB, err := hub.NewRoomWithID(ctx, "failure-room-b")
				require.NoError(t, err)
				// Existing destination membership takes the reconnect path; the local
				// original bus still belongs to A and must survive failed subscription.
				roomB.NewClient("failure-client")
				require.NoError(t, hub.SaveRoom(ctx, roomB))
				boundary.failSubscription = "planning-poker:updates:" + roomB.ID
				_, err = join.Execute(ctx, usecase.JoinRoomCommand{RoomID: roomB.ID, SenderID: "failure-client", Bus: &stubBus{roomID: roomB.ID}})
				require.Error(t, err)
				require.True(t, domain.HasBusInRoom(hub, "failure-client", roomA.ID))
				assert.Equal(t, float64(1), recorder.value(metric.PlanningPokerActiveUsersMetric))
				assert.Equal(t, float64(1), recorder.value(metric.PlanningPokerActiveRoomsMetric))
			} else {
				boundary.failLoad = true
				require.Error(t, leave.Execute(ctx, usecase.LeaveRoomCommand{RoomID: roomA.ID, SenderID: "failure-client"}))
				assert.Zero(t, recorder.value(metric.PlanningPokerActiveUsersMetric))
				assert.Zero(t, recorder.value(metric.PlanningPokerActiveRoomsMetric))
				boundary.failLoad = false
			}
			require.NoError(t, leave.Execute(ctx, usecase.LeaveRoomCommand{RoomID: roomA.ID, SenderID: "failure-client"}))
			assert.Zero(t, recorder.value(metric.PlanningPokerActiveUsersMetric))
			assert.Zero(t, recorder.value(metric.PlanningPokerActiveRoomsMetric))
		})
	}
}
