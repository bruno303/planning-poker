package entity

//go:generate go tool mockgen -destination mocks.go -package entity . ClientCollection

import (
	"context"
	"fmt"
	"slices"
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
		ID                string
		Deck              Deck
		startedAt         time.Time
		Clients           ClientCollection
		CurrentStory      string
		Reveal            bool
		Round             RoundResult
		BacklogMode       bool
		Stories           []Story
		CurrentStoryIndex int
		RoomVersion       uint64
	}
)

func NewRoom(clients ClientCollection) *Room {
	return NewRoomWithDeck(clients, DefaultDeck())
}

func NewRoomWithDeck(clients ClientCollection, deck Deck) *Room {
	return NewRoomWithIDAndDeck(uuid.NewString(), clients, deck)
}

func NewRoomWithID(id string, clients ClientCollection) *Room {
	return NewRoomWithIDAndDeck(id, clients, DefaultDeck())
}

func NewRoomWithIDAndDeck(id string, clients ClientCollection, deck Deck) *Room {
	room := NewRoomWithIDAndStartedAt(id, clients, time.Now().UTC())
	room.Deck = deck.Clone()
	return room
}

// NewRoomWithIDAndStartedAt creates a room with a persisted start time.
func NewRoomWithIDAndStartedAt(id string, clients ClientCollection, startedAt time.Time) *Room {
	return &Room{
		ID:           id,
		Deck:         DefaultDeck(),
		startedAt:    startedAt.UTC(),
		Clients:      clients,
		CurrentStory: "",
		Reveal:       false,
		BacklogMode:  true,
	}
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
		r.Round = RoundResult{}
		return
	}

	r.Round = EvaluateVotes(r.Deck, r.voterCards())
	if r.BacklogMode && r.CurrentStoryIndex >= 0 && r.CurrentStoryIndex < len(r.Stories) {
		r.Stories[r.CurrentStoryIndex].Result = r.Round.Average
		r.Stories[r.CurrentStoryIndex].MostAppearingVotes = r.Round.MostCommon
		r.Stories[r.CurrentStoryIndex].Voted = true
	}
}

func (r *Room) voterCards() []string {
	clients := r.Clients.Values()
	cards := make([]string, 0, len(clients))
	for _, client := range clients {
		if client.IsSpectator || client.CurrentVote == nil || *client.CurrentVote == "" {
			continue
		}
		cards = append(cards, *client.CurrentVote)
	}
	return cards
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
	if vote != nil && *vote != "" && !r.Deck.Contains(*vote) {
		return fmt.Errorf("vote %q: %w", *vote, domainerror.ErrVoteNotInDeck)
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
