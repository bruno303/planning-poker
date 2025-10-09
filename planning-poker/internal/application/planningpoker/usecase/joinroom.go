package usecase

import (
	"context"
	"fmt"
	"planning-poker/internal/application/lock"
	"planning-poker/internal/application/planningpoker/usecase/dto"
	"planning-poker/internal/domain"
	"planning-poker/internal/domain/entity"

	"github.com/bruno303/go-toolkit/pkg/log"
	"github.com/google/uuid"
)

type (
	JoinRoomCommand struct {
		RoomID     string
		SenderID   string
		BusFactory func(clientID string) domain.Bus
	}
	JoinRoomOutput struct {
		Client *entity.Client
		Room   *entity.Room
		Bus    domain.Bus
	}
	JoinRoomUseCase struct {
		hub         domain.Hub
		lockManager lock.LockManager
		logger      log.Logger
	}
)

func NewJoinRoomUseCase(hub domain.Hub, lockManager lock.LockManager) JoinRoomUseCase {
	return JoinRoomUseCase{
		hub:         hub,
		lockManager: lockManager,
		logger:      log.NewLogger("usecase.joinroom"),
	}
}

func (uc JoinRoomUseCase) Execute(ctx context.Context, cmd JoinRoomCommand) (*JoinRoomOutput, error) {
	output, err := uc.lockManager.WithLock(ctx, cmd.RoomID, func(ctx context.Context) (any, error) {

		room, ok := uc.hub.GetRoom(cmd.RoomID)
		if !ok {
			return nil, fmt.Errorf("room %s not found", cmd.RoomID)
		}

		uc.logger.Debug(ctx, "creating client for room %s", room.ID)
		clientID := uuid.NewString()
		client := room.NewClient(clientID)
		uc.hub.AddClient(client)

		uc.logger.Debug(ctx, "creating bus for client %s on room %s", clientID, room.ID)
		bus := cmd.BusFactory(client.ID)
		uc.hub.AddBus(client.ID, bus)

		output := &JoinRoomOutput{Client: client, Room: room, Bus: bus}

		uc.logger.Debug(ctx, "sending update client ID command for client %s on room %s", clientID, room.ID)
		if err := bus.Send(ctx, dto.NewUpdateClientIDCommand(client.ID)); err != nil {
			return output, fmt.Errorf("failed to send update client ID command: %w", err)
		}

		return output, nil
	})

	if err != nil {
		return nil, err
	}

	return output.(*JoinRoomOutput), nil
}
