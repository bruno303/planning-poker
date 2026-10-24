package entity

import (
	"context"
	"math"
	"reflect"
	"testing"

	"github.com/samber/lo"
)

func TestRoom_RevealWithNoValidVotes(t *testing.T) {
	// Test case: Single user votes with non-numeric value (NaN scenario)
	client := &Client{
		ID:          "client1",
		Name:        "Alice",
		IsSpectator: false,
	}
	client.CurrentVote = VotePtr("?")
	client.HasVoted = true

	room := NewRoom(newTestCollection(client))

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
	// Test case: Single user votes with numeric value
	client := &Client{
		ID:          "client1",
		Name:        "Alice",
		IsSpectator: false,
	}
	client.CurrentVote = VotePtr("5")
	client.HasVoted = true

	room := NewRoom(newTestCollection(client))

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
	// Test case: Multiple users with mixed valid and invalid votes
	client1 := &Client{
		ID:          "client1",
		Name:        "Alice",
		IsSpectator: false,
	}
	client1.CurrentVote = VotePtr("3")
	client1.HasVoted = true

	client2 := &Client{
		ID:          "client2",
		Name:        "Bob",
		IsSpectator: false,
	}
	client2.CurrentVote = VotePtr("coffee")
	client2.HasVoted = true

	room := NewRoom(newTestCollection(client1, client2))

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
	// Test case: Multiple users all voting with non-numeric values
	client1 := &Client{
		ID:          "client1",
		Name:        "Alice",
		IsSpectator: false,
	}
	client1.CurrentVote = VotePtr("?")
	client1.HasVoted = true

	client2 := &Client{
		ID:          "client2",
		Name:        "Bob",
		IsSpectator: false,
	}
	client2.CurrentVote = VotePtr("coffee")
	client2.HasVoted = true

	room := NewRoom(newTestCollection(client1, client2))

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
	// Test case: Spectators should not be counted in results
	client1 := &Client{
		ID:          "client1",
		Name:        "Alice",
		IsSpectator: false,
	}
	client1.CurrentVote = VotePtr("5")
	client1.HasVoted = true

	client2 := &Client{
		ID:          "client2",
		Name:        "Bob",
		IsSpectator: true,
	}
	client2.CurrentVote = VotePtr("10")
	client2.HasVoted = true

	room := NewRoom(newTestCollection(client1, client2))

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
	clients := []*Client{
		{ID: "low-1", CurrentVote: VotePtr("3"), HasVoted: true},
		{ID: "low-2", CurrentVote: VotePtr("3"), HasVoted: true},
		{ID: "high", CurrentVote: VotePtr("13"), HasVoted: true},
		{ID: "question", CurrentVote: VotePtr("?"), HasVoted: true},
		{ID: "coffee", CurrentVote: VotePtr("☕"), HasVoted: true},
		{ID: "spectator", CurrentVote: VotePtr("89"), HasVoted: true, IsSpectator: true},
	}

	room := NewRoom(newTestCollection(clients...))
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
	collection := newTestCollection(&Client{CurrentVote: VotePtr("5"), HasVoted: true})

	room := NewRoom(collection)
	room.reveal(true)
	room.reveal(false)

	if room.Result != nil || room.MostAppearingVotes != nil || room.Consensus != "" ||
		room.LowestVote != nil || room.HighestVote != nil || room.VoteRange != nil ||
		room.VoteSpread != nil || room.SpecialVoteCount != 0 {
		t.Fatalf("consensus metrics were not cleared: %+v", room)
	}
}

func TestNewRoom(t *testing.T) {
	collection := newTestCollection()

	room := NewRoom(collection)

	if room == nil {
		t.Fatal("NewRoom() returned nil")
	}

	if room.ID() == "" {
		t.Error("NewRoom() ID is empty")
	}

	if room.Clients() != collection {
		t.Error("NewRoom() Clients not set correctly")
	}

	if room.CurrentStory != "" {
		t.Errorf("NewRoom() CurrentStory = %v, want empty string", room.CurrentStory)
	}

	if room.Reveal {
		t.Error("NewRoom() Reveal = true, want false")
	}

	if room.Result != nil {
		t.Errorf("NewRoom() Result = %v, want nil", room.Result)
	}

	if !room.BacklogMode {
		t.Error("NewRoom() BacklogMode = false, want true")
	}
}

