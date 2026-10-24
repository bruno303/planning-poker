package redis

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/samber/lo"

	"planning-poker/internal/domain/entity"
	"planning-poker/internal/infra/boundaries/hub/clientcollection"
)

func TestSerializeDeserializeRoom(t *testing.T) {
	// Create a room with some clients
	startedAt := time.Date(2026, time.September, 3, 12, 0, 0, 0, time.FixedZone("BRT", -3*60*60))
	originalRoom := entity.NewRoomFromSnapshot(entity.RoomSnapshot{
		ID:           "test-room-123",
		StartedAt:    startedAt,
		DeckType:     entity.DeckTypeTShirt,
		Version:      7,
		CurrentStory: "User Story #42",
		Revealed:     true,
		Outcome: &entity.RoundOutcome{
			Consensus:        "Medium",
			LowestVote:       lo.ToPtr("S"),
			HighestVote:      lo.ToPtr("L"),
			VoteRange:        lo.ToPtr(5),
			VoteSpread:       lo.ToPtr(2),
			SpecialVoteCount: 1,
		},
		BacklogMode: true,
		Stories: []entity.Story{{
			ID:                 "story-1",
			Name:               "Backlog story",
			Result:             lo.ToPtr(float32(8)),
			MostAppearingVotes: []string{"8"},
			Voted:              true,
		}},
	}, clientcollection.New())

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
	client2.CurrentVote = entity.VotePtr(vote)
	client2.HasVoted = true
	client2.VotedAt = lo.ToPtr(time.Date(2026, time.September, 3, 12, 1, 0, 0, time.FixedZone("BRT", -3*60*60)))

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

	assertSerializedRoomProperties(t, originalRoom, deserializedRoom)
	assertSerializedStories(t, deserializedRoom)
	assertSerializedClients(t, originalRoom, deserializedRoom)
}

func assertSerializedRoomProperties(t *testing.T, originalRoom, deserializedRoom *entity.Room) {
	t.Helper()

	if deserializedRoom.ID() != originalRoom.ID() {
		t.Errorf("Expected room ID %s, got %s", originalRoom.ID(), deserializedRoom.ID())
	}
	if deserializedRoom.DeckType != originalRoom.DeckType {
		t.Errorf("Expected deck type %q, got %q", originalRoom.DeckType, deserializedRoom.DeckType)
	}
	if original, deserialized := originalRoom.Snapshot(), deserializedRoom.Snapshot(); original.CurrentStory != deserialized.CurrentStory {
		t.Errorf("Expected story %s, got %s", original.CurrentStory, deserialized.CurrentStory)
	}
	if !deserializedRoom.StartedAt().Equal(originalRoom.StartedAt()) || deserializedRoom.StartedAt().Location() != time.UTC {
		t.Errorf("Expected room start time %v in UTC, got %v", originalRoom.StartedAt(), deserializedRoom.StartedAt())
	}
	if deserializedRoom.Revealed() != originalRoom.Revealed() {
		t.Errorf("Expected reveal %v, got %v", originalRoom.Revealed(), deserializedRoom.Revealed())
	}
	if !reflect.DeepEqual(deserializedRoom.Outcome(), originalRoom.Outcome()) {
		t.Errorf("Expected outcome %+v, got %+v", originalRoom.Outcome(), deserializedRoom.Outcome())
	}
	if deserializedRoom.SpecialVoteCount() != originalRoom.SpecialVoteCount() {
		t.Errorf("Expected special vote count %d, got %d", originalRoom.SpecialVoteCount(), deserializedRoom.SpecialVoteCount())
	}
	if deserializedRoom.RoomVersion != originalRoom.RoomVersion {
		t.Errorf("Expected room version %d, got %d", originalRoom.RoomVersion, deserializedRoom.RoomVersion)
	}
}

