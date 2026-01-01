package entity

import (
	"context"
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

func TestNewRoom(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCC := NewMockClientCollection(ctrl)

	room := NewRoom(mockCC)

	if room == nil {
		t.Fatal("NewRoom() returned nil")
	}

	if room.ID == "" {
		t.Error("NewRoom() ID is empty")
	}

	if room.Clients != mockCC {
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
}

func TestRoom_NewClient(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCC := NewMockClientCollection(ctrl)
	mockCC.EXPECT().Add(gomock.Any()).Times(3)
	mockCC.EXPECT().Count().Return(1).Times(1) // First client
	mockCC.EXPECT().Count().Return(2).Times(1) // Second client
	mockCC.EXPECT().Count().Return(3).Times(1) // Third client

	room := NewRoom(mockCC)

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
	if client1.room != room {
		t.Error("Client room not set correctly")
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
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		client1 := &Client{ID: "client1", IsOwner: true, IsSpectator: false}
		client2 := &Client{ID: "client2", IsOwner: false, IsSpectator: false}

		mockCC := NewMockClientCollection(ctrl)
		mockCC.EXPECT().Remove("client1")
		mockCC.EXPECT().Filter(gomock.Any()).Return(mockCC).AnyTimes()
		mockCC.EXPECT().Count().Return(0).Times(1) // After removal, 0 owners
		mockCC.EXPECT().Count().Return(1).Times(1) // But 1 client remains
		mockCC.EXPECT().First().Return(client2, true)
		mockCC.EXPECT().Values().Return([]*Client{client2}).AnyTimes()

		room := NewRoom(mockCC)
		client1.room = room
		client2.room = room

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
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		client1 := &Client{ID: "client1", IsOwner: true, IsSpectator: false}
		client2 := &Client{ID: "client2", IsOwner: true, IsSpectator: false}

		mockCC := NewMockClientCollection(ctrl)
		mockCC.EXPECT().Remove("client1")
		mockCC.EXPECT().Filter(gomock.Any()).Return(mockCC).AnyTimes()
		mockCC.EXPECT().Count().Return(1).Times(1) // 1 owner remains
		mockCC.EXPECT().Values().Return([]*Client{client2}).AnyTimes()

		room := NewRoom(mockCC)
		client1.room = room
		client2.room = room

		err := room.RemoveClient(ctx, "client1")
		if err != nil {
			t.Errorf("RemoveClient() error = %v", err)
		}
	})
}

func TestRoom_NewVoting(t *testing.T) {
	ctx := context.Background()

	t.Run("should start new voting when client is owner", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		client := &Client{ID: "client1", IsOwner: true, IsSpectator: false}
		client.CurrentVote = lo.ToPtr("5")
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
		room.CurrentStory = "Old story"
		client.room = room

		err := room.NewVoting(ctx, "client1")
		if err != nil {
			t.Errorf("NewVoting() error = %v", err)
		}

		if room.Reveal {
			t.Error("NewVoting() should set Reveal to false")
		}

		if room.CurrentStory != "" {
			t.Errorf("NewVoting() CurrentStory = %v, want empty", room.CurrentStory)
		}

		if client.HasVoted {
			t.Error("NewVoting() should reset client votes")
		}
	})

	t.Run("should fail when client is not owner", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		client := &Client{ID: "client1", IsOwner: false, IsSpectator: false}

		mockCC := NewMockClientCollection(ctrl)
		mockCC.EXPECT().Filter(gomock.Any()).Return(mockCC)
		mockCC.EXPECT().First().Return(client, true)

		room := NewRoom(mockCC)
		client.room = room

		err := room.NewVoting(ctx, "client1")
		if err == nil {
			t.Error("NewVoting() expected error for non-owner")
		}
	})

	t.Run("should fail when client not found", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockCC := NewMockClientCollection(ctrl)
		mockCC.EXPECT().Filter(gomock.Any()).Return(mockCC)
		mockCC.EXPECT().First().Return(nil, false)

		room := NewRoom(mockCC)

		err := room.NewVoting(ctx, "nonexistent")
		if err == nil {
			t.Error("NewVoting() expected error for nonexistent client")
		}
	})
}

