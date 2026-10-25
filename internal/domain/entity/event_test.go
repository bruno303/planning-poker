package entity_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"planning-poker/internal/domain/entity"
	"planning-poker/internal/infra/boundaries/hub/clientcollection"
)

const (
	eventTestRoomID = "room-events"
	eventBufferCap  = 256
)

// newEventRoom builds a room whose backlog matches the given state.
func newEventRoom(
	t *testing.T,
	backlogMode bool,
	stories []entity.Story,
	currentIndex int,
) (*entity.Room, *entity.Client, *entity.Client) {
	t.Helper()

	room := entity.NewRoomFromSnapshot(entity.RoomSnapshot{
		ID:                eventTestRoomID,
		BacklogMode:       backlogMode,
		Stories:           stories,
		CurrentStoryIndex: currentIndex,
	}, clientcollection.New())
	owner := room.NewClient("owner")
	owner.IsOwner = true
	participant := room.NewClient("participant")

	return room, owner, participant
}

func TestEventNamesAndOccurredAt(t *testing.T) {
	at := time.Date(2026, time.October, 25, 10, 30, 0, 0, time.UTC)
	tests := []struct {
		name  string
		event entity.Event
		want  string
	}{
		{name: "vote cast", event: entity.VoteCast{At: at}, want: "vote-cast"},
		{name: "votes revealed", event: entity.VotesRevealed{At: at}, want: "votes-revealed"},
		{name: "story selected", event: entity.StorySelected{At: at}, want: "story-selected"},
		{name: "story estimated", event: entity.StoryEstimated{At: at}, want: "story-estimated"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.event.Name(); got != tt.want {
				t.Fatalf("Name() = %q, want %q", got, tt.want)
			}
			if got := tt.event.OccurredAt(); !got.Equal(at) {
				t.Fatalf("OccurredAt() = %v, want %v", got, at)
			}
		})
	}
}

func TestVoteCastRecordedWhenVoteIsStored(t *testing.T) {
	tests := []struct {
		name      string
		vote      *entity.Vote
		wantLabel string
	}{
		{name: "estimate", vote: entity.VotePtr("5"), wantLabel: "5"},
		{name: "retraction", vote: nil, wantLabel: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			room, _, participant := newEventRoom(t, true, nil, 0)

			if err := room.Vote(context.Background(), participant.ID, tt.vote); err != nil {
				t.Fatalf("Vote returned error: %v", err)
			}

			events := room.PullEvents()
			if len(events) != 1 {
				t.Fatalf("recorded events = %d, want 1", len(events))
			}
			cast, ok := events[0].(entity.VoteCast)
			if !ok {
				t.Fatalf("event has type %T, want entity.VoteCast", events[0])
			}
			if cast.RoomID != eventTestRoomID || cast.ClientID != participant.ID {
				t.Fatalf("VoteCast identifiers = %+v", cast)
			}
			if cast.Vote == nil {
				if tt.wantLabel != "" {
					t.Fatalf("VoteCast vote = nil, want %q", tt.wantLabel)
				}
			} else if got := cast.Vote.Label(); got != tt.wantLabel {
				t.Fatalf("VoteCast vote = %q, want %q", got, tt.wantLabel)
			}
			if cast.OccurredAt().IsZero() {
				t.Fatal("VoteCast OccurredAt is zero")
			}
		})
	}
}

func TestVoteIgnoredWhileRevealedRecordsNoEvent(t *testing.T) {
	room, owner, participant := newEventRoom(t, true, nil, 0)
	ctx := context.Background()

	if err := room.ToggleReveal(ctx, owner.ID); err != nil {
		t.Fatalf("ToggleReveal returned error: %v", err)
	}
	room.PullEvents()

	if err := room.Vote(ctx, participant.ID, entity.VotePtr("5")); err != nil {
		t.Fatalf("Vote returned error: %v", err)
	}

	if events := room.PullEvents(); len(events) != 0 {
		t.Fatalf("ignored vote recorded %d events, want 0", len(events))
	}
}

