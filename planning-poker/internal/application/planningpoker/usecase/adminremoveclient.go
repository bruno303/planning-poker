package usecase

import (
	"context"
	"fmt"
	"planning-poker/internal/domain"
	"planning-poker/internal/domain/entity"

	"github.com/bruno303/go-toolkit/pkg/log"
)

type (
	AdminRemoveClientCommand struct {
		RoomID   string
		ClientID string
	}
	adminRemoveClientUseCase struct {
		leaveRoom UseCase[LeaveRoomCommand]
		hub       domain.Hub
		logger    log.Logger
	}
)

var _ UseCase[AdminRemoveClientCommand] = (*adminRemoveClientUseCase)(nil)

func NewAdminRemoveClientUseCase(
	leaveRoom UseCase[LeaveRoomCommand],
	hub domain.Hub,
) *adminRemoveClientUseCase {
	return &adminRemoveClientUseCase{
		leaveRoom: leaveRoom,
		hub:       hub,
		logger:    log.NewLogger("usecase.adminremoveclient"),
	}
}

func (uc *adminRemoveClientUseCase) Execute(ctx context.Context, cmd AdminRemoveClientCommand) error {
	uc.logger.Info(ctx, "Admin removing client %s from room %s", cmd.ClientID, cmd.RoomID)

	room, err := uc.hub.LoadRoom(ctx, cmd.RoomID)
	if err != nil {
		uc.logger.Error(ctx, "Error loading room", err)
		return fmt.Errorf("load room: %w", err)
	}

	if room.Clients.Filter(func(c *entity.Client) bool { return c.ID == cmd.ClientID }).Count() == 0 {
		uc.logger.Warn(ctx, "Client %s not found in room %s", cmd.ClientID, cmd.RoomID)
		return fmt.Errorf("client %s not found: %w", cmd.ClientID, domain.ErrClientNotFound)
	}

	bus, busExists := uc.hub.GetBus(cmd.ClientID)

	if err := uc.leaveRoom.Execute(ctx, LeaveRoomCommand{RoomID: cmd.RoomID, SenderID: cmd.ClientID}); err != nil {
		uc.logger.Error(ctx, "Error removing client from room", err)
		return fmt.Errorf("remove client: %w", err)
	}

	// Close the WebSocket bus after removal.
	// GetBus is called before leaveRoom because leaveRoom -> hub.RemoveClient -> RemoveBus
	// would remove the bus from the hub's map, making GetBus return nil afterwards.
	if busExists {
		if err := bus.Close(); err != nil {
			uc.logger.Error(ctx, "Failed to close WebSocket bus for client", err)
		}
	}

	uc.logger.Info(ctx, "Admin successfully removed client %s from room %s", cmd.ClientID, cmd.RoomID)
	return nil
}