func TestRoom_ResetVoting(t *testing.T) {
	ctx := context.Background()

	t.Run("should reset voting when client is owner", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		client := &Client{ID: "client1", IsOwner: true, IsSpectator: false}
		client.CurrentVote = lo.ToPtr("5")
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
		client.room = room

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

		client := &Client{ID: "client1", IsOwner: false}

		mockCC := NewMockClientCollection(ctrl)
		mockCC.EXPECT().Filter(gomock.Any()).Return(mockCC)
		mockCC.EXPECT().First().Return(client, true)

		room := NewRoom(mockCC)
		client.room = room

		err := room.ResetVoting(ctx, "client1")
		if err == nil {
			t.Error("ResetVoting() expected error for non-owner")
		}
	})
}

func TestRoom_CountOwners(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCC := NewMockClientCollection(ctrl)
	mockFilteredCC := NewMockClientCollection(ctrl)
	mockCC.EXPECT().Filter(gomock.Any()).Return(mockFilteredCC)
	mockFilteredCC.EXPECT().Count().Return(2)

	room := NewRoom(mockCC)

	count := room.CountOwners()
	if count != 2 {
		t.Errorf("CountOwners() = %v, want 2", count)
	}
}

func TestRoom_ToggleSpectator(t *testing.T) {
	ctx := context.Background()

	t.Run("should toggle spectator status when client is owner", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		owner := &Client{ID: "owner", IsOwner: true, IsSpectator: false}
		target := &Client{ID: "target", IsOwner: false, IsSpectator: false, HasVoted: false}

		mockCC := NewMockClientCollection(ctrl)
		mockCC.EXPECT().Filter(gomock.Any()).Return(mockCC).Times(3) // getClient twice + checkReveal
		mockCC.EXPECT().First().Return(owner, true).Times(1)
		mockCC.EXPECT().First().Return(target, true).Times(1)
		mockCC.EXPECT().Values().Return([]*Client{owner, target}).AnyTimes()

		room := NewRoom(mockCC)
		owner.room = room
		target.room = room

		err := room.ToggleSpectator(ctx, "owner", "target")
		if err != nil {
			t.Errorf("ToggleSpectator() error = %v", err)
		}

		if !target.IsSpectator {
			t.Error("ToggleSpectator() should toggle spectator status to true")
		}

		// Toggle back
		mockCC.EXPECT().Filter(gomock.Any()).Return(mockCC).Times(3)
		mockCC.EXPECT().First().Return(owner, true).Times(1)
		mockCC.EXPECT().First().Return(target, true).Times(1)

		err = room.ToggleSpectator(ctx, "owner", "target")
		if err != nil {
			t.Errorf("ToggleSpectator() error = %v", err)
		}

		if target.IsSpectator {
			t.Error("ToggleSpectator() should toggle spectator status back to false")
		}
	})

	t.Run("should fail when client is not owner", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		client := &Client{ID: "client1", IsOwner: false}

		mockCC := NewMockClientCollection(ctrl)
		mockCC.EXPECT().Filter(gomock.Any()).Return(mockCC)
		mockCC.EXPECT().First().Return(client, true)

		room := NewRoom(mockCC)

		err := room.ToggleSpectator(ctx, "client1", "target")
		if err == nil {
			t.Error("ToggleSpectator() expected error for non-owner")
		}
	})

	t.Run("should fail when target client not found", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		owner := &Client{ID: "owner", IsOwner: true}

		mockCC := NewMockClientCollection(ctrl)
		mockCC.EXPECT().Filter(gomock.Any()).Return(mockCC).Times(2)
		mockCC.EXPECT().First().Return(owner, true).Times(1)
		mockCC.EXPECT().First().Return(nil, false).Times(1)

		room := NewRoom(mockCC)

		err := room.ToggleSpectator(ctx, "owner", "nonexistent")
		if err == nil {
			t.Error("ToggleSpectator() expected error for nonexistent target")
		}
	})
}

