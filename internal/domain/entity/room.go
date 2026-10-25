package entity

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// eventBufferLimit caps the number of domain events retained on a room.
const eventBufferLimit = 256

type (
	Room struct {
		id        string
		startedAt time.Time
		presence  *Presence
		round     *VotingRound
		backlog   *Backlog
		events    []Event

		deckType DeckType
		version  RoomVersion
	}
)

func NewRoom(clients ClientCollection) *Room {
	return NewRoomWithID(uuid.NewString(), clients)
}

func NewRoomWithID(id string, clients ClientCollection) *Room {
	return NewRoomWithIDAndStartedAt(id, clients, time.Now().UTC())
}

// NewRoomWithIDAndStartedAt creates a room with a persisted start time.
func NewRoomWithIDAndStartedAt(id string, clients ClientCollection, startedAt time.Time) *Room {
	return &Room{
		id:        id,
		startedAt: startedAt.UTC(),
		presence:  NewPresence(id, clients),
		round:     NewVotingRound(),
		backlog:   NewBacklog(id),
		deckType:  DeckTypeFibonacci,
	}
}

// NewRoomWithDeck creates a room fixed to the given voting deck.
func NewRoomWithDeck(clients ClientCollection, deckType DeckType) *Room {
	return NewRoomWithIDAndDeck(uuid.NewString(), clients, deckType)
}

// NewRoomWithIDAndDeck creates a room with an explicit ID fixed to the given
// voting deck.
func NewRoomWithIDAndDeck(id string, clients ClientCollection, deckType DeckType) *Room {
	room := NewRoomWithIDAndStartedAt(id, clients, time.Now().UTC())
	room.deckType = deckType
	return room
}

// ID returns the room's stable identifier.
func (r *Room) ID() string {
	return r.id
}

// Version returns the aggregate's optimistic-concurrency revision.
func (r *Room) Version() RoomVersion {
	return r.version
}

// SetVersion applies a revision produced by a persistence adapter. Domain
// operations must never call it; only hubs bump the version after a successful
// save.
func (r *Room) SetVersion(v RoomVersion) {
	r.version = v
}

// PullEvents returns the buffered domain events in recording order and resets
// the buffer. Callers own the returned slice.
func (r *Room) PullEvents() []Event {
	events := r.events
	r.events = nil
	return events
}

// recordEvent appends a domain event, dropping the oldest buffered events so
// the buffer never exceeds eventBufferLimit.
func (r *Room) recordEvent(event Event) {
	r.events = append(r.events, event)
	if len(r.events) > eventBufferLimit {
		r.events = r.events[len(r.events)-eventBufferLimit:]
	}
}

// recordStorySelected records a StorySelected event for the room's current
// story. Callers must only invoke it after the current story index moved.
func (r *Room) recordStorySelected() {
	story, ok := r.backlog.Current()
	if !ok {
		return
	}

	r.recordEvent(StorySelected{
		RoomID:    r.id,
		StoryID:   story.ID,
		StoryName: story.Name,
		At:        time.Now().UTC(),
	})
}

// Clients returns the room's client collection.
func (r *Room) Clients() ClientCollection {
	return r.presence.Clients()
}

// DeckType returns the room's configured deck type.
func (r *Room) DeckType() DeckType {
	return r.deckType
}

// Deck returns the room's immutable deck descriptor. Unknown or empty deck
// types resolve to Fibonacci so repository tests and zero-value rooms stay
// usable.
func (r *Room) Deck() Deck {
	return deckForType(r.deckType)
}

// StartedAt returns the room start time as a value copy.
func (r *Room) StartedAt() time.Time {
	return r.startedAt
}

func (r *Room) NewClient(id string) *Client {
	return r.presence.Add(id)
}

func (r *Room) RemoveClient(ctx context.Context, clientID string) error {
	r.presence.Remove(clientID)
	r.checkReveal()

	return nil
}

func (r *Room) ResetVoting(ctx context.Context, clientID string) error {
	if _, err := r.presence.RequireOwner(clientID, "reset voting"); err != nil {
		return err
	}

	r.reveal(false)

	r.presence.ClearVotes()
	return nil
}

func (r *Room) checkReveal() {
	if r.presence.AllActiveVoted() {
		r.reveal(true)
	}
}

func (r *Room) ToggleSpectator(ctx context.Context, clientID string, targetClientID string) error {
	if err := r.presence.ToggleSpectator(ctx, clientID, targetClientID); err != nil {
		return err
	}

	// Votes are only cleared while the round is hidden, mirroring the previous
	// Client.Vote behaviour.
	if target, ok := r.presence.Find(targetClientID); ok && !r.round.Revealed() {
		r.presence.SetVote(target, nil)
	}

	r.checkReveal()
	return nil
}

func (r *Room) ToggleReveal(ctx context.Context, clientID string) error {
	if _, err := r.presence.RequireOwner(clientID, "toggle reveal"); err != nil {
		return err
	}

	r.reveal(!r.round.Revealed())
	return nil
}

// reveal orchestrates the round transition: hiding always clears the outcome,
// while revealing computes it once from the votes currently recorded in the
// presence and stamps the estimate onto the current backlog story.
func (r *Room) reveal(reveal bool) {
	if !reveal {
		r.round.Hide()
		return
	}

	deck := r.Deck()
	votes := r.currentVotes()
	if !r.round.Reveal(func() *RoundOutcome {
		return computeOutcome(deck, votes)
	}) {
		return
	}

	outcome := r.round.Outcome()
	r.recordEvent(VotesRevealed{
		RoomID:  r.id,
		Outcome: outcome.clone(),
		At:      time.Now().UTC(),
	})

	if !r.backlog.Mode() {
		return
	}

	r.backlog.ApplyOutcome(outcome)
	if story, ok := r.backlog.Current(); ok {
		r.recordEvent(StoryEstimated{
			RoomID:  r.id,
			StoryID: story.ID,
			Result:  cloneFloat32(story.Result),
			At:      time.Now().UTC(),
		})
	}
}

// currentVotes returns the non-spectator, non-empty votes in client iteration
// order.
func (r *Room) currentVotes() []Vote {
	votes := make([]Vote, 0, r.presence.Count())
	for _, client := range r.presence.Values() {
		if client.IsSpectator || client.CurrentVote == nil || client.CurrentVote.IsEmpty() {
			continue
		}
		votes = append(votes, *client.CurrentVote)
	}

	return votes
}

func (r *Room) Vote(ctx context.Context, clientID string, vote *Vote) error {
	client, err := r.presence.Require(clientID)
	if err != nil {
		return err
	}

	if vote != nil && !vote.IsEmpty() {
		deck := r.Deck()
		if !deck.HasCard(vote.Label()) {
			return fmt.Errorf("vote %q is not part of deck %s", vote.Label(), deck.ID)
		}
	}

	if r.round.Revealed() {
		r.checkReveal()
		return nil
	}

	r.presence.SetVote(client, vote)
	r.recordEvent(VoteCast{
		RoomID:   r.id,
		ClientID: clientID,
		Vote:     vote,
		At:       time.Now().UTC(),
	})
	r.checkReveal()

	return nil
}
