package inmemory

import (
	"context"
	"errors"
	"planning-poker/internal/domain"
	"planning-poker/internal/domain/entity"
	"sync"
	"testing"

	"go.uber.org/mock/gomock"
)

func TestNewRoom(t *testing.T) {
	ctx := context.Background()
	hub := NewHub()
	room, err := hub.NewRoom(ctx)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if room == nil {
		t.Fatal("expected room to be non-nil")
	}
	if len(hub.Rooms) != 1 {
		t.Errorf("expected hub.Rooms to have 1 room, got %d", len(hub.Rooms))
	}
	if hub.Rooms[room.ID] != room {
		t.Errorf("expected hub.Rooms[0] to be the created room")
	}
	if room.Clients == nil {
		t.Error("expected room.Clients to be initialized")
	}
}

func TestNewRoomWithID(t *testing.T) {
	ctx := context.Background()
	hub := NewHub()

	room, err := hub.NewRoomWithID(ctx, "room-123")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if room == nil {
		t.Fatal("expected room to be non-nil")
	}
	if room.ID != "room-123" {
		t.Fatalf("expected room ID room-123, got %s", room.ID)
	}
	if hub.Rooms[room.ID] != room {
		t.Fatal("expected hub to store room under explicit ID")
	}
}

func TestLoadRoom(t *testing.T) {
	ctx := context.Background()
	hub := NewHub()
	room, err := hub.NewRoom(ctx)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	got, err := hub.LoadRoom(ctx, room.ID)
	if err != nil {
		t.Fatalf("expected to find room but got error: %v", err)
	}
	if got != room {
		t.Errorf("expected to get the created room, got different room")
	}

	_, err = hub.LoadRoom(ctx, "non-existent-id")
	if !errors.Is(err, domain.ErrRoomNotFound) {
		t.Fatalf("expected ErrRoomNotFound for non-existent room ID, got %v", err)
	}
}

func TestRemoveRoom(t *testing.T) {
	ctx := context.Background()
	hub := NewHub()
	room1, err := hub.NewRoom(ctx)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	room2, err := hub.NewRoom(ctx)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(hub.Rooms) != 2 {
		t.Fatalf("expected 2 rooms, got %d", len(hub.Rooms))
	}

	hub.RemoveRoom(room1.ID)
	if len(hub.Rooms) != 1 {
		t.Errorf("expected 1 room after removal, got %d", len(hub.Rooms))
	}
	if hub.Rooms[room2.ID] != room2 {
		t.Errorf("expected remaining room to be room2")
	}

	// Remove non-existent room should not panic or change state
	hub.RemoveRoom("non-existent-id")
	if len(hub.Rooms) != 1 {
		t.Errorf("expected 1 room after removing non-existent, got %d", len(hub.Rooms))
	}
}

func TestAddAndFindClient(t *testing.T) {
	hub := NewHub()
	client := &entity.Client{ID: "client1", Name: "Alice"}

	hub.AddClient(client)

	got, ok := hub.FindClientByID(client.ID)
	if !ok {
		t.Fatalf("expected to find client but got not found")
	}
	if got != client {
		t.Errorf("expected to get the added client, got different client")
	}

	_, ok = hub.FindClientByID("non-existent-id")
	if ok {
		t.Error("expected \"false\" for non-existent client ID, got \"true\"")
	}
}

func TestAddAndGetBus(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	hub := NewHub()
	clientID := "client1"
	mockBus := domain.NewMockBus(ctrl)

	hub.AddBus(ctx, clientID, mockBus)

	got, ok := hub.GetBus(clientID)
	if !ok {
		t.Fatalf("expected to find bus but got not found")
	}
	if got != mockBus {
		t.Errorf("expected to get the added bus, got different bus")
	}

	_, ok = hub.GetBus("non-existent-id")
	if ok {
		t.Error("expected \"false\" for non-existent bus, got \"true\"")
	}
}

