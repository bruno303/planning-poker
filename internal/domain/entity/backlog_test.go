package entity_test

import (
	"context"
	"reflect"
	"testing"

	"planning-poker/internal/domain/entity"
	"planning-poker/internal/infra/boundaries/hub/clientcollection"

	"github.com/samber/lo"
)

func newBacklogRoom(stories []entity.Story, currentIndex, version int) (*entity.Room, *entity.Client, *entity.Client) {
	room := &entity.Room{
		ID:                "room-1",
		Clients:           clientcollection.New(),
		BacklogMode:       true,
		Stories:           stories,
		CurrentStoryIndex: currentIndex,
		BacklogVersion:    version,
	}
	owner := room.NewClient("owner")
	owner.IsOwner = true
	participant := room.NewClient("participant")
	return room, owner, participant
}

func TestRoomStoryCreationAssignsStableIDsAndIncrementsVersion(t *testing.T) {
	room, owner, _ := newBacklogRoom(nil, 0, 0)

	if err := room.AddStory(context.Background(), owner.ID, "First"); err != nil {
		t.Fatalf("first AddStory returned error: %v", err)
	}
	if err := room.AddStory(context.Background(), owner.ID, "Second"); err != nil {
		t.Fatalf("second AddStory returned error: %v", err)
	}
	if room.Stories[0].ID == "" || room.Stories[1].ID == "" || room.Stories[0].ID == room.Stories[1].ID {
		t.Fatalf("stories do not have distinct stable IDs: %+v", room.Stories)
	}
	if room.BacklogVersion != 2 {
		t.Fatalf("backlog version after adds = %d, want 2", room.BacklogVersion)
	}

	if err := room.RemoveStory(context.Background(), owner.ID, 1); err != nil {
		t.Fatalf("RemoveStory returned error: %v", err)
	}
	if room.BacklogVersion != 3 {
		t.Fatalf("backlog version after remove = %d, want 3", room.BacklogVersion)
	}
}

func TestRoomReorderStory(t *testing.T) {
	tests := []struct {
		name          string
		storyID       string
		targetIndex   int
		wantOrder     []string
		wantCurrentID string
		wantVersion   int
	}{
		{
			name:          "first to last",
			storyID:       "a",
			targetIndex:   3,
			wantOrder:     []string{"b", "c", "d", "a"},
			wantCurrentID: "b",
			wantVersion:   8,
		},
		{
			name:          "middle to first",
			storyID:       "c",
			targetIndex:   0,
			wantOrder:     []string{"c", "a", "b", "d"},
			wantCurrentID: "b",
			wantVersion:   8,
		},
		{
			name:          "last to middle",
			storyID:       "d",
			targetIndex:   1,
			wantOrder:     []string{"a", "d", "b", "c"},
			wantCurrentID: "b",
			wantVersion:   8,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			room, owner, _ := newBacklogRoom([]entity.Story{
				{ID: "a", Name: "A"},
				{ID: "b", Name: "B", Result: lo.ToPtr(float32(5)), MostAppearingVotes: []int{5}, Voted: true},
				{ID: "c", Name: "C"},
				{ID: "d", Name: "D"},
			}, 1, 7)

			if err := room.ReorderStory(context.Background(), owner.ID, tt.storyID, tt.targetIndex, 7); err != nil {
				t.Fatalf("ReorderStory returned error: %v", err)
			}

			gotOrder := make([]string, len(room.Stories))
			for i, story := range room.Stories {
				gotOrder[i] = story.ID
			}
			if !reflect.DeepEqual(gotOrder, tt.wantOrder) {
				t.Fatalf("story order = %v, want %v", gotOrder, tt.wantOrder)
			}
			if room.Stories[room.CurrentStoryIndex].ID != tt.wantCurrentID {
				t.Fatalf("current story = %q, want %q", room.Stories[room.CurrentStoryIndex].ID, tt.wantCurrentID)
			}
			if room.BacklogVersion != tt.wantVersion {
				t.Fatalf("backlog version = %d, want %d", room.BacklogVersion, tt.wantVersion)
			}
			var estimated *entity.Story
			for i := range room.Stories {
				if room.Stories[i].ID == "b" {
					estimated = &room.Stories[i]
					break
				}
			}
			if estimated == nil || estimated.Result == nil || *estimated.Result != 5 || !reflect.DeepEqual(estimated.MostAppearingVotes, []int{5}) || !estimated.Voted {
				t.Fatal("estimated result was not preserved during reorder")
			}
		})
	}
}

