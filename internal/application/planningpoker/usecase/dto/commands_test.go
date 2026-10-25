package dto

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/samber/lo"

	"planning-poker/internal/domain/entity"
	"planning-poker/internal/infra/boundaries/hub/clientcollection"
)

func TestNewRoomStateCommand(t *testing.T) {
	vote := "5"
	startedAt := time.Date(2026, time.September, 3, 12, 0, 0, 0, time.UTC)
	votedAt := time.Date(2026, time.September, 3, 12, 1, 0, 0, time.UTC)
	clients := []*entity.Client{
		{ID: "1", Name: "Alice", CurrentVote: entity.VotePtr(vote), HasVoted: true, VotedAt: &votedAt, IsSpectator: false, IsOwner: true},
		{ID: "2", Name: "Bob", CurrentVote: nil, HasVoted: false, IsSpectator: true, IsOwner: false},
	}

	clientCollection := clientcollection.New()
	for _, client := range clients {
		clientCollection.Add(client)
	}

	room := entity.NewRoomFromSnapshot(entity.RoomSnapshot{
		ID:           "room1",
		StartedAt:    startedAt,
		CurrentStory: "Story 1",
		Revealed:     true,
		Outcome: &entity.RoundOutcome{
			Result:             lo.ToPtr(float32(5)),
			MostAppearingVotes: []string{"1", "2"},
			Consensus:          "Medium",
			LowestVote:         lo.ToPtr("3"),
			HighestVote:        lo.ToPtr("8"),
			VoteRange:          lo.ToPtr(5),
			VoteSpread:         lo.ToPtr(2),
			SpecialVoteCount:   1,
		},
		BacklogMode: true,
		Version:     4,
	}, clientCollection)
	got := NewRoomStateCommand(room)
	want := RoomState{
		Type:         "room-state",
		StartedAt:    &startedAt,
		Deck:         entity.DefaultDeck(),
		CurrentStory: "Story 1",
		Reveal:       true,
		Participants: []Participant{
			{ID: "1", Name: "Alice", Vote: &vote, HasVoted: true, VotedAt: &votedAt, IsSpectator: false, IsOwner: true},
			{ID: "2", Name: "Bob", Vote: nil, HasVoted: false, IsSpectator: true, IsOwner: false},
		},
		Result:             lo.ToPtr(float32(5)),
		MostAppearingVotes: []string{"1", "2"},
		Consensus:          "Medium",
		LowestVote:         lo.ToPtr("3"),
		HighestVote:        lo.ToPtr("8"),
		VoteRange:          lo.ToPtr(5),
		VoteSpread:         lo.ToPtr(2),
		SpecialVoteCount:   1,
		BacklogMode:        true,
		Stories:            []Story{},
		CurrentStoryIndex:  0,
		RoomVersion:        4,
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("NewRoomStateCommand() = %+v, want %+v", got, want)
	}
}

// TestNewRoomStateCommandPreservesEmptyMostAppearingVotes keeps the websocket
// payload serializing mostAppearingVotes as [] for a revealed round without
// countable votes, at room and story level.
func TestNewRoomStateCommandPreservesEmptyMostAppearingVotes(t *testing.T) {
	room := entity.NewRoomFromSnapshot(entity.RoomSnapshot{
		ID:           "room1",
		Clients:      []entity.ClientSnapshot{{ID: "owner", Name: "Alice", IsOwner: true}},
		CurrentStory: "Story 1",
		BacklogMode:  true,
		Stories:      []entity.Story{{ID: "story-1", Name: "Story 1"}},
	}, clientcollection.New())

	if err := room.ToggleReveal(context.Background(), "owner"); err != nil {
		t.Fatalf("ToggleReveal returned error: %v", err)
	}

	got := NewRoomStateCommand(room)
	if got.MostAppearingVotes == nil || len(got.MostAppearingVotes) != 0 {
		t.Fatalf("room mostAppearingVotes = %#v, want a non-nil empty slice", got.MostAppearingVotes)
	}
	if len(got.Stories) != 1 {
		t.Fatalf("stories = %d, want 1", len(got.Stories))
	}
	if got.Stories[0].MostAppearingVotes == nil || len(got.Stories[0].MostAppearingVotes) != 0 {
		t.Fatalf("story mostAppearingVotes = %#v, want a non-nil empty slice", got.Stories[0].MostAppearingVotes)
	}
}

func TestNewRoomStateCommandIncludesDeckKind(t *testing.T) {
	room := entity.NewRoomWithIDAndDeck("room1", clientcollection.New(), entity.DeckTypeTShirt)

	got := NewRoomStateCommand(room)
	if got.Deck.Kind != entity.DeckKindCategorical {
		t.Errorf("room state deck kind = %q, want %q", got.Deck.Kind, entity.DeckKindCategorical)
	}
}

func TestNewUpdateClientIDCommand(t *testing.T) {
	got := NewUpdateClientIDCommand("client-123")
	want := UpdateClientID{
		Type:     "update-client-id",
		ClientID: "client-123",
	}
	if got != want {
		t.Errorf("NewUpdateClientIDCommand() = %+v, want %+v", got, want)
	}
}

func TestMapToParticipants(t *testing.T) {
	votedAt := time.Date(2026, time.September, 3, 12, 1, 0, 0, time.UTC)
	clients := []*entity.Client{
		{ID: "1", Name: "Alice", CurrentVote: entity.VotePtr("5"), HasVoted: true, VotedAt: &votedAt, IsSpectator: false, IsOwner: false},
		{ID: "2", Name: "Bob", CurrentVote: entity.VotePtr("3"), HasVoted: true, IsSpectator: false, IsOwner: true},
		{ID: "3", Name: "Charlie", CurrentVote: entity.VotePtr("?"), HasVoted: true, IsSpectator: true, IsOwner: false},
	}
	participants := MapToParticipants(clients)

	expected := []Participant{
		{ID: "1", Name: "Alice", Vote: lo.ToPtr("5"), HasVoted: true, VotedAt: &votedAt, IsSpectator: false, IsOwner: false},
		{ID: "2", Name: "Bob", Vote: lo.ToPtr("3"), HasVoted: true, IsSpectator: false, IsOwner: true},
		{ID: "3", Name: "Charlie", Vote: lo.ToPtr("?"), HasVoted: true, IsSpectator: true, IsOwner: false},
	}

	if !reflect.DeepEqual(participants, expected) {
		t.Errorf("Expected %v, got %v", expected, participants)
	}
}
