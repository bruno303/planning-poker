package domain

import (
	"context"
	"planning-poker/internal/domain/entity"
)

type Hub interface {
	FindClientByID(clientID string) (*entity.Client, bool)
	AddClient(c *entity.Client)
	RemoveClient(ctx context.Context, clientID string, roomID string) error

	NewRoom(ctx context.Context, owner string) *entity.Room
	GetRoom(ctx context.Context, roomID string) (*entity.Room, bool)
	RemoveRoom(roomID string)
	BroadcastToRoom(ctx context.Context, roomID string, message any) error

	GetBus(clientID string) (Bus, bool)
	AddBus(clientID string, bus Bus)
	RemoveBus(clientID string)
}