func assertSerializedStories(t *testing.T, deserializedRoom *entity.Room) {
	t.Helper()

	stories := deserializedRoom.Stories()
	if len(stories) != 1 || stories[0].ID != "story-1" {
		t.Fatalf("story ID was not preserved: %+v", stories)
	}
	if stories[0].Result == nil || *stories[0].Result != 8 || !stories[0].Voted {
		t.Fatalf("story estimate was not preserved: %+v", stories[0])
	}
}

func assertSerializedClients(t *testing.T, originalRoom, deserializedRoom *entity.Room) {
	t.Helper()
	originalClient2, ok := originalRoom.Clients().Filter(func(c *entity.Client) bool {
		return c.ID == "client-2"
	}).First()
	if !ok {
		t.Fatal("Original client 2 not found")
	}

	if deserializedRoom.Clients().Count() != originalRoom.Clients().Count() {
		t.Errorf("Expected %d clients, got %d", originalRoom.Clients().Count(), deserializedRoom.Clients().Count())
	}

	deserializedClient1, ok := deserializedRoom.Clients().Filter(func(c *entity.Client) bool {
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

	deserializedClient2, ok := deserializedRoom.Clients().Filter(func(c *entity.Client) bool {
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
	if deserializedClient2.CurrentVote == nil || deserializedClient2.CurrentVote.Label() != "5" {
		t.Errorf("Expected client 2 vote to be 5, got %v", deserializedClient2.CurrentVote.Label())
	}
	if deserializedClient2.VotedAt == nil || !deserializedClient2.VotedAt.Equal(*originalClient2.VotedAt) || deserializedClient2.VotedAt.Location() != time.UTC {
		t.Errorf("Expected client 2 vote time %v in UTC, got %v", originalClient2.VotedAt, deserializedClient2.VotedAt)
	}
}

func TestDeserializeRoomRejectsStoryWithoutID(t *testing.T) {
	data := []byte(`{"id":"room-legacy","backlogMode":true,"stories":[{"name":"Legacy story","mostAppearingVotes":[],"voted":false}]}`)

	if _, err := DeserializeRoom(data, clientcollection.New()); err == nil {
		t.Fatal("DeserializeRoom accepted a story without an ID")
	}
}

func TestDeserializeRoomSupportsLegacyRecordsWithoutTimestamps(t *testing.T) {
	data := []byte(`{"id":"room-legacy","clients":[{"id":"client-1","name":"Alice","hasVoted":false,"isSpectator":false,"isOwner":true}],"backlogMode":true,"stories":[]}`)

	room, err := DeserializeRoom(data, clientcollection.New())
	if err != nil {
		t.Fatalf("DeserializeRoom returned error for legacy record: %v", err)
	}
	if !room.StartedAt().IsZero() {
		t.Fatalf("legacy room start time = %v, want zero", room.StartedAt())
	}
	client, ok := room.Clients().First()
	if !ok {
		t.Fatal("legacy client was not restored")
	}
	if client.VotedAt != nil {
		t.Fatalf("legacy client vote time = %v, want nil", client.VotedAt)
	}

	serialized, err := SerializeRoom(room)
	if err != nil {
		t.Fatalf("SerializeRoom returned error: %v", err)
	}
	var wire map[string]any
	if err := json.Unmarshal(serialized, &wire); err != nil {
		t.Fatalf("serialized legacy room is invalid JSON: %v", err)
	}
	if _, ok := wire["startedAt"]; ok {
		t.Fatal("legacy room unexpectedly gained startedAt")
	}
	clients, ok := wire["clients"].([]any)
	if !ok || len(clients) != 1 {
		t.Fatalf("serialized legacy clients = %v", wire["clients"])
	}
	clientWire, ok := clients[0].(map[string]any)
	if !ok {
		t.Fatalf("serialized legacy client = %v", clients[0])
	}
	if _, ok := clientWire["votedAt"]; ok {
		t.Fatal("legacy client unexpectedly gained votedAt")
	}
}
