package entity_test

import (
	"context"
	"testing"
	"time"

	"planning-poker/internal/domain/entity"
	"planning-poker/internal/infra/boundaries/hub/clientcollection"
)

func TestNewRoomInitializesUTCStartTime(t *testing.T) {
	room := entity.NewRoom(clientcollection.New())

	if room.StartedAt().IsZero() {
		t.Fatal("room start time is zero")
	}
	if room.StartedAt().Location() != time.UTC {
		t.Fatalf("room start time location = %v, want UTC", room.StartedAt().Location())
	}
}

func TestRoomStartedAtReturnsValueCopy(t *testing.T) {
	room := entity.NewRoom(clientcollection.New())
	startedAt := room.StartedAt()
	startedAt = startedAt.Add(time.Hour)

	if room.StartedAt().Equal(startedAt) {
		t.Fatal("mutating the returned start time changed the room")
	}
}

func TestClientVoteTracksLatestNonEmptySelectionAndClearsOnReset(t *testing.T) {
	room := entity.NewRoom(clientcollection.New())
	owner := room.NewClient(entity.ClientID("owner"))
	room.NewClient(entity.ClientID("participant"))

	firstVote := "5"
	if err := room.Vote(context.Background(), owner.ID, entity.VotePtr(firstVote)); err != nil {
		t.Fatalf("first vote returned error: %v", err)
	}
	if owner.VotedAt == nil {
		t.Fatal("first non-empty vote did not get a timestamp")
	}
	firstVotedAt := owner.VotedAt

	secondVote := "8"
	if err := room.Vote(context.Background(), owner.ID, entity.VotePtr(secondVote)); err != nil {
		t.Fatalf("second vote returned error: %v", err)
	}
	if owner.VotedAt == nil {
		t.Fatal("replacement non-empty vote did not get a timestamp")
	}
	if owner.VotedAt == nil || owner.VotedAt == firstVotedAt {
		t.Fatal("replacement vote did not replace the timestamp")
	}

	if err := room.ResetVoting(context.Background(), owner.ID); err != nil {
		t.Fatalf("reset voting returned error: %v", err)
	}
	if owner.CurrentVote != nil || owner.HasVoted || owner.VotedAt != nil {
		t.Fatalf("reset did not clear vote state: %+v", owner)
	}
}

func TestRoomRoundResetsClearParticipantVoteTimestamps(t *testing.T) {
	tests := []struct {
		name   string
		setup  func(*entity.Room)
		action func(context.Context, *entity.Room, string) error
	}{
		{
			name: "select story",
			action: func(ctx context.Context, room *entity.Room, ownerID string) error {
				return room.SelectStory(ctx, entity.ClientID(ownerID), entity.StoryID("pending"))
			},
		},
		{
			name: "advance story",
			action: func(ctx context.Context, room *entity.Room, ownerID string) error {
				return room.AdvanceToNextStory(ctx, entity.ClientID(ownerID))
			},
		},
		{
			name:  "previous story",
			setup: func(room *entity.Room) { room.CurrentStoryIndex = 1 },
			action: func(ctx context.Context, room *entity.Room, ownerID string) error {
				return room.PrevStory(ctx, entity.ClientID(ownerID))
			},
		},
		{
			name: "remove active story",
			action: func(ctx context.Context, room *entity.Room, ownerID string) error {
				return room.RemoveStory(ctx, entity.ClientID(ownerID), entity.StoryID("current"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			room := entity.NewRoom(clientcollection.New())
			owner := room.NewClient(entity.ClientID("owner"))
			participant := room.NewClient(entity.ClientID("participant"))
			room.Stories = []entity.Story{{ID: entity.StoryID("current"), Name: "Current"}, {ID: entity.StoryID("pending"), Name: "Pending"}}
			vote := "5"
			votedAt := time.Date(2026, time.September, 3, 12, 0, 0, 0, time.UTC)
			for _, client := range []*entity.Client{owner, participant} {
				client.CurrentVote = entity.VotePtr(vote)
				client.HasVoted = true
				client.VotedAt = &votedAt
			}
			if tt.setup != nil {
				tt.setup(room)
			}

			if err := tt.action(context.Background(), room, owner.ID.String()); err != nil {
				t.Fatalf("round reset returned error: %v", err)
			}
			for _, client := range []*entity.Client{owner, participant} {
				if client.CurrentVote != nil || client.HasVoted || client.VotedAt != nil {
					t.Fatalf("round reset did not clear client state: %+v", client)
				}
			}
		})
	}
}

func TestRoomRemovingOnlyActiveStoryClearsVoteTimestamps(t *testing.T) {
	room := entity.NewRoom(clientcollection.New())
	owner := room.NewClient(entity.ClientID("owner"))
	participant := room.NewClient(entity.ClientID("participant"))
	room.Stories = []entity.Story{{ID: entity.StoryID("only"), Name: "Only story"}}
	vote := "5"
	votedAt := time.Now().UTC()
	for _, client := range []*entity.Client{owner, participant} {
		client.CurrentVote = entity.VotePtr(vote)
		client.HasVoted = true
		client.VotedAt = &votedAt
	}

	if err := room.RemoveStory(context.Background(), owner.ID, entity.StoryID("only")); err != nil {
		t.Fatalf("remove story returned error: %v", err)
	}
	for _, client := range []*entity.Client{owner, participant} {
		if client.CurrentVote != nil || client.HasVoted || client.VotedAt != nil {
			t.Fatalf("only-story removal did not clear client state: %+v", client)
		}
	}
}

func TestRoomStoryTitleEditPreservesVoteTimestamp(t *testing.T) {
	room := entity.NewRoom(clientcollection.New())
	owner := room.NewClient(entity.ClientID("owner"))
	room.Stories = []entity.Story{{ID: entity.StoryID("current"), Name: "Original"}}
	vote := "5"
	votedAt := time.Date(2026, time.September, 3, 12, 0, 0, 0, time.UTC)
	owner.CurrentVote = entity.VotePtr(vote)
	owner.HasVoted = true
	owner.VotedAt = &votedAt

	if err := room.SetCurrentStory(context.Background(), owner.ID, "Edited"); err != nil {
		t.Fatalf("story title edit returned error: %v", err)
	}
	if owner.VotedAt == nil || !owner.VotedAt.Equal(votedAt) || owner.CurrentVote == nil || owner.CurrentVote.String() != vote {
		t.Fatalf("story title edit changed vote state: %+v", owner)
	}
}
