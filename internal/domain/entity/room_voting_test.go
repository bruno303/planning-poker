package entity

import (
	"context"
	"math"
	"reflect"
	"testing"

	"github.com/samber/lo"
	"go.uber.org/mock/gomock"
)

func TestRoom_RevealWithNoValidVotes(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Test case: Single user votes with non-numeric value (NaN scenario)
	client := &Client{
		ID:          cid("client1"),
		Name:        "Alice",
		IsSpectator: false,
	}
	client.CurrentVote = VotePtr("?")
	client.HasVoted = true

	mockCC := NewMockClientCollection(ctrl)
	mockCC.EXPECT().Values().Return([]*Client{client}).AnyTimes()

	room := NewRoom(mockCC)

	// Reveal the votes
	room.reveal(true)

	// The result should be nil when there are no valid numeric votes
	if room.Result != nil {
		if math.IsNaN(float64(*room.Result)) || math.IsInf(float64(*room.Result), 0) {
			t.Errorf("expected Result to be nil when no valid numeric votes, got NaN or Inf: %v", *room.Result)
		} else {
			t.Errorf("expected Result to be nil when no valid numeric votes, got: %v", *room.Result)
		}
	}
}

func TestRoom_RevealWithSingleValidVote(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Test case: Single user votes with numeric value
	client := &Client{
		ID:          cid("client1"),
		Name:        "Alice",
		IsSpectator: false,
	}
	client.CurrentVote = VotePtr("5")
	client.HasVoted = true

	mockCC := NewMockClientCollection(ctrl)
	mockCC.EXPECT().Values().Return([]*Client{client}).AnyTimes()

	room := NewRoom(mockCC)

	// Reveal the votes
	room.reveal(true)

	// The result should be 5
	if room.Result == nil {
		t.Errorf("expected Result to be 5, got nil")
	} else if *room.Result != 5 {
		t.Errorf("expected Result to be 5, got: %v", *room.Result)
	}
}

func TestRoom_RevealWithMixedVotes(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Test case: Multiple users with mixed valid and invalid votes
	client1 := &Client{
		ID:          cid("client1"),
		Name:        "Alice",
		IsSpectator: false,
	}
	client1.CurrentVote = VotePtr("3")
	client1.HasVoted = true

	client2 := &Client{
		ID:          cid("client2"),
		Name:        "Bob",
		IsSpectator: false,
	}
	client2.CurrentVote = VotePtr("coffee")
	client2.HasVoted = true

	mockCC := NewMockClientCollection(ctrl)
	mockCC.EXPECT().Values().Return([]*Client{client1, client2}).AnyTimes()

	room := NewRoom(mockCC)

	// Reveal the votes
	room.reveal(true)

	// The result should be 3 (average of only the valid vote)
	if room.Result == nil {
		t.Errorf("expected Result to be 3, got nil")
	} else if *room.Result != 3 {
		t.Errorf("expected Result to be 3, got: %v", *room.Result)
	}
}

func TestRoom_RevealWithAllInvalidVotes(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Test case: Multiple users all voting with non-numeric values
	client1 := &Client{
		ID:          cid("client1"),
		Name:        "Alice",
		IsSpectator: false,
	}
	client1.CurrentVote = VotePtr("?")
	client1.HasVoted = true

	client2 := &Client{
		ID:          cid("client2"),
		Name:        "Bob",
		IsSpectator: false,
	}
	client2.CurrentVote = VotePtr("coffee")
	client2.HasVoted = true

	mockCC := NewMockClientCollection(ctrl)
	mockCC.EXPECT().Values().Return([]*Client{client1, client2}).AnyTimes()

	room := NewRoom(mockCC)

	// Reveal the votes
	room.reveal(true)

	// The result should be nil when there are no valid numeric votes
	if room.Result != nil {
		if math.IsNaN(float64(*room.Result)) || math.IsInf(float64(*room.Result), 0) {
			t.Errorf("expected Result to be nil when no valid numeric votes, got NaN or Inf: %v", *room.Result)
		} else {
			t.Errorf("expected Result to be nil when no valid numeric votes, got: %v", *room.Result)
		}
	}
}

