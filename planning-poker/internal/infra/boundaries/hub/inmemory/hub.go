package inmemory

import (
	"context"
	"errors"
	"fmt"
	"planning-poker/internal/domain"
	"planning-poker/internal/domain/entity"
	"planning-poker/internal/infra/boundaries/hub/clientcollection"

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

func (h *InMemoryHub) NewRoom(ctx context.Context) (*entity.Room, error) {
	room, _ := trace.Trace(ctx, trace.NameConfig("InMemoryHub", "NewRoom"), func(ctx context.Context) (any, error) {
		room := entity.NewRoom(clientcollection.New())
		h.Rooms[room.ID] = room
		return room, nil
	})

	return room.(*entity.Room), nil
}

func (h *InMemoryHub) NewRoomWithID(ctx context.Context, roomID string) (*entity.Room, error) {
	room, _ := trace.Trace(ctx, trace.NameConfig("InMemoryHub", "NewRoomWithID"), func(ctx context.Context) (any, error) {
		room := entity.NewRoomWithID(roomID, clientcollection.New())
		h.Rooms[room.ID] = room
		return room, nil
	})

	return room.(*entity.Room), nil
}

func (h *InMemoryHub) LoadRoom(_ context.Context, roomID string) (*entity.Room, error) {
	room, ok := h.Rooms[roomID]
	if !ok {
		return nil, domain.ErrRoomNotFound
	}

	return room, nil
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

func (h *InMemoryHub) AddBus(_ context.Context, clientID string, bus domain.Bus) {
	h.Buses[clientID] = bus
}

func (h *InMemoryHub) GetBus(clientID string) (domain.Bus, bool) {
	bus, ok := h.Buses[clientID]
	return bus, ok
}

func (h *InMemoryHub) RemoveBus(_ context.Context, clientID string) {
	delete(h.Buses, clientID)
}

func (h *InMemoryHub) RemoveClient(ctx context.Context, clientID string, roomID string) error {
	_, err := trace.Trace(ctx, trace.NameConfig("InMemoryHub", "RemoveClient"), func(ctx context.Context) (any, error) {

		delete(h.Clients, clientID)
		h.RemoveBus(ctx, clientID)

		room, err := h.LoadRoom(ctx, roomID)
		if err != nil {
			if errors.Is(err, domain.ErrRoomNotFound) {
				return nil, nil
			}

			return nil, err
		}

		err = room.RemoveClient(ctx, clientID)
		if err != nil {
			return nil, err
		}
		if room.IsEmpty() {
			h.RemoveRoom(room.ID)
		}
		return nil, nil
	})

	return err
}

func (h *InMemoryHub) SaveRoom(_ context.Context, room *entity.Room) error {
	// In-memory hub doesn't need to persist
	return nil
}

func (h *InMemoryHub) BroadcastToRoom(ctx context.Context, roomID string, message any) error {
	_, err := trace.Trace(ctx, trace.NameConfig("InMemoryHub", "BroadcastToRoom"), func(ctx context.Context) (any, error) {

		room, err := h.LoadRoom(ctx, roomID)
		if err != nil {
			return nil, err
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
