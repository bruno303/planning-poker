package entity

import (
	"context"
	"fmt"

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
)

func (r *Room) NewClient(id ClientID) *Client {
	client := newClient(id)
	r.Clients.Add(client)

	if r.Clients.Count() == 1 {
		client.IsOwner = true
	}

	return client
}

func (r *Room) RemoveClient(ctx context.Context, clientID ClientID) error {
	r.Clients.Remove(clientID.String())

	if r.CountOwners() == 0 && r.Clients.Count() > 0 {
		if client, ok := r.Clients.First(); ok {
			client.IsOwner = true
		}
	}

	r.checkReveal()

	return nil
}

func (r *Room) FindClient(clientID ClientID) (*Client, bool) {
	return r.Clients.Filter(func(client *Client) bool {
		return client.ID == clientID
	}).First()
}

func (r *Room) CountOwners() int {
	return r.Clients.Filter(func(client *Client) bool {
		return client.IsOwner
	}).Count()
}

func (r *Room) ToggleSpectator(ctx context.Context, clientID ClientID, targetClientID ClientID) error {
	client, ok := r.FindClient(clientID)
	if !ok {
		return fmt.Errorf("client %s not found in room %s", clientID, r.ID)
	}
	if !client.IsOwner {
		return fmt.Errorf("only the room owner can toggle ownership")
	}

	if targetClient, ok := r.FindClient(targetClientID); ok {
		targetClient.IsSpectator = !targetClient.IsSpectator
		r.applyVote(targetClient, nil)
		r.checkReveal()
	} else {
		return fmt.Errorf("target client %s not found in room %s", targetClientID, r.ID)
	}
	return nil
}

func (r *Room) ToggleOwner(ctx context.Context, clientID ClientID, targetClientID ClientID) error {
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
func (r *Room) AdminToggleOwner(ctx context.Context, targetClientID ClientID) error {
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

func (r *Room) UpdateClientName(ctx context.Context, clientID ClientID, name string) error {
	client, ok := r.FindClient(clientID)
	if !ok {
		return fmt.Errorf("client %s not found in room %s", clientID, r.ID)
	}

	client.UpdateName(ctx, name)

	return nil
}

func (r *Room) IsEmpty() bool {
	return r.Clients.Count() == 0
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
