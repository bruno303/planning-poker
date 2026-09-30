package dto

import (
	"time"

	"planning-poker/internal/domain/entity"
	"planning-poker/internal/infra/boundaries/hub/clientcollection"
	"reflect"
	"testing"

	"github.com/samber/lo"
)

func TestNewRoomStateCommand(t *testing.T) {
	vote := "5"
	startedAt := time.Date(2026, time.September, 3, 12, 0, 0, 0, time.UTC)
	votedAt := time.Date(2026, time.September, 3, 12, 1, 0, 0, time.UTC)
	clients := []*entity.Client{
		{ID: "1", Name: "Alice", CurrentVote: &vote, HasVoted: true, VotedAt: &votedAt, IsSpectator: false, IsOwner: true},
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
	room.MostCommonVotes = []string{"1", "2"}
	room.Consensus = "Medium"
	room.LowestVote = lo.ToPtr(3)
	room.HighestVote = lo.ToPtr(8)
	room.VoteRange = lo.ToPtr(5)
	room.VoteSpread = lo.ToPtr(2)
	room.NonNumericVoteCount = 1
	room.BacklogMode = true
	room.RoomVersion = 4
	got := NewRoomStateCommand(room)
	want := RoomState{
		Type:         "room-state",
		Deck:         VotingDeck{ID: entity.DeckIDFibonacci, Name: "Fibonacci", Kind: entity.DeckKindNumeric, Cards: []string{"0", "1", "2", "3", "5", "8", "13", "21", "34", "55", "89", "?", "☕"}},
		StartedAt:    &startedAt,
		CurrentStory: "Story 1",
		Reveal:       true,
		Participants: []Participant{
			{ID: "1", Name: "Alice", Vote: &vote, HasVoted: true, VotedAt: &votedAt, IsSpectator: false, IsOwner: true},
			{ID: "2", Name: "Bob", Vote: nil, HasVoted: false, IsSpectator: true, IsOwner: false},
		},
		Result:              lo.ToPtr(float32(5)),
		MostCommonVotes:     []string{"1", "2"},
		MostAppearingVotes:  []int{1, 2},
		Consensus:           "Medium",
		LowestVote:          lo.ToPtr(3),
		HighestVote:         lo.ToPtr(8),
		VoteRange:           lo.ToPtr(5),
		VoteSpread:          lo.ToPtr(2),
		NonNumericVoteCount: 1,
		BacklogMode:         true,
		Stories:             []Story{},
		CurrentStoryIndex:   0,
		RoomVersion:         4,
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("NewRoomStateCommand() = %+v, want %+v", got, want)
	}
}

func TestNewRoomStateCommandProjectsCategoricalModesWithoutNumericEstimates(t *testing.T) {
	room, err := entity.NewRoomWithDeckID(clientcollection.New(), entity.DeckIDTShirt)
	if err != nil {
		t.Fatal(err)
	}
	room.MostCommonVotes = []string{"M", "L"}
	room.Stories = []entity.Story{{ID: "story-1", Name: "Story", MostCommonVotes: []string{"M", "L"}, Voted: true}}

	state := NewRoomStateCommand(room)
	if state.Deck.ID != entity.DeckIDTShirt || state.Deck.Kind != entity.DeckKindCategorical || !reflect.DeepEqual(state.Deck.Cards, []string{"XS", "S", "M", "L", "XL", "XXL", "?", "☕"}) {
		t.Fatalf("categorical deck descriptor = %+v", state.Deck)
	}
	if !reflect.DeepEqual(state.MostCommonVotes, []string{"M", "L"}) || len(state.MostAppearingVotes) != 0 {
		t.Fatalf("categorical room modes = %v legacy numeric modes = %v", state.MostCommonVotes, state.MostAppearingVotes)
	}
	if len(state.Stories) != 1 || !reflect.DeepEqual(state.Stories[0].MostCommonVotes, []string{"M", "L"}) || len(state.Stories[0].MostAppearingVotes) != 0 {
		t.Fatalf("categorical story state = %+v", state.Stories)
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
		{ID: "1", Name: "Alice", CurrentVote: lo.ToPtr("5"), HasVoted: true, VotedAt: &votedAt, IsSpectator: false, IsOwner: false},
		{ID: "2", Name: "Bob", CurrentVote: lo.ToPtr("3"), HasVoted: true, IsSpectator: false, IsOwner: true},
		{ID: "3", Name: "Charlie", CurrentVote: lo.ToPtr("?"), HasVoted: true, IsSpectator: true, IsOwner: false},
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
