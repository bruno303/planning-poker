package usecase

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	"planning-poker/internal/application/planningpoker/metric"
	"planning-poker/internal/domain"
	"planning-poker/internal/infra/boundaries/hub/inmemory"
	infralock "planning-poker/internal/infra/lock"

	"go.uber.org/mock/gomock"
)

type localPresenceBus struct {
	roomID  string
	sendErr error
}

func (b *localPresenceBus) RoomID() string                  { return b.roomID }
func (b *localPresenceBus) Detach()                         {}
func (b *localPresenceBus) Close() error                    { return nil }
func (b *localPresenceBus) Listen(context.Context)          {}
func (b *localPresenceBus) Send(context.Context, any) error { return b.sendErr }

func (b *localPresenceBus) String() string {
	return fmt.Sprintf("localPresenceBus(%s)", b.roomID)
}

type localPresenceHarness struct {
	hub      *inmemory.InMemoryHub
	join     JoinRoomUseCase
	leave    *leaveRoomUseCase
	recorder *metricRecorder
}

func newLocalPresenceHarness(t *testing.T) *localPresenceHarness {
	t.Helper()
	ctrl := gomock.NewController(t)
	t.Cleanup(ctrl.Finish)

	hub := inmemory.NewHub()
	lockManager := infralock.NewInMemoryLockManager()
	pokerMetric, recorder := newTestPlanningPokerMetric(ctrl)
	guard := NewPresenceGuard()

	return &localPresenceHarness{
		hub:      hub,
		join:     NewJoinRoomUseCase(hub, lockManager, pokerMetric, guard),
		leave:    NewLeaveRoomUseCase(hub, lockManager, pokerMetric, guard),
		recorder: recorder,
	}
}

func (h *localPresenceHarness) createRoom(t *testing.T, roomID string) {
	t.Helper()
	if _, err := h.hub.NewRoomWithID(context.Background(), roomID); err != nil {
		t.Fatalf("create room %s: %v", roomID, err)
	}
}

func (h *localPresenceHarness) joinRoom(t *testing.T, roomID, clientID string) {
	t.Helper()
	_, err := h.join.Execute(context.Background(), JoinRoomCommand{
		RoomID:   roomID,
		SenderID: clientID,
		Bus:      &localPresenceBus{roomID: roomID},
	})
	if err != nil {
		t.Fatalf("join %s as %s: %v", roomID, clientID, err)
	}
}

func (h *localPresenceHarness) leaveRoom(t *testing.T, roomID, clientID string) {
	t.Helper()
	if err := h.leave.Execute(context.Background(), LeaveRoomCommand{RoomID: roomID, SenderID: clientID}); err != nil {
		t.Fatalf("leave %s as %s: %v", roomID, clientID, err)
	}
}

func (h *localPresenceHarness) activeUsers() float64 {
	return sumMetricCalls(h.recorder.getCalls(), metric.PlanningPokerActiveUsersMetric)
}

func (h *localPresenceHarness) activeRooms() float64 {
	return sumMetricCalls(h.recorder.getCalls(), metric.PlanningPokerActiveRoomsMetric)
}

