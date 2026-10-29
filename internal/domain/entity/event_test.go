package entity

import (
	"context"
	"testing"
	"time"
)

// eventTestClientCollection is a minimal in-memory ClientCollection used by
// the event tests so they do not depend on infrastructure adapters.
type eventTestClientCollection struct {
	clients []*Client
}

func (c *eventTestClientCollection) Add(client *Client) {
	c.clients = append(c.clients, client)
}

func (c *eventTestClientCollection) Remove(clientID string) {
	for index, client := range c.clients {
		if client.ID == ClientID(clientID) {
			c.clients = append(c.clients[:index], c.clients[index+1:]...)
			return
		}
	}
}

func (c *eventTestClientCollection) Count() int {
	return len(c.clients)
}

func (c *eventTestClientCollection) First() (*Client, bool) {
	if len(c.clients) == 0 {
		return nil, false
	}
	return c.clients[0], true
}

func (c *eventTestClientCollection) ForEach(f func(client *Client)) {
	for _, client := range c.clients {
		f(client)
	}
}

func (c *eventTestClientCollection) Filter(match func(client *Client) bool) ClientCollection {
	filtered := &eventTestClientCollection{}
	for _, client := range c.clients {
		if match(client) {
			filtered.clients = append(filtered.clients, client)
		}
	}
	return filtered
}

func (c *eventTestClientCollection) Values() []*Client {
	return c.clients
}

func newEventTestRoom(clients ...*Client) *Room {
	room := NewRoomWithID("room-events", &eventTestClientCollection{})
	for _, client := range clients {
		room.Clients.Add(client)
	}
	return room
}

// eventsOf type-filters buffered events by their concrete value type.
func eventsOf[T Event](events []Event) []T {
	matched := make([]T, 0, len(events))
	for _, event := range events {
		if typed, ok := event.(T); ok {
			matched = append(matched, typed)
		}
	}
	return matched
}

func assertOccurredAtUTC(t *testing.T, occurredAt time.Time) {
	t.Helper()

	if occurredAt.IsZero() {
		t.Fatal("OccurredAt is zero")
	}
	if occurredAt.Location() != time.UTC {
		t.Fatalf("OccurredAt location = %v, want UTC", occurredAt.Location())
	}
}

func TestRoomPullEventsDrainsBuffer(t *testing.T) {
	room := newEventTestRoom()

	if events := room.PullEvents(); events != nil {
		t.Fatalf("PullEvents() on empty buffer = %v, want nil", events)
	}

	room.recordEvent(VoteCast{RoomID: room.ID, ClientID: cid("client1"), Vote: Vote("5")})
	room.recordEvent(StorySelected{RoomID: room.ID, StoryID: sid("story-1")})

	first := room.PullEvents()
	if len(first) != 2 {
		t.Fatalf("first PullEvents() = %d events, want 2", len(first))
	}
	if first[0].Name() != eventNameVoteCast {
		t.Errorf("first event name = %q, want %q", first[0].Name(), eventNameVoteCast)
	}
	if first[1].Name() != eventNameStorySelected {
		t.Errorf("second event name = %q, want %q", first[1].Name(), eventNameStorySelected)
	}

	if second := room.PullEvents(); second != nil {
		t.Fatalf("second PullEvents() = %v, want nil", second)
	}
	if room.events != nil {
		t.Fatalf("buffer after drain = %v, want nil", room.events)
	}
}

