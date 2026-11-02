package entity

import (
	"context"
	"testing"

	"go.uber.org/mock/gomock"
)

func TestRoom_SetCurrentStory(t *testing.T) {
	ctx := context.Background()

	t.Run("should set current story when client is owner", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		owner := &Client{ID: cid("owner"), IsOwner: true}

		mockCC := NewMockClientCollection(ctrl)
		mockCC.EXPECT().Filter(gomock.Any()).Return(mockCC)
		mockCC.EXPECT().First().Return(owner, true)

		room := NewRoom(mockCC)

		err := room.SetCurrentStory(ctx, cid("owner"), "New Story")
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

		client := &Client{ID: cid("client1"), IsOwner: false}

		mockCC := NewMockClientCollection(ctrl)
		mockCC.EXPECT().Filter(gomock.Any()).Return(mockCC)
		mockCC.EXPECT().First().Return(client, true)

		room := NewRoom(mockCC)

		err := room.SetCurrentStory(ctx, cid("client1"), "New Story")
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

		err := room.SetCurrentStory(ctx, cid("nonexistent"), "New Story")
		if err == nil {
			t.Error("SetCurrentStory() expected error for nonexistent client")
		}
	})
}

func newOwnerRoomForTest(ctrl *gomock.Controller) (*Room, *Client) {
	client := &Client{ID: cid("client1"), IsOwner: true, IsSpectator: false}
	mockCC := NewMockClientCollection(ctrl)
	mockCC.EXPECT().Filter(gomock.Any()).Return(mockCC).AnyTimes()
	mockCC.EXPECT().First().Return(client, true).AnyTimes()
	mockCC.EXPECT().ForEach(gomock.Any()).AnyTimes()
	mockCC.EXPECT().Values().Return([]*Client{}).AnyTimes()
	room := NewRoom(mockCC)
	return room, client
}

func TestRoom_ToggleBacklogMode(t *testing.T) {
	t.Run("should enable backlog mode and migrate current story", testToggleBacklogModeEnable)
	t.Run("should disable backlog mode and restore current story", testToggleBacklogModeDisable)
	t.Run("should fail when client is not owner", testToggleBacklogModeNonOwner)
	t.Run("should fail when client not found", testToggleBacklogModeClientNotFound)
}