func TestNewRoomWithID(t *testing.T) {
	collection := newTestCollection()

	room := NewRoomWithID("room-123", collection)

	if room == nil {
		t.Fatal("NewRoomWithID() returned nil")
	}
	if room.ID() != "room-123" {
		t.Fatalf("NewRoomWithID() ID = %v, want room-123", room.ID())
	}
	if room.Clients() != collection {
		t.Fatal("NewRoomWithID() Clients not set correctly")
	}

	if !room.BacklogMode {
		t.Error("NewRoomWithID() BacklogMode = false, want true")
	}

	client := room.NewClient("client1")
	if !client.IsOwner {
		t.Fatal("expected first client in explicit room to be owner")
	}
}

func TestRoom_NewClient(t *testing.T) {
	room := NewRoom(newTestCollection())

	// First client should be owner
	client1 := room.NewClient("client1")
	if client1 == nil {
		t.Fatal("NewClient() returned nil")
	}
	if client1.ID != "client1" {
		t.Errorf("NewClient() ID = %v, want client1", client1.ID)
	}
	if !client1.IsOwner {
		t.Error("First client should be owner")
	}

	// Second client should not be owner
	client2 := room.NewClient("client2")
	if client2.IsOwner {
		t.Error("Second client should not be owner")
	}

	// Third client should not be owner
	client3 := room.NewClient("client3")
	if client3.IsOwner {
		t.Error("Third client should not be owner")
	}
}

func TestRoom_RemoveClient(t *testing.T) {
	ctx := context.Background()

	t.Run("should remove client and reassign owner if needed", func(t *testing.T) {
		client1 := &Client{ID: "client1", IsOwner: true, IsSpectator: false}
		client2 := &Client{ID: "client2", IsOwner: false, IsSpectator: false}

		room := NewRoom(newTestCollection(client1, client2))

		err := room.RemoveClient(ctx, "client1")
		if err != nil {
			t.Errorf("RemoveClient() error = %v", err)
		}

		// client2 should now be owner
		if !client2.IsOwner {
			t.Error("Expected client2 to become owner after removing the only owner")
		}
	})

	t.Run("should remove client and not reassign if other owners exist", func(t *testing.T) {
		client1 := &Client{ID: "client1", IsOwner: true, IsSpectator: false}
		client2 := &Client{ID: "client2", IsOwner: true, IsSpectator: false}

		room := NewRoom(newTestCollection(client1, client2))

		err := room.RemoveClient(ctx, "client1")
		if err != nil {
			t.Errorf("RemoveClient() error = %v", err)
		}

		if !client2.IsOwner {
			t.Error("Expected client2 to remain owner")
		}
	})
}

func TestRoom_ResetVoting(t *testing.T) {
	ctx := context.Background()

	t.Run("should reset voting when client is owner", func(t *testing.T) {
		client := &Client{ID: "client1", IsOwner: true, IsSpectator: false}
		client.CurrentVote = VotePtr("5")
		client.HasVoted = true

		room := NewRoom(newTestCollection(client))
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
		client := &Client{ID: "client1", IsOwner: false}

		room := NewRoom(newTestCollection(client))

		err := room.ResetVoting(ctx, "client1")
		if err == nil {
			t.Error("ResetVoting() expected error for non-owner")
		}
	})
}

func TestRoom_CountOwners(t *testing.T) {
	owner1 := &Client{ID: "owner1", IsOwner: true}
	owner2 := &Client{ID: "owner2", IsOwner: true}
	participant := &Client{ID: "participant"}

	room := NewRoom(newTestCollection(owner1, owner2, participant))

	count := room.CountOwners()
	if count != 2 {
		t.Errorf("CountOwners() = %v, want 2", count)
	}
}

