package dto

import (
	"context"
	"time"

	"planning-poker/internal/domain/entity"
	"planning-poker/internal/infra/boundaries/hub/clientcollection"
	"planning-poker/internal/protocol"
	"reflect"
	"testing"

	"github.com/samber/lo"
)

func TestNewRoomStateCommandDoesNotMutateRoomClients(t *testing.T) {
	newRoomWithJoinOrder := func() *entity.Room {
		room := entity.NewRoomWithIDAndStartedAt("room1", clientcollection.New(), time.Now().UTC())
		// Join order is deliberately not alphabetical so DTO sorting stays observable.
		for _, client := range []*entity.Client{
			{ID: "charlie", Name: "Charlie", IsOwner: true},
			{ID: "bob", Name: "Bob"},
			{ID: "alice", Name: "Alice"},
		} {
			room.Clients.Add(client)
		}
		return room
	}

	clientIDs := func(clients []*entity.Client) []string {
		ids := make([]string, 0, len(clients))
		for _, client := range clients {
			ids = append(ids, client.ID)
		}
		return ids
	}

	t.Run("keeps client join order", func(t *testing.T) {
		room := newRoomWithJoinOrder()
		want := clientIDs(room.Clients.Values())

		_ = NewRoomStateCommand(room)

		got := clientIDs(room.Clients.Values())
		if !reflect.DeepEqual(got, want) {
			t.Errorf("NewRoomStateCommand mutated client order: got %v, want %v", got, want)
		}
	})

	t.Run("keeps participants sorted by name", func(t *testing.T) {
		room := newRoomWithJoinOrder()

		state := NewRoomStateCommand(room)

		got := lo.Map(state.Participants, func(p Participant, _ int) string { return p.Name })
		want := []string{"Alice", "Bob", "Charlie"}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("participant order = %v, want %v", got, want)
		}
	})

	t.Run("owner fallback follows join order", func(t *testing.T) {
		room := newRoomWithJoinOrder()
		_ = NewRoomStateCommand(room)

		if err := room.RemoveClient(context.Background(), "charlie"); err != nil {
			t.Fatalf("RemoveClient() error = %v", err)
		}

		first, ok := room.Clients.First()
		if !ok {
			t.Fatal("expected a remaining client after removing the owner")
		}
		if first.ID != "bob" || !first.IsOwner {
			t.Errorf("owner fallback promoted %q (owner=%v), want %q to become owner", first.ID, first.IsOwner, "bob")
		}
	})
}

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

	room := entity.NewRoomWithIDAndStartedAt("room1", clientCollection, startedAt)
	room.CurrentStory = "Story 1"
	room.Reveal = true
	room.Result = lo.ToPtr(float32(5))
	room.MostAppearingVotes = []string{"1", "2"}
	room.Consensus = "Medium"
	room.LowestVote = lo.ToPtr("3")
	room.HighestVote = lo.ToPtr("8")
	room.VoteRange = lo.ToPtr(5)
	room.VoteSpread = lo.ToPtr(2)
	room.SpecialVoteCount = 1
	room.BacklogMode = true
	room.RoomVersion = 4
	deck := entity.DefaultDeck()
	got := NewRoomStateCommand(room)
	want := protocol.RoomState{
		Type:      protocol.MessageTypeRoomState,
		StartedAt: &startedAt,
		Deck: protocol.DeckDescriptor{
			ID:    deck.ID,
			Name:  deck.Name,
			Kind:  protocol.DeckKind(deck.Kind),
			Cards: deck.Cards,
		},
		CurrentStory: "Story 1",
		Reveal:       true,
		Participants: []protocol.Participant{
			{ID: "1", Name: "Alice", Vote: &vote, HasVoted: true, VotedAt: &votedAt, IsSpectator: false, IsOwner: true},
			{ID: "2", Name: "Bob", Vote: nil, HasVoted: false, IsSpectator: true, IsOwner: false},
		},
		Result:             lo.ToPtr(float32(5)),
		MostAppearingVotes: []string{"1", "2"},
		Consensus:          lo.ToPtr(protocol.ConsensusLevelMedium),
		LowestVote:         lo.ToPtr("3"),
		HighestVote:        lo.ToPtr("8"),
		VoteRange:          lo.ToPtr(5),
		VoteSpread:         lo.ToPtr(2),
		SpecialVoteCount:   lo.ToPtr(1),
		BacklogMode:        lo.ToPtr(true),
		Stories:            []protocol.Story{},
		CurrentStoryIndex:  lo.ToPtr(0),
		RoomVersion:        4,
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("NewRoomStateCommand() = %+v, want %+v", got, want)
	}
}

func TestNewRoomStateCommandIncludesDeckKind(t *testing.T) {
	room := entity.NewRoomWithIDAndDeck("room1", clientcollection.New(), entity.DeckTypeTShirt)

	got := NewRoomStateCommand(room)
	if got.Deck.Kind != protocol.DeckKindCategorical {
		t.Errorf("room state deck kind = %q, want %q", got.Deck.Kind, protocol.DeckKindCategorical)
	}
}

func TestNewUpdateClientIDCommand(t *testing.T) {
	got := NewUpdateClientIDCommand("client-123")
	want := protocol.UpdateClientID{
		Type:     protocol.MessageTypeUpdateClientID,
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

	expected := []protocol.Participant{
		{ID: "1", Name: "Alice", Vote: lo.ToPtr("5"), HasVoted: true, VotedAt: &votedAt, IsSpectator: false, IsOwner: false},
		{ID: "2", Name: "Bob", Vote: lo.ToPtr("3"), HasVoted: true, IsSpectator: false, IsOwner: true},
		{ID: "3", Name: "Charlie", Vote: lo.ToPtr("?"), HasVoted: true, IsSpectator: true, IsOwner: false},
	}

	if !reflect.DeepEqual(participants, expected) {
		t.Errorf("Expected %v, got %v", expected, participants)
	}
}
