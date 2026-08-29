package entity

//go:generate go tool mockgen -destination mocks.go -package entity . ClientCollection

import (
	"context"
	"fmt"
	"slices"
	"strconv"

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
		ID                  string
		Clients             ClientCollection
		CurrentStory        string
		Reveal              bool
		Result              *float32
		MostAppearingVotes  []int
		Consensus           string
		LowestVote          *int
		HighestVote         *int
		VoteRange           *int
		VoteSpread          *int
		NonNumericVoteCount int
		BacklogMode         bool
		Stories             []Story
		CurrentStoryIndex   int
	}
)

func NewRoom(clients ClientCollection) *Room {
	return NewRoomWithID(uuid.NewString(), clients)
}

func NewRoomWithID(id string, clients ClientCollection) *Room {
	return &Room{
		ID:           id,
		Clients:      clients,
		CurrentStory: "",
		Reveal:       false,
		Result:       nil,
		BacklogMode:  true,
	}
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

func (r *Room) NewVoting(ctx context.Context, clientID string) error {
	client, ok := r.FindClient(clientID)
	if !ok {
		return fmt.Errorf("client %s not found in room %s", clientID, r.ID)
	}
	if !client.IsOwner {
		return fmt.Errorf("only the room owner can start a new voting")
	}

	if !r.BacklogMode {
		r.CurrentStory = ""
	}
	r.reveal(false)
	r.Clients.ForEach(func(c *Client) {
		c.Vote(ctx, nil)
	})

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
	client, ok := r.FindClient(clientID)
	if !ok {
		return fmt.Errorf("client %s not found in room %s", clientID, r.ID)
	}
	if !client.IsOwner {
		return fmt.Errorf("only the room owner can reorder stories")
	}
	if storyID == "" {
		return fmt.Errorf("story ID cannot be empty")
	}
	if targetIndex < 0 || targetIndex >= len(r.Stories) {
		return fmt.Errorf("target story index %d out of range", targetIndex)
	}

	storyIndex := r.storyIndexByID(storyID)
	if storyIndex == -1 {
		return fmt.Errorf("story %s not found in room %s", storyID, r.ID)
	}
	if storyIndex == targetIndex {
		return nil
	}

	currentStoryIndex := r.CurrentStoryIndex
	currentStoryID := ""
	if currentStoryIndex >= 0 && currentStoryIndex < len(r.Stories) {
		currentStoryID = r.Stories[currentStoryIndex].ID
	}

	story := r.Stories[storyIndex]
	r.Stories = append(r.Stories[:storyIndex], r.Stories[storyIndex+1:]...)
	r.Stories = slices.Insert(r.Stories, targetIndex, story)
	if currentStoryID != "" {
		r.CurrentStoryIndex = r.storyIndexByID(currentStoryID)
	} else {
		if currentStoryIndex == storyIndex {
			currentStoryIndex = targetIndex
		} else {
			if storyIndex < currentStoryIndex {
				currentStoryIndex--
			}
			if targetIndex <= currentStoryIndex {
				currentStoryIndex++
			}
		}
		r.CurrentStoryIndex = currentStoryIndex
	}
	return nil
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
		return fmt.Errorf("only the room owner can start a new voting")
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

	metrics := r.collectVotes()
	r.MostAppearingVotes = mostAppearingVotes(metrics.counts, getMostVoteCount(metrics.counts))

	if metrics.count > 0 {
		r.Result = lo.ToPtr(metrics.sum / metrics.count)
	} else {
		r.Result = nil
	}

	r.Consensus, r.LowestVote, r.HighestVote, r.VoteRange, r.VoteSpread = calculateConsensus(metrics.values)
	r.NonNumericVoteCount = metrics.nonNumericCount
	if r.BacklogMode && r.CurrentStoryIndex >= 0 && r.CurrentStoryIndex < len(r.Stories) {
		r.Stories[r.CurrentStoryIndex].Result = r.Result
		r.Stories[r.CurrentStoryIndex].MostAppearingVotes = r.MostAppearingVotes
		r.Stories[r.CurrentStoryIndex].Voted = true
	}
}

type voteMetrics struct {
	sum             float32
	count           float32
	counts          map[int]int
	values          []int
	nonNumericCount int
}

func (r *Room) collectVotes() voteMetrics {
	metrics := voteMetrics{counts: make(map[int]int)}

	for _, client := range r.Clients.Values() {
		if client.IsSpectator || client.CurrentVote == nil {
			continue
		}

		vote, err := strconv.Atoi(*client.CurrentVote)
		if err != nil {
			metrics.nonNumericCount++
			continue
		}

		metrics.sum += float32(vote)
		metrics.count++
		metrics.counts[vote]++
		metrics.values = append(metrics.values, vote)
	}

	return metrics
}

func mostAppearingVotes(votes map[int]int, mostVoteCount int) []int {
	mostVotes := make([]int, 0)
	for vote, count := range votes {
		if count == mostVoteCount {
			mostVotes = append(mostVotes, vote)
		}
	}
	slices.Sort(mostVotes)

	return mostVotes
}

func getMostVoteCount(voteMap map[int]int) int {
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

var planningPokerDeck = []int{0, 1, 2, 3, 5, 8, 13, 21, 34, 55, 89}

func calculateConsensus(votes []int) (string, *int, *int, *int, *int) {
	if len(votes) == 0 {
		return consensusUnavailable, nil, nil, nil, nil
	}

	minVote, maxVote := votes[0], votes[0]
	voteCounts := make(map[int]int, len(votes))
	for _, vote := range votes {
		voteCounts[vote]++
		if vote < minVote {
			minVote = vote
		}
		if vote > maxVote {
			maxVote = vote
		}
	}

	voteRange := maxVote - minVote
	minPosition, minKnown := deckPosition(minVote)
	maxPosition, maxKnown := deckPosition(maxVote)
	var spread *int
	if minKnown && maxKnown {
		deckSpread := maxPosition - minPosition
		spread = lo.ToPtr(deckSpread)
	}

	mostVoteCount := getMostVoteCount(voteCounts)
	strongMajority := mostVoteCount >= (2*len(votes)+2)/3
	consensus := consensusUnavailable
	if spread != nil {
		consensus = consensusLow
	}
	switch {
	case minVote == maxVote:
		consensus = consensusHigh
	case spread != nil && strongMajority && *spread <= 1:
		consensus = consensusHigh
	case spread != nil && *spread <= 2:
		consensus = consensusMedium
	}

	return consensus, lo.ToPtr(minVote), lo.ToPtr(maxVote), lo.ToPtr(voteRange), spread
}

func deckPosition(vote int) (int, bool) {
	position, ok := slices.BinarySearch(planningPokerDeck, vote)
	return position, ok
}

func (r *Room) clearConsensus() {
	r.Result = nil
	r.MostAppearingVotes = nil
	r.Consensus = ""
	r.LowestVote = nil
	r.HighestVote = nil
	r.VoteRange = nil
	r.VoteSpread = nil
	r.NonNumericVoteCount = 0
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
