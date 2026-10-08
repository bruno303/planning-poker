package bus

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"planning-poker/internal/application/planningpoker/metric"
	"planning-poker/internal/application/planningpoker/usecase"
	"planning-poker/internal/domain"
	"planning-poker/internal/infra/boundaries/hub/inmemory"
	infralock "planning-poker/internal/infra/lock"
)

// stubBus is a minimal domain.Bus used as the replacement socket in the
// reconnect regression test.
type stubBus struct {
	roomID string
}

func (b *stubBus) RoomID() string                  { return b.roomID }
func (b *stubBus) Detach()                         {}
func (b *stubBus) Close() error                    { return nil }
func (b *stubBus) Send(context.Context, any) error { return nil }
func (b *stubBus) Listen(context.Context)          {}

// gatedHub blocks the first GetBus for clientID until release is closed, so the
// reconnect join is held while it holds the presence guard.
type gatedHub struct {
	*inmemory.InMemoryHub
	clientID string
	entered  chan struct{}
	release  chan struct{}
	blocked  atomic.Bool
}

func (h *gatedHub) GetBus(clientID string) (domain.Bus, bool) {
	if clientID == h.clientID && h.blocked.CompareAndSwap(false, true) {
		close(h.entered)
		<-h.release
	}
	return h.InMemoryHub.GetBus(clientID)
}

// TestWebsocketBus_ReconnectDoesNotDeadlockOtherRooms reproduces the reconnect
// deadlock: a reconnect join holds the presence guard while the old socket's
// pending leave waits for it, and the reconnect itself waits on the old
// socket's close. Before the fix this hung the reconnect and every other room.
func TestWebsocketBus_ReconnectDoesNotDeadlockOtherRooms(t *testing.T) {
	const (
		roomID      = "room-shared"
		otherRoomID = "room-other"
		senderID    = "reconnect-client"
	)
	ctx := context.Background()

	serverConn, clientConn := websocketPair(t)
	defer clientConn.Close()

	baseHub := inmemory.NewHub()
	if _, err := baseHub.NewRoomWithID(ctx, roomID); err != nil {
		t.Fatalf("create room: %v", err)
	}
	room, err := baseHub.LoadRoom(ctx, roomID)
	if err != nil {
		t.Fatalf("load room: %v", err)
	}
	room.NewClient(senderID)

	lockManager := infralock.NewInMemoryLockManager()
	pokerMetric := metric.NewPlanningPokerMetric()

	hub := &gatedHub{
		InMemoryHub: baseHub,
		clientID:    senderID,
		entered:     make(chan struct{}),
		release:     make(chan struct{}),
	}
	leave := usecase.NewLeaveRoomUseCase(hub, lockManager, pokerMetric)
	join := usecase.NewJoinRoomUseCase(hub, lockManager, pokerMetric)

	oldBus := NewWebsocketBus(senderID, roomID, serverConn, hub, usecase.UseCasesFacade{LeaveRoom: leave}, WebSocketConfig{})
	if err := hub.AddBus(ctx, senderID, oldBus); err != nil {
		t.Fatalf("add old bus: %v", err)
	}

	newBus := &stubBus{roomID: roomID}
	joinResult := make(chan error, 1)
	go func() {
		_, err := join.Execute(ctx, usecase.JoinRoomCommand{
			RoomID:   roomID,
			SenderID: senderID,
			Bus:      newBus,
		})
		joinResult <- err
	}()

	select {
	case <-hub.entered:
	case <-time.After(5 * time.Second):
		t.Fatal("reconnect join never reached the presence-guarded section")
	}

	closeResult := make(chan error, 1)
	go func() {
		closeResult <- oldBus.Close()
	}()

	otherResult := make(chan error, 1)
	go func() {
		_, err := join.Execute(ctx, usecase.JoinRoomCommand{
			RoomID:   otherRoomID,
			SenderID: "other-client",
			Bus:      &stubBus{roomID: otherRoomID},
		})
		otherResult <- err
	}()

	// Let the old socket's leave reach the presence guard before releasing the
	// reconnect join.
	time.Sleep(200 * time.Millisecond)
	close(hub.release)

	select {
	case err := <-joinResult:
		if err != nil {
			t.Fatalf("reconnect join failed: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("reconnect join did not complete: deadlock")
	}

	select {
	case err := <-otherResult:
		if err != nil {
			t.Fatalf("unrelated room join failed: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("unrelated room join did not complete while the old socket was closing")
	}

	select {
	case err := <-closeResult:
		if err != nil {
			t.Fatalf("old bus close failed: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("old socket close did not complete")
	}

	bus, ok := hub.GetBus(senderID)
	if !ok {
		t.Fatal("client has no local bus after reconnect")
	}
	if bus != domain.Bus(newBus) {
		t.Fatal("superseded socket removed the replacement bus")
	}
	if count := hub.GetClientsOfRoom(roomID); count != 1 {
		t.Fatalf("room %s clients = %d, want 1", roomID, count)
	}
}
