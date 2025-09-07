package planningpoker

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/samber/lo"
)

type Room struct {
	ID           string
	Owner        string
	OwnerClient  *Client
	Clients      map[string]*Client
	CurrentStory string
	Reveal       bool
	Hub          *Hub
	lock         sync.Mutex
}

type Hub struct {
	lock  sync.Mutex
	Rooms map[string]*Room
}

func NewHub() *Hub {
	return &Hub{
		Rooms: make(map[string]*Room),
	}
}

func (h *Hub) NewRoom(owner string) *Room {
	h.lock.Lock()
	defer h.lock.Unlock()

	roomID := uuid.NewString()
	room := &Room{
		ID:      roomID,
		Clients: make(map[string]*Client),
		Owner:   owner,
	}
	h.Rooms[roomID] = room
	room.Hub = h
	return room
}

func (h *Hub) GetRoom(roomID string) (*Room, error) {
	h.lock.Lock()
	defer h.lock.Unlock()

	if room, exists := h.Rooms[roomID]; exists {
		return room, nil
	}

	return nil, fmt.Errorf("room %s not found", roomID)
}

func (h *Hub) RemoveRoom(roomID string) {
	h.lock.Lock()
	defer h.lock.Unlock()

	delete(h.Rooms, roomID)
}

func (r *Room) AddClient(client *Client) {
	r.lock.Lock()
	defer r.lock.Unlock()

	r.Clients[client.ID] = client
	if len(r.Clients) == 1 {
		client.IsOwner = true
	}
}

func (r *Room) RemoveClient(ctx context.Context, clientID string) {
	r.lock.Lock()
	defer r.lock.Unlock()

	delete(r.Clients, clientID)
	if len(r.Clients) == 0 {
		r.Hub.RemoveRoom(r.ID)
		return
	}

	if r.CountOwners() == 0 {
		if client, ok := lo.First(lo.Values(r.Clients)); ok {
			client.IsOwner = true
		}
	}

	r.Broadcast(ctx, NewRoomStateCommand(r))
}

func (r *Room) Broadcast(ctx context.Context, message any) {
	for _, client := range r.Clients {
		client.Send(ctx, message)
	}
}

func (r *Room) NewVoting() {
	r.lock.Lock()
	defer r.lock.Unlock()

	r.CurrentStory = ""
	r.Reveal = false
	for _, client := range r.Clients {
		client.HasVoted = false
		client.Vote = nil
	}
}

func (r *Room) ResetVoting() {
	r.lock.Lock()
	defer r.lock.Unlock()

	r.Reveal = false
	for _, client := range r.Clients {
		client.HasVoted = false
		client.Vote = nil
	}
}

func (r *Room) checkReveal() {
	r.lock.Lock()
	defer r.lock.Unlock()

	activeClients := lo.Filter(lo.Values(r.Clients), func(client *Client, _ int) bool {
		return !client.IsSpectator
	})

	if lo.EveryBy(activeClients, func(client *Client) bool {
		return client.HasVoted
	}) {
		r.Reveal = true
	}
}

func (r *Room) CountOwners() int {
	ownerCount := 0
	for _, client := range r.Clients {
		if client.IsOwner {
			ownerCount++
		}
	}
	return ownerCount
}

func (r *Room) ToggleSpectator(ctx context.Context, clientID string) {
	for _, client := range r.Clients {
		if client.ID == clientID {
			client.IsSpectator = !client.IsSpectator
			client.vote(ctx, nil)
			return
		}
	}
}

func (r *Room) ToggleOwner(clientID string) {
	ownerCount := r.CountOwners()
	for _, client := range r.Clients {
		if client.ID == clientID {
			if client.IsOwner && ownerCount == 1 {
				return
			}
			client.IsOwner = !client.IsOwner
		}
	}
}

func (r *Room) SetCurrentStory(story string) {
	r.lock.Lock()
	defer r.lock.Unlock()

	r.CurrentStory = story
}