func TestRoomVoteEmitsVoteCast(t *testing.T) {
	tests := []struct {
		name      string
		vote      *Vote
		revealed  bool
		wantEvent bool
	}{
		{name: "numeric vote emits vote cast", vote: VotePtr("5"), wantEvent: true},
		{name: "empty vote emits nothing", vote: VotePtr("")},
		{name: "nil vote emits nothing", vote: nil},
		{name: "vote while revealed emits nothing", vote: VotePtr("5"), revealed: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			room := newEventTestRoom(
				&Client{ID: cid("client1")},
				&Client{ID: cid("client2")},
			)
			room.Reveal = test.revealed

			if err := room.Vote(context.Background(), cid("client1"), test.vote); err != nil {
				t.Fatalf("Vote() error = %v", err)
			}

			events := room.PullEvents()
			casts := eventsOf[VoteCast](events)

			if !test.wantEvent {
				if len(events) != 0 {
					t.Fatalf("events = %v, want none", events)
				}
				return
			}

			if len(casts) != 1 {
				t.Fatalf("VoteCast events = %d, want 1", len(casts))
			}
			cast := casts[0]
			if cast.RoomID != room.ID {
				t.Errorf("RoomID = %q, want %q", cast.RoomID, room.ID)
			}
			if cast.ClientID != cid("client1") {
				t.Errorf("ClientID = %q, want client1", cast.ClientID)
			}
			if cast.Vote != Vote("5") {
				t.Errorf("Vote = %q, want 5", cast.Vote)
			}
			assertOccurredAtUTC(t, cast.OccurredAt)
		})
	}
}

func TestRoomRevealEmitsVotesRevealed(t *testing.T) {
	client := &Client{ID: cid("client1"), CurrentVote: VotePtr("5"), HasVoted: true}
	room := newEventTestRoom(client)

	room.reveal(true)

	revealed := eventsOf[VotesRevealed](room.PullEvents())
	if len(revealed) != 1 {
		t.Fatalf("VotesRevealed events = %d, want 1", len(revealed))
	}

	event := revealed[0]
	if event.RoomID != room.ID {
		t.Errorf("RoomID = %q, want %q", event.RoomID, room.ID)
	}
	if event.Consensus != room.Consensus {
		t.Errorf("Consensus = %q, want %q", event.Consensus, room.Consensus)
	}
	if event.Result == nil || *event.Result != 5 {
		t.Fatalf("Result = %v, want 5", event.Result)
	}
	if event.Result == room.Result {
		t.Error("Result aliases room state, want a clone")
	}
	assertOccurredAtUTC(t, event.OccurredAt)
}

func TestRoomRevealEmitsVotesRevealedOnce(t *testing.T) {
	client := &Client{ID: cid("client1"), CurrentVote: VotePtr("5"), HasVoted: true}
	room := newEventTestRoom(client)

	room.reveal(true)
	room.reveal(true)

	events := room.PullEvents()
	if revealed := eventsOf[VotesRevealed](events); len(revealed) != 1 {
		t.Fatalf("VotesRevealed events = %d, want 1", len(revealed))
	}
	if estimated := eventsOf[StoryEstimated](events); len(estimated) != 0 {
		t.Fatalf("StoryEstimated events = %d, want 0", len(estimated))
	}
}

func TestRoomRevealEmitsStoryEstimated(t *testing.T) {
	client := &Client{ID: cid("client1"), CurrentVote: VotePtr("5"), HasVoted: true}
	room := newEventTestRoom(client)
	room.Stories = []Story{{ID: sid("story-1"), Name: "First story"}}
	room.CurrentStoryIndex = 0
	room.reveal(true)

	events := room.PullEvents()
	if revealed := eventsOf[VotesRevealed](events); len(revealed) != 1 {
		t.Fatalf("VotesRevealed events = %d, want 1", len(revealed))
	}

	estimated := eventsOf[StoryEstimated](events)
	if len(estimated) != 1 {
		t.Fatalf("StoryEstimated events = %d, want 1", len(estimated))
	}

	event := estimated[0]
	if event.RoomID != room.ID {
		t.Errorf("RoomID = %q, want %q", event.RoomID, room.ID)
	}
	if event.StoryID != sid("story-1") {
		t.Errorf("StoryID = %q, want story-1", event.StoryID)
	}
	if event.Result == nil || *event.Result != 5 {
		t.Fatalf("Result = %v, want 5", event.Result)
	}
	if event.Result == room.Result {
		t.Error("Result aliases room state, want a clone")
	}
	if event.Result == room.Stories[0].Result {
		t.Error("Result aliases the story result, want a clone")
	}
	assertOccurredAtUTC(t, event.OccurredAt)
}