func TestRoom_RevealWithSpectators(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Test case: Spectators should not be counted in results
	client1 := &Client{
		ID:          cid("client1"),
		Name:        "Alice",
		IsSpectator: false,
	}
	client1.CurrentVote = VotePtr("5")
	client1.HasVoted = true

	client2 := &Client{
		ID:          cid("client2"),
		Name:        "Bob",
		IsSpectator: true,
	}
	client2.CurrentVote = VotePtr("10")
	client2.HasVoted = true

	mockCC := NewMockClientCollection(ctrl)
	mockCC.EXPECT().Values().Return([]*Client{client1, client2}).AnyTimes()

	room := NewRoom(mockCC)

	// Reveal the votes
	room.reveal(true)

	// The result should be 5 (only Alice's vote counts)
	if room.Result == nil {
		t.Errorf("expected Result to be 5, got nil")
	} else if *room.Result != 5 {
		t.Errorf("expected Result to be 5, got: %v", *room.Result)
	}
}

func TestCalculateConsensus(t *testing.T) {
	fibonacci, _ := DeckByType(DeckTypeFibonacci)
	tests := []struct {
		name       string
		votes      []string
		consensus  string
		lowest     *string
		highest    *string
		voteRange  *int
		voteSpread *int
	}{
		{
			name:      "no numeric votes",
			consensus: consensusUnavailable,
		},
		{
			name:       "single vote",
			votes:      []string{"5"},
			consensus:  consensusHigh,
			lowest:     lo.ToPtr("5"),
			highest:    lo.ToPtr("5"),
			voteRange:  lo.ToPtr(0),
			voteSpread: lo.ToPtr(0),
		},
		{
			name:       "strong majority on adjacent estimates",
			votes:      []string{"3", "5", "5"},
			consensus:  consensusHigh,
			lowest:     lo.ToPtr("3"),
			highest:    lo.ToPtr("5"),
			voteRange:  lo.ToPtr(2),
			voteSpread: lo.ToPtr(1),
		},
		{
			name:       "modest two-step spread",
			votes:      []string{"3", "5", "8"},
			consensus:  consensusMedium,
			lowest:     lo.ToPtr("3"),
			highest:    lo.ToPtr("8"),
			voteRange:  lo.ToPtr(5),
			voteSpread: lo.ToPtr(2),
		},
		{
			name:       "large spread",
			votes:      []string{"3", "13"},
			consensus:  consensusLow,
			lowest:     lo.ToPtr("3"),
			highest:    lo.ToPtr("13"),
			voteRange:  lo.ToPtr(10),
			voteSpread: lo.ToPtr(3),
		},
		{
			name:      "unknown ordered values",
			votes:     []string{"4", "6"},
			consensus: consensusUnavailable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			summary := calculateConsensus(fibonacci, tt.votes)
			assertConsensus(t,
				consensusResult{summary.Consensus, summary.LowestVote, summary.HighestVote, summary.VoteRange, summary.VoteSpread},
				consensusResult{tt.consensus, tt.lowest, tt.highest, tt.voteRange, tt.voteSpread},
			)
		})
	}
}

func TestCalculateConsensusTShirtUsesDeckPositions(t *testing.T) {
	deck, _ := DeckByType(DeckTypeTShirt)

	summary := calculateConsensus(deck, []string{"XS", "S", "M"})

	if summary.Consensus != consensusMedium {
		t.Errorf("consensus = %q, want %q", summary.Consensus, consensusMedium)
	}
	if summary.LowestVote == nil || *summary.LowestVote != "XS" {
		t.Errorf("lowest vote = %v, want XS", summary.LowestVote)
	}
	if summary.HighestVote == nil || *summary.HighestVote != "M" {
		t.Errorf("highest vote = %v, want M", summary.HighestVote)
	}
	if summary.VoteSpread == nil || *summary.VoteSpread != 2 {
		t.Errorf("vote spread = %v, want 2", summary.VoteSpread)
	}
	if summary.VoteRange != nil {
		t.Errorf("vote range = %v, want nil for t-shirt deck", summary.VoteRange)
	}

	single := calculateConsensus(deck, []string{"L", "L"})
	if single.Consensus != consensusHigh || single.VoteSpread == nil || *single.VoteSpread != 0 {
		t.Errorf("single-estimate summary = %+v, want High with spread 0", single)
	}
}

type consensusResult struct {
	consensus  string
	lowest     *string
	highest    *string
	voteRange  *int
	voteSpread *int
}

