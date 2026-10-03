package entity

//go:generate go tool mockgen -destination mocks.go -package entity . ClientCollection

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/samber/lo"

	"planning-poker/internal/domain/domainerror"
)

type (
	ClientCollection interface {
		Add(client *Client)
		Remove(clientID string)
		Count() int
		First() (*Client, bool)
		ForEach(f func(client *Client))
		Filter(f func(client *Client) bool) ClientCollection
		Values() []*Client
	}

	Room struct {
		ID                 string
		startedAt          time.Time
		Clients            ClientCollection
		DeckType           DeckType
		CurrentStory       string
		Reveal             bool
		Result             *float32
		MostAppearingVotes []string
		Consensus          string
		LowestVote         *string
		HighestVote        *string
		VoteRange          *int
		VoteSpread         *int
		SpecialVoteCount   int
		BacklogMode        bool
		Stories            []Story
		CurrentStoryIndex  int
		RoomVersion        uint64
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
		ID:           id,
		startedAt:    startedAt.UTC(),
		Clients:      clients,
		DeckType:     DeckTypeFibonacci,
		CurrentStory: "",
		Reveal:       false,
		Result:       nil,
		BacklogMode:  true,
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
	client := newClient(id)
	r.Clients.Add(client)
	client.room = r

	if r.Clients.Count() == 1 {
		client.IsOwner = true
	}

	return client
}

func (r *Room) RemoveClient(ctx context.Context, clientID string) error {
	r.Clients.Remove(clientID)

	if r.CountOwners() == 0 && r.Clients.Count() > 0 {
		if client, ok := r.Clients.First(); ok {
			client.IsOwner = true
		}
	}

	r.checkReveal()

	return nil
}

func (r *Room) ToggleBacklogMode(ctx context.Context, clientID string) error {
	client, ok := r.FindClient(clientID)
	if !ok {
		return fmt.Errorf("client %s not found in room %s", clientID, r.ID)
	}
	if !client.IsOwner {
		return fmt.Errorf("only the room owner can toggle backlog mode")
	}

	if !r.BacklogMode {
		r.BacklogMode = true
		if r.CurrentStory != "" {
			r.Stories = []Story{{ID: uuid.NewString(), Name: r.CurrentStory}}
			r.CurrentStoryIndex = 0
		}
	} else {
		r.BacklogMode = false
		if name := r.getCurrentStoryName(); name != "" {
			r.CurrentStory = name
		}
		r.Stories = nil
		r.CurrentStoryIndex = 0
	}
	return nil
}

func (r *Room) AddStory(ctx context.Context, clientID string, name string) error {
	client, ok := r.FindClient(clientID)
	if !ok {
		return fmt.Errorf("client %s not found in room %s", clientID, r.ID)
	}
	if !client.IsOwner {
		return fmt.Errorf("only the room owner can add a story")
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
	client, ok := r.FindClient(clientID)
	if !ok {
		return fmt.Errorf("client %s not found in room %s", clientID, r.ID)
	}
	if !client.IsOwner {
		return fmt.Errorf("only the room owner can remove a story")
	}
	if storyID == "" {
		return fmt.Errorf("story ID cannot be empty")
	}
	index := r.storyIndexByID(storyID)
	if index == -1 {
		return fmt.Errorf("story %s not found in room %s", storyID, r.ID)
	}

	if index == r.CurrentStoryIndex {
		if len(r.Stories) == 1 {
			r.CurrentStoryIndex = 0
			r.Stories = nil
			r.reveal(false)
			r.Clients.ForEach(func(c *Client) {
				c.Vote(ctx, nil)
			})
			return nil
		} else if index == len(r.Stories)-1 {
			r.CurrentStoryIndex--
		}
		r.reveal(false)
		r.Clients.ForEach(func(c *Client) {
			c.Vote(ctx, nil)
		})
	} else if index < r.CurrentStoryIndex {
		r.CurrentStoryIndex--
	}

	r.Stories = append(r.Stories[:index], r.Stories[index+1:]...)
	return nil
}

// SelectStory selects a pending story and starts a fresh voting round.
func (r *Room) SelectStory(ctx context.Context, clientID string, storyID string) error {
	client, ok := r.FindClient(clientID)
	if !ok {
		return fmt.Errorf("client %s not found in room %s", clientID, r.ID)
	}
	if !client.IsOwner {
		return fmt.Errorf("only the room owner can select a story")
	}
	if storyID == "" {
		return fmt.Errorf("story ID cannot be empty")
	}

	index := r.storyIndexByID(storyID)
	if index == -1 {
		return fmt.Errorf("story %s not found in room %s", storyID, r.ID)
	}
	if index == r.CurrentStoryIndex {
		return fmt.Errorf("story %s is already the current story", storyID)
	}
	if r.Stories[index].Voted {
		return fmt.Errorf("estimated story %s cannot be selected", storyID)
	}

	r.CurrentStoryIndex = index
	r.reveal(false)
	r.Clients.ForEach(func(c *Client) {
		c.Vote(ctx, nil)
	})
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
	client, ok := r.FindClient(clientID)
	if !ok {
		return 0, fmt.Errorf("client %s not found in room %s", clientID, r.ID)
	}
	if !client.IsOwner {
		return 0, fmt.Errorf("only the room owner can reorder stories")
	}
	if storyID == "" {
		return 0, fmt.Errorf("story ID cannot be empty")
	}
	if targetIndex < 0 || targetIndex >= len(r.Stories) {
		return 0, fmt.Errorf("target story index %d out of range", targetIndex)
	}

	storyIndex := r.storyIndexByID(storyID)
	if storyIndex == -1 {
		return 0, fmt.Errorf("story %s not found in room %s", storyID, r.ID)
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
	client, ok := r.FindClient(clientID)
	if !ok {
		return fmt.Errorf("client %s not found in room %s", clientID, r.ID)
	}
	if !client.IsOwner {
		return fmt.Errorf("only the room owner can advance to the next story")
	}

	if r.CurrentStoryIndex < len(r.Stories)-1 {
		r.CurrentStoryIndex++
		r.reveal(false)
		r.Clients.ForEach(func(c *Client) {
			c.Vote(ctx, nil)
		})
	}
	return nil
}

func (r *Room) PrevStory(ctx context.Context, clientID string) error {
	client, ok := r.FindClient(clientID)
	if !ok {
		return fmt.Errorf("client %s not found in room %s", clientID, r.ID)
	}
	if !client.IsOwner {
		return fmt.Errorf("only the room owner can go to the previous story")
	}

	if r.CurrentStoryIndex > 0 {
		r.CurrentStoryIndex--
		r.reveal(false)
		r.Clients.ForEach(func(c *Client) {
			c.Vote(ctx, nil)
		})
	}
	return nil
}

func (r *Room) EffectiveCurrentStory() string {
	if r.BacklogMode && len(r.Stories) > 0 && r.CurrentStoryIndex < len(r.Stories) {
		return r.Stories[r.CurrentStoryIndex].Name
	}
	return r.CurrentStory
}

func (r *Room) getCurrentStoryName() string {
	if len(r.Stories) > 0 && r.CurrentStoryIndex < len(r.Stories) {
		return r.Stories[r.CurrentStoryIndex].Name
	}
	return ""
}

func (r *Room) ResetVoting(ctx context.Context, clientID string) error {
	client, ok := r.FindClient(clientID)
	if !ok {
		return fmt.Errorf("client %s not found in room %s", clientID, r.ID)
	}
	if !client.IsOwner {
		return fmt.Errorf("only the room owner can reset voting")
	}

	r.reveal(false)

	r.Clients.ForEach(func(c *Client) {
		c.Vote(ctx, nil)
	})
	return nil
}

func (r *Room) checkReveal() {
	activeClients := r.Clients.Filter(func(client *Client) bool {
		return !client.IsSpectator
	})

	if lo.EveryBy(activeClients.Values(), func(client *Client) bool {
		return client.HasVoted
	}) {
		r.reveal(true)
	}
}

func (r *Room) CountOwners() int {
	return r.Clients.Filter(func(client *Client) bool {
		return client.IsOwner
	}).Count()
}

func (r *Room) ToggleSpectator(ctx context.Context, clientID string, targetClientID string) error {
	client, ok := r.FindClient(clientID)
	if !ok {
		return fmt.Errorf("client %s not found in room %s", clientID, r.ID)
	}
	if !client.IsOwner {
		return fmt.Errorf("only the room owner can toggle ownership")
	}

	if targetClient, ok := r.FindClient(targetClientID); ok {
		targetClient.IsSpectator = !targetClient.IsSpectator
		targetClient.Vote(ctx, nil)
		r.checkReveal()
	} else {
		return fmt.Errorf("target client %s not found in room %s", targetClientID, r.ID)
	}
	return nil
}

func (r *Room) ToggleOwner(ctx context.Context, clientID string, targetClientID string) error {
	client, ok := r.FindClient(clientID)
	if !ok {
		return fmt.Errorf("client %s not found in room %s", clientID, r.ID)
	}
	if !client.IsOwner {
		return fmt.Errorf("only the room owner can toggle ownership")
	}

	owners := r.Clients.Filter(func(client *Client) bool {
		return client.IsOwner
	})

	ownerCount := owners.Count()

	if ownerCount == 1 {
		if first, ok := owners.First(); ok && first.ID == targetClientID && first.IsOwner {
			// Prevent removing the last owner
			return nil
		}
	}

	if targetClient, ok := r.FindClient(targetClientID); ok {
		targetClient.IsOwner = !targetClient.IsOwner
	} else {
		return fmt.Errorf("target client %s not found in room %s", targetClientID, r.ID)
	}
	return nil
}

// AdminToggleOwner toggles a client's owner status without checking
// that the caller is an owner
func (r *Room) AdminToggleOwner(ctx context.Context, targetClientID string) error {
	owners := r.Clients.Filter(func(client *Client) bool {
		return client.IsOwner
	})

	ownerCount := owners.Count()

	// Prevent removing the last owner
	if ownerCount == 1 {
		if first, ok := owners.First(); ok && first.ID == targetClientID && first.IsOwner {
			return domainerror.ErrLastOwner
		}
	}

	if targetClient, ok := r.FindClient(targetClientID); ok {
		targetClient.IsOwner = !targetClient.IsOwner
	} else {
		return fmt.Errorf("target client %s not found in room %s: %w", targetClientID, r.ID, domainerror.ErrClientNotFound)
	}

	return nil
}

func (r *Room) SetCurrentStory(ctx context.Context, clientID string, story string) error {
	client, ok := r.FindClient(clientID)
	if !ok {
		return fmt.Errorf("client %s not found in room %s", clientID, r.ID)
	}
	if !client.IsOwner {
		return fmt.Errorf("only the room owner can set the current story")
	}

	if r.BacklogMode && r.CurrentStoryIndex >= 0 && r.CurrentStoryIndex < len(r.Stories) {
		r.Stories[r.CurrentStoryIndex].Name = story
	} else {
		r.CurrentStory = story
	}
	return nil
}

func (r *Room) ToggleReveal(ctx context.Context, clientID string) error {
	client, ok := r.FindClient(clientID)
	if !ok {
		return fmt.Errorf("client %s not found in room %s", clientID, r.ID)
	}
	if !client.IsOwner {
		return fmt.Errorf("only the room owner can toggle reveal")
	}

	r.reveal(!r.Reveal)
	return nil
}

func (r *Room) reveal(reveal bool) {
	if reveal && r.Reveal {
		return
	}

	r.Reveal = reveal

	if !reveal {
		r.clearConsensus()
		return
	}

	deck := r.Deck()
	metrics := r.collectVotes(deck)
	r.MostAppearingVotes = mostAppearingVotes(metrics.counts, getMostVoteCount(metrics.counts), deck)

	if metrics.count > 0 {
		r.Result = lo.ToPtr(metrics.sum / metrics.count)
	} else {
		r.Result = nil
	}

	summary := calculateConsensus(deck, metrics.ordered)
	r.Consensus = summary.Consensus
	r.LowestVote = summary.LowestVote
	r.HighestVote = summary.HighestVote
	r.VoteRange = summary.VoteRange
	r.VoteSpread = summary.VoteSpread
	r.SpecialVoteCount = metrics.specialCount
	if r.BacklogMode && r.CurrentStoryIndex >= 0 && r.CurrentStoryIndex < len(r.Stories) {
		r.Stories[r.CurrentStoryIndex].Result = r.Result
		r.Stories[r.CurrentStoryIndex].MostAppearingVotes = r.MostAppearingVotes
		r.Stories[r.CurrentStoryIndex].Voted = true
	}
}

type voteMetrics struct {
	sum          float32
	count        float32
	counts       map[string]int
	ordered      []string
	specialCount int
}

func (r *Room) collectVotes(deck Deck) voteMetrics {
	metrics := voteMetrics{counts: make(map[string]int)}

	for _, client := range r.Clients.Values() {
		if client.IsSpectator || client.CurrentVote == nil {
			continue
		}

		vote := *client.CurrentVote
		if vote == "" {
			continue
		}

		if _, ok := deck.Position(vote); !ok {
			metrics.specialCount++
			continue
		}

		if value, err := strconv.Atoi(vote); err == nil {
			metrics.sum += float32(value)
			metrics.count++
		}
		metrics.counts[vote]++
		metrics.ordered = append(metrics.ordered, vote)
	}

	return metrics
}

func mostAppearingVotes(votes map[string]int, mostVoteCount int, deck Deck) []string {
	mostVotes := make([]string, 0)
	for vote, count := range votes {
		if count == mostVoteCount {
			mostVotes = append(mostVotes, vote)
		}
	}
	slices.SortFunc(mostVotes, func(a, b string) int {
		aPosition, _ := deck.Position(a)
		bPosition, _ := deck.Position(b)
		return aPosition - bPosition
	})

	return mostVotes
}

func getMostVoteCount(voteMap map[string]int) int {
	var mostVoteCount int
	for _, count := range voteMap {
		if count > mostVoteCount {
			mostVoteCount = count
		}
	}

	return mostVoteCount
}

const (
	consensusHigh        = "High"
	consensusMedium      = "Medium"
	consensusLow         = "Low"
	consensusUnavailable = "Unavailable"
)

type voteSummary struct {
	Consensus   string
	LowestVote  *string
	HighestVote *string
	VoteRange   *int
	VoteSpread  *int
}

// calculateConsensus evaluates ordered vote labels using deck positions.
// VoteRange is a numeric distance only for numeric decks; VoteSpread always
// measures the distance between ordered deck positions.
func calculateConsensus(deck Deck, votes []string) voteSummary {
	if len(votes) == 0 {
		return voteSummary{Consensus: consensusUnavailable}
	}

	voteCounts := make(map[string]int, len(votes))
	for _, vote := range votes {
		voteCounts[vote]++
	}

	minPosition, maxPosition := -1, -1
	var minLabel, maxLabel string
	for _, vote := range votes {
		position, ok := deck.Position(vote)
		if !ok {
			continue
		}
		if minPosition == -1 || position < minPosition {
			minPosition, minLabel = position, vote
		}
		if position > maxPosition {
			maxPosition, maxLabel = position, vote
		}
	}

	summary := voteSummary{Consensus: consensusUnavailable}
	if minPosition == -1 {
		return summary
	}

	spread := maxPosition - minPosition
	summary.VoteSpread = lo.ToPtr(spread)
	summary.LowestVote = lo.ToPtr(minLabel)
	summary.HighestVote = lo.ToPtr(maxLabel)
	summary.Consensus = consensusLow

	mostVoteCount := getMostVoteCount(voteCounts)
	strongMajority := mostVoteCount >= (2*len(votes)+2)/3
	switch {
	case minLabel == maxLabel:
		summary.Consensus = consensusHigh
	case strongMajority && spread <= 1:
		summary.Consensus = consensusHigh
	case spread <= 2:
		summary.Consensus = consensusMedium
	}

	if minValue, minErr := strconv.Atoi(minLabel); minErr == nil {
		if maxValue, maxErr := strconv.Atoi(maxLabel); maxErr == nil {
			summary.VoteRange = lo.ToPtr(maxValue - minValue)
		}
	}

	return summary
}

func (r *Room) clearConsensus() {
	r.Result = nil
	r.MostAppearingVotes = nil
	r.Consensus = ""
	r.LowestVote = nil
	r.HighestVote = nil
	r.VoteRange = nil
	r.VoteSpread = nil
	r.SpecialVoteCount = 0
}

func (r *Room) IsEmpty() bool {
	return r.Clients.Count() == 0
}

func (r *Room) FindClient(clientID string) (*Client, bool) {
	return r.Clients.Filter(func(client *Client) bool {
		return client.ID == clientID
	}).First()
}

func (r *Room) Vote(ctx context.Context, clientID string, vote *string) error {
	client, ok := r.FindClient(clientID)
	if !ok {
		return fmt.Errorf("client %s not found in room %s", clientID, r.ID)
	}

	if vote != nil && *vote != "" {
		deck := r.Deck()
		if !deck.HasCard(*vote) {
			return fmt.Errorf("vote %q is not part of deck %s", *vote, deck.ID)
		}
	}

	client.Vote(ctx, vote)
	r.checkReveal()

	return nil
}

func (r *Room) UpdateClientName(ctx context.Context, clientID string, name string) error {
	client, ok := r.FindClient(clientID)
	if !ok {
		return fmt.Errorf("client %s not found in room %s", clientID, r.ID)
	}

	client.UpdateName(ctx, name)

	return nil
}
