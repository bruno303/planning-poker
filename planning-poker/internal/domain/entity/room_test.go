package entity

import (
	"math"
	"testing"

	"github.com/samber/lo"
	"go.uber.org/mock/gomock"
)

func TestRoom_RevealWithNoValidVotes(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Test case: Single user votes with non-numeric value (NaN scenario)
	client := &Client{
		ID:          "client1",
		Name:        "Alice",
		IsSpectator: false,
	}
	client.CurrentVote = lo.ToPtr("?")
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
		ID:          "client1",
		Name:        "Alice",
		IsSpectator: false,
	}
	client.CurrentVote = lo.ToPtr("5")
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
		ID:          "client1",
		Name:        "Alice",
		IsSpectator: false,
	}
	client1.CurrentVote = lo.ToPtr("3")
	client1.HasVoted = true

	client2 := &Client{
		ID:          "client2",
		Name:        "Bob",
		IsSpectator: false,
	}
	client2.CurrentVote = lo.ToPtr("coffee")
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
		ID:          "client1",
		Name:        "Alice",
		IsSpectator: false,
	}
	client1.CurrentVote = lo.ToPtr("?")
	client1.HasVoted = true

	client2 := &Client{
		ID:          "client2",
		Name:        "Bob",
		IsSpectator: false,
	}
	client2.CurrentVote = lo.ToPtr("coffee")
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
		ID:          "client1",
		Name:        "Alice",
		IsSpectator: false,
	}
	client1.CurrentVote = lo.ToPtr("5")
	client1.HasVoted = true

	client2 := &Client{
		ID:          "client2",
		Name:        "Bob",
		IsSpectator: true,
	}
	client2.CurrentVote = lo.ToPtr("10")
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
