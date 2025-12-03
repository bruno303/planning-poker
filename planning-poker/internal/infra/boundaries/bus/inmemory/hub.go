package inmemory

import (
	"context"
	"fmt"
	"planning-poker/internal/domain"
	"planning-poker/internal/domain/entity"

	"github.com/bruno303/go-toolkit/pkg/log"
	"github.com/bruno303/go-toolkit/pkg/trace"
	"github.com/samber/lo"
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

func (h *InMemoryHub) NewRoom(ctx context.Context, owner string) *entity.Room {
	room, _ := trace.Trace(ctx, trace.NameConfig("InMemoryHub", "NewRoom"), func(ctx context.Context) (any, error) {

		room := entity.NewRoom(NewInMemoryClientCollection())
		h.Rooms[room.ID] = room
		return room, nil
	})

	return room.(*entity.Room)
}

func (h *InMemoryHub) GetRoom(ctx context.Context, roomID string) (*entity.Room, bool) {
	room, ok := h.Rooms[roomID]
	return room, ok
}

func (h *InMemoryHub) RemoveRoom(roomID string) {
	delete(h.Rooms, roomID)
}

func (h *InMemoryHub) FindClientByID(clientID string) (*entity.Client, bool) {
	client, ok := h.Clients[clientID]
	return client, ok
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
	_, err := trace.Trace(ctx, trace.NameConfig("InMemoryHub", "RemoveClient"), func(ctx context.Context) (any, error) {

		delete(h.Clients, clientID)
		h.RemoveBus(clientID)

		if room, ok := h.GetRoom(ctx, roomID); ok {
			err := room.RemoveClient(ctx, clientID)
			if err != nil {
				return nil, err
			}
			if room.IsEmpty() {
				h.RemoveRoom(room.ID)
			}
		}
		return nil, nil
	})

	return err
}

func (h *InMemoryHub) BroadcastToRoom(ctx context.Context, roomID string, message any) error {
	_, err := trace.Trace(ctx, trace.NameConfig("InMemoryHub", "BroadcastToRoom"), func(ctx context.Context) (any, error) {

		room, ok := h.GetRoom(ctx, roomID)
		if !ok {
			return nil, fmt.Errorf("room %s not found", roomID)
		}

		for _, client := range room.Clients.Values() {
			bus, ok := h.GetBus(client.ID)
			if !ok {
				h.logger.Warn(ctx, "bus not found for client %s", client.ID)
				continue
			}
			if err := bus.Send(ctx, message); err != nil {
				return nil, fmt.Errorf("failed to send message to client %s: %w", client.ID, err)
			}
		}
		return nil, nil
	})

	return err
}

func (h *InMemoryHub) GetRooms() []*entity.Room {
	return lo.MapToSlice(h.Rooms, func(key string, room *entity.Room) *entity.Room {
		return room
	})
}
