package entity

//go:generate mockgen -destination mocks.go -package entity . ClientCollection

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/samber/lo"
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
		ID           string
		Owner        string
		OwnerClient  *Client
		Clients      ClientCollection
		CurrentStory string
		Reveal       bool
	}
)

func NewRoom(owner string, clients ClientCollection) *Room {
	return &Room{
		ID:           uuid.NewString(),
		Owner:        owner,
		Clients:      clients,
		CurrentStory: "",
		Reveal:       false,
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
	client, ok := r.getClient(clientID)
	if !ok {
		return fmt.Errorf("client %s not found in room %s", clientID, r.ID)
	}
	if !client.IsOwner {
		return fmt.Errorf("only the room owner can start a new voting")
	}

	r.CurrentStory = ""
	r.Reveal = false
	r.Clients.ForEach(func(c *Client) {
		c.Vote(ctx, nil)
	})

	return nil
}

func (r *Room) ResetVoting(ctx context.Context, clientID string) error {
	client, ok := r.getClient(clientID)
	if !ok {
		return fmt.Errorf("client %s not found in room %s", clientID, r.ID)
	}
	if !client.IsOwner {
		return fmt.Errorf("only the room owner can start a new voting")
	}

	r.Reveal = false

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
		r.Reveal = true
	}
}

func (r *Room) CountOwners() int {
	return r.Clients.Filter(func(client *Client) bool {
		return client.IsOwner
	}).Count()
}

func (r *Room) ToggleSpectator(ctx context.Context, clientID string, targetClientID string) error {
	client, ok := r.getClient(clientID)
	if !ok {
		return fmt.Errorf("client %s not found in room %s", clientID, r.ID)
	}
	if !client.IsOwner {
		return fmt.Errorf("only the room owner can toggle ownership")
	}

	if targetClient, ok := r.getClient(targetClientID); ok {
		targetClient.IsSpectator = !targetClient.IsSpectator
		targetClient.Vote(ctx, nil)
		r.checkReveal()
	} else {
		return fmt.Errorf("target client %s not found in room %s", targetClientID, r.ID)
	}

	return nil
}

func (r *Room) ToggleOwner(ctx context.Context, clientID string, targetClientID string) error {
	client, ok := r.getClient(clientID)
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

	if targetClient, ok := r.getClient(targetClientID); ok {
		targetClient.IsOwner = !targetClient.IsOwner
	} else {
		return fmt.Errorf("target client %s not found in room %s", targetClientID, r.ID)
	}

	return nil
}

func (r *Room) SetCurrentStory(ctx context.Context, clientID string, story string) error {
	client, ok := r.getClient(clientID)
	if !ok {
		return fmt.Errorf("client %s not found in room %s", clientID, r.ID)
	}
	if !client.IsOwner {
		return fmt.Errorf("only the room owner can set the current story")
	}

	r.CurrentStory = story
	return nil
}

func (r *Room) ToggleReveal(ctx context.Context, clientID string) error {
	client, ok := r.getClient(clientID)
	if !ok {
		return fmt.Errorf("client %s not found in room %s", clientID, r.ID)
	}
	if !client.IsOwner {
		return fmt.Errorf("only the room owner can toggle reveal")
	}

	r.Reveal = !r.Reveal
	return nil
}

func (r *Room) IsEmpty() bool {
	return r.Clients.Count() == 0
}

func (r *Room) getClient(clientID string) (*Client, bool) {
	return r.Clients.Filter(func(client *Client) bool {
		return client.ID == clientID
	}).First()
}

func (r *Room) Vote(ctx context.Context, clientID string, vote *string) error {
	client, ok := r.getClient(clientID)
	if !ok {
		return fmt.Errorf("client %s not found in room %s", clientID, r.ID)
	}

	client.Vote(ctx, vote)
	r.checkReveal()

	return nil
}

func (r *Room) UpdateClientName(ctx context.Context, clientID string, name string) error {
	client, ok := r.getClient(clientID)
	if !ok {
		return fmt.Errorf("client %s not found in room %s", clientID, r.ID)
	}

	client.UpdateName(ctx, name)

	return nil
}
