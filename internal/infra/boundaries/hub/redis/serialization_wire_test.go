package redis

import (
	"bytes"
	"context"
	"encoding/json"
	"slices"
	"testing"
	"time"

	"planning-poker/internal/domain/entity"
	"planning-poker/internal/infra/boundaries/hub/clientcollection"
)

// Wire contract keys for the persisted room record. They mirror the JSON tags
// in SerializedRoom, SerializedClient and SerializedStory, including which
// fields carry omitempty.
var (
	serializedRoomWireKeys = []string{
		"id",
		"startedAt",
		"clients",
		"deckType",
		"currentStory",
		"reveal",
		"result",
		"mostAppearingVotes",
		"consensus",
		"lowestVote",
		"highestVote",
		"voteRange",
		"voteSpread",
		"specialVoteCount",
		"backlogMode",
		"stories",
		"currentStoryIndex",
		"roomVersion",
	}

	votedClientWireKeys = []string{
		"id",
		"name",
		"currentVote",
		"hasVoted",
		"votedAt",
		"isSpectator",
		"isOwner",
	}

	pendingClientWireKeys = []string{
		"id",
		"name",
		"hasVoted",
		"isSpectator",
		"isOwner",
	}

	estimatedStoryWireKeys = []string{
		"id",
		"name",
		"result",
		"mostAppearingVotes",
		"voted",
	}

	pendingStoryWireKeys = []string{
		"id",
		"name",
		"mostAppearingVotes",
		"voted",
	}
)

var wireStartedAt = time.Date(2026, time.October, 1, 9, 30, 0, 0, time.UTC)

func TestSerializeRoomWireContract(t *testing.T) {
	t.Run("revealed backlog room exposes the full key set", func(t *testing.T) {
		room := newRevealedBacklogRoom(t)

		decoded := decodeSerializedRoomWire(t, room)

		assertKeySet(t, "serialized room", decoded, serializedRoomWireKeys)
		assertJSONNumber(t, decoded, "roomVersion")

		if got := decoded["id"]; got != "room-wire" {
			t.Errorf("key %q = %#v, want %q", "id", got, "room-wire")
		}
		if got := decoded["reveal"]; got != true {
			t.Errorf("key %q = %#v, want true", "reveal", got)
		}
		if got := decoded["backlogMode"]; got != true {
			t.Errorf("key %q = %#v, want true", "backlogMode", got)
		}

		clients := objectList(t, decoded, "clients")
		if len(clients) != 3 {
			t.Fatalf("key %q has %d entries, want 3", "clients", len(clients))
		}

		clientKeys := map[string][]string{
			"Alice": votedClientWireKeys,
			"Bob":   votedClientWireKeys,
			"Carol": pendingClientWireKeys,
		}
		for _, client := range clients {
			name, _ := client["name"].(string)
			want, ok := clientKeys[name]
			if !ok {
				t.Fatalf("unexpected client %#v", client)
			}
			assertKeySet(t, "serialized client "+name, client, want)
		}

		stories := objectList(t, decoded, "stories")
		if len(stories) != 1 {
			t.Fatalf("key %q has %d entries, want 1", "stories", len(stories))
		}
		assertKeySet(t, "serialized story", stories[0], estimatedStoryWireKeys)
		if got := stories[0]["voted"]; got != true {
			t.Errorf("story key %q = %#v, want true", "voted", got)
		}
	})

	t.Run("mostAppearingVotes is null or an empty array", func(t *testing.T) {
		cases := []struct {
			name          string
			build         func(t *testing.T) *entity.Room
			wantNull      bool
			wantStoryKeys []string
		}{
			{
				name:          "hidden round keeps null",
				build:         newPendingBacklogRoom,
				wantNull:      true,
				wantStoryKeys: pendingStoryWireKeys,
			},
			{
				name:          "revealed round without countable votes uses an empty array",
				build:         newRevealedEmptyBacklogRoom,
				wantNull:      false,
				wantStoryKeys: pendingStoryWireKeys,
			},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				decoded := decodeSerializedRoomWire(t, tc.build(t))

				assertKeyPresent(t, decoded, "mostAppearingVotes")
				assertVotesValue(t, decoded, "mostAppearingVotes", tc.wantNull)

				stories := objectList(t, decoded, "stories")
				if len(stories) != 1 {
					t.Fatalf("key %q has %d entries, want 1", "stories", len(stories))
				}
				assertKeySet(t, "serialized story", stories[0], tc.wantStoryKeys)
				assertVotesValue(t, stories[0], "mostAppearingVotes", tc.wantNull)
			})
		}
	})
}

// newRevealedBacklogRoom builds an owner, a voter and a spectator on a backlog
// story whose votes auto-reveal once every non-spectator has voted.
func newRevealedBacklogRoom(t *testing.T) *entity.Room {
	t.Helper()

	ctx := context.Background()
	room := newWireRoom(t)
	owner, voter, spectator := addWireParticipants(t, room)

	if err := room.ToggleSpectator(ctx, owner.ID, spectator.ID); err != nil {
		t.Fatalf("toggle spectator: %v", err)
	}
	if err := room.AddStory(ctx, owner.ID, "Checkout flow"); err != nil {
		t.Fatalf("add story: %v", err)
	}

	specialVote := entity.SpecialVoteUnknown
	if err := room.Vote(ctx, owner.ID, entity.VotePtr(specialVote)); err != nil {
		t.Fatalf("owner vote: %v", err)
	}
	five := "5"
	if err := room.Vote(ctx, voter.ID, entity.VotePtr(five)); err != nil {
		t.Fatalf("voter vote: %v", err)
	}
	if !room.Reveal {
		t.Fatal("fixture expects the room to auto-reveal once every voter has voted")
	}

	return room
}

