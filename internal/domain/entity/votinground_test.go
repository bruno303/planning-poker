package entity

import (
	"math"
	"reflect"
	"testing"
	"time"

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
	if result := room.Result(); result != nil {
		if math.IsNaN(float64(*result)) || math.IsInf(float64(*result), 0) {
			t.Errorf("expected Result to be nil when no valid numeric votes, got NaN or Inf: %v", *result)
		} else {
			t.Errorf("expected Result to be nil when no valid numeric votes, got: %v", *result)
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
	if result := room.Result(); result == nil {
		t.Errorf("expected Result to be 5, got nil")
	} else if *result != 5 {
		t.Errorf("expected Result to be 5, got: %v", *result)
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
	if result := room.Result(); result == nil {
		t.Errorf("expected Result to be 3, got nil")
	} else if *result != 3 {
		t.Errorf("expected Result to be 3, got: %v", *result)
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
	if result := room.Result(); result != nil {
		if math.IsNaN(float64(*result)) || math.IsInf(float64(*result), 0) {
			t.Errorf("expected Result to be nil when no valid numeric votes, got NaN or Inf: %v", *result)
		} else {
			t.Errorf("expected Result to be nil when no valid numeric votes, got: %v", *result)
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
	if result := room.Result(); result == nil {
		t.Errorf("expected Result to be 5, got nil")
	} else if *result != 5 {
		t.Errorf("expected Result to be 5, got: %v", *result)
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

	if consensus := room.Consensus(); consensus != consensusLow {
		t.Errorf("consensus = %q, want %q", consensus, consensusLow)
	}
	if !reflect.DeepEqual(room.LowestVote(), lo.ToPtr("3")) {
		t.Errorf("lowest vote = %v, want 3", room.LowestVote())
	}
	if !reflect.DeepEqual(room.HighestVote(), lo.ToPtr("13")) {
		t.Errorf("highest vote = %v, want 13", room.HighestVote())
	}
	if count := room.SpecialVoteCount(); count != 2 {
		t.Errorf("special vote count = %d, want 2", count)
	}
	if !reflect.DeepEqual(room.MostAppearingVotes(), []string{"3"}) {
		t.Errorf("most appearing votes = %v, want [3]", room.MostAppearingVotes())
	}
}

func TestRoom_HideVotesClearsConsensusMetrics(t *testing.T) {
	collection := newTestCollection(&Client{CurrentVote: VotePtr("5"), HasVoted: true})

	room := NewRoom(collection)
	room.reveal(true)
	room.reveal(false)

	if room.Result() != nil || room.MostAppearingVotes() != nil || room.Consensus() != "" ||
		room.LowestVote() != nil || room.HighestVote() != nil || room.VoteRange() != nil ||
		room.VoteSpread() != nil || room.SpecialVoteCount() != 0 {
		t.Fatalf("consensus metrics were not cleared: %+v", room.Outcome())
	}
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

		if len(room.MostAppearingVotes()) != 1 {
			t.Errorf("Expected 1 most appearing vote, got %v", len(room.MostAppearingVotes()))
		}

		if votes := room.MostAppearingVotes(); len(votes) > 0 && votes[0] != "5" {
			t.Errorf("Expected most appearing vote to be 5, got %v", votes[0])
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
		if len(room.MostAppearingVotes()) != 2 {
			t.Errorf("Expected 2 most appearing votes, got %v", len(room.MostAppearingVotes()))
		}
	})
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

	if result := room.Result(); result != nil {
		t.Errorf("t-shirt result = %v, want nil average", result)
	}
	if voteRange := room.VoteRange(); voteRange != nil {
		t.Errorf("t-shirt vote range = %v, want nil", voteRange)
	}
	if count := room.SpecialVoteCount(); count != 1 {
		t.Errorf("special vote count = %d, want 1", count)
	}
	if !reflect.DeepEqual(room.MostAppearingVotes(), []string{"S", "M", "L"}) {
		t.Errorf("most appearing votes = %v, want [S M L]", room.MostAppearingVotes())
	}
	if room.LowestVote() == nil || *room.LowestVote() != "S" || room.HighestVote() == nil || *room.HighestVote() != "L" {
		t.Errorf("extreme votes = (%v, %v), want (S, L)", room.LowestVote(), room.HighestVote())
	}
	if voteSpread := room.VoteSpread(); voteSpread == nil || *voteSpread != 2 {
		t.Errorf("vote spread = %v, want 2", voteSpread)
	}
}

func TestVotingRound_StartsHidden(t *testing.T) {
	round := NewVotingRound()

	if round.Revealed() {
		t.Error("new round is revealed, want hidden")
	}
	if outcome := round.Outcome(); outcome != nil {
		t.Errorf("new round outcome = %+v, want nil", outcome)
	}
	if story := round.CurrentStoryName(); story != "" {
		t.Errorf("new round current story = %q, want empty", story)
	}
}

func TestVotingRound_SetCurrentStoryName(t *testing.T) {
	round := NewVotingRound()

	round.SetCurrentStoryName("Story 1")

	if story := round.CurrentStoryName(); story != "Story 1" {
		t.Errorf("current story = %q, want Story 1", story)
	}
}

func TestVotingRound_RevealComputesOutcomeOnce(t *testing.T) {
	round := NewVotingRound()
	calls := 0
	compute := func() *RoundOutcome {
		calls++
		return &RoundOutcome{Consensus: consensusHigh}
	}

	if !round.Reveal(compute) {
		t.Fatal("expected Reveal to report a transition")
	}
	if calls != 1 {
		t.Fatalf("compute calls = %d, want 1", calls)
	}
	if outcome := round.Outcome(); outcome == nil || outcome.Consensus != consensusHigh {
		t.Fatalf("outcome = %+v, want consensus %q", outcome, consensusHigh)
	}

	if round.Reveal(func() *RoundOutcome {
		calls++
		return &RoundOutcome{Consensus: consensusLow}
	}) {
		t.Fatal("expected Reveal on a revealed round to be a no-op")
	}
	if calls != 1 {
		t.Fatalf("compute calls after no-op = %d, want 1", calls)
	}
	if outcome := round.Outcome(); outcome == nil || outcome.Consensus != consensusHigh {
		t.Fatalf("outcome was overwritten: %+v", outcome)
	}
}

func TestVotingRound_HideClearsOutcome(t *testing.T) {
	round := NewVotingRound()
	round.Reveal(func() *RoundOutcome { return &RoundOutcome{Consensus: consensusHigh} })

	round.Hide()

	if round.Revealed() {
		t.Error("round is still revealed after Hide")
	}
	if outcome := round.Outcome(); outcome != nil {
		t.Errorf("outcome after Hide = %+v, want nil", outcome)
	}
}

func TestVotingRound_Toggle(t *testing.T) {
	round := NewVotingRound()

	if !round.Toggle(func() *RoundOutcome { return &RoundOutcome{Consensus: consensusMedium} }) {
		t.Fatal("expected Toggle to reveal a hidden round")
	}
	if !round.Revealed() {
		t.Error("round is not revealed after Toggle")
	}
	if outcome := round.Outcome(); outcome == nil || outcome.Consensus != consensusMedium {
		t.Fatalf("outcome = %+v, want consensus %q", outcome, consensusMedium)
	}

	if round.Toggle(func() *RoundOutcome { return &RoundOutcome{Consensus: consensusLow} }) {
		t.Fatal("expected Toggle to hide a revealed round")
	}
	if round.Revealed() {
		t.Error("round is still revealed after the second Toggle")
	}
	if outcome := round.Outcome(); outcome != nil {
		t.Errorf("outcome after hiding Toggle = %+v, want nil", outcome)
	}
}

func TestRoundOutcomeClone(t *testing.T) {
	original := &RoundOutcome{
		Result:             lo.ToPtr(float32(5)),
		MostAppearingVotes: []string{"5", "8"},
		Consensus:          consensusMedium,
		LowestVote:         lo.ToPtr("5"),
		HighestVote:        lo.ToPtr("8"),
		VoteRange:          lo.ToPtr(3),
		VoteSpread:         lo.ToPtr(1),
		SpecialVoteCount:   1,
	}

	cloned := original.clone()
	if !reflect.DeepEqual(cloned, original) {
		t.Fatalf("clone = %+v, want %+v", cloned, original)
	}
	if cloned == original {
		t.Fatal("clone returned the original pointer")
	}

	cloned.MostAppearingVotes[0] = "13"
	*cloned.Result = 13
	*cloned.LowestVote = "13"
	*cloned.VoteRange = 8
	*cloned.VoteSpread = 3

	if original.MostAppearingVotes[0] != "5" || *original.Result != 5 || *original.LowestVote != "5" ||
		*original.VoteRange != 3 || *original.VoteSpread != 1 {
		t.Fatalf("clone aliased the original outcome: %+v", original)
	}
}

func TestRoundOutcomeClonePreservesEmptyMostAppearingVotes(t *testing.T) {
	original := &RoundOutcome{MostAppearingVotes: []string{}}

	cloned := original.clone()
	if cloned.MostAppearingVotes == nil {
		t.Fatal("clone turned a non-nil empty MostAppearingVotes into nil")
	}
	if len(cloned.MostAppearingVotes) != 0 {
		t.Fatalf("cloned MostAppearingVotes = %v, want empty", cloned.MostAppearingVotes)
	}

	var nilOutcome RoundOutcome
	if votes := nilOutcome.clone().MostAppearingVotes; votes != nil {
		t.Fatalf("clone turned a nil MostAppearingVotes into %#v", votes)
	}
}

func TestRoundOutcomeNilClone(t *testing.T) {
	var outcome *RoundOutcome

	if cloned := outcome.clone(); cloned != nil {
		t.Fatalf("nil outcome clone = %+v, want nil", cloned)
	}
}

func TestRoom_AccessorsAreEmptyWhileHidden(t *testing.T) {
	room := NewRoom(newTestCollection())

	if room.Outcome() != nil {
		t.Errorf("hidden room outcome = %+v, want nil", room.Outcome())
	}
	if room.Result() != nil || room.MostAppearingVotes() != nil || room.Consensus() != "" ||
		room.LowestVote() != nil || room.HighestVote() != nil || room.VoteRange() != nil ||
		room.VoteSpread() != nil || room.SpecialVoteCount() != 0 {
		t.Fatalf("hidden room exposed outcome values: %+v", room.Outcome())
	}
}

func TestRoom_RevealIsNoOpOnceRevealed(t *testing.T) {
	client := &Client{ID: "client1", CurrentVote: VotePtr("5"), HasVoted: true}
	room := NewRoomWithID("room-1", newTestCollection(client))

	room.reveal(true)
	first := room.Outcome()

	// Bypass Room.Vote so the stored vote changes while the round is revealed.
	client.CurrentVote = VotePtr("8")
	room.reveal(true)

	if room.Outcome() != first {
		t.Fatalf("revealing an already revealed round recomputed the outcome")
	}
	if result := room.Result(); result == nil || *result != 5 {
		t.Fatalf("revealed result = %v, want 5", result)
	}
}

func TestRoomSnapshotRoundTripPreservesRoundState(t *testing.T) {
	votedAt := time.Date(2026, time.October, 1, 10, 30, 0, 0, time.UTC)
	client := &Client{
		ID:          "client1",
		Name:        "Alice",
		CurrentVote: VotePtr("5"),
		HasVoted:    true,
		VotedAt:     &votedAt,
		IsOwner:     true,
	}
	room := NewRoomWithID("room-1", newTestCollection(client))
	room.round.SetCurrentStoryName("Story 1")
	room.reveal(true)

	snapshot := room.Snapshot()
	if !snapshot.Revealed {
		t.Fatal("snapshot is hidden, want revealed")
	}
	if snapshot.Outcome == nil {
		t.Fatal("snapshot outcome is nil, want the revealed outcome")
	}
	if snapshot.CurrentStory != "Story 1" {
		t.Fatalf("snapshot current story = %q, want Story 1", snapshot.CurrentStory)
	}
	if !reflect.DeepEqual(snapshot.Outcome, room.Outcome()) {
		t.Fatalf("snapshot outcome = %+v, want %+v", snapshot.Outcome, room.Outcome())
	}

	restored := NewRoomFromSnapshot(snapshot, newTestCollection())
	if !restored.Revealed() {
		t.Fatal("restored room is hidden, want revealed")
	}
	if !reflect.DeepEqual(restored.Outcome(), room.Outcome()) {
		t.Fatalf("restored outcome = %+v, want %+v", restored.Outcome(), room.Outcome())
	}
	if restored.round.CurrentStoryName() != "Story 1" {
		t.Fatalf("restored current story = %q, want Story 1", restored.round.CurrentStoryName())
	}
	restoredClient, ok := restored.FindClient("client1")
	if !ok {
		t.Fatal("restored room lost its client")
	}
	if restoredClient.CurrentVote == nil || restoredClient.CurrentVote.Label() != "5" ||
		!restoredClient.HasVoted || restoredClient.VotedAt == nil || !restoredClient.VotedAt.Equal(votedAt) {
		t.Fatalf("restored client = %+v, want the persisted vote and timestamp", restoredClient)
	}
}

func TestRoomSnapshotDoesNotAliasOutcome(t *testing.T) {
	client := &Client{ID: "client1", CurrentVote: VotePtr("5"), HasVoted: true}
	room := NewRoomWithID("room-1", newTestCollection(client))
	room.reveal(true)

	snapshot := room.Snapshot()
	*snapshot.Outcome.Result = 99
	snapshot.Outcome.MostAppearingVotes[0] = "13"
	*snapshot.Outcome.LowestVote = "13"
	*snapshot.Outcome.VoteSpread = 7

	if result := room.Result(); result == nil || *result != 5 {
		t.Fatalf("snapshot mutated the room result: %v", result)
	}
	if votes := room.MostAppearingVotes(); len(votes) != 1 || votes[0] != "5" {
		t.Fatalf("snapshot mutated the room most appearing votes: %v", votes)
	}
	if lowest := room.LowestVote(); lowest == nil || *lowest != "5" {
		t.Fatalf("snapshot mutated the room lowest vote: %v", lowest)
	}
	if spread := room.VoteSpread(); spread == nil || *spread != 0 {
		t.Fatalf("snapshot mutated the room vote spread: %v", spread)
	}
}

func TestRoomSnapshotHiddenRoundHasNoOutcome(t *testing.T) {
	room := NewRoomWithID("room-1", newTestCollection())

	snapshot := room.Snapshot()
	if snapshot.Revealed {
		t.Fatal("hidden room snapshot is revealed")
	}
	if snapshot.Outcome != nil {
		t.Fatalf("hidden room snapshot outcome = %+v, want nil", snapshot.Outcome)
	}

	restored := NewRoomFromSnapshot(snapshot, newTestCollection())
	if restored.Revealed() || restored.Outcome() != nil {
		t.Fatalf("restored hidden room exposes reveal=%v outcome=%+v", restored.Revealed(), restored.Outcome())
	}
}
