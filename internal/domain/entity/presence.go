package entity

//go:generate go tool mockgen -destination mocks.go -package entity . ClientCollection

import (
	"context"
	"fmt"
	"time"

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

	// Presence owns the clients of a room together with their ownership,
	// spectator state and current votes.
	Presence struct {
		roomID  string
		clients ClientCollection
	}
)

func NewPresence(roomID string, clients ClientCollection) *Presence {
	return &Presence{
		roomID:  roomID,
		clients: clients,
	}
}

// Add registers a new client. The first client of the room becomes its owner.
func (p *Presence) Add(id string) *Client {
	client := newClient(id)
	p.clients.Add(client)

	if p.clients.Count() == 1 {
		client.IsOwner = true
	}

	return client
}

// Remove removes a client and reassigns ownership to the first remaining client
// when no owner is left.
func (p *Presence) Remove(id string) {
	p.clients.Remove(id)

	if p.OwnerCount() == 0 && p.clients.Count() > 0 {
		if client, ok := p.clients.First(); ok {
			client.IsOwner = true
		}
	}
}

func (p *Presence) Find(id string) (*Client, bool) {
	return p.clients.Filter(func(client *Client) bool {
		return client.ID == id
	}).First()
}

func (p *Presence) Clients() ClientCollection {
	return p.clients
}

func (p *Presence) Values() []*Client {
	return p.clients.Values()
}

func (p *Presence) Count() int {
	return p.clients.Count()
}

func (p *Presence) IsEmpty() bool {
	return p.clients.Count() == 0
}

func (p *Presence) OwnerCount() int {
	return p.clients.Filter(func(client *Client) bool {
		return client.IsOwner
	}).Count()
}

// Require returns the client with the given ID or the standard missing-client
// error.
func (p *Presence) Require(clientID string) (*Client, error) {
	client, ok := p.Find(clientID)
	if !ok {
		return nil, fmt.Errorf("client %s not found in room %s", clientID, p.roomID)
	}

	return client, nil
}

// RequireOwner returns the client with the given ID when it exists and is an
// owner. The action is used in the authorization error message.
func (p *Presence) RequireOwner(clientID, action string) (*Client, error) {
	client, err := p.Require(clientID)
	if err != nil {
		return nil, err
	}

	if !client.IsOwner {
		return nil, fmt.Errorf("only the room owner can %s", action)
	}

	return client, nil
}

func (p *Presence) ToggleSpectator(_ context.Context, actorID, targetID string) error {
	if _, err := p.RequireOwner(actorID, "toggle ownership"); err != nil {
		return err
	}

	target, ok := p.Find(targetID)
	if !ok {
		return fmt.Errorf("target client %s not found in room %s", targetID, p.roomID)
	}

	target.IsSpectator = !target.IsSpectator

	return nil
}

func (p *Presence) ToggleOwner(_ context.Context, actorID, targetID string) error {
	if _, err := p.RequireOwner(actorID, "toggle ownership"); err != nil {
		return err
	}

	owners := p.clients.Filter(func(client *Client) bool {
		return client.IsOwner
	})
	if owners.Count() == 1 {
		if first, ok := owners.First(); ok && first.ID == targetID && first.IsOwner {
			// Prevent removing the last owner
			return nil
		}
	}

	target, ok := p.Find(targetID)
	if !ok {
		return fmt.Errorf("target client %s not found in room %s", targetID, p.roomID)
	}

	target.IsOwner = !target.IsOwner

	return nil
}

// AdminToggleOwner toggles a client's owner status without checking that the
// caller is an owner.
func (p *Presence) AdminToggleOwner(_ context.Context, targetID string) error {
	owners := p.clients.Filter(func(client *Client) bool {
		return client.IsOwner
	})
	if owners.Count() == 1 {
		if first, ok := owners.First(); ok && first.ID == targetID && first.IsOwner {
			return domainerror.ErrLastOwner
		}
	}

	target, ok := p.Find(targetID)
	if !ok {
		return fmt.Errorf("target client %s not found in room %s: %w", targetID, p.roomID, domainerror.ErrClientNotFound)
	}

	target.IsOwner = !target.IsOwner

	return nil
}

// SetVote stores the vote on the client and keeps its derived flags in sync.
func (p *Presence) SetVote(client *Client, vote *Vote) {
	client.CurrentVote = vote

	if vote != nil && !vote.IsEmpty() {
		client.HasVoted = true
		now := time.Now().UTC()
		client.VotedAt = &now
		return
	}

	client.HasVoted = false
	client.VotedAt = nil
}

// ClearVotes clears the vote of every client.
func (p *Presence) ClearVotes() {
	p.clients.ForEach(func(client *Client) {
		p.SetVote(client, nil)
	})
}

// AllActiveVoted reports whether every non-spectator client has voted. An empty
// active set counts as voted, mirroring lo.EveryBy semantics.
func (p *Presence) AllActiveVoted() bool {
	activeClients := p.clients.Filter(func(client *Client) bool {
		return !client.IsSpectator
	})

	return lo.EveryBy(activeClients.Values(), func(client *Client) bool {
		return client.HasVoted
	})
}
