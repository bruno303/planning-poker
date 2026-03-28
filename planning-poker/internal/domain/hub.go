package domain

import (
	"context"
	"errors"
	"planning-poker/internal/domain/entity"
)

var ErrRoomNotFound = errors.New("room not found")

type (
	Hub interface {
		FindClientByID(clientID string) (*entity.Client, bool)
		AddClient(c *entity.Client)
		RemoveClient(ctx context.Context, clientID string, roomID string) error

		NewRoom(ctx context.Context) (*entity.Room, error)
		NewRoomWithID(ctx context.Context, roomID string) (*entity.Room, error)
		LoadRoom(ctx context.Context, roomID string) (*entity.Room, error)
		RemoveRoom(roomID string)
		SaveRoom(ctx context.Context, room *entity.Room) error
		BroadcastToRoom(ctx context.Context, roomID string, message any) error

		GetBus(clientID string) (Bus, bool)
		AddBus(ctx context.Context, clientID string, bus Bus)
		RemoveBus(ctx context.Context, clientID string)
	}
	AdminHub interface {
		GetRooms() []*entity.Room
	}
)