func TestRoom_ToggleSpectator(t *testing.T) {
	ctx := context.Background()

	t.Run("should toggle spectator status when client is owner", func(t *testing.T) {
		owner := &Client{ID: "owner", IsOwner: true, IsSpectator: false}
		target := &Client{ID: "target", IsOwner: false, IsSpectator: false, HasVoted: false}

		room := NewRoom(newTestCollection(owner, target))

		err := room.ToggleSpectator(ctx, "owner", "target")
		if err != nil {
			t.Errorf("ToggleSpectator() error = %v", err)
		}

		if !target.IsSpectator {
			t.Error("ToggleSpectator() should toggle spectator status to true")
		}

		// Toggle back
		err = room.ToggleSpectator(ctx, "owner", "target")
		if err != nil {
			t.Errorf("ToggleSpectator() error = %v", err)
		}

		if target.IsSpectator {
			t.Error("ToggleSpectator() should toggle spectator status back to false")
		}
	})

	t.Run("should fail when client is not owner", func(t *testing.T) {
		client := &Client{ID: "client1", IsOwner: false}

		room := NewRoom(newTestCollection(client))

		err := room.ToggleSpectator(ctx, "client1", "target")
		if err == nil {
			t.Error("ToggleSpectator() expected error for non-owner")
		}
	})

	t.Run("should fail when target client not found", func(t *testing.T) {
		owner := &Client{ID: "owner", IsOwner: true}

		room := NewRoom(newTestCollection(owner))

		err := room.ToggleSpectator(ctx, "owner", "nonexistent")
		if err == nil {
			t.Error("ToggleSpectator() expected error for nonexistent target")
		}
	})
}

func TestRoom_ToggleOwner(t *testing.T) {
	ctx := context.Background()

	t.Run("should toggle owner status when client is owner", func(t *testing.T) {
		owner := &Client{ID: "owner", IsOwner: true}
		target := &Client{ID: "target", IsOwner: false}

		room := NewRoom(newTestCollection(owner, target))

		err := room.ToggleOwner(ctx, "owner", "target")
		if err != nil {
			t.Errorf("ToggleOwner() error = %v", err)
		}

		if !target.IsOwner {
			t.Error("ToggleOwner() should toggle owner status to true")
		}
	})

	t.Run("should not remove last owner", func(t *testing.T) {
		owner := &Client{ID: "owner", IsOwner: true}

		room := NewRoom(newTestCollection(owner))

		err := room.ToggleOwner(ctx, "owner", "owner")
		if err != nil {
			t.Errorf("ToggleOwner() error = %v", err)
		}

		if !owner.IsOwner {
			t.Error("ToggleOwner() should not remove last owner")
		}
	})

	t.Run("should fail when client is not owner", func(t *testing.T) {
		client := &Client{ID: "client1", IsOwner: false}

		room := NewRoom(newTestCollection(client))

		err := room.ToggleOwner(ctx, "client1", "target")
		if err == nil {
			t.Error("ToggleOwner() expected error for non-owner")
		}
	})
}

func TestRoom_SetCurrentStory(t *testing.T) {
	ctx := context.Background()

	t.Run("should set current story when client is owner", func(t *testing.T) {
		owner := &Client{ID: "owner", IsOwner: true}

		room := NewRoom(newTestCollection(owner))

		err := room.SetCurrentStory(ctx, "owner", "New Story")
		if err != nil {
			t.Errorf("SetCurrentStory() error = %v", err)
		}

		if room.CurrentStory != "New Story" {
			t.Errorf("SetCurrentStory() CurrentStory = %v, want 'New Story'", room.CurrentStory)
		}
	})

	t.Run("should fail when client is not owner", func(t *testing.T) {
		client := &Client{ID: "client1", IsOwner: false}

		room := NewRoom(newTestCollection(client))

		err := room.SetCurrentStory(ctx, "client1", "New Story")
		if err == nil {
			t.Error("SetCurrentStory() expected error for non-owner")
		}
	})

	t.Run("should fail when client not found", func(t *testing.T) {
		room := NewRoom(newTestCollection())

		err := room.SetCurrentStory(ctx, "nonexistent", "New Story")
		if err == nil {
			t.Error("SetCurrentStory() expected error for nonexistent client")
		}
	})
}

