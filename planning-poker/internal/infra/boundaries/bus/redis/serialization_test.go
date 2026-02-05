package redis

import (
	"planning-poker/internal/domain/entity"
	"planning-poker/internal/infra/boundaries/bus/clientcollection"
	"testing"

	"github.com/samber/lo"
)

func TestSerializeDeserializeRoom(t *testing.T) {
	// Create a room with some clients
	originalRoom := entity.NewRoom(clientcollection.New())
	originalRoom.ID = "test-room-123"
	originalRoom.CurrentStory = "User Story #42"
	originalRoom.Reveal = false

	// Add some clients
	client1 := originalRoom.NewClient("client-1")
	client1.Name = "Alice"
	client1.IsOwner = true
	client1.IsSpectator = false

	client2 := originalRoom.NewClient("client-2")
	client2.Name = "Bob"
	client2.IsOwner = false
	client2.IsSpectator = false
	vote := "5"
	client2.CurrentVote = &vote
	client2.HasVoted = true

	// Serialize the room
	data, err := SerializeRoom(originalRoom)
	if err != nil {
		t.Fatalf("Failed to serialize room: %v", err)
	}

	// Deserialize the room
	deserializedRoom, err := DeserializeRoom(data, clientcollection.New())
	if err != nil {
		t.Fatalf("Failed to deserialize room: %v", err)
	}

	// Verify room properties
	if deserializedRoom.ID != originalRoom.ID {
		t.Errorf("Expected room ID %s, got %s", originalRoom.ID, deserializedRoom.ID)
	}
	if deserializedRoom.CurrentStory != originalRoom.CurrentStory {
		t.Errorf("Expected story %s, got %s", originalRoom.CurrentStory, deserializedRoom.CurrentStory)
	}
	if deserializedRoom.Reveal != originalRoom.Reveal {
		t.Errorf("Expected reveal %v, got %v", originalRoom.Reveal, deserializedRoom.Reveal)
	}

	// Verify client count
	if deserializedRoom.Clients.Count() != originalRoom.Clients.Count() {
		t.Errorf("Expected %d clients, got %d", originalRoom.Clients.Count(), deserializedRoom.Clients.Count())
	}

	// Verify client properties
	deserializedClient1, ok := deserializedRoom.Clients.Filter(func(c *entity.Client) bool {
		return c.ID == "client-1"
	}).First()
	if !ok {
		t.Fatal("Client 1 not found in deserialized room")
	}
	if deserializedClient1.Name != "Alice" {
		t.Errorf("Expected client name Alice, got %s", deserializedClient1.Name)
	}
	if !deserializedClient1.IsOwner {
		t.Error("Expected client 1 to be owner")
	}

	deserializedClient2, ok := deserializedRoom.Clients.Filter(func(c *entity.Client) bool {
		return c.ID == "client-2"
	}).First()
	if !ok {
		t.Fatal("Client 2 not found in deserialized room")
	}
	if deserializedClient2.Name != "Bob" {
		t.Errorf("Expected client name Bob, got %s", deserializedClient2.Name)
	}
	if !deserializedClient2.HasVoted {
		t.Error("Expected client 2 to have voted")
	}
	if deserializedClient2.CurrentVote == nil || *deserializedClient2.CurrentVote != "5" {
		t.Errorf("Expected client 2 vote to be 5, got %v", lo.FromPtr(deserializedClient2.CurrentVote))
	}
}
