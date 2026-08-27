package redis

import (
	"planning-poker/internal/domain/entity"
	"planning-poker/internal/infra/boundaries/hub/clientcollection"
	"testing"

	"github.com/google/uuid"
	"github.com/samber/lo"
)

func TestSerializeDeserializeRoom(t *testing.T) {
	// Create a room with some clients
	originalRoom := entity.NewRoom(clientcollection.New())
	originalRoom.ID = "test-room-123"
	originalRoom.CurrentStory = "User Story #42"
	originalRoom.Reveal = false
	originalRoom.Consensus = "Medium"
	originalRoom.LowestVote = lo.ToPtr(3)
	originalRoom.HighestVote = lo.ToPtr(8)
	originalRoom.VoteRange = lo.ToPtr(5)
	originalRoom.VoteSpread = lo.ToPtr(2)
	originalRoom.NonNumericVoteCount = 1
	originalRoom.BacklogVersion = 6
	originalRoom.Stories = []entity.Story{{
		ID:                 "story-1",
		Name:               "Backlog story",
		Result:             lo.ToPtr(float32(8)),
		MostAppearingVotes: []int{8},
		Voted:              true,
	}}

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
	if deserializedRoom.Consensus != originalRoom.Consensus {
		t.Errorf("Expected consensus %q, got %q", originalRoom.Consensus, deserializedRoom.Consensus)
	}
	if *deserializedRoom.LowestVote != *originalRoom.LowestVote ||
		*deserializedRoom.HighestVote != *originalRoom.HighestVote ||
		*deserializedRoom.VoteRange != *originalRoom.VoteRange ||
		*deserializedRoom.VoteSpread != *originalRoom.VoteSpread {
		t.Errorf("numeric consensus metrics were not preserved")
	}
	if deserializedRoom.NonNumericVoteCount != originalRoom.NonNumericVoteCount {
		t.Errorf("Expected non-numeric vote count %d, got %d", originalRoom.NonNumericVoteCount, deserializedRoom.NonNumericVoteCount)
	}
	if deserializedRoom.BacklogVersion != originalRoom.BacklogVersion {
		t.Errorf("Expected backlog version %d, got %d", originalRoom.BacklogVersion, deserializedRoom.BacklogVersion)
	}
	if len(deserializedRoom.Stories) != 1 || deserializedRoom.Stories[0].ID != "story-1" {
		t.Fatalf("story ID was not preserved: %+v", deserializedRoom.Stories)
	}
	if deserializedRoom.Stories[0].Result == nil || *deserializedRoom.Stories[0].Result != 8 || !deserializedRoom.Stories[0].Voted {
		t.Fatalf("story estimate was not preserved: %+v", deserializedRoom.Stories[0])
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

func TestDeserializeRoomBackfillsLegacyStoryIDs(t *testing.T) {
	data := []byte(`{"id":"room-legacy","backlogMode":true,"stories":[{"name":"Legacy story","mostAppearingVotes":[],"voted":false}]}`)

	room, err := DeserializeRoom(data, clientcollection.New())
	if err != nil {
		t.Fatalf("DeserializeRoom returned error: %v", err)
	}
	if room.BacklogVersion != 0 {
		t.Fatalf("legacy backlog version = %d, want 0", room.BacklogVersion)
	}
	if len(room.Stories) != 1 || room.Stories[0].ID == "" {
		t.Fatalf("legacy story ID was not backfilled: %+v", room.Stories)
	}
	if _, err := uuid.Parse(room.Stories[0].ID); err != nil {
		t.Fatalf("backfilled story ID is not a UUID: %v", err)
	}
}
