package entity

//go:generate go tool mockgen -destination mocks.go -package entity . ClientCollection

import (
	"context"
	"fmt"
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
		ID                 string
		Clients            ClientCollection
		CurrentStory       string
		Reveal             bool
		Result             *float32
		MostAppearingVotes []int
		BacklogMode        bool
		Stories            []Story
		CurrentStoryIndex  int
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
			r.Stories = []Story{{Name: r.CurrentStory}}
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

	r.Stories = append(r.Stories, Story{Name: name})
	if len(r.Stories) == 1 {
		r.CurrentStoryIndex = 0
	}

	return nil
}

func (r *Room) RemoveStory(ctx context.Context, clientID string, index int) error {
	client, ok := r.FindClient(clientID)
	if !ok {
		return fmt.Errorf("client %s not found in room %s", clientID, r.ID)
	}
	if !client.IsOwner {
		return fmt.Errorf("only the room owner can remove a story")
	}

	if index < 0 || index >= len(r.Stories) {
		return fmt.Errorf("story index %d out of range", index)
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

	if r.Reveal && r.BacklogMode && r.CurrentStoryIndex >= 0 && r.CurrentStoryIndex < len(r.Stories) {
		r.Stories[r.CurrentStoryIndex].Result = r.Result
		r.Stories[r.CurrentStoryIndex].MostAppearingVotes = r.MostAppearingVotes
		r.Stories[r.CurrentStoryIndex].Voted = true
	}

	return nil
}

func (r *Room) reveal(reveal bool) {
	r.Reveal = reveal

	if !reveal {
		r.Result = nil
		return
	}

	var voteSum float32 = 0
	var voteCount float32 = 0
	var votesCountMap = make(map[int]int)

	for _, client := range r.Clients.Values() {
		if !client.IsSpectator {
			if client.CurrentVote != nil {
				if vote, err := strconv.Atoi(*client.CurrentVote); err == nil {
					voteSum += float32(vote)
					voteCount++
					votesCountMap[vote]++
				}
			}
		}
	}

	r.MostAppearingVotes = []int{}

	mostVoteCount := r.getMostVoteCount(votesCountMap)
	for vote, count := range votesCountMap {
		if count == mostVoteCount {
			r.MostAppearingVotes = append(r.MostAppearingVotes, vote)
		}
	}

	if voteCount > 0 {
		r.Result = lo.ToPtr(voteSum / voteCount)
	} else {
		r.Result = nil
	}
}

func (r *Room) getMostVoteCount(voteMap map[int]int) int {
	var mostVoteCount int
	for _, count := range voteMap {
		if count > mostVoteCount {
			mostVoteCount = count
		}
	}

	return mostVoteCount
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
