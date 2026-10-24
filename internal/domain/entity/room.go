package entity

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/google/uuid"
)

type (
	Room struct {
		id        string
		startedAt time.Time
		presence  *Presence
		round     *VotingRound

		DeckType          DeckType
		BacklogMode       bool
		Stories           []Story
		CurrentStoryIndex int
		RoomVersion       uint64
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
		id:          id,
		startedAt:   startedAt.UTC(),
		presence:    NewPresence(id, clients),
		round:       NewVotingRound(),
		DeckType:    DeckTypeFibonacci,
		BacklogMode: true,
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
	room.DeckType = deckType
	return room
}

// ID returns the room's stable identifier.
func (r *Room) ID() string {
	return r.id
}

// Clients returns the room's client collection.
func (r *Room) Clients() ClientCollection {
	return r.presence.Clients()
}

// Deck returns the room's immutable deck descriptor. Unknown or empty deck
// types resolve to Fibonacci so repository tests and zero-value rooms stay
// usable.
func (r *Room) Deck() Deck {
	return deckForType(r.DeckType)
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

func (r *Room) ToggleBacklogMode(ctx context.Context, clientID string) error {
	if _, err := r.presence.RequireOwner(clientID, "toggle backlog mode"); err != nil {
		return err
	}

	if !r.BacklogMode {
		r.BacklogMode = true
		if currentStory := r.round.CurrentStoryName(); currentStory != "" {
			r.Stories = []Story{{ID: uuid.NewString(), Name: currentStory}}
			r.CurrentStoryIndex = 0
		}
	} else {
		r.BacklogMode = false
		if name := r.getCurrentStoryName(); name != "" {
			r.round.SetCurrentStoryName(name)
		}
		r.Stories = nil
		r.CurrentStoryIndex = 0
	}
	return nil
}

func (r *Room) AddStory(ctx context.Context, clientID string, name string) error {
	if _, err := r.presence.RequireOwner(clientID, "add a story"); err != nil {
		return err
	}

	if !r.BacklogMode {
		r.BacklogMode = true
	}

	r.Stories = append(r.Stories, Story{ID: uuid.NewString(), Name: name})
	if len(r.Stories) == 1 {
		r.CurrentStoryIndex = 0
	}
	return nil
}

func (r *Room) RemoveStory(ctx context.Context, clientID string, storyID string) error {
	if _, err := r.presence.RequireOwner(clientID, "remove a story"); err != nil {
		return err
	}
	if storyID == "" {
		return fmt.Errorf("story ID cannot be empty")
	}
	index := r.storyIndexByID(storyID)
	if index == -1 {
		return fmt.Errorf("story %s not found in room %s", storyID, r.id)
	}

	if index == r.CurrentStoryIndex {
		if len(r.Stories) == 1 {
			r.CurrentStoryIndex = 0
			r.Stories = nil
			r.reveal(false)
			r.presence.ClearVotes()
			return nil
		} else if index == len(r.Stories)-1 {
			r.CurrentStoryIndex--
		}
		r.reveal(false)
		r.presence.ClearVotes()
	} else if index < r.CurrentStoryIndex {
		r.CurrentStoryIndex--
	}

	r.Stories = append(r.Stories[:index], r.Stories[index+1:]...)
	return nil
}

// SelectStory selects a pending story and starts a fresh voting round.
func (r *Room) SelectStory(ctx context.Context, clientID string, storyID string) error {
	if _, err := r.presence.RequireOwner(clientID, "select a story"); err != nil {
		return err
	}
	if storyID == "" {
		return fmt.Errorf("story ID cannot be empty")
	}

	index := r.storyIndexByID(storyID)
	if index == -1 {
		return fmt.Errorf("story %s not found in room %s", storyID, r.id)
	}
	if index == r.CurrentStoryIndex {
		return fmt.Errorf("story %s is already the current story", storyID)
	}
	if r.Stories[index].Voted {
		return fmt.Errorf("estimated story %s cannot be selected", storyID)
	}

	r.CurrentStoryIndex = index
	r.reveal(false)
	r.presence.ClearVotes()
	return nil
}

// ReorderStory moves a story to targetIndex. The current story remains
// identified by its stable ID.
func (r *Room) ReorderStory(ctx context.Context, clientID string, storyID string, targetIndex int) error {
	storyIndex, err := r.validateReorderStory(clientID, storyID, targetIndex)
	if err != nil {
		return err
	}
	if storyIndex == targetIndex {
		return nil
	}

	currentStoryID := r.currentStoryID()
	story := r.Stories[storyIndex]
	r.Stories = append(r.Stories[:storyIndex], r.Stories[storyIndex+1:]...)
	r.Stories = slices.Insert(r.Stories, targetIndex, story)
	r.updateCurrentStoryIndex(storyIndex, targetIndex, currentStoryID)
	return nil
}

func (r *Room) validateReorderStory(clientID string, storyID string, targetIndex int) (int, error) {
	if _, err := r.presence.RequireOwner(clientID, "reorder stories"); err != nil {
		return 0, err
	}
	if storyID == "" {
		return 0, fmt.Errorf("story ID cannot be empty")
	}
	if targetIndex < 0 || targetIndex >= len(r.Stories) {
		return 0, fmt.Errorf("target story index %d out of range", targetIndex)
	}

	storyIndex := r.storyIndexByID(storyID)
	if storyIndex == -1 {
		return 0, fmt.Errorf("story %s not found in room %s", storyID, r.id)
	}

	return storyIndex, nil
}

func (r *Room) currentStoryID() string {
	if r.CurrentStoryIndex < 0 || r.CurrentStoryIndex >= len(r.Stories) {
		return ""
	}
	return r.Stories[r.CurrentStoryIndex].ID
}

func (r *Room) updateCurrentStoryIndex(storyIndex, targetIndex int, currentStoryID string) {
	if currentStoryID != "" {
		r.CurrentStoryIndex = r.storyIndexByID(currentStoryID)
		return
	}
	if r.CurrentStoryIndex == storyIndex {
		r.CurrentStoryIndex = targetIndex
		return
	}
	if storyIndex < r.CurrentStoryIndex {
		r.CurrentStoryIndex--
	}
	if targetIndex <= r.CurrentStoryIndex {
		r.CurrentStoryIndex++
	}
}

func (r *Room) storyIndexByID(storyID string) int {
	for index, story := range r.Stories {
		if story.ID == storyID {
			return index
		}
	}
	return -1
}

func (r *Room) AdvanceToNextStory(ctx context.Context, clientID string) error {
	if _, err := r.presence.RequireOwner(clientID, "advance to the next story"); err != nil {
		return err
	}

	if r.CurrentStoryIndex < len(r.Stories)-1 {
		r.CurrentStoryIndex++
		r.reveal(false)
		r.presence.ClearVotes()
	}
	return nil
}

func (r *Room) PrevStory(ctx context.Context, clientID string) error {
	if _, err := r.presence.RequireOwner(clientID, "go to the previous story"); err != nil {
		return err
	}

	if r.CurrentStoryIndex > 0 {
		r.CurrentStoryIndex--
		r.reveal(false)
		r.presence.ClearVotes()
	}
	return nil
}

func (r *Room) EffectiveCurrentStory() string {
	if r.BacklogMode && len(r.Stories) > 0 && r.CurrentStoryIndex < len(r.Stories) {
		return r.Stories[r.CurrentStoryIndex].Name
	}
	return r.round.CurrentStoryName()
}

func (r *Room) getCurrentStoryName() string {
	if len(r.Stories) > 0 && r.CurrentStoryIndex < len(r.Stories) {
		return r.Stories[r.CurrentStoryIndex].Name
	}
	return ""
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

func (r *Room) CountOwners() int {
	return r.presence.OwnerCount()
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

func (r *Room) ToggleOwner(ctx context.Context, clientID string, targetClientID string) error {
	return r.presence.ToggleOwner(ctx, clientID, targetClientID)
}

// AdminToggleOwner toggles a client's owner status without checking
// that the caller is an owner
func (r *Room) AdminToggleOwner(ctx context.Context, targetClientID string) error {
	return r.presence.AdminToggleOwner(ctx, targetClientID)
}

func (r *Room) SetCurrentStory(ctx context.Context, clientID string, story string) error {
	if _, err := r.presence.RequireOwner(clientID, "set the current story"); err != nil {
		return err
	}

	if r.BacklogMode && r.CurrentStoryIndex >= 0 && r.CurrentStoryIndex < len(r.Stories) {
		r.Stories[r.CurrentStoryIndex].Name = story
	} else {
		r.round.SetCurrentStoryName(story)
	}
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

	r.stampCurrentStory()
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

func (r *Room) stampCurrentStory() {
	if !r.BacklogMode || r.CurrentStoryIndex < 0 || r.CurrentStoryIndex >= len(r.Stories) {
		return
	}

	outcome := r.round.Outcome()
	if outcome == nil {
		return
	}

	story := &r.Stories[r.CurrentStoryIndex]
	story.Result = outcome.Result
	story.MostAppearingVotes = outcome.MostAppearingVotes
	story.Voted = true
}

// Revealed reports whether the current round's votes are visible.
func (r *Room) Revealed() bool {
	return r.round.Revealed()
}

// Outcome returns the current round's outcome, or nil while the round is
// hidden.
func (r *Room) Outcome() *RoundOutcome {
	return r.round.Outcome()
}

// Result returns the mean of the numeric votes, or nil when unavailable.
func (r *Room) Result() *float32 {
	if outcome := r.round.Outcome(); outcome != nil {
		return outcome.Result
	}

	return nil
}

// MostAppearingVotes returns the most frequent votes ordered by deck position.
func (r *Room) MostAppearingVotes() []string {
	if outcome := r.round.Outcome(); outcome != nil {
		return outcome.MostAppearingVotes
	}

	return nil
}

// Consensus returns the consensus level of the current round.
func (r *Room) Consensus() string {
	if outcome := r.round.Outcome(); outcome != nil {
		return outcome.Consensus
	}

	return ""
}

// LowestVote returns the lowest vote by deck position.
func (r *Room) LowestVote() *string {
	if outcome := r.round.Outcome(); outcome != nil {
		return outcome.LowestVote
	}

	return nil
}

// HighestVote returns the highest vote by deck position.
func (r *Room) HighestVote() *string {
	if outcome := r.round.Outcome(); outcome != nil {
		return outcome.HighestVote
	}

	return nil
}

// VoteRange returns the numeric distance between the extreme votes.
func (r *Room) VoteRange() *int {
	if outcome := r.round.Outcome(); outcome != nil {
		return outcome.VoteRange
	}

	return nil
}

// VoteSpread returns the deck-position distance between the extreme votes.
func (r *Room) VoteSpread() *int {
	if outcome := r.round.Outcome(); outcome != nil {
		return outcome.VoteSpread
	}

	return nil
}

// SpecialVoteCount returns how many special cards were cast.
func (r *Room) SpecialVoteCount() int {
	if outcome := r.round.Outcome(); outcome != nil {
		return outcome.SpecialVoteCount
	}

	return 0
}

func (r *Room) IsEmpty() bool {
	return r.presence.IsEmpty()
}

func (r *Room) FindClient(clientID string) (*Client, bool) {
	return r.presence.Find(clientID)
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
	r.checkReveal()

	return nil
}

func (r *Room) UpdateClientName(ctx context.Context, clientID string, name string) error {
	client, err := r.presence.Require(clientID)
	if err != nil {
		return err
	}

	client.UpdateName(ctx, name)

	return nil
}
