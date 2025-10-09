package inmemory

import (
	"context"
	"fmt"
	"planning-poker/internal/domain"
	"planning-poker/internal/domain/entity"

	"github.com/bruno303/go-toolkit/pkg/log"
)

type InMemoryHub struct {
	Rooms   map[string]*entity.Room
	Clients map[string]*entity.Client
	Buses   map[string]domain.Bus
	logger  log.Logger
}

var _ domain.Hub = (*InMemoryHub)(nil)

func NewHub() *InMemoryHub {
	return &InMemoryHub{
		Rooms:   make(map[string]*entity.Room),
		Clients: make(map[string]*entity.Client),
		Buses:   make(map[string]domain.Bus),
		logger:  log.NewLogger("inmemory.hub"),
	}
}

func (h *InMemoryHub) NewRoom(owner string) *entity.Room {
	room := entity.NewRoom(owner, NewInMemoryClientCollection())
	h.Rooms[room.ID] = room
	return room
}

func (h *InMemoryHub) GetRoom(roomID string) (*entity.Room, bool) {
	for _, room := range h.Rooms {
		if room.ID == roomID {
			return room, true
		}
	}
	return nil, false
}

func (h *InMemoryHub) RemoveRoom(roomID string) {
	delete(h.Rooms, roomID)
}

func (h *InMemoryHub) FindClientByID(clientID string) (*entity.Client, bool) {
	for _, c := range h.Clients {
		if c.ID == clientID {
			return c, true
		}
	}
	return nil, false
}

func (h *InMemoryHub) AddClient(c *entity.Client) {
	h.Clients[c.ID] = c
}

func (h *InMemoryHub) AddBus(clientID string, bus domain.Bus) {
	h.Buses[clientID] = bus
}

func (h *InMemoryHub) GetBus(clientID string) (domain.Bus, bool) {
	bus, ok := h.Buses[clientID]
	return bus, ok
}

func (h *InMemoryHub) RemoveBus(clientID string) {
	delete(h.Buses, clientID)
}

func (h *InMemoryHub) RemoveClient(ctx context.Context, clientID string, roomID string) error {
	delete(h.Clients, clientID)
	h.RemoveBus(clientID)

	if room, ok := h.GetRoom(roomID); ok {
		room.RemoveClient(ctx, clientID)
		if room.IsEmpty() {
			h.RemoveRoom(room.ID)
		}
	}
	return nil
}

func (h *InMemoryHub) BroadcastToRoom(ctx context.Context, roomID string, message any) error {
	room, ok := h.GetRoom(roomID)
	if !ok {
		return fmt.Errorf("room %s not found", roomID)
	}

	for _, client := range room.Clients.Values() {
		bus, ok := h.GetBus(client.ID)
		if !ok {
			h.logger.Warn(ctx, "bus not found for client %s", client.ID)
			continue
		}
		if err := bus.Send(ctx, message); err != nil {
			return fmt.Errorf("failed to send message to client %s: %w", client.ID, err)
		}
	}
	return nil
}