// newPendingBacklogRoom builds a backlog story that has not been voted on.
func newPendingBacklogRoom(t *testing.T) *entity.Room {
	t.Helper()

	ctx := context.Background()
	room := newWireRoom(t)
	owner, _, _ := addWireParticipants(t, room)

	if err := room.AddStory(ctx, owner.ID, "Checkout flow"); err != nil {
		t.Fatalf("add story: %v", err)
	}

	return room
}

// newRevealedEmptyBacklogRoom builds a revealed backlog story nobody voted on,
// so the revealed round has no countable votes.
func newRevealedEmptyBacklogRoom(t *testing.T) *entity.Room {
	t.Helper()

	ctx := context.Background()
	room := newWireRoom(t)
	owner, _, _ := addWireParticipants(t, room)

	if err := room.AddStory(ctx, owner.ID, "Checkout flow"); err != nil {
		t.Fatalf("add story: %v", err)
	}
	if err := room.ToggleReveal(ctx, owner.ID); err != nil {
		t.Fatalf("toggle reveal: %v", err)
	}
	if !room.Reveal {
		t.Fatal("fixture expects the room to be revealed")
	}

	return room
}

func newWireRoom(t *testing.T) *entity.Room {
	t.Helper()

	room := entity.NewRoomWithIDAndStartedAt("room-wire", clientcollection.New(), wireStartedAt)
	room.RoomVersion = entity.RoomVersion(42)
	return room
}

func addWireParticipants(t *testing.T, room *entity.Room) (owner, voter, spectator *entity.Client) {
	t.Helper()

	owner = room.NewClient(entity.ClientID("owner-1"))
	owner.Name = "Alice"
	voter = room.NewClient(entity.ClientID("voter-1"))
	voter.Name = "Bob"
	spectator = room.NewClient(entity.ClientID("spectator-1"))
	spectator.Name = "Carol"

	if !owner.IsOwner {
		t.Fatal("fixture expects the first client to be the room owner")
	}
	return owner, voter, spectator
}

func decodeSerializedRoomWire(t *testing.T, room *entity.Room) map[string]any {
	t.Helper()

	payload, err := SerializeRoom(room)
	if err != nil {
		t.Fatalf("serialize room: %v", err)
	}
	return decodeWireObject(t, payload)
}

func decodeWireObject(t *testing.T, payload []byte) map[string]any {
	t.Helper()

	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.UseNumber()

	var decoded map[string]any
	if err := decoder.Decode(&decoded); err != nil {
		t.Fatalf("decode payload %s: %v", payload, err)
	}
	return decoded
}

func assertKeySet(t *testing.T, label string, object map[string]any, wantKeys []string) {
	t.Helper()

	got := make([]string, 0, len(object))
	for key := range object {
		got = append(got, key)
	}
	slices.Sort(got)

	want := slices.Clone(wantKeys)
	slices.Sort(want)

	if !slices.Equal(got, want) {
		t.Errorf("%s keys = %v, want %v", label, got, want)
	}
}

func assertKeyPresent(t *testing.T, object map[string]any, key string) {
	t.Helper()

	if _, ok := object[key]; !ok {
		t.Fatalf("key %q is missing from %#v", key, object)
	}
}

func assertJSONNumber(t *testing.T, object map[string]any, key string) {
	t.Helper()

	value, ok := object[key]
	if !ok {
		t.Fatalf("key %q is missing", key)
	}
	if _, ok := value.(json.Number); !ok {
		t.Errorf("key %q = %#v (%T), want a json.Number", key, value, value)
	}
}

// assertVotesValue checks the nil-versus-empty distinction of a
// mostAppearingVotes field: JSON null when nil, JSON [] when empty.
func assertVotesValue(t *testing.T, object map[string]any, key string, wantNull bool) {
	t.Helper()

	value := object[key]
	if wantNull {
		if value != nil {
			t.Errorf("key %q = %#v, want null", key, value)
		}
		return
	}

	votes, ok := value.([]any)
	if !ok {
		t.Fatalf("key %q = %#v (%T), want an empty array", key, value, value)
	}
	if len(votes) != 0 {
		t.Errorf("key %q = %#v, want an empty array", key, value)
	}
}

func objectList(t *testing.T, object map[string]any, key string) []map[string]any {
	t.Helper()

	value, ok := object[key]
	if !ok {
		t.Fatalf("key %q is missing", key)
	}

	list, ok := value.([]any)
	if !ok {
		t.Fatalf("key %q = %#v (%T), want an array", key, value, value)
	}

	objects := make([]map[string]any, 0, len(list))
	for _, entry := range list {
		entryObject, ok := entry.(map[string]any)
		if !ok {
			t.Fatalf("key %q contains %#v (%T), want an object", key, entry, entry)
		}
		objects = append(objects, entryObject)
	}
	return objects
}