func TestRoomReorderStoryRejectsInvalidCommandsWithoutMutation(t *testing.T) {
	tests := []struct {
		name        string
		clientID    string
		storyID     string
		targetIndex int
		version     int
	}{
		{name: "unknown story", clientID: "owner", storyID: "missing", targetIndex: 1, version: 7},
		{name: "invalid target", clientID: "owner", storyID: "a", targetIndex: 2, version: 7},
		{name: "stale version", clientID: "owner", storyID: "a", targetIndex: 1, version: 6},
		{name: "non owner", clientID: "participant", storyID: "a", targetIndex: 1, version: 7},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			room, _, _ := newBacklogRoom([]entity.Story{{ID: "a", Name: "A"}, {ID: "b", Name: "B"}}, 0, 7)
			before := append([]entity.Story(nil), room.Stories...)

			if err := room.ReorderStory(context.Background(), tt.clientID, tt.storyID, tt.targetIndex, tt.version); err == nil {
				t.Fatal("expected ReorderStory to fail")
			}
			if !reflect.DeepEqual(room.Stories, before) || room.CurrentStoryIndex != 0 || room.BacklogVersion != 7 {
				t.Fatalf("invalid reorder mutated room: %+v", room)
			}
		})
	}
}

func TestRoomSelectStoryResetsActiveVotingAndPreservesEstimates(t *testing.T) {
	estimatedResult := float32(8)
	room, owner, participant := newBacklogRoom([]entity.Story{
		{ID: "current", Name: "Current"},
		{ID: "pending", Name: "Pending"},
		{ID: "estimated", Name: "Estimated", Result: &estimatedResult, MostAppearingVotes: []int{8}, Voted: true},
	}, 0, 4)
	vote := "5"
	owner.CurrentVote = &vote
	owner.HasVoted = true
	participant.CurrentVote = &vote
	participant.HasVoted = true
	room.Reveal = true
	room.Result = lo.ToPtr(float32(5))

	if err := room.SelectStory(context.Background(), owner.ID, "pending"); err != nil {
		t.Fatalf("SelectStory returned error: %v", err)
	}

	if room.CurrentStoryIndex != 1 || room.Reveal || room.Result != nil {
		t.Fatalf("unexpected selected story state: %+v", room)
	}
	for _, client := range []*entity.Client{owner, participant} {
		if client.HasVoted || client.CurrentVote != nil {
			t.Fatalf("client vote was not reset: %+v", client)
		}
	}
	if room.Stories[2].Result == nil || *room.Stories[2].Result != estimatedResult || !room.Stories[2].Voted {
		t.Fatal("estimated story result was changed")
	}
}

func TestRoomSelectStoryRejectsCurrentEstimatedAndNonOwner(t *testing.T) {
	tests := []struct {
		name     string
		clientID string
		storyID  string
	}{
		{name: "current", clientID: "owner", storyID: "current"},
		{name: "estimated", clientID: "owner", storyID: "estimated"},
		{name: "non owner", clientID: "participant", storyID: "pending"},
		{name: "unknown", clientID: "owner", storyID: "missing"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			room, _, _ := newBacklogRoom([]entity.Story{
				{ID: "current", Name: "Current"},
				{ID: "pending", Name: "Pending"},
				{ID: "estimated", Name: "Estimated", Voted: true},
			}, 0, 2)
			before := append([]entity.Story(nil), room.Stories...)

			if err := room.SelectStory(context.Background(), tt.clientID, tt.storyID); err == nil {
				t.Fatal("expected SelectStory to fail")
			}
			if !reflect.DeepEqual(room.Stories, before) || room.CurrentStoryIndex != 0 || room.BacklogVersion != 2 {
				t.Fatalf("invalid selection mutated room: %+v", room)
			}
		})
	}
}