func TestVotesRevealedOnQuorumClonesOutcome(t *testing.T) {
	room, owner, participant := newEventRoom(t, true, nil, 0)
	ctx := context.Background()

	if err := room.Vote(ctx, owner.ID, entity.VotePtr("5")); err != nil {
		t.Fatalf("owner Vote returned error: %v", err)
	}
	if err := room.Vote(ctx, participant.ID, entity.VotePtr("8")); err != nil {
		t.Fatalf("participant Vote returned error: %v", err)
	}
	if !room.Revealed() {
		t.Fatal("quorum did not reveal the round")
	}

	events := room.PullEvents()
	if len(events) != 3 {
		t.Fatalf("recorded events = %d, want 3", len(events))
	}
	if events[0].Name() != "vote-cast" || events[1].Name() != "vote-cast" {
		t.Fatalf("first two events = %q, %q, want vote-cast twice", events[0].Name(), events[1].Name())
	}
	revealed, ok := events[2].(entity.VotesRevealed)
	if !ok {
		t.Fatalf("event has type %T, want entity.VotesRevealed", events[2])
	}
	if revealed.RoomID != eventTestRoomID {
		t.Fatalf("VotesRevealed room = %q, want %q", revealed.RoomID, eventTestRoomID)
	}
	if revealed.Outcome == nil || revealed.Outcome.Result == nil || *revealed.Outcome.Result != 6.5 {
		t.Fatalf("VotesRevealed outcome = %+v, want result 6.5", revealed.Outcome)
	}

	live := room.Outcome()
	if live == nil {
		t.Fatal("revealed room has no outcome")
	}
	if live == revealed.Outcome {
		t.Fatal("VotesRevealed shares the live outcome pointer")
	}
	*live.Result = 100
	live.MostAppearingVotes = append(live.MostAppearingVotes, "corrupted")

	if *revealed.Outcome.Result != 6.5 {
		t.Fatalf("mutating the room changed the event result to %v", *revealed.Outcome.Result)
	}
	for _, vote := range revealed.Outcome.MostAppearingVotes {
		if vote == "corrupted" {
			t.Fatal("mutating the room changed the event votes")
		}
	}
}

func TestVotesRevealedOnManualRevealOutsideBacklog(t *testing.T) {
	room, owner, participant := newEventRoom(t, false, nil, 0)
	ctx := context.Background()

	if err := room.Vote(ctx, participant.ID, entity.VotePtr("5")); err != nil {
		t.Fatalf("Vote returned error: %v", err)
	}
	room.PullEvents()

	if err := room.ToggleReveal(ctx, owner.ID); err != nil {
		t.Fatalf("ToggleReveal returned error: %v", err)
	}

	events := room.PullEvents()
	if len(events) != 1 {
		t.Fatalf("recorded events = %d, want 1", len(events))
	}
	revealed, ok := events[0].(entity.VotesRevealed)
	if !ok {
		t.Fatalf("event has type %T, want entity.VotesRevealed", events[0])
	}
	if revealed.Outcome == nil || revealed.Outcome.Result == nil || *revealed.Outcome.Result != 5 {
		t.Fatalf("VotesRevealed outcome = %+v, want result 5", revealed.Outcome)
	}

	if err := room.ToggleReveal(ctx, owner.ID); err != nil {
		t.Fatalf("second ToggleReveal returned error: %v", err)
	}
	if events := room.PullEvents(); len(events) != 0 {
		t.Fatalf("hiding the round recorded %d events, want 0", len(events))
	}
}

func TestRevealIsNoOpWhenRoundIsAlreadyRevealed(t *testing.T) {
	room, owner, participant := newEventRoom(t, true, nil, 0)
	ctx := context.Background()

	if err := room.Vote(ctx, owner.ID, entity.VotePtr("5")); err != nil {
		t.Fatalf("owner Vote returned error: %v", err)
	}
	if err := room.Vote(ctx, participant.ID, entity.VotePtr("8")); err != nil {
		t.Fatalf("participant Vote returned error: %v", err)
	}
	room.PullEvents()

	// Removing the participant leaves a fully voted active set, so the room
	// checks for quorum again and must not record a second reveal.
	if err := room.RemoveClient(ctx, participant.ID); err != nil {
		t.Fatalf("RemoveClient returned error: %v", err)
	}

	if events := room.PullEvents(); len(events) != 0 {
		t.Fatalf("no-op reveal recorded %d events, want 0", len(events))
	}
}