func assertConsensus(t *testing.T, got, want consensusResult) {
	t.Helper()

	if got.consensus != want.consensus {
		t.Errorf("consensus = %q, want %q", got.consensus, want.consensus)
	}
	if !reflect.DeepEqual(got.lowest, want.lowest) {
		t.Errorf("lowest = %v, want %v", got.lowest, want.lowest)
	}
	if !reflect.DeepEqual(got.highest, want.highest) {
		t.Errorf("highest = %v, want %v", got.highest, want.highest)
	}
	if !reflect.DeepEqual(got.voteRange, want.voteRange) {
		t.Errorf("vote range = %v, want %v", got.voteRange, want.voteRange)
	}
	if !reflect.DeepEqual(got.voteSpread, want.voteSpread) {
		t.Errorf("vote spread = %v, want %v", got.voteSpread, want.voteSpread)
	}
}

func TestRoom_RevealCalculatesConsensusAndCountsSpecialVotes(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	clients := []*Client{
		{ID: cid("low-1"), CurrentVote: VotePtr("3"), HasVoted: true},
		{ID: cid("low-2"), CurrentVote: VotePtr("3"), HasVoted: true},
		{ID: cid("high"), CurrentVote: VotePtr("13"), HasVoted: true},
		{ID: cid("question"), CurrentVote: VotePtr("?"), HasVoted: true},
		{ID: cid("coffee"), CurrentVote: VotePtr("☕"), HasVoted: true},
		{ID: cid("spectator"), CurrentVote: VotePtr("89"), HasVoted: true, IsSpectator: true},
	}
	mockCC := NewMockClientCollection(ctrl)
	mockCC.EXPECT().Values().Return(clients).AnyTimes()

	room := NewRoom(mockCC)
	room.reveal(true)

	if room.Consensus != consensusLow {
		t.Errorf("consensus = %q, want %q", room.Consensus, consensusLow)
	}
	if !reflect.DeepEqual(room.LowestVote, lo.ToPtr("3")) {
		t.Errorf("lowest vote = %v, want 3", room.LowestVote)
	}
	if !reflect.DeepEqual(room.HighestVote, lo.ToPtr("13")) {
		t.Errorf("highest vote = %v, want 13", room.HighestVote)
	}
	if room.SpecialVoteCount != 2 {
		t.Errorf("special vote count = %d, want 2", room.SpecialVoteCount)
	}
	if !reflect.DeepEqual(room.MostAppearingVotes, []string{"3"}) {
		t.Errorf("most appearing votes = %v, want [3]", room.MostAppearingVotes)
	}
}

func TestRoom_HideVotesClearsConsensusMetrics(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCC := NewMockClientCollection(ctrl)
	mockCC.EXPECT().Values().Return([]*Client{{CurrentVote: VotePtr("5"), HasVoted: true}}).AnyTimes()

	room := NewRoom(mockCC)
	room.reveal(true)
	room.reveal(false)

	if room.Result != nil || room.MostAppearingVotes != nil || room.Consensus != "" ||
		room.LowestVote != nil || room.HighestVote != nil || room.VoteRange != nil ||
		room.VoteSpread != nil || room.SpecialVoteCount != 0 {
		t.Fatalf("consensus metrics were not cleared: %+v", room)
	}
}

func TestRoom_ResetVoting(t *testing.T) {
	ctx := context.Background()

	t.Run("should reset voting when client is owner", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		client := &Client{ID: cid("client1"), IsOwner: true, IsSpectator: false}
		client.CurrentVote = VotePtr("5")
		client.HasVoted = true

		mockCC := NewMockClientCollection(ctrl)
		mockCC.EXPECT().Filter(gomock.Any()).Return(mockCC).Times(1)
		mockCC.EXPECT().First().Return(client, true)
		mockCC.EXPECT().ForEach(gomock.Any()).Do(func(f func(*Client)) {
			f(client)
		})
		mockCC.EXPECT().Values().Return([]*Client{}).AnyTimes()

		room := NewRoom(mockCC)
		room.Reveal = true
		room.CurrentStory = "Keep this story"

		err := room.ResetVoting(ctx, "client1")
		if err != nil {
			t.Errorf("ResetVoting() error = %v", err)
		}

		if room.Reveal {
			t.Error("ResetVoting() should set Reveal to false")
		}

		if room.CurrentStory != "Keep this story" {
			t.Error("ResetVoting() should not change CurrentStory")
		}

		if client.HasVoted {
			t.Error("ResetVoting() should reset client votes")
		}
	})

	t.Run("should fail when client is not owner", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		client := &Client{ID: cid("client1"), IsOwner: false}

		mockCC := NewMockClientCollection(ctrl)
		mockCC.EXPECT().Filter(gomock.Any()).Return(mockCC)
		mockCC.EXPECT().First().Return(client, true)

		room := NewRoom(mockCC)

		err := room.ResetVoting(ctx, "client1")
		if err == nil {
			t.Error("ResetVoting() expected error for non-owner")
		}
	})
}