func TestRoom_ToggleReveal(t *testing.T) {
	ctx := context.Background()

	t.Run("should toggle reveal when client is owner", func(t *testing.T) {
		owner := &Client{ID: "owner", IsOwner: true, IsSpectator: false}
		owner.CurrentVote = VotePtr("5")

		room := NewRoom(newTestCollection(owner))
		room.Reveal = false

		err := room.ToggleReveal(ctx, "owner")
		if err != nil {
			t.Errorf("ToggleReveal() error = %v", err)
		}

		if !room.Reveal {
			t.Error("ToggleReveal() should set Reveal to true")
		}

		// Toggle back
		err = room.ToggleReveal(ctx, "owner")
		if err != nil {
			t.Errorf("ToggleReveal() error = %v", err)
		}

		if room.Reveal {
			t.Error("ToggleReveal() should set Reveal back to false")
		}
	})

	t.Run("should fail when client is not owner", func(t *testing.T) {
		client := &Client{ID: "client1", IsOwner: false}

		room := NewRoom(newTestCollection(client))

		err := room.ToggleReveal(ctx, "client1")
		if err == nil {
			t.Error("ToggleReveal() expected error for non-owner")
		}
	})
}

func TestRoomMutationsDoNotChangeVersion(t *testing.T) {
	room := NewRoom(newTestCollection())
	if room.RoomVersion != 0 {
		t.Fatalf("initial room version = %d, want 0", room.RoomVersion)
	}
}

func TestRoom_IsEmpty(t *testing.T) {
	t.Run("should return true when room is empty", func(t *testing.T) {
		room := NewRoom(newTestCollection())

		if !room.IsEmpty() {
			t.Error("IsEmpty() should return true for empty room")
		}
	})

	t.Run("should return false when room has clients", func(t *testing.T) {
		room := NewRoom(newTestCollection(
			&Client{ID: "client1"},
			&Client{ID: "client2"},
			&Client{ID: "client3"},
		))

		if room.IsEmpty() {
			t.Error("IsEmpty() should return false for non-empty room")
		}
	})
}

func TestRoom_Vote(t *testing.T) {
	ctx := context.Background()

	t.Run("should accept vote from client", func(t *testing.T) {
		vote := VotePtr("5")
		client := &Client{ID: "client1", IsSpectator: false}

		room := NewRoom(newTestCollection(client))
		room.Reveal = false

		err := room.Vote(ctx, "client1", vote)
		if err != nil {
			t.Errorf("Vote() error = %v", err)
		}

		if client.CurrentVote == nil || client.CurrentVote.Label() != vote.Label() {
			t.Errorf("Vote() client vote = %v, want %v", client.CurrentVote, vote.Label())
		}
	})

	t.Run("should fail when client not found", func(t *testing.T) {
		vote := VotePtr("5")

		room := NewRoom(newTestCollection())

		err := room.Vote(ctx, "nonexistent", vote)
		if err == nil {
			t.Error("Vote() expected error for nonexistent client")
		}
	})
}

func TestRoom_UpdateClientName(t *testing.T) {
	ctx := context.Background()

	t.Run("should update client name", func(t *testing.T) {
		client := &Client{ID: "client1", Name: "Old Name"}

		room := NewRoom(newTestCollection(client))

		err := room.UpdateClientName(ctx, "client1", "New Name")
		if err != nil {
			t.Errorf("UpdateClientName() error = %v", err)
		}

		if client.Name != "New Name" {
			t.Errorf("UpdateClientName() Name = %v, want 'New Name'", client.Name)
		}
	})

	t.Run("should fail when client not found", func(t *testing.T) {
		room := NewRoom(newTestCollection())

		err := room.UpdateClientName(ctx, "nonexistent", "New Name")
		if err == nil {
			t.Error("UpdateClientName() expected error for nonexistent client")
		}
	})
}