func TestStorySelectedOnlyWhenIndexMoves(t *testing.T) {
	stories := []entity.Story{
		{ID: "story-a", Name: "Alpha"},
		{ID: "story-b", Name: "Beta"},
	}
	tests := []struct {
		name          string
		currentIndex  int
		action        func(context.Context, *entity.Room, *entity.Client) error
		wantErr       bool
		wantStoryID   string
		wantStoryName string
	}{
		{
			name:         "select",
			currentIndex: 0,
			action: func(ctx context.Context, room *entity.Room, owner *entity.Client) error {
				return room.SelectStory(ctx, owner.ID, "story-b")
			},
			wantStoryID:   "story-b",
			wantStoryName: "Beta",
		},
		{
			name:         "select current story",
			currentIndex: 0,
			action: func(ctx context.Context, room *entity.Room, owner *entity.Client) error {
				return room.SelectStory(ctx, owner.ID, "story-a")
			},
			wantErr: true,
		},
		{
			name:         "advance",
			currentIndex: 0,
			action: func(ctx context.Context, room *entity.Room, owner *entity.Client) error {
				return room.AdvanceToNextStory(ctx, owner.ID)
			},
			wantStoryID:   "story-b",
			wantStoryName: "Beta",
		},
		{
			name:         "advance blocked",
			currentIndex: 1,
			action: func(ctx context.Context, room *entity.Room, owner *entity.Client) error {
				return room.AdvanceToNextStory(ctx, owner.ID)
			},
		},
		{
			name:         "previous",
			currentIndex: 1,
			action: func(ctx context.Context, room *entity.Room, owner *entity.Client) error {
				return room.PrevStory(ctx, owner.ID)
			},
			wantStoryID:   "story-a",
			wantStoryName: "Alpha",
		},
		{
			name:         "previous blocked",
			currentIndex: 0,
			action: func(ctx context.Context, room *entity.Room, owner *entity.Client) error {
				return room.PrevStory(ctx, owner.ID)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			room, owner, _ := newEventRoom(t, true, stories, tt.currentIndex)

			err := tt.action(context.Background(), room, owner)
			if tt.wantErr && err == nil {
				t.Fatal("action returned no error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("action returned error: %v", err)
			}

			events := room.PullEvents()
			if tt.wantStoryID == "" {
				if len(events) != 0 {
					t.Fatalf("recorded events = %d, want 0", len(events))
				}
				return
			}
			if len(events) != 1 {
				t.Fatalf("recorded events = %d, want 1", len(events))
			}
			selected, ok := events[0].(entity.StorySelected)
			if !ok {
				t.Fatalf("event has type %T, want entity.StorySelected", events[0])
			}
			if selected.RoomID != eventTestRoomID {
				t.Fatalf("StorySelected room = %q, want %q", selected.RoomID, eventTestRoomID)
			}
			if selected.StoryID != tt.wantStoryID || selected.StoryName != tt.wantStoryName {
				t.Fatalf("StorySelected story = %q/%q, want %q/%q",
					selected.StoryID, selected.StoryName, tt.wantStoryID, tt.wantStoryName)
			}
		})
	}
}

func TestStoryEstimatedWhenBacklogRevealStampsStory(t *testing.T) {
	room, owner, participant := newEventRoom(t, true, []entity.Story{{ID: "story-a", Name: "Alpha"}}, 0)
	ctx := context.Background()

	if err := room.Vote(ctx, owner.ID, entity.VotePtr("5")); err != nil {
		t.Fatalf("owner Vote returned error: %v", err)
	}
	if err := room.Vote(ctx, participant.ID, entity.VotePtr("8")); err != nil {
		t.Fatalf("participant Vote returned error: %v", err)
	}

	events := room.PullEvents()
	if len(events) != 4 {
		t.Fatalf("recorded events = %d, want 4", len(events))
	}
	if events[2].Name() != "votes-revealed" || events[3].Name() != "story-estimated" {
		t.Fatalf("reveal events = %q, %q, want votes-revealed then story-estimated",
			events[2].Name(), events[3].Name())
	}
	estimated, ok := events[3].(entity.StoryEstimated)
	if !ok {
		t.Fatalf("event has type %T, want entity.StoryEstimated", events[3])
	}
	if estimated.RoomID != eventTestRoomID || estimated.StoryID != "story-a" {
		t.Fatalf("StoryEstimated identifiers = %+v", estimated)
	}
	if estimated.Result == nil || *estimated.Result != 6.5 {
		t.Fatalf("StoryEstimated result = %v, want 6.5", estimated.Result)
	}

	stamped := room.Stories()[0]
	if stamped.Result == nil || *stamped.Result != 6.5 || !stamped.Voted {
		t.Fatalf("current story was not stamped: %+v", stamped)
	}
	if room.Outcome() != nil && estimated.Result == room.Outcome().Result {
		t.Fatal("StoryEstimated shares the live result pointer")
	}
}

func TestStoryEstimatedIsNotRecordedWithoutCurrentStory(t *testing.T) {
	room, owner, participant := newEventRoom(t, true, nil, 0)
	ctx := context.Background()

	if err := room.Vote(ctx, owner.ID, entity.VotePtr("5")); err != nil {
		t.Fatalf("owner Vote returned error: %v", err)
	}
	if err := room.Vote(ctx, participant.ID, entity.VotePtr("8")); err != nil {
		t.Fatalf("participant Vote returned error: %v", err)
	}

	for _, event := range room.PullEvents() {
		if event.Name() == "story-estimated" {
			t.Fatal("reveal without a current story recorded story-estimated")
		}
	}
}

func TestPullEventsDrainsAndResetsBuffer(t *testing.T) {
	room, _, participant := newEventRoom(t, true, nil, 0)
	ctx := context.Background()

	if events := room.PullEvents(); len(events) != 0 {
		t.Fatalf("room without events returned %d events", len(events))
	}

	if err := room.Vote(ctx, participant.ID, entity.VotePtr("3")); err != nil {
		t.Fatalf("first Vote returned error: %v", err)
	}
	if err := room.Vote(ctx, participant.ID, entity.VotePtr("5")); err != nil {
		t.Fatalf("second Vote returned error: %v", err)
	}

	first := room.PullEvents()
	if len(first) != 2 {
		t.Fatalf("pulled events = %d, want 2", len(first))
	}
	if events := room.PullEvents(); len(events) != 0 {
		t.Fatalf("second pull returned %d events, want 0", len(events))
	}

	if err := room.Vote(ctx, participant.ID, entity.VotePtr("8")); err != nil {
		t.Fatalf("third Vote returned error: %v", err)
	}
	if len(first) != 2 {
		t.Fatalf("pulled slice was reused: length = %d, want 2", len(first))
	}
	if events := room.PullEvents(); len(events) != 1 {
		t.Fatalf("pulled events after reset = %d, want 1", len(events))
	}
}

func TestEventBufferDropsOldestEvents(t *testing.T) {
	const clientCount = 300

	room := entity.NewRoomWithID(eventTestRoomID, clientcollection.New())
	ctx := context.Background()

	clients := make([]*entity.Client, 0, clientCount)
	for i := 0; i < clientCount; i++ {
		clients = append(clients, room.NewClient(fmt.Sprintf("client-%03d", i)))
	}
	for _, client := range clients {
		if err := room.Vote(ctx, client.ID, entity.VotePtr("1")); err != nil {
			t.Fatalf("Vote for %s returned error: %v", client.ID, err)
		}
	}

	events := room.PullEvents()
	if len(events) != eventBufferCap {
		t.Fatalf("buffered events = %d, want %d", len(events), eventBufferCap)
	}

	// 300 vote-cast events plus the quorum reveal are trimmed to the newest
	// 256, dropping the 45 oldest vote-cast events.
	first, ok := events[0].(entity.VoteCast)
	if !ok {
		t.Fatalf("first buffered event has type %T, want entity.VoteCast", events[0])
	}
	if first.ClientID != "client-045" {
		t.Fatalf("oldest buffered event = %q, want client-045", first.ClientID)
	}
	if last := events[len(events)-1]; last.Name() != "votes-revealed" {
		t.Fatalf("newest buffered event = %q, want votes-revealed", last.Name())
	}
}

func TestSnapshotDoesNotCarryEvents(t *testing.T) {
	room, _, participant := newEventRoom(t, true, nil, 0)

	if err := room.Vote(context.Background(), participant.ID, entity.VotePtr("5")); err != nil {
		t.Fatalf("Vote returned error: %v", err)
	}
	if len(room.PullEvents()) != 1 {
		t.Fatal("Vote did not record an event")
	}

	if err := room.Vote(context.Background(), participant.ID, entity.VotePtr("8")); err != nil {
		t.Fatalf("second Vote returned error: %v", err)
	}

	restored := entity.NewRoomFromSnapshot(room.Snapshot(), clientcollection.New())
	if events := restored.PullEvents(); len(events) != 0 {
		t.Fatalf("restored room carries %d events, want 0", len(events))
	}
}