func testToggleBacklogModeEnable(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	room, _ := newOwnerRoomForTest(ctrl)
	room.BacklogMode = false
	room.CurrentStory = "Story 1"

	err := room.ToggleBacklogMode(context.Background(), cid("client1"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !room.BacklogMode || len(room.Stories) != 1 || room.Stories[0].Name != "Story 1" || room.CurrentStoryIndex != 0 {
		t.Errorf("unexpected enabled backlog state: %+v", room)
	}
}

func testToggleBacklogModeDisable(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	room, _ := newOwnerRoomForTest(ctrl)
	room.BacklogMode = true
	room.Stories = []Story{{Name: "Story A"}, {Name: "Story B"}}
	room.CurrentStoryIndex = 1

	err := room.ToggleBacklogMode(context.Background(), cid("client1"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if room.BacklogMode || room.CurrentStory != "Story B" || len(room.Stories) != 0 {
		t.Errorf("unexpected disabled backlog state: %+v", room)
	}
}

func testToggleBacklogModeNonOwner(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	nonOwner := &Client{ID: cid("client2"), IsOwner: false, IsSpectator: false}
	mockCC := NewMockClientCollection(ctrl)
	mockCC.EXPECT().Filter(gomock.Any()).Return(mockCC)
	mockCC.EXPECT().First().Return(nonOwner, true)
	room := &Room{ID: "room1", Clients: mockCC}

	if err := room.ToggleBacklogMode(context.Background(), cid("client2")); err == nil {
		t.Error("expected error, got nil")
	}
}

func testToggleBacklogModeClientNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockCC := NewMockClientCollection(ctrl)
	mockCC.EXPECT().Filter(gomock.Any()).Return(mockCC)
	mockCC.EXPECT().First().Return(nil, false)
	room := &Room{ID: "room1", Clients: mockCC}

	if err := room.ToggleBacklogMode(context.Background(), cid("nonexistent")); err == nil {
		t.Error("expected error, got nil")
	}
}

func TestRoom_AddStory(t *testing.T) {
	t.Run("should add story and enable backlog mode", testAddStoryEnablesBacklog)
	t.Run("should keep CurrentStoryIndex on subsequent adds", testAddStoryKeepsCurrentIndex)
	t.Run("should fail when not owner", testAddStoryNonOwner)
}

func testAddStoryEnablesBacklog(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	owner := &Client{ID: cid("client1"), IsOwner: true, IsSpectator: false}
	mockCC := NewMockClientCollection(ctrl)
	mockCC.EXPECT().Filter(gomock.Any()).Return(mockCC)
	mockCC.EXPECT().First().Return(owner, true)
	mockCC.EXPECT().Values().Return([]*Client{}).AnyTimes()
	room := &Room{ID: "room1", Clients: mockCC}

	if err := room.AddStory(context.Background(), cid("client1"), "Story 1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !room.BacklogMode || len(room.Stories) != 1 || room.Stories[0].Name != "Story 1" || room.CurrentStoryIndex != 0 {
		t.Errorf("unexpected story state: %+v", room)
	}
}

func testAddStoryKeepsCurrentIndex(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	owner := &Client{ID: cid("client1"), IsOwner: true, IsSpectator: false}
	mockCC := NewMockClientCollection(ctrl)
	mockCC.EXPECT().Filter(gomock.Any()).Return(mockCC)
	mockCC.EXPECT().First().Return(owner, true)
	mockCC.EXPECT().Values().Return([]*Client{}).AnyTimes()
	room := &Room{ID: "room1", Clients: mockCC, BacklogMode: true, Stories: []Story{{Name: "Story 1"}}, CurrentStoryIndex: 0}

	if err := room.AddStory(context.Background(), cid("client1"), "Story 2"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(room.Stories) != 2 || room.CurrentStoryIndex != 0 {
		t.Errorf("unexpected story state: %+v", room)
	}
}

func testAddStoryNonOwner(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	nonOwner := &Client{ID: cid("client2"), IsOwner: false, IsSpectator: false}
	mockCC := NewMockClientCollection(ctrl)
	mockCC.EXPECT().Filter(gomock.Any()).Return(mockCC)
	mockCC.EXPECT().First().Return(nonOwner, true)
	room := &Room{ID: "room1", Clients: mockCC}

	if err := room.AddStory(context.Background(), cid("client2"), "Story 1"); err == nil {
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

func newRoomWithOwnerAndStories(ctrl *gomock.Controller, owner *Client, stories []Story, currentIndex int) *Room {
	mockCC := NewMockClientCollection(ctrl)
	mockCC.EXPECT().Filter(gomock.Any()).Return(mockCC)
	mockCC.EXPECT().First().Return(owner, true)
	mockCC.EXPECT().ForEach(gomock.Any()).Do(func(f func(*Client)) { f(owner) }).AnyTimes()
	mockCC.EXPECT().Values().Return([]*Client{}).AnyTimes()
	room := NewRoom(mockCC)
	room.BacklogMode = true
	room.Stories = stories
	room.CurrentStoryIndex = currentIndex
	return room
}

func testRemoveStoryAtIndex(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	room := newRoomWithOwnerAndStories(ctrl, &Client{ID: cid("client1"), IsOwner: true}, []Story{{ID: sid("a"), Name: "A"}, {ID: sid("b"), Name: "B"}, {ID: sid("c"), Name: "C"}}, 0)

	if err := room.RemoveStory(context.Background(), cid("client1"), sid("c")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(room.Stories) != 2 {
		t.Errorf("expected 2 stories, got %d", len(room.Stories))
	}
}

func testRemoveStoryBeforeCurrent(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	room := newRoomWithOwnerAndStories(ctrl, &Client{ID: cid("client1"), IsOwner: true}, []Story{{ID: sid("a"), Name: "A"}, {ID: sid("b"), Name: "B"}, {ID: sid("c"), Name: "C"}}, 1)

	if err := room.RemoveStory(context.Background(), cid("client1"), sid("a")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if room.CurrentStoryIndex != 0 {
		t.Errorf("expected CurrentStoryIndex 0, got %d", room.CurrentStoryIndex)
	}
}

func testRemoveCurrentStoryResetsVotes(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	owner := &Client{ID: cid("client1"), IsOwner: true, CurrentVote: VotePtr("5"), HasVoted: true}
	room := newRoomWithOwnerAndStories(ctrl, owner, []Story{{ID: sid("a"), Name: "A"}, {ID: sid("b"), Name: "B"}}, 0)
	room.Reveal = true

	if err := room.RemoveStory(context.Background(), cid("client1"), sid("a")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if room.Reveal || owner.HasVoted {
		t.Error("removing the current story should reset reveal and votes")
	}
}

func testRemoveLastStory(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	room := newRoomWithOwnerAndStories(ctrl, &Client{ID: cid("client1"), IsOwner: true}, []Story{{ID: sid("only"), Name: "Only"}}, 0)

	if err := room.RemoveStory(context.Background(), cid("client1"), sid("only")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(room.Stories) != 0 {
		t.Errorf("expected 0 stories, got %d", len(room.Stories))
	}
}

func testRemoveStoryInvalidIndex(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	room := newRoomWithOwnerAndStories(ctrl, &Client{ID: cid("client1"), IsOwner: true}, []Story{{ID: sid("a"), Name: "A"}}, 0)

	if err := room.RemoveStory(context.Background(), cid("client1"), sid("missing")); err == nil {
		t.Error("expected error, got nil")
	}
}

func testRemoveStoryNonOwner(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	room := newRoomWithOwnerAndStories(ctrl, &Client{ID: cid("client2")}, []Story{{ID: sid("a"), Name: "A"}}, 0)

	if err := room.RemoveStory(context.Background(), cid("client2"), sid("a")); err == nil {
		t.Error("expected error, got nil")
	}
}

func TestRoom_AdvanceToNextStory(t *testing.T) {
	ctx := context.Background()

	t.Run("should advance to next story and reset votes", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		owner := &Client{ID: cid("client1"), IsOwner: true, IsSpectator: false}
		owner.CurrentVote = VotePtr("5")
		owner.HasVoted = true
		mockCC := NewMockClientCollection(ctrl)
		mockCC.EXPECT().Filter(gomock.Any()).Return(mockCC)
		mockCC.EXPECT().First().Return(owner, true)
		mockCC.EXPECT().ForEach(gomock.Any()).Do(func(f func(*Client)) { f(owner) })
		mockCC.EXPECT().Values().Return([]*Client{}).AnyTimes()
		room := NewRoom(mockCC)
		room.BacklogMode = true
		room.Reveal = true
		room.Stories = []Story{{Name: "A"}, {Name: "B"}, {Name: "C"}}
		room.CurrentStoryIndex = 0

		err := room.AdvanceToNextStory(ctx, cid("client1"))
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
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		owner := &Client{ID: cid("client1"), IsOwner: true, IsSpectator: false}
		mockCC := NewMockClientCollection(ctrl)
		mockCC.EXPECT().Filter(gomock.Any()).Return(mockCC)
		mockCC.EXPECT().First().Return(owner, true)
		room := NewRoom(mockCC)
		room.BacklogMode = true
		room.Stories = []Story{{Name: "A"}, {Name: "B"}}
		room.CurrentStoryIndex = 1

		err := room.AdvanceToNextStory(ctx, cid("client1"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if room.CurrentStoryIndex != 1 {
			t.Errorf("expected CurrentStoryIndex 1, got %d", room.CurrentStoryIndex)
		}
	})

	t.Run("should fail when not owner", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		nonOwner := &Client{ID: cid("client2"), IsOwner: false, IsSpectator: false}
		mockCC := NewMockClientCollection(ctrl)
		mockCC.EXPECT().Filter(gomock.Any()).Return(mockCC)
		mockCC.EXPECT().First().Return(nonOwner, true)
		room := NewRoom(mockCC)
		room.BacklogMode = true
		room.Stories = []Story{{Name: "A"}, {Name: "B"}}

		err := room.AdvanceToNextStory(ctx, cid("client2"))
		if err == nil {
			t.Error("expected error, got nil")
		}
	})
}

func TestRoom_PrevStory(t *testing.T) {
	ctx := context.Background()

	t.Run("should move to previous story and reset votes", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		owner := &Client{ID: cid("client1"), IsOwner: true, IsSpectator: false}
		owner.CurrentVote = VotePtr("5")
		owner.HasVoted = true
		mockCC := NewMockClientCollection(ctrl)
		mockCC.EXPECT().Filter(gomock.Any()).Return(mockCC)
		mockCC.EXPECT().First().Return(owner, true)
		mockCC.EXPECT().ForEach(gomock.Any()).Do(func(f func(*Client)) { f(owner) })
		mockCC.EXPECT().Values().Return([]*Client{}).AnyTimes()
		room := NewRoom(mockCC)
		room.BacklogMode = true
		room.Reveal = true
		room.Stories = []Story{{Name: "A"}, {Name: "B"}, {Name: "C"}}
		room.CurrentStoryIndex = 2

		err := room.PrevStory(ctx, cid("client1"))
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
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		owner := &Client{ID: cid("client1"), IsOwner: true, IsSpectator: false}
		mockCC := NewMockClientCollection(ctrl)
		mockCC.EXPECT().Filter(gomock.Any()).Return(mockCC)
		mockCC.EXPECT().First().Return(owner, true)
		room := NewRoom(mockCC)
		room.BacklogMode = true
		room.Stories = []Story{{Name: "A"}, {Name: "B"}}
		room.CurrentStoryIndex = 0

		err := room.PrevStory(ctx, cid("client1"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if room.CurrentStoryIndex != 0 {
			t.Errorf("expected CurrentStoryIndex 0, got %d", room.CurrentStoryIndex)
		}
	})

	t.Run("should fail when not owner", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		nonOwner := &Client{ID: cid("client2"), IsOwner: false, IsSpectator: false}
		mockCC := NewMockClientCollection(ctrl)
		mockCC.EXPECT().Filter(gomock.Any()).Return(mockCC)
		mockCC.EXPECT().First().Return(nonOwner, true)
		room := NewRoom(mockCC)
		room.BacklogMode = true
		room.Stories = []Story{{Name: "A"}, {Name: "B"}}

		err := room.PrevStory(ctx, cid("client2"))
		if err == nil {
			t.Error("expected error, got nil")
		}
	})
}

func TestRoom_EffectiveCurrentStory(t *testing.T) {
	t.Run("should return story from backlog when backlog mode is on", func(t *testing.T) {
		room := &Room{
			BacklogMode:       true,
			CurrentStory:      "old",
			Stories:           []Story{{Name: "Story 1"}, {Name: "Story 2"}},
			CurrentStoryIndex: 1,
		}

		result := room.EffectiveCurrentStory()
		if result != "Story 2" {
			t.Errorf("expected 'Story 2', got '%s'", result)
		}
	})

	t.Run("should return CurrentStory when backlog mode is off", func(t *testing.T) {
		room := &Room{
			BacklogMode:       false,
			CurrentStory:      "Standalone Story",
			Stories:           []Story{{Name: "Story 1"}},
			CurrentStoryIndex: 0,
		}

		result := room.EffectiveCurrentStory()
		if result != "Standalone Story" {
			t.Errorf("expected 'Standalone Story', got '%s'", result)
		}
	})

	t.Run("should return empty string when no stories and no current story", func(t *testing.T) {
		room := &Room{
			BacklogMode:       true,
			CurrentStory:      "",
			Stories:           []Story{},
			CurrentStoryIndex: 0,
		}

		result := room.EffectiveCurrentStory()
		if result != "" {
			t.Errorf("expected empty string, got '%s'", result)
		}
	})
}