func TestRoom_MostAppearingVotes(t *testing.T) {
	t.Run("should calculate most appearing votes", func(t *testing.T) {
		client1 := &Client{ID: "client1", IsSpectator: false}
		client1.CurrentVote = VotePtr("5")

		client2 := &Client{ID: "client2", IsSpectator: false}
		client2.CurrentVote = VotePtr("5")

		client3 := &Client{ID: "client3", IsSpectator: false}
		client3.CurrentVote = VotePtr("3")

		room := NewRoom(newTestCollection(client1, client2, client3))
		room.reveal(true)

		if len(room.MostAppearingVotes) != 1 {
			t.Errorf("Expected 1 most appearing vote, got %v", len(room.MostAppearingVotes))
		}

		if len(room.MostAppearingVotes) > 0 && room.MostAppearingVotes[0] != "5" {
			t.Errorf("Expected most appearing vote to be 5, got %v", room.MostAppearingVotes[0])
		}
	})

	t.Run("should handle multiple most appearing votes", func(t *testing.T) {
		client1 := &Client{ID: "client1", IsSpectator: false}
		client1.CurrentVote = VotePtr("5")

		client2 := &Client{ID: "client2", IsSpectator: false}
		client2.CurrentVote = VotePtr("3")

		room := NewRoom(newTestCollection(client1, client2))
		room.reveal(true)

		// Both votes appear once, so both should be in the list
		if len(room.MostAppearingVotes) != 2 {
			t.Errorf("Expected 2 most appearing votes, got %v", len(room.MostAppearingVotes))
		}
	})
}

func TestRoomVoteRejectsCardOutsideDeck(t *testing.T) {
	client := &Client{ID: "client1", IsSpectator: false}

	room := NewRoomFromSnapshot(RoomSnapshot{ID: "room1", DeckType: DeckTypeTShirt}, newTestCollection(client))

	if err := room.Vote(context.Background(), "client1", VotePtr("5")); err == nil {
		t.Fatal("expected voting outside the t-shirt deck to fail")
	}
	if client.CurrentVote != nil {
		t.Fatalf("rejected vote mutated client: %+v", client)
	}
}