func TestRemoveBus(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	hub := NewHub()
	clientID1 := "client1"
	clientID2 := "client2"
	mockBus1 := domain.NewMockBus(ctrl)
	mockBus2 := domain.NewMockBus(ctrl)

	hub.AddBus(ctx, clientID1, mockBus1)
	hub.AddBus(ctx, clientID2, mockBus2)

	if len(hub.Buses) != 2 {
		t.Fatalf("expected 2 buses, got %d", len(hub.Buses))
	}

	hub.RemoveBus(ctx, clientID1)
	if len(hub.Buses) != 1 {
		t.Errorf("expected 1 bus after removal, got %d", len(hub.Buses))
	}

	_, ok := hub.GetBus(clientID1)
	if ok {
		t.Error("expected bus1 to be removed")
	}

	_, ok = hub.GetBus(clientID2)
	if !ok {
		t.Error("expected bus2 to still exist")
	}

	// Remove non-existent bus should not panic
	hub.RemoveBus(ctx, "non-existent-id")
	if len(hub.Buses) != 1 {
		t.Errorf("expected 1 bus after removing non-existent, got %d", len(hub.Buses))
	}
}

func TestRemoveClient_Success(t *testing.T) {
	ctx := context.Background()
	hub := NewHub()

	room, err := hub.NewRoom(ctx)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	client := &entity.Client{ID: "client1", Name: "Alice"}
	hub.AddClient(client)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockBus := domain.NewMockBus(ctrl)
	hub.AddBus(ctx, client.ID, mockBus)

	// Add client to room
	room.Clients.Add(client)

	err = hub.RemoveClient(ctx, client.ID, room.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	_, ok := hub.FindClientByID(client.ID)
	if ok {
		t.Error("expected client to be removed from hub")
	}

	_, ok = hub.GetBus(client.ID)
	if ok {
		t.Error("expected bus to be removed")
	}
}

func TestRemoveClient_RoomNotFound(t *testing.T) {
	ctx := context.Background()
	hub := NewHub()

	client := &entity.Client{ID: "client1", Name: "Alice"}
	hub.AddClient(client)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockBus := domain.NewMockBus(ctrl)
	hub.AddBus(ctx, client.ID, mockBus)

	err := hub.RemoveClient(ctx, client.ID, "non-existent-room")
	if err != nil {
		t.Fatalf("expected no error when room not found, got %v", err)
	}

	_, ok := hub.FindClientByID(client.ID)
	if ok {
		t.Error("expected client to be removed even when room not found")
	}

	_, ok = hub.GetBus(client.ID)
	if ok {
		t.Error("expected bus to be removed even when room not found")
	}
}

func TestRemoveClient_EmptyRoomRemoval(t *testing.T) {
	ctx := context.Background()
	hub := NewHub()

	room, err := hub.NewRoom(ctx)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	client := &entity.Client{ID: "client1", Name: "Alice"}
	hub.AddClient(client)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockBus := domain.NewMockBus(ctrl)
	hub.AddBus(ctx, client.ID, mockBus)

	room.Clients.Add(client)

	if len(hub.Rooms) != 1 {
		t.Fatalf("expected 1 room, got %d", len(hub.Rooms))
	}

	err = hub.RemoveClient(ctx, client.ID, room.ID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Room should be removed because it's empty
	if len(hub.Rooms) != 0 {
		t.Errorf("expected room to be removed when empty, got %d rooms", len(hub.Rooms))
	}
}

func TestBroadcastToRoom_Success(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	hub := NewHub()
	room, err := hub.NewRoom(ctx)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	client1 := &entity.Client{ID: "client1", Name: "Alice"}
	client2 := &entity.Client{ID: "client2", Name: "Bob"}

	hub.AddClient(client1)
	hub.AddClient(client2)
	room.Clients.Add(client1)
	room.Clients.Add(client2)

	mockBus1 := domain.NewMockBus(ctrl)
	mockBus2 := domain.NewMockBus(ctrl)
	hub.AddBus(ctx, client1.ID, mockBus1)
	hub.AddBus(ctx, client2.ID, mockBus2)

	message := map[string]string{"type": "test", "data": "hello"}

	mockBus1.EXPECT().Send(gomock.Any(), message).Return(nil)
	mockBus2.EXPECT().Send(gomock.Any(), message).Return(nil)

	err = hub.BroadcastToRoom(ctx, room.ID, message)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestBroadcastToRoom_RoomNotFound(t *testing.T) {
	ctx := context.Background()
	hub := NewHub()

	message := map[string]string{"type": "test"}

	err := hub.BroadcastToRoom(ctx, "non-existent-room", message)
	if err == nil {
		t.Fatal("expected error for non-existent room, got nil")
	}
	if !errors.Is(err, domain.ErrRoomNotFound) {
		t.Errorf("expected ErrRoomNotFound, got %v", err)
	}
}

func TestBroadcastToRoom_BusNotFound(t *testing.T) {
	ctx := context.Background()
	hub := NewHub()

	room, err := hub.NewRoom(ctx)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	client := &entity.Client{ID: "client1", Name: "Alice"}
	hub.AddClient(client)
	room.Clients.Add(client)

	// No bus added for client
	message := map[string]string{"type": "test"}

	err = hub.BroadcastToRoom(ctx, room.ID, message)
	// Should not error, just log warning
	if err != nil {
		t.Fatalf("expected no error when bus not found, got %v", err)
	}
}

func TestBroadcastToRoom_SendError(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	hub := NewHub()
	room, err := hub.NewRoom(ctx)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	client := &entity.Client{ID: "client1", Name: "Alice"}
	hub.AddClient(client)
	room.Clients.Add(client)

	mockBus := domain.NewMockBus(ctrl)
	hub.AddBus(ctx, client.ID, mockBus)

	message := map[string]string{"type": "test"}
	expectedError := errors.New("send failed")

	mockBus.EXPECT().Send(gomock.Any(), message).Return(expectedError)

	err = hub.BroadcastToRoom(ctx, room.ID, message)
	if err == nil {
		t.Fatal("expected error when send fails, got nil")
	}
	if err.Error() != "failed to send message to client client1: send failed" {
		t.Errorf("unexpected error message: %v", err.Error())
	}
}

func TestGetRooms(t *testing.T) {
	ctx := context.Background()
	hub := NewHub()

	// Empty hub
	rooms := hub.GetRooms()
	if len(rooms) != 0 {
		t.Errorf("expected 0 rooms for empty hub, got %d", len(rooms))
	}

	// Add rooms
	room1, err := hub.NewRoom(ctx)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	room2, err := hub.NewRoom(ctx)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	room3, err := hub.NewRoom(ctx)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	rooms = hub.GetRooms()
	if len(rooms) != 3 {
		t.Fatalf("expected 3 rooms, got %d", len(rooms))
	}

	// Check all rooms are in the slice
	roomIDs := make(map[string]bool)
	for _, room := range rooms {
		roomIDs[room.ID] = true
	}

	if !roomIDs[room1.ID] {
		t.Error("room1 not found in GetRooms result")
	}
	if !roomIDs[room2.ID] {
		t.Error("room2 not found in GetRooms result")
	}
	if !roomIDs[room3.ID] {
		t.Error("room3 not found in GetRooms result")
	}
}

func TestInterfaceCompliance(t *testing.T) {
	var _ domain.Hub = (*InMemoryHub)(nil)
	var _ domain.AdminHub = (*InMemoryHub)(nil)
}

func TestSaveRoom(t *testing.T) {
	ctx := context.Background()
	hub := NewHub()
	room, err := hub.NewRoom(ctx)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Change some state in the room
	room.Reveal = true
	err = hub.SaveRoom(ctx, room)
	if err != nil {
		t.Fatalf("expected no error from SaveRoom, got %v", err)
	}

	// Retrieve the room and check the state
	got, err := hub.LoadRoom(ctx, room.ID)
	if err != nil {
		t.Fatalf("expected to find room after SaveRoom, got error: %v", err)
	}
	if !got.Reveal {
		t.Errorf("expected room.Reveal to be true, got false")
	}
	if got.ExpectedPersistedRoomVersion() != got.RoomVersion {
		t.Errorf("expected SaveRoom to record persisted version %d, got %d", got.RoomVersion, got.ExpectedPersistedRoomVersion())
	}
}

func TestSaveRoom_RejectsStaleVersion(t *testing.T) {
	hub := NewHub()
	room := entity.NewRoom(nil)
	room.RoomVersion = 2

	expectedVersion := uint64(1)
	err := hub.SaveRoomIfVersion(context.Background(), room, &expectedVersion)
	if !errors.Is(err, domain.ErrStaleRoomVersion) {
		t.Fatalf("expected stale room version error, got %v", err)
	}
}

func TestSaveRoomIfVersion_StaleMutationDoesNotChangeStoredRoom(t *testing.T) {
	hub := NewHub()
	room, err := hub.NewRoom(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	expectedVersion := uint64(0)
	room.RoomVersion = 1
	if err := hub.SaveRoomIfVersion(context.Background(), room, &expectedVersion); err != nil {
		t.Fatal(err)
	}

	room.Reveal = true
	room.RoomVersion = 2
	expectedVersion = 0
	if err := hub.SaveRoomIfVersion(context.Background(), room, &expectedVersion); !errors.Is(err, domain.ErrStaleRoomVersion) {
		t.Fatalf("expected stale error, got %v", err)
	}
	stored, err := hub.LoadRoom(context.Background(), room.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Reveal || stored.RoomVersion != 1 {
		t.Fatalf("stale mutation changed stored room: reveal=%v version=%d", stored.Reveal, stored.RoomVersion)
	}
}

func TestSaveRoomIfVersion_SerializesConcurrentSaves(t *testing.T) {
	hub := NewHub()
	baseline, _ := hub.NewRoomWithID(context.Background(), "room")
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for i := 0; i < 2; i++ {
		candidate := entity.NewRoomWithID(baseline.ID, nil)
		candidate.RoomVersion = 1
		expectedVersion := uint64(0)
		wg.Add(1)
		go func() {
			defer wg.Done()
			results <- hub.SaveRoomIfVersion(context.Background(), candidate, &expectedVersion)
		}()
	}
	wg.Wait()
	close(results)

	successes := 0
	for err := range results {
		if err == nil {
			successes++
		} else if !errors.Is(err, domain.ErrStaleRoomVersion) {
			t.Fatalf("unexpected error: %v", err)
		}
	}
	if successes != 1 {
		t.Fatalf("expected exactly one successful save, got %d", successes)
	}
}

func TestSaveRoomIfVersion_RejectsStaleNoOpAndTracksSuccessfulSave(t *testing.T) {
	hub := NewHub()
	room := entity.NewRoom(nil)
	room.RoomVersion = 1

	expectedVersion := uint64(0)
	if err := hub.SaveRoomIfVersion(context.Background(), room, &expectedVersion); err != nil {
		t.Fatalf("expected initial conditional save to succeed, got %v", err)
	}
	if got := room.ExpectedPersistedRoomVersion(); got != 1 {
		t.Fatalf("expected persisted version 1 after save, got %d", got)
	}

	if err := hub.SaveRoomIfVersion(context.Background(), room, &expectedVersion); !errors.Is(err, domain.ErrStaleRoomVersion) {
		t.Fatalf("expected stale no-op to be rejected, got %v", err)
	}

	expectedVersion = 1
	if err := hub.SaveRoomIfVersion(context.Background(), room, &expectedVersion); err != nil {
		t.Fatalf("expected subsequent conditional save to succeed, got %v", err)
	}
}
