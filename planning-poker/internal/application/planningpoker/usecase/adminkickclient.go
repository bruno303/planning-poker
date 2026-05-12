package usecase

import (
	"context"
	"fmt"
	"planning-poker/internal/application/planningpoker/usecase/dto"
	"planning-poker/internal/domain"
	"planning-poker/internal/domain/entity"

	"github.com/bruno303/go-toolkit/pkg/log"
)

type (
	AdminKickClientCommand struct {
		RoomID   string
		ClientID string
	}
	adminKickClientUseCase struct {
		leaveRoom UseCase[LeaveRoomCommand]
		hub       domain.Hub
		logger    log.Logger
	}
)

var _ UseCase[AdminKickClientCommand] = (*adminKickClientUseCase)(nil)

func NewAdminKickClientUseCase(
	leaveRoom UseCase[LeaveRoomCommand],
	hub domain.Hub,
) *adminKickClientUseCase {
	return &adminKickClientUseCase{
		leaveRoom: leaveRoom,
		hub:       hub,
		logger:    log.NewLogger("usecase.adminkickclient"),
	}
}

func (uc *adminKickClientUseCase) Execute(ctx context.Context, cmd AdminKickClientCommand) error {
	uc.logger.Info(ctx, "Admin kicking client %s from room %s", cmd.ClientID, cmd.RoomID)

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

	// GetBus is called before leaveRoom because leaveRoom -> hub.RemoveClient -> RemoveBus
	// would remove the bus from the hub's map, making GetBus return nil afterwards.
	if busExists {
		if err := bus.Send(ctx, dto.NewKickNotification()); err != nil {
			uc.logger.Error(ctx, "Failed to send kick notification to client", err)
		}
		if err := bus.Close(); err != nil {
			uc.logger.Error(ctx, "Failed to close WebSocket bus for client", err)
		}
	}

	uc.logger.Info(ctx, "Admin successfully kicked client %s from room %s", cmd.ClientID, cmd.RoomID)
	return nil
}