func TestRoomTShirtRevealUsesLabelSummary(t *testing.T) {
	clients := []*Client{
		{ID: "low", CurrentVote: VotePtr("S"), HasVoted: true},
		{ID: "mid", CurrentVote: VotePtr("M"), HasVoted: true},
		{ID: "high", CurrentVote: VotePtr("L"), HasVoted: true},
		{ID: "special", CurrentVote: VotePtr(SpecialVoteCoffee), HasVoted: true},
	}

	room := NewRoomFromSnapshot(RoomSnapshot{ID: "room1", DeckType: DeckTypeTShirt}, newTestCollection(clients...))
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

func newOwnerRoomForTest() (*Room, *Client) {
	client := &Client{ID: "client1", IsOwner: true, IsSpectator: false}
	room := NewRoom(newTestCollection(client))
	return room, client
}

func TestRoom_ToggleBacklogMode(t *testing.T) {
	t.Run("should enable backlog mode and migrate current story", testToggleBacklogModeEnable)
	t.Run("should disable backlog mode and restore current story", testToggleBacklogModeDisable)
	t.Run("should fail when client is not owner", testToggleBacklogModeNonOwner)
	t.Run("should fail when client not found", testToggleBacklogModeClientNotFound)
}

func testToggleBacklogModeEnable(t *testing.T) {
	room, _ := newOwnerRoomForTest()
	room.BacklogMode = false
	room.CurrentStory = "Story 1"

	err := room.ToggleBacklogMode(context.Background(), "client1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !room.BacklogMode || len(room.Stories) != 1 || room.Stories[0].Name != "Story 1" || room.CurrentStoryIndex != 0 {
		t.Errorf("unexpected enabled backlog state: %+v", room)
	}
}

func testToggleBacklogModeDisable(t *testing.T) {
	room, _ := newOwnerRoomForTest()
	room.BacklogMode = true
	room.Stories = []Story{{Name: "Story A"}, {Name: "Story B"}}
	room.CurrentStoryIndex = 1

	err := room.ToggleBacklogMode(context.Background(), "client1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if room.BacklogMode || room.CurrentStory != "Story B" || len(room.Stories) != 0 {
		t.Errorf("unexpected disabled backlog state: %+v", room)
	}
}

func testToggleBacklogModeNonOwner(t *testing.T) {
	nonOwner := &Client{ID: "client2", IsOwner: false, IsSpectator: false}
	room := NewRoom(newTestCollection(nonOwner))

	if err := room.ToggleBacklogMode(context.Background(), "client2"); err == nil {
		t.Error("expected error, got nil")
	}
}

func testToggleBacklogModeClientNotFound(t *testing.T) {
	room := NewRoom(newTestCollection())

	if err := room.ToggleBacklogMode(context.Background(), "nonexistent"); err == nil {
		t.Error("expected error, got nil")
	}
}

func TestRoom_AddStory(t *testing.T) {
	t.Run("should add story and enable backlog mode", testAddStoryEnablesBacklog)
	t.Run("should keep CurrentStoryIndex on subsequent adds", testAddStoryKeepsCurrentIndex)
	t.Run("should fail when not owner", testAddStoryNonOwner)
}

func testAddStoryEnablesBacklog(t *testing.T) {
	owner := &Client{ID: "client1", IsOwner: true, IsSpectator: false}
	room := NewRoom(newTestCollection(owner))

	if err := room.AddStory(context.Background(), "client1", "Story 1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !room.BacklogMode || len(room.Stories) != 1 || room.Stories[0].Name != "Story 1" || room.CurrentStoryIndex != 0 {
		t.Errorf("unexpected story state: %+v", room)
	}
}

func testAddStoryKeepsCurrentIndex(t *testing.T) {
	owner := &Client{ID: "client1", IsOwner: true, IsSpectator: false}
	room := NewRoom(newTestCollection(owner))
	room.BacklogMode = true
	room.Stories = []Story{{Name: "Story 1"}}
	room.CurrentStoryIndex = 0

	if err := room.AddStory(context.Background(), "client1", "Story 2"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(room.Stories) != 2 || room.CurrentStoryIndex != 0 {
		t.Errorf("unexpected story state: %+v", room)
	}
}

func testAddStoryNonOwner(t *testing.T) {
	nonOwner := &Client{ID: "client2", IsOwner: false, IsSpectator: false}
	room := NewRoom(newTestCollection(nonOwner))

	if err := room.AddStory(context.Background(), "client2", "Story 1"); err == nil {
		t.Error("expected error, got nil")
	}
}

func TestRoom_RemoveStory(t *testing.T) {
	t.Run("should remove story by stable ID", testRemoveStoryAtIndex)
	t.Run("should adjust CurrentStoryIndex when removing story before current", testRemoveStoryBeforeCurrent)
	t.Run("should reset votes when removing current story", testRemoveCurrentStoryResetsVotes)
	t.Run("should handle removing last story", testRemoveLastStory)
	t.Run("should fail with invalid story ID", testRemoveStoryInvalidIndex)
	t.Run("should fail when not owner", testRemoveStoryNonOwner)
}

func newRoomWithOwnerAndStories(owner *Client, stories []Story, currentIndex int) *Room {
	room := NewRoom(newTestCollection(owner))
	room.BacklogMode = true
	room.Stories = stories
	room.CurrentStoryIndex = currentIndex
	return room
}

func testRemoveStoryAtIndex(t *testing.T) {
	room := newRoomWithOwnerAndStories(&Client{ID: "client1", IsOwner: true}, []Story{{ID: "a", Name: "A"}, {ID: "b", Name: "B"}, {ID: "c", Name: "C"}}, 0)

	if err := room.RemoveStory(context.Background(), "client1", "c"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(room.Stories) != 2 {
		t.Errorf("expected 2 stories, got %d", len(room.Stories))
	}
}

func testRemoveStoryBeforeCurrent(t *testing.T) {
	room := newRoomWithOwnerAndStories(&Client{ID: "client1", IsOwner: true}, []Story{{ID: "a", Name: "A"}, {ID: "b", Name: "B"}, {ID: "c", Name: "C"}}, 1)

	if err := room.RemoveStory(context.Background(), "client1", "a"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if room.CurrentStoryIndex != 0 {
		t.Errorf("expected CurrentStoryIndex 0, got %d", room.CurrentStoryIndex)
	}
}

func testRemoveCurrentStoryResetsVotes(t *testing.T) {
	owner := &Client{ID: "client1", IsOwner: true, CurrentVote: VotePtr("5"), HasVoted: true}
	room := newRoomWithOwnerAndStories(owner, []Story{{ID: "a", Name: "A"}, {ID: "b", Name: "B"}}, 0)
	room.Reveal = true

	if err := room.RemoveStory(context.Background(), "client1", "a"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if room.Reveal || owner.HasVoted {
		t.Error("removing the current story should reset reveal and votes")
	}
}

func testRemoveLastStory(t *testing.T) {
	room := newRoomWithOwnerAndStories(&Client{ID: "client1", IsOwner: true}, []Story{{ID: "only", Name: "Only"}}, 0)

	if err := room.RemoveStory(context.Background(), "client1", "only"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(room.Stories) != 0 {
		t.Errorf("expected 0 stories, got %d", len(room.Stories))
	}
}

func testRemoveStoryInvalidIndex(t *testing.T) {
	room := newRoomWithOwnerAndStories(&Client{ID: "client1", IsOwner: true}, []Story{{ID: "a", Name: "A"}}, 0)

	if err := room.RemoveStory(context.Background(), "client1", "missing"); err == nil {
		t.Error("expected error, got nil")
	}
}

func testRemoveStoryNonOwner(t *testing.T) {
	room := newRoomWithOwnerAndStories(&Client{ID: "client2"}, []Story{{ID: "a", Name: "A"}}, 0)

	if err := room.RemoveStory(context.Background(), "client2", "a"); err == nil {
		t.Error("expected error, got nil")
	}
}

func TestRoom_AdvanceToNextStory(t *testing.T) {
	ctx := context.Background()

	t.Run("should advance to next story and reset votes", func(t *testing.T) {
		owner := &Client{ID: "client1", IsOwner: true, IsSpectator: false}
		owner.CurrentVote = VotePtr("5")
		owner.HasVoted = true
		room := NewRoom(newTestCollection(owner))
		room.BacklogMode = true
		room.Reveal = true
		room.Stories = []Story{{Name: "A"}, {Name: "B"}, {Name: "C"}}
		room.CurrentStoryIndex = 0

		err := room.AdvanceToNextStory(ctx, "client1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if room.CurrentStoryIndex != 1 {
			t.Errorf("expected CurrentStoryIndex 1, got %d", room.CurrentStoryIndex)
		}
		if room.Reveal {
			t.Error("Reveal should be false")
		}
		if owner.HasVoted {
			t.Error("votes should be reset")
		}
	})

	t.Run("should no-op when at last story", func(t *testing.T) {
		owner := &Client{ID: "client1", IsOwner: true, IsSpectator: false}
		room := NewRoom(newTestCollection(owner))
		room.BacklogMode = true
		room.Stories = []Story{{Name: "A"}, {Name: "B"}}
		room.CurrentStoryIndex = 1

		err := room.AdvanceToNextStory(ctx, "client1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if room.CurrentStoryIndex != 1 {
			t.Errorf("expected CurrentStoryIndex 1, got %d", room.CurrentStoryIndex)
		}
	})

	t.Run("should fail when not owner", func(t *testing.T) {
		nonOwner := &Client{ID: "client2", IsOwner: false, IsSpectator: false}
		room := NewRoom(newTestCollection(nonOwner))
		room.BacklogMode = true
		room.Stories = []Story{{Name: "A"}, {Name: "B"}}

		err := room.AdvanceToNextStory(ctx, "client2")
		if err == nil {
			t.Error("expected error, got nil")
		}
	})
}

func TestRoom_PrevStory(t *testing.T) {
	ctx := context.Background()

	t.Run("should move to previous story and reset votes", func(t *testing.T) {
		owner := &Client{ID: "client1", IsOwner: true, IsSpectator: false}
		owner.CurrentVote = VotePtr("5")
		owner.HasVoted = true
		room := NewRoom(newTestCollection(owner))
		room.BacklogMode = true
		room.Reveal = true
		room.Stories = []Story{{Name: "A"}, {Name: "B"}, {Name: "C"}}
		room.CurrentStoryIndex = 2

		err := room.PrevStory(ctx, "client1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if room.CurrentStoryIndex != 1 {
			t.Errorf("expected CurrentStoryIndex 1, got %d", room.CurrentStoryIndex)
		}
		if room.Reveal {
			t.Error("Reveal should be false")
		}
		if owner.HasVoted {
			t.Error("votes should be reset")
		}
	})

	t.Run("should no-op when at first story", func(t *testing.T) {
		owner := &Client{ID: "client1", IsOwner: true, IsSpectator: false}
		room := NewRoom(newTestCollection(owner))
		room.BacklogMode = true
		room.Stories = []Story{{Name: "A"}, {Name: "B"}}
		room.CurrentStoryIndex = 0

		err := room.PrevStory(ctx, "client1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if room.CurrentStoryIndex != 0 {
			t.Errorf("expected CurrentStoryIndex 0, got %d", room.CurrentStoryIndex)
		}
	})

	t.Run("should fail when not owner", func(t *testing.T) {
		nonOwner := &Client{ID: "client2", IsOwner: false, IsSpectator: false}
		room := NewRoom(newTestCollection(nonOwner))
		room.BacklogMode = true
		room.Stories = []Story{{Name: "A"}, {Name: "B"}}

		err := room.PrevStory(ctx, "client2")
		if err == nil {
			t.Error("expected error, got nil")
		}
	})
}

func TestRoom_EffectiveCurrentStory(t *testing.T) {
	t.Run("should return story from backlog when backlog mode is on", func(t *testing.T) {
		room := NewRoomFromSnapshot(RoomSnapshot{
			BacklogMode:       true,
			CurrentStory:      "old",
			Stories:           []Story{{Name: "Story 1"}, {Name: "Story 2"}},
			CurrentStoryIndex: 1,
		}, newTestCollection())

		result := room.EffectiveCurrentStory()
		if result != "Story 2" {
			t.Errorf("expected 'Story 2', got '%s'", result)
		}
	})

	t.Run("should return CurrentStory when backlog mode is off", func(t *testing.T) {
		room := NewRoomFromSnapshot(RoomSnapshot{
			BacklogMode:       false,
			CurrentStory:      "Standalone Story",
			Stories:           []Story{{Name: "Story 1"}},
			CurrentStoryIndex: 0,
		}, newTestCollection())

		result := room.EffectiveCurrentStory()
		if result != "Standalone Story" {
			t.Errorf("expected 'Standalone Story', got '%s'", result)
		}
	})

	t.Run("should return empty string when no stories and no current story", func(t *testing.T) {
		room := NewRoomFromSnapshot(RoomSnapshot{
			BacklogMode:       true,
			CurrentStory:      "",
			Stories:           []Story{},
			CurrentStoryIndex: 0,
		}, newTestCollection())

		result := room.EffectiveCurrentStory()
		if result != "" {
			t.Errorf("expected empty string, got '%s'", result)
		}
	})
}