func TestRoomSelectStoryEmitsStorySelected(t *testing.T) {
	owner := &Client{ID: cid("owner"), IsOwner: true}
	room := newEventTestRoom(owner)
	room.Stories = []Story{
		{ID: sid("story-1"), Name: "First story"},
		{ID: sid("story-2"), Name: "Second story"},
	}
	room.CurrentStoryIndex = 0

	if err := room.SelectStory(context.Background(), cid("owner"), sid("story-2")); err != nil {
		t.Fatalf("SelectStory() error = %v", err)
	}

	events := room.PullEvents()
	selected := eventsOf[StorySelected](events)
	if len(selected) != 1 {
		t.Fatalf("StorySelected events = %d, want 1", len(selected))
	}

	event := selected[0]
	if event.RoomID != room.ID {
		t.Errorf("RoomID = %q, want %q", event.RoomID, room.ID)
	}
	if event.StoryID != sid("story-2") {
		t.Errorf("StoryID = %q, want story-2", event.StoryID)
	}
	assertOccurredAtUTC(t, event.OccurredAt)

	if revealed := eventsOf[VotesRevealed](events); len(revealed) != 0 {
		t.Fatalf("VotesRevealed events = %d, want 0", len(revealed))
	}
}

func TestRoomSelectStoryRejectedEmitsNothing(t *testing.T) {
	tests := []struct {
		name     string
		clientID ClientID
		storyID  StoryID
	}{
		{name: "already current story", clientID: cid("owner"), storyID: sid("story-1")},
		{name: "unknown story", clientID: cid("owner"), storyID: sid("missing")},
		{name: "empty story id", clientID: cid("owner"), storyID: sid("")},
		{name: "non owner", clientID: cid("guest"), storyID: sid("story-2")},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			owner := &Client{ID: cid("owner"), IsOwner: true}
			guest := &Client{ID: cid("guest")}
			room := newEventTestRoom(owner, guest)
			room.Stories = []Story{
				{ID: sid("story-1"), Name: "First story"},
				{ID: sid("story-2"), Name: "Second story"},
			}
			room.CurrentStoryIndex = 0

			if err := room.SelectStory(context.Background(), test.clientID, test.storyID); err == nil {
				t.Fatal("SelectStory() error = nil, want rejection")
			}
			if events := room.PullEvents(); len(events) != 0 {
				t.Fatalf("events = %v, want none", events)
			}
		})
	}
}

func TestRoomPullEventsReturnsCopy(t *testing.T) {
	room := newEventTestRoom()
	room.recordEvent(VoteCast{RoomID: room.ID, ClientID: cid("client1"), Vote: Vote("5")})

	events := room.PullEvents()
	if len(events) != 1 {
		t.Fatalf("PullEvents() = %d events, want 1", len(events))
	}

	events[0] = nil
	if room.events != nil {
		t.Fatalf("mutating the pulled slice changed the room buffer: %v", room.events)
	}
	if again := room.PullEvents(); again != nil {
		t.Fatalf("second PullEvents() = %v, want nil", again)
	}
}

func TestEventNames(t *testing.T) {
	tests := []struct {
		name  string
		event Event
		want  string
	}{
		{name: "vote cast", event: VoteCast{}, want: "vote-cast"},
		{name: "votes revealed", event: VotesRevealed{}, want: "votes-revealed"},
		{name: "story selected", event: StorySelected{}, want: "story-selected"},
		{name: "story estimated", event: StoryEstimated{}, want: "story-estimated"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := test.event.Name(); got != test.want {
				t.Errorf("Name() = %q, want %q", got, test.want)
			}
		})
	}
}