func TestActiveMetricsMatchLocalPresence(t *testing.T) {
	t.Parallel()
	type step struct {
		room, client string
		leave        bool
	}
	tests := []struct {
		name  string
		steps []step
	}{
		{"first join", []step{{"a", "1", false}}},
		{"shared room", []step{{"a", "1", false}, {"a", "2", false}}},
		{"two rooms", []step{{"a", "1", false}, {"b", "2", false}}},
		{"reconnect", []step{{"a", "1", false}, {"a", "1", false}}},
		{"reconnect after disconnect", []step{{"a", "1", false}, {"a", "1", true}, {"a", "1", false}}},
		{"switch then late leave", []step{{"a", "1", false}, {"b", "1", false}, {"a", "1", true}}},
		{"switch with another client remaining", []step{{"a", "1", false}, {"a", "2", false}, {"b", "1", false}, {"a", "1", true}}},
		{"switch to occupied room", []step{{"a", "1", false}, {"b", "2", false}, {"b", "1", false}, {"a", "1", true}}},
		{"switch then disconnect", []step{{"a", "1", false}, {"b", "1", false}, {"a", "1", true}, {"b", "1", true}}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := newLocalPresenceHarness(t)
			h.createRoom(t, "a")
			for i, step := range tc.steps {
				if step.leave {
					h.leaveRoom(t, step.room, step.client)
				} else {
					h.joinRoom(t, step.room, step.client)
				}
				users, rooms := 0, 0
				for _, room := range []string{"a", "b"} {
					count := h.hub.GetClientsOfRoom(room)
					users += count
					if count > 0 {
						rooms++
					}
				}
				if h.activeUsers() != float64(users) || h.activeRooms() != float64(rooms) {
					t.Fatalf("step %d: active users/rooms = %v/%v, local presence = %d/%d", i, h.activeUsers(), h.activeRooms(), users, rooms)
				}
				if !step.leave && !domain.HasBusInRoom(h.hub, step.client, step.room) {
					t.Fatal("joined bus belongs to another room")
				}
			}
		})
	}
}

var _ domain.Bus = (*localPresenceBus)(nil)

func TestConcurrentSwitchesFromSharedRoomKeepMetricsBalanced(t *testing.T) {
	for range 30 {
		h := newLocalPresenceHarness(t)
		h.createRoom(t, "source")
		h.createRoom(t, "destination-1")
		h.createRoom(t, "destination-2")
		h.joinRoom(t, "source", "client-1")
		h.joinRoom(t, "source", "client-2")
		start := make(chan struct{})
		results := make(chan error, 2)
		var wg sync.WaitGroup
		for i := 1; i <= 2; i++ {
			wg.Go(func() {
				<-start
				_, err := h.join.Execute(context.Background(), JoinRoomCommand{
					RoomID: fmt.Sprintf("destination-%d", i), SenderID: fmt.Sprintf("client-%d", i),
					Bus: &localPresenceBus{roomID: fmt.Sprintf("destination-%d", i)},
				})
				results <- err
			})
		}
		close(start)
		wg.Wait()
		close(results)
		for err := range results {
			if err != nil {
				t.Fatal(err)
			}
		}
		if got := h.activeRooms(); got != 2 {
			t.Fatalf("active rooms = %v, want 2", got)
		}
		h.leaveRoom(t, "source", "client-1")
		h.leaveRoom(t, "source", "client-2")
		h.leaveRoom(t, "destination-1", "client-1")
		h.leaveRoom(t, "destination-2", "client-2")
		if h.activeUsers() != 0 || h.activeRooms() != 0 {
			t.Fatal("metrics did not settle at zero")
		}
	}
}

func TestFailedReplacementJoinReconcilesPresence(t *testing.T) {
	for _, destination := range []string{"room-a", "room-b"} {
		t.Run(destination, func(t *testing.T) {
			h := newLocalPresenceHarness(t)
			h.createRoom(t, "room-a")
			h.joinRoom(t, "room-a", "client")
			failure := errors.New("socket send failed")
			_, err := h.join.Execute(context.Background(), JoinRoomCommand{
				RoomID: destination, SenderID: "client", Bus: &localPresenceBus{roomID: destination, sendErr: failure},
			})
			if !errors.Is(err, failure) {
				t.Fatalf("got %v, want send failure", err)
			}
			want := float64(0)
			if destination != "room-a" {
				want = 1
				if !domain.HasBusInRoom(h.hub, "client", "room-a") {
					t.Fatal("original room's bus was not restored")
				}
			}
			if h.activeUsers() != want || h.activeRooms() != want {
				t.Fatalf("active users/rooms = %v/%v, want %v/%v", h.activeUsers(), h.activeRooms(), want, want)
			}
			h.leaveRoom(t, "room-a", "client")
			if h.activeUsers() != 0 || h.activeRooms() != 0 {
				t.Fatal("cleanup did not settle at zero")
			}
		})
	}
}