func TestRoom_ToggleOwner(t *testing.T) {
	ctx := context.Background()

	t.Run("should toggle owner status when client is owner", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		owner := &Client{ID: "owner", IsOwner: true}
		target := &Client{ID: "target", IsOwner: false}

		mockCC := NewMockClientCollection(ctrl)
		mockFilteredCC := NewMockClientCollection(ctrl)

		// First getClient for owner
		mockCC.EXPECT().Filter(gomock.Any()).Return(mockCC).Times(1)
		mockCC.EXPECT().First().Return(owner, true).Times(1)

		// Filter for owners
		mockCC.EXPECT().Filter(gomock.Any()).Return(mockFilteredCC).Times(1)
		mockFilteredCC.EXPECT().Count().Return(2) // 2 owners exist

		// Second getClient for target
		mockCC.EXPECT().Filter(gomock.Any()).Return(mockCC).Times(1)
		mockCC.EXPECT().First().Return(target, true).Times(1)

		room := NewRoom(mockCC)

		err := room.ToggleOwner(ctx, "owner", "target")
		if err != nil {
			t.Errorf("ToggleOwner() error = %v", err)
		}

		if !target.IsOwner {
			t.Error("ToggleOwner() should toggle owner status to true")
		}
	})

	t.Run("should not remove last owner", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		owner := &Client{ID: "owner", IsOwner: true}

		mockCC := NewMockClientCollection(ctrl)
		mockFilteredCC := NewMockClientCollection(ctrl)

		mockCC.EXPECT().Filter(gomock.Any()).Return(mockCC).Times(1)
		mockCC.EXPECT().First().Return(owner, true).Times(1)

		mockCC.EXPECT().Filter(gomock.Any()).Return(mockFilteredCC).Times(1)
		mockFilteredCC.EXPECT().Count().Return(1) // Only 1 owner
		mockFilteredCC.EXPECT().First().Return(owner, true)

		room := NewRoom(mockCC)

		err := room.ToggleOwner(ctx, "owner", "owner")
		if err != nil {
			t.Errorf("ToggleOwner() error = %v", err)
		}

		if !owner.IsOwner {
			t.Error("ToggleOwner() should not remove last owner")
		}
	})

	t.Run("should fail when client is not owner", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		client := &Client{ID: "client1", IsOwner: false}

		mockCC := NewMockClientCollection(ctrl)
		mockCC.EXPECT().Filter(gomock.Any()).Return(mockCC)
		mockCC.EXPECT().First().Return(client, true)

		room := NewRoom(mockCC)

		err := room.ToggleOwner(ctx, "client1", "target")
		if err == nil {
			t.Error("ToggleOwner() expected error for non-owner")
		}
	})
}

func TestRoom_SetCurrentStory(t *testing.T) {
	ctx := context.Background()

	t.Run("should set current story when client is owner", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		owner := &Client{ID: "owner", IsOwner: true}

		mockCC := NewMockClientCollection(ctrl)
		mockCC.EXPECT().Filter(gomock.Any()).Return(mockCC)
		mockCC.EXPECT().First().Return(owner, true)

		room := NewRoom(mockCC)

		err := room.SetCurrentStory(ctx, "owner", "New Story")
		if err != nil {
			t.Errorf("SetCurrentStory() error = %v", err)
		}

		if room.CurrentStory != "New Story" {
			t.Errorf("SetCurrentStory() CurrentStory = %v, want 'New Story'", room.CurrentStory)
		}
	})

	t.Run("should fail when client is not owner", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		client := &Client{ID: "client1", IsOwner: false}

		mockCC := NewMockClientCollection(ctrl)
		mockCC.EXPECT().Filter(gomock.Any()).Return(mockCC)
		mockCC.EXPECT().First().Return(client, true)

		room := NewRoom(mockCC)

		err := room.SetCurrentStory(ctx, "client1", "New Story")
		if err == nil {
			t.Error("SetCurrentStory() expected error for non-owner")
		}
	})

	t.Run("should fail when client not found", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockCC := NewMockClientCollection(ctrl)
		mockCC.EXPECT().Filter(gomock.Any()).Return(mockCC)
		mockCC.EXPECT().First().Return(nil, false)

		room := NewRoom(mockCC)

		err := room.SetCurrentStory(ctx, "nonexistent", "New Story")
		if err == nil {
			t.Error("SetCurrentStory() expected error for nonexistent client")
		}
	})
}

