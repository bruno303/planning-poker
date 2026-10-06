package domain

import (
	"context"
	"planning-poker/internal/domain/entity"
)

type (
	Hub interface {
		FindClientByID(clientID string) (*entity.Client, bool)
		AddClient(c *entity.Client)
		RemoveClient(ctx context.Context, clientID string, roomID string) error

		NewRoom(ctx context.Context) (*entity.Room, error)
		NewRoomWithDeck(ctx context.Context, deckType entity.DeckType) (*entity.Room, error)
		NewRoomWithID(ctx context.Context, roomID string) (*entity.Room, error)
		LoadRoom(ctx context.Context, roomID string) (*entity.Room, error)
		RemoveRoom(roomID string)
		SaveRoom(ctx context.Context, room *entity.Room) error
		SaveRoomIfVersion(ctx context.Context, room *entity.Room, expectedVersion *uint64) error
		BroadcastToRoom(ctx context.Context, roomID string, message any) error

		GetBus(clientID string) (Bus, bool)
		// GetClientsOfRoom returns the number of clients with an active bus in roomID on
		// this instance (local only; not a cluster-wide count).
		GetClientsOfRoom(roomID string) int
		AddBus(ctx context.Context, clientID string, bus Bus) error
		RemoveBus(ctx context.Context, clientID string)
	}
	AdminHub interface {
		GetRooms() []*entity.Room
	}
)

// GetBusOfRoom excludes a client's bus when it belongs to another room.
func GetBusOfRoom(hub Hub, clientID, roomID string) (Bus, bool) {
	bus, ok := hub.GetBus(clientID)
	if !ok || bus.RoomID() != roomID {
		return nil, false
	}
	return bus, true
}

// HasBusInRoom reports local presence using the room-aware bus lookup.
func HasBusInRoom(hub Hub, clientID, roomID string) bool {
	_, ok := GetBusOfRoom(hub, clientID, roomID)
	return ok
}
