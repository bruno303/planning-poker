package redishub_test

import (
	"context"
	"os"
	redishub "planning-poker/internal/infra/boundaries/bus/redis"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

func setupRedisClient() *redis.Client {
	addr := os.Getenv("REDIS_HOST")
	port := os.Getenv("REDIS_PORT")
	if port == "" {
		port = "6379"
	}
	if addr == "" {
		addr = "localhost"
	}
	return redis.NewClient(&redis.Options{
		Addr: addr + ":" + port,
		DB:   1,
	})
}

func TestIntegration_RoomLifecycle(t *testing.T) {
	client := setupRedisClient()
	client.FlushDB(context.Background())

	hub, err := redishub.NewRedisHub(context.Background(), client)
	assert.NoError(t, err)

	room := hub.NewRoom(context.Background(), "owner1")
	assert.NotNil(t, room)
	assert.NotEmpty(t, room.ID)

	// Room should be persisted in Redis
	persisted, ok := hub.GetRoom(context.Background(), room.ID)
	assert.True(t, ok)
	assert.Equal(t, room.ID, persisted.ID)

	// Add client and verify mapping
	client1 := room.NewClient("client1")
	room.Clients.Add(client1)
	client1.WithRoom(room)
	hub.AddClient(client1)
	found, ok := hub.FindClientByID("client1")
	assert.True(t, ok)
	assert.Equal(t, client1.ID, found.ID)

	// Remove client and verify cleanup
	hub.RemoveClient(context.Background(), "client1", room.ID)
	_, ok = hub.FindClientByID("client1")
	assert.False(t, ok)

	// Remove room and verify deletion
	hub.RemoveRoom(room.ID)
	_, ok = hub.GetRoom(context.Background(), room.ID)
	assert.False(t, ok)
}

func TestIntegration_RoomTTLExpiry(t *testing.T) {
	client := setupRedisClient()
	client.FlushDB(context.Background())

	hub, err := redishub.NewRedisHub(context.Background(), client)
	assert.NoError(t, err)

	room := hub.NewRoom(context.Background(), "owner2")
	assert.NotNil(t, room)

	// Set short TTL for test
	key := "planning-poker:room:" + room.ID
	client.Expire(context.Background(), key, 2*time.Second)

	time.Sleep(3 * time.Second)
	_, ok := hub.GetRoom(context.Background(), room.ID)
	assert.False(t, ok)
}

func TestIntegration_GetBus(t *testing.T) {
	client := setupRedisClient()
	client.FlushDB(context.Background())

	hub, err := redishub.NewRedisHub(context.Background(), client)
	assert.NoError(t, err)

	room := hub.NewRoom(context.Background(), "owner-getbus")
	assert.NotNil(t, room)

	client1 := room.NewClient("client-getbus-1")
	room.Clients.Add(client1)
	client1.WithRoom(room)
	hub.AddClient(client1)

	bus := &mockBus{roomID: room.ID}
	hub.AddBus(context.Background(), client1.ID, bus)

	foundBus, ok := hub.GetBus(client1.ID)
	assert.True(t, ok)
	assert.Equal(t, bus, foundBus)

	_, ok = hub.GetBus("nonexistent-client")
	assert.False(t, ok)
}

func TestIntegration_PubSubInvalidMessage(t *testing.T) {
	client := setupRedisClient()
	client.FlushDB(context.Background())

	hub, err := redishub.NewRedisHub(context.Background(), client)
	assert.NoError(t, err)

	room := hub.NewRoom(context.Background(), "owner-pubsub")
	assert.NotNil(t, room)

	client1 := room.NewClient("client-pubsub-1")
	room.Clients.Add(client1)
	client1.WithRoom(room)
	hub.AddClient(client1)

	receivedMessages := make(chan any, 10)
	bus := &mockBusWithReceive{roomID: room.ID, received: receivedMessages}
	hub.AddBus(context.Background(), client1.ID, bus)

	// Give listener time to start
	time.Sleep(100 * time.Millisecond)

	// Publish an invalid JSON message directly to Redis to trigger unmarshal error
	channel := "planning-poker:updates:" + room.ID
	err = client.Publish(context.Background(), channel, "not-valid-json").Err()
	assert.NoError(t, err)

	// Wait a bit to ensure the invalid message was processed (but not forwarded)
	time.Sleep(200 * time.Millisecond)

	// No messages should have been received due to unmarshal error
	select {
	case <-receivedMessages:
		t.Fatal("should not have received a message due to unmarshal error")
	default:
		// Expected: no messages received
	}

	// Now publish a valid message to verify the listener is still working
	validMsg := `{"roomId":"` + room.ID + `","payload":{"test":"data"}}`
	err = client.Publish(context.Background(), channel, validMsg).Err()
	assert.NoError(t, err)

	// Should receive the valid message
	select {
	case msg := <-receivedMessages:
		assert.NotNil(t, msg)
	case <-time.After(2 * time.Second):
		t.Fatal("did not receive valid message after unmarshal error")
	}
}

func TestIntegration_PubSubUnsubscribe(t *testing.T) {
	client := setupRedisClient()
	client.FlushDB(context.Background())

	hub, err := redishub.NewRedisHub(context.Background(), client)
	assert.NoError(t, err)

	room := hub.NewRoom(context.Background(), "owner-unsub")
	assert.NotNil(t, room)

	// Add two clients to the same room
	client1 := room.NewClient("client-unsub-1")
	room.Clients.Add(client1)
	client1.WithRoom(room)
	hub.AddClient(client1)

	client2 := room.NewClient("client-unsub-2")
	room.Clients.Add(client2)
	client2.WithRoom(room)
	hub.AddClient(client2)

	bus1 := &mockBus{roomID: room.ID}
	bus2 := &mockBus{roomID: room.ID}
	hub.AddBus(context.Background(), client1.ID, bus1)
	hub.AddBus(context.Background(), client2.ID, bus2)

	// Give listener time to start
	time.Sleep(100 * time.Millisecond)

	// Verify pub/sub is active (RoomClientCounts should show 2 clients)
	assert.Equal(t, 2, hub.GetClientsOfRoom(room.ID))

	// Remove first bus - should still have subscription
	hub.RemoveBus(context.Background(), client1.ID)
	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, 1, hub.GetClientsOfRoom(room.ID))

	// Remove second bus - should unsubscribe and trigger channel close (!ok path)
	hub.RemoveBus(context.Background(), client2.ID)
	time.Sleep(200 * time.Millisecond)

	// Verify the room client count is removed
	exists := hub.GetClientsOfRoom(room.ID) > 0
	assert.False(t, exists, "room client count should be removed after all clients disconnect")
}

