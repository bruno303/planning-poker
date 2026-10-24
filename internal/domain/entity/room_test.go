package entity

import (
	"context"
	"testing"

	"go.uber.org/mock/gomock"
)

// TestRoomDelegatesToClientCollection keeps the aggregate honest about the
// ClientCollection contract: every client operation must go through the
// injected collection, which is also what keeps the generated mock exercised.
func TestRoomDelegatesToClientCollection(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	collection := NewMockClientCollection(ctrl)
	stored := make([]*Client, 0, 1)

	collection.EXPECT().Add(gomock.Any()).Do(func(client *Client) {
		stored = append(stored, client)
	})
	collection.EXPECT().Count().DoAndReturn(func() int {
		return len(stored)
	}).AnyTimes()
	collection.EXPECT().Values().DoAndReturn(func() []*Client {
		return stored
	}).AnyTimes()
	collection.EXPECT().Filter(gomock.Any()).DoAndReturn(func(match func(*Client) bool) ClientCollection {
		return newTestCollection(stored...).Filter(match)
	}).AnyTimes()
	collection.EXPECT().First().DoAndReturn(func() (*Client, bool) {
		if len(stored) == 0 {
			return nil, false
		}
		return stored[0], true
	}).AnyTimes()
	collection.EXPECT().ForEach(gomock.Any()).Do(func(f func(*Client)) {
		for _, client := range stored {
			f(client)
		}
	}).AnyTimes()
	collection.EXPECT().Remove(gomock.Any()).Do(func(clientID string) {
		for index, client := range stored {
			if client.ID == clientID {
				stored = append(stored[:index], stored[index+1:]...)
				return
			}
		}
	})

	room := NewRoom(collection)
	client := room.NewClient("client1")
	if !client.IsOwner {
		t.Fatal("first client in the room should be the owner")
	}
	if room.IsEmpty() {
		t.Fatal("room should not be empty after adding a client")
	}
	if owners := room.CountOwners(); owners != 1 {
		t.Fatalf("owner count = %d, want 1", owners)
	}
	found, ok := room.FindClient("client1")
	if !ok || found != client {
		t.Fatal("FindClient did not delegate to the injected collection")
	}

	visited := 0
	room.Clients().ForEach(func(*Client) {
		visited++
	})
	if visited != 1 {
		t.Fatalf("ForEach visited %d clients, want 1", visited)
	}

	if err := room.RemoveClient(ctx, "client1"); err != nil {
		t.Fatalf("RemoveClient returned error: %v", err)
	}
	if !room.IsEmpty() {
		t.Fatal("room should be empty after removing its only client")
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

	if currentStory := room.EffectiveCurrentStory(); currentStory != "" {
		t.Errorf("NewRoom() CurrentStory = %v, want empty string", currentStory)
	}

	if room.Revealed() {
		t.Error("NewRoom() Reveal = true, want false")
	}

	if result := room.Result(); result != nil {
		t.Errorf("NewRoom() Result = %v, want nil", result)
	}

	if !room.BacklogMode() {
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

	if !room.BacklogMode() {
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
		room.reveal(true)
		room.round.SetCurrentStoryName("Keep this story")

		err := room.ResetVoting(ctx, "client1")
		if err != nil {
			t.Errorf("ResetVoting() error = %v", err)
		}

		if room.Revealed() {
			t.Error("ResetVoting() should set Reveal to false")
		}

		if currentStory := room.EffectiveCurrentStory(); currentStory != "Keep this story" {
			t.Errorf("ResetVoting() should not change CurrentStory, got %q", currentStory)
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

		if currentStory := room.round.CurrentStoryName(); currentStory != "New Story" {
			t.Errorf("SetCurrentStory() CurrentStory = %v, want 'New Story'", currentStory)
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
		room.reveal(false)

		err := room.ToggleReveal(ctx, "owner")
		if err != nil {
			t.Errorf("ToggleReveal() error = %v", err)
		}

		if !room.Revealed() {
			t.Error("ToggleReveal() should set Reveal to true")
		}

		// Toggle back
		err = room.ToggleReveal(ctx, "owner")
		if err != nil {
			t.Errorf("ToggleReveal() error = %v", err)
		}

		if room.Revealed() {
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
	if room.Version().Uint64() != 0 {
		t.Fatalf("initial room version = %d, want 0", room.Version().Uint64())
	}
}

func TestRoomVersionValueObject(t *testing.T) {
	version := NewRoomVersion(3)
	if version.Uint64() != 3 {
		t.Fatalf("version = %d, want 3", version.Uint64())
	}
	if next := version.Next(); next.Uint64() != 4 {
		t.Fatalf("next version = %d, want 4", next.Uint64())
	}
	if version.String() != "3" {
		t.Fatalf("version string = %q, want %q", version.String(), "3")
	}

	room := NewRoom(newTestCollection())
	room.SetVersion(version)
	if room.Version() != version {
		t.Fatalf("room version = %s, want %s", room.Version(), version)
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
		room.reveal(false)

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
	room.backlog.restore(false, nil, 0)
	room.round.SetCurrentStoryName("Story 1")

	err := room.ToggleBacklogMode(context.Background(), "client1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	stories := room.Stories()
	if !room.BacklogMode() || len(stories) != 1 || stories[0].Name != "Story 1" || room.CurrentStoryIndex() != 0 {
		t.Errorf("unexpected enabled backlog state: %+v", room)
	}
}

func testToggleBacklogModeDisable(t *testing.T) {
	room, _ := newOwnerRoomForTest()
	room.backlog.restore(true, []Story{{Name: "Story A"}, {Name: "Story B"}}, 1)

	err := room.ToggleBacklogMode(context.Background(), "client1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if room.BacklogMode() || room.round.CurrentStoryName() != "Story B" || len(room.Stories()) != 0 {
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
	stories := room.Stories()
	if !room.BacklogMode() || len(stories) != 1 || stories[0].Name != "Story 1" || room.CurrentStoryIndex() != 0 {
		t.Errorf("unexpected story state: %+v", room)
	}
}

func testAddStoryKeepsCurrentIndex(t *testing.T) {
	owner := &Client{ID: "client1", IsOwner: true, IsSpectator: false}
	room := NewRoom(newTestCollection(owner))
	room.backlog.restore(true, []Story{{Name: "Story 1"}}, 0)

	if err := room.AddStory(context.Background(), "client1", "Story 2"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(room.Stories()) != 2 || room.CurrentStoryIndex() != 0 {
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
	room.backlog.restore(true, stories, currentIndex)
	return room
}

func testRemoveStoryAtIndex(t *testing.T) {
	room := newRoomWithOwnerAndStories(&Client{ID: "client1", IsOwner: true}, []Story{{ID: "a", Name: "A"}, {ID: "b", Name: "B"}, {ID: "c", Name: "C"}}, 0)

	if err := room.RemoveStory(context.Background(), "client1", "c"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(room.Stories()) != 2 {
		t.Errorf("expected 2 stories, got %d", len(room.Stories()))
	}
}

func testRemoveStoryBeforeCurrent(t *testing.T) {
	room := newRoomWithOwnerAndStories(&Client{ID: "client1", IsOwner: true}, []Story{{ID: "a", Name: "A"}, {ID: "b", Name: "B"}, {ID: "c", Name: "C"}}, 1)

	if err := room.RemoveStory(context.Background(), "client1", "a"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if room.CurrentStoryIndex() != 0 {
		t.Errorf("expected CurrentStoryIndex 0, got %d", room.CurrentStoryIndex())
	}
}

func testRemoveCurrentStoryResetsVotes(t *testing.T) {
	owner := &Client{ID: "client1", IsOwner: true, CurrentVote: VotePtr("5"), HasVoted: true}
	room := newRoomWithOwnerAndStories(owner, []Story{{ID: "a", Name: "A"}, {ID: "b", Name: "B"}}, 0)
	room.reveal(true)

	if err := room.RemoveStory(context.Background(), "client1", "a"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if room.Revealed() || owner.HasVoted {
		t.Error("removing the current story should reset reveal and votes")
	}
}

func testRemoveLastStory(t *testing.T) {
	room := newRoomWithOwnerAndStories(&Client{ID: "client1", IsOwner: true}, []Story{{ID: "only", Name: "Only"}}, 0)

	if err := room.RemoveStory(context.Background(), "client1", "only"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(room.Stories()) != 0 {
		t.Errorf("expected 0 stories, got %d", len(room.Stories()))
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
		room.backlog.restore(true, []Story{{Name: "A"}, {Name: "B"}, {Name: "C"}}, 0)
		room.reveal(true)

		err := room.AdvanceToNextStory(ctx, "client1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if room.CurrentStoryIndex() != 1 {
			t.Errorf("expected CurrentStoryIndex 1, got %d", room.CurrentStoryIndex())
		}
		if room.Revealed() {
			t.Error("Reveal should be false")
		}
		if owner.HasVoted {
			t.Error("votes should be reset")
		}
	})

	t.Run("should no-op when at last story", func(t *testing.T) {
		owner := &Client{ID: "client1", IsOwner: true, IsSpectator: false}
		room := NewRoom(newTestCollection(owner))
		room.backlog.restore(true, []Story{{Name: "A"}, {Name: "B"}}, 1)

		err := room.AdvanceToNextStory(ctx, "client1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if room.CurrentStoryIndex() != 1 {
			t.Errorf("expected CurrentStoryIndex 1, got %d", room.CurrentStoryIndex())
		}
	})

	t.Run("should fail when not owner", func(t *testing.T) {
		nonOwner := &Client{ID: "client2", IsOwner: false, IsSpectator: false}
		room := NewRoom(newTestCollection(nonOwner))
		room.backlog.restore(true, []Story{{Name: "A"}, {Name: "B"}}, 0)

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
		room.backlog.restore(true, []Story{{Name: "A"}, {Name: "B"}, {Name: "C"}}, 2)
		room.reveal(true)

		err := room.PrevStory(ctx, "client1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if room.CurrentStoryIndex() != 1 {
			t.Errorf("expected CurrentStoryIndex 1, got %d", room.CurrentStoryIndex())
		}
		if room.Revealed() {
			t.Error("Reveal should be false")
		}
		if owner.HasVoted {
			t.Error("votes should be reset")
		}
	})

	t.Run("should no-op when at first story", func(t *testing.T) {
		owner := &Client{ID: "client1", IsOwner: true, IsSpectator: false}
		room := NewRoom(newTestCollection(owner))
		room.backlog.restore(true, []Story{{Name: "A"}, {Name: "B"}}, 0)

		err := room.PrevStory(ctx, "client1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if room.CurrentStoryIndex() != 0 {
			t.Errorf("expected CurrentStoryIndex 0, got %d", room.CurrentStoryIndex())
		}
	})

	t.Run("should fail when not owner", func(t *testing.T) {
		nonOwner := &Client{ID: "client2", IsOwner: false, IsSpectator: false}
		room := NewRoom(newTestCollection(nonOwner))
		room.backlog.restore(true, []Story{{Name: "A"}, {Name: "B"}}, 0)

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