func TestRoom_ToggleReveal(t *testing.T) {
	ctx := context.Background()

	t.Run("should toggle reveal when client is owner", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		owner := &Client{ID: cid("owner"), IsOwner: true, IsSpectator: false}
		owner.CurrentVote = VotePtr("5")

		mockCC := NewMockClientCollection(ctrl)
		mockCC.EXPECT().Filter(gomock.Any()).Return(mockCC)
		mockCC.EXPECT().First().Return(owner, true)
		mockCC.EXPECT().Values().Return([]*Client{owner}).AnyTimes()

		room := NewRoom(mockCC)
		room.Reveal = false

		err := room.ToggleReveal(ctx, "owner")
		if err != nil {
			t.Errorf("ToggleReveal() error = %v", err)
		}

		if !room.Reveal {
			t.Error("ToggleReveal() should set Reveal to true")
		}

		// Toggle back
		mockCC.EXPECT().Filter(gomock.Any()).Return(mockCC)
		mockCC.EXPECT().First().Return(owner, true)

		err = room.ToggleReveal(ctx, "owner")
		if err != nil {
			t.Errorf("ToggleReveal() error = %v", err)
		}

		if room.Reveal {
			t.Error("ToggleReveal() should set Reveal back to false")
		}
	})

	t.Run("should fail when client is not owner", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		client := &Client{ID: cid("client1"), IsOwner: false}

		mockCC := NewMockClientCollection(ctrl)
		mockCC.EXPECT().Filter(gomock.Any()).Return(mockCC)
		mockCC.EXPECT().First().Return(client, true)

		room := NewRoom(mockCC)

		err := room.ToggleReveal(ctx, "client1")
		if err == nil {
			t.Error("ToggleReveal() expected error for non-owner")
		}
	})
}

func TestRoom_Vote(t *testing.T) {
	ctx := context.Background()

	t.Run("should accept vote from client", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		vote := "5"
		client := &Client{ID: cid("client1"), IsSpectator: false}

		mockCC := NewMockClientCollection(ctrl)
		mockCC.EXPECT().Filter(gomock.Any()).Return(mockCC).Times(2) // getClient + checkReveal
		mockCC.EXPECT().First().Return(client, true)
		mockCC.EXPECT().Values().Return([]*Client{client}).AnyTimes()

		room := NewRoom(mockCC)
		room.Reveal = false

		err := room.Vote(ctx, cid("client1"), VotePtr(vote))
		if err != nil {
			t.Errorf("Vote() error = %v", err)
		}

		if client.CurrentVote == nil || client.CurrentVote.String() != vote {
			t.Errorf("Vote() client vote = %v, want %v", client.CurrentVote, vote)
		}
	})

	t.Run("should fail when client not found", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		vote := "5"
		mockCC := NewMockClientCollection(ctrl)
		mockCC.EXPECT().Filter(gomock.Any()).Return(mockCC)
		mockCC.EXPECT().First().Return(nil, false)

		room := NewRoom(mockCC)

		err := room.Vote(ctx, cid("nonexistent"), VotePtr(vote))
		if err == nil {
			t.Error("Vote() expected error for nonexistent client")
		}
	})
}

func TestRoom_MostAppearingVotes(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	t.Run("should calculate most appearing votes", func(t *testing.T) {
		client1 := &Client{ID: cid("client1"), IsSpectator: false}
		client1.CurrentVote = VotePtr("5")

		client2 := &Client{ID: cid("client2"), IsSpectator: false}
		client2.CurrentVote = VotePtr("5")

		client3 := &Client{ID: cid("client3"), IsSpectator: false}
		client3.CurrentVote = VotePtr("3")

		mockCC := NewMockClientCollection(ctrl)
		mockCC.EXPECT().Values().Return([]*Client{client1, client2, client3}).AnyTimes()

		room := NewRoom(mockCC)
		room.reveal(true)

		if len(room.MostAppearingVotes) != 1 {
			t.Errorf("Expected 1 most appearing vote, got %v", len(room.MostAppearingVotes))
		}

		if len(room.MostAppearingVotes) > 0 && room.MostAppearingVotes[0] != "5" {
			t.Errorf("Expected most appearing vote to be 5, got %v", room.MostAppearingVotes[0])
		}
	})

	t.Run("should handle multiple most appearing votes", func(t *testing.T) {
		client1 := &Client{ID: cid("client1"), IsSpectator: false}
		client1.CurrentVote = VotePtr("5")

		client2 := &Client{ID: cid("client2"), IsSpectator: false}
		client2.CurrentVote = VotePtr("3")

		mockCC := NewMockClientCollection(ctrl)
		mockCC.EXPECT().Values().Return([]*Client{client1, client2}).AnyTimes()

		room := NewRoom(mockCC)
		room.reveal(true)

		// Both votes appear once, so both should be in the list
		if len(room.MostAppearingVotes) != 2 {
			t.Errorf("Expected 2 most appearing votes, got %v", len(room.MostAppearingVotes))
		}
	})
}

