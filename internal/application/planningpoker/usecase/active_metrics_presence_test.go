package usecase

import (
	"context"
	"fmt"
	"testing"

	"planning-poker/internal/application/planningpoker/metric"
	"planning-poker/internal/domain"
	"planning-poker/internal/infra/boundaries/hub/inmemory"
	infralock "planning-poker/internal/infra/lock"

	"go.uber.org/mock/gomock"
)

// localPresenceBus is a minimal domain.Bus used to drive the real hubs in the
// tests below. The hub keeps one bus per client, so the room the bus was built
// for is what defines local presence.
type localPresenceBus struct {
	roomID string
}

func (b *localPresenceBus) RoomID() string                  { return b.roomID }
func (b *localPresenceBus) Detach()                         {}
func (b *localPresenceBus) Close() error                    { return nil }
func (b *localPresenceBus) Listen(context.Context)          {}
func (b *localPresenceBus) Send(context.Context, any) error { return nil }

func (b *localPresenceBus) String() string {
	return fmt.Sprintf("localPresenceBus(%s)", b.roomID)
}

// localPresenceHarness wires the real in-memory hub and lock manager to the real
// join/leave use cases and records every emitted metric. It lets tests assert
// the invariant the mocked hub cannot express: the process-local active metrics
// must match real local presence after any sequence of joins and leaves.
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

	return &localPresenceHarness{
		hub:      hub,
		join:     NewJoinRoomUseCase(hub, lockManager, pokerMetric),
		leave:    NewLeaveRoomUseCase(hub, lockManager, pokerMetric),
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

// TestActiveMetricsMatchLocalPresence is the regression test for negative active
// metrics: a client that switches rooms on the same instance while the previous
// socket is still open must not drift the counters.
func TestActiveMetricsMatchLocalPresence(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		scenario func(t *testing.T, h *localPresenceHarness)
		// wantUsers/wantRooms are the net values the process-local counters must
		// hold once the scenario finishes.
		wantUsers float64
		wantRooms float64
	}{
		{
			name: "first client joining an existing room counts one user and one room",
			scenario: func(t *testing.T, h *localPresenceHarness) {
				h.createRoom(t, "room-a")
				h.joinRoom(t, "room-a", "client-1")
			},
			wantUsers: 1,
			wantRooms: 1,
		},
		{
			name: "second local client in the same room adds a user but not a room",
			scenario: func(t *testing.T, h *localPresenceHarness) {
				h.createRoom(t, "room-a")
				h.joinRoom(t, "room-a", "client-1")
				h.joinRoom(t, "room-a", "client-2")
			},
			wantUsers: 2,
			wantRooms: 1,
		},
		{
			name: "two local rooms count two rooms",
			scenario: func(t *testing.T, h *localPresenceHarness) {
				h.createRoom(t, "room-a")
				h.createRoom(t, "room-b")
				h.joinRoom(t, "room-a", "client-1")
				h.joinRoom(t, "room-b", "client-2")
			},
			wantUsers: 2,
			wantRooms: 2,
		},
		{
			name: "same room reconnect after the old socket closed stays balanced",
			scenario: func(t *testing.T, h *localPresenceHarness) {
				h.createRoom(t, "room-a")
				h.joinRoom(t, "room-a", "client-1")
				h.leaveRoom(t, "room-a", "client-1")
				h.joinRoom(t, "room-a", "client-1")
			},
			wantUsers: 1,
			wantRooms: 1,
		},
		{
			name: "cross-room switch where the old leave arrives while the client is in the new room",
			scenario: func(t *testing.T, h *localPresenceHarness) {
				h.createRoom(t, "room-a")
				h.joinRoom(t, "room-a", "client-1")
				// The client opens room-b while room-a's socket is still open. The
				// hub now holds the bus for room-b; room-a is no longer locally
				// present on this instance.
				h.joinRoom(t, "room-b", "client-1")
				// The old room-a socket close is processed after the room-b join.
				h.leaveRoom(t, "room-a", "client-1")
			},
			wantUsers: 1,
			wantRooms: 1,
		},
		{
			name: "cross-room switch followed by disconnect settles at zero",
			scenario: func(t *testing.T, h *localPresenceHarness) {
				h.createRoom(t, "room-a")
				h.joinRoom(t, "room-a", "client-1")
				h.joinRoom(t, "room-b", "client-1")
				// The late room-a leave is a no-op for the counters; the client is
				// still connected here through room-b.
				h.leaveRoom(t, "room-a", "client-1")
				h.leaveRoom(t, "room-b", "client-1")
			},
			wantUsers: 0,
			wantRooms: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := newLocalPresenceHarness(t)
			tc.scenario(t, h)

			if got := h.activeUsers(); got != tc.wantUsers {
				t.Errorf("active users = %v, want %v", got, tc.wantUsers)
			}
			if got := h.activeRooms(); got != tc.wantRooms {
				t.Errorf("active rooms = %v, want %v", got, tc.wantRooms)
			}
		})
	}
}

// TestCrossRoomSwitchClearsLocalPresence pins the hub-level invariant the metric
// assertions above depend on: once the client's bus belongs to the new room, the
// previous room no longer reports the client as locally present.
func TestCrossRoomSwitchClearsLocalPresence(t *testing.T) {
	t.Parallel()

	h := newLocalPresenceHarness(t)
	h.createRoom(t, "room-a")
	h.createRoom(t, "room-b")

	h.joinRoom(t, "room-a", "client-1")
	if !h.hub.HasBusInRoom("client-1", "room-a") {
		t.Fatal("expected the client to be locally present in room-a after joining")
	}

	h.joinRoom(t, "room-b", "client-1")
	if h.hub.HasBusInRoom("client-1", "room-a") {
		t.Error("expected room-a to stop reporting local presence after the client moved to room-b")
	}
	if !h.hub.HasBusInRoom("client-1", "room-b") {
		t.Error("expected the client to be locally present in room-b after joining")
	}
	if got := h.hub.GetClientsOfRoom("room-a"); got != 0 {
		t.Errorf("room-a local clients = %d, want 0", got)
	}
	if got := h.hub.GetClientsOfRoom("room-b"); got != 1 {
		t.Errorf("room-b local clients = %d, want 1", got)
	}
}

// TestLeaveRoomIgnoresBusFromAnotherRoom covers the leave path directly: a leave
// for a room the client no longer holds a bus for must not emit metrics.
func TestLeaveRoomIgnoresBusFromAnotherRoom(t *testing.T) {
	t.Parallel()

	h := newLocalPresenceHarness(t)
	h.createRoom(t, "room-a")
	h.createRoom(t, "room-b")
	h.joinRoom(t, "room-a", "client-1")
	h.joinRoom(t, "room-b", "client-1")

	h.leaveRoom(t, "room-a", "client-1")

	if got := h.activeUsers(); got != 1 {
		t.Errorf("active users = %v, want 1 while the client is still connected in room-b", got)
	}
	if got := h.activeRooms(); got != 1 {
		t.Errorf("active rooms = %v, want 1 while the client is still connected in room-b", got)
	}
}

var _ domain.Bus = (*localPresenceBus)(nil)