func TestIntegration_BroadcastToRoom(t *testing.T) {
	client := setupRedisClient()
	client.FlushDB(context.Background())

	hub, err := redishub.NewRedisHub(context.Background(), client)
	assert.NoError(t, err)

	room := hub.NewRoom(context.Background(), "owner-broadcast")
	assert.NotNil(t, room)

	client1 := room.NewClient("client-broadcast-1")
	room.Clients.Add(client1)
	client1.WithRoom(room)
	hub.AddClient(client1)

	receivedMessages := make(chan any, 10)
	bus := &mockBusWithReceive{roomID: room.ID, received: receivedMessages}
	hub.AddBus(context.Background(), client1.ID, bus)

	// Give listener time to start
	time.Sleep(100 * time.Millisecond)

	// Broadcast a message to the room
	testMessage := map[string]string{"type": "test", "data": "hello"}
	err = hub.BroadcastToRoom(context.Background(), room.ID, testMessage)
	assert.NoError(t, err)

	// Should receive the message
	select {
	case msg := <-receivedMessages:
		assert.NotNil(t, msg)
	case <-time.After(2 * time.Second):
		t.Fatal("did not receive broadcast message")
	}
}

func TestIntegration_GetRooms(t *testing.T) {
	client := setupRedisClient()
	client.FlushDB(context.Background())

	hub, err := redishub.NewRedisHub(context.Background(), client)
	assert.NoError(t, err)

	// Initially empty
	rooms := hub.GetRooms()
	assert.Empty(t, rooms)

	// Create multiple rooms
	room1 := hub.NewRoom(context.Background(), "owner-rooms-1")
	room2 := hub.NewRoom(context.Background(), "owner-rooms-2")
	room3 := hub.NewRoom(context.Background(), "owner-rooms-3")

	// Get all rooms
	rooms = hub.GetRooms()
	assert.Len(t, rooms, 3)

	// Verify all rooms are present
	roomIDs := make(map[string]bool)
	for _, r := range rooms {
		roomIDs[r.ID] = true
	}
	assert.True(t, roomIDs[room1.ID])
	assert.True(t, roomIDs[room2.ID])
	assert.True(t, roomIDs[room3.ID])
}

func TestIntegration_SaveRoom(t *testing.T) {
	client := setupRedisClient()
	client.FlushDB(context.Background())

	hub, err := redishub.NewRedisHub(context.Background(), client)
	assert.NoError(t, err)

	room := hub.NewRoom(context.Background(), "owner-save")
	assert.NotNil(t, room)

	// Modify room state
	room.CurrentStory = "Updated Story"
	room.Reveal = true
	result := float32(5.5)
	room.Result = &result

	// Save the room
	err = hub.SaveRoom(context.Background(), room)
	assert.NoError(t, err)

	// Retrieve and verify the changes were persisted
	retrieved, ok := hub.GetRoom(context.Background(), room.ID)
	assert.True(t, ok)
	assert.Equal(t, "Updated Story", retrieved.CurrentStory)
	assert.True(t, retrieved.Reveal)
	assert.NotNil(t, retrieved.Result)
	assert.Equal(t, float32(5.5), *retrieved.Result)
}

func TestIntegration_Close(t *testing.T) {
	client := setupRedisClient()
	client.FlushDB(context.Background())

	hub, err := redishub.NewRedisHub(context.Background(), client)
	assert.NoError(t, err)

	room := hub.NewRoom(context.Background(), "owner-close")
	client1 := room.NewClient("client-close-1")
	room.Clients.Add(client1)
	client1.WithRoom(room)
	hub.AddClient(client1)

	bus := &mockBus{roomID: room.ID}
	hub.AddBus(context.Background(), client1.ID, bus)

	// Give listener time to start
	time.Sleep(100 * time.Millisecond)

	// Close the hub
	err = hub.Close()
	assert.NoError(t, err)

	// Wait for any goroutines to finish
	time.Sleep(200 * time.Millisecond)
}

// mockBus implements domain.Bus minimally for test
type mockBus struct {
	roomID string
}

func (m *mockBus) RoomID() string                          { return m.roomID }
func (m *mockBus) Send(ctx context.Context, msg any) error { return nil }
func (m *mockBus) Close() error                            { return nil }
func (m *mockBus) Listen(ctx context.Context)              {}

// mockBusWithReceive implements domain.Bus and captures received messages
type mockBusWithReceive struct {
	roomID   string
	received chan any
}

func (m *mockBusWithReceive) RoomID() string { return m.roomID }
func (m *mockBusWithReceive) Send(ctx context.Context, msg any) error {
	m.received <- msg
	return nil
}
func (m *mockBusWithReceive) Close() error               { return nil }
func (m *mockBusWithReceive) Listen(ctx context.Context) {}