func TestRoomVoteRejectsCardOutsideDeck(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	client := &Client{ID: cid("client1"), IsSpectator: false}
	mockCC := NewMockClientCollection(ctrl)
	mockCC.EXPECT().Filter(gomock.Any()).Return(mockCC)
	mockCC.EXPECT().First().Return(client, true)

	room := &Room{ID: "room1", Clients: mockCC, DeckType: DeckTypeTShirt}

	if err := room.Vote(context.Background(), cid("client1"), VotePtr("5")); err == nil {
		t.Fatal("expected voting outside the t-shirt deck to fail")
	}
	if client.CurrentVote != nil {
		t.Fatalf("rejected vote mutated client: %+v", client)
	}
}

func TestRoom_VoteIgnoredWhileRevealed(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	client := &Client{ID: cid("client1"), IsSpectator: false}
	mockCC := NewMockClientCollection(ctrl)
	mockCC.EXPECT().Filter(gomock.Any()).Return(mockCC).Times(2) // FindClient + checkReveal
	mockCC.EXPECT().First().Return(client, true)
	mockCC.EXPECT().Values().Return([]*Client{client}).AnyTimes()

	room := NewRoom(mockCC)
	room.Reveal = true

	if err := room.Vote(context.Background(), cid("client1"), VotePtr("5")); err != nil {
		t.Fatalf("Vote() error = %v", err)
	}

	if client.CurrentVote != nil {
		t.Fatalf("vote while revealed mutated client vote: %v", client.CurrentVote)
	}
	if client.HasVoted {
		t.Fatal("vote while revealed marked client as voted")
	}
	if client.VotedAt != nil {
		t.Fatalf("vote while revealed set VotedAt: %v", client.VotedAt)
	}
	if !room.Reveal {
		t.Fatal("Vote() while revealed changed the reveal state")
	}
}

func TestRoomTShirtRevealUsesLabelSummary(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	clients := []*Client{
		{ID: cid("low"), CurrentVote: VotePtr("S"), HasVoted: true},
		{ID: cid("mid"), CurrentVote: VotePtr("M"), HasVoted: true},
		{ID: cid("high"), CurrentVote: VotePtr("L"), HasVoted: true},
		{ID: cid("special"), CurrentVote: VotePtr(SpecialVoteCoffee), HasVoted: true},
	}
	mockCC := NewMockClientCollection(ctrl)
	mockCC.EXPECT().Values().Return(clients).AnyTimes()

	room := &Room{ID: "room1", Clients: mockCC, DeckType: DeckTypeTShirt}
	room.reveal(true)

	if room.Result != nil {
		t.Errorf("t-shirt result = %v, want nil average", room.Result)
	}
	if room.VoteRange != nil {
		t.Errorf("t-shirt vote range = %v, want nil", room.VoteRange)
	}
	if room.SpecialVoteCount != 1 {
		t.Errorf("special vote count = %d, want 1", room.SpecialVoteCount)
	}
	if !reflect.DeepEqual(room.MostAppearingVotes, []string{"S", "M", "L"}) {
		t.Errorf("most appearing votes = %v, want [S M L]", room.MostAppearingVotes)
	}
	if room.LowestVote == nil || *room.LowestVote != "S" || room.HighestVote == nil || *room.HighestVote != "L" {
		t.Errorf("extreme votes = (%v, %v), want (S, L)", room.LowestVote, room.HighestVote)
	}
	if room.VoteSpread == nil || *room.VoteSpread != 2 {
		t.Errorf("vote spread = %v, want 2", room.VoteSpread)
	}
}