func TestRoom_ToggleReveal(t *testing.T) {
	ctx := context.Background()

	t.Run("should toggle reveal when client is owner", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		owner := &Client{ID: "owner", IsOwner: true, IsSpectator: false}
		owner.CurrentVote = lo.ToPtr("5")

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

		client := &Client{ID: "client1", IsOwner: false}

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

func TestRoom_IsEmpty(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCC := NewMockClientCollection(ctrl)

	t.Run("should return true when room is empty", func(t *testing.T) {
		mockCC.EXPECT().Count().Return(0)
		room := NewRoom(mockCC)

		if !room.IsEmpty() {
			t.Error("IsEmpty() should return true for empty room")
		}
	})

	t.Run("should return false when room has clients", func(t *testing.T) {
		mockCC.EXPECT().Count().Return(3)
		room := NewRoom(mockCC)

		if room.IsEmpty() {
			t.Error("IsEmpty() should return false for non-empty room")
		}
	})
}

func TestRoom_Vote(t *testing.T) {
	ctx := context.Background()

	t.Run("should accept vote from client", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		vote := "5"
		client := &Client{ID: "client1", IsSpectator: false}

		mockCC := NewMockClientCollection(ctrl)
		mockCC.EXPECT().Filter(gomock.Any()).Return(mockCC).Times(2) // getClient + checkReveal
		mockCC.EXPECT().First().Return(client, true)
		mockCC.EXPECT().Values().Return([]*Client{client}).AnyTimes()

		room := NewRoom(mockCC)
		room.Reveal = false
		client.room = room

		err := room.Vote(ctx, "client1", &vote)
		if err != nil {
			t.Errorf("Vote() error = %v", err)
		}

		if client.CurrentVote == nil || *client.CurrentVote != vote {
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

		err := room.Vote(ctx, "nonexistent", &vote)
		if err == nil {
			t.Error("Vote() expected error for nonexistent client")
		}
	})
}

func TestRoom_UpdateClientName(t *testing.T) {
	ctx := context.Background()

	t.Run("should update client name", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		client := &Client{ID: "client1", Name: "Old Name"}

		mockCC := NewMockClientCollection(ctrl)
		mockCC.EXPECT().Filter(gomock.Any()).Return(mockCC)
		mockCC.EXPECT().First().Return(client, true)

		room := NewRoom(mockCC)

		err := room.UpdateClientName(ctx, "client1", "New Name")
		if err != nil {
			t.Errorf("UpdateClientName() error = %v", err)
		}

		if client.Name != "New Name" {
			t.Errorf("UpdateClientName() Name = %v, want 'New Name'", client.Name)
		}
	})

	t.Run("should fail when client not found", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockCC := NewMockClientCollection(ctrl)
		mockCC.EXPECT().Filter(gomock.Any()).Return(mockCC)
		mockCC.EXPECT().First().Return(nil, false)

		room := NewRoom(mockCC)

		err := room.UpdateClientName(ctx, "nonexistent", "New Name")
		if err == nil {
			t.Error("UpdateClientName() expected error for nonexistent client")
		}
	})
}

func TestRoom_MostAppearingVotes(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	t.Run("should calculate most appearing votes", func(t *testing.T) {
		client1 := &Client{ID: "client1", IsSpectator: false}
		client1.CurrentVote = lo.ToPtr("5")

		client2 := &Client{ID: "client2", IsSpectator: false}
		client2.CurrentVote = lo.ToPtr("5")

		client3 := &Client{ID: "client3", IsSpectator: false}
		client3.CurrentVote = lo.ToPtr("3")

		mockCC := NewMockClientCollection(ctrl)
		mockCC.EXPECT().Values().Return([]*Client{client1, client2, client3}).AnyTimes()

		room := NewRoom(mockCC)
		room.reveal(true)

		if len(room.MostAppearingVotes) != 1 {
			t.Errorf("Expected 1 most appearing vote, got %v", len(room.MostAppearingVotes))
		}

		if len(room.MostAppearingVotes) > 0 && room.MostAppearingVotes[0] != 5 {
			t.Errorf("Expected most appearing vote to be 5, got %v", room.MostAppearingVotes[0])
		}
	})

	t.Run("should handle multiple most appearing votes", func(t *testing.T) {
		client1 := &Client{ID: "client1", IsSpectator: false}
		client1.CurrentVote = lo.ToPtr("5")

		client2 := &Client{ID: "client2", IsSpectator: false}
		client2.CurrentVote = lo.ToPtr("3")

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
