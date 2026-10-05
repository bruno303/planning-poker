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

		// Bus returns the bus this instance currently holds for clientID, if any.
		// A client holds at most one bus per instance.
		Bus(clientID string) (Bus, bool)
		// HasBusInRoom reports whether clientID currently has an active bus for
		// roomID on this instance.
		HasBusInRoom(clientID string, roomID string) bool
		// BusIfInRoom returns the bus for clientID on this instance, but only when
		// that bus belongs to roomID. A client can only hold one bus per instance,
		// so a bus registered for another room must not be returned here.
		BusIfInRoom(clientID string, roomID string) (Bus, bool)
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
